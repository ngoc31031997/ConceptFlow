package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"authoring/internal/application"
	"authoring/internal/domain"
)

// researchUseCase is the research step beyond the generic unit API.
type researchUseCase interface {
	State(ctx context.Context, projectID string) (application.ResearchState, error)
	GenerateQuestions(ctx context.Context, projectID string, replaceEdited bool, note string) ([]application.QuestionView, error)
	AddQuestion(ctx context.Context, projectID, text string) (application.QuestionView, error)
	DeleteQuestion(ctx context.Context, projectID, id string) error
	ResearchPrompt(ctx context.Context, projectID string) (string, error)
	FollowUpPrompt(ctx context.Context, projectID string) (string, error)
	IngestPaste(ctx context.Context, projectID, raw string) (application.PasteResult, error)
	AddFact(ctx context.Context, projectID string, in application.FactInput) (application.FactView, error)
	SetFactFlags(ctx context.Context, projectID, id string, flags application.FactFlags) (application.FactView, error)
	Verify(ctx context.Context, projectID string, ids []string) error
	SetSkipped(ctx context.Context, projectID string, skipped bool) (application.GateView, error)
}

// WithResearch enables the /research routes.
func (rt *Router) WithResearch(uc researchUseCase) *Router {
	rt.research = uc
	return rt
}

func (rt *Router) researchRoutes(r chi.Router) {
	const base = "/v1/projects/{project_id}/research"
	r.Get(base, rt.handleGetResearch)
	r.Post(base+"/questions/generate", rt.handleGenerateQuestions)
	r.Post(base+"/questions", rt.handleAddQuestion)
	r.Delete(base+"/questions/{question_id}", rt.handleDeleteQuestion)
	r.Get(base+"/prompt", rt.handleResearchPrompt)
	r.Get(base+"/follow-up-prompt", rt.handleFollowUpPrompt)
	r.Post(base+"/paste", rt.handlePasteResearch)
	r.Post(base+"/facts", rt.handleAddFact)
	r.Put(base+"/facts/{fact_id}", rt.handleSetFactFlags)
	r.Post(base+"/verify", rt.handleVerifyFacts)
	r.Put(base+"/skip", rt.handleSkipResearch)
}

// researchEnabled answers 404 when the research step is not wired.
func (rt *Router) researchEnabled(w http.ResponseWriter) bool {
	if rt.research == nil {
		writeError(w, http.StatusNotFound, "research is not enabled")
		return false
	}
	return true
}

// writeQuestionRunError maps the errors of the AI writing the questions.
func writeQuestionRunError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrResearchReplyUnusable):
		writeError(w, http.StatusBadGateway, err.Error()+". Thử sinh lại, hoặc tự thêm câu hỏi.")
	case errors.Is(err, application.ErrLLMNotConfigured):
		writeError(w, http.StatusServiceUnavailable,
			"Chưa cấu hình HIVE_API_KEY trong .env (llm-service) nên AI không sinh được câu hỏi. Bạn tự thêm câu hỏi.")
	default:
		writeUnitRunError(w, err)
	}
}

type questionResponse struct {
	ID    string `json:"id"`
	Text  string `json:"text"`
	Mine  bool   `json:"mine"`
	Stale bool   `json:"stale"`
}

func questionOf(q application.QuestionView) questionResponse {
	return questionResponse{ID: q.ID, Text: q.Text, Mine: q.Mine, Stale: q.Stale}
}

func questionsOf(qs []application.QuestionView) []questionResponse {
	out := make([]questionResponse, 0, len(qs))
	for _, q := range qs {
		out = append(out, questionOf(q))
	}
	return out
}

func (rt *Router) handleGetResearch(w http.ResponseWriter, r *http.Request) {
	if !rt.researchEnabled(w) {
		return
	}
	state, err := rt.research.State(r.Context(), chi.URLParam(r, "project_id"))
	if err != nil {
		writeUnitError(w, err)
		return
	}
	facts := state.Facts
	if facts == nil {
		facts = []application.FactView{}
	}
	coverage := state.Coverage
	if coverage == nil {
		coverage = []domain.QuestionCoverage{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"gate":            state.Gate,
		"questions":       questionsOf(state.Questions),
		"facts":           facts,
		"pastes":          state.Pastes,
		"coverage":        coverage,
		"brief_approved":  state.BriefApproved,
		"questions_stale": state.QuestionsStale,
	})
}

func (rt *Router) handleGenerateQuestions(w http.ResponseWriter, r *http.Request) {
	if !rt.researchEnabled(w) {
		return
	}
	var body struct {
		ReplaceEdited bool   `json:"replace_edited"`
		Note          string `json:"note"`
	}
	if r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}
	ctx, finish := rt.trackOperation(r, domain.UnitKindResearchQuestion)
	questions, err := rt.research.GenerateQuestions(ctx, chi.URLParam(r, "project_id"), body.ReplaceEdited, body.Note)
	finish(err)
	if err != nil {
		writeQuestionRunError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"questions": questionsOf(questions)})
}

func (rt *Router) handleAddQuestion(w http.ResponseWriter, r *http.Request) {
	if !rt.researchEnabled(w) {
		return
	}
	var body struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	q, err := rt.research.AddQuestion(r.Context(), chi.URLParam(r, "project_id"), body.Text)
	if err != nil {
		writeUnitError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, questionOf(q))
}

func (rt *Router) handleDeleteQuestion(w http.ResponseWriter, r *http.Request) {
	if !rt.researchEnabled(w) {
		return
	}
	if err := rt.research.DeleteQuestion(r.Context(), chi.URLParam(r, "project_id"), chi.URLParam(r, "question_id")); err != nil {
		writeUnitError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *Router) handleResearchPrompt(w http.ResponseWriter, r *http.Request) {
	if !rt.researchEnabled(w) {
		return
	}
	prompt, err := rt.research.ResearchPrompt(r.Context(), chi.URLParam(r, "project_id"))
	if err != nil {
		writeUnitError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"prompt": prompt})
}

func (rt *Router) handleFollowUpPrompt(w http.ResponseWriter, r *http.Request) {
	if !rt.researchEnabled(w) {
		return
	}
	prompt, err := rt.research.FollowUpPrompt(r.Context(), chi.URLParam(r, "project_id"))
	if err != nil {
		writeUnitError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"prompt": prompt})
}

func (rt *Router) handlePasteResearch(w http.ResponseWriter, r *http.Request) {
	if !rt.researchEnabled(w) {
		return
	}
	var body struct {
		Raw string `json:"raw"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	result, err := rt.research.IngestPaste(r.Context(), chi.URLParam(r, "project_id"), body.Raw)
	if err != nil {
		writeUnitError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"added": result.Added, "updated": result.Updated, "rejected": result.Rejected, "log": result.Log,
	})
}

func (rt *Router) handleAddFact(w http.ResponseWriter, r *http.Request) {
	if !rt.researchEnabled(w) {
		return
	}
	var body struct {
		Question    string          `json:"question"`
		ClaimVI     string          `json:"claim_vi"`
		Type        domain.FactType `json:"type"`
		SourceTitle string          `json:"source_title"`
		Publisher   string          `json:"publisher"`
		Year        string          `json:"year"`
		URL         string          `json:"url"`
		Quote       string          `json:"quote"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	fact, err := rt.research.AddFact(r.Context(), chi.URLParam(r, "project_id"), application.FactInput{
		Question: body.Question, ClaimVI: body.ClaimVI, Type: body.Type, SourceTitle: body.SourceTitle,
		Publisher: body.Publisher, Year: body.Year, URL: body.URL, Quote: body.Quote,
	})
	if err != nil {
		writeUnitError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, fact)
}

func (rt *Router) handleSetFactFlags(w http.ResponseWriter, r *http.Request) {
	if !rt.researchEnabled(w) {
		return
	}
	var body struct {
		Type *domain.FactType `json:"type"`
		Must *bool            `json:"must"`
		Keep *bool            `json:"keep"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	fact, err := rt.research.SetFactFlags(r.Context(), chi.URLParam(r, "project_id"), chi.URLParam(r, "fact_id"),
		application.FactFlags{Type: body.Type, Must: body.Must, Keep: body.Keep})
	if err != nil {
		writeUnitError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, fact)
}

func (rt *Router) handleVerifyFacts(w http.ResponseWriter, r *http.Request) {
	if !rt.researchEnabled(w) {
		return
	}
	var body struct {
		FactIDs []string `json:"fact_ids"`
	}
	if r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}
	if err := rt.research.Verify(r.Context(), chi.URLParam(r, "project_id"), body.FactIDs); err != nil {
		writeUnitError(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (rt *Router) handleSkipResearch(w http.ResponseWriter, r *http.Request) {
	if !rt.researchEnabled(w) {
		return
	}
	var body struct {
		Skipped bool `json:"skipped"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	gate, err := rt.research.SetSkipped(r.Context(), chi.URLParam(r, "project_id"), body.Skipped)
	if err != nil {
		writeUnitError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, gate)
}
