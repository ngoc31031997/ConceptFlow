package application

import (
	"context"
	"time"

	"orchestrator/internal/domain"
)

// StartRenderSagaInput is the parsed body of POST /v1/sagas/render
// (interface-contracts.md).
type StartRenderSagaInput struct {
	ProjectID           string
	ScriptContent       string
	PluginID            string
	CategoryHint        string // Content Plugin's business-rules.md Rule 1 — Creator-chosen, applied to every scene (Revision 2026-09-05)
	ContentLanguage     domain.ContentLanguage
	BackgroundMusicPath *string // optional, business-rules.md Rule 3

	// CR-001 — narration/subtitle switches chosen by the Creator at submit time.
	TTSEnabled bool
	VoiceID    string
	// SubtitlesEnabled is the pre-CR-015 shape, still accepted from a caller
	// that has not adopted SubtitleMode; SubtitleMode wins when both are
	// sent (a client migrating one field at a time should not regress).
	SubtitlesEnabled bool
	SubtitleMode     domain.SubtitleMode
	SubtitleStyle    *domain.SubtitleStyle

	// CR-004 — empty means DefaultRenderQuality.
	RenderQuality domain.RenderQuality
	// CR-005 FR14.2 — 0 means DefaultBackgroundMusicVolume.
	BackgroundMusicVolume float64
}

// StartRenderSagaOutput is returned to the HTTP layer for the 201 response.
type StartRenderSagaOutput struct {
	SagaID string
	Status domain.ProjectStatus
}

// StartRenderSagaUseCase implements Saga step 1 (business-logic-model.md
// "Bước 1 — Parse Script"): creates the Project, opens the first SagaStep,
// and dispatches the parse_script command via the Outbox.
type StartRenderSagaUseCase struct {
	repo      domain.ProjectRepositoryPort
	publisher domain.CommandPublisherPort
}

// NewStartRenderSagaUseCase constructs the use case with its two port
// dependencies (constructor injection — dependency-injection.md).
func NewStartRenderSagaUseCase(repo domain.ProjectRepositoryPort, publisher domain.CommandPublisherPort) *StartRenderSagaUseCase {
	return &StartRenderSagaUseCase{repo: repo, publisher: publisher}
}

// Execute creates a new Project (status=draft), generates a fresh saga_id
// (a new saga_id per Saga instance — interface-contracts.md Question 9),
// opens the parse_script SagaStep as in_progress, enqueues the parse_script
// command, and advances the Project to parsing_script.
func (uc *StartRenderSagaUseCase) Execute(ctx context.Context, input StartRenderSagaInput) (*StartRenderSagaOutput, error) {
	sagaID := newUUID()

	quality := input.RenderQuality
	if !quality.IsValid() {
		quality = domain.DefaultRenderQuality
	}

	// CR-015: SubtitleMode is authoritative when valid; otherwise fall back
	// to the legacy boolean, which reproduces exactly the one behaviour it
	// ever meant (burn-in) rather than guessing at a new one.
	subtitleMode := input.SubtitleMode
	if !subtitleMode.IsValid() {
		subtitleMode = domain.SubtitleModeFromLegacy(input.SubtitlesEnabled)
	}

	project := &domain.Project{
		ProjectID:           input.ProjectID,
		Status:              domain.StatusDraft,
		SagaID:              sagaID,
		ScriptContent:       input.ScriptContent,
		PluginID:            input.PluginID,
		CategoryHint:        input.CategoryHint,
		ContentLanguage:     input.ContentLanguage,
		BackgroundMusicPath: input.BackgroundMusicPath,
		TTSEnabled:          input.TTSEnabled,
		VoiceID:             input.VoiceID,
		// Kept in lockstep with SubtitleMode rather than taken verbatim from
		// input, so anything still reading the legacy field (an older
		// client of GET /v1/projects/{id}) sees a value consistent with
		// what actually got rendered.
		SubtitlesEnabled:      subtitleMode.NeedsCues(),
		SubtitleMode:          subtitleMode,
		SubtitleStyle:         input.SubtitleStyle,
		RenderQuality:         quality,
		BackgroundMusicVolume: input.BackgroundMusicVolume,
	}
	if err := uc.repo.Save(ctx, project); err != nil {
		return nil, err
	}

	step := &domain.SagaStep{
		SagaID:   sagaID,
		StepName: domain.StepParseScript,
		Status:   domain.SagaStepInProgress,
	}
	if err := uc.repo.UpdateStep(ctx, step); err != nil {
		return nil, err
	}

	envelope := domain.Envelope{
		MessageID: newUUID(),
		SagaID:    sagaID,
		ProjectID: input.ProjectID,
		EventType: string(domain.StepParseScript),
		Payload: map[string]interface{}{
			"script_content": input.ScriptContent,
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	if err := uc.publisher.PublishCommand(ctx, "script_processing", envelope); err != nil {
		return nil, err
	}

	if err := uc.repo.UpdateStatus(ctx, input.ProjectID, domain.StatusParsingScript); err != nil {
		return nil, err
	}

	return &StartRenderSagaOutput{SagaID: sagaID, Status: domain.StatusParsingScript}, nil
}
