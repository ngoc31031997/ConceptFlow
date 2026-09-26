package authoring

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"orchestrator/internal/application"
	"orchestrator/internal/domain"
)

func TestClient_TopicSimilarSummariesAndDelete(t *testing.T) {
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Method+" "+r.URL.RequestURI())
		switch {
		case r.Method == http.MethodPut:
			var b map[string]any
			_ = json.NewDecoder(r.Body).Decode(&b)
			if b["topic"] != "t" || b["language"] != "vi" {
				t.Errorf("unexpected put body %v", b)
			}
			w.WriteHeader(http.StatusNoContent)
		case r.URL.Path == "/internal/v1/authoring/similar":
			_, _ = w.Write([]byte(`[{"project_id":"p2","topic":"t","created_at":"2026-09-01T00:00:00Z"}]`))
		case r.URL.Path == "/internal/v1/authoring/summaries":
			_, _ = w.Write([]byte(`{"p1":{"topic":"t","story":true,"storyboard":false,"code":false}}`))
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := NewClient(srv.URL, time.Second)
	ctx := context.Background()

	if err := c.SaveAuthoringTopic(ctx, "p1", "t", domain.LanguageVietnamese); err != nil {
		t.Fatal(err)
	}
	sim, err := c.FindSimilarTopics(ctx, domain.LanguageVietnamese, "t", "p1")
	if err != nil || len(sim) != 1 || sim[0].ProjectID != "p2" {
		t.Fatalf("similar = %+v err=%v", sim, err)
	}
	sums, err := c.Summaries(ctx, []string{"p1"})
	if err != nil || sums["p1"] != (application.AuthoringSummary{Topic: "t", Story: true}) {
		t.Fatalf("summaries = %+v err=%v", sums, err)
	}
	if err := c.DeleteAuthoring(ctx, "p1"); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 4 {
		t.Fatalf("expected 4 calls, got %v", seen)
	}
}

func TestClient_ErrorStatusIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(500) }))
	defer srv.Close()
	if _, err := NewClient(srv.URL, time.Second).Summaries(context.Background(), []string{"p1"}); err == nil {
		t.Fatal("a 500 must surface as an error")
	}
}
