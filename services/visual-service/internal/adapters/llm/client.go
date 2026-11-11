// Package llm is visual-service's client of llm-service: the model calls of
// step 8, the prompt library, the model catalog and the step default model
// (ADR-0048). Nothing is cached and nothing falls back to a local copy: when
// llm-service cannot answer, the caller gets an error that names it.
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
	"net/url"
	"strings"
	"time"

	"visual/internal/application"
	"visual/internal/domain"
)

// Client implements application.LLMPort, ExternalReplier, PromptLibraryPort
// and ModelCatalogPort.
type Client struct {
	baseURL string
	// timeout bounds a model call; 0 waits as long as llm-service does.
	timeout time.Duration
	http    *http.Client
}

// NewClient returns a client of the llm-service at baseURL.
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{baseURL: strings.TrimSuffix(baseURL, "/"), timeout: timeout, http: &http.Client{}}
}

type wireError struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

func (e wireError) err() error {
	if e.Kind == "malformed" {
		return fmt.Errorf("%w: %s", domain.ErrUnreadableReply, e.Message)
	}
	return fmt.Errorf("%w (%s): %s", application.ErrLLMFailed, e.Kind, e.Message)
}

func (c *Client) do(ctx context.Context, method, path string, body any) (*http.Response, context.CancelFunc, error) {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, nil, err
		}
		reader = bytes.NewReader(payload)
	}
	cancel := func() {}
	if c.timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		cancel()
		return nil, nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("%w: gọi llm-service: %v", application.ErrLLMFailed, err)
	}
	return resp, cancel, nil
}

// statusError reads a non-2xx answer of llm-service.
func statusError(resp *http.Response) error {
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var nested struct {
		Error wireError `json:"error"`
	}
	if json.Unmarshal(raw, &nested) == nil && nested.Error.Message != "" {
		return nested.Error.err()
	}
	var flat struct {
		Error string `json:"error"`
	}
	msg := strings.TrimSpace(string(raw))
	if json.Unmarshal(raw, &flat) == nil && flat.Error != "" {
		msg = flat.Error
	}
	return fmt.Errorf("%w: llm-service trả %d: %s", application.ErrLLMFailed, resp.StatusCode, msg)
}

type chatResult struct {
	Content string          `json:"content"`
	JSON    json.RawMessage `json:"json"`
}

// Chat sends one turn to /v1/chat and reads its NDJSON stream to the result.
func (c *Client) Chat(ctx context.Context, req application.ChatRequest) (application.ChatResult, error) {
	resp, cancel, err := c.do(ctx, http.MethodPost, "/v1/chat", map[string]any{
		"provider": "hive", "system": req.System, "user": req.User, "model": req.Model, "max_tokens": req.MaxTokens,
		"project_id": req.ProjectID, "role": req.Role, "step": req.Step, "expect_json": req.ExpectJSON,
	})
	if err != nil {
		return application.ChatResult{}, err
	}
	defer cancel()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return application.ChatResult{}, statusError(resp)
	}
	rd := bufio.NewReaderSize(resp.Body, 64*1024)
	for {
		line, rerr := rd.ReadBytes('\n')
		if trimmed := bytes.TrimSpace(line); len(trimmed) > 0 {
			var ev struct {
				Type  string    `json:"type"`
				Error wireError `json:"error"`
				chatResult
			}
			if err := json.Unmarshal(trimmed, &ev); err != nil {
				return application.ChatResult{}, fmt.Errorf("%w: sự kiện llm-service không đọc được: %v", application.ErrLLMFailed, err)
			}
			switch ev.Type {
			case "result":
				if req.ExpectJSON && len(ev.JSON) == 0 {
					return application.ChatResult{}, fmt.Errorf("%w: llm-service trả lời không kèm JSON", domain.ErrUnreadableReply)
				}
				return application.ChatResult{Content: ev.Content, JSON: ev.JSON}, nil
			case "error":
				return application.ChatResult{}, ev.Error.err()
			}
		}
		if rerr != nil {
			if errors.Is(rerr, io.EOF) {
				return application.ChatResult{}, fmt.Errorf("%w: llm-service đóng luồng mà chưa có kết quả", application.ErrLLMFailed)
			}
			return application.ChatResult{}, fmt.Errorf("%w: đọc luồng llm-service: %v", application.ErrLLMFailed, rerr)
		}
	}
}

// ChatExternal has llm-service take reply, pasted from an AI outside the
// system, as the answer to req's prompt and log it (ADR-0047).
func (c *Client) ChatExternal(ctx context.Context, req application.ChatRequest, reply string) (application.ChatResult, error) {
	resp, cancel, err := c.do(ctx, http.MethodPost, "/v1/chat/external", map[string]any{
		"system": req.System, "user": req.User, "reply": reply, "expect_json": req.ExpectJSON,
		"project_id": req.ProjectID, "role": req.Role, "step": req.Step,
	})
	if err != nil {
		return application.ChatResult{}, err
	}
	defer cancel()
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return application.ChatResult{}, statusError(resp)
	}
	var res chatResult
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return application.ChatResult{}, fmt.Errorf("%w: %v", application.ErrLLMFailed, err)
	}
	return application.ChatResult{Content: res.Content, JSON: res.JSON}, nil
}

// libraryCall sends one JSON request and decodes a 200 answer into out.
func (c *Client) libraryCall(ctx context.Context, method, path string, body, out any) error {
	resp, cancel, err := c.do(ctx, method, path, body)
	if err != nil {
		return err
	}
	defer cancel()
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return application.ErrPromptNotFound
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return statusError(resp)
	}
	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("%w: %v", application.ErrLLMFailed, err)
	}
	return nil
}

// ActiveTemplate returns the template role runs on: the active prompt, else
// the role's system prompt.
func (c *Client) ActiveTemplate(ctx context.Context, role string) (string, error) {
	var p struct {
		TemplateText string `json:"template_text"`
	}
	if err := c.libraryCall(ctx, http.MethodGet, "/internal/v1/prompts/"+url.PathEscape(role), nil, &p); err != nil {
		return "", fmt.Errorf("lấy prompt %q từ llm-service: %w", role, err)
	}
	return p.TemplateText, nil
}

// PushSystemPrompts makes prompts the whole set of system prompts owner ships.
func (c *Client) PushSystemPrompts(ctx context.Context, owner string, prompts []domain.SystemPrompt) error {
	type item struct {
		Role         string `json:"role"`
		Name         string `json:"name"`
		TemplateText string `json:"template_text"`
	}
	items := make([]item, 0, len(prompts))
	for _, p := range prompts {
		items = append(items, item{Role: p.Role, Name: p.Name, TemplateText: p.TemplateText})
	}
	if err := c.libraryCall(ctx, http.MethodPut, "/internal/v1/prompts/system", map[string]any{"owner": owner, "prompts": items}, nil); err != nil {
		return fmt.Errorf("đẩy prompt hệ thống lên llm-service: %w", err)
	}
	return nil
}

// Models returns llm-service's model catalog.
func (c *Client) Models(ctx context.Context) ([]application.ModelOption, error) {
	var out struct {
		Models []application.ModelOption `json:"models"`
	}
	if err := c.libraryCall(ctx, http.MethodGet, "/internal/v1/models", nil, &out); err != nil {
		return nil, fmt.Errorf("lấy danh sách model từ llm-service: %w", err)
	}
	return out.Models, nil
}

// StepDefaultModel returns the default model of step.
func (c *Client) StepDefaultModel(ctx context.Context, step string) (string, error) {
	var out struct {
		StepModels map[string]string `json:"step_models"`
	}
	if err := c.libraryCall(ctx, http.MethodGet, "/internal/v1/step-models", nil, &out); err != nil {
		return "", fmt.Errorf("lấy model mặc định từ llm-service: %w", err)
	}
	m, ok := out.StepModels[step]
	if !ok {
		return "", fmt.Errorf("%w: llm-service chưa có bước %q", application.ErrLLMFailed, step)
	}
	return m, nil
}
