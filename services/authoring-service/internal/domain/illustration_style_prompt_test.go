package domain

import (
	"fmt"
	"strings"
	"testing"
)

// The style rules ship as the system row of the illustration_style role, the
// Creator may replace them in the prompt library, and they are never a prompt
// for an external AI.
func TestIllustrationStyleIsASystemPromptRow(t *testing.T) {
	var found *Prompt
	for _, p := range SystemPrompts() {
		if p.Role == RoleIllustrationStyle {
			found = &p
		}
	}
	if found == nil {
		t.Fatal("no system-illustration_style row")
	}
	if found.ID != "system-illustration_style" || found.TemplateText != IllustrationStyleGuide() {
		t.Fatalf("system row: %+v", found.ID)
	}
	if !ValidPromptRole("illustration_style") || IsRenderablePromptRole("illustration_style") {
		t.Fatal("illustration_style must be a known role that is never rendered")
	}
	if !IsRenderablePromptRole("story_architect") || IsRenderablePromptRole("nope") {
		t.Fatal("renderable roles are the known ones but the style rules")
	}
}

// The shipped rules name no channel (the reference videos are only a
// reference) and keep every [Sn] line in the form web-gui splits rule names by.
func TestShippedStyleRulesNameNoChannelAndKeepEveryRule(t *testing.T) {
	text := IllustrationStyleGuide()
	if strings.Contains(text, "Vẽ Chuyện") {
		t.Error("the style rules still name a channel")
	}
	for n := 1; n <= 25; n++ {
		if !strings.Contains(text, fmt.Sprintf("\n- [S%d] ", n)) {
			t.Errorf("rule S%d is missing or not on its own `- [S%d] NAME:` line", n, n)
		}
	}
	for _, want := range []string{"Sức sống của hình", "nền sáng bão hoà", `<Person framing="bust">`, "<ReachingHand>"} {
		if !strings.Contains(text, want) {
			t.Errorf("style rules lack %q", want)
		}
	}
}
