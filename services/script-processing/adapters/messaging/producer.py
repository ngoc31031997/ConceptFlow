"""Event envelope builders for script_parsed / parse_failed (interface-contracts.md).

Only builds the envelope dict — the consumer writes it to the Outbox, and
OutboxRelay (adapters/persistence/relay.py) is the only place that
actually publishes (ADR-0013).
"""

from __future__ import annotations

import uuid
from datetime import UTC, datetime

from domain.models import Scene

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


def success_envelope(saga_id: str, project_id: str, scenes: list[Scene]) -> dict:
    return build_envelope(
        saga_id,
        project_id,
        {
            "event_type": "script_parsed",
            "scenes": [
                {
                    "scene_index": s.scene_index,
                    "narration_text": s.narration_text,
                    "illustration_hint": s.illustration_hint,
                    "code_snippet": s.code_snippet,
                    "code_language": s.code_language,
                }
                for s in scenes
            ],
        },
    )


def failure_envelope(saga_id: str, project_id: str, line_number: int | None, reason: str) -> dict:
    return build_envelope(
        saga_id,
        project_id,
        {
            "event_type": "parse_failed",
            "error_message": f"line {line_number}: {reason}" if line_number else reason,
            "line_number": line_number,
            "reason": reason,
        },
    )
