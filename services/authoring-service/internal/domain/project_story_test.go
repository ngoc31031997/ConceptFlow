package domain

import (
	"errors"
	"strings"
	"testing"
)

const storyReply = "```json\n" + `{"central_question":"Điều gì xảy ra khi ta rơi vào hố đen?","core_answer":"Người rơi và người ở xa nhận thông tin khác nhau.","narrative_arc":[
{"beat":"Hook","text":"Người ngoài có thấy ta vượt ranh giới không?","facts":[]},
{"beat":"Hai góc nhìn","text":"Người rơi và người ở xa.","facts":["f1","F02"]},
{"beat":"Payoff","text":"Tổng hợp điều mỗi góc nhìn biết.","facts":["F01"]}]}` + "\n```"

func TestParseStoryReplyReadsTheThreeFields(t *testing.T) {
	story, err := ParseStoryReply(storyReply, knownFacts("F01", "F02"))
	if err != nil {
		t.Fatalf("ParseStoryReply: %v", err)
	}
	if story.CentralQuestion == "" || story.CoreAnswer == "" || len(story.NarrativeArc) != 3 {
		t.Fatalf("story = %+v, want both texts and three beats", story)
	}
	if got := strings.Join(story.NarrativeArc[1].Facts, ","); got != "F01,F02" {
		t.Errorf("beat 2 facts = %s, want the codes in canonical form F01,F02", got)
	}
	if got := strings.Join(story.CitedFacts(), ","); got != "F01,F02" {
		t.Errorf("cited facts = %s, want each code once", got)
	}
}

func TestParseStoryReplyAcceptsAnyBeatCountFromThreeToTwelve(t *testing.T) {
	beats := make([]string, 0, MaxArcBeats)
	for i := 0; i < MaxArcBeats; i++ {
		beats = append(beats, `{"beat":"Nhịp","text":"Nội dung","facts":[]}`)
	}
	for _, n := range []int{MinArcBeats, 7, MaxArcBeats} {
		reply := `{"central_question":"Q","core_answer":"A","narrative_arc":[` + strings.Join(beats[:n], ",") + `]}`
		story, err := ParseStoryReply(reply, nil)
		if err != nil || len(story.NarrativeArc) != n {
			t.Errorf("%d beats: got %d beats, err %v, want them kept", n, len(story.NarrativeArc), err)
		}
	}
}

func TestParseStoryReplyRefusesAnUnusableReply(t *testing.T) {
	two := `{"beat":"Nhịp","text":"Nội dung","facts":[]}`
	cases := map[string]string{
		"no json":       "xin lỗi, tôi không làm được",
		"broken json":   `{"central_question":`,
		"no question":   `{"central_question":"","core_answer":"A","narrative_arc":[` + two + `,` + two + `,` + two + `]}`,
		"no answer":     `{"central_question":"Q","core_answer":" ","narrative_arc":[` + two + `,` + two + `,` + two + `]}`,
		"too few beats": `{"central_question":"Q","core_answer":"A","narrative_arc":[` + two + `,` + two + `]}`,
		"beat no text":  `{"central_question":"Q","core_answer":"A","narrative_arc":[` + two + `,` + two + `,{"beat":"Payoff","text":"","facts":[]}]}`,
		"unknown fact":  `{"central_question":"Q","core_answer":"A","narrative_arc":[` + two + `,` + two + `,{"beat":"P","text":"T","facts":["F09"]}]}`,
		"not a fact id": `{"central_question":"Q","core_answer":"A","narrative_arc":[` + two + `,` + two + `,{"beat":"P","text":"T","facts":["xyz"]}]}`,
		"empty arc":     `{"central_question":"Q","core_answer":"A","narrative_arc":[]}`,
	}
	for name, reply := range cases {
		if _, err := ParseStoryReply(reply, knownFacts("F01")); !errors.Is(err, ErrStoryReplyUnusable) {
			t.Errorf("%s: err = %v, want ErrStoryReplyUnusable", name, err)
		}
	}
}

func TestParseStoryReplyRefusesACitationWhenThereIsNoFactTable(t *testing.T) {
	two := `{"beat":"Nhịp","text":"Nội dung","facts":[]}`
	reply := `{"central_question":"Q","core_answer":"A","narrative_arc":[` + two + `,` + two + `,{"beat":"P","text":"T","facts":["F01"]}]}`
	if _, err := ParseStoryReply(reply, nil); !errors.Is(err, ErrStoryReplyUnusable) {
		t.Errorf("err = %v, want ErrStoryReplyUnusable when no fact is known", err)
	}
}

func TestStoryValidateNamesWhatIsMissing(t *testing.T) {
	err := StoryStructure{CoreAnswer: "A"}.Validate()
	if !errors.Is(err, ErrStoryInvalid) || !strings.Contains(err.Error(), "Câu hỏi trung tâm") || !strings.Contains(err.Error(), "Mạch kể") {
		t.Errorf("err = %v, want one naming the empty question and arc", err)
	}
}

func TestStoryHashIgnoresWhitespaceAndFactOrderDuplicates(t *testing.T) {
	a := StoryStructure{CentralQuestion: " Q ", CoreAnswer: "A", NarrativeArc: []ArcBeat{{Beat: "H", Text: "T", Facts: []string{"F01", "F01"}}}}
	b := StoryStructure{CentralQuestion: "Q", CoreAnswer: "A ", NarrativeArc: []ArcBeat{{Beat: "H", Text: "T ", Facts: []string{"F01"}}}}
	if a.Hash() != b.Hash() {
		t.Error("equal stories hash differently")
	}
	b.NarrativeArc[0].Text = "Khác"
	if a.Hash() == b.Hash() {
		t.Error("a changed beat keeps the hash")
	}
}

func TestKeepStoryCreatorFieldsCopiesOnlyTheirs(t *testing.T) {
	current := StoryStructure{CentralQuestion: "Của tôi", CoreAnswer: "Cũ", NarrativeArc: []ArcBeat{{Beat: "X", Text: "Y"}}}
	written := StoryStructure{CentralQuestion: "AI", CoreAnswer: "AI", NarrativeArc: []ArcBeat{{Beat: "Mới", Text: "Mới"}}}
	got := KeepStoryCreatorFields(written, current, []string{StoryFieldCentralQuestion})
	if got.CentralQuestion != "Của tôi" || got.CoreAnswer != "AI" || got.NarrativeArc[0].Beat != "Mới" {
		t.Errorf("got %+v, want only the question kept from the Creator", got)
	}
}

func TestBeatFieldsPresentTheStoryToTheBeatSheet(t *testing.T) {
	s := StoryStructure{CentralQuestion: "Q", CoreAnswer: "A", NarrativeArc: []ArcBeat{
		{Beat: "Hook", Text: "Một", Facts: []string{"F01"}}, {Beat: "Giữa", Text: "Hai"}, {Beat: "Payoff", Text: "Ba", Facts: []string{"F02"}},
	}}
	f := s.BeatFields()
	if f[BeatFieldQuestion].Text != "Q" || f[BeatFieldPayoff].Text != "A" {
		t.Errorf("question/payoff = %q/%q, want Q/A", f[BeatFieldQuestion].Text, f[BeatFieldPayoff].Text)
	}
	if want := "Một [F01]\nHai\nBa [F02]"; f[BeatFieldChain].Text != want {
		t.Errorf("chain = %q, want %q", f[BeatFieldChain].Text, want)
	}
	if got := strings.Join(f[BeatFieldChain].Facts, ","); got != "F01,F02" {
		t.Errorf("chain facts = %s, want F01,F02", got)
	}
}

func TestResearchHashChangesWithKeptFactsOnly(t *testing.T) {
	kept := StoredFact{ID: "F01", Fact: Fact{ClaimVI: "Một", Type: FactTypeFact, Keep: true}}
	dropped := StoredFact{ID: "F02", Fact: Fact{ClaimVI: "Hai", Type: FactTypeFact, Keep: false}}
	base := ResearchHash([]StoredFact{kept})
	if ResearchHash([]StoredFact{kept, dropped}) != base {
		t.Error("a dropped fact changes the hash")
	}
	kept.Fact.ClaimVI = "Một khác"
	if ResearchHash([]StoredFact{kept}) == base {
		t.Error("a changed claim keeps the hash")
	}
}
