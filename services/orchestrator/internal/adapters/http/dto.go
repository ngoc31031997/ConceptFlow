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
	ContentLanguage     string  `json:"voice_language"`
	BackgroundMusicPath *string `json:"background_music_path,omitempty"`

	// CR-001. TTSEnabled is a pointer so an omitted field keeps the pre-CR-001
	// default (narration on) instead of decoding to false.
	TTSEnabled       *bool  `json:"tts_enabled,omitempty"`
	VoiceID          string `json:"voice_id,omitempty"`
	SubtitlesEnabled bool   `json:"subtitles_enabled,omitempty"`
	// CR-015 FR41 — "off" | "track" | "burn_in" | "both". Wins over
	// SubtitlesEnabled when both are present (start_render_saga.go).
	SubtitleMode          string                `json:"subtitle_mode,omitempty"`
	SubtitleStyle         *domain.SubtitleStyle `json:"subtitle_style,omitempty"`
	RenderQuality         string                `json:"render_quality,omitempty"`
	VideoFormatID         string                `json:"video_format_id,omitempty"`
	ReviewEnabled         *bool                 `json:"review_enabled,omitempty"`
	BackgroundMusicVolume float64               `json:"background_music_volume,omitempty"`
}

// startPublishSagaRequest is the body of POST /v1/sagas/publish.
type startPublishSagaRequest struct {
	ProjectID     string   `json:"project_id"`
	YoutubeTitle  string   `json:"youtube_title"`
	Description   *string  `json:"description,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	Visibility    string   `json:"visibility"`
	PublishAt     *string  `json:"publish_at,omitempty"`
	ThumbnailPath *string  `json:"thumbnail_path,omitempty"`
	ChannelID     *string  `json:"channel_id,omitempty"`
	// CR-021 FR61.3 — the conscious action that gets past a blocking QC
	// finding. Absent means "no": an override has to be asked for.
	AcknowledgeQC bool `json:"acknowledge_qc,omitempty"`
}

// qcFindingResponse is one entry of the QC report response
// (GET /v1/projects/{project_id}/qc-report).
type qcFindingResponse struct {
	Rule             string  `json:"rule"`
	Severity         string  `json:"severity"`
	Message          string  `json:"message"`
	TimestampSeconds float64 `json:"timestamp_seconds"`
}

// qcReportResponse is the GET /v1/projects/{project_id}/qc-report body
// (CR-021 FR61.1/FR61.2).
//
// Status "not_scored" with an empty findings list is a normal, successful
// response, not an error — the GUI says so rather than showing a green tick the
// measurement never earned.
type qcReportResponse struct {
	ProjectID string              `json:"project_id"`
	Status    string              `json:"status"`
	Reason    *string             `json:"reason,omitempty"`
	Findings  []qcFindingResponse `json:"findings"`
	CreatedAt *string             `json:"created_at,omitempty"`
	// OverriddenAt is set once the Creator published past a blocking finding
	// (FR61.3), so the record of that decision is visible where the findings are.
	OverriddenAt *string `json:"overridden_at,omitempty"`
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
	ProjectID        string                  `json:"project_id"`
	Status           string                  `json:"status"`
	VideoPath        *string                 `json:"video_path,omitempty"`
	Scenes           []sceneResponse         `json:"scenes"`
	PluginID         string                  `json:"plugin_id"`
	CategoryHint     string                  `json:"category_hint"`
	ContentLanguage  string                  `json:"voice_language"`
	TTSEnabled       bool                    `json:"tts_enabled"`
	VoiceID          string                  `json:"voice_id,omitempty"`
	SubtitlesEnabled bool                    `json:"subtitles_enabled"`
	SubtitleMode     string                  `json:"subtitle_mode"`
	SubtitleStyle    *domain.SubtitleStyle   `json:"subtitle_style,omitempty"`
	RenderQuality    string                  `json:"render_quality"`
	VideoFormatID    string                  `json:"video_format_id"`
	ReviewEnabled    bool                    `json:"review_enabled"`
	Beats            []domain.BeatOccurrence `json:"beats"`
	Warnings         []string                `json:"validation_warnings"`
	VideoFormatVer   int                     `json:"video_format_version"`
	YoutubeVideoURL  *string                 `json:"youtube_video_url,omitempty"`
	// CR-015 FR39.4 — absent when no caption was requested, otherwise
	// "uploaded" | "skipped_no_scope" | "failed".
	CaptionStatus *string `json:"caption_status,omitempty"`
	ErrorMessage  *string `json:"error_message,omitempty"`
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

// suggestMetadataResponse is the 200 response of
// POST /v1/projects/{project_id}/suggest-metadata.
type suggestMetadataResponse struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

// errorResponse is the JSON body for non-2xx responses.
//
// Code is set only where the client has to branch on *which* failure it was,
// not merely report it. Publishing returns 409 both for a project in the wrong
// status and for a QC block (CR-021 FR61.3), and only the second one offers the
// Creator a way through — leaving the GUI to tell them apart by matching on
// prose would break the moment the wording changes.
type errorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

// ErrorCodeQCBlocked marks the 409 that `acknowledge_qc: true` can get past.
const ErrorCodeQCBlocked = "qc_blocked"

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
		ProjectID:        p.ProjectID,
		Status:           string(p.Status),
		VideoPath:        p.VideoPath,
		Scenes:           scenes,
		PluginID:         p.PluginID,
		CategoryHint:     p.CategoryHint,
		ContentLanguage:  string(p.ContentLanguage),
		TTSEnabled:       p.TTSEnabled,
		VoiceID:          p.VoiceID,
		SubtitlesEnabled: p.SubtitlesEnabled,
		SubtitleMode:     string(p.SubtitleMode),
		SubtitleStyle:    p.SubtitleStyle,
		RenderQuality:    string(p.RenderQuality),
		VideoFormatID:    p.VideoFormatID,
		ReviewEnabled:    p.ReviewEnabled,
		Beats:            p.Beats,
		Warnings:         p.ValidationWarnings,
		VideoFormatVer:   p.VideoFormatVersion,
		YoutubeVideoURL:  p.YoutubeVideoURL,
		CaptionStatus:    p.CaptionStatus,
		ErrorMessage:     p.ErrorMessage,
	}
}
