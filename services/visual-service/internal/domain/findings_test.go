package domain

import (
	"errors"
	"regexp"
	"strings"
	"testing"
)

func TestParseReviewReply_RefusesWhatDoesNotExist(t *testing.T) {
	prev := ReviewState{Findings: []Finding{{ID: "f1"}}}
	bad := []string{
		`not json`,
		`{"findings":[{"group":"taste","unit_key":"color_system","text":"x"}]}`,
		`{"findings":[{"group":"accuracy","unit_key":"palette","text":"x"}]}`,
		`{"findings":[{"group":"accuracy","unit_key":"color_system","field":"colours","text":"x"}]}`,
		`{"findings":[{"group":"accuracy","unit_key":"color_system","text":" "}]}`,
		`{"findings":[{"group":"accuracy","unit_key":"color_system","text":"x","same_as":"f9"}]}`,
		`{"findings":[{"group":"accuracy","unit_key":"color_system","text":"x","same_as":"f1"},{"group":"accuracy","unit_key":"color_system","text":"y","same_as":"f1"}]}`,
	}
	for _, b := range bad {
		if _, err := ParseReviewReply([]byte(b), prev); !errors.Is(err, ErrUnreadableReply) {
			t.Errorf("%s: %v", b, err)
		}
	}
	if _, err := ParseReviewReply([]byte(`{"findings":[{"group":"feasibility","unit_key":"feasibility","field":"checks","text":"x","same_as":"f1"}]}`), prev); err != nil {
		t.Error(err)
	}
}

func TestMergeReview_KeepsIDsFixesWhatIsGoneAndKeepsAcceptance(t *testing.T) {
	first := MergeReview(ReviewState{NextSeq: 1}, ReviewReply{Findings: []ReviewReplyItem{
		{Group: "accuracy", UnitKey: "representation_rules", Text: "a"},
		{Group: "clarity", UnitKey: "typography", Text: "b"},
		{Group: "feasibility", UnitKey: "feasibility", Text: "c"},
	}})
	if first.Rounds != 1 || first.NextSeq != 4 || first.Findings[2].ID != "f3" {
		t.Fatalf("first %+v", first)
	}
	first.Findings[1].State, first.Findings[1].Why = FindingAccepted, "giữ"
	first.Findings[2].State = FindingApplied
	second := MergeReview(first, ReviewReply{Findings: []ReviewReplyItem{
		{Group: "clarity", UnitKey: "typography", Text: "b again", SameAs: "f2"},
		{Group: "feasibility", UnitKey: "feasibility", Text: "c again", SameAs: "f3"},
		{Group: "consistency", UnitKey: "color_system", Text: "d"},
	}})
	want := map[string]string{"f1": FindingFixed, "f2": FindingAccepted, "f3": FindingOpen, "f4": FindingOpen}
	for _, f := range second.Findings {
		if want[f.ID] != f.State {
			t.Errorf("%s: %s want %s", f.ID, f.State, want[f.ID])
		}
	}
	if second.Findings[1].Why != "giữ" || second.OpenCount() != 2 || second.Rounds != 2 {
		t.Errorf("second %+v", second)
	}
}

func TestApplyFix(t *testing.T) {
	out, err := ApplyFix([]byte(`{"captions":"cũ","examples":["a"]}`), "captions", "mới")
	if err != nil || !sameJSONT(out, `{"captions":"mới","examples":["a"]}`) {
		t.Errorf("%s %v", out, err)
	}
	out, _ = ApplyFix([]byte(`{"examples":["a"]}`), "examples", "b")
	if !sameJSONT(out, `{"examples":["a","b"]}`) {
		t.Errorf("%s", out)
	}
	for _, field := range []string{"labels", "missing"} {
		if _, err := ApplyFix([]byte(`{"labels":{"max_words":3}}`), field, "x"); !errors.Is(err, ErrFindingNotApplicable) {
			t.Errorf("%s: %v", field, err)
		}
	}
}

func TestCheckDecisionHoldsWhileTheDetailIsTheSame(t *testing.T) {
	c := MachineCheck{ID: CheckTerms, Detail: "Chưa có cách vẽ: X."}
	r := ReviewState{Checks: map[string]CheckDecision{CheckTerms: {Why: "ok", Fingerprint: Fingerprint(c.Detail)}}}
	if _, ok := r.CheckAccepted(c); !ok {
		t.Error("accepted")
	}
	c.Detail = "Chưa có cách vẽ: X, Y."
	if _, ok := r.CheckAccepted(c); ok {
		t.Error("a changed detail opens the check again")
	}
}

func TestSystemPromptsCoverEveryPart(t *testing.T) {
	ps := SystemPrompts()
	if len(ps) != 10 || ps[8].Role != PromptRoleReview || ps[9].Role != PromptRoleReferenceYouTube || ps[9].Name != ReferenceYouTubeName {
		t.Fatalf("%d prompts", len(ps))
	}
	for _, p := range ps {
		if p.TemplateText == "" || p.Name == "" {
			t.Errorf("%s is empty", p.Role)
		}
	}
}

func TestOnlyDirectionAndVisualLanguageReadReferences(t *testing.T) {
	for _, p := range SystemPrompts() {
		reads := strings.Contains(p.TemplateText, "{{refs}}")
		want := p.Role == PromptRoleOf(UnitCreativeDirection) || p.Role == PromptRoleOf(UnitVisualLanguage)
		if reads != want {
			t.Errorf("%s reads references = %v, want %v", p.Role, reads, want)
		}
	}
	for _, u := range ConceptUnits {
		reads := false
		for _, s := range u.Reads {
			reads = reads || s == SourceRefs
		}
		if reads != (u.Key == UnitCreativeDirection || u.Key == UnitVisualLanguage) {
			t.Errorf("%s reads refs = %v", u.Key, reads)
		}
	}
}

func TestReferenceYouTubePromptIsStoredOnlyAndTakesOneVariable(t *testing.T) {
	vars := regexp.MustCompile(`\{\{([a-z_]+)\}\}`)
	var yt SystemPrompt
	for _, p := range SystemPrompts() {
		if p.Role == PromptRoleReferenceYouTube {
			yt = p
			continue
		}
		if strings.Contains(p.TemplateText, "{{url}}") {
			t.Errorf("%s must not take the link", p.Role)
		}
	}
	if yt.Role == "" {
		t.Fatal("the reference prompt is missing")
	}
	found := map[string]bool{}
	for _, m := range vars.FindAllStringSubmatch(yt.TemplateText, -1) {
		found[m[1]] = true
	}
	if len(found) != 1 || !found["url"] {
		t.Errorf("variables %v, want only url", found)
	}
	if !strings.HasPrefix(yt.TemplateText, "Bạn là art director phân tích cách vẽ") || !strings.HasSuffix(yt.TemplateText, `"tags": []}`) {
		t.Error("the approved wording changed")
	}
	if strings.Contains(yt.TemplateText, "\"error\": \"lý do\"") == false {
		t.Error("the error reply is part of the approved wording")
	}
}
