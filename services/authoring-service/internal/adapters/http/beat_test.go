package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"authoring/internal/application"
	"authoring/internal/domain"
)

// fakeBeats answers the beat sheet API with fixed values or a fixed error and
// keeps what it was asked.
type fakeBeats struct {
	err    error
	note   string
	after  string
	blank  bool
	order  []string
	key    string
	genRun int
}

func (f *fakeBeats) State(context.Context, string) (application.BeatState, error) {
	return application.BeatState{Status: domain.UnitPending, Beats: []application.BeatView{{Key: "b1"}}}, f.err
}
func (f *fakeBeats) Generate(_ context.Context, _, note string) error {
	f.note = note
	f.genRun++
	return f.err
}
func (f *fakeBeats) Add(_ context.Context, _, after string, blank bool, note string) (string, error) {
	f.after, f.blank, f.note = after, blank, note
	return "b2", f.err
}
func (f *fakeBeats) Reorder(_ context.Context, _ string, order []string) error {
	f.order = order
	return f.err
}
func (f *fakeBeats) Delete(_ context.Context, _, key string) error {
	f.key = key
	return f.err
}

func serveBeats(f *fakeBeats, method, path, body string) *httptest.ResponseRecorder {
	rt := NewRouter(&fakeSuggestMetadata{}, nil).WithBeats(f).WithOperations(application.NewOperations())
	rec := httptest.NewRecorder()
	rt.Handler().ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
	return rec
}

func TestBeatsStateIsServedUnderV1(t *testing.T) {
	rec := serveBeats(&fakeBeats{}, http.MethodGet, "/v1/projects/p1/beats", "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"key":"b1"`) {
		t.Errorf("status = %d, body %s, want the state with its beats", rec.Code, rec.Body)
	}
}

func TestBeatsRunsPassTheirArgumentsOn(t *testing.T) {
	f := &fakeBeats{}
	rec := serveBeats(f, http.MethodPost, "/v1/projects/p1/beats/generate", `{"note":"ngắn lại"}`)
	if rec.Code != http.StatusOK || f.note != "ngắn lại" || f.genRun != 1 || !strings.Contains(rec.Body.String(), `"beats"`) {
		t.Errorf("generate: status %d note %q runs %d", rec.Code, f.note, f.genRun)
	}
	rec = serveBeats(f, http.MethodPost, "/v1/projects/p1/beats", `{"after":"b1","blank":true}`)
	if rec.Code != http.StatusOK || f.after != "b1" || !f.blank {
		t.Errorf("add: status %d after %q blank %v", rec.Code, f.after, f.blank)
	}
	rec = serveBeats(f, http.MethodPut, "/v1/projects/p1/beats/order", `{"order":["b2","b1"]}`)
	if rec.Code != http.StatusOK || strings.Join(f.order, ",") != "b2,b1" {
		t.Errorf("order: status %d order %v", rec.Code, f.order)
	}
	rec = serveBeats(f, http.MethodDelete, "/v1/projects/p1/beats/b1", "")
	if rec.Code != http.StatusOK || f.key != "b1" {
		t.Errorf("delete: status %d key %q", rec.Code, f.key)
	}
}

func TestBeatsOrderNeedsABody(t *testing.T) {
	if rec := serveBeats(&fakeBeats{}, http.MethodPut, "/v1/projects/p1/beats/order", `nope`); rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestBeatsErrorsMapToStatuses(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		method string
		path   string
		want   int
	}{
		{"unusable reply", fmt.Errorf("x: %w", domain.ErrBeatReplyUnusable), http.MethodPost, "/v1/projects/p1/beats/generate", http.StatusBadGateway},
		{"no model", application.ErrLLMNotConfigured, http.MethodPost, "/v1/projects/p1/beats/generate", http.StatusServiceUnavailable},
		{"story not approved", fmt.Errorf("%w: %w", domain.ErrInvalidStatus, application.ErrBeatsNeedStory), http.MethodPost, "/v1/projects/p1/beats/generate", http.StatusConflict},
		{"sheet full", application.ErrBeatSheetFull, http.MethodPost, "/v1/projects/p1/beats", http.StatusBadRequest},
		{"no such beat", domain.ErrUnitNotFound, http.MethodDelete, "/v1/projects/p1/beats/b9", http.StatusNotFound},
	}
	for _, c := range cases {
		if rec := serveBeats(&fakeBeats{err: c.err}, c.method, c.path, ""); rec.Code != c.want {
			t.Errorf("%s: status = %d, want %d (body %s)", c.name, rec.Code, c.want, rec.Body)
		}
	}
	if rec := serveBeats(&fakeBeats{err: errors.New("db down")}, http.MethodGet, "/v1/projects/p1/beats", ""); rec.Code != http.StatusInternalServerError {
		t.Errorf("an unexpected error must be a 500, got %d", rec.Code)
	}
}

func TestBeatsRoutesAreAbsentWhenNotWired(t *testing.T) {
	rt := NewRouter(&fakeSuggestMetadata{}, nil)
	rec := httptest.NewRecorder()
	rt.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/projects/p1/beats", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 without the beat sheet step", rec.Code)
	}
}
