package application

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"visual/internal/domain"
)

func TestExternalAI_PreviewModeHoldsThePartAndStoresNothing(t *testing.T) {
	r := newRig(t)
	preview := NewExternalAIRun(ExternalAIPreview, goodReplies[domain.UnitVisualLanguage])
	_, err := r.desk.GenerateUnit(WithExternalAI(ctx, preview), project, domain.UnitVisualLanguage, "")
	if !errors.Is(err, ErrExternalAIPreviewReady) {
		t.Fatalf("preview mode stops for the preview: %v", err)
	}
	raw, ok := preview.Preview()
	if !ok {
		t.Fatal("the preview is held")
	}
	var got struct {
		Key     string          `json:"key"`
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(raw, &got); err != nil || got.Key != domain.UnitVisualLanguage || !strings.Contains(string(got.Content), "shape_rules") {
		t.Errorf("preview %s: %v", raw, err)
	}
	if len(r.store.units) != 0 {
		t.Error("preview mode stores nothing")
	}
}

func TestExternalAI_ReviewPreviewHoldsTheMergedReviewAndPasteStoresIt(t *testing.T) {
	r := newRig(t)
	r.queueAll()
	mustState(t)(r.desk.Generate(ctx, project, "", false))
	reply := `{"findings":[{"group":"feasibility","unit_key":"feasibility","field":"","text":"nặng","detail":"d","fix":"","same_as":""}],"notes":[]}`
	preview := NewExternalAIRun(ExternalAIPreview, reply)
	if _, err := r.desk.Review(WithExternalAI(ctx, preview), project, ""); !errors.Is(err, ErrExternalAIPreviewReady) {
		t.Fatalf("preview mode stops for the preview: %v", err)
	}
	raw, ok := preview.Preview()
	if !ok || !strings.Contains(string(raw), "nặng") {
		t.Fatalf("preview %s", raw)
	}
	if st := mustState(t)(r.desk.State(ctx, project)); len(st.Review.Findings) != 0 {
		t.Error("preview mode stores no review")
	}
	paste := NewExternalAIRun(ExternalAIPaste, reply)
	st := mustState(t)(r.desk.Review(WithExternalAI(ctx, paste), project, ""))
	if len(st.Review.Findings) != 1 {
		t.Errorf("paste stores the review: %+v", st.Review)
	}
}
