package visual

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"orchestrator/internal/domain"
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

func TestProjectKit_ReadsTheKit(t *testing.T) {
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Write([]byte(`{"ref_id":"r1","kit":{"focus":"mid","grain":0.2}}`))
	}))
	defer srv.Close()
	kit, err := NewClient(srv.URL, time.Second).ProjectKit(context.Background(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	if path != "/internal/v1/visual/p1/kit" || kit["focus"] != "mid" || kit["grain"] != 0.2 {
		t.Fatalf("path %s kit %v", path, kit)
	}
}

func TestProjectKit_MapsStatuses(t *testing.T) {
	for status, want := range map[int]error{404: domain.ErrNoReferenceStyle, 409: domain.ErrReferenceStyleHasNoKit} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) }))
		_, err := NewClient(srv.URL, time.Second).ProjectKit(context.Background(), "p1")
		srv.Close()
		if !errors.Is(err, want) {
			t.Fatalf("status %d: err %v, want %v", status, err, want)
		}
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(500) }))
	defer srv.Close()
	if _, err := NewClient(srv.URL, time.Second).ProjectKit(context.Background(), "p1"); err == nil {
		t.Fatal("a 500 must surface as an error")
	}
}
