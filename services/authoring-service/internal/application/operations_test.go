package application

import (
	"errors"
	"testing"
	"time"
)

func TestOperations_TracksProgressAndKeepsCountersOnFailure(t *testing.T) {
	ops := NewOperations()
	clock := time.Unix(1000, 0)
	ops.now = func() time.Time { return clock }

	ops.Start("op1", "suggest_metadata")
	if op, _ := ops.Get("op1"); op.Phase != "waiting" || op.Status != "running" || op.Total != nil {
		t.Fatalf("fresh operation should wait with unknown total, got %+v", op)
	}

	clock = clock.Add(5 * time.Second)
	ops.Progress("op1", ChatProgress{ReasoningChars: 300})
	ops.Progress("op1", ChatProgress{ReasoningChars: 300, ContentChars: 40})
	op, _ := ops.Get("op1")
	if op.Phase != "writing" || op.ContentChars != 40 || op.ElapsedMs != 5000 {
		t.Fatalf("unexpected progress %+v", op)
	}

	clock = clock.Add(2 * time.Second)
	ops.Finish("op1", &LLMError{Kind: ErrKindBalance, Err: errors.New("no credit")})
	clock = clock.Add(time.Minute)
	op, _ = ops.Get("op1")
	if op.Status != "failed" || op.Error != "balance" || op.ContentChars != 40 || op.ElapsedMs != 7000 {
		t.Fatalf("failure must keep counters, freeze time and classify the error, got %+v", op)
	}
}

func TestOperations_UnknownAndEmptyID(t *testing.T) {
	ops := NewOperations()
	ops.Start("", "x")
	if _, ok := ops.Get(""); ok {
		t.Fatal("an empty id must not be registered")
	}
	if _, ok := ops.Get("nope"); ok {
		t.Fatal("unknown id must be absent")
	}
}
