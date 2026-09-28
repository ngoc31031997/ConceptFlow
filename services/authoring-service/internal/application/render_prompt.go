package application

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"authoring/internal/domain"
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
	archetypes  ArchetypeListPort // optional (CR-041); nil renders {{video_archetypes}} empty
}

// ArchetypeListPort is the one read the renderer needs from the archetype table.
type ArchetypeListPort interface {
	ListArchetypes(ctx context.Context) ([]domain.VideoArchetype, error)
}

// WithArchetypes lets the renderer fill {{video_archetypes}} from the table.
func (uc *RenderPromptUseCase) WithArchetypes(p ArchetypeListPort) *RenderPromptUseCase {
	uc.archetypes = p
	return uc
}

// archetypeSection is {{video_archetypes}}. A failed read renders the "no kinds"
// text rather than failing the prompt over a missing nicety.
func (uc *RenderPromptUseCase) archetypeSection(ctx context.Context) string {
	var list []domain.VideoArchetype
	if uc.archetypes != nil {
		list, _ = uc.archetypes.ListArchetypes(ctx)
	}
	return domain.BuildVideoArchetypesSection(list)
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
		"narration_language_rule": domain.NarrationLanguageRuleFor(language, engineOf(role)),
		"previous_output":         previous,
		"format_beats":            "",
		"video_archetypes":        "",
		"subtitle_zone":           domain.SubtitleZone(project, language),
	}

	// The beat sheet only means something for the step that writes the
	// outline; fetching a format for the others would be work whose result
	// nothing reads.
	if role == domain.RoleStoryArchitect {
		vars["video_archetypes"] = uc.archetypeSection(ctx)
	}
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
		//
		// CR-048 T3 — nor is the whole outline. This text is the system prompt
		// of every chunk and repair call, and each of those user turns already
		// carries its shots' invariant/narration/visual; the beats, candidate
		// situations and word counts only give the model more to reason about.
		// Keep the core lines. An outline whose labels cannot be found (edited
		// by hand, format changed) is passed whole and logged — never cut
		// silently.
		if strings.TrimSpace(story) != "" {
			core, missing := extractStoryCore(story)
			switch {
			case core == "":
				slog.Warn("code step: no core labels found in the story outline — passing it whole",
					"project_id", projectID, "role", string(role), "story_chars", len(story))
				add(story)
			default:
				if len(missing) > 0 {
					slog.Warn("code step: story outline is missing some core labels",
						"project_id", projectID, "role", string(role), "missing", strings.Join(missing, ", "))
				}
				add(core)
			}
		}
	}

	if len(parts) == 0 {
		return "(chưa có dàn ý/storyboard/code đã lưu ở các bước trước)", nil
	}
	return strings.Join(parts, "\n\n---\n\n"), nil
}

// storyCoreLabels are the lines of the Story Architect's output
// (storyArchitectVI, "## OUTPUT") that the code step keeps, in the order the
// outline prints them. "Thế giới chính" sits under KHUNG BÀI.
var storyCoreLabels = []string{
	"KIỂU VIDEO",
	"Thế giới chính",
	"CÂU HỎI CỐT LÕI",
	"INSIGHT CỐT LÕI",
	"SAI LẦM TRỰC GIÁC",
	"AHA MOMENT",
}

const storyAhaLabel = "AHA MOMENT"

// storyAhaSubLabels are the two lines printed under AHA MOMENT.
var storyAhaSubLabels = []string{"Tôi từng nghĩ", "Nhưng bây giờ tôi nhận ra"}

// extractStoryCore returns the core lines of a Story Architect outline and
// the labels it could not find. core is "" when no label was found at all.
//
// Tolerant of what models and Creators do to the format: leading whitespace,
// list markers, markdown headings and bold ("**KIỂU VIDEO:**", "**KIỂU
// VIDEO**:"), letter case, and a value written on the lines below its label.
// Only the first occurrence of a label counts.
func extractStoryCore(story string) (core string, missing []string) {
	lines := strings.Split(strings.ReplaceAll(story, "\r\n", "\n"), "\n")
	found := map[string]bool{}
	var out []string
	inAha := false

	for i := 0; i < len(lines); i++ {
		line := cleanStoryLine(lines[i])
		if line == "" {
			continue
		}
		if inAha {
			if label, _, ok := matchStoryLabel(line, storyAhaSubLabels); ok && !found[label] {
				found[label] = true
				out = append(out, "  "+line)
				continue
			}
			inAha = false
		}
		label, value, ok := matchStoryLabel(line, storyCoreLabels)
		if !ok || found[label] {
			continue
		}
		found[label] = true
		if value == "" {
			// The value sits on the lines below: take them up to a blank line
			// or the next "LABEL:" line.
			var cont []string
			for i+1 < len(lines) {
				next := cleanStoryLine(lines[i+1])
				if next == "" || looksLikeStoryLabel(next) {
					break
				}
				cont = append(cont, next)
				i++
			}
			if len(cont) > 0 {
				line = strings.TrimRight(line, " ") + " " + strings.Join(cont, " ")
			}
		}
		out = append(out, line)
		inAha = label == storyAhaLabel
	}

	for _, l := range storyCoreLabels {
		if !found[l] {
			missing = append(missing, l)
		}
	}
	if len(out) == 0 {
		return "", missing
	}
	return strings.Join(out, "\n"), missing
}

// cleanStoryLine strips indentation, list/heading/quote markers and markdown
// bold so a label can be recognised however the outline was formatted.
func cleanStoryLine(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimLeft(s, "#>-*• \t")
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "__", "")
	return strings.TrimSpace(s)
}

// matchStoryLabel reports which of labels line starts with, followed by ":"
// (spaces allowed before it), ignoring case. value is what follows the colon.
func matchStoryLabel(line string, labels []string) (label, value string, ok bool) {
	// Compare rune by rune: upper-casing keeps the rune count, not
	// necessarily the byte count.
	runes := []rune(line)
	for _, l := range labels {
		lr := []rune(l)
		if len(runes) < len(lr) || strings.ToUpper(string(runes[:len(lr)])) != strings.ToUpper(l) {
			continue
		}
		rest := strings.TrimLeft(string(runes[len(lr):]), " \t")
		if strings.HasPrefix(rest, ":") {
			return l, strings.TrimSpace(rest[1:]), true
		}
	}
	return "", "", false
}

// looksLikeStoryLabel reports a line that opens another field of the outline
// ("BEAT hook — Mở đầu:", "KHUNG BÀI:", "Tôi từng nghĩ: ..."): a short head
// of at most eight words, then a colon. It bounds how far a label whose value
// sits on the lines below may read.
func looksLikeStoryLabel(line string) bool {
	head, _, ok := strings.Cut(line, ":")
	if !ok {
		return false
	}
	n := len(strings.Fields(head))
	return n > 0 && n <= 8
}

// engineOf is the render engine a role writes for; only Remotion's roles differ.
func engineOf(role domain.PromptRole) string {
	if isRemotionRole(role) {
		return "remotion"
	}
	return "manim"
}
