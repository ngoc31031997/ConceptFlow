package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"authoring/internal/domain"
)

// projectIllustrationsUseCase backs a video's drawing list (CR-044).
type projectIllustrationsUseCase interface {
	List(ctx context.Context, projectID string) ([]domain.ProjectIllustration, error)
	Plan(ctx context.Context, projectID, model string) ([]domain.ProjectIllustration, error)
	Draw(ctx context.Context, projectID, rowID, model string) (domain.ProjectIllustration, error)
	SetSkipped(ctx context.Context, projectID, rowID string, skipped bool) (domain.ProjectIllustration, error)
}

// WithProjectIllustrations enables /v1/projects/{id}/illustrations.
func (rt *Router) WithProjectIllustrations(u projectIllustrationsUseCase) *Router {
	rt.projectIllustrations = u
	return rt
}

func (rt *Router) projectIllustrationRoutes(r chi.Router) {
	r.Get("/v1/projects/{project_id}/illustrations", rt.handleListProjectIllustrations)
	r.Post("/v1/projects/{project_id}/illustrations/plan", rt.handlePlanProjectIllustrations)
	r.Post("/v1/projects/{project_id}/illustrations/{row_id}/draw", rt.handleDrawProjectIllustration)
	r.Post("/v1/projects/{project_id}/illustrations/{row_id}/skip", rt.handleSkipProjectIllustration)
}

func (rt *Router) projectIllustrationsEnabled(w http.ResponseWriter) bool {
	if rt.projectIllustrations == nil {
		writeError(w, http.StatusNotFound, "video illustrations are not enabled")
		return false
	}
	return true
}

func writeRows(w http.ResponseWriter, rows []domain.ProjectIllustration) {
	ready := true
	for _, r := range rows {
		if !r.Ready() {
			ready = false
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"illustrations": rows, "ready": ready})
}

func (rt *Router) handleListProjectIllustrations(w http.ResponseWriter, r *http.Request) {
	if !rt.projectIllustrationsEnabled(w) {
		return
	}
	rows, err := rt.projectIllustrations.List(r.Context(), chi.URLParam(r, "project_id"))
	if err != nil {
		illustrationError(w, err)
		return
	}
	writeRows(w, rows)
}

func (rt *Router) handlePlanProjectIllustrations(w http.ResponseWriter, r *http.Request) {
	if !rt.projectIllustrationsEnabled(w) {
		return
	}
	var b struct {
		Model string `json:"model"`
	}
	_ = json.NewDecoder(r.Body).Decode(&b) // body is optional
	rows, err := rt.projectIllustrations.Plan(r.Context(), chi.URLParam(r, "project_id"), b.Model)
	if err != nil {
		illustrationError(w, err)
		return
	}
	writeRows(w, rows)
}

func (rt *Router) handleDrawProjectIllustration(w http.ResponseWriter, r *http.Request) {
	if !rt.projectIllustrationsEnabled(w) {
		return
	}
	var b struct {
		Model string `json:"model"`
	}
	_ = json.NewDecoder(r.Body).Decode(&b)
	row, err := rt.projectIllustrations.Draw(r.Context(), chi.URLParam(r, "project_id"), chi.URLParam(r, "row_id"), b.Model)
	if err != nil {
		illustrationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}

func (rt *Router) handleSkipProjectIllustration(w http.ResponseWriter, r *http.Request) {
	if !rt.projectIllustrationsEnabled(w) {
		return
	}
	var b struct {
		Skipped bool `json:"skipped"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	row, err := rt.projectIllustrations.SetSkipped(r.Context(), chi.URLParam(r, "project_id"), chi.URLParam(r, "row_id"), b.Skipped)
	if err != nil {
		illustrationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, row)
}
