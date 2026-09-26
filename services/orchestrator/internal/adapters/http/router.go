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

// promptsUseCase backs the prompt library (CR-031): a list of prompts per
// role, one active. System rows are read-only; Creator rows are editable.
type promptsUseCase interface {
	Active(ctx context.Context, role domain.PromptRole) (domain.Prompt, error)
	List(ctx context.Context, role domain.PromptRole) ([]domain.Prompt, error)
	Create(ctx context.Context, role domain.PromptRole, name, templateText string) (domain.Prompt, error)
	Copy(ctx context.Context, id string) (domain.Prompt, error)
	Update(ctx context.Context, id, name, templateText string) (domain.Prompt, error)
	Activate(ctx context.Context, id string) (domain.Prompt, error)
	Delete(ctx context.Context, id string) error
}

// renderPromptUseCase backs CR-027 FR77's GET
// /v1/projects/{id}/prompts/{role} — the prompt fully substituted, so the
// Copy button and the server's own generate call use the same text.
type renderPromptUseCase interface {
	Execute(ctx context.Context, projectID string, role domain.PromptRole) (application.RenderedPrompt, error)
}

// saveAuthoringModeUseCase backs CR-027 FR79's PUT
// /v1/projects/{id}/authoring/mode — the step-1 working mode, stored per
// project so it survives a reload, another browser and a restart.
type saveAuthoringModeUseCase interface {
	Execute(ctx context.Context, projectID, mode string) error
}

// saveAuthoringModelsUseCase backs PUT /v1/projects/{id}/authoring/models —
// the model-per-step picker's choice for the three authoring tabs.
type saveAuthoringModelsUseCase interface {
	Execute(ctx context.Context, projectID string, models domain.AuthoringStepModels) error
}

// generateAuthoringUseCase backs CR-027 FR78's POST
// /v1/projects/{id}/authoring/{step}/generate — the second way to do a step,
// beside the Copy-prompt round trip, which stays exactly as it was (FR77.4).
type generateAuthoringUseCase interface {
	Execute(ctx context.Context, projectID, step string) (application.GeneratedStep, error)
	Available() bool
	Provider() string
}

// authoringProgressReader is the optional live-progress side of the generate
// use case; a use case without it simply has no progress endpoint.
type authoringProgressReader interface {
	Progress(projectID, step string) application.AuthoringProgress
}

// saveAuthoringStoryUseCase backs CR-025 step 1's POST
// /v1/projects/{id}/authoring/story.
type saveAuthoringStoryUseCase interface {
	Execute(ctx context.Context, projectID, content, topic string) error
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

// getAuthoringStateUseCase backs GET /v1/projects/{id}/authoring, letting the
// wizard rehydrate saved story/storyboard/code on reload/back-navigation.
type getAuthoringStateUseCase interface {
	Execute(ctx context.Context, projectID string) (application.AuthoringState, error)
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

// saveWizardPositionUseCase backs PUT /v1/projects/{id}/wizard-position.
type saveWizardPositionUseCase interface {
	SaveWizardPosition(ctx context.Context, projectID string, step int, route string) error
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
	prompts                 promptsUseCase
	renderPrompt            renderPromptUseCase
	generateAuthoring       generateAuthoringUseCase
	projectErrors           application.ProjectErrorLogPort
	defaultModel            string
	saveWizardPosition      saveWizardPositionUseCase
	saveAuthoringMode       saveAuthoringModeUseCase
	saveAuthoringModels     saveAuthoringModelsUseCase
	saveWizardSettings      saveWizardSettingsUseCase
	saveAuthoringStory      saveAuthoringStoryUseCase
	saveAuthoringStoryboard saveAuthoringStoryboardUseCase
	saveAuthoringCode       saveAuthoringCodeUseCase
	getAuthoringState       getAuthoringStateUseCase
	createProjectDraft      createProjectDraftUseCase
	updateProjectTopic      updateProjectTopicUseCase
}

// WithRenderPrompt enables CR-027 FR77's server-side prompt rendering.
func (rt *Router) WithRenderPrompt(renderPrompt renderPromptUseCase) *Router {
	rt.renderPrompt = renderPrompt
	return rt
}

// WithAuthoringMode enables CR-027 FR79's persisted step-1 working mode.
func (rt *Router) WithAuthoringMode(saveAuthoringMode saveAuthoringModeUseCase) *Router {
	rt.saveAuthoringMode = saveAuthoringMode
	return rt
}

// WithAuthoringModels enables the model-per-step picker's PUT
// /v1/projects/{id}/authoring/models.
func (rt *Router) WithAuthoringModels(saveAuthoringModels saveAuthoringModelsUseCase) *Router {
	rt.saveAuthoringModels = saveAuthoringModels
	return rt
}

// WithGenerateAuthoring enables CR-027 FR78's run-a-step-with-AI endpoint.
// Left unwired (no API key), the route answers 404 and GET /v1/llm/status
// reports disabled, so the GUI hides the button and says why instead of
// offering one that fails on the first press (FR79.4).
// WithDefaultModel tells GET /v1/llm/status which concrete model an empty
// ("server default") choice resolves to, so the GUI can name it.
func (rt *Router) WithDefaultModel(id string) *Router {
	rt.defaultModel = id
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

func (rt *Router) WithGenerateAuthoring(generateAuthoring generateAuthoringUseCase) *Router {
	rt.generateAuthoring = generateAuthoring
	return rt
}

// WithPrompts attaches the CR-031 prompt library, enabling GET
// /v1/prompts/{role} and the /v1/admin/prompts routes. Without it the routes
// answer 404, the same "unwired means absent" posture as WithQCReports.
func (rt *Router) WithPrompts(prompts promptsUseCase) *Router {
	rt.prompts = prompts
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

// WithAuthoringState attaches CR-025's read-side use case, enabling
// GET /v1/projects/{project_id}/authoring.
func (rt *Router) WithAuthoringState(getAuthoringState getAuthoringStateUseCase) *Router {
	rt.getAuthoringState = getAuthoringState
	return rt
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

// WithWizardPosition enables PUT /v1/projects/{project_id}/wizard-position.
func (rt *Router) WithWizardPosition(save saveWizardPositionUseCase) *Router {
	rt.saveWizardPosition = save
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
	// CR-028 FR83/FR84/FR85: the project row now exists from wizard step 1.
	r.Post("/v1/projects", rt.handleCreateProjectDraft)
	r.Patch("/v1/projects/{project_id}/topic", rt.handleUpdateProjectTopic)
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
	r.Get("/v1/prompts/{role}", rt.handleGetActivePrompt)
	r.Get("/v1/admin/prompts", rt.handleListPrompts)
	r.Post("/v1/admin/prompts", rt.handleCreatePrompt)
	r.Post("/v1/admin/prompts/{id}/copy", rt.handleCopyPrompt)
	r.Put("/v1/admin/prompts/{id}", rt.handleUpdatePrompt)
	r.Post("/v1/admin/prompts/{id}/activate", rt.handleActivatePrompt)
	r.Delete("/v1/admin/prompts/{id}", rt.handleDeletePrompt)
	r.Post("/v1/projects/{project_id}/authoring/story", rt.handleSaveAuthoringStory)
	r.Post("/v1/projects/{project_id}/authoring/storyboard", rt.handleSaveAuthoringStoryboard)
	r.Post("/v1/projects/{project_id}/authoring/code", rt.handleSaveAuthoringCode)
	r.Get("/v1/projects/{project_id}/authoring", rt.handleGetAuthoringState)
	// CR-027 FR77.2 — the prompt with every {{variable}} already filled in.
	r.Get("/v1/projects/{project_id}/prompts/{role}", rt.handleRenderPrompt)
	// CR-027 FR78/FR79 — run a step with the API, and tell the GUI whether
	// that option exists at all before it draws the button.
	r.Post("/v1/projects/{project_id}/authoring/{step}/generate", rt.handleGenerateAuthoring)
	r.Get("/v1/projects/{project_id}/errors", rt.handleListProjectErrors)
	r.Get("/v1/llm/status", rt.handleLLMStatus)
	// CR-027 FR79 — the step-1 working mode, remembered per project.
	r.Put("/v1/projects/{project_id}/authoring/mode", rt.handleSaveAuthoringMode)
	r.Put("/v1/projects/{project_id}/authoring/models", rt.handleSaveAuthoringModels)
	r.Put("/v1/projects/{project_id}/settings", rt.handleSaveWizardSettings)
	r.Get("/v1/projects/{project_id}/authoring/{step}/progress", rt.handleAuthoringProgress)
	r.Put("/v1/projects/{project_id}/wizard-position", rt.handleSaveWizardPosition)
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
	case errors.Is(err, domain.ErrInvalidWizardInput):
		writeError(w, http.StatusBadRequest, err.Error())
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

// promptError maps the library's failures to HTTP: a missing row is 404, an
// attempt to change a system row is 403, anything else the caller sent wrong
// is 400.
func promptError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, application.ErrPromptNotFound):
		writeError(w, http.StatusNotFound, "prompt not found")
	case errors.Is(err, application.ErrPromptReadOnly):
		writeError(w, http.StatusForbidden, "system prompts are read-only — copy it to make one you can edit")
	default:
		writeError(w, http.StatusBadRequest, err.Error())
	}
}

// handleGetActivePrompt serves the wording a role currently runs on — what the
// wizard reads. Shape kept as the pre-CR-031 response (template_text) with the
// library facts added alongside.
func (rt *Router) handleGetActivePrompt(w http.ResponseWriter, r *http.Request) {
	if rt.prompts == nil {
		writeError(w, http.StatusNotFound, "prompts are not enabled")
		return
	}
	role := chi.URLParam(r, "role")
	if !domain.ValidPromptRole(role) {
		writeError(w, http.StatusBadRequest, "unknown role")
		return
	}
	p, err := rt.prompts.Active(r.Context(), domain.PromptRole(role))
	if err != nil {
		promptError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// handleListPrompts serves the library for the admin screen, optionally one
// role via ?role=.
func (rt *Router) handleListPrompts(w http.ResponseWriter, r *http.Request) {
	if rt.prompts == nil {
		writeError(w, http.StatusNotFound, "prompts are not enabled")
		return
	}
	prompts, err := rt.prompts.List(r.Context(), domain.PromptRole(r.URL.Query().Get("role")))
	if err != nil {
		promptError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"prompts": prompts})
}

type promptBody struct {
	Role         string `json:"role"`
	Name         string `json:"name"`
	TemplateText string `json:"template_text"`
}

func decodePromptBody(w http.ResponseWriter, r *http.Request) (promptBody, bool) {
	var req promptBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return req, false
	}
	return req, true
}

// handleCreatePrompt adds a Creator-owned prompt to a role's list.
func (rt *Router) handleCreatePrompt(w http.ResponseWriter, r *http.Request) {
	if rt.prompts == nil {
		writeError(w, http.StatusNotFound, "prompts are not enabled")
		return
	}
	req, ok := decodePromptBody(w, r)
	if !ok {
		return
	}
	p, err := rt.prompts.Create(r.Context(), domain.PromptRole(req.Role), req.Name, req.TemplateText)
	if err != nil {
		promptError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

// handleCopyPrompt duplicates a row into a new Creator-owned one — the only
// way to start from a system prompt.
func (rt *Router) handleCopyPrompt(w http.ResponseWriter, r *http.Request) {
	if rt.prompts == nil {
		writeError(w, http.StatusNotFound, "prompts are not enabled")
		return
	}
	p, err := rt.prompts.Copy(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		promptError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

// handleUpdatePrompt edits a Creator-owned prompt; a system one answers 403.
func (rt *Router) handleUpdatePrompt(w http.ResponseWriter, r *http.Request) {
	if rt.prompts == nil {
		writeError(w, http.StatusNotFound, "prompts are not enabled")
		return
	}
	req, ok := decodePromptBody(w, r)
	if !ok {
		return
	}
	p, err := rt.prompts.Update(r.Context(), chi.URLParam(r, "id"), req.Name, req.TemplateText)
	if err != nil {
		promptError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// handleActivatePrompt makes a row the one its role runs on.
func (rt *Router) handleActivatePrompt(w http.ResponseWriter, r *http.Request) {
	if rt.prompts == nil {
		writeError(w, http.StatusNotFound, "prompts are not enabled")
		return
	}
	p, err := rt.prompts.Activate(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		promptError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// handleDeletePrompt removes a Creator-owned prompt; a system one answers 403.
func (rt *Router) handleDeletePrompt(w http.ResponseWriter, r *http.Request) {
	if rt.prompts == nil {
		writeError(w, http.StatusNotFound, "prompts are not enabled")
		return
	}
	if err := rt.prompts.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		promptError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
		// CR-027 D0 — optional: an empty topic leaves the stored one alone
		// (see PromptTemplateRepository.SaveAuthoringStory), so a browser
		// running pre-CR-027 JavaScript keeps working unchanged.
		Topic string `json:"topic"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if err := rt.saveAuthoringStory.Execute(r.Context(), projectID, req.Content, req.Topic); err != nil {
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

// handleGetAuthoringState serves every saved authoring output (story,
// storyboard, code) so the wizard can rehydrate on reload/
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
		// CR-027 FR79 — how the Creator is working step 1, so the wizard
		// restores the choice on reload or on another machine instead of
		// falling back to copy-and-paste. Always "manual" or "ai".
		"mode":       state.Mode,
		"topic":      state.Topic,
		"story":      state.Story,
		"storyboard": state.Storyboard,
		"code":       state.Code,
		// Model-per-step picker's saved choice for each tab, "" meaning
		// "server default" — same rehydrate-on-reload reasoning as mode.
		"story_model":      state.Models.Story,
		"storyboard_model": state.Models.Storyboard,
		"code_model":       state.Models.Code,
	})
}

// handleSaveAuthoringModels stores the model-per-step picker's choice for
// the three authoring tabs — the follow-up to CR-027 FR79 that lets each
// step call a different Hive model instead of the one HIVE_MODEL hardcodes.
//
// PUT, not POST, for the same reason as handleSaveAuthoringMode: it replaces
// the triple, and the GUI writes it on every change.
func (rt *Router) handleSaveAuthoringModels(w http.ResponseWriter, r *http.Request) {
	if rt.saveAuthoringModels == nil {
		writeError(w, http.StatusNotFound, "authoring models is not enabled")
		return
	}
	projectID := chi.URLParam(r, "project_id")

	var req saveAuthoringModelsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	models := domain.AuthoringStepModels{Story: req.Story, Storyboard: req.Storyboard, Code: req.Code}
	if err := rt.saveAuthoringModels.Execute(r.Context(), projectID, models); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

// handleAuthoringProgress reports how far a running generate call has got.
func (rt *Router) handleAuthoringProgress(w http.ResponseWriter, r *http.Request) {
	pr, ok := rt.generateAuthoring.(authoringProgressReader)
	if !ok {
		writeError(w, http.StatusNotFound, "progress is not available")
		return
	}
	writeJSON(w, http.StatusOK, pr.Progress(chi.URLParam(r, "project_id"), chi.URLParam(r, "step")))
}

// handleSaveWizardPosition remembers which wizard screen a draft was left on,
// so reopening the project returns to it. Idempotent: the GUI sends it on
// every screen change.
func (rt *Router) handleSaveWizardPosition(w http.ResponseWriter, r *http.Request) {
	if rt.saveWizardPosition == nil {
		writeError(w, http.StatusNotFound, "wizard position is not enabled")
		return
	}
	var req struct {
		Route string `json:"route"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	step, ok := domain.WizardStepForRoute(req.Route)
	if !ok {
		writeError(w, http.StatusBadRequest, "unknown wizard route")
		return
	}
	if err := rt.saveWizardPosition.SaveWizardPosition(r.Context(), chi.URLParam(r, "project_id"), step, req.Route); err != nil {
		writeUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleSaveAuthoringMode stores how the Creator works step 1 (CR-027 FR79).
//
// PUT, not POST: it replaces one value, and sending it twice must mean the
// same as sending it once — the GUI writes it on every toggle.
func (rt *Router) handleSaveAuthoringMode(w http.ResponseWriter, r *http.Request) {
	if rt.saveAuthoringMode == nil {
		writeError(w, http.StatusNotFound, "authoring mode is not enabled")
		return
	}
	projectID := chi.URLParam(r, "project_id")

	var req saveAuthoringModeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := rt.saveAuthoringMode.Execute(r.Context(), projectID, req.Mode); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleRenderPrompt serves one role's prompt with every {{variable}}
// already substituted (CR-027 FR77.2).
//
// web-gui's Copy button moves to this endpoint. Before, the browser fetched
// the raw template and did the substitution itself — which is why the server
// could not produce a prompt at all, and why the two paths could have drifted
// once the server started producing them too.
func (rt *Router) handleRenderPrompt(w http.ResponseWriter, r *http.Request) {
	if rt.renderPrompt == nil {
		writeError(w, http.StatusNotFound, "prompt rendering is not enabled")
		return
	}
	projectID := chi.URLParam(r, "project_id")
	role := chi.URLParam(r, "role")
	if !domain.ValidPromptRole(role) {
		writeError(w, http.StatusBadRequest, "unknown role")
		return
	}

	rendered, err := rt.renderPrompt.Execute(r.Context(), projectID, domain.PromptRole(role))
	if err != nil {
		if errors.Is(err, domain.ErrProjectNotFound) {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rendered)
}

// llmStatusResponse tells the GUI whether the "Chạy bằng AI" button has
// anything to call (CR-027 FR79.4).
type llmStatusResponse struct {
	Enabled  bool   `json:"enabled"`
	Provider string `json:"provider"`
	// Reason is filled only when Enabled is false, in Vietnamese, pointing at
	// the file the Creator has to edit — a disabled button that does not say
	// why is a bug report waiting to happen.
	Reason string `json:"reason,omitempty"`
	// Models is the model-per-step picker's catalog (domain.
	// AuthoringModelCatalog) — served here rather than hardcoded a second
	// time in web-gui, so the GUI's dropdown and the server's own validation
	// (SaveAuthoringModelsUseCase) can never drift apart. Empty when the AI
	// path itself is unavailable — nothing to pick a model for.
	Models []domain.AuthoringModelOption `json:"models,omitempty"`
	// DefaultModel is the concrete model id an empty choice resolves to.
	DefaultModel string `json:"default_model,omitempty"`
}

func (rt *Router) handleLLMStatus(w http.ResponseWriter, r *http.Request) {
	if rt.generateAuthoring == nil || !rt.generateAuthoring.Available() {
		writeJSON(w, http.StatusOK, llmStatusResponse{
			Enabled: false,
			Reason:  "Chưa cấu hình HIVE_API_KEY cho llm-service (hoặc llm-service không chạy) — dùng nút Copy prompt như cũ.",
		})
		return
	}
	writeJSON(w, http.StatusOK, llmStatusResponse{
		Enabled: true, Provider: rt.generateAuthoring.Provider(),
		Models: domain.AuthoringModelCatalog, DefaultModel: rt.defaultModel,
	})
}

// handleGenerateAuthoring runs one authoring step through the configured
// provider (CR-027 FR78.1).
//
// It is an addition, not a replacement: GET .../prompts/{role} still serves
// the same text for the Copy button, and every failure below names the
// copy-out path as the way through.
func (rt *Router) handleGenerateAuthoring(w http.ResponseWriter, r *http.Request) {
	if rt.generateAuthoring == nil {
		writeError(w, http.StatusNotFound, "chạy bằng AI chưa được bật trên máy chủ này")
		return
	}
	projectID := chi.URLParam(r, "project_id")
	step := chi.URLParam(r, "step")

	result, err := rt.generateAuthoring.Execute(r.Context(), projectID, step)
	if err != nil {
		writeGenerateError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// errorCause is the underlying reason of an LLMError without its
// "provider: kind:" prefix, which the Creator has no use for.
func errorCause(err error) string {
	var llmErr *application.LLMError
	if errors.As(err, &llmErr) && llmErr.Err != nil {
		return llmErr.Err.Error()
	}
	return err.Error()
}

// writeGenerateError maps a failed run onto a status code and a Vietnamese
// sentence that says what to do about it (FR76.6/FR79.3). "AI failed" would
// send a Creator with an empty Hive balance to go rewrite their prompt.
func writeGenerateError(w http.ResponseWriter, err error) {
	const fallback = " Hoặc dùng nút Copy prompt như cũ."

	switch {
	case errors.Is(err, application.ErrGenerateBusy):
		writeError(w, http.StatusConflict, "Một lượt chạy AI cho bước này đang diễn ra, chờ nó xong đã.")
		return
	case errors.Is(err, application.ErrLLMNotConfigured):
		writeError(w, http.StatusServiceUnavailable,
			"Chưa cấu hình HIVE_API_KEY trong .env (llm-service) nên không gọi được AI."+fallback)
		return
	case errors.Is(err, domain.ErrProjectNotFound):
		writeError(w, http.StatusNotFound, "project not found")
		return
	}

	status := http.StatusBadGateway
	var message string
	switch application.LLMErrorKindOf(err) {
	case application.ErrKindAuth:
		status = http.StatusBadGateway
		message = "API key bị từ chối — kiểm tra lại HIVE_API_KEY trong .env (llm-service)."
	case application.ErrKindBalance:
		message = "Tài khoản Hive hết số dư — nạp thêm ở dashboard Hive."
	case application.ErrKindRateLimit:
		status = http.StatusTooManyRequests
		message = "Hive đang chặn vì gọi quá nhanh — chờ một lát rồi thử lại."
	case application.ErrKindTimeout:
		status = http.StatusGatewayTimeout
		message = "AI không trả lời trong thời gian cho phép — thử lại, hoặc tăng HIVE_TIMEOUT_SECONDS / LLM_SERVICE_TIMEOUT_SECONDS."
	case application.ErrKindBudget, application.ErrKindTruncated:
		message = "Câu trả lời bị cắt vì hết hạn mức token — tăng HIVE_MAX_OUTPUT_TOKENS rồi chạy lại. Chi tiết API trả về được lưu ở nhật ký lỗi của project."
	case application.ErrKindEmpty:
		message = "AI trả về rỗng — thử chạy lại, hoặc sửa lời prompt ở trang Prompt."
	case application.ErrKindMalformed:
		// CR-039: llm-service says what was wrong (an unusable storyboard, code
		// the model could not get into shape) — that is what the Creator needs.
		message = "Kết quả AI không dùng được: " + errorCause(err)
	case application.ErrKindServer:
		var llmErr *application.LLMError
		if errors.As(err, &llmErr) && llmErr.Provider == "llm-service" {
			message = "Không gọi được llm-service (" + errorCause(err) + ") — kiểm tra container llm-service."
		} else {
			message = "Nhà cung cấp AI đang lỗi phía họ — thử lại sau."
		}
	default:
		// Not a provider failure: a bad step name, a locked project, a failed
		// save. Those already carry their own message.
		// Not a provider failure: a bad step name, a locked project, a
		// prompt over the input cap. Those already carry their own message.
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeError(w, status, message+fallback)
}
