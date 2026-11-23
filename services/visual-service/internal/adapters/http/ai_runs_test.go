package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"visual/internal/application"
)

// recordingRuns answers Start by replaying the request through the router it
// was given, the way application.AIRuns does, and keeps what came back.
type recordingRuns struct {
	aiRunsDesk
	rt     *Router
	status int
	body   string
}

func (r *recordingRuns) Start(ctx context.Context, projectID, label string, req application.AIRunRequest, channel application.AIRunChannel) (application.AIRun, error) {
	status, body := r.rt.Replay(ctx, req)
	r.status, r.body = status, string(body)
	return application.AIRun{ID: "r1", ProjectID: projectID, Path: req.Path}, nil
}

func TestAIRuns_AReplayFromInsideARequestIsRoutedAfresh(t *testing.T) {
	sb := &fakeStoryboard{}
	rt := NewRouter(nil).WithStoryboard(sb)
	runs := &recordingRuns{rt: rt}
	rt.WithAIRuns(runs)
	h := rt.Handler()

	body := `{"label":"x","method":"POST","path":"/v1/projects/p1/storyboard/units/VP-002/generate","body":"{\"note\":\"gọn\"}","channel":"manual"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/projects/p1/visual/ai-runs", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("start: %d %s", rec.Code, rec.Body.String())
	}
	if runs.status != http.StatusOK || sb.last != "unit VP-002 gọn" {
		t.Fatalf("the replay reaches the unit route with its own path and body: %d %q %q", runs.status, sb.last, runs.body)
	}
}
