"""AMQP command consumer — handles synthesize_speech from tts.commands
(interface-contracts.md, ADR-0014).

Idempotency and event publishing go through the Inbox/Outbox pattern
(adapters/persistence/, see ADR-0013): the consumer never publishes to
RabbitMQ directly, it only
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
from adapters.messaging.cancellation import REGISTRY
from adapters.messaging.producer import (
    failure_envelope,
    lines_envelope,
    lines_failure_envelope,
    success_envelope,
)
from adapters.messaging.progress import ProgressPublisher
from adapters.persistence.inbox import InboxRepository
from adapters.persistence.outbox import OutboxRepository
from application.synthesize_lines import LineSpeechRequest, SynthesizeLinesUseCase
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
        """Gate on cancellation, then process. A command the Creator cancelled
        while it waited in the queue is acked and never run; one that is running
        sees the cancel through REGISTRY (checked between scenes) and its events
        are dropped by CancelAwareOutbox."""
        try:
            envelope = json.loads(message.body)
            project_id, command_ts = envelope["project_id"], envelope.get("timestamp")
        except (ValueError, TypeError, KeyError):
            await self._handle(message)  # let the normal path report the malformed body
            return
        if REGISTRY.is_cancelled(project_id, command_ts):
            logger.info("Bỏ lệnh synthesize_speech của project_id=%s vì đã bị huỷ", project_id)
            await message.ack()
            return
        with REGISTRY.command(project_id, command_ts):
            await self._handle(message)

    async def _handle(self, message: AckableMessage) -> None:
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

        # The batch runs in a worker thread: synthesis blocks for minutes, and on
        # the event loop that would also stop it from hearing a cancel.
        loop = asyncio.get_running_loop()

        def on_scene_done(scene_index: int, scene_total: int) -> None:
            if self._progress is None:
                return
            # From the worker thread back onto the loop; fire-and-forget so
            # progress never slows the real work.
            asyncio.run_coroutine_threadsafe(
                self._progress.publish_scene_progress(project_id, scene_index, scene_total), loop
            )

        outcome = await asyncio.to_thread(
            self._batch_use_case.execute,
            project_id,
            scenes,
            on_scene_done=on_scene_done,
            should_stop=REGISTRY.current_cancelled,
        )

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


class SynthesizeLinesCommandHandler:
    """Wires: inbox check -> read each line -> outbox enqueue (+ inbox mark) -> ack.

    Reads script lines outside a render saga (ADR-0040). The answer is always a
    lines_synthesized event with one entry per line; only an unexpected failure
    of the whole command becomes lines_synthesis_failed.
    """

    def __init__(
        self,
        use_case: SynthesizeLinesUseCase,
        pool: asyncpg.Pool,
        inbox: InboxRepository,
        outbox: OutboxRepository,
    ) -> None:
        self._use_case = use_case
        self._pool = pool
        self._inbox = inbox
        self._outbox = outbox

    async def handle(self, message: AckableMessage) -> None:
        try:
            envelope = json.loads(message.body)
            project_id, command_ts = envelope["project_id"], envelope.get("timestamp")
        except (ValueError, TypeError, KeyError):
            await self._handle(message)
            return
        if REGISTRY.is_cancelled(project_id, command_ts):
            logger.info("Bỏ lệnh synthesize_lines của project_id=%s vì đã bị huỷ", project_id)
            await message.ack()
            return
        with REGISTRY.command(project_id, command_ts):
            await self._handle(message)

    async def _handle(self, message: AckableMessage) -> None:
        envelope = json.loads(message.body)
        message_id = envelope["message_id"]
        saga_id = envelope["saga_id"]
        project_id = envelope["project_id"]
        set_correlation_id(saga_id)

        if await self._inbox.has_processed(message_id):
            logger.info("Skipping already-processed message_id=%s", message_id)
            await message.ack()
            return

        lines = [
            LineSpeechRequest(
                line_id=item["line_id"],
                text=item["text"],
                language=item["language"],
                voice_id=item.get("voice_id") or None,
                ref=item.get("ref", ""),
            )
            for item in envelope["payload"]["lines"]
        ]
        try:
            results = await asyncio.to_thread(
                self._use_case.execute, project_id, lines, should_stop=REGISTRY.current_cancelled
            )
            event_type = "lines_synthesized"
            out_envelope = lines_envelope(saga_id, project_id, results)
        except Exception as exc:  # noqa: BLE001 — reported to the caller, not swallowed
            logger.exception("synthesize_lines failed for project_id=%s", project_id)
            event_type = "lines_synthesis_failed"
            out_envelope = lines_failure_envelope(saga_id, project_id, f"{type(exc).__name__}: {exc}")

        async with self._pool.acquire() as conn, conn.transaction():
            await self._outbox.enqueue(
                conn, aggregate_id=project_id, event_type=event_type, envelope=out_envelope
            )
            await self._inbox.mark_processed(conn, message_id)

        await message.ack()


class TtsCommandDispatcher:
    """`tts.commands` carries synthesize_speech, synthesize_lines and
    purge_project_artifacts. Anything without an event_type is a
    synthesize_speech command."""

    def __init__(
        self,
        synthesize: SynthesizeSpeechCommandHandler,
        purge=None,
        lines: SynthesizeLinesCommandHandler | None = None,
    ) -> None:
        self._synthesize = synthesize
        self._purge = purge
        self._lines = lines

    async def handle(self, message: AckableMessage) -> None:
        try:
            command = json.loads(message.body).get("event_type")
        except (ValueError, TypeError, AttributeError):
            command = None
        if command == "purge_project_artifacts" and self._purge is not None:
            await self._purge.handle(message)
            return
        if command == "synthesize_lines" and self._lines is not None:
            await self._lines.handle(message)
            return
        await self._synthesize.handle(message)
