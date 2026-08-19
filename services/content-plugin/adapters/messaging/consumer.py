"""AMQP command consumer — handles classify_scenes from
content_plugin.commands (interface-contracts.md, business-logic-model.md).

Revision (ADR-0013): idempotency + event publishing now go through the
Inbox/Outbox pattern (adapters/persistence/) instead of an in-memory
IdempotencyStore + direct exchange.publish(). The consumer never calls
RabbitMQ to publish — it only enqueues the event to the Outbox table,
atomically with marking the message processed in the Inbox table.
"""

from __future__ import annotations

import json
import logging
from typing import Protocol

import asyncpg

from adapters.logging.correlation import set_correlation_id
from adapters.messaging.producer import failure_envelope, success_envelope
from adapters.persistence.inbox import InboxRepository
from adapters.persistence.outbox import OutboxRepository
from application.classify_scene import BatchClassificationFailure, ClassifyScenesBatchUseCase
from domain.models import Scene

logger = logging.getLogger(__name__)


class AckableMessage(Protocol):
    """Minimal surface of aio-pika's IncomingMessage we depend on."""

    body: bytes

    async def ack(self) -> None: ...


class ClassifySceneCommandHandler:
    """Wires: inbox check -> batch classify -> outbox enqueue (+ inbox mark) -> ack."""

    def __init__(
        self,
        batch_use_case: ClassifyScenesBatchUseCase,
        pool: asyncpg.Pool,
        inbox: InboxRepository,
        outbox: OutboxRepository,
    ) -> None:
        self._batch_use_case = batch_use_case
        self._pool = pool
        self._inbox = inbox
        self._outbox = outbox

    async def handle(self, message: AckableMessage) -> None:
        envelope = json.loads(message.body)
        message_id = envelope["message_id"]
        saga_id = envelope["saga_id"]
        project_id = envelope["project_id"]
        set_correlation_id(saga_id)

        if await self._inbox.has_processed(message_id):
            logger.info("Skipping already-processed message_id=%s", message_id)
            await message.ack()
            return

        payload = envelope["payload"]
        plugin_id = payload["plugin_id"]
        scenes = [
            Scene(
                scene_index=s["scene_index"],
                narration_text=s["narration_text"],
                category_hint=s["category_hint"],
                illustration_hint=s.get("illustration_hint"),
                code_snippet=s.get("code_snippet"),
            )
            for s in payload["scenes"]
        ]

        outcome = self._batch_use_case.execute(plugin_id, scenes)

        if isinstance(outcome, BatchClassificationFailure):
            logger.warning("classify_scenes failed for project_id=%s: %s", project_id, outcome.error_message)
            event_type = "classification_failed"
            out_envelope = failure_envelope(saga_id, project_id, outcome.error_message)
        else:
            event_type = "scenes_classified"
            out_envelope = success_envelope(saga_id, project_id, outcome.results)

        async with self._pool.acquire() as conn, conn.transaction():
            await self._outbox.enqueue(
                conn, aggregate_id=project_id, event_type=event_type, envelope=out_envelope
            )
            await self._inbox.mark_processed(conn, message_id)

        await message.ack()
