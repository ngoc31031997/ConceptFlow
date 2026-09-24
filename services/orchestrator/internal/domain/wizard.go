package domain

// Wizard steps the Creator walks through. The GUI shows exactly seven; the
// backend stores which one a project has reached so a reload, another browser
// or a restart of the stack resumes in the right place.
const (
	WizardStepIdea     = 1 // Ý tưởng — topic / situation
	WizardStepConfig   = 2 // Cấu hình — language, engine, authoring mode, voice, format, quality, music
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

// wizardRoutes maps every wizard screen a draft can be left on to the wizard
// step it belongs to.
var wizardRoutes = map[string]int{
	"/":                         WizardStepIdea,
	"/create/script/settings":   WizardStepConfig,
	"/create/script/outline":    WizardStepScript,
	"/create/script/storyboard": WizardStepScript,
	"/create/script/code":       WizardStepScript,
}

// WizardStepForRoute returns the wizard step of a route and whether the route
// is a known wizard screen. The client sends the route, so it is validated
// here rather than stored blindly.
func WizardStepForRoute(route string) (int, bool) {
	step, ok := wizardRoutes[route]
	return step, ok
}
