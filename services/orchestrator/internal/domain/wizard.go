package domain

import "fmt"

// Wizard steps the Creator walks through. The GUI shows exactly seven; the
// backend stores which one a project has reached so a reload, another browser
// or a restart of the stack resumes in the right place.
const (
	WizardStepIdea     = 1 // Ý tưởng — topic / situation
	WizardStepConfig   = 2 // Cấu hình — language, engine, authoring mode, voice, format, output mode
	WizardStepScript   = 3 // Script — 1a outline, 1b storyboard, 1c code
	WizardStepValidate = 4 // Validate — parse + dry run, then the outline gate
	WizardStepProcess  = 5 // Xử lý — TTS, render, assemble, clips
	WizardStepResult   = 6 // Kết quả
	WizardStepPublish  = 7 // Đăng
)

// LastAuthoredWizardStep is the highest step a Creator can advance to by
// pressing "Tiếp tục" on a form. Steps after it are driven by the saga's
// status — see WizardStepForStatus.
const LastAuthoredWizardStep = WizardStepScript

// WizardSettings is everything wizard step 2 collects. It maps one-to-one
// onto columns of projects that already exist, so the render saga later reads
// the same values back rather than trusting whatever the browser resends.
type WizardSettings struct {
	ContentLanguage       ContentLanguage
	RenderEngine          RenderEngine
	TTSEnabled            bool
	VoiceID               string
	SubtitleMode          SubtitleMode
	SubtitleStyle         *SubtitleStyle
	RenderQuality         RenderQuality
	VideoFormatID         string
	VideoOutputMode       VideoOutputMode
	BackgroundMusicPath   *string
	BackgroundMusicVolume float64
	VideoFont             string
}

// WizardSettingsPatch is a partial update of WizardSettings: a nil field is
// left as stored. BackgroundMusicPath "" clears the track. Confirm also moves
// the project on to step 3 (the Creator pressed "Tiếp tục").
type WizardSettingsPatch struct {
	ContentLanguage       *ContentLanguage
	RenderEngine          *RenderEngine
	TTSEnabled            *bool
	VoiceID               *string
	SubtitleMode          *SubtitleMode
	SubtitleStyle         *SubtitleStyle
	RenderQuality         *RenderQuality
	VideoFormatID         *string
	VideoOutputMode       *VideoOutputMode
	BackgroundMusicPath   *string
	BackgroundMusicVolume *float64
	VideoFont             *string
	Confirm               bool
}

// ApplyShortDefaults completes a patch that switches the project to the
// vertical short with what a short needs and the Creator did not pick in the
// same patch: the short format, the Remotion engine (Manim has no portrait
// frame) and no subtitles (a short carries its keywords in the picture).
func ApplyShortDefaults(p *WizardSettingsPatch) {
	if p.VideoOutputMode == nil || *p.VideoOutputMode != ModeShortOnly {
		return
	}
	if p.VideoFormatID == nil {
		id := FormatVerticalShort60s.ID
		p.VideoFormatID = &id
	}
	if p.RenderEngine == nil {
		engine := RenderEngineRemotion
		p.RenderEngine = &engine
	}
	if p.SubtitleMode == nil {
		off := SubtitleModeOff
		p.SubtitleMode = &off
	}
}

// CheckOutputSettings refuses a combination the render saga cannot build: a
// short needs a short format and the Remotion engine, and a short format
// belongs to a short.
func CheckOutputSettings(mode VideoOutputMode, format VideoFormat, engine RenderEngine) error {
	short := mode == ModeShortOnly
	switch {
	case short && !format.IsShort():
		return fmt.Errorf("%w: short dọc cần format ngắn (tối đa %d giây), đang chọn %q",
			ErrInvalidWizardInput, ShortFormatMaxSeconds, format.Name)
	case !short && format.IsShort():
		return fmt.Errorf("%w: format %q dành cho short dọc — chọn đầu ra Short dọc hoặc một format video dài",
			ErrInvalidWizardInput, format.Name)
	case short && engine != RenderEngineRemotion:
		return fmt.Errorf("%w: short dọc chỉ dựng được bằng Remotion", ErrInvalidWizardInput)
	}
	return nil
}

// failedStageOrder ranks each failed_at_<step> status by how far the saga got,
// in saga order. A status that is not here (qc, clips, publish) is past every
// step a wizard setting feeds.
var failedStageOrder = map[ProjectStatus]int{
	StatusFailedParseScript:      0,
	StatusFailedValidateScript:   1,
	StatusFailedSynthesizeSpeech: 2,
	StatusFailedRenderScenes:     3,
	StatusFailedAssembleVideo:    4,
}

// editableUntil reports whether a setting whose first real reader is the saga
// step ranked `stage` may still change: before the saga starts, at the review
// gate (nothing costly has run), or after a failure at or before that step —
// then a retry of the failed step, and every step after it, reads the new
// value. After the reader has run, a change would only relabel output that was
// made with the old value, so it is refused.
func editableUntil(status ProjectStatus, stage int) bool {
	if status == StatusDraft || status == StatusAwaitingReview {
		return true
	}
	order, ok := failedStageOrder[status]
	return ok && order <= stage
}

// renderStage and assembleStage are the failedStageOrder ranks of the steps
// that read the render settings (quality, video font) and the merge settings
// (subtitles, background music).
const (
	renderStage   = 3
	assembleStage = 4
)

// touchesAuthoring reports whether p changes anything the authoring steps
// (script, visual, code) or TTS read, or confirms step 2.
func (p WizardSettingsPatch) touchesAuthoring() bool {
	return p.ContentLanguage != nil || p.RenderEngine != nil || p.TTSEnabled != nil ||
		p.VoiceID != nil || p.VideoFormatID != nil || p.VideoOutputMode != nil || p.Confirm
}

func (p WizardSettingsPatch) touchesRender() bool {
	return p.RenderQuality != nil || p.VideoFont != nil
}

func (p WizardSettingsPatch) touchesAssemble() bool {
	return p.SubtitleMode != nil || p.SubtitleStyle != nil ||
		p.BackgroundMusicPath != nil || p.BackgroundMusicVolume != nil
}

// WizardPatchAllowed decides whether p may be stored on a project in status.
//
// Each group of settings stays editable up to the saga step that first reads
// it, so a failure there can be fixed in place and retried:
//   - authoring/TTS settings (language, engine, voice, format, output mode):
//     a draft or any failed project — unchanged from before;
//   - render settings (quality, video font): until render_scenes has run;
//   - merge settings (subtitles, background music): until assemble_video has run.
func WizardPatchAllowed(status ProjectStatus, p WizardSettingsPatch) bool {
	if p.touchesAuthoring() && !IsAuthoringEditable(status) {
		return false
	}
	if p.touchesRender() && !editableUntil(status, renderStage) {
		return false
	}
	if p.touchesAssemble() && !editableUntil(status, assembleStage) {
		return false
	}
	return true
}

// WizardStepForStatus is the wizard step a saga status belongs to. A draft
// says nothing here (0) — its step is whatever was stored.
func WizardStepForStatus(status ProjectStatus) int {
	switch status {
	case StatusDraft:
		return 0
	case StatusParsingScript, StatusValidatingScript, StatusAwaitingReview,
		StatusFailedParseScript, StatusFailedValidateScript,
		ProjectStatus("classifying_scenes"): // retired saga step; old rows may still hold it
		return WizardStepValidate
	case StatusReadyToPublish:
		return WizardStepResult
	case StatusPublishing, StatusPublished, StatusFailedPublishVideo:
		return WizardStepPublish
	default:
		return WizardStepProcess
	}
}

// EffectiveWizardStep is the step a Creator should land on for this project:
// the furthest of what was stored and what the saga status implies.
func EffectiveWizardStep(p *Project) int {
	step := p.WizardStep
	if s := WizardStepForStatus(p.Status); s > step {
		step = s
	}
	if step < WizardStepIdea {
		step = WizardStepIdea
	}
	return step
}
