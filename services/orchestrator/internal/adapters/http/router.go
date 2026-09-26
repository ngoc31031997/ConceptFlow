package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
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

type forkProjectUseCase interface {
	Execute(ctx context.Context, sourceID string, fromStep int) (*application.ForkProjectOutput, error)
}

// forkedFromReader is the optional lineage read on the project store.
type forkedFromReader interface {
	ForkedFrom(ctx context.Context, projectID string) (string, error)
}

type cancelStepUseCase interface {
	Execute(ctx context.Context, projectID string) (*application.CancelStepOutput, error)
}

// channelAssetsUseCase backs the two CR-023 correction endpoints. Normalize
// only publishes an AMQP command (no HTTP call to video-assembly); Preview
// only reads Orchestrator's own channel_asset_pointers projection.
type channelAssetsUseCase interface {
	Normalize(ctx context.Context, in application.NormalizeChannelAssetInput) error
	Preview(ctx context.Context) ([]domain.ChannelAssetPointer, error)
}

// createProjectDraftUseCase backs CR-028 FR83.1's POST /v1/projects — the
// project row is created here, at wizard step 1, instead of at
// POST /v1/sagas/render.
type createProjectDraftUseCase interface {
	Execute(ctx context.Context, input application.CreateProjectDraftInput) (*application.CreateProjectDraftOutput, error)
}

// updateProjectTopicUseCase backs CR-028 FR83.2's PATCH
// /v1/projects/{id}/topic.
type updateProjectTopicUseCase interface {
	Execute(ctx context.Context, input application.UpdateProjectTopicInput) (*application.UpdateProjectTopicOutput, error)
}

// saveWizardSettingsUseCase backs PUT /v1/projects/{id}/settings (wizard step 2).
type saveWizardSettingsUseCase interface {
	Execute(ctx context.Context, projectID string, s domain.WizardSettings) error
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
	// The three below serve authoring-service's internal reads (CR-040 FR111).
	GetStatus(ctx context.Context, projectID string) (domain.ProjectStatus, error)
	GetVideoFormat(ctx context.Context, formatID string, version int) (domain.VideoFormat, error)
	GetVoiceCalibration(ctx context.Context, voiceID string) (domain.VoiceCalibration, error)
	// Save persists the Creator's clip selections (CR-007 D7's POST
	// /v1/projects/{id}/clips) — a plain field update, not a use case, since
	// it only stores intent and never triggers anything by itself (merged into
	// generate_clips's request list at dispatch time instead).
	Save(ctx context.Context, project *domain.Project) error
}

type deleteProjectUseCase interface {
	Execute(ctx context.Context, projectID string) error
}

// WithDeleteProject switches DELETE /v1/projects/{id} to the delete saga.
func (rt *Router) WithDeleteProject(d deleteProjectUseCase) *Router {
	rt.deleteProject = d
	return rt
}

// Router holds the REST handlers' use case dependencies.
type Router struct {
	startRenderSaga    startRenderSagaUseCase
	startPublishSaga   startPublishSagaUseCase
	retryStep          retryStepUseCase
	deleteProject      deleteProjectUseCase
	reviewOutline      reviewOutlineUseCase
	projects           projectStore
	channelAssets      channelAssetsUseCase
	qcReports          qcReportReader
	projectErrors      application.ProjectErrorLogPort
	projectEvents      domain.ProjectEventPort
	cancelStep         cancelStepUseCase
	forkProject        forkProjectUseCase
	saveWizardSettings saveWizardSettingsUseCase
	createProjectDraft createProjectDraftUseCase
	updateProjectTopic updateProjectTopicUseCase
	// authored says which authoring artefacts a draft holds — that lives in
	// authoring-service now (CR-040 FR111). nil = none known.
	authored authoredContentReader
}

// authoredContentReader tells the flow position of a draft which of the three
// authoring outputs exist, from authoring-service.
type authoredContentReader interface {
	Summaries(ctx context.Context, projectIDs []string) (map[string]application.AuthoringSummary, error)
}

// WithAuthoredContent attaches the authoring-service reader used by flowFor.
func (rt *Router) WithAuthoredContent(a authoredContentReader) *Router {
	rt.authored = a
	return rt
}

// WithProjectErrors enables GET /v1/projects/{id}/errors, the read side of the
// project_errors trace.
func (rt *Router) WithProjectErrors(log application.ProjectErrorLogPort) *Router {
	rt.projectErrors = log
	return rt
}

func (rt *Router) handleListProjectErrors(w http.ResponseWriter, r *http.Request) {
	if rt.projectErrors == nil {
		writeJSON(w, http.StatusOK, []application.ProjectError{})
		return
	}
	errs, err := rt.projectErrors.ListProjectErrors(r.Context(), chi.URLParam(r, "project_id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "không đọc được nhật ký lỗi")
		return
	}
	writeJSON(w, http.StatusOK, errs)
}

// WithProjectEvents enables the journey log endpoints (GET
// /v1/projects/{id}/events and GET /v1/events).
func (rt *Router) WithProjectEvents(events domain.ProjectEventPort) *Router {
	rt.projectEvents = events
	return rt
}

func (rt *Router) handleListProjectEvents(w http.ResponseWriter, r *http.Request) {
	if rt.projectEvents == nil {
		writeJSON(w, http.StatusOK, []domain.ProjectEvent{})
		return
	}
	events, err := rt.projectEvents.ListProjectEvents(r.Context(), chi.URLParam(r, "project_id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "không đọc được nhật ký dự án")
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (rt *Router) handleListRecentEvents(w http.ResponseWriter, r *http.Request) {
	if rt.projectEvents == nil {
		writeJSON(w, http.StatusOK, []domain.ProjectEvent{})
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	events, err := rt.projectEvents.ListRecentProjectEvents(r.Context(), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "không đọc được nhật ký dự án")
		return
	}
	writeJSON(w, http.StatusOK, events)
}

// WithForkProject enables POST /v1/projects/{id}/fork.
func (rt *Router) WithForkProject(f forkProjectUseCase) *Router {
	rt.forkProject = f
	return rt
}

// handleFork creates a new project from this one, starting again at from_step
// (2..5). The original is untouched.
func (rt *Router) handleFork(w http.ResponseWriter, r *http.Request) {
	if rt.forkProject == nil {
		writeError(w, http.StatusNotFound, "fork is not enabled")
		return
	}
	var req struct {
		FromStep int `json:"from_step"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	out, err := rt.forkProject.Execute(r.Context(), chi.URLParam(r, "project_id"), req.FromStep)
	if err != nil {
		if errors.Is(err, application.ErrForkStepInvalid) {
			writeError(w, http.StatusBadRequest, "Chỉ tạo bản mới được từ bước Cấu hình đến bước Code.")
			return
		}
		writeUseCaseError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"project_id": out.ProjectID, "from_step": out.FromStep, "needs_music_reselect": out.NeedsMusicReselect,
	})
}

// WithCancelStep enables POST /v1/projects/{id}/cancel.
func (rt *Router) WithCancelStep(c cancelStepUseCase) *Router {
	rt.cancelStep = c
	return rt
}

// handleCancel stops the step a project is running and leaves it at that step
// (failed, marked cancelled) so the Creator can retry it.
func (rt *Router) handleCancel(w http.ResponseWriter, r *http.Request) {
	if rt.cancelStep == nil {
		writeError(w, http.StatusNotFound, "cancel is not enabled")
		return
	}
	out, err := rt.cancelStep.Execute(r.Context(), chi.URLParam(r, "project_id"))
	if err != nil {
		if errors.Is(err, domain.ErrInvalidStatus) {
			writeError(w, http.StatusConflict, "Không có bước nào đang chạy để huỷ.")
			return
		}
		writeUseCaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"step": string(out.Step), "status": string(out.Status)})
}

// WithProjectDrafts attaches CR-028's early-draft use cases, enabling
// POST /v1/projects and PATCH /v1/projects/{project_id}/topic.
func (rt *Router) WithProjectDrafts(createProjectDraft createProjectDraftUseCase, updateProjectTopic updateProjectTopicUseCase) *Router {
	rt.createProjectDraft = createProjectDraft
	rt.updateProjectTopic = updateProjectTopic
	return rt
}

// WithWizard attaches the wizard's per-step saves, enabling
// PUT /v1/projects/{project_id}/settings.
func (rt *Router) WithWizard(saveSettings saveWizardSettingsUseCase) *Router {
	rt.saveWizardSettings = saveSettings
	return rt
}

// WithQCReports attaches the QC report store, enabling
// GET /v1/projects/{project_id}/qc-report (CR-021 FR61.1/FR61.2). Without it
// the route answers 404, the same way the CR-023 routes do when unwired.
func (rt *Router) WithQCReports(qcReports qcReportReader) *Router {
	rt.qcReports = qcReports
	return rt
}

// NewRouter constructs the Router with its dependencies (module-structure.md).
// channelAssets may be nil in tests that do not exercise CR-023's routes.
func NewRouter(startRenderSaga startRenderSagaUseCase, startPublishSaga startPublishSagaUseCase, retryStep retryStepUseCase, projects projectStore, reviewOutline reviewOutlineUseCase, channelAssets channelAssetsUseCase) *Router {
	return &Router{startRenderSaga: startRenderSaga, startPublishSaga: startPublishSaga, retryStep: retryStep, projects: projects, reviewOutline: reviewOutline, channelAssets: channelAssets}
}

// Handler builds the chi.Router with all routes (health + REST endpoints).
func (rt *Router) Handler() http.Handler {
	r := chi.NewRouter()
	r.Get("/health", rt.handleHealth)
	// Internal — called by authoring-service only (never routed by the gateway).
	r.Get("/internal/v1/projects/{project_id}", rt.handleInternalGetProject)
	r.Get("/internal/v1/projects/{project_id}/status", rt.handleInternalProjectStatus)
	r.Get("/internal/v1/formats/{format_id}", rt.handleInternalGetFormat)
	r.Get("/internal/v1/voices/{voice_id}/calibration", rt.handleInternalVoiceCalibration)
	r.Post("/internal/v1/projects/{project_id}/errors", rt.handleInternalAppendError)
	r.Post("/internal/v1/projects/{project_id}/events", rt.handleInternalAppendEvent)
	r.Post("/v1/sagas/render", rt.handleStartRenderSaga)
	r.Post("/v1/sagas/publish", rt.handleStartPublishSaga)
	r.Get("/v1/projects", rt.handleListProjects)
	// CR-028 FR83/FR84/FR85: the project row now exists from wizard step 1.
	r.Post("/v1/projects", rt.handleCreateProjectDraft)
	r.Patch("/v1/projects/{project_id}/topic", rt.handleUpdateProjectTopic)
	r.Get("/v1/voice-calibration", rt.handleVoiceCalibration)
	r.Get("/v1/formats", rt.handleListFormats)
	r.Post("/v1/formats", rt.handleSaveFormat)
	r.Get("/v1/projects/{project_id}", rt.handleGetProject)
	r.Post("/v1/projects/{project_id}/retry", rt.handleRetry)
	r.Post("/v1/projects/{project_id}/cancel", rt.handleCancel)
	r.Post("/v1/projects/{project_id}/fork", rt.handleFork)
	r.Post("/v1/projects/{project_id}/approve", rt.handleApproveOutline)
	r.Post("/v1/projects/{project_id}/reject", rt.handleRejectOutline)
	r.Post("/v1/projects/{project_id}/narration", rt.handleEditNarration)
	r.Delete("/v1/projects/{project_id}", rt.handleDeleteProject)
	r.Get("/v1/operations/{operation_id}", rt.handleGetOperation)
	r.Post("/v1/channel-assets/{kind}", rt.handleNormalizeChannelAsset)
	r.Get("/v1/channel-assets/preview", rt.handleChannelAssetPreview)
	r.Get("/v1/projects/{project_id}/qc-report", rt.handleQCReport)
	r.Post("/v1/projects/{project_id}/clips", rt.handleCreateClip)
	r.Get("/v1/projects/{project_id}/clips", rt.handleListClips)
	// CR-025: prompt wording moved to the DB. Public read (web-gui's wizard
	// fetches the current template at runtime); admin list/update (the
	// PromptSettingsPage editor). No auth guard exists on this router today —
	// same "add plainly, don't invent auth" posture the plan called for; see
	// the router_test.go note and the final report's followup item.
	// CR-027 FR77.2 — the prompt with every {{variable}} already filled in.
	// CR-027 FR78/FR79 — run a step with the API, and tell the GUI whether
	// that option exists at all before it draws the button.
	r.Get("/v1/projects/{project_id}/errors", rt.handleListProjectErrors)
	r.Get("/v1/projects/{project_id}/events", rt.handleListProjectEvents)
	r.Get("/v1/events", rt.handleListRecentEvents)
	// CR-027 FR79 — the step-1 working mode, remembered per project.
	r.Put("/v1/projects/{project_id}/settings", rt.handleSaveWizardSettings)
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
		RenderEngine:          domain.RenderEngine(req.RenderEngine),
		BackgroundMusicVolume: req.BackgroundMusicVolume,
		VideoFont:             req.VideoFont,
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

// handleCreateProjectDraft backs CR-028 FR83.1 — POST /v1/projects, called
// by wizard step 1 as soon as the Creator finishes typing a topic. Replaces
// the client-generated UUID ProjectDraftContext used to keep locally.
func (rt *Router) handleCreateProjectDraft(w http.ResponseWriter, r *http.Request) {
	if rt.createProjectDraft == nil {
		writeError(w, http.StatusNotFound, "project drafts are not enabled")
		return
	}
	var req createProjectDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	lang := domain.ContentLanguage(req.ContentLanguage)
	if lang != domain.LanguageVietnamese && lang != domain.LanguageEnglish {
		writeError(w, http.StatusBadRequest, "content_language must be 'vi' or 'en'")
		return
	}
	if req.RenderEngine != "" && !domain.RenderEngine(req.RenderEngine).IsValid() {
		writeError(w, http.StatusBadRequest, "render_engine must be 'manim' or 'remotion'")
		return
	}
	out, err := rt.createProjectDraft.Execute(r.Context(), application.CreateProjectDraftInput{
		ProjectID:       req.ProjectID,
		Topic:           req.Topic,
		ContentLanguage: lang,
		RenderEngine:    domain.RenderEngine(req.RenderEngine),
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, createProjectDraftResponse{
		ProjectID:       out.ProjectID,
		SimilarProjects: toSimilarProjectsResponse(out.SimilarProjects),
	})
}

// handleUpdateProjectTopic backs CR-028 FR83.2 — PATCH
// /v1/projects/{project_id}/topic, called when the Creator returns to step 1
// and edits the topic of a draft they already created. 409s once render has
// started (FR84.2 — same lock as the authoring saves).
func (rt *Router) handleUpdateProjectTopic(w http.ResponseWriter, r *http.Request) {
	if rt.updateProjectTopic == nil {
		writeError(w, http.StatusNotFound, "project drafts are not enabled")
		return
	}
	projectID := chi.URLParam(r, "project_id")
	var req updateProjectTopicRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	out, err := rt.updateProjectTopic.Execute(r.Context(), application.UpdateProjectTopicInput{
		ProjectID: projectID,
		Topic:     req.Topic,
	})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updateProjectTopicResponse{SimilarProjects: toSimilarProjectsResponse(out.SimilarProjects)})
}

func (rt *Router) handleGetProject(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	project, err := rt.projects.Get(r.Context(), projectID)
	if err != nil {
		writeUseCaseError(w, err)
		return
	}
	resp := toProjectResponse(project)
	resp.FlowStep, resp.RunState = rt.flowFor(r.Context(), project)
	if fr, ok := rt.projects.(forkedFromReader); ok {
		resp.ForkedFrom, _ = fr.ForkedFrom(r.Context(), projectID)
	}
	writeJSON(w, http.StatusOK, resp)
}

// flowFor places a project in the 13-step flow. Only a draft on the script
// step needs the authoring content to tell 1a from 1b from 1c.
func (rt *Router) flowFor(ctx context.Context, p *domain.Project) (int, string) {
	var content domain.AuthoredContent
	if p.Status == domain.StatusDraft && p.WizardStep >= domain.WizardStepScript && rt.authored != nil {
		if sums, err := rt.authored.Summaries(ctx, []string{p.ProjectID}); err == nil {
			if st, ok := sums[p.ProjectID]; ok {
				content = domain.AuthoredContent{Story: st.Story, Storyboard: st.Storyboard, Code: st.Code}
			}
		}
	}
	fs := domain.FlowStateFor(p.Status, domain.EffectiveWizardStep(p), content)
	return fs.Step, string(domain.RunStateOf(p.Status, p.ErrorMessage))
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
	// CR-040 FR114.2: the delete saga answers 202 — the project disappears once
	// every service has removed its own files.
	if rt.deleteProject != nil {
		if err := rt.deleteProject.Execute(r.Context(), projectID); err != nil {
			writeUseCaseError(w, err)
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "deleting"})
		return
	}
	if err := rt.projects.Delete(r.Context(), projectID); err != nil {
		writeUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// deleteProgressReader is the optional progress side of the delete saga.
type deleteProgressReader interface {
	Progress(ctx context.Context, projectID string) (application.DeleteProgress, error)
}

// handleGetOperation serves GET /v1/operations/{id} (CR-040 FR116.3) for the
// operations the orchestrator owns: `delete:<project_id>` is the delete saga,
// reported as done/total purge owners. Anything else is authoring-service's.
func (rt *Router) handleGetOperation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "operation_id")
	projectID, ok := strings.CutPrefix(id, "delete:")
	reader, hasReader := rt.deleteProject.(deleteProgressReader)
	if !ok || !hasReader {
		writeError(w, http.StatusNotFound, "operation not found")
		return
	}
	p, err := reader.Progress(r.Context(), projectID)
	if err != nil {
		writeUseCaseError(w, err)
		return
	}
	status := "running"
	switch {
	case p.Gone:
		status = "succeeded"
	case p.Failed != "":
		status = "failed"
	}
	body := map[string]any{
		"kind": "delete_project", "phase": "purge", "reasoning_chars": 0, "content_chars": 0,
		"done": p.Done, "total": p.Total, "elapsed_ms": 0, "status": status, "error": nil,
	}
	if p.Failed != "" {
		body["error"] = p.Failed
	}
	writeJSON(w, http.StatusOK, body)
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
	case errors.Is(err, domain.ErrInvalidWizardInput):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrInvalidStatus), errors.Is(err, domain.ErrProjectBusy):
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

// handleSaveWizardSettings persists wizard step 2 when the Creator presses
// "Tiếp tục". PUT: it replaces the whole settings set and is safe to repeat.
// 409 once the render has started, the same lock the authoring saves use.
func (rt *Router) handleSaveWizardSettings(w http.ResponseWriter, r *http.Request) {
	if rt.saveWizardSettings == nil {
		writeError(w, http.StatusNotFound, "wizard settings is not enabled")
		return
	}
	var req saveWizardSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	ttsEnabled := true
	if req.TTSEnabled != nil {
		ttsEnabled = *req.TTSEnabled
	}
	err := rt.saveWizardSettings.Execute(r.Context(), chi.URLParam(r, "project_id"), domain.WizardSettings{
		ContentLanguage:       domain.ContentLanguage(req.ContentLanguage),
		RenderEngine:          domain.RenderEngine(req.RenderEngine),
		TTSEnabled:            ttsEnabled,
		VoiceID:               req.VoiceID,
		SubtitleMode:          domain.SubtitleMode(req.SubtitleMode),
		SubtitleStyle:         req.SubtitleStyle,
		RenderQuality:         domain.RenderQuality(req.RenderQuality),
		VideoFormatID:         req.VideoFormatID,
		VideoOutputMode:       domain.VideoOutputMode(req.VideoOutputMode),
		BackgroundMusicPath:   req.BackgroundMusicPath,
		BackgroundMusicVolume: req.BackgroundMusicVolume,
		VideoFont:             req.VideoFont,
	})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// --- internal API for authoring-service (CR-040 FR111) ----------------------

// handleInternalGetProject returns the whole Project — the settings, script and
// chapters the prompt renderer and the metadata suggester read.
func (rt *Router) handleInternalGetProject(w http.ResponseWriter, r *http.Request) {
	p, err := rt.projects.Get(r.Context(), chi.URLParam(r, "project_id"))
	if err != nil {
		writeUseCaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (rt *Router) handleInternalProjectStatus(w http.ResponseWriter, r *http.Request) {
	status, err := rt.projects.GetStatus(r.Context(), chi.URLParam(r, "project_id"))
	if err != nil {
		writeUseCaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": string(status)})
}

func (rt *Router) handleInternalGetFormat(w http.ResponseWriter, r *http.Request) {
	version, _ := strconv.Atoi(r.URL.Query().Get("version"))
	f, err := rt.projects.GetVideoFormat(r.Context(), chi.URLParam(r, "format_id"), version)
	if err != nil {
		writeUseCaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, f)
}

func (rt *Router) handleInternalVoiceCalibration(w http.ResponseWriter, r *http.Request) {
	c, err := rt.projects.GetVoiceCalibration(r.Context(), chi.URLParam(r, "voice_id"))
	if err != nil {
		writeUseCaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (rt *Router) handleInternalAppendError(w http.ResponseWriter, r *http.Request) {
	if rt.projectErrors == nil {
		writeError(w, http.StatusNotFound, "project error log is not enabled")
		return
	}
	var e application.ProjectError
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := rt.projectErrors.AppendProjectError(r.Context(), chi.URLParam(r, "project_id"), e); err != nil {
		writeUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *Router) handleInternalAppendEvent(w http.ResponseWriter, r *http.Request) {
	if rt.projectEvents == nil {
		writeError(w, http.StatusNotFound, "project journal is not enabled")
		return
	}
	var e domain.ProjectEvent
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	e.ProjectID = chi.URLParam(r, "project_id")
	if err := rt.projectEvents.AppendProjectEvent(r.Context(), e); err != nil {
		writeUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
