package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

// --- CR-040 FR113: POST /v1/prompt-renders ---

type fakePromptRenderer struct{ got application.RenderInput }

func (f *fakePromptRenderer) Execute(context.Context, string, domain.PromptRole) (application.RenderedPrompt, error) {
	return application.RenderedPrompt{}, nil
}

func (f *fakePromptRenderer) Render(_ context.Context, in application.RenderInput) (application.RenderedPrompt, error) {
	f.got = in
	return application.RenderedPrompt{Role: in.Role, Prompt: "rendered:" + in.Topic}, nil
}

func TestHandlePromptRenders_PassesTheDraftThrough(t *testing.T) {
	fake := &fakePromptRenderer{}
	router := NewRouter(nil, nil).WithRenderPrompt(fake)

	body, _ := json.Marshal(map[string]any{"role": "story_architect", "language": "en", "topic": "loops", "subtitle_mode": "burn_in"})
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/v1/prompt-renders", bytes.NewReader(body)))

	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "rendered:loops") {
		t.Fatalf("unexpected %d: %s", rec.Code, rec.Body.String())
	}
	if fake.got.Role != domain.RoleStoryArchitect || fake.got.Language != "en" || fake.got.SubtitleMode != "burn_in" {
		t.Fatalf("draft values were not forwarded: %+v", fake.got)
	}
}

func TestHandlePromptRenders_RejectsAnUnknownRoleAnd404sWhenUnwired(t *testing.T) {
	router := NewRouter(nil, nil).WithRenderPrompt(&fakePromptRenderer{})
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/v1/prompt-renders", strings.NewReader(`{"role":"nope"}`)))
	if rec.Code != 400 {
		t.Fatalf("unknown role: want 400, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	NewRouter(nil, nil).Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/v1/prompt-renders", strings.NewReader(`{"role":"story_architect"}`)))
	if rec.Code != 404 {
		t.Fatalf("unwired: want 404, got %d", rec.Code)
	}
}

type fakeUsageStats struct {
	stats []application.ModelUsageStats
	err   error
	step  string
	phase string
}

func (f *fakeUsageStats) ModelUsageStats(_ context.Context, step, phase string, _ time.Time) ([]application.ModelUsageStats, error) {
	f.step, f.phase = step, phase
	return f.stats, f.err
}

// CR-050 FR-19: the status carries the measured cost of each model's code
// chunks, and a failed read is reported as such, never as "no data".
func TestHandleLLMStatusCodeStats(t *testing.T) {
	stats := &fakeUsageStats{stats: []application.ModelUsageStats{{
		Model: "zai-org/glm-5.3-flash", Calls: 20, OK: 18, Failures: map[string]int{"budget": 2},
		AvgCompletionTokens: 97870, AvgDurationMs: 527000,
	}}}
	rec := httptest.NewRecorder()
	newGenerateRouter(&fakeGenerateAuthoring{available: true}).WithUsageStats(stats).Handler().ServeHTTP(rec,
		httptest.NewRequest("GET", "/v1/llm/status", nil))
	var got llmStatusResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	glm := got.CodeStats["zai-org/glm-5.3-flash"]
	if stats.step != "code" || stats.phase != "chunk" || glm.OK != 18 || glm.Failures["budget"] != 2 || got.CodeStatsError != "" {
		t.Fatalf("status = %+v (queried %s/%s)", got, stats.step, stats.phase)
	}
	for _, m := range got.Models {
		if m.ID == "ollama" && m.CodeOK {
			t.Error("ollama offered for the code step")
		}
	}

	rec = httptest.NewRecorder()
	newGenerateRouter(&fakeGenerateAuthoring{available: true}).WithUsageStats(&fakeUsageStats{err: errors.New("db down")}).
		Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/v1/llm/status", nil))
	got = llmStatusResponse{}
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if rec.Code != 200 || got.CodeStats != nil || got.CodeStatsError == "" {
		t.Fatalf("failed read: code=%d stats=%v err=%q, want 200 with the error stated", rec.Code, got.CodeStats, got.CodeStatsError)
	}
}

func TestDescribeGenerateErrorModelNotForCode(t *testing.T) {
	status, msg := DescribeGenerateError(&application.ErrModelNotForCode{Model: "ollama"})
	if status != 400 || !strings.Contains(msg, "ollama") {
		t.Fatalf("got %d %q, want 400 naming the model", status, msg)
	}
}
