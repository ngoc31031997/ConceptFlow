package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

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

// suggestShortScriptUseCase backs CR-026's short-script assistant (FR71) —
// no project_id, unlike suggestPublishMetadataUseCase, since a Creator can
// start a short from a bare topic without an existing project.
type suggestShortScriptUseCase interface {
	Execute(ctx context.Context, topic, sourceScriptContent string, language domain.ContentLanguage) (string, error)
}

// channelAssetsUseCase backs the two CR-023 correction endpoints. Normalize
// only publishes an AMQP command (no HTTP call to video-assembly); Preview
// only reads Orchestrator's own channel_asset_pointers projection.
type channelAssetsUseCase interface {
	Normalize(ctx context.Context, in application.NormalizeChannelAssetInput) error
	Preview(ctx context.Context) ([]domain.ChannelAssetPointer, error)
}

// promptTemplatesUseCase backs CR-025's prompt-template CRUD endpoints (the
// admin screen, and web-gui's runtime read of the current wording).
type promptTemplatesUseCase interface {
	Get(ctx context.Context, role domain.PromptRole, language string) (domain.PromptTemplate, error)
	List(ctx context.Context) ([]domain.PromptTemplate, error)
	Update(ctx context.Context, role domain.PromptRole, language, templateText string) (domain.PromptTemplate, error)
}

// saveAuthoringStoryUseCase backs CR-025 step 1's POST
// /v1/projects/{id}/authoring/story.
type saveAuthoringStoryUseCase interface {
	Execute(ctx context.Context, projectID, content string) error
}

// saveAuthoringStoryboardUseCase backs CR-025 step 2's POST
// /v1/projects/{id}/authoring/storyboard.
type saveAuthoringStoryboardUseCase interface {
	Execute(ctx context.Context, projectID, content string) error
}

// saveAuthoringCodeUseCase backs CR-025 step 3's POST
// /v1/projects/{id}/authoring/code.
type saveAuthoringCodeUseCase interface {
	Execute(ctx context.Context, projectID, content string) error
}

// saveAuthoringReviewUseCase backs CR-025 step 4's POST
// /v1/projects/{id}/authoring/review.
type saveAuthoringReviewUseCase interface {
	Execute(ctx context.Context, projectID, content string) error
}

// getAuthoringStateUseCase backs GET /v1/projects/{id}/authoring, letting the
// wizard rehydrate saved story/storyboard/code/review on reload/back-navigation.
type getAuthoringStateUseCase interface {
	Execute(ctx context.Context, projectID string) (application.AuthoringState, error)
}

// qcReportReader is the single read this router needs from the QC report
// store — narrower than domain.QCReportPort on purpose, so the GET endpoint
// cannot accidentally write.
type qcReportReader interface {
	LatestQCReport(ctx context.Context, projectID string) (*domain.QCReport, error)
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
	// Save persists the Creator's clip selections (CR-007 D7's POST
	// /v1/projects/{id}/clips) — a plain field update, not a use case, since
	// it only stores intent and never triggers anything by itself (merged into
	// generate_clips's request list at dispatch time instead).
	Save(ctx context.Context, project *domain.Project) error
}

// Router holds the REST handlers' use case dependencies.
type Router struct {
	startRenderSaga         startRenderSagaUseCase
	startPublishSaga        startPublishSagaUseCase
	retryStep               retryStepUseCase
	reviewOutline           reviewOutlineUseCase
	projects                projectStore
	suggestPublishMetadata  suggestPublishMetadataUseCase
	channelAssets           channelAssetsUseCase
	qcReports               qcReportReader
	suggestShortScript      suggestShortScriptUseCase
	promptTemplates         promptTemplatesUseCase
	saveAuthoringStory      saveAuthoringStoryUseCase
	saveAuthoringStoryboard saveAuthoringStoryboardUseCase
	saveAuthoringCode       saveAuthoringCodeUseCase
	saveAuthoringReview     saveAuthoringReviewUseCase
	getAuthoringState       getAuthoringStateUseCase
}

// WithPromptTemplates attaches CR-025's prompt-template use case, enabling
// GET /v1/prompts/{role} and the /v1/admin/prompts routes. Without it the
// routes answer 404, the same "unwired means absent" posture as
// WithQCReports.
func (rt *Router) WithPromptTemplates(promptTemplates promptTemplatesUseCase) *Router {
	rt.promptTemplates = promptTemplates
	return rt
}

// WithAuthoringStory attaches CR-025 step 1's save-story use case, enabling
// POST /v1/projects/{project_id}/authoring/story.
func (rt *Router) WithAuthoringStory(saveAuthoringStory saveAuthoringStoryUseCase) *Router {
	rt.saveAuthoringStory = saveAuthoringStory
	return rt
}

// WithAuthoringStoryboard attaches CR-025 step 2's save-storyboard use case,
// enabling POST /v1/projects/{project_id}/authoring/storyboard.
func (rt *Router) WithAuthoringStoryboard(saveAuthoringStoryboard saveAuthoringStoryboardUseCase) *Router {
	rt.saveAuthoringStoryboard = saveAuthoringStoryboard
	return rt
}

// WithAuthoringCode attaches CR-025 step 3's save-code use case, enabling
// POST /v1/projects/{project_id}/authoring/code.
func (rt *Router) WithAuthoringCode(saveAuthoringCode saveAuthoringCodeUseCase) *Router {
	rt.saveAuthoringCode = saveAuthoringCode
	return rt
}

// WithAuthoringReview attaches CR-025 step 4's save-review use case, enabling
// POST /v1/projects/{project_id}/authoring/review.
func (rt *Router) WithAuthoringReview(saveAuthoringReview saveAuthoringReviewUseCase) *Router {
	rt.saveAuthoringReview = saveAuthoringReview
	return rt
}

// WithAuthoringState attaches CR-025's read-side use case, enabling
// GET /v1/projects/{project_id}/authoring.
func (rt *Router) WithAuthoringState(getAuthoringState getAuthoringStateUseCase) *Router {
	rt.getAuthoringState = getAuthoringState
	return rt
}

// WithQCReports attaches the QC report store, enabling
// GET /v1/projects/{project_id}/qc-report (CR-021 FR61.1/FR61.2). Without it
// the route answers 404, the same way the CR-023 routes do when unwired.
func (rt *Router) WithQCReports(qcReports qcReportReader) *Router {
	rt.qcReports = qcReports
	return rt
}

// WithShortScriptSuggester attaches CR-026's short-script assistant,
// enabling POST /v1/short-script-suggestions. Without it the route answers
// 404 — same "unwired means absent, not broken" posture as WithQCReports.
func (rt *Router) WithShortScriptSuggester(suggestShortScript suggestShortScriptUseCase) *Router {
	rt.suggestShortScript = suggestShortScript
	return rt
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
	r.Get("/v1/projects/{project_id}/qc-report", rt.handleQCReport)
	r.Post("/v1/projects/{project_id}/clips", rt.handleCreateClip)
	r.Get("/v1/projects/{project_id}/clips", rt.handleListClips)
	r.Post("/v1/short-script-suggestions", rt.handleSuggestShortScript)
	// CR-025: prompt wording moved to the DB. Public read (web-gui's wizard
	// fetches the current template at runtime); admin list/update (the
	// PromptSettingsPage editor). No auth guard exists on this router today —
	// same "add plainly, don't invent auth" posture the plan called for; see
	// the router_test.go note and the final report's followup item.
	r.Get("/v1/prompts/{role}", rt.handleGetPromptTemplate)
	r.Get("/v1/admin/prompts", rt.handleListPromptTemplates)
	r.Put("/v1/admin/prompts/{role}", rt.handleUpdatePromptTemplate)
	r.Post("/v1/projects/{project_id}/authoring/story", rt.handleSaveAuthoringStory)
	r.Post("/v1/projects/{project_id}/authoring/storyboard", rt.handleSaveAuthoringStoryboard)
	r.Post("/v1/projects/{project_id}/authoring/code", rt.handleSaveAuthoringCode)
	r.Post("/v1/projects/{project_id}/authoring/review", rt.handleSaveAuthoringReview)
	r.Get("/v1/projects/{project_id}/authoring", rt.handleGetAuthoringState)
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
		VideoOutputMode:       domain.VideoOutputMode(req.VideoOutputMode),
		CompanionProjectID:    req.CompanionProjectID,
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
		AcknowledgeQC: req.AcknowledgeQC,
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

func (rt *Router) handleSuggestShortScript(w http.ResponseWriter, r *http.Request) {
	if rt.suggestShortScript == nil {
		writeError(w, http.StatusNotFound, "short script suggestions are not enabled")
		return
	}

	var req suggestShortScriptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	lang := domain.ContentLanguage(req.Language)
	if lang != domain.LanguageVietnamese && lang != domain.LanguageEnglish {
		writeError(w, http.StatusBadRequest, "language must be 'vi' or 'en'")
		return
	}

	script, err := rt.suggestShortScript.Execute(r.Context(), req.Topic, req.SourceScriptContent, lang)
	if err != nil {
		slog.Error("suggest-short-script failed", "error", err.Error())
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, suggestShortScriptResponse{ScriptContent: script})
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

// handleQCReport serves the project's latest automated QC report
// (CR-021 FR61.1) — what FR61.2's ResultPage renders above the publish button.
//
// A project that was never scored answers 200 with status "not_scored" and no
// findings, rather than 404. The GUI needs to draw something either way, and
// "we have not measured this" is an answer, not a missing resource.
func (rt *Router) handleQCReport(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	if rt.qcReports == nil {
		writeError(w, http.StatusNotFound, "quality checks are not enabled")
		return
	}

	report, err := rt.qcReports.LatestQCReport(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read the qc report")
		return
	}
	if report == nil {
		writeJSON(w, http.StatusOK, qcReportResponse{
			ProjectID: projectID,
			Status:    string(domain.QCStatusNotScored),
			Findings:  []qcFindingResponse{},
		})
		return
	}

	findings := make([]qcFindingResponse, 0, len(report.Findings))
	for _, f := range report.Findings {
		findings = append(findings, qcFindingResponse{
			Rule: f.Rule, Severity: f.Severity, Message: f.Message, TimestampSeconds: f.TimestampSeconds,
		})
	}
	out := qcReportResponse{
		ProjectID: report.ProjectID,
		Status:    string(report.Status),
		Reason:    report.Reason,
		Findings:  findings,
	}
	if !report.CreatedAt.IsZero() {
		created := report.CreatedAt.Format(time.RFC3339)
		out.CreatedAt = &created
	}
	if report.OverriddenAt != nil {
		overridden := report.OverriddenAt.Format(time.RFC3339)
		out.OverriddenAt = &overridden
	}
	writeJSON(w, http.StatusOK, out)
}

// handleCreateClip stores a Creator-entered vertical-clip selection (CR-007
// FR19.2/D3/D7) — {name, start_seconds, end_seconds, presets}. It only saves
// intent onto Project.ClipRequests; the actual cut happens later, when the
// Render Saga reaches generate_clips and merges this list with whatever the
// script's `with self.clip(...)` calls produced (D3 — a matching name here
// wins over the script one).
//
// Each requested preset is validated independently (FR19.6/19.7): a segment
// that does not fit "short" is rejected for that preset alone and reported in
// rejected_presets, while any preset that does fit is still saved and
// reported in accepted_presets. Only when EVERY preset is rejected does this
// answer 400 — a single bad preset must never block the ones the Creator got
// right.
func (rt *Router) handleCreateClip(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")

	var req createClipRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	if len(req.Presets) == 0 {
		writeError(w, http.StatusBadRequest, "presets is required (\"short\" and/or \"long\")")
		return
	}

	duration := req.EndSeconds - req.StartSeconds
	accepted := make([]string, 0, len(req.Presets))
	rejected := make(map[string]string, len(req.Presets))
	for _, preset := range req.Presets {
		if err := domain.ValidateClipDuration(duration, preset); err != nil {
			rejected[preset] = err.Error()
			continue
		}
		accepted = append(accepted, preset)
	}
	if len(accepted) == 0 {
		writeJSON(w, http.StatusBadRequest, createClipResponse{
			Name: req.Name, AcceptedPresets: accepted, RejectedPresets: rejected,
		})
		return
	}

	project, err := rt.projects.Get(r.Context(), projectID)
	if err != nil {
		writeUseCaseError(w, err)
		return
	}
	// Upsert by name (D3): a Creator refining the same clip's timing sends
	// another POST with the same name rather than accumulating duplicates.
	clipRequests := make([]map[string]interface{}, 0, len(project.ClipRequests)+1)
	for _, existing := range project.ClipRequests {
		if s, _ := existing["name"].(string); s != req.Name {
			clipRequests = append(clipRequests, existing)
		}
	}
	presetsAny := make([]interface{}, len(accepted))
	for i, p := range accepted {
		presetsAny[i] = p
	}
	clipRequests = append(clipRequests, map[string]interface{}{
		"name":          req.Name,
		"start_seconds": req.StartSeconds,
		"end_seconds":   req.EndSeconds,
		"presets":       presetsAny,
	})
	project.ClipRequests = clipRequests

	if err := rt.projects.Save(r.Context(), project); err != nil {
		writeError(w, http.StatusInternalServerError, "could not save the clip selection")
		return
	}

	writeJSON(w, http.StatusAccepted, createClipResponse{
		Name: req.Name, AcceptedPresets: accepted, RejectedPresets: rejected,
	})
}

// handleListClips serves the outcome of generate_clips (CR-007 D7/FR20.1) —
// what the results screen offers for download. An empty list (not 404) when
// the saga has not reached generate_clips yet, or when a project predates
// this CR: "nothing generated yet" is a normal state, not a missing resource.
func (rt *Router) handleListClips(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	project, err := rt.projects.Get(r.Context(), projectID)
	if err != nil {
		writeUseCaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"clips": toClipResultResponses(project.Clips)})
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
	// 409 rather than 422: nothing about the request is malformed, the project
	// is simply in a state that does not allow publishing yet — and the message
	// tells the Creator the one field that changes that (CR-021 FR61.3).
	case errors.Is(err, domain.ErrQCBlocked):
		writeErrorCode(w, http.StatusConflict, err.Error(), ErrorCodeQCBlocked)
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}

func writeErrorCode(w http.ResponseWriter, status int, message string, code string) {
	writeJSON(w, status, errorResponse{Error: message, Code: code})
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

// handleGetPromptTemplate serves the current wording for one pipeline role
// (CR-025). ?language defaults to "vi" — the primary content language this
// codebase's Creator-facing strings are written in.
func (rt *Router) handleGetPromptTemplate(w http.ResponseWriter, r *http.Request) {
	if rt.promptTemplates == nil {
		writeError(w, http.StatusNotFound, "prompt templates are not enabled")
		return
	}
	role := chi.URLParam(r, "role")
	if !domain.ValidPromptRole(role) {
		writeError(w, http.StatusBadRequest, "unknown role")
		return
	}
	language := r.URL.Query().Get("language")
	if language == "" {
		language = "vi"
	}

	template, err := rt.promptTemplates.Get(r.Context(), domain.PromptRole(role), language)
	if err != nil {
		writeError(w, http.StatusNotFound, "no template for this role/language")
		return
	}
	writeJSON(w, http.StatusOK, template)
}

// handleListPromptTemplates serves every role/language row for the admin
// editor screen (CR-025).
func (rt *Router) handleListPromptTemplates(w http.ResponseWriter, r *http.Request) {
	if rt.promptTemplates == nil {
		writeError(w, http.StatusNotFound, "prompt templates are not enabled")
		return
	}
	templates, err := rt.promptTemplates.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read prompt templates")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"templates": templates})
}

// handleUpdatePromptTemplate saves an editor's wording change and bumps
// version (CR-025). No destructive history is kept — unlike video_formats,
// a stale prompt does not need to stay reproducible against past renders.
func (rt *Router) handleUpdatePromptTemplate(w http.ResponseWriter, r *http.Request) {
	if rt.promptTemplates == nil {
		writeError(w, http.StatusNotFound, "prompt templates are not enabled")
		return
	}
	role := chi.URLParam(r, "role")

	var req struct {
		Language     string `json:"language"`
		TemplateText string `json:"template_text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	updated, err := rt.promptTemplates.Update(r.Context(), domain.PromptRole(role), req.Language, req.TemplateText)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// handleSaveAuthoringStory stores the Story Architect output a Creator
// pasted back after the external-AI round trip (CR-025 step 1).
func (rt *Router) handleSaveAuthoringStory(w http.ResponseWriter, r *http.Request) {
	if rt.saveAuthoringStory == nil {
		writeError(w, http.StatusNotFound, "authoring pipeline is not enabled")
		return
	}
	projectID := chi.URLParam(r, "project_id")

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if err := rt.saveAuthoringStory.Execute(r.Context(), projectID, req.Content); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"project_id": projectID})
}

// handleSaveAuthoringStoryboard stores the Visual Director output a Creator
// pasted back after the external-AI round trip (CR-025 step 2).
func (rt *Router) handleSaveAuthoringStoryboard(w http.ResponseWriter, r *http.Request) {
	if rt.saveAuthoringStoryboard == nil {
		writeError(w, http.StatusNotFound, "authoring pipeline is not enabled")
		return
	}
	projectID := chi.URLParam(r, "project_id")

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if err := rt.saveAuthoringStoryboard.Execute(r.Context(), projectID, req.Content); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"project_id": projectID})
}

// handleSaveAuthoringCode stores the Manim Engineer output a Creator pasted
// back after the external-AI round trip (CR-025 step 3).
func (rt *Router) handleSaveAuthoringCode(w http.ResponseWriter, r *http.Request) {
	if rt.saveAuthoringCode == nil {
		writeError(w, http.StatusNotFound, "authoring pipeline is not enabled")
		return
	}
	projectID := chi.URLParam(r, "project_id")

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if err := rt.saveAuthoringCode.Execute(r.Context(), projectID, req.Content); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"project_id": projectID})
}

// handleSaveAuthoringReview stores the Script Reviewer verdict a Creator
// pasted back after the external-AI round trip (CR-025 step 4).
func (rt *Router) handleSaveAuthoringReview(w http.ResponseWriter, r *http.Request) {
	if rt.saveAuthoringReview == nil {
		writeError(w, http.StatusNotFound, "authoring pipeline is not enabled")
		return
	}
	projectID := chi.URLParam(r, "project_id")

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if err := rt.saveAuthoringReview.Execute(r.Context(), projectID, req.Content); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"project_id": projectID})
}

// handleGetAuthoringState serves every saved authoring output (story,
// storyboard, code, review) so the wizard can rehydrate on reload/
// back-navigation instead of relying solely on client-side draft state.
// Missing outputs come back as "" rather than 404 — a step not yet saved is
// a normal state.
func (rt *Router) handleGetAuthoringState(w http.ResponseWriter, r *http.Request) {
	if rt.getAuthoringState == nil {
		writeError(w, http.StatusNotFound, "authoring pipeline is not enabled")
		return
	}
	projectID := chi.URLParam(r, "project_id")

	state, err := rt.getAuthoringState.Execute(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read authoring state")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"story":      state.Story,
		"storyboard": state.Storyboard,
		"code":       state.Code,
		"review":     state.Review,
	})
}
