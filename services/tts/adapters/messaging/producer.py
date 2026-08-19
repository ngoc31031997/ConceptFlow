"""Event envelope builders for speech_synthesized / synthesis_failed
(interface-contracts.md, ADR-0014).

Mirrors Content Plugin Service's producer.py (ADR-0013): only builds the
envelope dict — the consumer writes it to the Outbox, and OutboxRelay
(adapters/persistence/relay.py) is the only place that actually publishes.
"""

from __future__ import annotations

import uuid
from datetime import UTC, datetime

from application.synthesize_speech_batch import SceneSpeechResult

EVENTS_EXCHANGE = "events.direct"
EVENTS_ROUTING_KEY = "orchestrator"
SCHEMA_VERSION = "1.0"


def build_envelope(saga_id: str, project_id: str, payload: dict) -> dict:
    return {
        "message_id": str(uuid.uuid4()),
        "saga_id": saga_id,
        "project_id": project_id,
        "schema_version": SCHEMA_VERSION,
        "timestamp": datetime.now(UTC).isoformat(),
        "payload": payload,
    }


def success_envelope(saga_id: str, project_id: str, results: list[SceneSpeechResult]) -> dict:
    return build_envelope(
        saga_id,
        project_id,
        {
            "event_type": "speech_synthesized",
            "scenes": [
                {
                    "scene_index": r.scene_index,
                    "audio_path": r.audio_path,
                    "duration_seconds": r.duration_seconds,
                }
                for r in results
            ],
        },
    )


def failure_envelope(saga_id: str, project_id: str, error_message: str) -> dict:
    return build_envelope(
        saga_id, project_id, {"event_type": "synthesis_failed", "error_message": error_message}
    )
