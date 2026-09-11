"""Unit tests for CR-007's `generate_clips` command handler and its
`adapters/clips/vertical_clip.py` support code.

`FfmpegVideoAssembler._run_ffmpeg` is patched everywhere so these tests never
shell out to a real ffmpeg — same style as test_messaging.py's
FakeVideoAssembler standing in for the assembler.
"""

from __future__ import annotations

import json

import pytest

from adapters.clips.vertical_clip import ClipRequest, generate_clip, slugify
from adapters.messaging.consumer import GenerateClipsCommandHandler
from adapters.persistence.inbox import InboxRepository
from adapters.persistence.outbox import OutboxRepository
from domain.clip_rules import ClipThresholds
from domain.models import SubtitleCue
from tests.adapters.fake_postgres import FakePool


class FakeMessage:
    def __init__(self, body: bytes) -> None:
        self.body = body
        self.acked = False

    async def ack(self) -> None:
        self.acked = True


@pytest.fixture(autouse=True)
def no_real_ffmpeg(monkeypatch):
    """Every test in this file cuts a clip through generate_clip/_run_ffmpeg —
    stub it out so no real ffmpeg binary is required, and record calls for
    assertions that need to inspect the command."""
    calls: list[list[str]] = []

    def _fake_run_ffmpeg(args):
        calls.append(args)
        # -y ... <output_path> is always the last element (see
        # adapters/clips/vertical_clip.py::generate_clip's cmd construction).
        output_path = args[-1]
        import os

        os.makedirs(os.path.dirname(output_path), exist_ok=True)
        with open(output_path, "w") as f:
            f.write("fake mp4 bytes")

    monkeypatch.setattr(
        "adapters.clips.vertical_clip.FfmpegVideoAssembler._run_ffmpeg", staticmethod(_fake_run_ffmpeg)
    )
    return calls


# --- slugify -------------------------------------------------------------


def test_slugify_lowercases_and_replaces_special_characters() -> None:
    assert slugify('ví dụ chạy thật!') == "v-d-ch-y-th-t"


def test_slugify_never_returns_empty() -> None:
    assert slugify("   ") == "clip"


# --- generate_clip: validation --------------------------------------------


def test_generate_clip_reports_error_without_calling_ffmpeg(no_real_ffmpeg, tmp_path, monkeypatch) -> None:
    monkeypatch.setattr("adapters.storage.artifact_paths.SHARED_VOLUME_ROOT", str(tmp_path))
    request = ClipRequest(name="ví dụ chạy thật", start_seconds=42.5, end_seconds=96.0)  # 53.5s
    result = generate_clip(
        project_id="project-1",
        video_path=str(tmp_path / "final.mp4"),
        request=request,
        preset="long",
        intro_duration_seconds=0.0,
        subtitle_cues=[],
        thresholds=ClipThresholds(),
    )
    assert result["status"] == "error"
    assert "long" in result["error_message"]
    assert no_real_ffmpeg == []  # never called ffmpeg for an invalid duration


def test_generate_clip_ok_writes_output_path(no_real_ffmpeg, tmp_path, monkeypatch) -> None:
    monkeypatch.setattr("adapters.storage.artifact_paths.SHARED_VOLUME_ROOT", str(tmp_path))
    request = ClipRequest(name="ví dụ chạy thật", start_seconds=42.5, end_seconds=96.0)  # 53.5s
    result = generate_clip(
        project_id="project-1",
        video_path=str(tmp_path / "final.mp4"),
        request=request,
        preset="short",
        intro_duration_seconds=0.0,
        subtitle_cues=[],
        thresholds=ClipThresholds(),
    )
    assert result["status"] == "ok"
    assert result["duration_seconds"] == pytest.approx(53.5)
    assert result["output_path"] == str(tmp_path / "project-1" / "clips" / "v-d-ch-y-th-t_short.mp4")
    assert len(no_real_ffmpeg) == 1


def test_generate_clip_adds_intro_duration_to_start_and_end(no_real_ffmpeg, tmp_path, monkeypatch) -> None:
    """D5 rủi ro / CR-023: clip_marks' t_start/t_end are Manim-video seconds,
    not yet shifted by the intro — video-assembly must add the offset itself,
    the same amount ffmpeg_assembler's effective_lead_in already applies."""
    monkeypatch.setattr("adapters.storage.artifact_paths.SHARED_VOLUME_ROOT", str(tmp_path))
    request = ClipRequest(name="clip", start_seconds=10.0, end_seconds=40.0)
    generate_clip(
        project_id="project-1",
        video_path=str(tmp_path / "final.mp4"),
        request=request,
        preset="short",
        intro_duration_seconds=5.0,
        subtitle_cues=[],
        thresholds=ClipThresholds(),
    )
    cmd = no_real_ffmpeg[0]
    ss_index = cmd.index("-ss")
    t_index = cmd.index("-t")
    assert cmd[ss_index + 1] == "15.000"  # 10.0 + 5.0
    assert cmd[t_index + 1] == "30.000"  # duration unaffected by the shift


# --- generate_clip: subtitle cue shifting/clamping ------------------------


def test_generate_clip_burns_subtitles_shifted_and_clamped_to_clip(
    no_real_ffmpeg, tmp_path, monkeypatch
) -> None:
    monkeypatch.setattr("adapters.storage.artifact_paths.SHARED_VOLUME_ROOT", str(tmp_path))
    request = ClipRequest(name="clip", start_seconds=10.0, end_seconds=20.0)  # 10s clip
    cues = [
        SubtitleCue(scene_index=0, text="trước hẳn", start_time=0.0, end_time=5.0),  # outside, dropped
        SubtitleCue(scene_index=1, text="vắt biên đầu", start_time=8.0, end_time=12.0),  # clamp start
        SubtitleCue(scene_index=2, text="trong khoảng", start_time=13.0, end_time=15.0),  # untouched shift
        SubtitleCue(scene_index=3, text="vắt biên cuối", start_time=19.0, end_time=25.0),  # clamp end
        SubtitleCue(scene_index=4, text="sau hẳn", start_time=21.0, end_time=30.0),  # outside, dropped
    ]

    generate_clip(
        project_id="project-1",
        video_path=str(tmp_path / "final.mp4"),
        request=request,
        preset="short",
        intro_duration_seconds=0.0,
        subtitle_cues=cues,
        thresholds=ClipThresholds(),
    )

    ass_path = tmp_path / "project-1" / "clips" / "clip_short.ass"
    assert ass_path.exists()
    content = ass_path.read_text(encoding="utf-8")
    # vắt biên đầu: start clamped to clip's 0, end at 12-10=2
    assert "0:00:00.00,0:00:02.00" in content
    # trong khoảng: shifted by -10
    assert "0:00:03.00,0:00:05.00" in content
    # vắt biên cuối: start at 19-10=9, end clamped to clip duration (10)
    assert "0:00:09.00,0:00:10.00" in content
    assert "trước hẳn" not in content
    assert "sau hẳn" not in content

    # Style is the vertical one (D6): PlayRes matches 1080x1920, not 1920x1080.
    assert "PlayResX: 1080" in content
    assert "PlayResY: 1920" in content


def test_generate_clip_with_no_intersecting_cues_skips_subtitle_burn(
    no_real_ffmpeg, tmp_path, monkeypatch
) -> None:
    monkeypatch.setattr("adapters.storage.artifact_paths.SHARED_VOLUME_ROOT", str(tmp_path))
    request = ClipRequest(name="clip", start_seconds=100.0, end_seconds=130.0)
    cues = [SubtitleCue(scene_index=0, text="không liên quan", start_time=0.0, end_time=5.0)]

    generate_clip(
        project_id="project-1",
        video_path=str(tmp_path / "final.mp4"),
        request=request,
        preset="short",
        intro_duration_seconds=0.0,
        subtitle_cues=cues,
        thresholds=ClipThresholds(),
    )

    ass_path = tmp_path / "project-1" / "clips" / "clip_short.ass"
    assert not ass_path.exists()


# --- GenerateClipsCommandHandler: message-level behaviour -----------------


def make_generate_clips_envelope(
    message_id: str = "msg-clips-1",
    video_path: str = "/shared/project-1/video/final.mp4",
    intro_duration_seconds: float = 0.0,
    requests: list[dict] | None = None,
    subtitle_cues: list[dict] | None = None,
) -> bytes:
    envelope = {
        "message_id": message_id,
        "saga_id": "saga-1",
        "project_id": "project-1",
        "event_type": "generate_clips",
        "schema_version": "1.0",
        "timestamp": "2026-09-11T00:00:00Z",
        "payload": {
            "event_type": "generate_clips",
            "video_path": video_path,
            "intro_duration_seconds": intro_duration_seconds,
            "subtitle_cues": subtitle_cues or [],
            "requests": requests
            if requests is not None
            else [
                {
                    "name": "ví dụ chạy thật",
                    "start_seconds": 42.5,
                    "end_seconds": 102.5,  # 60s — the one duration valid for both presets
                    "presets": ["short", "long"],
                }
            ],
        },
    }
    return json.dumps(envelope).encode("utf-8")


def _build_handler(tmp_path, monkeypatch) -> tuple[GenerateClipsCommandHandler, FakePool]:
    monkeypatch.setattr("adapters.storage.artifact_paths.SHARED_VOLUME_ROOT", str(tmp_path))
    pool = FakePool()
    inbox = InboxRepository(pool)
    outbox = OutboxRepository()
    handler = GenerateClipsCommandHandler(pool, inbox, outbox, ClipThresholds())
    return handler, pool


def _clips_generated_payload(pool: FakePool) -> dict:
    event = next(iter(pool.store.outbox_events.values()))
    return event["payload"]["payload"]


@pytest.mark.asyncio
async def test_one_request_two_presets_both_ok(no_real_ffmpeg, tmp_path, monkeypatch) -> None:
    handler, pool = _build_handler(tmp_path, monkeypatch)
    message = FakeMessage(make_generate_clips_envelope())

    await handler.handle(message)

    assert message.acked is True
    payload = _clips_generated_payload(pool)
    assert payload["event_type"] == "clips_generated"
    clips = payload["clips"]
    assert len(clips) == 2
    assert all(c["status"] == "ok" for c in clips)
    presets = {c["preset"] for c in clips}
    assert presets == {"short", "long"}


@pytest.mark.asyncio
async def test_short_preset_error_does_not_block_long_preset(no_real_ffmpeg, tmp_path, monkeypatch) -> None:
    """FR19.7/D1 — a 75s segment is invalid for `short` but valid for `long`;
    the loop must not stop after the first failure."""
    handler, pool = _build_handler(tmp_path, monkeypatch)
    requests = [
        {
            "name": "đoạn 75 giây",
            "start_seconds": 0.0,
            "end_seconds": 75.0,
            "presets": ["short", "long"],
        }
    ]
    message = FakeMessage(make_generate_clips_envelope(requests=requests))

    await handler.handle(message)

    payload = _clips_generated_payload(pool)
    clips = {c["preset"]: c for c in payload["clips"]}
    assert clips["short"]["status"] == "error"
    assert "short" in clips["short"]["error_message"]
    assert clips["long"]["status"] == "ok"
    assert clips["long"]["duration_seconds"] == pytest.approx(75.0)


@pytest.mark.asyncio
async def test_generate_clips_is_idempotent_on_message_id(no_real_ffmpeg, tmp_path, monkeypatch) -> None:
    handler, pool = _build_handler(tmp_path, monkeypatch)
    body = make_generate_clips_envelope()

    await handler.handle(FakeMessage(body))
    await handler.handle(FakeMessage(body))

    assert len(pool.store.outbox_events) == 1


@pytest.mark.asyncio
async def test_generate_clips_intro_duration_shifts_both_presets(no_real_ffmpeg, tmp_path, monkeypatch) -> None:
    handler, pool = _build_handler(tmp_path, monkeypatch)
    requests = [
        {
            "name": "clip với intro",
            "start_seconds": 10.0,
            "end_seconds": 70.0,  # 60s segment
            "presets": ["short", "long"],
        }
    ]
    message = FakeMessage(
        make_generate_clips_envelope(requests=requests, intro_duration_seconds=3.0)
    )

    await handler.handle(message)

    # -ss must reflect start_seconds + intro_duration_seconds for every call.
    ss_values = {cmd[cmd.index("-ss") + 1] for cmd in no_real_ffmpeg}
    assert ss_values == {"13.000"}
