"""Unit tests for the messaging adapter: consumer + Inbox/Outbox (ADR-0013).

Uses fakes for the AMQP surface (AckableMessage) and a FakePool standing
in for asyncpg (tests/adapters/fake_postgres.py) so no real RabbitMQ or
PostgreSQL connection is needed.
"""

from __future__ import annotations

import json

import pytest

from adapters.messaging.consumer import ParseScriptCommandHandler
from adapters.parsing.manim_script_parser import ManimScriptParser
from adapters.persistence.inbox import InboxRepository
from adapters.persistence.outbox import OutboxRepository
from application.parse_script import ParseScriptUseCase
from tests.adapters.fake_postgres import FakePool


class FakeMessage:
    def __init__(self, body: bytes) -> None:
        self.body = body
        self.acked = False

    async def ack(self) -> None:
        self.acked = True


VALID_SCRIPT = (
    "from conceptflow import *\n\n"
    "class DemoScene(ConceptFlowScene):\n"
    "    def construct(self):\n"
    '        self.narrate("hello")\n'
)


def make_envelope(message_id: str = "msg-1", raw_script: str = VALID_SCRIPT) -> bytes:
    envelope = {
        "message_id": message_id,
        "saga_id": "saga-1",
        "project_id": "project-1",
        "schema_version": "1.0",
        "timestamp": "2026-08-07T00:00:00Z",
        "payload": {"script_content": raw_script},
    }
    return json.dumps(envelope).encode("utf-8")


@pytest.fixture
def handler() -> tuple[ParseScriptCommandHandler, FakePool]:
    use_case = ParseScriptUseCase(ManimScriptParser())
    pool = FakePool()
    inbox = InboxRepository(pool)
    outbox = OutboxRepository()
    command_handler = ParseScriptCommandHandler(use_case, pool, inbox, outbox)
    return command_handler, pool


@pytest.mark.asyncio
async def test_enqueues_success_event_to_outbox_and_acks(handler) -> None:
    command_handler, pool = handler
    message = FakeMessage(make_envelope())

    await command_handler.handle(message)

    assert message.acked is True
    assert len(pool.store.outbox_events) == 1
    event = next(iter(pool.store.outbox_events.values()))
    assert event["event_type"] == "script_parsed"
    # Sau CR-018 event này chỉ mang tên class Scene: lời thoại đến từ
    # `script_validated` của Rendering, vì chỉ lượt dry mới biết chúng theo
    # đúng thứ tự chạy thật.
    assert event["payload"]["payload"]["scene_class_name"] == "DemoScene"
    assert "scenes" not in event["payload"]["payload"]


@pytest.mark.asyncio
async def test_enqueues_failure_event_on_syntax_error(handler) -> None:
    command_handler, pool = handler
    message = FakeMessage(make_envelope(raw_script="not a manim script"))

    await command_handler.handle(message)

    assert message.acked is True
    event = next(iter(pool.store.outbox_events.values()))
    assert event["event_type"] == "parse_failed"
    # Thông báo này hiện thẳng lên GUI cho Creator đọc, nên nó viết tiếng Việt —
    # cùng nguyên tắc với thông báo của lint (CR-017). Log nội bộ vẫn tiếng Anh.
    assert "class Scene" in event["payload"]["payload"]["reason"]


@pytest.mark.asyncio
async def test_marks_message_processed_in_inbox(handler) -> None:
    command_handler, pool = handler
    message = FakeMessage(make_envelope(message_id="msg-1"))

    await command_handler.handle(message)

    assert "msg-1" in pool.store.processed_message_ids


@pytest.mark.asyncio
async def test_skips_reprocessing_duplicate_message_id(handler) -> None:
    command_handler, pool = handler
    pool.store.processed_message_ids.add("msg-1")
    message = FakeMessage(make_envelope(message_id="msg-1"))

    await command_handler.handle(message)

    assert message.acked is True
    assert len(pool.store.outbox_events) == 0
