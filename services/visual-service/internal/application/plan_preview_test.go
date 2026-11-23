package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestPlanGenerateBeat_PreviewStopsBeforeStoring(t *testing.T) {
	f := newAIFixture(t)
	f.desk.WithAI(NewExternalAIProvider(f.llm, f.llm), seedPrompts{}, planModels{}, 1000)
	before := fmt.Sprint(f.store.units)
	run := NewExternalAIRun(ExternalAIPreview, beatReply(unitJSON("xem trước", "b1.1", "b1.2", "b1.3")))
	_, err := f.desk.GenerateBeat(WithExternalAI(context.Background(), run), project, "b1", "")
	if !errors.Is(err, ErrExternalAIPreviewReady) {
		t.Fatalf("preview mode stops once the reply is read: %v", err)
	}
	raw, ok := run.Preview()
	if !ok || !strings.Contains(string(raw), `"narration_summary":"xem trước"`) {
		t.Errorf("the preview holds the units: %s", raw)
	}
	if fmt.Sprint(f.store.units) != before {
		t.Errorf("nothing is stored in preview mode")
	}
}

func TestPlanReview_PreviewStopsBeforeStoring(t *testing.T) {
	f := newAIFixture(t)
	f.desk.WithAI(NewExternalAIProvider(f.llm, f.llm), seedPrompts{}, planModels{}, 1000)
	before := fmt.Sprint(f.store.units)
	reply := reviewReply(finding("clarity", "VP-001", "communication_goal", "mục tiêu mơ hồ", ""))
	run := NewExternalAIRun(ExternalAIPreview, reply)
	_, err := f.desk.Review(WithExternalAI(context.Background(), run), project, "")
	if !errors.Is(err, ErrExternalAIPreviewReady) {
		t.Fatalf("preview mode stops once the reply is read: %v", err)
	}
	raw, ok := run.Preview()
	if !ok || !strings.Contains(string(raw), "mục tiêu mơ hồ") {
		t.Errorf("the preview holds the merged review: %s", raw)
	}
	if fmt.Sprint(f.store.units) != before {
		t.Errorf("nothing is stored in preview mode")
	}
	paste := NewExternalAIRun(ExternalAIPaste, reply)
	st, err := f.desk.Review(WithExternalAI(context.Background(), paste), project, "")
	if err != nil || len(st.Review.Findings) != 1 {
		t.Fatalf("paste mode still stores: %v", err)
	}
}
