// Package orchestrator is authoring-service's only route to project data: the
// orchestrator owns projects, video formats, voice calibration and the project
// journal, and this client reads and appends to them over its internal HTTP API
// (CR-040 FR111.3). authoring-service never opens the orchestrator's database.
package orchestrator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"authoring/internal/application"
	"authoring/internal/domain"
)

// Client implements the project-side ports the authoring use cases depend on.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient returns a client for the orchestrator at baseURL.
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{baseURL: baseURL, http: &http.Client{Timeout: timeout}}
}

// do sends one request and decodes a JSON body into out (nil = ignore). A 404
// becomes domain.ErrProjectNotFound so callers keep the same error contract the
// repository-backed version had.
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
		return fmt.Errorf("orchestrator %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return domain.ErrProjectNotFound
	}
	if resp.StatusCode/100 != 2 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("orchestrator %s %s: status %d: %s", method, path, resp.StatusCode, msg)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// Get returns the project (settings, script, chapters...) the prompt renderer
// and the metadata suggester read.
func (c *Client) Get(ctx context.Context, projectID string) (*domain.Project, error) {
	var p domain.Project
	if err := c.do(ctx, http.MethodGet, "/internal/v1/projects/"+url.PathEscape(projectID), nil, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// GetStatus is the narrow read the authoring lock (CR-028 FR84.2) needs.
func (c *Client) GetStatus(ctx context.Context, projectID string) (domain.ProjectStatus, error) {
	var out struct {
		Status domain.ProjectStatus `json:"status"`
	}
	if err := c.do(ctx, http.MethodGet, "/internal/v1/projects/"+url.PathEscape(projectID)+"/status", nil, &out); err != nil {
		return "", err
	}
	return out.Status, nil
}

// GetVideoFormat resolves a format at the version the project was created against.
func (c *Client) GetVideoFormat(ctx context.Context, formatID string, version int) (domain.VideoFormat, error) {
	var f domain.VideoFormat
	err := c.do(ctx, http.MethodGet,
		"/internal/v1/formats/"+url.PathEscape(formatID)+"?version="+strconv.Itoa(version), nil, &f)
	return f, err
}

// GetVoiceCalibration returns what a voice has actually been measured doing.
func (c *Client) GetVoiceCalibration(ctx context.Context, voiceID string) (domain.VoiceCalibration, error) {
	var v domain.VoiceCalibration
	err := c.do(ctx, http.MethodGet, "/internal/v1/voices/"+url.PathEscape(voiceID)+"/calibration", nil, &v)
	return v, err
}

// AppendProjectError appends to the project's error log.
func (c *Client) AppendProjectError(ctx context.Context, projectID string, e application.ProjectError) error {
	return c.do(ctx, http.MethodPost, "/internal/v1/projects/"+url.PathEscape(projectID)+"/errors", e, nil)
}

// AppendProjectEvent appends to the project's journey log.
func (c *Client) AppendProjectEvent(ctx context.Context, e domain.ProjectEvent) error {
	return c.do(ctx, http.MethodPost, "/internal/v1/projects/"+url.PathEscape(e.ProjectID)+"/events", e, nil)
}

// PromptRenderContext joins this client with the local authoring repository into
// what RenderPromptUseCase reads: the project from the orchestrator, the
// authoring artefacts from this service's own database.
type PromptRenderContext struct {
	Projects  *Client
	Authoring interface {
		GetAuthoringTopic(ctx context.Context, projectID string) (string, error)
		GetAuthoringStory(ctx context.Context, projectID string) (string, error)
		GetAuthoringStoryboard(ctx context.Context, projectID string) (string, error)
		GetAuthoringCode(ctx context.Context, projectID string) (string, error)
	}
}

func (c PromptRenderContext) Get(ctx context.Context, projectID string) (*domain.Project, error) {
	return c.Projects.Get(ctx, projectID)
}
func (c PromptRenderContext) GetAuthoringTopic(ctx context.Context, id string) (string, error) {
	return c.Authoring.GetAuthoringTopic(ctx, id)
}
func (c PromptRenderContext) GetAuthoringStory(ctx context.Context, id string) (string, error) {
	return c.Authoring.GetAuthoringStory(ctx, id)
}
func (c PromptRenderContext) GetAuthoringStoryboard(ctx context.Context, id string) (string, error) {
	return c.Authoring.GetAuthoringStoryboard(ctx, id)
}
func (c PromptRenderContext) GetAuthoringCode(ctx context.Context, id string) (string, error) {
	return c.Authoring.GetAuthoringCode(ctx, id)
}
