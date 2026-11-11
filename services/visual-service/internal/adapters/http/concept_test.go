package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"visual/internal/application"
	"visual/internal/domain"
)

// fakeDesk answers every call with err, or a state naming the call.
type fakeDesk struct {
	err  error
	last string
}

func (f *fakeDesk) ret(call string) (*application.ConceptState, error) {
	f.last = call
	if f.err != nil {
		return nil, f.err
	}
	return &application.ConceptState{Model: call}, nil
}
func (f *fakeDesk) State(context.Context, string) (*application.ConceptState, error) {
	return f.ret("state")
}
func (f *fakeDesk) SetRefs(_ context.Context, _ string, ids []string) (*application.ConceptState, error) {
	return f.ret("refs:" + strings.Join(ids, ","))
}
func (f *fakeDesk) SetModel(_ context.Context, _ string, m string) (*application.ConceptState, error) {
	return f.ret("model:" + m)
}
func (f *fakeDesk) Generate(_ context.Context, _ string, note string, all bool) (*application.ConceptState, error) {
	return f.ret(fmt.Sprintf("generate:%s:%v", note, all))
}
func (f *fakeDesk) GenerateUnit(ctx context.Context, _ string, key, note string) (*application.ConceptState, error) {
	if application.UsingExternalAI(ctx) {
		// The provider builds the prompt of a prompt-mode request and stops.
		if _, err := application.NewExternalAIProvider(nil, nil).Chat(ctx, application.ChatRequest{System: "PROMPT"}); err != nil {
			return nil, err
		}
	}
	return f.ret("unit:" + key + ":" + note)
}
func (f *fakeDesk) Edit(_ context.Context, _ string, key string, fields map[string]json.RawMessage) (*application.ConceptState, error) {
	return f.ret(fmt.Sprintf("edit:%s:%d", key, len(fields)))
}
func (f *fakeDesk) Approve(_ context.Context, _ string, key string) (*application.ConceptState, error) {
	return f.ret("approve:" + key)
}
func (f *fakeDesk) Unapprove(_ context.Context, _ string, key string) (*application.ConceptState, error) {
	return f.ret("unapprove:" + key)
}
func (f *fakeDesk) ApproveAll(context.Context, string) (*application.ConceptState, error) {
	return f.ret("approve-all")
}
func (f *fakeDesk) Review(_ context.Context, _ string, note string) (*application.ConceptState, error) {
	return f.ret("review:" + note)
}
func (f *fakeDesk) MarkFindingRewritten(_ context.Context, _, id, _ string) (*application.ConceptState, error) {
	return f.ret("rewritten:" + id)
}

func (f *fakeDesk) ApplyFinding(_ context.Context, _ string, id string) (*application.ConceptState, error) {
	return f.ret("apply:" + id)
}
func (f *fakeDesk) AcceptFinding(_ context.Context, _ string, id, why string) (*application.ConceptState, error) {
	return f.ret("accept:" + id + ":" + why)
}
func (f *fakeDesk) Lock(_ context.Context, _ string, note string) (*application.ConceptState, error) {
	return f.ret("lock:" + note)
}
func (f *fakeDesk) Version(_ context.Context, _ string, n int) (application.VersionRecord, error) {
	f.last = fmt.Sprint("version:", n)
	return application.VersionRecord{Version: n}, f.err
}
func (f *fakeDesk) LatestDocument(_ context.Context, _ string, step string) (json.RawMessage, error) {
	f.last = "latest:" + step
	if f.err != nil {
		return nil, f.err
	}
	return json.RawMessage(`{"document_id":"visual-concept-v1"}`), nil
}

type del struct{}

func (del) Execute(context.Context, string) error { return nil }

func serve(t *testing.T, desk *fakeDesk, method, path, body string, header map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	h := NewRouter(del{}).WithConcept(desk).Handler()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	for k, v := range header {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestConceptRoutes(t *testing.T) {
	base := "/v1/projects/test-cr112b-h/visual-concept"
	cases := []struct{ method, path, body, want string }{
		{"GET", base, "", "state"},
		{"PUT", base + "/refs", `{"ref_ids":["R01","R02"]}`, "refs:R01,R02"},
		{"PUT", base + "/model", `{"model":"m"}`, "model:m"},
		{"POST", base + "/generate", `{"note":"n","rewrite_all":true}`, "generate:n:true"},
		{"POST", base + "/generate", ``, "generate::false"},
		{"POST", base + "/units/typography/generate", `{"note":"x"}`, "unit:typography:x"},
		{"POST", base + "/units/typography/generate", `{"note":"x","finding_id":"f1"}`, "rewritten:f1"},
		{"PUT", base + "/units/typography", `{"fields":{"font":"A","captions":"c"}}`, "edit:typography:2"},
		{"POST", base + "/units/typography/approve", ``, "approve:typography"},
		{"POST", base + "/units/typography/unapprove", ``, "unapprove:typography"},
		{"POST", base + "/approve-all", ``, "approve-all"},
		{"POST", base + "/review", `{"note":"r"}`, "review:r"},
		{"POST", base + "/findings/f1/apply", ``, "apply:f1"},
		{"POST", base + "/findings/m-terms/accept", `{"why":"w"}`, "accept:m-terms:w"},
		{"POST", base + "/lock", `{"note":"v"}`, "lock:v"},
		{"GET", base + "/versions/2", ``, "version:2"},
		{"GET", "/internal/v1/visual/test-cr112b-h/concept", ``, "latest:"},
	}
	for _, c := range cases {
		desk := &fakeDesk{}
		rec := serve(t, desk, c.method, c.path, c.body, nil)
		if rec.Code != http.StatusOK || desk.last != c.want {
			t.Errorf("%s %s: %d %q want %q", c.method, c.path, rec.Code, desk.last, c.want)
		}
	}
	if rec := serve(t, &fakeDesk{}, "PUT", base+"/refs", `{bad`, nil); rec.Code != http.StatusBadRequest {
		t.Errorf("bad body %d", rec.Code)
	}
	if rec := serve(t, &fakeDesk{}, "GET", base+"/versions/zero", ``, nil); rec.Code != http.StatusBadRequest {
		t.Errorf("bad version %d", rec.Code)
	}
}

func TestConceptErrors(t *testing.T) {
	cases := []struct {
		err    error
		status int
	}{
		{domain.ErrScriptNotLocked, http.StatusConflict},
		{domain.ErrUnknownUnit, http.StatusNotFound},
		{domain.ErrNoVersion, http.StatusNotFound},
		{domain.ErrInvalidEdit, http.StatusBadRequest},
		{domain.ErrUnknownReference, http.StatusBadRequest},
		{domain.ErrLockNotReady, http.StatusConflict},
		{domain.ErrNothingToLock, http.StatusConflict},
		{domain.ErrUnitRunning, http.StatusConflict},
		{domain.ErrUnreadableReply, http.StatusBadGateway},
		{application.ErrLLMFailed, http.StatusBadGateway},
		{application.ErrAuthoringFailed, http.StatusBadGateway},
		{fmt.Errorf("boom"), http.StatusInternalServerError},
	}
	for _, c := range cases {
		rec := serve(t, &fakeDesk{err: c.err}, "GET", "/v1/projects/p/visual-concept", "", nil)
		if rec.Code != c.status {
			t.Errorf("%v: %d want %d", c.err, rec.Code, c.status)
		}
	}
	rec := serve(t, &fakeDesk{err: domain.ErrScriptNotLocked}, "GET", "/v1/projects/p/visual-concept", "", nil)
	var body struct{ Error, Code string }
	_ = json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Code != "script_not_locked" || body.Error == "" {
		t.Errorf("body %s", rec.Body)
	}
	if rec := serve(t, &fakeDesk{err: domain.ErrUnknownStep}, "GET", "/internal/v1/visual/p/concept?step=x", "", nil); rec.Code != http.StatusBadRequest {
		t.Errorf("unknown step %d", rec.Code)
	}
}

func TestConceptExternalAI(t *testing.T) {
	base := "/v1/projects/p/visual-concept"
	rec := serve(t, &fakeDesk{}, "POST", base+"/units/typography/generate", `{}`, map[string]string{"X-External-AI": "prompt"})
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"external_prompt":"PROMPT"`) {
		t.Errorf("prompt mode %d %s", rec.Code, rec.Body)
	}
	rec = serve(t, &fakeDesk{}, "POST", base+"/units/typography/generate", `{"note":"n"}`, map[string]string{"X-External-AI": "paste"})
	if rec.Code != http.StatusBadRequest {
		t.Errorf("paste without a reply %d", rec.Code)
	}
	if rec := serve(t, &fakeDesk{}, "POST", base+"/review", `{}`, map[string]string{"X-External-AI": "nope"}); rec.Code != http.StatusBadRequest {
		t.Errorf("unknown mode %d", rec.Code)
	}
	desk := &fakeDesk{}
	if rec := serve(t, desk, "POST", base+"/generate", `{}`, map[string]string{"X-External-AI": "prompt"}); rec.Code != http.StatusOK || desk.last != "generate::false" {
		t.Error("generating every part runs many turns and ignores the external-AI header")
	}
}
