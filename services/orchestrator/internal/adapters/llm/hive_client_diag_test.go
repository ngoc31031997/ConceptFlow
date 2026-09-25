package llm

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"orchestrator/internal/application"
)

func TestChat_HTTPErrorCarriesStatusRequestIDAndBody(t *testing.T) {
	c, done := newHiveTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-Id", "req-123")
		w.WriteHeader(http.StatusPaymentRequired)
		_, _ = w.Write([]byte(`{"error":{"message":"insufficient balance","code":"no_credit"}}`))
	})
	defer done()

	_, err := c.Chat(context.Background(), application.ChatRequest{System: "s", User: "u", MaxTokens: 50})
	var llmErr *application.LLMError
	if !asLLMError(err, &llmErr) {
		t.Fatalf("want LLMError, got %v", err)
	}
	for _, want := range []string{"http=402", "req-123", "insufficient balance", "max_tokens=50", "model=test-model"} {
		if !strings.Contains(llmErr.Diag, want) {
			t.Errorf("Diag missing %q:\n%s", want, llmErr.Diag)
		}
	}
}

// Hive can answer 200 and then fail inside the stream. That used to surface as
// a bare "empty answer".
func TestChat_ErrorInsideStreamIsSurfaced(t *testing.T) {
	c, done := newHiveTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"error\":{\"message\":\"upstream model overloaded\"}}\n\n"))
	})
	defer done()

	_, err := c.Chat(context.Background(), application.ChatRequest{User: "u"})
	var llmErr *application.LLMError
	if !asLLMError(err, &llmErr) {
		t.Fatalf("want LLMError, got %v", err)
	}
	if llmErr.Kind != application.ErrKindServer || !strings.Contains(llmErr.Err.Error(), "overloaded") {
		t.Errorf("kind=%s err=%v, want the stream's own message under kind server", llmErr.Kind, llmErr.Err)
	}
	if !strings.Contains(llmErr.Diag, "stream_error") {
		t.Errorf("Diag = %s", llmErr.Diag)
	}
}

func TestChat_TruncationDiagShowsFinishReasonAndStreamShape(t *testing.T) {
	c, done := newHiveTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"abc\"}}]}\n\n" +
			"data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"length\"}]}\n\n" +
			"data: {\"usage\":{\"prompt_tokens\":1,\"completion_tokens\":9}}\n\n" +
			"data: [DONE]\n\n"))
	})
	defer done()

	_, err := c.Chat(context.Background(), application.ChatRequest{User: "u", MaxTokens: 9})
	var llmErr *application.LLMError
	if !asLLMError(err, &llmErr) || llmErr.Kind != application.ErrKindTruncated {
		t.Fatalf("want truncated, got %v", err)
	}
	for _, want := range []string{"finish_reason: length", "done_marker=true", "content_chars=3", "completion_tokens"} {
		if !strings.Contains(llmErr.Diag, want) {
			t.Errorf("Diag missing %q:\n%s", want, llmErr.Diag)
		}
	}
}
