package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"orchestrator/internal/application"
	"orchestrator/internal/domain"
)

// The 4 interfaces below narrow each REST handler's dependency to exactly
// the use case method it calls, so router_test.go can wire in small fake
// use cases without depending on application's port fakes.

type startRenderSagaUseCase interface {
	Execute(ctx context.Context, input application.StartRenderSagaInput) (*application.StartRenderSagaOutput, error)
}

type startPublishSagaUseCase interface {
	Execute(ctx context.Context, input application.StartPublishSagaInput) (*application.StartPublishSagaOutput, error)
}

type retryStepUseCase interface {
	Execute(ctx context.Context, projectID string) (*application.RetryStepOutput, error)
}

type suggestPublishMetadataUseCase interface {
	Execute(ctx context.Context, projectID string) (*application.SuggestPublishMetadataOutput, error)
}

// projectStore is the read/delete capability the project-list and
// project-detail/delete endpoints need; satisfied directly by
// domain.ProjectRepositoryPort.
type projectStore interface {
	Get(ctx context.Context, projectID string) (*domain.Project, error)
	List(ctx context.Context) ([]domain.ProjectSummary, error)
	Delete(ctx context.Context, projectID string) error
}

// Router holds the REST handlers' use case dependencies.
type Router struct {
	startRenderSaga        startRenderSagaUseCase
	startPublishSaga       startPublishSagaUseCase
	retryStep              retryStepUseCase
	projects               projectStore
	suggestPublishMetadata suggestPublishMetadataUseCase
}

// NewRouter constructs the Router with its dependencies (module-structure.md).
func NewRouter(startRenderSaga startRenderSagaUseCase, startPublishSaga startPublishSagaUseCase, retryStep retryStepUseCase, projects projectStore, suggestPublishMetadata suggestPublishMetadataUseCase) *Router {
	return &Router{startRenderSaga: startRenderSaga, startPublishSaga: startPublishSaga, retryStep: retryStep, projects: projects, suggestPublishMetadata: suggestPublishMetadata}
}

// Handler builds the chi.Router with all routes (health + REST endpoints).
func (rt *Router) Handler() http.Handler {
	r := chi.NewRouter()
	r.Get("/health", rt.handleHealth)
	r.Post("/v1/sagas/render", rt.handleStartRenderSaga)
	r.Post("/v1/sagas/publish", rt.handleStartPublishSaga)
	r.Get("/v1/projects", rt.handleListProjects)
	r.Get("/v1/projects/{project_id}", rt.handleGetProject)
	r.Post("/v1/projects/{project_id}/retry", rt.handleRetry)
	r.Delete("/v1/projects/{project_id}", rt.handleDeleteProject)
	r.Post("/v1/projects/{project_id}/suggest-metadata", rt.handleSuggestMetadata)
	return r
}

func (rt *Router) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (rt *Router) handleStartRenderSaga(w http.ResponseWriter, r *http.Request) {
	var req startRenderSagaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ProjectID == "" || req.ScriptContent == "" {
		writeError(w, http.StatusBadRequest, "project_id and script_content are required")
		return
	}
	lang := domain.ContentLanguage(req.ContentLanguage)
	if lang != domain.LanguageVietnamese && lang != domain.LanguageEnglish {
		writeError(w, http.StatusBadRequest, "voice_language must be 'vi' or 'en'")
		return
	}

	ttsEnabled := true
	if req.TTSEnabled != nil {
		ttsEnabled = *req.TTSEnabled
	}

	out, err := rt.startRenderSaga.Execute(r.Context(), application.StartRenderSagaInput{
		ProjectID:           req.ProjectID,
		ScriptContent:       req.ScriptContent,
		PluginID:            req.PluginID,
		CategoryHint:        req.CategoryHint,
		ContentLanguage:     lang,
		BackgroundMusicPath: req.BackgroundMusicPath,
		TTSEnabled:          ttsEnabled,
		VoiceID:             req.VoiceID,
		SubtitlesEnabled:    req.SubtitlesEnabled,
		SubtitleStyle:       req.SubtitleStyle,
	})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, sagaStartedResponse{SagaID: out.SagaID, Status: string(out.Status)})
}

func (rt *Router) handleStartPublishSaga(w http.ResponseWriter, r *http.Request) {
	var req startPublishSagaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ProjectID == "" || req.YoutubeTitle == "" {
		writeError(w, http.StatusBadRequest, "project_id and youtube_title are required")
		return
	}
	visibility := domain.Visibility(req.Visibility)
	switch visibility {
	case domain.VisibilityPublic, domain.VisibilityUnlisted, domain.VisibilityPrivate:
	default:
		writeError(w, http.StatusBadRequest, "visibility must be 'public', 'unlisted' or 'private'")
		return
	}
	if req.PublishAt != nil && visibility != domain.VisibilityPrivate {
		writeError(w, http.StatusBadRequest, "publish_at requires visibility 'private' (YouTube schedules it public at that time)")
		return
	}

	out, err := rt.startPublishSaga.Execute(r.Context(), application.StartPublishSagaInput{
		ProjectID:     req.ProjectID,
		Title:         req.YoutubeTitle,
		Description:   req.Description,
		Tags:          req.Tags,
		Visibility:    visibility,
		PublishAt:     req.PublishAt,
		ThumbnailPath: req.ThumbnailPath,
	})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, sagaStartedResponse{SagaID: out.SagaID, Status: string(out.Status)})
}

func (rt *Router) handleSuggestMetadata(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	out, err := rt.suggestPublishMetadata.Execute(r.Context(), projectID)
	if err != nil {
		slog.Error("suggest-metadata failed", "project_id", projectID, "error", err.Error())
		writeUseCaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, suggestMetadataResponse{
		Title:       out.Title,
		Description: out.Description,
		Tags:        out.Tags,
	})
}

func (rt *Router) handleGetProject(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	project, err := rt.projects.Get(r.Context(), projectID)
	if err != nil {
		writeUseCaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toProjectResponse(project))
}

func (rt *Router) handleListProjects(w http.ResponseWriter, r *http.Request) {
	summaries, err := rt.projects.List(r.Context())
	if err != nil {
		writeUseCaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toProjectListResponse(summaries))
}

func (rt *Router) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	if err := rt.projects.Delete(r.Context(), projectID); err != nil {
		writeUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *Router) handleRetry(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	out, err := rt.retryStep.Execute(r.Context(), projectID)
	if err != nil {
		writeUseCaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, retryResponse{SagaID: out.SagaID, Status: string(out.Status)})
}

// writeUseCaseError maps domain sentinel errors to the HTTP status codes
// specified in interface-contracts.md (404 for not found, 409 for invalid
// status preconditions); anything else is a 500.
func writeUseCaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrProjectNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrInvalidStatus):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
