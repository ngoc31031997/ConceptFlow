package domain

import (
	"errors"
	"strings"
	"testing"
)

const briefReply = `{"audience":" Khán giả phổ thông từ cấp 2 trở lên. ","duration_seconds":270,"tone":"Tò mò, dễ hiểu, chính xác.","learning_objectives":["Giải thích chân trời sự kiện ở mức khái niệm."," ","giải thích chân trời sự kiện ở mức khái niệm.","Hiểu lực thủy triều phụ thuộc vào điều kiện vật lý."],"avoid":["Mô tả chân trời sự kiện như một lớp vỏ cứng."]}`

func validBrief() ProjectBrief {
	return ProjectBrief{
		Audience: "a", DurationSeconds: 270, Tone: "t",
		LearningObjectives: []string{"o"}, Avoid: []string{"v"},
	}
}

func TestParseProjectBriefReply_ReadsFiveFields(t *testing.T) {
	got, err := ParseProjectBriefReply(briefReply, ModeLongOnly)
	if err != nil {
		t.Fatal(err)
	}
	if got.Audience != "Khán giả phổ thông từ cấp 2 trở lên." || got.DurationSeconds != 270 {
		t.Errorf("audience/duration read wrong: %+v", got)
	}
	if len(got.LearningObjectives) != 2 {
		t.Errorf("objectives = %v, want 2 (blank and repeated dropped)", got.LearningObjectives)
	}
}

func TestParseProjectBriefReply_RefusesWhatIsNotWhole(t *testing.T) {
	cases := map[string]string{
		"no json":          "xin lỗi",
		"missing tone":     `{"audience":"a","duration_seconds":270,"learning_objectives":["o"],"avoid":["v"]}`,
		"duration string":  `{"audience":"a","duration_seconds":"4 phút","tone":"t","learning_objectives":["o"],"avoid":["v"]}`,
		"no avoid":         `{"audience":"a","duration_seconds":270,"tone":"t","learning_objectives":["o"],"avoid":[]}`,
		"short for a long": `{"audience":"a","duration_seconds":30,"tone":"t","learning_objectives":["o"],"avoid":["v"]}`,
	}
	for name, reply := range cases {
		if _, err := ParseProjectBriefReply(reply, ModeLongOnly); !errors.Is(err, ErrProjectBriefReplyUnusable) {
			t.Errorf("%s: err = %v, want ErrProjectBriefReplyUnusable", name, err)
		}
	}
}

func TestProjectBriefValidate_DurationFollowsTheVideoForm(t *testing.T) {
	b := validBrief()
	b.DurationSeconds = 90
	if err := b.Validate(ModeShortOnly); err != nil {
		t.Errorf("90 s short refused: %v", err)
	}
	if err := b.Validate(ModeLongOnly); !errors.Is(err, ErrProjectBriefInvalid) {
		t.Errorf("90 s long: err = %v, want ErrProjectBriefInvalid", err)
	}
	b.DurationSeconds = 270
	if err := b.Validate(ModeShortOnly); !errors.Is(err, ErrProjectBriefInvalid) {
		t.Errorf("270 s short: err = %v, want ErrProjectBriefInvalid", err)
	}
	if err := b.Validate(ModeLongOnly); err != nil {
		t.Errorf("270 s long refused: %v", err)
	}
}

func TestProjectBriefValidate_ListsStayWithinTheirCaps(t *testing.T) {
	many := func(n int) []string {
		var out []string
		for i := 0; i < n; i++ {
			out = append(out, strings.Repeat("x", i+1))
		}
		return out
	}
	b := validBrief()
	b.LearningObjectives = many(MaxLearningObjectives)
	b.Avoid = many(MaxAvoidItems)
	if err := b.Validate(ModeLongOnly); err != nil {
		t.Fatalf("lists at their caps refused: %v", err)
	}
	b.LearningObjectives = many(MaxLearningObjectives + 1)
	if err := b.Validate(ModeLongOnly); !errors.Is(err, ErrProjectBriefInvalid) {
		t.Errorf("too many objectives: err = %v", err)
	}
	b = validBrief()
	b.Avoid = many(MaxAvoidItems + 1)
	if err := b.Validate(ModeLongOnly); !errors.Is(err, ErrProjectBriefInvalid) {
		t.Errorf("too many avoid items: err = %v", err)
	}
	b = validBrief()
	b.Avoid = []string{strings.Repeat("y", MaxBriefItemRunes+1)}
	if err := b.Validate(ModeLongOnly); !errors.Is(err, ErrProjectBriefInvalid) {
		t.Errorf("long avoid item: err = %v", err)
	}
}

func TestProjectBriefHashIgnoresStraySpaces(t *testing.T) {
	a := validBrief()
	b := validBrief()
	b.Audience = "  a "
	b.Avoid = []string{" v ", ""}
	if a.Hash() != b.Hash() {
		t.Error("stray spaces changed the hash")
	}
	b.DurationSeconds = 271
	if a.Hash() == b.Hash() {
		t.Error("a different duration kept the same hash")
	}
}

func TestFormatDuration(t *testing.T) {
	cases := map[int]string{0: "0:00", 59: "0:59", 270: "4:30", 3599: "59:59", 3600: "1:00:00", 3725: "1:02:05"}
	for sec, want := range cases {
		if got := FormatDuration(sec); got != want {
			t.Errorf("FormatDuration(%d) = %q, want %q", sec, got, want)
		}
	}
}

func TestParseBriefReviewReply(t *testing.T) {
	ok := `{"audience_fit":{"rating":"good","comment":"a"},"duration_fit":{"rating":"fair","comment":"b"},"objectives":{"rating":"good","comment":"c"},"avoid_list":{"rating":"weak","comment":"d"},"unclear":["x"," "],"suggestions":[]}`
	got, err := ParseBriefReviewReply(ok)
	if err != nil {
		t.Fatal(err)
	}
	if got.AvoidList.Rating != RatingWeak || len(got.Unclear) != 1 {
		t.Errorf("review read wrong: %+v", got)
	}
	for name, reply := range map[string]string{
		"missing aspect": `{"audience_fit":{"rating":"good","comment":"a"}}`,
		"odd rating":     strings.Replace(ok, `"weak"`, `"awful"`, 1),
		"no comment":     strings.Replace(ok, `"comment":"d"`, `"comment":" "`, 1),
	} {
		if _, err := ParseBriefReviewReply(reply); !errors.Is(err, ErrBriefReviewReplyUnusable) {
			t.Errorf("%s: err = %v, want ErrBriefReviewReplyUnusable", name, err)
		}
	}
}

func TestKeepProjectBriefCreatorFields(t *testing.T) {
	written := validBrief()
	current := validBrief()
	current.Tone = "do Creator viết"
	current.Avoid = []string{"tránh A"}
	got := KeepProjectBriefCreatorFields(written, current, []string{BriefFieldTone, BriefFieldAvoid})
	if got.Tone != "do Creator viết" || len(got.Avoid) != 1 || got.Avoid[0] != "tránh A" {
		t.Errorf("Creator fields not kept: %+v", got)
	}
	if got.Audience != "a" {
		t.Error("a field the Creator did not write was overwritten")
	}
}
