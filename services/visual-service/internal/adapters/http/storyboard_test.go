package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"visual/internal/application"
	"visual/internal/domain"
)

// fakeStoryboard answers every call with err, or a screen whose PendingUnits
// names the call.
type fakeStoryboard struct {
	err  error
	last string
}

func (f *fakeStoryboard) ret(call string) (*application.StoryboardScreen, error) {
	f.last = call
	if f.err != nil {
		return nil, f.err
	}
	return &application.StoryboardScreen{PendingUnits: []string{call}}, nil
}
func (f *fakeStoryboard) State(context.Context, string) (*application.StoryboardScreen, error) {
	return f.ret("state")
}
func (f *fakeStoryboard) SetModel(_ context.Context, _ string, m string) (*application.StoryboardScreen, error) {
	return f.ret("model " + m)
}
func (f *fakeStoryboard) GenerateAssets(_ context.Context, _ string, note string) (*application.StoryboardScreen, error) {
	return f.ret("assets " + note)
}
func (f *fakeStoryboard) GenerateAssetDetail(_ context.Context, _ string, id, note string) (*application.StoryboardScreen, error) {
	return f.ret("detail " + id + " " + note)
}
func (f *fakeStoryboard) GenerateAllAssetDetails(_ context.Context, _ string, note string) (*application.StoryboardScreen, error) {
	return f.ret("details " + note)
}
func (f *fakeStoryboard) AddAsset(_ context.Context, _ string, fields map[string]json.RawMessage) (*application.StoryboardScreen, error) {
	return f.ret(fmt.Sprintf("add-asset:%d", len(fields)))
}
func (f *fakeStoryboard) EditAsset(_ context.Context, _ string, id string, fields map[string]json.RawMessage) (*application.StoryboardScreen, error) {
	return f.ret(fmt.Sprintf("edit-asset:%s:%d", id, len(fields)))
}
func (f *fakeStoryboard) DeleteAsset(_ context.Context, _ string, id string) (*application.StoryboardScreen, error) {
	return f.ret("delete-asset:" + id)
}
func (f *fakeStoryboard) DrawAsset(_ context.Context, _ string, id string) (*application.StoryboardScreen, error) {
	return f.ret("draw:" + id)
}
func (f *fakeStoryboard) RedrawAsset(_ context.Context, _ string, id, note string) (*application.StoryboardScreen, error) {
	return f.ret("redraw:" + id + ":" + note)
}
func (f *fakeStoryboard) ApproveAsset(_ context.Context, _ string, id, name string) (*application.StoryboardScreen, error) {
	return f.ret("approve-asset:" + id + ":" + name)
}
func (f *fakeStoryboard) UnapproveAsset(_ context.Context, _ string, id string) (*application.StoryboardScreen, error) {
	return f.ret("unapprove-asset:" + id)
}
func (f *fakeStoryboard) ApproveAssets(context.Context, string) (*application.StoryboardScreen, error) {
	return f.ret("approve-assets")
}
func (f *fakeStoryboard) UnapproveAssets(context.Context, string) (*application.StoryboardScreen, error) {
	return f.ret("unapprove-assets")
}
func (f *fakeStoryboard) Generate(_ context.Context, _ string, note string) (*application.StoryboardScreen, error) {
	return f.ret("generate " + note)
}
func (f *fakeStoryboard) GenerateUnit(_ context.Context, _ string, id, note string) (*application.StoryboardScreen, error) {
	return f.ret("unit " + id + " " + note)
}
func (f *fakeStoryboard) AddShot(_ context.Context, _ string, unit string, fields map[string]json.RawMessage) (*application.StoryboardScreen, error) {
	return f.ret(fmt.Sprintf("add-shot:%s:%d", unit, len(fields)))
}
func (f *fakeStoryboard) EditShot(_ context.Context, _ string, unit, shot string, fields map[string]json.RawMessage) (*application.StoryboardScreen, error) {
	return f.ret(fmt.Sprintf("edit-shot:%s:%s:%d", unit, shot, len(fields)))
}
func (f *fakeStoryboard) DeleteShot(_ context.Context, _ string, unit, shot string) (*application.StoryboardScreen, error) {
	return f.ret("delete-shot:" + unit + ":" + shot)
}
func (f *fakeStoryboard) DrawFrame(_ context.Context, _ string, unit, shot string) (*application.StoryboardScreen, error) {
	return f.ret("frame:" + unit + ":" + shot)
}
func (f *fakeStoryboard) ApproveShot(_ context.Context, _ string, unit, shot string) (*application.StoryboardScreen, error) {
	return f.ret("approve-shot:" + unit + ":" + shot)
}
func (f *fakeStoryboard) UnapproveShot(_ context.Context, _ string, unit, shot string) (*application.StoryboardScreen, error) {
	return f.ret("unapprove-shot:" + unit + ":" + shot)
}
func (f *fakeStoryboard) ApproveAll(context.Context, string) (*application.StoryboardScreen, error) {
	return f.ret("approve-all")
}
func (f *fakeStoryboard) Review(_ context.Context, _ string, note string) (*application.StoryboardScreen, error) {
	return f.ret("review " + note)
}
func (f *fakeStoryboard) ApplyFinding(_ context.Context, _ string, id string) (*application.StoryboardScreen, error) {
	return f.ret("apply:" + id)
}
func (f *fakeStoryboard) AcceptFinding(_ context.Context, _ string, id, why string) (*application.StoryboardScreen, error) {
	return f.ret("accept:" + id + ":" + why)
}
func (f *fakeStoryboard) Lock(_ context.Context, _ string, note string) (*application.StoryboardScreen, error) {
	return f.ret("lock:" + note)
}
func (f *fakeStoryboard) Version(_ context.Context, _ string, n int) (application.StoryboardVersionRecord, error) {
	f.last = fmt.Sprintf("version:%d", n)
	return application.StoryboardVersionRecord{Version: n}, f.err
}
func (f *fakeStoryboard) LatestDocument(context.Context, string) (json.RawMessage, error) {
	f.last = "latest"
	return json.RawMessage(`{"document_id":"storyboard-v1"}`), f.err
}
func (f *fakeStoryboard) AssetPreview(_ context.Context, _ string, id string) ([]byte, error) {
	f.last = "preview:" + id
	return []byte("\x89PNG"), f.err
}
func (f *fakeStoryboard) FrameImage(_ context.Context, _ string, shot string) ([]byte, error) {
	f.last = "image:" + shot
	return []byte("\x89PNG"), f.err
}

func storyboardRequest(f *fakeStoryboard, method, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	NewRouter(&fakeDelete{}).WithStoryboard(f).Handler().ServeHTTP(rec, req)
	return rec
}

func TestStoryboardRoutesReachTheDesk(t *testing.T) {
	const base = "/v1/projects/p1/storyboard"
	for _, c := range []struct{ method, path, body, want string }{
		{"GET", base + "/", "", "state"},
		{"PUT", base + "/model", `{"model":"deepseek-chat"}`, "model deepseek-chat"},
		{"POST", base + "/assets/generate", `{"note":"n"}`, "assets n"},
		{"POST", base + "/assets/AST-ASTRO-001/detail", `{"note":"n"}`, "detail AST-ASTRO-001 n"},
		{"POST", base + "/assets/detail", `{"note":"n"}`, "details n"},
		{"POST", base + "/assets", `{"fields":{"name":"Astro"}}`, "add-asset:1"},
		{"POST", base + "/assets/approve", "", "approve-assets"},
		{"POST", base + "/assets/unapprove", "", "unapprove-assets"},
		{"PUT", base + "/assets/AST-ASTRO-001", `{"fields":{"portrayal":"x","tier":"hero"}}`, "edit-asset:AST-ASTRO-001:2"},
		{"DELETE", base + "/assets/AST-ASTRO-001", "", "delete-asset:AST-ASTRO-001"},
		{"POST", base + "/assets/AST-ASTRO-001/draw", "", "draw:AST-ASTRO-001"},
		{"POST", base + "/assets/AST-ASTRO-001/redraw", `{"note":"to hơn"}`, "redraw:AST-ASTRO-001:to hơn"},
		{"POST", base + "/assets/AST-ASTRO-001/approve", "", "approve-asset:AST-ASTRO-001:"},
		{"POST", base + "/assets/AST-ASTRO-001/approve", `{"project_name":"Hố đen"}`, "approve-asset:AST-ASTRO-001:Hố đen"},
		{"GET", base + "/assets/AST-ASTRO-001/preview", "", "preview:AST-ASTRO-001"},
		{"POST", base + "/assets/AST-ASTRO-001/unapprove", "", "unapprove-asset:AST-ASTRO-001"},
		{"POST", base + "/generate", `{"note":"n"}`, "generate n"},
		{"POST", base + "/units/VP-003/generate", `{"note":"n"}`, "unit VP-003 n"},
		{"POST", base + "/units/VP-001/shots", `{"fields":{"shot_purpose":"x"}}`, "add-shot:VP-001:1"},
		{"PUT", base + "/units/VP-001/shots/SH-002", `{"fields":{"shot_purpose":"x","notes":[]}}`, "edit-shot:VP-001:SH-002:2"},
		{"DELETE", base + "/units/VP-001/shots/SH-002", "", "delete-shot:VP-001:SH-002"},
		{"POST", base + "/units/VP-001/shots/SH-002/frame", "", "frame:VP-001:SH-002"},
		{"POST", base + "/units/VP-001/shots/SH-002/approve", "", "approve-shot:VP-001:SH-002"},
		{"POST", base + "/units/VP-001/shots/SH-002/unapprove", "", "unapprove-shot:VP-001:SH-002"},
		{"POST", base + "/approve-all", "", "approve-all"},
		{"POST", base + "/review", `{"note":"n"}`, "review n"},
		{"POST", base + "/findings/f1/apply", "", "apply:f1"},
		{"POST", base + "/findings/m-asset-unused/accept", `{"why":"cố ý"}`, "accept:m-asset-unused:cố ý"},
		{"POST", base + "/lock", `{"note":"v1"}`, "lock:v1"},
		{"GET", base + "/versions/2", "", "version:2"},
		{"GET", base + "/frames/SH-004", "", "image:SH-004"},
		{"GET", "/internal/v1/visual/p1/storyboard", "", "latest"},
	} {
		f := &fakeStoryboard{}
		rec := storyboardRequest(f, c.method, c.path, c.body)
		if rec.Code != 200 || f.last != c.want {
			t.Errorf("%s %s: status %d, call %q, want 200 and %q: %s", c.method, c.path, rec.Code, f.last, c.want, rec.Body.String())
		}
	}
}

func TestStoryboardFrameRouteServesThePNG(t *testing.T) {
	rec := storyboardRequest(&fakeStoryboard{}, "GET", "/v1/projects/p1/storyboard/frames/SH-001", "")
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "image/png" || rec.Body.String() != "\x89PNG" {
		t.Errorf("status %d type %q body %q", rec.Code, rec.Header().Get("Content-Type"), rec.Body.String())
	}
	rec = storyboardRequest(&fakeStoryboard{err: &application.RenderingError{Status: 404, Message: "chưa vẽ"}}, "GET", "/v1/projects/p1/storyboard/frames/SH-001", "")
	if rec.Code != 404 {
		t.Errorf("a frame not drawn: want 404, got %d", rec.Code)
	}
}

func TestStoryboardAssetPreviewRoute(t *testing.T) {
	const path = "/v1/projects/p1/storyboard/assets/AST-A-001/preview"
	rec := storyboardRequest(&fakeStoryboard{}, "GET", path, "")
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "image/png" || rec.Header().Get("Cache-Control") != "private, no-cache" {
		t.Errorf("status %d headers %v", rec.Code, rec.Header())
	}
	if rec := storyboardRequest(&fakeStoryboard{err: &application.RenderingError{Status: 404, Message: "chưa có ảnh"}}, "GET", path, ""); rec.Code != 404 {
		t.Errorf("no drawing: want 404, got %d", rec.Code)
	}
}

func TestStoryboardRoutesMapErrors(t *testing.T) {
	const base = "/v1/projects/p1/storyboard"
	for _, c := range []struct {
		err    error
		status int
		code   string
	}{
		{domain.ErrSequenceNotLocked, 409, "sequence_not_locked"},
		{domain.ErrSequenceOutdated, 409, "sequence_outdated"},
		{domain.ErrPlanNotLocked, 409, "plan_not_locked"},
		{domain.ErrNoKit, 409, "no_kit"},
		{domain.ErrAssetsMissing, 409, "assets_missing"},
		{domain.ErrAssetInUse, 409, "asset_in_use"},
		{domain.ErrAssetNoDetail, 409, "asset_no_detail"},
		{domain.ErrLastShot, 409, "last_shot"},
		{domain.ErrLockNotReady, 409, ""},
		{domain.ErrNothingToLock, 409, ""},
		{domain.ErrUnitInvalid, 409, ""},
		{domain.ErrUnknownAsset, 404, ""},
		{domain.ErrUnknownShot, 404, ""},
		{domain.ErrUnknownPlanUnit, 404, ""},
		{domain.ErrInvalidEdit, 400, ""},
		{domain.ErrUnreadableReply, 502, ""},
		{application.ErrRenderingFailed, 502, ""},
		{application.ErrLLMNotConfigured, 502, ""},
	} {
		rec := storyboardRequest(&fakeStoryboard{err: c.err}, "POST", base+"/lock", `{}`)
		var body struct{ Error, Code string }
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		if rec.Code != c.status || body.Code != c.code || body.Error == "" {
			t.Errorf("%v: want %d %q, got %d %+v", c.err, c.status, c.code, rec.Code, body)
		}
	}
	if rec := storyboardRequest(&fakeStoryboard{}, "GET", base+"/versions/x", ""); rec.Code != 400 {
		t.Errorf("a version that is not a number: want 400, got %d", rec.Code)
	}
	if rec := storyboardRequest(&fakeStoryboard{}, "PUT", base+"/units/VP-001/shots/SH-001", `not json`); rec.Code != 400 {
		t.Errorf("a body that is not JSON: want 400, got %d", rec.Code)
	}
	if rec := storyboardRequest(&fakeStoryboard{err: domain.ErrNoVersion}, "GET", "/internal/v1/visual/p1/storyboard", ""); rec.Code != 404 {
		t.Errorf("no locked storyboard: want 404, got %d", rec.Code)
	}
}

func TestStoryboardRoutesPassARenderingRefusalOnWithItsStatusAndKeys(t *testing.T) {
	const base = "/v1/projects/p1/storyboard"
	refusal := func(status int, extra map[string]json.RawMessage) error {
		return fmt.Errorf("AST-X: %w", &application.RenderingError{Status: status, Message: "lý do", Extra: extra})
	}
	for _, c := range []struct {
		err    error
		status int
		code   string
		key    string
	}{
		{refusal(409, map[string]json.RawMessage{"assets": json.RawMessage(`[{"asset_id":"AST-A-001","reason":"chưa duyệt"}]`)}), 409, "assets_not_ready", "assets"},
		{refusal(409, map[string]json.RawMessage{"missing_frames": json.RawMessage(`["SH-002"]`)}), 409, "missing_frames", "missing_frames"},
		{refusal(400, nil), 400, "", ""},
		{refusal(422, nil), 422, "", ""},
		{refusal(503, nil), 503, "", ""},
		{refusal(500, nil), 502, "", ""},
	} {
		rec := storyboardRequest(&fakeStoryboard{err: c.err}, "POST", base+"/lock", `{}`)
		var body map[string]json.RawMessage
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		var code string
		_ = json.Unmarshal(body["code"], &code)
		if rec.Code != c.status || code != c.code || (c.key != "" && body[c.key] == nil) || !strings.Contains(string(body["error"]), "lý do") {
			t.Errorf("%v: want %d %q, got %d %s", c.err, c.status, c.code, rec.Code, rec.Body.String())
		}
	}
}

func TestStoryboardAIRoutesTakeTheExternalAIHeader(t *testing.T) {
	const base = "/v1/projects/p1/storyboard"
	for _, path := range []string{"/assets/generate", "/assets/AST-A-001/detail", "/units/VP-001/generate", "/review"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", base+path, strings.NewReader(`{}`))
		req.Header.Set("X-External-AI", "nope")
		NewRouter(&fakeDelete{}).WithStoryboard(&fakeStoryboard{}).Handler().ServeHTTP(rec, req)
		if rec.Code != 400 {
			t.Errorf("%s with a bad X-External-AI: want 400, got %d", path, rec.Code)
		}
	}
	// Asset drawing and frame drawing are the system's: the header changes nothing.
	f := &fakeStoryboard{}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", base+"/assets/AST-A-001/draw", strings.NewReader(`{}`))
	req.Header.Set("X-External-AI", "nope")
	NewRouter(&fakeDelete{}).WithStoryboard(f).Handler().ServeHTTP(rec, req)
	if rec.Code != 200 || f.last != "draw:AST-A-001" {
		t.Errorf("drawing has no external mode: %d %q", rec.Code, f.last)
	}
	// Writing every detail at once is the system's too: one external prompt cannot answer many turns.
	f = &fakeStoryboard{}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest("POST", base+"/assets/detail", strings.NewReader(`{}`))
	req.Header.Set("X-External-AI", "nope")
	NewRouter(&fakeDelete{}).WithStoryboard(f).Handler().ServeHTTP(rec, req)
	if rec.Code != 200 || f.last != "details " {
		t.Errorf("writing every detail has no external mode: %d %q", rec.Code, f.last)
	}
}
