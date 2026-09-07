"""Event envelope builders for the Rendering Service's 2 event types
(rendering_completed / rendering_failed — Manim-script input mode).

Only builds envelope dicts — the consumer writes them to the Outbox, and
OutboxRelay (adapters/persistence/relay.py) is the only place that
actually publishes (ADR-0013).
"""

from __future__ import annotations

import uuid
from datetime import UTC, datetime

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


def rendering_completed_envelope(
    saga_id: str,
    project_id: str,
    video_path: str,
    wait_offsets: list[float],
    video_duration_seconds: float,
) -> dict:
    """wait_offsets / video_duration_seconds added by CR-002 (FR10.2).

    wait_offsets[i] is where narration segment i actually starts in the
    rendered video. The Orchestrator validates it against its own scene count
    before passing it to Video Assembly.
    """
    return build_envelope(
        saga_id,
        project_id,
        {
            "event_type": "rendering_completed",
            "video_path": video_path,
            "wait_offsets": wait_offsets,
            "video_duration_seconds": video_duration_seconds,
        },
    )


def rendering_failed_envelope(saga_id: str, project_id: str, error_message: str) -> dict:
    return build_envelope(
        saga_id, project_id, {"event_type": "rendering_failed", "error_message": error_message}
    )
