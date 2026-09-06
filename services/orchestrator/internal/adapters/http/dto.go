// Package http implements the REST adapter: a chi router exposing the 4
// endpoints described in interface-contracts.md plus GET /health, translating
// between JSON request/response bodies and application-layer use case
// input/output types.
package http

import (
	"time"

	"orchestrator/internal/domain"
)

// startRenderSagaRequest is the body of POST /v1/sagas/render.
type startRenderSagaRequest struct {
	ProjectID           string  `json:"project_id"`
	ScriptContent       string  `json:"script_content"`
	PluginID            string  `json:"plugin_id"`
	CategoryHint        string  `json:"category_hint"`
	VoiceLanguage       string  `json:"voice_language"`
	BackgroundMusicPath *string `json:"background_music_path,omitempty"`
}

// startPublishSagaRequest is the body of POST /v1/sagas/publish.
type startPublishSagaRequest struct {
	ProjectID    string   `json:"project_id"`
	YoutubeTitle string   `json:"youtube_title"`
	Description  *string  `json:"description,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	Visibility   string   `json:"visibility"`
}

// sagaStartedResponse is the 201 response shape shared by both saga-start
// endpoints.
type sagaStartedResponse struct {
	SagaID string `json:"saga_id"`
	Status string `json:"status"`
}

// retryResponse is the 200 response of POST /v1/projects/{project_id}/retry.
type retryResponse struct {
	SagaID string `json:"saga_id"`
	Status string `json:"status"`
}

// sceneResponse mirrors domain.Scene for the GET /v1/projects/{id} response.
type sceneResponse struct {
	SceneIndex          int     `json:"scene_index"`
	NarrationText       string  `json:"narration_text"`
	IllustrationHint    string  `json:"illustration_hint"`
	CodeSnippet         *string `json:"code_snippet,omitempty"`
	CodeLanguage        *string `json:"code_language,omitempty"`
	Category            string  `json:"category,omitempty"`
	AnimationTemplateID string  `json:"animation_template_id,omitempty"`
	AudioPath           string  `json:"audio_path,omitempty"`
	DurationSeconds     float64 `json:"duration_seconds,omitempty"`
	ClipPath            string  `json:"clip_path,omitempty"`
}

// projectResponse is the GET /v1/projects/{project_id} response
// (interface-contracts.md).
type projectResponse struct {
	ProjectID       string          `json:"project_id"`
	Status          string          `json:"status"`
	VideoPath       *string         `json:"video_path,omitempty"`
	Scenes          []sceneResponse `json:"scenes"`
	PluginID        string          `json:"plugin_id"`
	CategoryHint    string          `json:"category_hint"`
	VoiceLanguage   string          `json:"voice_language"`
	YoutubeVideoURL *string         `json:"youtube_video_url,omitempty"`
	ErrorMessage    *string         `json:"error_message,omitempty"`
}

// projectSummaryResponse is one entry of the GET /v1/projects (list) response.
type projectSummaryResponse struct {
	ProjectID    string  `json:"project_id"`
	Status       string  `json:"status"`
	VideoPath    *string `json:"video_path,omitempty"`
	ErrorMessage *string `json:"error_message,omitempty"`
	UpdatedAt    string  `json:"updated_at"`
}

// projectListResponse is the GET /v1/projects response body.
type projectListResponse struct {
	Projects []projectSummaryResponse `json:"projects"`
}

// errorResponse is the JSON body for non-2xx responses.
type errorResponse struct {
	Error string `json:"error"`
}

func toProjectListResponse(summaries []domain.ProjectSummary) projectListResponse {
	projects := make([]projectSummaryResponse, 0, len(summaries))
	for _, s := range summaries {
		projects = append(projects, projectSummaryResponse{
			ProjectID:    s.ProjectID,
			Status:       string(s.Status),
			VideoPath:    s.VideoPath,
			ErrorMessage: s.ErrorMessage,
			UpdatedAt:    s.UpdatedAt.Format(time.RFC3339),
		})
	}
	return projectListResponse{Projects: projects}
}

func toProjectResponse(p *domain.Project) projectResponse {
	scenes := make([]sceneResponse, 0, len(p.Scenes))
	for _, s := range p.Scenes {
		scenes = append(scenes, sceneResponse{
			SceneIndex:          s.SceneIndex,
			NarrationText:       s.NarrationText,
			IllustrationHint:    s.IllustrationHint,
			CodeSnippet:         s.CodeSnippet,
			CodeLanguage:        s.CodeLanguage,
			Category:            s.Category,
			AnimationTemplateID: s.AnimationTemplateID,
			AudioPath:           s.AudioPath,
			DurationSeconds:     s.DurationSeconds,
			ClipPath:            s.ClipPath,
		})
	}
	return projectResponse{
		ProjectID:       p.ProjectID,
		Status:          string(p.Status),
		VideoPath:       p.VideoPath,
		Scenes:          scenes,
		PluginID:        p.PluginID,
		CategoryHint:    p.CategoryHint,
		VoiceLanguage:   string(p.VoiceLanguage),
		YoutubeVideoURL: p.YoutubeVideoURL,
		ErrorMessage:    p.ErrorMessage,
	}
}
