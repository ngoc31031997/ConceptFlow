package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"orchestrator/internal/domain"
)

// SimilarProject is one match CR-028 FR85 surfaces back to the Creator when
// a topic collides (after normalization) with an existing project's saved
// topic, in the same content_language.
type SimilarProject struct {
	ProjectID string
	Topic     string
	Status    domain.ProjectStatus
	CreatedAt time.Time
}

// ProjectDraftPort is the persistence capability CR-028's early-draft use
// cases need. Save reuses domain.ProjectRepositoryPort's existing upsert
// (INSERT ... ON CONFLICT DO UPDATE) — the same Save call StartRenderSaga
// already makes at render time now simply updates the row this created
// instead of inserting a fresh one.
type ProjectDraftPort interface {
	Save(ctx context.Context, project *domain.Project) error
	GetStatus(ctx context.Context, projectID string) (domain.ProjectStatus, error)
	// GetStatusAndLanguage is GetStatus plus the project's content_language,
	// which FR83.2's re-check of FR85's collision list needs (comparison is
	// scoped to "same language" — see NormalizeTopic's doc comment).
	GetStatusAndLanguage(ctx context.Context, projectID string) (domain.ProjectStatus, domain.ContentLanguage, error)
	SaveAuthoringTopic(ctx context.Context, projectID, topic string) error
	FindSimilarTopics(ctx context.Context, language domain.ContentLanguage, normalizedTopic, excludeProjectID string) ([]SimilarProject, error)
	// SaveRenderEngine persists CR-030's engine choice as soon as the
	// Creator makes it, instead of only at render-submit time — the
	// server-side authoring chain (RenderPromptUseCase.RoleFor) reads
	// project.RenderEngine to pick storyboard/code prompts, so a chain run
	// from tab 1a would otherwise always see the "manim" default even when
	// the Creator picked Remotion at "/".
	SaveRenderEngine(ctx context.Context, projectID string, engine domain.RenderEngine) error
}

// CreateProjectDraftInput is the parsed body of POST /v1/projects.
type CreateProjectDraftInput struct {
	// ProjectID is optional: when empty, the server generates a fresh one
	// (CR-028 FR83.1's default). When set, it is the id web-gui's
	// ProjectDraftContext already generated client-side for this draft
	// (unchanged from before this CR) — accepting it here avoids a much
	// larger refactor of every page that already reads draft.projectId
	// synchronously, while still creating the projects row at step 1
	// instead of at render time.
	ProjectID       string
	Topic           string
	ContentLanguage domain.ContentLanguage
	// RenderEngine is optional (CR-030): "" means the caller is not
	// declaring an engine this call (e.g. a topic-only debounce save) —
	// leave whatever is already on the row untouched, same "only touch what
	// was sent" rule the topic field already follows.
	RenderEngine domain.RenderEngine
}

// CreateProjectDraftOutput is returned to the HTTP layer for the 201
// response.
type CreateProjectDraftOutput struct {
	ProjectID       string
	SimilarProjects []SimilarProject
}

// CreateProjectDraftUseCase backs CR-028 FR83.1: a Project row (status=draft)
// is created as soon as the Creator finishes typing a topic (wizard step 1),
// not at POST /v1/sagas/render. Everything the authoring wizard writes from
// then on (project_authoring) has a real project_id to hang off of, so a
// Creator who abandons the wizard leaves a listable/deletable draft instead
// of an invisible orphan (the risk docs/review/data-flow-review.md flagged).
type CreateProjectDraftUseCase struct {
	repo ProjectDraftPort
}

func NewCreateProjectDraftUseCase(repo ProjectDraftPort) *CreateProjectDraftUseCase {
	return &CreateProjectDraftUseCase{repo: repo}
}

func (uc *CreateProjectDraftUseCase) Execute(ctx context.Context, input CreateProjectDraftInput) (*CreateProjectDraftOutput, error) {
	if input.ContentLanguage != domain.LanguageVietnamese && input.ContentLanguage != domain.LanguageEnglish {
		return nil, fmt.Errorf("content_language must be 'vi' or 'en'")
	}

	projectID := input.ProjectID
	if projectID == "" {
		projectID = newUUID()
	}

	// Idempotency guard: a Creator who edits the topic again on step 1
	// re-sends this same POST (the client does not distinguish "first
	// save" from "edit" — see UpdateProjectTopicUseCase's doc comment for
	// why PATCH exists anyway for callers that do distinguish). If the
	// project already exists, Save()'s full-column upsert would silently
	// reset every field a later wizard step already set (settings, TTS,
	// intro/outro, ...) back to these bare defaults — so skip it entirely
	// once the row is there, and just refresh the topic.
	_, err := uc.repo.GetStatus(ctx, projectID)
	projectExists := err == nil
	if err != nil && err != domain.ErrProjectNotFound {
		return nil, err
	}
	if !projectExists {
		project := &domain.Project{
			ProjectID:       projectID,
			Status:          domain.StatusDraft,
			ContentLanguage: input.ContentLanguage,
			VideoFormatID:   domain.DefaultVideoFormatID,
			ReviewEnabled:   true,
			TTSEnabled:      true,
			RenderQuality:   domain.DefaultRenderQuality,
			RenderEngine:    domain.DefaultRenderEngine,
			VideoOutputMode: domain.DefaultVideoOutputMode,
			IntroEnabled:    true,
			OutroEnabled:    true,
		}
		if err := uc.repo.Save(ctx, project); err != nil {
			return nil, err
		}
	}

	if input.RenderEngine != "" {
		if !input.RenderEngine.IsValid() {
			return nil, fmt.Errorf("render_engine must be %q or %q", domain.RenderEngineManim, domain.RenderEngineRemotion)
		}
		if err := uc.repo.SaveRenderEngine(ctx, projectID, input.RenderEngine); err != nil {
			return nil, err
		}
	}

	topic := strings.TrimSpace(input.Topic)
	var similar []SimilarProject
	if topic != "" {
		if err := uc.repo.SaveAuthoringTopic(ctx, projectID, topic); err != nil {
			return nil, err
		}
		var err error
		similar, err = uc.repo.FindSimilarTopics(ctx, input.ContentLanguage, domain.NormalizeTopic(topic), projectID)
		if err != nil {
			return nil, err
		}
	}

	return &CreateProjectDraftOutput{ProjectID: projectID, SimilarProjects: similar}, nil
}

// UpdateProjectTopicInput is the parsed body of PATCH /v1/projects/{id}/topic.
type UpdateProjectTopicInput struct {
	ProjectID string
	Topic     string
}

// UpdateProjectTopicOutput mirrors CreateProjectDraftOutput's collision list
// so the GUI shows the same banner whether the topic was just set (FR83.1)
// or edited afterward (FR83.2).
type UpdateProjectTopicOutput struct {
	SimilarProjects []SimilarProject
}

// UpdateProjectTopicUseCase backs CR-028 FR83.2: Creator returns to step 1
// and edits the topic of a draft they already created. Locked the same way
// authoring saves are (FR84.2) — once render has started, the topic that
// produced the rendered script should not silently change under it.
type UpdateProjectTopicUseCase struct {
	repo ProjectDraftPort
}

func NewUpdateProjectTopicUseCase(repo ProjectDraftPort) *UpdateProjectTopicUseCase {
	return &UpdateProjectTopicUseCase{repo: repo}
}

func (uc *UpdateProjectTopicUseCase) Execute(ctx context.Context, input UpdateProjectTopicInput) (*UpdateProjectTopicOutput, error) {
	if input.ProjectID == "" {
		return nil, fmt.Errorf("project_id is required")
	}
	topic := strings.TrimSpace(input.Topic)
	if topic == "" {
		return nil, fmt.Errorf("topic is required")
	}

	status, language, err := uc.repo.GetStatusAndLanguage(ctx, input.ProjectID)
	if err != nil {
		return nil, err
	}
	if !domain.IsAuthoringEditable(status) {
		return nil, domain.ErrInvalidStatus
	}

	if err := uc.repo.SaveAuthoringTopic(ctx, input.ProjectID, topic); err != nil {
		return nil, err
	}
	similar, err := uc.repo.FindSimilarTopics(ctx, language, domain.NormalizeTopic(topic), input.ProjectID)
	if err != nil {
		return nil, err
	}
	return &UpdateProjectTopicOutput{SimilarProjects: similar}, nil
}
