package application

import (
	"context"
	"time"

	"orchestrator/internal/domain"
)

// StartPublishSagaInput is the parsed body of POST /v1/sagas/publish
// (interface-contracts.md).
type StartPublishSagaInput struct {
	ProjectID   string
	Title       string
	Description *string
	Tags        []string
	Visibility  domain.Visibility
	PublishAt   *string // RFC3339 — only set alongside Visibility == private (validated by the HTTP layer)
}

// StartPublishSagaOutput is returned to the HTTP layer for the 201 response.
type StartPublishSagaOutput struct {
	SagaID string
	Status domain.ProjectStatus
}

// StartPublishSagaUseCase implements Saga step 6 (business-logic-model.md
// "Bước 6 — Publish Video"): validates the project is ready_to_publish,
// stores the youtube metadata, and dispatches publish_video via the Outbox.
type StartPublishSagaUseCase struct {
	repo      domain.ProjectRepositoryPort
	publisher domain.CommandPublisherPort
}

// NewStartPublishSagaUseCase constructs the use case with its port
// dependencies.
func NewStartPublishSagaUseCase(repo domain.ProjectRepositoryPort, publisher domain.CommandPublisherPort) *StartPublishSagaUseCase {
	return &StartPublishSagaUseCase{repo: repo, publisher: publisher}
}

// Execute validates Project.Status == ready_to_publish (business-rules.md
// Rule 4 — else domain.ErrInvalidStatus, translated to 409 by the HTTP
// layer), persists the youtube metadata onto Project, opens the
// publish_video SagaStep, and dispatches the command.
func (uc *StartPublishSagaUseCase) Execute(ctx context.Context, input StartPublishSagaInput) (*StartPublishSagaOutput, error) {
	project, err := uc.repo.Get(ctx, input.ProjectID)
	if err != nil {
		return nil, err
	}
	if project.Status != domain.StatusReadyToPublish {
		return nil, domain.ErrInvalidStatus
	}

	sagaID := newUUID()

	project.SagaID = sagaID
	project.YoutubeTitle = &input.Title
	project.YoutubeDescription = input.Description
	project.YoutubeTags = input.Tags
	visibility := input.Visibility
	project.YoutubeVisibility = &visibility
	project.YoutubePublishAt = input.PublishAt
	if err := uc.repo.Save(ctx, project); err != nil {
		return nil, err
	}

	step := &domain.SagaStep{
		SagaID:   sagaID,
		StepName: domain.StepPublishVideo,
		Status:   domain.SagaStepInProgress,
	}
	if err := uc.repo.UpdateStep(ctx, step); err != nil {
		return nil, err
	}

	envelope := domain.Envelope{
		MessageID: newUUID(),
		SagaID:    sagaID,
		ProjectID: input.ProjectID,
		EventType: string(domain.StepPublishVideo),
		Payload:   publishVideoPayload(project),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	if err := uc.publisher.PublishCommand(ctx, "publisher", envelope); err != nil {
		return nil, err
	}

	if err := uc.repo.UpdateStatus(ctx, input.ProjectID, domain.StatusPublishing); err != nil {
		return nil, err
	}

	return &StartPublishSagaOutput{SagaID: sagaID, Status: domain.StatusPublishing}, nil
}

// publishVideoPayload builds the publish_video command payload from Project
// data (video_path from step 5 + youtube metadata from Saga Publish input —
// interface-contracts.md).
func publishVideoPayload(project *domain.Project) map[string]interface{} {
	payload := map[string]interface{}{
		"video_path": derefString(project.VideoPath),
		"title":      derefString(project.YoutubeTitle),
		"tags":       project.YoutubeTags,
	}
	if project.YoutubeDescription != nil {
		payload["description"] = *project.YoutubeDescription
	}
	if project.YoutubeVisibility != nil {
		payload["visibility"] = string(*project.YoutubeVisibility)
	}
	if project.YoutubePublishAt != nil {
		payload["publish_at"] = *project.YoutubePublishAt
	}
	return payload
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
