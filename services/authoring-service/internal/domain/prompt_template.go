package domain

import "strings"

// PromptRole identifies which stage of the authoring pipeline a
// template drives: Story Architect → Visual Director → Remotion Engineer.
//
// Không có vai trò Script Reviewer. Cột review_content và những dòng prompt
// cũ trong DB được giữ nguyên — không gì đọc chúng, và xoá dữ liệu của Creator
// để dọn dẹp là cái giá không đáng.
type PromptRole string

const (
	RoleStoryArchitect PromptRole = "story_architect"
	RoleVisualDirector PromptRole = "visual_director"
	// The code step: turns the storyboard into a Remotion composition.
	RoleRemotionEngineer PromptRole = "remotion_engineer"

	// The AI flow ("Chạy bằng AI") has its own prompts; the manual
	// (Copy) flow keeps the four above untouched. See prompt_template_seeds_ai.go.
	RoleVisualDirectorAI   PromptRole = "visual_director_ai"
	RoleRemotionEngineerAI PromptRole = "remotion_engineer_ai"

	// Library roles rendered by POST /v1/prompt-renders. None of them is a step
	// of the authoring pipeline; each is a "copy this to an external AI" prompt.
	RoleRemotionAdjust  PromptRole = "remotion_adjust"  // fix an existing Remotion component
	RoleThumbnailDesign PromptRole = "thumbnail_design" // write an image-generation prompt for the thumbnail

	// The idea step's writer: turns a topic into the creative brief. Filled by
	// the brief use case with the brief's own values (the Creator's note, the
	// fields the Creator wrote), so it is not rendered by the generic renderer.
	RoleCreativeBrief PromptRole = "creative_brief"

	// The research step's question writer: turns the approved brief into the
	// questions to research. Filled by the research use case, so it is not
	// rendered by the generic renderer.
	RoleResearchQuestions PromptRole = "research_questions"

	// The text the Creator copies into an outside research tool. Filled by the
	// research use case with the brief and the questions, so it is not rendered
	// by the generic renderer.
	RoleResearchPrompt PromptRole = "research_prompt"

	// The story step's proposer of approaches and writer of the story frame.
	// Filled by the story use case with the brief, the facts and the Creator's
	// choices, so they are not rendered by the generic renderer.
	RoleStoryApproaches PromptRole = "story_approaches"
	RoleStoryFrame      PromptRole = "story_frame"

	// The beat sheet step's writer of the beats; filled by the beat use case
	// with the story, the facts and the Creator's choices.
	RoleBeatSheet PromptRole = "beat_sheet"
)

// ValidPromptRole reports whether role is one of the known pipeline roles.
func ValidPromptRole(role string) bool {
	switch PromptRole(role) {
	case RoleStoryArchitect, RoleVisualDirector, RoleRemotionEngineer,
		RoleVisualDirectorAI, RoleRemotionEngineerAI,
		RoleRemotionAdjust, RoleThumbnailDesign,
		RoleCreativeBrief, RoleResearchQuestions, RoleResearchPrompt,
		RoleStoryApproaches, RoleStoryFrame, RoleBeatSheet:
		return true
	default:
		return false
	}
}

// IsRenderablePromptRole reports whether role is a known role whose prompt
// can be rendered with a project's or the Creator's values: every role but
// the brief writer, the research roles, the story roles and the beat writer, whose values come
// from their own use cases.
func IsRenderablePromptRole(role string) bool {
	switch PromptRole(role) {
	case RoleCreativeBrief, RoleResearchQuestions, RoleResearchPrompt, RoleStoryApproaches, RoleStoryFrame, RoleBeatSheet:
		return false
	}
	return ValidPromptRole(role)
}

// PromptTemplate is one role/language's editable prompt text.
//
// Text lives in Postgres, not in web-gui source, so an editor can fix wording
// without a frontend rebuild/deploy. Placeholders are plain `{{name}}` tokens
// substituted with strings.ReplaceAll — no templating engine was already in
// use anywhere in this Go codebase, so introducing text/template for 4 rows
// would be more machinery than the problem needs.
type PromptTemplate struct {
	Role         PromptRole `json:"role"`
	Language     string     `json:"language"` // "vi" | "en"
	TemplateText string     `json:"template_text"`
	Version      int        `json:"version"`
	UpdatedAt    string     `json:"updated_at,omitempty"` // RFC3339, set by the repository
}

// RenderPromptTemplate does plain-string placeholder substitution — the
// simplest option available given no templating engine is otherwise used in
// this codebase. Placeholders that have no entry in values are left as-is,
// which surfaces missing data instead of silently deleting the token.
func RenderPromptTemplate(templateText string, values map[string]string) string {
	out := templateText
	for key, value := range values {
		out = strings.ReplaceAll(out, "{{"+key+"}}", value)
	}
	return out
}

// Prompt is one row of the prompt library: a named wording for one
// pipeline role. Each role owns a list of these.
//
// Exactly one row per role is active at a time and is what the pipeline
// renders. The row with IsSystem set ships in the binary — seeded on every
// start, read-only to everyone. Any other row was written by a Creator, who
// may edit, delete and activate it. There is no version: a Creator's copy is
// theirs, and the shipped row is simply whatever this binary carries.
//
// No language either. Prompts are written in Vietnamese; the language of the
// narration a video ends up with comes from {{narration_language_rule}}.
type Prompt struct {
	ID           string     `json:"id"`
	Role         PromptRole `json:"role"`
	Name         string     `json:"name"`
	TemplateText string     `json:"template_text"`
	IsSystem     bool       `json:"is_system"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    string     `json:"created_at"`
	UpdatedAt    string     `json:"updated_at"`
}

// SystemPromptID is the fixed id of a role's shipped row, so seeding can
// upsert it in place across restarts.
func SystemPromptID(role PromptRole) string { return "system-" + string(role) }

// SystemPromptName is what the shipped row is called in the admin list.
const SystemPromptName = "Mặc định của hệ thống"

// SystemPrompts returns the shipped rows: one per role, Vietnamese wording.
func SystemPrompts() []Prompt {
	var out []Prompt
	for _, t := range DefaultPromptTemplates() {
		if t.Language != "vi" {
			continue
		}
		out = append(out, Prompt{
			ID: SystemPromptID(t.Role), Role: t.Role, Name: SystemPromptName,
			TemplateText: t.TemplateText, IsSystem: true,
		})
	}
	return out
}
