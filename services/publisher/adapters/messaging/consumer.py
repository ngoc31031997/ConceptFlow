"""AMQP command consumer — handles publish_video from publisher.commands
(interface-contracts.md).

PublishVideoUseCase.publish() can run for a while (YouTube upload, up to
UPLOAD_TIMEOUT_SECONDS). Calling it directly from this coroutine would
block the asyncio event loop for that duration — starving RabbitMQ
heartbeats, the OutboxRelay, and the OAuth REST routes. It's therefore
run via asyncio.to_thread() (mirror Unit 6's AssembleVideoCommandHandler).

Exactly one Outbox row per command (video_published or publish_failed) —
no per-scene/progress events, unlike Unit 5.
"""

from __future__ import annotations

import asyncio
import json
import logging
from typing import Protocol

import asyncpg

from adapters.logging.correlation import set_correlation_id
from adapters.messaging.producer import publish_failed_envelope, video_published_envelope
from adapters.persistence.inbox import InboxRepository
from adapters.persistence.outbox import OutboxRepository
from application.publish_video import PublishVideoUseCase
from domain.errors import InvalidPublishRequestError, MissingCredentialError, UploadError
from domain.models import PublishRequest

logger = logging.getLogger(__name__)


class AckableMessage(Protocol):
    """Minimal surface of aio-pika's IncomingMessage we depend on."""

    body: bytes

    async def ack(self) -> None: ...


class PublishVideoCommandHandler:
    def __init__(
        self,
        use_case: PublishVideoUseCase,
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

        payload = envelope["payload"]
        request = PublishRequest(
            project_id=project_id,
            video_path=payload["video_path"],
            title=payload["title"],
            description=payload.get("description"),
            tags=payload.get("tags") or [],
            visibility=payload.get("visibility", ""),
            publish_at=payload.get("publish_at"),
        )

        try:
            result = await asyncio.to_thread(self._use_case.publish, request)
        except (InvalidPublishRequestError, MissingCredentialError, UploadError) as exc:
            logger.warning("publish_video failed for project_id=%s: %s", project_id, exc)
            event_type = "publish_failed"
            out_envelope = publish_failed_envelope(saga_id, project_id, str(exc))
        else:
            event_type = "video_published"
            out_envelope = video_published_envelope(saga_id, project_id, result.youtube_video_url)

        async with self._pool.acquire() as conn, conn.transaction():
            await self._outbox.enqueue(
                conn, aggregate_id=project_id, event_type=event_type, envelope=out_envelope
            )
            await self._inbox.mark_processed(conn, message_id)

        await message.ack()
