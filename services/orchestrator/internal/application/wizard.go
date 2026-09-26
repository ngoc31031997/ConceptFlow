package application

import (
	"context"
	"fmt"

	"orchestrator/internal/domain"
)

// WizardPort is the persistence the wizard's step 2 saves need.
type WizardPort interface {
	GetStatus(ctx context.Context, projectID string) (domain.ProjectStatus, error)
	PatchWizardSettings(ctx context.Context, projectID string, p domain.WizardSettingsPatch) error
}

// requireDraft is the same lock the authoring saves use (CR-028 FR84.2): once
// the saga has started, the wizard's inputs must not change under a render.
func requireDraft(ctx context.Context, repo WizardPort, projectID string) error {
	if projectID == "" {
		return fmt.Errorf("%w: project_id is required", domain.ErrInvalidWizardInput)
	}
	status, err := repo.GetStatus(ctx, projectID)
	if err != nil {
		return err
	}
	if !domain.IsAuthoringEditable(status) {
		return domain.ErrInvalidStatus
	}
	return nil
}

// PatchWizardSettingsUseCase persists wizard step 2 ("Cấu hình") field by
// field as the Creator changes it; a patch with Confirm set (the "Tiếp tục"
// press) also moves the project on to step 3.
type PatchWizardSettingsUseCase struct {
	repo WizardPort
}

func NewPatchWizardSettingsUseCase(repo WizardPort) *PatchWizardSettingsUseCase {
	return &PatchWizardSettingsUseCase{repo: repo}
}

// Execute validates only the fields present in p and stores them. A project
// past draft answers ErrInvalidStatus.
func (uc *PatchWizardSettingsUseCase) Execute(ctx context.Context, projectID string, p domain.WizardSettingsPatch) error {
	if p.ContentLanguage != nil && *p.ContentLanguage != domain.LanguageVietnamese && *p.ContentLanguage != domain.LanguageEnglish {
		return fmt.Errorf("%w: voice_language must be 'vi' or 'en'", domain.ErrInvalidWizardInput)
	}
	if p.RenderEngine != nil && !p.RenderEngine.IsValid() {
		return fmt.Errorf("%w: render_engine must be 'manim' or 'remotion'", domain.ErrInvalidWizardInput)
	}
	if p.RenderQuality != nil && !p.RenderQuality.IsValid() {
		return fmt.Errorf("%w: unknown render_quality", domain.ErrInvalidWizardInput)
	}
	if p.VideoOutputMode != nil && !p.VideoOutputMode.IsValid() {
		return fmt.Errorf("%w: unknown video_output_mode", domain.ErrInvalidWizardInput)
	}
	if p.SubtitleMode != nil && !p.SubtitleMode.IsValid() {
		return fmt.Errorf("%w: unknown subtitle_mode", domain.ErrInvalidWizardInput)
	}
	if p.BackgroundMusicVolume != nil && (*p.BackgroundMusicVolume < 0 || *p.BackgroundMusicVolume > 1) {
		return fmt.Errorf("%w: background_music_volume must be between 0 and 1", domain.ErrInvalidWizardInput)
	}
	if p.VideoFont != nil && !domain.ValidVideoFont(*p.VideoFont) {
		return fmt.Errorf("%w: unknown video_font", domain.ErrInvalidWizardInput)
	}
	if p.VideoFormatID != nil {
		f := formatOrDefault(*p.VideoFormatID)
		p.VideoFormatID = &f
	}

	if err := requireDraft(ctx, uc.repo, projectID); err != nil {
		return err
	}
	return uc.repo.PatchWizardSettings(ctx, projectID, p)
}
