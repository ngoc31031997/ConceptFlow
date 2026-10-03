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
// factored into parts for the AI flow. These are the SHA-256 of the shipped
// manual templates; a deliberate wording change updates the hash here.
var goldenManualPrompts = map[PromptRole]string{
	RoleStoryArchitect:   "7f831e761dca2b49f3dc5a65d6ee91d7f9055875cad6a831086284b18d34237e",
	RoleVisualDirector:   "54cda71ec1f61c8c7a4edf5024999bbf1bb7fd941ca2f1c7c38e653226c7eb39",
	RoleManimEngineer:    "20c479ccf1cb414926bba423de9721dcc04b5fc6fe68f0f094a854c7442d1e54",
	RoleRemotionEngineer: "f3a2fa10f958f3729585b066e914b70be014489397d1128c00586d95b854affe",
}

// Every scene declares its own background colour, bright unless the setting
// says why it is dark: no Remotion-side prompt falls back to a default dark one.
func TestPromptsHaveNoDefaultDarkBackground(t *testing.T) {
	for _, role := range []PromptRole{RoleVisualDirector, RoleVisualDirectorAI, RoleRemotionEngineer, RoleRemotionEngineerAI} {
		if text := aiTemplate(t, role); strings.Contains(text, "#080E1C") {
			t.Errorf("%s still falls back to the default background #080E1C", role)
		}
	}
	for _, role := range []PromptRole{RoleVisualDirector, RoleVisualDirectorAI} {
		if text := aiTemplate(t, role); !strings.Contains(text, "MỖI CẢNH BẮT BUỘC có một vai trò màu nền") {
			t.Errorf("%s does not require a background role per scene", role)
		}
	}
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

// --- the AI flow's prompts ---------------------------------------------

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
		"frame": true, "frame_width": true, "frame_height": true, "safe_area": true, "frame_rules": true,
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
	for _, want := range []string{`"scenes"`, `"palette"`, `"lines"`, `"say"`, `"show"`, "1 đến 4 câu mỗi shot",
		"#RRGGBB", "JSON", `"layout"`, "{{subtitle_zone}}"} {
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
	for _, want := range []string{"KHÔNG viết cả file", "ShotN_M", "LAYOUT", "PALETTE.", "L1. **Vùng an toàn.**", "LottieClip",
		"khoá viết y như danh sách trong tin nhắn (vd. `PALETTE.nenTroi`)", "viết dạng `PALETTE.khoá` với khoá có trong danh sách?",
		"không có `div` nào mang `transform: scale(...)` để tự zoom/lia?",
		"function ShotN_M({duration, lines}: ShotProps)", "ShotProps = {duration: number; lines: number[]}",
		"lineSpan(lines, i, duration)", "trong 12 frame đầu kể từ `lines[i]`"} {
		if !strings.Contains(remo, want) {
			t.Errorf("remotion_engineer_ai lacks %q", want)
		}
	}
	if strings.Contains(remo, "PALETTE_") {
		t.Error("remotion_engineer_ai must not spell a PALETTE_ name: the model copies it into the code")
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

// The director gets rules 12–18 and a list of buildable materials; the
// Manim engineer animates during narration and does not default to title cards.
func TestCinematicRulesAndMotionDuringNarration(t *testing.T) {
	for _, role := range []PromptRole{RoleVisualDirector, RoleVisualDirectorAI} {
		text := aiTemplate(t, role)
		for _, want := range []string{"NHỊP THAY ĐỔI", "CHO NGƯỜI XEM ĐOÁN TRƯỚC", "DIỄN XUẤT BẰNG NÉT MẶT, DÁNG VÀ CHUYỂN ĐỘNG",
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

// The Story Architect picks a video archetype and says so first, while
// every beat id still comes from the chosen format.
func TestStoryArchitectAsksForAnArchetypeAndKeepsTheFormatsBeats(t *testing.T) {
	text := aiTemplate(t, RoleStoryArchitect)
	for _, want := range []string{
		"KIỂU VIDEO: <mã kiểu>", "CẢNH BÁO FORMAT", "{{format_beats}}",
		"{{video_archetypes}}", "kiểu: <mã>", "VIẾT ĐỂ VẼ", "tối đa 20 từ",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("story_architect lacks %q", want)
		}
	}
	if strings.LastIndex(text, "KIỂU VIDEO: <mã kiểu>") > strings.LastIndex(text, "TÌNH HUỐNG ỨNG VIÊN:") {
		t.Error("the KIỂU VIDEO line must come before the rest of the output")
	}
}
