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

	"visual/internal/application"
	"visual/internal/domain"
)

// fakePlan answers every call with err, or a state whose Draft names the call.
type fakePlan struct {
	err  error
	last string
}

func (f *fakePlan) ret(call string) (*application.PlanState, error) {
	f.last = call
	if f.err != nil {
		return nil, f.err
	}
	return &application.PlanState{Draft: []string{call}}, nil
}
func (f *fakePlan) State(context.Context, string) (*application.PlanState, error) {
	return f.ret("state")
}
func (f *fakePlan) Edit(_ context.Context, _ string, id string, fields map[string]json.RawMessage) (*application.PlanState, error) {
	return f.ret(fmt.Sprintf("edit:%s:%d", id, len(fields)))
}
func (f *fakePlan) SetAnswer(_ context.Context, _ string, id, q, a string) (*application.PlanState, error) {
	return f.ret("answer:" + id + ":" + q + ":" + a)
}
func (f *fakePlan) Split(_ context.Context, _ string, id, at string) (*application.PlanState, error) {
	return f.ret("split:" + id + ":" + at)
}
func (f *fakePlan) Merge(_ context.Context, _ string, id, with string) (*application.PlanState, error) {
	return f.ret("merge:" + id + ":" + with)
}
func (f *fakePlan) Approve(_ context.Context, _ string, id string) (*application.PlanState, error) {
	return f.ret("approve:" + id)
}
func (f *fakePlan) Unapprove(_ context.Context, _ string, id string) (*application.PlanState, error) {
	return f.ret("unapprove:" + id)
}
func (f *fakePlan) ApproveAll(context.Context, string) (*application.PlanState, error) {
	return f.ret("approve-all")
}
func (f *fakePlan) ApplyFinding(_ context.Context, _ string, id string) (*application.PlanState, error) {
	return f.ret("apply:" + id)
}
func (f *fakePlan) AcceptFinding(_ context.Context, _ string, id, why string) (*application.PlanState, error) {
	return f.ret("accept:" + id + ":" + why)
}
func (f *fakePlan) Lock(_ context.Context, _ string, note string) (*application.PlanState, error) {
	return f.ret("lock:" + note)
}
func (f *fakePlan) Version(_ context.Context, _ string, n int) (application.PlanVersionRecord, error) {
	f.last = fmt.Sprintf("version:%d", n)
	return application.PlanVersionRecord{Version: n}, f.err
}
func (f *fakePlan) LatestDocument(context.Context, string) (json.RawMessage, error) {
	f.last = "latest"
	return json.RawMessage(`{"document_id":"visual-plan-v1"}`), f.err
}

func planRequest(f *fakePlan, method, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	NewRouter(&fakeDelete{}).WithPlan(f).Handler().ServeHTTP(rec, req)
	return rec
}

func TestPlanRoutesReachTheDesk(t *testing.T) {
	const base = "/v1/projects/p1/visual-plan"
	for _, c := range []struct{ method, path, body, want string }{
		{"GET", base + "/", "", "state"},
		{"PUT", base + "/units/VP-001", `{"fields":{"communication_goal":"x"}}`, "edit:VP-001:1"},
		{"PUT", base + "/units/VP-001/questions/Q1", `{"answer":"chia đôi"}`, "answer:VP-001:Q1:chia đôi"},
		{"POST", base + "/units/VP-001/split", `{"at_line":"b1.2"}`, "split:VP-001:b1.2"},
		{"POST", base + "/units/VP-001/merge", `{"with":"VP-002"}`, "merge:VP-001:VP-002"},
		{"POST", base + "/units/VP-001/approve", "", "approve:VP-001"},
		{"POST", base + "/units/VP-001/unapprove", "", "unapprove:VP-001"},
		{"POST", base + "/approve-all", "", "approve-all"},
		{"POST", base + "/findings/f1/apply", "", "apply:f1"},
		{"POST", base + "/findings/f1/accept", `{"why":"ổn"}`, "accept:f1:ổn"},
		{"POST", base + "/lock", `{"note":"v1"}`, "lock:v1"},
	} {
		f := &fakePlan{}
		rec := planRequest(f, c.method, c.path, c.body)
		var st application.PlanState
		_ = json.Unmarshal(rec.Body.Bytes(), &st)
		if rec.Code != http.StatusOK || len(st.Draft) != 1 || st.Draft[0] != c.want {
			t.Errorf("%s %s: status %d, desk saw %q, want %q (%s)", c.method, c.path, rec.Code, f.last, c.want, rec.Body.String())
		}
	}
}

func TestPlanVersionAndLatestDocument(t *testing.T) {
	f := &fakePlan{}
	if rec := planRequest(f, "GET", "/v1/projects/p1/visual-plan/versions/3", ""); rec.Code != 200 || f.last != "version:3" {
		t.Errorf("version: %d %s", rec.Code, f.last)
	}
	if rec := planRequest(f, "GET", "/v1/projects/p1/visual-plan/versions/zero", ""); rec.Code != 400 {
		t.Errorf("a non-number version is a bad request, got %d", rec.Code)
	}
	rec := planRequest(f, "GET", "/internal/v1/visual/p1/plan", "")
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "visual-plan-v1") {
		t.Errorf("latest plan: %d %s", rec.Code, rec.Body.String())
	}
	f.err = domain.ErrNoVersion
	if rec := planRequest(f, "GET", "/internal/v1/visual/p1/plan", ""); rec.Code != http.StatusNotFound {
		t.Errorf("no locked plan yet must be 404, got %d", rec.Code)
	}
}

func TestPlanErrorsMapToStatusAndCode(t *testing.T) {
	for _, c := range []struct {
		err    error
		status int
		code   string
	}{
		{domain.ErrScriptNotLocked, 409, "script_not_locked"},
		{domain.ErrConceptNotLocked, 409, "concept_not_locked"},
		{domain.ErrBeatStale, 409, "beat_stale"},
		{domain.ErrUnitInvalid, 409, ""},
		{domain.ErrLockNotReady, 409, ""},
		{domain.ErrUnknownPlanUnit, 404, ""},
		{domain.ErrUnknownQuestion, 404, ""},
		{domain.ErrNotSplittable, 400, ""},
		{domain.ErrNotMergeable, 400, ""},
		{domain.ErrInvalidEdit, 400, ""},
		{application.ErrAuthoringFailed, 502, ""},
		{errors.New("boom"), 500, ""},
	} {
		rec := planRequest(&fakePlan{err: c.err}, "POST", "/v1/projects/p1/visual-plan/units/VP-001/approve", "")
		var body struct{ Code string }
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		if rec.Code != c.status || body.Code != c.code {
			t.Errorf("%v: got %d %q, want %d %q", c.err, rec.Code, body.Code, c.status, c.code)
		}
	}
}

func TestPlanBadBodyIsRefused(t *testing.T) {
	if rec := planRequest(&fakePlan{}, "POST", "/v1/projects/p1/visual-plan/units/VP-001/split", `{not json`); rec.Code != 400 {
		t.Errorf("a malformed body must be a bad request, got %d", rec.Code)
	}
}

func TestPlanRoutesAreAbsentWithoutADesk(t *testing.T) {
	rec := httptest.NewRecorder()
	NewRouter(&fakeDelete{}).Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/v1/projects/p1/visual-plan/", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("no plan desk wired: want 404, got %d", rec.Code)
	}
}
