package application

import (
	"context"
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
