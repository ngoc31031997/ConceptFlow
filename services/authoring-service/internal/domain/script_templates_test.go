package domain

import (
	"regexp"
	"strings"
	"testing"
)

// The assertions web-gui's contentLanguage.test.ts held for these texts, kept
// now that the texts live here (CR-040 FR113).
func TestBuiltinScriptTemplates(t *testing.T) {
	tpl := BuiltinScriptTemplates()

	for lang, text := range map[string]string{"vi": tpl.StarterScript.VI, "en": tpl.StarterScript.EN} {
		if !strings.Contains(text, "self.narrate(") {
			t.Errorf("%s starter has no self.narrate(...) — a video with no narration", lang)
		}
		if strings.Contains(text, "# NARRATION:") || regexp.MustCompile(`self\.wait\(\s*AUTO\s*\)`).MatchString(text) {
			t.Errorf("%s starter still uses the removed marker convention", lang)
		}
		if !strings.Contains(text, "from conceptflow import *") {
			t.Errorf("%s starter is not written against the design system", lang)
		}
	}
	if tpl.StarterScript.VI == tpl.StarterScript.EN {
		t.Error("each language needs its own starter")
	}

	snippets := map[string]string{
		"hook vi": tpl.HookSnippet.VI, "hook en": tpl.HookSnippet.EN,
		"end vi": tpl.EndScreenSnippet.VI, "end en": tpl.EndScreenSnippet.EN,
	}
	for name, s := range snippets {
		if strings.Contains(s, "# NARRATION:") || strings.Contains(s, "wait(AUTO)") {
			t.Errorf("%s uses the removed marker convention", name)
		}
	}
	if !strings.Contains(tpl.HookSnippet.VI, "self.hook(") || !strings.Contains(tpl.HookSnippet.EN, "self.hook(") {
		t.Error("hook snippets must use self.hook(")
	}
	if !strings.Contains(tpl.EndScreenSnippet.EN, "subscribe") || !strings.Contains(tpl.EndScreenSnippet.VI, "đăng ký kênh") {
		t.Error("each end-screen snippet must be written in its own language")
	}
}
