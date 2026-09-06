"""Unit tests for the messaging adapter: consumer + Inbox/Outbox (ADR-0013).

Uses fakes for the AMQP surface (AckableMessage) and a FakePool standing
in for asyncpg (tests/adapters/fake_postgres.py), plus a FakeVideoAssembler
standing in for ffmpeg — no real RabbitMQ/PostgreSQL/ffmpeg needed.
"""

from __future__ import annotations

import json
import os

import pytest

from adapters.messaging.consumer import AssembleVideoCommandHandler
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
    def __init__(self, fail_with: Exception | None = None) -> None:
        self._fail_with = fail_with

    def assemble(self, request: VideoAssemblyRequest, output_path: str) -> None:
        if self._fail_with is not None:
            raise self._fail_with
        _touch(output_path)


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


@pytest.mark.asyncio
async def test_skips_reprocessing_duplicate_message_id(shared_volume_root) -> None:
    handler, pool = _build_handler(FakeVideoAssembler())
    pool.store.processed_message_ids.add("msg-1")
    message = FakeMessage(make_envelope(message_id="msg-1", shared_volume_root=shared_volume_root))

    await handler.handle(message)

    assert message.acked is True
    assert len(pool.store.outbox_events) == 0
