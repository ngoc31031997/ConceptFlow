package rendering

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"visual/internal/application"
	"visual/internal/domain"
)

// recorded is the last request a fake rendering saw.
type recorded struct {
	method, path, query string
	body                map[string]json.RawMessage
}

func fakeRendering(t *testing.T, status int, answer string) (*Client, *recorded) {
	t.Helper()
	rec := &recorded{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.method, rec.path, rec.query = r.Method, r.URL.Path, r.URL.RawQuery
		raw, _ := io.ReadAll(r.Body)
		rec.body = nil
		_ = json.Unmarshal(raw, &rec.body)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(answer))
	}))
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, time.Second), rec
}

func TestStoryboardClient_AssetsGoToTheProjectsAssetRoutes(t *testing.T) {
	ctx := context.Background()
	asset := `{"asset_id":"AST-ASTRO-001","concept_id":"astro","name":"Astro","kind":"figure","tier":"hero","version":2,"status":"drawn","to_library":true,"library_folder_id":"f","warnings":[{"message":"m","line":3}]}`
	c, rec := fakeRendering(t, 200, `{"assets":[`+asset+`]}`)
	list, err := c.ListAssets(ctx, "p1")
	if err != nil || len(list) != 1 || list[0].AssetID != "AST-ASTRO-001" || list[0].Version != 2 || !list[0].ToLibrary || list[0].Warnings[0].Message != "m" {
		t.Fatalf("%+v %v", list, err)
	}
	if rec.method != "GET" || rec.path != "/internal/v1/projects/p1/assets" {
		t.Errorf("%s %s", rec.method, rec.path)
	}
	c, rec = fakeRendering(t, 201, asset)
	if _, err := c.CreateAsset(ctx, "p1", application.AssetInput{Name: "Astro", Kind: "figure", MustNotImply: []string{}}); err != nil || rec.method != "POST" {
		t.Fatalf("create: %v %s", err, rec.method)
	}
	if string(rec.body["name"]) != `"Astro"` || string(rec.body["must_not_imply"]) != `[]` {
		t.Errorf("body %v", rec.body)
	}
	c, rec = fakeRendering(t, 200, asset)
	if _, err := c.UpdateAsset(ctx, "p1", "AST-ASTRO-001", application.AssetInput{Name: "Astro"}); err != nil || rec.method != "PUT" || rec.path != "/internal/v1/projects/p1/assets/AST-ASTRO-001" {
		t.Errorf("update: %v %s %s", err, rec.method, rec.path)
	}
	c, rec = fakeRendering(t, 204, ``)
	if err := c.DeleteAsset(ctx, "p1", "AST-ASTRO-001"); err != nil || rec.method != "DELETE" {
		t.Errorf("delete: %v %s", err, rec.method)
	}
	c, rec = fakeRendering(t, 200, asset)
	for action, call := range map[string]func() error{
		"approve": func() error {
			_, err := c.ApproveAsset(ctx, "p1", "AST-ASTRO-001", application.AssetApproveRequest{RefID: "R12"})
			return err
		},
		"unapprove": func() error { _, err := c.UnapproveAsset(ctx, "p1", "AST-ASTRO-001"); return err },
	} {
		if err := call(); err != nil || rec.path != "/internal/v1/projects/p1/assets/AST-ASTRO-001/"+action {
			t.Errorf("%s: %v %s", action, err, rec.path)
		}
	}
}

func TestStoryboardClient_AnAssetIsDrawnInThreeCallsThatCarryTheKitAndStyle(t *testing.T) {
	ctx := context.Background()
	draw := application.AssetDrawContext{Kit: json.RawMessage(`{"ink":"#000"}`), RefID: "R12"}
	base := "/internal/v1/projects/p1/assets/AST-ASTRO-001/"

	c, rec := fakeRendering(t, 200, `{"system":"S","user":"U"}`)
	line := 4
	prompt, err := c.AssetDrawPrompt(ctx, "p1", "AST-ASTRO-001", application.AssetDrawPromptRequest{AssetDrawContext: draw, Redraw: true, Note: "to hơn",
		FixError: &application.AssetDrawFix{Code: "x", Error: application.AssetDrawError{Kind: "syntax", Message: "m", Line: &line}}})
	if err != nil || prompt.System != "S" || prompt.User != "U" || rec.method != "POST" || rec.path != base+"draw-prompt" {
		t.Fatalf("draw-prompt: %+v %v %s %s", prompt, err, rec.method, rec.path)
	}
	if string(rec.body["kit"]) != `{"ink":"#000"}` || string(rec.body["ref_id"]) != `"R12"` || string(rec.body["redraw"]) != "true" || string(rec.body["note"]) != `"to hơn"` ||
		string(rec.body["fix_error"]) != `{"code":"x","error":{"kind":"syntax","message":"m","line":4,"snippet":null}}` {
		t.Errorf("draw-prompt body %v", rec.body)
	}
	c, rec = fakeRendering(t, 200, `{"system":"S","user":"U"}`)
	if _, err := c.AssetDrawPrompt(ctx, "p1", "AST-ASTRO-001", application.AssetDrawPromptRequest{AssetDrawContext: draw}); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"note", "redraw", "fix_error"} {
		if _, ok := rec.body[key]; ok {
			t.Errorf("a first drawing sends no %q: %v", key, rec.body)
		}
	}

	c, rec = fakeRendering(t, 200, `{"code":"export function Astro(){}","png_base64":"UE5H","warnings":[{"message":"w","line":2}]}`)
	preview, err := c.AssetDrawCheck(ctx, "p1", "AST-ASTRO-001", application.AssetDrawCheckRequest{AssetDrawContext: draw, Reply: "reply"})
	if err != nil || preview.Code != "export function Astro(){}" || preview.PNGBase64 != "UE5H" || len(preview.Warnings) != 1 || *preview.Warnings[0].Line != 2 {
		t.Fatalf("draw-check: %+v %v", preview, err)
	}
	if rec.path != base+"draw-check" || string(rec.body["reply"]) != `"reply"` || string(rec.body["ref_id"]) != `"R12"` {
		t.Errorf("draw-check: %s %v", rec.path, rec.body)
	}

	c, rec = fakeRendering(t, 200, `{"asset_id":"AST-ASTRO-001","version":3,"status":"drawn"}`)
	asset, err := c.AssetDrawApply(ctx, "p1", "AST-ASTRO-001", application.AssetDrawApplyRequest{AssetDrawContext: draw, Code: "code"})
	if err != nil || asset.Version != 3 || asset.Status != "drawn" || rec.path != base+"draw-apply" || string(rec.body["code"]) != `"code"` {
		t.Fatalf("draw-apply: %+v %v %s %v", asset, err, rec.path, rec.body)
	}
}

func TestStoryboardClient_RefusedDrawnCodeIsAStructuredError(t *testing.T) {
	ctx := context.Background()
	c, _ := fakeRendering(t, 422, `{"error":"Thiếu dấu đóng thẻ <g>.","detail":{"kind":"syntax","message":"Thiếu dấu đóng thẻ <g>.","line":12,"snippet":"12: > <g>"}}`)
	_, err := c.AssetDrawCheck(ctx, "p1", "AST-ASTRO-001", application.AssetDrawCheckRequest{Reply: "r"})
	var drawn *application.AssetDrawCheckError
	if !errors.As(err, &drawn) || drawn.Detail.Kind != "syntax" || *drawn.Detail.Line != 12 || *drawn.Detail.Snippet != "12: > <g>" || err.Error() != "Thiếu dấu đóng thẻ <g>." {
		t.Fatalf("draw-check 422: %v", err)
	}
	c, _ = fakeRendering(t, 422, `{"error":"x","detail":{"kind":"no_code","message":"Không có code.","line":null,"snippet":null}}`)
	_, err = c.AssetDrawApply(ctx, "p1", "AST-ASTRO-001", application.AssetDrawApplyRequest{Code: "c"})
	if !errors.As(err, &drawn) || drawn.Detail.Line != nil || drawn.Detail.Snippet != nil {
		t.Fatalf("draw-apply 422: %v", err)
	}
	// A 422 without the structured detail, and the other refusals, stay rendering's refusal.
	for status, body := range map[int]string{422: `{"error":"thư mục Hình mẫu"}`, 409: `{"error":"chưa có chi tiết","code":"asset_no_detail"}`, 503: `{"error":"bộ dựng không chạy"}`} {
		c, _ = fakeRendering(t, status, body)
		_, err := c.AssetDrawPrompt(ctx, "p1", "AST-ASTRO-001", application.AssetDrawPromptRequest{})
		var re *application.RenderingError
		if !errors.As(err, &re) || re.Status != status || errors.As(err, &drawn) {
			t.Errorf("%d: want a RenderingError, got %v", status, err)
		}
		if status == 409 && string(re.Extra["code"]) != `"asset_no_detail"` {
			t.Errorf("409 keeps rendering's code: %v", re.Extra)
		}
	}
}

func TestStoryboardClient_ALibraryOfApprovedDrawings(t *testing.T) {
	c, rec := fakeRendering(t, 200, `{"illustrations":[{"id":"i1","kind":"figure","name":"Nguoi","title":"t","tags":["a"],"description":"d","detail_level":"rich","code":"x"}]}`)
	got, err := c.ApprovedLibrary(context.Background())
	if err != nil || len(got) != 1 || got[0].ID != "i1" || got[0].DetailLevel != "rich" || rec.path != "/v1/illustrations" || rec.query != "status=approved" {
		t.Fatalf("%+v %v %s?%s", got, err, rec.path, rec.query)
	}
}

func TestStoryboardClient_FramesAndVersions(t *testing.T) {
	ctx := context.Background()
	c, rec := fakeRendering(t, 200, `{"frame_path":"storyboard/SH-001.png","frame_hash":"h","spec_hash":"abc","error":"","cache_hit":true}`)
	res, err := c.DrawFrame(ctx, "p1", application.FrameRequest{FrameSpec: domain.FrameSpec{ShotID: "SH-001", Kit: json.RawMessage(`{}`), Placements: []domain.FramePlacement{{AssetID: "A", Role: "subject", Position: "center", Version: 3}}},
		UnitID: "VP-001", BeatID: "b1", SpecHash: "abc"})
	if err != nil || res.FramePath != "storyboard/SH-001.png" || !res.CacheHit || rec.path != "/internal/v1/projects/p1/storyboard/frames" {
		t.Fatalf("%+v %v %s", res, err, rec.path)
	}
	for _, key := range []string{"shot_id", "unit_id", "beat_id", "spec_hash", "kit", "size", "background", "placements", "camera_view", "on_screen_text", "text_color"} {
		if _, ok := rec.body[key]; !ok {
			t.Errorf("the frame body lacks %q: %v", key, rec.body)
		}
	}
	if _, ok := rec.body["scale"]; ok {
		t.Error("the spec has no scale: rendering decides the size")
	}
	if string(rec.body["placements"]) != `[{"asset_id":"A","role":"subject","position":"center"}]` {
		t.Errorf("a placement carries no version: %s", rec.body["placements"])
	}

	c, rec = fakeRendering(t, 201, `{"project_id":"p1","version":1,"sequence_version":1,"note":"n","locked_at":"2026-11-20T10:00:00Z","document":{"shots":[]}}`)
	v, err := c.LockStoryboard(ctx, "p1", json.RawMessage(`{"shots":[]}`), 1, "n")
	if err != nil || v.Version != 1 || v.LockedAt.IsZero() || rec.path != "/internal/v1/projects/p1/storyboard/versions" || string(rec.body["sequence_version"]) != "1" {
		t.Fatalf("%+v %v %s %v", v, err, rec.path, rec.body)
	}
	c, rec = fakeRendering(t, 200, `{"version":3,"locked_at":"2026-11-20T10:00:00Z","document":{}}`)
	if v, err := c.LatestStoryboard(ctx, "p1"); err != nil || v.Version != 3 || rec.path != "/internal/v1/projects/p1/storyboard/versions/latest" {
		t.Errorf("latest: %+v %v %s", v, err, rec.path)
	}
	if v, err := c.StoryboardVersion(ctx, "p1", 2); err != nil || rec.path != "/internal/v1/projects/p1/storyboard/versions/2" {
		t.Errorf("one version: %+v %v %s", v, err, rec.path)
	}
	c, _ = fakeRendering(t, 404, `{"error":"chưa có bản chốt"}`)
	if _, err := c.LatestStoryboard(ctx, "p1"); !errors.Is(err, domain.ErrNoVersion) {
		t.Errorf("no locked storyboard: want ErrNoVersion, got %v", err)
	}
	c, rec = fakeRendering(t, 200, "\x89PNG")
	if png, err := c.FrameImage(ctx, "p1", "SH-001"); err != nil || string(png) != "\x89PNG" || rec.path != "/internal/v1/projects/p1/storyboard/frames/SH-001" {
		t.Errorf("image: %q %v %s", png, err, rec.path)
	}
	c, rec = fakeRendering(t, 200, "\x89PNG")
	if png, err := c.AssetPreview(ctx, "p1", "AST-A-001"); err != nil || string(png) != "\x89PNG" || rec.path != "/internal/v1/projects/p1/assets/AST-A-001/preview" {
		t.Errorf("asset preview: %q %v %s", png, err, rec.path)
	}
	c, _ = fakeRendering(t, 404, `{"error":"chưa có ảnh"}`)
	var re *application.RenderingError
	if _, err := c.AssetPreview(ctx, "p1", "AST-A-001"); !errors.As(err, &re) || re.Status != 404 {
		t.Errorf("no drawing: want rendering's 404, got %v", err)
	}
	c, rec = fakeRendering(t, 204, ``)
	if err := c.PurgeStoryboard(ctx, "p1"); err != nil || rec.method != "DELETE" || rec.path != "/internal/v1/projects/p1/storyboard" {
		t.Errorf("purge: %v %s %s", err, rec.method, rec.path)
	}
}

func TestStoryboardClient_ARefusalKeepsItsStatusReasonAndKeys(t *testing.T) {
	c, _ := fakeRendering(t, 409, `{"error":"tài sản chưa dùng được","assets":[{"asset_id":"AST-A-001","reason":"chưa duyệt"}]}`)
	_, err := c.DrawFrame(context.Background(), "p1", application.FrameRequest{})
	var re *application.RenderingError
	if !errors.As(err, &re) || re.Status != 409 || re.Message != "tài sản chưa dùng được" || re.Extra["assets"] == nil {
		t.Fatalf("%+v", re)
	}
	if errors.Is(err, application.ErrRenderingFailed) {
		t.Error("a refusal is an answer, not a failure to reach rendering")
	}
	c, _ = fakeRendering(t, 502, `<html>bad gateway</html>`)
	if _, err := c.ListAssets(context.Background(), "p1"); !errors.As(err, &re) || re.Status != 502 || re.Message == "" {
		t.Errorf("a body that is not JSON still carries its text: %v", err)
	}
}

func TestStoryboardClient_NoAnswerNamesRendering(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	dead := NewClient(srv.URL, time.Second)
	srv.Close()
	if _, err := dead.ListAssets(context.Background(), "p1"); !errors.Is(err, application.ErrRenderingFailed) {
		t.Errorf("want ErrRenderingFailed, got %v", err)
	}
	c, _ := fakeRendering(t, 200, `not json`)
	if _, err := c.ListAssets(context.Background(), "p1"); !errors.Is(err, application.ErrRenderingFailed) {
		t.Errorf("an unreadable answer: want ErrRenderingFailed, got %v", err)
	}
}
