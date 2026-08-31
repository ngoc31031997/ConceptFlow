package http

import (
	"context"
	"encoding/json"
	"errors"
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

// projectReader is the minimal read capability GET /v1/projects/{id} needs;
// satisfied directly by domain.ProjectRepositoryPort.
type projectReader interface {
	Get(ctx context.Context, projectID string) (*domain.Project, error)
}

// Router holds the REST handlers' use case dependencies.
type Router struct {
	startRenderSaga  startRenderSagaUseCase
	startPublishSaga startPublishSagaUseCase
	retryStep        retryStepUseCase
	projects         projectReader
}

// NewRouter constructs the Router with its 4 dependencies (module-structure.md).
func NewRouter(startRenderSaga startRenderSagaUseCase, startPublishSaga startPublishSagaUseCase, retryStep retryStepUseCase, projects projectReader) *Router {
	return &Router{startRenderSaga: startRenderSaga, startPublishSaga: startPublishSaga, retryStep: retryStep, projects: projects}
}

// Handler builds the chi.Router with all 5 routes (4 REST endpoints + health).
func (rt *Router) Handler() http.Handler {
	r := chi.NewRouter()
	r.Get("/health", rt.handleHealth)
	r.Post("/v1/sagas/render", rt.handleStartRenderSaga)
	r.Post("/v1/sagas/publish", rt.handleStartPublishSaga)
	r.Get("/v1/projects/{project_id}", rt.handleGetProject)
	r.Post("/v1/projects/{project_id}/retry", rt.handleRetry)
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
	if req.ProjectID == "" || req.ScriptContent == "" || req.PluginID == "" {
		writeError(w, http.StatusBadRequest, "project_id, script_content and plugin_id are required")
		return
	}
	lang := domain.VoiceLanguage(req.VoiceLanguage)
	if lang != domain.LanguageVietnamese && lang != domain.LanguageEnglish {
		writeError(w, http.StatusBadRequest, "voice_language must be 'vi' or 'en'")
		return
	}

	out, err := rt.startRenderSaga.Execute(r.Context(), application.StartRenderSagaInput{
		ProjectID:           req.ProjectID,
		ScriptContent:       req.ScriptContent,
		PluginID:            req.PluginID,
		VoiceLanguage:       lang,
		BackgroundMusicPath: req.BackgroundMusicPath,
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

	out, err := rt.startPublishSaga.Execute(r.Context(), application.StartPublishSagaInput{
		ProjectID:   req.ProjectID,
		Title:       req.YoutubeTitle,
		Description: req.Description,
		Tags:        req.Tags,
		Visibility:  visibility,
	})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, sagaStartedResponse{SagaID: out.SagaID, Status: string(out.Status)})
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
