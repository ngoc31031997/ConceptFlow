package application

import (
	"context"
	"testing"
	"time"

	"orchestrator/internal/domain"
)

// withFastSynthesisRetryBackoff shrinks the module-level backoff schedule for
// the duration of a test, restoring the real one (15s/30s/60s) afterwards, so
// these tests don't actually sleep ~1m45s.
func withFastSynthesisRetryBackoff(t *testing.T) {
	t.Helper()
	original := synthesisAutoRetryBackoff
	synthesisAutoRetryBackoff = []time.Duration{time.Millisecond, time.Millisecond, time.Millisecond}
	t.Cleanup(func() { synthesisAutoRetryBackoff = original })
}

func TestHandleStepEventUseCase_SynthesisFailed_TransientRetriesWithoutFailingTheStep(t *testing.T) {
	withFastSynthesisRetryBackoff(t)
	uc, repo, pub, prog := newTestUseCase()
	repo.projects["proj-1"] = &domain.Project{ProjectID: "proj-1", Status: domain.StatusSynthesizingSpeech, VoiceID: "vi-VN-HoaiMyNeural"}
	repo.steps[stepKey("saga-retry-1", domain.StepSynthesizeSpeech)] = &domain.SagaStep{SagaID: "saga-retry-1", StepName: domain.StepSynthesizeSpeech, Status: domain.SagaStepInProgress}

	err := uc.Execute(context.Background(), StepEvent{
		SagaID: "saga-retry-1", ProjectID: "proj-1", EventType: "synthesis_failed",
		Payload: map[string]interface{}{"error_message": "tts_engine_failure: Edge TTS refused 4 attempts for voice vi-VN-HoaiMyNeural: No audio was received."},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	project, _ := repo.Get(context.Background(), "proj-1")
	if project.Status != domain.StatusSynthesizingSpeech {
		t.Fatalf("a transient failure must not change project status, got %s", project.Status)
	}
	step, _ := repo.GetStep(context.Background(), "saga-retry-1", domain.StepSynthesizeSpeech)
	if step.Status != domain.SagaStepInProgress {
		t.Fatalf("a transient failure must leave the step in_progress, got %s", step.Status)
	}
	if len(pub.published) != 1 || pub.published[0].routingKey != "tts" {
		t.Fatalf("expected one re-dispatched command to tts, got %+v", pub.published)
	}
	if prog.last() != nil {
		t.Fatalf("a silent retry must not publish a failed progress message, got %+v", prog.last())
	}
}

func TestHandleStepEventUseCase_SynthesisFailed_ExhaustsBudgetThenFailsTheStep(t *testing.T) {
	withFastSynthesisRetryBackoff(t)
	uc, repo, pub, prog := newTestUseCase()
	repo.projects["proj-1"] = &domain.Project{ProjectID: "proj-1", Status: domain.StatusSynthesizingSpeech, VoiceID: "vi-VN-HoaiMyNeural"}
	repo.steps[stepKey("saga-retry-2", domain.StepSynthesizeSpeech)] = &domain.SagaStep{SagaID: "saga-retry-2", StepName: domain.StepSynthesizeSpeech, Status: domain.SagaStepInProgress}

	errMsg := "tts_engine_failure: Edge TTS refused 4 attempts for voice vi-VN-HoaiMyNeural: No audio was received."
	event := StepEvent{
		SagaID: "saga-retry-2", ProjectID: "proj-1", EventType: "synthesis_failed",
		Payload: map[string]interface{}{"error_message": errMsg},
	}

	// synthesisAutoRetryBackoff has 3 entries: the first three failures retry
	// silently (each re-dispatch leaves the step in_progress), the fourth
	// exhausts the budget and must fail the step for real.
	for i := 0; i < len(synthesisAutoRetryBackoff); i++ {
		if err := uc.Execute(context.Background(), event); err != nil {
			t.Fatalf("attempt %d: unexpected error: %v", i+1, err)
		}
		step, _ := repo.GetStep(context.Background(), "saga-retry-2", domain.StepSynthesizeSpeech)
		if step.Status != domain.SagaStepInProgress {
			t.Fatalf("attempt %d: expected step still in_progress, got %s", i+1, step.Status)
		}
	}
	if len(pub.published) != len(synthesisAutoRetryBackoff) {
		t.Fatalf("expected %d re-dispatched commands, got %d", len(synthesisAutoRetryBackoff), len(pub.published))
	}

	if err := uc.Execute(context.Background(), event); err != nil {
		t.Fatalf("final attempt: unexpected error: %v", err)
	}
	project, _ := repo.Get(context.Background(), "proj-1")
	if project.Status != domain.StatusFailedSynthesizeSpeech {
		t.Fatalf("expected failed_at_synthesize_speech once the budget is exhausted, got %s", project.Status)
	}
	step, _ := repo.GetStep(context.Background(), "saga-retry-2", domain.StepSynthesizeSpeech)
	if step.Status != domain.SagaStepFailed {
		t.Fatalf("expected step failed once the budget is exhausted, got %s", step.Status)
	}
	if prog.last() == nil || prog.last().ErrorMessage == nil || *prog.last().ErrorMessage != errMsg {
		t.Fatalf("expected the error to finally surface in a progress message, got %+v", prog.last())
	}
	// No re-dispatch beyond the budget.
	if len(pub.published) != len(synthesisAutoRetryBackoff) {
		t.Fatalf("expected no extra dispatch once the budget is exhausted, got %d", len(pub.published))
	}
}

func TestHandleStepEventUseCase_SynthesisFailed_PermanentErrorsNeverRetry(t *testing.T) {
	withFastSynthesisRetryBackoff(t)
	uc, repo, pub, prog := newTestUseCase()
	repo.projects["proj-1"] = &domain.Project{ProjectID: "proj-1", Status: domain.StatusSynthesizingSpeech}
	repo.steps[stepKey("saga-retry-3", domain.StepSynthesizeSpeech)] = &domain.SagaStep{SagaID: "saga-retry-3", StepName: domain.StepSynthesizeSpeech, Status: domain.SagaStepInProgress}

	err := uc.Execute(context.Background(), StepEvent{
		SagaID: "saga-retry-3", ProjectID: "proj-1", EventType: "synthesis_failed",
		Payload: map[string]interface{}{"error_message": "empty_text: scene_index=2"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	project, _ := repo.Get(context.Background(), "proj-1")
	if project.Status != domain.StatusFailedSynthesizeSpeech {
		t.Fatalf("a permanent error must fail the step immediately, got %s", project.Status)
	}
	if len(pub.published) != 0 {
		t.Fatalf("a permanent error must never be auto-retried, got %+v", pub.published)
	}
	if prog.last() == nil || prog.last().ErrorMessage == nil || *prog.last().ErrorMessage != "empty_text: scene_index=2" {
		t.Fatalf("expected the permanent error surfaced immediately, got %+v", prog.last())
	}
}
