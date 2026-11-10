package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"authoring/internal/application"
	"authoring/internal/domain"
)

// This file is the client side of llm-service's prompt library, model catalog
// and per-step default models. Nothing is cached and nothing falls back to a
// local copy: when llm-service cannot answer, the caller gets an error that
// says so.

// libraryCall sends one JSON request and decodes a 200 answer into out (nil
// = ignore the body; 204 is also accepted).
func (c *Client) libraryCall(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		reader = bytes.NewReader(payload)
	}
	ctx, cancel := c.ctx(ctx)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return unreachable(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return application.ErrPromptNotFound
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		var msg struct {
			Error string `json:"error"`
		}
		text := strings.TrimSpace(string(raw))
		if json.Unmarshal(raw, &msg) == nil && msg.Error != "" {
			text = msg.Error
		}
		if resp.StatusCode == http.StatusBadRequest {
			return fmt.Errorf("%w: %s", domain.ErrInvalidWizardInput, text)
		}
		return &application.LLMError{Kind: application.ErrKindServer, Provider: "llm-service",
			Err: fmt.Errorf("llm-service returned %d: %s", resp.StatusCode, text)}
	}
	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return &application.LLMError{Kind: application.ErrKindMalformed, Provider: "llm-service",
			Err: fmt.Errorf("decode response: %w", err)}
	}
	return nil
}

// GetActive returns the prompt a role runs on: the active one, else the
// role's system prompt.
func (c *Client) GetActive(ctx context.Context, role domain.PromptRole) (domain.Prompt, error) {
	var p domain.Prompt
	if err := c.libraryCall(ctx, http.MethodGet, "/internal/v1/prompts/"+url.PathEscape(string(role)), nil, &p); err != nil {
		return domain.Prompt{}, fmt.Errorf("lấy prompt %q từ llm-service: %w", role, err)
	}
	return p, nil
}

// PushSystemPrompts makes prompts the whole set of system prompts owner ships.
// Prompts of the owner that are no longer listed are removed by llm-service;
// the Creator's own prompts are never touched.
func (c *Client) PushSystemPrompts(ctx context.Context, owner string, prompts []domain.Prompt) error {
	type item struct {
		Role         string `json:"role"`
		Name         string `json:"name"`
		TemplateText string `json:"template_text"`
	}
	items := make([]item, 0, len(prompts))
	for _, p := range prompts {
		items = append(items, item{Role: string(p.Role), Name: p.Name, TemplateText: p.TemplateText})
	}
	body := map[string]any{"owner": owner, "prompts": items}
	if err := c.libraryCall(ctx, http.MethodPut, "/internal/v1/prompts/system", body, nil); err != nil {
		return fmt.Errorf("đẩy prompt hệ thống lên llm-service: %w", err)
	}
	return nil
}

// ModelCatalog returns the models a step can run on, first one being the
// default of a step with none saved.
func (c *Client) ModelCatalog(ctx context.Context) ([]domain.AuthoringModelOption, error) {
	var out struct {
		Models []domain.AuthoringModelOption `json:"models"`
	}
	if err := c.libraryCall(ctx, http.MethodGet, "/internal/v1/models", nil, &out); err != nil {
		return nil, fmt.Errorf("lấy danh sách model từ llm-service: %w", err)
	}
	return out.Models, nil
}

// GetStepDefaultModels returns the default model of every configurable step.
func (c *Client) GetStepDefaultModels(ctx context.Context) (map[string]string, error) {
	var out struct {
		StepModels map[string]string `json:"step_models"`
	}
	if err := c.libraryCall(ctx, http.MethodGet, "/internal/v1/step-models", nil, &out); err != nil {
		return nil, fmt.Errorf("lấy model mặc định từ llm-service: %w", err)
	}
	return out.StepModels, nil
}
