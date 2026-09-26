package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"orchestrator/internal/domain"
)

// RetryStepOutput is returned to the HTTP layer for the 200 response.
type RetryStepOutput struct {
	SagaID string
	Status domain.ProjectStatus
}

// stepRoutingKey and stepInProgressStatus mirror the routing keys /
// "in-progress" ProjectStatus used when each step was first dispatched —
// reused here so retry re-enters the same state the original dispatch did.
var stepRoutingKey = map[domain.StepName]string{
	domain.StepValidateScript:   "rendering",
	domain.StepSynthesizeSpeech: "tts",
	domain.StepRenderScenes:     "rendering",
	domain.StepAssembleVideo:    "video_assembly",
	// CR-021 D1: same queue as assemble_video — the QC worker lives inside
	// video-assembly, told apart by event_type.
	domain.StepQCVideo:      "video_assembly",
	domain.StepPublishVideo: "publisher",
}

var stepInProgressStatus = map[domain.StepName]domain.ProjectStatus{
	domain.StepValidateScript:   domain.StatusValidatingScript,
	domain.StepSynthesizeSpeech: domain.StatusSynthesizingSpeech,
	domain.StepRenderScenes:     domain.StatusRendering,
	domain.StepAssembleVideo:    domain.StatusAssemblingVideo,
	domain.StepQCVideo:          domain.StatusRunningQC,
	domain.StepPublishVideo:     domain.StatusPublishing,
}

// failedStatusToStep is the inverse of domain.FailedStatusForStep, used to
// recover which step a failed_at_<step> status refers to.
var failedStatusToStep = map[domain.ProjectStatus]domain.StepName{
	domain.StatusFailedParseScript:      domain.StepParseScript,
	domain.StatusFailedValidateScript:   domain.StepValidateScript,
	domain.StatusFailedSynthesizeSpeech: domain.StepSynthesizeSpeech,
	domain.StatusFailedRenderScenes:     domain.StepRenderScenes,
	domain.StatusFailedAssembleVideo:    domain.StepAssembleVideo,
	// Only reachable when the qc_completed message itself was unusable — QC
	// never reports a failure of its own (FR61.4).
	domain.StatusFailedQCVideo: domain.StepQCVideo,
	domain.StatusFailedPublishVideo:     domain.StepPublishVideo,
}

// RetryStepUseCase implements Rule 5: reconstructs the failed step's command
// payload purely from data already accumulated on Project, and publishes it
// with a brand new message_id via the Outbox — no compensating rollback, no
// re-invocation of earlier steps.
type RetryStepUseCase struct {
	repo      domain.ProjectRepositoryPort
	publisher domain.CommandPublisherPort
}

// NewRetryStepUseCase constructs the use case with its port dependencies.
func NewRetryStepUseCase(repo domain.ProjectRepositoryPort, publisher domain.CommandPublisherPort) *RetryStepUseCase {
	return &RetryStepUseCase{repo: repo, publisher: publisher}
}

// Execute validates Project.Status is failed_at_<step> (else
// domain.ErrInvalidStatus → 409), rebuilds that step's command payload from
// Project, and re-dispatches it.
func (uc *RetryStepUseCase) Execute(ctx context.Context, projectID string) (*RetryStepOutput, error) {
	project, err := uc.repo.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(string(project.Status), "failed_at_") {
		return nil, domain.ErrInvalidStatus
	}
	stepName, ok := failedStatusToStep[project.Status]
	if !ok {
		return nil, domain.ErrInvalidStatus
	}

	// CR-040 FR110: a project that failed at the retired parse_script step
	// resumes at validate_script, which now finds the scene class itself.
	if stepName == domain.StepParseScript {
		stepName = domain.StepValidateScript
	}

	sagaID := project.SagaID
	payload, err := rebuildPayload(stepName, project)
	if err != nil {
		return nil, err
	}

	if err := uc.repo.UpdateStep(ctx, &domain.SagaStep{SagaID: sagaID, StepName: stepName, Status: domain.SagaStepInProgress}); err != nil {
		return nil, err
	}

	envelope := domain.Envelope{
		MessageID: newUUID(), // new message_id, per Rule 5
		SagaID:    sagaID,
		ProjectID: projectID,
		EventType: string(stepName),
		Payload:   payload,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	if err := uc.publisher.PublishCommand(ctx, stepRoutingKey[stepName], envelope); err != nil {
		return nil, err
	}

	nextStatus := stepInProgressStatus[stepName]
	if err := uc.repo.UpdateStatus(ctx, projectID, nextStatus); err != nil {
		return nil, err
	}

	return &RetryStepOutput{SagaID: sagaID, Status: nextStatus}, nil
}

// rebuildPayload reconstructs the command payload for stepName purely from
// Project's accumulated data (Rule 5).
func rebuildPayload(stepName domain.StepName, project *domain.Project) (map[string]interface{}, error) {
	switch stepName {
	case domain.StepValidateScript:
		// Must mirror the original dispatch in handle_step_event.go's
		// onScriptParsed: rendering's consumer reads script_content and
		// scene_class_name unconditionally, so an incomplete payload here
		// crashes it with a KeyError instead of failing the step.
		engine := project.RenderEngine
		if !engine.IsValid() {
			engine = domain.DefaultRenderEngine
		}
		return map[string]interface{}{
			"script_content":   project.ScriptContent,
			"scene_class_name": project.ManimSceneClassName,
			"render_quality":   string(project.RenderQuality),
			"engine":           string(engine),
		}, nil
	case domain.StepSynthesizeSpeech:
		return map[string]interface{}{"scenes": scenesToPayloadForSynthesis(project.Scenes, string(project.ContentLanguage), project.VoiceID)}, nil
	case domain.StepRenderScenes:
		// feature/remotion-engine: without "engine" here, retrying this step
		// silently fell back to Manim (rendering's consumer.py defaults an
		// absent engine to "manim") regardless of what render_engine the
		// project actually picked — the render would then run the Manim AST
		// lint against a Remotion (.tsx) script and fail immediately.
		engine := project.RenderEngine
		if !engine.IsValid() {
			engine = domain.DefaultRenderEngine
		}
		return map[string]interface{}{
			"scenes":           scenesToPayload(project.Scenes),
			"script_content":   project.ScriptContent,
			"scene_class_name": project.ManimSceneClassName,
			"engine":           string(engine),
		}, nil
	case domain.StepAssembleVideo:
		return assembleVideoPayload(project), nil
	case domain.StepQCVideo:
		return qcVideoPayload(project), nil
	case domain.StepPublishVideo:
		return publishVideoPayload(project), nil
	default:
		// Never publish an empty payload: a consumer that reads its fields
		// unconditionally would crash on it and redeliver forever. A step
		// with no rebuild rule is a bug here, not a retryable command.
		return nil, fmt.Errorf("retry: no payload rebuild rule for step %q", stepName)
	}
}
