package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// previewCtx returns a context carrying one external-AI run in mode with reply.
func previewCtx(mode ExternalAIMode, reply string) (context.Context, *ExternalAIRun) {
	run := NewExternalAIRun(mode, reply)
	return WithExternalAI(context.Background(), run), run
}

func TestSequenceAI_PreviewStopsBeforeStoringMotifs(t *testing.T) {
	f := newSeqAIFixture(t)
	f.desk.WithAI(NewExternalAIProvider(f.llm, f.llm), seedPrompts{}, seqModels{}, 1000)
	reply := motifsReply(motifJSON("", "Dán", anchorBee))
	before := fmt.Sprint(f.store.units)
	ctx, run := previewCtx(ExternalAIPreview, reply)
	if _, err := f.desk.GenerateMotifs(ctx, project, ""); !errors.Is(err, ErrExternalAIPreviewReady) {
		t.Fatalf("preview stops: %v", err)
	}
	if p, ok := run.Preview(); !ok || !strings.Contains(string(p), "Dán") {
		t.Fatalf("preview holds the motifs: %s", p)
	}
	if fmt.Sprint(f.store.units) != before {
		t.Fatalf("nothing stored in preview mode")
	}
	ctx, _ = previewCtx(ExternalAIPaste, reply)
	st, err := f.desk.GenerateMotifs(ctx, project, "")
	if err != nil || len(st.Motifs.Motifs) != 1 {
		t.Fatalf("paste mode still stores: %v", err)
	}
}

func TestSequenceAI_PreviewStopsBeforeStoringAUnit(t *testing.T) {
	f := newSeqAIFixture(t)
	f.approvedMotifs()
	f.desk.WithAI(NewExternalAIProvider(f.llm, f.llm), seedPrompts{}, seqModels{}, 1000)
	reply := itemReply(seqItemJSON("VP-001", true, "slow", "", "MOTIF-01"))
	before := fmt.Sprint(f.store.units)
	ctx, run := previewCtx(ExternalAIPreview, reply)
	if _, err := f.desk.GenerateUnit(ctx, project, "VP-001", ""); !errors.Is(err, ErrExternalAIPreviewReady) {
		t.Fatalf("preview stops: %v", err)
	}
	if p, ok := run.Preview(); !ok || !strings.Contains(string(p), "VP-001") {
		t.Fatalf("preview holds the item: %s", p)
	}
	if fmt.Sprint(f.store.units) != before {
		t.Fatalf("nothing stored in preview mode")
	}
	ctx, _ = previewCtx(ExternalAIPaste, reply)
	if _, err := f.desk.GenerateUnit(ctx, project, "VP-001", ""); err != nil {
		t.Fatalf("paste mode still stores: %v", err)
	}
	if fmt.Sprint(f.store.units) == before {
		t.Fatalf("paste mode stored the item")
	}
}

func TestSequenceAI_PreviewStopsBeforeStoringAReview(t *testing.T) {
	f := newSeqAIFixture(t)
	f.approvedMotifs()
	f.writeItem("VP-001", true, "MOTIF-01")
	f.desk.WithAI(NewExternalAIProvider(f.llm, f.llm), seedPrompts{}, seqModels{}, 1000)
	reply := reviewReply(finding("clarity", "VP-001", "purpose_in_flow", "mơ hồ", ""))
	before := fmt.Sprint(f.store.units)
	ctx, run := previewCtx(ExternalAIPreview, reply)
	if _, err := f.desk.Review(ctx, project, ""); !errors.Is(err, ErrExternalAIPreviewReady) {
		t.Fatalf("preview stops: %v", err)
	}
	if p, ok := run.Preview(); !ok || !strings.Contains(string(p), "mơ hồ") {
		t.Fatalf("preview holds the review: %s", p)
	}
	if fmt.Sprint(f.store.units) != before {
		t.Fatalf("nothing stored in preview mode")
	}
	ctx, _ = previewCtx(ExternalAIPaste, reply)
	st, err := f.desk.Review(ctx, project, "")
	if err != nil || st.Review.OpenCount != 1 {
		t.Fatalf("paste mode still stores: %v", err)
	}
}
