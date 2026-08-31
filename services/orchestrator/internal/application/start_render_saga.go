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
	VoiceLanguage       domain.VoiceLanguage
	BackgroundMusicPath *string // optional, business-rules.md Rule 3
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

	project := &domain.Project{
		ProjectID:           input.ProjectID,
		Status:              domain.StatusDraft,
		SagaID:              sagaID,
		ScriptContent:       input.ScriptContent,
		PluginID:            input.PluginID,
		VoiceLanguage:       input.VoiceLanguage,
		BackgroundMusicPath: input.BackgroundMusicPath,
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
