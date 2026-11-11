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

	"visual/internal/application"
	"visual/internal/domain"
)

func TestChat_ReadsTheStreamToItsResult(t *testing.T) {
	var got map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &got)
		_, _ = w.Write([]byte(`{"type":"progress","content_chars":3}` + "\n" + `{"type":"result","content":"x","json":{"a":1}}` + "\n"))
	}))
	defer srv.Close()
	res, err := NewClient(srv.URL, 0).Chat(context.Background(), application.ChatRequest{System: "s", User: "u", Model: "m", Role: "visual_concept_typography", Step: "visual_concept", ProjectID: "test-cr112b-l", ExpectJSON: true})
	if err != nil || string(res.JSON) != `{"a":1}` {
		t.Fatalf("%s %v", res.JSON, err)
	}
	if got["role"] != "visual_concept_typography" || got["step"] != "visual_concept" || got["expect_json"] != true || got["model"] != "m" {
		t.Errorf("request %v", got)
	}
}

func TestChat_Errors(t *testing.T) {
	cases := []struct {
		body string
		want error
	}{
		{`{"type":"error","error":{"kind":"malformed","message":"no json"}}` + "\n", domain.ErrUnreadableReply},
		{`{"type":"error","error":{"kind":"timeout","message":"slow"}}` + "\n", application.ErrLLMFailed},
		{`{"type":"progress"}` + "\n", application.ErrLLMFailed},
	}
	for _, c := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(c.body)) }))
		_, err := NewClient(srv.URL, 0).Chat(context.Background(), application.ChatRequest{ExpectJSON: true})
		srv.Close()
		if !errors.Is(err, c.want) {
			t.Errorf("%s: %v", c.body, err)
		}
	}
}

func TestLibrary(t *testing.T) {
	var pushed map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/internal/v1/prompts/visual_concept_review":
			_, _ = w.Write([]byte(`{"template_text":"T"}`))
		case r.URL.Path == "/internal/v1/prompts/system":
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &pushed)
			_, _ = w.Write([]byte(`{}`))
		case r.URL.Path == "/internal/v1/step-models":
			_, _ = w.Write([]byte(`{"step_models":{"visual_concept":"ollama"}}`))
		case r.URL.Path == "/internal/v1/models":
			_, _ = w.Write([]byte(`{"models":[{"id":"ollama","label":"O","local":true}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := NewClient(srv.URL, 0)
	ctx := context.Background()
	if tpl, err := c.ActiveTemplate(ctx, "visual_concept_review"); err != nil || tpl != "T" {
		t.Errorf("%q %v", tpl, err)
	}
	if _, err := c.ActiveTemplate(ctx, "nope"); !errors.Is(err, application.ErrPromptNotFound) {
		t.Errorf("%v", err)
	}
	if err := c.PushSystemPrompts(ctx, domain.PromptOwner, domain.SystemPrompts()); err != nil || pushed["owner"] != "visual" || len(pushed["prompts"].([]any)) != 12 {
		t.Errorf("push %v %v", pushed["owner"], err)
	}
	if m, err := c.StepDefaultModel(ctx, "visual_concept"); err != nil || m != "ollama" {
		t.Errorf("%q %v", m, err)
	}
	if _, err := c.StepDefaultModel(ctx, "other"); !errors.Is(err, application.ErrLLMFailed) {
		t.Error("a step llm-service does not know is an error")
	}
	if ms, err := c.Models(ctx); err != nil || len(ms) != 1 || !ms[0].Local {
		t.Errorf("%v %v", ms, err)
	}
}

func TestChatExternal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/v1/chat/external") {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":{"kind":"malformed","provider":"external","message":"no json"}}`))
	}))
	defer srv.Close()
	if _, err := NewClient(srv.URL, 0).ChatExternal(context.Background(), application.ChatRequest{ExpectJSON: true}, "x"); !errors.Is(err, domain.ErrUnreadableReply) {
		t.Errorf("%v", err)
	}
}
