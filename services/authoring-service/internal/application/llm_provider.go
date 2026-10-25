package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// ErrLLMNotConfigured means no API key is set, so asking a model is not
// possible for this deployment.
var ErrLLMNotConfigured = errors.New("chưa cấu hình API key cho nhà cung cấp AI")

// userTurnNudge is the one-line nudge that follows the system prompt. Some
// providers answer an empty user turn with a question back.
const userTurnNudge = "Hãy thực hiện nhiệm vụ trên và chỉ trả về kết quả theo đúng định dạng đã mô tả."

// LLMProviderPort is the one way the application talks to a large language
// model. One adapter implements it: the llm-service client,
// which is the only thing that talks to Hive or Ollama.
//
// Deliberately narrower than the adapter's own surface. The metadata use case
// keeps its task-shaped port (MetadataSuggesterPort) with its own prompt and
// response shape; this
// one carries a prompt and hands back text, because the authoring pipeline's
// prompts already live in the database and the caller assembles them.
type LLMProviderPort interface {
	// Name identifies the provider in error messages and the GUI status.
	Name() string
	Chat(ctx context.Context, req ChatRequest) (ChatResult, error)
}

// ChatRequest is one turn: a system prompt and a user prompt.
type ChatRequest struct {
	System string
	User   string
	// MaxTokens caps the response. Generous by default — a reasoning model
	// spends part of this budget thinking before it writes a single
	// character of the answer.
	MaxTokens   int
	Temperature float64
	// Model overrides the adapter's own configured model (HIVE_MODEL) for
	// this one call — the model-per-step picker at wizard step 1.
	// "" means "use the adapter's default", for every caller that does not set
	// it and every project that never touched the picker.
	Model string
	// Where the call comes from, written to llm-service's call log: the
	// project it serves, the role (the step's AI role) and, for a step that
	// calls the model more than once, the step and phase.
	ProjectID string
	Role      string
	Step      string
	Phase     string
	// ExpectJSON asks llm-service for the first JSON object of the reply
	// (one repair turn if it cannot be read); it comes back as ChatResult.JSON.
	ExpectJSON bool
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
	// JSON is the reply's JSON object, set when the request had ExpectJSON.
	JSON  json.RawMessage
	Usage TokenUsage
	// CallID is the llm-service call-log row of this call.
	CallID string
}

// TokenUsage is the measured cost of one call, as llm-service reported it.
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
	// ReasoningChars is the reasoning text llm-service counted on the stream,
	// known even when the provider sent no usage record.
	ReasoningChars int
	// UsageMissing: the provider sent no usage record (a stream cut short, a
	// failed call), so the token counts above are not what was billed.
	UsageMissing bool
}

// LLMErrorKind classifies a failed call so the GUI can tell the Creator what
// to actually do about it. Lumping these together as
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
	// model this means the whole allowance went on thinking:
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
	// a truncated answer still costs tokens, and the cost is still reported.
	Usage TokenUsage
	// Partial is the text that did arrive before a ErrKindTruncated cut-off.
	// Empty for every other kind.
	Partial string
	// Diag is the provider's own account of the failed call — HTTP status,
	// request id, finish_reason, the response body — for the error log. Not
	// shown to the Creator.
	Diag string
	// CallID is the call-log row of the failed call ("" when none was logged).
	CallID string
	Err    error
}

func (e *LLMError) Error() string {
	msg := fmt.Sprintf("%s: %s: %v", e.Provider, e.Kind, e.Err)
	if e.CallID != "" {
		msg += fmt.Sprintf(" (mã cuộc gọi %s)", e.CallID)
	}
	return msg
}

func (e *LLMError) Unwrap() error { return e.Err }

// LLMErrorKindOf reports the kind of err, or "" if err is not an LLMError.
func LLMErrorKindOf(err error) LLMErrorKind {
	var llmErr *LLMError
	if errors.As(err, &llmErr) {
		return llmErr.Kind
	}
	return ""
}
