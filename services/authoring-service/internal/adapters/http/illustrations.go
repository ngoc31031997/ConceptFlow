package http

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"authoring/internal/application"
	"authoring/internal/domain"
)

// illustrationsUseCase backs the illustration library (CR-044).
type illustrationsUseCase interface {
	Folders(ctx context.Context) ([]domain.IllustrationFolder, error)
	CreateFolder(ctx context.Context, f domain.IllustrationFolder) (domain.IllustrationFolder, error)
	DeleteFolder(ctx context.Context, id string) error
	List(ctx context.Context, f application.IllustrationFilter) ([]domain.Illustration, error)
	Get(ctx context.Context, id string) (domain.Illustration, error)
	Try(ctx context.Context, name, code string) (application.IllustrationPreview, error)
	Create(ctx context.Context, i domain.Illustration) (domain.Illustration, error)
	Update(ctx context.Context, id string, i domain.Illustration) (domain.Illustration, error)
	SetStatus(ctx context.Context, id string, status domain.IllustrationStatus) (domain.Illustration, error)
	Delete(ctx context.Context, id string) error
	Preview(ctx context.Context, id string) (png, gif []byte, err error)
	Rerender(ctx context.Context, id string) (png, gif []byte, err error)
	Draw(ctx context.Context, req application.DrawRequest) (domain.Illustration, error)
	Redraw(ctx context.Context, id, note, model string) (domain.Illustration, error)
}

// WithIllustrations enables the illustration library routes.
func (rt *Router) WithIllustrations(u illustrationsUseCase) *Router {
	rt.illustrations = u
	return rt
}

func (rt *Router) illustrationRoutes(r chi.Router) {
	r.Get("/v1/illustration-folders", rt.handleListIllustrationFolders)
	r.Post("/v1/admin/illustration-folders", rt.handleCreateIllustrationFolder)
	r.Delete("/v1/admin/illustration-folders/{id}", rt.handleDeleteIllustrationFolder)
	r.Get("/v1/illustration-style", rt.handleIllustrationStyle)
	r.Get("/v1/illustrations", rt.handleListIllustrations)
	r.Get("/v1/illustrations/{id}", rt.handleGetIllustration)
	r.Get("/v1/illustrations/{id}/preview.png", rt.handleIllustrationPreview("png"))
	r.Get("/v1/illustrations/{id}/preview.gif", rt.handleIllustrationPreview("gif"))
	r.Post("/v1/illustration-tries", rt.handleTryIllustration)
	r.Post("/v1/admin/illustrations", rt.handleCreateIllustration)
	r.Put("/v1/admin/illustrations/{id}", rt.handleUpdateIllustration)
	r.Post("/v1/admin/illustrations/{id}/status", rt.handleIllustrationStatus)
	r.Post("/v1/admin/illustrations/{id}/rerender", rt.handleRerenderIllustration)
	r.Post("/v1/admin/illustrations/draw", rt.handleDrawIllustration)
	r.Post("/v1/admin/illustrations/{id}/redraw", rt.handleRedrawIllustration)
	r.Delete("/v1/admin/illustrations/{id}", rt.handleDeleteIllustration)
}

func illustrationError(w http.ResponseWriter, err error) {
	var invalid *application.InvalidIllustrationError
	var llmErr *application.LLMError
	switch {
	case errors.Is(err, application.ErrDrawerDisabled):
		writeError(w, http.StatusServiceUnavailable, err.Error())
	case errors.As(err, &llmErr), errors.Is(err, application.ErrLLMNotConfigured):
		status, msg := DescribeGenerateError(err)
		writeError(w, status, msg)
	case errors.As(err, &invalid):
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": invalid.Error(), "diagnostics": invalid.Diagnostics})
	case errors.Is(err, application.ErrIllustrationNotFound), errors.Is(err, application.ErrFolderNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, application.ErrIllustrationReadOnly), errors.Is(err, application.ErrFolderReadOnly):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, application.ErrIllustrationNameTaken), errors.Is(err, application.ErrFolderTaken),
		errors.Is(err, application.ErrFolderNotEmpty):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusBadRequest, err.Error())
	}
}

func (rt *Router) illustrationsEnabled(w http.ResponseWriter) bool {
	if rt.illustrations == nil {
		writeError(w, http.StatusNotFound, "illustration library is not enabled")
		return false
	}
	return true
}

type illustrationBody struct {
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	FolderID    string   `json:"folder_id"`
	Tags        []string `json:"tags"`
	Description string   `json:"description"`
	Usage       string   `json:"usage"`
	Code        string   `json:"code"`
}

func decodeIllustration(w http.ResponseWriter, r *http.Request) (domain.Illustration, bool) {
	var b illustrationBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return domain.Illustration{}, false
	}
	return domain.Illustration{Name: b.Name, Title: b.Title, FolderID: b.FolderID, Tags: b.Tags,
		Description: b.Description, Usage: b.Usage, Code: b.Code}, true
}

func (rt *Router) handleListIllustrationFolders(w http.ResponseWriter, r *http.Request) {
	if !rt.illustrationsEnabled(w) {
		return
	}
	list, err := rt.illustrations.Folders(r.Context())
	if err != nil {
		illustrationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"folders": list})
}

func (rt *Router) handleCreateIllustrationFolder(w http.ResponseWriter, r *http.Request) {
	if !rt.illustrationsEnabled(w) {
		return
	}
	var f domain.IllustrationFolder
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	out, err := rt.illustrations.CreateFolder(r.Context(), f)
	if err != nil {
		illustrationError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (rt *Router) handleDeleteIllustrationFolder(w http.ResponseWriter, r *http.Request) {
	if !rt.illustrationsEnabled(w) {
		return
	}
	if err := rt.illustrations.DeleteFolder(r.Context(), chi.URLParam(r, "id")); err != nil {
		illustrationError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *Router) handleListIllustrations(w http.ResponseWriter, r *http.Request) {
	if !rt.illustrationsEnabled(w) {
		return
	}
	q := r.URL.Query()
	list, err := rt.illustrations.List(r.Context(), application.IllustrationFilter{
		FolderID: q.Get("folder"), Query: q.Get("q"), Status: domain.IllustrationStatus(q.Get("status")),
	})
	if err != nil {
		illustrationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"illustrations": list})
}

func (rt *Router) handleGetIllustration(w http.ResponseWriter, r *http.Request) {
	if !rt.illustrationsEnabled(w) {
		return
	}
	out, err := rt.illustrations.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		illustrationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// handleIllustrationPreview serves the stored image itself, so web-gui can use
// it as an <img src>. The URL carries ?v=<version>, so it may be cached.
func (rt *Router) handleIllustrationPreview(kind string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !rt.illustrationsEnabled(w) {
			return
		}
		png, gif, err := rt.illustrations.Preview(r.Context(), chi.URLParam(r, "id"))
		if err != nil {
			illustrationError(w, err)
			return
		}
		body, mime := png, "image/png"
		if kind == "gif" {
			body, mime = gif, "image/gif"
		}
		if len(body) == 0 {
			writeError(w, http.StatusNotFound, "no "+kind+" preview for this illustration")
			return
		}
		w.Header().Set("Content-Type", mime)
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.Header().Set("Cache-Control", "private, max-age=86400")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

func previewJSON(png, gif []byte) map[string]any {
	return map[string]any{
		"png": base64.StdEncoding.EncodeToString(png),
		"gif": base64.StdEncoding.EncodeToString(gif),
	}
}

func (rt *Router) handleTryIllustration(w http.ResponseWriter, r *http.Request) {
	if !rt.illustrationsEnabled(w) {
		return
	}
	var b struct {
		Name string `json:"name"`
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	out, err := rt.illustrations.Try(r.Context(), b.Name, b.Code)
	if err != nil {
		illustrationError(w, err)
		return
	}
	body := previewJSON(out.PNG, out.GIF)
	body["warnings"] = out.Warnings
	writeJSON(w, http.StatusOK, body)
}

func (rt *Router) handleCreateIllustration(w http.ResponseWriter, r *http.Request) {
	if !rt.illustrationsEnabled(w) {
		return
	}
	i, ok := decodeIllustration(w, r)
	if !ok {
		return
	}
	out, err := rt.illustrations.Create(r.Context(), i)
	if err != nil {
		illustrationError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (rt *Router) handleUpdateIllustration(w http.ResponseWriter, r *http.Request) {
	if !rt.illustrationsEnabled(w) {
		return
	}
	i, ok := decodeIllustration(w, r)
	if !ok {
		return
	}
	out, err := rt.illustrations.Update(r.Context(), chi.URLParam(r, "id"), i)
	if err != nil {
		illustrationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (rt *Router) handleIllustrationStatus(w http.ResponseWriter, r *http.Request) {
	if !rt.illustrationsEnabled(w) {
		return
	}
	var b struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	out, err := rt.illustrations.SetStatus(r.Context(), chi.URLParam(r, "id"), domain.IllustrationStatus(b.Status))
	if err != nil {
		illustrationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (rt *Router) handleRerenderIllustration(w http.ResponseWriter, r *http.Request) {
	if !rt.illustrationsEnabled(w) {
		return
	}
	png, gif, err := rt.illustrations.Rerender(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		illustrationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, previewJSON(png, gif))
}

func (rt *Router) handleDeleteIllustration(w http.ResponseWriter, r *http.Request) {
	if !rt.illustrationsEnabled(w) {
		return
	}
	if err := rt.illustrations.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		illustrationError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleIllustrationStyle serves the channel's style rules and the ids of the
// reference drawings, which web-gui shows with their previews (CR-044).
func (rt *Router) handleIllustrationStyle(w http.ResponseWriter, _ *http.Request) {
	ids := []string{}
	for _, e := range domain.ExemplarIllustrations() {
		ids = append(ids, e.ID)
	}
	writeJSON(w, http.StatusOK, map[string]any{"rules": domain.IllustrationStyleGuide(), "exemplar_ids": ids})
}

func (rt *Router) handleDrawIllustration(w http.ResponseWriter, r *http.Request) {
	if !rt.illustrationsEnabled(w) {
		return
	}
	var b struct {
		Description string `json:"description"`
		FolderID    string `json:"folder_id"`
		Name        string `json:"name"`
		Model       string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	out, err := rt.illustrations.Draw(r.Context(), application.DrawRequest{
		Description: b.Description, FolderID: b.FolderID, Name: b.Name, Model: b.Model,
	})
	if err != nil {
		illustrationError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (rt *Router) handleRedrawIllustration(w http.ResponseWriter, r *http.Request) {
	if !rt.illustrationsEnabled(w) {
		return
	}
	var b struct {
		Note  string `json:"note"`
		Model string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	out, err := rt.illustrations.Redraw(r.Context(), chi.URLParam(r, "id"), b.Note, b.Model)
	if err != nil {
		illustrationError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}
