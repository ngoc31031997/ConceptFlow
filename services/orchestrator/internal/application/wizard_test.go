package application_test

import (
	"context"
	"errors"
	"testing"

	"orchestrator/internal/application"
	"orchestrator/internal/domain"
)

type fakeWizardRepo struct {
	status   domain.ProjectStatus
	settings *domain.WizardSettings
	step     int
}

func (f *fakeWizardRepo) GetStatus(context.Context, string) (domain.ProjectStatus, error) {
	return f.status, nil
}
func (f *fakeWizardRepo) SaveWizardSettings(_ context.Context, _ string, s domain.WizardSettings) error {
	f.settings = &s
	f.step = domain.WizardStepScript
	return nil
}

func TestSaveWizardSettings_DefaultsAndAdvancesToScript(t *testing.T) {
	repo := &fakeWizardRepo{status: domain.StatusDraft}
	uc := application.NewSaveWizardSettingsUseCase(repo)

	err := uc.Execute(context.Background(), "p1", domain.WizardSettings{ContentLanguage: domain.LanguageVietnamese, TTSEnabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if repo.settings.RenderQuality != domain.DefaultRenderQuality || repo.settings.RenderEngine != domain.DefaultRenderEngine ||
		repo.settings.SubtitleMode != domain.SubtitleModeOff || repo.settings.VideoFormatID != domain.DefaultVideoFormatID {
		t.Errorf("defaults not applied: %+v", repo.settings)
	}
	if repo.step != domain.WizardStepScript {
		t.Errorf("step = %d, want %d", repo.step, domain.WizardStepScript)
	}
}

func TestSaveWizardSettings_RejectsBadInputAndStartedProjects(t *testing.T) {
	uc := application.NewSaveWizardSettingsUseCase(&fakeWizardRepo{status: domain.StatusDraft})
	err := uc.Execute(context.Background(), "p1", domain.WizardSettings{ContentLanguage: "fr"})
	if !errors.Is(err, domain.ErrInvalidWizardInput) {
		t.Errorf("bad language: err = %v", err)
	}
	err = uc.Execute(context.Background(), "p1", domain.WizardSettings{ContentLanguage: "vi", RenderQuality: "8k"})
	if !errors.Is(err, domain.ErrInvalidWizardInput) {
		t.Errorf("bad quality: err = %v", err)
	}

	started := application.NewSaveWizardSettingsUseCase(&fakeWizardRepo{status: domain.StatusRendering})
	err = started.Execute(context.Background(), "p1", domain.WizardSettings{ContentLanguage: "vi"})
	if !errors.Is(err, domain.ErrInvalidStatus) {
		t.Errorf("started project: err = %v, want ErrInvalidStatus", err)
	}
}

func TestEffectiveWizardStep(t *testing.T) {
	cases := []struct {
		status domain.ProjectStatus
		stored int
		want   int
	}{
		{domain.StatusDraft, 0, 1},
		{domain.StatusDraft, 3, 3},
		{domain.StatusAwaitingReview, 3, 4},
		{domain.StatusRendering, 3, 5},
		{domain.StatusFailedRenderScenes, 3, 5},
		{domain.StatusReadyToPublish, 3, 6},
		{domain.StatusPublished, 3, 7},
	}
	for _, c := range cases {
		if got := domain.EffectiveWizardStep(&domain.Project{Status: c.status, WizardStep: c.stored}); got != c.want {
			t.Errorf("%s stored=%d: got %d, want %d", c.status, c.stored, got, c.want)
		}
	}
}
