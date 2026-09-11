package application

import (
	"context"
	"errors"
	"testing"

	"orchestrator/internal/domain"
)

func TestStartPublishSagaUseCase_Execute(t *testing.T) {
	repo := newFakeRepo()
	pub := &fakePublisher{}
	uc := NewStartPublishSagaUseCase(repo, pub)

	videoPath := "video.mp4"
	repo.projects["proj-1"] = &domain.Project{
		ProjectID: "proj-1",
		Status:    domain.StatusReadyToPublish,
		VideoPath: &videoPath,
	}

	out, err := uc.Execute(context.Background(), StartPublishSagaInput{
		ProjectID:  "proj-1",
		Title:      "My Video",
		Visibility: domain.VisibilityPublic,
		Tags:       []string{"go", "concept"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Status != domain.StatusPublishing {
		t.Fatalf("expected status publishing, got %s", out.Status)
	}

	last := pub.last()
	if last == nil || last.routingKey != "publisher" {
		t.Fatalf("expected command published to publisher routing key, got %+v", last)
	}
	if last.envelope.Payload["video_path"] != videoPath {
		t.Fatalf("expected video_path %s in payload, got %v", videoPath, last.envelope.Payload)
	}
}

func TestStartPublishSagaUseCase_RejectsWrongStatus(t *testing.T) {
	repo := newFakeRepo()
	pub := &fakePublisher{}
	uc := NewStartPublishSagaUseCase(repo, pub)

	repo.projects["proj-1"] = &domain.Project{ProjectID: "proj-1", Status: domain.StatusRendering}

	_, err := uc.Execute(context.Background(), StartPublishSagaInput{ProjectID: "proj-1", Title: "x", Visibility: domain.VisibilityPublic})
	if err != domain.ErrInvalidStatus {
		t.Fatalf("expected ErrInvalidStatus, got %v", err)
	}
}

// CR-012: the chosen channel has to survive all the way into the AMQP
// payload, since that is the only thing the Publisher sees.
func TestStartPublishSagaUseCase_CarriesChannelIDIntoPayload(t *testing.T) {
	repo := newFakeRepo()
	pub := &fakePublisher{}
	uc := NewStartPublishSagaUseCase(repo, pub)

	videoPath := "video.mp4"
	repo.projects["proj-1"] = &domain.Project{
		ProjectID: "proj-1",
		Status:    domain.StatusReadyToPublish,
		VideoPath: &videoPath,
	}

	channelID := "UC_channel_two"
	if _, err := uc.Execute(context.Background(), StartPublishSagaInput{
		ProjectID:  "proj-1",
		Title:      "My Video",
		Visibility: domain.VisibilityPublic,
		ChannelID:  &channelID,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := pub.last().envelope.Payload["channel_id"]; got != channelID {
		t.Fatalf("expected channel_id %q in payload, got %v", channelID, got)
	}
	if stored := repo.projects["proj-1"].YoutubeChannelID; stored == nil || *stored != channelID {
		t.Fatalf("expected channel_id persisted on the project, got %v", stored)
	}
}

// CR-015 FR38.4/FR39.3: caption_path and its language have to reach the
// Publisher the same way thumbnail_path does — through this payload — since
// that's the only thing the Publisher sees.
func TestStartPublishSagaUseCase_CarriesCaptionPathAndLanguageIntoPayload(t *testing.T) {
	repo := newFakeRepo()
	pub := &fakePublisher{}
	uc := NewStartPublishSagaUseCase(repo, pub)

	videoPath := "video.mp4"
	captionPath := "/shared/proj-1/video/final.srt"
	repo.projects["proj-1"] = &domain.Project{
		ProjectID:       "proj-1",
		Status:          domain.StatusReadyToPublish,
		VideoPath:       &videoPath,
		CaptionPath:     &captionPath,
		ContentLanguage: domain.LanguageVietnamese,
	}

	if _, err := uc.Execute(context.Background(), StartPublishSagaInput{
		ProjectID: "proj-1", Title: "My Video", Visibility: domain.VisibilityPublic,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	payload := pub.last().envelope.Payload
	if payload["caption_path"] != captionPath {
		t.Fatalf("expected caption_path %q in payload, got %v", captionPath, payload["caption_path"])
	}
	if payload["caption_language"] != "vi" {
		t.Fatalf("expected caption_language vi in payload, got %v", payload["caption_language"])
	}
}

// A project with no caption track (subtitles off, or burn-in only) must not
// send either key — a Publisher reading a present-but-empty caption_path
// would attempt to upload a caption track from a path that does not exist.
func TestStartPublishSagaUseCase_OmitsCaptionFieldsWhenUnset(t *testing.T) {
	repo := newFakeRepo()
	pub := &fakePublisher{}
	uc := NewStartPublishSagaUseCase(repo, pub)

	videoPath := "video.mp4"
	repo.projects["proj-1"] = &domain.Project{
		ProjectID: "proj-1",
		Status:    domain.StatusReadyToPublish,
		VideoPath: &videoPath,
	}

	if _, err := uc.Execute(context.Background(), StartPublishSagaInput{
		ProjectID: "proj-1", Title: "My Video", Visibility: domain.VisibilityPublic,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	payload := pub.last().envelope.Payload
	if _, ok := payload["caption_path"]; ok {
		t.Fatalf("expected no caption_path key, got %v", payload["caption_path"])
	}
	if _, ok := payload["caption_language"]; ok {
		t.Fatalf("expected no caption_language key, got %v", payload["caption_language"])
	}
}

// Projects created before CR-012 carry no channel. The key must be absent
// rather than present-and-null, because the Publisher reads a missing key as
// "use the default channel".
func TestStartPublishSagaUseCase_OmitsChannelIDWhenUnset(t *testing.T) {
	repo := newFakeRepo()
	pub := &fakePublisher{}
	uc := NewStartPublishSagaUseCase(repo, pub)

	videoPath := "video.mp4"
	repo.projects["proj-1"] = &domain.Project{
		ProjectID: "proj-1",
		Status:    domain.StatusReadyToPublish,
		VideoPath: &videoPath,
	}

	if _, err := uc.Execute(context.Background(), StartPublishSagaInput{
		ProjectID:  "proj-1",
		Title:      "My Video",
		Visibility: domain.VisibilityPublic,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, present := pub.last().envelope.Payload["channel_id"]; present {
		t.Fatal("expected channel_id to be absent from the payload when no channel was chosen")
	}
}

// blockingReport is a QCReport with one blocking finding, the shared fixture
// for the QC-gate tests below (CR-021 D5/FR61.3).
func blockingReport(projectID string) domain.QCReport {
	return domain.QCReport{
		ProjectID: projectID,
		Status:    domain.QCStatusHasFindings,
		Findings: []domain.QCFinding{
			{Rule: "frame_overflow", Severity: domain.QCSeverityBlocking, Message: "text overflows frame", TimestampSeconds: 12.5},
		},
	}
}

// TestStartPublishSagaUseCase_QCEnforceFalse_DoesNotBlockEvenWithBlockingFinding
// is D5's default (Decision #3): QC_ENFORCE=false means findings are shown but
// never stop a publish, no matter their severity.
func TestStartPublishSagaUseCase_QCEnforceFalse_DoesNotBlockEvenWithBlockingFinding(t *testing.T) {
	repo := newFakeRepo()
	pub := &fakePublisher{}
	qc := newFakeQCReports()
	_ = qc.SaveQCReport(context.Background(), blockingReport("proj-1"))
	uc := NewStartPublishSagaUseCase(repo, pub).WithQCGate(qc, false)

	videoPath := "video.mp4"
	repo.projects["proj-1"] = &domain.Project{ProjectID: "proj-1", Status: domain.StatusReadyToPublish, VideoPath: &videoPath}

	out, err := uc.Execute(context.Background(), StartPublishSagaInput{
		ProjectID: "proj-1", Title: "My Video", Visibility: domain.VisibilityPublic,
	})
	if err != nil {
		t.Fatalf("expected publish to proceed with QC_ENFORCE=false, got error: %v", err)
	}
	if out.Status != domain.StatusPublishing {
		t.Fatalf("expected publishing, got %s", out.Status)
	}
}

// TestStartPublishSagaUseCase_QCEnforceTrue_BlocksOnBlockingFinding is FR61.3's
// core rule: with the gate enforced, a blocking finding refuses the publish
// unless the Creator acknowledges it.
func TestStartPublishSagaUseCase_QCEnforceTrue_BlocksOnBlockingFinding(t *testing.T) {
	repo := newFakeRepo()
	pub := &fakePublisher{}
	qc := newFakeQCReports()
	_ = qc.SaveQCReport(context.Background(), blockingReport("proj-1"))
	uc := NewStartPublishSagaUseCase(repo, pub).WithQCGate(qc, true)

	videoPath := "video.mp4"
	repo.projects["proj-1"] = &domain.Project{ProjectID: "proj-1", Status: domain.StatusReadyToPublish, VideoPath: &videoPath}

	_, err := uc.Execute(context.Background(), StartPublishSagaInput{
		ProjectID: "proj-1", Title: "My Video", Visibility: domain.VisibilityPublic,
	})
	if !errors.Is(err, domain.ErrQCBlocked) {
		t.Fatalf("expected ErrQCBlocked, got %v", err)
	}
	if len(pub.published) != 0 {
		t.Fatalf("expected no command published when blocked, got %d", len(pub.published))
	}
}

// TestStartPublishSagaUseCase_QCEnforceTrue_AcknowledgeQCOverridesAndRecords is
// FR61.3's escape hatch: acknowledge_qc:true lets the publish through, and the
// bypass must be recorded (overridden_at) so it is never a silent override.
func TestStartPublishSagaUseCase_QCEnforceTrue_AcknowledgeQCOverridesAndRecords(t *testing.T) {
	repo := newFakeRepo()
	pub := &fakePublisher{}
	qc := newFakeQCReports()
	_ = qc.SaveQCReport(context.Background(), blockingReport("proj-1"))
	uc := NewStartPublishSagaUseCase(repo, pub).WithQCGate(qc, true)

	videoPath := "video.mp4"
	repo.projects["proj-1"] = &domain.Project{ProjectID: "proj-1", Status: domain.StatusReadyToPublish, VideoPath: &videoPath}

	out, err := uc.Execute(context.Background(), StartPublishSagaInput{
		ProjectID: "proj-1", Title: "My Video", Visibility: domain.VisibilityPublic,
		AcknowledgeQC: true,
	})
	if err != nil {
		t.Fatalf("expected publish to proceed with acknowledge_qc, got error: %v", err)
	}
	if out.Status != domain.StatusPublishing {
		t.Fatalf("expected publishing, got %s", out.Status)
	}
	if len(pub.published) != 1 {
		t.Fatalf("expected exactly one command published, got %d", len(pub.published))
	}

	report, _ := qc.LatestQCReport(context.Background(), "proj-1")
	if report == nil || report.OverriddenAt == nil {
		t.Fatalf("expected overridden_at recorded on the qc report, got %+v", report)
	}
	if len(report.OverriddenFindings) != 1 || report.OverriddenFindings[0].Rule != "frame_overflow" {
		t.Fatalf("expected the blocking finding copied into overridden_findings, got %+v", report.OverriddenFindings)
	}
}

// TestStartPublishSagaUseCase_QCEnforceTrue_NoBlockingFindingsProceeds ensures
// the gate only ever reacts to blocking severity — a report with only
// warnings, or no report at all, must never require acknowledge_qc.
func TestStartPublishSagaUseCase_QCEnforceTrue_NoBlockingFindingsProceeds(t *testing.T) {
	repo := newFakeRepo()
	pub := &fakePublisher{}
	qc := newFakeQCReports()
	_ = qc.SaveQCReport(context.Background(), domain.QCReport{
		ProjectID: "proj-1", Status: domain.QCStatusHasFindings,
		Findings: []domain.QCFinding{{Rule: "lufs_deviation", Severity: domain.QCSeverityWarning, Message: "quiet mix"}},
	})
	uc := NewStartPublishSagaUseCase(repo, pub).WithQCGate(qc, true)

	videoPath := "video.mp4"
	repo.projects["proj-1"] = &domain.Project{ProjectID: "proj-1", Status: domain.StatusReadyToPublish, VideoPath: &videoPath}

	_, err := uc.Execute(context.Background(), StartPublishSagaInput{
		ProjectID: "proj-1", Title: "My Video", Visibility: domain.VisibilityPublic,
	})
	if err != nil {
		t.Fatalf("expected publish to proceed with only warning findings, got error: %v", err)
	}
}
