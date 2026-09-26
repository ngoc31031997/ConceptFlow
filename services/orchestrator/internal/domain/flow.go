package domain

// The 13-step production flow the Creator sees. It is derived from the saga
// status plus what the draft already holds, never stored: one function decides
// where a project is, so the wizard, the project list and the event log cannot
// disagree the way wizard_route and the real content used to.
//
// Review (7) is a screen, not a saga state: validate finishes at
// awaiting_review and nothing runs until the Creator presses "start render".
const (
	FlowInit       = 1  // Khởi tạo — topic, project row
	FlowConfig     = 2  // Cấu hình
	FlowStory      = 3  // Kịch bản
	FlowVisual     = 4  // Visual
	FlowCode       = 5  // Code
	FlowValidate   = 6  // Validate (parse + dry run)
	FlowReview     = 7  // Review — screen only
	FlowTTS        = 8  // TTS
	FlowRender     = 9  // Render hoạt hình
	FlowMerge      = 10 // Merge (+ QC)
	FlowSplit      = 11 // Cắt video short
	FlowResult     = 12 // Kết quả
	FlowPublish    = 13 // Publish
	FlowStepsTotal = 13
)

// RunState is what is happening at the current flow step.
type RunState string

const (
	RunIdle    RunState = "idle"    // waiting for the Creator
	RunRunning RunState = "running" // a worker or the AI is on it
	RunFailed  RunState = "failed"  // stopped with an error the Creator must read
	RunDone    RunState = "done"    // finished, nothing further at this step
	// RunCancelled is a failed_at_<step> project whose step the Creator stopped
	// on purpose. Same recovery as a failure (retry that step), different words.
	RunCancelled RunState = "cancelled"
)

// CancelledErrorMessage is the error message a user-cancelled step is left
// with. It marks the difference between "stopped by the Creator" and "failed".
const CancelledErrorMessage = "Đã huỷ bởi người dùng"

// RunStateOf is FlowStateFor's run state, with a failed step the Creator
// cancelled reported as cancelled.
func RunStateOf(status ProjectStatus, errorMessage *string) RunState {
	st := FlowStateFor(status, 0, AuthoredContent{}).State
	if st == RunFailed && errorMessage != nil && *errorMessage == CancelledErrorMessage {
		return RunCancelled
	}
	return st
}

// AuthoredContent says which of the three authoring artefacts a draft holds.
type AuthoredContent struct {
	Story, Storyboard, Code bool
}

// FlowState is where a project stands in the 13-step flow.
type FlowState struct {
	Step  int      `json:"step"`
	State RunState `json:"run_state"`
}

// A draft on the script steps stands at the first of 3/4/5 whose result is
// still missing (story done → Visual; storyboard done → Code), so a project
// forked "from Visual" sits at step 4 and an opened draft resumes where work is.
//
// FlowStateFor derives the flow position. storedWizardStep is the raw wizard
// step of a draft (1, 2 or 3); content decides which of 3/4/5 a draft on the
// script step is really at, so a draft whose route says "outline" but whose
// storyboard and code already exist reports the true step.
func FlowStateFor(status ProjectStatus, storedWizardStep int, content AuthoredContent) FlowState {
	switch status {
	case StatusDraft:
		switch {
		case storedWizardStep <= WizardStepConfig:
			// A server-side draft exists only once the Creator got past step 1
			// (the row is created by "Tiếp tục"), so the earliest place it can
			// be is Config. FlowInit is a client-only state before the row.
			return FlowState{FlowConfig, RunIdle}
		case content.Storyboard:
			// Storyboard done: what is left is the code (also where a draft with
			// code already sits, waiting for the Creator to start validation).
			return FlowState{FlowCode, RunIdle}
		case content.Story:
			return FlowState{FlowVisual, RunIdle}
		}
		return FlowState{FlowStory, RunIdle}
	case StatusParsingScript, StatusValidatingScript, ProjectStatus("classifying_scenes"):
		return FlowState{FlowValidate, RunRunning}
	case StatusFailedParseScript, StatusFailedValidateScript:
		return FlowState{FlowValidate, RunFailed}
	case StatusAwaitingReview:
		return FlowState{FlowReview, RunIdle}
	case StatusSynthesizingSpeech:
		return FlowState{FlowTTS, RunRunning}
	case StatusFailedSynthesizeSpeech:
		return FlowState{FlowTTS, RunFailed}
	case StatusRendering:
		return FlowState{FlowRender, RunRunning}
	case StatusFailedRenderScenes:
		return FlowState{FlowRender, RunFailed}
	case StatusAssemblingVideo, StatusRunningQC:
		return FlowState{FlowMerge, RunRunning}
	case StatusFailedAssembleVideo, StatusFailedQCVideo:
		return FlowState{FlowMerge, RunFailed}
	case StatusGeneratingClips:
		return FlowState{FlowSplit, RunRunning}
	case StatusFailedGenerateClips:
		return FlowState{FlowSplit, RunFailed}
	case StatusReadyToPublish:
		return FlowState{FlowResult, RunIdle}
	case StatusPublishing:
		return FlowState{FlowPublish, RunRunning}
	case StatusPublished:
		return FlowState{FlowPublish, RunDone}
	case StatusFailedPublishVideo:
		return FlowState{FlowPublish, RunFailed}
	}
	return FlowState{FlowInit, RunIdle}
}

// FlowStepForAuthoring maps an authoring step name to its flow step.
func FlowStepForAuthoring(step string) int {
	switch step {
	case "story":
		return FlowStory
	case "storyboard":
		return FlowVisual
	case "code":
		return FlowCode
	}
	return 0
}

// FlowStepLabel is the Creator-facing name of a flow step.
var FlowStepLabel = map[int]string{
	FlowInit: "Khởi tạo", FlowConfig: "Cấu hình", FlowStory: "Kịch bản", FlowVisual: "Visual",
	FlowCode: "Code", FlowValidate: "Validate", FlowReview: "Review", FlowTTS: "TTS",
	FlowRender: "Render hoạt hình", FlowMerge: "Merge", FlowSplit: "Cắt video short",
	FlowResult: "Kết quả", FlowPublish: "Publish",
}
