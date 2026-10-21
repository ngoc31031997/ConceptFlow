package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"orchestrator/internal/domain"
)

type fakeDraftAuthoring struct {
	claim   []VoiceLine
	claimed int
	results []VoiceLineResult
	failed  []string
	err     error
}

func (f *fakeDraftAuthoring) ClaimVoiceLines(context.Context, string) ([]VoiceLine, error) {
	f.claimed++
	return f.claim, f.err
}
func (f *fakeDraftAuthoring) SaveVoiceResults(_ context.Context, _ string, r []VoiceLineResult) error {
	f.results = r
	return f.err
}
func (f *fakeDraftAuthoring) FailVoiceLines(_ context.Context, _, reason string) error {
	f.failed = append(f.failed, reason)
	return nil
}

type captureCommands struct {
	key string
	env domain.Envelope
	n   int
	err error
}

func (c *captureCommands) PublishCommand(_ context.Context, key string, e domain.Envelope) error {
	c.key, c.env = key, e
	c.n++
	return c.err
}

func TestDispatchSendsTheWaitingLinesToTTSInOneCommand(t *testing.T) {
	a := &fakeDraftAuthoring{claim: []VoiceLine{
		{LineID: "b1.1", Text: "Câu một.", Language: "vi", VoiceID: "v", Ref: "r1"},
		{LineID: "b1.2", Text: "Câu hai.", Language: "vi", VoiceID: "v", Ref: "r2"},
	}}
	c := &captureCommands{}
	if err := NewDraftVoiceUseCase(a, c).Dispatch(context.Background(), "p1"); err != nil {
		t.Fatal(err)
	}
	if c.n != 1 || c.key != "tts" || c.env.ProjectID != "p1" || c.env.EventType != CommandSynthesizeLines || c.env.MessageID == "" || c.env.SagaID == "" || c.env.Timestamp == "" {
		t.Fatalf("command = %+v key %q", c.env, c.key)
	}
	lines := c.env.Payload["lines"].([]map[string]interface{})
	if len(lines) != 2 || lines[0]["line_id"] != "b1.1" || lines[1]["ref"] != "r2" || c.env.Payload["event_type"] != CommandSynthesizeLines {
		t.Errorf("payload = %+v", c.env.Payload)
	}
}

func TestDispatchWithNothingWaitingSendsNothing(t *testing.T) {
	a, c := &fakeDraftAuthoring{}, &captureCommands{}
	if err := NewDraftVoiceUseCase(a, c).Dispatch(context.Background(), "p1"); err != nil || c.n != 0 {
		t.Errorf("err=%v commands=%d", err, c.n)
	}
}

func TestDispatchFailsTheLinesWhenTheCommandCannotBeQueued(t *testing.T) {
	a := &fakeDraftAuthoring{claim: []VoiceLine{{LineID: "b1.1", Text: "Câu.", Language: "vi", Ref: "r"}}}
	c := &captureCommands{err: errors.New("outbox down")}
	err := NewDraftVoiceUseCase(a, c).Dispatch(context.Background(), "p1")
	if err == nil || len(a.failed) != 1 || !strings.Contains(a.failed[0], "outbox down") {
		t.Errorf("err=%v failed=%v: a command that was not queued must fail the lines, not leave them running", err, a.failed)
	}
}

func TestHandleEventPassesFilesBackAndFailuresToo(t *testing.T) {
	a := &fakeDraftAuthoring{}
	uc := NewDraftVoiceUseCase(a, &captureCommands{})
	err := uc.HandleEvent(context.Background(), "p1", EventLinesSynthesized, map[string]interface{}{
		"lines": []interface{}{
			map[string]interface{}{"line_id": "b1.1", "ref": "r1", "audio_path": "/shared/p1/audio/b1-1_k.wav", "duration_seconds": 3.5, "cache_key": "k", "cache_hit": true},
			map[string]interface{}{"line_id": "b1.2", "ref": "r2", "error": "không đọc được"},
		},
	})
	if err != nil || len(a.results) != 2 || a.results[0].DurationSeconds != 3.5 || !a.results[0].CacheHit || a.results[1].Error == "" {
		t.Fatalf("err=%v results=%+v", err, a.results)
	}
	if err := uc.HandleEvent(context.Background(), "p1", EventLinesFailed, map[string]interface{}{"error_message": "engine hỏng"}); err != nil || a.failed[0] != "engine hỏng" {
		t.Errorf("err=%v failed=%v", err, a.failed)
	}
	if err := uc.HandleEvent(context.Background(), "p1", EventLinesSynthesized, map[string]interface{}{}); err == nil {
		t.Error("an answer without lines is an error, not a success")
	}
}

func TestIsDraftVoiceEvent(t *testing.T) {
	if !IsDraftVoiceEvent("lines_synthesized") || !IsDraftVoiceEvent("lines_synthesis_failed") || IsDraftVoiceEvent("speech_synthesized") {
		t.Error("only the two draft voice answers are draft voice events")
	}
}
