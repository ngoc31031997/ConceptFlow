// Package visual is the orchestrator's client for visual-service, which owns
// the visual pre-production steps. Over its internal HTTP API the orchestrator
// reads the drawing kit a render uses and removes a deleted project's data.
package visual

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"orchestrator/internal/domain"
)

// Client implements application.VisualCleanupPort and application.KitPort.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient returns a client for visual-service at baseURL.
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{baseURL: baseURL, http: &http.Client{Timeout: timeout}}
}

// DeleteVisual removes a deleted project's visual pre-production data. A
// project visual-service never saw is not an error there.
func (c *Client) DeleteVisual(ctx context.Context, projectID string) error {
	path := "/internal/v1/visual/" + url.PathEscape(projectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("visual-service DELETE %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("visual-service DELETE %s: status %d: %s", path, resp.StatusCode, msg)
	}
	return nil
}

// ProjectKit is the drawing kit of the reference style the project picked at
// step 8. visual-service answers 404 when no style is picked and 409 when the
// style has no kit; those map to domain.ErrNoReferenceStyle and
// domain.ErrReferenceStyleHasNoKit.
func (c *Client) ProjectKit(ctx context.Context, projectID string) (map[string]interface{}, error) {
	path := "/internal/v1/visual/" + url.PathEscape(projectID) + "/kit"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("visual-service GET %s: %w", path, err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return nil, domain.ErrNoReferenceStyle
	case http.StatusConflict:
		return nil, domain.ErrReferenceStyleHasNoKit
	default:
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("visual-service GET %s: status %d: %s", path, resp.StatusCode, msg)
	}
	var body struct {
		RefID string                 `json:"ref_id"`
		Kit   map[string]interface{} `json:"kit"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("visual-service GET %s: decode: %w", path, err)
	}
	if len(body.Kit) == 0 {
		return nil, domain.ErrReferenceStyleHasNoKit
	}
	return body.Kit, nil
}
