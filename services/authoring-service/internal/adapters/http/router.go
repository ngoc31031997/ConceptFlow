package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"authoring/internal/application"
	"authoring/internal/domain"
)

// InternalAuthoring is what the orchestrator's internal calls need beyond the
// public use cases: the raw authoring state with no draft lock (the copy made
// by a fork is brand new), the topic-collision search, the list summaries and
// the cleanup when a project is deleted.
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
	prompts                 promptsUseCase
	archetypes              archetypesUseCase
	visualLanguages         visualLanguagesUseCase
	illustrations           illustrationsUseCase
	projectIllustrations    projectIllustrationsUseCase
	renderPrompt            renderPromptUseCase
	llmStatus               llmStatusSource
	defaultModelSetting     defaultModelUseCase
	saveAuthoringMode       saveAuthoringModeUseCase
	saveAuthoringModels     saveAuthoringModelsUseCase
	saveAuthoringStory      saveAuthoringStoryUseCase
	saveAuthoringStoryboard saveAuthoringStoryboardUseCase
	saveAuthoringCode       saveAuthoringCodeUseCase
	getAuthoringState       getAuthoringStateUseCase
	internal                InternalAuthoring
	operations              *application.Operations
	reviewUnits             reviewUnitsUseCase
	idea                    ideaUseCase
	creativeBrief           creativeBriefUseCase
	projectBrief            projectBriefUseCase
	research                researchUseCase
	story                   storyUseCase
	beats                   beatUseCase
	script                  scriptUseCase
	scriptLock              scriptLockUseCase
	voiceLines              voiceLinesUseCase
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
	r.Post("/v1/operations/{operation_id}/cancel", rt.handleCancelOperation)
	r.Get("/v1/projects/{project_id}/operations", rt.handleListProjectOperations)
	r.Post("/v1/projects/{project_id}/suggest-metadata", rt.handleSuggestMetadata)
	// Prompt wording lives in the DB. Public read (the wizard fetches the
	// current template at runtime); admin list/update (the PromptSettingsPage editor).
	r.Get("/v1/video-archetypes", rt.handleListArchetypes)
	rt.visualLanguageRoutes(r)
	rt.illustrationRoutes(r)
	rt.projectIllustrationRoutes(r)
	// The idea step's brief and the review units of every step. See ADR-0035.
	rt.reviewUnitRoutes(r)
	// The research step: questions, the prompt for an outside tool, pasted answers, facts.
	rt.researchRoutes(r)
	// The story step: approaches, the six fields of the frame, the facts it uses.
	rt.storyRoutes(r)
	// The beat sheet step: the beats, their order and the AI runs that write them.
	rt.beatRoutes(r)
	// The script step: the dialogue of each beat, its checks and the reading of each line.
	rt.scriptRoutes(r)
	// The lock step: the review of the whole script, the handoff notes and the locked versions.
	rt.scriptLockRoutes(r)
	r.Post("/v1/admin/video-archetypes", rt.handleCreateArchetype)
	r.Post("/v1/admin/video-archetypes/{id}/copy", rt.handleCopyArchetype)
	r.Put("/v1/admin/video-archetypes/{id}", rt.handleUpdateArchetype)
	r.Delete("/v1/admin/video-archetypes/{id}", rt.handleDeleteArchetype)
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
	// The prompt with every {{variable}} already filled in.
	r.Get("/v1/projects/{project_id}/prompts/{role}", rt.handleRenderPrompt)
	// The same render for what the browser has in hand but has
	// not saved (draft topic, pasted script, unapplied subtitle style).
	r.Post("/v1/prompt-renders", rt.handlePromptRenders)
	// Starter scripts and hook/end-screen snippets (static text).
	// Run a step with the API, and tell the GUI whether that
	// option exists at all before it draws the button.
	r.Get("/v1/llm/status", rt.handleLLMStatus)
	r.Put("/v1/projects/{project_id}/authoring/mode", rt.handleSaveAuthoringMode)
	r.Put("/v1/projects/{project_id}/authoring/models", rt.handleSaveAuthoringModels)

	// Internal — called by the orchestrator only (never routed by the gateway).
	r.Get("/internal/v1/authoring/summaries", rt.handleInternalSummaries)
	r.Get("/internal/v1/authoring/similar", rt.handleInternalSimilar)
	r.Post("/internal/v1/authoring/{project_id}/voice-lines/claim", rt.handleClaimVoiceLines)
	r.Post("/internal/v1/authoring/{project_id}/voice-lines/results", rt.handleVoiceLineResults)
	r.Post("/internal/v1/authoring/{project_id}/voice-lines/fail", rt.handleFailVoiceLines)
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
// step exists.
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

// promptsUseCase backs the prompt library: a list of prompts per
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

// renderPromptUseCase backs GET
// /v1/projects/{id}/prompts/{role} — the prompt fully substituted, so the
// Copy button and the server's own generate call use the same text.
type renderPromptUseCase interface {
	Execute(ctx context.Context, projectID string, role domain.PromptRole) (application.RenderedPrompt, error)
}

// saveAuthoringModeUseCase backs PUT
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

// llmStatusSource says whether a model can be asked at all: llm-service is
// reachable and holds a key.
type llmStatusSource interface {
	Ready() bool
	Name() string
}

// saveAuthoringStoryUseCase backs step 1's POST
// /v1/projects/{id}/authoring/story.
type saveAuthoringStoryUseCase interface {
	Execute(ctx context.Context, projectID, content, topic string) error
}

// saveAuthoringStoryboardUseCase backs step 2's POST
// /v1/projects/{id}/authoring/storyboard.
type saveAuthoringStoryboardUseCase interface {
	Execute(ctx context.Context, projectID, content string) error
}

// saveAuthoringCodeUseCase backs step 3's POST
// /v1/projects/{id}/authoring/code.
type saveAuthoringCodeUseCase interface {
	Execute(ctx context.Context, projectID, content string) error
}

// getAuthoringStateUseCase backs GET /v1/projects/{id}/authoring, letting the
// wizard rehydrate saved story/storyboard/code on reload/back-navigation.
type getAuthoringStateUseCase interface {
	Execute(ctx context.Context, projectID string) (application.AuthoringState, error)
}

// WithLLMStatus enables GET /v1/llm/status: whether the AI path can be offered.
func (rt *Router) WithLLMStatus(status llmStatusSource) *Router {
	rt.llmStatus = status
	return rt
}

// WithRenderPrompt enables server-side prompt rendering.
func (rt *Router) WithRenderPrompt(renderPrompt renderPromptUseCase) *Router {
	rt.renderPrompt = renderPrompt
	return rt
}

// WithAuthoringMode enables the persisted step-1 working mode.
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

// WithDefaultModelSetting enables the app setting that names the default model
// of each step, and makes GET /v1/llm/status report them.
func (rt *Router) WithDefaultModelSetting(uc defaultModelUseCase) *Router {
	rt.defaultModelSetting = uc
	return rt
}

// codeStatsWindow is how far back the code step's model numbers look.
const codeStatsWindow = 30 * 24 * time.Hour

// WithPrompts attaches the prompt library, enabling GET
// /v1/prompts/{role} and the /v1/admin/prompts routes. Without it the routes
// answer 404, the same "unwired means absent" posture as WithQCReports.
func (rt *Router) WithPrompts(prompts promptsUseCase) *Router {
	rt.prompts = prompts
	return rt
}

// WithAuthoringStory attaches step 1's save-story use case, enabling
// POST /v1/projects/{project_id}/authoring/story.
func (rt *Router) WithAuthoringStory(saveAuthoringStory saveAuthoringStoryUseCase) *Router {
	rt.saveAuthoringStory = saveAuthoringStory
	return rt
}

// WithAuthoringStoryboard attaches step 2's save-storyboard use case,
// enabling POST /v1/projects/{project_id}/authoring/storyboard.
func (rt *Router) WithAuthoringStoryboard(saveAuthoringStoryboard saveAuthoringStoryboardUseCase) *Router {
	rt.saveAuthoringStoryboard = saveAuthoringStoryboard
	return rt
}

// WithAuthoringCode attaches step 3's save-code use case, enabling
// POST /v1/projects/{project_id}/authoring/code.
func (rt *Router) WithAuthoringCode(saveAuthoringCode saveAuthoringCodeUseCase) *Router {
	rt.saveAuthoringCode = saveAuthoringCode
	return rt
}

// WithAuthoringState attaches the authoring read-side use case, enabling
// GET /v1/projects/{project_id}/authoring.
func (rt *Router) WithAuthoringState(getAuthoringState getAuthoringStateUseCase) *Router {
	rt.getAuthoringState = getAuthoringState
	return rt
}

// WithOperations attaches the registry behind GET /v1/operations/{id}.
// Without it suggestions still work, just without a live
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
	ctx, cancel := context.WithCancel(r.Context())
	rt.operations.Start(id, kind, chi.URLParam(r, "project_id"), cancel)
	ctx = application.WithProgress(ctx, func(p application.ChatProgress) { rt.operations.Progress(id, p) })
	return ctx, func(err error) {
		rt.operations.Finish(id, err)
		cancel()
	}
}

// handleCancelOperation stops a running long call by its id, from any page: the
// cancellation token the GUI holds only in the page that started the call.
func (rt *Router) handleCancelOperation(w http.ResponseWriter, r *http.Request) {
	if rt.operations == nil {
		writeError(w, http.StatusNotFound, "operations are not available")
		return
	}
	if !rt.operations.Cancel(chi.URLParam(r, "operation_id")) {
		writeError(w, http.StatusNotFound, "operation not running")
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

// handleListProjectOperations lists the long calls still running for a project,
// so a page opened again can pick up the progress of a call it did not start.
func (rt *Router) handleListProjectOperations(w http.ResponseWriter, r *http.Request) {
	running := []application.RunningOperation{}
	if rt.operations != nil {
		running = append(running, rt.operations.Running(chi.URLParam(r, "project_id"))...)
	}
	writeJSON(w, http.StatusOK, map[string]any{"operations": running})
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
// wizard reads: template_text, with the library facts alongside.
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
// pasted back after the external-AI round trip (step 1).
func (rt *Router) handleSaveAuthoringStory(w http.ResponseWriter, r *http.Request) {
	if rt.saveAuthoringStory == nil {
		writeError(w, http.StatusNotFound, "authoring pipeline is not enabled")
		return
	}
	projectID := chi.URLParam(r, "project_id")

	var req struct {
		Content string `json:"content"`
		// Optional: an empty topic leaves the stored one alone
		// (see PromptTemplateRepository.SaveAuthoringStory).
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
// pasted back after the external-AI round trip (step 2).
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

// handleSaveAuthoringCode stores the Remotion Engineer output a Creator pasted
// back after the external-AI round trip (step 3).
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
		// How the Creator is working step 1, so the wizard
		// restores the choice on reload or on another machine instead of
		// falling back to copy-and-paste. Always "manual" or "ai".
		"mode":       state.Mode,
		"topic":      state.Topic,
		"story":      state.Story,
		"storyboard": state.Storyboard,
		"code":       state.Code,
		// Model-per-step picker's saved choice for each tab, "" meaning
		// "server default" — same rehydrate-on-reload reasoning as mode.
		"brief_model":       state.Models.Brief,
		"publish_model":     state.Models.Publish,
		"research_model":    state.Models.Research,
		"story_frame_model": state.Models.StoryFrame,
		"beat_sheet_model":  state.Models.BeatSheet,
		"story_model":       state.Models.Story,
		"storyboard_model":  state.Models.Storyboard,
		"code_model":        state.Models.Code,
		// The visual language the project is pinned to; "" for a project with no pin.
		"visual_language_id":      state.Language.ID,
		"visual_language_version": strconv.Itoa(state.Language.Version),
		"kit_key":                 state.Language.KitKey,
	})
}

// handleSaveAuthoringModels stores the model-per-step picker's choice for
// the three authoring tabs, which lets each
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
	models := domain.AuthoringStepModels{
		Brief: req.Brief, Publish: req.Publish, Research: req.Research, StoryFrame: req.StoryFrame, BeatSheet: req.BeatSheet,
		Story: req.Story, Storyboard: req.Storyboard, Code: req.Code,
	}
	if err := rt.saveAuthoringModels.Execute(r.Context(), projectID, models); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleSaveAuthoringMode stores how the Creator works step 1.
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
// already substituted.
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
	if !domain.IsRenderablePromptRole(role) {
		writeError(w, http.StatusBadRequest, "role is not renderable")
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

// promptRenderer is the stateless side of the prompt renderer.
type promptRenderer interface {
	Render(ctx context.Context, in application.RenderInput) (application.RenderedPrompt, error)
}

type promptRenderRequest struct {
	Role             string `json:"role"`
	Language         string `json:"language"`
	Topic            string `json:"topic"`
	Script           string `json:"script"`
	PreviousOutput   string `json:"previous_output"`
	SubtitleMode     string `json:"subtitle_mode"`
	SubtitleFontSize string `json:"subtitle_font_size"`
	SubtitlePosition string `json:"subtitle_position"`
	FormatID         string `json:"format_id"`
	FormatVersion    int    `json:"format_version"`
	VoiceID          string `json:"voice_id"`
	// "long" | "short": which frame {{frame}} and the other frame variables describe.
	VideoOutputMode string `json:"video_output_mode"`
	// The project whose approved creative brief fills {{creative_brief}}.
	ProjectID string `json:"project_id"`
}

// handlePromptRenders serves POST /v1/prompt-renders: the
// prompt for one library role with the caller's draft values substituted. It
// replaces the string assembly web-gui did in the browser, so the text a
// Creator copies and the text the server sends to the model come from one place.
func (rt *Router) handlePromptRenders(w http.ResponseWriter, r *http.Request) {
	renderer, ok := rt.renderPrompt.(promptRenderer)
	if !ok {
		writeError(w, http.StatusNotFound, "prompt rendering is not enabled")
		return
	}
	var req promptRenderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if !domain.ValidPromptRole(req.Role) {
		writeError(w, http.StatusBadRequest, "unknown role")
		return
	}
	if !domain.IsRenderablePromptRole(req.Role) {
		writeError(w, http.StatusBadRequest, "role is not renderable")
		return
	}
	rendered, err := renderer.Render(r.Context(), application.RenderInput{
		Role: domain.PromptRole(req.Role), Language: req.Language, Topic: req.Topic, Script: req.Script,
		PreviousOutput: req.PreviousOutput, SubtitleMode: req.SubtitleMode,
		SubtitleFontSize: req.SubtitleFontSize, SubtitlePosition: req.SubtitlePosition,
		FormatID: req.FormatID, FormatVersion: req.FormatVersion, VoiceID: req.VoiceID,
		VideoOutputMode: req.VideoOutputMode, ProjectID: req.ProjectID,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rendered)
}

// llmStatusResponse tells the GUI whether the "Chạy bằng AI" button has
// anything to call.
type llmStatusResponse struct {
	Enabled  bool   `json:"enabled"`
	Provider string `json:"provider"`
	// Reason is filled only when Enabled is false, in Vietnamese, pointing at
	// the file the Creator has to edit — a disabled button that does not say
	// why leaves the Creator stuck.
	Reason string `json:"reason,omitempty"`
	// Models is the model-per-step picker's catalog (domain.
	// AuthoringModelCatalog) — served here rather than hardcoded a second
	// time in web-gui, so the GUI's dropdown and the server's own validation
	// (SaveAuthoringModelsUseCase) can never drift apart. Empty when the AI
	// path itself is unavailable — nothing to pick a model for.
	Models []domain.AuthoringModelOption `json:"models,omitempty"`
	// StepDefaults is the default model of every step; the GUI pre-selects it
	// where a step's model is chosen.
	StepDefaults map[string]string `json:"step_defaults,omitempty"`
}

func (rt *Router) handleLLMStatus(w http.ResponseWriter, r *http.Request) {
	if rt.llmStatus == nil || !rt.llmStatus.Ready() {
		writeJSON(w, http.StatusOK, llmStatusResponse{
			Enabled: false,
			Reason:  "Chưa cấu hình HIVE_API_KEY cho llm-service (hoặc llm-service không chạy) — dùng nút Copy prompt như cũ.",
		})
		return
	}
	resp := llmStatusResponse{
		Enabled: true, Provider: rt.llmStatus.Name(),
		Models: domain.AuthoringModelCatalog, StepDefaults: rt.effectiveStepDefaults(r.Context()),
	}
	writeJSON(w, http.StatusOK, resp)
}

// errorCause is the underlying reason of an LLMError without its
// "provider: kind:" prefix, which the Creator has no use for.
func errorCause(err error) string {
	var llmErr *application.LLMError
	if errors.As(err, &llmErr) && llmErr.Err != nil {
		if llmErr.CallID != "" {
			return fmt.Sprintf("%s (mã cuộc gọi %s)", llmErr.Err.Error(), llmErr.CallID)
		}
		return llmErr.Err.Error()
	}
	return err.Error()
}

// DescribeGenerateError is the status code and Creator-facing sentence for a
// failed model call, shared by every handler that asks a model.
func DescribeGenerateError(err error) (int, string) {
	const fallback = " Hoặc dùng nút Copy prompt như cũ."

	switch {
	case errors.Is(err, application.ErrLLMNotConfigured):
		return http.StatusServiceUnavailable,
			"Chưa cấu hình HIVE_API_KEY trong .env (llm-service) nên không gọi được AI." + fallback
	case errors.Is(err, domain.ErrProjectNotFound):
		return http.StatusNotFound, "project not found"
	}
	// Not a failure — the code step waits for the Creator's review.
	var pending *application.ErrIllustrationsPending
	if errors.As(err, &pending) {
		return http.StatusConflict, pending.Error()
	}
	if errors.Is(err, application.ErrIllustrationsNotPlanned) || errors.Is(err, application.ErrIllustrationsStale) {
		return http.StatusConflict, err.Error()
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
		// llm-service says what was wrong (an unusable storyboard, code
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
