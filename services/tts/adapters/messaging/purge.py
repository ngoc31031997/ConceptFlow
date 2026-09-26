"""purge_project_artifacts — TTS's half of the project-delete saga (CR-040 FR114.2).

Orchestrator sends this after it has refused to delete a project with a step
running and marked it `deleting`. This service removes only the files it owns
(see docs/contracts/shared-artifacts.md) and reports back, so the project row
goes away only once every writer has cleaned up. Idempotent: purging a project
that has nothing on disk still reports success.
"""

from __future__ import annotations

import asyncio
import json
import logging
from collections.abc import Callable

import asyncpg

from adapters.logging.correlation import set_correlation_id
from adapters.messaging.producer import build_envelope
from adapters.persistence.inbox import InboxRepository
from adapters.persistence.outbox import OutboxRepository

logger = logging.getLogger(__name__)

SERVICE_NAME = "tts"


class PurgeProjectArtifactsCommandHandler:
    def __init__(
        self,
        purge: Callable[[str], None],
        pool: asyncpg.Pool,
        inbox: InboxRepository,
        outbox: OutboxRepository,
    ) -> None:
        self._purge = purge
        self._pool = pool
        self._inbox = inbox
        self._outbox = outbox

    async def handle(self, message) -> None:
        envelope = json.loads(message.body)
        message_id = envelope["message_id"]
        saga_id = envelope["saga_id"]
        project_id = envelope["project_id"]
        set_correlation_id(saga_id)

        if await self._inbox.has_processed(message_id):
            await message.ack()
            return

        try:
            await asyncio.to_thread(self._purge, project_id)
        except (OSError, ValueError) as exc:
            logger.warning("purge_project_artifacts failed for project_id=%s: %s", project_id, exc)
            event_type = "purge_failed"
            payload = {"event_type": event_type, "service": SERVICE_NAME, "error_message": str(exc)}
        else:
            event_type = "artifacts_purged"
            payload = {"event_type": event_type, "service": SERVICE_NAME}

        async with self._pool.acquire() as conn, conn.transaction():
            await self._outbox.enqueue(
                conn,
                aggregate_id=project_id,
                event_type=event_type,
                envelope=build_envelope(saga_id, project_id, payload),
            )
            await self._inbox.mark_processed(conn, message_id)
        await message.ack()
