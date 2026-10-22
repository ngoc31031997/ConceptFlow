package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

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

// creativeBriefUseCase is the idea step beyond the generic unit API.
type creativeBriefUseCase interface {
	State(ctx context.Context, projectID string) (application.BriefState, error)
	SetTargetDuration(ctx context.Context, projectID string, d domain.TargetDuration) error
	Generate(ctx context.Context, projectID, note string) (domain.ReviewUnit, error)
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

// WithCreativeBrief enables the /brief routes.
func (rt *Router) WithCreativeBrief(uc creativeBriefUseCase) *Router {
	rt.creativeBrief = uc
	return rt
}

func (rt *Router) reviewUnitRoutes(r chi.Router) {
	r.Get("/v1/settings/ai-model", rt.handleGetAIModelSetting)
	r.Put("/v1/settings/ai-model/steps/{step}", rt.handlePutAIModelStep)
	r.Get("/v1/projects/{project_id}/brief", rt.handleGetBrief)
	r.Put("/v1/projects/{project_id}/brief/target-duration", rt.handleSetTargetDuration)
	r.Post("/v1/projects/{project_id}/brief/generate", rt.handleGenerateBrief)
	r.Get("/v1/projects/{project_id}/units/{kind}/{key}", rt.handleGetUnit)
	r.Put("/v1/projects/{project_id}/units/{kind}/{key}", rt.handleEditUnit)
	r.Post("/v1/projects/{project_id}/units/{kind}/{key}/redo", rt.handleRedoUnit)
	r.Post("/v1/projects/{project_id}/units/{kind}/{key}/approve", rt.handleApproveUnit)
	r.Post("/v1/projects/{project_id}/units/{kind}/{key}/unapprove", rt.handleUnapproveUnit)
	r.Get("/v1/projects/{project_id}/units/{kind}/{key}/impact", rt.handleUnitImpact)
}

// briefStateResponse is GET /v1/projects/{id}/brief.
type briefStateResponse struct {
	Unit                  *domain.ReviewUnit `json:"unit"`
	TargetDuration        string             `json:"target_duration"`
	StoryStale            bool               `json:"story_stale"`
	StoryPromptReadsBrief bool               `json:"story_prompt_reads_brief"`
}

func (rt *Router) handleGetBrief(w http.ResponseWriter, r *http.Request) {
	if rt.creativeBrief == nil {
		writeError(w, http.StatusNotFound, "creative brief is not enabled")
		return
	}
	state, err := rt.creativeBrief.State(r.Context(), chi.URLParam(r, "project_id"))
	if err != nil {
		writeUnitError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, briefStateResponse{
		Unit: state.Unit, TargetDuration: string(state.TargetDuration),
		StoryStale: state.StoryStale, StoryPromptReadsBrief: state.StoryPromptReadsBrief,
	})
}

func (rt *Router) handleSetTargetDuration(w http.ResponseWriter, r *http.Request) {
	if rt.creativeBrief == nil {
		writeError(w, http.StatusNotFound, "creative brief is not enabled")
		return
	}
	var body struct {
		TargetDuration string `json:"target_duration"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := rt.creativeBrief.SetTargetDuration(r.Context(), chi.URLParam(r, "project_id"), domain.TargetDuration(body.TargetDuration)); err != nil {
		writeUnitError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

func (rt *Router) handleGenerateBrief(w http.ResponseWriter, r *http.Request) {
	if rt.creativeBrief == nil {
		writeError(w, http.StatusNotFound, "creative brief is not enabled")
		return
	}
	note, err := readNote(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	ctx, finish := rt.trackOperation(r, domain.UnitKindBrief)
	unit, err := rt.creativeBrief.Generate(ctx, chi.URLParam(r, "project_id"), note)
	finish(err)
	if err != nil {
		writeUnitRunError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, unit)
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
	case errors.Is(err, domain.ErrUnitNotFound), errors.Is(err, domain.ErrUnknownUnitKind), errors.Is(err, domain.ErrProjectNotFound),
		errors.Is(err, application.ErrFindingNotFound), errors.Is(err, domain.ErrNoLockedVersion):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrInvalidUnitEdit), errors.Is(err, application.ErrBriefIncomplete), errors.Is(err, domain.ErrInvalidWizardInput),
		errors.Is(err, domain.ErrNoResearchJSON), errors.Is(err, domain.ErrPasteTooLarge), errors.Is(err, domain.ErrResearchEmpty),
		errors.Is(err, application.ErrStoryIncomplete), errors.Is(err, application.ErrStoryNothingToWrite),
		errors.Is(err, application.ErrBeatNoGoal), errors.Is(err, application.ErrBeatSheetFull),
		errors.Is(err, application.ErrScriptBeatEmpty), errors.Is(err, application.ErrScriptBeatHasLines), errors.Is(err, application.ErrScriptLineNotFlagged),
		errors.Is(err, domain.ErrHandoffEmpty), errors.Is(err, application.ErrHandoffExists), errors.Is(err, application.ErrFindingNotApplicable):
		writeError(w, http.StatusBadRequest, reasonOf(err, domain.ErrInvalidUnitEdit, domain.ErrInvalidWizardInput))
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
		errors.Is(err, domain.ErrInvalidUnitEdit), errors.Is(err, domain.ErrInvalidWizardInput),
		errors.Is(err, domain.ErrInvalidStatus):
		writeUnitError(w, err)
	case errors.Is(err, domain.ErrBriefReplyUnusable), errors.Is(err, domain.ErrBeatReplyUnusable), errors.Is(err, domain.ErrScriptReplyUnusable),
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
