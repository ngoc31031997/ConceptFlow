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

// fakeStory answers the story API with fixed values or a fixed error and keeps
// what it was asked.
type fakeStory struct {
	err      error
	note     string
	key      string
	factID   string
	use      bool
	frameRun int
}

func (f *fakeStory) State(context.Context, string) (application.StoryState, error) {
	return application.StoryState{Status: domain.UnitPending, Fields: []application.StoryFieldView{{Key: "question", Text: "Vì sao?"}}}, f.err
}
func (f *fakeStory) GenerateApproaches(_ context.Context, _, note string) ([]application.StoryApproachView, error) {
	f.note = note
	return []application.StoryApproachView{{Key: "A1", StoryApproach: domain.StoryApproach{Code: "HÀNH-TRÌNH"}}}, f.err
}
func (f *fakeStory) SelectApproach(_ context.Context, _, key string) error {
	f.key = key
	return f.err
}
func (f *fakeStory) SaveApproachAsArchetype(_ context.Context, _, key string) (application.StoryApproachView, error) {
	f.key = key
	return application.StoryApproachView{Key: key, StoryApproach: domain.StoryApproach{ArchetypeID: "row-1"}}, f.err
}
func (f *fakeStory) GenerateFrame(_ context.Context, _, note string) error {
	f.note = note
	f.frameRun++
	return f.err
}
func (f *fakeStory) SetFactUse(_ context.Context, _, factID string, use bool) error {
	f.factID, f.use = factID, use
	return f.err
}

func serveStory(f *fakeStory, method, path, body string) *httptest.ResponseRecorder {
	rt := NewRouter(&fakeSuggestMetadata{}, nil).WithStory(f).WithOperations(application.NewOperations())
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	rt.Handler().ServeHTTP(rec, req)
	return rec
}

func TestStoryStateIsServedUnderV1(t *testing.T) {
	rec := serveStory(&fakeStory{}, http.MethodGet, "/v1/projects/p1/story", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	var got struct {
		Status string `json:"status"`
		Fields []struct {
			Key string `json:"key"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got.Status != "pending" || len(got.Fields) != 1 || got.Fields[0].Key != "question" {
		t.Errorf("body = %s (%v), want the state with its fields", rec.Body, err)
	}
}

func TestStoryRunsPassTheNoteOn(t *testing.T) {
	f := &fakeStory{}
	rec := serveStory(f, http.MethodPost, "/v1/projects/p1/story/approaches/generate", `{"note":"thử cách khác"}`)
	if rec.Code != http.StatusOK || f.note != "thử cách khác" || !strings.Contains(rec.Body.String(), `"key":"A1"`) {
		t.Errorf("approaches: status %d note %q body %s", rec.Code, f.note, rec.Body)
	}
	rec = serveStory(f, http.MethodPost, "/v1/projects/p1/story/generate", "")
	if rec.Code != http.StatusOK || f.frameRun != 1 || !strings.Contains(rec.Body.String(), `"fields"`) {
		t.Errorf("generate: status %d runs %d body %s", rec.Code, f.frameRun, rec.Body)
	}
}

func TestStorySelectsAnApproachAndSavesAKind(t *testing.T) {
	f := &fakeStory{}
	if rec := serveStory(f, http.MethodPut, "/v1/projects/p1/story/approach", `{"key":"A2"}`); rec.Code != http.StatusNoContent || f.key != "A2" {
		t.Errorf("select: status %d key %q", rec.Code, f.key)
	}
	if rec := serveStory(f, http.MethodPost, "/v1/projects/p1/story/approaches/A3/archetype", ""); rec.Code != http.StatusOK || f.key != "A3" {
		t.Errorf("archetype: status %d key %q", rec.Code, f.key)
	}
}

func TestStoryFactUseNeedsTheFlag(t *testing.T) {
	f := &fakeStory{}
	if rec := serveStory(f, http.MethodPut, "/v1/projects/p1/story/facts/F02", `{"use":false}`); rec.Code != http.StatusNoContent || f.factID != "F02" || f.use {
		t.Errorf("status %d fact %q use %v, want F02 turned off", rec.Code, f.factID, f.use)
	}
	if rec := serveStory(f, http.MethodPut, "/v1/projects/p1/story/facts/F02", `{}`); rec.Code != http.StatusBadRequest {
		t.Errorf("a body without the flag must be refused, got %d", rec.Code)
	}
}

func TestStoryErrorsMapToStatuses(t *testing.T) {
	cases := []struct {
		name string
		err  error
		path string
		want int
	}{
		{"unusable frame reply", fmt.Errorf("x: %w", domain.ErrStoryReplyUnusable), "/v1/projects/p1/story/generate", http.StatusBadGateway},
		{"unusable approaches reply", fmt.Errorf("x: %w", domain.ErrApproachReplyUnusable), "/v1/projects/p1/story/approaches/generate", http.StatusBadGateway},
		{"no model", application.ErrLLMNotConfigured, "/v1/projects/p1/story/generate", http.StatusServiceUnavailable},
		{"step not open", fmt.Errorf("%w: %w", domain.ErrInvalidStatus, application.ErrStoryNeedsBrief), "/v1/projects/p1/story/generate", http.StatusConflict},
		{"nothing to write", application.ErrStoryNothingToWrite, "/v1/projects/p1/story/generate", http.StatusBadRequest},
	}
	for _, c := range cases {
		rec := serveStory(&fakeStory{err: c.err}, http.MethodPost, c.path, "")
		if rec.Code != c.want {
			t.Errorf("%s: status = %d, want %d (body %s)", c.name, rec.Code, c.want, rec.Body)
		}
	}
	if rec := serveStory(&fakeStory{err: errors.New("db down")}, http.MethodGet, "/v1/projects/p1/story", ""); rec.Code != http.StatusInternalServerError {
		t.Errorf("an unexpected error must be a 500, got %d", rec.Code)
	}
}

func TestStoryRoutesAreAbsentWhenNotWired(t *testing.T) {
	rt := NewRouter(&fakeSuggestMetadata{}, nil)
	rec := httptest.NewRecorder()
	rt.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/projects/p1/story", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 without the story step", rec.Code)
	}
}
