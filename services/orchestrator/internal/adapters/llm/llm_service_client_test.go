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

	"orchestrator/internal/application"
	"orchestrator/internal/domain"
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

func TestSuggestMetadataAndShortScript(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		switch r.URL.Path {
		case "/v1/suggest-metadata":
			if body["language"] != "vi" || body["script_content"] != "S" || body["category_hint"] != "C" {
				t.Errorf("metadata body = %v", body)
			}
			_, _ = w.Write([]byte(`{"title":"T","description":"D","tags":null,"usage":{}}`))
		case "/v1/suggest-short-script":
			_, _ = w.Write([]byte(`{"script":"from conceptflow import *","usage":{}}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	})
	title, desc, tags, err := c.Suggest(context.Background(), "S", "C", domain.LanguageVietnamese)
	if err != nil || title != "T" || desc != "D" {
		t.Fatalf("Suggest: %q %q %v", title, desc, err)
	}
	if tags == nil || len(tags) != 0 {
		t.Errorf("tags = %#v, want an empty non-nil slice (the API contract says array, never null)", tags)
	}
	script, err := c.SuggestShortScript(context.Background(), "topic", "", domain.LanguageEnglish)
	if err != nil || script != "from conceptflow import *" {
		t.Fatalf("SuggestShortScript: %q %v", script, err)
	}
}

func TestANon200AnswerCarriesTheServicesErrorKind(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":{"kind":"balance","provider":"ollama","message":"no credit","usage":{}}}`))
	})
	_, _, _, err := c.Suggest(context.Background(), "S", "C", domain.LanguageVietnamese)
	if application.LLMErrorKindOf(err) != application.ErrKindBalance {
		t.Fatalf("err = %v, want balance", err)
	}

	bad := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`<html>boom</html>`))
	})
	if _, err := bad.SuggestShortScript(context.Background(), "t", "", domain.LanguageEnglish); application.LLMErrorKindOf(err) != application.ErrKindServer {
		t.Fatalf("err = %v, want server", err)
	}
}

func TestFinalizeStoryboard(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["content"] != "RAW" || body["model"] != "m" {
			t.Errorf("body = %v", body)
		}
		_, _ = w.Write([]byte(`{"storyboard":"{}","shots":12,"repaired":true,"usage":{"prompt_tokens":7,"completion_tokens":9}}`))
	})
	got, err := c.FinalizeStoryboard(context.Background(), "RAW", "m", 1000)
	if err != nil || got.Storyboard != "{}" || got.Shots != 12 || !got.Repaired || got.Usage.PromptTokens != 7 {
		t.Fatalf("got %+v err %v", got, err)
	}

	bad := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":{"kind":"malformed","message":"still invalid","usage":{"prompt_tokens":4}}}`))
	})
	_, err = bad.FinalizeStoryboard(context.Background(), "RAW", "", 0)
	var llmErr *application.LLMError
	if !asLLMError(err, &llmErr) || llmErr.Kind != application.ErrKindMalformed || llmErr.Usage.PromptTokens != 4 {
		t.Fatalf("err = %v", err)
	}
}

func TestGenerateCodeStreamsEventsAndReturnsTheResult(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["engine"] != "remotion" || body["storyboard"] != "SB" || body["system"] != "SYS" {
			t.Errorf("body = %v", body)
		}
		ndjson(w,
			`{"type":"phase","phase":"layout"}`,
			`{"type":"phase","phase":"chunks","total":3}`,
			`{"type":"chunk_done","index":1,"total":3,"done":1}`,
			`{"type":"phase","phase":"repair","round":1,"total":3,"targets":["1.3"]}`,
			`{"type":"result","code":"CODE","check_ok":false,"repair_rounds":3,"scene_class_name":"XScene","warnings":["w"],`+
				`"diagnostics":[{"message":"boom","line":9},{"message":"no line","line":null}],`+
				`"calls":[{"phase":"chunk","label":"1.1-1.10","ok":true,"cached":false,"duration_ms":1500,"usage":{"model":"m","prompt_tokens":5,"completion_tokens":6}}]}`)
	})
	var events []application.CodeEvent
	res, err := c.GenerateCode(context.Background(),
		application.CodeGenRequest{Engine: "remotion", Storyboard: "SB", System: "SYS"},
		func(e application.CodeEvent) { events = append(events, e) })
	if err != nil {
		t.Fatalf("GenerateCode: %v", err)
	}
	if len(events) != 4 || events[2].Type != "chunk_done" || events[2].Done != 1 || events[3].Round != 1 || events[3].Targets[0] != "1.3" {
		t.Errorf("events = %+v", events)
	}
	if res.Code != "CODE" || res.CheckOK || res.RepairRounds != 3 || res.SceneClassName != "XScene" || len(res.Warnings) != 1 {
		t.Errorf("result = %+v", res)
	}
	if len(res.Diagnostics) != 2 || res.Diagnostics[0].Line != 9 || res.Diagnostics[1].Line != 0 {
		t.Errorf("diagnostics = %+v", res.Diagnostics)
	}
	if len(res.Calls) != 1 || res.Calls[0].Duration != 1500*time.Millisecond || res.Calls[0].Usage.PromptTokens != 5 {
		t.Errorf("calls = %+v", res.Calls)
	}
}

func TestGenerateCodeFailureKeepsTheBilledCalls(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, _ *http.Request) {
		ndjson(w, `{"type":"error","error":{"kind":"balance","provider":"hive","message":"out of credit"},`+
			`"calls":[{"phase":"layout","label":"LAYOUT","ok":true,"usage":{"model":"m","prompt_tokens":100}},{"phase":"chunk","label":"1.1-1.10","ok":false,"error_kind":"balance"}]}`)
	})
	_, err := c.GenerateCode(context.Background(), application.CodeGenRequest{Engine: "manim"}, nil)
	var llmErr *application.LLMError
	if !asLLMError(err, &llmErr) || llmErr.Kind != application.ErrKindBalance {
		t.Fatalf("err = %v", err)
	}
	if len(llmErr.Calls) != 2 || llmErr.Calls[0].Usage.PromptTokens != 100 || llmErr.Calls[1].ErrorKind != application.ErrKindBalance {
		t.Errorf("calls = %+v", llmErr.Calls)
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
