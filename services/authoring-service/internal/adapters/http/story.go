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

// Operation kinds a story AI run is recorded under, so that any page can follow it.
const (
	operationStoryApproach = "story_approach"
	operationStory         = "story"
)

// storyUseCase is the story step beyond the generic unit API.
type storyUseCase interface {
	State(ctx context.Context, projectID string) (application.StoryState, error)
	GenerateApproaches(ctx context.Context, projectID, note string) ([]application.StoryApproachView, error)
	SelectApproach(ctx context.Context, projectID, key string) error
	SaveApproachAsArchetype(ctx context.Context, projectID, key string) (application.StoryApproachView, error)
	GenerateFrame(ctx context.Context, projectID, note string) error
	SetFactUse(ctx context.Context, projectID, factID string, use bool) error
}

// WithStory enables the /story routes.
func (rt *Router) WithStory(uc storyUseCase) *Router {
	rt.story = uc
	return rt
}

func (rt *Router) storyRoutes(r chi.Router) {
	const base = "/v1/projects/{project_id}/story"
	r.Get(base, rt.handleGetStory)
	r.Post(base+"/approaches/generate", rt.handleGenerateApproaches)
	r.Put(base+"/approach", rt.handleSelectApproach)
	r.Post(base+"/approaches/{key}/archetype", rt.handleSaveApproachArchetype)
	r.Post(base+"/generate", rt.handleGenerateStory)
	r.Put(base+"/facts/{fact_id}", rt.handleSetStoryFactUse)
}

// storyEnabled answers 404 when the story step is not wired.
func (rt *Router) storyEnabled(w http.ResponseWriter) bool {
	if rt.story == nil {
		writeError(w, http.StatusNotFound, "story is not enabled")
		return false
	}
	return true
}

// writeStoryRunError maps the errors of an AI run of the story step.
func writeStoryRunError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrStoryReplyUnusable), errors.Is(err, domain.ErrApproachReplyUnusable):
		writeError(w, http.StatusBadGateway, err.Error()+". Thử làm lại, hoặc tự viết các ô.")
	case errors.Is(err, application.ErrLLMNotConfigured):
		writeError(w, http.StatusServiceUnavailable,
			"Chưa cấu hình HIVE_API_KEY trong .env (llm-service) nên AI không viết được. Bạn tự viết các ô rồi duyệt.")
	default:
		writeUnitRunError(w, err)
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
	writeJSON(w, http.StatusOK, state)
}

// noteBody reads the optional {"note": "..."} body of an AI run.
func noteBody(w http.ResponseWriter, r *http.Request) (string, bool) {
	var body struct {
		Note string `json:"note"`
	}
	if r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return "", false
		}
	}
	return body.Note, true
}

func (rt *Router) handleGenerateApproaches(w http.ResponseWriter, r *http.Request) {
	if !rt.storyEnabled(w) {
		return
	}
	note, ok := noteBody(w, r)
	if !ok {
		return
	}
	ctx, finish := rt.trackOperation(r, operationStoryApproach)
	approaches, err := rt.story.GenerateApproaches(ctx, chi.URLParam(r, "project_id"), note)
	finish(err)
	if err != nil {
		writeStoryRunError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"approaches": approaches})
}

func (rt *Router) handleSelectApproach(w http.ResponseWriter, r *http.Request) {
	if !rt.storyEnabled(w) {
		return
	}
	var body struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := rt.story.SelectApproach(r.Context(), chi.URLParam(r, "project_id"), body.Key); err != nil {
		writeUnitError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (rt *Router) handleSaveApproachArchetype(w http.ResponseWriter, r *http.Request) {
	if !rt.storyEnabled(w) {
		return
	}
	approach, err := rt.story.SaveApproachAsArchetype(r.Context(), chi.URLParam(r, "project_id"), chi.URLParam(r, "key"))
	if err != nil {
		writeUnitError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, approach)
}

func (rt *Router) handleGenerateStory(w http.ResponseWriter, r *http.Request) {
	if !rt.storyEnabled(w) {
		return
	}
	note, ok := noteBody(w, r)
	if !ok {
		return
	}
	ctx, finish := rt.trackOperation(r, operationStory)
	err := rt.story.GenerateFrame(ctx, chi.URLParam(r, "project_id"), note)
	finish(err)
	if err != nil {
		writeStoryRunError(w, err)
		return
	}
	state, err := rt.story.State(r.Context(), chi.URLParam(r, "project_id"))
	if err != nil {
		writeUnitError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (rt *Router) handleSetStoryFactUse(w http.ResponseWriter, r *http.Request) {
	if !rt.storyEnabled(w) {
		return
	}
	var body struct {
		Use *bool `json:"use"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Use == nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := rt.story.SetFactUse(r.Context(), chi.URLParam(r, "project_id"), chi.URLParam(r, "fact_id"), *body.Use); err != nil {
		writeUnitError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
