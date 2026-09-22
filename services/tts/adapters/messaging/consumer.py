"""AMQP command consumer — handles synthesize_speech from tts.commands
(interface-contracts.md, ADR-0014).

Revision (ADR-0013): idempotency + event publishing go through the
Inbox/Outbox pattern (adapters/persistence/), mirroring Content Plugin
Service — the consumer never publishes to RabbitMQ directly, it only
enqueues the event to the Outbox, atomically with marking the message
processed in the Inbox.
"""

from __future__ import annotations

import asyncio
import json
import logging
from typing import Protocol

import asyncpg

from adapters.logging.correlation import set_correlation_id
from adapters.messaging.producer import failure_envelope, success_envelope
from adapters.messaging.progress import ProgressPublisher
from adapters.persistence.inbox import InboxRepository
from adapters.persistence.outbox import OutboxRepository
from application.synthesize_speech_batch import (
    BatchSynthesisFailure,
    SceneSpeechRequest,
    SynthesizeSpeechBatchUseCase,
)

logger = logging.getLogger(__name__)


class AckableMessage(Protocol):
    """Minimal surface of aio-pika's IncomingMessage we depend on."""

    body: bytes

    async def ack(self) -> None: ...


class SynthesizeSpeechCommandHandler:
    """Wires: inbox check -> batch synthesize -> outbox enqueue (+ inbox mark) -> ack."""

    def __init__(
        self,
        batch_use_case: SynthesizeSpeechBatchUseCase,
        pool: asyncpg.Pool,
        inbox: InboxRepository,
        outbox: OutboxRepository,
        progress: ProgressPublisher | None = None,
    ) -> None:
        self._batch_use_case = batch_use_case
        self._pool = pool
        self._inbox = inbox
        self._outbox = outbox
        self._progress = progress

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
        scenes = [
            SceneSpeechRequest(
                scene_index=s["scene_index"],
                narration_text=s["narration_text"],
                language=s["language"],
                voice_id=s.get("voice_id"),
            )
            for s in payload["scenes"]
        ]

        def on_scene_done(scene_index: int, scene_total: int) -> None:
            if self._progress is None:
                return
            # Fire-and-forget from sync code: handle() is already running
            # inside the event loop, so scheduling a task here does not block
            # the batch loop waiting on RabbitMQ I/O (CR-029 — same posture as
            # rendering's heartbeat: progress must never slow down real work).
            asyncio.ensure_future(
                self._progress.publish_scene_progress(project_id, scene_index, scene_total)
            )

        outcome = self._batch_use_case.execute(project_id, scenes, on_scene_done=on_scene_done)

        if isinstance(outcome, BatchSynthesisFailure):
            logger.warning(
                "synthesize_speech failed for project_id=%s: %s", project_id, outcome.error_message
            )
            event_type = "synthesis_failed"
            out_envelope = failure_envelope(saga_id, project_id, outcome.error_message)
        else:
            event_type = "speech_synthesized"
            out_envelope = success_envelope(saga_id, project_id, outcome.results)

        async with self._pool.acquire() as conn, conn.transaction():
            await self._outbox.enqueue(
                conn, aggregate_id=project_id, event_type=event_type, envelope=out_envelope
            )
            await self._inbox.mark_processed(conn, message_id)

        await message.ack()
