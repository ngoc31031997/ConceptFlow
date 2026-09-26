package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"authoring/internal/application"
	"authoring/internal/domain"
)

type fakeSuggestMetadata struct {
	out *application.SuggestPublishMetadataOutput
	err error
}

func (f *fakeSuggestMetadata) Execute(_ context.Context, _ string) (*application.SuggestPublishMetadataOutput, error) {
	return f.out, f.err
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
	router := NewRouter(nil, nil)

	body, _ := json.Marshal(map[string]string{"topic": "chủ đề", "language": "vi"})
	req := httptest.NewRequest("POST", "/v1/short-script-suggestions", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)

	if rec.Code != 404 {
		t.Fatalf("expected 404 when unwired, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleSuggestShortScript_ReturnsTheDraft(t *testing.T) {
	router := NewRouter(nil, nil).
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
	router := NewRouter(nil, nil).
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
	router := NewRouter(nil, nil).
		WithShortScriptSuggester(&fakeSuggestShortScript{err: errors.New("topic or source_script_content is required")})

	body, _ := json.Marshal(map[string]string{"topic": "", "language": "vi"})
	req := httptest.NewRequest("POST", "/v1/short-script-suggestions", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, req)

	if rec.Code != 400 {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

// fakeGenerateAuthoring stands in for CR-027 FR78's use case.
type fakeGenerateAuthoring struct {
	out       application.GeneratedStep
	err       error
	available bool
	gotStep   string
}

func (f *fakeGenerateAuthoring) Execute(_ context.Context, _, step string) (application.GeneratedStep, error) {
	f.gotStep = step
	return f.out, f.err
}
func (f *fakeGenerateAuthoring) Available() bool  { return f.available }
func (f *fakeGenerateAuthoring) Provider() string { return "hive" }

func newGenerateRouter(gen generateAuthoringUseCase) *Router {
	rt := NewRouter(&fakeSuggestMetadata{}, nil)
	if gen != nil {
		rt = rt.WithGenerateAuthoring(gen)
	}
	return rt
}

func TestHandleGenerateAuthoring_OK(t *testing.T) {
	gen := &fakeGenerateAuthoring{
		available: true,
		out: application.GeneratedStep{
			Step: "story", Role: "story_architect", Content: "BEAT 1", Provider: "hive",
		},
	}
	rec := httptest.NewRecorder()
	newGenerateRouter(gen).Handler().ServeHTTP(rec,
		httptest.NewRequest("POST", "/v1/projects/p1/authoring/story/generate", nil))

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if gen.gotStep != "story" {
		t.Errorf("step = %q, want story", gen.gotStep)
	}
	var out application.GeneratedStep
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out.Content != "BEAT 1" {
		t.Errorf("content = %q, want the generated text", out.Content)
	}
}

// CR-030 — bước duyệt đã bị bỏ, nên "review" không còn là một step hợp lệ:
// route vẫn khớp, nhưng use case từ chối nó.
func TestHandleGenerateAuthoring_ForwardsStepVerbatim(t *testing.T) {
	gen := &fakeGenerateAuthoring{available: true}
	rec := httptest.NewRecorder()
	newGenerateRouter(gen).Handler().ServeHTTP(rec,
		httptest.NewRequest("POST", "/v1/projects/p1/authoring/code/generate", nil))

	if gen.gotStep != "code" {
		t.Errorf("step = %q, want code", gen.gotStep)
	}
}

// Unwired means absent, not broken: without a key the route 404s and the
// Copy-prompt path is the one that works (FR83.2).
func TestHandleGenerateAuthoring_NotWired(t *testing.T) {
	rec := httptest.NewRecorder()
	newGenerateRouter(nil).Handler().ServeHTTP(rec,
		httptest.NewRequest("POST", "/v1/projects/p1/authoring/story/generate", nil))

	if rec.Code != 404 {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestHandleGenerateAuthoring_BusyIsConflict(t *testing.T) {
	gen := &fakeGenerateAuthoring{available: true, err: application.ErrGenerateBusy}
	rec := httptest.NewRecorder()
	newGenerateRouter(gen).Handler().ServeHTTP(rec,
		httptest.NewRequest("POST", "/v1/projects/p1/authoring/story/generate", nil))

	if rec.Code != 409 {
		t.Fatalf("expected 409 for a double click, got %d: %s", rec.Code, rec.Body.String())
	}
}

// FR79.3 — each provider failure names its own cause AND the copy-out way
// through. "AI failed" would send a Creator with an empty balance to go
// rewrite their prompt.
func TestHandleGenerateAuthoring_ProviderErrorsAreClassified(t *testing.T) {
	cases := []struct {
		kind       application.LLMErrorKind
		wantStatus int
		wantIn     string
	}{
		{application.ErrKindAuth, 502, "HIVE_API_KEY"},
		{application.ErrKindBalance, 502, "số dư"},
		{application.ErrKindRateLimit, 429, "quá nhanh"},
		{application.ErrKindTimeout, 504, "HIVE_TIMEOUT_SECONDS"},
		{application.ErrKindTruncated, 502, "HIVE_MAX_OUTPUT_TOKENS"},
	}
	for _, tc := range cases {
		gen := &fakeGenerateAuthoring{
			available: true,
			err:       &application.LLMError{Kind: tc.kind, Provider: "hive", Err: errors.New("boom")},
		}
		rec := httptest.NewRecorder()
		newGenerateRouter(gen).Handler().ServeHTTP(rec,
			httptest.NewRequest("POST", "/v1/projects/p1/authoring/story/generate", nil))

		if rec.Code != tc.wantStatus {
			t.Errorf("%s: status = %d, want %d", tc.kind, rec.Code, tc.wantStatus)
		}
		body := rec.Body.String()
		if !strings.Contains(body, tc.wantIn) {
			t.Errorf("%s: body %q, want it to mention %q", tc.kind, body, tc.wantIn)
		}
		if !strings.Contains(body, "Copy prompt") {
			t.Errorf("%s: body %q, want it to point at the copy-out fallback", tc.kind, body)
		}
	}
}

func TestHandleLLMStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	newGenerateRouter(&fakeGenerateAuthoring{available: true}).Handler().ServeHTTP(rec,
		httptest.NewRequest("GET", "/v1/llm/status", nil))
	var enabled llmStatusResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &enabled)
	if !enabled.Enabled || enabled.Provider != "hive" {
		t.Errorf("status = %+v, want enabled hive", enabled)
	}

	rec = httptest.NewRecorder()
	newGenerateRouter(nil).Handler().ServeHTTP(rec,
		httptest.NewRequest("GET", "/v1/llm/status", nil))
	var disabled llmStatusResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &disabled)
	if disabled.Enabled {
		t.Error("want disabled when no provider is wired")
	}
	if disabled.Reason == "" {
		t.Error("want a reason so the GUI can explain the missing button (FR79.4)")
	}
}

// fakeSaveAuthoringMode backs CR-027 FR79's mode endpoint.
type fakeSaveAuthoringMode struct {
	gotMode string
	err     error
}

func (f *fakeSaveAuthoringMode) Execute(_ context.Context, _, mode string) error {
	f.gotMode = mode
	return f.err
}

func TestHandleSaveAuthoringMode(t *testing.T) {
	save := &fakeSaveAuthoringMode{}
	rt := NewRouter(&fakeSuggestMetadata{}, nil).
		WithAuthoringMode(save)

	rec := httptest.NewRecorder()
	rt.Handler().ServeHTTP(rec, httptest.NewRequest("PUT", "/v1/projects/p1/authoring/mode",
		bytes.NewReader([]byte(`{"mode":"ai"}`))))

	if rec.Code != 204 {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	if save.gotMode != "ai" {
		t.Errorf("mode = %q, want ai", save.gotMode)
	}

	// A rejected mode is a 400 the GUI can show, not a 500.
	save.err = errors.New(`mode must be "manual" or "ai"`)
	rec = httptest.NewRecorder()
	rt.Handler().ServeHTTP(rec, httptest.NewRequest("PUT", "/v1/projects/p1/authoring/mode",
		bytes.NewReader([]byte(`{"mode":"sometimes"}`))))
	if rec.Code != 400 {
		t.Errorf("expected 400 for an unknown mode, got %d", rec.Code)
	}
}

// Unwired means absent: a deployment without this use case 404s rather than
// pretending to remember the choice.
func TestHandleSaveAuthoringMode_NotWired(t *testing.T) {
	rt := NewRouter(&fakeSuggestMetadata{}, nil)
	rec := httptest.NewRecorder()
	rt.Handler().ServeHTTP(rec, httptest.NewRequest("PUT", "/v1/projects/p1/authoring/mode",
		bytes.NewReader([]byte(`{"mode":"ai"}`))))
	if rec.Code != 404 {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}