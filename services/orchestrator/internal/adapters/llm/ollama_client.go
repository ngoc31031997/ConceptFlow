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

const maxTitleLength = 100

// Suggest asks the model for an SEO-oriented YouTube title, description and
// tag list derived from the project's script content and category.
func (c *OllamaClient) Suggest(ctx context.Context, scriptContent, categoryHint string) (string, string, []string, error) {
	prompt := fmt.Sprintf(`Bạn là chuyên gia SEO YouTube. Dựa trên kịch bản video dưới đây (chủ đề: %q), hãy tạo:
- "title": tiêu đề hấp dẫn, chuẩn SEO, tối đa 100 ký tự, tiếng Việt.
- "description": mô tả 2-4 câu, chuẩn SEO, có từ khóa liên quan, tiếng Việt.
- "tags": mảng 5-10 từ khóa liên quan (tiếng Việt, không dấu # ).

Chỉ trả về duy nhất một JSON object với đúng 3 khóa "title", "description", "tags". Không thêm giải thích.

Kịch bản:
%s`, categoryHint, scriptContent)

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
	if len(title) > maxTitleLength {
		title = title[:maxTitleLength]
	}
	return title, strings.TrimSpace(suggestion.Description), normalizeTags(suggestion.Tags), nil
}
