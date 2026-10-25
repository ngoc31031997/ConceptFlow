package domain

import (
	"errors"
	"strings"
	"testing"
)

func knownF01(id string) bool { return id == "F01" || id == "F02" }

func TestParseScriptReplyReadsLinesAndNormalisesFactCodes(t *testing.T) {
	got, err := ParseScriptReply("{\"lines\":[{\"text\":\" Câu một. \",\"facts\":[\"f01\",\"F01\"]},{\"text\":\"Câu hai.\",\"facts\":[]}]}", knownF01)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Text != "Câu một." || strings.Join(got[0].Facts, ",") != "F01" {
		t.Errorf("got %+v, want trimmed text and each fact once", got)
	}
}

func TestParseScriptReplyRefusesWhatCannotBeStored(t *testing.T) {
	for name, reply := range map[string]string{
		"no JSON":      "xin lỗi",
		"broken JSON":  `{"lines":[`,
		"no lines":     `{"lines":[]}`,
		"empty text":   `{"lines":[{"text":"  ","facts":[]}]}`,
		"unknown fact": `{"lines":[{"text":"Câu.","facts":["F99"]}]}`,
		"bad fact":     `{"lines":[{"text":"Câu.","facts":["abc"]}]}`,
		"too long":     `{"lines":[{"text":"` + strings.Repeat("a", MaxLineRunes+1) + `","facts":[]}]}`,
	} {
		if _, err := ParseScriptReply(reply, knownF01); !errors.Is(err, ErrScriptReplyUnusable) {
			t.Errorf("%s: err = %v, want ErrScriptReplyUnusable", name, err)
		}
	}
}

func TestParseLineSuggestion(t *testing.T) {
	if text, err := ParseLineSuggestion(`{"text":" Câu thay. "}`); err != nil || text != "Câu thay." {
		t.Errorf("text=%q err=%v", text, err)
	}
	if _, err := ParseLineSuggestion(`{"text":""}`); !errors.Is(err, ErrScriptReplyUnusable) {
		t.Errorf("err = %v, want ErrScriptReplyUnusable", err)
	}
}

func TestScriptBeatValidateRejectsDuplicateIDsAndUnknownFacts(t *testing.T) {
	ok := ScriptBeat{Lines: []ScriptLine{{ID: "b1.1", Text: "Một.", Facts: []string{"F01"}}, {ID: "b1.2", Text: "Hai.", Facts: []string{}}}, NextSeq: 3}
	if err := ok.Validate(knownF01); err != nil {
		t.Errorf("valid beat: %v", err)
	}
	dup := ScriptBeat{Lines: []ScriptLine{{ID: "b1.1", Text: "Một."}, {ID: "b1.1", Text: "Hai."}}}
	if dup.Validate(nil) == nil {
		t.Error("a repeated line id must be refused")
	}
	bad := ScriptBeat{Lines: []ScriptLine{{ID: "x1", Text: "Một."}}}
	if bad.Validate(nil) == nil {
		t.Error("an id that is not <beat>.<n> must be refused")
	}
	unknown := ScriptBeat{Lines: []ScriptLine{{ID: "b1.1", Text: "Một.", Facts: []string{"F99"}}}}
	if unknown.Validate(knownF01) == nil {
		t.Error("an unknown fact must be refused")
	}
}

func TestScriptLineBeat(t *testing.T) {
	if key, ok := ScriptLineBeat("b12.3"); !ok || key != "b12" {
		t.Errorf("key=%q ok=%v", key, ok)
	}
	for _, id := range []string{"", "b1", "b0.1", "b1.0", "1.1", "b1.1.1"} {
		if _, ok := ScriptLineBeat(id); ok {
			t.Errorf("%q must not be a line id", id)
		}
	}
}

func TestFlagLineTable(t *testing.T) {
	long := strings.TrimSpace(strings.Repeat("từ ", MaxLineWords+1))
	cases := []struct {
		text, kind string
	}{
		{"Quả sung chín vào mùa thu.", ""},
		{"Ong sung dài chừng hai milimét.", ""},
		{long, LineFlagLong},
		{strings.TrimSpace(strings.Repeat("từ ", MaxLineWords)), ""},
		{"Ong (loài nhỏ) bay vào.", LineFlagHard},
		{"Chất CO2 thoát ra.", LineFlagHard},
		{"Nó nặng 12345 gam.", LineFlagHard},
		{"Các nhà khoa học đo bằng DNA.", LineFlagHard},
		{"Tăng 20% mỗi năm.", LineFlagHard},
		{"Năm 2024 là mốc.", ""},
	}
	for _, c := range cases {
		got := FlagLine(c.text)
		if c.kind == "" && got != nil {
			t.Errorf("%q flagged: %+v", c.text, got)
		}
		if c.kind != "" && (got == nil || got.Kind != c.kind) {
			t.Errorf("%q: flag = %+v, want %s", c.text, got, c.kind)
		}
	}
}

func TestCheckOpening(t *testing.T) {
	short := []ScriptLine{{ID: "b1.1", Text: "Một câu ngắn."}}
	if c := CheckOpening(short, []float64{4}); !c.OK {
		t.Errorf("a short first line is a good opening: %+v", c)
	}
	long := []ScriptLine{{ID: "b1.1", Text: "Một câu rất dài kể lể."}, {ID: "b1.2", Text: "Vì sao vậy?"}}
	if c := CheckOpening(long, []float64{9, 3}); c.OK {
		t.Errorf("a question that starts after 7 seconds is too late: %+v", c)
	}
	asked := []ScriptLine{{ID: "b1.1", Text: "Một câu dài hơn bảy giây."}, {ID: "b1.2", Text: "Vì sao vậy?"}}
	if c := CheckOpening(asked, []float64{8, 3}); c.OK {
		t.Errorf("the second line starts at 8 s: %+v", c)
	}
	early := []ScriptLine{{ID: "b1.1", Text: "Ong là ai?"}, {ID: "b1.2", Text: "Một câu rất dài dòng."}}
	if c := CheckOpening(early, []float64{8, 3}); !c.OK {
		t.Errorf("a question inside the first seconds counts: %+v", c)
	}
	if c := CheckOpening(nil, nil); c.OK {
		t.Error("no lines, no opening")
	}
}

func TestCheckClosing(t *testing.T) {
	if c := CheckClosing(&ScriptLine{ID: "b3.2", Text: "Đó là vòng đời của sung."}); !c.OK {
		t.Errorf("%+v", c)
	}
	if c := CheckClosing(&ScriptLine{ID: "b3.2", Text: "Đó là vòng đời của sung"}); c.OK {
		t.Errorf("a closing line without a sentence mark: %+v", c)
	}
	if c := CheckClosing(nil); c.OK {
		t.Error("no lines, no closing")
	}
}

func TestFlagJoin(t *testing.T) {
	if j := FlagJoin("Ong vào quả sung.", "Trong quả có hoa."); j != nil {
		t.Errorf("a smooth join flagged: %+v", j)
	}
	if j := FlagJoin("Ong vào quả sung.", "Ong mất cánh."); j == nil {
		t.Error("the same opening word twice must be flagged")
	}
	if j := FlagJoin("Ong vào quả sung", "Trong quả có hoa."); j == nil {
		t.Error("a previous line without a sentence mark must be flagged")
	}
	if j := FlagJoin("", "Trong quả."); j != nil {
		t.Error("nothing to join, nothing to flag")
	}
}

func TestParseScriptReplyReadsTheSpokenWording(t *testing.T) {
	got, err := ParseScriptReply(`{"lines":[{"text":"D = (1 - m) / p","spoken":" Đê bằng một trừ mờ, rồi chia cho bê. ","facts":[]},{"text":"Câu hai.","facts":[]}]}`, knownF01)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Spoken != "Đê bằng một trừ mờ, rồi chia cho bê." || got[1].Spoken != "" {
		t.Errorf("got %+v, want the trimmed spoken wording only where the AI gave one", got)
	}
	long := `{"lines":[{"text":"Câu.","spoken":"` + strings.Repeat("a", MaxLineRunes+1) + `","facts":[]}]}`
	if _, err := ParseScriptReply(long, knownF01); !errors.Is(err, ErrScriptReplyUnusable) {
		t.Errorf("err = %v, want a too-long spoken wording refused", err)
	}
}

func TestSpokenTextFallsBackToTheWrittenText(t *testing.T) {
	l := ScriptLine{Text: "D = (1 - m) / p"}
	if l.SpokenText() != l.Text {
		t.Errorf("SpokenText = %q, want the text when Spoken is empty", l.SpokenText())
	}
	l.Spoken = " Đê bằng một trừ mờ, rồi chia cho bê. "
	if want := "Đê bằng một trừ mờ, rồi chia cho bê."; l.SpokenText() != want {
		t.Errorf("SpokenText = %q, want %q", l.SpokenText(), want)
	}
	if FlagLine(l.SpokenText()) != nil {
		t.Error("a line read from words must not be flagged for the symbols it shows")
	}
}

func TestScriptSHAChangesWhenOnlyTheSpokenWordingChanges(t *testing.T) {
	a := ScriptSHA([][]ScriptLine{{{ID: "b1.1", Text: "D = 1"}}})
	b := ScriptSHA([][]ScriptLine{{{ID: "b1.1", Text: "D = 1", Spoken: "Đê bằng một"}}})
	if a == b {
		t.Error("the reading is part of the script a review judged")
	}
}
