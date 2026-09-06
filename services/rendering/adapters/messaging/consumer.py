"""AMQP command consumer — handles render_scenes from rendering.commands
(Manim-script input mode).

RenderScriptUseCase.render() can legitimately run for minutes (a full Manim
subprocess, up to RENDER_TIMEOUT_SECONDS). Calling it directly from this
coroutine would block the asyncio event loop for that entire duration —
starving RabbitMQ heartbeats and the OutboxRelay. It's therefore run via
asyncio.to_thread().
"""

from __future__ import annotations

import asyncio
import json
import logging
from typing import Protocol

import asyncpg

from adapters.logging.correlation import set_correlation_id
from adapters.messaging.producer import rendering_completed_envelope, rendering_failed_envelope
from adapters.persistence.inbox import InboxRepository
from adapters.persistence.outbox import OutboxRepository
from application.render_script import RenderScriptUseCase
from domain.errors import AnimationEngineError, InvalidDurationError
from domain.models import NarrationSegment, ScriptRenderRequest

logger = logging.getLogger(__name__)


class AckableMessage(Protocol):
    """Minimal surface of aio-pika's IncomingMessage we depend on."""

    body: bytes

    async def ack(self) -> None: ...


class RenderScriptCommandHandler:
    def __init__(
        self,
        use_case: RenderScriptUseCase,
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
        request = ScriptRenderRequest(
            project_id=project_id,
            script_content=payload["script_content"],
            scene_class_name=payload["scene_class_name"],
            narration_segments=[
                NarrationSegment(
                    scene_index=s["scene_index"],
                    audio_path=s["audio_path"],
                    duration_seconds=s["duration_seconds"],
                )
                for s in payload["scenes"]
            ],
        )

        try:
            result = await asyncio.to_thread(self._use_case.render, request)
        except (ValueError, InvalidDurationError, AnimationEngineError) as exc:
            logger.warning("render_scenes failed for project_id=%s: %s", project_id, exc)
            event_type = "rendering_failed"
            final_envelope = rendering_failed_envelope(saga_id, project_id, str(exc))
        else:
            event_type = "rendering_completed"
            final_envelope = rendering_completed_envelope(saga_id, project_id, result.video_path)

        async with self._pool.acquire() as conn, conn.transaction():
            await self._outbox.enqueue(
                conn, aggregate_id=project_id, event_type=event_type, envelope=final_envelope
            )
            await self._inbox.mark_processed(conn, message_id)

        await message.ack()
