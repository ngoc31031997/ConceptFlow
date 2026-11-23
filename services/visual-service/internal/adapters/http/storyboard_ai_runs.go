package http

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"visual/internal/application"
)

// storyboardAIRuns backs the AI runs of step 11: one AI turn with a preview
// before it is stored.
type storyboardAIRuns interface {
	Start(ctx context.Context, projectID string, task application.AIRunTask, target, note string, channel application.AIRunChannel) (application.AIRun, error)
	Reply(ctx context.Context, projectID, id, reply string) (application.AIRun, error)
	Apply(ctx context.Context, projectID, id string) (application.AIRun, *application.StoryboardScreen, error)
	Cancel(ctx context.Context, projectID, id string) (application.AIRun, error)
	Get(ctx context.Context, projectID, id string) (application.AIRun, error)
	Latest(ctx context.Context, projectID string) ([]application.AIRun, error)
}

// WithStoryboardAIRuns adds the AI run routes of step 11; call it before Handler.
func (rt *Router) WithStoryboardAIRuns(runs storyboardAIRuns) *Router {
	rt.aiRuns = runs
	return rt
}

// mountAIRuns adds the routes under /v1/projects/{project_id}/storyboard.
func (rt *Router) mountAIRuns(r chi.Router) {
	r.Post("/ai-runs", rt.aiRunStart)
	r.Get("/ai-runs", rt.aiRunLatest)
	r.Get("/ai-runs/{rid}", rt.aiRunGet)
	r.Post("/ai-runs/{rid}/reply", rt.aiRunReply)
	r.Post("/ai-runs/{rid}/apply", rt.aiRunApply)
	r.Post("/ai-runs/{rid}/cancel", rt.aiRunCancel)
}

func (rt *Router) aiRunStart(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Task    application.AIRunTask    `json:"task"`
		Target  string                   `json:"target"`
		Note    string                   `json:"note"`
		Channel application.AIRunChannel `json:"channel"`
	}
	if err := decodeBody(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	run, err := rt.aiRuns.Start(r.Context(), pid(r), in.Task, in.Target, in.Note, in.Channel)
	if err != nil {
		writeConceptError(w, err)
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

// aiRunApply answers with the run and the step's screen once the reply is
// stored; a reply that no longer fits answers with the error, and the run
// (now unreadable) can be read back with GET.
func (rt *Router) aiRunApply(w http.ResponseWriter, r *http.Request) {
	run, st, err := rt.aiRuns.Apply(r.Context(), pid(r), chi.URLParam(r, "rid"))
	if err != nil {
		writeConceptError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"run": run, "state": st})
}

func (rt *Router) aiRunCancel(w http.ResponseWriter, r *http.Request) {
	run, err := rt.aiRuns.Cancel(r.Context(), pid(r), chi.URLParam(r, "rid"))
	writeConcept(w, run, err)
}
