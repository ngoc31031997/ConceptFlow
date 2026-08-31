package application

import (
	"context"
	"testing"

	"orchestrator/internal/domain"
)

func TestRetryStepUseCase_RebuildsRenderScenesPayload(t *testing.T) {
	repo := newFakeRepo()
	pub := &fakePublisher{}
	uc := NewRetryStepUseCase(repo, pub)

	repo.projects["proj-1"] = &domain.Project{
		ProjectID: "proj-1",
		SagaID:    "saga-1",
		Status:    domain.StatusFailedRenderScenes,
		Scenes: []domain.Scene{
			{SceneIndex: 0, NarrationText: "n0", AudioPath: "a0.wav"},
			{SceneIndex: 1, NarrationText: "n1", AudioPath: "a1.wav"},
		},
	}

	out, err := uc.Execute(context.Background(), "proj-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Status != domain.StatusRendering {
		t.Fatalf("expected status rendering, got %s", out.Status)
	}
	if out.SagaID != "saga-1" {
		t.Fatalf("expected same saga_id reused, got %s", out.SagaID)
	}

	last := pub.last()
	if last == nil || last.routingKey != "rendering" {
		t.Fatalf("expected command published to rendering routing key, got %+v", last)
	}
	if last.envelope.MessageID == "" {
		t.Fatal("expected a new message_id on retry (Rule 5)")
	}

	step, err := repo.GetStep(context.Background(), "saga-1", domain.StepRenderScenes)
	if err != nil {
		t.Fatalf("expected saga step to exist: %v", err)
	}
	if step.Status != domain.SagaStepInProgress {
		t.Fatalf("expected render_scenes step in_progress after retry, got %s", step.Status)
	}
}

func TestRetryStepUseCase_RejectsNonFailedStatus(t *testing.T) {
	repo := newFakeRepo()
	pub := &fakePublisher{}
	uc := NewRetryStepUseCase(repo, pub)

	repo.projects["proj-1"] = &domain.Project{ProjectID: "proj-1", Status: domain.StatusRendering}

	_, err := uc.Execute(context.Background(), "proj-1")
	if err != domain.ErrInvalidStatus {
		t.Fatalf("expected ErrInvalidStatus, got %v", err)
	}
}
