// Package domain contains the Orchestrator Service's core model: Project (the
// Saga's aggregate root and single source of truth), Scene, SagaStep, and the
// ProjectStatus state machine. Nothing in this package imports pgx, amqp091-go,
// or chi — adapters depend on domain, never the reverse.
package domain

import (
	"strings"
	"time"
)

// ProjectStatus is the state machine driving both the Render Saga (5 steps)
// and the Publish Saga (1 step). It has 9 happy-path values plus 6
// failed_at_<step> values, one per step that can fail (business-rules.md
// Rule 4, domain-entities.md).
type ProjectStatus string

const (
	StatusDraft            ProjectStatus = "draft"
	StatusParsingScript    ProjectStatus = "parsing_script"
	StatusValidatingScript ProjectStatus = "validating_script"
	// Saga dừng lại chờ Creator duyệt dàn ý. Đây là đường đi bình
	// thường, KHÔNG phải một trạng thái lỗi — nó cố ý không nằm trong nhóm
	// failed_at_*.
	StatusAwaitingReview     ProjectStatus = "awaiting_review"
	StatusSynthesizingSpeech ProjectStatus = "synthesizing_speech"
	StatusRendering          ProjectStatus = "rendering"
	StatusAssemblingVideo    ProjectStatus = "assembling_video"
	// Video đã ghép xong và đang được chấm chất lượng tự động. Nằm
	// giữa assembling_video và ready_to_publish — không phải trạng thái chờ
	// người, Creator không phải làm gì ở đây.
	StatusRunningQC              ProjectStatus = "running_qc"
	StatusReadyToPublish         ProjectStatus = "ready_to_publish"
	StatusPublishing             ProjectStatus = "publishing"
	StatusPublished              ProjectStatus = "published"
	StatusFailedParseScript      ProjectStatus = "failed_at_parse_script"
	StatusFailedValidateScript   ProjectStatus = "failed_at_validate_script"
	StatusFailedSynthesizeSpeech ProjectStatus = "failed_at_synthesize_speech"
	StatusFailedRenderScenes     ProjectStatus = "failed_at_render_scenes"
	StatusFailedAssembleVideo    ProjectStatus = "failed_at_assemble_video"
	// QC KHÔNG có nhánh failed từ phía chấm điểm — không chấm
	// được vẫn phát qc_completed với status="not_scored" và project vẫn về
	// ready_to_publish. Trạng thái này chỉ dùng khi chính message hỏng (không
	// dựng nổi envelope), đúng ngữ nghĩa các bước khác. Một cổng hỏng không
	// được biến thành cổng khoá.
	StatusFailedQCVideo      ProjectStatus = "failed_at_qc_video"
	StatusFailedPublishVideo ProjectStatus = "failed_at_publish_video"
	// Delete is a saga. The project stays `deleting` (hidden
	// from the list, not in flight) until every service that writes to
	// shared_artifacts has confirmed it removed its own files.
	StatusDeleting ProjectStatus = "deleting"
)

// IsInFlight reports whether a saga step is executing right now, i.e. a
// worker may still be writing files for the project. Waiting for the Creator
// (draft, awaiting_review, ready_to_publish, published) and failed_at_* are not
// in flight. An in-flight project must not be deleted.
func (s ProjectStatus) IsInFlight() bool {
	switch s {
	case StatusParsingScript, StatusValidatingScript, StatusSynthesizingSpeech,
		StatusRendering, StatusAssemblingVideo, StatusRunningQC,
		StatusPublishing:
		return true
	}
	return false
}

// DoneWithLibrary reports whether a project no longer needs the library
// drawings it uses: it reached the result screen (Kết quả) or
// later, or it is being deleted. Every other status, failed_at_* included
// since a failed step can run again, may still render them.
func (s ProjectStatus) DoneWithLibrary() bool {
	switch s {
	case StatusReadyToPublish, StatusPublishing, StatusPublished, StatusFailedPublishVideo, StatusDeleting:
		return true
	}
	return false
}

// Delete-saga steps: one per service that owns files on
// shared_artifacts. Stored in saga_steps like any step so no schema change is
// needed; the row is deleted with the project.
const (
	StepPurgeTTS           StepName = "purge_tts"
	StepPurgeRendering     StepName = "purge_rendering"
	StepPurgeVideoAssembly StepName = "purge_video_assembly"
)

// PurgeTargets maps the `service` an artifacts_purged event names to its step
// and to the routing key its purge command is sent on.
var PurgeTargets = []struct {
	Service    string
	Step       StepName
	RoutingKey string
}{
	{"tts", StepPurgeTTS, "tts"},
	{"rendering", StepPurgeRendering, "rendering"},
	{"video_assembly", StepPurgeVideoAssembly, "video_assembly"},
}

// StepName identifies one of the 6 Saga steps. It is the value stored on
// SagaStep.StepName and used as the key for the event_type → step_name
// mapping (business-logic-model.md).
type StepName string

const (
	StepParseScript      StepName = "parse_script"
	StepValidateScript   StepName = "validate_script"
	StepSynthesizeSpeech StepName = "synthesize_speech"
	StepRenderScenes     StepName = "render_scenes"
	StepAssembleVideo    StepName = "assemble_video"
	// Bước riêng, nhưng worker sống trong video-assembly (nơi đã có
	// sẵn ffmpeg/ffprobe và chính file video vừa ghép). Lệnh `qc_video` đi trên
	// đúng queue `video_assembly.commands` mà `assemble_video` đang đi.
	StepQCVideo      StepName = "qc_video"
	StepPublishVideo StepName = "publish_video"
)

// IsFailedStatus reports whether s is any failed_at_<step> status.
func IsFailedStatus(s ProjectStatus) bool {
	return strings.HasPrefix(string(s), "failed_at_")
}

// IsAuthoringEditable is true while a Creator may still change the inputs of a
// project: a draft, or one that failed at ANY step — so a failed project can be
// fixed (topic, settings, prompts, script) and re-run from whichever step.
func IsAuthoringEditable(s ProjectStatus) bool {
	return s == StatusDraft || IsFailedStatus(s)
}

// FailedStatusForStep returns the failed_at_<step> ProjectStatus
// corresponding to a given StepName (business-rules.md Rule 4).
func FailedStatusForStep(step StepName) ProjectStatus {
	switch step {
	case StepParseScript:
		return StatusFailedParseScript
	case StepValidateScript:
		return StatusFailedValidateScript
	case StepSynthesizeSpeech:
		return StatusFailedSynthesizeSpeech
	case StepRenderScenes:
		return StatusFailedRenderScenes
	case StepAssembleVideo:
		return StatusFailedAssembleVideo
	case StepQCVideo:
		return StatusFailedQCVideo
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

// ContentLanguage is the language a project's *content* is produced in.
// It drives the TTS voice, the narration-pacing estimate,
// the subtitles, and the language the SEO metadata and thumbnail prompt are
// generated in — everything the audience sees or hears.
//
// It is deliberately NOT the Creator's interface language. Someone Vietnamese
// running an English channel wants a Vietnamese UI producing English content,
// so the two are separate axes and only this one lives on the Project.
//
// The JSON field and database column are still named `voice_language`: renaming
// them would break every stored project for no functional gain.
// The name is historical, the meaning is now broader.
type ContentLanguage string

const (
	LanguageVietnamese ContentLanguage = "vi"
	LanguageEnglish    ContentLanguage = "en"
)

// RenderQuality is the resolution/framerate the Creator chose for a project.
// A draft pass at 720p30 is for checking the content; the
// upload pass should be 1080p60 or better, since anything less is below what a
// monetized channel should publish and wastes Manim's smooth motion.
type RenderQuality string

const (
	// Quality480p15 is Manim's lowest preset (-ql) — for iterating on a script's
	// content/timing quickly, not for anything a viewer will ever see. Not the
	// default for anything; a Creator opts into it explicitly per render.
	Quality480p15  RenderQuality = "480p15"
	Quality720p30  RenderQuality = "720p30"
	Quality1080p60 RenderQuality = "1080p60"
	Quality4k60    RenderQuality = "4k60"
)

// DefaultRenderQuality is what a project gets when the Creator did not choose —
// including every project created before this field existed.
const DefaultRenderQuality = Quality1080p60

// RenderEngine is which rendering backend produces this project's video —
// Manim (Python, the original engine) or Remotion (React/TypeScript,
// headless-Chromium via @remotion/renderer). It is orthogonal to
// RenderQuality: both engines are driven by the same resolution/framerate
// choice.
type RenderEngine string

const (
	RenderEngineManim    RenderEngine = "manim"
	RenderEngineRemotion RenderEngine = "remotion"
)

// DefaultRenderEngine preserves the only behaviour that existed before this
// field did — every project (including ones created before it existed) is a
// Manim project unless the Creator opts into Remotion.
const DefaultRenderEngine = RenderEngineManim

// IsValid reports whether e is a render engine the Rendering Service knows
// how to run.
func (e RenderEngine) IsValid() bool {
	switch e {
	case RenderEngineManim, RenderEngineRemotion:
		return true
	}
	return false
}

// IsValid reports whether q is a quality the Rendering Service can honour.
func (q RenderQuality) IsValid() bool {
	switch q {
	case Quality480p15, Quality720p30, Quality1080p60, Quality4k60:
		return true
	}
	return false
}

// VideoOutputMode is which shape a project produces: the 16:9 long-form
// video, or the vertical short built at 1080x1920 from the start (its own
// script, storyboard and render; see ADR-0031).
type VideoOutputMode string

const (
	ModeLongOnly  VideoOutputMode = "long"
	ModeShortOnly VideoOutputMode = "short"
)

// DefaultVideoOutputMode is the mode of a project that never chose one: the
// long-form video.
const DefaultVideoOutputMode = ModeLongOnly

// IsValid reports whether m is a mode the saga knows how to route.
func (m VideoOutputMode) IsValid() bool {
	switch m {
	case ModeLongOnly, ModeShortOnly:
		return true
	}
	return false
}

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
	SceneIndex    int    `json:"scene_index"`
	NarrationText string `json:"narration_text"`
	// Khung hình lúc câu này được nói, dạng "Text×2, Arrow".
	// Chỉ dùng cho màn duyệt dàn ý; không ảnh hưởng gì tới render.
	Visual              string  `json:"visual,omitempty"`
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
	ManimSceneClassName string // the Manim `Scene` subclass Rendering must execute — set from script_parsed (Manim-script input mode)
	PluginID            string
	CategoryHint        string // business-rules.md Rule 1 — Creator-chosen, applied to every scene
	ContentLanguage     ContentLanguage
	BackgroundMusicPath *string // optional static input, set at Saga start, reused unchanged at assemble_video (Rule 3)
	// Music level, 0.0-1.0. Zero means "unset"; assembly
	// substitutes its own default so an old project keeps the previous 0.2.
	BackgroundMusicVolume float64
	// Font for text drawn inside a Remotion video. Empty means
	// DefaultVideoFont. Manim ignores it: its fonts come from the theme.
	VideoFont string

	// Narration and subtitles are independently switchable per project.
	// When TTSEnabled is false the synthesize_speech step is skipped entirely and
	// Scene.DurationSeconds is filled from EstimateNarrationDuration instead.
	TTSEnabled bool
	VoiceID    string

	// Hình dạng video project này được dựng theo. Version chốt lại tại
	// thời điểm render, nên sửa format sau đó không làm sai lệch dàn ý hay
	// chapter của video cũ.
	VideoFormatID      string
	VideoFormatVersion int

	// Các beat lượt dry quan sát được, và cảnh báo từ bước validate.
	// Cả hai chỉ tồn tại để dựng màn duyệt dàn ý.
	Beats              []BeatOccurrence
	ValidationWarnings []string

	// ReviewEnabled bật cổng duyệt dàn ý. Mặc định bật; tắt được cho
	// những lần chạy mà Creator đã biết rõ mình muốn gì — một cổng không bỏ qua
	// được sẽ biến thành thao tác bấm cho xong và mất hết giá trị.
	ReviewEnabled bool
	// SubtitlesEnabled is kept for wire/schema backward compatibility (a
	// caller that never adopts subtitle_mode) but SubtitleMode is the
	// source of truth — see project_repository.go's Get for how a legacy row
	// without that column gets one.
	SubtitlesEnabled bool
	SubtitleMode     SubtitleMode
	SubtitleStyle    *SubtitleStyle

	// resolution/framerate for this project's render.
	RenderQuality RenderQuality

	// RenderEngine is which rendering backend (Manim or Remotion) executes
	// this project's script. Empty (pre-existing projects) is treated as
	// DefaultRenderEngine everywhere it is read.
	RenderEngine RenderEngine

	// Which shape of output this project produces: the 16:9 long-form video
	// or the vertical short.
	VideoOutputMode VideoOutputMode

	// Whether the fixed channel intro/outro sting is
	// attached at assemble_video. Both default true (long-form channel
	// identity is opt-out, not opt-in).
	IntroEnabled bool
	OutroEnabled bool
	// IntroAssetID/OutroAssetID are resolved once, at assemble_video dispatch
	// time, from video-assembly's channel_assets and then
	// persisted here — nil when disabled or when no active asset was found
	// (the saga proceeds without one rather than failing). Storing the
	// resolved id rather than re-resolving it keeps RetryStepUseCase's Rule 5
	// guarantee: a retry rebuilds the command purely from Project, with no
	// second lookup that could disagree with what was actually dispatched.
	IntroAssetID *string
	OutroAssetID *string

	Scenes []Scene

	// Chapter markers from the script, resolved to timestamps only
	// once Rendering reports where each narration actually starts.
	Chapters []Chapter

	RenderedVideoPath *string // the single Manim-rendered video (silent), set by rendering_completed — distinct from VideoPath (post-assembly, with audio muxed in)
	VideoPath         *string
	// The .srt caption track Video Assembly wrote alongside
	// VideoPath, or nil when subtitle_mode didn't produce one (off/burn_in
	// only, or subtitles disabled). Flows to the Publish Saga the same way
	// YoutubeThumbnailPath does.
	CaptionPath *string
	// CaptionStatus mirrors the Publisher's PublishResult.caption_status —
	// nil when no caption was requested; otherwise
	// "uploaded" | "skipped_no_scope" | "failed". Surfaced to the GUI so a
	// silently skipped or failed caption is not invisible.
	CaptionStatus *string

	// Where each narration segment actually begins in RenderedVideoPath,
	// measured by Rendering. WaitOffsets[i] belongs to Scenes[i] in scene_index
	// order. It is NOT the running sum of Scene.DurationSeconds: the animation
	// between narrations pushes every later segment further out, and assuming
	// otherwise desynchronised audio, subtitles and video by the accumulated
	// animation time.
	WaitOffsets          []float64
	RenderedVideoSeconds float64 // Rendering's measured length of RenderedVideoPath

	// What was on screen at each narration mark, as measured
	// by Rendering's `{"kind":"layout"}` records and carried out on
	// `rendering_completed`. Orchestrator stores it and hands it straight back
	// to the QC worker on `qc_video`; it never interprets it, which is why the
	// element type is the raw decoded object rather than a struct. Typing it
	// here would mean this service has to be redeployed in lockstep every time
	// Rendering measures one more property of a mobject, for no gain: the only
	// code that reads a bbox is `video-assembly/domain/qc_rules.py`.
	LayoutMarks []map[string]interface{}

	// CompanionProjectID links two independent projects that
	// cover the same topic as two different outputs — a long-form video and
	// a short-form one with its own dedicated script (not a crop of the
	// long one). Self-referencing, no FK: the two projects have independent
	// lifecycles, and deleting one must never fail because the other still
	// points to it. Set both ways when the second project is created
	// (StartRenderSagaUseCase), best-effort — a missing/failed link never
	// blocks the video it points from.
	CompanionProjectID *string

	YoutubeTitle         *string
	YoutubeDescription   *string
	YoutubeTags          []string
	YoutubeVisibility    *Visibility
	YoutubePublishAt     *string // RFC3339 — schedules the video to auto-go-public at this time (only valid alongside YoutubeVisibility == private, per YouTube Data API)
	YoutubeThumbnailPath *string // absolute path on shared_artifacts, set by a manual upload before Publish Saga starts
	YoutubeChannelID     *string // which connected YouTube channel to publish to; nil means the Publisher's default channel
	YoutubeVideoURL      *string

	ErrorMessage *string

	// WizardStep is the furthest wizard step the Creator has confirmed with
	// "Tiếp tục" (1-3; later steps are implied by Status — see
	// EffectiveWizardStep). Read-only here: Save does not write it, only
	// SaveWizardSettings does, so a full-row upsert from the saga can never
	// move a Creator backwards.
	WizardStep int
	// WizardRoute is the wizard screen the Creator last had open on this draft
	// (see ValidWizardRoute); "" when never recorded.
	WizardRoute string
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
	RenderEngine RenderEngine
	// WizardStep is the stored (raw) step; use EffectiveWizardStep for display.
	WizardStep int
	// Topic is the Creator's idea, so the list can name a project by it.
	Topic string
	// FlowStep/RunState place it in the 13-step flow (see FlowStateFor).
	FlowStep int
	RunState RunState
	// ForkedFrom is the project this one was forked from, "" if none.
	ForkedFrom string
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

// AuthoringMode is how the Creator works step 1 of the wizard:
// copy each prompt out to an external AI and paste the answer back, or let the
// server render the prompt and call the provider itself.
//
// One value per project, not per tab. A Creator who decided to run this script
// through the API does not want to decide again on 1b, 1c and 1d — and because
// it lives in project_authoring rather than only in the browser, the decision
// survives a reload, another browser, and a restart of this service, whichever
// step the project is sitting on.
type AuthoringMode string

const (
	// AuthoringModeManual is the copy-prompt-out round trip — the way
	// that works with no API
	// key, no credit, or a provider outage. It is the default
	// for every project.
	AuthoringModeManual AuthoringMode = "manual"
	// AuthoringModeAI lets the server render the prompt and call the provider.
	// It is only offered where a provider is actually configured.
	AuthoringModeAI AuthoringMode = "ai"
)

// ValidAuthoringMode reports whether s names a mode.
func ValidAuthoringMode(s string) bool {
	return AuthoringMode(s) == AuthoringModeManual || AuthoringMode(s) == AuthoringModeAI
}

// NormalizeAuthoringMode turns anything unrecognised — "" from a project whose
// row predates the column, or a value from a newer client — into the default.
// Read paths normalise; the write path (SaveAuthoringModeUseCase) rejects
// instead, so a disagreement surfaces where it can be fixed.
func NormalizeAuthoringMode(s string) string {
	if ValidAuthoringMode(s) {
		return s
	}
	return string(AuthoringModeManual)
}

// AuthoringModelOption is one entry of the Hive model picker the Creator sees
// at wizard step 1 (in-app authoring, one model per step).
// ID is the exact string Hive's chat-completions API expects in the "model"
// field — the two values below are the ones the token measurements were
// taken against (see llm_provider.go's TokenUsage doc and hive_client.go).
type AuthoringModelOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	// CodeOK is false for a model the code step (and the drawings, which are
	// code too) refuses — see ModelAllowedForStep.
	CodeOK bool `json:"code_ok"`
}

// AuthoringModelCatalog is the fixed list the picker offers. "" is not listed
// (the GUI shows the concrete model instead, via /v1/llm/status default_model)
// but stays valid on the write path: it means "use the server's configured
// default" (HIVE_MODEL), which is what an empty/legacy project_authoring row
// still means today.
var AuthoringModelCatalog = []AuthoringModelOption{
	{ID: "deepseek-ai/deepseek-v4.1-flash", Label: "DeepSeek V4.1 Flash", CodeOK: true},
	{ID: "zai-org/glm-5.3-flash", Label: "GLM-5.3-Flash", CodeOK: true},
	// Local model served by the `ollama` container. llm-service routes any id
	// "ollama" or "ollama/<model>" to Ollama instead of Hive; bare "ollama"
	// means whatever OLLAMA_MODEL is set to.
	{ID: "ollama", Label: "Ollama (local AI)"},
}

// ValidAuthoringModel reports whether id is "" (server default) or one of
// AuthoringModelCatalog's entries. Like ValidAuthoringMode, the write path
// rejects anything else rather than silently falling back — a client and
// server that disagree about which models exist need to be told, not papered
// over with a default that quietly ignores the Creator's choice.
func ValidAuthoringModel(id string) bool {
	if id == "" {
		return true
	}
	for _, opt := range AuthoringModelCatalog {
		if opt.ID == id {
			return true
		}
	}
	return false
}

// ModelAllowedForStep reports whether model id may serve an authoring step.
// The code step — and the illustrations step, whose drawings
// are Remotion code written with the code step's model — refuses the local
// Ollama model: every code chunk sent to it timed out (5/5 measured), and a
// local model is not strong enough to write a Remotion or Manim scene.
func ModelAllowedForStep(step, id string) bool {
	if step != "code" && step != "illustrations" {
		return true
	}
	return id != "ollama" && !strings.HasPrefix(id, "ollama/")
}

// AuthoringStepModels is the per-step model choice for one project — one Hive
// model id (or "" for the server default) per tab of "Bước 1 — Script". Only
// meaningful when AuthoringMode is "ai"; a manual project simply carries
// empty strings here.
type AuthoringStepModels struct {
	Story      string
	Storyboard string
	Code       string
}

// ModelFor returns the chosen model for one authoring step ("story",
// "storyboard", or "code"), or "" for an unrecognised step — callers already
// validate the step elsewhere (RoleFor), so this never has to reject one.
func (m AuthoringStepModels) ModelFor(step string) string {
	switch step {
	case "story":
		return m.Story
	case "storyboard":
		return m.Storyboard
	case "code":
		return m.Code
	default:
		return ""
	}
}
