// Package llm — this file implements application.LLMProviderPort against
// Hive's OpenAI-compatible chat-completions API (CR-027 FR76.2).
//
// No SDK. The wire format is two JSON structs and a bearer header; pulling in
// an OpenAI client library to save forty lines would buy a dependency whose
// own idea of the API drifts independently of Hive's.
package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"orchestrator/internal/application"
)

// HiveClient calls POST {baseURL}/chat/completions.
type HiveClient struct {
	baseURL    string
	apiKey     string
	model      string
	maxRetries int
	httpClient *http.Client
	sleep      func(time.Duration) // swapped out in tests
}

func NewHiveClient(baseURL, apiKey, model string, timeout time.Duration, maxRetries int) *HiveClient {
	return &HiveClient{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		apiKey:     apiKey,
		model:      model,
		maxRetries: maxRetries,
		httpClient: &http.Client{Timeout: timeout},
		sleep:      time.Sleep,
	}
}

func (c *HiveClient) Name() string { return "hive" }

type hiveMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	// ReasoningContent is what a reasoning model thought before answering.
	// Never shown to the Creator and never saved — it is read only so an
	// empty Content can be explained (see D13 / ErrKindBudget).
	ReasoningContent string `json:"reasoning_content"`
}

type hiveRequest struct {
	Model    string        `json:"model"`
	Messages []hiveMessage `json:"messages"`
	// omitempty: 0 means "no cap" (HIVE_MAX_OUTPUT_TOKENS=0) — the field is
	// left out and the provider applies its own ceiling.
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Temperature float64 `json:"temperature"`
	// Stream: Hive documents it as required. true → Server-Sent Events with
	// usage in the last chunk; the reply is read chunk by chunk so progress can
	// be reported and no hop sits idle for minutes waiting on one big body.
	Stream bool `json:"stream"`
}

type hiveChoice struct {
	Message      hiveMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// hiveUsage accepts BOTH shapes the two documented models actually return
// (measured 2026-09-21, CR-027 D12):
//
//	glm-5.3-flash       reasoning_tokens at the TOP level, prompt_tokens_details null
//	deepseek-v4.1-flash reasoning_tokens NESTED in completion_tokens_details,
//	                    plus prompt_tokens_details.cached_tokens
//
// Reading only one shape silently records zero for the other model, which is
// worse than not measuring at all: the usage screen would look fine and be
// wrong.
type hiveUsage struct {
	PromptTokens      int `json:"prompt_tokens"`
	CompletionTokens  int `json:"completion_tokens"`
	ReasoningTokens   int `json:"reasoning_tokens"`
	CompletionDetails *struct {
		ReasoningTokens int `json:"reasoning_tokens"`
	} `json:"completion_tokens_details"`
	PromptDetails *struct {
		CachedTokens int `json:"cached_tokens"`
	} `json:"prompt_tokens_details"`
}

func (u hiveUsage) reasoning() int {
	if u.ReasoningTokens > 0 {
		return u.ReasoningTokens
	}
	if u.CompletionDetails != nil {
		return u.CompletionDetails.ReasoningTokens
	}
	return 0
}

func (u hiveUsage) cached() int {
	if u.PromptDetails != nil {
		return u.PromptDetails.CachedTokens
	}
	return 0
}

type hiveResponse struct {
	Model   string       `json:"model"`
	Choices []hiveChoice `json:"choices"`
	Usage   hiveUsage    `json:"usage"`
	diag    *streamDiag  // set by readHiveStream only
}

// Chat runs one completion, retrying only the failures that retrying can fix.
func (c *HiveClient) Chat(ctx context.Context, req application.ChatRequest) (application.ChatResult, error) {
	if c.apiKey == "" {
		return application.ChatResult{}, &application.LLMError{
			Kind: application.ErrKindNotConfigured, Provider: c.Name(),
			Err: errors.New("HIVE_API_KEY is not set"),
		}
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			c.sleep(backoff(attempt))
		}
		result, err := c.chatOnce(ctx, req)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if !retryable(err) || ctx.Err() != nil {
			break
		}
	}
	return application.ChatResult{}, lastErr
}

// retryable: throttling and provider-side faults clear on their own; a bad
// key, an empty wallet or an exhausted token budget will answer identically
// however many times we ask.
func retryable(err error) bool {
	switch application.LLMErrorKindOf(err) {
	case application.ErrKindRateLimit, application.ErrKindServer:
		return true
	default:
		return false
	}
}

// backoff: 1s, 2s, 4s… plus jitter. Hive's documented default ceiling is 5
// requests per second, so retries that all fire on the same beat would keep
// re-colliding. Same shape as the AzureTTSAdapter backoff from CR-013.
func backoff(attempt int) time.Duration {
	base := time.Second << (attempt - 1)
	return base + time.Duration(rand.Int63n(int64(250*time.Millisecond)))
}

// hiveDiag accumulates what is known about one call as it progresses, so that
// whichever way it fails the error carries the API's own account of it — HTTP
// status, request id, finish_reason, the raw body — not just our summary.
type hiveDiag struct {
	started      time.Time
	model        string
	maxTokens    int
	temperature  float64
	systemChars  int
	userChars    int
	status       int
	requestID    string
	contentType  string
	finishReason string
	body         string // raw error body, or the last stream payload
	stream       *streamDiag
}

func (d *hiveDiag) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "request: model=%s max_tokens=%d temperature=%g system_chars=%d user_chars=%d",
		d.model, d.maxTokens, d.temperature, d.systemChars, d.userChars)
	fmt.Fprintf(&b, "\nelapsed: %s", time.Since(d.started).Round(time.Millisecond))
	if d.status != 0 {
		fmt.Fprintf(&b, "\nresponse: http=%d content_type=%q request_id=%q", d.status, d.contentType, d.requestID)
	} else {
		b.WriteString("\nresponse: none (no HTTP response received)")
	}
	if d.finishReason != "" {
		fmt.Fprintf(&b, "\nfinish_reason: %s", d.finishReason)
	}
	if st := d.stream; st != nil {
		fmt.Fprintf(&b, "\nstream: chunks=%d done_marker=%t reasoning_chars=%d content_chars=%d",
			st.chunks, st.sawDone, st.reasoningChars, st.contentChars)
		if st.errorPayload != "" {
			fmt.Fprintf(&b, "\nstream_error: %s", st.errorPayload)
		}
	}
	if d.body != "" {
		fmt.Fprintf(&b, "\nbody: %s", d.body)
	}
	return b.String()
}

func (c *HiveClient) chatOnce(ctx context.Context, req application.ChatRequest) (application.ChatResult, error) {
	diag := &hiveDiag{
		started: time.Now(), maxTokens: req.MaxTokens, temperature: req.Temperature,
		systemChars: len(req.System), userChars: len(req.User),
	}
	fail := func(kind application.LLMErrorKind, usage application.TokenUsage, err error) (application.ChatResult, error) {
		return application.ChatResult{}, &application.LLMError{
			Kind: kind, Provider: c.Name(), Usage: usage, Diag: diag.String(), Err: err,
		}
	}

	messages := make([]hiveMessage, 0, 2)
	if strings.TrimSpace(req.System) != "" {
		messages = append(messages, hiveMessage{Role: "system", Content: req.System})
	}
	messages = append(messages, hiveMessage{Role: "user", Content: req.User})

	// req.Model is the model-per-step picker's choice for this call; ""
	// (no override, or a project that never touched the picker) falls back
	// to the adapter's own configured model (HIVE_MODEL).
	requestModel := c.model
	if req.Model != "" {
		requestModel = req.Model
	}
	diag.model = requestModel
	body, err := json.Marshal(hiveRequest{
		Model: requestModel, Messages: messages,
		MaxTokens: req.MaxTokens, Temperature: req.Temperature, Stream: true,
	})
	if err != nil {
		return fail(application.ErrKindMalformed, application.TokenUsage{}, fmt.Errorf("marshal request: %w", err))
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return fail(application.ErrKindMalformed, application.TokenUsage{}, fmt.Errorf("build request: %w", err))
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		// A client-side deadline and a dead network arrive as the same error
		// type; the Creator-facing advice differs, so split them.
		kind := application.ErrKindServer
		if errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "Client.Timeout") {
			kind = application.ErrKindTimeout
		}
		return fail(kind, application.TokenUsage{}, fmt.Errorf("call hive: %w", err))
	}
	defer resp.Body.Close()
	diag.status = resp.StatusCode
	diag.contentType = resp.Header.Get("Content-Type")
	diag.requestID = firstHeader(resp.Header, "X-Request-Id", "X-Request-ID", "Request-Id", "Cf-Ray")

	var parsed hiveResponse
	if resp.StatusCode == http.StatusOK && strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
		var serr error
		parsed, serr = readHiveStream(resp.Body, req.OnProgress)
		diag.stream = parsed.diag
		if parsed.diag != nil {
			diag.body = parsed.diag.lastPayload
		}
		if serr != nil {
			kind := application.ErrKindServer
			if errors.Is(serr, context.DeadlineExceeded) || errors.Is(serr, context.Canceled) {
				kind = application.ErrKindTimeout
			}
			return fail(kind, application.TokenUsage{}, fmt.Errorf("read stream: %w", serr))
		}
	} else {
		raw, err := io.ReadAll(resp.Body)
		if err != nil {
			return fail(application.ErrKindServer, application.TokenUsage{}, fmt.Errorf("read response: %w", err))
		}
		if resp.StatusCode != http.StatusOK {
			diag.body = snippetN(raw, 4000)
			return fail(kindForStatus(resp.StatusCode), application.TokenUsage{},
				fmt.Errorf("hive returned %d: %s", resp.StatusCode, snippet(raw)))
		}
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return fail(application.ErrKindMalformed, application.TokenUsage{},
				fmt.Errorf("decode response: %w", err))
		}
	}

	model := parsed.Model
	if model == "" {
		// Hive did not echo the model back — fall back to the one this
		// call actually requested (the per-step override if there was one,
		// else the adapter's own default), not blindly c.model, or a usage
		// row for an overridden call would misreport which model was billed.
		model = requestModel
	}
	usage := application.TokenUsage{
		Model:            model,
		PromptTokens:     parsed.Usage.PromptTokens,
		CompletionTokens: parsed.Usage.CompletionTokens,
		ReasoningTokens:  parsed.Usage.reasoning(),
		CachedTokens:     parsed.Usage.cached(),
	}

	if parsed.diag != nil && parsed.diag.errorPayload != "" && parsed.diag.contentChars == 0 {
		return fail(application.ErrKindServer, usage,
			fmt.Errorf("hive sent an error inside the stream: %s", parsed.diag.errorPayload))
	}
	if len(parsed.Choices) == 0 {
		return fail(application.ErrKindEmpty, usage, errors.New("no choices in response"))
	}
	choice := parsed.Choices[0]
	diag.finishReason = choice.FinishReason
	content := strings.TrimSpace(choice.Message.Content)

	// D13 — "empty" and "ran out of room" are different problems with
	// different fixes, and a reasoning model turns the second into the first.
	// DeepSeek reports finish_reason "stop" even when max_tokens ended the
	// reply mid-reasoning (measured 2026-09-24): an empty answer that used the
	// whole budget is a budget problem, not an "empty" one.
	hitCap := req.MaxTokens > 0 && usage.CompletionTokens >= req.MaxTokens
	if choice.FinishReason == "length" || (content == "" && hitCap) {
		if content == "" {
			return fail(application.ErrKindBudget, usage, fmt.Errorf(
				"the whole token budget went on reasoning before any answer was written (%d reasoning tokens)",
				usage.ReasoningTokens))
		}
		return application.ChatResult{}, &application.LLMError{
			Kind: application.ErrKindTruncated, Provider: c.Name(), Usage: usage,
			Partial: content, Diag: diag.String(), Err: errors.New("response was cut off at max_tokens"),
		}
	}
	if content == "" {
		return fail(application.ErrKindEmpty, usage, errors.New("model returned empty content"))
	}

	return application.ChatResult{Content: content, Usage: usage}, nil
}

func kindForStatus(status int) application.LLMErrorKind {
	switch {
	case status == http.StatusUnauthorized, status == http.StatusForbidden:
		return application.ErrKindAuth
	case status == http.StatusPaymentRequired:
		return application.ErrKindBalance
	case status == http.StatusTooManyRequests:
		return application.ErrKindRateLimit
	case status >= 500:
		return application.ErrKindServer
	default:
		return application.ErrKindMalformed
	}
}

// snippet keeps a provider error body short enough to log without dumping an
// HTML error page into the logs.
func snippet(body []byte) string { return snippetN(body, 300) }

func snippetN(body []byte, max int) string {
	s := strings.TrimSpace(string(body))
	if len(s) > max {
		return s[:max] + "…"
	}
	return s
}

func firstHeader(h http.Header, names ...string) string {
	for _, n := range names {
		if v := h.Get(n); v != "" {
			return v
		}
	}
	return ""
}

// streamChunk is one SSE "data:" payload. Reasoning models put their thinking
// in delta.reasoning_content and the answer in delta.content; the final chunk
// has no choices and carries usage.
type streamChunk struct {
	Model   string `json:"model"`
	Choices []struct {
		Delta struct {
			Content          *string `json:"content"`
			ReasoningContent *string `json:"reasoning_content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage *hiveUsage `json:"usage"`
	// Error is set by an API that fails mid-stream: it answers 200 and then
	// sends {"error": ...} as an ordinary data chunk. Ignoring it turns a
	// provider fault into a mysterious empty answer.
	Error json.RawMessage `json:"error"`
}

// streamDiag is what a stream looked like, for the error log.
type streamDiag struct {
	chunks         int
	sawDone        bool
	reasoningChars int
	contentChars   int
	errorPayload   string
	lastPayload    string
}

// readHiveStream folds an SSE reply into the same hiveResponse the
// non-streaming path decodes, reporting running sizes to onProgress.
func readHiveStream(body io.Reader, onProgress func(application.ChatProgress)) (hiveResponse, error) {
	var (
		out       hiveResponse
		content   strings.Builder
		reasoning int
		finish    string
	)
	sd := &streamDiag{}
	out.diag = sd
	rd := bufio.NewReaderSize(body, 64*1024)
	for {
		line, err := rd.ReadString('\n')
		line = strings.TrimSpace(line)
		if payload, ok := strings.CutPrefix(line, "data:"); ok {
			payload = strings.TrimSpace(payload)
			if payload == "[DONE]" {
				sd.sawDone = true
				break
			}
			sd.chunks++
			// Keep the last payload that was not a plain delta: usage chunks and
			// error chunks are the interesting ones, and deltas are the bulk.
			var chunk streamChunk
			if jerr := json.Unmarshal([]byte(payload), &chunk); jerr != nil {
				sd.lastPayload = snippetN([]byte(payload), 1000)
			} else {
				if len(chunk.Error) > 0 && string(chunk.Error) != "null" {
					sd.errorPayload = snippetN(chunk.Error, 2000)
					sd.lastPayload = snippetN([]byte(payload), 2000)
				} else if len(chunk.Choices) == 0 {
					sd.lastPayload = snippetN([]byte(payload), 1000)
				}
				if chunk.Model != "" {
					out.Model = chunk.Model
				}
				if chunk.Usage != nil {
					out.Usage = *chunk.Usage
				}
				for _, ch := range chunk.Choices {
					if ch.Delta.Content != nil {
						content.WriteString(*ch.Delta.Content)
					}
					if ch.Delta.ReasoningContent != nil {
						reasoning += len(*ch.Delta.ReasoningContent)
					}
					if ch.FinishReason != "" {
						finish = ch.FinishReason
					}
				}
				if onProgress != nil && len(chunk.Choices) > 0 {
					onProgress(application.ChatProgress{ReasoningChars: reasoning, ContentChars: content.Len()})
				}
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return hiveResponse{}, err
		}
	}
	sd.reasoningChars, sd.contentChars = reasoning, content.Len()
	out.Choices = []hiveChoice{{
		Message:      hiveMessage{Role: "assistant", Content: content.String()},
		FinishReason: finish,
	}}
	return out, nil
}
