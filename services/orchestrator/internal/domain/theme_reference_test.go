package domain

import (
	"strings"
	"testing"
)

func TestManimEngineerSeedsCarryExpandedThemeReference(t *testing.T) {
	for _, lang := range []string{"vi"} {
		tpl, ok := DefaultPromptTemplate(RoleManimEngineer, lang)
		if !ok {
			t.Fatalf("no manim_engineer seed for %s", lang)
		}
		if strings.Contains(tpl.TemplateText, "{{theme_reference}}") {
			t.Errorf("%s: placeholder was not expanded", lang)
		}
		if !strings.Contains(tpl.TemplateText, "self.theme.accent") ||
			!strings.Contains(tpl.TemplateText, "CENTER") {
			t.Errorf("%s: theme reference missing from seed", lang)
		}
	}
}
