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
