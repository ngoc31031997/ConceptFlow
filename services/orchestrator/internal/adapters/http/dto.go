// Package http implements the REST adapter: a chi router exposing the 4
// endpoints described in interface-contracts.md plus GET /health, translating
// between JSON request/response bodies and application-layer use case
// input/output types.
package http

import (
	"time"

	"orchestrator/internal/application"
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
	SubtitleMode  string                `json:"subtitle_mode,omitempty"`
	SubtitleStyle *domain.SubtitleStyle `json:"subtitle_style,omitempty"`
	RenderQuality string                `json:"render_quality,omitempty"`
	// "manim" | "remotion" — empty means DefaultRenderEngine ("manim").
	RenderEngine          string  `json:"render_engine,omitempty"`
	VideoFormatID         string  `json:"video_format_id,omitempty"`
	ReviewEnabled         *bool   `json:"review_enabled,omitempty"`
	BackgroundMusicVolume float64 `json:"background_music_volume,omitempty"`
	VideoFont             string  `json:"video_font,omitempty"`
	// "long" | "short" | "both" — empty means DefaultVideoOutputMode ("long").
	VideoOutputMode string `json:"video_output_mode,omitempty"`
	// CR-026 D1 — project_id of the companion video covering the same
	// topic (the other of the long-form/short-form pair), if any.
	CompanionProjectID *string `json:"companion_project_id,omitempty"`
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
	// FlowStep/RunState place the project in the 13-step flow (see
	// domain.FlowStateFor); the GUI resumes and locks from these, not from
	// wizard_route.
	FlowStep int    `json:"flow_step"`
	RunState string `json:"run_state"`
	// ForkedFrom is the project this one was forked from ("" if none).
	ForkedFrom       string                  `json:"forked_from,omitempty"`
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
	RenderEngine     string                  `json:"render_engine"`
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
	// CR-007 D7 — the vertical clips generate_clips produced, if the saga has
	// reached that step yet.
	Clips []clipResultResponse `json:"clips,omitempty"`
	// ScriptContent, BackgroundMusicPath and BackgroundMusicVolume round out
	// what StartRenderSagaInput needs — the GUI's "render lại ở chất lượng
	// khác" (bug report) resubmits POST /v1/sagas/render for this same
	// project_id once it is done, and it can only carry over settings it can
	// actually read back from here.
	ScriptContent         string  `json:"script_content"`
	BackgroundMusicPath   *string `json:"background_music_path,omitempty"`
	BackgroundMusicVolume float64 `json:"background_music_volume,omitempty"`
	VideoFont             string  `json:"video_font,omitempty"`
	// "long" | "short" | "both" (CR-007 follow-up) — which output(s) the
	// Result screen should feature, and whether generate_clips ran at all.
	VideoOutputMode string `json:"video_output_mode"`
	// CR-026 D1/D6 — id only, not the nested project: the GUI re-fetches it
	// through the same GET /v1/projects/{id} it already calls for anything
	// else, rather than orchestrator embedding one project inside another.
	CompanionProjectID *string `json:"companion_project_id,omitempty"`
	// WizardStep (1-7) is where the Creator should resume: the furthest step
	// confirmed with "Tiếp tục", or the one the saga status implies.
	WizardStep int `json:"wizard_step"`
	// WizardRoute is the wizard screen last open on a draft ("" if unknown).
	WizardRoute string `json:"wizard_route,omitempty"`
	// SubtitleStyle/BackgroundMusic* above already round-trip step 2 settings.
}

// projectSummaryResponse is one entry of the GET /v1/projects (list) response.
type projectSummaryResponse struct {
	ProjectID    string  `json:"project_id"`
	Status       string  `json:"status"`
	VideoPath    *string `json:"video_path,omitempty"`
	ErrorMessage *string `json:"error_message,omitempty"`
	UpdatedAt    string  `json:"updated_at"`
	// "manim" | "remotion" — which engine rendered (or will render) this
	// project's video, so the video list can mark which is which.
	RenderEngine string `json:"render_engine"`
	// WizardStep (1-7) is the step the project is at, so the list can show
	// "Bước N — …" next to the saga status.
	WizardStep int `json:"wizard_step"`
	// Topic names the project by its idea; FlowStep/RunState place it in the
	// 13-step flow; ForkedFrom links a fork to its source.
	Topic      string `json:"topic,omitempty"`
	FlowStep   int    `json:"flow_step"`
	RunState   string `json:"run_state"`
	ForkedFrom string `json:"forked_from,omitempty"`
}

// projectListResponse is the GET /v1/projects response body.
type projectListResponse struct {
	Projects []projectSummaryResponse `json:"projects"`
}

// suggestShortScriptRequest is the body of POST /v1/short-script-suggestions
// (CR-026 FR71.1). No project_id: a Creator can draft a short from a bare
// topic without an existing project. SourceScriptContent is optional context
// pulled from an existing long-form project when called from its Result
// screen — the topic alone is enough to draft something without it.
type suggestShortScriptRequest struct {
	Topic               string `json:"topic"`
	Language            string `json:"language"`
	SourceScriptContent string `json:"source_script_content,omitempty"`
}

// suggestShortScriptResponse is the 200 response of
// POST /v1/short-script-suggestions.
type suggestShortScriptResponse struct {
	ScriptContent string `json:"script_content"`
}

// suggestMetadataResponse is the 200 response of
// POST /v1/projects/{project_id}/suggest-metadata.
type suggestMetadataResponse struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

// createClipRequest is the body of POST /v1/projects/{project_id}/clips
// (CR-007 FR19.2/D3/D7) — a Creator-entered clip selection.
type createClipRequest struct {
	Name         string   `json:"name"`
	StartSeconds float64  `json:"start_seconds"`
	EndSeconds   float64  `json:"end_seconds"`
	Presets      []string `json:"presets"`
}

// clipResultResponse mirrors domain.ClipResult for
// GET /v1/projects/{project_id}/clips (CR-007 D7/FR20.1).
type clipResultResponse struct {
	Name            string  `json:"name"`
	Preset          string  `json:"preset"`
	Status          string  `json:"status"`
	OutputPath      string  `json:"output_path,omitempty"`
	DurationSeconds float64 `json:"duration_seconds,omitempty"`
	ErrorMessage    string  `json:"error_message,omitempty"`
}

// createClipResponse is the 200/202 response of
// POST /v1/projects/{project_id}/clips. AcceptedPresets/RejectedPresets let
// the GUI show exactly which preset(s) were saved and which were refused and
// why (FR19.7 — one bad preset must never sink the request the Creator did
// get right).
type createClipResponse struct {
	Name            string            `json:"name"`
	AcceptedPresets []string          `json:"accepted_presets"`
	RejectedPresets map[string]string `json:"rejected_presets,omitempty"`
}

func toClipResultResponses(clips []domain.ClipResult) []clipResultResponse {
	out := make([]clipResultResponse, 0, len(clips))
	for _, c := range clips {
		out = append(out, clipResultResponse{
			Name: c.Name, Preset: c.Preset, Status: c.Status,
			OutputPath: c.OutputPath, DurationSeconds: c.DurationSeconds, ErrorMessage: c.ErrorMessage,
		})
	}
	return out
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

// patchWizardSettingsRequest is the body of PATCH /v1/projects/{id}/settings —
// wizard step 2 ("Cấu hình"), sent per field as the Creator changes it. Only
// the fields present are written. Field names match POST /v1/sagas/render so
// the two never drift. background_music_path "" clears the track; confirm
// (the "Tiếp tục" press) moves the project on to step 3.
type patchWizardSettingsRequest struct {
	ContentLanguage       *string               `json:"voice_language,omitempty"`
	RenderEngine          *string               `json:"render_engine,omitempty"`
	TTSEnabled            *bool                 `json:"tts_enabled,omitempty"`
	VoiceID               *string               `json:"voice_id,omitempty"`
	SubtitleMode          *string               `json:"subtitle_mode,omitempty"`
	SubtitleStyle         *domain.SubtitleStyle `json:"subtitle_style,omitempty"`
	RenderQuality         *string               `json:"render_quality,omitempty"`
	VideoFormatID         *string               `json:"video_format_id,omitempty"`
	VideoOutputMode       *string               `json:"video_output_mode,omitempty"`
	BackgroundMusicPath   *string               `json:"background_music_path,omitempty"`
	BackgroundMusicVolume *float64              `json:"background_music_volume,omitempty"`
	VideoFont             *string               `json:"video_font,omitempty"`
	Confirm               bool                  `json:"confirm,omitempty"`
}

// saveAuthoringModeRequest is the body of PUT
// /v1/projects/{id}/authoring/mode (CR-027 FR79): "manual" or "ai".
type saveAuthoringModeRequest struct {
	Mode string `json:"mode"`
}

// saveAuthoringModelsRequest is the body of PUT
// /v1/projects/{id}/authoring/models — the model-per-step picker's choice
// for each of the three tabs, "" meaning "server default".
type saveAuthoringModelsRequest struct {
	Story      string `json:"story"`
	Storyboard string `json:"storyboard"`
	Code       string `json:"code"`
}

// ErrorCodeQCBlocked marks the 409 that `acknowledge_qc: true` can get past.
const ErrorCodeQCBlocked = "qc_blocked"

// createProjectDraftRequest is the body of POST /v1/projects (CR-028
// FR83.1) — sent as soon as the Creator finishes typing a topic on wizard
// step 1, well before there is any script.
type createProjectDraftRequest struct {
	// ProjectID is optional — see CreateProjectDraftInput's doc comment for
	// why web-gui sends its own client-generated id here.
	ProjectID       string `json:"project_id,omitempty"`
	Topic           string `json:"topic"`
	ContentLanguage string `json:"content_language"`
	// RenderEngine is optional (CR-030): "" means "unchanged" — see
	// CreateProjectDraftInput.RenderEngine's doc comment.
	RenderEngine string `json:"render_engine,omitempty"`
}

// updateProjectTopicRequest is the body of PATCH
// /v1/projects/{project_id}/topic (CR-028 FR83.2).
type updateProjectTopicRequest struct {
	Topic string `json:"topic"`
}

// similarProjectResponse is one entry of the FR85 collision-warning list.
type similarProjectResponse struct {
	ProjectID string    `json:"project_id"`
	Topic     string    `json:"topic"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// createProjectDraftResponse is the 201 response of POST /v1/projects.
type createProjectDraftResponse struct {
	ProjectID       string                   `json:"project_id"`
	SimilarProjects []similarProjectResponse `json:"similar_projects"`
}

// updateProjectTopicResponse is the 200 response of PATCH
// /v1/projects/{project_id}/topic.
type updateProjectTopicResponse struct {
	SimilarProjects []similarProjectResponse `json:"similar_projects"`
}

func toSimilarProjectsResponse(in []application.SimilarProject) []similarProjectResponse {
	out := make([]similarProjectResponse, 0, len(in))
	for _, s := range in {
		out = append(out, similarProjectResponse{
			ProjectID: s.ProjectID,
			Topic:     s.Topic,
			Status:    string(s.Status),
			CreatedAt: s.CreatedAt,
		})
	}
	return out
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
			RenderEngine: string(s.RenderEngine),
			WizardStep:   domain.EffectiveWizardStep(&domain.Project{Status: s.Status, WizardStep: s.WizardStep}),
			Topic:        s.Topic,
			FlowStep:     s.FlowStep,
			RunState:     string(s.RunState),
			ForkedFrom:   s.ForkedFrom,
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
		RenderEngine:     string(p.RenderEngine),
		VideoFormatID:    p.VideoFormatID,
		ReviewEnabled:    p.ReviewEnabled,
		Beats:            p.Beats,
		Warnings:         p.ValidationWarnings,
		VideoFormatVer:   p.VideoFormatVersion,
		YoutubeVideoURL:  p.YoutubeVideoURL,
		CaptionStatus:    p.CaptionStatus,
		ErrorMessage:     p.ErrorMessage,
		Clips:            toClipResultResponses(p.Clips),

		ScriptContent:         p.ScriptContent,
		BackgroundMusicPath:   p.BackgroundMusicPath,
		BackgroundMusicVolume: p.BackgroundMusicVolume,
		VideoFont:             p.VideoFont,
		VideoOutputMode:       string(p.VideoOutputMode),
		CompanionProjectID:    p.CompanionProjectID,
		WizardStep:            domain.EffectiveWizardStep(p),
		WizardRoute:           p.WizardRoute,
	}
}
