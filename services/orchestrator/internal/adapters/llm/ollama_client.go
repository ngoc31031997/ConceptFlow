// Package llm implements application.MetadataSuggesterPort against a
// self-hosted Ollama instance (docker-compose service "ollama") — the
// Creator chose a local model over a paid API to avoid external API keys.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"orchestrator/internal/domain"
)

// OllamaClient calls Ollama's /api/generate with format:"json" so the model
// is constrained to emit a single JSON object we can parse directly.
type OllamaClient struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

func NewOllamaClient(baseURL, model string, timeout time.Duration) *OllamaClient {
	return &OllamaClient{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		model:      model,
		httpClient: &http.Client{Timeout: timeout},
	}
}

type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	Format string `json:"format"`
}

type generateResponse struct {
	Response string `json:"response"`
}

// suggestedMetadata mirrors the model's JSON output. Tags is left as
// json.RawMessage because models constrained only to "valid JSON" (not a
// fixed schema) sometimes emit tags as a comma-separated string instead of
// an array — normalizeTags below accepts either shape.
type suggestedMetadata struct {
	Title       string          `json:"title"`
	Description string          `json:"description"`
	Tags        json.RawMessage `json:"tags"`
}

func normalizeTags(raw json.RawMessage) []string {
	var asArray []string
	if err := json.Unmarshal(raw, &asArray); err == nil {
		return asArray
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		parts := strings.Split(asString, ",")
		tags := make([]string, 0, len(parts))
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				tags = append(tags, trimmed)
			}
		}
		return tags
	}
	return nil
}

// YouTube limits titles to 100 *characters*.
const maxTitleLength = 100

// buildSuggestPrompt asks for metadata in the project's content language
// (CR-008 FR21.3). This used to hardcode "tiếng Việt" three times and take no
// language at all, so an English channel got English narration and subtitles
// alongside a Vietnamese title, description and tags.
//
// The prompt itself is written in English regardless of the target language:
// instruction-following is more reliable when the instructions are in the
// language these models are most heavily trained on, and it keeps one prompt
// to maintain instead of one per language.
// truncateTitle caps a title at YouTube's limit by runes, not bytes.
//
// This used to be `title[:maxTitleLength]`, which slices bytes: a Vietnamese
// title (2-3 bytes per accented character) was cut mid-character, sending
// invalid UTF-8 to the YouTube API. Bytes are also simply the wrong unit —
// YouTube counts characters.
func truncateTitle(title string) string {
	if runes := []rune(title); len(runes) > maxTitleLength {
		return string(runes[:maxTitleLength])
	}
	return title
}

func buildSuggestPrompt(scriptContent, categoryHint string, language domain.ContentLanguage) string {
	languageName := domain.ProfileFor(language).EnglishName
	return fmt.Sprintf(`You are a YouTube SEO expert. Based on the video script below (topic: %q), produce:
- "title": a compelling, SEO-friendly title, at most %d characters, written in %s.
- "description": a 2-4 sentence SEO-friendly description with relevant keywords, written in %s.
- "tags": an array of 5-10 relevant keywords, written in %s, with no "#" characters.

Every piece of text you return must be written in %s.

Return exactly one JSON object with exactly the three keys "title", "description" and "tags". Do not add any explanation.

Script:
%s`, categoryHint, maxTitleLength, languageName, languageName, languageName, languageName, scriptContent)
}

// Suggest asks the model for an SEO-oriented YouTube title, description and
// tag list derived from the project's script content and category.
func (c *OllamaClient) Suggest(
	ctx context.Context, scriptContent, categoryHint string, language domain.ContentLanguage,
) (string, string, []string, error) {
	prompt := buildSuggestPrompt(scriptContent, categoryHint, language)

	reqBody, err := json.Marshal(generateRequest{Model: c.model, Prompt: prompt, Stream: false, Format: "json"})
	if err != nil {
		return "", "", nil, fmt.Errorf("marshal ollama request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/generate", bytes.NewReader(reqBody))
	if err != nil {
		return "", "", nil, fmt.Errorf("build ollama request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", "", nil, fmt.Errorf("call ollama: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", nil, fmt.Errorf("read ollama response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", "", nil, fmt.Errorf("ollama returned %d: %s", resp.StatusCode, string(body))
	}

	var genResp generateResponse
	if err := json.Unmarshal(body, &genResp); err != nil {
		return "", "", nil, fmt.Errorf("parse ollama envelope: %w", err)
	}

	var suggestion suggestedMetadata
	if err := json.Unmarshal([]byte(genResp.Response), &suggestion); err != nil {
		return "", "", nil, fmt.Errorf("parse model output as JSON: %w", err)
	}

	title := strings.TrimSpace(suggestion.Title)
	title = truncateTitle(title)
	return title, strings.TrimSpace(suggestion.Description), normalizeTags(suggestion.Tags), nil
}
