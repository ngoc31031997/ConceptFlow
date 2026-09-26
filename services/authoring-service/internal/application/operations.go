package application

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Operation is what the GUI polls for a long call (CR-040 FR116.3): where the
// model is, how much it has written and how long it has taken. `Done`/`Total`
// are nil when the total is unknown, so the GUI never invents a percentage
// (FR116.4).
type Operation struct {
	Kind           string `json:"kind"`
	Phase          string `json:"phase"` // "waiting" | "reasoning" | "writing"
	ReasoningChars int    `json:"reasoning_chars"`
	ContentChars   int    `json:"content_chars"`
	Done           *int   `json:"done"`
	Total          *int   `json:"total"`
	ElapsedMs      int64  `json:"elapsed_ms"`
	Status         string `json:"status"` // "running" | "succeeded" | "failed"
	Error          string `json:"error,omitempty"`

	started  time.Time
	finished time.Time
}

// operationRetention is how long a finished operation stays readable, so the
// last poll after completion still sees the outcome and the error's counters.
const operationRetention = 10 * time.Minute

// Operations is an in-memory registry of long calls. It is best-effort by
// design: losing it on a restart only loses a progress bar.
type Operations struct {
	mu  sync.Mutex
	ops map[string]*Operation
	now func() time.Time
}

func NewOperations() *Operations {
	return &Operations{ops: map[string]*Operation{}, now: time.Now}
}

// Start registers id as running. The caller picks the id (the GUI sends one
// with the request), so it can poll before the call has returned.
func (o *Operations) Start(id, kind string) {
	if id == "" {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.gcLocked()
	o.ops[id] = &Operation{Kind: kind, Phase: "waiting", Status: "running", started: o.now()}
}

// Progress records the streamed size of the reply.
func (o *Operations) Progress(id string, p ChatProgress) {
	o.mu.Lock()
	defer o.mu.Unlock()
	op := o.ops[id]
	if op == nil || op.Status != "running" {
		return
	}
	op.ReasoningChars, op.ContentChars = p.ReasoningChars, p.ContentChars
	if p.ContentChars > 0 {
		op.Phase = "writing"
	} else if p.ReasoningChars > 0 {
		op.Phase = "reasoning"
	}
}

// Finish closes the operation; a non-nil err marks it failed and keeps the
// counters and elapsed time already reached (FR116.5).
func (o *Operations) Finish(id string, err error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	op := o.ops[id]
	if op == nil {
		return
	}
	op.finished = o.now()
	if err != nil {
		op.Status, op.Error = "failed", operationErrorKind(err)
		return
	}
	op.Status = "succeeded"
}

// Get returns a snapshot of the operation, or false when it is unknown.
func (o *Operations) Get(id string) (Operation, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	op := o.ops[id]
	if op == nil {
		return Operation{}, false
	}
	out := *op
	end := o.now()
	if !op.finished.IsZero() {
		end = op.finished
	}
	out.ElapsedMs = end.Sub(op.started).Milliseconds()
	return out, true
}

func (o *Operations) gcLocked() {
	cutoff := o.now().Add(-operationRetention)
	for id, op := range o.ops {
		if !op.finished.IsZero() && op.finished.Before(cutoff) {
			delete(o.ops, id)
		}
	}
}

// operationErrorKind is the classified error the GUI shows: the LLM error kind
// (`balance`, `budget`, `server`...) when there is one, else `server`.
func operationErrorKind(err error) string {
	var llmErr *LLMError
	if errors.As(err, &llmErr) {
		return string(llmErr.Kind)
	}
	return string(ErrKindServer)
}

type progressCtxKey struct{}

// WithProgress attaches a streaming-progress callback to ctx; adapters that
// stream (the llm-service client) report the reply's size through it.
func WithProgress(ctx context.Context, fn func(ChatProgress)) context.Context {
	return context.WithValue(ctx, progressCtxKey{}, fn)
}

// ProgressFrom returns the callback WithProgress attached, or nil.
func ProgressFrom(ctx context.Context) func(ChatProgress) {
	fn, _ := ctx.Value(progressCtxKey{}).(func(ChatProgress))
	return fn
}
