package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"authoring/internal/application"
	"authoring/internal/domain"
)

// archetypesUseCase backs the video-archetype table (CR-041).
type archetypesUseCase interface {
	List(ctx context.Context) ([]domain.VideoArchetype, error)
	Create(ctx context.Context, a domain.VideoArchetype) (domain.VideoArchetype, error)
	Copy(ctx context.Context, id string) (domain.VideoArchetype, error)
	Update(ctx context.Context, id string, a domain.VideoArchetype) (domain.VideoArchetype, error)
	Delete(ctx context.Context, id string) error
}

// WithArchetypes enables the /v1/video-archetypes routes.
func (rt *Router) WithArchetypes(a archetypesUseCase) *Router {
	rt.archetypes = a
	return rt
}

func archetypeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, application.ErrArchetypeNotFound):
		writeError(w, http.StatusNotFound, "video archetype not found")
	case errors.Is(err, application.ErrArchetypeReadOnly):
		writeError(w, http.StatusForbidden, "system video archetypes are read-only — copy it to make one you can edit")
	case errors.Is(err, application.ErrArchetypeCodeTaken):
		writeError(w, http.StatusConflict, "code is already used by another video archetype")
	default:
		writeError(w, http.StatusBadRequest, err.Error())
	}
}

func (rt *Router) archetypesEnabled(w http.ResponseWriter) bool {
	if rt.archetypes == nil {
		writeError(w, http.StatusNotFound, "video archetypes are not enabled")
		return false
	}
	return true
}

func decodeArchetype(w http.ResponseWriter, r *http.Request) (domain.VideoArchetype, bool) {
	var req struct {
		Code      string `json:"code"`
		Name      string `json:"name"`
		WhenToUse string `json:"when_to_use"`
		Playbook  string `json:"playbook"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return domain.VideoArchetype{}, false
	}
	return domain.VideoArchetype{Code: req.Code, Name: req.Name, WhenToUse: req.WhenToUse, Playbook: req.Playbook}, true
}

func (rt *Router) handleListArchetypes(w http.ResponseWriter, r *http.Request) {
	if !rt.archetypesEnabled(w) {
		return
	}
	list, err := rt.archetypes.List(r.Context())
	if err != nil {
		archetypeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"video_archetypes": list})
}

func (rt *Router) handleCreateArchetype(w http.ResponseWriter, r *http.Request) {
	if !rt.archetypesEnabled(w) {
		return
	}
	a, ok := decodeArchetype(w, r)
	if !ok {
		return
	}
	out, err := rt.archetypes.Create(r.Context(), a)
	if err != nil {
		archetypeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (rt *Router) handleCopyArchetype(w http.ResponseWriter, r *http.Request) {
	if !rt.archetypesEnabled(w) {
		return
	}
	out, err := rt.archetypes.Copy(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		archetypeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (rt *Router) handleUpdateArchetype(w http.ResponseWriter, r *http.Request) {
	if !rt.archetypesEnabled(w) {
		return
	}
	a, ok := decodeArchetype(w, r)
	if !ok {
		return
	}
	out, err := rt.archetypes.Update(r.Context(), chi.URLParam(r, "id"), a)
	if err != nil {
		archetypeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (rt *Router) handleDeleteArchetype(w http.ResponseWriter, r *http.Request) {
	if !rt.archetypesEnabled(w) {
		return
	}
	if err := rt.archetypes.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		archetypeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
