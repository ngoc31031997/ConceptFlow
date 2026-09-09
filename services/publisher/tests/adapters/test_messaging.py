"""Unit tests for the messaging adapter: consumer + Inbox/Outbox (ADR-0013).

Uses fakes for the AMQP surface (AckableMessage) and a FakePool standing
in for asyncpg (tests/adapters/fake_postgres.py), plus a FakePublishVideoUseCase
standing in for the YouTube upload — no real RabbitMQ/PostgreSQL/Google
API needed.
"""

from __future__ import annotations

import json

import pytest

from adapters.messaging.consumer import PublishVideoCommandHandler
from adapters.persistence.inbox import InboxRepository
from adapters.persistence.outbox import OutboxRepository
from domain.errors import UploadError
from domain.models import PublishResult
from tests.adapters.fake_postgres import FakePool


class FakePublishVideoUseCase:
    def __init__(self, result: PublishResult | None = None, fail_with: Exception | None = None) -> None:
        self._result = result
        self._fail_with = fail_with
        self.last_request = None

    def publish(self, request):
        self.last_request = request
        if self._fail_with is not None:
            raise self._fail_with
        return self._result


class FakeMessage:
    def __init__(self, body: bytes) -> None:
        self.body = body
        self.acked = False

    async def ack(self) -> None:
        self.acked = True


def make_envelope(message_id: str = "msg-1") -> bytes:
    envelope = {
        "message_id": message_id,
        "saga_id": "saga-1",
        "project_id": "project-1",
        "schema_version": "1.0",
        "timestamp": "2026-08-24T00:00:00Z",
        "payload": {
            "video_path": "/shared/project-1/video/final.mp4",
            "title": "My Video",
            "description": "desc",
            "tags": ["python"],
            "visibility": "public",
        },
    }
    return json.dumps(envelope).encode("utf-8")


def _build_handler(use_case) -> tuple[PublishVideoCommandHandler, FakePool]:
    pool = FakePool()
    inbox = InboxRepository(pool)
    outbox = OutboxRepository()
    handler = PublishVideoCommandHandler(use_case, pool, inbox, outbox)
    return handler, pool


@pytest.mark.asyncio
async def test_enqueues_success_event_to_outbox_and_acks() -> None:
    use_case = FakePublishVideoUseCase(result=PublishResult(youtube_video_url="https://youtu.be/abc"))
    handler, pool = _build_handler(use_case)
    message = FakeMessage(make_envelope())

    await handler.handle(message)

    assert message.acked is True
    assert len(pool.store.outbox_events) == 1
    event = next(iter(pool.store.outbox_events.values()))
    assert event["event_type"] == "video_published"
    assert event["payload"]["payload"]["youtube_video_url"] == "https://youtu.be/abc"


@pytest.mark.asyncio
async def test_caption_status_is_carried_into_the_video_published_event() -> None:
    """CR-015 FR39.4: caption_status travels through the outbox event the
    same way youtube_video_url does, so the Orchestrator/GUI can surface a
    silently failed or skipped caption upload."""
    use_case = FakePublishVideoUseCase(
        result=PublishResult(youtube_video_url="https://youtu.be/abc", caption_status="uploaded")
    )
    handler, pool = _build_handler(use_case)
    message = FakeMessage(make_envelope())

    await handler.handle(message)

    event = next(iter(pool.store.outbox_events.values()))
    assert event["payload"]["payload"]["caption_status"] == "uploaded"


@pytest.mark.asyncio
async def test_no_caption_status_key_when_no_caption_was_requested() -> None:
    use_case = FakePublishVideoUseCase(
        result=PublishResult(youtube_video_url="https://youtu.be/abc", caption_status=None)
    )
    handler, pool = _build_handler(use_case)
    message = FakeMessage(make_envelope())

    await handler.handle(message)

    event = next(iter(pool.store.outbox_events.values()))
    assert "caption_status" not in event["payload"]["payload"]


@pytest.mark.asyncio
async def test_caption_path_and_language_are_parsed_from_the_payload() -> None:
    use_case = FakePublishVideoUseCase(result=PublishResult(youtube_video_url="https://youtu.be/abc"))
    handler, _ = _build_handler(use_case)
    envelope = json.loads(make_envelope())
    envelope["payload"]["caption_path"] = "/shared/project-1/video/final.srt"
    envelope["payload"]["caption_language"] = "vi"
    message = FakeMessage(json.dumps(envelope).encode("utf-8"))

    await handler.handle(message)

    assert use_case.last_request.caption_path == "/shared/project-1/video/final.srt"
    assert use_case.last_request.caption_language == "vi"


@pytest.mark.asyncio
async def test_missing_caption_fields_in_payload_default_to_none() -> None:
    use_case = FakePublishVideoUseCase(result=PublishResult(youtube_video_url="https://youtu.be/abc"))
    handler, _ = _build_handler(use_case)
    message = FakeMessage(make_envelope())

    await handler.handle(message)

    assert use_case.last_request.caption_path is None
    assert use_case.last_request.caption_language is None


@pytest.mark.asyncio
async def test_enqueues_failure_event_on_upload_error() -> None:
    use_case = FakePublishVideoUseCase(fail_with=UploadError("boom"))
    handler, pool = _build_handler(use_case)
    message = FakeMessage(make_envelope())

    await handler.handle(message)

    assert message.acked is True
    event = next(iter(pool.store.outbox_events.values()))
    assert event["event_type"] == "publish_failed"
    assert "boom" in event["payload"]["payload"]["error_message"]


@pytest.mark.asyncio
async def test_marks_message_processed_in_inbox() -> None:
    use_case = FakePublishVideoUseCase(result=PublishResult(youtube_video_url="https://youtu.be/abc"))
    handler, pool = _build_handler(use_case)
    message = FakeMessage(make_envelope(message_id="msg-1"))

    await handler.handle(message)

    assert "msg-1" in pool.store.processed_message_ids


@pytest.mark.asyncio
async def test_skips_reprocessing_duplicate_message_id() -> None:
    use_case = FakePublishVideoUseCase(result=PublishResult(youtube_video_url="https://youtu.be/abc"))
    handler, pool = _build_handler(use_case)
    pool.store.processed_message_ids.add("msg-1")
    message = FakeMessage(make_envelope(message_id="msg-1"))

    await handler.handle(message)

    assert message.acked is True
    assert len(pool.store.outbox_events) == 0
