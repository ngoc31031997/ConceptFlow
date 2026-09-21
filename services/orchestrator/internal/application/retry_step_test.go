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

// Retrying validate_script used to publish an empty payload, which crashed
// rendering's consumer with KeyError: 'script_content' and left the message
// redelivering forever.
func TestRetryStepUseCase_RebuildsValidateScriptPayload(t *testing.T) {
	repo := newFakeRepo()
	pub := &fakePublisher{}
	uc := NewRetryStepUseCase(repo, pub)

	repo.projects["proj-1"] = &domain.Project{
		ProjectID:           "proj-1",
		SagaID:              "saga-1",
		Status:              domain.StatusFailedValidateScript,
		ScriptContent:       "from conceptflow import *\n",
		ManimSceneClassName: "DemoScene",
		RenderQuality:       domain.DefaultRenderQuality,
	}

	out, err := uc.Execute(context.Background(), "proj-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Status != domain.StatusValidatingScript {
		t.Fatalf("expected status validating_script, got %s", out.Status)
	}

	last := pub.last()
	if last == nil || last.routingKey != "rendering" {
		t.Fatalf("expected command published to rendering routing key, got %+v", last)
	}
	if last.envelope.EventType != string(domain.StepValidateScript) {
		t.Fatalf("expected validate_script event_type, got %s", last.envelope.EventType)
	}
	for _, key := range []string{"script_content", "scene_class_name", "render_quality", "engine"} {
		if _, ok := last.envelope.Payload[key]; !ok {
			t.Fatalf("payload is missing %q — rendering's consumer reads it unconditionally", key)
		}
	}
	if last.envelope.Payload["script_content"] != "from conceptflow import *\n" {
		t.Fatalf("expected the project's script content, got %v", last.envelope.Payload["script_content"])
	}
}
