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
// topic collisions.
type fakeDraftRepo struct {
	saved   map[string]*domain.Project
	topics  map[string]string
	saveErr error
}

func newFakeDraftRepo() *fakeDraftRepo {
	return &fakeDraftRepo{
		saved:  map[string]*domain.Project{},
		topics: map[string]string{},
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

func (f *fakeDraftRepo) Get(_ context.Context, projectID string) (*domain.Project, error) {
	p, ok := f.saved[projectID]
	if !ok {
		return nil, domain.ErrProjectNotFound
	}
	copied := *p
	return &copied, nil
}

// TestCreateProjectDraft_NewProjectIsRemotion: every new draft runs on the one
// engine there is.
func TestCreateProjectDraft_NewProjectIsRemotion(t *testing.T) {
	repo := newFakeDraftRepo()
	uc := application.NewCreateProjectDraftUseCase(repo)

	out, err := uc.Execute(context.Background(), application.CreateProjectDraftInput{
		ProjectID: "p1", ContentLanguage: domain.LanguageVietnamese,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := repo.saved[out.ProjectID].RenderEngine; got != domain.RenderEngineRemotion {
		t.Fatalf("a new draft must be a Remotion project, got %q", got)
	}
}

func TestCreateProjectDraft_AShortOfALongVideoStartsAsALinkedShort(t *testing.T) {
	repo := newFakeDraftRepo()
	repo.saved["long-1"] = &domain.Project{ProjectID: "long-1", Status: domain.StatusReadyToPublish, VideoOutputMode: domain.ModeLongOnly}
	uc := application.NewCreateProjectDraftUseCase(repo)
	out, err := uc.Execute(context.Background(), application.CreateProjectDraftInput{
		ProjectID: "short-1", Topic: "Vì sao gọi là tiểu đường", ContentLanguage: domain.LanguageVietnamese, ShortOf: "long-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	short := repo.saved[out.ProjectID]
	if short.VideoOutputMode != domain.ModeShortOnly || short.VideoFormatID != domain.FormatVerticalShort60s.ID ||
		short.RenderEngine != domain.RenderEngineRemotion || short.SubtitleMode != domain.SubtitleModeBurnIn ||
		short.SubtitleStyle == nil || short.SubtitleStyle.Position != "top" {
		t.Errorf("short draft = mode %s format %s engine %s subtitles %s style %+v",
			short.VideoOutputMode, short.VideoFormatID, short.RenderEngine, short.SubtitleMode, short.SubtitleStyle)
	}
	if short.CompanionProjectID == nil || *short.CompanionProjectID != "long-1" {
		t.Error("the short must point at its long video")
	}
	if long := repo.saved["long-1"]; long.CompanionProjectID == nil || *long.CompanionProjectID != "short-1" {
		t.Error("the long video must point back at its short")
	}
}

func TestCreateProjectDraft_AShortOfAnUnknownVideoIsRefused(t *testing.T) {
	uc := application.NewCreateProjectDraftUseCase(newFakeDraftRepo())
	_, err := uc.Execute(context.Background(), application.CreateProjectDraftInput{
		ProjectID: "short-1", Topic: "x", ContentLanguage: domain.LanguageVietnamese, ShortOf: "missing",
	})
	if err == nil {
		t.Fatal("a short of a project that does not exist must be refused")
	}
}
