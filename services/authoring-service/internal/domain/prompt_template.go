package domain

import "strings"

// PromptRole identifies which stage of the CR-025 authoring pipeline a
// template drives: Story Architect → Visual Director → Manim Engineer, with
// remotion_engineer replacing manim_engineer when the project renders with
// Remotion. The first two steps are engine agnostic — only the code step
// forks by render engine.
//
// CR-030 bỏ hẳn vai trò thứ tư, Script Reviewer: bước duyệt không còn tồn tại
// trong sản phẩm. Cột review_content và những dòng prompt cũ trong DB vẫn nằm
// yên đó — không có gì đọc chúng nữa, và xoá dữ liệu của Creator để dọn dẹp là
// cái giá không đáng.
type PromptRole string

const (
	RoleStoryArchitect PromptRole = "story_architect"
	RoleVisualDirector PromptRole = "visual_director"
	RoleManimEngineer  PromptRole = "manim_engineer"
	// feature/remotion-engine: a single flat prompt (topic -> code), the same
	// shape the Manim path had before CR-025 split it into 4 roles — Remotion
	// has no design system/multi-step pipeline yet, so one prompt is the
	// whole story for now. Selected instead of story_architect when the
	// Creator's project has render_engine=remotion (see ScriptAssistant.tsx).
	RoleRemotionEngineer PromptRole = "remotion_engineer"

	// CR-039 — the AI flow ("Chạy bằng AI") has its own prompts; the manual
	// (Copy) flow keeps the four above untouched. See prompt_template_seeds_ai.go.
	RoleVisualDirectorAI   PromptRole = "visual_director_ai"
	RoleManimEngineerAI    PromptRole = "manim_engineer_ai"
	RoleRemotionEngineerAI PromptRole = "remotion_engineer_ai"

	// CR-040 FR113 — the prompts web-gui used to assemble in the browser, now
	// library roles rendered by POST /v1/prompt-renders. None of them is a step
	// of the authoring pipeline; each is a "copy this to an external AI" prompt.
	RoleManimAdjust     PromptRole = "manim_adjust"     // fix an existing Manim script (narrate calls, design system)
	RoleRemotionAdjust  PromptRole = "remotion_adjust"  // fix an existing Remotion component
	RoleShortScript     PromptRole = "short_script"     // draft a Shorts/TikTok script
	RoleThumbnailDesign PromptRole = "thumbnail_design" // write an image-generation prompt for the thumbnail
)

// ValidPromptRole reports whether role is one of the known pipeline roles.
func ValidPromptRole(role string) bool {
	switch PromptRole(role) {
	case RoleStoryArchitect, RoleVisualDirector, RoleManimEngineer,
		RoleRemotionEngineer, RoleVisualDirectorAI, RoleManimEngineerAI, RoleRemotionEngineerAI,
		RoleManimAdjust, RoleRemotionAdjust, RoleShortScript, RoleThumbnailDesign:
		return true
	default:
		return false
	}
}

// PromptTemplate is one role/language's editable prompt text (CR-025).
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

// Prompt is one row of the prompt library (CR-031): a named wording for one
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
