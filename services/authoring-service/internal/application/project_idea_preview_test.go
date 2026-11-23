package application

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func externalIdeaFixture() *projectIdeaFixture {
	f := newProjectIdeaFixture()
	f.uc.provider = NewExternalAIProvider(f.llm, &pastedReplier{})
	return f
}

func TestProjectIdeaGenerateStopsForPreviewBeforeStoring(t *testing.T) {
	f := externalIdeaFixture()
	run := NewExternalAIRun(ExternalAIPreview, projectIdeaReply)
	_, err := f.uc.Generate(WithExternalAI(context.Background(), run), "p1", "")
	if !errors.Is(err, ErrExternalAIPreviewReady) {
		t.Fatalf("err = %v, want ErrExternalAIPreviewReady", err)
	}
	preview, ok := run.Preview()
	if !ok || !strings.Contains(string(preview), "Người rơi thấy gì, người ở xa thấy gì?") || !strings.Contains(string(preview), "Chân trời sự kiện là gì?") {
		t.Errorf("preview = %s", preview)
	}
	if f.store.idea != nil {
		t.Error("the preview stored an idea")
	}
}

func TestProjectIdeaGenerateStoresInPasteMode(t *testing.T) {
	f := externalIdeaFixture()
	run := NewExternalAIRun(ExternalAIPaste, projectIdeaReply)
	if _, err := f.uc.Generate(WithExternalAI(context.Background(), run), "p1", ""); err != nil {
		t.Fatal(err)
	}
	if f.store.idea == nil || f.store.idea.Idea.CentralQuestion == "" {
		t.Error("paste mode did not store the idea")
	}
}

func TestProjectIdeaReviewStopsForPreviewBeforeStoring(t *testing.T) {
	f := externalIdeaFixture()
	if _, err := f.uc.Generate(context.Background(), "p1", ""); err != nil {
		t.Fatal(err)
	}
	run := NewExternalAIRun(ExternalAIPreview, ideaReviewReply)
	_, err := f.uc.Review(WithExternalAI(context.Background(), run), "p1")
	if !errors.Is(err, ErrExternalAIPreviewReady) {
		t.Fatalf("err = %v, want ErrExternalAIPreviewReady", err)
	}
	preview, ok := run.Preview()
	if !ok || !strings.Contains(string(preview), "Câu hỏi rõ.") {
		t.Errorf("preview = %s", preview)
	}
	if f.store.review != nil {
		t.Error("the preview stored a review")
	}
	paste := NewExternalAIRun(ExternalAIPaste, ideaReviewReply)
	if _, err := f.uc.Review(WithExternalAI(context.Background(), paste), "p1"); err != nil {
		t.Fatal(err)
	}
	if f.store.review == nil {
		t.Error("paste mode did not store the review")
	}
}
