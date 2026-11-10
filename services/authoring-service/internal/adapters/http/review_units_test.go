package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"authoring/internal/application"
	"authoring/internal/domain"
)

// fakeUnits answers the unit API with a fixed unit or a fixed error.
type fakeUnits struct {
	unit       domain.ReviewUnit
	err        error
	lastFields map[string]string
	lastNote   string
}

func (f *fakeUnits) Get(context.Context, string, string, string) (domain.ReviewUnit, error) {
	return f.unit, f.err
}
func (f *fakeUnits) Edit(_ context.Context, _, _, _ string, fields map[string]string) (domain.ReviewUnit, error) {
	f.lastFields = fields
	return f.unit, f.err
}
func (f *fakeUnits) Redo(_ context.Context, _, _, _, note string) (domain.ReviewUnit, error) {
	f.lastNote = note
	return f.unit, f.err
}
func (f *fakeUnits) Impact(context.Context, string, string, string) ([]string, error) {
	return []string{"Kịch bản"}, f.err
}
func (f *fakeUnits) Approve(context.Context, string, string, string) (domain.ReviewUnit, error) {
	return f.unit, f.err
}
func (f *fakeUnits) Unapprove(context.Context, string, string, string) (domain.ReviewUnit, error) {
	return f.unit, f.err
}

func serveUnits(units *fakeUnits, method, path, body string) *httptest.ResponseRecorder {
	rt := NewRouter(&fakeSuggestMetadata{}, nil).WithReviewUnits(units)
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

func TestUnitRoutesMapErrorsToStatusCodes(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		method string
		path   string
		body   string
		want   int
	}{
		{"unknown kind", fmt.Errorf("%w: \"beat\"", domain.ErrUnknownUnitKind), "GET", "/v1/projects/p1/units/beat/1", "", http.StatusNotFound},
		{"missing unit", domain.ErrUnitNotFound, "GET", "/v1/projects/p1/units/brief/main", "", http.StatusNotFound},
		{"bad edit", fmt.Errorf("%w: brief không có ô \"x\"", domain.ErrInvalidUnitEdit), "PUT", "/v1/projects/p1/units/brief/main", `{"fields":{"x":"y"}}`, http.StatusBadRequest},
		{"locked project", domain.ErrInvalidStatus, "POST", "/v1/projects/p1/units/brief/main/unapprove", "", http.StatusConflict},
		{"unusable reply", fmt.Errorf("%w: thiếu ô feel", domain.ErrProjectBriefReplyUnusable), "POST", "/v1/projects/p1/units/brief/main/redo", `{"note":"n"}`, http.StatusBadGateway},
		{"no model", application.ErrLLMNotConfigured, "POST", "/v1/projects/p1/units/brief/main/redo", "", http.StatusServiceUnavailable},
	}
	for _, c := range cases {
		rec := serveUnits(&fakeUnits{err: c.err}, c.method, c.path, c.body)
		if rec.Code != c.want {
			t.Errorf("%s: status %d, want %d (body %s)", c.name, rec.Code, c.want, rec.Body.String())
		}
	}
}

func TestEditAndRedoPassTheBodyThrough(t *testing.T) {
	units := &fakeUnits{unit: domain.ReviewUnit{Kind: domain.UnitKindResearchQuestion, Key: "Q1"}}
	if rec := serveUnits(units, "PUT", "/v1/projects/p1/units/brief/main", `{"fields":{"tone":"ấm"}}`); rec.Code != http.StatusOK {
		t.Fatalf("edit: status %d", rec.Code)
	}
	if units.lastFields["tone"] != "ấm" {
		t.Errorf("edit fields not passed through: %v", units.lastFields)
	}
	if rec := serveUnits(units, "POST", "/v1/projects/p1/units/brief/main/redo", `{"note":"ngắn hơn"}`); rec.Code != http.StatusOK {
		t.Fatalf("redo: status %d", rec.Code)
	}
	if units.lastNote != "ngắn hơn" {
		t.Errorf("redo note not passed through: %q", units.lastNote)
	}
}

func TestImpactReturnsItsItems(t *testing.T) {
	rec := serveUnits(&fakeUnits{}, "GET", "/v1/projects/p1/units/brief/main/impact", "")
	var body unitImpactResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || len(body.Items) != 1 {
		t.Fatalf("impact body %s, err %v", rec.Body.String(), err)
	}
}

// fakeModelInfo is llm-service's catalog and step defaults with fixed answers.
type fakeModelInfo struct {
	steps map[string]string
	err   error
}

func (f *fakeModelInfo) ModelCatalog(context.Context) ([]domain.AuthoringModelOption, error) {
	if f.err != nil {
		return nil, f.err
	}
	return []domain.AuthoringModelOption{{ID: "ollama", Label: "Ollama", Local: true}, {ID: "m2", Label: "M2"}}, nil
}

func (f *fakeModelInfo) GetStepDefaultModels(context.Context) (map[string]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.steps, nil
}

func TestLLMStatusReportsTheCatalogAndStepDefaultsFromLLMService(t *testing.T) {
	info := &fakeModelInfo{steps: map[string]string{"brief": "m2", "code": "ollama"}}
	rt := NewRouter(&fakeSuggestMetadata{}, nil).WithLLMStatus(&fakeLLMStatus{ready: true}).WithModelInfo(info, info)
	rec := httptest.NewRecorder()
	rt.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/v1/llm/status", nil))
	var got llmStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !got.Enabled || len(got.Models) != 2 || got.Models[0].ID != "ollama" || got.StepDefaults["brief"] != "m2" {
		t.Fatalf("status = %+v", got)
	}
}

// An llm-service that cannot give the catalog is said so, not papered over with a list of our own.
func TestLLMStatusSaysWhenTheCatalogCannotBeRead(t *testing.T) {
	info := &fakeModelInfo{err: fmt.Errorf("llm-service down")}
	rt := NewRouter(&fakeSuggestMetadata{}, nil).WithLLMStatus(&fakeLLMStatus{ready: true}).WithModelInfo(info, info)
	rec := httptest.NewRecorder()
	rt.Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/v1/llm/status", nil))
	var got llmStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Enabled || !strings.Contains(got.Reason, "llm-service down") || len(got.Models) != 0 {
		t.Fatalf("status = %+v", got)
	}
}

// fakeIdea is the first step's use case with a fixed answer.
type fakeIdea struct {
	state   application.ProjectIdeaState
	idea    domain.StoredIdea
	review  domain.IdeaReview
	err     error
	patched application.IdeaPatch
}

func (f *fakeIdea) State(context.Context, string) (application.ProjectIdeaState, error) {
	return f.state, f.err
}
func (f *fakeIdea) Generate(context.Context, string, string) (domain.StoredIdea, error) {
	return f.idea, f.err
}
func (f *fakeIdea) Edit(_ context.Context, _ string, p application.IdeaPatch) (domain.StoredIdea, error) {
	f.patched = p
	return f.idea, f.err
}
func (f *fakeIdea) Approve(context.Context, string) (domain.StoredIdea, error) { return f.idea, f.err }
func (f *fakeIdea) Unapprove(context.Context, string) (domain.StoredIdea, error) {
	return f.idea, f.err
}
func (f *fakeIdea) Review(context.Context, string) (domain.IdeaReview, error) {
	return f.review, f.err
}

func serveIdea(idea *fakeIdea, method, path, body string) *httptest.ResponseRecorder {
	rt := NewRouter(&fakeSuggestMetadata{}, nil).WithReviewUnits(&fakeUnits{}).WithIdea(idea)
	rec := httptest.NewRecorder()
	rt.Handler().ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func TestIdeaRoutes(t *testing.T) {
	stored := domain.StoredIdea{
		Idea:   domain.ProjectIdea{CentralQuestion: "Rơi vào thì sao?", CuriosityPoints: []string{"a"}, Scope: "s"},
		Status: domain.IdeaApproved,
	}
	state := application.ProjectIdeaState{Idea: &stored, Subject: "Rơi vào hố đen?", Language: domain.LanguageVietnamese, InputChanged: true}
	rec := serveIdea(&fakeIdea{state: state}, "GET", "/v1/projects/p1/idea", "")
	var body ideaStateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK || body.Idea == nil || body.Idea.Status != "approved" || body.Idea.CuriosityPoints[0] != "a" ||
		body.Subject != "Rơi vào hố đen?" || body.Language != "vi" || !body.InputChanged || body.Review != nil {
		t.Errorf("idea state: status %d body %s", rec.Code, rec.Body.String())
	}
	if rec := serveIdea(&fakeIdea{}, "GET", "/v1/projects/p1/idea", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"idea":null`) {
		t.Errorf("no idea yet: status %d body %s", rec.Code, rec.Body.String())
	}
	if rec := serveIdea(&fakeIdea{idea: stored}, "POST", "/v1/projects/p1/idea/generate", ""); rec.Code != http.StatusOK {
		t.Errorf("generate: status %d", rec.Code)
	}
	unusable := &fakeIdea{err: fmt.Errorf("%w: thiếu ô", domain.ErrProjectIdeaReplyUnusable)}
	if rec := serveIdea(unusable, "POST", "/v1/projects/p1/idea/generate", ""); rec.Code != http.StatusBadGateway {
		t.Errorf("an unusable reply: status %d, want 502", rec.Code)
	}
	if rec := serveIdea(&fakeIdea{err: application.ErrLLMNotConfigured}, "POST", "/v1/projects/p1/idea/generate", ""); rec.Code != http.StatusServiceUnavailable {
		t.Errorf("no model: status %d, want 503", rec.Code)
	}
	if rec := serveIdea(&fakeIdea{err: domain.ErrInvalidStatus}, "POST", "/v1/projects/p1/idea/generate", ""); rec.Code != http.StatusConflict {
		t.Errorf("a locked project: status %d, want 409", rec.Code)
	}
}

func TestIdeaEditApproveReviewRoutes(t *testing.T) {
	stored := domain.StoredIdea{Idea: domain.ProjectIdea{CentralQuestion: "q?"}, Status: domain.IdeaPending}
	f := &fakeIdea{idea: stored}
	rec := serveIdea(f, "PUT", "/v1/projects/p1/idea", `{"scope":"Phạm vi mới","curiosity_points":["a","b"]}`)
	if rec.Code != http.StatusOK || f.patched.Scope == nil || *f.patched.Scope != "Phạm vi mới" ||
		f.patched.CuriosityPoints == nil || len(*f.patched.CuriosityPoints) != 2 || f.patched.CentralQuestion != nil {
		t.Errorf("edit: status %d patch %+v", rec.Code, f.patched)
	}
	if rec := serveIdea(f, "PUT", "/v1/projects/p1/idea", `{bad`); rec.Code != http.StatusBadRequest {
		t.Errorf("a bad body: status %d, want 400", rec.Code)
	}
	if rec := serveIdea(f, "POST", "/v1/projects/p1/idea/approve", ""); rec.Code != http.StatusOK {
		t.Errorf("approve: status %d", rec.Code)
	}
	if rec := serveIdea(f, "POST", "/v1/projects/p1/idea/unapprove", ""); rec.Code != http.StatusOK {
		t.Errorf("unapprove: status %d", rec.Code)
	}
	if rec := serveIdea(&fakeIdea{err: domain.ErrIdeaNotFound}, "POST", "/v1/projects/p1/idea/approve", ""); rec.Code != http.StatusNotFound {
		t.Errorf("approving nothing: status %d, want 404", rec.Code)
	}
	if rec := serveIdea(&fakeIdea{err: fmt.Errorf("%w: Phạm vi", domain.ErrProjectIdeaInvalid)}, "POST", "/v1/projects/p1/idea/approve", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("approving an incomplete idea: status %d, want 400", rec.Code)
	}
	review := domain.IdeaReview{Clarity: domain.IdeaAspect{Rating: domain.RatingGood, Comment: "Rõ."}}
	rec = serveIdea(&fakeIdea{review: review}, "POST", "/v1/projects/p1/idea/review", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"clarity"`) {
		t.Errorf("review: status %d body %s", rec.Code, rec.Body.String())
	}
	if rec := serveIdea(&fakeIdea{err: fmt.Errorf("%w: x", domain.ErrIdeaReviewReplyUnusable)}, "POST", "/v1/projects/p1/idea/review", ""); rec.Code != http.StatusBadGateway {
		t.Errorf("an unusable review: status %d, want 502", rec.Code)
	}
}

// fakeProjectBrief is the second step's use case with a fixed answer.
type fakeProjectBrief struct {
	state   application.ProjectBriefState
	brief   domain.StoredBrief
	review  domain.BriefReview
	err     error
	patched application.BriefPatch
}

func (f *fakeProjectBrief) State(context.Context, string) (application.ProjectBriefState, error) {
	return f.state, f.err
}
func (f *fakeProjectBrief) Generate(context.Context, string, string) (domain.StoredBrief, error) {
	return f.brief, f.err
}
func (f *fakeProjectBrief) Edit(_ context.Context, _ string, p application.BriefPatch) (domain.StoredBrief, error) {
	f.patched = p
	return f.brief, f.err
}
func (f *fakeProjectBrief) Approve(context.Context, string) (domain.StoredBrief, error) {
	return f.brief, f.err
}
func (f *fakeProjectBrief) Unapprove(context.Context, string) (domain.StoredBrief, error) {
	return f.brief, f.err
}
func (f *fakeProjectBrief) Review(context.Context, string) (domain.BriefReview, error) {
	return f.review, f.err
}

func serveProjectBrief(brief *fakeProjectBrief, method, path, body string) *httptest.ResponseRecorder {
	rt := NewRouter(&fakeSuggestMetadata{}, nil).WithReviewUnits(&fakeUnits{}).WithProjectBrief(brief)
	rec := httptest.NewRecorder()
	rt.Handler().ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func TestProjectBriefRoutes(t *testing.T) {
	stored := domain.StoredBrief{
		Brief:  domain.ProjectBrief{Audience: "a", DurationSeconds: 270, Tone: "t", LearningObjectives: []string{"o"}, Avoid: []string{"v"}},
		Status: domain.BriefApproved,
	}
	idea := domain.ProjectIdea{CentralQuestion: "q?", CuriosityPoints: []string{"c"}, Scope: "s"}
	state := application.ProjectBriefState{
		Brief: &stored, Idea: &idea, Subject: "Rơi vào hố đen?", Language: domain.LanguageVietnamese,
		VideoMode: domain.ModeLongOnly, MinDurationSeconds: 121, MaxDurationSeconds: 1800, IdeaChanged: true,
	}
	rec := serveProjectBrief(&fakeProjectBrief{state: state}, "GET", "/v1/projects/p1/creative-brief", "")
	var body projectBriefStateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusOK || body.Brief == nil || body.Brief.DurationSeconds != 270 || body.Idea == nil ||
		body.Idea.Scope != "s" || body.MinDurationSeconds != 121 || body.MaxDurationSeconds != 1800 || !body.IdeaChanged || body.Review != nil {
		t.Errorf("brief state: status %d body %s", rec.Code, rec.Body.String())
	}
	if rec := serveProjectBrief(&fakeProjectBrief{}, "GET", "/v1/projects/p1/creative-brief", ""); rec.Code != http.StatusOK ||
		!strings.Contains(rec.Body.String(), `"brief":null`) || !strings.Contains(rec.Body.String(), `"idea":null`) {
		t.Errorf("no brief or idea yet: status %d body %s", rec.Code, rec.Body.String())
	}
	if rec := serveProjectBrief(&fakeProjectBrief{brief: stored}, "POST", "/v1/projects/p1/creative-brief/generate", ""); rec.Code != http.StatusOK {
		t.Errorf("generate: status %d", rec.Code)
	}
	unusable := &fakeProjectBrief{err: fmt.Errorf("%w: thiếu ô", domain.ErrProjectBriefReplyUnusable)}
	if rec := serveProjectBrief(unusable, "POST", "/v1/projects/p1/creative-brief/generate", ""); rec.Code != http.StatusBadGateway {
		t.Errorf("an unusable reply: status %d, want 502", rec.Code)
	}
	if rec := serveProjectBrief(&fakeProjectBrief{err: application.ErrLLMNotConfigured}, "POST", "/v1/projects/p1/creative-brief/generate", ""); rec.Code != http.StatusServiceUnavailable {
		t.Errorf("no model: status %d, want 503", rec.Code)
	}
	needsIdea := &fakeProjectBrief{err: fmt.Errorf("%w: %w", domain.ErrInvalidStatus, domain.ErrBriefNeedsApprovedIdea)}
	if rec := serveProjectBrief(needsIdea, "POST", "/v1/projects/p1/creative-brief/generate", ""); rec.Code != http.StatusConflict {
		t.Errorf("idea not approved: status %d, want 409", rec.Code)
	}
}

func TestProjectBriefEditApproveReviewRoutes(t *testing.T) {
	f := &fakeProjectBrief{brief: domain.StoredBrief{Status: domain.BriefPending}}
	rec := serveProjectBrief(f, "PUT", "/v1/projects/p1/creative-brief", `{"duration_seconds":300,"avoid":["a","b"]}`)
	if rec.Code != http.StatusOK || f.patched.DurationSeconds == nil || *f.patched.DurationSeconds != 300 ||
		f.patched.Avoid == nil || len(*f.patched.Avoid) != 2 || f.patched.Tone != nil {
		t.Errorf("edit: status %d patch %+v", rec.Code, f.patched)
	}
	if rec := serveProjectBrief(f, "PUT", "/v1/projects/p1/creative-brief", `{bad`); rec.Code != http.StatusBadRequest {
		t.Errorf("a bad body: status %d, want 400", rec.Code)
	}
	for _, path := range []string{"approve", "unapprove"} {
		if rec := serveProjectBrief(f, "POST", "/v1/projects/p1/creative-brief/"+path, ""); rec.Code != http.StatusOK {
			t.Errorf("%s: status %d", path, rec.Code)
		}
	}
	if rec := serveProjectBrief(&fakeProjectBrief{err: domain.ErrBriefNotFound}, "POST", "/v1/projects/p1/creative-brief/approve", ""); rec.Code != http.StatusNotFound {
		t.Errorf("approving nothing: status %d, want 404", rec.Code)
	}
	if rec := serveProjectBrief(&fakeProjectBrief{err: fmt.Errorf("%w: thời lượng", domain.ErrProjectBriefInvalid)}, "POST", "/v1/projects/p1/creative-brief/approve", ""); rec.Code != http.StatusBadRequest {
		t.Errorf("approving an invalid brief: status %d, want 400", rec.Code)
	}
	review := domain.BriefReview{AudienceFit: domain.IdeaAspect{Rating: domain.RatingGood, Comment: "Rõ."}}
	rec = serveProjectBrief(&fakeProjectBrief{review: review}, "POST", "/v1/projects/p1/creative-brief/review", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"audience_fit"`) {
		t.Errorf("review: status %d body %s", rec.Code, rec.Body.String())
	}
	if rec := serveProjectBrief(&fakeProjectBrief{err: fmt.Errorf("%w: x", domain.ErrBriefReviewReplyUnusable)}, "POST", "/v1/projects/p1/creative-brief/review", ""); rec.Code != http.StatusBadGateway {
		t.Errorf("an unusable review: status %d, want 502", rec.Code)
	}
}
