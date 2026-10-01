package application

import (
	"context"
	"fmt"

	"orchestrator/internal/domain"
)

// WizardPort is the persistence the wizard's step 2 saves need.
type WizardPort interface {
	GetStatus(ctx context.Context, projectID string) (domain.ProjectStatus, error)
	Get(ctx context.Context, projectID string) (*domain.Project, error)
	GetVideoFormat(ctx context.Context, formatID string, version int) (domain.VideoFormat, error)
	PatchWizardSettings(ctx context.Context, projectID string, p domain.WizardSettingsPatch) error
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

// Execute validates only the fields present in p and stores them. A field
// whose reading step has already run answers ErrInvalidStatus
// (domain.WizardPatchAllowed): render settings stay editable at the review gate
// and after a failure up to render_scenes, merge settings up to assemble_video.
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
	domain.ApplyShortDefaults(&p)
	if p.VideoFormatID != nil {
		f := formatOrDefault(*p.VideoFormatID)
		p.VideoFormatID = &f
	}

	if projectID == "" {
		return fmt.Errorf("%w: project_id is required", domain.ErrInvalidWizardInput)
	}
	status, err := uc.repo.GetStatus(ctx, projectID)
	if err != nil {
		return err
	}
	if !domain.WizardPatchAllowed(status, p) {
		return domain.ErrInvalidStatus
	}
	if p.Confirm {
		if err := uc.checkOutput(ctx, projectID, p); err != nil {
			return err
		}
	}
	return uc.repo.PatchWizardSettings(ctx, projectID, p)
}

// checkOutput holds the settings the project will have once p is saved to
// domain.CheckOutputSettings. Fields are saved one by one as the Creator
// edits them, so the combination is only judged when step 2 is confirmed.
func (uc *PatchWizardSettingsUseCase) checkOutput(ctx context.Context, projectID string, p domain.WizardSettingsPatch) error {
	project, err := uc.repo.Get(ctx, projectID)
	if err != nil {
		return err
	}
	mode, formatID, engine := project.VideoOutputMode, project.VideoFormatID, project.RenderEngine
	if p.VideoOutputMode != nil {
		mode = *p.VideoOutputMode
	}
	if p.VideoFormatID != nil {
		formatID = *p.VideoFormatID
	}
	if p.RenderEngine != nil {
		engine = *p.RenderEngine
	}
	if engine == "" {
		engine = domain.DefaultRenderEngine
	}
	format, err := uc.repo.GetVideoFormat(ctx, formatOrDefault(formatID), 0)
	if err != nil {
		return fmt.Errorf("load video format: %w", err)
	}
	return domain.CheckOutputSettings(mode, format, engine)
}
