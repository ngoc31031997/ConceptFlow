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
import subprocess
from typing import Protocol

import asyncpg

from adapters.assembly.ffmpeg_assembler import (
    AUDIO_ENCODE_ARGS,
    CONTAINER_ARGS,
    DEFAULT_ASSEMBLY_TIMEOUT_SECONDS,
    LOUDNORM_FILTER,
    VIDEO_ENCODE_ARGS,
    FfmpegVideoAssembler,
)
from adapters.logging.correlation import set_correlation_id
from adapters.messaging.producer import (
    assembly_failed_envelope,
    channel_asset_normalized_envelope,
    video_assembled_envelope,
)
from adapters.persistence.channel_assets import ChannelAssetsRepository
from adapters.persistence.inbox import InboxRepository
from adapters.persistence.outbox import OutboxRepository
from adapters.storage.artifact_paths import (
    channel_asset_with_music_path,
    ensure_parent_dir,
    normalized_channel_asset_path,
)
from application.assemble_video import AssembleVideoUseCase
from domain.errors import AssemblyEngineError, MissingArtifactError
from domain.models import (
    ChannelAsset,
    NarrationSegment,
    SubtitleCue,
    SubtitleStyle,
    VideoAssemblyRequest,
)

logger = logging.getLogger(__name__)

# Target frame size per render_quality, for NormalizeChannelAssetCommandHandler's
# transcode of a Creator-uploaded file (CR-023 D8). Mirrors the resolutions
# implied by rendering/adapters/rendering/manim_renderer.py's Manim quality
# flags (-qm/-qh/-qk), kept here as plain ffmpeg scale targets since this
# service has no Manim dependency of its own.
QUALITY_FRAME_SIZE = {
    "720p30": (1280, 720),
    "1080p60": (1920, 1080),
    "4k60": (3840, 2160),
}
DEFAULT_QUALITY_FRAME_SIZE = QUALITY_FRAME_SIZE["1080p60"]

# `asset_role` on normalize_channel_asset (CR-023 FR65.4/FR66.5): whether
# file_path is the intro/outro clip or the music bed that goes into it.
# Absent means "video" — the shape the command had before FR66.5 landed.
ASSET_ROLE_VIDEO = "video"
ASSET_ROLE_MUSIC = "music"


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
        channel_assets: ChannelAssetsRepository | None = None,
    ) -> None:
        self._use_case = use_case
        self._pool = pool
        self._inbox = inbox
        self._outbox = outbox
        # CR-023 D2: optional so every pre-existing call site/test (none of
        # which know about channel assets) keeps working unchanged — a
        # command with no intro_asset_id/outro_asset_id never touches this.
        self._channel_assets = channel_assets

    async def _resolve_channel_asset(self, asset_id: str | None) -> tuple[str | None, float]:
        """Resolves an opaque intro_asset_id/outro_asset_id to (video_path,
        duration_seconds). Best-effort, like Orchestrator's own
        resolveChannelAssets (handle_step_event.go): a missing repository, an
        unknown id, or a lookup error just means assembling without that
        asset — it never fails assemble_video (Rule: intro/outro is
        decoration, not a required artifact)."""
        if not asset_id or self._channel_assets is None:
            return None, 0.0
        try:
            asset = await self._channel_assets.get_by_id(asset_id)
        except Exception:  # noqa: BLE001 — best-effort lookup, see docstring
            logger.exception("could not resolve channel asset_id=%s, assembling without it", asset_id)
            return None, 0.0
        if asset is None:
            logger.warning("channel asset_id=%s not found, assembling without it", asset_id)
            return None, 0.0
        return asset.video_path, asset.duration_seconds

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
        # CR-023 D2: intro_asset_id/outro_asset_id are opaque ids Orchestrator
        # read from its own channel_asset_pointers projection — absent
        # (omitted, not null) when the Creator's toggle is off or no active
        # asset was found (see assembleVideoPayload in handle_step_event.go).
        intro_video_path, intro_duration_seconds = await self._resolve_channel_asset(
            payload.get("intro_asset_id")
        )
        outro_video_path, _ = await self._resolve_channel_asset(payload.get("outro_asset_id"))

        request = VideoAssemblyRequest(
            project_id=project_id,
            video_path=payload["video_path"],
            narration_segments=_parse_narration_segments(payload),
            video_duration_seconds=float(payload.get("video_duration_seconds") or 0.0),
            background_music_path=payload.get("background_music_path"),
            background_music_volume=float(payload.get("background_music_volume") or 0.2),
            subtitle_cues=_parse_subtitle_cues(payload.get("subtitle_cues")),
            subtitle_style=_parse_subtitle_style(payload.get("subtitle_style")),
            # Default matches VideoAssemblyRequest's own default: a command
            # already in the queue when CR-015 ships carries no subtitle_mode
            # at all, and must keep producing exactly what it produced before
            # (burn-in), not silently switch to a caption track.
            subtitle_mode=payload.get("subtitle_mode") or "burn_in",
            intro_video_path=intro_video_path,
            intro_duration_seconds=intro_duration_seconds,
            outro_video_path=outro_video_path,
        )

        try:
            result = await asyncio.to_thread(self._use_case.assemble, request)
        except (MissingArtifactError, AssemblyEngineError) as exc:
            logger.warning("assemble_video failed for project_id=%s: %s", project_id, exc)
            event_type = "assembly_failed"
            out_envelope = assembly_failed_envelope(saga_id, project_id, str(exc))
        else:
            event_type = "video_assembled"
            out_envelope = video_assembled_envelope(
                saga_id, project_id, result.video_path, result.caption_path
            )

        async with self._pool.acquire() as conn, conn.transaction():
            await self._outbox.enqueue(
                conn, aggregate_id=project_id, event_type=event_type, envelope=out_envelope
            )
            await self._inbox.mark_processed(conn, message_id)

        await message.ack()


class ChannelAssetRenderedEventHandler:
    """Consumes rendering's `channel_asset_rendered` event off the
    `video_assembly.channel_asset_events` queue (CR-023 correction —
    infra/rabbitmq/definitions.json binds it to events.direct/"orchestrator",
    the same routing key rendering already publishes that event under).

    Registers the rendered file in `channel_assets` under the quality
    rendering names in the event (FR65.5 — one asset per quality), then
    re-announces it as `channel_asset_normalized` so Orchestrator's
    channel_asset_pointers projection picks it up.
    """

    def __init__(
        self,
        pool: asyncpg.Pool,
        channel_assets: ChannelAssetsRepository,
        inbox: InboxRepository,
        outbox: OutboxRepository,
    ) -> None:
        self._pool = pool
        self._channel_assets = channel_assets
        self._inbox = inbox
        self._outbox = outbox

    async def handle(self, message: AckableMessage) -> None:
        envelope = json.loads(message.body)
        payload = envelope.get("payload", {})
        if payload.get("event_type") != "channel_asset_rendered":
            # channel_asset_render_failed and anything else on this queue is
            # not this service's concern (Orchestrator's own subscription
            # handles the failure event) — ack so it is not redelivered forever.
            await message.ack()
            return

        message_id = envelope["message_id"]
        saga_id = envelope["saga_id"]
        project_id = envelope["project_id"]
        set_correlation_id(saga_id)

        if await self._inbox.has_processed(message_id):
            logger.info("Skipping already-processed message_id=%s", message_id)
            await message.ack()
            return

        kind = payload["kind"]
        video_path = payload["video_path"]
        duration_seconds = float(payload.get("video_duration_seconds") or 0.0)
        # Rendering names the quality it actually rendered at (FR65.5) — an
        # intro rendered at 1080p60 cannot be concatenated onto a 4k60 body,
        # so it is registered for that one quality and no other.
        render_quality = payload["render_quality"]
        # No real source_hash travels on this event (rendering doesn't
        # compute one for its own Manim output) — this is descriptive only,
        # not used for de-duplication here. FR65.6's cache check applies to
        # the Creator-upload path (NormalizeChannelAssetCommandHandler),
        # where a real content hash is available.
        source_hash = f"rendered:{kind}:{video_path}:{duration_seconds}"

        async with self._pool.acquire() as conn, conn.transaction():
            asset = await self._channel_assets.register_new_version(
                conn,
                kind=kind,
                render_quality=render_quality,
                source_hash=source_hash,
                video_path=video_path,
                music_path=None,
                duration_seconds=duration_seconds,
            )
            out_envelope = channel_asset_normalized_envelope(
                saga_id, project_id, asset.kind, asset.id, asset.render_quality, asset.version
            )
            await self._outbox.enqueue(
                conn,
                aggregate_id=asset.id,
                event_type="channel_asset_normalized",
                envelope=out_envelope,
            )
            await self._inbox.mark_processed(conn, message_id)

        await message.ack()


class NormalizeChannelAssetCommandHandler:
    """Handles `normalize_channel_asset` (from Orchestrator's
    ChannelAssetsUseCase.Normalize, CR-023 D8) — a Creator-uploaded intro/
    outro file api-gateway already wrote to the shared volume.

    Transcodes it to the target render_quality's frame size with ffmpeg
    (reusing ffmpeg_assembler's VIDEO_ENCODE_ARGS/AUDIO_ENCODE_ARGS so an
    uploaded intro carries the same upload-grade settings as an assembled
    video), loudnorms its audio to -14 LUFS if it has any (FR66.5), and
    registers the result in `channel_assets` — skipping the transcode
    entirely when `source_hash` matches what's already active for
    (kind, render_quality) (FR65.6).
    """

    def __init__(
        self,
        pool: asyncpg.Pool,
        channel_assets: ChannelAssetsRepository,
        inbox: InboxRepository,
        outbox: OutboxRepository,
        timeout_seconds: int = DEFAULT_ASSEMBLY_TIMEOUT_SECONDS,
    ) -> None:
        self._pool = pool
        self._channel_assets = channel_assets
        self._inbox = inbox
        self._outbox = outbox
        self._timeout_seconds = timeout_seconds

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
        kind = payload["kind"]
        file_path = payload["file_path"]
        source_hash = payload["source_hash"]
        render_quality = payload["render_quality"]
        # Commands published before asset_role existed carry only video.
        asset_role = payload.get("asset_role") or ASSET_ROLE_VIDEO

        async with self._pool.acquire() as conn:
            active = await self._channel_assets.get_active(conn, kind, render_quality)

        if asset_role == ASSET_ROLE_MUSIC:
            await self._handle_music(
                message,
                message_id=message_id,
                saga_id=saga_id,
                project_id=project_id,
                kind=kind,
                render_quality=render_quality,
                music_path=file_path,
                music_source_hash=source_hash,
                active=active,
            )
            return

        if active is not None and active.source_hash == source_hash:
            # FR65.6: source unchanged since the active version — do not
            # re-transcode, do not publish a new channel_asset_normalized.
            logger.info(
                "normalize_channel_asset: %s/%s already up to date (source_hash match), skipping",
                kind,
                render_quality,
            )
            async with self._pool.acquire() as conn, conn.transaction():
                await self._inbox.mark_processed(conn, message_id)
            await message.ack()
            return

        output_path = normalized_channel_asset_path(kind, render_quality)
        try:
            duration_seconds = await asyncio.to_thread(
                self._transcode, file_path, output_path, render_quality
            )
        except AssemblyEngineError as exc:
            logger.warning("normalize_channel_asset failed for kind=%s: %s", kind, exc)
            # No failure event exists for this command (D8 leaves it to the
            # Creator to retry the upload) — mark processed so a redelivery
            # of the same bad file does not retry forever.
            async with self._pool.acquire() as conn, conn.transaction():
                await self._inbox.mark_processed(conn, message_id)
            await message.ack()
            return

        async with self._pool.acquire() as conn, conn.transaction():
            asset = await self._channel_assets.register_new_version(
                conn,
                kind=kind,
                render_quality=render_quality,
                source_hash=source_hash,
                video_path=output_path,
                music_path=None,
                duration_seconds=duration_seconds,
            )
            out_envelope = channel_asset_normalized_envelope(
                saga_id, project_id, asset.kind, asset.id, asset.render_quality, asset.version
            )
            await self._outbox.enqueue(
                conn, aggregate_id=asset.id, event_type="channel_asset_normalized", envelope=out_envelope
            )
            await self._inbox.mark_processed(conn, message_id)

        await message.ack()

    async def _handle_music(
        self,
        message: AckableMessage,
        *,
        message_id: str,
        saga_id: str,
        project_id: str,
        kind: str,
        render_quality: str,
        music_path: str,
        music_source_hash: str,
        active: ChannelAsset | None,
    ) -> None:
        """FR66.5 / CR-023 D5 — the music bed is normalised to -14 LUFS and
        muxed into the sting clip HERE, at asset-build time, so assembly of a
        project only ever concatenates a clip that already carries its audio
        (and FR66.6 holds for free: the body's background music is a separate
        stream over a separate segment)."""
        if active is None:
            # Nothing to mux the music into yet. This is not an error: the
            # Creator can upload the music before the clip, and video-assembly
            # has no way to hold the file until one arrives. The music stays on
            # the shared volume; re-uploading it once the clip is registered
            # builds the asset. Nothing is registered, so no version is burnt.
            logger.warning(
                "normalize_channel_asset(music): no active %s/%s video asset to mux %s into — "
                "upload the %s clip first, then re-upload the music",
                kind,
                render_quality,
                music_path,
                kind,
            )
            async with self._pool.acquire() as conn, conn.transaction():
                await self._inbox.mark_processed(conn, message_id)
            await message.ack()
            return

        if active.music_source_hash == music_source_hash:
            # FR65.6, music half: same music file already baked into the
            # active version — compared against music_source_hash, never
            # against source_hash (which describes the video source).
            logger.info(
                "normalize_channel_asset(music): %s/%s already carries this music, skipping",
                kind,
                render_quality,
            )
            async with self._pool.acquire() as conn, conn.transaction():
                await self._inbox.mark_processed(conn, message_id)
            await message.ack()
            return

        output_path = channel_asset_with_music_path(kind, render_quality, active.version + 1)
        try:
            await asyncio.to_thread(
                self._mux_music, active.video_path, music_path, output_path, active.duration_seconds
            )
        except AssemblyEngineError as exc:
            logger.warning("normalize_channel_asset(music) failed for kind=%s: %s", kind, exc)
            async with self._pool.acquire() as conn, conn.transaction():
                await self._inbox.mark_processed(conn, message_id)
            await message.ack()
            return

        async with self._pool.acquire() as conn, conn.transaction():
            asset = await self._channel_assets.register_new_version(
                conn,
                kind=kind,
                render_quality=render_quality,
                # The video source is unchanged, so its hash is carried over
                # verbatim — otherwise the next video upload would compare its
                # hash against a music hash and rebuild needlessly.
                source_hash=active.source_hash,
                video_path=output_path,
                music_path=music_path,
                duration_seconds=active.duration_seconds,
                music_source_hash=music_source_hash,
            )
            out_envelope = channel_asset_normalized_envelope(
                saga_id, project_id, asset.kind, asset.id, asset.render_quality, asset.version
            )
            await self._outbox.enqueue(
                conn, aggregate_id=asset.id, event_type="channel_asset_normalized", envelope=out_envelope
            )
            await self._inbox.mark_processed(conn, message_id)

        await message.ack()

    def _mux_music(
        self, video_path: str, music_path: str, output_path: str, duration_seconds: float
    ) -> None:
        """Loudnorms the music to -14 LUFS and lays it over the clip's video.

        The music REPLACES whatever audio the clip carried (`-map 0:v:0 -map
        1:a:0`) rather than being mixed with it: the mux input is the currently
        active asset, which on a second music upload is itself a muxed file, so
        mixing would stack the old bed under the new one. Replacing keeps the
        operation idempotent — the result depends only on (clip, music), not on
        upload history.

        `-t duration_seconds` (the video's own ffprobe'd length) rather than
        `-shortest`: a music bed longer than the sting must be cut to it, but a
        SHORTER one must not truncate the video, which is exactly what
        `-shortest` would do.
        """
        ensure_parent_dir(output_path)
        cmd = ["-y", "-i", video_path, "-i", music_path]
        cmd += ["-map", "0:v:0", "-map", "1:a:0"]
        cmd += ["-c:v", "copy"]  # video already transcoded to the target quality
        cmd += ["-af", LOUDNORM_FILTER]
        cmd += AUDIO_ENCODE_ARGS
        cmd += ["-t", f"{duration_seconds}"]
        cmd += CONTAINER_ARGS
        cmd += [output_path]
        FfmpegVideoAssembler._run_ffmpeg(cmd)

    def _transcode(self, file_path: str, output_path: str, render_quality: str) -> float:
        """Runs synchronously in a thread (mirrors AssembleVideoCommandHandler
        running the assembler off the event loop) — returns the transcoded
        file's duration in seconds."""
        width, height = QUALITY_FRAME_SIZE.get(render_quality, DEFAULT_QUALITY_FRAME_SIZE)
        ensure_parent_dir(output_path)
        has_audio = _probe_has_audio(file_path)

        cmd = ["-y", "-i", file_path]
        cmd += [
            "-vf",
            f"scale={width}:{height}:force_original_aspect_ratio=decrease,"
            f"pad={width}:{height}:(ow-iw)/2:(oh-ih)/2,setsar=1",
        ]
        cmd += VIDEO_ENCODE_ARGS
        if has_audio:
            cmd += ["-af", LOUDNORM_FILTER]
            cmd += AUDIO_ENCODE_ARGS
        else:
            cmd += ["-an"]
        cmd += CONTAINER_ARGS
        cmd += [output_path]

        FfmpegVideoAssembler._run_ffmpeg(cmd)
        return _probe_duration(output_path)


def _probe_has_audio(video_path: str) -> bool:
    """Whether video_path carries an audio stream at all — decides whether
    NormalizeChannelAssetCommandHandler applies loudnorm or `-an` (an input
    ffmpeg was never told to expect an audio track raises rather than
    silently producing one)."""
    result = subprocess.run(
        [
            "ffprobe", "-v", "error",
            "-select_streams", "a",
            "-show_entries", "stream=index",
            "-of", "csv=p=0",
            video_path,
        ],
        capture_output=True,
        text=True,
    )
    return result.returncode == 0 and bool(result.stdout.strip())


def _probe_duration(video_path: str) -> float:
    """Length of the just-transcoded file, so channel_assets.duration_seconds
    is real ffprobe'd data (same convention as FfmpegVideoAssembler's own
    _probe_audio_duration) rather than something the caller has to guess."""
    result = subprocess.run(
        [
            "ffprobe", "-v", "error",
            "-show_entries", "format=duration",
            "-of", "csv=p=0",
            video_path,
        ],
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        logger.warning("ffprobe could not read %s: %s", video_path, result.stderr.strip())
        return 0.0
    try:
        return float(result.stdout.strip())
    except ValueError:
        logger.warning("ffprobe returned an unreadable duration for %s", video_path)
        return 0.0


class VideoAssemblyCommandDispatcher:
    """One queue, two commands (CR-023, mirrors RenderingCommandDispatcher on
    the Rendering side): `video_assembly.commands` carries `assemble_video`
    (per-project, saga-driven) and now `normalize_channel_asset` (channel-
    wide, admin/upload-triggered) — one queue because both are this
    service's own work and both need to queue behind each other rather than
    run concurrently and contend for the same ffmpeg thread pool.
    """

    def __init__(
        self,
        assemble_video: AssembleVideoCommandHandler,
        normalize_channel_asset: NormalizeChannelAssetCommandHandler | None = None,
    ) -> None:
        self._handlers = {"assemble_video": assemble_video.handle}
        if normalize_channel_asset is not None:
            self._handlers["normalize_channel_asset"] = normalize_channel_asset.handle

    async def handle(self, message: AckableMessage) -> None:
        envelope = json.loads(message.body)
        # The dispatch key is the envelope's own top-level event_type
        # (domain.Envelope's `EventType` field on the orchestrator side —
        # ports.go — set to string(domain.StepAssembleVideo) == "assemble_video"
        # for that command, and "normalize_channel_asset" for the other one;
        # NOT payload["event_type"], which channel_assets.go happens to also
        # set redundantly but assembleVideoPayload never does).
        command = envelope.get("event_type")
        # A command already sitting in the queue when CR-023 ships may carry
        # no event_type at all (pre-CR-023 assemble_video shape) — default to
        # assemble_video so nothing already in flight starts being ignored.
        handler = self._handlers.get(command) or self._handlers.get("assemble_video")
        if handler is None:
            logger.warning("Unrecognized command %r on video_assembly.commands, ignoring", command)
            await message.ack()
            return
        await handler(message)
