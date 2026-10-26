package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"authoring/internal/domain"
)

// pastedReplier answers a pasted reply the way llm-service does, and keeps what it was given.
type pastedReplier struct {
	replies []string
}

func (p *pastedReplier) ChatExternal(_ context.Context, req ChatRequest, reply string) (ChatResult, error) {
	p.replies = append(p.replies, reply)
	return llmServiceReply(req, reply)
}

func externalStoryFixture(t *testing.T) (*storyFixture, *pastedReplier) {
	t.Helper()
	f := newStoryFixture()
	f.open(t)
	replier := &pastedReplier{}
	f.uc = NewStoryDeskUseCase(f.store, f.approved, f.standing, f.project, f.project, briefPrompts{},
		NewExternalAIProvider(f.llm, replier), 4000)
	return f, replier
}

func TestExternalPromptIsTheSystemAndUserTurnAsOneBlock(t *testing.T) {
	got := ExternalPrompt(ChatRequest{System: " Hệ thống \n", User: "Làm đi"})
	if got != "Hệ thống\n\nLàm đi" {
		t.Errorf("prompt = %q, want the system prompt, a blank line, then the user turn", got)
	}
	if got := ExternalPrompt(ChatRequest{System: "Chỉ hệ thống"}); got != "Chỉ hệ thống" {
		t.Errorf("prompt = %q, want no stray blank lines when a part is empty", got)
	}
}

func TestPromptModeBuildsTheSamePromptAsARealCallAndStoresNothing(t *testing.T) {
	f, _ := externalStoryFixture(t)
	run := NewExternalAIRun(ExternalAIPrompt, "")
	_, err := f.uc.Generate(WithExternalAI(context.Background(), run), "p1", "")
	if !errors.Is(err, ErrExternalAIPromptBuilt) {
		t.Fatalf("Generate error = %v, want ErrExternalAIPromptBuilt", err)
	}
	prompt, ok := run.Prompt()
	if !ok {
		t.Fatal("no prompt was captured")
	}
	if len(f.llm.systems) != 0 {
		t.Error("the model was asked in prompt mode")
	}
	if f.store.story != nil {
		t.Error("prompt mode stored a story")
	}

	real := newStoryFixture()
	real.open(t)
	real.generate(t, "")
	if want := ExternalPrompt(ChatRequest{System: real.llm.systems[0], User: userTurnNudge}); prompt != want {
		t.Errorf("prompt differs from the one a real call sends:\n got: %s\nwant: %s", prompt, want)
	}
}

func TestPasteModeRunsTheStepOnThePastedReplyWithoutAskingTheModel(t *testing.T) {
	f, replier := externalStoryFixture(t)
	run := NewExternalAIRun(ExternalAIPaste, "Đây là kết quả:\n"+storyReplyFull)
	s, err := f.uc.Generate(WithExternalAI(context.Background(), run), "p1", "")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if s.Story.CentralQuestion != "Vì sao ong chết trong quả sung?" || f.store.story == nil {
		t.Errorf("story = %+v, stored = %v, want the pasted story stored", s.Story, f.store.story != nil)
	}
	if len(f.llm.systems) != 0 || len(replier.replies) != 1 {
		t.Errorf("model calls = %d, pasted replies read = %d, want 0 and 1", len(f.llm.systems), len(replier.replies))
	}
}

func TestPasteModeAnUnusableReplyFailsOnceWithoutAskingAgain(t *testing.T) {
	f, replier := externalStoryFixture(t)
	run := NewExternalAIRun(ExternalAIPaste, `{"central_question":"","core_answer":"","narrative_arc":[]}`)
	_, err := f.uc.Generate(WithExternalAI(context.Background(), run), "p1", "")
	if !errors.Is(err, domain.ErrStoryReplyUnusable) {
		t.Fatalf("Generate error = %v, want the reply to be refused as unusable", err)
	}
	if f.store.story != nil || len(replier.replies) != 1 {
		t.Errorf("stored = %v, replies read = %d, want nothing stored and one read", f.store.story != nil, len(replier.replies))
	}
}

func TestExternalAIServesOneTurnPerRequest(t *testing.T) {
	provider := NewExternalAIProvider(&briefLLM{reply: "x"}, &pastedReplier{})
	ctx := WithExternalAI(context.Background(), NewExternalAIRun(ExternalAIPaste, "kết quả"))
	if _, err := provider.Chat(ctx, ChatRequest{User: "một"}); err != nil {
		t.Fatalf("first turn: %v", err)
	}
	if _, err := provider.Chat(ctx, ChatRequest{User: "hai"}); !errors.Is(err, ErrExternalAISingleTurn) {
		t.Errorf("second turn error = %v, want ErrExternalAISingleTurn", err)
	}
}

func TestWithoutAnExternalRunTheModelIsAsked(t *testing.T) {
	llm := &briefLLM{reply: "trả lời"}
	res, err := NewExternalAIProvider(llm, &pastedReplier{}).Chat(context.Background(), ChatRequest{System: "S", User: "U"})
	if err != nil || res.Content != "trả lời" || len(llm.systems) != 1 {
		t.Errorf("res = %+v, err = %v, model calls = %d, want the model's own reply", res, err, len(llm.systems))
	}
}

type notReadyProvider struct{ briefLLM }

func (*notReadyProvider) Ready() bool { return false }

func TestAnExternalRunNeedsNoModelKey(t *testing.T) {
	provider := NewExternalAIProvider(&notReadyProvider{}, &pastedReplier{})
	if err := requireProvider(context.Background(), provider); !errors.Is(err, ErrLLMNotConfigured) {
		t.Errorf("without an external run, error = %v, want ErrLLMNotConfigured", err)
	}
	ctx := WithExternalAI(context.Background(), NewExternalAIRun(ExternalAIPrompt, ""))
	if err := requireProvider(ctx, provider); err != nil {
		t.Errorf("with an external run, error = %v, want none", err)
	}
	if err := requireProvider(ctx, nil); !strings.Contains(err.Error(), "API key") {
		t.Errorf("a missing provider must still be refused, got %v", err)
	}
}
