package domain

import "context"

// Envelope is the outbound command message shape published to
// commands.direct — mirrors messaging-design.md's event envelope, reused for
// commands (message_id/saga_id/project_id/payload/timestamp).
type Envelope struct {
	MessageID string                 `json:"message_id"`
	SagaID    string                 `json:"saga_id"`
	ProjectID string                 `json:"project_id"`
	EventType string                 `json:"event_type"`
	Payload   map[string]interface{} `json:"payload"`
	Timestamp string                 `json:"timestamp"`
}

// CommandPublisherPort is implemented by adapters/amqp.Publisher. It is kept
// separate from ProgressPublisherPort (Interface Segregation) even though a
// single struct implements both, because use cases that only dispatch
// commands should not depend on progress-publishing capability.
type CommandPublisherPort interface {
	PublishCommand(ctx context.Context, routingKey string, envelope Envelope) error
}

// ProgressPublisherPort is implemented by adapters/amqp.Publisher; publishes
// ProgressMessage to progress.fanout (ADR-0017).
type ProgressPublisherPort interface {
	PublishProgress(ctx context.Context, msg ProgressMessage) error
}

// ProjectRepositoryPort abstracts persistence of Project and SagaStep so
// application/ use cases never depend on pgx directly (module-structure.md
// Dependency Direction).
type ProjectRepositoryPort interface {
	Get(ctx context.Context, projectID string) (*Project, error)
	List(ctx context.Context) ([]ProjectSummary, error)
	Save(ctx context.Context, project *Project) error
	UpdateStatus(ctx context.Context, projectID string, status ProjectStatus) error
	Delete(ctx context.Context, projectID string) error
	GetStep(ctx context.Context, sagaID string, stepName StepName) (*SagaStep, error)
	UpdateStep(ctx context.Context, step *SagaStep) error

	// RecordVoiceSamples folds one project's measurement into a voice's
	// running totals (CR-016 FR43.1). Additive rather than replacing, so a
	// voice's estimate keeps improving instead of swinging with the last
	// project rendered.
	RecordVoiceSamples(ctx context.Context, voiceID string, words int, seconds float64) error
	GetVoiceCalibration(ctx context.Context, voiceID string) (VoiceCalibration, error)
	ListVoiceCalibrations(ctx context.Context) ([]VoiceCalibration, error)

	// CR-019: hình dạng video, lưu dưới dạng dữ liệu sửa được.
	SeedVideoFormats(ctx context.Context) error
	GetVideoFormat(ctx context.Context, formatID string, version int) (VideoFormat, error)
	ListVideoFormats(ctx context.Context) ([]VideoFormat, error)
	SaveVideoFormat(ctx context.Context, format VideoFormat) (VideoFormat, error)
}
