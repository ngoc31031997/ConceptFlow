package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"authoring/internal/application"
	"authoring/internal/domain"
)

// externalStory is a story step whose Generate asks the model once, through the
// external-AI provider, the way a real step does.
type externalStory struct {
	fakeStory
	provider application.LLMProviderPort
}

func (e *externalStory) Generate(ctx context.Context, _, _ string) (domain.StoredStory, error) {
	if _, err := e.provider.Chat(ctx, application.ChatRequest{System: "Viết câu chuyện", User: "Làm đi"}); err != nil {
		return domain.StoredStory{}, err
	}
	e.runs++
	return e.stored(), nil
}

type unusedModel struct{ asked int }

func (*unusedModel) Name() string { return "fake" }
func (m *unusedModel) Chat(context.Context, application.ChatRequest) (application.ChatResult, error) {
	m.asked++
	return application.ChatResult{}, nil
}

type echoReplier struct{}

func (echoReplier) ChatExternal(_ context.Context, _ application.ChatRequest, reply string) (application.ChatResult, error) {
	return application.ChatResult{Content: reply}, nil
}

func serveExternal(mode, body string) (*httptest.ResponseRecorder, *externalStory, *unusedModel) {
	model := &unusedModel{}
	story := &externalStory{provider: application.NewExternalAIProvider(model, echoReplier{})}
	rt := NewRouter(&fakeSuggestMetadata{}, nil).WithStory(story).WithOperations(application.NewOperations())
	req := httptest.NewRequest(http.MethodPost, "/v1/projects/p1/story/generate", strings.NewReader(body))
	if mode != "" {
		req.Header.Set("X-External-AI", mode)
	}
	rec := httptest.NewRecorder()
	rt.Handler().ServeHTTP(rec, req)
	return rec, story, model
}

func TestPromptRequestAnswersWithThePromptAndRunsNothing(t *testing.T) {
	rec, story, model := serveExternal("prompt", `{"note":"ngắn hơn"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	var got struct {
		Prompt string `json:"external_prompt"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil || got.Prompt != "Viết câu chuyện\n\nLàm đi" {
		t.Errorf("body = %s (%v), want the prompt block", rec.Body, err)
	}
	if story.runs != 0 || model.asked != 0 {
		t.Errorf("step runs = %d, model calls = %d, want neither", story.runs, model.asked)
	}
}

func TestPasteRequestRunsTheStepOnThePastedReply(t *testing.T) {
	rec, story, model := serveExternal("paste", `{"note":"","external_reply":"{\"a\":1}"}`)
	if rec.Code != http.StatusOK || story.runs != 1 || model.asked != 0 {
		t.Errorf("status = %d, step runs = %d, model calls = %d, want 200, 1 and 0 (body %s)", rec.Code, story.runs, model.asked, rec.Body)
	}
}

func TestPasteRequestWithoutAReplyIsRefused(t *testing.T) {
	rec, story, _ := serveExternal("paste", `{"external_reply":"  "}`)
	if rec.Code != http.StatusBadRequest || story.runs != 0 {
		t.Errorf("status = %d, step runs = %d, want 400 and 0", rec.Code, story.runs)
	}
}

func TestUnknownExternalModeIsRefused(t *testing.T) {
	if rec, _, _ := serveExternal("both", `{}`); rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestRequestWithoutTheHeaderAsksTheModelAsBefore(t *testing.T) {
	rec, story, model := serveExternal("", `{}`)
	if rec.Code != http.StatusOK || story.runs != 1 || model.asked != 1 {
		t.Errorf("status = %d, step runs = %d, model calls = %d, want 200, 1 and 1", rec.Code, story.runs, model.asked)
	}
}

func TestPromptRequestPassesThroughAnErrorBeforeAnyPrompt(t *testing.T) {
	model := &unusedModel{}
	story := &externalStory{provider: application.NewExternalAIProvider(model, echoReplier{})}
	story.err = domain.ErrProjectNotFound
	rt := NewRouter(&fakeSuggestMetadata{}, nil).WithStory(failingStory{story}).WithOperations(application.NewOperations())
	req := httptest.NewRequest(http.MethodPost, "/v1/projects/p1/story/generate", strings.NewReader(`{}`))
	req.Header.Set("X-External-AI", "prompt")
	rec := httptest.NewRecorder()
	rt.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want the step's own 404", rec.Code)
	}
}

// failingStory fails before it reaches the model.
type failingStory struct{ *externalStory }

func (f failingStory) Generate(context.Context, string, string) (domain.StoredStory, error) {
	return domain.StoredStory{}, f.err
}
