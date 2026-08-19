"""AMQP command consumer — handles render_scenes from rendering.commands
(interface-contracts.md).

Unlike the other services' consumers, RenderScenesBatchUseCase.execute()
can legitimately run for minutes (Manim rendering, up to
RENDER_TIMEOUT_SECONDS per scene). Calling it directly from this
coroutine would block the asyncio event loop for that entire duration —
starving RabbitMQ heartbeats and the OutboxRelay. It's therefore run via
asyncio.to_thread(), with the batch's on_scene_start/on_scene_rendered
callbacks bridging back into the event loop (run_coroutine_threadsafe)
to perform their per-scene Outbox writes, each committed immediately
(NFR Design) so progress is visible in real time.
"""

from __future__ import annotations

import asyncio
import json
import logging
from typing import Protocol

import asyncpg

from adapters.logging.correlation import set_correlation_id
from adapters.messaging.producer import (
    rendering_completed_envelope,
    rendering_failed_envelope,
    scene_render_started_envelope,
    scene_rendered_envelope,
)
from adapters.persistence.inbox import InboxRepository
from adapters.persistence.outbox import OutboxRepository
from application.render_scenes_batch import BatchRenderFailure, RenderScenesBatchUseCase
from domain.models import SceneRenderRequest, SceneRenderResult

logger = logging.getLogger(__name__)


class AckableMessage(Protocol):
    """Minimal surface of aio-pika's IncomingMessage we depend on."""

    body: bytes

    async def ack(self) -> None: ...


class RenderScenesCommandHandler:
    def __init__(
        self,
        batch_use_case: RenderScenesBatchUseCase,
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

        requests = [
            SceneRenderRequest(
                project_id=project_id,
                scene_index=s["scene_index"],
                narration_text=s["narration_text"],
                illustration_hint=s.get("illustration_hint"),
                code_snippet=s.get("code_snippet"),
                code_language=s.get("code_language"),
                animation_template_id=s["animation_template_id"],
                audio_path=s["audio_path"],
                duration_seconds=s["duration_seconds"],
            )
            for s in envelope["payload"]["scenes"]
        ]

        loop = asyncio.get_running_loop()

        def on_scene_start(scene_index: int) -> None:
            out_envelope = scene_render_started_envelope(saga_id, project_id, scene_index)
            asyncio.run_coroutine_threadsafe(
                self._enqueue_immediately("scene_render_started", project_id, out_envelope), loop
            ).result()

        def on_scene_rendered(scene_index: int, result: SceneRenderResult) -> None:
            out_envelope = scene_rendered_envelope(saga_id, project_id, scene_index, result)
            asyncio.run_coroutine_threadsafe(
                self._enqueue_immediately("scene_rendered", project_id, out_envelope), loop
            ).result()

        outcome = await asyncio.to_thread(
            self._batch_use_case.execute, requests, on_scene_start, on_scene_rendered
        )

        if isinstance(outcome, BatchRenderFailure):
            logger.warning(
                "render_scenes failed for project_id=%s scene_index=%s: %s",
                project_id,
                outcome.scene_index,
                outcome.error_message,
            )
            event_type = "rendering_failed"
            final_envelope = rendering_failed_envelope(
                saga_id, project_id, outcome.scene_index, outcome.error_message
            )
        else:
            event_type = "rendering_completed"
            final_envelope = rendering_completed_envelope(saga_id, project_id, len(outcome.results))

        async with self._pool.acquire() as conn, conn.transaction():
            await self._outbox.enqueue(
                conn, aggregate_id=project_id, event_type=event_type, envelope=final_envelope
            )
            await self._inbox.mark_processed(conn, message_id)

        await message.ack()

    async def _enqueue_immediately(self, event_type: str, project_id: str, envelope: dict) -> None:
        """Writes+commits one Outbox row on its own, separate from the
        final event/Inbox transaction — so OutboxRelay can publish it
        right away instead of waiting for the whole batch to finish."""
        async with self._pool.acquire() as conn, conn.transaction():
            await self._outbox.enqueue(
                conn, aggregate_id=project_id, event_type=event_type, envelope=envelope
            )
