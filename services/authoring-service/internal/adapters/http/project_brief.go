package http

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"authoring/internal/application"
	"authoring/internal/domain"
)

// WithProjectBrief enables the /creative-brief routes.
func (rt *Router) WithProjectBrief(uc projectBriefUseCase) *Router {
	rt.projectBrief = uc
	return rt
}

func (rt *Router) projectBriefRoutes(r chi.Router) {
	r.Get("/v1/projects/{project_id}/creative-brief", rt.handleGetProjectBrief)
	r.Post("/v1/projects/{project_id}/creative-brief/generate", rt.handleGenerateProjectBrief)
	r.Put("/v1/projects/{project_id}/creative-brief", rt.handleEditProjectBrief)
	r.Post("/v1/projects/{project_id}/creative-brief/approve", rt.handleApproveProjectBrief)
	r.Post("/v1/projects/{project_id}/creative-brief/unapprove", rt.handleUnapproveProjectBrief)
	r.Post("/v1/projects/{project_id}/creative-brief/review", rt.handleReviewProjectBrief)
}

// projectBriefDTO is a brief as the screen reads it.
type projectBriefDTO struct {
	Audience           string     `json:"audience"`
	DurationSeconds    int        `json:"duration_seconds"`
	Tone               string     `json:"tone"`
	LearningObjectives []string   `json:"learning_objectives"`
	Avoid              []string   `json:"avoid"`
	CreatorFields      []string   `json:"creator_fields"`
	Status             string     `json:"status"`
	RedoCount          int        `json:"redo_count"`
	ApprovedAt         *time.Time `json:"approved_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

func toProjectBriefDTO(s domain.StoredBrief) projectBriefDTO {
	brief := s.Brief.Normalized()
	creator := s.CreatorFields
	if creator == nil {
		creator = []string{}
	}
	return projectBriefDTO{
		Audience: brief.Audience, DurationSeconds: brief.DurationSeconds, Tone: brief.Tone,
		LearningObjectives: brief.LearningObjectives, Avoid: brief.Avoid, CreatorFields: creator,
		Status: string(s.Status), RedoCount: s.RedoCount, ApprovedAt: s.ApprovedAt, UpdatedAt: s.UpdatedAt,
	}
}

// approvedIdeaDTO is the approved idea the brief is written from.
type approvedIdeaDTO struct {
	CentralQuestion string   `json:"central_question"`
	CuriosityPoints []string `json:"curiosity_points"`
	Scope           string   `json:"scope"`
}

// projectBriefStateResponse is GET /v1/projects/{id}/creative-brief. idea is
// null until the first step is approved; idea_changed says the approved idea is
// not the one the brief was written from; review_stale says the brief changed
// after the review read it.
type projectBriefStateResponse struct {
	Brief              *projectBriefDTO    `json:"brief"`
	Idea               *approvedIdeaDTO    `json:"idea"`
	Subject            string              `json:"subject"`
	Language           string              `json:"language"`
	VideoMode          string              `json:"video_mode"`
	MinDurationSeconds int                 `json:"min_duration_seconds"`
	MaxDurationSeconds int                 `json:"max_duration_seconds"`
	IdeaChanged        bool                `json:"idea_changed"`
	Review             *domain.BriefReview `json:"review"`
	ReviewStale        bool                `json:"review_stale"`
}

func (rt *Router) handleGetProjectBrief(w http.ResponseWriter, r *http.Request) {
	if rt.projectBrief == nil {
		writeError(w, http.StatusNotFound, "creative brief is not enabled")
		return
	}
	state, err := rt.projectBrief.State(r.Context(), chi.URLParam(r, "project_id"))
	if err != nil {
		writeUnitError(w, err)
		return
	}
	resp := projectBriefStateResponse{
		Subject: state.Subject, Language: string(state.Language), VideoMode: string(state.VideoMode),
		MinDurationSeconds: state.MinDurationSeconds, MaxDurationSeconds: state.MaxDurationSeconds,
		IdeaChanged: state.IdeaChanged, Review: state.Review, ReviewStale: state.ReviewStale,
	}
	if state.Brief != nil {
		dto := toProjectBriefDTO(*state.Brief)
		resp.Brief = &dto
	}
	if state.Idea != nil {
		idea := state.Idea.Normalized()
		resp.Idea = &approvedIdeaDTO{CentralQuestion: idea.CentralQuestion, CuriosityPoints: idea.CuriosityPoints, Scope: idea.Scope}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (rt *Router) handleGenerateProjectBrief(w http.ResponseWriter, r *http.Request) {
	if rt.projectBrief == nil {
		writeError(w, http.StatusNotFound, "creative brief is not enabled")
		return
	}
	note, err := readNote(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	ctx, finish := rt.trackOperation(r, "brief")
	brief, err := rt.projectBrief.Generate(ctx, chi.URLParam(r, "project_id"), note)
	finish(err)
	if err != nil {
		writeUnitRunError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toProjectBriefDTO(brief))
}

// editProjectBriefRequest is PUT /v1/projects/{id}/creative-brief; an absent
// field is left alone.
type editProjectBriefRequest struct {
	Audience           *string   `json:"audience"`
	DurationSeconds    *int      `json:"duration_seconds"`
	Tone               *string   `json:"tone"`
	LearningObjectives *[]string `json:"learning_objectives"`
	Avoid              *[]string `json:"avoid"`
}

func (rt *Router) handleEditProjectBrief(w http.ResponseWriter, r *http.Request) {
	if rt.projectBrief == nil {
		writeError(w, http.StatusNotFound, "creative brief is not enabled")
		return
	}
	var req editProjectBriefRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	brief, err := rt.projectBrief.Edit(r.Context(), chi.URLParam(r, "project_id"), application.BriefPatch{
		Audience: req.Audience, DurationSeconds: req.DurationSeconds, Tone: req.Tone,
		LearningObjectives: req.LearningObjectives, Avoid: req.Avoid,
	})
	if err != nil {
		writeUnitError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toProjectBriefDTO(brief))
}

func (rt *Router) handleApproveProjectBrief(w http.ResponseWriter, r *http.Request) {
	rt.projectBriefStatusChange(w, r, func(ctx context.Context, id string) (domain.StoredBrief, error) {
		return rt.projectBrief.Approve(ctx, id)
	})
}

func (rt *Router) handleUnapproveProjectBrief(w http.ResponseWriter, r *http.Request) {
	rt.projectBriefStatusChange(w, r, func(ctx context.Context, id string) (domain.StoredBrief, error) {
		return rt.projectBrief.Unapprove(ctx, id)
	})
}

func (rt *Router) projectBriefStatusChange(w http.ResponseWriter, r *http.Request, change func(context.Context, string) (domain.StoredBrief, error)) {
	if rt.projectBrief == nil {
		writeError(w, http.StatusNotFound, "creative brief is not enabled")
		return
	}
	brief, err := change(r.Context(), chi.URLParam(r, "project_id"))
	if err != nil {
		writeUnitError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toProjectBriefDTO(brief))
}

func (rt *Router) handleReviewProjectBrief(w http.ResponseWriter, r *http.Request) {
	if rt.projectBrief == nil {
		writeError(w, http.StatusNotFound, "creative brief is not enabled")
		return
	}
	ctx, finish := rt.trackOperation(r, "brief_review")
	review, err := rt.projectBrief.Review(ctx, chi.URLParam(r, "project_id"))
	finish(err)
	if err != nil {
		writeUnitRunError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, review)
}
