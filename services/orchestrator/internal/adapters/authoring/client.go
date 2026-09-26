// Package authoring is the orchestrator's client for authoring-service, which
// owns the prompt library, the 1a/1b/1c artefacts, the topic and the LLM
// calls (CR-040 FR111). The orchestrator only needs a few things from it —
// the topic collision search, list summaries, a fork's copy and a cleanup —
// over its internal HTTP API.
package authoring

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"orchestrator/internal/application"
	"orchestrator/internal/domain"
)

// Client implements application.AuthoringLookupPort and the fork's read/write port.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient returns a client for authoring-service at baseURL.
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{baseURL: baseURL, http: &http.Client{Timeout: timeout}}
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("authoring-service %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("authoring-service %s %s: status %d: %s", method, path, resp.StatusCode, msg)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

type state struct {
	Mode            string `json:"mode"`
	Topic           string `json:"topic"`
	Story           string `json:"story"`
	Storyboard      string `json:"storyboard"`
	Code            string `json:"code"`
	StoryModel      string `json:"story_model"`
	StoryboardModel string `json:"storyboard_model"`
	CodeModel       string `json:"code_model"`
}

func (c *Client) state(ctx context.Context, projectID string) (state, error) {
	var s state
	err := c.do(ctx, http.MethodGet, "/internal/v1/authoring/"+url.PathEscape(projectID), nil, &s)
	return s, err
}

func (c *Client) GetAuthoringTopic(ctx context.Context, id string) (string, error) {
	s, err := c.state(ctx, id)
	return s.Topic, err
}
func (c *Client) GetAuthoringMode(ctx context.Context, id string) (string, error) {
	s, err := c.state(ctx, id)
	return s.Mode, err
}
func (c *Client) GetAuthoringStory(ctx context.Context, id string) (string, error) {
	s, err := c.state(ctx, id)
	return s.Story, err
}
func (c *Client) GetAuthoringStoryboard(ctx context.Context, id string) (string, error) {
	s, err := c.state(ctx, id)
	return s.Storyboard, err
}
func (c *Client) GetAuthoringModels(ctx context.Context, id string) (domain.AuthoringStepModels, error) {
	s, err := c.state(ctx, id)
	return domain.AuthoringStepModels{Story: s.StoryModel, Storyboard: s.StoryboardModel, Code: s.CodeModel}, err
}

func (c *Client) put(ctx context.Context, projectID string, body map[string]any) error {
	return c.do(ctx, http.MethodPut, "/internal/v1/authoring/"+url.PathEscape(projectID), body, nil)
}

func (c *Client) SaveAuthoringTopic(ctx context.Context, id, topic string, language domain.ContentLanguage) error {
	return c.put(ctx, id, map[string]any{"topic": topic, "language": string(language)})
}
func (c *Client) SaveAuthoringStory(ctx context.Context, id, content, _ string) error {
	return c.put(ctx, id, map[string]any{"story": content})
}
func (c *Client) SaveAuthoringStoryboard(ctx context.Context, id, content string) error {
	return c.put(ctx, id, map[string]any{"storyboard": content})
}
func (c *Client) SaveAuthoringMode(ctx context.Context, id, mode string) error {
	return c.put(ctx, id, map[string]any{"mode": mode})
}
func (c *Client) SaveAuthoringModels(ctx context.Context, id string, m domain.AuthoringStepModels) error {
	return c.put(ctx, id, map[string]any{"models": m})
}

// FindSimilarTopics asks authoring-service which other projects saved the same
// (normalized) topic in this language. Status is left empty — the caller fills
// it from the projects it owns.
func (c *Client) FindSimilarTopics(ctx context.Context, language domain.ContentLanguage, normalizedTopic, excludeProjectID string) ([]application.SimilarProject, error) {
	q := url.Values{"language": {string(language)}, "topic": {normalizedTopic}, "exclude": {excludeProjectID}}
	var out []application.SimilarProject
	err := c.do(ctx, http.MethodGet, "/internal/v1/authoring/similar?"+q.Encode(), nil, &out)
	return out, err
}

// Summaries returns the topic and content flags for the given projects.
func (c *Client) Summaries(ctx context.Context, projectIDs []string) (map[string]application.AuthoringSummary, error) {
	out := map[string]application.AuthoringSummary{}
	if len(projectIDs) == 0 {
		return out, nil
	}
	q := url.Values{"ids": {strings.Join(projectIDs, ",")}}
	err := c.do(ctx, http.MethodGet, "/internal/v1/authoring/summaries?"+q.Encode(), nil, &out)
	return out, err
}

// DeleteAuthoring removes a deleted project's authoring row.
func (c *Client) DeleteAuthoring(ctx context.Context, projectID string) error {
	return c.do(ctx, http.MethodDelete, "/internal/v1/authoring/"+url.PathEscape(projectID), nil, nil)
}
