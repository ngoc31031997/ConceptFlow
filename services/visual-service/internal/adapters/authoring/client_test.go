package authoring

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"visual/internal/application"
	"visual/internal/domain"
)

func TestLatestScriptVersion(t *testing.T) {
	status := http.StatusOK
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/v1/authoring/test-cr112b-a/script-versions/latest" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"project_id":"test-cr112b-a","version":3,"parts":[{"key":"brief","sha":"s","content":{}}]}`))
	}))
	defer srv.Close()
	c := NewClient(srv.URL, time.Second)
	v, err := c.LatestScriptVersion(context.Background(), "test-cr112b-a")
	if err != nil || v.Version != 3 || v.Parts[0].Key != "brief" {
		t.Fatalf("%+v %v", v, err)
	}
	status = http.StatusNotFound
	if _, err := c.LatestScriptVersion(context.Background(), "test-cr112b-a"); !errors.Is(err, domain.ErrScriptNotLocked) {
		t.Errorf("404: %v", err)
	}
	status = http.StatusInternalServerError
	if _, err := c.LatestScriptVersion(context.Background(), "test-cr112b-a"); !errors.Is(err, application.ErrAuthoringFailed) {
		t.Errorf("500: %v", err)
	}
}
