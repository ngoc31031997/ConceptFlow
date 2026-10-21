package domain

import (
	"errors"
	"strings"
	"testing"
)

const beatsReply = "```json\n" + `{"beats":[{"key":"","fn":"Hook","title":"Quả sung","goal":"Gây tò mò","content":"Mở từ đĩa sung","facts":["f1"," F02 ","F1"],"dur":18,"next":"Zoom vào"},{"key":"b7","fn":"Payoff","title":"Ai nuôi ai","goal":"Trả lời","content":"Hai loài nương nhau","facts":[],"dur":30,"next":""}]}` + "\n```"

func TestParseBeatsReplyReadsBeatsInOrder(t *testing.T) {
	drafts, err := ParseBeatsReply(beatsReply, knownFacts("F01", "F02"))
	if err != nil {
		t.Fatalf("ParseBeatsReply: %v", err)
	}
	if len(drafts) != 2 || drafts[0].Key != "" || drafts[1].Key != "b7" {
		t.Fatalf("drafts = %+v, want two beats keyed \"\" and b7", drafts)
	}
	if got := strings.Join(drafts[0].Facts, ","); got != "F01,F02" {
		t.Errorf("facts = %q, want F01,F02 (normalised, each once)", got)
	}
}

func TestParseBeatsReplyRejectsUnusableReplies(t *testing.T) {
	cases := map[string]string{
		"no json":          "xin lỗi",
		"broken json":      `{"beats":[`,
		"no beats":         `{"beats":[]}`,
		"unknown function": strings.Replace(beatsReply, `"Hook"`, `"Mở bài"`, 1),
		"unknown fact":     strings.Replace(beatsReply, `" F02 "`, `"F09"`, 1),
		"bad key":          strings.Replace(beatsReply, `"b7"`, `"x7"`, 1),
		"missing goal":     strings.Replace(beatsReply, `"goal":"Gây tò mò"`, `"goal":""`, 1),
		"too short":        strings.Replace(beatsReply, `"dur":18`, `"dur":2`, 1),
	}
	for name, reply := range cases {
		if _, err := ParseBeatsReply(reply, knownFacts("F01", "F02")); !errors.Is(err, ErrBeatReplyUnusable) {
			t.Errorf("%s: err = %v, want ErrBeatReplyUnusable", name, err)
		}
	}
}

func TestParseBeatsReplyRejectsRepeatedKeys(t *testing.T) {
	reply := strings.Replace(beatsReply, `"key":""`, `"key":"b7"`, 1)
	if _, err := ParseBeatsReply(reply, knownFacts("F01", "F02")); !errors.Is(err, ErrBeatReplyUnusable) {
		t.Errorf("err = %v, want a repeated key to be refused", err)
	}
}

func TestBeatKeyFollowsSequence(t *testing.T) {
	if BeatKey(3) != "b3" || !IsBeatKey("b12") || IsBeatKey("b0") || IsBeatKey("b") || IsBeatKey("c1") || IsBeatKey("b1x") {
		t.Error("BeatKey and IsBeatKey disagree about the form b1, b2...")
	}
}

func TestTargetSecondsFollowsTheTarget(t *testing.T) {
	if lo, hi := TargetSeconds(ModeLongOnly, Duration4to5); lo != 240 || hi != 300 {
		t.Errorf("4-5 phút = %d..%d, want 240..300", lo, hi)
	}
	if lo, hi := TargetSeconds(ModeShortOnly, ""); lo != 0 || hi != 60 {
		t.Errorf("short = %d..%d, want 0..60", lo, hi)
	}
}

func TestCheckBeatSheetFlagsWhatIsOff(t *testing.T) {
	beats := []Beat{
		{Fn: "Hook", Goal: "a", Dur: 25, Facts: []string{"F01"}},
		{Fn: "Cao trào", Goal: "", Dur: 100, Facts: []string{}},
	}
	checks := CheckBeatSheet(beats, []string{"F01", "F02"}, 240, 300, "4–5 phút")
	want := []bool{false, false, false, false, false}
	for i, c := range checks {
		if c.OK != want[i] {
			t.Errorf("check %d (%q) ok = %v, want %v", i, c.Text, c.OK, want[i])
		}
	}
	if !strings.Contains(checks[2].Text, "thiếu F02") || !strings.Contains(checks[0].Text, "2:05") {
		t.Errorf("texts = %q / %q, want the total and the missing fact named", checks[0].Text, checks[2].Text)
	}
}

func TestCheckBeatSheetPassesAGoodSheet(t *testing.T) {
	beats := []Beat{
		{Fn: "Hook", Goal: "a", Dur: 20, Facts: []string{"F01"}},
		{Fn: "Payoff", Goal: "b", Dur: 230, Facts: []string{"F02"}},
	}
	for _, c := range CheckBeatSheet(beats, []string{"F01", "F02"}, 240, 300, "4–5 phút") {
		if !c.OK {
			t.Errorf("check %q failed on a good sheet", c.Text)
		}
	}
}

func TestBeatSheetRoleIsShippedAndTakesItsValues(t *testing.T) {
	if !ValidPromptRole(string(RoleBeatSheet)) || IsRenderablePromptRole(string(RoleBeatSheet)) {
		t.Fatal("beat_sheet must be a known role filled by its own use case")
	}
	out := RenderPromptTemplate(seededText(t, RoleBeatSheet), map[string]string{
		"topic": "ĐỀ-TÀI", "brief_language": "tiếng Việt", "target": "4–5 phút", "brief": "BRIEF",
		"story": "STORY", "facts": "FACTS", "current_beats": "b1 (B1)", "kept_beats": "- b1: goal", "redo_note": "GHI-CHÚ",
		"task": BeatTaskAdd("b1", "b2"),
	})
	if strings.Contains(out, "{{") {
		t.Errorf("an unfilled placeholder is left in the beat prompt: %q", out[strings.Index(out, "{{"):])
	}
	for _, want := range []string{"ĐỀ-TÀI", "4–5 phút", "GHI-CHÚ", "giữa beat b1 và beat b2", `{"beats"`} {
		if !strings.Contains(out, want) {
			t.Errorf("beat prompt lacks %q", want)
		}
	}
	if !strings.Contains(BeatTaskAdd("b3", ""), "cuối sau beat b3") || !strings.Contains(BeatTaskRedo("b2"), "khoá b2") {
		t.Error("the add-last and redo tasks must name their beats")
	}
}
