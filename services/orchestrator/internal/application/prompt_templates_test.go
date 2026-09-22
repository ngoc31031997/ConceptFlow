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
	review     map[string]string
	history    map[string][]string
	// status defaults to domain.StatusDraft (Go zero value is "", so
	// GetStatus below maps "" to draft) — set per project_id to simulate a
	// project whose render has already started (CR-028 FR84.2).
	status map[string]domain.ProjectStatus
	// mode backs CR-027 FR79's step-1 working mode. Unset means a project
	// whose row predates the column, which reads back as the default.
	mode map[string]string
}

func newFakeAuthoringStore() *fakeAuthoringStore {
	return &fakeAuthoringStore{
		topic:      map[string]string{},
		story:      map[string]string{},
		storyboard: map[string]string{},
		code:       map[string]string{},
		review:     map[string]string{},
		history:    map[string][]string{},
		status:     map[string]domain.ProjectStatus{},
		mode:       map[string]string{},
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

// GetStatus backs CR-028 FR84.2's authoring lock. Defaults to draft (unset
// entries) so every pre-existing test above, which never touches status,
// keeps passing unmodified.
func (f *fakeAuthoringStore) GetStatus(_ context.Context, projectID string) (domain.ProjectStatus, error) {
	if s, ok := f.status[projectID]; ok {
		return s, nil
	}
	return domain.StatusDraft, nil
}

// SaveAuthoringHistory backs CR-028 FR84.3.
func (f *fakeAuthoringStore) SaveAuthoringHistory(_ context.Context, projectID, fieldName, content string) error {
	key := projectID + ":" + fieldName
	f.history[key] = append(f.history[key], content)
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

func (f *fakeAuthoringStore) SaveAuthoringReview(_ context.Context, projectID, content string) error {
	f.review[projectID] = content
	return nil
}

func (f *fakeAuthoringStore) GetAuthoringReview(_ context.Context, projectID string) (string, error) {
	return f.review[projectID], nil
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

func TestSaveAuthoringReviewUseCase(t *testing.T) {
	store := newFakeAuthoringStore()
	uc := application.NewSaveAuthoringReviewUseCase(store, store, store)

	if err := uc.Execute(context.Background(), "", "some review"); err == nil {
		t.Fatal("expected error for empty project_id")
	}
	if err := uc.Execute(context.Background(), "p1", ""); err == nil {
		t.Fatal("expected error for empty content")
	}
	if err := uc.Execute(context.Background(), "p1", "the review"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := store.review["p1"]; got != "the review" {
		t.Fatalf("review not saved, got %q", got)
	}
}

func TestGetAuthoringStateUseCase(t *testing.T) {
	store := newFakeAuthoringStore()
	store.story["p1"] = "the story"
	store.storyboard["p1"] = "the storyboard"
	store.code["p1"] = "the code"
	store.review["p1"] = "the review"

	uc := application.NewGetAuthoringStateUseCase(store)
	state, err := uc.Execute(context.Background(), "p1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Story != "the story" || state.Storyboard != "the storyboard" || state.Code != "the code" || state.Review != "the review" {
		t.Fatalf("unexpected state: %+v", state)
	}

	// A project that never saved anything answers empty strings, not an error.
	empty, err := uc.Execute(context.Background(), "unknown")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if empty.Story != "" || empty.Storyboard != "" || empty.Code != "" || empty.Review != "" {
		t.Fatalf("expected empty state, got %+v", empty)
	}
}

// fakePromptStore backs the Reset tests — Reset's whole job is to look up the
// shipped default and hand it to Update, so an in-memory Update is enough to
// see what it decided to write.
type fakePromptStore struct {
	rows map[string]domain.PromptTemplate
}

func newFakePromptStore() *fakePromptStore {
	return &fakePromptStore{rows: map[string]domain.PromptTemplate{}}
}

func (f *fakePromptStore) key(role domain.PromptRole, language string) string {
	return string(role) + "/" + language
}

func (f *fakePromptStore) Get(_ context.Context, role domain.PromptRole, language string) (domain.PromptTemplate, error) {
	return f.rows[f.key(role, language)], nil
}

func (f *fakePromptStore) List(_ context.Context) ([]domain.PromptTemplate, error) {
	out := make([]domain.PromptTemplate, 0, len(f.rows))
	for _, t := range f.rows {
		out = append(out, t)
	}
	return out, nil
}

func (f *fakePromptStore) Update(_ context.Context, role domain.PromptRole, language, templateText string) (domain.PromptTemplate, error) {
	k := f.key(role, language)
	row := f.rows[k]
	row.Role, row.Language, row.TemplateText = role, language, templateText
	row.Version++
	f.rows[k] = row
	return row, nil
}

func TestResetPromptTemplateRestoresShippedWording(t *testing.T) {
	store := newFakePromptStore()
	// An editor has drifted this row away from the shipped default — the exact
	// situation Reset exists for, since seeding is insert-if-absent and would
	// never touch it.
	if _, err := store.Update(context.Background(), domain.RoleStoryArchitect, "vi", "prompt tự sửa"); err != nil {
		t.Fatalf("seeding the fake failed: %v", err)
	}

	uc := application.NewPromptTemplatesUseCase(store)
	restored, err := uc.Reset(context.Background(), domain.RoleStoryArchitect, "vi")
	if err != nil {
		t.Fatalf("Reset: %v", err)
	}

	want, ok := domain.DefaultPromptTemplate(domain.RoleStoryArchitect, "vi")
	if !ok {
		t.Fatal("no shipped default for story_architect/vi")
	}
	if restored.TemplateText != want.TemplateText {
		t.Error("Reset did not restore the shipped wording")
	}
	// A reset counts as an edit: version moves, so the admin screen shows that
	// something happened rather than appearing stuck.
	if restored.Version != 2 {
		t.Errorf("version = %d, want 2 (the editor's save, then the reset)", restored.Version)
	}
}

func TestResetPromptTemplateRejectsUnknownRoleAndLanguage(t *testing.T) {
	uc := application.NewPromptTemplatesUseCase(newFakePromptStore())

	if _, err := uc.Reset(context.Background(), domain.PromptRole("nonsense"), "vi"); err == nil {
		t.Error("expected an error for an unknown role")
	}
	// A valid role with no shipped default for that language must fail loudly
	// rather than overwriting the row with an empty string.
	if _, err := uc.Reset(context.Background(), domain.RoleStoryArchitect, "fr"); err == nil {
		t.Error("expected an error for a language with no shipped default")
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
