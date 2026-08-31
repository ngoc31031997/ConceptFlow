// Package amqp implements the messaging adapter: consuming the 12 Saga
// events (+ 6 DLQ queues) from RabbitMQ and publishing commands/progress
// messages, using amqp091-go directly (Go equivalent of the Python units'
// producer.py build_envelope convention — messaging-design.md).
package amqp

import (
	"encoding/json"

	"orchestrator/internal/domain"
)

// wireEnvelope is the JSON-on-the-wire shape shared by inbound events and
// outbound commands (messaging-design.md "Event Schema").
type wireEnvelope struct {
	MessageID string                 `json:"message_id"`
	SagaID    string                 `json:"saga_id"`
	ProjectID string                 `json:"project_id"`
	EventType string                 `json:"event_type"`
	Payload   map[string]interface{} `json:"payload"`
	Timestamp string                 `json:"timestamp"`
}

// EncodeEnvelope serializes a domain.Envelope (outbound command) to JSON
// bytes for publishing.
func EncodeEnvelope(e domain.Envelope) ([]byte, error) {
	w := wireEnvelope{
		MessageID: e.MessageID,
		SagaID:    e.SagaID,
		ProjectID: e.ProjectID,
		EventType: e.EventType,
		Payload:   e.Payload,
		Timestamp: e.Timestamp,
	}
	return json.Marshal(w)
}

// DecodeEnvelope parses inbound JSON bytes (an event delivered on
// orchestrator.events or a DLQ queue) into a domain.Envelope.
func DecodeEnvelope(body []byte) (domain.Envelope, error) {
	var w wireEnvelope
	if err := json.Unmarshal(body, &w); err != nil {
		return domain.Envelope{}, err
	}
	return domain.Envelope{
		MessageID: w.MessageID,
		SagaID:    w.SagaID,
		ProjectID: w.ProjectID,
		EventType: w.EventType,
		Payload:   w.Payload,
		Timestamp: w.Timestamp,
	}, nil
}
