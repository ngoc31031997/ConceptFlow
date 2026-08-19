"""Event envelope builders for the 4 Rendering Service event types
(interface-contracts.md).

Only builds envelope dicts — the consumer writes them to the Outbox, and
OutboxRelay (adapters/persistence/relay.py) is the only place that
actually publishes (ADR-0013).
"""

from __future__ import annotations

import uuid
from datetime import UTC, datetime

from domain.models import SceneRenderResult

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


def scene_render_started_envelope(saga_id: str, project_id: str, scene_index: int) -> dict:
    return build_envelope(
        saga_id, project_id, {"event_type": "scene_render_started", "scene_index": scene_index}
    )


def scene_rendered_envelope(
    saga_id: str, project_id: str, scene_index: int, result: SceneRenderResult
) -> dict:
    return build_envelope(
        saga_id,
        project_id,
        {
            "event_type": "scene_rendered",
            "scene_index": scene_index,
            "animation_path": result.animation_path,
            "duration_seconds": result.duration_seconds,
        },
    )


def rendering_completed_envelope(saga_id: str, project_id: str, scene_count: int) -> dict:
    return build_envelope(
        saga_id, project_id, {"event_type": "rendering_completed", "scene_count": scene_count}
    )


def rendering_failed_envelope(saga_id: str, project_id: str, scene_index: int, error_message: str) -> dict:
    return build_envelope(
        saga_id,
        project_id,
        {"event_type": "rendering_failed", "scene_index": scene_index, "error_message": error_message},
    )
