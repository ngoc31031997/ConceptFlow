"""Unit tests for the messaging adapter: consumer + Inbox/Outbox (ADR-0013).

Verifies the per-scene-immediate-commit behavior (NFR Design) using
FakePool — each scene_render_started/scene_rendered must land as its own
Outbox row (published_at still NULL) as soon as it happens, not only
after the whole batch/command finishes.
"""

from __future__ import annotations

import json
import os

import pytest

from adapters.messaging.consumer import RenderScenesCommandHandler
from adapters.persistence.inbox import InboxRepository
from adapters.persistence.outbox import OutboxRepository
from application.render_scene import RenderSceneUseCase
from application.render_scenes_batch import RenderScenesBatchUseCase
from domain.errors import AnimationEngineError
from domain.ports import AnimationRendererPort
from tests.adapters.fake_postgres import FakePool


def _write_stub_mp4(path: str) -> None:
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "wb") as f:
        f.write(b"stub-mp4-bytes")


class FakeAnimationRenderer(AnimationRendererPort):
    def __init__(self, fail_at_scene: int | None = None) -> None:
        self._fail_at_scene = fail_at_scene

    def render(self, request, output_path: str) -> float:
        if request.scene_index == self._fail_at_scene:
            raise AnimationEngineError("engine crashed")
        _write_stub_mp4(output_path)
        return 5.0


class FakeMessage:
    def __init__(self, body: bytes) -> None:
        self.body = body
        self.acked = False

    async def ack(self) -> None:
        self.acked = True


def make_envelope(message_id: str = "msg-1", scenes: list | None = None) -> bytes:
    scenes = scenes or [
        {
            "scene_index": 0,
            "narration_text": "hello",
            "illustration_hint": None,
            "code_snippet": None,
            "code_language": None,
            "animation_template_id": "concept_illustration",
            "audio_path": "/shared/project-1/audio/0_en.wav",
            "duration_seconds": 5.0,
        }
    ]
    envelope = {
        "message_id": message_id,
        "saga_id": "saga-1",
        "project_id": "project-1",
        "schema_version": "1.0",
        "timestamp": "2026-08-07T00:00:00Z",
        "payload": {"scenes": scenes},
    }
    return json.dumps(envelope).encode("utf-8")


@pytest.fixture
def shared_volume_root(tmp_path, monkeypatch):
    monkeypatch.setattr("adapters.storage.artifact_paths.SHARED_VOLUME_ROOT", str(tmp_path))
    return tmp_path


def _build_handler(renderer: AnimationRendererPort) -> tuple[RenderScenesCommandHandler, FakePool]:
    batch_use_case = RenderScenesBatchUseCase(RenderSceneUseCase(renderer))
    pool = FakePool()
    inbox = InboxRepository(pool)
    outbox = OutboxRepository()
    handler = RenderScenesCommandHandler(batch_use_case, pool, inbox, outbox)
    return handler, pool


@pytest.mark.asyncio
async def test_success_enqueues_started_rendered_and_completed_events(shared_volume_root) -> None:
    handler, pool = _build_handler(FakeAnimationRenderer())
    message = FakeMessage(make_envelope())

    await handler.handle(message)

    assert message.acked is True
    event_types = [row["event_type"] for row in pool.store.outbox_events.values()]
    assert event_types == ["scene_render_started", "scene_rendered", "rendering_completed"]


@pytest.mark.asyncio
async def test_failure_enqueues_started_and_failed_events_only(shared_volume_root) -> None:
    handler, pool = _build_handler(FakeAnimationRenderer(fail_at_scene=0))
    message = FakeMessage(make_envelope())

    await handler.handle(message)

    assert message.acked is True
    event_types = [row["event_type"] for row in pool.store.outbox_events.values()]
    assert event_types == ["scene_render_started", "rendering_failed"]


@pytest.mark.asyncio
async def test_marks_message_processed_in_inbox(shared_volume_root) -> None:
    handler, pool = _build_handler(FakeAnimationRenderer())
    message = FakeMessage(make_envelope(message_id="msg-1"))

    await handler.handle(message)

    assert "msg-1" in pool.store.processed_message_ids


@pytest.mark.asyncio
async def test_skips_reprocessing_duplicate_message_id(shared_volume_root) -> None:
    handler, pool = _build_handler(FakeAnimationRenderer())
    pool.store.processed_message_ids.add("msg-1")
    message = FakeMessage(make_envelope(message_id="msg-1"))

    await handler.handle(message)

    assert message.acked is True
    assert len(pool.store.outbox_events) == 0
