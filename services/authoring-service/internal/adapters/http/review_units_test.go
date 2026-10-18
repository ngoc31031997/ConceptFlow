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
		t.Errorf("target duration after the brief: status %d, want 409", rec.Code)
	}
	if rec := serveUnits(&fakeUnits{}, &fakeBrief{}, "POST", "/v1/projects/p1/brief/generate", ""); rec.Code != http.StatusOK {
		t.Errorf("generate: status %d", rec.Code)
	}
}

// fakeDefaultModel keeps the default model in memory.
type fakeDefaultModel struct {
	model string
	err   error
}

func (f *fakeDefaultModel) Get(context.Context) (string, error) { return f.model, nil }
func (f *fakeDefaultModel) Set(_ context.Context, model string) error {
	if f.err != nil {
		return f.err
	}
	f.model = model
	return nil
}

func TestDefaultModelSettingRoundTripsAndFeedsTheLLMStatus(t *testing.T) {
	setting := &fakeDefaultModel{}
	rt := NewRouter(&fakeSuggestMetadata{}, nil).WithReviewUnits(&fakeUnits{}).WithDefaultModel("server-model").WithDefaultModelSetting(setting)
	do := func(method, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		rt.Handler().ServeHTTP(rec, httptest.NewRequest(method, "/v1/settings/ai-model", strings.NewReader(body)))
		return rec
	}
	if rec := do("PUT", `{"default_model":"zai-org/glm-5.3-flash"}`); rec.Code != http.StatusNoContent {
		t.Fatalf("put: status %d %s", rec.Code, rec.Body.String())
	}
	var got aiModelSettingResponse
	if err := json.Unmarshal(do("GET", "").Body.Bytes(), &got); err != nil || got.DefaultModel != "zai-org/glm-5.3-flash" || got.ServerModel != "server-model" {
		t.Fatalf("get: %+v, %v", got, err)
	}
	if model := rt.effectiveDefaultModel(context.Background()); model != "zai-org/glm-5.3-flash" {
		t.Errorf("the status must report the setting, got %q", model)
	}
	setting.model = ""
	if model := rt.effectiveDefaultModel(context.Background()); model != "server-model" {
		t.Errorf("with no setting the server's model applies, got %q", model)
	}
	setting.err = fmt.Errorf("unknown model")
	if rec := do("PUT", `{"default_model":"x"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("a refused model must be a 400, got %d", rec.Code)
	}
}
