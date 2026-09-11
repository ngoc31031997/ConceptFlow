package application

import (
	"context"
	"time"

	"orchestrator/internal/domain"
)

// StartPublishSagaInput is the parsed body of POST /v1/sagas/publish
// (interface-contracts.md).
type StartPublishSagaInput struct {
	ProjectID     string
	Title         string
	Description   *string
	Tags          []string
	Visibility    domain.Visibility
	PublishAt     *string // RFC3339 — only set alongside Visibility == private (validated by the HTTP layer)
	ThumbnailPath *string // absolute path on shared_artifacts, from a prior POST /v1/projects/{id}/thumbnail upload
	ChannelID     *string // connected YouTube channel to publish to; nil leaves the choice to the Publisher's default (CR-012)
	// AcknowledgeQC is the Creator saying, deliberately, that they have read
	// the blocking QC findings and want to publish anyway (CR-021 FR61.3).
	// It only means anything when QC_ENFORCE is on, and it is recorded on the
	// report either way it is used — an override nobody can see afterwards is
	// not an override, it is a hole.
	AcknowledgeQC bool
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
	// qcReports and qcEnforce are CR-021's publish gate (D5). Both are
	// optional: with no report store, or with enforcement off, Execute behaves
	// exactly as it did before this CR.
	qcReports domain.QCReportPort
	qcEnforce bool
}

// NewStartPublishSagaUseCase constructs the use case with its port
// dependencies.
func NewStartPublishSagaUseCase(repo domain.ProjectRepositoryPort, publisher domain.CommandPublisherPort) *StartPublishSagaUseCase {
	return &StartPublishSagaUseCase{repo: repo, publisher: publisher}
}

// WithQCGate attaches CR-021's publish gate.
//
// enforce comes from QC_ENFORCE and defaults to false (D5 / Decision #3): until
// the thresholds have been calibrated against real footage, findings are shown
// and recorded but never block. Passing the flag in rather than reading the
// environment here keeps the use case testable and keeps config-reading in
// exactly one package.
func (uc *StartPublishSagaUseCase) WithQCGate(qcReports domain.QCReportPort, enforce bool) *StartPublishSagaUseCase {
	uc.qcReports = qcReports
	uc.qcEnforce = enforce
	return uc
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

	// CR-021 FR61.3: the gate lives here, not in the GUI. A button is not where
	// a rule is kept — anything that can POST this endpoint would otherwise
	// walk straight past it.
	if err := uc.checkQCGate(ctx, input); err != nil {
		return nil, err
	}

	sagaID := newUUID()

	project.SagaID = sagaID
	project.YoutubeTitle = &input.Title
	project.YoutubeDescription = input.Description
	project.YoutubeTags = input.Tags
	visibility := input.Visibility
	project.YoutubeVisibility = &visibility
	project.YoutubePublishAt = input.PublishAt
	project.YoutubeThumbnailPath = input.ThumbnailPath
	project.YoutubeChannelID = input.ChannelID
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

// checkQCGate refuses a publish whose latest QC report carries a blocking
// finding, unless the Creator acknowledged it (CR-021 FR61.3).
//
// Three ways this returns nil, and each is deliberate:
//   - QC_ENFORCE off (the default, D5) — findings are advice, not a lock, while
//     the thresholds are still being calibrated against real videos.
//   - No report at all — a project rendered before CR-021, or one whose report
//     failed to persist. FR61.4's principle covers both: an absent measurement
//     is not evidence of a problem.
//   - status not_scored, or findings that are all warnings — nothing blocking
//     was actually found.
func (uc *StartPublishSagaUseCase) checkQCGate(ctx context.Context, input StartPublishSagaInput) error {
	if uc.qcReports == nil {
		return nil
	}
	report, err := uc.qcReports.LatestQCReport(ctx, input.ProjectID)
	if err != nil {
		// A gate that cannot read its own report must open, not close
		// (FR61.4): a database hiccup is not a quality finding.
		return nil
	}
	if report == nil || !report.HasBlockingFindings() {
		return nil
	}

	if !uc.qcEnforce {
		return nil
	}
	if !input.AcknowledgeQC {
		return domain.ErrQCBlocked
	}

	// The override is recorded before the command goes out, so a publish can
	// never exist without the trace explaining why it was allowed. Best-effort
	// on the write itself: the Creator has already made the decision, and
	// refusing to act on it because the audit row failed would be punishing
	// them for an infrastructure problem.
	if err := uc.qcReports.RecordQCOverride(ctx, input.ProjectID, report.BlockingFindings()); err != nil {
		return nil
	}
	return nil
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
	if project.YoutubeThumbnailPath != nil {
		payload["thumbnail_path"] = *project.YoutubeThumbnailPath
	}
	if project.CaptionPath != nil {
		payload["caption_path"] = *project.CaptionPath
		// CR-015 FR39.3: YouTube's captions.insert needs a BCP-47 language
		// code to attach the track under. ContentLanguage's values ("vi",
		// "en" — CR-008) are already valid BCP-47 primary subtags, so no
		// translation table is needed. Wrong here means auto-translate
		// dubs from the wrong source language — exactly what CR-015 set
		// out to fix, not reintroduce.
		payload["caption_language"] = string(project.ContentLanguage)
	}
	// Omitted rather than sent as null when unset, so the Publisher's
	// payload.get("channel_id") keeps meaning "use the default channel"
	// for projects created before CR-012.
	if project.YoutubeChannelID != nil {
		payload["channel_id"] = *project.YoutubeChannelID
	}
	return payload
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
