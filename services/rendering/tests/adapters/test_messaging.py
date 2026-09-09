"""Unit tests for the messaging adapter: consumer + Inbox/Outbox (ADR-0013)."""

from __future__ import annotations

import json

import pytest

from adapters.messaging.consumer import RenderScriptCommandHandler
from adapters.persistence.inbox import InboxRepository
from adapters.persistence.outbox import OutboxRepository
from application.render_script import RenderScriptUseCase
from domain.errors import AnimationEngineError
from domain.ports import ManimScriptRendererPort
from tests.adapters.fake_postgres import FakePool


class FakeManimScriptRenderer(ManimScriptRendererPort):
    def __init__(self, should_fail: bool = False) -> None:
        self._should_fail = should_fail

    def dry_run(self, request):
        from domain.models import DryRunResult

        return DryRunResult(narrations=["dòng một", "dòng hai"])

    def render(self, request, output_path: str):
        if self._should_fail:
            raise AnimationEngineError("engine crashed")
        import os

        from domain.models import ScriptRenderResult

        os.makedirs(os.path.dirname(output_path), exist_ok=True)
        with open(output_path, "wb") as f:
            f.write(b"stub-mp4-bytes")
        offsets = [float(i) * 5.0 for i in range(len(request.narration_segments))]
        return ScriptRenderResult(
            video_path=output_path,
            wait_offsets=offsets,
            video_duration_seconds=(offsets[-1] + 10.0) if offsets else 0.0,
        )


class FakeMessage:
    def __init__(self, body: bytes) -> None:
        self.body = body
        self.acked = False

    async def ack(self) -> None:
        self.acked = True


def make_envelope(message_id: str = "msg-1", scenes: list | None = None) -> bytes:
    scenes = scenes or [
        {"scene_index": 0, "audio_path": "/shared/project-1/audio/0_en.wav", "duration_seconds": 5.0},
    ]
    envelope = {
        "message_id": message_id,
        "saga_id": "saga-1",
        "project_id": "project-1",
        "schema_version": "1.0",
        "timestamp": "2026-08-07T00:00:00Z",
        "payload": {
            "scenes": scenes,
            "script_content": (
                "class DemoScene(Scene):\n    def construct(self):\n        self.wait(AUTO)\n"
            ),
            "scene_class_name": "DemoScene",
        },
    }
    return json.dumps(envelope).encode("utf-8")


@pytest.fixture
def shared_volume_root(tmp_path, monkeypatch):
    monkeypatch.setattr("adapters.storage.artifact_paths.SHARED_VOLUME_ROOT", str(tmp_path))
    return tmp_path


def _build_handler(renderer: ManimScriptRendererPort) -> tuple[RenderScriptCommandHandler, FakePool]:
    use_case = RenderScriptUseCase(renderer)
    pool = FakePool()
    inbox = InboxRepository(pool)
    outbox = OutboxRepository()
    handler = RenderScriptCommandHandler(use_case, pool, inbox, outbox)
    return handler, pool


@pytest.mark.asyncio
async def test_success_enqueues_rendering_completed_with_video_path(shared_volume_root) -> None:
    handler, pool = _build_handler(FakeManimScriptRenderer())
    message = FakeMessage(make_envelope())

    await handler.handle(message)

    assert message.acked is True
    event = next(iter(pool.store.outbox_events.values()))
    assert event["event_type"] == "rendering_completed"
    assert event["payload"]["payload"]["video_path"].endswith("rendered.mp4")
    # CR-002 FR10.2: the event must carry where each narration actually starts.
    assert event["payload"]["payload"]["wait_offsets"] == [0.0]
    assert event["payload"]["payload"]["video_duration_seconds"] == 10.0


@pytest.mark.asyncio
async def test_failure_enqueues_rendering_failed(shared_volume_root) -> None:
    handler, pool = _build_handler(FakeManimScriptRenderer(should_fail=True))
    message = FakeMessage(make_envelope())

    await handler.handle(message)

    assert message.acked is True
    event = next(iter(pool.store.outbox_events.values()))
    assert event["event_type"] == "rendering_failed"
    assert event["payload"]["payload"]["error_message"] == "engine crashed"


@pytest.mark.asyncio
async def test_marks_message_processed_in_inbox(shared_volume_root) -> None:
    handler, pool = _build_handler(FakeManimScriptRenderer())
    message = FakeMessage(make_envelope(message_id="msg-1"))

    await handler.handle(message)

    assert "msg-1" in pool.store.processed_message_ids


@pytest.mark.asyncio
async def test_skips_reprocessing_duplicate_message_id(shared_volume_root) -> None:
    handler, pool = _build_handler(FakeManimScriptRenderer())
    pool.store.processed_message_ids.add("msg-1")
    message = FakeMessage(make_envelope(message_id="msg-1"))

    await handler.handle(message)

    assert message.acked is True
    assert len(pool.store.outbox_events) == 0
