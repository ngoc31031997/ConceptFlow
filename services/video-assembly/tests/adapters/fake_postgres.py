"""In-memory fakes standing in for an asyncpg Pool/Connection in unit
tests, so the messaging/persistence layer can be tested without a real
PostgreSQL instance. Mirrors just the surface consumer.py, relay.py, and
adapters/persistence/*.py actually use.
"""

from __future__ import annotations

import itertools
import json
from contextlib import asynccontextmanager


class FakeTransaction:
    async def __aenter__(self) -> FakeTransaction:
        return self

    async def __aexit__(self, *exc_info: object) -> None:
        return None


class FakeConnection:
    def __init__(self, store: FakePostgresStore) -> None:
        self._store = store

    def transaction(self) -> FakeTransaction:
        return FakeTransaction()

    async def execute(self, query: str, *args: object) -> str:
        normalized = " ".join(query.split())
        if normalized.startswith("INSERT INTO outbox_events"):
            aggregate_id, event_type, payload_json = args
            row_id = next(self._store.outbox_id_seq)
            self._store.outbox_events[row_id] = {
                "id": row_id,
                "aggregate_id": aggregate_id,
                "event_type": event_type,
                "payload": json.loads(payload_json),
                "published_at": None,
            }
        elif normalized.startswith("INSERT INTO processed_messages"):
            (message_id,) = args
            self._store.processed_message_ids.add(message_id)
        elif normalized.startswith("UPDATE outbox_events SET published_at"):
            (row_id,) = args
            self._store.outbox_events[row_id]["published_at"] = "now"
        else:
            raise NotImplementedError(query)
        return "OK"

    async def fetchrow(self, query: str, *args: object) -> dict | None:
        normalized = " ".join(query.split())
        if normalized.startswith("SELECT 1 FROM processed_messages"):
            (message_id,) = args
            return {"?column?": 1} if message_id in self._store.processed_message_ids else None
        raise NotImplementedError(query)

    async def fetch(self, query: str, *args: object) -> list[dict]:
        normalized = " ".join(query.split())
        if normalized.startswith("SELECT id, payload FROM outbox_events"):
            unpublished = [row for row in self._store.outbox_events.values() if row["published_at"] is None]
            unpublished.sort(key=lambda row: row["id"])
            return [{"id": row["id"], "payload": json.dumps(row["payload"])} for row in unpublished]
        raise NotImplementedError(query)


class FakePostgresStore:
    """Backs FakePool — holds the tables' state so tests can assert on it."""

    def __init__(self) -> None:
        self.outbox_events: dict[int, dict] = {}
        self.outbox_id_seq = itertools.count(1)
        self.processed_message_ids: set[str] = set()


class FakePool:
    def __init__(self) -> None:
        self.store = FakePostgresStore()

    @asynccontextmanager
    async def acquire(self):
        yield FakeConnection(self.store)
