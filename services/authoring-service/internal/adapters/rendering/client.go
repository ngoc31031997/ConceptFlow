// Package rendering calls the rendering service's illustration preview
// endpoint (CR-044): check a library drawing and render its PNG still and GIF.
package rendering

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"authoring/internal/application"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), http: &http.Client{Timeout: timeout}}
}

type previewRequest struct {
	Name  string         `json:"name"`
	Code  string         `json:"code"`
	Props map[string]any `json:"props"`
	GIF   bool           `json:"gif"`
}

type previewResponse struct {
	OK          bool                            `json:"ok"`
	Diagnostics []application.PreviewDiagnostic `json:"diagnostics"`
	PNG         string                          `json:"png"`
	GIF         string                          `json:"gif"`
	Error       string                          `json:"error"`
}

func (c *Client) PreviewIllustration(
	ctx context.Context, name, code string, props map[string]any, gif bool,
) (application.IllustrationPreview, error) {
	if props == nil {
		props = map[string]any{}
	}
	body, _ := json.Marshal(previewRequest{Name: name, Code: code, Props: props, GIF: gif})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/illustrations/preview", bytes.NewReader(body))
	if err != nil {
		return application.IllustrationPreview{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return application.IllustrationPreview{}, fmt.Errorf("rendering unreachable: %w", err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 64<<20))
	var out previewResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return application.IllustrationPreview{}, fmt.Errorf("rendering answered %d with an unreadable body", res.StatusCode)
	}
	if res.StatusCode != http.StatusOK {
		return application.IllustrationPreview{}, fmt.Errorf("rendering answered %d: %s", res.StatusCode, out.Error)
	}
	preview := application.IllustrationPreview{OK: out.OK, Diagnostics: out.Diagnostics}
	if preview.PNG, err = base64.StdEncoding.DecodeString(out.PNG); err != nil {
		return application.IllustrationPreview{}, fmt.Errorf("rendering sent a broken PNG: %w", err)
	}
	if preview.GIF, err = base64.StdEncoding.DecodeString(out.GIF); err != nil {
		return application.IllustrationPreview{}, fmt.Errorf("rendering sent a broken GIF: %w", err)
	}
	return preview, nil
}
