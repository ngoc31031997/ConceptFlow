package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http/httptest"
	"testing"

	"orchestrator/internal/application"
	"orchestrator/internal/domain"
)

type fakeStartRenderSaga struct {
	out *application.StartRenderSagaOutput
	err error
}

func (f *fakeStartRenderSaga) Execute(_ context.Context, _ application.StartRenderSagaInput) (*application.StartRenderSagaOutput, error) {
	return f.out, f.err
}

type fakeStartPublishSaga struct {
	out *application.StartPublishSagaOutput
	err error
}

func (f *fakeStartPublishSaga) Execute(_ context.Context, _ application.StartPublishSagaInput) (*application.StartPublishSagaOutput, error) {
	return f.out, f.err
}

type fakeRetryStep struct {
	out *application.RetryStepOutput
	err error
}

func (f *fakeRetryStep) Execute(_ context.Context, _ string) (*application.RetryStepOutput, error) {
	return f.out, f.err
}

type fakeChannelAssets struct {
	normalizeErr error
	normalizedIn application.NormalizeChannelAssetInput
	pointers     []domain.ChannelAssetPointer
	previewErr   error
}

func (f *fakeChannelAssets) Normalize(_ context.Context, in application.NormalizeChannelAssetInput) error {
	f.normalizedIn = in
	return f.normalizeErr
}

func (f *fakeChannelAssets) Preview(_ context.Context) ([]domain.ChannelAssetPointer, error) {
	return f.pointers, f.previewErr
}

type fakeProjectReader struct {
	project      *domain.Project
	err          error
	listOut      []domain.ProjectSummary
	listErr      error
	deleteErr    error
	calibrations []domain.VoiceCalibration
	savedFormat  *domain.VideoFormat
}

func (f *fakeProjectReader) ListVideoFormats(_ context.Context) ([]domain.VideoFormat, error) {
	return domain.BuiltinFormats(), nil
}

func (f *fakeProjectReader) SaveVideoFormat(_ context.Context, format domain.VideoFormat) (domain.VideoFormat, error) {
	f.savedFormat = &format
	format.Version = 7
	return format, nil
}

func (f *fakeProjectReader) ListVoiceCalibrations(_ context.Context) ([]domain.VoiceCalibration, error) {
	return f.calibrations, nil
}

func (f *fakeProjectReader) Get(_ context.Context, _ string) (*domain.Project, error) {
	return f.project, f.err
}

func (f *fakeProjectReader) List(_ context.Context) ([]domain.ProjectSummary, error) {
	return f.listOut, f.listErr
}

func (f *fakeProjectReader) Delete(_ context.Context, _ string) error {
	return f.deleteErr
}

func (f *fakeProjectReader) Save(_ context.Context, project *domain.Project) error {
	f.project = project
	return nil
}

type fakeSuggestMetadata struct {
	out *application.SuggestPublishMetadataOutput
	err error
}

func (f *fakeSuggestMetadata) Execute(_ context.Context, _ string) (*application.SuggestPublishMetadataOutput, error) {
	return f.out, f.err
}

func TestHandleStartRenderSaga_Created(t *testing.T) {
	router := NewRouter(
		&fakeStartRenderSaga{out: &application.StartRenderSagaOutput{SagaID: "saga-1", Status: domain.StatusParsingScript}},
		&fakeStartPublishSaga{}, &fakeRetryStep{}, &fakeProjectReader{}, &fakeSuggestMetadata{}, nil, nil)

	body, _ := json.Marshal(map[string]interface{}{
		"project_id": "p1", "script_content": "s", "plugin_id": "plugin", "category_hint": "concept", "voice_language": "en",
	})
	req := httptest.NewRequest("POST", "/v1/sagas/render", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)

	if rec.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp sagaStartedResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.SagaID != "saga-1" || resp.Status != string(domain.StatusParsingScript) {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestHandleStartRenderSaga_InvalidBody(t *testing.T) {
	router := NewRouter(&fakeStartRenderSaga{}, &fakeStartPublishSaga{}, &fakeRetryStep{}, &fakeProjectReader{}, &fakeSuggestMetadata{}, nil, nil)

	req := httptest.NewRequest("POST", "/v1/sagas/render", bytes.NewReader([]byte(`{"project_id":""}`)))
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)

	if rec.Code != 400 {
		t.Fatalf("expected 400 for missing required fields, got %d", rec.Code)
	}
}

func TestHandleStartPublishSaga_Conflict(t *testing.T) {
	router := NewRouter(
		&fakeStartRenderSaga{}, &fakeStartPublishSaga{err: domain.ErrInvalidStatus}, &fakeRetryStep{}, &fakeProjectReader{}, &fakeSuggestMetadata{}, nil, nil)

	body, _ := json.Marshal(map[string]interface{}{"project_id": "p1", "youtube_title": "t", "visibility": "public"})
	req := httptest.NewRequest("POST", "/v1/sagas/publish", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)

	if rec.Code != 409 {
		t.Fatalf("expected 409, got %d", rec.Code)
	}
}

func TestHandleGetProject_NotFound(t *testing.T) {
	router := NewRouter(
		&fakeStartRenderSaga{}, &fakeStartPublishSaga{}, &fakeRetryStep{},
		&fakeProjectReader{err: domain.ErrProjectNotFound}, &fakeSuggestMetadata{}, nil, nil)

	req := httptest.NewRequest("GET", "/v1/projects/unknown", nil)
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)

	if rec.Code != 404 {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleGetProject_OK(t *testing.T) {
	router := NewRouter(
		&fakeStartRenderSaga{}, &fakeStartPublishSaga{}, &fakeRetryStep{},
		&fakeProjectReader{project: &domain.Project{ProjectID: "p1", Status: domain.StatusDraft}}, &fakeSuggestMetadata{}, nil, nil)

	req := httptest.NewRequest("GET", "/v1/projects/p1", nil)
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

// TestHandleGetProject_CarriesFieldsNeededToRerenderAtAnotherQuality is the
// bug report's second half: the GUI resubmits POST /v1/sagas/render for the
// same project_id at a higher quality once a video is finalized, and can only
// do that with settings it can read back from here.
func TestHandleGetProject_CarriesFieldsNeededToRerenderAtAnotherQuality(t *testing.T) {
	musicPath := "/shared/p1/music/bg.mp3"
	router := NewRouter(
		&fakeStartRenderSaga{}, &fakeStartPublishSaga{}, &fakeRetryStep{},
		&fakeProjectReader{project: &domain.Project{
			ProjectID: "p1", Status: domain.StatusPublished,
			ScriptContent: "from conceptflow import *\n...", BackgroundMusicPath: &musicPath,
			BackgroundMusicVolume: 0.3,
		}}, &fakeSuggestMetadata{}, nil, nil)

	req := httptest.NewRequest("GET", "/v1/projects/p1", nil)
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if body["script_content"] != "from conceptflow import *\n..." {
		t.Fatalf("expected script_content in response, got %v", body["script_content"])
	}
	if body["background_music_path"] != musicPath {
		t.Fatalf("expected background_music_path in response, got %v", body["background_music_path"])
	}
	if body["background_music_volume"] != 0.3 {
		t.Fatalf("expected background_music_volume in response, got %v", body["background_music_volume"])
	}
}

func TestHandleRetry_OK(t *testing.T) {
	router := NewRouter(
		&fakeStartRenderSaga{}, &fakeStartPublishSaga{},
		&fakeRetryStep{out: &application.RetryStepOutput{SagaID: "saga-1", Status: domain.StatusRendering}},
		&fakeProjectReader{}, &fakeSuggestMetadata{}, nil, nil)

	req := httptest.NewRequest("POST", "/v1/projects/p1/retry", nil)
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleListProjects_OK(t *testing.T) {
	router := NewRouter(
		&fakeStartRenderSaga{}, &fakeStartPublishSaga{}, &fakeRetryStep{},
		&fakeProjectReader{listOut: []domain.ProjectSummary{{ProjectID: "p1", Status: domain.StatusFailedRenderScenes}}}, &fakeSuggestMetadata{}, nil, nil)

	req := httptest.NewRequest("GET", "/v1/projects", nil)
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp projectListResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if len(resp.Projects) != 1 || resp.Projects[0].ProjectID != "p1" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestHandleDeleteProject_NoContent(t *testing.T) {
	router := NewRouter(
		&fakeStartRenderSaga{}, &fakeStartPublishSaga{}, &fakeRetryStep{}, &fakeProjectReader{}, &fakeSuggestMetadata{}, nil, nil)

	req := httptest.NewRequest("DELETE", "/v1/projects/p1", nil)
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)

	if rec.Code != 204 {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleDeleteProject_NotFound(t *testing.T) {
	router := NewRouter(
		&fakeStartRenderSaga{}, &fakeStartPublishSaga{}, &fakeRetryStep{},
		&fakeProjectReader{deleteErr: domain.ErrProjectNotFound}, &fakeSuggestMetadata{}, nil, nil)

	req := httptest.NewRequest("DELETE", "/v1/projects/unknown", nil)
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)

	if rec.Code != 404 {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleHealth(t *testing.T) {
	router := NewRouter(&fakeStartRenderSaga{}, &fakeStartPublishSaga{}, &fakeRetryStep{}, &fakeProjectReader{}, &fakeSuggestMetadata{}, nil, nil)

	req := httptest.NewRequest("GET", "/health", nil)
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestHandleVoiceCalibration_OmitsVoicesBelowThreshold(t *testing.T) {
	// Giọng chưa đủ mẫu bị bỏ hẳn khỏi kết quả chứ không trả về một con số
	// độ tin cậy thấp: GUI rơi về hằng số theo ngôn ngữ, và đó là câu trả lời
	// trung thực hơn (CR-016 FR43.2).
	reader := &fakeProjectReader{calibrations: []domain.VoiceCalibration{
		{VoiceID: "vi-Enough", SampleCount: domain.MinCalibrationSamples, TotalWords: 900, TotalSecond: 360},
		{VoiceID: "vi-TooFew", SampleCount: 1, TotalWords: 300, TotalSecond: 120},
	}}
	router := NewRouter(nil, nil, nil, reader, nil, nil, nil)

	req := httptest.NewRequest("GET", "/v1/voice-calibration", nil)
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	var body struct {
		WordsPerMinute map[string]float64 `json:"words_per_minute"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body không phải JSON: %v", err)
	}
	if _, ok := body.WordsPerMinute["vi-TooFew"]; ok {
		t.Error("giọng chưa đủ mẫu không được xuất hiện")
	}
	if got := body.WordsPerMinute["vi-Enough"]; math.Abs(got-150) > 1e-9 {
		t.Errorf("WPM = %v, muốn 150", got)
	}
}

func TestHandleSaveFormat_RejectsAFormatWithNoBeats(t *testing.T) {
	reader := &fakeProjectReader{}
	router := NewRouter(nil, nil, nil, reader, nil, nil, nil)

	body := []byte(`{"id":"x","name":"X","min_seconds":60,"max_seconds":120,"beats":[]}`)
	req := httptest.NewRequest("POST", "/v1/formats", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)

	if rec.Code != 400 {
		t.Fatalf("status = %d", rec.Code)
	}
	if reader.savedFormat != nil {
		t.Error("không được lưu format rỗng beat")
	}
}

func TestHandleSaveFormat_StoresAsANewVersion(t *testing.T) {
	// FR51.6: không bao giờ ghi đè — project dựng theo version 3 phải tiếp tục
	// báo đúng beat của version 3.
	reader := &fakeProjectReader{}
	router := NewRouter(nil, nil, nil, reader, nil, nil, nil)

	body := []byte(`{"id":"visual_first_7min","name":"Của tôi","min_seconds":300,"max_seconds":480,
	                 "beats":[{"id":"hook","role":"hook","min_seconds":5,"max_seconds":12,"required":true,"max_repeat":1}]}`)
	req := httptest.NewRequest("POST", "/v1/formats", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)

	if rec.Code != 201 {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var saved domain.VideoFormat
	if err := json.Unmarshal(rec.Body.Bytes(), &saved); err != nil {
		t.Fatalf("body không phải JSON: %v", err)
	}
	if saved.Version != 7 {
		t.Errorf("phải trả về phiên bản mới, có %d", saved.Version)
	}
}

func TestHandleListFormats_ServesTheBuiltins(t *testing.T) {
	router := NewRouter(nil, nil, nil, &fakeProjectReader{}, nil, nil, nil)
	req := httptest.NewRequest("GET", "/v1/formats", nil)
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d", rec.Code)
	}
	var body struct {
		Formats []domain.VideoFormat `json:"formats"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body không phải JSON: %v", err)
	}
	if len(body.Formats) < 2 {
		t.Fatalf("muốn ít nhất 2 format, có %d", len(body.Formats))
	}
}

// CR-023 correction: normalize only publishes an AMQP command, never an HTTP
// call to video-assembly.
func TestHandleNormalizeChannelAsset_QueuesAndReturnsAccepted(t *testing.T) {
	fake := &fakeChannelAssets{}
	router := NewRouter(nil, nil, nil, &fakeProjectReader{}, nil, nil, fake)

	body := []byte(`{"file_path":"/data/uploads/intro.mp4","source_hash":"abc123"}`)
	req := httptest.NewRequest("POST", "/v1/channel-assets/intro", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)

	if rec.Code != 202 {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if fake.normalizedIn.Kind != "intro" || fake.normalizedIn.FilePath != "/data/uploads/intro.mp4" {
		t.Fatalf("unexpected normalize input: %+v", fake.normalizedIn)
	}
	// A body without asset_role keeps meaning "this is the sting clip".
	if fake.normalizedIn.AssetRole != application.AssetRoleVideo {
		t.Fatalf("asset_role = %q, muốn %q", fake.normalizedIn.AssetRole, application.AssetRoleVideo)
	}
}

// FR66.5: the music bed is uploaded through the same endpoint, distinguished
// only by asset_role.
func TestHandleNormalizeChannelAsset_PassesMusicAssetRoleThrough(t *testing.T) {
	fake := &fakeChannelAssets{}
	router := NewRouter(nil, nil, nil, &fakeProjectReader{}, nil, nil, fake)

	body := []byte(`{"file_path":"/data/uploads/music.mp3","source_hash":"m1","asset_role":"music"}`)
	req := httptest.NewRequest("POST", "/v1/channel-assets/outro", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)

	if rec.Code != 202 {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if fake.normalizedIn.AssetRole != application.AssetRoleMusic {
		t.Fatalf("asset_role = %q, muốn %q", fake.normalizedIn.AssetRole, application.AssetRoleMusic)
	}
}

func TestHandleNormalizeChannelAsset_RejectsUnknownAssetRole(t *testing.T) {
	router := NewRouter(nil, nil, nil, &fakeProjectReader{}, nil, nil, &fakeChannelAssets{})

	body := []byte(`{"file_path":"/data/uploads/x.mp4","asset_role":"subtitle"}`)
	req := httptest.NewRequest("POST", "/v1/channel-assets/intro", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)

	if rec.Code != 400 {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestHandleNormalizeChannelAsset_RejectsUnknownKind(t *testing.T) {
	router := NewRouter(nil, nil, nil, &fakeProjectReader{}, nil, nil, &fakeChannelAssets{})

	req := httptest.NewRequest("POST", "/v1/channel-assets/bogus", bytes.NewReader([]byte(`{"file_path":"x"}`)))
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)

	if rec.Code != 400 {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestHandleChannelAssetPreview_ServesPointers(t *testing.T) {
	fake := &fakeChannelAssets{pointers: []domain.ChannelAssetPointer{
		{Kind: "intro", RenderQuality: domain.Quality1080p60, AssetID: "asset-1", Version: 3},
	}}
	router := NewRouter(nil, nil, nil, &fakeProjectReader{}, nil, nil, fake)

	req := httptest.NewRequest("GET", "/v1/channel-assets/preview", nil)
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Assets []struct {
			Kind    string `json:"kind"`
			AssetID string `json:"asset_id"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body không phải JSON: %v", err)
	}
	if len(body.Assets) != 1 || body.Assets[0].AssetID != "asset-1" {
		t.Fatalf("unexpected response: %+v", body)
	}
}

// --- CR-026: POST /v1/short-script-suggestions ---

type fakeSuggestShortScript struct {
	script string
	err    error
}

func (f *fakeSuggestShortScript) Execute(_ context.Context, _, _ string, _ domain.ContentLanguage) (string, error) {
	return f.script, f.err
}

func TestHandleSuggestShortScript_404WhenUnwired(t *testing.T) {
	// Same "unwired means absent, not broken" posture as qc-report (CR-021).
	router := NewRouter(nil, nil, nil, &fakeProjectReader{}, nil, nil, nil)

	body, _ := json.Marshal(map[string]string{"topic": "chủ đề", "language": "vi"})
	req := httptest.NewRequest("POST", "/v1/short-script-suggestions", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)

	if rec.Code != 404 {
		t.Fatalf("expected 404 when unwired, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleSuggestShortScript_ReturnsTheDraft(t *testing.T) {
	router := NewRouter(nil, nil, nil, &fakeProjectReader{}, nil, nil, nil).
		WithShortScriptSuggester(&fakeSuggestShortScript{script: "from conceptflow import *\n"})

	body, _ := json.Marshal(map[string]string{"topic": "chủ đề", "language": "vi"})
	req := httptest.NewRequest("POST", "/v1/short-script-suggestions", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp suggestShortScriptResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp.ScriptContent != "from conceptflow import *\n" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestHandleSuggestShortScript_InvalidLanguage(t *testing.T) {
	router := NewRouter(nil, nil, nil, &fakeProjectReader{}, nil, nil, nil).
		WithShortScriptSuggester(&fakeSuggestShortScript{script: "x"})

	body, _ := json.Marshal(map[string]string{"topic": "chủ đề", "language": "fr"})
	req := httptest.NewRequest("POST", "/v1/short-script-suggestions", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)

	if rec.Code != 400 {
		t.Fatalf("expected 400 for an unsupported language, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleSuggestShortScript_UseCaseErrorIs400(t *testing.T) {
	// No project involved (unlike suggest-metadata) — every failure here is
	// either a bad request (blank topic+source) or an upstream model error,
	// neither of which is a 404/409 domain sentinel.
	router := NewRouter(nil, nil, nil, &fakeProjectReader{}, nil, nil, nil).
		WithShortScriptSuggester(&fakeSuggestShortScript{err: errors.New("topic or source_script_content is required")})

	body, _ := json.Marshal(map[string]string{"topic": "", "language": "vi"})
	req := httptest.NewRequest("POST", "/v1/short-script-suggestions", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)

	if rec.Code != 400 {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}
