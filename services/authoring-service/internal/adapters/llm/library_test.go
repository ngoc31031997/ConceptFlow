package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"authoring/internal/application"
	"authoring/internal/domain"
)

func TestGetActiveReadsTheInternalRoute(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/internal/v1/prompts/beat_sheet" {
			t.Errorf("%s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(domain.Prompt{ID: "system-beat_sheet", Role: domain.RoleBeatSheet, TemplateText: "T", IsActive: true})
	})
	p, err := c.GetActive(context.Background(), domain.RoleBeatSheet)
	if err != nil || p.TemplateText != "T" || p.ID != "system-beat_sheet" {
		t.Fatalf("got %+v, %v", p, err)
	}
}

func TestGetActiveSaysWhenThereIsNoPrompt(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) })
	if _, err := c.GetActive(context.Background(), domain.RoleBeatSheet); !errors.Is(err, application.ErrPromptNotFound) {
		t.Fatalf("err = %v, want ErrPromptNotFound", err)
	}
}

// When llm-service does not answer there is no prompt, and the error names llm-service.
func TestGetActiveFailsClearlyWhenLLMServiceIsDown(t *testing.T) {
	srv := serve(t, func(http.ResponseWriter, *http.Request) {})
	srv.baseURL = "http://127.0.0.1:1"
	_, err := srv.GetActive(context.Background(), domain.RoleBeatSheet)
	if err == nil || !strings.Contains(err.Error(), "llm-service") {
		t.Fatalf("err = %v, want one that names llm-service", err)
	}
}

func TestPushSystemPromptsSendsTheOwnerAndTheWholeSet(t *testing.T) {
	var got struct {
		Owner   string `json:"owner"`
		Prompts []struct {
			Role         string `json:"role"`
			Name         string `json:"name"`
			TemplateText string `json:"template_text"`
		} `json:"prompts"`
	}
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/internal/v1/prompts/system" {
			t.Errorf("%s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"owner":"authoring","written":2,"removed":0}`))
	})
	err := c.PushSystemPrompts(context.Background(), "authoring", []domain.Prompt{
		{Role: domain.RoleIdea, Name: "Mặc định", TemplateText: "A"},
		{Role: domain.RoleBeatSheet, Name: "Mặc định", TemplateText: "B"},
	})
	if err != nil || got.Owner != "authoring" || len(got.Prompts) != 2 || got.Prompts[1].TemplateText != "B" || got.Prompts[0].Role != string(domain.RoleIdea) {
		t.Fatalf("err %v, body %+v", err, got)
	}
}

func TestPushSystemPromptsReportsARefusal(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":"role 'idea' is owned by 'visual'"}`))
	})
	err := c.PushSystemPrompts(context.Background(), "authoring", nil)
	if err == nil || !strings.Contains(err.Error(), "owned by") {
		t.Fatalf("err = %v", err)
	}
}

func TestModelCatalogAndStepDefaults(t *testing.T) {
	c := serve(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/internal/v1/models":
			_, _ = w.Write([]byte(`{"models":[{"id":"ollama","label":"Ollama","local":true}]}`))
		case "/internal/v1/step-models":
			_, _ = w.Write([]byte(`{"step_models":{"brief":"ollama"},"steps":["brief"]}`))
		default:
			t.Errorf("path %s", r.URL.Path)
		}
	})
	models, err := c.ModelCatalog(context.Background())
	if err != nil || len(models) != 1 || !models[0].Local {
		t.Fatalf("catalog %+v, %v", models, err)
	}
	steps, err := c.GetStepDefaultModels(context.Background())
	if err != nil || steps["brief"] != "ollama" {
		t.Fatalf("steps %+v, %v", steps, err)
	}
}
