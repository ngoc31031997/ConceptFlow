package domain

import "testing"

func TestFlowStateFor(t *testing.T) {
	cases := []struct {
		name    string
		status  ProjectStatus
		wizard  int
		content AuthoredContent
		want    FlowState
	}{
		{"fresh draft row is already past init", StatusDraft, 1, AuthoredContent{}, FlowState{FlowConfig, RunIdle}},
		{"config", StatusDraft, 2, AuthoredContent{}, FlowState{FlowConfig, RunIdle}},
		{"script, nothing yet", StatusDraft, 3, AuthoredContent{}, FlowState{FlowStory, RunIdle}},
		{"story done → visual is next", StatusDraft, 3, AuthoredContent{Story: true}, FlowState{FlowVisual, RunIdle}},
		{"storyboard done → code is next", StatusDraft, 3, AuthoredContent{Story: true, Storyboard: true}, FlowState{FlowCode, RunIdle}},
		{"code beats stale route", StatusDraft, 3, AuthoredContent{Story: true, Storyboard: true, Code: true}, FlowState{FlowCode, RunIdle}},
		{"validating", StatusValidatingScript, 3, AuthoredContent{}, FlowState{FlowValidate, RunRunning}},
		{"validate failed", StatusFailedValidateScript, 3, AuthoredContent{}, FlowState{FlowValidate, RunFailed}},
		{"review", StatusAwaitingReview, 3, AuthoredContent{}, FlowState{FlowReview, RunIdle}},
		{"tts", StatusSynthesizingSpeech, 3, AuthoredContent{}, FlowState{FlowTTS, RunRunning}},
		{"render failed", StatusFailedRenderScenes, 3, AuthoredContent{}, FlowState{FlowRender, RunFailed}},
		{"qc is part of merge", StatusRunningQC, 3, AuthoredContent{}, FlowState{FlowMerge, RunRunning}},
		{"clips", StatusGeneratingClips, 3, AuthoredContent{}, FlowState{FlowSplit, RunRunning}},
		{"result", StatusReadyToPublish, 3, AuthoredContent{}, FlowState{FlowResult, RunIdle}},
		{"published", StatusPublished, 3, AuthoredContent{}, FlowState{FlowPublish, RunDone}},
	}
	for _, c := range cases {
		if got := FlowStateFor(c.status, c.wizard, c.content); got != c.want {
			t.Errorf("%s: got %+v, want %+v", c.name, got, c.want)
		}
	}
}

func TestFlowStateCoversEveryFailedStatus(t *testing.T) {
	steps := []StepName{StepParseScript, StepValidateScript, StepSynthesizeSpeech, StepRenderScenes,
		StepAssembleVideo, StepQCVideo, StepGenerateClips, StepPublishVideo}
	for _, s := range steps {
		st := FlowStateFor(FailedStatusForStep(s), 3, AuthoredContent{})
		if st.State != RunFailed {
			t.Errorf("%s: failed status maps to %+v, want run_state failed", s, st)
		}
	}
}

func TestRunStateOfReportsACancelledStepAsCancelled(t *testing.T) {
	cancelled := CancelledErrorMessage
	boom := "ffmpeg failed"
	if got := RunStateOf(StatusFailedRenderScenes, &cancelled); got != RunCancelled {
		t.Errorf("cancelled: got %s", got)
	}
	if got := RunStateOf(StatusFailedRenderScenes, &boom); got != RunFailed {
		t.Errorf("failed: got %s", got)
	}
	if got := RunStateOf(StatusRendering, &cancelled); got != RunRunning {
		t.Errorf("a stale message on a running project must not read as cancelled: got %s", got)
	}
}
