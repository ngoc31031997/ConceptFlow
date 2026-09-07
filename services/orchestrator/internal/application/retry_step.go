package application

import (
	"context"
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
	domain.StepParseScript:      "script_processing",
	domain.StepClassifyScenes:   "content_plugin",
	domain.StepSynthesizeSpeech: "tts",
	domain.StepRenderScenes:     "rendering",
	domain.StepAssembleVideo:    "video_assembly",
	domain.StepPublishVideo:     "publisher",
}

var stepInProgressStatus = map[domain.StepName]domain.ProjectStatus{
	domain.StepParseScript:      domain.StatusParsingScript,
	domain.StepClassifyScenes:   domain.StatusClassifyingScenes,
	domain.StepSynthesizeSpeech: domain.StatusSynthesizingSpeech,
	domain.StepRenderScenes:     domain.StatusRendering,
	domain.StepAssembleVideo:    domain.StatusAssemblingVideo,
	domain.StepPublishVideo:     domain.StatusPublishing,
}

// failedStatusToStep is the inverse of domain.FailedStatusForStep, used to
// recover which step a failed_at_<step> status refers to.
var failedStatusToStep = map[domain.ProjectStatus]domain.StepName{
	domain.StatusFailedParseScript:      domain.StepParseScript,
	domain.StatusFailedClassifyScenes:   domain.StepClassifyScenes,
	domain.StatusFailedSynthesizeSpeech: domain.StepSynthesizeSpeech,
	domain.StatusFailedRenderScenes:     domain.StepRenderScenes,
	domain.StatusFailedAssembleVideo:    domain.StepAssembleVideo,
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

	sagaID := project.SagaID
	payload := rebuildPayload(stepName, project)

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
func rebuildPayload(stepName domain.StepName, project *domain.Project) map[string]interface{} {
	switch stepName {
	case domain.StepParseScript:
		return map[string]interface{}{"script_content": project.ScriptContent}
	case domain.StepSynthesizeSpeech:
		return map[string]interface{}{"scenes": scenesToPayloadForSynthesis(project.Scenes, string(project.ContentLanguage), project.VoiceID)}
	case domain.StepRenderScenes:
		return map[string]interface{}{
			"scenes":           scenesToPayload(project.Scenes),
			"script_content":   project.ScriptContent,
			"scene_class_name": project.ManimSceneClassName,
		}
	case domain.StepAssembleVideo:
		return assembleVideoPayload(project)
	case domain.StepPublishVideo:
		return publishVideoPayload(project)
	default:
		return map[string]interface{}{}
	}
}
