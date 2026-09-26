package application

import (
	"context"
	"fmt"
	"time"

	"authoring/internal/domain"
)

// AuthoringStoryPort persists CR-025 step 1's pasted story outline, and from
// CR-027 D0 the topic it was written from.
type AuthoringStoryPort interface {
	SaveAuthoringStory(ctx context.Context, projectID, content, topic string) error
	GetAuthoringStory(ctx context.Context, projectID string) (string, error)
}

// AuthoringLockPort is the CR-028 FR84.2 precondition every authoring save
// shares: writes are unrestricted while the project is still status=draft,
// and refused once render has started.
type AuthoringLockPort interface {
	GetStatus(ctx context.Context, projectID string) (domain.ProjectStatus, error)
}

// AuthoringStep names one of the three chained authoring outputs.
type AuthoringStep string

const (
	AuthoringStepStory      AuthoringStep = "story"
	AuthoringStepStoryboard AuthoringStep = "storyboard"
	AuthoringStepCode       AuthoringStep = "code"
)

// AuthoringClearerPort empties the saved output of downstream steps. Each
// step is written from the one before it (story → storyboard → code), so when
// an upstream output changes, what was built on the old one is stale and is
// dropped rather than left to disagree with it. No history is kept.
type AuthoringClearerPort interface {
	ClearAuthoringSteps(ctx context.Context, projectID string, steps ...AuthoringStep) error
}

// checkAuthoringUnlocked is the FR84.2 precondition shared by all four
// SaveAuthoring*UseCase.Execute methods: a project that does not exist yet
// cannot have its authoring edited (CR-028 requires POST /v1/projects
// first), and one that has already started rendering is locked so the
// script that got rendered cannot silently change under it.
func checkAuthoringUnlocked(ctx context.Context, locks AuthoringLockPort, projectID string) error {
	status, err := locks.GetStatus(ctx, projectID)
	if err != nil {
		return err
	}
	// A project that failed at any step is editable too: its way forward is to
	// fix the inputs (prompt, script, settings) and re-run from any step.
	if domain.IsAuthoringEditable(status) {
		return nil
	}
	return domain.ErrInvalidStatus
}

// SaveAuthoringStoryUseCase stores the Story Architect output a Creator
// pasted back after the external-AI round trip (CR-025 step 1). It performs
// no saga/state-machine transition by itself — same "just persist intent"
// posture as the CR-007 clip-selection endpoint — later pipeline steps read
// it back via {{previous_output}}.
type SaveAuthoringStoryUseCase struct {
	authoring AuthoringStoryPort
	locks     AuthoringLockPort
	clearer   AuthoringClearerPort
}

func NewSaveAuthoringStoryUseCase(authoring AuthoringStoryPort, locks AuthoringLockPort, clearer AuthoringClearerPort) *SaveAuthoringStoryUseCase {
	return &SaveAuthoringStoryUseCase{authoring: authoring, locks: locks, clearer: clearer}
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
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return err
	}
	previous, err := uc.authoring.GetAuthoringStory(ctx, projectID)
	if err != nil {
		return err
	}
	if err := uc.authoring.SaveAuthoringStory(ctx, projectID, content, topic); err != nil {
		return err
	}
	if previous == content {
		return nil
	}
	return uc.clearer.ClearAuthoringSteps(ctx, projectID, AuthoringStepStoryboard, AuthoringStepCode)
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
	locks     AuthoringLockPort
	clearer   AuthoringClearerPort
}

func NewSaveAuthoringStoryboardUseCase(authoring AuthoringStoryboardPort, locks AuthoringLockPort, clearer AuthoringClearerPort) *SaveAuthoringStoryboardUseCase {
	return &SaveAuthoringStoryboardUseCase{authoring: authoring, locks: locks, clearer: clearer}
}

func (uc *SaveAuthoringStoryboardUseCase) Execute(ctx context.Context, projectID, content string) error {
	if projectID == "" {
		return fmt.Errorf("project_id is required")
	}
	if content == "" {
		return fmt.Errorf("content is required")
	}
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return err
	}
	previous, err := uc.authoring.GetAuthoringStoryboard(ctx, projectID)
	if err != nil {
		return err
	}
	if err := uc.authoring.SaveAuthoringStoryboard(ctx, projectID, content); err != nil {
		return err
	}
	if previous == content {
		return nil
	}
	return uc.clearer.ClearAuthoringSteps(ctx, projectID, AuthoringStepCode)
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
	locks     AuthoringLockPort
	clearer   AuthoringClearerPort
}

func NewSaveAuthoringCodeUseCase(authoring AuthoringCodePort, locks AuthoringLockPort, clearer AuthoringClearerPort) *SaveAuthoringCodeUseCase {
	return &SaveAuthoringCodeUseCase{authoring: authoring, locks: locks, clearer: clearer}
}

func (uc *SaveAuthoringCodeUseCase) Execute(ctx context.Context, projectID, content string) error {
	if projectID == "" {
		return fmt.Errorf("project_id is required")
	}
	if content == "" {
		return fmt.Errorf("content is required")
	}
	if err := checkAuthoringUnlocked(ctx, uc.locks, projectID); err != nil {
		return err
	}
	if err := uc.authoring.SaveAuthoringCode(ctx, projectID, content); err != nil {
		return err
	}
	return nil
}

// AuthoringStateReaderPort is the read side all four authoring outputs
// share — used by GET /v1/projects/{id}/authoring so the wizard can
// rehydrate on reload/back-navigation instead of relying solely on
// client-side draft state.
type AuthoringStateReaderPort interface {
	GetAuthoringTopic(ctx context.Context, projectID string) (string, error)
	// GetAuthoringMode returns "" for a project saved before CR-027 FR79, or
	// one whose Creator never touched the choice. Execute turns that into the
	// default rather than leaking an empty third value to the GUI.
	GetAuthoringMode(ctx context.Context, projectID string) (string, error)
	GetAuthoringStory(ctx context.Context, projectID string) (string, error)
	GetAuthoringStoryboard(ctx context.Context, projectID string) (string, error)
	GetAuthoringCode(ctx context.Context, projectID string) (string, error)
	// GetAuthoringModels returns the per-step Hive model choice, zero value
	// ("" for every step, meaning "server default") for a project saved
	// before this picker existed.
	GetAuthoringModels(ctx context.Context, projectID string) (domain.AuthoringStepModels, error)
}

// AuthoringState is what GET /v1/projects/{id}/authoring returns — every
// pipeline output saved so far, empty string when a step has not been saved.
type AuthoringState struct {
	// Mode is CR-027 FR79's step-1 working mode, always either "manual" or
	// "ai" — never "".
	Mode       string
	Topic      string
	Story      string
	Storyboard string
	Code       string
	// Models is the per-step Hive model choice, only meaningful when Mode is
	// "ai" — each field either "" (server default) or one of
	// domain.AuthoringModelCatalog's ids.
	Models domain.AuthoringStepModels
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
	mode, err := uc.authoring.GetAuthoringMode(ctx, projectID)
	if err != nil {
		return AuthoringState{}, err
	}
	models, err := uc.authoring.GetAuthoringModels(ctx, projectID)
	if err != nil {
		return AuthoringState{}, err
	}
	return AuthoringState{
		Mode: domain.NormalizeAuthoringMode(mode), Topic: topic,
		Story: story, Storyboard: storyboard, Code: code, Models: models,
	}, nil
}

// AuthoringModePort persists CR-027 FR79's step-1 working mode.
type AuthoringModePort interface {
	SaveAuthoringMode(ctx context.Context, projectID, mode string) error
}

// SaveAuthoringModeUseCase stores how the Creator is working step 1 — copy the
// prompts out by hand, or let the server call the provider (CR-027 FR79).
//
// Server-side because the choice covers all four tabs and a project can be
// picked up again on any of them: another browser, another machine, or after
// this stack restarts. The browser draft still holds it for the current
// session; this is what makes it survive.
//
// No draft lock and no history row, unlike the four content saves: this is not
// a pipeline artefact, it is how the Creator prefers to work. Refusing to
// remember a preference because the project has moved on to rendering would be
// a lock protecting nothing.
type SaveAuthoringModeUseCase struct {
	authoring AuthoringModePort
}

func NewSaveAuthoringModeUseCase(authoring AuthoringModePort) *SaveAuthoringModeUseCase {
	return &SaveAuthoringModeUseCase{authoring: authoring}
}

// Execute validates the mode and saves it. An unknown mode is rejected rather
// than normalised to the default: it means the caller and the server disagree
// about what modes exist, and silently storing "manual" would hide that.
func (uc *SaveAuthoringModeUseCase) Execute(ctx context.Context, projectID, mode string) error {
	if projectID == "" {
		return fmt.Errorf("project_id is required")
	}
	if !domain.ValidAuthoringMode(mode) {
		return fmt.Errorf("mode must be %q or %q", domain.AuthoringModeManual, domain.AuthoringModeAI)
	}
	return uc.authoring.SaveAuthoringMode(ctx, projectID, mode)
}

// AuthoringModelsPort persists the per-step Hive model choice — the
// model-per-step follow-up to CR-027's FR79 mode switch.
type AuthoringModelsPort interface {
	SaveAuthoringModels(ctx context.Context, projectID string, models domain.AuthoringStepModels) error
}

// SaveAuthoringModelsUseCase stores which Hive model each of the three
// authoring tabs calls when the Creator runs step 1 by API. One picker at
// step 1 sets all three at once (mirrors AuthoringMode: a Creator who has
// already decided this on 1a does not want to decide it again per tab), but
// each tab can still come back and change its own later, so all three save
// together here rather than being locked in per-tab.
type SaveAuthoringModelsUseCase struct {
	authoring AuthoringModelsPort
}

func NewSaveAuthoringModelsUseCase(authoring AuthoringModelsPort) *SaveAuthoringModelsUseCase {
	return &SaveAuthoringModelsUseCase{authoring: authoring}
}

// Execute validates every non-empty model id against the catalog and saves
// the triple. Like SaveAuthoringModeUseCase, an unknown id is rejected
// rather than silently normalised to the default — a client offering a model
// id this server does not recognise means the two have drifted apart, which
// silently falling back would hide.
func (uc *SaveAuthoringModelsUseCase) Execute(ctx context.Context, projectID string, models domain.AuthoringStepModels) error {
	if projectID == "" {
		return fmt.Errorf("project_id is required")
	}
	for _, id := range []string{models.Story, models.Storyboard, models.Code} {
		if !domain.ValidAuthoringModel(id) {
			return fmt.Errorf("unknown model %q", id)
		}
	}
	return uc.authoring.SaveAuthoringModels(ctx, projectID, models)
}

// SimilarProject is one match CR-028 FR85 surfaces back to the Creator when a
// topic collides (after normalization) with another project's saved topic in
// the same content language. Status is not known to authoring-service — the
// orchestrator, which owns projects, fills it in.
type SimilarProject struct {
	ProjectID string               `json:"project_id"`
	Topic     string               `json:"topic"`
	Status    domain.ProjectStatus `json:"status,omitempty"`
	CreatedAt time.Time            `json:"created_at"`
}

// AuthoringSummary is what the orchestrator's project list needs from each
// project's authoring row: the topic to name it by, and which artefacts exist.
type AuthoringSummary struct {
	Topic      string `json:"topic"`
	Story      bool   `json:"story"`
	Storyboard bool   `json:"storyboard"`
	Code       bool   `json:"code"`
}
