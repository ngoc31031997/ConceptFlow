// Package rendering calls the rendering service: the illustration preview
// (check a library drawing and render its PNG still and GIF), the kits it
// draws with and its shared stills.
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
	"authoring/internal/domain"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), http: &http.Client{Timeout: timeout}}
}

type previewRequest struct {
	KitKey string         `json:"kit_key"`
	Tier   string         `json:"tier"`
	Name   string         `json:"name"`
	Code   string         `json:"code"`
	Props  map[string]any `json:"props"`
	GIF    bool           `json:"gif"`
	Kind   string         `json:"kind"`
}

type previewResponse struct {
	OK          bool                            `json:"ok"`
	Diagnostics []application.PreviewDiagnostic `json:"diagnostics"`
	Warnings    []application.PreviewDiagnostic `json:"warnings"`
	PNG         string                          `json:"png"`
	GIF         string                          `json:"gif"`
	Error       string                          `json:"error"`
}

func (c *Client) PreviewIllustration(
	ctx context.Context, kitKey, name, code string, props map[string]any, gif bool, kind domain.IllustrationKind, tier domain.Tier,
) (application.IllustrationPreview, error) {
	if props == nil {
		props = map[string]any{}
	}
	body, _ := json.Marshal(previewRequest{
		KitKey: kitKey, Tier: string(domain.TierOrSupport(tier)),
		Name: name, Code: code, Props: props, GIF: gif, Kind: string(domain.KindOrFigure(kind)),
	})
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
	preview := application.IllustrationPreview{OK: out.OK, Diagnostics: out.Diagnostics, Warnings: out.Warnings}
	if preview.PNG, err = base64.StdEncoding.DecodeString(out.PNG); err != nil {
		return application.IllustrationPreview{}, fmt.Errorf("rendering sent a broken PNG: %w", err)
	}
	if preview.GIF, err = base64.StdEncoding.DecodeString(out.GIF); err != nil {
		return application.IllustrationPreview{}, fmt.Errorf("rendering sent a broken GIF: %w", err)
	}
	return preview, nil
}

type kitsResponse struct {
	Kits []struct {
		Key     string `json:"key"`
		Version int    `json:"version"`
	} `json:"kits"`
}

// Kits returns the version of every kit the rendering service draws with.
func (c *Client) Kits(ctx context.Context) (map[string]int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/kits", nil)
	if err != nil {
		return nil, err
	}
	var out kitsResponse
	if err := c.doJSON(req, &out); err != nil {
		return nil, err
	}
	kits := make(map[string]int, len(out.Kits))
	for _, k := range out.Kits {
		kits[k.Key] = k.Version
	}
	return kits, nil
}

type stillRequest struct {
	Composition string `json:"composition"`
	KitKey      string `json:"kit_key"`
}

type stillResponse struct {
	File     string `json:"file"`
	CacheHit bool   `json:"cache_hit"`
}

// Still renders (or finds cached) one sample composition of a kit.
func (c *Client) Still(ctx context.Context, composition, kitKey string) (string, bool, error) {
	body, _ := json.Marshal(stillRequest{Composition: composition, KitKey: kitKey})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/stills", bytes.NewReader(body))
	if err != nil {
		return "", false, err
	}
	req.Header.Set("Content-Type", "application/json")
	var out stillResponse
	if err := c.doJSON(req, &out); err != nil {
		return "", false, err
	}
	if out.File == "" {
		return "", false, fmt.Errorf("rendering answered a still without a file")
	}
	return out.File, out.CacheHit, nil
}

type pruneRequest struct {
	Keep        []string `json:"keep"`
	MinAgeHours float64  `json:"min_age_hours"`
}

type pruneResponse struct {
	Removed []string `json:"removed"`
}

// Prune deletes stills not in keep and older than minAgeHours.
func (c *Client) Prune(ctx context.Context, keep []string, minAgeHours float64) ([]string, error) {
	if keep == nil {
		keep = []string{}
	}
	body, _ := json.Marshal(pruneRequest{Keep: keep, MinAgeHours: minAgeHours})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/stills/prune", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	var out pruneResponse
	if err := c.doJSON(req, &out); err != nil {
		return nil, err
	}
	return out.Removed, nil
}

// doJSON sends req and decodes a 200 JSON answer into out; any other answer is an error that carries rendering's message.
func (c *Client) doJSON(req *http.Request, out any) error {
	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("rendering unreachable: %w", err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if res.StatusCode != http.StatusOK {
		var e struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(raw, &e)
		return fmt.Errorf("rendering answered %d: %s", res.StatusCode, e.Error)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("rendering answered an unreadable body: %w", err)
	}
	return nil
}
