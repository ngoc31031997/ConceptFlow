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

// operationStory is the kind a story AI run is recorded under, so that any page can follow it.
const operationStory = "story"

// storyUseCase is the story step.
type storyUseCase interface {
	State(ctx context.Context, projectID string) (application.StoryState, error)
	Generate(ctx context.Context, projectID, note string) (domain.StoredStory, error)
	Edit(ctx context.Context, projectID string, patch application.StoryPatch) (domain.StoredStory, error)
	Approve(ctx context.Context, projectID string) (domain.StoredStory, error)
	Unapprove(ctx context.Context, projectID string) (domain.StoredStory, error)
}

// WithStory enables the /story routes.
func (rt *Router) WithStory(uc storyUseCase) *Router {
	rt.story = uc
	return rt
}

func (rt *Router) storyRoutes(r chi.Router) {
	const base = "/v1/projects/{project_id}/story"
	r.Get(base, rt.handleGetStory)
	r.Post(base+"/generate", rt.handleGenerateStory)
	r.Put(base, rt.handleEditStory)
	r.Post(base+"/approve", rt.handleApproveStory)
	r.Post(base+"/unapprove", rt.handleUnapproveStory)
}

// storyEnabled answers 404 when the story step is not wired.
func (rt *Router) storyEnabled(w http.ResponseWriter) bool {
	if rt.story == nil {
		writeError(w, http.StatusNotFound, "story is not enabled")
		return false
	}
	return true
}

// storyDTO is a story as the screen reads it.
type storyDTO struct {
	CentralQuestion string             `json:"central_question"`
	CoreAnswer      string             `json:"core_answer"`
	NarrativeArc    []domain.ArcBeat   `json:"narrative_arc"`
	CreatorFields   []string           `json:"creator_fields"`
	Status          domain.StoryStatus `json:"status"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

func toStoryDTO(s domain.StoredStory) storyDTO {
	story := s.Story.Normalized()
	creator := s.CreatorFields
	if creator == nil {
		creator = []string{}
	}
	return storyDTO{
		CentralQuestion: story.CentralQuestion, CoreAnswer: story.CoreAnswer, NarrativeArc: story.NarrativeArc,
		CreatorFields: creator, Status: s.Status, UpdatedAt: s.UpdatedAt,
	}
}

func (rt *Router) handleGetStory(w http.ResponseWriter, r *http.Request) {
	if !rt.storyEnabled(w) {
		return
	}
	state, err := rt.story.State(r.Context(), chi.URLParam(r, "project_id"))
	if err != nil {
		writeUnitError(w, err)
		return
	}
	var story *storyDTO
	if state.Story != nil {
		dto := toStoryDTO(*state.Story)
		story = &dto
	}
	facts := state.Facts
	if facts == nil {
		facts = []application.StoryFactView{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"story":             story,
		"facts":             facts,
		"brief":             briefWire(state.Brief),
		"brief_approved":    state.Brief != nil,
		"research_resolved": state.ResearchResolved,
		"research_skipped":  state.ResearchSkipped,
		"brief_stale":       state.BriefStale,
		"research_stale":    state.ResearchStale,
	})
}

func (rt *Router) handleGenerateStory(w http.ResponseWriter, r *http.Request) {
	if !rt.storyEnabled(w) {
		return
	}
	note, err := readNote(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	ctx, finish := rt.trackOperation(r, operationStory)
	story, err := rt.story.Generate(ctx, chi.URLParam(r, "project_id"), note)
	finish(err)
	if err != nil {
		writeUnitRunError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toStoryDTO(story))
}

// editStoryRequest is PUT /v1/projects/{id}/story; an absent field is left alone.
type editStoryRequest struct {
	CentralQuestion *string           `json:"central_question"`
	CoreAnswer      *string           `json:"core_answer"`
	NarrativeArc    *[]domain.ArcBeat `json:"narrative_arc"`
}

func (rt *Router) handleEditStory(w http.ResponseWriter, r *http.Request) {
	if !rt.storyEnabled(w) {
		return
	}
	var req editStoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	story, err := rt.story.Edit(r.Context(), chi.URLParam(r, "project_id"), application.StoryPatch{
		CentralQuestion: req.CentralQuestion, CoreAnswer: req.CoreAnswer, NarrativeArc: req.NarrativeArc,
	})
	if err != nil {
		writeUnitError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toStoryDTO(story))
}

func (rt *Router) handleApproveStory(w http.ResponseWriter, r *http.Request) {
	rt.storyStatusChange(w, r, func(ctx context.Context, id string) (domain.StoredStory, error) {
		return rt.story.Approve(ctx, id)
	})
}

func (rt *Router) handleUnapproveStory(w http.ResponseWriter, r *http.Request) {
	rt.storyStatusChange(w, r, func(ctx context.Context, id string) (domain.StoredStory, error) {
		return rt.story.Unapprove(ctx, id)
	})
}

func (rt *Router) storyStatusChange(w http.ResponseWriter, r *http.Request, change func(context.Context, string) (domain.StoredStory, error)) {
	if !rt.storyEnabled(w) {
		return
	}
	story, err := change(r.Context(), chi.URLParam(r, "project_id"))
	if err != nil {
		writeUnitError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toStoryDTO(story))
}
