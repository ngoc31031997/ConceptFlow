"""Progress publisher for assemble_video/generate_clips (CR-029).

Mirrors services/rendering/adapters/messaging/progress.py and
services/tts/adapters/messaging/progress.py: both steps here run one or more
blocking ffmpeg subprocesses in a worker thread (asyncio.to_thread), long
enough that silence between the "assembling_video"/"generating_clips" status
and the final event reads as a hang.

Published to `progress.fanout` (ADR-0017), fire-and-forget and deliberately
NOT routed through the Outbox — progress is UX-only, never part of the
durable, exactly-once state Outbox/Inbox exist to protect
(messaging-design.md).

publish_stage_progress/publish_clip_progress are called from a worker thread
(the assembler/clip loop runs via asyncio.to_thread), so they schedule the
actual publish onto the event loop with run_coroutine_threadsafe rather than
awaiting directly — there is no running loop in that thread to await on.
"""

from __future__ import annotations

import asyncio
import json
import logging

import aio_pika

logger = logging.getLogger(__name__)

PROGRESS_EXCHANGE = "progress.fanout"
ASSEMBLE_VIDEO_STEP = "assemble_video"
GENERATE_CLIPS_STEP = "generate_clips"


class ProgressPublisher:
    def __init__(self, exchange: aio_pika.abc.AbstractExchange, loop: asyncio.AbstractEventLoop) -> None:
        self._exchange = exchange
        self._loop = loop

    def publish_stage_progress(self, project_id: str, stage_index: int, stage_total: int) -> None:
        """One event per completed ffmpeg pass (main mux, then intro/outro
        concat when the project has one) — CR-029: by unit of work done, not
        by tick, so at most 2 pings per assembly."""
        self._schedule(
            {
                "project_id": project_id,
                "step": ASSEMBLE_VIDEO_STEP,
                "status": "in_progress",
                "stage_index": stage_index,
                "stage_total": stage_total,
            }
        )

    def publish_clip_progress(self, project_id: str, clip_index: int, clip_total: int) -> None:
        """One event per finished (request, preset) clip cut."""
        self._schedule(
            {
                "project_id": project_id,
                "step": GENERATE_CLIPS_STEP,
                "status": "in_progress",
                "clip_index": clip_index,
                "clip_total": clip_total,
            }
        )

    def _schedule(self, message: dict) -> None:
        async def _publish() -> None:
            try:
                await self._exchange.publish(
                    aio_pika.Message(json.dumps(message).encode()), routing_key=""
                )
            except Exception:  # noqa: BLE001 — progress is UX-only, never fail real work
                logger.exception("Could not publish progress for %s", message.get("project_id"))

        try:
            asyncio.run_coroutine_threadsafe(_publish(), self._loop)
        except Exception:  # noqa: BLE001 — same posture: never let progress break the caller
            logger.exception("Could not schedule progress publish for %s", message.get("project_id"))
