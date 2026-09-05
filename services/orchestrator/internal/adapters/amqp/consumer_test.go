package amqp

import (
	"testing"

	"orchestrator/internal/domain"
)

// TestResolveEventType_PrefersPayloadEventType guards a real bug found via
// live E2E testing: every downstream Python service nests "event_type"
// inside payload (Unit 1's approved envelope standard), but Orchestrator
// used to read only a top-level envelope field that those services never
// set — silently dropping every single event and hanging every Saga.
func TestResolveEventType_PrefersPayloadEventType(t *testing.T) {
	envelope := domain.Envelope{
		Payload: map[string]interface{}{"event_type": "script_parsed", "scenes": []interface{}{}},
	}

	if got := resolveEventType(envelope); got != "script_parsed" {
		t.Fatalf("expected script_parsed from payload, got %q", got)
	}
}

// TestResolveEventType_FallsBackToTopLevel covers the DLQ decoding path,
// where Orchestrator's own dead-lettered commands carry EventType at the
// top level and have no nested payload.event_type.
func TestResolveEventType_FallsBackToTopLevel(t *testing.T) {
	envelope := domain.Envelope{
		EventType: "parse_script",
		Payload:   map[string]interface{}{"script_content": "## Scene 1\nhello"},
	}

	if got := resolveEventType(envelope); got != "parse_script" {
		t.Fatalf("expected fallback to top-level parse_script, got %q", got)
	}
}
