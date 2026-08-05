"""AMQP event producer — publishes scenes_classified / classification_failed
to the events.direct exchange (routing key "orchestrator"), per
messaging-design.md (Unit 1) and interface-contracts.md (Unit 2).
"""

from __future__ import annotations

import json
import uuid
from datetime import UTC, datetime
from typing import Protocol

from domain.models import ClassificationResult

EVENTS_EXCHANGE = "events.direct"
EVENTS_ROUTING_KEY = "orchestrator"
SCHEMA_VERSION = "1.0"


class PublishableExchange(Protocol):
    """Minimal surface of aio-pika's Exchange we depend on — lets tests
    inject a fake without pulling in a real AMQP connection."""

    async def publish(self, message: object, routing_key: str) -> None: ...


def _envelope(saga_id: str, project_id: str, payload: dict) -> dict:
    return {
        "message_id": str(uuid.uuid4()),
        "saga_id": saga_id,
        "project_id": project_id,
        "schema_version": SCHEMA_VERSION,
        "timestamp": datetime.now(UTC).isoformat(),
        "payload": payload,
    }


class ScenesClassifiedEventPublisher:
    def __init__(self, exchange: PublishableExchange, make_message) -> None:
        # `make_message` wraps a body into an aio-pika Message (durable,
        # persistent) — injected so this class stays free of aio-pika
        # imports and is trivially unit-testable.
        self._exchange = exchange
        self._make_message = make_message

    async def publish_success(
        self, saga_id: str, project_id: str, results: list[ClassificationResult]
    ) -> None:
        envelope = _envelope(
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
        await self._publish(envelope)

    async def publish_failure(self, saga_id: str, project_id: str, error_message: str) -> None:
        envelope = _envelope(
            saga_id,
            project_id,
            {"event_type": "classification_failed", "error_message": error_message},
        )
        await self._publish(envelope)

    async def _publish(self, envelope: dict) -> None:
        body = json.dumps(envelope).encode("utf-8")
        message = self._make_message(body)
        await self._exchange.publish(message, routing_key=EVENTS_ROUTING_KEY)
