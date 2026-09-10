package application

import (
	"context"
	"fmt"

	"orchestrator/internal/domain"
)

// MetadataSuggesterPort is implemented by the Ollama adapter — kept generic
// (not "OllamaPort") so a different local/self-hosted model can be swapped
// in later without touching this use case.
type MetadataSuggesterPort interface {
	// language is the project's ContentLanguage (CR-008 FR21.3) — the metadata
	// must come back in the language the audience speaks, not the one the
	// prompt happens to be written in.
	Suggest(ctx context.Context, scriptContent, categoryHint string, language domain.ContentLanguage) (title, description string, tags []string, err error)
}

// SuggestPublishMetadataOutput carries the AI-drafted YouTube metadata back
// to the HTTP layer. It is a suggestion only — the Creator reviews/edits it
// in PublishForm before it is ever sent to StartPublishSagaUseCase.
type SuggestPublishMetadataOutput struct {
	Title       string
	Description string
	Tags        []string
}

// SuggestPublishMetadataUseCase drafts an SEO-oriented title/description/tags
// from the project's script content, via a self-hosted LLM (no Saga step —
// this never mutates Project or dispatches a command).
type SuggestPublishMetadataUseCase struct {
	repo      domain.ProjectRepositoryPort
	suggester MetadataSuggesterPort
}

func NewSuggestPublishMetadataUseCase(repo domain.ProjectRepositoryPort, suggester MetadataSuggesterPort) *SuggestPublishMetadataUseCase {
	return &SuggestPublishMetadataUseCase{repo: repo, suggester: suggester}
}

func (uc *SuggestPublishMetadataUseCase) Execute(ctx context.Context, projectID string) (*SuggestPublishMetadataOutput, error) {
	project, err := uc.repo.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if project.ScriptContent == "" {
		return nil, fmt.Errorf("project has no script content to summarize")
	}

	title, summary, tags, err := uc.suggester.Suggest(
		ctx, project.ScriptContent, project.CategoryHint, project.ContentLanguage,
	)
	if err != nil {
		return nil, fmt.Errorf("suggest metadata: %w", err)
	}

	// CR-006 FR15.2/FR18.2 — chapters come from the Creator's own markers,
	// timed by the offsets Rendering measured, so they land on the same frame
	// as the narration that introduces them. The model is never asked to guess
	// timestamps; it could not know them.
	// CR-023 D6: chapter timestamps must land after the channel intro, when
	// this project actually has one attached — intro duration is not plumbed
	// through to metadata suggestion yet (it is resolved at assemble_video
	// time from channel_assets, which this use case never touches), so 0.0
	// preserves the exact pre-CR-023 behaviour here.
	chapterLines := domain.BuildChapterTimestamps(
		project.Chapters, project.WaitOffsets, project.RenderedVideoSeconds, 0.0,
	)
	description := domain.ComposeDescription(
		summary, chapterLines, callToActionFor(project.ContentLanguage), tags,
	)

	return &SuggestPublishMetadataOutput{Title: title, Description: description, Tags: tags}, nil
}

// callToActionFor is written per language rather than generated, since it is
// fixed boilerplate — spending model tokens and risking a bad translation on
// one unchanging sentence would be worse than writing it once.
func callToActionFor(language domain.ContentLanguage) string {
	if language == domain.LanguageVietnamese {
		return "Nếu video hữu ích, hãy đăng ký kênh để không bỏ lỡ những bài tiếp theo."
	}
	return "If this helped, subscribe so you don't miss the next one."
}
