// Package rendering is visual-service's client of the rendering service: it
// lists the fonts rendering can draw text in. Nothing is cached and there is no
// built-in list: when rendering does not answer, the caller gets an error that
// names it.
package rendering

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"visual/internal/application"
)

// Client implements application.FontCatalogPort.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient returns a client of the rendering service at baseURL.
func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{baseURL: strings.TrimSuffix(baseURL, "/"), http: &http.Client{Timeout: timeout}}
}

// Fonts reads GET /v1/fonts. An empty list is an error: a list with no font
// would reject every font.
func (c *Client) Fonts(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/fonts", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", application.ErrRenderingFailed, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
		return nil, fmt.Errorf("%w: trả %d: %s", application.ErrRenderingFailed, resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var out struct {
		Fonts []string `json:"fonts"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("%w: câu trả lời không đọc được: %v", application.ErrRenderingFailed, err)
	}
	if len(out.Fonts) == 0 {
		return nil, fmt.Errorf("%w: danh sách font trống", application.ErrRenderingFailed)
	}
	return out.Fonts, nil
}
