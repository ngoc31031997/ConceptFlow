package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"authoring/internal/application"
	"authoring/internal/domain"
)

// reviewUnitsUseCase is the generic unit API every step's units go through.
type reviewUnitsUseCase interface {
	Get(ctx context.Context, projectID, kind, key string) (domain.ReviewUnit, error)
	Edit(ctx context.Context, projectID, kind, key string, fields map[string]string) (domain.ReviewUnit, error)
	Redo(ctx context.Context, projectID, kind, key, note string) (domain.ReviewUnit, error)
	Impact(ctx context.Context, projectID, kind, key string) ([]string, error)
	Approve(ctx context.Context, projectID, kind, key string) (domain.ReviewUnit, error)
	Unapprove(ctx context.Context, projectID, kind, key string) (domain.ReviewUnit, error)
}

// ideaUseCase is the first step: the idea the AI writes from the Creator's text,
// the Creator's edits and approval, and the AI's review of it.
type ideaUseCase interface {
	State(ctx context.Context, projectID string) (application.ProjectIdeaState, error)
	Generate(ctx context.Context, projectID, note string) (domain.StoredIdea, error)
	Edit(ctx context.Context, projectID string, patch application.IdeaPatch) (domain.StoredIdea, error)
	Approve(ctx context.Context, projectID string) (domain.StoredIdea, error)
	Unapprove(ctx context.Context, projectID string) (domain.StoredIdea, error)
	Review(ctx context.Context, projectID string) (domain.IdeaReview, error)
}

// projectBriefUseCase is the second step: the brief the AI writes from the
// approved idea, the Creator's edits and approval, and the AI's review of it.
type projectBriefUseCase interface {
	State(ctx context.Context, projectID string) (application.ProjectBriefState, error)
	Generate(ctx context.Context, projectID, note string) (domain.StoredBrief, error)
	Edit(ctx context.Context, projectID string, patch application.BriefPatch) (domain.StoredBrief, error)
	Approve(ctx context.Context, projectID string) (domain.StoredBrief, error)
	Unapprove(ctx context.Context, projectID string) (domain.StoredBrief, error)
	Review(ctx context.Context, projectID string) (domain.BriefReview, error)
}

// defaultModelUseCase is the app setting for the default AI model of each step.
type defaultModelUseCase interface {
	StepModels(ctx context.Context) (map[string]string, error)
	SetStep(ctx context.Context, step, model string) error
}

// effectiveStepDefaults is the default model of every configurable step. An
// unreadable setting is logged and read as none, so the GUI still opens.
func (rt *Router) effectiveStepDefaults(ctx context.Context) map[string]string {
	if rt.defaultModelSetting == nil {
		return map[string]string{}
	}
	steps, err := rt.defaultModelSetting.StepModels(ctx)
	if err != nil {
		slog.Warn("could not read the step default models", "error", err.Error())
		return map[string]string{}
	}
	return steps
}

// aiModelSettingResponse is GET /v1/settings/ai-model: the default model of
// every step, and the steps that can carry one.
type aiModelSettingResponse struct {
	StepModels map[string]string `json:"step_models"`
	Steps      []string          `json:"steps"`
}

func (rt *Router) handleGetAIModelSetting(w http.ResponseWriter, r *http.Request) {
	if rt.defaultModelSetting == nil {
		writeError(w, http.StatusNotFound, "default model setting is not enabled")
		return
	}
	steps, err := rt.defaultModelSetting.StepModels(r.Context())
	if err != nil {
		slog.Error("read step default models failed", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, aiModelSettingResponse{StepModels: steps, Steps: domain.ConfigurableModelSteps})
}

func (rt *Router) handlePutAIModelStep(w http.ResponseWriter, r *http.Request) {
	if rt.defaultModelSetting == nil {
		writeError(w, http.StatusNotFound, "default model setting is not enabled")
		return
	}
	var body struct {
		Model string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := rt.defaultModelSetting.SetStep(r.Context(), chi.URLParam(r, "step"), body.Model); err != nil {
		if errors.Is(err, domain.ErrInvalidWizardInput) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		slog.Error("save step default model failed", "error", err.Error())
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// WithReviewUnits enables the /units routes.
func (rt *Router) WithReviewUnits(uc reviewUnitsUseCase) *Router {
	rt.reviewUnits = uc
	return rt
}

// WithIdea enables the /idea routes.
func (rt *Router) WithIdea(uc ideaUseCase) *Router {
	rt.idea = uc
	return rt
}

func (rt *Router) reviewUnitRoutes(r chi.Router) {
	r.Get("/v1/settings/ai-model", rt.handleGetAIModelSetting)
	r.Put("/v1/settings/ai-model/steps/{step}", rt.handlePutAIModelStep)
	rt.projectBriefRoutes(r)
	r.Get("/v1/projects/{project_id}/idea", rt.handleGetIdea)
	r.Post("/v1/projects/{project_id}/idea/generate", rt.handleGenerateIdea)
	r.Put("/v1/projects/{project_id}/idea", rt.handleEditIdea)
	r.Post("/v1/projects/{project_id}/idea/approve", rt.handleApproveIdea)
	r.Post("/v1/projects/{project_id}/idea/unapprove", rt.handleUnapproveIdea)
	r.Post("/v1/projects/{project_id}/idea/review", rt.handleReviewIdea)
	r.Get("/v1/projects/{project_id}/units/{kind}/{key}", rt.handleGetUnit)
	r.Put("/v1/projects/{project_id}/units/{kind}/{key}", rt.handleEditUnit)
	r.Post("/v1/projects/{project_id}/units/{kind}/{key}/redo", rt.handleRedoUnit)
	r.Post("/v1/projects/{project_id}/units/{kind}/{key}/approve", rt.handleApproveUnit)
	r.Post("/v1/projects/{project_id}/units/{kind}/{key}/unapprove", rt.handleUnapproveUnit)
	r.Get("/v1/projects/{project_id}/units/{kind}/{key}/impact", rt.handleUnitImpact)
}

// ideaDTO is an idea as the screen reads it.
type ideaDTO struct {
	CentralQuestion string     `json:"central_question"`
	CuriosityPoints []string   `json:"curiosity_points"`
	Scope           string     `json:"scope"`
	CreatorFields   []string   `json:"creator_fields"`
	Status          string     `json:"status"`
	RedoCount       int        `json:"redo_count"`
	ApprovedAt      *time.Time `json:"approved_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func toIdeaDTO(s domain.StoredIdea) ideaDTO {
	idea := s.Idea.Normalized()
	creator := s.CreatorFields
	if creator == nil {
		creator = []string{}
	}
	return ideaDTO{
		CentralQuestion: idea.CentralQuestion, CuriosityPoints: idea.CuriosityPoints,
		Scope: idea.Scope, CreatorFields: creator, Status: string(s.Status), RedoCount: s.RedoCount,
		ApprovedAt: s.ApprovedAt, UpdatedAt: s.UpdatedAt,
	}
}

// ideaStateResponse is GET /v1/projects/{id}/idea. A review exists only after
// the AI read the idea; review_stale says the idea changed after that.
type ideaStateResponse struct {
	Idea         *ideaDTO           `json:"idea"`
	Subject      string             `json:"subject"`
	Language     string             `json:"language"`
	InputChanged bool               `json:"input_changed"`
	Review       *domain.IdeaReview `json:"review"`
	ReviewStale  bool               `json:"review_stale"`
}

func (rt *Router) handleGetIdea(w http.ResponseWriter, r *http.Request) {
	if rt.idea == nil {
		writeError(w, http.StatusNotFound, "idea is not enabled")
		return
	}
	state, err := rt.idea.State(r.Context(), chi.URLParam(r, "project_id"))
	if err != nil {
		writeUnitError(w, err)
		return
	}
	resp := ideaStateResponse{
		Subject: state.Subject, Language: string(state.Language), InputChanged: state.InputChanged,
		Review: state.Review, ReviewStale: state.ReviewStale,
	}
	if state.Idea != nil {
		dto := toIdeaDTO(*state.Idea)
		resp.Idea = &dto
	}
	writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleGenerateIdea(w http.ResponseWriter, r *http.Request) {
	if rt.idea == nil {
		writeError(w, http.StatusNotFound, "idea is not enabled")
		return
	}
	note, err := readNote(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	ctx, finish := rt.trackOperation(r, "idea")
	idea, err := rt.idea.Generate(ctx, chi.URLParam(r, "project_id"), note)
	finish(err)
	if err != nil {
		writeUnitRunError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toIdeaDTO(idea))
}

// editIdeaRequest is PUT /v1/projects/{id}/idea; an absent field is left alone.
type editIdeaRequest struct {
	CentralQuestion *string   `json:"central_question"`
	CuriosityPoints *[]string `json:"curiosity_points"`
	Scope           *string   `json:"scope"`
}

func (rt *Router) handleEditIdea(w http.ResponseWriter, r *http.Request) {
	if rt.idea == nil {
		writeError(w, http.StatusNotFound, "idea is not enabled")
		return
	}
	var req editIdeaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	idea, err := rt.idea.Edit(r.Context(), chi.URLParam(r, "project_id"), application.IdeaPatch{
		CentralQuestion: req.CentralQuestion,
		CuriosityPoints: req.CuriosityPoints, Scope: req.Scope,
	})
	if err != nil {
		writeUnitError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toIdeaDTO(idea))
}

func (rt *Router) handleApproveIdea(w http.ResponseWriter, r *http.Request) {
	rt.ideaStatusChange(w, r, func(ctx context.Context, id string) (domain.StoredIdea, error) {
		return rt.idea.Approve(ctx, id)
	})
}

func (rt *Router) handleUnapproveIdea(w http.ResponseWriter, r *http.Request) {
	rt.ideaStatusChange(w, r, func(ctx context.Context, id string) (domain.StoredIdea, error) {
		return rt.idea.Unapprove(ctx, id)
	})
}

func (rt *Router) ideaStatusChange(w http.ResponseWriter, r *http.Request, change func(context.Context, string) (domain.StoredIdea, error)) {
	if rt.idea == nil {
		writeError(w, http.StatusNotFound, "idea is not enabled")
		return
	}
	idea, err := change(r.Context(), chi.URLParam(r, "project_id"))
	if err != nil {
		writeUnitError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toIdeaDTO(idea))
}

func (rt *Router) handleReviewIdea(w http.ResponseWriter, r *http.Request) {
	if rt.idea == nil {
		writeError(w, http.StatusNotFound, "idea is not enabled")
		return
	}
	ctx, finish := rt.trackOperation(r, "idea_review")
	review, err := rt.idea.Review(ctx, chi.URLParam(r, "project_id"))
	finish(err)
	if err != nil {
		writeUnitRunError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, review)
}

// readNote reads an optional {"note": "..."} body.
func readNote(r *http.Request) (string, error) {
	if r.ContentLength == 0 {
		return "", nil
	}
	var body struct {
		Note string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return body.Note, nil
}

func unitParams(r *http.Request) (projectID, kind, key string) {
	return chi.URLParam(r, "project_id"), chi.URLParam(r, "kind"), chi.URLParam(r, "key")
}

func (rt *Router) handleGetUnit(w http.ResponseWriter, r *http.Request) {
	if rt.reviewUnits == nil {
		writeError(w, http.StatusNotFound, "review units are not enabled")
		return
	}
	projectID, kind, key := unitParams(r)
	unit, err := rt.reviewUnits.Get(r.Context(), projectID, kind, key)
	if err != nil {
		writeUnitError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, unit)
}

func (rt *Router) handleEditUnit(w http.ResponseWriter, r *http.Request) {
	if rt.reviewUnits == nil {
		writeError(w, http.StatusNotFound, "review units are not enabled")
		return
	}
	var body struct {
		Fields map[string]string `json:"fields"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	projectID, kind, key := unitParams(r)
	unit, err := rt.reviewUnits.Edit(r.Context(), projectID, kind, key, body.Fields)
	if err != nil {
		writeUnitError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, unit)
}

func (rt *Router) handleRedoUnit(w http.ResponseWriter, r *http.Request) {
	if rt.reviewUnits == nil {
		writeError(w, http.StatusNotFound, "review units are not enabled")
		return
	}
	note, err := readNote(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	projectID, kind, key := unitParams(r)
	ctx, finish := rt.trackOperation(r, kind)
	unit, err := rt.reviewUnits.Redo(ctx, projectID, kind, key, note)
	finish(err)
	if err != nil {
		writeUnitRunError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, unit)
}

func (rt *Router) handleApproveUnit(w http.ResponseWriter, r *http.Request) {
	rt.handleUnitStatus(w, r, true)
}

func (rt *Router) handleUnapproveUnit(w http.ResponseWriter, r *http.Request) {
	rt.handleUnitStatus(w, r, false)
}

func (rt *Router) handleUnitStatus(w http.ResponseWriter, r *http.Request, approve bool) {
	if rt.reviewUnits == nil {
		writeError(w, http.StatusNotFound, "review units are not enabled")
		return
	}
	projectID, kind, key := unitParams(r)
	var unit domain.ReviewUnit
	var err error
	if approve {
		unit, err = rt.reviewUnits.Approve(r.Context(), projectID, kind, key)
	} else {
		unit, err = rt.reviewUnits.Unapprove(r.Context(), projectID, kind, key)
	}
	if err != nil {
		writeUnitError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, unit)
}

// unitImpactResponse lists what changing a unit sends back in later steps.
type unitImpactResponse struct {
	Items []string `json:"items"`
}

func (rt *Router) handleUnitImpact(w http.ResponseWriter, r *http.Request) {
	if rt.reviewUnits == nil {
		writeError(w, http.StatusNotFound, "review units are not enabled")
		return
	}
	projectID, kind, key := unitParams(r)
	items, err := rt.reviewUnits.Impact(r.Context(), projectID, kind, key)
	if err != nil {
		writeUnitError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, unitImpactResponse{Items: items})
}

// writeUnitError maps the unit API's errors onto status codes; the message is
// the error's own text, which names what was wrong.
func writeUnitError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrUnitNotFound), errors.Is(err, domain.ErrUnknownUnitKind), errors.Is(err, domain.ErrProjectNotFound), errors.Is(err, domain.ErrIdeaNotFound), errors.Is(err, domain.ErrBriefNotFound), errors.Is(err, domain.ErrStoryNotFound),
		errors.Is(err, application.ErrFindingNotFound), errors.Is(err, domain.ErrNoLockedVersion):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrInvalidUnitEdit), errors.Is(err, domain.ErrProjectIdeaInvalid), errors.Is(err, domain.ErrProjectBriefInvalid), errors.Is(err, domain.ErrInvalidWizardInput),
		errors.Is(err, domain.ErrNoResearchJSON), errors.Is(err, domain.ErrPasteTooLarge), errors.Is(err, domain.ErrResearchEmpty),
		errors.Is(err, domain.ErrStoryInvalid), errors.Is(err, application.ErrStoryNothingToWrite),
		errors.Is(err, application.ErrBeatNoGoal), errors.Is(err, application.ErrBeatSheetFull),
		errors.Is(err, application.ErrScriptBeatEmpty), errors.Is(err, application.ErrScriptBeatHasLines), errors.Is(err, application.ErrScriptLineNotFlagged),
		errors.Is(err, domain.ErrHandoffEmpty), errors.Is(err, application.ErrHandoffExists), errors.Is(err, application.ErrFindingNotApplicable):
		writeError(w, http.StatusBadRequest, reasonOf(err, domain.ErrInvalidUnitEdit, domain.ErrInvalidWizardInput, domain.ErrProjectIdeaInvalid, domain.ErrProjectBriefInvalid, domain.ErrStoryInvalid))
	case errors.Is(err, domain.ErrInvalidStatus):
		writeError(w, http.StatusConflict, reasonOf(err, domain.ErrInvalidStatus))
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

// reasonOf is the text of err the Creator reads: when err wraps one of the
// sentinels with a reason of its own ("sentinel: reason"), just the reason, so
// the English name of the sentinel does not lead a Vietnamese message.
func reasonOf(err error, sentinels ...error) string {
	msg := err.Error()
	for _, s := range sentinels {
		if rest, ok := strings.CutPrefix(msg, s.Error()+": "); ok {
			return rest
		}
	}
	return msg
}

// writeUnitRunError maps the error of an AI write or redo: the unit API's own
// errors first, then the reply that could not be used, then every model
// failure the way the other AI runs report it.
func writeUnitRunError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrUnitNotFound), errors.Is(err, domain.ErrUnknownUnitKind),
		errors.Is(err, domain.ErrInvalidUnitEdit), errors.Is(err, domain.ErrInvalidWizardInput), errors.Is(err, domain.ErrIdeaNotFound), errors.Is(err, domain.ErrBriefNotFound),
		errors.Is(err, domain.ErrProjectIdeaInvalid), errors.Is(err, domain.ErrProjectBriefInvalid), errors.Is(err, domain.ErrStoryNotFound), errors.Is(err, domain.ErrStoryInvalid), errors.Is(err, domain.ErrInvalidStatus):
		writeUnitError(w, err)
	case errors.Is(err, domain.ErrProjectIdeaReplyUnusable), errors.Is(err, domain.ErrIdeaReviewReplyUnusable), errors.Is(err, domain.ErrProjectBriefReplyUnusable), errors.Is(err, domain.ErrBriefReviewReplyUnusable), errors.Is(err, domain.ErrStoryReplyUnusable), errors.Is(err, domain.ErrBeatReplyUnusable), errors.Is(err, domain.ErrScriptReplyUnusable),
		errors.Is(err, domain.ErrReviewReplyUnusable), errors.Is(err, domain.ErrHandoffReplyUnusable):
		writeError(w, http.StatusBadGateway, err.Error()+". Thử làm lại, hoặc tự điền các ô.")
	case errors.Is(err, application.ErrLLMNotConfigured):
		writeError(w, http.StatusServiceUnavailable,
			"Chưa cấu hình HIVE_API_KEY trong .env (llm-service) nên AI không viết được. Bạn tự điền các ô rồi duyệt.")
	default:
		status, message := DescribeGenerateError(err)
		writeError(w, status, message)
	}
}
