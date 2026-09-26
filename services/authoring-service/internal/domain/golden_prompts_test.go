package domain

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
	"testing"
)

var placeholderRe = regexp.MustCompile(`\{\{([a-z_]+)\}\}`)

// story_architect was re-baselined on purpose by CR-041 (video archetypes).
//
// The manual (Copy-prompt) flow must not change when the shared prompt text is
// factored into parts for the AI flow (CR-039). These are the SHA-256 of the
// shipped templates as they were before the split.
var goldenManualPrompts = map[PromptRole]string{
	RoleStoryArchitect:   "c7263faacab4a6d964dad521602534119ecc1e14abe72d8c19a0cb5a538624ba",
	RoleVisualDirector:   "5fb59d05fbdbb3e45d6985c20dab2ae5484064a39c170b4b2fea7f6c20558e36",
	RoleManimEngineer:    "85ff46357c76b59556b6a4c2e5298c57e5bcda13589071a4d72101efcae10fff",
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
		"channel_identity": true, "format_beats": true, "subtitle_zone": true, "video_archetypes": true,
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
	for _, want := range []string{`"scenes"`, `"palette"`, `"narration"`, "#RRGGBB", "JSON", `"layout"`, "{{subtitle_zone}}"} {
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

// CR-042: the director gets rules 12–18 and a list of buildable materials; the
// Manim engineer animates during narration and no longer defaults to title cards.
func TestCinematicRulesAndMotionDuringNarration(t *testing.T) {
	for _, role := range []PromptRole{RoleVisualDirector, RoleVisualDirectorAI} {
		text := aiTemplate(t, role)
		for _, want := range []string{"NHỊP THAY ĐỔI", "CHO NGƯỜI XEM ĐOÁN TRƯỚC", "DIỄN XUẤT BẰNG CHUYỂN ĐỘNG",
			"KHUNG KẾT VẦN VỚI KHUNG MỞ", "HOOK KHÔNG PHẢI THẺ TIÊU ĐỀ", "vật liệu dựng tốt", "suy ngẫm"} {
			if !strings.Contains(text, want) {
				t.Errorf("%s lacks %q", role, want)
			}
		}
		if strings.Contains(text, "Không có chuyển động trang trí: vật lắc lư, nhấp nháy, xoay vòng mà không thêm ý nào là rác.\n") {
			t.Errorf("%s rule 5 still bans all ambient motion", role)
		}
	}
	for _, role := range []PromptRole{RoleManimEngineer, RoleManimEngineerAI} {
		text := aiTemplate(t, role)
		if !strings.Contains(text, `self.narrate("`) || !strings.Contains(text, "drift=True") {
			t.Errorf("%s does not teach narrate(text, animation...) / drift", role)
		}
		if strings.Contains(text, "đặt TRƯỚC lời gọi") {
			t.Errorf("%s still tells the model to play before narrating", role)
		}
	}
	manual := aiTemplate(t, RoleManimEngineer)
	for _, want := range []string{"hook_card", "recap_card", "KHÔNG hiện thẻ tiêu đề"} {
		if !strings.Contains(manual, want) {
			t.Errorf("manual manim engineer lacks %q", want)
		}
	}
}

// CR-041: the Story Architect picks a video archetype and says so first, while
// every beat id still comes from the chosen format.
func TestStoryArchitectAsksForAnArchetypeAndKeepsTheFormatsBeats(t *testing.T) {
	text := aiTemplate(t, RoleStoryArchitect)
	for _, want := range []string{
		"KIỂU VIDEO: <mã kiểu>", "CẢNH BÁO FORMAT", "{{format_beats}}",
		"{{video_archetypes}}", "kiểu: <mã>",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("story_architect lacks %q", want)
		}
	}
	if strings.LastIndex(text, "KIỂU VIDEO: <mã kiểu>") > strings.LastIndex(text, "TÌNH HUỐNG ỨNG VIÊN:") {
		t.Error("the KIỂU VIDEO line must come before the rest of the output")
	}
}
