package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"authoring/internal/application"
	"authoring/internal/domain"
)

// fakeResearch answers the research API with fixed values or a fixed error and
// keeps what it was asked.
type fakeResearch struct {
	err       error
	state     application.ResearchState
	replace   bool
	note      string
	text      string
	raw       string
	fact      application.FactInput
	flags     application.FactFlags
	factID    string
	verifyIDs []string
	skipped   bool
	deletedID string
}

func (f *fakeResearch) State(context.Context, string) (application.ResearchState, error) {
	return f.state, f.err
}
func (f *fakeResearch) GenerateQuestions(_ context.Context, _ string, replace bool, note string) ([]application.QuestionView, error) {
	f.replace, f.note = replace, note
	return []application.QuestionView{{ID: "Q1", Text: "Một?"}}, f.err
}
func (f *fakeResearch) AddQuestion(_ context.Context, _, text string) (application.QuestionView, error) {
	f.text = text
	return application.QuestionView{ID: "Q2", Text: text, Mine: true}, f.err
}
func (f *fakeResearch) DeleteQuestion(_ context.Context, _, id string) error {
	f.deletedID = id
	return f.err
}
func (f *fakeResearch) ResearchPrompt(context.Context, string) (string, error) {
	return "PROMPT", f.err
}
func (f *fakeResearch) FollowUpPrompt(context.Context, string) (string, error) {
	return "FOLLOW-UP", f.err
}
func (f *fakeResearch) IngestPaste(_ context.Context, _, raw string) (application.PasteResult, error) {
	f.raw = raw
	return application.PasteResult{
		Added: []string{"F01"}, Updated: []string{}, Rejected: []domain.PasteRejected{{Row: 2, Reason: "thiếu type"}},
		Log: application.PasteLog{N: 1},
	}, f.err
}
func (f *fakeResearch) AddFact(_ context.Context, _ string, in application.FactInput) (application.FactView, error) {
	f.fact = in
	return application.FactView{ID: "F01", Fact: domain.Fact{ClaimVI: in.ClaimVI, Type: in.Type, Keep: true}, Mine: []string{"claim_vi"}}, f.err
}
func (f *fakeResearch) SetFactFlags(_ context.Context, _, id string, flags application.FactFlags) (application.FactView, error) {
	f.factID, f.flags = id, flags
	return application.FactView{ID: id, Fact: domain.Fact{Keep: true}, Mine: []string{}}, f.err
}
func (f *fakeResearch) Verify(_ context.Context, _ string, ids []string) error {
	f.verifyIDs = ids
	return f.err
}
func (f *fakeResearch) SetSkipped(_ context.Context, _ string, skipped bool) (application.GateView, error) {
	f.skipped = skipped
	return application.GateView{Status: domain.UnitPending, Skipped: skipped, Resolved: skipped}, f.err
}

func serveResearch(f *fakeResearch, method, path, body string) *httptest.ResponseRecorder {
	rt := NewRouter(&fakeSuggestMetadata{}, nil).WithResearch(f).WithOperations(application.NewOperations())
	rec := httptest.NewRecorder()
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
	}
	rt.Handler().ServeHTTP(rec, req)
	return rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("body is not a JSON object: %q", rec.Body.String())
	}
	return out
}

func TestGetResearchReturnsTheWholeScreen(t *testing.T) {
	f := &fakeResearch{state: application.ResearchState{
		Gate:      application.GateView{Status: domain.UnitPending},
		Questions: []application.QuestionView{{ID: "Q1", Text: "Một?", Mine: true, Stale: true}},
		Facts: []application.FactView{{
			ID: "F01", Mine: []string{"claim_vi"},
			Fact: domain.Fact{
				Question: "Q1", ClaimVI: "Dữ kiện", Type: domain.FactTypeFact, URL: "https://a.org/", Keep: true,
				Link: domain.FactLink{State: domain.LinkBad, Why: "không có đoạn trích", Nearest: "câu gần nhất"},
			},
		}},
		Pastes:         []application.PasteLog{{N: 1, Added: []string{"F01"}}},
		Coverage:       []domain.QuestionCoverage{{Question: "Q1", Sources: 1, BadFacts: []string{"F01"}, Weak: true}},
		BriefApproved:  true,
		QuestionsStale: true,
	}}
	rec := serveResearch(f, http.MethodGet, "/v1/projects/p1/research", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body.String())
	}
	body := decode(t, rec)
	for _, key := range []string{"gate", "questions", "facts", "pastes", "coverage", "brief_approved", "questions_stale"} {
		if _, ok := body[key]; !ok {
			t.Errorf("response lacks %q: %s", key, rec.Body.String())
		}
	}
	fact := body["facts"].([]any)[0].(map[string]any)
	for key, want := range map[string]any{"id": "F01", "claim_vi": "Dữ kiện", "type": "fact", "url": "https://a.org/", "keep": true} {
		if fact[key] != want {
			t.Errorf("fact[%q] = %v, want %v", key, fact[key], want)
		}
	}
	link := fact["link"].(map[string]any)
	if link["state"] != "bad" || link["nearest"] != "câu gần nhất" {
		t.Errorf("link = %v, want the check result with its nearest sentence", link)
	}
	question := body["questions"].([]any)[0].(map[string]any)
	if question["id"] != "Q1" || question["mine"] != true || question["stale"] != true {
		t.Errorf("question = %v", question)
	}
	coverage := body["coverage"].([]any)[0].(map[string]any)
	if coverage["weak"] != true || coverage["sources"] != float64(1) {
		t.Errorf("coverage = %v", coverage)
	}
}

func TestEmptyResearchStateListsAreArraysNotNull(t *testing.T) {
	rec := serveResearch(&fakeResearch{}, http.MethodGet, "/v1/projects/p1/research", "")
	for _, key := range []string{`"facts":[]`, `"questions":[]`, `"coverage":[]`} {
		if !strings.Contains(strings.ReplaceAll(rec.Body.String(), " ", ""), key) {
			t.Errorf("body %s lacks %s", rec.Body.String(), key)
		}
	}
}

func TestResearchRoutesPassTheirInputs(t *testing.T) {
	f := &fakeResearch{}
	cases := []struct {
		name, method, path, body string
		status                   int
		check                    func() bool
	}{
		{"generate questions", http.MethodPost, "/v1/projects/p1/research/questions/generate", `{"replace_edited":true,"note":"cơ chế"}`, 200,
			func() bool { return f.replace && f.note == "cơ chế" }},
		{"generate questions with no body", http.MethodPost, "/v1/projects/p1/research/questions/generate", "", 200,
			func() bool { return !f.replace && f.note == "" }},
		{"add question", http.MethodPost, "/v1/projects/p1/research/questions", `{"text":"Hỏi gì?"}`, 201, func() bool { return f.text == "Hỏi gì?" }},
		{"delete question", http.MethodDelete, "/v1/projects/p1/research/questions/Q3", "", 204, func() bool { return f.deletedID == "Q3" }},
		{"prompt", http.MethodGet, "/v1/projects/p1/research/prompt", "", 200, func() bool { return true }},
		{"follow-up prompt", http.MethodGet, "/v1/projects/p1/research/follow-up-prompt", "", 200, func() bool { return true }},
		{"paste", http.MethodPost, "/v1/projects/p1/research/paste", `{"raw":"nội dung dán"}`, 200, func() bool { return f.raw == "nội dung dán" }},
		{"add fact", http.MethodPost, "/v1/projects/p1/research/facts",
			`{"question":"Q1","claim_vi":"Tay","type":"hypothesis","url":"https://a.org/","quote":"q","year":"2020","source_title":"S","publisher":"P"}`, 201,
			func() bool {
				return f.fact.Question == "Q1" && f.fact.ClaimVI == "Tay" && f.fact.Type == domain.FactTypeHypothesis &&
					f.fact.URL == "https://a.org/" && f.fact.Year == "2020" && f.fact.SourceTitle == "S" && f.fact.Publisher == "P"
			}},
		{"set fact flags", http.MethodPut, "/v1/projects/p1/research/facts/F04", `{"type":"misconception","must":true}`, 200,
			func() bool {
				return f.factID == "F04" && *f.flags.Type == domain.FactTypeMisconception && *f.flags.Must && f.flags.Keep == nil
			}},
		{"verify", http.MethodPost, "/v1/projects/p1/research/verify", `{"fact_ids":["F01","F02"]}`, 202,
			func() bool { return strings.Join(f.verifyIDs, ",") == "F01,F02" }},
		{"verify everything", http.MethodPost, "/v1/projects/p1/research/verify", "", 202, func() bool { return len(f.verifyIDs) == 0 }},
		{"skip", http.MethodPut, "/v1/projects/p1/research/skip", `{"skipped":true}`, 200, func() bool { return f.skipped }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := serveResearch(f, c.method, c.path, c.body)
			if rec.Code != c.status {
				t.Fatalf("status = %d, want %d, body %s", rec.Code, c.status, rec.Body.String())
			}
			if !c.check() {
				t.Fatalf("the use case did not receive the request's values: %+v", f)
			}
		})
	}
}

func TestResearchResponsesCarryTheirFields(t *testing.T) {
	f := &fakeResearch{}
	prompt := decode(t, serveResearch(f, http.MethodGet, "/v1/projects/p1/research/prompt", ""))
	if prompt["prompt"] != "PROMPT" {
		t.Errorf("prompt = %v", prompt)
	}
	paste := decode(t, serveResearch(f, http.MethodPost, "/v1/projects/p1/research/paste", `{"raw":"x"}`))
	rejected := paste["rejected"].([]any)[0].(map[string]any)
	if paste["added"].([]any)[0] != "F01" || rejected["row"] != float64(2) || rejected["reason"] != "thiếu type" {
		t.Errorf("paste = %v, want the rows read and left out", paste)
	}
	gate := decode(t, serveResearch(f, http.MethodPut, "/v1/projects/p1/research/skip", `{"skipped":true}`))
	if gate["skipped"] != true || gate["resolved"] != true || gate["status"] != "pending" {
		t.Errorf("gate = %v", gate)
	}
}

func TestResearchRoutesMapErrorsToStatusCodes(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		method string
		path   string
		body   string
		status int
	}{
		{"no JSON block", domain.ErrNoResearchJSON, http.MethodPost, "/v1/projects/p1/research/paste", `{"raw":"x"}`, 400},
		{"paste too large", domain.ErrPasteTooLarge, http.MethodPost, "/v1/projects/p1/research/paste", `{"raw":"x"}`, 400},
		{"empty table", domain.ErrResearchEmpty, http.MethodPut, "/v1/projects/p1/research/skip", `{"skipped":true}`, 400},
		{"bad fact", fmt.Errorf("%w: x", domain.ErrInvalidUnitEdit), http.MethodPost, "/v1/projects/p1/research/facts", `{}`, 400},
		{"missing fact", domain.ErrUnitNotFound, http.MethodPut, "/v1/projects/p1/research/facts/F9", `{}`, 404},
		{"locked project", domain.ErrInvalidStatus, http.MethodPost, "/v1/projects/p1/research/paste", `{"raw":"x"}`, 409},
		{"no brief", fmt.Errorf("%w: chưa có brief", domain.ErrInvalidStatus), http.MethodGet, "/v1/projects/p1/research/prompt", "", 409},
		{"unusable AI reply", domain.ErrResearchReplyUnusable, http.MethodPost, "/v1/projects/p1/research/questions/generate", "", 502},
		{"no model configured", application.ErrLLMNotConfigured, http.MethodPost, "/v1/projects/p1/research/questions/generate", "", 503},
		{"unexpected failure", errors.New("boom"), http.MethodGet, "/v1/projects/p1/research", "", 500},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := serveResearch(&fakeResearch{err: c.err}, c.method, c.path, c.body)
			if rec.Code != c.status {
				t.Fatalf("status = %d, want %d, body %s", rec.Code, c.status, rec.Body.String())
			}
			if strings.Contains(rec.Body.String(), "boom") {
				t.Errorf("an unexpected error's text must not reach the client: %s", rec.Body.String())
			}
		})
	}
}

func TestResearchErrorsReadInTheCreatorsWordsWithoutTheSentinelName(t *testing.T) {
	cases := []struct {
		name string
		err  error
		path string
		want string
	}{
		{"no brief", fmt.Errorf("%w: chưa có brief, viết brief ở bước Ý tưởng trước", domain.ErrInvalidStatus), "/v1/projects/p1/research/prompt", "chưa có brief, viết brief ở bước Ý tưởng trước"},
		{"bad edit", fmt.Errorf("%w: dữ kiện không được trống", domain.ErrInvalidUnitEdit), "/v1/projects/p1/research/facts", "dữ kiện không được trống"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			method := http.MethodGet
			body := ""
			if strings.HasSuffix(c.path, "/facts") {
				method, body = http.MethodPost, `{}`
			}
			rec := serveResearch(&fakeResearch{err: c.err}, method, c.path, body)
			got := decode(t, rec)["error"]
			if got != c.want {
				t.Fatalf("error = %q, want %q", got, c.want)
			}
		})
	}
	// An error with no reason of its own keeps the sentinel's text.
	rec := serveResearch(&fakeResearch{err: domain.ErrInvalidStatus}, http.MethodGet, "/v1/projects/p1/research/prompt", "")
	if decode(t, rec)["error"] != domain.ErrInvalidStatus.Error() {
		t.Fatalf("a bare sentinel must keep its own text, got %s", rec.Body.String())
	}
}

func TestQuestionRunErrorsNameTheQuestionsNotTheBrief(t *testing.T) {
	rec := serveResearch(&fakeResearch{err: application.ErrLLMNotConfigured}, http.MethodPost, "/v1/projects/p1/research/questions/generate", "")
	if !strings.Contains(rec.Body.String(), "câu hỏi") {
		t.Errorf("body = %s, want a message about questions", rec.Body.String())
	}
	rec = serveResearch(&fakeResearch{err: domain.ErrResearchReplyUnusable}, http.MethodPost, "/v1/projects/p1/research/questions/generate", "")
	if !strings.Contains(rec.Body.String(), "tự thêm câu hỏi") {
		t.Errorf("body = %s, want the advice to add questions by hand", rec.Body.String())
	}
}

func TestBadResearchBodiesAreRefused(t *testing.T) {
	for _, path := range []string{
		"/v1/projects/p1/research/questions", "/v1/projects/p1/research/paste", "/v1/projects/p1/research/facts", "/v1/projects/p1/research/skip",
	} {
		method := http.MethodPost
		if strings.HasSuffix(path, "skip") {
			method = http.MethodPut
		}
		if rec := serveResearch(&fakeResearch{}, method, path, "{không phải JSON"); rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", path, rec.Code)
		}
	}
}

func TestResearchRoutesAre404WhenNotWired(t *testing.T) {
	rt := NewRouter(&fakeSuggestMetadata{}, nil)
	rec := httptest.NewRecorder()
	rt.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/projects/p1/research", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestGenerateQuestionsIsTrackedAsAnOperation(t *testing.T) {
	ops := application.NewOperations()
	rt := NewRouter(&fakeSuggestMetadata{}, nil).WithResearch(&fakeResearch{}).WithOperations(ops)
	req := httptest.NewRequest(http.MethodPost, "/v1/projects/p1/research/questions/generate", nil)
	req.Header.Set("X-Operation-Id", "op-questions")
	rt.Handler().ServeHTTP(httptest.NewRecorder(), req)
	op, ok := ops.Get("op-questions")
	if !ok {
		t.Fatal("the operation id the screen sent must be registered")
	}
	if op.Kind != domain.UnitKindResearchQuestion {
		t.Errorf("operation kind = %q, want %q", op.Kind, domain.UnitKindResearchQuestion)
	}
}
