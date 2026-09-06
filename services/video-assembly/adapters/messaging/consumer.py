"""AMQP command consumer — handles assemble_video from video_assembly.commands
(interface-contracts.md).

Like Rendering Service, AssembleVideoUseCase.assemble() can run for a
while (ffmpeg mux/concat/overlay, up to ASSEMBLY_TIMEOUT_SECONDS). Calling
it directly from this coroutine would block the asyncio event loop for
that duration — starving RabbitMQ heartbeats and the OutboxRelay. It's
therefore run via asyncio.to_thread().

Unlike Rendering Service, there is no per-scene progress event — a single
command always produces exactly one Outbox row (Low-Level Design
Question 10), written in the same transaction as the Inbox mark.
"""

from __future__ import annotations

import asyncio
import json
import logging
from typing import Protocol

import asyncpg

from adapters.logging.correlation import set_correlation_id
from adapters.messaging.producer import assembly_failed_envelope, video_assembled_envelope
from adapters.persistence.inbox import InboxRepository
from adapters.persistence.outbox import OutboxRepository
from application.assemble_video import AssembleVideoUseCase
from domain.errors import AssemblyEngineError, MissingArtifactError
from domain.models import VideoAssemblyRequest

logger = logging.getLogger(__name__)


class AckableMessage(Protocol):
    """Minimal surface of aio-pika's IncomingMessage we depend on."""

    body: bytes

    async def ack(self) -> None: ...


class AssembleVideoCommandHandler:
    def __init__(
        self,
        use_case: AssembleVideoUseCase,
        pool: asyncpg.Pool,
        inbox: InboxRepository,
        outbox: OutboxRepository,
    ) -> None:
        self._use_case = use_case
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
        request = VideoAssemblyRequest(
            project_id=project_id,
            video_path=payload["video_path"],
            audio_segments=payload["audio_segments"],
            background_music_path=payload.get("background_music_path"),
        )

        try:
            result = await asyncio.to_thread(self._use_case.assemble, request)
        except (MissingArtifactError, AssemblyEngineError) as exc:
            logger.warning("assemble_video failed for project_id=%s: %s", project_id, exc)
            event_type = "assembly_failed"
            out_envelope = assembly_failed_envelope(saga_id, project_id, str(exc))
        else:
            event_type = "video_assembled"
            out_envelope = video_assembled_envelope(saga_id, project_id, result.video_path)

        async with self._pool.acquire() as conn, conn.transaction():
            await self._outbox.enqueue(
                conn, aggregate_id=project_id, event_type=event_type, envelope=out_envelope
            )
            await self._inbox.mark_processed(conn, message_id)

        await message.ack()
