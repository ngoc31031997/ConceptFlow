package application_test

import (
	"context"
	"errors"
	"testing"

	"orchestrator/internal/application"
	"orchestrator/internal/domain"
)

type fakeWizardRepo struct {
	status domain.ProjectStatus
	patch  *domain.WizardSettingsPatch
}

func (f *fakeWizardRepo) GetStatus(context.Context, string) (domain.ProjectStatus, error) {
	return f.status, nil
}
func (f *fakeWizardRepo) PatchWizardSettings(_ context.Context, _ string, p domain.WizardSettingsPatch) error {
	f.patch = &p
	return nil
}

func TestPatchWizardSettings_PassesOnlyGivenFields(t *testing.T) {
	repo := &fakeWizardRepo{status: domain.StatusDraft}
	uc := application.NewPatchWizardSettingsUseCase(repo)
	q := domain.RenderQuality("720p30")

	if err := uc.Execute(context.Background(), "p1", domain.WizardSettingsPatch{RenderQuality: &q}); err != nil {
		t.Fatal(err)
	}
	if repo.patch.RenderQuality == nil || *repo.patch.RenderQuality != q {
		t.Errorf("quality not passed: %+v", repo.patch)
	}
	if repo.patch.ContentLanguage != nil || repo.patch.RenderEngine != nil || repo.patch.VideoFormatID != nil || repo.patch.Confirm {
		t.Errorf("absent fields must stay nil: %+v", repo.patch)
	}
}

func TestPatchWizardSettings_RejectsBadInputAndStartedProjects(t *testing.T) {
	uc := application.NewPatchWizardSettingsUseCase(&fakeWizardRepo{status: domain.StatusDraft})
	fr := domain.ContentLanguage("fr")
	err := uc.Execute(context.Background(), "p1", domain.WizardSettingsPatch{ContentLanguage: &fr})
	if !errors.Is(err, domain.ErrInvalidWizardInput) {
		t.Errorf("bad language: err = %v", err)
	}
	bad := domain.RenderQuality("8k")
	err = uc.Execute(context.Background(), "p1", domain.WizardSettingsPatch{RenderQuality: &bad})
	if !errors.Is(err, domain.ErrInvalidWizardInput) {
		t.Errorf("bad quality: err = %v", err)
	}
	vol := 1.5
	err = uc.Execute(context.Background(), "p1", domain.WizardSettingsPatch{BackgroundMusicVolume: &vol})
	if !errors.Is(err, domain.ErrInvalidWizardInput) {
		t.Errorf("bad volume: err = %v", err)
	}

	started := application.NewPatchWizardSettingsUseCase(&fakeWizardRepo{status: domain.StatusRendering})
	vi := domain.LanguageVietnamese
	err = started.Execute(context.Background(), "p1", domain.WizardSettingsPatch{ContentLanguage: &vi})
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
