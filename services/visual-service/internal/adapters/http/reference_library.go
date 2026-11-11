package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"visual/internal/application"
	"visual/internal/domain"
)

// referenceLibraryUseCase backs the /v1/reference-styles routes.
type referenceLibraryUseCase interface {
	List(ctx context.Context) ([]domain.ReferenceStyle, error)
	Create(ctx context.Context, in domain.ReferenceStyleInput) (domain.ReferenceStyle, error)
	Update(ctx context.Context, id string, in domain.ReferenceStyleInput) (domain.ReferenceStyle, error)
	Delete(ctx context.Context, id string) error
	UsedBy(ctx context.Context, id string) ([]string, error)
	Validate(in domain.ReferenceStyleInput) (domain.ReferenceStyleInput, error)
}

// WithReferenceLibrary adds the reference library routes to the router.
func (rt *Router) WithReferenceLibrary(library referenceLibraryUseCase) *Router {
	rt.referenceLibrary = library
	return rt
}

// mountReferenceLibrary registers the library routes when a library is set.
func (rt *Router) mountReferenceLibrary(r chi.Router) {
	if rt.referenceLibrary == nil {
		return
	}
	r.Get("/v1/reference-styles", rt.handleListReferenceStyles)
	r.Post("/v1/reference-styles", rt.handleCreateReferenceStyle)
	r.Post("/v1/reference-styles/import", rt.handleImportReferenceStyle)
	r.Put("/v1/reference-styles/{id}", rt.handleUpdateReferenceStyle)
	r.Delete("/v1/reference-styles/{id}", rt.handleDeleteReferenceStyle)
	r.Get("/v1/reference-styles/{id}/usage", rt.handleReferenceStyleUsage)
}

type referenceStylesResponse struct {
	Styles []domain.ReferenceStyle `json:"styles"`
}

type referenceUser struct {
	ProjectID string `json:"project_id"`
}

type referenceUsageResponse struct {
	Projects []referenceUser `json:"projects"`
}

func (rt *Router) handleListReferenceStyles(w http.ResponseWriter, r *http.Request) {
	styles, err := rt.referenceLibrary.List(r.Context())
	if err != nil {
		writeReferenceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, referenceStylesResponse{Styles: styles})
}

func (rt *Router) handleCreateReferenceStyle(w http.ResponseWriter, r *http.Request) {
	in, ok := decodeReferenceInput(w, r)
	if !ok {
		return
	}
	style, err := rt.referenceLibrary.Create(r.Context(), in)
	if err != nil {
		writeReferenceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, style)
}

func (rt *Router) handleUpdateReferenceStyle(w http.ResponseWriter, r *http.Request) {
	in, ok := decodeReferenceInput(w, r)
	if !ok {
		return
	}
	style, err := rt.referenceLibrary.Update(r.Context(), chi.URLParam(r, "id"), in)
	if err != nil {
		writeReferenceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, style)
}

func (rt *Router) handleDeleteReferenceStyle(w http.ResponseWriter, r *http.Request) {
	if err := rt.referenceLibrary.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeReferenceError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleReferenceStyleUsage lists the videos that picked a description, so the
// confirmation of an edit or delete can name them.
func (rt *Router) handleReferenceStyleUsage(w http.ResponseWriter, r *http.Request) {
	projects, err := rt.referenceLibrary.UsedBy(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeReferenceError(w, r, err)
		return
	}
	users := make([]referenceUser, 0, len(projects))
	for _, p := range projects {
		users = append(users, referenceUser{ProjectID: p})
	}
	writeJSON(w, http.StatusOK, referenceUsageResponse{Projects: users})
}

// pastedReferenceStyle is the JSON the Creator pastes: a description, or the
// {"error": "..."} an outside AI answers when it could not read the link.
type pastedReferenceStyle struct {
	domain.ReferenceStyleInput
	Error string `json:"error"`
}

type importedReferenceResponse struct {
	Style domain.ReferenceStyleInput `json:"style"`
}

// handleImportReferenceStyle checks a pasted description and returns it
// cleaned. It stores nothing: the Creator reviews the form, then saves.
func (rt *Router) handleImportReferenceStyle(w http.ResponseWriter, r *http.Request) {
	var pasted pastedReferenceStyle
	if err := json.NewDecoder(r.Body).Decode(&pasted); err != nil {
		writeError(w, http.StatusBadRequest, "nội dung dán vào không phải JSON hợp lệ")
		return
	}
	if pasted.Error != "" {
		writeError(w, http.StatusBadRequest, pasted.Error)
		return
	}
	style, err := rt.referenceLibrary.Validate(pasted.ReferenceStyleInput)
	if err != nil {
		writeReferenceError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, importedReferenceResponse{Style: style})
}

func decodeReferenceInput(w http.ResponseWriter, r *http.Request) (domain.ReferenceStyleInput, bool) {
	var in domain.ReferenceStyleInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "nội dung gửi lên không phải JSON hợp lệ")
		return in, false
	}
	return in, true
}

func writeReferenceError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidReferenceStyle):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, application.ErrReferenceStyleNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	default:
		slog.ErrorContext(r.Context(), "reference library request failed", "path", r.URL.Path, "error", err)
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}
