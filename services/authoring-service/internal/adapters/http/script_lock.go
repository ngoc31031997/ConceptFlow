package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"

	"authoring/internal/application"
	"authoring/internal/domain"
)

// Operation kinds the lock step's AI runs are recorded under, so that any page can follow them.
const (
	operationScriptReview  = "script_review"
	operationScriptHandoff = "handoff_beat"
)

// scriptLockUseCase is the lock step beyond the generic unit API.
type scriptLockUseCase interface {
	State(ctx context.Context, projectID string) (application.ScriptLockState, error)
	Review(ctx context.Context, projectID string) error
	ApplyFix(ctx context.Context, projectID, findingID string) error
	AddToHandoff(ctx context.Context, projectID, findingID string) error
	Accept(ctx context.Context, projectID, findingID, why string) error
	AddNote(ctx context.Context, projectID, text string) error
	AddTerm(ctx context.Context, projectID string, t domain.GlossaryTerm) error
	GenerateHandoff(ctx context.Context, projectID, beatKey string) error
	ConfirmHandoff(ctx context.Context, projectID, beatKey string) error
	Lock(ctx context.Context, projectID, note string) (application.VersionView, error)
	HandoffForRole(ctx context.Context, projectID, role string) (domain.HandoffPackage, error)
}

// WithScriptLock enables the /script-lock routes and the internal handoff route.
func (rt *Router) WithScriptLock(uc scriptLockUseCase) *Router {
	rt.scriptLock = uc
	return rt
}

func (rt *Router) scriptLockRoutes(r chi.Router) {
	const base = "/v1/projects/{project_id}/script-lock"
	r.Get(base, rt.handleGetScriptLock)
	r.Post(base+"/review", rt.handleScriptLockReview)
	r.Post(base+"/findings/{finding_id}/apply", rt.findingAction(func(uc scriptLockUseCase, ctx context.Context, p, id string, _ string) error {
		return uc.ApplyFix(ctx, p, id)
	}))
	r.Post(base+"/findings/{finding_id}/handoff", rt.findingAction(func(uc scriptLockUseCase, ctx context.Context, p, id string, _ string) error {
		return uc.AddToHandoff(ctx, p, id)
	}))
	r.Post(base+"/findings/{finding_id}/accept", rt.findingAction(func(uc scriptLockUseCase, ctx context.Context, p, id string, why string) error {
		return uc.Accept(ctx, p, id, why)
	}))
	r.Post(base+"/notes", rt.handleScriptLockNote)
	r.Post(base+"/glossary", rt.handleScriptLockTerm)
	r.Post(base+"/handoff/generate", rt.handleScriptLockHandoff)
	r.Post(base+"/handoff/{key}/confirm", rt.handleScriptLockConfirm)
	r.Post(base+"/lock", rt.handleScriptLockLock)
	r.Get("/internal/v1/authoring/{project_id}/handoff", rt.handleHandoffForRole)
}

func (rt *Router) scriptLockEnabled(w http.ResponseWriter) bool {
	if rt.scriptLock == nil {
		writeError(w, http.StatusNotFound, "script lock is not enabled")
		return false
	}
	return true
}

func (rt *Router) writeScriptLockState(w http.ResponseWriter, r *http.Request) {
	state, err := rt.scriptLock.State(r.Context(), chi.URLParam(r, "project_id"))
	if err != nil {
		writeUnitError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (rt *Router) handleGetScriptLock(w http.ResponseWriter, r *http.Request) {
	if rt.scriptLockEnabled(w) {
		rt.writeScriptLockState(w, r)
	}
}

func (rt *Router) handleScriptLockReview(w http.ResponseWriter, r *http.Request) {
	if !rt.scriptLockEnabled(w) {
		return
	}
	ctx, finish := rt.trackOperation(r, operationScriptReview)
	err := rt.scriptLock.Review(ctx, chi.URLParam(r, "project_id"))
	finish(err)
	if err != nil {
		writeUnitRunError(w, err)
		return
	}
	rt.writeScriptLockState(w, r)
}

// findingAction builds the handler of an action on one finding; the optional
// body {"why": "..."} is passed along as the reason.
func (rt *Router) findingAction(do func(uc scriptLockUseCase, ctx context.Context, projectID, findingID, why string) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !rt.scriptLockEnabled(w) {
			return
		}
		var body struct {
			Why string `json:"why"`
		}
		if r.ContentLength != 0 {
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				writeError(w, http.StatusBadRequest, "invalid request body")
				return
			}
		}
		// chi hands back path parameters as they were sent; a machine finding's id has a colon, which the web app escapes.
		id, err := url.PathUnescape(chi.URLParam(r, "finding_id"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid finding id")
			return
		}
		if err := do(rt.scriptLock, r.Context(), chi.URLParam(r, "project_id"), id, body.Why); err != nil {
			writeUnitError(w, err)
			return
		}
		rt.writeScriptLockState(w, r)
	}
}

func (rt *Router) handleScriptLockNote(w http.ResponseWriter, r *http.Request) {
	if !rt.scriptLockEnabled(w) {
		return
	}
	var body struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := rt.scriptLock.AddNote(r.Context(), chi.URLParam(r, "project_id"), body.Text); err != nil {
		writeUnitError(w, err)
		return
	}
	rt.writeScriptLockState(w, r)
}

func (rt *Router) handleScriptLockTerm(w http.ResponseWriter, r *http.Request) {
	if !rt.scriptLockEnabled(w) {
		return
	}
	var body struct {
		Term  string `json:"term"`
		Say   string `json:"say"`
		Draw  string `json:"draw"`
		Avoid string `json:"avoid"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	t := domain.GlossaryTerm{Term: body.Term, Say: body.Say, Draw: body.Draw, Avoid: body.Avoid}
	if err := rt.scriptLock.AddTerm(r.Context(), chi.URLParam(r, "project_id"), t); err != nil {
		writeUnitError(w, err)
		return
	}
	rt.writeScriptLockState(w, r)
}

func (rt *Router) handleScriptLockHandoff(w http.ResponseWriter, r *http.Request) {
	if !rt.scriptLockEnabled(w) {
		return
	}
	var body struct {
		Beat string `json:"beat"`
	}
	if r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}
	ctx, finish := rt.trackOperation(r, operationScriptHandoff)
	err := rt.scriptLock.GenerateHandoff(ctx, chi.URLParam(r, "project_id"), body.Beat)
	finish(err)
	if err != nil {
		writeUnitRunError(w, err)
		return
	}
	rt.writeScriptLockState(w, r)
}

func (rt *Router) handleScriptLockConfirm(w http.ResponseWriter, r *http.Request) {
	if !rt.scriptLockEnabled(w) {
		return
	}
	if err := rt.scriptLock.ConfirmHandoff(r.Context(), chi.URLParam(r, "project_id"), chi.URLParam(r, "key")); err != nil {
		writeUnitError(w, err)
		return
	}
	rt.writeScriptLockState(w, r)
}

func (rt *Router) handleScriptLockLock(w http.ResponseWriter, r *http.Request) {
	if !rt.scriptLockEnabled(w) {
		return
	}
	var body struct {
		Note string `json:"note"`
	}
	if r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}
	if _, err := rt.scriptLock.Lock(r.Context(), chi.URLParam(r, "project_id"), body.Note); err != nil {
		writeUnitError(w, err)
		return
	}
	rt.writeScriptLockState(w, r)
}

// handleHandoffForRole is what a later step calls to read the part of the
// latest locked version that its role reads.
func (rt *Router) handleHandoffForRole(w http.ResponseWriter, r *http.Request) {
	if !rt.scriptLockEnabled(w) {
		return
	}
	pkg, err := rt.scriptLock.HandoffForRole(r.Context(), chi.URLParam(r, "project_id"), r.URL.Query().Get("role"))
	if err != nil {
		writeUnitError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, pkg)
}
