package application

import (
	"context"
	"log/slog"
	"time"
)

// LLMUsageRecord is one row of llm_usage: what a single call to a language
// model cost (CR-027 FR82.1). Failed calls are recorded too — a truncated
// answer is billed like any other.
type LLMUsageRecord struct {
	Provider string
	Model    string
	// Role and Step name the authoring pipeline slot this call served
	// (story_architect / storyboard, …). Both empty for calls that belong to
	// no pipeline step, such as suggest-metadata.
	Role string
	Step string
	// Phase breaks one step into its calls (CR-039): "layout", "cast",
	// "chunk", "repair", "storyboard_fix". Empty when the step is one call.
	Phase string
	// ProjectID is empty for calls made before any project exists — a
	// Creator can draft a short script from nothing (CR-026 FR71.1).
	ProjectID        string
	PromptTokens     int
	CompletionTokens int
	ReasoningTokens  int
	CachedTokens     int
	Duration         time.Duration
	OK               bool
	ErrorKind        LLMErrorKind
}

// LLMUsagePort writes usage rows.
type LLMUsagePort interface {
	RecordLLMUsage(ctx context.Context, rec LLMUsageRecord) error
}

// LLMUsageRecorder wraps the port so that measurement can never break the
// feature it measures (CR-027 FR82.5).
//
// A failed insert is logged and dropped. The alternative — returning the
// error to the caller — would mean a Creator whose draft came back perfectly
// well sees "generation failed" because a bookkeeping row did not land. The
// numbers matter, but not that much.
type LLMUsageRecorder struct {
	port   LLMUsagePort
	logger *slog.Logger
}

func NewLLMUsageRecorder(port LLMUsagePort, logger *slog.Logger) *LLMUsageRecorder {
	if logger == nil {
		logger = slog.Default()
	}
	return &LLMUsageRecorder{port: port, logger: logger}
}

// Record persists one call. It never returns an error, and it tolerates a nil
// port so a deployment without the usage table still serves requests.
func (r *LLMUsageRecorder) Record(ctx context.Context, rec LLMUsageRecord) {
	if r == nil || r.port == nil {
		return
	}
	if err := r.port.RecordLLMUsage(ctx, rec); err != nil {
		// Warn, not Error: nothing the Creator did has failed, and the
		// orchestrator's logger only emits Warn and above.
		r.logger.Warn("could not record llm usage",
			"error", err,
			"provider", rec.Provider,
			"model", rec.Model,
			"step", rec.Step,
			"project_id", rec.ProjectID,
		)
	}
}

// RecordFor builds a record from one call's outcome — the shape every caller
// needs, kept in one place so no call site invents its own field mapping.
// err == nil means success; otherwise the LLMError's kind and its billed
// usage are carried over, because a call that failed after the provider did
// work still costs money.
func RecordFor(provider, role, step, projectID string, usage TokenUsage, started time.Time, err error) LLMUsageRecord {
	rec := LLMUsageRecord{
		Provider:         provider,
		Model:            usage.Model,
		Role:             role,
		Step:             step,
		ProjectID:        projectID,
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		ReasoningTokens:  usage.ReasoningTokens,
		CachedTokens:     usage.CachedTokens,
		Duration:         time.Since(started),
		OK:               err == nil,
	}
	if err != nil {
		rec.ErrorKind = LLMErrorKindOf(err)
		// A failure carries its own billed usage (see LLMError.Usage); the
		// caller's ChatResult is empty in that case, so prefer the error's.
		if billed := billedUsage(err); billed.Model != "" || billed.PromptTokens > 0 {
			rec.Model = billed.Model
			rec.PromptTokens = billed.PromptTokens
			rec.CompletionTokens = billed.CompletionTokens
			rec.ReasoningTokens = billed.ReasoningTokens
			rec.CachedTokens = billed.CachedTokens
		}
	}
	return rec
}
