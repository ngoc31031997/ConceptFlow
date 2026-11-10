package visual

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDeleteVisual_SendsTheProjectsDelete(t *testing.T) {
	var method, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	if err := NewClient(srv.URL, time.Second).DeleteVisual(context.Background(), "p1"); err != nil {
		t.Fatal(err)
	}
	if method != http.MethodDelete || path != "/internal/v1/visual/p1" {
		t.Fatalf("got %s %s", method, path)
	}
}

func TestDeleteVisual_ErrorStatusIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(500) }))
	defer srv.Close()
	if err := NewClient(srv.URL, time.Second).DeleteVisual(context.Background(), "p1"); err == nil {
		t.Fatal("a 500 must surface as an error")
	}
}
