package amqp

import (
	"testing"

	"orchestrator/internal/domain"
)

func TestEnvelope_RoundTrip(t *testing.T) {
	original := domain.Envelope{
		MessageID: "msg-1",
		SagaID:    "saga-1",
		ProjectID: "proj-1",
		EventType: "render_scenes",
		Payload: map[string]interface{}{
			"scenes": []interface{}{
				map[string]interface{}{"scene_index": float64(0), "narration_text": "hello"},
			},
		},
		Timestamp: "2026-08-31T00:00:00Z",
	}

	bytes, err := EncodeEnvelope(original)
	if err != nil {
		t.Fatalf("unexpected encode error: %v", err)
	}

	decoded, err := DecodeEnvelope(bytes)
	if err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}

	if decoded.MessageID != original.MessageID || decoded.SagaID != original.SagaID ||
		decoded.ProjectID != original.ProjectID || decoded.EventType != original.EventType ||
		decoded.Timestamp != original.Timestamp {
		t.Fatalf("round-trip mismatch: got %+v, want %+v", decoded, original)
	}

	scenes, ok := decoded.Payload["scenes"].([]interface{})
	if !ok || len(scenes) != 1 {
		t.Fatalf("expected payload.scenes to round-trip as []interface{} with 1 item, got %v", decoded.Payload["scenes"])
	}
}

func TestDecodeEnvelope_InvalidJSON(t *testing.T) {
	if _, err := DecodeEnvelope([]byte("not json")); err == nil {
		t.Fatal("expected an error decoding invalid JSON")
	}
}
