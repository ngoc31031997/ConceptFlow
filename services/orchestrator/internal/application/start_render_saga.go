package application

import (
	"context"
	"log/slog"
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

	// CR-019 — empty means DefaultVideoFormatID.
	VideoFormatID string

	// CR-024 FR69.7 — nil means "on". A pointer rather than a bool because the
	// zero value of a bool is false, and defaulting this to off would silently
	// remove the gate for every caller that does not know about it yet.
	ReviewEnabled *bool
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

	// CR-023 FR67.1/FR67.2 — nil means "on", same reasoning as ReviewEnabled
	// above: the zero value of a bool is false, and defaulting the channel
	// identity off would silently drop it for every caller unaware of these
	// fields yet.
	IntroEnabled *bool
	OutroEnabled *bool

	// CR-007 follow-up — empty means DefaultVideoOutputMode ("long").
	VideoOutputMode domain.VideoOutputMode

	// CR-026 D1 — nil means this project stands alone. When set, it must
	// name an existing project covering the same topic; the two get linked
	// both ways (best-effort — see Execute).
	CompanionProjectID *string
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

	outputMode := input.VideoOutputMode
	if !outputMode.IsValid() {
		outputMode = domain.DefaultVideoOutputMode
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
		VideoFormatID:       formatOrDefault(input.VideoFormatID),
		ReviewEnabled:       input.ReviewEnabled == nil || *input.ReviewEnabled,
		// Kept in lockstep with SubtitleMode rather than taken verbatim from
		// input, so anything still reading the legacy field (an older
		// client of GET /v1/projects/{id}) sees a value consistent with
		// what actually got rendered.
		SubtitlesEnabled:      subtitleMode.NeedsCues(),
		SubtitleMode:          subtitleMode,
		SubtitleStyle:         input.SubtitleStyle,
		RenderQuality:         quality,
		BackgroundMusicVolume: input.BackgroundMusicVolume,
		IntroEnabled:          input.IntroEnabled == nil || *input.IntroEnabled,
		OutroEnabled:          input.OutroEnabled == nil || *input.OutroEnabled,
		VideoOutputMode:       outputMode,
		CompanionProjectID:    input.CompanionProjectID,
	}
	if err := uc.repo.Save(ctx, project); err != nil {
		return nil, err
	}

	// CR-026 D1: link the other project back to this new one. Best-effort —
	// a Creator who typed a stale/wrong companion id, or a race with that
	// project being deleted, must never cost them the video they are
	// actually here to create.
	if input.CompanionProjectID != nil {
		if companion, err := uc.repo.Get(ctx, *input.CompanionProjectID); err != nil {
			slog.Warn("could not load companion project to link back", "companion_project_id", *input.CompanionProjectID, "error", err)
		} else {
			companion.CompanionProjectID = &project.ProjectID
			if err := uc.repo.Save(ctx, companion); err != nil {
				slog.Warn("could not save companion project link", "companion_project_id", *input.CompanionProjectID, "error", err)
			}
		}
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

// formatOrDefault keeps every project pointing at a real format, including the
// ones created before formats existed (CR-019 FR51.3).
func formatOrDefault(formatID string) string {
	if formatID == "" {
		return domain.DefaultVideoFormatID
	}
	return formatID
}
