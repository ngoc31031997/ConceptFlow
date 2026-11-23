package application

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestProjectBriefGenerateStopsForPreviewBeforeStoring(t *testing.T) {
	f := newProjectBriefFixture()
	run := NewExternalAIRun(ExternalAIPreview, "")
	_, err := f.uc.Generate(WithExternalAI(context.Background(), run), "p1", "ngắn hơn")
	if !errors.Is(err, ErrExternalAIPreviewReady) {
		t.Fatalf("want ErrExternalAIPreviewReady, got %v", err)
	}
	if f.store.brief != nil {
		t.Error("a previewed brief was stored")
	}
	raw, ok := run.Preview()
	if !ok || !strings.Contains(string(raw), `"duration_seconds":270`) || !strings.Contains(string(raw), `"audience"`) {
		t.Errorf("preview does not hold the brief: %s", raw)
	}
	paste := NewExternalAIRun(ExternalAIPaste, projectBriefReply)
	if _, err := f.uc.Generate(WithExternalAI(context.Background(), paste), "p1", ""); err != nil {
		t.Fatal(err)
	}
	if f.store.brief == nil {
		t.Error("paste mode did not store the brief")
	}
}

func TestProjectBriefReviewStopsForPreviewBeforeStoring(t *testing.T) {
	f := newProjectBriefFixture()
	ctx := context.Background()
	if _, err := f.uc.Generate(ctx, "p1", ""); err != nil {
		t.Fatal(err)
	}
	f.llm.reply = briefReviewReply
	run := NewExternalAIRun(ExternalAIPreview, "")
	_, err := f.uc.Review(WithExternalAI(ctx, run), "p1")
	if !errors.Is(err, ErrExternalAIPreviewReady) {
		t.Fatalf("want ErrExternalAIPreviewReady, got %v", err)
	}
	if f.store.review != nil {
		t.Error("a previewed review was stored")
	}
	raw, ok := run.Preview()
	if !ok || !strings.Contains(string(raw), `"audience_fit"`) || !strings.Contains(string(raw), "Thêm một dòng về kích thước") {
		t.Errorf("preview does not hold the review: %s", raw)
	}
	paste := NewExternalAIRun(ExternalAIPaste, briefReviewReply)
	if _, err := f.uc.Review(WithExternalAI(ctx, paste), "p1"); err != nil {
		t.Fatal(err)
	}
	if f.store.review == nil {
		t.Error("paste mode did not store the review")
	}
}
