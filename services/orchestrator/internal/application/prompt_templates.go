package application

import (
	"context"
	"fmt"

	"orchestrator/internal/domain"
)

// PromptTemplatePort is the persistence capability CR-025's prompt-template
// use case needs — narrower than the full postgres.PromptTemplateRepository
// so tests can fake just this.
type PromptTemplatePort interface {
	Get(ctx context.Context, role domain.PromptRole, language string) (domain.PromptTemplate, error)
	List(ctx context.Context) ([]domain.PromptTemplate, error)
	Update(ctx context.Context, role domain.PromptRole, language, templateText string) (domain.PromptTemplate, error)
}

// PromptTemplatesUseCase backs CR-025's prompt-template CRUD endpoints — the
// admin screen edits wording here instead of in web-gui source, and web-gui's
// wizard reads the current wording at runtime instead of a hardcoded string.
type PromptTemplatesUseCase struct {
	templates PromptTemplatePort
}

func NewPromptTemplatesUseCase(templates PromptTemplatePort) *PromptTemplatesUseCase {
	return &PromptTemplatesUseCase{templates: templates}
}

func (uc *PromptTemplatesUseCase) Get(ctx context.Context, role domain.PromptRole, language string) (domain.PromptTemplate, error) {
	return uc.templates.Get(ctx, role, language)
}

func (uc *PromptTemplatesUseCase) List(ctx context.Context) ([]domain.PromptTemplate, error) {
	return uc.templates.List(ctx)
}

func (uc *PromptTemplatesUseCase) Update(ctx context.Context, role domain.PromptRole, language, templateText string) (domain.PromptTemplate, error) {
	if !domain.ValidPromptRole(string(role)) {
		return domain.PromptTemplate{}, fmt.Errorf("invalid role %q", role)
	}
	if language != "vi" && language != "en" {
		return domain.PromptTemplate{}, fmt.Errorf("language must be 'vi' or 'en'")
	}
	if templateText == "" {
		return domain.PromptTemplate{}, fmt.Errorf("template_text is required")
	}
	return uc.templates.Update(ctx, role, language, templateText)
}

// Reset overwrites a template with the default shipped in this binary.
//
// It goes through Update rather than a dedicated repository call, so a reset
// bumps version and updated_at exactly like a manual save does — from the
// audit trail's point of view a reset IS an edit, one that happens to paste
// the shipped text. This is the deliberate counterpart to seeding staying
// insert-if-absent: shipped wording reaches a running database only when
// someone asks for it here.
func (uc *PromptTemplatesUseCase) Reset(ctx context.Context, role domain.PromptRole, language string) (domain.PromptTemplate, error) {
	if !domain.ValidPromptRole(string(role)) {
		return domain.PromptTemplate{}, fmt.Errorf("invalid role %q", role)
	}
	def, ok := domain.DefaultPromptTemplate(role, language)
	if !ok {
		return domain.PromptTemplate{}, fmt.Errorf("no built-in default for role %q language %q", role, language)
	}
	return uc.templates.Update(ctx, role, language, def.TemplateText)
}

// AuthoringStoryPort persists CR-025 step 1's pasted story outline, and from
// CR-027 D0 the topic it was written from.
type AuthoringStoryPort interface {
	SaveAuthoringStory(ctx context.Context, projectID, content, topic string) error
	GetAuthoringStory(ctx context.Context, projectID string) (string, error)
}

// SaveAuthoringStoryUseCase stores the Story Architect output a Creator
// pasted back after the external-AI round trip (CR-025 step 1). It performs
// no saga/state-machine transition by itself — same "just persist intent"
// posture as the CR-007 clip-selection endpoint — later pipeline steps read
// it back via {{previous_output}}.
type SaveAuthoringStoryUseCase struct {
	authoring AuthoringStoryPort
}

func NewSaveAuthoringStoryUseCase(authoring AuthoringStoryPort) *SaveAuthoringStoryUseCase {
	return &SaveAuthoringStoryUseCase{authoring: authoring}
}

// Execute saves the outline and, when one is supplied, the topic. topic is
// deliberately NOT required: the outline is what this step exists to store,
// and refusing to save it because a topic is missing would break every
// pre-CR-027 caller for a field they do not know about.
func (uc *SaveAuthoringStoryUseCase) Execute(ctx context.Context, projectID, content, topic string) error {
	if projectID == "" {
		return fmt.Errorf("project_id is required")
	}
	if content == "" {
		return fmt.Errorf("content is required")
	}
	return uc.authoring.SaveAuthoringStory(ctx, projectID, content, topic)
}

// AuthoringStoryboardPort persists CR-025 step 2's pasted storyboard.
type AuthoringStoryboardPort interface {
	SaveAuthoringStoryboard(ctx context.Context, projectID, content string) error
	GetAuthoringStoryboard(ctx context.Context, projectID string) (string, error)
}

// SaveAuthoringStoryboardUseCase stores the Visual Director output a Creator
// pasted back after the external-AI round trip (CR-025 step 2) — same
// "just persist intent" posture as SaveAuthoringStoryUseCase.
type SaveAuthoringStoryboardUseCase struct {
	authoring AuthoringStoryboardPort
}

func NewSaveAuthoringStoryboardUseCase(authoring AuthoringStoryboardPort) *SaveAuthoringStoryboardUseCase {
	return &SaveAuthoringStoryboardUseCase{authoring: authoring}
}

func (uc *SaveAuthoringStoryboardUseCase) Execute(ctx context.Context, projectID, content string) error {
	if projectID == "" {
		return fmt.Errorf("project_id is required")
	}
	if content == "" {
		return fmt.Errorf("content is required")
	}
	return uc.authoring.SaveAuthoringStoryboard(ctx, projectID, content)
}

// AuthoringCodePort persists CR-025 step 3's pasted Manim code.
type AuthoringCodePort interface {
	SaveAuthoringCode(ctx context.Context, projectID, content string) error
	GetAuthoringCode(ctx context.Context, projectID string) (string, error)
}

// SaveAuthoringCodeUseCase stores the Manim Engineer output a Creator pasted
// back after the external-AI round trip (CR-025 step 3) — same "just persist
// intent" posture as SaveAuthoringStoryUseCase/SaveAuthoringStoryboardUseCase.
type SaveAuthoringCodeUseCase struct {
	authoring AuthoringCodePort
}

func NewSaveAuthoringCodeUseCase(authoring AuthoringCodePort) *SaveAuthoringCodeUseCase {
	return &SaveAuthoringCodeUseCase{authoring: authoring}
}

func (uc *SaveAuthoringCodeUseCase) Execute(ctx context.Context, projectID, content string) error {
	if projectID == "" {
		return fmt.Errorf("project_id is required")
	}
	if content == "" {
		return fmt.Errorf("content is required")
	}
	return uc.authoring.SaveAuthoringCode(ctx, projectID, content)
}

// AuthoringReviewPort persists CR-025 step 4's pasted reviewer verdict.
type AuthoringReviewPort interface {
	SaveAuthoringReview(ctx context.Context, projectID, content string) error
	GetAuthoringReview(ctx context.Context, projectID string) (string, error)
}

// SaveAuthoringReviewUseCase stores the Script Reviewer verdict a Creator
// pasted back after the external-AI round trip (CR-025 step 4) — same
// "just persist intent" posture as the other three authoring save use cases.
type SaveAuthoringReviewUseCase struct {
	authoring AuthoringReviewPort
}

func NewSaveAuthoringReviewUseCase(authoring AuthoringReviewPort) *SaveAuthoringReviewUseCase {
	return &SaveAuthoringReviewUseCase{authoring: authoring}
}

func (uc *SaveAuthoringReviewUseCase) Execute(ctx context.Context, projectID, content string) error {
	if projectID == "" {
		return fmt.Errorf("project_id is required")
	}
	if content == "" {
		return fmt.Errorf("content is required")
	}
	return uc.authoring.SaveAuthoringReview(ctx, projectID, content)
}

// AuthoringStateReaderPort is the read side all four authoring outputs
// share — used by GET /v1/projects/{id}/authoring so the wizard can
// rehydrate on reload/back-navigation instead of relying solely on
// client-side draft state.
type AuthoringStateReaderPort interface {
	GetAuthoringTopic(ctx context.Context, projectID string) (string, error)
	GetAuthoringStory(ctx context.Context, projectID string) (string, error)
	GetAuthoringStoryboard(ctx context.Context, projectID string) (string, error)
	GetAuthoringCode(ctx context.Context, projectID string) (string, error)
	GetAuthoringReview(ctx context.Context, projectID string) (string, error)
}

// AuthoringState is what GET /v1/projects/{id}/authoring returns — every
// pipeline output saved so far, empty string when a step has not been saved.
type AuthoringState struct {
	Topic      string
	Story      string
	Storyboard string
	Code       string
	Review     string
}

// GetAuthoringStateUseCase backs the read side of CR-025's authoring pipeline.
type GetAuthoringStateUseCase struct {
	authoring AuthoringStateReaderPort
}

func NewGetAuthoringStateUseCase(authoring AuthoringStateReaderPort) *GetAuthoringStateUseCase {
	return &GetAuthoringStateUseCase{authoring: authoring}
}

func (uc *GetAuthoringStateUseCase) Execute(ctx context.Context, projectID string) (AuthoringState, error) {
	topic, err := uc.authoring.GetAuthoringTopic(ctx, projectID)
	if err != nil {
		return AuthoringState{}, err
	}
	story, err := uc.authoring.GetAuthoringStory(ctx, projectID)
	if err != nil {
		return AuthoringState{}, err
	}
	storyboard, err := uc.authoring.GetAuthoringStoryboard(ctx, projectID)
	if err != nil {
		return AuthoringState{}, err
	}
	code, err := uc.authoring.GetAuthoringCode(ctx, projectID)
	if err != nil {
		return AuthoringState{}, err
	}
	review, err := uc.authoring.GetAuthoringReview(ctx, projectID)
	if err != nil {
		return AuthoringState{}, err
	}
	return AuthoringState{Topic: topic, Story: story, Storyboard: storyboard, Code: code, Review: review}, nil
}
