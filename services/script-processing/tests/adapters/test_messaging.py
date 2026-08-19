"""Unit tests for the messaging adapter: consumer + Inbox/Outbox (ADR-0013).

Uses fakes for the AMQP surface (AckableMessage) and a FakePool standing
in for asyncpg (tests/adapters/fake_postgres.py) so no real RabbitMQ or
PostgreSQL connection is needed.
"""

from __future__ import annotations

import json

import pytest

from adapters.messaging.consumer import ParseScriptCommandHandler
from adapters.parsing.markdown_parser import MarkdownScriptParser
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


def make_envelope(message_id: str = "msg-1", raw_script: str = "## Scene 1\nhello") -> bytes:
    envelope = {
        "message_id": message_id,
        "saga_id": "saga-1",
        "project_id": "project-1",
        "schema_version": "1.0",
        "timestamp": "2026-08-07T00:00:00Z",
        "payload": {"raw_script": raw_script},
    }
    return json.dumps(envelope).encode("utf-8")


@pytest.fixture
def handler() -> tuple[ParseScriptCommandHandler, FakePool]:
    use_case = ParseScriptUseCase(MarkdownScriptParser())
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
    assert event["payload"]["payload"]["scenes"][0]["narration_text"] == "hello"


@pytest.mark.asyncio
async def test_enqueues_failure_event_on_syntax_error(handler) -> None:
    command_handler, pool = handler
    message = FakeMessage(make_envelope(raw_script="no headings here"))

    await command_handler.handle(message)

    assert message.acked is True
    event = next(iter(pool.store.outbox_events.values()))
    assert event["event_type"] == "parse_failed"
    assert event["payload"]["payload"]["reason"] == "no scenes found"


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
