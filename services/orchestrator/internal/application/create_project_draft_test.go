package application_test

import (
	"context"
	"testing"

	"orchestrator/internal/application"
	"orchestrator/internal/domain"
)

// fakeDraftRepo is a minimal in-memory application.ProjectDraftPort, just
// enough for CreateProjectDraftUseCase's own logic — it does not model
// FindSimilarTopics collisions beyond "none" since no test here exercises
// FR85.
type fakeDraftRepo struct {
	saved        map[string]*domain.Project
	topics       map[string]string
	renderEngine map[string]domain.RenderEngine
	saveErr      error
}

func newFakeDraftRepo() *fakeDraftRepo {
	return &fakeDraftRepo{
		saved:        map[string]*domain.Project{},
		topics:       map[string]string{},
		renderEngine: map[string]domain.RenderEngine{},
	}
}

func (f *fakeDraftRepo) Save(_ context.Context, project *domain.Project) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	cp := *project
	f.saved[project.ProjectID] = &cp
	return nil
}

func (f *fakeDraftRepo) GetStatus(_ context.Context, projectID string) (domain.ProjectStatus, error) {
	p, ok := f.saved[projectID]
	if !ok {
		return "", domain.ErrProjectNotFound
	}
	return p.Status, nil
}

func (f *fakeDraftRepo) GetStatusAndLanguage(_ context.Context, projectID string) (domain.ProjectStatus, domain.ContentLanguage, error) {
	p, ok := f.saved[projectID]
	if !ok {
		return "", "", domain.ErrProjectNotFound
	}
	return p.Status, p.ContentLanguage, nil
}

func (f *fakeDraftRepo) SaveAuthoringTopic(_ context.Context, projectID, topic string, _ domain.ContentLanguage) error {
	f.topics[projectID] = topic
	return nil
}

func (f *fakeDraftRepo) FindSimilarTopics(_ context.Context, _ domain.ContentLanguage, _ string, _ string) ([]application.SimilarProject, error) {
	return nil, nil
}

func (f *fakeDraftRepo) SaveRenderEngine(_ context.Context, projectID string, engine domain.RenderEngine) error {
	f.renderEngine[projectID] = engine
	return nil
}

// TestCreateProjectDraft_NewProjectDefaultsEngineButRenderEngineOverrides —
// CR-030: a fresh row starts on DefaultRenderEngine like before, but a
// caller that already knows the Creator's choice (tab 1a's chain, or "/"
// picking Remotion) gets it persisted in the same call, not only at
// render-submit time.
func TestCreateProjectDraft_NewProjectDefaultsEngineButRenderEngineOverrides(t *testing.T) {
	repo := newFakeDraftRepo()
	uc := application.NewCreateProjectDraftUseCase(repo)

	out, err := uc.Execute(context.Background(), application.CreateProjectDraftInput{
		ProjectID:       "p1",
		ContentLanguage: domain.LanguageVietnamese,
		RenderEngine:    domain.RenderEngineRemotion,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.renderEngine[out.ProjectID] != domain.RenderEngineRemotion {
		t.Fatalf("render engine not saved, got %q", repo.renderEngine[out.ProjectID])
	}
}

// TestCreateProjectDraft_EmptyRenderEngineLeavesItUntouched — the debounced
// topic-only save (ScriptOutlineStepPage's useEffect) must not silently
// reset an engine the Creator already picked back to the Manim default.
func TestCreateProjectDraft_EmptyRenderEngineLeavesItUntouched(t *testing.T) {
	repo := newFakeDraftRepo()
	uc := application.NewCreateProjectDraftUseCase(repo)

	if _, err := uc.Execute(context.Background(), application.CreateProjectDraftInput{
		ProjectID: "p1", ContentLanguage: domain.LanguageVietnamese, RenderEngine: domain.RenderEngineRemotion,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := uc.Execute(context.Background(), application.CreateProjectDraftInput{
		ProjectID: "p1", Topic: "chủ đề mới", ContentLanguage: domain.LanguageVietnamese,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.renderEngine["p1"] != domain.RenderEngineRemotion {
		t.Fatalf("a later call without render_engine must not touch it, got %q", repo.renderEngine["p1"])
	}
}

// TestCreateProjectDraft_InvalidRenderEngineIsRejected — a typo or a stale
// client must not silently fall back to a default; it should fail loudly the
// same way an invalid content_language already does.
func TestCreateProjectDraft_InvalidRenderEngineIsRejected(t *testing.T) {
	repo := newFakeDraftRepo()
	uc := application.NewCreateProjectDraftUseCase(repo)

	if _, err := uc.Execute(context.Background(), application.CreateProjectDraftInput{
		ProjectID: "p1", ContentLanguage: domain.LanguageVietnamese, RenderEngine: "manin-typo",
	}); err == nil {
		t.Fatal("expected an error for an invalid render_engine")
	}
	if _, ok := repo.renderEngine["p1"]; ok {
		t.Fatal("an invalid render_engine must not be saved")
	}
}
