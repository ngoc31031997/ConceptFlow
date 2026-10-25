package domain

import (
	"errors"
	"regexp"
	"strings"
	"testing"
)

var placeholderRe = regexp.MustCompile(`\{\{([a-z_]+)\}\}`)

// aiTemplate returns the shipped wording of role.
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

const fullBriefJSON = `{"audience":"Người xem phổ thông","know":"Sung là chùm hoa kín","understand":"Vòng đời ong gắn với quả","feel":"Ngạc nhiên","tone":"Tò mò, ấm","out_of_scope":"Không tư vấn ăn uống"}`

func TestParseBriefReplyReadsTheJSONObject(t *testing.T) {
	brief, err := ParseBriefReply(fullBriefJSON)
	if err != nil {
		t.Fatalf("a reply with every field should parse: %v", err)
	}
	if brief.Audience != "Người xem phổ thông" || brief.OutOfScope != "Không tư vấn ăn uống" {
		t.Errorf("fields read wrong: %+v", brief)
	}
}

func TestParseBriefReplyRefusesAReplyMissingAField(t *testing.T) {
	_, err := ParseBriefReply(`{"audience":"A","know":"K","understand":"U","feel":"","tone":"T"}`)
	if !errors.Is(err, ErrBriefReplyUnusable) {
		t.Fatalf("a reply with an empty and a missing field must be unusable, got %v", err)
	}
	if !strings.Contains(err.Error(), "người xem sẽ cảm thấy") || !strings.Contains(err.Error(), "Ngoài phạm vi") {
		t.Errorf("the error must name the missing fields as the screen labels them, got %q", err)
	}
}

func TestParseBriefReplyRefusesAReplyWithoutJSON(t *testing.T) {
	if _, err := ParseBriefReply("Xin lỗi, tôi không làm được."); !errors.Is(err, ErrBriefReplyUnusable) {
		t.Fatalf("a reply with no JSON must be unusable, got %v", err)
	}
}

func TestParseBriefReplyRefusesAnUnescapedQuoteInsideAField(t *testing.T) {
	reply := `{"audience":"Người "xem" phổ thông","know":"b","understand":"c","feel":"d","tone":"e","out_of_scope":"f"}`
	if _, err := ParseBriefReply(reply); !errors.Is(err, ErrBriefReplyUnusable) {
		t.Fatalf("an unescaped quote inside a field must be unusable, got %v", err)
	}
}

func TestBriefPromptForbidsStraightQuotesInsideFields(t *testing.T) {
	if !strings.Contains(creativeBriefVI, `KHÔNG dùng dấu nháy kép thẳng`) {
		t.Fatal("the brief prompt must tell the model not to put straight double quotes inside field text")
	}
}

func TestBriefPromptRanksTheCreatorsNoteAboveTheCurrentBrief(t *testing.T) {
	for _, want := range []string{"ƯU TIÊN HƠN brief hiện có", "mọi ô khác bị ảnh hưởng", "không được bỏ qua ghi chú"} {
		if !strings.Contains(creativeBriefVI, want) {
			t.Fatalf("the brief prompt must tell the model that the note outranks the current brief: missing %q", want)
		}
	}
}

func TestKeepCreatorFieldsKeepsOnlyTheCreatorsFields(t *testing.T) {
	written := CreativeBrief{Know: "AI K", Audience: "AI A", Tone: "AI T"}
	current := CreativeBrief{Know: "Creator K", Audience: "old A", Tone: "Creator T"}
	got := KeepCreatorFields(written, current, []string{"know", "tone"})
	if got.Know != "Creator K" || got.Tone != "Creator T" {
		t.Errorf("the Creator's fields must survive the redo, got %+v", got)
	}
	if got.Audience != "AI A" {
		t.Errorf("a field the Creator did not write takes the AI's text, got %q", got.Audience)
	}
}

func TestTargetDurationLabel(t *testing.T) {
	cases := []struct {
		mode     VideoOutputMode
		d        TargetDuration
		language string
		want     string
	}{
		{ModeLongOnly, Duration5to7, "vi", "5–7 phút"},
		{ModeLongOnly, Duration7to10, "en", "7–10 minutes"},
		{ModeLongOnly, "", "vi", "4–5 phút"},
		{ModeShortOnly, Duration5to7, "vi", "dưới 60 giây"},
		{ModeShortOnly, "", "en", "under 60 seconds"},
	}
	for _, c := range cases {
		if got := TargetDurationLabel(c.mode, c.d, c.language); got != c.want {
			t.Errorf("TargetDurationLabel(%q, %q, %q) = %q, want %q", c.mode, c.d, c.language, got, c.want)
		}
	}
}

func TestBriefSectionListsEveryFieldWithItsLabel(t *testing.T) {
	brief, err := ParseBriefReply(fullBriefJSON)
	if err != nil {
		t.Fatal(err)
	}
	idea := Idea{Question: "Vì sao ong chết trong quả sung?", Why: "w", Known: "k", Unknown: "u"}
	section := BriefSection(idea, brief, "4–5 phút", "YouTube")
	for _, want := range []string{
		"Câu hỏi trung tâm cần trả lời: Vì sao ong chết trong quả sung?", "Thời lượng mục tiêu: 4–5 phút",
		"Ngoài phạm vi (video không nói tới): Không tư vấn ăn uống", "Nền tảng phát hành: YouTube",
	} {
		if !strings.Contains(section, want) {
			t.Errorf("the brief section lacks %q:\n%s", want, section)
		}
	}
}

func TestCreativeBriefPromptUsesOnlyTheVariablesTheUseCaseFills(t *testing.T) {
	known := map[string]bool{
		"topic": true, "idea": true, "video_kind": true, "target_duration": true, "platform": true,
		"brief_language": true, "current_brief": true, "kept_fields": true, "redo_note": true,
	}
	text := aiTemplate(t, RoleCreativeBrief)
	for _, m := range placeholderRe.FindAllStringSubmatch(text, -1) {
		if !known[m[1]] {
			t.Errorf("creative_brief uses {{%s}}, which the brief use case does not fill", m[1])
		}
	}
	for _, field := range BriefFields {
		if !strings.Contains(text, `"`+field+`"`) {
			t.Errorf("the creative_brief prompt does not ask for the %q key", field)
		}
	}
}

func TestDefaultPlatformFollowsTheVideoForm(t *testing.T) {
	if got := DefaultPlatform(ModeShortOnly); got != PlatformYouTubeShorts {
		t.Errorf("a short defaults to YouTube Shorts, got %q", got)
	}
	if got := DefaultPlatform(ModeLongOnly); got != PlatformYouTube {
		t.Errorf("a long video defaults to YouTube, got %q", got)
	}
	if ValidPlatform("TikTok") || !ValidPlatform(PlatformWebsite) {
		t.Error("only the listed platforms are valid")
	}
}
