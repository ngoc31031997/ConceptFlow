package application

import (
	"context"
	"errors"
	"testing"

	"orchestrator/internal/domain"
)

type fakeShortScriptSuggester struct {
	script string
	err    error
	// captured for assertions on what the use case actually passed through.
	gotTopic, gotSource string
	gotLanguage         domain.ContentLanguage
}

func (f *fakeShortScriptSuggester) SuggestShortScript(_ context.Context, topic, source string, language domain.ContentLanguage) (string, error) {
	f.gotTopic, f.gotSource, f.gotLanguage = topic, source, language
	return f.script, f.err
}

func TestSuggestShortScriptUseCase_ReturnsTheDraft(t *testing.T) {
	suggester := &fakeShortScriptSuggester{script: "from conceptflow import *\n"}
	uc := NewSuggestShortScriptUseCase(suggester)

	script, err := uc.Execute(context.Background(), "Vòng lặp for trong Java", "", domain.LanguageVietnamese)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if script != "from conceptflow import *\n" {
		t.Fatalf("expected the suggester's script to pass through verbatim, got %q", script)
	}
	if suggester.gotTopic != "Vòng lặp for trong Java" {
		t.Fatalf("expected topic to reach the suggester, got %q", suggester.gotTopic)
	}
}

func TestSuggestShortScriptUseCase_RejectsEmptyTopicAndSource(t *testing.T) {
	// Neither a topic nor an existing script to draw context from — nothing
	// for the model to write about.
	uc := NewSuggestShortScriptUseCase(&fakeShortScriptSuggester{})

	_, err := uc.Execute(context.Background(), "  ", "  ", domain.LanguageVietnamese)
	if err == nil {
		t.Fatal("expected an error when both topic and source_script_content are blank")
	}
}

func TestSuggestShortScriptUseCase_SourceScriptContentAloneIsEnough(t *testing.T) {
	// Called from an existing long-form project's Result page: no separate
	// topic typed in, the long script itself is the context (CR-026 FR71.1).
	suggester := &fakeShortScriptSuggester{script: "script"}
	uc := NewSuggestShortScriptUseCase(suggester)

	_, err := uc.Execute(context.Background(), "", "from conceptflow import *\n...", domain.LanguageEnglish)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSuggestShortScriptUseCase_PropagatesSuggesterError(t *testing.T) {
	uc := NewSuggestShortScriptUseCase(&fakeShortScriptSuggester{err: errors.New("ollama unreachable")})

	_, err := uc.Execute(context.Background(), "chủ đề", "", domain.LanguageVietnamese)
	if err == nil {
		t.Fatal("expected the suggester's error to propagate")
	}
}
