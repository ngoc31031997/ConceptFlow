"""AMQP command consumer — handles classify_scenes from
content_plugin.commands (interface-contracts.md, business-logic-model.md).
"""

from __future__ import annotations

import json
import logging
from typing import Protocol

from adapters.logging.correlation import set_correlation_id
from adapters.messaging.idempotency import IdempotencyStore
from adapters.messaging.producer import ScenesClassifiedEventPublisher
from application.classify_scene import BatchClassificationFailure, ClassifyScenesBatchUseCase
from domain.models import Scene

logger = logging.getLogger(__name__)


class AckableMessage(Protocol):
    """Minimal surface of aio-pika's IncomingMessage we depend on."""

    body: bytes

    async def ack(self) -> None: ...


class ClassifySceneCommandHandler:
    """Wires: idempotency check -> batch classify -> publish event -> ack.

    Depends only on ClassifyScenesBatchUseCase (application layer) and
    ScenesClassifiedEventPublisher (adapter) — the real aio-pika
    consuming loop (in main.py) just forwards each delivered message
    here.
    """

    def __init__(
        self,
        batch_use_case: ClassifyScenesBatchUseCase,
        publisher: ScenesClassifiedEventPublisher,
        idempotency_store: IdempotencyStore,
    ) -> None:
        self._batch_use_case = batch_use_case
        self._publisher = publisher
        self._idempotency_store = idempotency_store

    async def handle(self, message: AckableMessage) -> None:
        envelope = json.loads(message.body)
        message_id = envelope["message_id"]
        saga_id = envelope["saga_id"]
        project_id = envelope["project_id"]
        set_correlation_id(saga_id)

        if self._idempotency_store.already_processed(message_id):
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
            await self._publisher.publish_failure(saga_id, project_id, outcome.error_message)
        else:
            await self._publisher.publish_success(saga_id, project_id, outcome.results)

        self._idempotency_store.mark_processed(message_id)
        await message.ack()
