package application

import (
	"context"
	"fmt"
	"log/slog"

	"authoring/internal/domain"
)

// WithStoryboardChecks enables the checks that run after step 1b succeeds:
// the project's format supplies the per-beat budgets and the
// voice calibration the speaking rate — the same two sources the outline
// prompt's beat sheet is built from. Nil formats disables the length check;
// nil calibration uses the language default rate.
func (uc *GenerateAuthoringUseCase) WithStoryboardChecks(formats FormatLookupPort, calibration VoiceCalibrationPort) *GenerateAuthoringUseCase {
	uc.formats, uc.calibration = formats, calibration
	return uc
}

// storyboardWarnings runs the non-blocking checks on a finalized storyboard
// and returns what the Creator should look at before paying for
// the code step. Nothing here can fail the run: a check that cannot run says so
// as a warning instead of staying silent.
func (uc *GenerateAuthoringUseCase) storyboardWarnings(ctx context.Context, project *domain.Project, storyboard string) []string {
	language := string(project.ContentLanguage)
	if language != "vi" && language != "en" {
		language = "vi" // same fallback as the prompt renderer
	}
	// No format on the project means the outline was written without a
	// beat sheet, so there is no budget to hold the storyboard to.
	checkLength := uc.formats != nil && project.VideoFormatID != ""
	// The kit and its keyword table are Vietnamese and Remotion-only.
	checkIllustrated := project.RenderEngine == domain.RenderEngineRemotion && language == "vi"
	if !checkLength && !checkIllustrated {
		return nil
	}

	scenes, err := domain.ParseStoryboardScenes(storyboard)
	if err != nil {
		return []string{fmt.Sprintf("Không kiểm được độ dài lời thoại và hình minh hoạ của storyboard: %v", err)}
	}

	var warnings []string
	if checkLength {
		format, err := uc.formats.GetVideoFormat(ctx, project.VideoFormatID, project.VideoFormatVersion)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf(
				"Không kiểm được độ dài lời thoại: không tải được format %q (v%d): %v",
				project.VideoFormatID, project.VideoFormatVersion, err))
		} else {
			warnings = append(warnings, domain.CheckNarrationBudgets(scenes, format, language, uc.voiceWPM(ctx, project.VoiceID))...)
		}
	}
	if checkIllustrated {
		warnings = append(warnings, domain.CheckIllustratedNarration(scenes)...)
		warnings = append(warnings, domain.CheckSceneSettings(scenes)...)
	}
	if len(warnings) > 0 {
		slog.Info("storyboard warnings", "project_id", project.ProjectID, "count", len(warnings))
	}
	return warnings
}

// voiceWPM is the voice's measured rate, or 0 for the language default — the
// rule beatsFor applies when it renders the outline's beat sheet.
func (uc *GenerateAuthoringUseCase) voiceWPM(ctx context.Context, voiceID string) float64 {
	if uc.calibration == nil || voiceID == "" {
		return 0
	}
	c, err := uc.calibration.GetVoiceCalibration(ctx, voiceID)
	if err != nil {
		return 0
	}
	if measured, ok := c.WordsPerMinute(); ok {
		return measured
	}
	return 0
}
