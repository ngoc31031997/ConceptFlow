"""Event envelope builders for video_assembled / assembly_failed
(interface-contracts.md).

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


def video_assembled_envelope(
    saga_id: str, project_id: str, video_path: str, caption_path: str | None = None
) -> dict:
    payload = {"event_type": "video_assembled", "video_path": video_path}
    if caption_path:
        # Absent rather than null when there is no caption track (CR-015
        # FR38.4) — mirrors how thumbnail_path already flows downstream.
        payload["caption_path"] = caption_path
    return build_envelope(saga_id, project_id, payload)


def assembly_failed_envelope(saga_id: str, project_id: str, error_message: str) -> dict:
    return build_envelope(
        saga_id, project_id, {"event_type": "assembly_failed", "error_message": error_message}
    )
