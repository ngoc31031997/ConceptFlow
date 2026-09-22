package application_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"orchestrator/internal/application"
	"orchestrator/internal/domain"
)

// stubProvider stands in for Hive. It records the request so a test can check
// that the prompt sent is the rendered one — the whole point of FR77.3 is that
// the AI path and the Copy path cannot diverge.
type stubProvider struct {
	req     application.ChatRequest
	content string
	err     error
	calls   int
	// block, when non-nil, holds the call open so a second, concurrent call
	// hits the in-flight lock; entered is closed once the call is inside, so
	// the test waits on a signal instead of spinning on a shared counter.
	block   chan struct{}
	entered chan struct{}
}

func (s *stubProvider) Name() string { return "stub" }

func (s *stubProvider) Chat(_ context.Context, req application.ChatRequest) (application.ChatResult, error) {
	s.calls++
	s.req = req
	if s.entered != nil {
		close(s.entered)
	}
	if s.block != nil {
		<-s.block
	}
	if s.err != nil {
		return application.ChatResult{}, s.err
	}
	return application.ChatResult{
		Content: s.content,
		Usage:   application.TokenUsage{Model: "stub-model", PromptTokens: 10, CompletionTokens: 5},
	}, nil
}

type recordingSaver struct {
	content string
	topic   string
	err     error
	calls   int
}

func (r *recordingSaver) Execute(_ context.Context, _, content, topic string) error {
	r.calls++
	if r.err != nil {
		return r.err
	}
	r.content, r.topic = content, topic
	return nil
}

type contentSaver struct {
	content string
	calls   int
}

func (c *contentSaver) Execute(_ context.Context, _, content string) error {
	c.calls++
	c.content = content
	return nil
}

func newGenerateFixture(t *testing.T, provider *stubProvider, story *recordingSaver, maxInputChars int) (*application.GenerateAuthoringUseCase, *contentSaver) {
	t.Helper()
	renderCtx := &fakeRenderContext{
		project: &domain.Project{
			ProjectID: "p1", ContentLanguage: domain.LanguageVietnamese,
			RenderEngine: domain.RenderEngineManim,
		},
		topic: "Vòng lặp for",
	}
	storyboard := &contentSaver{}
	uc := application.NewGenerateAuthoringUseCase(
		newRenderer("Chủ đề: {{topic}}", renderCtx),
		provider, nil, renderCtx,
		story, storyboard, &contentSaver{}, &contentSaver{},
		maxInputChars, 16000,
	)
	return uc, storyboard
}

func TestGenerateAuthoringStorySendsRenderedPromptAndSaves(t *testing.T) {
	provider := &stubProvider{content: "  BEAT 1 — mở đầu  "}
	story := &recordingSaver{}
	uc, _ := newGenerateFixture(t, provider, story, 0)

	got, err := uc.Execute(context.Background(), "p1", "story", "")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if !strings.Contains(provider.req.System, "Chủ đề: Vòng lặp for") {
		t.Errorf("provider got prompt %q, want the rendered template", provider.req.System)
	}
	if got.Content != "BEAT 1 — mở đầu" {
		t.Errorf("content = %q, want it trimmed", got.Content)
	}
	if got.Role != string(domain.RoleStoryArchitect) {
		t.Errorf("role = %q, want story_architect", got.Role)
	}
	if story.content != "BEAT 1 — mở đầu" {
		t.Errorf("saved %q, want the generated outline", story.content)
	}
	// The topic is already on the project (CR-028 FR83.1); an AI run must not
	// overwrite it with a blank derived from nothing.
	if story.topic != "" {
		t.Errorf("saved topic = %q, want it left untouched", story.topic)
	}
}

func TestGenerateAuthoringMapsStepToEngineRole(t *testing.T) {
	provider := &stubProvider{content: "storyboard"}
	uc, storyboard := newGenerateFixture(t, provider, &recordingSaver{}, 0)

	got, err := uc.Execute(context.Background(), "p1", "storyboard", "")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got.Role != string(domain.RoleVisualDirector) {
		t.Errorf("role = %q, want visual_director for a manim project", got.Role)
	}
	if storyboard.calls != 1 {
		t.Errorf("storyboard saver calls = %d, want 1 (and no other step touched)", storyboard.calls)
	}
}

func TestGenerateAuthoringRejectsUnknownStep(t *testing.T) {
	uc, _ := newGenerateFixture(t, &stubProvider{content: "x"}, &recordingSaver{}, 0)
	if _, err := uc.Execute(context.Background(), "p1", "outline", ""); err == nil {
		t.Fatal("want an error for an unknown step")
	}
}

func TestGenerateAuthoringRefusesPromptOverInputCap(t *testing.T) {
	provider := &stubProvider{content: "x"}
	uc, _ := newGenerateFixture(t, provider, &recordingSaver{}, 5)

	_, err := uc.Execute(context.Background(), "p1", "story", "")
	if err == nil || !strings.Contains(err.Error(), "HIVE_MAX_INPUT_CHARS") {
		t.Fatalf("err = %v, want the input-cap refusal", err)
	}
	if provider.calls != 0 {
		t.Errorf("provider called %d times, want 0 — the cap exists to avoid the call", provider.calls)
	}
}

func TestGenerateAuthoringEmptyAnswerIsClassified(t *testing.T) {
	provider := &stubProvider{content: "   "}
	story := &recordingSaver{}
	uc, _ := newGenerateFixture(t, provider, story, 0)

	_, err := uc.Execute(context.Background(), "p1", "story", "")
	if application.LLMErrorKindOf(err) != application.ErrKindEmpty {
		t.Fatalf("err = %v, want kind %q", err, application.ErrKindEmpty)
	}
	if story.calls != 0 {
		t.Errorf("saver called %d times, want 0 — nothing was generated", story.calls)
	}
}

func TestGenerateAuthoringReturnsContentWhenSaveFails(t *testing.T) {
	provider := &stubProvider{content: "dàn ý"}
	story := &recordingSaver{err: errors.New("project đã khoá")}
	uc, _ := newGenerateFixture(t, provider, story, 0)

	got, err := uc.Execute(context.Background(), "p1", "story", "")
	if err != nil {
		t.Fatalf("Execute: %v — a billed call must not be thrown away", err)
	}
	if got.Content != "dàn ý" {
		t.Errorf("content = %q, want the generated text back", got.Content)
	}
	if got.SaveError == "" {
		t.Error("want SaveError set so the GUI can say the save did not land")
	}
}

func TestGenerateAuthoringSecondConcurrentCallIsBusy(t *testing.T) {
	provider := &stubProvider{content: "dàn ý", block: make(chan struct{}), entered: make(chan struct{})}
	uc, _ := newGenerateFixture(t, provider, &recordingSaver{}, 0)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = uc.Execute(context.Background(), "p1", "story", "")
	}()
	// Wait until the provider is actually inside the call, so the second
	// attempt races the lock rather than the goroutine scheduler.
	<-provider.entered

	_, err := uc.Execute(context.Background(), "p1", "story", "")
	if !errors.Is(err, application.ErrGenerateBusy) {
		t.Fatalf("err = %v, want ErrGenerateBusy", err)
	}
	close(provider.block)
	wg.Wait()
	if provider.calls != 1 {
		t.Errorf("provider called %d times, want 1 — a double click must not bill twice", provider.calls)
	}
}

func TestGenerateAuthoringWithoutProviderIsNotConfigured(t *testing.T) {
	uc := application.NewGenerateAuthoringUseCase(nil, nil, nil, nil, nil, nil, nil, nil, 0, 0)
	if uc.Available() {
		t.Error("Available() = true with no provider")
	}
	if _, err := uc.Execute(context.Background(), "p1", "story", ""); !errors.Is(err, application.ErrLLMNotConfigured) {
		t.Fatalf("err = %v, want ErrLLMNotConfigured", err)
	}
}
