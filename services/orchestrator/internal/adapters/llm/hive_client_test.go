package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"orchestrator/internal/application"
)

// asLLMError is errors.As with the import kept local to this file.
func asLLMError(err error, target **application.LLMError) bool {
	return errors.As(err, target)
}

func newHiveTestClient(t *testing.T, handler http.HandlerFunc) (*HiveClient, func()) {
	t.Helper()
	srv := httptest.NewServer(handler)
	c := NewHiveClient(srv.URL, "test-key", "test-model", 5*time.Second, 2)
	c.sleep = func(time.Duration) {} // no real waiting in tests
	return c, srv.Close
}

func okBody(content, finish string, usage map[string]any) string {
	body := map[string]any{
		"model": "test-model",
		"choices": []map[string]any{{
			"message":       map[string]any{"role": "assistant", "content": content},
			"finish_reason": finish,
		}},
		"usage": usage,
	}
	b, _ := json.Marshal(body)
	return string(b)
}

func TestChat_SendsBearerKeyAndBothMessages(t *testing.T) {
	var gotAuth string
	var gotReq hiveRequest
	c, done := newHiveTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotReq)
		_, _ = w.Write([]byte(okBody("hello", "stop", map[string]any{})))
	})
	defer done()

	if _, err := c.Chat(context.Background(), application.ChatRequest{
		System: "SYS", User: "USR", MaxTokens: 99,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotAuth != "Bearer test-key" {
		t.Fatalf("expected a bearer key header, got %q", gotAuth)
	}
	if len(gotReq.Messages) != 2 || gotReq.Messages[0].Role != "system" || gotReq.Messages[1].Role != "user" {
		t.Fatalf("expected a system then a user message, got %+v", gotReq.Messages)
	}
	if gotReq.MaxTokens != 99 {
		t.Fatalf("max_tokens did not reach the provider, got %d", gotReq.MaxTokens)
	}
}

func TestChat_OmitsTheSystemMessageWhenEmpty(t *testing.T) {
	var gotReq hiveRequest
	c, done := newHiveTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotReq)
		_, _ = w.Write([]byte(okBody("hi", "stop", map[string]any{})))
	})
	defer done()

	if _, err := c.Chat(context.Background(), application.ChatRequest{User: "only user"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotReq.Messages) != 1 || gotReq.Messages[0].Role != "user" {
		t.Fatalf("an empty system prompt must not become an empty message: %+v", gotReq.Messages)
	}
}

// TestChat_ReadsBothUsageShapes is the CR-027 D12 regression. The two models
// Hive documents report reasoning tokens in two different places (measured
// 2026-09-21); a parser written for one silently records zero for the other,
// which makes the usage screen confidently wrong.
func TestChat_ReadsBothUsageShapes(t *testing.T) {
	cases := []struct {
		name          string
		usage         map[string]any
		wantReasoning int
		wantCached    int
	}{
		{
			name: "glm puts reasoning_tokens at the top level",
			usage: map[string]any{
				"prompt_tokens": 29, "completion_tokens": 122,
				"reasoning_tokens": 66, "prompt_tokens_details": nil,
			},
			wantReasoning: 66,
		},
		{
			name: "deepseek nests it in completion_tokens_details",
			usage: map[string]any{
				"prompt_tokens": 28, "completion_tokens": 62,
				"completion_tokens_details": map[string]any{"reasoning_tokens": 7},
				"prompt_tokens_details":     map[string]any{"cached_tokens": 20},
			},
			wantReasoning: 7,
			wantCached:    20,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, done := newHiveTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(okBody("answer", "stop", tc.usage)))
			})
			defer done()

			res, err := c.Chat(context.Background(), application.ChatRequest{User: "q"})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.Usage.ReasoningTokens != tc.wantReasoning {
				t.Fatalf("reasoning tokens: want %d, got %d", tc.wantReasoning, res.Usage.ReasoningTokens)
			}
			if res.Usage.CachedTokens != tc.wantCached {
				t.Fatalf("cached tokens: want %d, got %d", tc.wantCached, res.Usage.CachedTokens)
			}
		})
	}
}

// TestChat_EmptyContentIsNotAlwaysTheSameFailure covers CR-027 D13. Measured:
// glm-5.3-flash with a small max_tokens returns empty content, a filled
// reasoning_content and finish_reason "length" — the budget went on thinking.
// Telling the Creator "the model returned nothing" would send them to rewrite
// a prompt that is fine.
func TestChat_EmptyContentIsNotAlwaysTheSameFailure(t *testing.T) {
	cases := []struct {
		name     string
		content  string
		finish   string
		wantKind application.LLMErrorKind
	}{
		{"budget eaten by reasoning", "", "length", application.ErrKindBudget},
		{"cut off mid-answer", "half an ans", "length", application.ErrKindTruncated},
		{"finished cleanly with nothing", "", "stop", application.ErrKindEmpty},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, done := newHiveTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(okBody(tc.content, tc.finish,
					map[string]any{"prompt_tokens": 10, "completion_tokens": 5, "reasoning_tokens": 5})))
			})
			defer done()

			_, err := c.Chat(context.Background(), application.ChatRequest{User: "q"})
			if got := application.LLMErrorKindOf(err); got != tc.wantKind {
				t.Fatalf("want kind %q, got %q (err=%v)", tc.wantKind, got, err)
			}
		})
	}
}

// TestChat_BilledTokensSurviveAFailure — a truncated answer still costs
// money, so llm_usage must be able to record it (FR82).
func TestChat_BilledTokensSurviveAFailure(t *testing.T) {
	c, done := newHiveTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(okBody("cut", "length",
			map[string]any{"prompt_tokens": 500, "completion_tokens": 16000})))
	})
	defer done()

	_, err := c.Chat(context.Background(), application.ChatRequest{User: "q"})
	var llmErr *application.LLMError
	if !asLLMError(err, &llmErr) {
		t.Fatalf("expected an LLMError, got %v", err)
	}
	if llmErr.Usage.PromptTokens != 500 || llmErr.Usage.CompletionTokens != 16000 {
		t.Fatalf("billed tokens were dropped on failure: %+v", llmErr.Usage)
	}
}

func TestChat_StatusToErrorKind(t *testing.T) {
	cases := []struct {
		status   int
		wantKind application.LLMErrorKind
	}{
		{http.StatusUnauthorized, application.ErrKindAuth},
		{http.StatusForbidden, application.ErrKindAuth},
		{http.StatusPaymentRequired, application.ErrKindBalance},
		{http.StatusTooManyRequests, application.ErrKindRateLimit},
		{http.StatusInternalServerError, application.ErrKindServer},
		{http.StatusBadRequest, application.ErrKindMalformed},
	}
	for _, tc := range cases {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			c, done := newHiveTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(`{"message":"nope"}`))
			})
			defer done()

			_, err := c.Chat(context.Background(), application.ChatRequest{User: "q"})
			if got := application.LLMErrorKindOf(err); got != tc.wantKind {
				t.Fatalf("status %d: want %q, got %q", tc.status, tc.wantKind, got)
			}
		})
	}
}

// TestChat_RetriesOnlyWhatRetryingCanFix — throttling and provider faults
// clear on their own; a dead key answers identically however many times we
// ask, and each extra attempt is another billable request.
func TestChat_RetriesOnlyWhatRetryingCanFix(t *testing.T) {
	cases := []struct {
		name      string
		status    int
		wantCalls int32
	}{
		{"rate limit is retried", http.StatusTooManyRequests, 3}, // 1 + maxRetries(2)
		{"server error is retried", http.StatusInternalServerError, 3},
		{"bad key is not retried", http.StatusUnauthorized, 1},
		{"empty wallet is not retried", http.StatusPaymentRequired, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var calls int32
			c, done := newHiveTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				atomic.AddInt32(&calls, 1)
				w.WriteHeader(tc.status)
			})
			defer done()

			_, _ = c.Chat(context.Background(), application.ChatRequest{User: "q"})
			if got := atomic.LoadInt32(&calls); got != tc.wantCalls {
				t.Fatalf("want %d attempts, got %d", tc.wantCalls, got)
			}
		})
	}
}

func TestChat_RecoversWhenARetrySucceeds(t *testing.T) {
	var calls int32
	c, done := newHiveTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(okBody("second time lucky", "stop", map[string]any{})))
	})
	defer done()

	res, err := c.Chat(context.Background(), application.ChatRequest{User: "q"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Content != "second time lucky" {
		t.Fatalf("got %q", res.Content)
	}
}

// TestChat_WithoutAKeyFailsFastAndSaysWhy — FR83.2: running without a key is
// a supported state, so it must not look like a provider outage.
func TestChat_WithoutAKeyFailsFastAndSaysWhy(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
	}))
	defer srv.Close()
	c := NewHiveClient(srv.URL, "", "test-model", time.Second, 2)

	_, err := c.Chat(context.Background(), application.ChatRequest{User: "q"})
	if got := application.LLMErrorKindOf(err); got != application.ErrKindNotConfigured {
		t.Fatalf("want %q, got %q", application.ErrKindNotConfigured, got)
	}
	if atomic.LoadInt32(&calls) != 0 {
		t.Fatal("a missing key must not reach the network")
	}
	if !strings.Contains(err.Error(), "HIVE_API_KEY") {
		t.Fatalf("the error should name the variable to set, got %v", err)
	}
}

func TestChat_MalformedJSONIsItsOwnKind(t *testing.T) {
	c, done := newHiveTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html>not json</html>"))
	})
	defer done()

	_, err := c.Chat(context.Background(), application.ChatRequest{User: "q"})
	if got := application.LLMErrorKindOf(err); got != application.ErrKindMalformed {
		t.Fatalf("want %q, got %q", application.ErrKindMalformed, got)
	}
}

func TestChat_TrimsWhitespaceAroundTheAnswer(t *testing.T) {
	c, done := newHiveTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(okBody("\n\n  answer  \n", "stop", map[string]any{})))
	})
	defer done()

	res, err := c.Chat(context.Background(), application.ChatRequest{User: "q"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Content != "answer" {
		t.Fatalf("got %q", res.Content)
	}
}
