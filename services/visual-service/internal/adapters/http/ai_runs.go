package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"visual/internal/application"
)

// aiRunsDesk backs the AI runs of steps 8 to 11: one AI turn of a step, replayed
// from the step's own request, with a preview before it is stored (ADR-0052).
type aiRunsDesk interface {
	Start(ctx context.Context, projectID, label string, req application.AIRunRequest, channel application.AIRunChannel) (application.AIRun, error)
	Reply(ctx context.Context, projectID, id, reply string) (application.AIRun, error)
	Apply(ctx context.Context, projectID, id string) (application.AIRun, json.RawMessage, error)
	Cancel(ctx context.Context, projectID, id string) (application.AIRun, error)
	Get(ctx context.Context, projectID, id string) (application.AIRun, error)
	Latest(ctx context.Context, projectID string) ([]application.AIRun, error)
}

// WithAIRuns adds the AI run routes; call it before Handler.
func (rt *Router) WithAIRuns(runs aiRunsDesk) *Router {
	rt.aiRuns = runs
	return rt
}

// Replay serves one step request inside the service under ctx, which carries
// the external-AI mode, through the same routes the gateway reaches. The routing
// state of the request that started the run is dropped so the replay is routed
// afresh.
func (rt *Router) Replay(ctx context.Context, req application.AIRunRequest) (int, []byte) {
	if rt.handler == nil {
		return http.StatusServiceUnavailable, []byte(`{"error":"router not built"}`)
	}
	ctx = context.WithValue(ctx, chi.RouteCtxKey, nil)
	r, err := http.NewRequestWithContext(ctx, req.Method, req.Path, strings.NewReader(req.Body))
	if err != nil {
		return http.StatusBadRequest, []byte(`{"error":"invalid replayed request"}`)
	}
	r.Header.Set("Content-Type", "application/json")
	held := &heldResponse{header: http.Header{}, status: http.StatusOK}
	rt.handler.ServeHTTP(held, r)
	return held.status, held.body.Bytes()
}

func (rt *Router) mountAIRuns(r chi.Router) {
	r.Route("/v1/projects/{project_id}/visual/ai-runs", func(r chi.Router) {
		r.Post("/", rt.aiRunStart)
		r.Get("/", rt.aiRunLatest)
		r.Get("/{rid}", rt.aiRunGet)
		r.Post("/{rid}/reply", rt.aiRunReply)
		r.Post("/{rid}/apply", rt.aiRunApply)
		r.Post("/{rid}/cancel", rt.aiRunCancel)
	})
}

// writeAIRunError answers a refusal of the replayed step request with the
// step's own status and body, and any other error like the step routes do.
func writeAIRunError(w http.ResponseWriter, err error) {
	var refused *application.AIRunHTTPError
	if errors.As(err, &refused) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(refused.Status)
		_, _ = w.Write(refused.Body)
		return
	}
	writeConceptError(w, err)
}

func (rt *Router) aiRunStart(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Label   string                   `json:"label"`
		Method  string                   `json:"method"`
		Path    string                   `json:"path"`
		Body    string                   `json:"body"`
		Channel application.AIRunChannel `json:"channel"`
	}
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	run, err := rt.aiRuns.Start(r.Context(), pid(r), in.Label,
		application.AIRunRequest{Method: in.Method, Path: in.Path, Body: in.Body}, in.Channel)
	if err != nil {
		writeAIRunError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, run)
}

func (rt *Router) aiRunLatest(w http.ResponseWriter, r *http.Request) {
	runs, err := rt.aiRuns.Latest(r.Context(), pid(r))
	if err != nil {
		writeConceptError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"runs": runs})
}

func (rt *Router) aiRunGet(w http.ResponseWriter, r *http.Request) {
	run, err := rt.aiRuns.Get(r.Context(), pid(r), chi.URLParam(r, "rid"))
	writeConcept(w, run, err)
}

// aiRunReply takes {"raw": …}, the body the AI Bridge extension posts for a
// fixed-prompt item, and the one the paste box sends. It answers 200 with the
// run whether or not the reply could be read: the run says which.
func (rt *Router) aiRunReply(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Raw string `json:"raw"`
	}
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	run, err := rt.aiRuns.Reply(r.Context(), pid(r), chi.URLParam(r, "rid"), in.Raw)
	writeConcept(w, run, err)
}

// aiRunApply answers with the run and what the step answered once the reply is
// stored; a reply that no longer fits answers with the step's refusal, and the
// run (now unreadable) can be read back with GET.
func (rt *Router) aiRunApply(w http.ResponseWriter, r *http.Request) {
	run, state, err := rt.aiRuns.Apply(r.Context(), pid(r), chi.URLParam(r, "rid"))
	if err != nil {
		writeAIRunError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"run": run, "state": state})
}

func (rt *Router) aiRunCancel(w http.ResponseWriter, r *http.Request) {
	run, err := rt.aiRuns.Cancel(r.Context(), pid(r), chi.URLParam(r, "rid"))
	writeConcept(w, run, err)
}
