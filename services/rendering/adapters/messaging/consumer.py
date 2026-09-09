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
from adapters.messaging.producer import (
    rendering_completed_envelope,
    rendering_failed_envelope,
    script_validated_envelope,
    validation_failed_envelope,
)
from adapters.messaging.progress import ProgressPublisher
from adapters.persistence.inbox import InboxRepository
from adapters.persistence.outbox import OutboxRepository
from application.render_script import RenderScriptUseCase
from application.validate_script import ScriptValidationError, ValidateScriptUseCase
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
        progress: ProgressPublisher | None = None,
    ) -> None:
        self._use_case = use_case
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
        request = ScriptRenderRequest(
            project_id=project_id,
            script_content=payload["script_content"],
            scene_class_name=payload["scene_class_name"],
            narration_segments=[
                NarrationSegment(
                    scene_index=s["scene_index"],
                    audio_path=s.get("audio_path"),
                    duration_seconds=s["duration_seconds"],
                )
                for s in payload["scenes"]
            ],
            render_quality=payload.get("render_quality"),
        )

        try:
            # The renderer's heartbeat fires on the render thread, but aio-pika
            # is only safe to touch from the event loop — hence the hop back.
            loop = asyncio.get_running_loop()

            def emit_heartbeat(elapsed: float, animation_index: int | None) -> None:
                if self._progress is None:
                    return
                asyncio.run_coroutine_threadsafe(
                    self._progress.publish_render_heartbeat(project_id, elapsed, animation_index),
                    loop,
                )

            self._use_case.set_heartbeat(emit_heartbeat)
            result = await asyncio.to_thread(self._use_case.render, request)
        except (ValueError, InvalidDurationError, AnimationEngineError) as exc:
            logger.warning("render_scenes failed for project_id=%s: %s", project_id, exc)
            event_type = "rendering_failed"
            final_envelope = rendering_failed_envelope(saga_id, project_id, str(exc))
        else:
            event_type = "rendering_completed"
            final_envelope = rendering_completed_envelope(
                saga_id,
                project_id,
                result.video_path,
                result.wait_offsets,
                result.video_duration_seconds,
            )

        async with self._pool.acquire() as conn, conn.transaction():
            await self._outbox.enqueue(
                conn, aggregate_id=project_id, event_type=event_type, envelope=final_envelope
            )
            await self._inbox.mark_processed(conn, message_id)

        await message.ack()


class ValidateScriptCommandHandler:
    """Cổng kiểm tra trước TTS (CR-020 FR56).

    Lượt dry là một subprocess Manim, nên cũng phải chạy qua `to_thread` như
    lượt render thật — nếu không nó khoá event loop và bỏ đói heartbeat của
    RabbitMQ lẫn OutboxRelay.
    """

    def __init__(
        self,
        use_case: ValidateScriptUseCase,
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
            # Lượt dry không dùng tới thời lượng — nó là thứ sinh ra chúng.
            narration_segments=[],
            render_quality=payload.get("render_quality"),
        )

        try:
            result = await asyncio.to_thread(self._use_case.validate, request)
        except (ScriptValidationError, AnimationEngineError, ValueError) as exc:
            logger.warning("validate_script failed for project_id=%s: %s", project_id, exc)
            event_type = "validation_failed"
            final_envelope = validation_failed_envelope(saga_id, project_id, str(exc))
        else:
            event_type = "script_validated"
            final_envelope = script_validated_envelope(
                saga_id,
                project_id,
                result.dry_run.narrations,
                result.dry_run.beats,
                result.dry_run.chapters,
                [str(issue) for issue in result.warnings],
            )

        async with self._pool.acquire() as conn, conn.transaction():
            await self._outbox.enqueue(
                conn, aggregate_id=project_id, event_type=event_type, envelope=final_envelope
            )
            await self._inbox.mark_processed(conn, message_id)

        await message.ack()


class RenderingCommandDispatcher:
    """Một queue, hai lệnh (CR-020).

    `rendering.commands` giờ mang cả `validate_script` (lượt dry, trước TTS) lẫn
    `render_scenes` (lượt thật, sau TTS). Dùng chung một queue thay vì mở queue
    thứ hai vì cả hai đều là công việc của cùng service, cùng cần Manim, và cùng
    phải xếp hàng sau nhau — hai queue chỉ tạo ra khả năng chúng chạy song song
    và tranh nhau CPU của cùng một container.
    """

    def __init__(
        self,
        validate: ValidateScriptCommandHandler,
        render: RenderScriptCommandHandler,
    ) -> None:
        self._handlers = {
            "validate_script": validate.handle,
            "render_scenes": render.handle,
        }

    async def handle(self, message: AckableMessage) -> None:
        envelope = json.loads(message.body)
        command = envelope.get("event_type")
        handler = self._handlers.get(command)
        if handler is None:
            # Ack chứ không nack: một lệnh không hiểu được sẽ không tự hiểu được
            # ở lần thử lại, nên nack chỉ tạo vòng lặp vô tận qua DLQ.
            logger.warning("Bỏ qua lệnh không rõ %r cho rendering.commands", command)
            await message.ack()
            return
        await handler(message)
