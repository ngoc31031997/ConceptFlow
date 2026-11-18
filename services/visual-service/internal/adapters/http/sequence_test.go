package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"visual/internal/application"
	"visual/internal/domain"
)

// fakeSequence answers every call with err, or a state whose Draft names the call.
type fakeSequence struct {
	err  error
	last string
}

func (f *fakeSequence) ret(call string) (*application.SequenceState, error) {
	f.last = call
	if f.err != nil {
		return nil, f.err
	}
	return &application.SequenceState{Draft: []string{call}}, nil
}
func (f *fakeSequence) State(context.Context, string) (*application.SequenceState, error) {
	return f.ret("state")
}
func (f *fakeSequence) SetModel(_ context.Context, _ string, m string) (*application.SequenceState, error) {
	return f.ret("model " + m)
}
func (f *fakeSequence) GenerateMotifs(_ context.Context, _ string, note string) (*application.SequenceState, error) {
	return f.ret("motifs " + note)
}
func (f *fakeSequence) Generate(_ context.Context, _ string, note string) (*application.SequenceState, error) {
	return f.ret("generate " + note)
}
func (f *fakeSequence) GenerateBeat(_ context.Context, _ string, k, note string) (*application.SequenceState, error) {
	return f.ret("beat " + k + " " + note)
}
func (f *fakeSequence) GenerateUnit(_ context.Context, _ string, id, note string) (*application.SequenceState, error) {
	return f.ret("unit " + id + " " + note)
}
func (f *fakeSequence) Review(_ context.Context, _ string, note string) (*application.SequenceState, error) {
	return f.ret("review " + note)
}
func (f *fakeSequence) Edit(_ context.Context, _ string, id string, fields map[string]json.RawMessage) (*application.SequenceState, error) {
	return f.ret(fmt.Sprintf("edit:%s:%d", id, len(fields)))
}
func (f *fakeSequence) Approve(_ context.Context, _ string, id string) (*application.SequenceState, error) {
	return f.ret("approve:" + id)
}
func (f *fakeSequence) Unapprove(_ context.Context, _ string, id string) (*application.SequenceState, error) {
	return f.ret("unapprove:" + id)
}
func (f *fakeSequence) ApproveAll(context.Context, string) (*application.SequenceState, error) {
	return f.ret("approve-all")
}
func (f *fakeSequence) ConfirmBeat(_ context.Context, _ string, beat string) (*application.SequenceState, error) {
	return f.ret("confirm:" + beat)
}
func (f *fakeSequence) AddMotif(_ context.Context, _ string, fields map[string]json.RawMessage) (*application.SequenceState, error) {
	return f.ret(fmt.Sprintf("add-motif:%d", len(fields)))
}
func (f *fakeSequence) EditMotif(_ context.Context, _ string, id string, fields map[string]json.RawMessage) (*application.SequenceState, error) {
	return f.ret(fmt.Sprintf("edit-motif:%s:%d", id, len(fields)))
}
func (f *fakeSequence) DeleteMotif(_ context.Context, _ string, id string) (*application.SequenceState, error) {
	return f.ret("delete-motif:" + id)
}
func (f *fakeSequence) ApproveMotifs(context.Context, string) (*application.SequenceState, error) {
	return f.ret("approve-motifs")
}
func (f *fakeSequence) UnapproveMotifs(context.Context, string) (*application.SequenceState, error) {
	return f.ret("unapprove-motifs")
}
func (f *fakeSequence) ApplyFinding(_ context.Context, _ string, id string) (*application.SequenceState, error) {
	return f.ret("apply:" + id)
}
func (f *fakeSequence) AcceptFinding(_ context.Context, _ string, id, why string) (*application.SequenceState, error) {
	return f.ret("accept:" + id + ":" + why)
}
func (f *fakeSequence) Lock(_ context.Context, _ string, note string) (*application.SequenceState, error) {
	return f.ret("lock:" + note)
}
func (f *fakeSequence) Version(_ context.Context, _ string, n int) (application.SequenceVersionRecord, error) {
	f.last = fmt.Sprintf("version:%d", n)
	return application.SequenceVersionRecord{Version: n}, f.err
}
func (f *fakeSequence) LatestDocument(context.Context, string) (json.RawMessage, error) {
	f.last = "latest"
	return json.RawMessage(`{"document_id":"visual-sequence-v1"}`), f.err
}

func sequenceRequest(f *fakeSequence, method, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	NewRouter(&fakeDelete{}).WithSequence(f).Handler().ServeHTTP(rec, req)
	return rec
}

func TestSequenceRoutesReachTheDesk(t *testing.T) {
	const base = "/v1/projects/p1/visual-sequence"
	for _, c := range []struct{ method, path, body, want string }{
		{"GET", base + "/", "", "state"},
		{"PUT", base + "/model", `{"model":"deepseek-chat"}`, "model deepseek-chat"},
		{"POST", base + "/motifs/generate", `{"note":"n"}`, "motifs n"},
		{"POST", base + "/generate", `{"note":"n"}`, "generate n"},
		{"POST", base + "/beats/b2/generate", `{"note":"n"}`, "beat b2 n"},
		{"POST", base + "/units/VP-003/generate", `{"note":"n"}`, "unit VP-003 n"},
		{"POST", base + "/review", `{"note":"n"}`, "review n"},
		{"PUT", base + "/units/VP-001", `{"fields":{"purpose_in_flow":"x","pacing":{}}}`, "edit:VP-001:2"},
		{"POST", base + "/units/VP-001/approve", "", "approve:VP-001"},
		{"POST", base + "/units/VP-001/unapprove", "", "unapprove:VP-001"},
		{"POST", base + "/approve-all", "", "approve-all"},
		{"POST", base + "/beats/b2/confirm", "", "confirm:b2"},
		{"POST", base + "/motifs", `{"fields":{"name":"Ong"}}`, "add-motif:1"},
		{"PUT", base + "/motifs/MOTIF-01", `{"fields":{"name":"Ong","role":"neo"}}`, "edit-motif:MOTIF-01:2"},
		{"DELETE", base + "/motifs/MOTIF-01", "", "delete-motif:MOTIF-01"},
		{"POST", base + "/motifs/approve", "", "approve-motifs"},
		{"POST", base + "/motifs/unapprove", "", "unapprove-motifs"},
		{"POST", base + "/findings/f1/apply", "", "apply:f1"},
		{"POST", base + "/findings/m-seq-via/accept", `{"why":"cố ý"}`, "accept:m-seq-via:cố ý"},
		{"POST", base + "/lock", `{"note":"v1"}`, "lock:v1"},
		{"GET", base + "/versions/2", "", "version:2"},
		{"GET", "/internal/v1/visual/p1/sequence", "", "latest"},
	} {
		f := &fakeSequence{}
		rec := sequenceRequest(f, c.method, c.path, c.body)
		if rec.Code != 200 || f.last != c.want {
			t.Errorf("%s %s: status %d, call %q, want 200 and %q: %s", c.method, c.path, rec.Code, f.last, c.want, rec.Body.String())
		}
	}
}

func TestSequenceRoutesMapErrors(t *testing.T) {
	const base = "/v1/projects/p1/visual-sequence"
	for _, c := range []struct {
		err    error
		status int
		code   string
	}{
		{domain.ErrPlanNotLocked, 409, "plan_not_locked"},
		{domain.ErrPlanOutdated, 409, "plan_outdated"},
		{domain.ErrConceptNotLocked, 409, "concept_not_locked"},
		{domain.ErrBeatStale, 409, "beat_stale"},
		{domain.ErrMotifsNotApproved, 409, "motifs_not_approved"},
		{domain.ErrUnreadableReply, 502, ""},
		{domain.ErrNoMotif, 409, ""},
		{domain.ErrUnknownMotif, 404, ""},
		{domain.ErrUnknownPlanUnit, 404, ""},
		{domain.ErrInvalidEdit, 400, ""},
	} {
		rec := sequenceRequest(&fakeSequence{err: c.err}, "POST", base+"/lock", `{}`)
		var body struct{ Error, Code string }
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		if rec.Code != c.status || body.Code != c.code || body.Error == "" {
			t.Errorf("%v: want %d %q, got %d %+v", c.err, c.status, c.code, rec.Code, body)
		}
	}
	if rec := sequenceRequest(&fakeSequence{}, "GET", base+"/versions/x", ""); rec.Code != 400 {
		t.Errorf("a version that is not a number: want 400, got %d", rec.Code)
	}
	if rec := sequenceRequest(&fakeSequence{}, "PUT", base+"/units/VP-001", `not json`); rec.Code != 400 {
		t.Errorf("a body that is not JSON: want 400, got %d", rec.Code)
	}
	if rec := sequenceRequest(&fakeSequence{err: domain.ErrNoVersion}, "GET", "/internal/v1/visual/p1/sequence", ""); rec.Code != 404 {
		t.Errorf("no locked sequence: want 404, got %d", rec.Code)
	}
}

func TestSequenceAIRoutesTakeTheExternalAIHeader(t *testing.T) {
	const base = "/v1/projects/p1/visual-sequence"
	for _, path := range []string{"/motifs/generate", "/units/VP-001/generate", "/review"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", base+path, strings.NewReader(`{}`))
		req.Header.Set("X-External-AI", "nope")
		NewRouter(&fakeDelete{}).WithSequence(&fakeSequence{}).Handler().ServeHTTP(rec, req)
		if rec.Code != 400 {
			t.Errorf("%s with a bad X-External-AI: want 400, got %d", path, rec.Code)
		}
	}
}
