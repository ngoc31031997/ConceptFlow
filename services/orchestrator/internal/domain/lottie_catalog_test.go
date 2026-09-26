package domain

import (
	"strings"
	"testing"
)

// CR-038: {{lottie_catalog}} is baked into the Remotion Engineer at seed time.
// A leftover placeholder would reach the model as literal text.
func TestRemotionEngineerSeedExpandsLottieCatalog(t *testing.T) {
	for _, tpl := range DefaultPromptTemplates() {
		if tpl.Role != RoleRemotionEngineer {
			continue
		}
		if strings.Contains(tpl.TemplateText, "{{lottie_catalog}}") {
			t.Fatal("{{lottie_catalog}} was not expanded at seed time")
		}
		if !strings.Contains(tpl.TemplateText, "CLIP HOẠT HÌNH DỰNG SẴN") {
			t.Fatal("remotion engineer prompt lost its Lottie section")
		}
		return
	}
	t.Fatal("no remotion_engineer seed found")
}

// The Visual Director stays engine-agnostic: it must never be told about a
// clip catalog that a Manim project cannot use.
func TestVisualDirectorDoesNotSeeLottieCatalog(t *testing.T) {
	for _, tpl := range DefaultPromptTemplates() {
		if tpl.Role == RoleVisualDirector && strings.Contains(tpl.TemplateText, "LottieClip") {
			t.Fatal("visual director prompt mentions LottieClip")
		}
	}
}
