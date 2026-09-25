package application

import (
	"context"
	"errors"
	"fmt"
)

// LLMProviderPort is the one way the application talks to a large language
// model (CR-027 FR76.1). Two adapters implement it: the Hive client, and a
// thin wrapper over the existing Ollama client.
//
// Deliberately narrower than either adapter's own surface. The CR-014/CR-026
// use cases keep their task-shaped ports (MetadataSuggesterPort,
// ShortScriptSuggesterPort) with their own prompts and response shapes; this
// one carries a prompt and hands back text, because the authoring pipeline's
// prompts already live in the database and the caller assembles them.
type LLMProviderPort interface {
	// Name identifies the provider in llm_usage rows and error messages.
	Name() string
	Chat(ctx context.Context, req ChatRequest) (ChatResult, error)
}

// ChatRequest is one turn: a system prompt and a user prompt.
type ChatRequest struct {
	System string
	User   string
	// MaxTokens caps the response. Generous by default — a reasoning model
	// spends part of this budget thinking before it writes a single
	// character of the answer (CR-027 D13).
	MaxTokens   int
	Temperature float64
	// Model overrides the adapter's own configured model (HIVE_MODEL) for
	// this one call — the model-per-step picker at wizard step 1 (follow-up
	// to CR-027). "" means "use the adapter's default", so every caller that
	// predates this field, and every project that never touched the picker,
	// keeps behaving exactly as before.
	Model string
	// OnProgress, when set, is called as the reply streams in with running
	// totals — a provider that cannot stream simply never calls it. It runs on
	// the calling goroutine, so it must return quickly.
	OnProgress func(ChatProgress)
}

// ChatProgress is the running size of a streaming reply. Characters, not
// tokens: usage only arrives with the final chunk, but a growing count is
// enough to show the run is alive and which phase it is in.
type ChatProgress struct {
	ReasoningChars int
	ContentChars   int
}

// ChatResult is what came back, plus what it cost.
type ChatResult struct {
	Content string
	Usage   TokenUsage
}

// TokenUsage is the measured cost of one call, written to llm_usage so the
// Creator can see spend in the web GUI instead of on a provider dashboard
// (CR-027 FR82).
//
// ReasoningTokens is broken out rather than folded into CompletionTokens
// because it is not free: glm-5.3-flash spent 66 of 122 completion tokens
// reasoning before answering a one-sentence question. A usage screen that
// hides that cannot explain why one model costs twice another for the same
// visible output.
//
// CachedTokens is the part of the prompt the provider served from its cache.
// Ollama reports neither, and zeroes there mean "not reported", which the
// usage screen says in words rather than drawing as 0.
type TokenUsage struct {
	Model            string
	PromptTokens     int
	CompletionTokens int
	ReasoningTokens  int
	CachedTokens     int
}

// LLMErrorKind classifies a failed call so the GUI can tell the Creator what
// to actually do about it (CR-027 FR76.6 / D11). Lumping these together as
// "the AI failed" sends them looking in the wrong place — the fix for a dead
// key and the fix for an exhausted token budget have nothing in common.
type LLMErrorKind string

const (
	// ErrKindAuth — key missing, wrong, or revoked.
	ErrKindAuth LLMErrorKind = "auth"
	// ErrKindBalance — account out of credit.
	ErrKindBalance LLMErrorKind = "balance"
	// ErrKindRateLimit — provider throttled us and retries did not clear it.
	ErrKindRateLimit LLMErrorKind = "rate_limit"
	// ErrKindServer — provider-side 5xx that outlived its retries.
	ErrKindServer LLMErrorKind = "server"
	// ErrKindTimeout — no answer inside the configured timeout.
	ErrKindTimeout LLMErrorKind = "timeout"
	// ErrKindBudget — stopped on length with NOTHING written. On a reasoning
	// model this means the whole allowance went on thinking (CR-027 D13):
	// measured, glm-5.3-flash with max_tokens=5 returned empty content and a
	// filled reasoning_content. The fix is a bigger budget, not a new prompt.
	ErrKindBudget LLMErrorKind = "budget"
	// ErrKindTruncated — stopped on length mid-answer. Same fix as budget.
	ErrKindTruncated LLMErrorKind = "truncated"
	// ErrKindEmpty — finished cleanly and still said nothing. A real model or
	// prompt problem, unlike budget/truncated.
	ErrKindEmpty LLMErrorKind = "empty"
	// ErrKindMalformed — the response was not the JSON we expect.
	ErrKindMalformed LLMErrorKind = "malformed"
	// ErrKindNotConfigured — no provider is available (no API key).
	ErrKindNotConfigured LLMErrorKind = "not_configured"
)

// LLMError carries the kind alongside the underlying cause.
type LLMError struct {
	Kind     LLMErrorKind
	Provider string
	// Usage is filled in when the provider billed us despite the failure —
	// a truncated answer still costs tokens, and llm_usage must record it.
	Usage TokenUsage
	// Partial is the text that did arrive before a ErrKindTruncated cut-off.
	// Empty for every other kind.
	Partial string
	// Diag is the provider's own account of the failed call — HTTP status,
	// request id, finish_reason, the response body — for the error log. Not
	// shown to the Creator.
	Diag string
	Err  error
}

func (e *LLMError) Error() string {
	return fmt.Sprintf("%s: %s: %v", e.Provider, e.Kind, e.Err)
}

func (e *LLMError) Unwrap() error { return e.Err }

// billedUsage returns the tokens a failed call was still charged for, or a
// zero value when err carries none.
func billedUsage(err error) TokenUsage {
	var llmErr *LLMError
	if errors.As(err, &llmErr) {
		return llmErr.Usage
	}
	return TokenUsage{}
}

// LLMErrorKindOf reports the kind of err, or "" if err is not an LLMError.
func LLMErrorKindOf(err error) LLMErrorKind {
	var llmErr *LLMError
	if errors.As(err, &llmErr) {
		return llmErr.Kind
	}
	return ""
}

// partialOf returns the text a truncated call managed to write, or "".
func partialOf(err error) string {
	var llmErr *LLMError
	if errors.As(err, &llmErr) {
		return llmErr.Partial
	}
	return ""
}
