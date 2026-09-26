package application

import (
	"testing"

	"orchestrator/internal/domain"
)

func TestClassifySagaFailureOnlySaysWhatTheTextSupports(t *testing.T) {
	cases := []struct {
		step domain.StepName
		msg  string
		want string
	}{
		{domain.StepRenderScenes, "Manim render timed out after 1800s", "hết thời gian"},
		{domain.StepSynthesizeSpeech, "Azure TTS failed: quota exceeded", "hết hạn mức"},
		{domain.StepAssembleVideo, "Failed to reach upstream: connection refused", "hạ tầng"},
		{domain.StepValidateScript, "NameError: name 'x' is not defined", "đầu vào"},
		{domain.StepParseScript, "syntax error", "đầu vào"},
		{domain.StepRenderScenes, "ffmpeg exited with code 1", ""}, // unknown stays unknown
	}
	for _, c := range cases {
		if got := ClassifySagaFailure(c.step, c.msg); got != c.want {
			t.Errorf("%s %q: got %q, want %q", c.step, c.msg, got, c.want)
		}
	}
}
