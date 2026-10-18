package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"authoring/internal/application"
	"authoring/internal/domain"
)

type fakeSaveModels struct{ saved domain.AuthoringStepModels }

func (f *fakeSaveModels) Execute(_ context.Context, _ string, models domain.AuthoringStepModels) error {
	f.saved = models
	return nil
}

type fakeAuthoringState struct{ state application.AuthoringState }

func (f *fakeAuthoringState) Execute(context.Context, string) (application.AuthoringState, error) {
	return f.state, nil
}

func TestSaveModelsKeepsTheResearchAndPublishChoices(t *testing.T) {
	save := &fakeSaveModels{}
	rt := NewRouter(&fakeSuggestMetadata{}, nil).WithAuthoringModels(save)
	body := `{"brief":"b","publish":"p","research":"r","story":"s","storyboard":"sb","code":"c"}`
	rec := httptest.NewRecorder()
	rt.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPut, "/v1/projects/p1/authoring/models", strings.NewReader(body)))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	want := domain.AuthoringStepModels{Brief: "b", Publish: "p", Research: "r", Story: "s", Storyboard: "sb", Code: "c"}
	if save.saved != want {
		t.Fatalf("saved = %+v, want %+v", save.saved, want)
	}
}

func TestAuthoringStateReportsEveryStepModel(t *testing.T) {
	state := &fakeAuthoringState{state: application.AuthoringState{
		Mode:   "ai",
		Models: domain.AuthoringStepModels{Brief: "b", Publish: "p", Research: "r", Story: "s"},
	}}
	rt := NewRouter(&fakeSuggestMetadata{}, nil).WithAuthoringState(state)
	rec := httptest.NewRecorder()
	rt.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/projects/p1/authoring", nil))
	var got map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"brief_model": "b", "publish_model": "p", "research_model": "r", "story_model": "s"} {
		if got[key] != want {
			t.Errorf("%s = %q, want %q", key, got[key], want)
		}
	}
}
