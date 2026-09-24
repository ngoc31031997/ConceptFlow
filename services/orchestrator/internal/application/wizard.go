package application

import (
	"context"
	"fmt"

	"orchestrator/internal/domain"
)

// WizardPort is the persistence the wizard's "Tiếp tục" saves need.
type WizardPort interface {
	GetStatus(ctx context.Context, projectID string) (domain.ProjectStatus, error)
	SaveWizardSettings(ctx context.Context, projectID string, s domain.WizardSettings) error
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

// SaveWizardSettingsUseCase persists wizard step 2 ("Cấu hình") when the
// Creator presses "Tiếp tục", and moves the project on to step 3.
type SaveWizardSettingsUseCase struct {
	repo WizardPort
}

func NewSaveWizardSettingsUseCase(repo WizardPort) *SaveWizardSettingsUseCase {
	return &SaveWizardSettingsUseCase{repo: repo}
}

// Execute validates s, fills defaults for omitted fields the way the render
// saga does, and stores it. A project past draft answers ErrInvalidStatus.
func (uc *SaveWizardSettingsUseCase) Execute(ctx context.Context, projectID string, s domain.WizardSettings) error {
	if s.ContentLanguage != domain.LanguageVietnamese && s.ContentLanguage != domain.LanguageEnglish {
		return fmt.Errorf("%w: voice_language must be 'vi' or 'en'", domain.ErrInvalidWizardInput)
	}
	if s.RenderEngine == "" {
		s.RenderEngine = domain.DefaultRenderEngine
	}
	if !s.RenderEngine.IsValid() {
		return fmt.Errorf("%w: render_engine must be 'manim' or 'remotion'", domain.ErrInvalidWizardInput)
	}
	if s.RenderQuality == "" {
		s.RenderQuality = domain.DefaultRenderQuality
	}
	if !s.RenderQuality.IsValid() {
		return fmt.Errorf("%w: unknown render_quality", domain.ErrInvalidWizardInput)
	}
	if s.VideoOutputMode == "" {
		s.VideoOutputMode = domain.DefaultVideoOutputMode
	}
	if !s.VideoOutputMode.IsValid() {
		return fmt.Errorf("%w: unknown video_output_mode", domain.ErrInvalidWizardInput)
	}
	if s.SubtitleMode == "" {
		s.SubtitleMode = domain.SubtitleModeOff
	}
	if !s.SubtitleMode.IsValid() {
		return fmt.Errorf("%w: unknown subtitle_mode", domain.ErrInvalidWizardInput)
	}
	if s.BackgroundMusicVolume < 0 || s.BackgroundMusicVolume > 1 {
		return fmt.Errorf("%w: background_music_volume must be between 0 and 1", domain.ErrInvalidWizardInput)
	}
	if !domain.ValidVideoFont(s.VideoFont) {
		return fmt.Errorf("%w: unknown video_font", domain.ErrInvalidWizardInput)
	}
	s.VideoFormatID = formatOrDefault(s.VideoFormatID)

	if err := requireDraft(ctx, uc.repo, projectID); err != nil {
		return err
	}
	return uc.repo.SaveWizardSettings(ctx, projectID, s)
}
