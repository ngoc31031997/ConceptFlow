package application_test

import (
	"context"
	"testing"

	"orchestrator/internal/application"
)

// fakeAuthoringStore backs the tests below for CR-025's authoring
// save/read use cases — a bare in-memory map is enough since these use
// cases do nothing but validate and delegate.
type fakeAuthoringStore struct {
	story      map[string]string
	storyboard map[string]string
}

func newFakeAuthoringStore() *fakeAuthoringStore {
	return &fakeAuthoringStore{story: map[string]string{}, storyboard: map[string]string{}}
}

func (f *fakeAuthoringStore) SaveAuthoringStory(_ context.Context, projectID, content string) error {
	f.story[projectID] = content
	return nil
}

func (f *fakeAuthoringStore) GetAuthoringStory(_ context.Context, projectID string) (string, error) {
	return f.story[projectID], nil
}

func (f *fakeAuthoringStore) SaveAuthoringStoryboard(_ context.Context, projectID, content string) error {
	f.storyboard[projectID] = content
	return nil
}

func (f *fakeAuthoringStore) GetAuthoringStoryboard(_ context.Context, projectID string) (string, error) {
	return f.storyboard[projectID], nil
}

func TestSaveAuthoringStoryUseCase(t *testing.T) {
	store := newFakeAuthoringStore()
	uc := application.NewSaveAuthoringStoryUseCase(store)

	if err := uc.Execute(context.Background(), "", "some story"); err == nil {
		t.Fatal("expected error for empty project_id")
	}
	if err := uc.Execute(context.Background(), "p1", ""); err == nil {
		t.Fatal("expected error for empty content")
	}
	if err := uc.Execute(context.Background(), "p1", "the story"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := store.story["p1"]; got != "the story" {
		t.Fatalf("story not saved, got %q", got)
	}
}

func TestSaveAuthoringStoryboardUseCase(t *testing.T) {
	store := newFakeAuthoringStore()
	uc := application.NewSaveAuthoringStoryboardUseCase(store)

	if err := uc.Execute(context.Background(), "", "some storyboard"); err == nil {
		t.Fatal("expected error for empty project_id")
	}
	if err := uc.Execute(context.Background(), "p1", ""); err == nil {
		t.Fatal("expected error for empty content")
	}
	if err := uc.Execute(context.Background(), "p1", "the storyboard"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := store.storyboard["p1"]; got != "the storyboard" {
		t.Fatalf("storyboard not saved, got %q", got)
	}
}

func TestGetAuthoringStateUseCase(t *testing.T) {
	store := newFakeAuthoringStore()
	store.story["p1"] = "the story"
	store.storyboard["p1"] = "the storyboard"

	uc := application.NewGetAuthoringStateUseCase(store)
	state, err := uc.Execute(context.Background(), "p1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Story != "the story" || state.Storyboard != "the storyboard" {
		t.Fatalf("unexpected state: %+v", state)
	}

	// A project that never saved anything answers empty strings, not an error.
	empty, err := uc.Execute(context.Background(), "unknown")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if empty.Story != "" || empty.Storyboard != "" {
		t.Fatalf("expected empty state, got %+v", empty)
	}
}
