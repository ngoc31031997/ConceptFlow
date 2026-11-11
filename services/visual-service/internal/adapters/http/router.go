// Package http is visual-service's HTTP adapter.
package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"visual/internal/application"
)

// deleteProjectUseCase backs DELETE /internal/v1/visual/{project_id}.
type deleteProjectUseCase interface {
	Execute(ctx context.Context, projectID string) error
}

// Router builds visual-service's routes.
type Router struct {
	deleteProject    deleteProjectUseCase
	concept          conceptDesk
	plan             planDesk
	referenceLibrary referenceLibraryUseCase
}

// NewRouter returns the router.
func NewRouter(deleteProject deleteProjectUseCase) *Router {
	return &Router{deleteProject: deleteProject}
}

// Handler returns the chi router with every route.
func (rt *Router) Handler() http.Handler {
	r := chi.NewRouter()
	r.Get("/health", rt.handleHealth)
	r.Delete("/internal/v1/visual/{project_id}", rt.handleDeleteProject)
	if rt.concept != nil {
		rt.mountConcept(r)
	}
	if rt.plan != nil {
		rt.mountPlan(r)
	}
	rt.mountReferenceLibrary(r)
	return r
}

func (rt *Router) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleDeleteProject removes a deleted video's data; the orchestrator calls it
// at the end of the delete saga.
func (rt *Router) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	err := rt.deleteProject.Execute(r.Context(), chi.URLParam(r, "project_id"))
	switch {
	case errors.Is(err, application.ErrNoProjectID):
		writeError(w, http.StatusBadRequest, err.Error())
	case err != nil:
		writeError(w, http.StatusInternalServerError, "internal error")
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
