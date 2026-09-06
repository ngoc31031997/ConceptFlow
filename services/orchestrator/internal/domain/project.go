// Package domain contains the Orchestrator Service's core model: Project (the
// Saga's aggregate root and single source of truth), Scene, SagaStep, and the
// ProjectStatus state machine. Nothing in this package imports pgx, amqp091-go,
// or chi — adapters depend on domain, never the reverse.
package domain

import "time"

// ProjectStatus is the state machine driving both the Render Saga (5 steps)
// and the Publish Saga (1 step). It has 9 happy-path values plus 6
// failed_at_<step> values, one per step that can fail (business-rules.md
// Rule 4, domain-entities.md).
type ProjectStatus string

const (
	StatusDraft                  ProjectStatus = "draft"
	StatusParsingScript          ProjectStatus = "parsing_script"
	StatusClassifyingScenes      ProjectStatus = "classifying_scenes"
	StatusSynthesizingSpeech     ProjectStatus = "synthesizing_speech"
	StatusRendering              ProjectStatus = "rendering"
	StatusAssemblingVideo        ProjectStatus = "assembling_video"
	StatusReadyToPublish         ProjectStatus = "ready_to_publish"
	StatusPublishing             ProjectStatus = "publishing"
	StatusPublished              ProjectStatus = "published"
	StatusFailedParseScript      ProjectStatus = "failed_at_parse_script"
	StatusFailedClassifyScenes   ProjectStatus = "failed_at_classify_scenes"
	StatusFailedSynthesizeSpeech ProjectStatus = "failed_at_synthesize_speech"
	StatusFailedRenderScenes     ProjectStatus = "failed_at_render_scenes"
	StatusFailedAssembleVideo    ProjectStatus = "failed_at_assemble_video"
	StatusFailedPublishVideo     ProjectStatus = "failed_at_publish_video"
)

// StepName identifies one of the 6 Saga steps. It is the value stored on
// SagaStep.StepName and used as the key for the event_type → step_name
// mapping (business-logic-model.md).
type StepName string

const (
	StepParseScript      StepName = "parse_script"
	StepClassifyScenes   StepName = "classify_scenes"
	StepSynthesizeSpeech StepName = "synthesize_speech"
	StepRenderScenes     StepName = "render_scenes"
	StepAssembleVideo    StepName = "assemble_video"
	StepPublishVideo     StepName = "publish_video"
)

// FailedStatusForStep returns the failed_at_<step> ProjectStatus
// corresponding to a given StepName (business-rules.md Rule 4).
func FailedStatusForStep(step StepName) ProjectStatus {
	switch step {
	case StepParseScript:
		return StatusFailedParseScript
	case StepClassifyScenes:
		return StatusFailedClassifyScenes
	case StepSynthesizeSpeech:
		return StatusFailedSynthesizeSpeech
	case StepRenderScenes:
		return StatusFailedRenderScenes
	case StepAssembleVideo:
		return StatusFailedAssembleVideo
	case StepPublishVideo:
		return StatusFailedPublishVideo
	default:
		return ""
	}
}

// SagaStepStatus is the lifecycle of one SagaStep record — the step-level
// idempotency guard described in nfr-design-patterns.md ("2 Tầng").
type SagaStepStatus string

const (
	SagaStepInProgress SagaStepStatus = "in_progress"
	SagaStepCompleted  SagaStepStatus = "completed"
	SagaStepFailed     SagaStepStatus = "failed"
)

// VoiceLanguage restricts Project.VoiceLanguage to the two supported values.
type VoiceLanguage string

const (
	LanguageVietnamese VoiceLanguage = "vi"
	LanguageEnglish    VoiceLanguage = "en"
)

// Visibility restricts youtube visibility to the three values accepted by
// the Publish Saga input (interface-contracts.md POST /v1/sagas/publish).
type Visibility string

const (
	VisibilityPublic   Visibility = "public"
	VisibilityUnlisted Visibility = "unlisted"
	VisibilityPrivate  Visibility = "private"
)

// Scene is a value object accumulated across Render Saga steps 1-4, keyed by
// SceneIndex. Fields are populated incrementally: ScriptParsed fields at step
// 1, Category/AnimationTemplateID at step 2, AudioPath/DurationSeconds at
// step 3 (the single source of truth for AudioPath — business-rules.md Rule
// 2), ClipPath at step 4.
type Scene struct {
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

// Project is the aggregate root and single source of truth accumulating all
// data across the Render Saga and the Publish Saga (domain-entities.md). It
// carries everything RetryStepUseCase needs to reconstruct any command
// payload without re-invoking earlier steps (business-rules.md Rule 5).
type Project struct {
	ProjectID string
	Status    ProjectStatus
	SagaID    string // current/most-recent saga_id (Render or Publish — a new one is generated per Saga, interface-contracts.md Question 9)

	ScriptContent       string
	PluginID            string
	CategoryHint        string // Content Plugin's business-rules.md Rule 1 — Creator-chosen, applied to every scene (Revision 2026-09-05)
	VoiceLanguage       VoiceLanguage
	BackgroundMusicPath *string // optional static input, set at Saga start, reused unchanged at assemble_video (Rule 3)

	Scenes []Scene

	VideoPath *string

	YoutubeTitle       *string
	YoutubeDescription *string
	YoutubeTags        []string
	YoutubeVisibility  *Visibility
	YoutubePublishAt   *string // RFC3339 — schedules the video to auto-go-public at this time (only valid alongside YoutubeVisibility == private, per YouTube Data API)
	YoutubeThumbnailPath *string // absolute path on shared_artifacts, set by a manual upload before Publish Saga starts
	YoutubeVideoURL    *string

	ErrorMessage *string
}

// ProjectSummary is the lightweight projection returned by GET /v1/projects
// (list view) — deliberately excludes Scenes/ScriptContent so listing every
// project never pulls their (potentially large) JSONB payload into memory.
type ProjectSummary struct {
	ProjectID    string
	Status       ProjectStatus
	VideoPath    *string
	ErrorMessage *string
	UpdatedAt    time.Time
}

// SagaStep tracks the processing state of a single step within one Saga
// instance (one row per step attempted). A Project can have many SagaID
// values over its lifetime (Render Saga then Publish Saga are two distinct
// sagas for the same project_id — domain-entities.md).
type SagaStep struct {
	SagaID       string
	StepName     StepName
	Status       SagaStepStatus
	ErrorMessage *string
}

// ProgressMessage is published to progress.fanout after every
// HandleStepEventUseCase processing, including the scene_rendered progress
// event which does not advance the state machine (business-rules.md Rule 7).
type ProgressMessage struct {
	ProjectID    string  `json:"project_id"`
	Step         string  `json:"step"`
	Status       string  `json:"status"` // in_progress | completed | failed
	SceneIndex   *int    `json:"scene_index,omitempty"`
	SceneTotal   *int    `json:"scene_total,omitempty"`
	ErrorMessage *string `json:"error_message,omitempty"`
}
