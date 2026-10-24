package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestParseProjectIdeaReply_ReadsFourFields(t *testing.T) {
	reply := "Đây:\n```json\n{\"central_question\":\" Người rơi thấy gì? \",\"curiosity_points\":[\"Chân trời sự kiện là gì?\",\" \",\"chân trời sự kiện là gì?\",\"Có bị kéo dài không?\"],\"scope\":\"Trực quan, không toán.\"}\n```"
	got, err := ParseProjectIdeaReply(reply)
	if err != nil {
		t.Fatal(err)
	}
	if got.CentralQuestion != "Người rơi thấy gì?" {
		t.Errorf("central question = %q", got.CentralQuestion)
	}
	if len(got.CuriosityPoints) != 2 {
		t.Errorf("curiosity points = %v, want 2 (blank and repeated dropped)", got.CuriosityPoints)
	}
}

func TestParseProjectIdeaReply_RefusesWhatIsNotWhole(t *testing.T) {
	cases := map[string]string{
		"no json":       "xin lỗi",
		"missing scope": `{"central_question":"b?","curiosity_points":["c"]}`,
		"no points":     `{"central_question":"b?","curiosity_points":[],"scope":"d"}`,
		"points string": `{"central_question":"b?","curiosity_points":"c","scope":"d"}`,
	}
	for name, reply := range cases {
		if _, err := ParseProjectIdeaReply(reply); !errors.Is(err, ErrProjectIdeaReplyUnusable) {
			t.Errorf("%s: err = %v, want ErrProjectIdeaReplyUnusable", name, err)
		}
	}
}

func TestProjectIdeaValidate(t *testing.T) {
	ok := ProjectIdea{CentralQuestion: "b?", CuriosityPoints: []string{"c"}, Scope: "d"}
	if err := ok.Validate(); err != nil {
		t.Fatalf("valid idea refused: %v", err)
	}
	long := ok
	long.CuriosityPoints = []string{strings.Repeat("x", 301)}
	if err := long.Validate(); !errors.Is(err, ErrProjectIdeaInvalid) {
		t.Errorf("long point: err = %v", err)
	}
	many := ok
	many.CuriosityPoints = nil
	for i := 0; i < MaxCuriosityPoints+1; i++ {
		many.CuriosityPoints = append(many.CuriosityPoints, strings.Repeat("y", i+1))
	}
	if err := many.Validate(); !errors.Is(err, ErrProjectIdeaInvalid) {
		t.Errorf("too many points: err = %v", err)
	}
	empty := ok
	empty.Scope = "  "
	if err := empty.Validate(); !errors.Is(err, ErrProjectIdeaInvalid) {
		t.Errorf("empty scope: err = %v", err)
	}
}

func TestProjectIdeaHash_IgnoresSpacesAndTracksContent(t *testing.T) {
	a := ProjectIdea{CentralQuestion: "b?", CuriosityPoints: []string{"c"}, Scope: "d"}
	b := ProjectIdea{CentralQuestion: " b? ", CuriosityPoints: []string{" c ", ""}, Scope: "d"}
	if a.Hash() != b.Hash() {
		t.Error("spaces and blank points changed the hash")
	}
	b.Scope = "e"
	if a.Hash() == b.Hash() {
		t.Error("a changed scope kept the hash")
	}
}

func TestKeepProjectIdeaCreatorFields(t *testing.T) {
	current := ProjectIdea{Scope: "mine", CuriosityPoints: []string{"mine 1"}}
	written := ProjectIdea{CentralQuestion: "ai?", CuriosityPoints: []string{"ai 1"}, Scope: "ai"}
	got := KeepProjectIdeaCreatorFields(written, current, []string{IdeaFieldScope, IdeaFieldCuriosityPoints})
	if got.Scope != "mine" || got.CuriosityPoints[0] != "mine 1" {
		t.Errorf("creator fields not kept: %+v", got)
	}
	if got.CentralQuestion != "ai?" {
		t.Errorf("AI fields lost: %+v", got)
	}
}

func TestParseIdeaReviewReply(t *testing.T) {
	good := `{"clarity":{"rating":"good","comment":"Rõ."},"researchability":{"rating":"fair","comment":"Cần nguồn."},"scope_fit":{"rating":"weak","comment":"Rộng."},"unclear":["Ai xem?"," "],"suggestions":[]}`
	r, err := ParseIdeaReviewReply(good)
	if err != nil {
		t.Fatal(err)
	}
	if r.ScopeFit.Rating != RatingWeak || len(r.Unclear) != 1 || r.Suggestions == nil {
		t.Errorf("review = %+v", r)
	}
	bad := map[string]string{
		"missing aspect": `{"clarity":{"rating":"good","comment":"a"},"researchability":{"rating":"good","comment":"a"}}`,
		"odd rating":     `{"clarity":{"rating":"great","comment":"a"},"researchability":{"rating":"good","comment":"a"},"scope_fit":{"rating":"good","comment":"a"}}`,
		"no comment":     `{"clarity":{"rating":"good","comment":""},"researchability":{"rating":"good","comment":"a"},"scope_fit":{"rating":"good","comment":"a"}}`,
	}
	for name, reply := range bad {
		if _, err := ParseIdeaReviewReply(reply); !errors.Is(err, ErrIdeaReviewReplyUnusable) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
}

func TestProjectIdeaPromptsCarryTheirPlaceholders(t *testing.T) {
	for role, vars := range map[PromptRole][]string{
		RoleProjectIdea:       {"{{topic}}", "{{idea_language}}", "{{current_idea}}", "{{kept_fields}}", "{{redo_note}}"},
		RoleProjectIdeaReview: {"{{idea}}", "{{idea_language}}"},
	} {
		tpl, ok := DefaultPromptTemplate(role, "vi")
		if !ok {
			t.Fatalf("no seed for %s", role)
		}
		for _, v := range vars {
			if !strings.Contains(tpl.TemplateText, v) {
				t.Errorf("%s misses %s", role, v)
			}
		}
		if !ValidPromptRole(string(role)) || IsRenderablePromptRole(string(role)) {
			t.Errorf("%s must be a valid role that is not renderable", role)
		}
	}
}

func TestProjectIdeaPromptAsksForTheThreeFields(t *testing.T) {
	tpl, _ := DefaultPromptTemplate(RoleProjectIdea, "vi")
	for _, key := range []string{"`central_question`", "`curiosity_points`", "`scope`", "Chỉ xuất JSON cuối cùng."} {
		if !strings.Contains(tpl.TemplateText, key) {
			t.Errorf("the idea prompt lacks %s", key)
		}
	}
	if strings.Contains(tpl.TemplateText, "refined_topic") || strings.Contains(tpl.TemplateText, "{{subject}}") {
		t.Error("the idea prompt still names a field or placeholder the step no longer has")
	}
}
