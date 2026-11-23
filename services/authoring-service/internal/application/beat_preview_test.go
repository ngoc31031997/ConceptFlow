package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"authoring/internal/domain"
)

const (
	beatRedoReply = `{"beats":[{"key":"b2","fn":"Cao trào","title":"Viết lại","goal":"G","content":"C","facts":["F02"],"dur":40,"next":"N"}]}`
	beatAddReply  = `{"beats":[{"key":"","fn":"Cao trào","title":"Mới","goal":"G","content":"C","facts":[],"dur":20,"next":"N"}]}`
)

// previewCtx is a request that reads the AI reply and stops before storing it.
func previewCtx() context.Context {
	return WithExternalAI(context.Background(), NewExternalAIRun(ExternalAIPreview, ""))
}

func TestRedoInPreviewModeStopsBeforeStoringAndHoldsTheBeat(t *testing.T) {
	f := newBeatFixture()
	f.generate(t)
	_, before := f.beat(t, "b2")
	f.llm.say(beatRedoReply)
	ctx := previewCtx()
	err := f.uc.Redo(ctx, "p1", "b2", "ngắn hơn")
	if !errors.Is(err, ErrExternalAIPreviewReady) {
		t.Fatalf("Redo = %v, want ErrExternalAIPreviewReady", err)
	}
	raw, ok := externalAIFrom(ctx).Preview()
	if !ok {
		t.Fatal("no preview held")
	}
	var got struct {
		Key  string      `json:"key"`
		Beat domain.Beat `json:"beat"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got.Key != "b2" || got.Beat.Title != "Viết lại" || got.Beat.Fn != before.Fn {
		t.Errorf("preview = %s, want b2 with the new title and the skeleton's function", raw)
	}
	unit, after := f.beat(t, "b2")
	if after.Title != before.Title || unit.RedoCount != 0 {
		t.Errorf("stored beat changed in preview mode: %+v redo=%d", after, unit.RedoCount)
	}
}

func TestRedoInPasteModeStillStores(t *testing.T) {
	f := newBeatFixture()
	f.generate(t)
	f.llm.say(beatRedoReply)
	ctx := WithExternalAI(context.Background(), NewExternalAIRun(ExternalAIPaste, ""))
	if err := f.uc.Redo(ctx, "p1", "b2", ""); err != nil {
		t.Fatalf("Redo: %v", err)
	}
	if _, b := f.beat(t, "b2"); b.Title != "Viết lại" {
		t.Errorf("title = %q, want the reply stored", b.Title)
	}
}

func TestAddInPreviewModeStopsBeforeStoringAndHoldsTheBeat(t *testing.T) {
	f := newBeatFixture()
	f.generate(t)
	f.llm.say(beatAddReply)
	ctx := previewCtx()
	_, err := f.uc.Add(ctx, "p1", "b1", false, "")
	if !errors.Is(err, ErrExternalAIPreviewReady) {
		t.Fatalf("Add = %v, want ErrExternalAIPreviewReady", err)
	}
	raw, ok := externalAIFrom(ctx).Preview()
	if !ok || !strings.Contains(string(raw), `"after":"b1"`) || !strings.Contains(string(raw), `"title":"Mới"`) {
		t.Errorf("preview = %s, want the new beat behind b1", raw)
	}
	if state, _ := f.uc.State(context.Background(), "p1"); len(state.Beats) != 3 {
		t.Errorf("beats = %d, want 3: nothing stored in preview mode", len(state.Beats))
	}
}

func TestAddInPasteModeStillStores(t *testing.T) {
	f := newBeatFixture()
	f.generate(t)
	f.llm.say(beatAddReply)
	ctx := WithExternalAI(context.Background(), NewExternalAIRun(ExternalAIPaste, ""))
	if key, err := f.uc.Add(ctx, "p1", "b1", false, ""); err != nil || key != "b4" {
		t.Fatalf("Add = %q, %v, want b4", key, err)
	}
}
