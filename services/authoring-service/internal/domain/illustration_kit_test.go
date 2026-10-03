package domain

import (
	"strings"
	"testing"
)

// The director must ask for pictures of what the narration talks
// about, not stand-in shapes; it must still not name an engine's API.
func TestVisualDirectorIllustratesWhatIsSaid(t *testing.T) {
	for _, role := range []PromptRole{RoleVisualDirector, RoleVisualDirectorAI} {
		text := aiTemplate(t, role)
		for _, want := range []string{"MINH HOẠ ĐÚNG CÁI ĐANG NÓI", "TRANH PHẲNG KỂ CHUYỆN", "Tắt tiếng", "MỘT MÀU NỀN SÁNG, BÃO HOÀ", "MỖI CẢNH BẮT BUỘC có một vai trò màu nền", "chiếc răng"} {
			if !strings.Contains(text, want) {
				t.Errorf("%s lacks %q", role, want)
			}
		}
		// The old rule 18 steered every concrete subject into abstract shapes.
		if strings.Contains(text, "nhân vật hữu cơ có cử động phức tạp") || strings.Contains(text, "Nền video CỐ ĐỊNH") {
			t.Errorf("%s still carries the shapes-only / fixed-background rule", role)
		}
		for _, engineAPI := range []string{"conceptflow-mini", "<Person", "Backdrop"} {
			if strings.Contains(text, engineAPI) {
				t.Errorf("%s names engine API %q; the director stays engine-agnostic", role, engineAPI)
			}
		}
	}
}

// Both Remotion roles receive the kit, expanded at seed time.
func TestRemotionEngineersGetTheIllustrationKit(t *testing.T) {
	for _, role := range []PromptRole{RoleRemotionEngineer, RoleRemotionEngineerAI} {
		text := aiTemplate(t, role)
		if strings.Contains(text, "{{illustration_kit}}") {
			t.Fatalf("%s: {{illustration_kit}} was not expanded at seed time", role)
		}
		for _, want := range []string{"## C3. BỘ MINH HOẠ PHẲNG", "<Person", "<Tooth", "<Germ", "<Backdrop", "BẮT BUỘC dùng component của bộ"} {
			if !strings.Contains(text, want) {
				t.Errorf("%s lacks %q", role, want)
			}
		}
		if strings.Contains(text, "KHÔNG tô nền cho khung hình hay cho") {
			t.Errorf("%s still forbids every background, including Backdrop", role)
		}
		if strings.Contains(text, "¤") {
			t.Errorf("%s still contains the ¤ backtick stand-in", role)
		}
	}
}
