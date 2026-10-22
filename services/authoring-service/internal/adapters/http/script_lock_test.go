package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"authoring/internal/application"
	"authoring/internal/domain"
)

type fakeLock struct {
	err    error
	called string
	id     string
	why    string
	role   string
	beat   string
	note   string
}

func (f *fakeLock) State(context.Context, string) (application.ScriptLockState, error) {
	return application.ScriptLockState{ScriptReady: true, NextVersion: 1}, f.err
}
func (f *fakeLock) Review(context.Context, string) error { f.called = "review"; return f.err }
func (f *fakeLock) ApplyFix(_ context.Context, _, id string) error {
	f.called, f.id = "apply", id
	return f.err
}
func (f *fakeLock) AddToHandoff(_ context.Context, _, id string) error {
	f.called, f.id = "handoff", id
	return f.err
}
func (f *fakeLock) Accept(_ context.Context, _, id, why string) error {
	f.called, f.id, f.why = "accept", id, why
	return f.err
}
func (f *fakeLock) AddNote(context.Context, string, string) error { f.called = "note"; return f.err }
func (f *fakeLock) AddTerm(context.Context, string, domain.GlossaryTerm) error {
	f.called = "term"
	return f.err
}
func (f *fakeLock) GenerateHandoff(_ context.Context, _, beat string) error {
	f.called, f.beat = "generate", beat
	return f.err
}
func (f *fakeLock) ConfirmHandoff(_ context.Context, _, beat string) error {
	f.called, f.beat = "confirm", beat
	return f.err
}
func (f *fakeLock) Lock(_ context.Context, _, note string) (application.VersionView, error) {
	f.called, f.note = "lock", note
	return application.VersionView{Version: 1}, f.err
}
func (f *fakeLock) HandoffForRole(_ context.Context, _, role string) (domain.HandoffPackage, error) {
	f.role = role
	return domain.HandoffPackage{Version: 2}, f.err
}

func lockDo(rt *Router, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	rt.Handler().ServeHTTP(rec, req)
	return rec
}

func TestScriptLockRoutesCallTheUseCaseAndReturnTheScreen(t *testing.T) {
	f := &fakeLock{}
	rt := NewRouter(&fakeSuggestMetadata{}, nil).WithScriptLock(f)
	cases := []struct {
		method, path, body, called string
	}{
		{http.MethodPost, "/v1/projects/p1/script-lock/review", "", "review"},
		{http.MethodPost, "/v1/projects/p1/script-lock/findings/r1/apply", "", "apply"},
		{http.MethodPost, "/v1/projects/p1/script-lock/findings/r1/handoff", "", "handoff"},
		{http.MethodPost, "/v1/projects/p1/script-lock/findings/m-must/accept", `{"why":"đã biết"}`, "accept"},
		{http.MethodPost, "/v1/projects/p1/script-lock/notes", `{"text":"x"}`, "note"},
		{http.MethodPost, "/v1/projects/p1/script-lock/glossary", `{"term":"a"}`, "term"},
		{http.MethodPost, "/v1/projects/p1/script-lock/handoff/generate", `{"beat":"b2"}`, "generate"},
		{http.MethodPost, "/v1/projects/p1/script-lock/handoff/b2/confirm", "", "confirm"},
		{http.MethodPost, "/v1/projects/p1/script-lock/lock", `{"note":"v1"}`, "lock"},
	}
	for _, c := range cases {
		f.called = ""
		rec := lockDo(rt, c.method, c.path, c.body)
		if rec.Code != http.StatusOK || f.called != c.called || !strings.Contains(rec.Body.String(), `"script_ready":true`) {
			t.Errorf("%s: status %d called %q body %s", c.path, rec.Code, f.called, rec.Body.String())
		}
	}
	if f.id != "m-must" || f.why != "đã biết" || f.beat != "b2" || f.note != "v1" {
		t.Errorf("arguments not passed through: %+v", f)
	}
	if rec := lockDo(rt, http.MethodGet, "/v1/projects/p1/script-lock", ""); rec.Code != http.StatusOK {
		t.Errorf("GET status %d", rec.Code)
	}
}

func TestScriptLockMapsErrorsToStatuses(t *testing.T) {
	cases := map[string]struct {
		err  error
		code int
	}{
		"not ready":      {errors.Join(domain.ErrInvalidStatus, domain.ErrLockNotReady), http.StatusConflict},
		"no finding":     {application.ErrFindingNotFound, http.StatusNotFound},
		"not applicable": {application.ErrFindingNotApplicable, http.StatusBadRequest},
		"bad reply":      {domain.ErrReviewReplyUnusable, http.StatusBadGateway},
		"no ai":          {application.ErrLLMNotConfigured, http.StatusServiceUnavailable},
	}
	for name, c := range cases {
		rt := NewRouter(&fakeSuggestMetadata{}, nil).WithScriptLock(&fakeLock{err: c.err})
		path := "/v1/projects/p1/script-lock/lock"
		if name == "bad reply" || name == "no ai" {
			path = "/v1/projects/p1/script-lock/review"
		}
		if name == "no finding" || name == "not applicable" {
			path = "/v1/projects/p1/script-lock/findings/x/apply"
		}
		if rec := lockDo(rt, http.MethodPost, path, ""); rec.Code != c.code {
			t.Errorf("%s: status %d, want %d (%s)", name, rec.Code, c.code, rec.Body.String())
		}
	}
}

func TestHandoffRouteReadsTheRoleFromTheQuery(t *testing.T) {
	f := &fakeLock{}
	rt := NewRouter(&fakeSuggestMetadata{}, nil).WithScriptLock(f)
	rec := lockDo(rt, http.MethodGet, "/internal/v1/authoring/p1/handoff?role=storyboard", "")
	if rec.Code != http.StatusOK || f.role != "storyboard" || !strings.Contains(rec.Body.String(), `"version":2`) {
		t.Errorf("status %d role %q body %s", rec.Code, f.role, rec.Body.String())
	}
	f.err = domain.ErrNoLockedVersion
	if rec := lockDo(rt, http.MethodGet, "/internal/v1/authoring/p1/handoff?role=code", ""); rec.Code != http.StatusNotFound {
		t.Errorf("no locked version: status %d, want 404", rec.Code)
	}
}

func TestScriptLockIsOffWithoutTheUseCase(t *testing.T) {
	rt := NewRouter(&fakeSuggestMetadata{}, nil)
	if rec := lockDo(rt, http.MethodGet, "/v1/projects/p1/script-lock", ""); rec.Code != http.StatusNotFound {
		t.Errorf("status %d, want 404", rec.Code)
	}
}

func TestFindingIDWithAColonArrivesDecoded(t *testing.T) {
	f := &fakeLock{}
	rt := NewRouter(&fakeSuggestMetadata{}, nil).WithScriptLock(f)
	rec := lockDo(rt, http.MethodPost, "/v1/projects/p1/script-lock/findings/m-flag%3Ab3.2/accept", `{"why":"x"}`)
	if rec.Code != http.StatusOK || f.id != "m-flag:b3.2" {
		t.Errorf("status %d id %q, want the id decoded to m-flag:b3.2", rec.Code, f.id)
	}
}
