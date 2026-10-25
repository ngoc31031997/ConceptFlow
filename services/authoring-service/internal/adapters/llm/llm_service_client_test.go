package llm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"authoring/internal/application"
	"authoring/internal/domain"
)

func ndjson(w http.ResponseWriter, lines ...string) {
	w.Header().Set("Content-Type", "application/x-ndjson")
	for _, l := range lines {
		_, _ = w.Write([]byte(l + "\n"))
	}
}

func serve(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, 5*time.Second)
}

func TestChatStreamsProgressThenReturnsTheResult(t *testing.T) {
	var gotBody map[string]any
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		ndjson(w,
			`{"type":"progress","reasoning_chars":12,"content_chars":0}`,
			`{"type":"progress","reasoning_chars":12,"content_chars":40}`,
			`{"type":"result","content":"xin chào","provider":"hive","usage":{"model":"deepseek","prompt_tokens":10,"completion_tokens":5,"reasoning_tokens":3,"cached_tokens":2}}`)
	})

	var seen []application.ChatProgress
	res, err := c.Chat(context.Background(), application.ChatRequest{
		System: "SYS", User: "USR", Model: "m", MaxTokens: 99, Temperature: 0.5,
		OnProgress: func(p application.ChatProgress) { seen = append(seen, p) },
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if res.Content != "xin chào" || res.Usage.ReasoningTokens != 3 || res.Usage.CachedTokens != 2 || res.Usage.Model != "deepseek" {
		t.Errorf("result = %+v", res)
	}
	if len(seen) != 2 || seen[1].ContentChars != 40 {
		t.Errorf("progress = %+v", seen)
	}
	if gotBody["system"] != "SYS" || gotBody["user"] != "USR" || gotBody["model"] != "m" || gotBody["max_tokens"] != float64(99) {
		t.Errorf("request body = %v", gotBody)
	}
}

func TestChatSendsWhereTheCallComesFromAndReadsTheJSONAndTheCallID(t *testing.T) {
	var gotBody map[string]any
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		ndjson(w, `{"type":"result","content":"raw","json":{"title":"T"},"provider":"hive","call_id":"c-1","usage":{}}`)
	})
	res, err := c.Chat(context.Background(), application.ChatRequest{
		User: "u", ProjectID: "p1", Role: "story_frame", Step: "story", Phase: "write", ExpectJSON: true,
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if string(res.JSON) != `{"title":"T"}` || res.CallID != "c-1" {
		t.Errorf("result = %+v", res)
	}
	if gotBody["project_id"] != "p1" || gotBody["role"] != "story_frame" || gotBody["step"] != "story" ||
		gotBody["phase"] != "write" || gotBody["expect_json"] != true {
		t.Errorf("request body = %v", gotBody)
	}
}

func TestChatThatExpectedJSONAndGotNoneIsAMalformedErrorWithTheCallID(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		ndjson(w, `{"type":"result","content":"raw","provider":"hive","call_id":"c-2","usage":{}}`)
	})
	_, err := c.Chat(context.Background(), application.ChatRequest{User: "u", ExpectJSON: true})
	var llmErr *application.LLMError
	if !asLLMError(err, &llmErr) || llmErr.Kind != application.ErrKindMalformed || llmErr.CallID != "c-2" {
		t.Fatalf("err = %v, want a malformed LLMError carrying c-2", err)
	}
	if !strings.Contains(err.Error(), "c-2") {
		t.Errorf("the message does not name the call: %v", err)
	}
}

func TestChatErrorCarriesTheCallIDOfTheLoggedCall(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		ndjson(w, `{"type":"error","error":{"kind":"malformed","provider":"hive","message":"khối JSON không đọc được","call_id":"c-3","usage":{}}}`)
	})
	_, err := c.Chat(context.Background(), application.ChatRequest{User: "u", ExpectJSON: true})
	var llmErr *application.LLMError
	if !asLLMError(err, &llmErr) || llmErr.CallID != "c-3" {
		t.Fatalf("err = %v, want an LLMError carrying c-3", err)
	}
}

func TestChatErrorKeepsKindBilledUsagePartialAndDiag(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		ndjson(w, `{"type":"error","error":{"kind":"truncated","provider":"hive","message":"cut off","partial":"half","diag":"finish_reason=length","usage":{"model":"m","prompt_tokens":8,"completion_tokens":50}},"calls":[]}`)
	})
	_, err := c.Chat(context.Background(), application.ChatRequest{User: "u"})
	var llmErr *application.LLMError
	if !asLLMError(err, &llmErr) {
		t.Fatalf("err = %v, want an LLMError", err)
	}
	if llmErr.Kind != application.ErrKindTruncated || llmErr.Partial != "half" || llmErr.Usage.CompletionTokens != 50 || llmErr.Diag != "finish_reason=length" {
		t.Errorf("llm error = %+v", llmErr)
	}
}

func TestAStreamThatEndsWithoutAResultIsAnErrorNotAnEmptyAnswer(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		ndjson(w, `{"type":"progress","reasoning_chars":1,"content_chars":0}`)
	})
	_, err := c.Chat(context.Background(), application.ChatRequest{User: "u"})
	if application.LLMErrorKindOf(err) != application.ErrKindServer || !strings.Contains(err.Error(), "without a result") {
		t.Fatalf("err = %v", err)
	}
}

func TestAnUnreachableServiceIsAServerErrorAndATimeoutIsATimeout(t *testing.T) {
	dead := NewClient("http://127.0.0.1:1", time.Second)
	if _, err := dead.Chat(context.Background(), application.ChatRequest{User: "u"}); application.LLMErrorKindOf(err) != application.ErrKindServer {
		t.Fatalf("unreachable: err = %v", err)
	}

	// The server only notices the client hanging up once the request body has
	// been read, so drain it before waiting.
	slow := serve(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		<-r.Context().Done()
	})
	slow.timeout = 50 * time.Millisecond
	if _, err := slow.Chat(context.Background(), application.ChatRequest{User: "u"}); application.LLMErrorKindOf(err) != application.ErrKindTimeout {
		t.Fatalf("slow: err = %v, want timeout", err)
	}
}

func TestSuggestMetadata(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch r.URL.Path {
		case "/v1/suggest-metadata":
			if body["language"] != "vi" || body["script_content"] != "S" || body["category_hint"] != "C" || body["model"] != "ollama" {
				t.Errorf("metadata body = %v", body)
			}
			_, _ = w.Write([]byte(`{"title":"T","description":"D","tags":null,"usage":{}}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	})
	title, desc, tags, err := c.Suggest(context.Background(), "S", "C", domain.LanguageVietnamese, "ollama")
	if err != nil || title != "T" || desc != "D" {
		t.Fatalf("Suggest: %q %q %v", title, desc, err)
	}
	if tags == nil || len(tags) != 0 {
		t.Errorf("tags = %#v, want an empty non-nil slice (the API contract says array, never null)", tags)
	}
}

func TestANon200AnswerCarriesTheServicesErrorKind(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":{"kind":"balance","provider":"ollama","message":"no credit","usage":{}}}`))
	})
	_, _, _, err := c.Suggest(context.Background(), "S", "C", domain.LanguageVietnamese, "ollama")
	if application.LLMErrorKindOf(err) != application.ErrKindBalance {
		t.Fatalf("err = %v, want balance", err)
	}

	bad := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`<html>boom</html>`))
	})
	if _, _, _, err := bad.Suggest(context.Background(), "S", "C", domain.LanguageEnglish, ""); application.LLMErrorKindOf(err) != application.ErrKindServer {
		t.Fatalf("err = %v, want server", err)
	}
}

func TestReadyReflectsTheServicesHiveKeyAndIsCached(t *testing.T) {
	hits := 0
	configured := false
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		hits++
		_, _ = w.Write([]byte(`{"status":"ok","hive_configured":` + map[bool]string{true: "true", false: "false"}[configured] + `}`))
	})
	if c.Ready() {
		t.Fatal("Ready with no Hive key, want false")
	}
	configured = true
	if c.Ready() { // still inside the cache window
		t.Fatal("Ready flipped inside the cache window")
	}
	c.readyAt = time.Time{}
	if !c.Ready() {
		t.Fatal("Ready after the key appeared and the cache expired, want true")
	}
	if hits != 2 {
		t.Errorf("health hits = %d, want 2", hits)
	}
	if NewClient("http://127.0.0.1:1", time.Second).Ready() {
		t.Error("Ready with llm-service down, want false")
	}
}

func asLLMError(err error, target **application.LLMError) bool { return errors.As(err, target) }
