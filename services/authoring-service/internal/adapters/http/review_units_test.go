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

// fakeBrief answers the brief API with a fixed state or a fixed error.
type fakeBrief struct {
	state application.BriefState
	err   error
}

func (f *fakeBrief) State(context.Context, string) (application.BriefState, error) {
	return f.state, f.err
}
func (f *fakeBrief) SetTargetDuration(context.Context, string, domain.TargetDuration) error {
	return f.err
}
func (f *fakeBrief) SetPlatform(context.Context, string, string) error {
	return f.err
}
func (f *fakeBrief) Generate(context.Context, string, string) (domain.ReviewUnit, error) {
	return domain.ReviewUnit{Kind: domain.UnitKindBrief}, f.err
}

func serveUnits(units *fakeUnits, brief *fakeBrief, method, path, body string) *httptest.ResponseRecorder {
	rt := NewRouter(&fakeSuggestMetadata{}, nil).WithReviewUnits(units).WithCreativeBrief(brief)
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
		{"incomplete brief", fmt.Errorf("%w: feel", application.ErrBriefIncomplete), "POST", "/v1/projects/p1/units/brief/main/approve", "", http.StatusBadRequest},
		{"locked project", domain.ErrInvalidStatus, "POST", "/v1/projects/p1/units/brief/main/unapprove", "", http.StatusConflict},
		{"unusable reply", fmt.Errorf("%w: thiếu ô feel", domain.ErrBriefReplyUnusable), "POST", "/v1/projects/p1/units/brief/main/redo", `{"note":"n"}`, http.StatusBadGateway},
		{"no model", application.ErrLLMNotConfigured, "POST", "/v1/projects/p1/units/brief/main/redo", "", http.StatusServiceUnavailable},
	}
	for _, c := range cases {
		rec := serveUnits(&fakeUnits{err: c.err}, &fakeBrief{}, c.method, c.path, c.body)
		if rec.Code != c.want {
			t.Errorf("%s: status %d, want %d (body %s)", c.name, rec.Code, c.want, rec.Body.String())
		}
	}
}

func TestEditAndRedoPassTheBodyThrough(t *testing.T) {
	units := &fakeUnits{unit: domain.ReviewUnit{Kind: domain.UnitKindBrief, Key: domain.BriefUnitKey}}
	if rec := serveUnits(units, &fakeBrief{}, "PUT", "/v1/projects/p1/units/brief/main", `{"fields":{"tone":"ấm"}}`); rec.Code != http.StatusOK {
		t.Fatalf("edit: status %d", rec.Code)
	}
	if units.lastFields["tone"] != "ấm" {
		t.Errorf("edit fields not passed through: %v", units.lastFields)
	}
	if rec := serveUnits(units, &fakeBrief{}, "POST", "/v1/projects/p1/units/brief/main/redo", `{"note":"ngắn hơn"}`); rec.Code != http.StatusOK {
		t.Fatalf("redo: status %d", rec.Code)
	}
	if units.lastNote != "ngắn hơn" {
		t.Errorf("redo note not passed through: %q", units.lastNote)
	}
}

func TestImpactReturnsItsItems(t *testing.T) {
	rec := serveUnits(&fakeUnits{}, &fakeBrief{}, "GET", "/v1/projects/p1/units/brief/main/impact", "")
	var body unitImpactResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || len(body.Items) != 1 {
		t.Fatalf("impact body %s, err %v", rec.Body.String(), err)
	}
}

func TestBriefRoutes(t *testing.T) {
	state := application.BriefState{TargetDuration: domain.Duration4to5, StoryStale: true, StoryPromptReadsBrief: true}
	rec := serveUnits(&fakeUnits{}, &fakeBrief{state: state}, "GET", "/v1/projects/p1/brief", "")
	var body briefStateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Unit != nil || body.TargetDuration != "4-5" || !body.StoryStale || !body.StoryPromptReadsBrief {
		t.Errorf("brief state body wrong: %+v", body)
	}
	if rec := serveUnits(&fakeUnits{}, &fakeBrief{}, "PUT", "/v1/projects/p1/brief/target-duration", `{"target_duration":"5-7"}`); rec.Code != http.StatusNoContent {
		t.Errorf("target duration: status %d", rec.Code)
	}
	if rec := serveUnits(&fakeUnits{}, &fakeBrief{err: domain.ErrInvalidStatus}, "PUT", "/v1/projects/p1/brief/target-duration", `{"target_duration":"5-7"}`); rec.Code != http.StatusConflict {
		t.Errorf("target duration of a project that is no longer a draft: status %d, want 409", rec.Code)
	}
	if rec := serveUnits(&fakeUnits{}, &fakeBrief{}, "PUT", "/v1/projects/p1/brief/platform", `{"platform":"YouTube"}`); rec.Code != http.StatusNoContent {
		t.Errorf("platform: status %d", rec.Code)
	}
	if rec := serveUnits(&fakeUnits{}, &fakeBrief{err: domain.ErrInvalidWizardInput}, "PUT", "/v1/projects/p1/brief/platform", `{"platform":"TikTok"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("platform outside the list: status %d, want 400", rec.Code)
	}
	if rec := serveUnits(&fakeUnits{}, &fakeBrief{}, "POST", "/v1/projects/p1/brief/generate", ""); rec.Code != http.StatusOK {
		t.Errorf("generate: status %d", rec.Code)
	}
}

// fakeDefaultModel keeps the default model of each step in memory.
type fakeDefaultModel struct {
	steps map[string]string
	err   error
}

func (f *fakeDefaultModel) StepModels(context.Context) (map[string]string, error) {
	return domain.WithStepModelFallback(f.steps), nil
}

func (f *fakeDefaultModel) SetStep(_ context.Context, step, model string) error {
	if f.err != nil {
		return f.err
	}
	if f.steps == nil {
		f.steps = map[string]string{}
	}
	f.steps[step] = model
	return nil
}

func TestStepDefaultModelSettingRoundTripsAndFeedsTheLLMStatus(t *testing.T) {
	setting := &fakeDefaultModel{}
	rt := NewRouter(&fakeSuggestMetadata{}, nil).WithReviewUnits(&fakeUnits{}).WithDefaultModelSetting(setting)
	do := func(method, path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		rt.Handler().ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
		return rec
	}
	var got aiModelSettingResponse
	if err := json.Unmarshal(do("GET", "/v1/settings/ai-model", "").Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.StepModels["brief"] != "ollama" || len(got.StepModels) != len(domain.ConfigurableModelSteps) || len(got.Steps) != len(domain.ConfigurableModelSteps) {
		t.Fatalf("with nothing saved every step is on the first catalog model, got %+v", got)
	}
	if rec := do("PUT", "/v1/settings/ai-model/steps/brief", `{"model":"zai-org/glm-5.3-flash"}`); rec.Code != http.StatusNoContent {
		t.Fatalf("put: status %d %s", rec.Code, rec.Body.String())
	}
	if steps := rt.effectiveStepDefaults(context.Background()); steps["brief"] != "zai-org/glm-5.3-flash" || steps["code"] != "ollama" {
		t.Errorf("the status must report the saved default and leave other steps alone, got %v", steps)
	}
	setting.err = fmt.Errorf("%w: step", domain.ErrInvalidWizardInput)
	if rec := do("PUT", "/v1/settings/ai-model/steps/render", `{"model":"ollama"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("a refused step must be a 400, got %d", rec.Code)
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
