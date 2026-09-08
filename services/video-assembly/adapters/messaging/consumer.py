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
from domain.models import NarrationSegment, SubtitleCue, SubtitleStyle, VideoAssemblyRequest

logger = logging.getLogger(__name__)


def _parse_narration_segments(payload: dict) -> list[NarrationSegment]:
    """Reads CR-002's `narration_segments`, falling back to the pre-CR-002
    `audio_segments` shape.

    The fallback exists for commands already sitting in the queue when this
    ships. It reproduces the old, wrong end-to-end placement, so it is logged:
    such a project needs re-rendering to get correct timing, and silently
    producing a desynchronised video is exactly what CR-002 set out to stop.
    """
    raw = payload.get("narration_segments")
    if raw is not None:
        return [
            NarrationSegment(
                audio_path=item["audio_path"], start_time=float(item["start_time"])
            )
            for item in raw
        ]

    legacy = payload.get("audio_segments") or []
    if legacy:
        logger.warning(
            "assemble_video command carries the pre-CR-002 audio_segments shape; "
            "narration will be laid end to end and WILL drift out of sync with "
            "the animation — re-render this project to fix it"
        )
    return [NarrationSegment(audio_path=path, start_time=0.0) for path in legacy]


def _parse_subtitle_cues(raw: list[dict] | None) -> list[SubtitleCue] | None:
    """Absent when the Creator disabled subtitles (CR-001 FR9.1)."""
    if not raw:
        return None
    return [
        SubtitleCue(
            scene_index=cue["scene_index"],
            text=cue["text"],
            start_time=cue["start_time"],
            end_time=cue["end_time"],
        )
        for cue in raw
    ]


def _parse_subtitle_style(raw: dict | None) -> SubtitleStyle | None:
    if not raw:
        return None
    return SubtitleStyle(
        font_size=raw.get("font_size", "medium"),
        text_color=raw.get("text_color", "#FFFFFF"),
        background_opacity=raw.get("background_opacity", 0.6),
        position=raw.get("position", "bottom"),
    )


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
            narration_segments=_parse_narration_segments(payload),
            video_duration_seconds=float(payload.get("video_duration_seconds") or 0.0),
            background_music_path=payload.get("background_music_path"),
            background_music_volume=float(payload.get("background_music_volume") or 0.2),
            subtitle_cues=_parse_subtitle_cues(payload.get("subtitle_cues")),
            subtitle_style=_parse_subtitle_style(payload.get("subtitle_style")),
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
