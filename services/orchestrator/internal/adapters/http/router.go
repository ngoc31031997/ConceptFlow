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

type reviewOutlineUseCase interface {
	Approve(ctx context.Context, projectID string) error
	Reject(ctx context.Context, projectID string) error
	EditNarration(ctx context.Context, projectID string, sceneIndex int, newText string) error
}

type retryStepUseCase interface {
	Execute(ctx context.Context, projectID string) (*application.RetryStepOutput, error)
}

type suggestPublishMetadataUseCase interface {
	Execute(ctx context.Context, projectID string) (*application.SuggestPublishMetadataOutput, error)
}

// channelAssetsUseCase backs the two CR-023 correction endpoints. Normalize
// only publishes an AMQP command (no HTTP call to video-assembly); Preview
// only reads Orchestrator's own channel_asset_pointers projection.
type channelAssetsUseCase interface {
	Normalize(ctx context.Context, in application.NormalizeChannelAssetInput) error
	Preview(ctx context.Context) ([]domain.ChannelAssetPointer, error)
}

// projectStore is the read/delete capability the project-list and
// project-detail/delete endpoints need; satisfied directly by
// domain.ProjectRepositoryPort.
type projectStore interface {
	ListVoiceCalibrations(ctx context.Context) ([]domain.VoiceCalibration, error)
	ListVideoFormats(ctx context.Context) ([]domain.VideoFormat, error)
	SaveVideoFormat(ctx context.Context, format domain.VideoFormat) (domain.VideoFormat, error)
	Get(ctx context.Context, projectID string) (*domain.Project, error)
	List(ctx context.Context) ([]domain.ProjectSummary, error)
	Delete(ctx context.Context, projectID string) error
}

// Router holds the REST handlers' use case dependencies.
type Router struct {
	startRenderSaga        startRenderSagaUseCase
	startPublishSaga       startPublishSagaUseCase
	retryStep              retryStepUseCase
	reviewOutline          reviewOutlineUseCase
	projects               projectStore
	suggestPublishMetadata suggestPublishMetadataUseCase
	channelAssets          channelAssetsUseCase
}

// NewRouter constructs the Router with its dependencies (module-structure.md).
// channelAssets may be nil in tests that do not exercise CR-023's routes.
func NewRouter(startRenderSaga startRenderSagaUseCase, startPublishSaga startPublishSagaUseCase, retryStep retryStepUseCase, projects projectStore, suggestPublishMetadata suggestPublishMetadataUseCase, reviewOutline reviewOutlineUseCase, channelAssets channelAssetsUseCase) *Router {
	return &Router{startRenderSaga: startRenderSaga, startPublishSaga: startPublishSaga, retryStep: retryStep, projects: projects, suggestPublishMetadata: suggestPublishMetadata, reviewOutline: reviewOutline, channelAssets: channelAssets}
}

// Handler builds the chi.Router with all routes (health + REST endpoints).
func (rt *Router) Handler() http.Handler {
	r := chi.NewRouter()
	r.Get("/health", rt.handleHealth)
	r.Post("/v1/sagas/render", rt.handleStartRenderSaga)
	r.Post("/v1/sagas/publish", rt.handleStartPublishSaga)
	r.Get("/v1/projects", rt.handleListProjects)
	r.Get("/v1/voice-calibration", rt.handleVoiceCalibration)
	r.Get("/v1/formats", rt.handleListFormats)
	r.Post("/v1/formats", rt.handleSaveFormat)
	r.Get("/v1/projects/{project_id}", rt.handleGetProject)
	r.Post("/v1/projects/{project_id}/retry", rt.handleRetry)
	r.Post("/v1/projects/{project_id}/approve", rt.handleApproveOutline)
	r.Post("/v1/projects/{project_id}/reject", rt.handleRejectOutline)
	r.Post("/v1/projects/{project_id}/narration", rt.handleEditNarration)
	r.Delete("/v1/projects/{project_id}", rt.handleDeleteProject)
	r.Post("/v1/projects/{project_id}/suggest-metadata", rt.handleSuggestMetadata)
	r.Post("/v1/channel-assets/{kind}", rt.handleNormalizeChannelAsset)
	r.Get("/v1/channel-assets/preview", rt.handleChannelAssetPreview)
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
		ProjectID:             req.ProjectID,
		ScriptContent:         req.ScriptContent,
		PluginID:              req.PluginID,
		VideoFormatID:         req.VideoFormatID,
		ReviewEnabled:         req.ReviewEnabled,
		CategoryHint:          req.CategoryHint,
		ContentLanguage:       lang,
		BackgroundMusicPath:   req.BackgroundMusicPath,
		TTSEnabled:            ttsEnabled,
		VoiceID:               req.VoiceID,
		SubtitlesEnabled:      req.SubtitlesEnabled,
		SubtitleMode:          domain.SubtitleMode(req.SubtitleMode),
		SubtitleStyle:         req.SubtitleStyle,
		RenderQuality:         domain.RenderQuality(req.RenderQuality),
		BackgroundMusicVolume: req.BackgroundMusicVolume,
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
		ChannelID:     req.ChannelID,
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

// handleNormalizeChannelAsset triggers video-assembly to normalize a
// Creator-uploaded intro/outro file (CR-023 correction, FR65.1). api-gateway
// has already written the file to the shared volume and computed its hash;
// this handler only publishes the normalize_channel_asset AMQP command — it
// never calls video-assembly over HTTP (no such server exists).
func (rt *Router) handleNormalizeChannelAsset(w http.ResponseWriter, r *http.Request) {
	if rt.channelAssets == nil {
		writeError(w, http.StatusNotFound, "channel assets are not enabled")
		return
	}
	kind := chi.URLParam(r, "kind")
	if kind != "intro" && kind != "outro" {
		writeError(w, http.StatusBadRequest, "kind must be 'intro' or 'outro'")
		return
	}

	var req struct {
		FilePath      string `json:"file_path"`
		SourceHash    string `json:"source_hash"`
		RenderQuality string `json:"render_quality"`
		AssetRole     string `json:"asset_role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.FilePath == "" {
		writeError(w, http.StatusBadRequest, "file_path is required")
		return
	}
	// asset_role tells video-assembly whether file_path is the sting clip or
	// its music bed (FR66.5). Absent means "video", the only thing this
	// endpoint used to accept.
	role := req.AssetRole
	if role == "" {
		role = application.AssetRoleVideo
	}
	if role != application.AssetRoleVideo && role != application.AssetRoleMusic {
		writeError(w, http.StatusBadRequest, "asset_role must be 'video' or 'music'")
		return
	}
	quality := domain.RenderQuality(req.RenderQuality)
	if req.RenderQuality == "" {
		quality = domain.DefaultRenderQuality
	}

	err := rt.channelAssets.Normalize(r.Context(), application.NormalizeChannelAssetInput{
		Kind:          kind,
		FilePath:      req.FilePath,
		SourceHash:    req.SourceHash,
		RenderQuality: quality,
		AssetRole:     role,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not queue normalize")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"kind": kind, "status": "queued"})
}

// handleChannelAssetPreview serves Orchestrator's own channel_asset_pointers
// projection (CR-023 correction, FR67.4's data half — no HTTP call to
// video-assembly, whose full channel_assets table Orchestrator never sees).
func (rt *Router) handleChannelAssetPreview(w http.ResponseWriter, r *http.Request) {
	if rt.channelAssets == nil {
		writeError(w, http.StatusNotFound, "channel assets are not enabled")
		return
	}
	pointers, err := rt.channelAssets.Preview(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read channel assets")
		return
	}
	type pointerResponse struct {
		Kind          string `json:"kind"`
		RenderQuality string `json:"render_quality"`
		AssetID       string `json:"asset_id"`
		Version       int    `json:"version"`
	}
	out := make([]pointerResponse, 0, len(pointers))
	for _, p := range pointers {
		out = append(out, pointerResponse{Kind: p.Kind, RenderQuality: string(p.RenderQuality), AssetID: p.AssetID, Version: p.Version})
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"assets": out})
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

// handleVoiceCalibration serves the measured reading rate of every voice that
// has enough samples (CR-016 FR43.2).
//
// The Web GUI uses it to make the authoring-time estimate match what the
// Creator's own voice actually does, instead of a constant that was never
// checked against anything. Voices below the sample threshold are omitted
// rather than reported with a low-confidence number — the GUI falls back to the
// language default for those, which is the honest answer.
func (rt *Router) handleVoiceCalibration(w http.ResponseWriter, r *http.Request) {
	rows, err := rt.projects.ListVoiceCalibrations(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read voice calibration")
		return
	}

	out := make(map[string]float64, len(rows))
	for _, row := range rows {
		if wpm, ok := row.WordsPerMinute(); ok {
			out[row.VoiceID] = wpm
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"words_per_minute": out})
}

// handleListFormats serves the video formats the Creator can pick from
// (CR-019 FR51.3).
func (rt *Router) handleListFormats(w http.ResponseWriter, r *http.Request) {
	formats, err := rt.projects.ListVideoFormats(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read video formats")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"formats": formats})
}

// handleSaveFormat stores a format as a new version (CR-019 FR51.5).
//
// Cloning and editing is the same operation as creating: post a format with a
// new id to clone, or with an existing id to add a version to it. There is no
// destructive edit, because a project rendered against version 3 has to keep
// reporting version 3's beats.
func (rt *Router) handleSaveFormat(w http.ResponseWriter, r *http.Request) {
	var format domain.VideoFormat
	if err := json.NewDecoder(r.Body).Decode(&format); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if format.ID == "" || format.Name == "" {
		writeError(w, http.StatusBadRequest, "format id and name are required")
		return
	}
	if len(format.Beats) == 0 {
		writeError(w, http.StatusBadRequest, "a format needs at least one beat")
		return
	}
	if format.MinSeconds < 0 || format.MaxSeconds < format.MinSeconds {
		writeError(w, http.StatusBadRequest, "max_seconds must not be below min_seconds")
		return
	}

	saved, err := rt.projects.SaveVideoFormat(r.Context(), format)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not save the format")
		return
	}
	writeJSON(w, http.StatusCreated, saved)
}

// handleApproveOutline releases a saga waiting at the review gate (CR-024 FR69.2).
func (rt *Router) handleApproveOutline(w http.ResponseWriter, r *http.Request) {
	rt.handleReviewDecision(w, r, rt.reviewOutline.Approve)
}

// handleRejectOutline ends the saga so the Creator can go and edit (FR69.3).
func (rt *Router) handleRejectOutline(w http.ResponseWriter, r *http.Request) {
	rt.handleReviewDecision(w, r, rt.reviewOutline.Reject)
}

func (rt *Router) handleReviewDecision(w http.ResponseWriter, r *http.Request, decide func(context.Context, string) error) {
	projectID := chi.URLParam(r, "project_id")
	if rt.reviewOutline == nil {
		writeError(w, http.StatusNotFound, "review gate is not enabled")
		return
	}

	switch err := decide(r.Context(), projectID); {
	case err == nil:
		writeJSON(w, http.StatusOK, map[string]string{"project_id": projectID})
	case errors.Is(err, application.ErrNotAwaitingReview):
		// 409 rather than 400: nothing about the request is malformed, the
		// project has simply moved on. Approving twice lands here, which is
		// what makes the second press harmless (FR69.4).
		writeError(w, http.StatusConflict, "project is not awaiting review")
	default:
		writeError(w, http.StatusInternalServerError, "could not record the decision")
	}
}

// handleEditNarration rewrites one narration line while the project waits at
// the review gate (CR-024 FR70).
func (rt *Router) handleEditNarration(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	if rt.reviewOutline == nil {
		writeError(w, http.StatusNotFound, "review gate is not enabled")
		return
	}

	var req struct {
		SceneIndex int    `json:"scene_index"`
		Text       string `json:"narration_text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	switch err := rt.reviewOutline.EditNarration(r.Context(), projectID, req.SceneIndex, req.Text); {
	case err == nil:
		writeJSON(w, http.StatusAccepted, map[string]string{"project_id": projectID})
	case errors.Is(err, application.ErrNotAwaitingReview):
		writeError(w, http.StatusConflict, "project is not awaiting review")
	case errors.Is(err, domain.ErrNarrationNotEditable):
		// 422 rather than 400: the request is well formed, but this particular
		// line cannot be traced back to one place in the source — a loop or an
		// f-string produced it. The message says which, so the GUI can explain
		// instead of just refusing.
		writeError(w, http.StatusUnprocessableEntity, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "could not edit the narration")
	}
}
