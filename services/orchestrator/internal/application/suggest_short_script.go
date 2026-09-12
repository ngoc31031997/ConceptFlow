package application

import (
	"context"
	"fmt"
	"strings"

	"orchestrator/internal/domain"
)

// ShortScriptSuggesterPort is implemented by the Ollama adapter (CR-026
// FR71) — kept separate from MetadataSuggesterPort because the two draft
// entirely different things (SEO metadata vs. Manim source code) with
// different prompts and no shared response shape.
type ShortScriptSuggesterPort interface {
	SuggestShortScript(ctx context.Context, topic, sourceScriptContent string, language domain.ContentLanguage) (script string, err error)
}

// SuggestShortScriptUseCase drafts a standalone short-form Manim script
// (CR-026 FR71) — no Saga step, no Project mutation. Takes no project_id: a
// Creator can start a short from scratch, not only from an existing
// long-form project (FR71.1's source_script_content is optional context).
type SuggestShortScriptUseCase struct {
	suggester ShortScriptSuggesterPort
}

func NewSuggestShortScriptUseCase(suggester ShortScriptSuggesterPort) *SuggestShortScriptUseCase {
	return &SuggestShortScriptUseCase{suggester: suggester}
}

func (uc *SuggestShortScriptUseCase) Execute(
	ctx context.Context, topic, sourceScriptContent string, language domain.ContentLanguage,
) (string, error) {
	if strings.TrimSpace(topic) == "" && strings.TrimSpace(sourceScriptContent) == "" {
		return "", fmt.Errorf("topic or source_script_content is required")
	}

	script, err := uc.suggester.SuggestShortScript(ctx, topic, sourceScriptContent, language)
	if err != nil {
		return "", fmt.Errorf("suggest short script: %w", err)
	}
	return script, nil
}
