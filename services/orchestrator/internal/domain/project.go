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
	StatusDraft            ProjectStatus = "draft"
	StatusParsingScript    ProjectStatus = "parsing_script"
	StatusValidatingScript ProjectStatus = "validating_script"
	// CR-024: Saga dừng lại chờ Creator duyệt dàn ý. Đây là đường đi bình
	// thường, KHÔNG phải một trạng thái lỗi — nó cố ý không nằm trong nhóm
	// failed_at_* (FR69.1).
	StatusAwaitingReview     ProjectStatus = "awaiting_review"
	StatusSynthesizingSpeech ProjectStatus = "synthesizing_speech"
	StatusRendering          ProjectStatus = "rendering"
	StatusAssemblingVideo    ProjectStatus = "assembling_video"
	// CR-021 D2: video đã ghép xong và đang được chấm chất lượng tự động. Nằm
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
	// CR-021 FR61.4: QC KHÔNG có nhánh failed từ phía chấm điểm — không chấm
	// được vẫn phát qc_completed với status="not_scored" và project vẫn về
	// ready_to_publish. Trạng thái này chỉ dùng khi chính message hỏng (không
	// dựng nổi envelope), đúng ngữ nghĩa các bước khác. Một cổng hỏng không
	// được biến thành cổng khoá.
	StatusFailedQCVideo ProjectStatus = "failed_at_qc_video"
	// CR-007 D1: generate_clips sits between qc_video and publish_video —
	// cutting vertical clips from a video QC already flagged just multiplies
	// the same defect into two or three clips, and clips should exist before
	// the Creator reaches the results screen.
	StatusGeneratingClips ProjectStatus = "generating_clips"
	// StatusFailedGenerateClips is used only when the generate_clips MESSAGE
	// itself is broken (undeliverable command, dead-lettered) — never when an
	// individual clip fails. D1: a clip's own failure is carried as
	// status="error" on that one entry of clips_generated, and the saga still
	// proceeds to ready_to_publish (a vertical clip is a derivative product,
	// not the main video).
	StatusFailedGenerateClips ProjectStatus = "failed_at_generate_clips"
	StatusFailedPublishVideo  ProjectStatus = "failed_at_publish_video"
)

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
	// CR-021 D1: bước riêng, nhưng worker sống trong video-assembly (nơi đã có
	// sẵn ffmpeg/ffprobe và chính file video vừa ghép). Lệnh `qc_video` đi trên
	// đúng queue `video_assembly.commands` mà `assemble_video` đang đi.
	StepQCVideo StepName = "qc_video"
	// CR-007 D1: worker lives in video-assembly (same queue/dispatcher as
	// assemble_video/qc_video/normalize_channel_asset — event_type tells them apart).
	StepGenerateClips StepName = "generate_clips"
	StepPublishVideo  StepName = "publish_video"
)

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
	case StepGenerateClips:
		return StatusFailedGenerateClips
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

// ContentLanguage is the language a project's *content* is produced in
// (CR-008 FR21.1). It drives the TTS voice, the narration-pacing estimate,
// the subtitles, and the language the SEO metadata and thumbnail prompt are
// generated in — everything the audience sees or hears.
//
// It is deliberately NOT the Creator's interface language. Someone Vietnamese
// running an English channel wants a Vietnamese UI producing English content,
// so the two are separate axes and only this one lives on the Project.
//
// The JSON field and database column are still named `voice_language`: renaming
// them would break every project created before CR-008 for no functional gain
// (CR-008 §C2). The name is historical, the meaning is now broader.
type ContentLanguage string

const (
	LanguageVietnamese ContentLanguage = "vi"
	LanguageEnglish    ContentLanguage = "en"
)

// RenderQuality is the resolution/framerate the Creator chose for a project
// (CR-004 FR12.6). A draft pass at 720p30 is for checking the content; the
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

// IsValid reports whether q is a quality the Rendering Service can honour.
func (q RenderQuality) IsValid() bool {
	switch q {
	case Quality480p15, Quality720p30, Quality1080p60, Quality4k60:
		return true
	}
	return false
}

// VideoOutputMode is which of the two shapes a project produces (CR-007
// follow-up): the standard 16:9 long-form video, the vertical Shorts/TikTok
// clip(s) cut from it, or both. A clip is always derived from the assembled
// 16:9 video (CR-007 D1 — no standalone vertical production), so
// ModeShortOnly still runs the full render pipeline as source material; the
// mode only decides whether generate_clips runs at all, and which output the
// Result screen puts front and center.
type VideoOutputMode string

const (
	ModeLongOnly  VideoOutputMode = "long"
	ModeShortOnly VideoOutputMode = "short"
	ModeBoth      VideoOutputMode = "both"
)

// DefaultVideoOutputMode preserves the only behaviour that existed before
// this field did: every project produces its long-form video, and never
// spends the extra generate_clips round-trip unless the Creator opts in.
const DefaultVideoOutputMode = ModeLongOnly

// IsValid reports whether m is a mode the saga knows how to route.
func (m VideoOutputMode) IsValid() bool {
	switch m {
	case ModeLongOnly, ModeShortOnly, ModeBoth:
		return true
	}
	return false
}

// WantsClips reports whether the saga should run generate_clips at all.
func (m VideoOutputMode) WantsClips() bool {
	return m == ModeShortOnly || m == ModeBoth
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
	// CR-024 FR68.5: khung hình lúc câu này được nói, dạng "Text×2, Arrow".
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
	CategoryHint        string // Content Plugin's business-rules.md Rule 1 — Creator-chosen, applied to every scene (Revision 2026-09-05)
	ContentLanguage     ContentLanguage
	BackgroundMusicPath *string // optional static input, set at Saga start, reused unchanged at assemble_video (Rule 3)
	// CR-005 FR14.2 — music level, 0.0-1.0. Zero means "unset"; assembly
	// substitutes its own default so an old project keeps the previous 0.2.
	BackgroundMusicVolume float64

	// CR-001 — narration and subtitles are independently switchable per project.
	// When TTSEnabled is false the synthesize_speech step is skipped entirely and
	// Scene.DurationSeconds is filled from EstimateNarrationDuration instead.
	TTSEnabled bool
	VoiceID    string

	// CR-019: hình dạng video project này được dựng theo. Version chốt lại tại
	// thời điểm render, nên sửa format sau đó không làm sai lệch dàn ý hay
	// chapter của video cũ (FR51.6).
	VideoFormatID      string
	VideoFormatVersion int

	// CR-024: các beat lượt dry quan sát được, và cảnh báo từ bước validate.
	// Cả hai chỉ tồn tại để dựng màn duyệt dàn ý.
	Beats              []BeatOccurrence
	ValidationWarnings []string

	// ReviewEnabled bật cổng duyệt dàn ý (FR69.7). Mặc định bật; tắt được cho
	// những lần chạy mà Creator đã biết rõ mình muốn gì — một cổng không bỏ qua
	// được sẽ biến thành thao tác bấm cho xong và mất hết giá trị.
	ReviewEnabled bool
	// SubtitlesEnabled is kept for wire/schema backward compatibility (a
	// caller that never adopts subtitle_mode) but SubtitleMode is the
	// source of truth from CR-015 on — see project_repository.go's Get for
	// how a row from before that column existed gets one anyway.
	SubtitlesEnabled bool
	SubtitleMode     SubtitleMode
	SubtitleStyle    *SubtitleStyle

	// CR-004 — resolution/framerate for this project's render.
	RenderQuality RenderQuality

	// Which shape(s) of output this project produces — long-form, short
	// clips, or both. Drives whether generate_clips runs at all.
	VideoOutputMode VideoOutputMode

	// CR-023 FR67.1/67.2 — whether the fixed channel intro/outro sting is
	// attached at assemble_video. Both default true (long-form channel
	// identity is opt-out, not opt-in).
	IntroEnabled bool
	OutroEnabled bool
	// IntroAssetID/OutroAssetID are resolved once, at assemble_video dispatch
	// time, from video-assembly's channel_assets (CR-023 D1/D2) and then
	// persisted here — nil when disabled or when no active asset was found
	// (the saga proceeds without one rather than failing). Storing the
	// resolved id rather than re-resolving it keeps RetryStepUseCase's Rule 5
	// guarantee: a retry rebuilds the command purely from Project, with no
	// second lookup that could disagree with what was actually dispatched.
	IntroAssetID *string
	OutroAssetID *string

	Scenes []Scene

	// CR-006 — chapter markers from the script, resolved to timestamps only
	// once Rendering reports where each narration actually starts.
	Chapters []Chapter

	RenderedVideoPath *string // the single Manim-rendered video (silent), set by rendering_completed — distinct from VideoPath (post-assembly, with audio muxed in)
	VideoPath         *string
	// CR-015 FR38.4 — the .srt caption track Video Assembly wrote alongside
	// VideoPath, or nil when subtitle_mode didn't produce one (off/burn_in
	// only, or subtitles disabled). Flows to the Publish Saga the same way
	// YoutubeThumbnailPath does.
	CaptionPath *string
	// CaptionStatus mirrors the Publisher's PublishResult.caption_status
	// (CR-015 FR39.4) — nil when no caption was requested; otherwise
	// "uploaded" | "skipped_no_scope" | "failed". Surfaced to the GUI so a
	// silently skipped or failed caption is not invisible.
	CaptionStatus *string

	// CR-002 — where each narration segment actually begins in RenderedVideoPath,
	// measured by Rendering. WaitOffsets[i] belongs to Scenes[i] in scene_index
	// order. It is NOT the running sum of Scene.DurationSeconds: the animation
	// between narrations pushes every later segment further out, and assuming
	// otherwise desynchronised audio, subtitles and video by the accumulated
	// animation time.
	WaitOffsets          []float64
	RenderedVideoSeconds float64 // Rendering's measured length of RenderedVideoPath

	// CR-021 FR58/D3 — what was on screen at each narration mark, as measured
	// by Rendering's `{"kind":"layout"}` records and carried out on
	// `rendering_completed`. Orchestrator stores it and hands it straight back
	// to the QC worker on `qc_video`; it never interprets it, which is why the
	// element type is the raw decoded object rather than a struct. Typing it
	// here would mean this service has to be redeployed in lockstep every time
	// Rendering measures one more property of a mobject, for no gain: the only
	// code that reads a bbox is `video-assembly/domain/qc_rules.py`.
	LayoutMarks []map[string]interface{}

	// CR-007 FR19.2 — the `with self.clip(...)` selections Rendering measured
	// on the real render pass, carried verbatim on rendering_completed
	// (kind="clip", name, t_start, t_end) exactly like LayoutMarks: Orchestrator
	// never reads a field inside, only stores it and hands it to
	// generate_clips's request-merging logic (D3).
	ClipMarks []map[string]interface{}

	// IntroDurationSeconds is the channel intro's real length, as measured by
	// video-assembly when it resolved IntroAssetID and folded it into
	// effective_lead_in (CR-023 D5) — carried out on video_assembled. It is
	// the only source of this number: Orchestrator's channel_asset_pointers
	// projection stores no duration, and Orchestrator does not call
	// video-assembly over HTTP (CR-023 correction). generate_clips needs it
	// to shift a Creator's clip selection by the same amount the narration
	// and subtitles were already shifted — without it, a clip cut from a
	// project with an intro enabled would be off by exactly the intro's
	// length (CR-007 D5 risk).
	IntroDurationSeconds float64
	// ClipRequests holds the Creator-entered clip selections from
	// POST /v1/projects/{id}/clips — {name, start_seconds, end_seconds,
	// presets}. Kept separate from ClipMarks (script-sourced) so D3's "trùng
	// tên thì GUI thắng" can be resolved at generate_clips dispatch time
	// without the two sources overwriting each other on arrival.
	ClipRequests []map[string]interface{}
	// Clips is the outcome of generate_clips, stored verbatim from
	// clips_generated (one entry per requested (name, preset) pair, "ok" or
	// "error" — D1: a clip failure never blocks the saga).
	Clips []ClipResult

	// CompanionProjectID (CR-026 D1) links two independent projects that
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
	YoutubeChannelID     *string // which connected YouTube channel to publish to; nil means the Publisher's default channel (CR-012)
	YoutubeVideoURL      *string

	ErrorMessage *string
}

// ClipResult is one (name, preset) outcome of the generate_clips step
// (CR-007), stored verbatim from clips_generated's "clips" array.
type ClipResult struct {
	Name            string  `json:"name"`
	Preset          string  `json:"preset"`
	Status          string  `json:"status"` // "ok" | "error"
	OutputPath      string  `json:"output_path,omitempty"`
	DurationSeconds float64 `json:"duration_seconds,omitempty"`
	ErrorMessage    string  `json:"error_message,omitempty"`
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
