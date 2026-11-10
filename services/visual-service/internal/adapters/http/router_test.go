package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeDelete struct {
	project string
	err     error
}

func (f *fakeDelete) Execute(_ context.Context, projectID string) error {
	f.project = projectID
	return f.err
}

func do(rt *Router, method, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	rt.Handler().ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func TestHealth(t *testing.T) {
	rec := do(NewRouter(&fakeDelete{}), http.MethodGet, "/health")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"ok"`) {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteProject(t *testing.T) {
	f := &fakeDelete{}
	if rec := do(NewRouter(f), http.MethodDelete, "/internal/v1/visual/p1"); rec.Code != http.StatusNoContent || f.project != "p1" {
		t.Fatalf("status %d project %q", rec.Code, f.project)
	}
	f.err = errors.New("db down")
	if rec := do(NewRouter(f), http.MethodDelete, "/internal/v1/visual/p1"); rec.Code != http.StatusInternalServerError {
		t.Fatalf("store failure: status %d, want 500", rec.Code)
	}
}
