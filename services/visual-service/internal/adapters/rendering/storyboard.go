package rendering

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"visual/internal/application"
	"visual/internal/domain"
)

// maxBody caps what is read from one answer of rendering.
const maxBody = 64 << 20

var _ application.StoryboardRenderingPort = (*Client)(nil)

// send makes one call to rendering and returns the status and body of its
// answer. A call that got no answer wraps application.ErrRenderingFailed.
func (c *Client) send(ctx context.Context, method, path string, body any) (int, []byte, error) {
	var rd io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		rd = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, rd)
	if err != nil {
		return 0, nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("%w: %v", application.ErrRenderingFailed, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return 0, nil, fmt.Errorf("%w: đọc câu trả lời: %v", application.ErrRenderingFailed, err)
	}
	return resp.StatusCode, raw, nil
}

// refusal turns a non-2xx answer into the error that carries rendering's status
// and reason.
func refusal(status int, raw []byte) error {
	re := &application.RenderingError{Status: status, Extra: map[string]json.RawMessage{}}
	var body map[string]json.RawMessage
	if json.Unmarshal(raw, &body) == nil {
		for k, v := range body {
			if k == "error" {
				_ = json.Unmarshal(v, &re.Message)
				continue
			}
			re.Extra[k] = v
		}
	}
	if re.Message == "" {
		re.Message = strings.TrimSpace(string(raw))
	}
	return re
}

// do makes a call that succeeds with 2xx and decodes the answer into out (nil
// for none).
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	status, raw, err := c.send(ctx, method, path, body)
	if err != nil {
		return err
	}
	if status < 200 || status > 299 {
		return refusal(status, raw)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("%w: câu trả lời không đọc được: %v", application.ErrRenderingFailed, err)
	}
	return nil
}

func assetsPath(projectID string) string {
	return "/internal/v1/projects/" + url.PathEscape(projectID) + "/assets"
}

func storyboardPath(projectID string) string {
	return "/internal/v1/projects/" + url.PathEscape(projectID) + "/storyboard"
}

// ListAssets reads the video's assets, oldest first.
func (c *Client) ListAssets(ctx context.Context, projectID string) ([]application.ProjectAsset, error) {
	var out struct {
		Assets []application.ProjectAsset `json:"assets"`
	}
	if err := c.do(ctx, http.MethodGet, assetsPath(projectID), nil, &out); err != nil {
		return nil, err
	}
	if out.Assets == nil {
		out.Assets = []application.ProjectAsset{}
	}
	return out.Assets, nil
}

// CreateAsset creates an asset.
func (c *Client) CreateAsset(ctx context.Context, projectID string, in application.AssetInput) (application.ProjectAsset, error) {
	var out application.ProjectAsset
	err := c.do(ctx, http.MethodPost, assetsPath(projectID), in, &out)
	return out, err
}

// UpdateAsset replaces the fields of an asset.
func (c *Client) UpdateAsset(ctx context.Context, projectID, assetID string, in application.AssetInput) (application.ProjectAsset, error) {
	var out application.ProjectAsset
	err := c.do(ctx, http.MethodPut, assetsPath(projectID)+"/"+url.PathEscape(assetID), in, &out)
	return out, err
}

// DeleteAsset deletes an asset.
func (c *Client) DeleteAsset(ctx context.Context, projectID, assetID string) error {
	return c.do(ctx, http.MethodDelete, assetsPath(projectID)+"/"+url.PathEscape(assetID), nil, nil)
}

func (c *Client) assetCall(ctx context.Context, projectID, assetID, action string, body any) (application.ProjectAsset, error) {
	var out application.ProjectAsset
	err := c.do(ctx, http.MethodPost, assetsPath(projectID)+"/"+url.PathEscape(assetID)+"/"+action, body, &out)
	return out, err
}

// drawCheckError reads rendering's 422 refusal of drawn code, {"error",
// "detail": {kind, message, line, snippet}}; ok is false for any other body.
func drawCheckError(raw []byte) (*application.AssetDrawCheckError, bool) {
	var body struct {
		Detail *application.AssetDrawError `json:"detail"`
	}
	if json.Unmarshal(raw, &body) != nil || body.Detail == nil || !body.Detail.Valid() {
		return nil, false
	}
	return &application.AssetDrawCheckError{Detail: *body.Detail}, true
}

// drawCall makes one of the drawing calls of an asset. Drawn code rendering
// refuses is an *application.AssetDrawCheckError; any other refusal is a
// *application.RenderingError.
func (c *Client) drawCall(ctx context.Context, projectID, assetID, action string, body, out any) error {
	status, raw, err := c.send(ctx, http.MethodPost, assetsPath(projectID)+"/"+url.PathEscape(assetID)+"/"+action, body)
	if err != nil {
		return err
	}
	if status == http.StatusUnprocessableEntity {
		if de, ok := drawCheckError(raw); ok {
			return de
		}
	}
	if status < 200 || status > 299 {
		return refusal(status, raw)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("%w: câu trả lời không đọc được: %v", application.ErrRenderingFailed, err)
	}
	return nil
}

// AssetDrawPrompt has rendering build the prompt of one drawing of an asset.
func (c *Client) AssetDrawPrompt(ctx context.Context, projectID, assetID string, req application.AssetDrawPromptRequest) (application.AssetDrawPrompt, error) {
	var out application.AssetDrawPrompt
	err := c.drawCall(ctx, projectID, assetID, "draw-prompt", req, &out)
	return out, err
}

// AssetDrawCheck has rendering check and render an AI's reply without storing it.
func (c *Client) AssetDrawCheck(ctx context.Context, projectID, assetID string, req application.AssetDrawCheckRequest) (application.AssetDrawPreview, error) {
	var out application.AssetDrawPreview
	err := c.drawCall(ctx, projectID, assetID, "draw-check", req, &out)
	return out, err
}

// AssetDrawApply has rendering check, render and store code as the asset's next version.
func (c *Client) AssetDrawApply(ctx context.Context, projectID, assetID string, req application.AssetDrawApplyRequest) (application.ProjectAsset, error) {
	var out application.ProjectAsset
	err := c.drawCall(ctx, projectID, assetID, "draw-apply", req, &out)
	return out, err
}

// ApproveAsset approves a drawn asset.
func (c *Client) ApproveAsset(ctx context.Context, projectID, assetID string, req application.AssetApproveRequest) (application.ProjectAsset, error) {
	return c.assetCall(ctx, projectID, assetID, "approve", req)
}

// UnapproveAsset puts an asset back to drawn.
func (c *Client) UnapproveAsset(ctx context.Context, projectID, assetID string) (application.ProjectAsset, error) {
	return c.assetCall(ctx, projectID, assetID, "unapprove", struct{}{})
}

// AssetPreview reads the PNG of an asset's current drawing.
func (c *Client) AssetPreview(ctx context.Context, projectID, assetID string) ([]byte, error) {
	status, raw, err := c.send(ctx, http.MethodGet, assetsPath(projectID)+"/"+url.PathEscape(assetID)+"/preview", nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, refusal(status, raw)
	}
	return raw, nil
}

// ApprovedLibrary lists the approved drawings of the illustration library.
func (c *Client) ApprovedLibrary(ctx context.Context) ([]application.LibraryDrawing, error) {
	var out struct {
		Illustrations []application.LibraryDrawing `json:"illustrations"`
	}
	if err := c.do(ctx, http.MethodGet, "/v1/illustrations?status=approved", nil, &out); err != nil {
		return nil, err
	}
	return out.Illustrations, nil
}

// DrawFrame has rendering draw one shot's frame.
func (c *Client) DrawFrame(ctx context.Context, projectID string, req application.FrameRequest) (application.FrameResult, error) {
	var out application.FrameResult
	err := c.do(ctx, http.MethodPost, storyboardPath(projectID)+"/frames", req, &out)
	return out, err
}

// FrameImage reads the PNG of a shot's last drawn frame.
func (c *Client) FrameImage(ctx context.Context, projectID, shotID string) ([]byte, error) {
	status, raw, err := c.send(ctx, http.MethodGet, storyboardPath(projectID)+"/frames/"+url.PathEscape(shotID), nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, refusal(status, raw)
	}
	return raw, nil
}

// LockStoryboard stores the locked document at rendering.
func (c *Client) LockStoryboard(ctx context.Context, projectID string, document json.RawMessage, sequenceVersion int, note string) (application.StoryboardVersionRecord, error) {
	var out application.StoryboardVersionRecord
	err := c.do(ctx, http.MethodPost, storyboardPath(projectID)+"/versions", map[string]any{
		"document": document, "sequence_version": sequenceVersion, "note": note}, &out)
	return out, err
}

// versionCall reads one locked storyboard; a 404 is domain.ErrNoVersion.
func (c *Client) versionCall(ctx context.Context, path string) (application.StoryboardVersionRecord, error) {
	var out application.StoryboardVersionRecord
	err := c.do(ctx, http.MethodGet, path, nil, &out)
	var re *application.RenderingError
	if errors.As(err, &re) && re.Status == http.StatusNotFound {
		return application.StoryboardVersionRecord{}, domain.ErrNoVersion
	}
	return out, err
}

// LatestStoryboard reads the latest locked storyboard, or domain.ErrNoVersion.
func (c *Client) LatestStoryboard(ctx context.Context, projectID string) (application.StoryboardVersionRecord, error) {
	return c.versionCall(ctx, storyboardPath(projectID)+"/versions/latest")
}

// StoryboardVersion reads one locked storyboard, or domain.ErrNoVersion.
func (c *Client) StoryboardVersion(ctx context.Context, projectID string, version int) (application.StoryboardVersionRecord, error) {
	return c.versionCall(ctx, fmt.Sprintf("%s/versions/%d", storyboardPath(projectID), version))
}

// PurgeStoryboard removes the video's assets, frames and versions.
func (c *Client) PurgeStoryboard(ctx context.Context, projectID string) error {
	return c.do(ctx, http.MethodDelete, storyboardPath(projectID), nil, nil)
}
