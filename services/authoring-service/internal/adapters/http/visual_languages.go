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

// visualLanguagesUseCase backs the visual-language store.
type visualLanguagesUseCase interface {
	List(ctx context.Context) ([]domain.VisualLanguage, error)
	Get(ctx context.Context, id string) (domain.VisualLanguage, error)
	Activate(ctx context.Context, id string) error
	Copy(ctx context.Context, id string) (domain.VisualLanguage, error)
	Update(ctx context.Context, id string, in application.VisualLanguageInput) (domain.VisualLanguage, error)
	Delete(ctx context.Context, id string) error
	Pinned(ctx context.Context, projectID string) (domain.PinnedLanguage, error)
	Samples(ctx context.Context, id string) ([]domain.LanguageSample, error)
	RebuildSamples(ctx context.Context, id string) ([]domain.LanguageSample, error)
}

// WithVisualLanguages enables the /v1/visual-languages routes.
func (rt *Router) WithVisualLanguages(u visualLanguagesUseCase) *Router {
	rt.visualLanguages = u
	return rt
}

func (rt *Router) visualLanguageRoutes(r chi.Router) {
	r.Get("/v1/visual-languages", rt.handleListVisualLanguages)
	r.Get("/v1/visual-languages/{id}", rt.handleGetVisualLanguage)
	r.Get("/v1/visual-languages/{id}/samples", rt.handleVisualLanguageSamples)
	r.Get("/v1/projects/{project_id}/visual-language", rt.handleProjectVisualLanguage)
	r.Post("/v1/admin/visual-languages/{id}/activate", rt.handleActivateVisualLanguage)
	r.Post("/v1/admin/visual-languages/{id}/copy", rt.handleCopyVisualLanguage)
	r.Put("/v1/admin/visual-languages/{id}", rt.handleUpdateVisualLanguage)
	r.Delete("/v1/admin/visual-languages/{id}", rt.handleDeleteVisualLanguage)
	r.Post("/v1/admin/visual-languages/{id}/samples", rt.handleRebuildVisualLanguageSamples)
}

func (rt *Router) visualLanguagesEnabled(w http.ResponseWriter) bool {
	if rt.visualLanguages == nil {
		writeError(w, http.StatusNotFound, "visual languages are not enabled")
		return false
	}
	return true
}

func visualLanguageError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrVisualLanguageNotFound), errors.Is(err, domain.ErrProjectNotPinned):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrVisualLanguageReadOnly):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, domain.ErrVisualLanguageActive), errors.Is(err, domain.ErrVisualLanguageInUse),
		errors.Is(err, domain.ErrVisualLanguageNameTaken):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrInvalidVisualLanguage), errors.Is(err, domain.ErrUnknownKitKey):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusBadGateway, err.Error())
	}
}

func (rt *Router) handleListVisualLanguages(w http.ResponseWriter, r *http.Request) {
	if !rt.visualLanguagesEnabled(w) {
		return
	}
	out, err := rt.visualLanguages.List(r.Context())
	if err != nil {
		visualLanguageError(w, err)
		return
	}
	if out == nil {
		out = []domain.VisualLanguage{}
	}
	writeJSON(w, http.StatusOK, out)
}

func (rt *Router) handleGetVisualLanguage(w http.ResponseWriter, r *http.Request) {
	if !rt.visualLanguagesEnabled(w) {
		return
	}
	out, err := rt.visualLanguages.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		visualLanguageError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (rt *Router) handleVisualLanguageSamples(w http.ResponseWriter, r *http.Request) {
	if !rt.visualLanguagesEnabled(w) {
		return
	}
	out, err := rt.visualLanguages.Samples(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		visualLanguageError(w, err)
		return
	}
	if out == nil {
		out = []domain.LanguageSample{}
	}
	writeJSON(w, http.StatusOK, out)
}

func (rt *Router) handleRebuildVisualLanguageSamples(w http.ResponseWriter, r *http.Request) {
	if !rt.visualLanguagesEnabled(w) {
		return
	}
	out, err := rt.visualLanguages.RebuildSamples(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		visualLanguageError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// handleProjectVisualLanguage serves the language version a project is pinned to.
func (rt *Router) handleProjectVisualLanguage(w http.ResponseWriter, r *http.Request) {
	if !rt.visualLanguagesEnabled(w) {
		return
	}
	out, err := rt.visualLanguages.Pinned(r.Context(), chi.URLParam(r, "project_id"))
	if err != nil {
		visualLanguageError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (rt *Router) handleActivateVisualLanguage(w http.ResponseWriter, r *http.Request) {
	if !rt.visualLanguagesEnabled(w) {
		return
	}
	if err := rt.visualLanguages.Activate(r.Context(), chi.URLParam(r, "id")); err != nil {
		visualLanguageError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *Router) handleCopyVisualLanguage(w http.ResponseWriter, r *http.Request) {
	if !rt.visualLanguagesEnabled(w) {
		return
	}
	out, err := rt.visualLanguages.Copy(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		visualLanguageError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (rt *Router) handleUpdateVisualLanguage(w http.ResponseWriter, r *http.Request) {
	if !rt.visualLanguagesEnabled(w) {
		return
	}
	var req struct {
		Name        string                     `json:"name"`
		Note        string                     `json:"note"`
		Description string                     `json:"description"`
		Rules       domain.VisualLanguageRules `json:"rules"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	out, err := rt.visualLanguages.Update(r.Context(), chi.URLParam(r, "id"), application.VisualLanguageInput{
		Name: req.Name, Note: req.Note, Description: req.Description, Rules: req.Rules,
	})
	if err != nil {
		visualLanguageError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (rt *Router) handleDeleteVisualLanguage(w http.ResponseWriter, r *http.Request) {
	if !rt.visualLanguagesEnabled(w) {
		return
	}
	if err := rt.visualLanguages.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		visualLanguageError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
