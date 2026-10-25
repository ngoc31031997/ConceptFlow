// Package llm is the orchestrator's only path to a language model: an HTTP
// client for llm-service, which owns the Hive and Ollama connections
// (through the OpenAI SDK), retries, error classification and the chunked code
// pipeline. Nothing in the orchestrator talks to a provider directly any more.
package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"authoring/internal/application"
	"authoring/internal/domain"
)

// Client implements application.LLMProviderPort and MetadataSuggesterPort.
type Client struct {
	baseURL string
	// timeout bounds a whole call; 0 waits as long as llm-service does (Hive
	// can take minutes on a reasoning model, and HIVE_TIMEOUT_SECONDS=0 means
	// "no timeout" there too).
	timeout time.Duration
	http    *http.Client

	mu          sync.Mutex
	readyAt     time.Time
	readyResult bool
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{baseURL: strings.TrimSuffix(baseURL, "/"), timeout: timeout, http: &http.Client{}}
}

// Name is the provider shown in the GUI status panel: the steps always run on Hive.
func (c *Client) Name() string { return "hive" }

// Ready reports whether the AI path can be offered: llm-service is reachable
// and holds a Hive key. Cached briefly — the GUI polls this.
func (c *Client) Ready() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if time.Since(c.readyAt) < 10*time.Second {
		return c.readyResult
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	ok := false
	if req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil); err == nil {
		if resp, err := c.http.Do(req); err == nil {
			var h struct {
				HiveConfigured bool `json:"hive_configured"`
			}
			if resp.StatusCode == http.StatusOK && json.NewDecoder(resp.Body).Decode(&h) == nil {
				ok = h.HiveConfigured
			}
			resp.Body.Close()
		}
	}
	c.readyAt, c.readyResult = time.Now(), ok
	return ok
}

// --- wire types ---------------------------------------------------------------

type wireUsage struct {
	Model            string `json:"model"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	ReasoningTokens  int    `json:"reasoning_tokens"`
	CachedTokens     int    `json:"cached_tokens"`
	ReasoningChars   int    `json:"reasoning_chars"`
	// Absent means reported.
	UsageReported *bool `json:"usage_reported"`
}

func (u wireUsage) usage() application.TokenUsage {
	return application.TokenUsage{
		Model: u.Model, PromptTokens: u.PromptTokens, CompletionTokens: u.CompletionTokens,
		ReasoningTokens: u.ReasoningTokens, CachedTokens: u.CachedTokens,
		ReasoningChars: u.ReasoningChars, UsageMissing: u.UsageReported != nil && !*u.UsageReported,
	}
}

type wireError struct {
	Kind     string    `json:"kind"`
	Provider string    `json:"provider"`
	Message  string    `json:"message"`
	Usage    wireUsage `json:"usage"`
	Partial  string    `json:"partial"`
	Diag     string    `json:"diag"`
	CallID   string    `json:"call_id"`
}

func (e wireError) llmError() *application.LLMError {
	provider := e.Provider
	if provider == "" {
		provider = "llm-service"
	}
	kind := application.LLMErrorKind(e.Kind)
	if kind == "" {
		kind = application.ErrKindServer
	}
	return &application.LLMError{
		Kind: kind, Provider: provider, Usage: e.Usage.usage(), Partial: e.Partial, Diag: e.Diag,
		CallID: e.CallID, Err: errors.New(e.Message),
	}
}

// errRouteMissing: llm-service answered 404 for the route itself.
var errRouteMissing = errors.New("llm-service has no such route")

func unreachable(err error) *application.LLMError {
	kind := application.ErrKindServer
	if errors.Is(err, context.DeadlineExceeded) {
		kind = application.ErrKindTimeout
	}
	return &application.LLMError{
		Kind: kind, Provider: "llm-service",
		Err: fmt.Errorf("call llm-service: %w", err),
	}
}

func (c *Client) ctx(parent context.Context) (context.Context, context.CancelFunc) {
	if c.timeout > 0 {
		return context.WithTimeout(parent, c.timeout)
	}
	return parent, func() {}
}

// --- streaming ----------------------------------------------------------------

// stream POSTs body to path and reads newline-delimited JSON events. onEvent
// sees every event that is not the final `result`/`error`; the final one is
// returned raw.
func (c *Client) stream(
	ctx context.Context, path string, body any, onEvent func(kind string, raw json.RawMessage),
) (json.RawMessage, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, &application.LLMError{Kind: application.ErrKindMalformed, Provider: "llm-service", Err: fmt.Errorf("marshal request: %w", err)}
	}
	ctx, cancel := c.ctx(ctx)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return nil, &application.LLMError{Kind: application.ErrKindMalformed, Provider: "llm-service", Err: fmt.Errorf("build request: %w", err)}
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, unreachable(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, statusError(resp)
	}

	rd := bufio.NewReaderSize(resp.Body, 64*1024)
	for {
		line, rerr := rd.ReadBytes('\n')
		if trimmed := bytes.TrimSpace(line); len(trimmed) > 0 {
			var head struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(trimmed, &head); err != nil {
				return nil, &application.LLMError{Kind: application.ErrKindMalformed, Provider: "llm-service",
					Err: fmt.Errorf("unreadable event from llm-service: %w", err)}
			}
			switch head.Type {
			case "result":
				return json.RawMessage(trimmed), nil
			case "error":
				var ev struct {
					Error wireError `json:"error"`
				}
				if err := json.Unmarshal(trimmed, &ev); err != nil {
					return nil, &application.LLMError{Kind: application.ErrKindMalformed, Provider: "llm-service", Err: err}
				}
				return nil, ev.Error.llmError()
			default:
				if onEvent != nil {
					onEvent(head.Type, json.RawMessage(trimmed))
				}
			}
		}
		if rerr != nil {
			if errors.Is(rerr, io.EOF) {
				return nil, &application.LLMError{Kind: application.ErrKindServer, Provider: "llm-service",
					Err: errors.New("llm-service closed the stream without a result")}
			}
			if errors.Is(rerr, context.DeadlineExceeded) || errors.Is(rerr, context.Canceled) {
				return nil, &application.LLMError{Kind: application.ErrKindTimeout, Provider: "llm-service", Err: rerr}
			}
			return nil, &application.LLMError{Kind: application.ErrKindServer, Provider: "llm-service", Err: fmt.Errorf("read stream: %w", rerr)}
		}
	}
}

// statusError turns a non-200 answer into an LLMError, reading llm-service's
// {"error": {...}} body when there is one.
func statusError(resp *http.Response) error {
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var body struct {
		Error wireError `json:"error"`
	}
	if json.Unmarshal(raw, &body) == nil && body.Error.Message != "" {
		return body.Error.llmError()
	}
	// A 404 without llm-service's own error body is a route this llm-service
	// does not have (an older build): not a failed call.
	if resp.StatusCode == http.StatusNotFound {
		return errRouteMissing
	}
	kind := application.ErrKindServer
	if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnprocessableEntity {
		kind = application.ErrKindMalformed
	}
	msg := strings.TrimSpace(string(raw))
	if len(msg) > 300 {
		msg = msg[:300] + "…"
	}
	return &application.LLMError{Kind: kind, Provider: "llm-service",
		Err: fmt.Errorf("llm-service returned %d: %s", resp.StatusCode, msg)}
}

// post is the non-streaming call: one JSON request, one JSON answer.
func (c *Client) post(ctx context.Context, path string, body, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return &application.LLMError{Kind: application.ErrKindMalformed, Provider: "llm-service", Err: err}
	}
	ctx, cancel := c.ctx(ctx)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return &application.LLMError{Kind: application.ErrKindMalformed, Provider: "llm-service", Err: err}
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return unreachable(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return statusError(resp)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return &application.LLMError{Kind: application.ErrKindMalformed, Provider: "llm-service", Err: fmt.Errorf("decode response: %w", err)}
	}
	return nil
}

// --- application.LLMProviderPort ---------------------------------------------

func (c *Client) Chat(ctx context.Context, req application.ChatRequest) (application.ChatResult, error) {
	body := map[string]any{
		"provider": "hive", "system": req.System, "user": req.User, "model": req.Model,
		"max_tokens": req.MaxTokens, "temperature": req.Temperature,
		"project_id": req.ProjectID, "role": req.Role, "step": req.Step, "phase": req.Phase,
		"expect_json": req.ExpectJSON,
	}
	raw, err := c.stream(ctx, "/v1/chat", body, func(kind string, ev json.RawMessage) {
		if kind != "progress" || req.OnProgress == nil {
			return
		}
		var p struct {
			ReasoningChars int `json:"reasoning_chars"`
			ContentChars   int `json:"content_chars"`
		}
		if json.Unmarshal(ev, &p) == nil {
			req.OnProgress(application.ChatProgress{ReasoningChars: p.ReasoningChars, ContentChars: p.ContentChars})
		}
	})
	if err != nil {
		return application.ChatResult{}, err
	}
	var res struct {
		Content string          `json:"content"`
		JSON    json.RawMessage `json:"json"`
		Usage   wireUsage       `json:"usage"`
		CallID  string          `json:"call_id"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return application.ChatResult{}, &application.LLMError{Kind: application.ErrKindMalformed, Provider: "llm-service", Err: err}
	}
	if req.ExpectJSON && len(res.JSON) == 0 {
		return application.ChatResult{}, &application.LLMError{
			Kind: application.ErrKindMalformed, Provider: "llm-service", CallID: res.CallID,
			Err: errors.New("llm-service answered without the JSON object that was asked for")}
	}
	return application.ChatResult{Content: res.Content, JSON: res.JSON, Usage: res.Usage.usage(), CallID: res.CallID}, nil
}

// --- MetadataSuggesterPort ----------------------------------------------------

// suggestCall posts a suggestion request. When ctx carries a progress callback
// it asks llm-service to stream and forwards the reply's size as
// it grows; otherwise it is the plain one-shot call.
func (c *Client) suggestCall(ctx context.Context, path string, body map[string]any, out any) error {
	report := application.ProgressFrom(ctx)
	if report == nil {
		return c.post(ctx, path, body, out)
	}
	body["stream"] = true
	raw, err := c.stream(ctx, path, body, func(kind string, ev json.RawMessage) {
		if kind != "progress" {
			return
		}
		var p struct {
			ReasoningChars int `json:"reasoning_chars"`
			ContentChars   int `json:"content_chars"`
		}
		if json.Unmarshal(ev, &p) == nil {
			report(application.ChatProgress{ReasoningChars: p.ReasoningChars, ContentChars: p.ContentChars})
		}
	})
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return &application.LLMError{Kind: application.ErrKindMalformed, Provider: "llm-service", Err: fmt.Errorf("decode response: %w", err)}
	}
	return nil
}

func (c *Client) Suggest(
	ctx context.Context, scriptContent, categoryHint string, language domain.ContentLanguage, model string,
) (string, string, []string, error) {
	var out struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Tags        []string `json:"tags"`
	}
	err := c.suggestCall(ctx, "/v1/suggest-metadata", map[string]any{
		"script_content": scriptContent, "category_hint": categoryHint, "language": string(language),
		"model": model,
	}, &out)
	if err != nil {
		return "", "", nil, err
	}
	if out.Tags == nil {
		out.Tags = []string{} // the API contract says tags is an array, never null
	}
	return out.Title, out.Description, out.Tags, nil
}
