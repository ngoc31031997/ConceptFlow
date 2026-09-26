package application

import (
	"context"
	"time"

	"orchestrator/internal/domain"
)

// runningStatusToStep maps a project status that has a worker on it to the
// saga step that worker is running. A status not listed (draft, awaiting
// review, ready, publishing) has nothing to cancel.
var runningStatusToStep = map[domain.ProjectStatus]domain.StepName{
	domain.StatusParsingScript:      domain.StepParseScript,
	domain.StatusValidatingScript:   domain.StepValidateScript,
	domain.StatusSynthesizingSpeech: domain.StepSynthesizeSpeech,
	domain.StatusRendering:          domain.StepRenderScenes,
	domain.StatusAssemblingVideo:    domain.StepAssembleVideo,
	domain.StatusRunningQC:          domain.StepQCVideo,
}

// CancelStepOutput is what the HTTP layer returns.
type CancelStepOutput struct {
	Step   domain.StepName
	Status domain.ProjectStatus
}

// CancelStepUseCase stops the step a project is running: broadcasts the cancel
// to the workers, then leaves the project failed at that same step with the
// cancelled marker. "Cancelled at step N stays at step N": Retry re-runs that
// step, and finished earlier steps (audio, render cache) are kept.
//
// The step is marked failed BEFORE the workers have actually stopped, and a
// late event from a worker is dropped by HandleStepEventUseCase because the
// step is no longer in_progress — so a slow kill cannot resurrect it.
type CancelStepUseCase struct {
	repo    domain.ProjectRepositoryPort
	control domain.ControlPublisherPort
	now     func() time.Time
}

func NewCancelStepUseCase(repo domain.ProjectRepositoryPort, control domain.ControlPublisherPort) *CancelStepUseCase {
	return &CancelStepUseCase{repo: repo, control: control, now: time.Now}
}

func (uc *CancelStepUseCase) Execute(ctx context.Context, projectID string) (*CancelStepOutput, error) {
	project, err := uc.repo.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	step, ok := runningStatusToStep[project.Status]
	if !ok {
		return nil, domain.ErrInvalidStatus
	}

	// Broadcast first: if the broker is down the Creator hears it as an error
	// and the project is untouched, instead of looking cancelled while a
	// render keeps running.
	if err := uc.control.PublishCancel(ctx, domain.CancelRequest{
		ProjectID: projectID, SagaID: project.SagaID, Step: string(step),
		At: uc.now().UTC().Format(time.RFC3339Nano),
	}); err != nil {
		return nil, err
	}

	msg := domain.CancelledErrorMessage
	if err := uc.repo.UpdateStep(ctx, &domain.SagaStep{
		SagaID: project.SagaID, StepName: step, Status: domain.SagaStepFailed, ErrorMessage: &msg,
	}); err != nil {
		return nil, err
	}
	failed := domain.FailedStatusForStep(step)
	if err := uc.repo.UpdateStatus(ctx, projectID, failed); err != nil {
		return nil, err
	}
	return &CancelStepOutput{Step: step, Status: failed}, nil
}
