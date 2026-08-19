"""Unit tests for InboxRepository/OutboxRepository (ADR-0013)."""

from __future__ import annotations

import pytest

from adapters.persistence.inbox import InboxRepository
from adapters.persistence.outbox import OutboxRepository
from tests.adapters.fake_postgres import FakePool


@pytest.mark.asyncio
async def test_inbox_has_processed_false_by_default() -> None:
    pool = FakePool()
    inbox = InboxRepository(pool)

    assert await inbox.has_processed("msg-1") is False


@pytest.mark.asyncio
async def test_inbox_mark_processed_then_has_processed_true() -> None:
    pool = FakePool()
    inbox = InboxRepository(pool)

    async with pool.acquire() as conn:
        await inbox.mark_processed(conn, "msg-1")

    assert await inbox.has_processed("msg-1") is True


@pytest.mark.asyncio
async def test_outbox_enqueue_stores_event() -> None:
    pool = FakePool()
    outbox = OutboxRepository()

    async with pool.acquire() as conn:
        await outbox.enqueue(
            conn, aggregate_id="project-1", event_type="scenes_classified", envelope={"foo": "bar"}
        )

    row = next(iter(pool.store.outbox_events.values()))
    assert row["aggregate_id"] == "project-1"
    assert row["event_type"] == "scenes_classified"
    assert row["payload"] == {"foo": "bar"}
    assert row["published_at"] is None
