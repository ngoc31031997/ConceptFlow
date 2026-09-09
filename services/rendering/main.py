"""Composition root for the Rendering Service.

Plain AMQP consumer with a PostgreSQL-backed Outbox/Inbox, mirroring
Content Plugin Service / TTS Service / Script Processing Service's
composition root shape (ADR-0013). No REST endpoint — readiness is
signaled via a sentinel file (Infrastructure Design).
"""

from __future__ import annotations

import asyncio
import logging
import os

import aio_pika

from adapters.messaging.consumer import (
    RenderingCommandDispatcher,
    RenderScriptCommandHandler,
    ValidateScriptCommandHandler,
)
from adapters.messaging.producer import EVENTS_EXCHANGE, EVENTS_ROUTING_KEY
from adapters.messaging.progress import PROGRESS_EXCHANGE, ProgressPublisher
from adapters.persistence.db import create_pool
from adapters.persistence.inbox import InboxRepository
from adapters.persistence.outbox import OutboxRepository
from adapters.persistence.relay import OutboxRelay
from adapters.rendering.manim_renderer import (
    CACHE_ROOT,
    DEFAULT_RENDER_MEMORY_LIMIT_GB,
    DEFAULT_RENDER_QUALITY,
    DEFAULT_RENDER_TIMEOUT_SECONDS,
    ManimScriptRenderer,
)
from application.render_script import RenderScriptUseCase
from application.validate_script import ValidateScriptUseCase

logging.basicConfig(level=logging.WARNING)
logger = logging.getLogger(__name__)

COMMANDS_QUEUE = "rendering.commands"
RABBITMQ_URL = os.environ["RABBITMQ_URL"]
READY_SENTINEL_PATH = "/tmp/ready"


async def run() -> None:
    timeout_seconds = int(os.environ.get("RENDER_TIMEOUT_SECONDS", DEFAULT_RENDER_TIMEOUT_SECONDS))
    memory_limit_gb = int(os.environ.get("RENDER_MEMORY_LIMIT_GB", DEFAULT_RENDER_MEMORY_LIMIT_GB))
    # RENDER_CACHE_ROOT="" turns caching off, restoring the old
    # tempdir + --disable_caching behaviour without a code change.
    cache_root = os.environ.get("RENDER_CACHE_ROOT", CACHE_ROOT) or None
    renderer = ManimScriptRenderer(
        timeout_seconds=timeout_seconds,
        memory_limit_gb=memory_limit_gb,
        cache_root=cache_root,
        quality=os.environ.get("RENDER_QUALITY", DEFAULT_RENDER_QUALITY),
    )
    use_case = RenderScriptUseCase(renderer)

    pool = await create_pool()
    inbox = InboxRepository(pool)
    outbox = OutboxRepository()

    connection = await aio_pika.connect_robust(RABBITMQ_URL)
    channel = await connection.channel()
    exchange = await channel.get_exchange(EVENTS_EXCHANGE)
    progress_exchange = await channel.get_exchange(PROGRESS_EXCHANGE)
    queue = await channel.get_queue(COMMANDS_QUEUE)

    def make_persistent_message(body: bytes) -> aio_pika.Message:
        return aio_pika.Message(body, delivery_mode=aio_pika.DeliveryMode.PERSISTENT)

    command_handler = RenderingCommandDispatcher(
        # Cổng kiểm tra chạy trước TTS (CR-020): script sai bị chặn trước khi
        # tiêu quota giọng đọc.
        ValidateScriptCommandHandler(ValidateScriptUseCase(renderer), pool, inbox, outbox),
        RenderScriptCommandHandler(
            use_case, pool, inbox, outbox, ProgressPublisher(progress_exchange)
        ),
    )
    relay = OutboxRelay(pool, exchange, make_persistent_message, EVENTS_ROUTING_KEY)
    relay.start()

    consumer_tag = await queue.consume(command_handler.handle)

    with open(READY_SENTINEL_PATH, "w") as f:
        f.write("ready")
    logger.info("Rendering Service ready — consuming '%s'", COMMANDS_QUEUE)

    try:
        await asyncio.Future()  # run forever
    finally:
        await queue.cancel(consumer_tag)
        await relay.stop()
        await connection.close()
        await pool.close()


if __name__ == "__main__":
    asyncio.run(run())
