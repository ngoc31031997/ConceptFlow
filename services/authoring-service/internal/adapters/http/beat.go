package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"authoring/internal/application"
)

// Operation kinds a beat sheet AI run is recorded under, so that any page can follow it.
const (
	operationBeatSheet = "beat_sheet"
	operationBeat      = "beat"
)

// beatUseCase is the beat sheet step.
type beatUseCase interface {
	State(ctx context.Context, projectID string) (application.BeatState, error)
	BuildSkeleton(ctx context.Context, projectID string) error
	ApproveSkeleton(ctx context.Context, projectID string) error
	Generate(ctx context.Context, projectID, note string, onlyUnwritten bool) error
	Add(ctx context.Context, projectID, after string, blank bool, note string) (string, error)
	Reorder(ctx context.Context, projectID string, order []string) error
	Delete(ctx context.Context, projectID, key string) error
	Edit(ctx context.Context, projectID, key string, fields map[string]string) error
	Redo(ctx context.Context, projectID, key, note string) error
	Approve(ctx context.Context, projectID, key string) error
	Unapprove(ctx context.Context, projectID, key string) error
	Impact(ctx context.Context, projectID, key string) ([]string, error)
}

// WithBeats enables the /beats routes.
func (rt *Router) WithBeats(uc beatUseCase) *Router {
	rt.beats = uc
	return rt
}

func (rt *Router) beatRoutes(r chi.Router) {
	const base = "/v1/projects/{project_id}/beats"
	r.Get(base, rt.handleGetBeats)
	r.Post(base+"/skeleton", rt.handleBuildBeatSkeleton)
	r.Post(base+"/skeleton/approve", rt.handleApproveBeatSkeleton)
	r.Post(base+"/generate", rt.handleGenerateBeats)
	r.Post(base, rt.handleAddBeat)
	r.Put(base+"/order", rt.handleReorderBeats)
	r.Delete(base+"/{key}", rt.handleDeleteBeat)
	r.Put(base+"/{key}", rt.handleEditBeat)
	r.Post(base+"/{key}/redo", rt.handleRedoBeat)
	r.Post(base+"/{key}/approve", rt.handleApproveBeat)
	r.Post(base+"/{key}/unapprove", rt.handleUnapproveBeat)
	r.Get(base+"/{key}/impact", rt.handleBeatImpact)
}

// beatsEnabled answers 404 when the beat sheet step is not wired.
func (rt *Router) beatsEnabled(w http.ResponseWriter) bool {
	if rt.beats == nil {
		writeError(w, http.StatusNotFound, "beats is not enabled")
		return false
	}
	return true
}

// writeBeatState answers with the whole screen.
func (rt *Router) writeBeatState(w http.ResponseWriter, r *http.Request) {
	state, err := rt.beats.State(r.Context(), chi.URLParam(r, "project_id"))
	if err != nil {
		writeUnitError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (rt *Router) handleGetBeats(w http.ResponseWriter, r *http.Request) {
	if rt.beatsEnabled(w) {
		rt.writeBeatState(w, r)
	}
}

func (rt *Router) handleBuildBeatSkeleton(w http.ResponseWriter, r *http.Request) {
	if !rt.beatsEnabled(w) {
		return
	}
	if err := rt.beats.BuildSkeleton(r.Context(), chi.URLParam(r, "project_id")); err != nil {
		writeUnitError(w, err)
		return
	}
	rt.writeBeatState(w, r)
}

func (rt *Router) handleApproveBeatSkeleton(w http.ResponseWriter, r *http.Request) {
	if !rt.beatsEnabled(w) {
		return
	}
	if err := rt.beats.ApproveSkeleton(r.Context(), chi.URLParam(r, "project_id")); err != nil {
		writeUnitError(w, err)
		return
	}
	rt.writeBeatState(w, r)
}

func (rt *Router) handleGenerateBeats(w http.ResponseWriter, r *http.Request) {
	if !rt.beatsEnabled(w) {
		return
	}
	var body struct {
		Note          string `json:"note"`
		OnlyUnwritten bool   `json:"only_unwritten"`
	}
	if r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}
	ctx, finish := rt.trackOperation(r, operationBeatSheet)
	err := rt.beats.Generate(ctx, chi.URLParam(r, "project_id"), body.Note, body.OnlyUnwritten)
	finish(err)
	if err != nil {
		writeUnitRunError(w, err)
		return
	}
	rt.writeBeatState(w, r)
}

func (rt *Router) handleAddBeat(w http.ResponseWriter, r *http.Request) {
	if !rt.beatsEnabled(w) {
		return
	}
	var body struct {
		After string `json:"after"`
		Blank bool   `json:"blank"`
		Note  string `json:"note"`
	}
	if r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}
	ctx, finish := rt.trackOperation(r, operationBeat)
	_, err := rt.beats.Add(ctx, chi.URLParam(r, "project_id"), body.After, body.Blank, body.Note)
	finish(err)
	if err != nil {
		writeUnitRunError(w, err)
		return
	}
	rt.writeBeatState(w, r)
}

func (rt *Router) handleReorderBeats(w http.ResponseWriter, r *http.Request) {
	if !rt.beatsEnabled(w) {
		return
	}
	var body struct {
		Order []string `json:"order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := rt.beats.Reorder(r.Context(), chi.URLParam(r, "project_id"), body.Order); err != nil {
		writeUnitError(w, err)
		return
	}
	rt.writeBeatState(w, r)
}

func (rt *Router) handleDeleteBeat(w http.ResponseWriter, r *http.Request) {
	if !rt.beatsEnabled(w) {
		return
	}
	if err := rt.beats.Delete(r.Context(), chi.URLParam(r, "project_id"), chi.URLParam(r, "key")); err != nil {
		writeUnitError(w, err)
		return
	}
	rt.writeBeatState(w, r)
}

func (rt *Router) handleEditBeat(w http.ResponseWriter, r *http.Request) {
	if !rt.beatsEnabled(w) {
		return
	}
	var body struct {
		Fields map[string]string `json:"fields"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := rt.beats.Edit(r.Context(), chi.URLParam(r, "project_id"), chi.URLParam(r, "key"), body.Fields); err != nil {
		writeUnitError(w, err)
		return
	}
	rt.writeBeatState(w, r)
}

func (rt *Router) handleRedoBeat(w http.ResponseWriter, r *http.Request) {
	if !rt.beatsEnabled(w) {
		return
	}
	note, err := readNote(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	ctx, finish := rt.trackOperation(r, operationBeat)
	err = rt.beats.Redo(ctx, chi.URLParam(r, "project_id"), chi.URLParam(r, "key"), note)
	finish(err)
	if err != nil {
		writeUnitRunError(w, err)
		return
	}
	rt.writeBeatState(w, r)
}

func (rt *Router) handleApproveBeat(w http.ResponseWriter, r *http.Request) {
	rt.handleBeatStatus(w, r, rt.beatsApprove)
}

func (rt *Router) handleUnapproveBeat(w http.ResponseWriter, r *http.Request) {
	rt.handleBeatStatus(w, r, rt.beatsUnapprove)
}

func (rt *Router) beatsApprove(r *http.Request) error {
	return rt.beats.Approve(r.Context(), chi.URLParam(r, "project_id"), chi.URLParam(r, "key"))
}

func (rt *Router) beatsUnapprove(r *http.Request) error {
	return rt.beats.Unapprove(r.Context(), chi.URLParam(r, "project_id"), chi.URLParam(r, "key"))
}

func (rt *Router) handleBeatStatus(w http.ResponseWriter, r *http.Request, change func(*http.Request) error) {
	if !rt.beatsEnabled(w) {
		return
	}
	if err := change(r); err != nil {
		writeUnitError(w, err)
		return
	}
	rt.writeBeatState(w, r)
}

func (rt *Router) handleBeatImpact(w http.ResponseWriter, r *http.Request) {
	if !rt.beatsEnabled(w) {
		return
	}
	items, err := rt.beats.Impact(r.Context(), chi.URLParam(r, "project_id"), chi.URLParam(r, "key"))
	if err != nil {
		writeUnitError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, unitImpactResponse{Items: items})
}
