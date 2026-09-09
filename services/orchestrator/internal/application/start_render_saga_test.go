package application

import (
	"context"
	"testing"

	"orchestrator/internal/domain"
)

func TestStartRenderSagaUseCase_Execute(t *testing.T) {
	repo := newFakeRepo()
	pub := &fakePublisher{}
	uc := NewStartRenderSagaUseCase(repo, pub)

	musicPath := "music.mp3"
	out, err := uc.Execute(context.Background(), StartRenderSagaInput{
		ProjectID:           "proj-1",
		ScriptContent:       "some script",
		PluginID:            "plugin-a",
		ContentLanguage:     domain.LanguageVietnamese,
		BackgroundMusicPath: &musicPath,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.SagaID == "" {
		t.Fatal("expected a generated saga_id")
	}
	if out.Status != domain.StatusParsingScript {
		t.Fatalf("expected status parsing_script, got %s", out.Status)
	}

	project, err := repo.Get(context.Background(), "proj-1")
	if err != nil {
		t.Fatalf("expected project to be saved: %v", err)
	}
	if project.Status != domain.StatusParsingScript {
		t.Fatalf("expected saved project status parsing_script, got %s", project.Status)
	}
	if project.BackgroundMusicPath == nil || *project.BackgroundMusicPath != musicPath {
		t.Fatal("expected background_music_path to be persisted (Rule 3)")
	}

	step, err := repo.GetStep(context.Background(), out.SagaID, domain.StepParseScript)
	if err != nil {
		t.Fatalf("expected parse_script saga step to exist: %v", err)
	}
	if step.Status != domain.SagaStepInProgress {
		t.Fatalf("expected parse_script step in_progress, got %s", step.Status)
	}

	last := pub.last()
	if last == nil {
		t.Fatal("expected a command to be published")
	}
	if last.routingKey != "script_processing" {
		t.Fatalf("expected routing key script_processing, got %s", last.routingKey)
	}
	if last.envelope.Payload["script_content"] != "some script" {
		t.Fatalf("expected script_content in payload, got %v", last.envelope.Payload)
	}
}

// TestStartRenderSagaUseCase_SubtitleModeIsPersisted is CR-015 FR41: the
// Creator's explicit choice among the four delivery modes must survive into
// the saved Project, for handle_step_event.go's assembleVideoPayload to
// pick up later.
func TestStartRenderSagaUseCase_SubtitleModeIsPersisted(t *testing.T) {
	repo := newFakeRepo()
	uc := NewStartRenderSagaUseCase(repo, &fakePublisher{})

	if _, err := uc.Execute(context.Background(), StartRenderSagaInput{
		ProjectID:       "proj-1",
		ScriptContent:   "script",
		ContentLanguage: domain.LanguageVietnamese,
		SubtitleMode:    domain.SubtitleModeTrack,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	project, _ := repo.Get(context.Background(), "proj-1")
	if project.SubtitleMode != domain.SubtitleModeTrack {
		t.Fatalf("expected SubtitleMode track, got %q", project.SubtitleMode)
	}
	if !project.SubtitlesEnabled {
		t.Fatal("expected legacy SubtitlesEnabled to mirror NeedsCues() for a non-off mode")
	}
}

// TestStartRenderSagaUseCase_FallsBackToLegacySubtitlesEnabled covers a
// caller that has not adopted subtitle_mode yet (an older client, or a
// direct API integration) — it must reproduce exactly the one behaviour
// SubtitlesEnabled ever meant (burn-in), not silently default to track.
func TestStartRenderSagaUseCase_FallsBackToLegacySubtitlesEnabled(t *testing.T) {
	repo := newFakeRepo()
	uc := NewStartRenderSagaUseCase(repo, &fakePublisher{})

	if _, err := uc.Execute(context.Background(), StartRenderSagaInput{
		ProjectID:        "proj-1",
		ScriptContent:    "script",
		ContentLanguage:  domain.LanguageVietnamese,
		SubtitlesEnabled: true,
		// SubtitleMode deliberately left unset.
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	project, _ := repo.Get(context.Background(), "proj-1")
	if project.SubtitleMode != domain.SubtitleModeBurnIn {
		t.Fatalf("expected SubtitleMode burn_in from the legacy boolean, got %q", project.SubtitleMode)
	}
}

// TestStartRenderSagaUseCase_SubtitleModeOffClearsLegacyFlag guards the
// other direction: an explicit "off" must not leave SubtitlesEnabled true
// from some stale caller-supplied value.
func TestStartRenderSagaUseCase_SubtitleModeOffClearsLegacyFlag(t *testing.T) {
	repo := newFakeRepo()
	uc := NewStartRenderSagaUseCase(repo, &fakePublisher{})

	if _, err := uc.Execute(context.Background(), StartRenderSagaInput{
		ProjectID:        "proj-1",
		ScriptContent:    "script",
		ContentLanguage:  domain.LanguageVietnamese,
		SubtitlesEnabled: true,
		SubtitleMode:     domain.SubtitleModeOff,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	project, _ := repo.Get(context.Background(), "proj-1")
	if project.SubtitlesEnabled {
		t.Fatal("expected SubtitlesEnabled false when the explicit mode is off")
	}
}

func TestStartRenderSagaUseCase_PublishFailure(t *testing.T) {
	repo := newFakeRepo()
	pub := &fakePublisher{failNext: true}
	uc := NewStartRenderSagaUseCase(repo, pub)

	_, err := uc.Execute(context.Background(), StartRenderSagaInput{
		ProjectID:       "proj-err",
		ScriptContent:   "x",
		PluginID:        "p",
		ContentLanguage: domain.LanguageEnglish,
	})
	if err == nil {
		t.Fatal("expected error when publish fails")
	}
}
