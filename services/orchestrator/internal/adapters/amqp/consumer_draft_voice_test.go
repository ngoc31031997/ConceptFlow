package amqp

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	amqp "github.com/rabbitmq/amqp091-go"

	"orchestrator/internal/application"
)

type recordingAck struct {
	acked, nacked int
}

func (r *recordingAck) Ack(uint64, bool) error        { r.acked++; return nil }
func (r *recordingAck) Nack(uint64, bool, bool) error { r.nacked++; return nil }
func (r *recordingAck) Reject(uint64, bool) error     { return nil }

type fakeInbox struct{ processed []string }

func (f *fakeInbox) HasProcessed(context.Context, string) (bool, error) { return false, nil }
func (f *fakeInbox) MarkProcessed(_ context.Context, id string) error {
	f.processed = append(f.processed, id)
	return nil
}

type sagaMustNotRun struct{ called int }

func (s *sagaMustNotRun) Execute(context.Context, application.StepEvent) error {
	s.called++
	return nil
}

type fakeDraftVoiceHandler struct {
	events []string
	failed []string
	err    error
}

func (f *fakeDraftVoiceHandler) HandleEvent(_ context.Context, _, eventType string, _ map[string]interface{}) error {
	f.events = append(f.events, eventType)
	return f.err
}
func (f *fakeDraftVoiceHandler) FailUndelivered(_ context.Context, _, reason string) error {
	f.failed = append(f.failed, reason)
	return nil
}

func delivery(t *testing.T, ack *recordingAck, body map[string]interface{}) amqp.Delivery {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return amqp.Delivery{Acknowledger: ack, Body: raw}
}

func TestLinesAnswersGoToTheDraftVoiceNotTheSaga(t *testing.T) {
	ack, inbox, saga, voice := &recordingAck{}, &fakeInbox{}, &sagaMustNotRun{}, &fakeDraftVoiceHandler{}
	c := NewConsumer(nil, inbox, saga, nil, slog.Default()).WithDraftVoice(voice)
	c.handleEventDelivery(context.Background(), delivery(t, ack, map[string]interface{}{
		"message_id": "m1", "saga_id": "s", "project_id": "p1",
		"payload": map[string]interface{}{"event_type": "lines_synthesized", "lines": []interface{}{}},
	}))
	if saga.called != 0 || len(voice.events) != 1 || voice.events[0] != "lines_synthesized" || ack.acked != 1 || len(inbox.processed) != 1 {
		t.Errorf("saga=%d events=%v acked=%d processed=%v", saga.called, voice.events, ack.acked, inbox.processed)
	}
}

func TestADraftVoiceAnswerThatKeepsFailingIsGivenUpOnAndTheLinesFailed(t *testing.T) {
	ack, inbox, voice := &recordingAck{}, &fakeInbox{}, &fakeDraftVoiceHandler{err: errors.New("authoring down")}
	c := NewConsumer(nil, inbox, &sagaMustNotRun{}, nil, slog.Default()).WithDraftVoice(voice)
	for i := 0; i < maxEventAttempts-1; i++ {
		bumpEventAttempt("m-fail")
	}
	c.handleEventDelivery(context.Background(), delivery(t, ack, map[string]interface{}{
		"message_id": "m-fail", "saga_id": "s", "project_id": "p1",
		"payload": map[string]interface{}{"event_type": "lines_synthesized"},
	}))
	if len(voice.failed) != 1 || ack.acked != 1 || ack.nacked != 0 {
		t.Errorf("failed=%v acked=%d nacked=%d: after the last attempt the lines must be failed and the message acked", voice.failed, ack.acked, ack.nacked)
	}
}

func TestADeadLetteredSynthesizeLinesCommandFailsOnlyItsLines(t *testing.T) {
	ack, voice := &recordingAck{}, &fakeDraftVoiceHandler{}
	// A nil repo proves the saga step is not touched: using it would panic.
	c := NewConsumer(nil, &fakeInbox{}, &sagaMustNotRun{}, nil, slog.Default()).WithDraftVoice(voice)
	c.handleDLQDelivery(context.Background(), delivery(t, ack, map[string]interface{}{
		"message_id": "m2", "saga_id": "s", "project_id": "p1", "event_type": "synthesize_lines",
		"payload": map[string]interface{}{"event_type": "synthesize_lines"},
	}))
	if len(voice.failed) != 1 || ack.acked != 1 {
		t.Errorf("failed=%v acked=%d", voice.failed, ack.acked)
	}
}
