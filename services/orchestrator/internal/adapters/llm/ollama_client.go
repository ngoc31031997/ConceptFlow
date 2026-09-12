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
	"regexp"
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

// normalizeTags never returns nil: a nil slice marshals to JSON `null`, and
// the API contract says tags is an array. Returning null made the GUI crash on
// `tags.join(", ")` the moment a model omitted the key.
func normalizeTags(raw json.RawMessage) []string {
	var asArray []string
	if err := json.Unmarshal(raw, &asArray); err == nil && asArray != nil {
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
	return []string{}
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

// maxScriptChars caps how much of the script reaches the model.
//
// Ollama defaults num_ctx to 2048 tokens. A full ten-minute script runs to
// ~17,000 characters, which overflows that window and makes the model return a
// well-formed but empty JSON object — measured: a 17.5k-character project
// produced no title on every attempt, while a 12.7k one was already marginal.
// Raising num_ctx instead would cost memory and latency on every request to fix
// a problem the model does not actually need the whole script to avoid: a title
// and a 2-4 sentence description are drawn from the opening, which is where the
// topic is stated.
const maxScriptChars = 4000

// truncateScript cuts by runes, not bytes — Vietnamese is multibyte, and
// slicing mid-rune would hand the model invalid UTF-8 (same reason as
// truncateTitle).
func truncateScript(script string) string {
	if runes := []rune(script); len(runes) > maxScriptChars {
		return string(runes[:maxScriptChars])
	}
	return script
}

func buildSuggestPrompt(scriptContent, categoryHint string, language domain.ContentLanguage) string {
	scriptContent = truncateScript(scriptContent)
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

// buildShortScriptSuggestionPrompt (CR-026 FR71) asks for a genuinely
// condensed Manim script — not a summary of the long-form script's Python
// code (that would be summarizing animation code, which does not mean
// anything), but a new short script on the same topic. sourceScriptContent,
// when present, is truncated the same way buildSuggestPrompt already does:
// only enough of the long-form narration to tell the model what the topic
// actually is, not the whole thing.
//
// No format:"json" here (unlike Suggest): the output is multi-line Python
// source, and forcing JSON would make the model escape every newline/quote,
// which is a much easier way to get invalid Python back than plain text is.
func buildShortScriptSuggestionPrompt(topic, sourceScriptContent string, language domain.ContentLanguage) string {
	languageName := domain.ProfileFor(language).EnglishName
	context := topic
	if strings.TrimSpace(sourceScriptContent) != "" {
		context = fmt.Sprintf("%s\n\nExisting long-form script on this topic, for context only — do not summarize or transform its code, just reuse the topic and key idea it teaches:\n%s", topic, truncateScript(sourceScriptContent))
	}
	return fmt.Sprintf(`You are writing a SHORT-FORM educational video script (Manim Community Edition v0.18) for YouTube Shorts/TikTok — 30 to 60 seconds of narration, ONE idea, no filler, hook in the first 2 seconds. This is NOT a trimmed-down version of a longer video; it must stand completely on its own.

Topic:
%s

Hard rules (the render pipeline rejects anything that breaks these):
1. Start with exactly: from conceptflow import *
2. Exactly one class, inheriting ConceptFlowScene, name ending in "Scene":
   class <TopicName>Scene(ConceptFlowScene):
       def construct(self):
           ...
3. The ENTIRE body of construct() must be wrapped in exactly one:
   with self.clip("short"):
       ...
   This is mandatory, not optional — it is how the pipeline recognizes this as a Shorts/TikTok clip.
4. Every line of narration is a call, not a comment: self.narrate("...")
5. Do not set colors, font sizes, or backgrounds by hand — the design system (ConceptFlowScene) handles all of that.
6. Available components: TitleCard, Callout, CodePanel, StepList, ComparisonSplit, Recap. Available scene methods: self.narrate(...), self.hook(...), self.recap([...]), self.call_to_action(...), self.title/heading/body/caption/formula/code(...), self.stack/row/fit(...), self.reveal/dismiss/swap/emphasize/clear_stage(...).
7. All narration text must be written in %s.

Before answering, check: does construct() start with a single with self.clip("short"): wrapping everything else? Is there exactly one Scene class? Is every spoken line a self.narrate(...) call, not a comment?

Respond with ONLY one Python code block (wrapped in `+"```python ... ```"+`), no explanation before or after it.`, context, languageName)
}

// stripCodeFence mirrors scriptPrompts.ts's stripMarkdownCodeFence — this is
// a different runtime (Go, not the browser), so it cannot import that file;
// duplicated deliberately rather than shared, since the two only need to
// agree on "what a fenced code block looks like", not stay in lockstep.
func stripCodeFence(script string) string {
	trimmed := strings.TrimSpace(script)
	fenceRe := regexp.MustCompile("(?s)```[a-zA-Z0-9]*\\r?\\n(.*?)\\r?\\n?```")
	if match := fenceRe.FindStringSubmatch(trimmed); match != nil {
		return match[1]
	}
	return script
}

// SuggestShortScript drafts a standalone short-form script for the same
// topic as scriptContent/topic (CR-026 FR71). Retries the same number of
// times as Suggest, for the same reason: a small local model occasionally
// returns something unusable on the first try, and one retry is cheap next
// to a Creator watching a spinner.
func (c *OllamaClient) SuggestShortScript(
	ctx context.Context, topic, sourceScriptContent string, language domain.ContentLanguage,
) (string, error) {
	var lastErr error
	for attempt := 1; attempt <= suggestMaxAttempts; attempt++ {
		script, err := c.suggestShortScriptOnce(ctx, topic, sourceScriptContent, language)
		if err == nil {
			return script, nil
		}
		lastErr = err
		if ctx.Err() != nil {
			break
		}
	}
	return "", lastErr
}

func (c *OllamaClient) suggestShortScriptOnce(
	ctx context.Context, topic, sourceScriptContent string, language domain.ContentLanguage,
) (string, error) {
	prompt := buildShortScriptSuggestionPrompt(topic, sourceScriptContent, language)

	reqBody, err := json.Marshal(generateRequest{Model: c.model, Prompt: prompt, Stream: false})
	if err != nil {
		return "", fmt.Errorf("marshal ollama request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/generate", bytes.NewReader(reqBody))
	if err != nil {
		return "", fmt.Errorf("build ollama request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("call ollama: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read ollama response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama returned %d: %s", resp.StatusCode, string(body))
	}

	var genResp generateResponse
	if err := json.Unmarshal(body, &genResp); err != nil {
		return "", fmt.Errorf("parse ollama envelope: %w", err)
	}

	script := strings.TrimSpace(stripCodeFence(genResp.Response))
	if script == "" {
		return "", fmt.Errorf("model returned an empty script")
	}
	return script, nil
}

// Suggest asks the model for an SEO-oriented YouTube title, description and
// tag list derived from the project's script content and category.
func (c *OllamaClient) Suggest(
	ctx context.Context, scriptContent, categoryHint string, language domain.ContentLanguage,
) (string, string, []string, error) {
	// A small local model asked only for "valid JSON" sometimes returns a
	// well-formed object with an empty title and no usable tags. That parses
	// fine, so nothing used to catch it and the Creator got blank fields.
	// One retry costs a few seconds and recovers most of those; two attempts
	// rather than four because each call is slow and a browser is waiting.
	var lastErr error
	for attempt := 1; attempt <= suggestMaxAttempts; attempt++ {
		title, description, tags, err := c.suggestOnce(ctx, scriptContent, categoryHint, language)
		if err == nil {
			return title, description, tags, nil
		}
		lastErr = err
		// Stop immediately if the caller gave up: retrying into a dead context
		// only delays the error the Creator is already waiting for.
		if ctx.Err() != nil {
			break
		}
	}
	return "", "", nil, lastErr
}

const suggestMaxAttempts = 2

func (c *OllamaClient) suggestOnce(
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

	title := truncateTitle(strings.TrimSpace(suggestion.Title))
	if title == "" {
		// Valid JSON, useless content. Reported as an error so the Creator sees
		// "gợi ý thất bại" and can retry, instead of a silently blank title
		// field they might publish as-is.
		return "", "", nil, fmt.Errorf("model returned no title")
	}
	return title, strings.TrimSpace(suggestion.Description), normalizeTags(suggestion.Tags), nil
}
