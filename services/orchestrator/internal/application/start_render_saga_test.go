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
