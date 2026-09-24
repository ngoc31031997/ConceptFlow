package application_test

import (
	"context"
	"testing"

	"orchestrator/internal/application"
	"orchestrator/internal/domain"
)

// fakeAuthoringStore backs the tests below for CR-025's authoring
// save/read use cases — a bare in-memory map is enough since these use
// cases do nothing but validate and delegate.
type fakeAuthoringStore struct {
	topic      map[string]string
	story      map[string]string
	storyboard map[string]string
	code       map[string]string
	// status defaults to domain.StatusDraft (Go zero value is "", so
	// GetStatus below maps "" to draft) — set per project_id to simulate a
	// project whose render has already started (CR-028 FR84.2).
	status map[string]domain.ProjectStatus
	// mode backs CR-027 FR79's step-1 working mode. Unset means a project
	// whose row predates the column, which reads back as the default.
	mode map[string]string
	// models backs the model-per-step picker. Unset means a project whose
	// row predates the columns, which reads back as the zero value (every
	// step "" — server default).
	models map[string]domain.AuthoringStepModels
}

func newFakeAuthoringStore() *fakeAuthoringStore {
	return &fakeAuthoringStore{
		topic:      map[string]string{},
		story:      map[string]string{},
		storyboard: map[string]string{},
		code:       map[string]string{},
		status:     map[string]domain.ProjectStatus{},
		mode:       map[string]string{},
		models:     map[string]domain.AuthoringStepModels{},
	}
}

// SaveAuthoringMode/GetAuthoringMode back CR-027 FR79's step-1 working mode.
func (f *fakeAuthoringStore) SaveAuthoringMode(_ context.Context, projectID, mode string) error {
	f.mode[projectID] = mode
	return nil
}

func (f *fakeAuthoringStore) GetAuthoringMode(_ context.Context, projectID string) (string, error) {
	return f.mode[projectID], nil
}

// SaveAuthoringModels/GetAuthoringModels back the model-per-step picker.
func (f *fakeAuthoringStore) SaveAuthoringModels(_ context.Context, projectID string, models domain.AuthoringStepModels) error {
	f.models[projectID] = models
	return nil
}

func (f *fakeAuthoringStore) GetAuthoringModels(_ context.Context, projectID string) (domain.AuthoringStepModels, error) {
	return f.models[projectID], nil
}

// GetStatus backs CR-028 FR84.2's authoring lock. Defaults to draft (unset
// entries) so every pre-existing test above, which never touches status,
// keeps passing unmodified.
func (f *fakeAuthoringStore) GetStatus(_ context.Context, projectID string) (domain.ProjectStatus, error) {
	if s, ok := f.status[projectID]; ok {
		return s, nil
	}
	return domain.StatusDraft, nil
}

// ClearAuthoringSteps backs the downstream clearing on an upstream change.
func (f *fakeAuthoringStore) ClearAuthoringSteps(_ context.Context, projectID string, steps ...application.AuthoringStep) error {
	for _, step := range steps {
		switch step {
		case application.AuthoringStepStory:
			f.story[projectID] = ""
		case application.AuthoringStepStoryboard:
			f.storyboard[projectID] = ""
		case application.AuthoringStepCode:
			f.code[projectID] = ""
		}
	}
	return nil
}

// SaveAuthoringStory mirrors the repository's CR-027 D0 rule: an empty topic
// leaves the stored one alone rather than clearing it.
func (f *fakeAuthoringStore) SaveAuthoringStory(_ context.Context, projectID, content, topic string) error {
	f.story[projectID] = content
	if topic != "" {
		f.topic[projectID] = topic
	}
	return nil
}

func (f *fakeAuthoringStore) GetAuthoringTopic(_ context.Context, projectID string) (string, error) {
	return f.topic[projectID], nil
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

func (f *fakeAuthoringStore) SaveAuthoringCode(_ context.Context, projectID, content string) error {
	f.code[projectID] = content
	return nil
}

func (f *fakeAuthoringStore) GetAuthoringCode(_ context.Context, projectID string) (string, error) {
	return f.code[projectID], nil
}

func TestSaveAuthoringStoryUseCase(t *testing.T) {
	store := newFakeAuthoringStore()
	uc := application.NewSaveAuthoringStoryUseCase(store, store, store)

	if err := uc.Execute(context.Background(), "", "some story", "a topic"); err == nil {
		t.Fatal("expected error for empty project_id")
	}
	if err := uc.Execute(context.Background(), "p1", "", "a topic"); err == nil {
		t.Fatal("expected error for empty content")
	}
	if err := uc.Execute(context.Background(), "p1", "the story", "why the sky is blue"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := store.story["p1"]; got != "the story" {
		t.Fatalf("story not saved, got %q", got)
	}
	if got := store.topic["p1"]; got != "why the sky is blue" {
		t.Fatalf("topic not saved, got %q", got)
	}
}

// TestSaveAuthoringStoryUseCase_EmptyTopicIsAllowedAndKeepsTheStoredOne covers
// CR-027 D0. A missing topic must not block saving the outline — that would
// break every pre-CR-027 caller for a field it does not know about — and it
// must not wipe a topic the Creator already gave us, since every later
// pipeline step renders {{topic}} from it.
func TestSaveAuthoringStoryUseCase_EmptyTopicIsAllowedAndKeepsTheStoredOne(t *testing.T) {
	store := newFakeAuthoringStore()
	uc := application.NewSaveAuthoringStoryUseCase(store, store, store)

	if err := uc.Execute(context.Background(), "p1", "v1", "the real topic"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := uc.Execute(context.Background(), "p1", "v2", ""); err != nil {
		t.Fatalf("an empty topic must not be an error: %v", err)
	}
	if got := store.story["p1"]; got != "v2" {
		t.Fatalf("outline should have been replaced, got %q", got)
	}
	if got := store.topic["p1"]; got != "the real topic" {
		t.Fatalf("an empty topic must leave the stored one alone, got %q", got)
	}
}

// Changing an upstream output drops what was built on the old one; saving the
// same text again must not.
func TestSaveAuthoring_ChangingUpstreamClearsDownstream(t *testing.T) {
	ctx := context.Background()
	store := newFakeAuthoringStore()
	story := application.NewSaveAuthoringStoryUseCase(store, store, store)
	storyboard := application.NewSaveAuthoringStoryboardUseCase(store, store, store)
	code := application.NewSaveAuthoringCodeUseCase(store, store, store)

	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(story.Execute(ctx, "p1", "outline", "topic"))
	must(storyboard.Execute(ctx, "p1", "board"))
	must(code.Execute(ctx, "p1", "code"))

	must(story.Execute(ctx, "p1", "outline", "topic"))
	if store.storyboard["p1"] != "board" || store.code["p1"] != "code" {
		t.Fatal("re-saving an identical outline must not clear anything")
	}

	must(storyboard.Execute(ctx, "p1", "board v2"))
	if store.code["p1"] != "" {
		t.Fatalf("a changed storyboard must clear the code, got %q", store.code["p1"])
	}
	if store.story["p1"] != "outline" {
		t.Fatal("changing the storyboard must not touch the outline")
	}

	must(code.Execute(ctx, "p1", "code v2"))
	must(story.Execute(ctx, "p1", "outline v2", ""))
	if store.storyboard["p1"] != "" || store.code["p1"] != "" {
		t.Fatal("a changed outline must clear the storyboard and the code")
	}
}

func TestSaveAuthoringStoryboardUseCase(t *testing.T) {
	store := newFakeAuthoringStore()
	uc := application.NewSaveAuthoringStoryboardUseCase(store, store, store)

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

func TestSaveAuthoringCodeUseCase(t *testing.T) {
	store := newFakeAuthoringStore()
	uc := application.NewSaveAuthoringCodeUseCase(store, store, store)

	if err := uc.Execute(context.Background(), "", "some code"); err == nil {
		t.Fatal("expected error for empty project_id")
	}
	if err := uc.Execute(context.Background(), "p1", ""); err == nil {
		t.Fatal("expected error for empty content")
	}
	if err := uc.Execute(context.Background(), "p1", "the code"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := store.code["p1"]; got != "the code" {
		t.Fatalf("code not saved, got %q", got)
	}
}

func TestGetAuthoringStateUseCase(t *testing.T) {
	store := newFakeAuthoringStore()
	store.story["p1"] = "the story"
	store.storyboard["p1"] = "the storyboard"
	store.code["p1"] = "the code"

	uc := application.NewGetAuthoringStateUseCase(store)
	state, err := uc.Execute(context.Background(), "p1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Story != "the story" || state.Storyboard != "the storyboard" || state.Code != "the code" {
		t.Fatalf("unexpected state: %+v", state)
	}

	// A project that never saved anything answers empty strings, not an error.
	empty, err := uc.Execute(context.Background(), "unknown")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if empty.Story != "" || empty.Storyboard != "" || empty.Code != "" {
		t.Fatalf("expected empty state, got %+v", empty)
	}
}

// CR-027 FR79 — the step-1 working mode lives in the database, so a project
// picked up again on any tab (another browser, after a restart) still knows how
// its Creator chose to work.
func TestSaveAuthoringModeRoundTrip(t *testing.T) {
	store := newFakeAuthoringStore()
	save := application.NewSaveAuthoringModeUseCase(store)
	read := application.NewGetAuthoringStateUseCase(store)

	if err := save.Execute(context.Background(), "p1", "ai"); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	state, err := read.Execute(context.Background(), "p1")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if state.Mode != "ai" {
		t.Errorf("mode = %q, want ai", state.Mode)
	}
}

// A project whose row predates the column reads back as manual — what it was
// actually doing — not as an empty third mode the GUI would have to guess at.
func TestGetAuthoringStateDefaultsModeToManual(t *testing.T) {
	store := newFakeAuthoringStore()
	state, err := application.NewGetAuthoringStateUseCase(store).Execute(context.Background(), "old-project")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if state.Mode != string(domain.AuthoringModeManual) {
		t.Errorf("mode = %q, want manual", state.Mode)
	}
}

// An unknown mode is rejected rather than normalised: it means the client and
// the server disagree about what modes exist, and quietly storing "manual"
// would hide that.
func TestSaveAuthoringModeRejectsUnknownMode(t *testing.T) {
	store := newFakeAuthoringStore()
	save := application.NewSaveAuthoringModeUseCase(store)

	if err := save.Execute(context.Background(), "p1", "sometimes"); err == nil {
		t.Fatal("want an error for an unknown mode")
	}
	if _, ok := store.mode["p1"]; ok {
		t.Error("nothing should have been stored")
	}
	if err := save.Execute(context.Background(), "", "ai"); err == nil {
		t.Fatal("want an error for a missing project_id")
	}
}
