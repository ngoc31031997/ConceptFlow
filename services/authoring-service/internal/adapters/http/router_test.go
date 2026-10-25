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

// fakeLLMStatus stands in for the llm-service client's view of itself.
type fakeLLMStatus struct{ ready bool }

func (f *fakeLLMStatus) Ready() bool  { return f.ready }
func (f *fakeLLMStatus) Name() string { return "hive" }

func newStatusRouter(status llmStatusSource) *Router {
	rt := NewRouter(&fakeSuggestMetadata{}, nil)
	if status != nil {
		rt = rt.WithLLMStatus(status)
	}
	return rt
}

func TestDescribeGenerateErrorClassifiesProviderErrors(t *testing.T) {
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
		status, message := DescribeGenerateError(&application.LLMError{Kind: tc.kind, Provider: "hive", Err: errors.New("boom")})
		if status != tc.wantStatus {
			t.Errorf("%s: status = %d, want %d", tc.kind, status, tc.wantStatus)
		}
		if !strings.Contains(message, tc.wantIn) {
			t.Errorf("%s: message %q, want it to mention %q", tc.kind, message, tc.wantIn)
		}
		if !strings.Contains(message, "Copy prompt") {
			t.Errorf("%s: message %q, want it to point at the copy-out fallback", tc.kind, message)
		}
	}
}

func TestDescribeGenerateErrorNamesTheLoggedCallOfAnUnreadableAnswer(t *testing.T) {
	status, message := DescribeGenerateError(&application.LLMError{
		Kind: application.ErrKindMalformed, Provider: "llm-service", CallID: "c-9",
		Err: errors.New("khối JSON không đọc được"),
	})
	if status != 502 || !strings.Contains(message, "khối JSON không đọc được") || !strings.Contains(message, "c-9") {
		t.Errorf("status = %d, message = %q, want the cause and the call id", status, message)
	}
}

func TestHandleLLMStatus(t *testing.T) {
	rec := httptest.NewRecorder()
	newStatusRouter(&fakeLLMStatus{ready: true}).Handler().ServeHTTP(rec,
		httptest.NewRequest("GET", "/v1/llm/status", nil))
	var enabled llmStatusResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &enabled)
	if !enabled.Enabled || enabled.Provider != "hive" {
		t.Errorf("status = %+v, want enabled hive", enabled)
	}
	for _, m := range enabled.Models {
		if m.ID == "ollama" && !m.Local {
			t.Error("ollama must be flagged local so the GUI can warn about it")
		}
	}

	for name, status := range map[string]llmStatusSource{"not wired": nil, "not ready": &fakeLLMStatus{}} {
		rec = httptest.NewRecorder()
		newStatusRouter(status).Handler().ServeHTTP(rec, httptest.NewRequest("GET", "/v1/llm/status", nil))
		var disabled llmStatusResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &disabled)
		if disabled.Enabled {
			t.Errorf("%s: want disabled", name)
		}
		if disabled.Reason == "" {
			t.Errorf("%s: want a reason so the GUI can explain the missing button", name)
		}
	}
}

// fakeSaveAuthoringMode backs the authoring mode endpoint.
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

// --- POST /v1/prompt-renders ---

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

// The illustration style is no longer a prompt role: the rules belong to the
// visual language a project is pinned to, so the old role is simply unknown.
func TestTheRetiredIllustrationStyleRoleIsAnUnknownRole(t *testing.T) {
	router := NewRouter(nil, nil).WithRenderPrompt(&fakePromptRenderer{})
	rec := httptest.NewRecorder()
	router.Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/v1/prompt-renders", strings.NewReader(`{"role":"illustration_style"}`)))
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), "unknown role") {
		t.Fatalf("want 400 unknown role, got %d %s", rec.Code, rec.Body.String())
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
