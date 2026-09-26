package application

import (
	"context"
	"fmt"
	"strings"

	"orchestrator/internal/domain"
)

// PromptRenderContextPort supplies everything a prompt needs that is not the
// template itself.
type PromptRenderContextPort interface {
	Get(ctx context.Context, projectID string) (*domain.Project, error)
	GetAuthoringTopic(ctx context.Context, projectID string) (string, error)
	GetAuthoringStory(ctx context.Context, projectID string) (string, error)
	GetAuthoringStoryboard(ctx context.Context, projectID string) (string, error)
	GetAuthoringCode(ctx context.Context, projectID string) (string, error)
}

// VoiceCalibrationPort returns what a voice has actually been measured doing
// (CR-016). The measurement is only trusted after enough samples; see
// domain.VoiceCalibration.WordsPerMinute.
type VoiceCalibrationPort interface {
	GetVoiceCalibration(ctx context.Context, voiceID string) (domain.VoiceCalibration, error)
}

// FormatLookupPort resolves the project's chosen format at the version the
// project was created against — a format edited since must not silently
// change the budgets an existing outline was written to.
type FormatLookupPort interface {
	GetVideoFormat(ctx context.Context, formatID string, version int) (domain.VideoFormat, error)
}

// RenderPromptUseCase fills a role's template with this project's data
// (CR-027 FR77).
//
// This used to happen in the browser: scriptPrompts.ts substituted the
// variables into the template web-gui had fetched. That left the server
// unable to produce a prompt, which FR78's generate endpoints need to do —
// and it meant the only copy of the substitution logic lived in a place the
// server could not reach.
//
// One renderer serves both paths. The Copy-prompt button and the Run-with-AI
// button must send identical text to the model; two implementations would
// let them drift on the same role with nothing in either output to show it.
// PromptActivePort is the one read the renderer needs from the prompt library.
type PromptActivePort interface {
	GetActive(ctx context.Context, role domain.PromptRole) (domain.Prompt, error)
}

type RenderPromptUseCase struct {
	prompts     PromptActivePort
	projects    PromptRenderContextPort
	formats     FormatLookupPort
	calibration VoiceCalibrationPort
}

func NewRenderPromptUseCase(
	prompts PromptActivePort,
	projects PromptRenderContextPort,
	formats FormatLookupPort,
	calibration VoiceCalibrationPort,
) *RenderPromptUseCase {
	return &RenderPromptUseCase{
		prompts: prompts, projects: projects,
		formats: formats, calibration: calibration,
	}
}

// RenderedPrompt is one fully substituted prompt.
type RenderedPrompt struct {
	Role     domain.PromptRole `json:"role"`
	Language string            `json:"language"`
	Prompt   string            `json:"prompt"`
	// PromptID/PromptName say which library row this text came from.
	PromptID   string `json:"prompt_id"`
	PromptName string `json:"prompt_name"`
	IsSystem   bool   `json:"is_system"`
}

// TopicPlaceholder is what {{topic}} becomes when the project has no topic
// saved — every project created before CR-027 D0. Identical to the string
// web-gui has always shown, so an old project's prompt reads exactly as it
// did before.
const TopicPlaceholder = "[DÁN CHỦ ĐỀ CỦA BẠN VÀO ĐÂY]"

// RoleFor maps a pipeline step to the prompt role, which depends on the
// project's render engine (CR-027 D4).
//
// This lives on the server on purpose. web-gui currently works it out in two
// separate pages, and a third copy in a third place is how the Remotion and
// Manim branches end up disagreeing about which prompt step 3 uses.
func RoleFor(step string, renderEngine string) (domain.PromptRole, error) {
	remotion := renderEngine == "remotion"
	switch step {
	case "story":
		// Shared: this step decides the story, not the pixels.
		return domain.RoleStoryArchitect, nil
	case "storyboard":
		// Shared: the director writes an engine-agnostic shooting script;
		// translating it into what the engine can render is the code step's job.
		return domain.RoleVisualDirector, nil
	case "code":
		if remotion {
			return domain.RoleRemotionEngineer, nil
		}
		return domain.RoleManimEngineer, nil
	default:
		return "", fmt.Errorf("unknown step %q", step)
	}
}

// AIRoleFor is RoleFor for the AI flow (CR-039): the story step is shared, the
// storyboard step asks for JSON, and the code step asks for shot functions
// only. The manual (Copy) flow keeps using RoleFor.
func AIRoleFor(step string, renderEngine string) (domain.PromptRole, error) {
	switch step {
	case "story":
		return domain.RoleStoryArchitect, nil
	case "storyboard":
		return domain.RoleVisualDirectorAI, nil
	case "code":
		if renderEngine == "remotion" {
			return domain.RoleRemotionEngineerAI, nil
		}
		return domain.RoleManimEngineerAI, nil
	default:
		return "", fmt.Errorf("unknown step %q", step)
	}
}

// Execute renders the prompt for one role of one project.
//
// CR-030 — không còn tham số lintResults: {{lint_results}} chỉ tồn tại cho
// bước duyệt (Script Reviewer), mà bước đó đã bị bỏ khỏi sản phẩm.
func (uc *RenderPromptUseCase) Execute(
	ctx context.Context, projectID string, role domain.PromptRole,
) (RenderedPrompt, error) {
	if projectID == "" {
		return RenderedPrompt{}, fmt.Errorf("project_id is required")
	}
	project, err := uc.projects.Get(ctx, projectID)
	if err != nil {
		return RenderedPrompt{}, fmt.Errorf("load project: %w", err)
	}

	language := string(project.ContentLanguage)
	if language != "vi" && language != "en" {
		language = "vi"
	}

	effective, err := uc.prompts.GetActive(ctx, role)
	if err != nil {
		return RenderedPrompt{}, fmt.Errorf("load template: %w", err)
	}

	vars, err := uc.variablesFor(ctx, projectID, project, role, language)
	if err != nil {
		return RenderedPrompt{}, err
	}

	out := effective.TemplateText
	for name, value := range vars {
		out = strings.ReplaceAll(out, "{{"+name+"}}", value)
	}

	return RenderedPrompt{
		Role: role, Language: language,
		Prompt: out, PromptID: effective.ID, PromptName: effective.Name, IsSystem: effective.IsSystem,
	}, nil
}

func (uc *RenderPromptUseCase) variablesFor(
	ctx context.Context, projectID string, project *domain.Project,
	role domain.PromptRole, language string,
) (map[string]string, error) {
	topic, err := uc.projects.GetAuthoringTopic(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("load topic: %w", err)
	}
	if strings.TrimSpace(topic) == "" {
		topic = TopicPlaceholder
	}

	previous, err := uc.previousOutputFor(ctx, projectID, role)
	if err != nil {
		return nil, err
	}

	vars := map[string]string{
		"topic":                   topic,
		"channel_identity":        domain.ChannelIdentity(language),
		"narration_language_rule": domain.NarrationLanguageRule(language),
		"previous_output":         previous,
		"format_beats":            "",
		"subtitle_zone":           domain.SubtitleZone(project, language),
	}

	// The beat sheet only means something for the step that writes the
	// outline; fetching a format for the others would be work whose result
	// nothing reads.
	if role == domain.RoleStoryArchitect && project.VideoFormatID != "" {
		format, err := uc.formats.GetVideoFormat(ctx, project.VideoFormatID, project.VideoFormatVersion)
		if err == nil {
			var wpm float64
			if project.VoiceID != "" {
				// A voice with too few samples is not an error: fall back to
				// the language default rather than refusing to render a
				// prompt over a missing nicety. Same rule the GUI applies.
				if c, calErr := uc.calibration.GetVoiceCalibration(ctx, project.VoiceID); calErr == nil {
					if measured, ok := c.WordsPerMinute(); ok {
						wpm = measured
					}
				}
			}
			vars["format_beats"] = domain.BuildStoryBeatSheetSection(format, language, wpm)
		}
	}

	return vars, nil
}

// previousOutputFor assembles {{previous_output}} the way each step needs it:
// every earlier artefact, oldest first, joined the way web-gui has always
// joined them.
func (uc *RenderPromptUseCase) previousOutputFor(
	ctx context.Context, projectID string, role domain.PromptRole,
) (string, error) {
	var parts []string
	add := func(s string) {
		if strings.TrimSpace(s) != "" {
			parts = append(parts, s)
		}
	}

	story, err := uc.projects.GetAuthoringStory(ctx, projectID)
	if err != nil {
		return "", fmt.Errorf("load story: %w", err)
	}
	storyboard, err := uc.projects.GetAuthoringStoryboard(ctx, projectID)
	if err != nil {
		return "", fmt.Errorf("load storyboard: %w", err)
	}
	switch role {
	case domain.RoleStoryArchitect:
		// Nothing comes before step 1.
	case domain.RoleVisualDirector, domain.RoleVisualDirectorAI:
		add(story)
	case domain.RoleManimEngineer, domain.RoleRemotionEngineer:
		add(story)
		add(storyboard)
	case domain.RoleManimEngineerAI, domain.RoleRemotionEngineerAI:
		// The storyboard is not in the system prompt: llm-service hands each
		// call its own slice of it, and the whole document only to the layout call.
		add(story)
	}

	if len(parts) == 0 {
		return "(chưa có dàn ý/storyboard/code đã lưu ở các bước trước)", nil
	}
	return strings.Join(parts, "\n\n---\n\n"), nil
}
