package application

import (
	"context"
	"fmt"
	"strings"

	"authoring/internal/domain"
)

// RenderInput is everything a stateless prompt render needs (CR-040 FR113).
//
// The wizard and the assistants hold unsaved drafts — the topic being typed,
// the script being pasted, the subtitle style not yet applied — so they send
// what they have instead of asking the server to read it back from a project
// row that may be stale. Every field is optional; a blank one becomes the
// placeholder the browser used to show, so a prompt rendered from nothing reads
// exactly as it did before this moved to the server.
type RenderInput struct {
	Role     domain.PromptRole
	Language string // "vi" | "en"; anything else is "vi", as elsewhere

	Topic          string
	Script         string // the Creator's existing code, for the *_adjust roles
	PreviousOutput string // earlier steps' artefacts, already joined

	SubtitleMode     string // off | track | burn_in | both
	SubtitleFontSize string // small | medium | large
	SubtitlePosition string // top | bottom

	// FormatID/FormatVersion/VoiceID feed {{format_beats}} (story_architect only).
	FormatID      string
	FormatVersion int
	VoiceID       string
}

const (
	manimScriptPlaceholder    = "<dán script Manim của bạn vào đây>"
	remotionScriptPlaceholder = "<dán code Remotion của bạn vào đây>"
	thumbnailTopicPlaceholder = "[DÁN CHỦ ĐỀ VIDEO CỦA BẠN VÀO ĐÂY]"

	visualDirectorEmptyPrevious = "(chưa có dàn ý câu chuyện đã lưu ở bước 3)"
	engineerEmptyPrevious       = "(chưa có dàn ý/storyboard đã lưu ở các bước trước)"
)

// isRemotionRole reports whether role writes/repairs Remotion code, which
// changes what the narration rule has to say.
func isRemotionRole(role domain.PromptRole) bool {
	return role == domain.RoleRemotionEngineer || role == domain.RoleRemotionEngineerAI || role == domain.RoleRemotionAdjust
}

// Render fills the role's active library prompt with the input.
//
// Substitution is a single pass (strings.Replacer), so text the Creator pasted
// in — a script that happens to contain "{{topic}}" — is never itself expanded.
func (uc *RenderPromptUseCase) Render(ctx context.Context, in RenderInput) (RenderedPrompt, error) {
	if !domain.ValidPromptRole(string(in.Role)) {
		return RenderedPrompt{}, fmt.Errorf("unknown role %q", in.Role)
	}
	language := in.Language
	if language != "vi" && language != "en" {
		language = "vi"
	}
	effective, err := uc.prompts.GetActive(ctx, in.Role)
	if err != nil {
		return RenderedPrompt{}, fmt.Errorf("load template: %w", err)
	}

	engine := "manim"
	if isRemotionRole(in.Role) {
		engine = "remotion"
	}

	topic := strings.TrimSpace(in.Topic)
	if topic == "" {
		topic = TopicPlaceholder
		if in.Role == domain.RoleThumbnailDesign {
			topic = thumbnailTopicPlaceholder
		}
	}
	script := strings.TrimSpace(in.Script)
	if script == "" {
		script = manimScriptPlaceholder
		if in.Role == domain.RoleRemotionAdjust {
			script = remotionScriptPlaceholder
		}
	}
	previous := in.PreviousOutput
	if strings.TrimSpace(previous) == "" {
		previous = engineerEmptyPrevious
		if in.Role == domain.RoleVisualDirector || in.Role == domain.RoleVisualDirectorAI {
			previous = visualDirectorEmptyPrevious
		}
	}

	style := domain.DefaultSubtitleStyle()
	if in.SubtitleFontSize != "" {
		style.FontSize = in.SubtitleFontSize
	}
	if in.SubtitlePosition != "" {
		style.Position = in.SubtitlePosition
	}

	beats := ""
	if in.Role == domain.RoleStoryArchitect && in.FormatID != "" {
		beats = uc.beatsFor(ctx, in, language)
	}

	r := strings.NewReplacer(
		"{{topic}}", topic,
		"{{script}}", script,
		"{{previous_output}}", previous,
		"{{channel_identity}}", domain.ChannelIdentity(language),
		"{{narration_language_rule}}", domain.NarrationLanguageRuleFor(language, engine),
		"{{narrate_example}}", domain.NarrateExample(language),
		"{{thumbnail_audience}}", domain.ThumbnailAudience(language),
		"{{format_beats}}", beats,
		"{{subtitle_zone}}", domain.SubtitleZoneFor(domain.SubtitleMode(in.SubtitleMode), style, language),
	)
	return RenderedPrompt{
		Role: in.Role, Language: language, Prompt: r.Replace(effective.TemplateText),
		PromptID: effective.ID, PromptName: effective.Name, IsSystem: effective.IsSystem,
	}, nil
}

// beatsFor renders {{format_beats}} at the speaking rate of the chosen voice. A
// missing format or an uncalibrated voice is not an error: the prompt still
// renders, without the section or at the language default rate.
func (uc *RenderPromptUseCase) beatsFor(ctx context.Context, in RenderInput, language string) string {
	format, err := uc.formats.GetVideoFormat(ctx, in.FormatID, in.FormatVersion)
	if err != nil {
		return ""
	}
	var wpm float64
	if in.VoiceID != "" {
		if c, calErr := uc.calibration.GetVoiceCalibration(ctx, in.VoiceID); calErr == nil {
			if measured, ok := c.WordsPerMinute(); ok {
				wpm = measured
			}
		}
	}
	return domain.BuildStoryBeatSheetSection(format, language, wpm)
}
