package application_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"orchestrator/internal/application"
)

type fakeUsagePort struct {
	rows []application.LLMUsageRecord
	err  error
}

func (f *fakeUsagePort) RecordLLMUsage(_ context.Context, rec application.LLMUsageRecord) error {
	if f.err != nil {
		return f.err
	}
	f.rows = append(f.rows, rec)
	return nil
}

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// TestRecorder_AFailedWriteNeverReachesTheCaller is CR-027 FR82.5. A Creator
// whose draft came back fine must not be told it failed because a
// bookkeeping row did not land.
func TestRecorder_AFailedWriteNeverReachesTheCaller(t *testing.T) {
	port := &fakeUsagePort{err: errors.New("database on fire")}
	rec := application.NewLLMUsageRecorder(port, quietLogger())

	// The compiler enforces the real guarantee: Record returns nothing at
	// all, so there is no error for a caller to accidentally propagate.
	rec.Record(context.Background(), application.LLMUsageRecord{Provider: "hive"})
}

func TestRecorder_ToleratesNoPortAtAll(t *testing.T) {
	rec := application.NewLLMUsageRecorder(nil, quietLogger())
	rec.Record(context.Background(), application.LLMUsageRecord{Provider: "hive"})

	var nilRecorder *application.LLMUsageRecorder
	nilRecorder.Record(context.Background(), application.LLMUsageRecord{})
}

func TestRecorder_WritesTheRowThrough(t *testing.T) {
	port := &fakeUsagePort{}
	rec := application.NewLLMUsageRecorder(port, quietLogger())

	rec.Record(context.Background(), application.LLMUsageRecord{
		Provider: "hive", Model: "m", Step: "code", PromptTokens: 10,
	})

	if len(port.rows) != 1 || port.rows[0].Step != "code" || port.rows[0].PromptTokens != 10 {
		t.Fatalf("row not written through: %+v", port.rows)
	}
}

func TestRecordFor_SuccessfulCall(t *testing.T) {
	started := time.Now().Add(-1500 * time.Millisecond)
	usage := application.TokenUsage{
		Model:        "deepseek-ai/deepseek-v4.1-flash",
		PromptTokens: 31, CompletionTokens: 49, CachedTokens: 20,
	}

	rec := application.RecordFor("hive", "manim_engineer", "code", "p1", usage, started, nil)

	if !rec.OK || rec.ErrorKind != "" {
		t.Fatalf("expected a successful row, got ok=%v kind=%q", rec.OK, rec.ErrorKind)
	}
	if rec.Model != usage.Model || rec.PromptTokens != 31 || rec.CachedTokens != 20 {
		t.Fatalf("usage did not carry over: %+v", rec)
	}
	if rec.Duration < time.Second {
		t.Fatalf("duration not measured, got %v", rec.Duration)
	}
}

// TestRecordFor_AFailedCallStillRecordsWhatItCost — a response truncated at
// max_tokens is billed in full. Recording zero there would understate spend
// by exactly the calls that waste the most of it.
func TestRecordFor_AFailedCallStillRecordsWhatItCost(t *testing.T) {
	err := &application.LLMError{
		Kind:     application.ErrKindTruncated,
		Provider: "hive",
		Usage: application.TokenUsage{
			Model: "m", PromptTokens: 500, CompletionTokens: 16000, ReasoningTokens: 900,
		},
		Err: errors.New("cut off"),
	}

	// ChatResult is empty on failure, so the caller passes a zero usage —
	// the billed numbers must come from the error itself.
	rec := application.RecordFor("hive", "manim_engineer", "code", "p1",
		application.TokenUsage{}, time.Now(), err)

	if rec.OK {
		t.Fatal("expected a failed row")
	}
	if rec.ErrorKind != application.ErrKindTruncated {
		t.Fatalf("want kind %q, got %q", application.ErrKindTruncated, rec.ErrorKind)
	}
	if rec.PromptTokens != 500 || rec.CompletionTokens != 16000 || rec.ReasoningTokens != 900 {
		t.Fatalf("billed tokens were dropped: %+v", rec)
	}
	if rec.Model != "m" {
		t.Fatalf("model was dropped: %+v", rec)
	}
}

func TestRecordFor_NonLLMErrorStillMarksTheRowFailed(t *testing.T) {
	rec := application.RecordFor("hive", "", "", "", application.TokenUsage{},
		time.Now(), errors.New("something else went wrong"))

	if rec.OK {
		t.Fatal("expected a failed row")
	}
	if rec.ErrorKind != "" {
		t.Fatalf("a non-LLM error has no kind, got %q", rec.ErrorKind)
	}
}
