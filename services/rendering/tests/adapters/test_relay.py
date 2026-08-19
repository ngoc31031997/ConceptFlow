"""Unit tests for OutboxRelay (ADR-0013)."""

from __future__ import annotations

import json

import pytest

from adapters.persistence.relay import OutboxRelay
from tests.adapters.fake_postgres import FakePool


class FakeExchange:
    def __init__(self) -> None:
        self.published: list[tuple[bytes, str]] = []

    async def publish(self, message: object, routing_key: str) -> None:
        self.published.append((message, routing_key))


@pytest.fixture
def relay_setup():
    pool = FakePool()
    exchange = FakeExchange()
    relay = OutboxRelay(pool, exchange, make_message=lambda body: body, routing_key="orchestrator")
    return pool, exchange, relay


@pytest.mark.asyncio
async def test_relay_once_publishes_unpublished_rows_and_marks_them(relay_setup) -> None:
    pool, exchange, relay = relay_setup
    async with pool.acquire() as conn:
        await conn.execute(
            "INSERT INTO outbox_events (aggregate_id, event_type, payload) VALUES ($1, $2, $3::jsonb)",
            "project-1",
            "scenes_classified",
            json.dumps({"hello": "world"}),
        )

    relayed_count = await relay.relay_once()

    assert relayed_count == 1
    assert len(exchange.published) == 1
    body, routing_key = exchange.published[0]
    assert json.loads(body) == {"hello": "world"}
    assert routing_key == "orchestrator"
    row = next(iter(pool.store.outbox_events.values()))
    assert row["published_at"] == "now"


@pytest.mark.asyncio
async def test_relay_once_is_noop_when_nothing_to_publish(relay_setup) -> None:
    pool, exchange, relay = relay_setup

    relayed_count = await relay.relay_once()

    assert relayed_count == 0
    assert exchange.published == []


@pytest.mark.asyncio
async def test_relay_once_skips_already_published_rows(relay_setup) -> None:
    pool, exchange, relay = relay_setup
    async with pool.acquire() as conn:
        await conn.execute(
            "INSERT INTO outbox_events (aggregate_id, event_type, payload) VALUES ($1, $2, $3::jsonb)",
            "project-1",
            "scenes_classified",
            json.dumps({"hello": "world"}),
        )
    await relay.relay_once()  # publishes + marks it

    relayed_count = await relay.relay_once()  # second poll

    assert relayed_count == 0
    assert len(exchange.published) == 1
