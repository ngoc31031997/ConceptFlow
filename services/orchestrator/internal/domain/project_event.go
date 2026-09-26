package domain

import (
	"context"
	"time"
)

// ProjectEvent is one line of a project's journey: it moved to a step, or a
// step started, finished or failed. Append-only and never read to make a
// decision — it is what the "Nhật ký" screen and later optimisation work read.
type ProjectEvent struct {
	ID         int64     `json:"id"`
	ProjectID  string    `json:"project_id"`
	At         time.Time `json:"at"`
	FlowStep   int       `json:"flow_step"`
	StepLabel  string    `json:"step_label"`
	RunState   RunState  `json:"run_state"` // running | done | failed | idle
	Source     string    `json:"source"`    // "authoring" | "saga"
	FromStatus string    `json:"from_status,omitempty"`
	// FromFlowStep is the step the project was leaving. For a saga move,
	// DurationMS is the time spent there, so it is attributed to this step, not
	// to FlowStep (the one entered).
	FromFlowStep int    `json:"from_flow_step,omitempty"`
	ToStatus     string `json:"to_status,omitempty"`
	DurationMS   int64  `json:"duration_ms,omitempty"`
	Detail       string `json:"detail,omitempty"` // error message for failed
	// Authoring runs only.
	ContentChars     int `json:"content_chars,omitempty"`
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
}

// ProjectEventPort appends and lists project events.
type ProjectEventPort interface {
	AppendProjectEvent(ctx context.Context, e ProjectEvent) error
	ListProjectEvents(ctx context.Context, projectID string) ([]ProjectEvent, error)
	// ListRecentProjectEvents is the cross-project feed, newest first.
	ListRecentProjectEvents(ctx context.Context, limit int) ([]ProjectEvent, error)
}
