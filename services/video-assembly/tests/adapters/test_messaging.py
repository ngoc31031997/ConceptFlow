"""Unit tests for the messaging adapter: consumer + Inbox/Outbox (ADR-0013).

Uses fakes for the AMQP surface (AckableMessage) and a FakePool standing
in for asyncpg (tests/adapters/fake_postgres.py), plus a FakeVideoAssembler
standing in for ffmpeg — no real RabbitMQ/PostgreSQL/ffmpeg needed.
"""

from __future__ import annotations

import json
import os
from unittest.mock import patch

import pytest

from adapters.messaging.consumer import (
    AssembleVideoCommandHandler,
    ChannelAssetRenderedEventHandler,
    NormalizeChannelAssetCommandHandler,
)
from adapters.persistence.channel_assets import ChannelAssetsRepository
from adapters.persistence.inbox import InboxRepository
from adapters.persistence.outbox import OutboxRepository
from application.assemble_video import AssembleVideoUseCase
from domain.errors import AssemblyEngineError
from domain.models import VideoAssemblyRequest
from domain.ports import VideoAssemblerPort
from tests.adapters.fake_postgres import FakePool


def _touch(path: str) -> None:
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w") as f:
        f.write("data")


class FakeVideoAssembler(VideoAssemblerPort):
    def __init__(self, fail_with: Exception | None = None, caption_path: str | None = None) -> None:
        self._fail_with = fail_with
        self._caption_path = caption_path

    def assemble(self, request: VideoAssemblyRequest, output_path: str) -> str | None:
        if self._fail_with is not None:
            raise self._fail_with
        _touch(output_path)
        if self._caption_path:
            _touch(self._caption_path)
        return self._caption_path


class FakeMessage:
    def __init__(self, body: bytes) -> None:
        self.body = body
        self.acked = False

    async def ack(self) -> None:
        self.acked = True


def make_envelope(message_id: str = "msg-1", shared_volume_root=None) -> bytes:
    video_path = str(shared_volume_root / "rendered.mp4")
    audio_path = str(shared_volume_root / "audio0.wav")
    _touch(video_path)
    _touch(audio_path)
    envelope = {
        "message_id": message_id,
        "saga_id": "saga-1",
        "project_id": "project-1",
        "schema_version": "1.0",
        "timestamp": "2026-08-24T00:00:00Z",
        "payload": {"video_path": video_path, "audio_segments": [audio_path]},
    }
    return json.dumps(envelope).encode("utf-8")


@pytest.fixture
def shared_volume_root(tmp_path, monkeypatch):
    monkeypatch.setattr("adapters.storage.artifact_paths.SHARED_VOLUME_ROOT", str(tmp_path))
    return tmp_path


def _build_handler(assembler: VideoAssemblerPort) -> tuple[AssembleVideoCommandHandler, FakePool]:
    use_case = AssembleVideoUseCase(assembler)
    pool = FakePool()
    inbox = InboxRepository(pool)
    outbox = OutboxRepository()
    handler = AssembleVideoCommandHandler(use_case, pool, inbox, outbox)
    return handler, pool


@pytest.mark.asyncio
async def test_enqueues_success_event_to_outbox_and_acks(shared_volume_root) -> None:
    handler, pool = _build_handler(FakeVideoAssembler())
    message = FakeMessage(make_envelope(shared_volume_root=shared_volume_root))

    await handler.handle(message)

    assert message.acked is True
    assert len(pool.store.outbox_events) == 1
    event = next(iter(pool.store.outbox_events.values()))
    assert event["event_type"] == "video_assembled"
    assert "final.mp4" in event["payload"]["payload"]["video_path"]


@pytest.mark.asyncio
async def test_caption_path_is_carried_into_the_video_assembled_event(shared_volume_root) -> None:
    """CR-015 FR38.4: caption_path travels through the outbox event the same
    way video_path does, so the Orchestrator can pick it up downstream."""
    caption_path = str(shared_volume_root / "project-1" / "video" / "final.srt")
    handler, pool = _build_handler(FakeVideoAssembler(caption_path=caption_path))
    message = FakeMessage(make_envelope(shared_volume_root=shared_volume_root))

    await handler.handle(message)

    event = next(iter(pool.store.outbox_events.values()))
    assert event["payload"]["payload"]["caption_path"] == caption_path


@pytest.mark.asyncio
async def test_no_caption_path_key_when_assembler_produced_none(shared_volume_root) -> None:
    """Absent rather than null (mirrors thumbnail_path already flowing this
    way) — a downstream reader distinguishing "no caption" from a bug that
    forgot to set the field should not have to treat null as valid data."""
    handler, pool = _build_handler(FakeVideoAssembler())
    message = FakeMessage(make_envelope(shared_volume_root=shared_volume_root))

    await handler.handle(message)

    event = next(iter(pool.store.outbox_events.values()))
    assert "caption_path" not in event["payload"]["payload"]


@pytest.mark.asyncio
async def test_enqueues_failure_event_on_engine_error(shared_volume_root) -> None:
    handler, pool = _build_handler(FakeVideoAssembler(fail_with=AssemblyEngineError("boom")))
    message = FakeMessage(make_envelope(shared_volume_root=shared_volume_root))

    await handler.handle(message)

    assert message.acked is True
    event = next(iter(pool.store.outbox_events.values()))
    assert event["event_type"] == "assembly_failed"
    assert "boom" in event["payload"]["payload"]["error_message"]


@pytest.mark.asyncio
async def test_marks_message_processed_in_inbox(shared_volume_root) -> None:
    handler, pool = _build_handler(FakeVideoAssembler())
    message = FakeMessage(make_envelope(message_id="msg-1", shared_volume_root=shared_volume_root))

    await handler.handle(message)

    assert "msg-1" in pool.store.processed_message_ids


class RecordingVideoAssembler(VideoAssemblerPort):
    """Captures the request it was called with, for asserting on how the
    consumer parsed the envelope payload."""

    def __init__(self) -> None:
        self.last_request: VideoAssemblyRequest | None = None

    def assemble(self, request: VideoAssemblyRequest, output_path: str) -> str | None:
        self.last_request = request
        _touch(output_path)
        return None


@pytest.mark.asyncio
async def test_missing_subtitle_mode_in_payload_defaults_to_burn_in(shared_volume_root) -> None:
    """A command already sitting in the queue when CR-015 ships carries no
    subtitle_mode key at all — it must keep producing exactly what it
    produced before (burn-in), not silently switch to a caption track."""
    assembler = RecordingVideoAssembler()
    handler, _ = _build_handler(assembler)
    message = FakeMessage(make_envelope(shared_volume_root=shared_volume_root))

    await handler.handle(message)

    assert assembler.last_request.subtitle_mode == "burn_in"


@pytest.mark.asyncio
async def test_subtitle_mode_in_payload_is_passed_through(shared_volume_root) -> None:
    assembler = RecordingVideoAssembler()
    handler, _ = _build_handler(assembler)
    envelope = json.loads(make_envelope(shared_volume_root=shared_volume_root))
    envelope["payload"]["subtitle_mode"] = "track"
    message = FakeMessage(json.dumps(envelope).encode("utf-8"))

    await handler.handle(message)

    assert assembler.last_request.subtitle_mode == "track"


@pytest.mark.asyncio
async def test_assemble_video_resolves_intro_asset_id_into_the_request(shared_volume_root) -> None:
    """CR-023 D2: intro_asset_id on the command is opaque — this handler must
    resolve it via ChannelAssetsRepository into the real video_path/duration
    the assembler needs."""
    pool = FakePool()
    inbox = InboxRepository(pool)
    outbox = OutboxRepository()
    channel_assets = ChannelAssetsRepository(pool)
    async with pool.acquire() as conn, conn.transaction():
        asset = await channel_assets.register_new_version(
            conn,
            kind="intro",
            render_quality="1080p60",
            source_hash="hash-1",
            video_path="/shared/channel-assets/intro/1080p60/normalized.mp4",
            music_path=None,
            duration_seconds=3.0,
        )
    assembler = RecordingVideoAssembler()
    use_case = AssembleVideoUseCase(assembler)
    handler = AssembleVideoCommandHandler(use_case, pool, inbox, outbox, channel_assets)

    envelope = json.loads(make_envelope(shared_volume_root=shared_volume_root))
    envelope["payload"]["intro_asset_id"] = asset.id
    message = FakeMessage(json.dumps(envelope).encode("utf-8"))

    await handler.handle(message)

    assert assembler.last_request.intro_video_path == "/shared/channel-assets/intro/1080p60/normalized.mp4"
    assert assembler.last_request.intro_duration_seconds == 3.0


@pytest.mark.asyncio
async def test_assemble_video_proceeds_without_intro_when_asset_id_unknown(shared_volume_root) -> None:
    """Best-effort like Orchestrator's own resolveChannelAssets: an id that
    does not resolve must not fail assemble_video."""
    pool = FakePool()
    inbox = InboxRepository(pool)
    outbox = OutboxRepository()
    channel_assets = ChannelAssetsRepository(pool)
    assembler = RecordingVideoAssembler()
    use_case = AssembleVideoUseCase(assembler)
    handler = AssembleVideoCommandHandler(use_case, pool, inbox, outbox, channel_assets)

    envelope = json.loads(make_envelope(shared_volume_root=shared_volume_root))
    envelope["payload"]["intro_asset_id"] = "does-not-exist"
    message = FakeMessage(json.dumps(envelope).encode("utf-8"))

    await handler.handle(message)

    assert message.acked is True
    assert assembler.last_request.intro_video_path is None
    event = next(iter(pool.store.outbox_events.values()))
    assert event["event_type"] == "video_assembled"


@pytest.mark.asyncio
async def test_skips_reprocessing_duplicate_message_id(shared_volume_root) -> None:
    handler, pool = _build_handler(FakeVideoAssembler())
    pool.store.processed_message_ids.add("msg-1")
    message = FakeMessage(make_envelope(message_id="msg-1", shared_volume_root=shared_volume_root))

    await handler.handle(message)

    assert message.acked is True
    assert len(pool.store.outbox_events) == 0


# --- CR-023: channel_asset_rendered / normalize_channel_asset -------------


class FakeProbeCompletedProcess:
    def __init__(self, returncode: int = 0, stdout: str = "", stderr: str = "") -> None:
        self.returncode = returncode
        self.stdout = stdout
        self.stderr = stderr


def fake_probe_run(cmd, *_args, **_kwargs):
    """subprocess.run stand-in for NormalizeChannelAssetCommandHandler's
    ffprobe (has-audio / duration) and ffmpeg (transcode) calls — no real
    binaries needed."""
    if cmd and cmd[0] == "ffprobe":
        if "stream=index" in cmd:
            return FakeProbeCompletedProcess(stdout="")  # no audio stream
        return FakeProbeCompletedProcess(stdout="4.0\n")  # duration
    return FakeProbeCompletedProcess()  # ffmpeg transcode "succeeds"


def make_normalize_envelope(
    message_id: str = "msg-n1",
    kind: str = "intro",
    file_path: str = "upload.mp4",
    source_hash: str = "hash-1",
    render_quality: str = "1080p60",
) -> bytes:
    envelope = {
        "message_id": message_id,
        "saga_id": "saga-1",
        "project_id": "channel-asset-upload",
        "event_type": "normalize_channel_asset",
        "schema_version": "1.0",
        "timestamp": "2026-09-10T00:00:00Z",
        "payload": {
            "event_type": "normalize_channel_asset",
            "kind": kind,
            "file_path": file_path,
            "source_hash": source_hash,
            "render_quality": render_quality,
        },
    }
    return json.dumps(envelope).encode("utf-8")


def _build_normalize_handler() -> tuple[NormalizeChannelAssetCommandHandler, FakePool]:
    pool = FakePool()
    inbox = InboxRepository(pool)
    outbox = OutboxRepository()
    channel_assets = ChannelAssetsRepository(pool)
    handler = NormalizeChannelAssetCommandHandler(pool, channel_assets, inbox, outbox)
    return handler, pool


@pytest.mark.asyncio
async def test_normalize_channel_asset_registers_and_publishes_normalized_event(
    shared_volume_root,
) -> None:
    handler, pool = _build_normalize_handler()

    with patch("subprocess.run", side_effect=fake_probe_run):
        await handler.handle(FakeMessage(make_normalize_envelope()))

    assert len(pool.store.channel_assets) == 1
    assert len(pool.store.outbox_events) == 1
    event = next(iter(pool.store.outbox_events.values()))
    assert event["event_type"] == "channel_asset_normalized"
    inner_payload = event["payload"]["payload"]
    assert inner_payload["kind"] == "intro"
    assert inner_payload["render_quality"] == "1080p60"
    assert inner_payload["version"] == 1
    assert "asset_id" in inner_payload


@pytest.mark.asyncio
async def test_normalize_channel_asset_skips_rebuild_on_matching_source_hash(
    shared_volume_root,
) -> None:
    """FR65.6: building twice with the same source_hash for the same
    (kind, render_quality) must not re-transcode or publish a second
    channel_asset_normalized — the second call is a no-op."""
    handler, pool = _build_normalize_handler()

    with patch("subprocess.run", side_effect=fake_probe_run) as mock_run:
        await handler.handle(
            FakeMessage(make_normalize_envelope(message_id="msg-n1", source_hash="same-hash"))
        )
        calls_after_first = mock_run.call_count
        await handler.handle(
            FakeMessage(make_normalize_envelope(message_id="msg-n2", source_hash="same-hash"))
        )

    assert mock_run.call_count == calls_after_first  # no new ffprobe/ffmpeg calls
    assert len(pool.store.channel_assets) == 1  # no new version registered
    assert len(pool.store.outbox_events) == 1  # only the first normalize published
    assert "msg-n2" in pool.store.processed_message_ids  # still acked/marked processed


@pytest.mark.asyncio
async def test_normalize_channel_asset_rebuilds_on_different_source_hash(
    shared_volume_root,
) -> None:
    handler, pool = _build_normalize_handler()

    with patch("subprocess.run", side_effect=fake_probe_run):
        await handler.handle(
            FakeMessage(make_normalize_envelope(message_id="msg-n1", source_hash="hash-a"))
        )
        await handler.handle(
            FakeMessage(make_normalize_envelope(message_id="msg-n2", source_hash="hash-b"))
        )

    assert len(pool.store.channel_assets) == 2
    assert len(pool.store.outbox_events) == 2
    active_rows = [row for row in pool.store.channel_assets.values() if row["superseded_at"] is None]
    assert len(active_rows) == 1
    assert active_rows[0]["source_hash"] == "hash-b"
    assert active_rows[0]["version"] == 2


def make_normalize_music_envelope(
    message_id: str = "msg-m1",
    kind: str = "intro",
    file_path: str = "/shared/channel-assets/intro/music.mp3",
    source_hash: str = "music-hash-1",
    render_quality: str = "1080p60",
) -> bytes:
    """Same command, asset_role="music" (FR66.5) — file_path is the bed, not
    a clip."""
    envelope = json.loads(
        make_normalize_envelope(
            message_id=message_id,
            kind=kind,
            file_path=file_path,
            source_hash=source_hash,
            render_quality=render_quality,
        )
    )
    envelope["payload"]["asset_role"] = "music"
    return json.dumps(envelope).encode("utf-8")


@pytest.mark.asyncio
async def test_normalize_music_without_an_active_video_asset_registers_nothing() -> None:
    """Nothing to mux into yet — the command is acked and marked processed,
    but no version is registered and no event published."""
    handler, pool = _build_normalize_handler()

    with patch("subprocess.run", side_effect=fake_probe_run) as mock_run:
        message = FakeMessage(make_normalize_music_envelope())
        await handler.handle(message)

    assert message.acked is True
    assert pool.store.channel_assets == {}
    assert pool.store.outbox_events == {}
    assert mock_run.call_count == 0  # no ffmpeg run at all
    assert "msg-m1" in pool.store.processed_message_ids


@pytest.mark.asyncio
async def test_normalize_music_muxes_into_the_active_video_asset(shared_volume_root) -> None:
    """FR66.5 / D5: the bed is baked into the clip at build time, producing a
    new version whose music_path is the uploaded file and whose duration is
    the clip's (the music must never lengthen the sting)."""
    handler, pool = _build_normalize_handler()

    with patch("subprocess.run", side_effect=fake_probe_run):
        await handler.handle(FakeMessage(make_normalize_envelope(message_id="msg-n1")))
        await handler.handle(FakeMessage(make_normalize_music_envelope(message_id="msg-m1")))

    assert len(pool.store.channel_assets) == 2
    active = [row for row in pool.store.channel_assets.values() if row["superseded_at"] is None]
    assert len(active) == 1
    row = active[0]
    assert row["version"] == 2
    assert row["music_path"] == "/shared/channel-assets/intro/music.mp3"
    assert row["music_source_hash"] == "music-hash-1"
    # The video source did not change, so its hash is carried over untouched.
    assert row["source_hash"] == "hash-1"
    assert row["duration_seconds"] == 4.0  # the clip's own ffprobe'd duration
    assert row["video_path"].endswith("with_music_v2.mp4")
    assert len(pool.store.outbox_events) == 2
    latest = pool.store.outbox_events[max(pool.store.outbox_events)]
    assert latest["event_type"] == "channel_asset_normalized"
    assert latest["payload"]["payload"]["version"] == 2


@pytest.mark.asyncio
async def test_normalize_music_skips_remux_on_matching_music_hash(shared_volume_root) -> None:
    """FR65.6 for the music half — cached against music_source_hash, not the
    video's source_hash."""
    handler, pool = _build_normalize_handler()

    with patch("subprocess.run", side_effect=fake_probe_run) as mock_run:
        await handler.handle(FakeMessage(make_normalize_envelope(message_id="msg-n1")))
        await handler.handle(FakeMessage(make_normalize_music_envelope(message_id="msg-m1")))
        calls_after_mux = mock_run.call_count
        await handler.handle(FakeMessage(make_normalize_music_envelope(message_id="msg-m2")))

    assert mock_run.call_count == calls_after_mux
    assert len(pool.store.channel_assets) == 2  # no third version
    assert len(pool.store.outbox_events) == 2
    assert "msg-m2" in pool.store.processed_message_ids


@pytest.mark.asyncio
async def test_normalize_video_after_music_still_caches_on_the_video_hash(shared_volume_root) -> None:
    """The reason music has its own hash column: re-sending the SAME video
    upload after a music upload must still be recognised as unchanged."""
    handler, pool = _build_normalize_handler()

    with patch("subprocess.run", side_effect=fake_probe_run) as mock_run:
        await handler.handle(FakeMessage(make_normalize_envelope(message_id="msg-n1")))
        await handler.handle(FakeMessage(make_normalize_music_envelope(message_id="msg-m1")))
        calls_after_mux = mock_run.call_count
        await handler.handle(FakeMessage(make_normalize_envelope(message_id="msg-n2")))

    assert mock_run.call_count == calls_after_mux  # no re-transcode
    assert len(pool.store.channel_assets) == 2


def make_channel_asset_rendered_envelope(
    message_id: str = "msg-r1",
    kind: str = "outro",
    video_path: str = "/shared/channel-assets/outro/1080p60/rendered.mp4",
    video_duration_seconds: float = 18.0,
    render_quality: str = "1080p60",
) -> bytes:
    envelope = {
        "message_id": message_id,
        "saga_id": "saga-1",
        "project_id": "channel-asset-admin",
        "schema_version": "1.0",
        "timestamp": "2026-09-10T00:00:00Z",
        "payload": {
            "event_type": "channel_asset_rendered",
            "kind": kind,
            "video_path": video_path,
            "video_duration_seconds": video_duration_seconds,
            "render_quality": render_quality,
        },
    }
    return json.dumps(envelope).encode("utf-8")


def _build_rendered_handler() -> tuple[ChannelAssetRenderedEventHandler, FakePool]:
    pool = FakePool()
    inbox = InboxRepository(pool)
    outbox = OutboxRepository()
    channel_assets = ChannelAssetsRepository(pool)
    handler = ChannelAssetRenderedEventHandler(pool, channel_assets, inbox, outbox)
    return handler, pool


@pytest.mark.asyncio
async def test_channel_asset_rendered_registers_for_the_rendered_quality_only() -> None:
    """FR65.5: one asset per quality. Rendering names the quality it actually
    rendered at, and only that row may be registered — a 1080p60 outro
    registered as the 4k60 one would fail the concat at assembly time."""
    handler, pool = _build_rendered_handler()

    await handler.handle(FakeMessage(make_channel_asset_rendered_envelope()))

    assert len(pool.store.channel_assets) == 1
    assert len(pool.store.outbox_events) == 1
    row = next(iter(pool.store.channel_assets.values()))
    assert row["render_quality"] == "1080p60"
    assert row["kind"] == "outro"
    assert row["video_path"] == "/shared/channel-assets/outro/1080p60/rendered.mp4"
    event = next(iter(pool.store.outbox_events.values()))
    assert event["event_type"] == "channel_asset_normalized"
    assert event["payload"]["payload"]["kind"] == "outro"
    assert event["payload"]["payload"]["render_quality"] == "1080p60"


@pytest.mark.asyncio
async def test_channel_asset_rendered_keeps_qualities_apart() -> None:
    """Two qualities of the same kind are two separate active assets, not one
    superseding the other."""
    handler, pool = _build_rendered_handler()

    await handler.handle(FakeMessage(make_channel_asset_rendered_envelope(message_id="msg-r1")))
    await handler.handle(
        FakeMessage(
            make_channel_asset_rendered_envelope(
                message_id="msg-r2",
                render_quality="4k60",
                video_path="/shared/channel-assets/outro/4k60/rendered.mp4",
            )
        )
    )

    active = [row for row in pool.store.channel_assets.values() if row["superseded_at"] is None]
    assert {row["render_quality"] for row in active} == {"1080p60", "4k60"}


@pytest.mark.asyncio
async def test_channel_asset_rendered_is_idempotent_on_duplicate_message_id() -> None:
    handler, pool = _build_rendered_handler()
    message = FakeMessage(make_channel_asset_rendered_envelope(message_id="msg-r1"))

    await handler.handle(message)
    await handler.handle(FakeMessage(make_channel_asset_rendered_envelope(message_id="msg-r1")))

    assert len(pool.store.channel_assets) == 1
    assert len(pool.store.outbox_events) == 1


@pytest.mark.asyncio
async def test_channel_asset_render_failed_event_is_ignored() -> None:
    """This queue also carries channel_asset_render_failed (Orchestrator's
    own concern) — video-assembly has nothing to register for a failed
    render, and must not choke on it."""
    handler, pool = _build_rendered_handler()
    envelope = json.loads(make_channel_asset_rendered_envelope())
    envelope["payload"]["event_type"] = "channel_asset_render_failed"
    message = FakeMessage(json.dumps(envelope).encode("utf-8"))

    await handler.handle(message)

    assert message.acked is True
    assert len(pool.store.channel_assets) == 0
    assert len(pool.store.outbox_events) == 0
