package application

import (
	"context"
	"errors"
	"testing"

	"orchestrator/internal/domain"
)

type forkRepo struct {
	source   *domain.Project
	saved    *domain.Project
	settings *domain.WizardSettings
	from     string
}

func (r *forkRepo) Get(context.Context, string) (*domain.Project, error) { return r.source, nil }
func (r *forkRepo) Save(_ context.Context, p *domain.Project) error      { r.saved = p; return nil }
func (r *forkRepo) SaveWizardSettings(_ context.Context, _ string, s domain.WizardSettings) error {
	r.settings = &s
	return nil
}
func (r *forkRepo) SetForkedFrom(_ context.Context, _, from string) error { r.from = from; return nil }

type forkAuthoring struct {
	topic, story, board, code string
	saved                     map[string]string
}

func (a *forkAuthoring) GetAuthoringTopic(context.Context, string) (string, error) {
	return a.topic, nil
}
func (a *forkAuthoring) GetAuthoringMode(context.Context, string) (string, error) { return "ai", nil }
func (a *forkAuthoring) GetAuthoringStory(context.Context, string) (string, error) {
	return a.story, nil
}
func (a *forkAuthoring) GetAuthoringStoryboard(context.Context, string) (string, error) {
	return a.board, nil
}
func (a *forkAuthoring) GetAuthoringCode(context.Context, string) (string, error) { return a.code, nil }
func (a *forkAuthoring) GetAuthoringModels(context.Context, string) (domain.AuthoringStepModels, error) {
	return domain.AuthoringStepModels{}, nil
}
func (a *forkAuthoring) SaveAuthoringTopic(_ context.Context, _, t string, _ domain.ContentLanguage) error {
	a.saved["topic"] = t
	return nil
}
func (a *forkAuthoring) SaveAuthoringStory(_ context.Context, _, c, _ string) error {
	a.saved["story"] = c
	return nil
}
func (a *forkAuthoring) SaveAuthoringStoryboard(_ context.Context, _, c string) error {
	a.saved["storyboard"] = c
	return nil
}
func (a *forkAuthoring) SaveAuthoringMode(_ context.Context, _, m string) error {
	a.saved["mode"] = m
	return nil
}
func (a *forkAuthoring) SaveAuthoringModels(context.Context, string, domain.AuthoringStepModels) error {
	return nil
}

func rendered() *domain.Project {
	music := "/shared/src/music/a.mp3"
	url := "https://youtu.be/x"
	video := "/shared/src/video.mp4"
	return &domain.Project{
		ProjectID: "src", SagaID: "saga-src", Status: domain.StatusReadyToPublish,
		ContentLanguage: domain.LanguageVietnamese, VoiceID: "vi-voice", RenderQuality: domain.DefaultRenderQuality,
		ScriptContent: "class X: pass", Scenes: []domain.Scene{{SceneIndex: 0}},
		VideoPath: &video, YoutubeVideoURL: &url, BackgroundMusicPath: &music, BackgroundMusicVolume: 0.3,
	}
}

func TestForkKeepsOnlyWhatComesBeforeTheChosenStep(t *testing.T) {
	cases := []struct {
		from                          int
		wantStory, wantBoard, wantCfg bool
	}{
		{domain.FlowConfig, false, false, false},
		{domain.FlowStory, false, false, true},
		{domain.FlowVisual, true, false, true},
		{domain.FlowCode, true, true, true},
	}
	for _, c := range cases {
		repo := &forkRepo{source: rendered()}
		auth := &forkAuthoring{topic: "topic", story: "STORY", board: "BOARD", code: "CODE", saved: map[string]string{}}
		out, err := NewForkProjectUseCase(repo, auth).Execute(context.Background(), "src", c.from)
		if err != nil {
			t.Fatalf("from %d: %v", c.from, err)
		}
		if out.ProjectID == "" || out.ProjectID == "src" || repo.saved.ProjectID != out.ProjectID || repo.saved.SagaID == "saga-src" {
			t.Errorf("from %d: fork must be a new project with its own saga, got %+v", c.from, repo.saved)
		}
		if repo.from != "src" {
			t.Errorf("from %d: forked_from = %q", c.from, repo.from)
		}
		if auth.saved["topic"] != "topic" {
			t.Errorf("from %d: topic not kept", c.from)
		}
		if _, ok := auth.saved["story"]; ok != c.wantStory {
			t.Errorf("from %d: story kept = %v, want %v", c.from, ok, c.wantStory)
		}
		if _, ok := auth.saved["storyboard"]; ok != c.wantBoard {
			t.Errorf("from %d: storyboard kept = %v, want %v", c.from, ok, c.wantBoard)
		}
		if (repo.settings != nil) != c.wantCfg {
			t.Errorf("from %d: settings marked done = %v, want %v", c.from, repo.settings != nil, c.wantCfg)
		}
		if _, ok := auth.saved["code"]; ok {
			t.Errorf("from %d: code must never be carried over", c.from)
		}
	}
}

func TestForkDropsEverythingTheRenderProduced(t *testing.T) {
	repo := &forkRepo{source: rendered()}
	auth := &forkAuthoring{saved: map[string]string{}}
	out, err := NewForkProjectUseCase(repo, auth).Execute(context.Background(), "src", domain.FlowCode)
	if err != nil {
		t.Fatal(err)
	}
	p := repo.saved
	if p.Status != domain.StatusDraft || p.ScriptContent != "" || len(p.Scenes) != 0 || p.VideoPath != nil || p.YoutubeVideoURL != nil {
		t.Errorf("render output leaked into the fork: %+v", p)
	}
	if p.VoiceID != "vi-voice" || p.ContentLanguage != domain.LanguageVietnamese {
		t.Errorf("settings must be copied: %+v", p)
	}
	if p.BackgroundMusicPath != nil || !out.NeedsMusicReselect {
		t.Errorf("music file belongs to the source's folder: path=%v reselect=%v", p.BackgroundMusicPath, out.NeedsMusicReselect)
	}
	if repo.source.VideoPath == nil || repo.source.Status != domain.StatusReadyToPublish {
		t.Error("the original must not be touched")
	}
}

func TestForkRejectsStepsItCannotStartAt(t *testing.T) {
	for _, step := range []int{0, 1, 6, 13} {
		if _, err := NewForkProjectUseCase(&forkRepo{source: rendered()}, &forkAuthoring{saved: map[string]string{}}).Execute(context.Background(), "src", step); !errors.Is(err, ErrForkStepInvalid) {
			t.Errorf("step %d: got %v", step, err)
		}
	}
}
