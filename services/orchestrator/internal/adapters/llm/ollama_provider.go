package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"orchestrator/internal/application"
)

// OllamaProvider adapts the existing OllamaClient to
// application.LLMProviderPort (CR-027 FR76.3).
//
// A wrapper rather than new methods on OllamaClient: suggest-metadata
// (CR-014) and suggest-short-script (CR-026) are working, shipped paths, and
// CR-027 has no business changing their signatures to add a second caller.
//
// Ollama is the fallback for the light tasks only. It is deliberately NOT a
// fallback for the authoring pipeline: those prompts run 10-14k characters
// against a model whose default num_ctx is 2048, and CR-014 measured what
// that produces — a well-formed, empty answer. Degrading silently to a
// useless draft is worse than an error that says what happened.
type OllamaProvider struct {
	client *OllamaClient
}

func NewOllamaProvider(client *OllamaClient) *OllamaProvider {
	return &OllamaProvider{client: client}
}

func (p *OllamaProvider) Name() string { return "ollama" }

// Chat concatenates system and user into Ollama's single prompt field —
// /api/generate has no role structure.
//
// Usage comes back zeroed: Ollama reports no token counts. Zero here means
// "not reported", and the usage screen says so in words rather than drawing
// a misleading 0 next to Hive's real numbers (CR-027 FR82.3).
func (p *OllamaProvider) Chat(ctx context.Context, req application.ChatRequest) (application.ChatResult, error) {
	fail := func(kind application.LLMErrorKind, err error) (application.ChatResult, error) {
		return application.ChatResult{}, &application.LLMError{
			Kind: kind, Provider: p.Name(), Err: err,
		}
	}

	prompt := req.User
	if strings.TrimSpace(req.System) != "" {
		prompt = req.System + "\n\n" + req.User
	}

	body, err := json.Marshal(generateRequest{Model: p.client.model, Prompt: prompt, Stream: false})
	if err != nil {
		return fail(application.ErrKindMalformed, fmt.Errorf("marshal ollama request: %w", err))
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.client.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return fail(application.ErrKindMalformed, fmt.Errorf("build ollama request: %w", err))
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.httpClient.Do(httpReq)
	if err != nil {
		kind := application.ErrKindServer
		if errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "Client.Timeout") {
			kind = application.ErrKindTimeout
		}
		return fail(kind, fmt.Errorf("call ollama: %w", err))
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fail(application.ErrKindServer, fmt.Errorf("read ollama response: %w", err))
	}
	if resp.StatusCode != http.StatusOK {
		return fail(kindForStatus(resp.StatusCode),
			fmt.Errorf("ollama returned %d: %s", resp.StatusCode, snippet(raw)))
	}

	var parsed generateResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return fail(application.ErrKindMalformed, fmt.Errorf("decode ollama response: %w", err))
	}
	content := strings.TrimSpace(parsed.Response)
	if content == "" {
		return fail(application.ErrKindEmpty, errors.New("ollama returned empty content"))
	}

	return application.ChatResult{
		Content: content,
		Usage:   application.TokenUsage{Model: p.client.model},
	}, nil
}
