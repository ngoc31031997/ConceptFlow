package domain

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
	"testing"
)

var placeholderRe = regexp.MustCompile(`\{\{([a-z_]+)\}\}`)

// The manual (Copy-prompt) flow must not change when the shared prompt text is
// factored into parts for the AI flow (CR-039). These are the SHA-256 of the
// shipped templates as they were before the split.
var goldenManualPrompts = map[PromptRole]string{
	RoleStoryArchitect:   "0e7ac7f0b0c7d7e0049c4ddc29cad4614364151785e479f1e50e0040f88dd28a",
	RoleVisualDirector:   "ae8ab5ebde484ca67ed63db3301e5d125954aa06d37c37aa1a86918abf09e7f6",
	RoleManimEngineer:    "16ae08523e73434ddd775ef13edf47b5ba3dc1584a2e102e52ca8ccbf9000ddd",
	RoleRemotionEngineer: "04e157efbf7c67934879797aa8273b146a425f98558a37a128ad37884c45b141",
}

func TestManualPromptsAreByteIdenticalToTheShippedOnes(t *testing.T) {
	for _, tpl := range DefaultPromptTemplates() {
		want, ok := goldenManualPrompts[tpl.Role]
		if !ok {
			continue // AI-flow roles have no manual golden
		}
		if got := fmt.Sprintf("%x", sha256.Sum256([]byte(tpl.TemplateText))); got != want {
			t.Errorf("%s changed: sha256 %s, want %s", tpl.Role, got, want)
		}
	}
}

// --- the AI flow's prompts (CR-039) ---------------------------------------------

func aiTemplate(t *testing.T, role PromptRole) string {
	t.Helper()
	for _, tpl := range DefaultPromptTemplates() {
		if tpl.Role == role {
			return tpl.TemplateText
		}
	}
	t.Fatalf("no shipped template for %s", role)
	return ""
}

func TestAIPromptsUseOnlyPlaceholdersTheRendererFills(t *testing.T) {
	known := map[string]bool{
		"topic": true, "previous_output": true, "narration_language_rule": true,
		"channel_identity": true, "format_beats": true, "subtitle_zone": true,
	}
	for _, role := range []PromptRole{RoleVisualDirectorAI, RoleManimEngineerAI, RoleRemotionEngineerAI} {
		text := aiTemplate(t, role)
		for _, m := range placeholderRe.FindAllStringSubmatch(text, -1) {
			if !known[m[1]] {
				t.Errorf("%s uses {{%s}}, which the renderer does not fill", role, m[1])
			}
		}
		if strings.Contains(text, "¤") {
			t.Errorf("%s still contains the ¤ backtick stand-in", role)
		}
	}
}

func TestVisualDirectorAIAsksForJSONAndKeepsTheCreativeBrief(t *testing.T) {
	ai, manual := aiTemplate(t, RoleVisualDirectorAI), aiTemplate(t, RoleVisualDirector)
	for _, want := range []string{`"scenes"`, `"palette"`, `"narration"`, "#RRGGBB", "JSON"} {
		if !strings.Contains(ai, want) {
			t.Errorf("visual_director_ai lacks %q", want)
		}
	}
	if strings.Contains(ai, "CẢNH <n> —") || strings.Contains(ai, "| MÁY:") {
		t.Error("visual_director_ai still asks for the prose shot format")
	}
	brief := manual[:strings.Index(manual, "## OUTPUT")]
	if !strings.HasPrefix(ai, brief) {
		t.Error("visual_director_ai must reuse the manual prompt's whole creative brief unchanged")
	}
}

func TestEngineerAIPromptsWriteShotsOnlyAndShareTheRulebook(t *testing.T) {
	remo, manim := aiTemplate(t, RoleRemotionEngineerAI), aiTemplate(t, RoleManimEngineerAI)
	for _, want := range []string{"KHÔNG viết cả file", "ShotN_M", "LAYOUT", "PALETTE.", "L1. **Vùng an toàn.**", "LottieClip"} {
		if !strings.Contains(remo, want) {
			t.Errorf("remotion_engineer_ai lacks %q", want)
		}
	}
	for _, gone := range []string{"export const narrations: string[]", "registerRoot(() =>", "SHOTS.length === narrations.length"} {
		if strings.Contains(remo[strings.Index(remo, "## D."):strings.Index(remo, "## E.")], gone) {
			t.Errorf("remotion_engineer_ai section D still demands %q", gone)
		}
	}
	for _, want := range []string{"KHÔNG viết cả file", "shot_N_M", "setup_cast", "self.reveal(obj)", "API ĐƯỢC PHÉP DÙNG"} {
		if !strings.Contains(manim, want) {
			t.Errorf("manim_engineer_ai lacks %q", want)
		}
	}
	// The three prebuilt beats emit their own beat, and the system already emits one per scene.
	if strings.Contains(manim, "Ba beat dựng sẵn — DÙNG CHÚNG") {
		t.Error("manim_engineer_ai still tells the model to use the prebuilt beats")
	}
	if strings.Contains(manim, "{{theme_reference}}") || strings.Contains(remo, "{{lottie_catalog}}") {
		t.Error("seed-time placeholders must be expanded")
	}
}
