package postgres

import (
	"encoding/json"
	"testing"

	"orchestrator/internal/domain"
)

// These tests cover only the pure-Go mapping logic ProjectRepository relies
// on (JSON (de)serialization of Scene/status values) — no live Postgres
// connection is required. Integration coverage against a real database is
// deliberately left to the Build & Test stage (Step 12 of the code
// generation plan).

func TestScenesJSONRoundTrip(t *testing.T) {
	audioPath := "a.wav"
	scenes := []domain.Scene{
		{SceneIndex: 0, NarrationText: "n0", Category: "explainer", AudioPath: audioPath, DurationSeconds: 2.5},
		{SceneIndex: 1, NarrationText: "n1"},
	}

	raw, err := json.Marshal(scenes)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var decoded []domain.Scene
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if len(decoded) != 2 {
		t.Fatalf("expected 2 scenes, got %d", len(decoded))
	}
	if decoded[0].AudioPath != audioPath || decoded[0].DurationSeconds != 2.5 {
		t.Fatalf("scene 0 round-trip mismatch: %+v", decoded[0])
	}
}

func TestProjectStatusStringConversion(t *testing.T) {
	cases := []domain.ProjectStatus{
		domain.StatusDraft, domain.StatusParsingScript, domain.StatusRendering,
		domain.StatusFailedRenderScenes, domain.StatusPublished,
	}
	for _, status := range cases {
		roundTripped := domain.ProjectStatus(string(status))
		if roundTripped != status {
			t.Fatalf("status string round-trip mismatch: %s != %s", roundTripped, status)
		}
	}
}

func TestSagaStepStatusStringConversion(t *testing.T) {
	for _, status := range []domain.SagaStepStatus{domain.SagaStepInProgress, domain.SagaStepCompleted, domain.SagaStepFailed} {
		if domain.SagaStepStatus(string(status)) != status {
			t.Fatalf("saga step status round-trip mismatch for %s", status)
		}
	}
}

func TestVoiceLanguageAndVisibilityRoundTrip(t *testing.T) {
	if domain.ContentLanguage(string(domain.LanguageVietnamese)) != domain.LanguageVietnamese {
		t.Fatal("voice_language round-trip mismatch")
	}
	if domain.Visibility(string(domain.VisibilityUnlisted)) != domain.VisibilityUnlisted {
		t.Fatal("visibility round-trip mismatch")
	}
}
