// Package llm — this file implements application.LLMProviderPort against
// Hive's OpenAI-compatible chat-completions API (CR-027 FR76.2).
//
// No SDK. The wire format is two JSON structs and a bearer header; pulling in
// an OpenAI client library to save forty lines would buy a dependency whose
// own idea of the API drifts independently of Hive's.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"orchestrator/internal/application"
)

// HiveClient calls POST {baseURL}/chat/completions.
type HiveClient struct {
	baseURL    string
	apiKey     string
	model      string
	maxRetries int
	httpClient *http.Client
	sleep      func(time.Duration) // swapped out in tests
}

func NewHiveClient(baseURL, apiKey, model string, timeout time.Duration, maxRetries int) *HiveClient {
	return &HiveClient{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		apiKey:     apiKey,
		model:      model,
		maxRetries: maxRetries,
		httpClient: &http.Client{Timeout: timeout},
		sleep:      time.Sleep,
	}
}

func (c *HiveClient) Name() string { return "hive" }

type hiveMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	// ReasoningContent is what a reasoning model thought before answering.
	// Never shown to the Creator and never saved — it is read only so an
	// empty Content can be explained (see D13 / ErrKindBudget).
	ReasoningContent string `json:"reasoning_content"`
}

type hiveRequest struct {
	Model       string        `json:"model"`
	Messages    []hiveMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
}

type hiveChoice struct {
	Message      hiveMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// hiveUsage accepts BOTH shapes the two documented models actually return
// (measured 2026-09-21, CR-027 D12):
//
//	glm-5.3-flash       reasoning_tokens at the TOP level, prompt_tokens_details null
//	deepseek-v4.1-flash reasoning_tokens NESTED in completion_tokens_details,
//	                    plus prompt_tokens_details.cached_tokens
//
// Reading only one shape silently records zero for the other model, which is
// worse than not measuring at all: the usage screen would look fine and be
// wrong.
type hiveUsage struct {
	PromptTokens      int `json:"prompt_tokens"`
	CompletionTokens  int `json:"completion_tokens"`
	ReasoningTokens   int `json:"reasoning_tokens"`
	CompletionDetails *struct {
		ReasoningTokens int `json:"reasoning_tokens"`
	} `json:"completion_tokens_details"`
	PromptDetails *struct {
		CachedTokens int `json:"cached_tokens"`
	} `json:"prompt_tokens_details"`
}

func (u hiveUsage) reasoning() int {
	if u.ReasoningTokens > 0 {
		return u.ReasoningTokens
	}
	if u.CompletionDetails != nil {
		return u.CompletionDetails.ReasoningTokens
	}
	return 0
}

func (u hiveUsage) cached() int {
	if u.PromptDetails != nil {
		return u.PromptDetails.CachedTokens
	}
	return 0
}

type hiveResponse struct {
	Model   string       `json:"model"`
	Choices []hiveChoice `json:"choices"`
	Usage   hiveUsage    `json:"usage"`
}

// Chat runs one completion, retrying only the failures that retrying can fix.
func (c *HiveClient) Chat(ctx context.Context, req application.ChatRequest) (application.ChatResult, error) {
	if c.apiKey == "" {
		return application.ChatResult{}, &application.LLMError{
			Kind: application.ErrKindNotConfigured, Provider: c.Name(),
			Err: errors.New("HIVE_API_KEY is not set"),
		}
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			c.sleep(backoff(attempt))
		}
		result, err := c.chatOnce(ctx, req)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if !retryable(err) || ctx.Err() != nil {
			break
		}
	}
	return application.ChatResult{}, lastErr
}

// retryable: throttling and provider-side faults clear on their own; a bad
// key, an empty wallet or an exhausted token budget will answer identically
// however many times we ask.
func retryable(err error) bool {
	switch application.LLMErrorKindOf(err) {
	case application.ErrKindRateLimit, application.ErrKindServer:
		return true
	default:
		return false
	}
}

// backoff: 1s, 2s, 4s… plus jitter. Hive's documented default ceiling is 5
// requests per second, so retries that all fire on the same beat would keep
// re-colliding. Same shape as the AzureTTSAdapter backoff from CR-013.
func backoff(attempt int) time.Duration {
	base := time.Second << (attempt - 1)
	return base + time.Duration(rand.Int63n(int64(250*time.Millisecond)))
}

func (c *HiveClient) chatOnce(ctx context.Context, req application.ChatRequest) (application.ChatResult, error) {
	fail := func(kind application.LLMErrorKind, usage application.TokenUsage, err error) (application.ChatResult, error) {
		return application.ChatResult{}, &application.LLMError{
			Kind: kind, Provider: c.Name(), Usage: usage, Err: err,
		}
	}

	messages := make([]hiveMessage, 0, 2)
	if strings.TrimSpace(req.System) != "" {
		messages = append(messages, hiveMessage{Role: "system", Content: req.System})
	}
	messages = append(messages, hiveMessage{Role: "user", Content: req.User})

	body, err := json.Marshal(hiveRequest{
		Model: c.model, Messages: messages,
		MaxTokens: req.MaxTokens, Temperature: req.Temperature,
	})
	if err != nil {
		return fail(application.ErrKindMalformed, application.TokenUsage{}, fmt.Errorf("marshal request: %w", err))
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return fail(application.ErrKindMalformed, application.TokenUsage{}, fmt.Errorf("build request: %w", err))
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		// A client-side deadline and a dead network arrive as the same error
		// type; the Creator-facing advice differs, so split them.
		kind := application.ErrKindServer
		if errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "Client.Timeout") {
			kind = application.ErrKindTimeout
		}
		return fail(kind, application.TokenUsage{}, fmt.Errorf("call hive: %w", err))
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fail(application.ErrKindServer, application.TokenUsage{}, fmt.Errorf("read response: %w", err))
	}

	if resp.StatusCode != http.StatusOK {
		return fail(kindForStatus(resp.StatusCode), application.TokenUsage{},
			fmt.Errorf("hive returned %d: %s", resp.StatusCode, snippet(raw)))
	}

	var parsed hiveResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return fail(application.ErrKindMalformed, application.TokenUsage{},
			fmt.Errorf("decode response: %w", err))
	}

	model := parsed.Model
	if model == "" {
		model = c.model
	}
	usage := application.TokenUsage{
		Model:            model,
		PromptTokens:     parsed.Usage.PromptTokens,
		CompletionTokens: parsed.Usage.CompletionTokens,
		ReasoningTokens:  parsed.Usage.reasoning(),
		CachedTokens:     parsed.Usage.cached(),
	}

	if len(parsed.Choices) == 0 {
		return fail(application.ErrKindEmpty, usage, errors.New("no choices in response"))
	}
	choice := parsed.Choices[0]
	content := strings.TrimSpace(choice.Message.Content)

	// D13 — "empty" and "ran out of room" are different problems with
	// different fixes, and a reasoning model turns the second into the first.
	if choice.FinishReason == "length" {
		if content == "" {
			return fail(application.ErrKindBudget, usage, fmt.Errorf(
				"the whole token budget went on reasoning before any answer was written (%d reasoning tokens)",
				usage.ReasoningTokens))
		}
		return fail(application.ErrKindTruncated, usage, errors.New("response was cut off at max_tokens"))
	}
	if content == "" {
		return fail(application.ErrKindEmpty, usage, errors.New("model returned empty content"))
	}

	return application.ChatResult{Content: content, Usage: usage}, nil
}

func kindForStatus(status int) application.LLMErrorKind {
	switch {
	case status == http.StatusUnauthorized, status == http.StatusForbidden:
		return application.ErrKindAuth
	case status == http.StatusPaymentRequired:
		return application.ErrKindBalance
	case status == http.StatusTooManyRequests:
		return application.ErrKindRateLimit
	case status >= 500:
		return application.ErrKindServer
	default:
		return application.ErrKindMalformed
	}
}

// snippet keeps a provider error body short enough to log without dumping an
// HTML error page into the logs.
func snippet(body []byte) string {
	const max = 300
	s := strings.TrimSpace(string(body))
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}
