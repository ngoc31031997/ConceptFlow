package application

import (
	"context"
	"time"
)

// ProjectError is one failure worth tracing back later, appended to the
// project's project_errors column. It is a log, not state: nothing reads it to
// decide what to do next, so a new field here can never change behaviour —
// which is what makes it safe to record generously.
type ProjectError struct {
	At time.Time `json:"at"`
	// Source names the subsystem that failed ("authoring" today), Step the
	// unit inside it ("story" | "storyboard" | "code").
	Source string `json:"source"`
	Step   string `json:"step,omitempty"`
	// Kind is the LLMErrorKind when the provider failed, "" otherwise.
	Kind     string `json:"kind,omitempty"`
	Provider string `json:"provider,omitempty"`
	Message  string `json:"message"`
	// Detail is the raw error chain, for the developer — Message is what the
	// Creator was told.
	Detail string `json:"detail,omitempty"`
	// PartialChars is how much of a cut-off answer had arrived, and
	PartialChars   int             `json:"partial_chars,omitempty"`
	Usage          *TokenUsageJSON `json:"usage,omitempty"`
	ElapsedSeconds int             `json:"elapsed_seconds,omitempty"`
}

// TokenUsageJSON is TokenUsage with wire names, kept separate so TokenUsage
// itself stays free of serialisation concerns.
type TokenUsageJSON struct {
	Model            string `json:"model,omitempty"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	ReasoningTokens  int    `json:"reasoning_tokens"`
}

// ProjectErrorLogPort appends ProjectError entries to the project's error log.
// The log lives with the project, in the orchestrator; only appending is
// needed here.
type ProjectErrorLogPort interface {
	AppendProjectError(ctx context.Context, projectID string, e ProjectError) error
}
