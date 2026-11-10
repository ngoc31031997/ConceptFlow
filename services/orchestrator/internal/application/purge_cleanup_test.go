package application

import (
	"context"
	"errors"
	"testing"

	"orchestrator/internal/domain"
)

type fakeAuthoringCleanup struct{ deleted []string }

func (f *fakeAuthoringCleanup) DeleteAuthoring(_ context.Context, projectID string) error {
	f.deleted = append(f.deleted, projectID)
	return nil
}

type fakeVisualCleanup struct {
	deleted []string
	err     error
}

func (f *fakeVisualCleanup) DeleteVisual(_ context.Context, projectID string) error {
	f.deleted = append(f.deleted, projectID)
	return f.err
}

// purgeAllButLast marks every purge target of saga s1 done except the last one.
func purgeAllButLast(repo *fakeRepo) domain.StepName {
	ctx := context.Background()
	repo.projects["p1"] = &domain.Project{ProjectID: "p1", SagaID: "s1"}
	last := domain.PurgeTargets[len(domain.PurgeTargets)-1]
	for _, t := range domain.PurgeTargets {
		status := domain.SagaStepCompleted
		if t.Step == last.Step {
			status = domain.SagaStepInProgress
		}
		_ = repo.UpdateStep(ctx, &domain.SagaStep{SagaID: "s1", StepName: t.Step, Status: status})
	}
	return last.Step
}

func lastPurgedEvent() StepEvent {
	last := domain.PurgeTargets[len(domain.PurgeTargets)-1]
	return StepEvent{EventType: "artifacts_purged", SagaID: "s1", ProjectID: "p1", Payload: map[string]interface{}{"service": last.Service}}
}

func TestPurge_LastOwnerDoneCleansAuthoringAndVisual(t *testing.T) {
	uc, repo, _, _ := newTestUseCase()
	authoring, visual := &fakeAuthoringCleanup{}, &fakeVisualCleanup{}
	uc.WithAuthoringCleanup(authoring).WithVisualCleanup(visual)
	purgeAllButLast(repo)

	if err := uc.Execute(context.Background(), lastPurgedEvent()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := repo.projects["p1"]; ok {
		t.Fatal("the project row must be gone")
	}
	if len(authoring.deleted) != 1 || len(visual.deleted) != 1 || visual.deleted[0] != "p1" {
		t.Fatalf("cleanups: authoring %v visual %v", authoring.deleted, visual.deleted)
	}
}

func TestPurge_VisualCleanupFailureDoesNotFailTheSaga(t *testing.T) {
	uc, repo, _, _ := newTestUseCase()
	visual := &fakeVisualCleanup{err: errors.New("visual-service down")}
	uc.WithVisualCleanup(visual)
	purgeAllButLast(repo)

	if err := uc.Execute(context.Background(), lastPurgedEvent()); err != nil {
		t.Fatalf("a failed visual cleanup must not fail the saga: %v", err)
	}
	if _, ok := repo.projects["p1"]; ok {
		t.Fatal("the project row must be gone")
	}
}

func TestPurge_NoCleanupWhileAnOwnerIsPending(t *testing.T) {
	uc, repo, _, _ := newTestUseCase()
	visual := &fakeVisualCleanup{}
	uc.WithVisualCleanup(visual)
	purgeAllButLast(repo)
	first := domain.PurgeTargets[0]
	_ = repo.UpdateStep(context.Background(), &domain.SagaStep{SagaID: "s1", StepName: first.Step, Status: domain.SagaStepInProgress})

	if err := uc.Execute(context.Background(), lastPurgedEvent()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(visual.deleted) != 0 {
		t.Fatalf("visual data removed before every owner confirmed: %v", visual.deleted)
	}
}
