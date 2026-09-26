package http

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"authoring/internal/application"
	"authoring/internal/domain"
)

// InternalAuthoring is what the orchestrator's internal calls need beyond the
// public use cases: the raw authoring state with no draft lock (the copy made
// by a fork is brand new), the topic-collision search, the list summaries and
// the cleanup when a project is deleted (CR-040 FR111).
type InternalAuthoring interface {
	SaveAuthoringTopic(ctx context.Context, projectID, topic string, language domain.ContentLanguage) error
	SaveAuthoringStory(ctx context.Context, projectID, content, topic string) error
	SaveAuthoringStoryboard(ctx context.Context, projectID, content string) error
	SaveAuthoringMode(ctx context.Context, projectID, mode string) error
	SaveAuthoringModels(ctx context.Context, projectID string, models domain.AuthoringStepModels) error
	FindSimilarTopics(ctx context.Context, language domain.ContentLanguage, normalizedTopic, excludeProjectID string) ([]application.SimilarProject, error)
	Summaries(ctx context.Context, projectIDs []string) (map[string]application.AuthoringSummary, error)
	DeleteAuthoring(ctx context.Context, projectID string) error
}

// Router holds authoring-service's REST handlers' use case dependencies.
type Router struct {
	suggestPublishMetadata  suggestPublishMetadataUseCase
	suggestShortScript      suggestShortScriptUseCase
	prompts                 promptsUseCase
	renderPrompt            renderPromptUseCase
	generateAuthoring       generateAuthoringUseCase
	authoringChain          authoringChainUseCase
	defaultModel            string
	saveAuthoringMode       saveAuthoringModeUseCase
	saveAuthoringModels     saveAuthoringModelsUseCase
	saveAuthoringStory      saveAuthoringStoryUseCase
	saveAuthoringStoryboard saveAuthoringStoryboardUseCase
	saveAuthoringCode       saveAuthoringCodeUseCase
	getAuthoringState       getAuthoringStateUseCase
	internal                InternalAuthoring
	operations              *application.Operations
}

// NewRouter constructs the router; every capability is attached with a With* method.
func NewRouter(suggestPublishMetadata suggestPublishMetadataUseCase, internal InternalAuthoring) *Router {
	return &Router{suggestPublishMetadata: suggestPublishMetadata, internal: internal}
}

// Handler builds the chi.Router with all routes (health + REST endpoints).
func (rt *Router) Handler() http.Handler {
	r := chi.NewRouter()
	r.Get("/health", rt.handleHealth)
	r.Get("/v1/operations/{operation_id}", rt.handleGetOperation)
	r.Post("/v1/projects/{project_id}/suggest-metadata", rt.handleSuggestMetadata)
	r.Post("/v1/short-script-suggestions", rt.handleSuggestShortScript)
	// CR-025: prompt wording lives in the DB. Public read (the wizard fetches the
	// current template at runtime); admin list/update (the PromptSettingsPage editor).
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
	// CR-027 FR78/FR79 — run a step with the API, and tell the GUI whether that
	// option exists at all before it draws the button.
	r.Post("/v1/projects/{project_id}/authoring/{step}/generate", rt.handleGenerateAuthoring)
	r.Post("/v1/projects/{project_id}/authoring/chain", rt.handleStartAuthoringChain)
	r.Get("/v1/projects/{project_id}/authoring/chain", rt.handleGetAuthoringChain)
	r.Get("/v1/llm/status", rt.handleLLMStatus)
	r.Put("/v1/projects/{project_id}/authoring/mode", rt.handleSaveAuthoringMode)
	r.Put("/v1/projects/{project_id}/authoring/models", rt.handleSaveAuthoringModels)
	r.Get("/v1/projects/{project_id}/authoring/{step}/progress", rt.handleAuthoringProgress)

	// Internal — called by the orchestrator only (never routed by the gateway).
	r.Get("/internal/v1/authoring/summaries", rt.handleInternalSummaries)
	r.Get("/internal/v1/authoring/similar", rt.handleInternalSimilar)
	r.Get("/internal/v1/authoring/{project_id}", rt.handleGetAuthoringState)
	r.Put("/internal/v1/authoring/{project_id}", rt.handleInternalPut)
	r.Delete("/internal/v1/authoring/{project_id}", rt.handleInternalDelete)
	return r
}

func writeUseCaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrProjectNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrInvalidWizardInput):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrInvalidStatus), errors.Is(err, domain.ErrProjectBusy):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

type internalPutRequest struct {
	Topic      *string                     `json:"topic"`
	Language   domain.ContentLanguage      `json:"language"`
	Story      *string                     `json:"story"`
	Storyboard *string                     `json:"storyboard"`
	Mode       *string                     `json:"mode"`
	Models     *domain.AuthoringStepModels `json:"models"`
}

// handleInternalPut writes whichever authoring fields are present, without the
// draft lock: a fork's copy is new, and a draft's topic is saved before any
// step exists (CR-040 FR111).
func (rt *Router) handleInternalPut(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	var req internalPutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	ctx := r.Context()
	var err error
	if req.Topic != nil {
		err = rt.internal.SaveAuthoringTopic(ctx, projectID, *req.Topic, req.Language)
	}
	if err == nil && req.Story != nil {
		err = rt.internal.SaveAuthoringStory(ctx, projectID, *req.Story, "")
	}
	if err == nil && req.Storyboard != nil {
		err = rt.internal.SaveAuthoringStoryboard(ctx, projectID, *req.Storyboard)
	}
	if err == nil && req.Mode != nil {
		err = rt.internal.SaveAuthoringMode(ctx, projectID, *req.Mode)
	}
	if err == nil && req.Models != nil {
		err = rt.internal.SaveAuthoringModels(ctx, projectID, *req.Models)
	}
	if err != nil {
		slog.Error("internal authoring put failed", "project_id", projectID, "error", err.Error())
		writeUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *Router) handleInternalDelete(w http.ResponseWriter, r *http.Request) {
	if err := rt.internal.DeleteAuthoring(r.Context(), chi.URLParam(r, "project_id")); err != nil {
		writeUseCaseError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *Router) handleInternalSimilar(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	out, err := rt.internal.FindSimilarTopics(r.Context(), domain.ContentLanguage(q.Get("language")), q.Get("topic"), q.Get("exclude"))
	if err != nil {
		writeUseCaseError(w, err)
		return
	}
	if out == nil {
		out = []application.SimilarProject{}
	}
	writeJSON(w, http.StatusOK, out)
}

func (rt *Router) handleInternalSummaries(w http.ResponseWriter, r *http.Request) {
	var ids []string
	for _, id := range strings.Split(r.URL.Query().Get("ids"), ",") {
		if id = strings.TrimSpace(id); id != "" {
			ids = append(ids, id)
		}
	}
	out, err := rt.internal.Summaries(r.Context(), ids)
	if err != nil {
		writeUseCaseError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
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

// authoringChainUseCase runs 1a/1b/1c in order on the server, so a run
// survives the browser closing (see application.AuthoringChainRunner).
type authoringChainUseCase interface {
	Start(projectID string, steps []string) error
	State(projectID string) (application.ChainState, bool)
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

// WithAuthoringChain enables POST/GET /v1/projects/{id}/authoring/chain.
func (rt *Router) WithAuthoringChain(chain authoringChainUseCase) *Router {
	rt.authoringChain = chain
	return rt
}

// handleStartAuthoringChain starts the chain and returns 202 at once; the GUI
// polls GET .../authoring/chain (and .../{step}/progress for the live phase).
func (rt *Router) handleStartAuthoringChain(w http.ResponseWriter, r *http.Request) {
	if rt.authoringChain == nil {
		writeError(w, http.StatusNotFound, "chạy bằng AI chưa được bật trên máy chủ này")
		return
	}
	var req struct {
		Steps []string `json:"steps"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	err := rt.authoringChain.Start(chi.URLParam(r, "project_id"), req.Steps)
	switch {
	case errors.Is(err, application.ErrChainBusy):
		writeError(w, http.StatusConflict, "Một lượt chạy AI cho dự án này đang diễn ra, chờ nó xong đã.")
	case errors.Is(err, application.ErrChainInvalid):
		writeError(w, http.StatusBadRequest, err.Error())
	case err != nil:
		writeError(w, http.StatusInternalServerError, "không bắt đầu được lượt chạy")
	default:
		w.WriteHeader(http.StatusAccepted)
	}
}

// handleGetAuthoringChain reports the running or last-finished chain. 200 with
// finished=false/running=false and no steps when none ever ran.
func (rt *Router) handleGetAuthoringChain(w http.ResponseWriter, r *http.Request) {
	if rt.authoringChain == nil {
		writeError(w, http.StatusNotFound, "chạy bằng AI chưa được bật trên máy chủ này")
		return
	}
	st, ok := rt.authoringChain.State(chi.URLParam(r, "project_id"))
	if !ok {
		st = application.ChainState{Steps: []string{}}
	}
	writeJSON(w, http.StatusOK, st)
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

// WithShortScriptSuggester attaches CR-026's short-script assistant,
// enabling POST /v1/short-script-suggestions. Without it the route answers
// 404 — same "unwired means absent, not broken" posture as WithQCReports.
func (rt *Router) WithShortScriptSuggester(suggestShortScript suggestShortScriptUseCase) *Router {
	rt.suggestShortScript = suggestShortScript
	return rt
}

// WithOperations attaches the registry behind GET /v1/operations/{id}
// (CR-040 FR116.3). Without it suggestions still work, just without a live
// progress card.
func (rt *Router) WithOperations(ops *application.Operations) *Router {
	rt.operations = ops
	return rt
}

// trackOperation registers the operation id the GUI sent in X-Operation-Id (so
// it can poll while this request is still open) and hands back a context that
// streams the model's progress into it. finish must be called with the
// outcome. A request without the header is simply not tracked.
func (rt *Router) trackOperation(r *http.Request, kind string) (context.Context, func(error)) {
	id := r.Header.Get("X-Operation-Id")
	if rt.operations == nil || id == "" {
		return r.Context(), func(error) {}
	}
	rt.operations.Start(id, kind)
	ctx := application.WithProgress(r.Context(), func(p application.ChatProgress) { rt.operations.Progress(id, p) })
	return ctx, func(err error) { rt.operations.Finish(id, err) }
}

func (rt *Router) handleGetOperation(w http.ResponseWriter, r *http.Request) {
	if rt.operations == nil {
		writeError(w, http.StatusNotFound, "operations are not available")
		return
	}
	op, ok := rt.operations.Get(chi.URLParam(r, "operation_id"))
	if !ok {
		writeError(w, http.StatusNotFound, "operation not found")
		return
	}
	writeJSON(w, http.StatusOK, op)
}

func (rt *Router) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func (rt *Router) handleSuggestMetadata(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "project_id")
	ctx, finish := rt.trackOperation(r, "suggest_metadata")
	out, err := rt.suggestPublishMetadata.Execute(ctx, projectID)
	finish(err)
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

	ctx, finish := rt.trackOperation(r, "suggest_short_script")
	script, err := rt.suggestShortScript.Execute(ctx, req.Topic, req.SourceScriptContent, lang)
	finish(err)
	if err != nil {
		slog.Error("suggest-short-script failed", "error", err.Error())
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, suggestShortScriptResponse{ScriptContent: script})
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

// handleAuthoringProgress reports how far a running generate call has got.
func (rt *Router) handleAuthoringProgress(w http.ResponseWriter, r *http.Request) {
	pr, ok := rt.generateAuthoring.(authoringProgressReader)
	if !ok {
		writeError(w, http.StatusNotFound, "progress is not available")
		return
	}
	writeJSON(w, http.StatusOK, pr.Progress(chi.URLParam(r, "project_id"), chi.URLParam(r, "step")))
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
	status, message := DescribeGenerateError(err)
	writeError(w, status, message)
}

// DescribeGenerateError is the status code and Creator-facing sentence for a
// failed run. Exported because the server-side chain runner reports the same
// sentence the HTTP path would have.
func DescribeGenerateError(err error) (int, string) {
	const fallback = " Hoặc dùng nút Copy prompt như cũ."

	switch {
	case errors.Is(err, application.ErrGenerateBusy):
		return http.StatusConflict, "Một lượt chạy AI cho bước này đang diễn ra, chờ nó xong đã."
	case errors.Is(err, application.ErrLLMNotConfigured):
		return http.StatusServiceUnavailable,
			"Chưa cấu hình HIVE_API_KEY trong .env (llm-service) nên không gọi được AI." + fallback
	case errors.Is(err, domain.ErrProjectNotFound):
		return http.StatusNotFound, "project not found"
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
		return http.StatusBadRequest, err.Error()
	}
	return status, message + fallback
}
