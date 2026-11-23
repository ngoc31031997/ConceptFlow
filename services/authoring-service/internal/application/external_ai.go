package application

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
)

// ExternalAIMode says how one request uses an AI outside the system.
type ExternalAIMode string

const (
	// ExternalAIPrompt builds the prompt of the call and stops before any
	// model is asked; the prompt is read back through ExternalAIRun.Prompt.
	ExternalAIPrompt ExternalAIMode = "prompt"
	// ExternalAIPaste answers the call with the reply the Creator pasted.
	ExternalAIPaste ExternalAIMode = "paste"
	// ExternalAIPreview answers the call with a pasted reply like
	// ExternalAIPaste, and stops the flow once the reply is read and checked,
	// before anything is stored (stopForPreview).
	ExternalAIPreview ExternalAIMode = "preview"
	// ExternalAICapture asks the system's model, keeps its reply, and stops the
	// flow like ExternalAIPreview.
	ExternalAICapture ExternalAIMode = "capture"
)

var (
	// ErrExternalAIPromptBuilt stops a flow once its prompt has been captured.
	// It is not a failure: the HTTP layer answers with the prompt instead.
	ErrExternalAIPromptBuilt = errors.New("prompt đã dựng xong, chưa gọi AI")
	// ErrExternalAISingleTurn is returned when a flow asks for a second model
	// turn in a request that carries one pasted reply.
	ErrExternalAISingleTurn = errors.New("AI ngoài chỉ chạy được một prompt mỗi lượt; chạy từng mục một")
	// ErrExternalAIPreviewReady stops a flow in preview or capture mode once the
	// reply is read; what it would store is read back through ExternalAIRun.Preview.
	ErrExternalAIPreviewReady = errors.New("kết quả đã đọc xong, chưa lưu")
)

// ExternalAIRun is the external-AI state of one request. It lets exactly one
// model call through and remembers the prompt of the first.
type ExternalAIRun struct {
	mode     ExternalAIMode
	reply    string
	taken    atomic.Bool
	prompt   atomic.Pointer[string]
	captured atomic.Pointer[string]
	preview  atomic.Pointer[json.RawMessage]
}

// Captured is the system model's reply kept in ExternalAICapture mode, and
// whether the model answered.
func (r *ExternalAIRun) Captured() (string, bool) {
	c := r.captured.Load()
	if c == nil {
		return "", false
	}
	return *c, true
}

// Preview is what the flow would have stored, as JSON, once it stopped in
// ExternalAIPreview or ExternalAICapture mode.
func (r *ExternalAIRun) Preview() (json.RawMessage, bool) {
	p := r.preview.Load()
	if p == nil {
		return nil, false
	}
	return *p, true
}

// stopForPreview is called by a flow right after the reply is read and checked
// and before anything is stored. In preview or capture mode it keeps v as the
// preview and stops the flow with ErrExternalAIPreviewReady; otherwise it does
// nothing. See ADR-0052.
func stopForPreview(ctx context.Context, v any) error {
	run := externalAIFrom(ctx)
	if run == nil || (run.mode != ExternalAIPreview && run.mode != ExternalAICapture) {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	raw := json.RawMessage(b)
	run.preview.Store(&raw)
	return ErrExternalAIPreviewReady
}

// NewExternalAIRun starts the external-AI state of a request. reply is the
// pasted text and only matters in ExternalAIPaste mode.
func NewExternalAIRun(mode ExternalAIMode, reply string) *ExternalAIRun {
	return &ExternalAIRun{mode: mode, reply: reply}
}

// Prompt is the prompt captured in ExternalAIPrompt mode, and whether one was.
func (r *ExternalAIRun) Prompt() (string, bool) {
	p := r.prompt.Load()
	if p == nil {
		return "", false
	}
	return *p, true
}

type externalAICtxKey struct{}

// WithExternalAI marks ctx so that the model calls made under it go through run.
func WithExternalAI(ctx context.Context, run *ExternalAIRun) context.Context {
	return context.WithValue(ctx, externalAICtxKey{}, run)
}

// externalAIFrom returns the run WithExternalAI attached, or nil.
func externalAIFrom(ctx context.Context) *ExternalAIRun {
	run, _ := ctx.Value(externalAICtxKey{}).(*ExternalAIRun)
	return run
}

// UsingExternalAI reports whether the calls under ctx go to an AI outside the
// system. Flows that would ask the model again after an unreadable reply do not
// under it: the Creator pastes again instead.
func UsingExternalAI(ctx context.Context) bool { return externalAIFrom(ctx) != nil }

// ExternalPrompt is the prompt of req as one block to paste into another AI:
// the system prompt, a blank line, then the user turn.
func ExternalPrompt(req ChatRequest) string {
	var parts []string
	for _, p := range []string{req.System, req.User} {
		if p = strings.TrimSpace(p); p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, "\n\n")
}

// ExternalReplier turns a pasted reply into the result of a chat turn, and logs
// it like a model call.
type ExternalReplier interface {
	ChatExternal(ctx context.Context, req ChatRequest, reply string) (ChatResult, error)
}

// ExternalAIProvider is an LLMProviderPort that sends calls to the model, except
// under a context that carries an ExternalAIRun: then it builds the prompt or
// takes the pasted reply, and never asks a model.
type ExternalAIProvider struct {
	model   LLMProviderPort
	replier ExternalReplier
}

// NewExternalAIProvider wraps model; replier reads pasted replies.
func NewExternalAIProvider(model LLMProviderPort, replier ExternalReplier) *ExternalAIProvider {
	return &ExternalAIProvider{model: model, replier: replier}
}

// Name is the wrapped provider's name.
func (p *ExternalAIProvider) Name() string { return p.model.Name() }

// Ready is the wrapped provider's readiness; an external-AI request does not
// need it (see requireProvider).
func (p *ExternalAIProvider) Ready() bool {
	r, ok := p.model.(interface{ Ready() bool })
	return !ok || r.Ready()
}

// Chat runs one turn. Under an ExternalAIRun only the first turn is served; a
// later one fails with ErrExternalAISingleTurn.
func (p *ExternalAIProvider) Chat(ctx context.Context, req ChatRequest) (ChatResult, error) {
	run := externalAIFrom(ctx)
	if run == nil {
		return p.model.Chat(ctx, req)
	}
	if !run.taken.CompareAndSwap(false, true) {
		return ChatResult{}, ErrExternalAISingleTurn
	}
	prompt := ExternalPrompt(req)
	run.prompt.Store(&prompt)
	switch run.mode {
	case ExternalAIPrompt:
		return ChatResult{}, ErrExternalAIPromptBuilt
	case ExternalAICapture:
		res, err := p.model.Chat(ctx, req)
		if strings.TrimSpace(res.Content) != "" {
			content := res.Content
			run.captured.Store(&content)
		}
		return res, err
	}
	return p.replier.ChatExternal(ctx, req, run.reply)
}

// requireProvider is the check every step makes before asking the model: a
// provider must exist and be ready. An external-AI request needs neither a key
// nor a model, only the provider that carries it.
func requireProvider(ctx context.Context, provider LLMProviderPort) error {
	if provider == nil {
		return ErrLLMNotConfigured
	}
	// Capture mode asks the system's model, so it needs it ready like a plain call.
	if run := externalAIFrom(ctx); run != nil && run.mode != ExternalAICapture {
		return nil
	}
	if r, ok := provider.(interface{ Ready() bool }); ok && !r.Ready() {
		return ErrLLMNotConfigured
	}
	return nil
}
