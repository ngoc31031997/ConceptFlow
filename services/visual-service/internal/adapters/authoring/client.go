// Package authoring is visual-service's client of authoring-service: it reads
// the latest locked script version step 8 is written from.
package authoring

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"visual/internal/application"
	"visual/internal/domain"
)

// Client implements application.AuthoringInputPort.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient returns a client of the authoring-service at baseURL.
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{baseURL: strings.TrimSuffix(baseURL, "/"), http: &http.Client{Timeout: timeout}}
}

// LatestScriptVersion reads GET /internal/v1/authoring/{id}/script-versions/latest.
// 404 means no version is locked yet.
func (c *Client) LatestScriptVersion(ctx context.Context, projectID string) (domain.ScriptVersion, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.baseURL+"/internal/v1/authoring/"+url.PathEscape(projectID)+"/script-versions/latest", nil)
	if err != nil {
		return domain.ScriptVersion{}, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return domain.ScriptVersion{}, fmt.Errorf("%w: %v", application.ErrAuthoringFailed, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return domain.ScriptVersion{}, domain.ErrScriptNotLocked
	}
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
		return domain.ScriptVersion{}, fmt.Errorf("%w: trả %d: %s", application.ErrAuthoringFailed, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var v domain.ScriptVersion
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return domain.ScriptVersion{}, fmt.Errorf("%w: %v", application.ErrAuthoringFailed, err)
	}
	return v, nil
}
