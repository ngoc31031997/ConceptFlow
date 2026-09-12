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

func TestStartRenderSagaUseCase_VideoOutputMode_DefaultsToLong(t *testing.T) {
	repo := newFakeRepo()
	uc := NewStartRenderSagaUseCase(repo, &fakePublisher{})

	if _, err := uc.Execute(context.Background(), StartRenderSagaInput{
		ProjectID:       "proj-1",
		ScriptContent:   "script",
		ContentLanguage: domain.LanguageVietnamese,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	project, _ := repo.Get(context.Background(), "proj-1")
	if project.VideoOutputMode != domain.ModeLongOnly {
		t.Fatalf("expected default video_output_mode=long, got %s", project.VideoOutputMode)
	}
}

func TestStartRenderSagaUseCase_VideoOutputMode_ExplicitChoiceWins(t *testing.T) {
	repo := newFakeRepo()
	uc := NewStartRenderSagaUseCase(repo, &fakePublisher{})

	if _, err := uc.Execute(context.Background(), StartRenderSagaInput{
		ProjectID:       "proj-1",
		ScriptContent:   "script",
		ContentLanguage: domain.LanguageVietnamese,
		VideoOutputMode: domain.ModeShortOnly,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	project, _ := repo.Get(context.Background(), "proj-1")
	if project.VideoOutputMode != domain.ModeShortOnly {
		t.Fatalf("expected video_output_mode=short, got %s", project.VideoOutputMode)
	}
}

func TestStartRenderSagaUseCase_CompanionProjectID_LinksBothWays(t *testing.T) {
	repo := newFakeRepo()
	uc := NewStartRenderSagaUseCase(repo, &fakePublisher{})

	// The long-form project already exists (as if rendered earlier).
	if _, err := uc.Execute(context.Background(), StartRenderSagaInput{
		ProjectID:       "proj-long",
		ScriptContent:   "script long",
		ContentLanguage: domain.LanguageVietnamese,
	}); err != nil {
		t.Fatalf("unexpected error creating the long project: %v", err)
	}

	companionID := "proj-long"
	if _, err := uc.Execute(context.Background(), StartRenderSagaInput{
		ProjectID:          "proj-short",
		ScriptContent:      "script short",
		ContentLanguage:    domain.LanguageVietnamese,
		VideoOutputMode:    domain.ModeShortOnly,
		CompanionProjectID: &companionID,
	}); err != nil {
		t.Fatalf("unexpected error creating the short project: %v", err)
	}

	short, _ := repo.Get(context.Background(), "proj-short")
	if short.CompanionProjectID == nil || *short.CompanionProjectID != "proj-long" {
		t.Fatalf("expected proj-short.CompanionProjectID = proj-long, got %v", short.CompanionProjectID)
	}

	long, _ := repo.Get(context.Background(), "proj-long")
	if long.CompanionProjectID == nil || *long.CompanionProjectID != "proj-short" {
		t.Fatalf("expected proj-long.CompanionProjectID linked back to proj-short, got %v", long.CompanionProjectID)
	}
}

func TestStartRenderSagaUseCase_CompanionProjectID_MissingCompanionIsBestEffort(t *testing.T) {
	// CR-026 D1: a stale/wrong companion id must never cost the Creator the
	// video they are actually here to create.
	repo := newFakeRepo()
	uc := NewStartRenderSagaUseCase(repo, &fakePublisher{})

	companionID := "does-not-exist"
	out, err := uc.Execute(context.Background(), StartRenderSagaInput{
		ProjectID:          "proj-short",
		ScriptContent:      "script short",
		ContentLanguage:    domain.LanguageVietnamese,
		CompanionProjectID: &companionID,
	})
	if err != nil {
		t.Fatalf("expected the render to still start despite a missing companion, got error: %v", err)
	}
	if out.Status != domain.StatusParsingScript {
		t.Fatalf("expected status parsing_script, got %s", out.Status)
	}

	short, _ := repo.Get(context.Background(), "proj-short")
	if short.CompanionProjectID == nil || *short.CompanionProjectID != "does-not-exist" {
		t.Fatalf("expected proj-short to still record its own companion_project_id, got %v", short.CompanionProjectID)
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
