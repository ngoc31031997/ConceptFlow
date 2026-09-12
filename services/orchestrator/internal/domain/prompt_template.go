package domain

import "strings"

// PromptRole identifies which stage of the CR-025 authoring pipeline a
// template drives: Story Architect → Visual Director → Manim Engineer →
// Script Reviewer.
type PromptRole string

const (
	RoleStoryArchitect PromptRole = "story_architect"
	RoleVisualDirector PromptRole = "visual_director"
	RoleManimEngineer  PromptRole = "manim_engineer"
	RoleScriptReviewer PromptRole = "script_reviewer"
)

// ValidPromptRole reports whether role is one of the 4 pipeline roles.
func ValidPromptRole(role string) bool {
	switch PromptRole(role) {
	case RoleStoryArchitect, RoleVisualDirector, RoleManimEngineer, RoleScriptReviewer:
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
