"""Event envelope builders for scenes_classified / classification_failed
(interface-contracts.md).

Revision (ADR-0013): this module used to publish directly to RabbitMQ.
It now only builds the envelope dict — the consumer writes it to the
Outbox in the same transaction as its Inbox mark, and OutboxRelay
(adapters/persistence/relay.py) is the only place that actually calls
exchange.publish().
"""

from __future__ import annotations

import uuid
from datetime import UTC, datetime

from domain.models import ClassificationResult

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


def success_envelope(saga_id: str, project_id: str, results: list[ClassificationResult]) -> dict:
    return build_envelope(
        saga_id,
        project_id,
        {
            "event_type": "scenes_classified",
            "scenes": [
                {
                    "scene_index": r.scene_index,
                    "category": r.category,
                    "animation_template_id": r.animation_template_id,
                }
                for r in results
            ],
        },
    )


def failure_envelope(saga_id: str, project_id: str, error_message: str) -> dict:
    return build_envelope(
        saga_id, project_id, {"event_type": "classification_failed", "error_message": error_message}
    )
