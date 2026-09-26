package domain

import _ "embed"

// CR-040 FR113 — the starter scripts and insertable snippets web-gui used to
// carry in scriptTemplates.ts. They are static, so they are embedded as the
// exact files the TypeScript held rather than retyped.
//
// The Vietnamese starter mirrors services/rendering/tests/fixtures/conceptflow_template.py,
// where lint and a real Manim render run against it: a template the lint itself
// rejects is the surest way to make a Creator distrust the whole system.

//go:embed templates/starter_script_vi.py
var starterScriptVI string

//go:embed templates/starter_script_en.py
var starterScriptEN string

//go:embed templates/hook_snippet_vi.txt
var hookSnippetVI string

//go:embed templates/hook_snippet_en.txt
var hookSnippetEN string

//go:embed templates/end_screen_snippet_vi.txt
var endScreenSnippetVI string

//go:embed templates/end_screen_snippet_en.txt
var endScreenSnippetEN string

// LanguageText is one text per content language.
type LanguageText struct {
	VI string `json:"vi"`
	EN string `json:"en"`
}

// ScriptTemplates is what GET /v1/script-templates serves.
type ScriptTemplates struct {
	StarterScript    LanguageText `json:"starter_script"`
	HookSnippet      LanguageText `json:"hook_snippet"`
	EndScreenSnippet LanguageText `json:"end_screen_snippet"`
}

// BuiltinScriptTemplates returns the shipped starter scripts and snippets.
func BuiltinScriptTemplates() ScriptTemplates {
	return ScriptTemplates{
		StarterScript:    LanguageText{VI: starterScriptVI, EN: starterScriptEN},
		HookSnippet:      LanguageText{VI: hookSnippetVI, EN: hookSnippetEN},
		EndScreenSnippet: LanguageText{VI: endScreenSnippetVI, EN: endScreenSnippetEN},
	}
}
