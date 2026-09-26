package application

import (
	"context"
	"testing"

	"orchestrator/internal/domain"
)

type fakeDeleteStore struct {
	sagaID string
	err    error
}

func (f fakeDeleteStore) BeginDelete(context.Context, string) (string, error) { return f.sagaID, f.err }

func TestDeleteProject_SendsPurgeToEveryOwner(t *testing.T) {
	repo := newFakeRepo()
	pub := &fakePublisher{}
	uc := NewDeleteProjectUseCase(fakeDeleteStore{sagaID: "s1"}, repo, pub)

	if err := uc.Execute(context.Background(), "p1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pub.published) != len(domain.PurgeTargets) {
		t.Fatalf("expected %d purge commands, got %d", len(domain.PurgeTargets), len(pub.published))
	}
	keys := map[string]bool{}
	for _, c := range pub.published {
		keys[c.routingKey] = true
		if c.envelope.EventType != "purge_project_artifacts" || c.envelope.ProjectID != "p1" {
			t.Fatalf("unexpected command %+v", c)
		}
	}
	for _, want := range []string{"tts", "rendering", "video_assembly"} {
		if !keys[want] {
			t.Errorf("no purge sent to %s", want)
		}
	}
}

func TestDeleteProject_BusyProjectSendsNothing(t *testing.T) {
	pub := &fakePublisher{}
	uc := NewDeleteProjectUseCase(fakeDeleteStore{err: domain.ErrProjectBusy}, newFakeRepo(), pub)

	if err := uc.Execute(context.Background(), "p1"); err != domain.ErrProjectBusy {
		t.Fatalf("expected ErrProjectBusy, got %v", err)
	}
	if len(pub.published) != 0 {
		t.Fatal("no command may be sent for a busy project")
	}
}

func TestDeleteProject_ProgressCountsConfirmedOwners(t *testing.T) {
	repo := newFakeRepo()
	repo.projects["p1"] = &domain.Project{ProjectID: "p1", SagaID: "s1"}
	uc := NewDeleteProjectUseCase(fakeDeleteStore{sagaID: "s1"}, repo, &fakePublisher{})
	ctx := context.Background()

	_ = repo.UpdateStep(ctx, &domain.SagaStep{SagaID: "s1", StepName: domain.PurgeTargets[0].Step, Status: domain.SagaStepCompleted})
	_ = repo.UpdateStep(ctx, &domain.SagaStep{SagaID: "s1", StepName: domain.PurgeTargets[1].Step, Status: domain.SagaStepInProgress})

	p, err := uc.Progress(ctx, "p1")
	if err != nil {
		t.Fatal(err)
	}
	if p.Done != 1 || p.Total != len(domain.PurgeTargets) || p.Gone || p.Failed != "" {
		t.Fatalf("unexpected progress %+v", p)
	}

	delete(repo.projects, "p1")
	p, err = uc.Progress(ctx, "p1")
	if err != nil || !p.Gone || p.Done != p.Total {
		t.Fatalf("a removed project means the saga finished, got %+v err=%v", p, err)
	}
}
