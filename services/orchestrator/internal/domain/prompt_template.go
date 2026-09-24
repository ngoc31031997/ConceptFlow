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
)

// ValidPromptRole reports whether role is one of the known pipeline roles.
func ValidPromptRole(role string) bool {
	switch PromptRole(role) {
	case RoleStoryArchitect, RoleVisualDirector, RoleManimEngineer,
		RoleRemotionEngineer:
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

// PromptOverride is the Creator's own wording for one role/language
// (CR-027 FR84.3). It lives in its own table, apart from the shipped
// PromptTemplate, so that seeding can overwrite the shipped text on every
// start without ever touching what the Creator wrote.
type PromptOverride struct {
	Role         PromptRole `json:"role"`
	Language     string     `json:"language"`
	TemplateText string     `json:"template_text"`
	// IsActive off keeps the wording but runs the shipped text instead. This
	// is what replaced the old destructive reset: switching back on restores
	// the Creator's version unchanged.
	IsActive bool `json:"is_active"`
	// BasedOnVersion is the shipped version this wording was written
	// against, so the admin screen can flag an override that has fallen
	// behind (FR84.7). 0 means "unknown" — a row created by the migration
	// from a database that predates this column.
	BasedOnVersion int    `json:"based_on_version"`
	UpdatedAt      string `json:"updated_at"`
}

// EffectivePromptTemplate is what the pipeline actually renders: the
// override when one is switched on, otherwise the shipped text.
//
// FromOverride and SeedVersion are carried so the admin screen can say which
// of the two is in force, and warn when the shipped wording has moved on
// underneath an active override (FR84.7).
type EffectivePromptTemplate struct {
	Role         PromptRole `json:"role"`
	Language     string     `json:"language"`
	TemplateText string     `json:"template_text"`
	FromOverride bool       `json:"from_override"`
	SeedVersion  int        `json:"seed_version"`
}
