package domain

import (
	"errors"
	"strings"
	"testing"
)

func knownFacts(ids ...string) func(string) bool {
	set := map[string]bool{}
	for _, id := range ids {
		set[id] = true
	}
	return func(id string) bool { return set[id] }
}

func TestParseRefLineReadsTrailingFactCodes(t *testing.T) {
	text, refs, err := ParseRefLine("Sung là chùm hoa kín. [f1, F02]", knownFacts("F01", "F02"))
	if err != nil {
		t.Fatalf("ParseRefLine: %v", err)
	}
	if text != "Sung là chùm hoa kín." || strings.Join(refs, ",") != "F01,F02" {
		t.Errorf("got %q %v, want the text without the brackets and codes F01,F02", text, refs)
	}
}

func TestParseRefLineKeepsBracketsThatAreNotFactCodes(t *testing.T) {
	text, refs, err := ParseRefLine("Ong [sic] chết", nil)
	if err != nil || text != "Ong [sic] chết" || len(refs) != 0 {
		t.Errorf("got %q %v %v, want the line untouched", text, refs, err)
	}
	text, refs, err = ParseRefLine("Nói về ong [thật ra là kiến]", nil)
	if err != nil || text != "Nói về ong [thật ra là kiến]" || len(refs) != 0 {
		t.Errorf("got %q %v %v, want a bracket of words to stay in the text", text, refs, err)
	}
}

func TestParseRefLineRejectsUnknownFact(t *testing.T) {
	if _, _, err := ParseRefLine("Một câu [F09]", knownFacts("F01")); err == nil || !strings.Contains(err.Error(), "F09") {
		t.Errorf("err = %v, want one naming F09", err)
	}
}

func TestNewStoryFieldChainDerivesFactsFromEachLine(t *testing.T) {
	f, err := NewStoryField("chain", "Bước một [F01]\n\nBước hai [F02, F01]\nBước ba", knownFacts("F01", "F02"))
	if err != nil {
		t.Fatalf("NewStoryField: %v", err)
	}
	if f.Text != "Bước một [F01]\nBước hai [F02, F01]\nBước ba" {
		t.Errorf("text = %q, want blank lines dropped", f.Text)
	}
	if strings.Join(f.Facts, ",") != "F01,F02" {
		t.Errorf("facts = %v, want F01,F02 once each", f.Facts)
	}
}

func TestNewStoryFieldChainRejectsTooManySteps(t *testing.T) {
	lines := strings.Repeat("một bước\n", MaxStoryChain+1)
	if _, err := NewStoryField("chain", lines, nil); err == nil {
		t.Error("a chain of more than the maximum steps must be refused")
	}
}

func TestNewStoryFieldPlainFieldsCiteNothing(t *testing.T) {
	f, err := NewStoryField("logline", "  Một câu [F01]  ", knownFacts("F01"))
	if err != nil {
		t.Fatalf("NewStoryField: %v", err)
	}
	if f.Text != "Một câu [F01]" || len(f.Facts) != 0 {
		t.Errorf("got %+v, want the text kept whole and no facts", f)
	}
	if _, err := NewStoryField("nothing", "x", nil); err == nil {
		t.Error("an unknown field key must be refused")
	}
}

const frameReply = "```json\n" + `{"question":"Vì sao?","promise":"Bạn sẽ biết.","chain":[{"text":"Sung kín","facts":["F01"]},{"text":"Ong chui vào","facts":["F02","F01"]}],"payoff":"Ong là cái giá","payoff_facts":["F02"],"logline":"Một con ong.","summary":"Mở từ đĩa sung.","used_facts":["F03"],"facts":[]}` + "\n```"

func TestParseStoryReplyReadsEveryField(t *testing.T) {
	r, err := ParseStoryReply(frameReply, StoryFieldKeys, knownFacts("F01", "F02", "F03"))
	if err != nil {
		t.Fatalf("ParseStoryReply: %v", err)
	}
	if got := r.Fields["chain"].Text; got != "Sung kín [F01]\nOng chui vào [F02, F01]" {
		t.Errorf("chain = %q", got)
	}
	if got := r.Fields["payoff"]; got.Text != "Ong là cái giá [F02]" || strings.Join(got.Facts, ",") != "F02" {
		t.Errorf("payoff = %+v", got)
	}
	if strings.Join(r.UsedFacts, ",") != "F03" {
		t.Errorf("used facts = %v, want F03", r.UsedFacts)
	}
}

func TestParseStoryReplyReadsOnlyTheFieldsAsked(t *testing.T) {
	r, err := ParseStoryReply(`{"logline":"Một câu."}`, []string{"logline"}, nil)
	if err != nil {
		t.Fatalf("ParseStoryReply: %v", err)
	}
	if len(r.Fields) != 1 || r.Fields["logline"].Text != "Một câu." {
		t.Errorf("fields = %+v, want only the logline", r.Fields)
	}
}

func TestParseStoryReplyRefusesWhatCannotBeStored(t *testing.T) {
	cases := map[string]string{
		"no json":       "xin lỗi, tôi không làm được",
		"broken json":   `{"question": "a" "promise": "b"}`,
		"missing field": `{"question":"a","promise":"b","chain":[{"text":"x","facts":[]}],"payoff":"p","payoff_facts":[],"logline":"l"}`,
		"unknown fact":  strings.Replace(frameReply, `"payoff_facts":["F02"]`, `"payoff_facts":["F77"]`, 1),
	}
	for name, reply := range cases {
		_, err := ParseStoryReply(reply, StoryFieldKeys, knownFacts("F01", "F02", "F03"))
		if !errors.Is(err, ErrStoryReplyUnusable) {
			t.Errorf("%s: err = %v, want ErrStoryReplyUnusable", name, err)
		}
	}
}

func TestParseStoryReplyAcceptsFactsTheModelListed(t *testing.T) {
	reply := `{"question":"q","promise":"p","chain":[{"text":"s","facts":["F01"]}],"payoff":"x","payoff_facts":["F01"],"logline":"l","summary":"u","used_facts":[],` +
		`"facts":[{"id":"F01","claim_vi":"Sung là chùm hoa","type":"fact","note":""}]}`
	r, err := ParseStoryReply(reply, StoryFieldKeys, knownFacts())
	if err != nil {
		t.Fatalf("ParseStoryReply: %v", err)
	}
	if len(r.Facts) != 1 || r.Facts[0].ID != "F01" || r.Facts[0].Type != FactTypeFact {
		t.Errorf("facts = %+v, want the one listed fact", r.Facts)
	}
	bad := strings.Replace(reply, `"type":"fact"`, `"type":"rumour"`, 1)
	if _, err := ParseStoryReply(bad, StoryFieldKeys, knownFacts()); !errors.Is(err, ErrStoryReplyUnusable) {
		t.Errorf("err = %v, want a listed fact of an unknown type refused", err)
	}
}

func TestParseApproachesReplyMarksNewKinds(t *testing.T) {
	reply := `{"approaches":[` +
		`{"code":"hành-trình","name":"Hành trình","angle":"đi vào quả","why":"F01 cho cú lật","refs":["F01"],"open":"Quả sung..."},` +
		`{"code":"VÒNG-ĐỜI","name":"Vòng đời","angle":"theo ong mẹ","why":"F04","refs":["f4"],"open":"Câu chuyện...","when_to_use":"chu trình khép kín","playbook":"theo một cá thể"}]}`
	got, err := ParseApproachesReply(reply, []string{"HÀNH-TRÌNH"}, knownFacts("F01", "F04"))
	if err != nil {
		t.Fatalf("ParseApproachesReply: %v", err)
	}
	if len(got) != 2 || got[0].IsNew || !got[1].IsNew {
		t.Fatalf("got %+v, want the table kind first and a new kind second", got)
	}
	if got[0].Code != "HÀNH-TRÌNH" || got[1].Refs[0] != "F04" {
		t.Errorf("codes or refs not normalised: %+v", got)
	}
}

func TestParseApproachesReplyRefusesWhatCannotBeSaved(t *testing.T) {
	cases := map[string]string{
		"no approaches":     `{"approaches":[]}`,
		"unknown fact":      `{"approaches":[{"code":"A","name":"n","angle":"a","why":"w","refs":["F99"],"open":"o"}]}`,
		"missing field":     `{"approaches":[{"code":"A","name":"n","angle":"a","why":"","refs":[],"open":"o"}]}`,
		"new without guide": `{"approaches":[{"code":"MỚI","name":"n","angle":"a","why":"w","refs":[],"open":"o"}]}`,
	}
	for name, reply := range cases {
		if _, err := ParseApproachesReply(reply, []string{"A"}, knownFacts("F01")); !errors.Is(err, ErrApproachReplyUnusable) {
			t.Errorf("%s: err = %v, want ErrApproachReplyUnusable", name, err)
		}
	}
}

func TestStoryRolesAreShippedAndFillTheirOwnValues(t *testing.T) {
	for _, role := range []PromptRole{RoleStoryApproaches, RoleStoryFrame} {
		if !ValidPromptRole(string(role)) || IsRenderablePromptRole(string(role)) {
			t.Errorf("%q must be a known role filled by its own use case", role)
		}
	}
	out := RenderPromptTemplate(seededText(t, RoleStoryFrame), map[string]string{
		"channel_identity": "KÊNH", "topic": "ĐỀ-TÀI", "brief_language": "tiếng Việt", "brief": "BRIEF", "facts": "F01: x",
		"facts_rule": storyFactsGivenVI, "facts_example": StoryFactsExample(false), "approach": "GÓC", "current_frame": "KHUNG", "kept_fields": "GIỮ", "redo_note": "GHI-CHÚ",
		"fields_to_write": "logline",
	})
	if strings.Contains(out, "{{") {
		t.Errorf("an unfilled placeholder is left in the frame prompt: %q", out[strings.Index(out, "{{"):])
	}
	out = RenderPromptTemplate(seededText(t, RoleStoryApproaches), map[string]string{
		"channel_identity": "KÊNH", "topic": "ĐỀ-TÀI", "brief_language": "tiếng Việt", "brief": "BRIEF", "facts": "F01: x",
		"archetypes": "A: x", "redo_note": "GHI-CHÚ", "refs_rule": StoryRefsRule(true), "refs_example": StoryRefsExample(true),
	})
	if strings.Contains(out, "{{") {
		t.Errorf("an unfilled placeholder is left in the approaches prompt: %q", out[strings.Index(out, "{{"):])
	}
}

func TestModelForStoryFrameIsItsOwnChoice(t *testing.T) {
	m := AuthoringStepModels{StoryFrame: "f", Story: "s", Research: "r"}
	if m.ModelFor("story_frame") != "f" || m.ModelFor("story") != "s" {
		t.Errorf("ModelFor wrong for %+v: the story frame must not share the script step's model", m)
	}
}
