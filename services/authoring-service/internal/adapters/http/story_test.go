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

// fakeStory answers the story API with a fixed story or a fixed error and keeps
// what it was asked.
type fakeStory struct {
	err   error
	note  string
	patch application.StoryPatch
	runs  int
}

func (f *fakeStory) stored() domain.StoredStory {
	return domain.StoredStory{
		ProjectID: "p1", Status: domain.StoryPending,
		Story: domain.StoryStructure{CentralQuestion: "Vì sao?", CoreAnswer: "Vì thế.", NarrativeArc: []domain.ArcBeat{{Beat: "Hook", Text: "Mở"}}},
	}
}

func (f *fakeStory) State(context.Context, string) (application.StoryState, error) {
	s := f.stored()
	brief := domain.ProjectBrief{Audience: "Học sinh", DurationSeconds: 300, Tone: "Ấm"}
	return application.StoryState{
		Story: &s, Brief: &brief, ResearchResolved: true,
		Facts: []application.StoryFactView{{ID: "F01", ClaimVI: "Một", Type: domain.FactTypeFact}},
	}, f.err
}
func (f *fakeStory) Generate(_ context.Context, _, note string) (domain.StoredStory, error) {
	f.note = note
	f.runs++
	return f.stored(), f.err
}
func (f *fakeStory) Edit(_ context.Context, _ string, patch application.StoryPatch) (domain.StoredStory, error) {
	f.patch = patch
	return f.stored(), f.err
}
func (f *fakeStory) Approve(context.Context, string) (domain.StoredStory, error) {
	s := f.stored()
	s.Status = domain.StoryApproved
	return s, f.err
}
func (f *fakeStory) Unapprove(context.Context, string) (domain.StoredStory, error) {
	return f.stored(), f.err
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
		Story struct {
			CentralQuestion string `json:"central_question"`
			NarrativeArc    []struct {
				Beat  string   `json:"beat"`
				Facts []string `json:"facts"`
			} `json:"narrative_arc"`
			Status string `json:"status"`
		} `json:"story"`
		Facts []struct {
			ID string `json:"id"`
		} `json:"facts"`
		Brief            map[string]any `json:"brief"`
		BriefApproved    bool           `json:"brief_approved"`
		ResearchResolved bool           `json:"research_resolved"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("body %s: %v", rec.Body, err)
	}
	if got.Story.CentralQuestion != "Vì sao?" || got.Story.Status != "pending" || len(got.Story.NarrativeArc) != 1 || got.Story.NarrativeArc[0].Facts == nil {
		t.Errorf("story = %+v, want the stored story with an empty (not null) list of facts", got.Story)
	}
	if len(got.Facts) != 1 || got.Brief["audience"] != "Học sinh" || !got.BriefApproved || !got.ResearchResolved {
		t.Errorf("body = %s, want the facts, the brief and where the steps stand", rec.Body)
	}
}

func TestStoryStateWithoutAStoryHasNullStory(t *testing.T) {
	rt := NewRouter(&fakeSuggestMetadata{}, nil).WithStory(&emptyStory{})
	rec := httptest.NewRecorder()
	rt.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/projects/p1/story", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"story":null`) || !strings.Contains(rec.Body.String(), `"facts":[]`) || !strings.Contains(rec.Body.String(), `"brief":null`) {
		t.Errorf("status %d body %s, want a null story and brief and no facts", rec.Code, rec.Body)
	}
}

// emptyStory is a project whose first steps are not done: no story, brief or fact.
type emptyStory struct{ fakeStory }

func (*emptyStory) State(context.Context, string) (application.StoryState, error) {
	return application.StoryState{}, nil
}

func TestStoryGenerateEditApproveAndUnapprove(t *testing.T) {
	f := &fakeStory{}
	rec := serveStory(f, http.MethodPost, "/v1/projects/p1/story/generate", `{"note":"ngắn hơn"}`)
	if rec.Code != http.StatusOK || f.runs != 1 || f.note != "ngắn hơn" || !strings.Contains(rec.Body.String(), `"central_question":"Vì sao?"`) {
		t.Errorf("generate: status %d runs %d note %q body %s", rec.Code, f.runs, f.note, rec.Body)
	}
	rec = serveStory(f, http.MethodPut, "/v1/projects/p1/story", `{"core_answer":"Đáp án","narrative_arc":[{"beat":"A","text":"a","facts":["F01"]}]}`)
	if rec.Code != http.StatusOK || f.patch.CentralQuestion != nil || *f.patch.CoreAnswer != "Đáp án" || f.patch.NarrativeArc == nil || (*f.patch.NarrativeArc)[0].Facts[0] != "F01" {
		t.Errorf("edit: status %d patch %+v, want only the sent fields", rec.Code, f.patch)
	}
	if rec := serveStory(f, http.MethodPost, "/v1/projects/p1/story/approve", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"status":"approved"`) {
		t.Errorf("approve: status %d body %s", rec.Code, rec.Body)
	}
	if rec := serveStory(f, http.MethodPost, "/v1/projects/p1/story/unapprove", ""); rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"status":"pending"`) {
		t.Errorf("unapprove: status %d body %s", rec.Code, rec.Body)
	}
	if rec := serveStory(f, http.MethodPut, "/v1/projects/p1/story", `{bad`); rec.Code != http.StatusBadRequest {
		t.Errorf("a broken body must be refused, got %d", rec.Code)
	}
}

func TestStoryErrorsMapToStatuses(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		method string
		path   string
		want   int
	}{
		{"unusable reply", fmt.Errorf("x: %w", domain.ErrStoryReplyUnusable), http.MethodPost, "/v1/projects/p1/story/generate", http.StatusBadGateway},
		{"no model", application.ErrLLMNotConfigured, http.MethodPost, "/v1/projects/p1/story/generate", http.StatusServiceUnavailable},
		{"step not open", fmt.Errorf("%w: %w", domain.ErrInvalidStatus, application.ErrStoryNeedsBrief), http.MethodPost, "/v1/projects/p1/story/generate", http.StatusConflict},
		{"nothing to write", fmt.Errorf("%w: %w", domain.ErrInvalidUnitEdit, application.ErrStoryNothingToWrite), http.MethodPost, "/v1/projects/p1/story/generate", http.StatusBadRequest},
		{"invalid story", fmt.Errorf("%w: còn trống", domain.ErrStoryInvalid), http.MethodPost, "/v1/projects/p1/story/approve", http.StatusBadRequest},
		{"no story to approve", domain.ErrStoryNotFound, http.MethodPost, "/v1/projects/p1/story/approve", http.StatusNotFound},
		{"bad edit", fmt.Errorf("%w: dữ kiện F09", domain.ErrInvalidUnitEdit), http.MethodPut, "/v1/projects/p1/story", http.StatusBadRequest},
	}
	for _, c := range cases {
		rec := serveStory(&fakeStory{err: c.err}, c.method, c.path, `{"core_answer":"x"}`)
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
