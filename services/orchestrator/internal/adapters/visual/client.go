// Package visual is the orchestrator's client for visual-service, which owns
// the visual pre-production steps. The orchestrator only removes a deleted
// project's data there, over its internal HTTP API.
package visual

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client implements application.VisualCleanupPort.
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
