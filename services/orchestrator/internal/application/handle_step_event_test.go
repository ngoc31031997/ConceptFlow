package application

import (
	"context"
	"testing"

	"orchestrator/internal/domain"
)

func newTestUseCase() (*HandleStepEventUseCase, *fakeRepo, *fakePublisher, *fakeProgress) {
	repo := newFakeRepo()
	pub := &fakePublisher{}
	prog := &fakeProgress{}
	return NewHandleStepEventUseCase(repo, pub, prog, nil), repo, pub, prog
}

func TestHandleStepEventUseCase_ScriptParsed_DispatchesClassifyScenes(t *testing.T) {
	uc, repo, pub, prog := newTestUseCase()
	repo.projects["proj-1"] = &domain.Project{ProjectID: "proj-1", Status: domain.StatusParsingScript, PluginID: "plugin-a"}
	repo.steps[stepKey("saga-1", domain.StepParseScript)] = &domain.SagaStep{SagaID: "saga-1", StepName: domain.StepParseScript, Status: domain.SagaStepInProgress}

	err := uc.Execute(context.Background(), StepEvent{
		SagaID: "saga-1", ProjectID: "proj-1", EventType: "script_parsed",
		Payload: map[string]interface{}{
			"scenes": []interface{}{
				map[string]interface{}{"scene_index": float64(0), "narration_text": "n0", "illustration_hint": "h0"},
				map[string]interface{}{"scene_index": float64(1), "narration_text": "n1", "illustration_hint": "h1"},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	project, _ := repo.Get(context.Background(), "proj-1")
	if project.Status != domain.StatusClassifyingScenes {
		t.Fatalf("expected classifying_scenes, got %s", project.Status)
	}
	if len(project.Scenes) != 2 {
		t.Fatalf("expected 2 scenes stored, got %d", len(project.Scenes))
	}

	last := pub.last()
	if last == nil || last.routingKey != "content_plugin" {
		t.Fatalf("expected classify_scenes command dispatched, got %+v", last)
	}

	if prog.last() == nil || prog.last().Status != "completed" {
		t.Fatalf("expected a completed progress message, got %+v", prog.last())
	}
}

func TestHandleStepEventUseCase_Rule1_SceneIndexMismatch(t *testing.T) {
	uc, repo, pub, prog := newTestUseCase()
	repo.projects["proj-1"] = &domain.Project{
		ProjectID: "proj-1",
		Status:    domain.StatusSynthesizingSpeech,
		Scenes: []domain.Scene{
			{SceneIndex: 0, NarrationText: "n0"},
			{SceneIndex: 1, NarrationText: "n1"},
		},
	}
	repo.steps[stepKey("saga-1", domain.StepSynthesizeSpeech)] = &domain.SagaStep{SagaID: "saga-1", StepName: domain.StepSynthesizeSpeech, Status: domain.SagaStepInProgress}

	err := uc.Execute(context.Background(), StepEvent{
		SagaID: "saga-1", ProjectID: "proj-1", EventType: "speech_synthesized",
		Payload: map[string]interface{}{
			"scenes": []interface{}{
				// only scene_index 0 present — mismatch against the 2 accumulated scenes
				map[string]interface{}{"scene_index": float64(0), "audio_path": "a0.wav", "duration_seconds": float64(1.5)},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	project, _ := repo.Get(context.Background(), "proj-1")
	if project.Status != domain.StatusFailedRenderScenes {
		t.Fatalf("expected failed_at_render_scenes on mismatch, got %s", project.Status)
	}
	if len(pub.published) != 0 {
		t.Fatalf("expected no render_scenes command dispatched on mismatch, got %d published", len(pub.published))
	}
	if prog.last() == nil || prog.last().Status != "failed" || prog.last().ErrorMessage == nil {
		t.Fatalf("expected a failed progress message with error_message, got %+v", prog.last())
	}
}

func TestHandleStepEventUseCase_Rule4_UnexpectedEventSkipped(t *testing.T) {
	uc, repo, pub, prog := newTestUseCase()
	repo.projects["proj-1"] = &domain.Project{ProjectID: "proj-1", Status: domain.StatusSynthesizingSpeech}
	// classify_scenes step already completed — a redelivered/out-of-order event should be skipped.
	repo.steps[stepKey("saga-1", domain.StepClassifyScenes)] = &domain.SagaStep{SagaID: "saga-1", StepName: domain.StepClassifyScenes, Status: domain.SagaStepCompleted}

	err := uc.Execute(context.Background(), StepEvent{
		SagaID: "saga-1", ProjectID: "proj-1", EventType: "scenes_classified",
		Payload: map[string]interface{}{},
	})
	if err != nil {
		t.Fatalf("expected no error (skip + ack), got %v", err)
	}
	if len(pub.published) != 0 {
		t.Fatalf("expected no command dispatched for unexpected event, got %d", len(pub.published))
	}
	if len(prog.messages) != 0 {
		t.Fatalf("expected no progress message for skipped unexpected event, got %d", len(prog.messages))
	}
	// status must remain unchanged
	project, _ := repo.Get(context.Background(), "proj-1")
	if project.Status != domain.StatusSynthesizingSpeech {
		t.Fatalf("expected status unchanged, got %s", project.Status)
	}
}

func TestHandleStepEventUseCase_FailureEvent(t *testing.T) {
	uc, repo, _, prog := newTestUseCase()
	repo.projects["proj-1"] = &domain.Project{ProjectID: "proj-1", Status: domain.StatusRendering}
	repo.steps[stepKey("saga-1", domain.StepRenderScenes)] = &domain.SagaStep{SagaID: "saga-1", StepName: domain.StepRenderScenes, Status: domain.SagaStepInProgress}

	err := uc.Execute(context.Background(), StepEvent{
		SagaID: "saga-1", ProjectID: "proj-1", EventType: "rendering_failed",
		Payload: map[string]interface{}{"error_message": "manim template crashed"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	project, _ := repo.Get(context.Background(), "proj-1")
	if project.Status != domain.StatusFailedRenderScenes {
		t.Fatalf("expected failed_at_render_scenes, got %s", project.Status)
	}
	step, _ := repo.GetStep(context.Background(), "saga-1", domain.StepRenderScenes)
	if step.Status != domain.SagaStepFailed {
		t.Fatalf("expected step failed, got %s", step.Status)
	}
	if prog.last() == nil || *prog.last().ErrorMessage != "manim template crashed" {
		t.Fatalf("expected error_message forwarded in progress message, got %+v", prog.last())
	}
}

func TestHandleStepEventUseCase_SceneRenderedProgressOnly(t *testing.T) {
	uc, repo, pub, prog := newTestUseCase()
	repo.projects["proj-1"] = &domain.Project{ProjectID: "proj-1", Status: domain.StatusRendering}

	err := uc.Execute(context.Background(), StepEvent{
		SagaID: "saga-1", ProjectID: "proj-1", EventType: "scene_rendered",
		Payload: map[string]interface{}{"scene_index": float64(2), "scene_total": float64(5)},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pub.published) != 0 {
		t.Fatalf("expected scene_rendered to never dispatch a command, got %d", len(pub.published))
	}
	last := prog.last()
	if last == nil || last.Status != "in_progress" || last.Step != string(domain.StepRenderScenes) {
		t.Fatalf("expected in_progress render_scenes progress message, got %+v", last)
	}
	if last.SceneIndex == nil || *last.SceneIndex != 2 || last.SceneTotal == nil || *last.SceneTotal != 5 {
		t.Fatalf("expected scene_index=2 scene_total=5, got %+v", last)
	}
	// state machine must not advance for a progress-only event
	project, _ := repo.Get(context.Background(), "proj-1")
	if project.Status != domain.StatusRendering {
		t.Fatalf("expected status unchanged by scene_rendered, got %s", project.Status)
	}
}

func TestHandleStepEventUseCase_Rule2_AudioPathFromStep3NotOverwritten(t *testing.T) {
	uc, repo, pub, _ := newTestUseCase()
	repo.projects["proj-1"] = &domain.Project{
		ProjectID: "proj-1",
		Status:    domain.StatusRendering,
		Scenes: []domain.Scene{
			{SceneIndex: 0, AudioPath: "from-step3.wav"},
		},
	}
	repo.steps[stepKey("saga-1", domain.StepRenderScenes)] = &domain.SagaStep{SagaID: "saga-1", StepName: domain.StepRenderScenes, Status: domain.SagaStepInProgress}

	err := uc.Execute(context.Background(), StepEvent{
		SagaID: "saga-1", ProjectID: "proj-1", EventType: "rendering_completed",
		Payload: map[string]interface{}{
			"scene_clip_paths": []interface{}{
				map[string]interface{}{"scene_index": float64(0), "clip_path": "clip0.mp4"},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	last := pub.last()
	if last == nil || last.routingKey != "video_assembly" {
		t.Fatalf("expected assemble_video dispatched, got %+v", last)
	}
	scenes, ok := last.envelope.Payload["scenes"].([]map[string]interface{})
	if !ok || len(scenes) != 1 {
		t.Fatalf("expected 1 scene in assemble_video payload, got %v", last.envelope.Payload["scenes"])
	}
	if scenes[0]["audio_path"] != "from-step3.wav" {
		t.Fatalf("expected audio_path sourced from step 3, got %v", scenes[0]["audio_path"])
	}
	if scenes[0]["clip_path"] != "clip0.mp4" {
		t.Fatalf("expected clip_path from rendering_completed, got %v", scenes[0]["clip_path"])
	}
}
