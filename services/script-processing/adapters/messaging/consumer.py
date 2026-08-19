"""AMQP command consumer — handles parse_script from script_processing.commands
(interface-contracts.md).

Mirrors Content Plugin Service / TTS Service (ADR-0013): the consumer
never publishes to RabbitMQ directly — it only enqueues the event to the
Outbox, atomically with marking the message processed in the Inbox.
"""

from __future__ import annotations

import json
import logging
from typing import Protocol

import asyncpg

from adapters.logging.correlation import set_correlation_id
from adapters.messaging.producer import failure_envelope, success_envelope
from adapters.persistence.inbox import InboxRepository
from adapters.persistence.outbox import OutboxRepository
from application.parse_script import ParseScriptUseCase
from domain.errors import ScriptSyntaxError

logger = logging.getLogger(__name__)


class AckableMessage(Protocol):
    """Minimal surface of aio-pika's IncomingMessage we depend on."""

    body: bytes

    async def ack(self) -> None: ...


class ParseScriptCommandHandler:
    """Wires: inbox check -> parse -> outbox enqueue (+ inbox mark) -> ack."""

    def __init__(
        self,
        use_case: ParseScriptUseCase,
        pool: asyncpg.Pool,
        inbox: InboxRepository,
        outbox: OutboxRepository,
    ) -> None:
        self._use_case = use_case
        self._pool = pool
        self._inbox = inbox
        self._outbox = outbox

    async def handle(self, message: AckableMessage) -> None:
        envelope = json.loads(message.body)
        message_id = envelope["message_id"]
        saga_id = envelope["saga_id"]
        project_id = envelope["project_id"]
        set_correlation_id(saga_id)

        if await self._inbox.has_processed(message_id):
            logger.info("Skipping already-processed message_id=%s", message_id)
            await message.ack()
            return

        raw_script = envelope["payload"]["raw_script"]

        try:
            parsed = self._use_case.parse(raw_script)
        except ScriptSyntaxError as exc:
            logger.warning(
                "parse_script failed for project_id=%s: line=%s reason=%s",
                project_id,
                exc.line_number,
                exc.reason,
            )
            event_type = "parse_failed"
            out_envelope = failure_envelope(saga_id, project_id, exc.line_number, exc.reason)
        else:
            event_type = "script_parsed"
            out_envelope = success_envelope(saga_id, project_id, parsed.scenes)

        async with self._pool.acquire() as conn, conn.transaction():
            await self._outbox.enqueue(
                conn, aggregate_id=project_id, event_type=event_type, envelope=out_envelope
            )
            await self._inbox.mark_processed(conn, message_id)

        await message.ack()
