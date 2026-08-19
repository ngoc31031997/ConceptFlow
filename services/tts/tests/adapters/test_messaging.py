"""Unit tests for the messaging adapter: consumer + Inbox/Outbox (ADR-0013, ADR-0014).

Uses fakes for the AMQP surface (AckableMessage) and a FakePool standing
in for asyncpg (tests/adapters/fake_postgres.py) so no real RabbitMQ or
PostgreSQL connection is needed.
"""

from __future__ import annotations

import json
import os
import wave

import pytest

from adapters.messaging.consumer import SynthesizeSpeechCommandHandler
from adapters.persistence.inbox import InboxRepository
from adapters.persistence.outbox import OutboxRepository
from application.synthesize_speech import SynthesizeSpeechUseCase
from application.synthesize_speech_batch import SynthesizeSpeechBatchUseCase
from domain.errors import TTSEngineError
from domain.ports import TTSEnginePort
from tests.adapters.fake_postgres import FakePool


def _write_silent_wav(path: str, duration_seconds: float = 2.0, framerate: int = 16000) -> None:
    os.makedirs(os.path.dirname(path), exist_ok=True)
    n_frames = int(duration_seconds * framerate)
    with wave.open(path, "wb") as wav_file:
        wav_file.setnchannels(1)
        wav_file.setsampwidth(2)
        wav_file.setframerate(framerate)
        wav_file.writeframes(b"\x00\x00" * n_frames)


class FakeTTSEngine(TTSEnginePort):
    def __init__(self, fail_with: Exception | None = None) -> None:
        self._fail_with = fail_with

    def synthesize(self, text: str, language: str, output_path: str) -> float:
        if self._fail_with is not None:
            raise self._fail_with
        _write_silent_wav(output_path, duration_seconds=2.0)
        return 2.0


class FakeMessage:
    def __init__(self, body: bytes) -> None:
        self.body = body
        self.acked = False

    async def ack(self) -> None:
        self.acked = True


def make_envelope(message_id: str = "msg-1", scenes: list | None = None) -> bytes:
    scenes = scenes or [{"scene_index": 0, "narration_text": "hello", "language": "en"}]
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


def _build_handler(engine: TTSEnginePort) -> tuple[SynthesizeSpeechCommandHandler, FakePool]:
    batch_use_case = SynthesizeSpeechBatchUseCase(SynthesizeSpeechUseCase(engine))
    pool = FakePool()
    inbox = InboxRepository(pool)
    outbox = OutboxRepository()
    handler = SynthesizeSpeechCommandHandler(batch_use_case, pool, inbox, outbox)
    return handler, pool


@pytest.mark.asyncio
async def test_enqueues_success_event_to_outbox_and_acks(shared_volume_root) -> None:
    handler, pool = _build_handler(FakeTTSEngine())
    message = FakeMessage(make_envelope())

    await handler.handle(message)

    assert message.acked is True
    assert len(pool.store.outbox_events) == 1
    event = next(iter(pool.store.outbox_events.values()))
    assert event["event_type"] == "speech_synthesized"
    assert event["payload"]["payload"]["scenes"][0]["duration_seconds"] == 2.0


@pytest.mark.asyncio
async def test_enqueues_failure_event_on_engine_error(shared_volume_root) -> None:
    handler, pool = _build_handler(FakeTTSEngine(fail_with=TTSEngineError("boom")))
    message = FakeMessage(make_envelope())

    await handler.handle(message)

    assert message.acked is True
    event = next(iter(pool.store.outbox_events.values()))
    assert event["event_type"] == "synthesis_failed"
    assert "tts_engine_failure" in event["payload"]["payload"]["error_message"]


@pytest.mark.asyncio
async def test_marks_message_processed_in_inbox(shared_volume_root) -> None:
    handler, pool = _build_handler(FakeTTSEngine())
    message = FakeMessage(make_envelope(message_id="msg-1"))

    await handler.handle(message)

    assert "msg-1" in pool.store.processed_message_ids


@pytest.mark.asyncio
async def test_skips_reprocessing_duplicate_message_id(shared_volume_root) -> None:
    handler, pool = _build_handler(FakeTTSEngine())
    pool.store.processed_message_ids.add("msg-1")
    message = FakeMessage(make_envelope(message_id="msg-1"))

    await handler.handle(message)

    assert message.acked is True
    assert len(pool.store.outbox_events) == 0
