package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"authoring/internal/application"
	"authoring/internal/domain"
)

func TestClient_GetDecodesProjectAndMapsNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/internal/v1/projects/p1":
			_ = json.NewEncoder(w).Encode(domain.Project{ProjectID: "p1", VoiceID: "v", ScriptContent: "s", Status: domain.StatusDraft})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c := NewClient(srv.URL, time.Second)

	p, err := c.Get(context.Background(), "p1")
	if err != nil || p.ProjectID != "p1" || p.VoiceID != "v" || p.ScriptContent != "s" {
		t.Fatalf("unexpected project %+v err=%v", p, err)
	}
	if _, err := c.Get(context.Background(), "nope"); !errors.Is(err, domain.ErrProjectNotFound) {
		t.Fatalf("a 404 must become ErrProjectNotFound, got %v", err)
	}
}

func TestClient_StatusFormatAndCalibrationPaths(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.RequestURI())
		switch r.URL.Path {
		case "/internal/v1/projects/p1/status":
			_, _ = w.Write([]byte(`{"status":"draft"}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer srv.Close()
	c := NewClient(srv.URL, time.Second)

	status, err := c.GetStatus(context.Background(), "p1")
	if err != nil || status != domain.StatusDraft {
		t.Fatalf("status = %q err=%v", status, err)
	}
	_, _ = c.GetVideoFormat(context.Background(), "essay", 3)
	_, _ = c.GetVoiceCalibration(context.Background(), "vi-VN-HoaiMy")
	want := []string{"/internal/v1/projects/p1/status", "/internal/v1/formats/essay?version=3", "/internal/v1/voices/vi-VN-HoaiMy/calibration"}
	for i, w := range want {
		if paths[i] != w {
			t.Errorf("request %d = %q, want %q", i, paths[i], w)
		}
	}
}

func TestClient_AppendsErrorsAndEvents(t *testing.T) {
	got := map[string]string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b map[string]any
		_ = json.NewDecoder(r.Body).Decode(&b)
		got[r.Method+" "+r.URL.Path] = b["message"].(string)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	c := NewClient(srv.URL, time.Second)

	if err := c.AppendProjectError(context.Background(), "p1", application.ProjectError{Message: "boom"}); err != nil {
		t.Fatal(err)
	}
	if got["POST /internal/v1/projects/p1/errors"] != "boom" {
		t.Fatalf("error not posted: %v", got)
	}
}
