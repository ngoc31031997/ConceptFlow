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
// A 404 with authoring's JSON error means no version is locked yet; any other
// 404 (an authoring-service without the route) is a failure of authoring.
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
		return domain.ScriptVersion{}, notFound(resp)
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

// notFound tells authoring's "no locked version" answer, a JSON body with an
// error, from a 404 of a router that does not know the route, so a stale
// authoring-service is reported as such instead of as an unlocked script.
func notFound(resp *http.Response) error {
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	var body struct {
		Error string `json:"error"`
	}
	if strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") && json.Unmarshal(raw, &body) == nil && body.Error != "" {
		return domain.ErrScriptNotLocked
	}
	return fmt.Errorf("%w: không có route bản kịch bản chốt (trả 404: %s); authoring-service có thể đang chạy bản cũ", application.ErrAuthoringFailed, strings.TrimSpace(string(raw)))
}
