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
	story      map[string]string
	storyboard map[string]string
	code       map[string]string
	review     map[string]string
}

func newFakeAuthoringStore() *fakeAuthoringStore {
	return &fakeAuthoringStore{
		story:      map[string]string{},
		storyboard: map[string]string{},
		code:       map[string]string{},
		review:     map[string]string{},
	}
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

func TestSaveAuthoringCodeUseCase(t *testing.T) {
	store := newFakeAuthoringStore()
	uc := application.NewSaveAuthoringCodeUseCase(store)

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
	uc := application.NewSaveAuthoringReviewUseCase(store)

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
