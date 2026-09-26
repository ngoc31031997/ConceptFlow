package application

import (
	"context"
	"errors"
	"time"

	"orchestrator/internal/domain"
)

// ProjectDeleteStore starts a delete atomically: it locks the project, refuses
// while a saga step is running (domain.ErrProjectBusy) and marks it deleting.
// Calling it again on a project already deleting is allowed — that is the
// retry after a failed purge.
type ProjectDeleteStore interface {
	BeginDelete(ctx context.Context, projectID string) (sagaID string, err error)
}

// DeleteProjectUseCase is the delete saga (CR-040 FR114.2): it asks every
// service that writes to shared_artifacts to remove its own files, and the
// project row is removed only when all of them have answered (see
// HandleStepEventUseCase.handlePurgeEvent). Nothing can recreate files after
// the delete because BeginDelete refuses a project with a running step.
type DeleteProjectUseCase struct {
	store     ProjectDeleteStore
	repo      domain.ProjectRepositoryPort
	publisher domain.CommandPublisherPort
}

func NewDeleteProjectUseCase(store ProjectDeleteStore, repo domain.ProjectRepositoryPort, publisher domain.CommandPublisherPort) *DeleteProjectUseCase {
	return &DeleteProjectUseCase{store: store, repo: repo, publisher: publisher}
}

// Execute marks the project deleting and sends purge_project_artifacts to each
// owner. It returns once the commands are queued; the deletion completes
// asynchronously.
func (uc *DeleteProjectUseCase) Execute(ctx context.Context, projectID string) error {
	sagaID, err := uc.store.BeginDelete(ctx, projectID)
	if err != nil {
		return err
	}
	for _, target := range domain.PurgeTargets {
		if err := uc.repo.UpdateStep(ctx, &domain.SagaStep{SagaID: sagaID, StepName: target.Step, Status: domain.SagaStepInProgress}); err != nil {
			return err
		}
		if err := uc.publisher.PublishCommand(ctx, target.RoutingKey, domain.Envelope{
			MessageID: newUUID(),
			SagaID:    sagaID,
			ProjectID: projectID,
			EventType: "purge_project_artifacts",
			Payload:   map[string]interface{}{"event_type": "purge_project_artifacts"},
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}); err != nil {
			return err
		}
	}
	return nil
}

// DeleteProgress is how far the delete saga has got (CR-040 FR116.2): how many
// of the purge owners have confirmed. Gone means the project row is already
// removed, i.e. the saga finished.
type DeleteProgress struct {
	Done   int
	Total  int
	Gone   bool
	Failed string
}

// Progress reads the delete saga's state for the GUI's progress card.
func (uc *DeleteProjectUseCase) Progress(ctx context.Context, projectID string) (DeleteProgress, error) {
	out := DeleteProgress{Total: len(domain.PurgeTargets)}
	project, err := uc.repo.Get(ctx, projectID)
	if errors.Is(err, domain.ErrProjectNotFound) {
		out.Done, out.Gone = out.Total, true
		return out, nil
	}
	if err != nil {
		return out, err
	}
	for _, t := range domain.PurgeTargets {
		step, err := uc.repo.GetStep(ctx, project.SagaID, t.Step)
		if errors.Is(err, domain.ErrSagaStepNotFound) {
			continue
		}
		if err != nil {
			return out, err
		}
		switch step.Status {
		case domain.SagaStepCompleted:
			out.Done++
		case domain.SagaStepFailed:
			out.Failed = t.Service
			if step.ErrorMessage != nil {
				out.Failed += ": " + *step.ErrorMessage
			}
		}
	}
	return out, nil
}
