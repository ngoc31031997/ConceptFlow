"""OutboxRepository — enqueue events for OutboxRelay to publish (ADR-0013).

Writing here (instead of publishing to RabbitMQ directly from the
consumer) is what makes "process the command" and "record the event to
publish" atomic: both happen in one DB transaction alongside
InboxRepository.mark_processed(), so a crash between them is impossible.
"""

from __future__ import annotations

import json

from adapters.persistence.inbox import Connection


class OutboxRepository:
    async def enqueue(self, conn: Connection, *, aggregate_id: str, event_type: str, envelope: dict) -> None:
        await conn.execute(
            "INSERT INTO outbox_events (aggregate_id, event_type, payload) VALUES ($1, $2, $3::jsonb)",
            aggregate_id,
            event_type,
            json.dumps(envelope),
        )
