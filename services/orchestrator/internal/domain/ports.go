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

// ChannelAssetPort resolves the currently-active channel intro/outro asset
// (CR-023 D1/D2, corrected). There is no HTTP server between backend
// services in this system, so Orchestrator does NOT call video-assembly to
// find the active asset — video-assembly's full channel_assets table (with
// real video_path) is never visible to Orchestrator. Instead Orchestrator
// keeps its own lightweight projection, channel_asset_pointers (kind,
// render_quality, asset_id, version — no path), populated by subscribing to
// channel_asset_rendered/channel_asset_normalized events the same way
// handle_step_event.go folds other services' saga events onto Project. This
// port is therefore backed by a local Postgres read, not a network call —
// implemented by adapters/postgres.ChannelAssetPointerRepository.
//
// Returns ("", nil) — not an error — when no active asset exists for
// (kind, quality): a channel that never uploaded an intro/outro is a normal
// state, not a failure, and assemble_video must proceed without one rather
// than fail the saga over it.
type ChannelAssetPort interface {
	LatestChannelAsset(ctx context.Context, kind string, quality RenderQuality) (assetID string, err error)

	// UpsertChannelAssetPointer records a newly-active asset for (kind,
	// quality), called from the channel_asset_rendered/channel_asset_normalized
	// event subscriber. version is whatever video-assembly assigned when it
	// registered the asset in its own channel_assets table.
	UpsertChannelAssetPointer(ctx context.Context, kind string, quality RenderQuality, assetID string, version int) error

	// ListChannelAssetPointers backs GET /v1/channel-assets/preview.
	ListChannelAssetPointers(ctx context.Context) ([]ChannelAssetPointer, error)
}

// QCReportPort persists and reads back the automated QC reports (CR-021 D6,
// FR61.1). Implemented by adapters/postgres.QCReportRepository.
//
// Only the latest report per project is ever read: a project is scored once per
// assemble_video, and an older report describes a video file that no longer
// exists. History is kept rather than overwritten so a re-render's report can
// be compared against the one it replaced while thresholds are being calibrated
// (CR-021's own answer to the false-positive risk).
type QCReportPort interface {
	// SaveQCReport stores one completed QC pass. A redelivered qc_completed
	// event must not accumulate duplicate reports, so implementations key on
	// (project_id, created_at) semantics of "one report per pass" — see the
	// repository for how the saga_id-less event is deduped.
	SaveQCReport(ctx context.Context, report QCReport) error

	// LatestQCReport returns the most recent report for a project, or
	// (nil, nil) when it was never scored — a project rendered before CR-021
	// has no report and must still be publishable (FR61.4's reasoning applies
	// identically to a missing report and an unscorable one).
	LatestQCReport(ctx context.Context, projectID string) (*QCReport, error)

	// RecordQCOverride stamps the latest report as deliberately bypassed
	// (FR61.3). Idempotent: overriding twice keeps the first timestamp, so a
	// double-clicked publish button does not rewrite the audit trail.
	RecordQCOverride(ctx context.Context, projectID string, findings []QCFinding) error
}

// ChannelAssetPointer is one row of Orchestrator's channel_asset_pointers
// projection (CR-023 correction).
type ChannelAssetPointer struct {
	Kind          string
	RenderQuality RenderQuality
	AssetID       string
	Version       int
}
