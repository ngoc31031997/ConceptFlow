package application

import (
	"context"
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
)

var (
	// ErrExternalAIPromptBuilt stops a flow once its prompt has been captured.
	// It is not a failure: the HTTP layer answers with the prompt instead.
	ErrExternalAIPromptBuilt = errors.New("prompt đã dựng xong, chưa gọi AI")
	// ErrExternalAISingleTurn is returned when a flow asks for a second model
	// turn in a request that carries one pasted reply.
	ErrExternalAISingleTurn = errors.New("AI ngoài chỉ chạy được một prompt mỗi lượt; chạy từng mục một")
)

// ExternalAIRun is the external-AI state of one request. It lets exactly one
// model call through and remembers the prompt of the first.
type ExternalAIRun struct {
	mode   ExternalAIMode
	reply  string
	taken  atomic.Bool
	prompt atomic.Pointer[string]
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
	if run.mode == ExternalAIPrompt {
		prompt := ExternalPrompt(req)
		run.prompt.Store(&prompt)
		return ChatResult{}, ErrExternalAIPromptBuilt
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
	if UsingExternalAI(ctx) {
		return nil
	}
	if r, ok := provider.(interface{ Ready() bool }); ok && !r.Ready() {
		return ErrLLMNotConfigured
	}
	return nil
}
