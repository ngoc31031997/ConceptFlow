"""Composition root for the Video Assembly Service.

Plain AMQP consumer with a PostgreSQL-backed Outbox/Inbox, mirroring
Content Plugin Service / TTS Service / Script Processing Service /
Rendering Service's composition root shape (ADR-0013). No REST endpoint —
readiness is signaled via a sentinel file (Infrastructure Design).
"""

from __future__ import annotations

import asyncio
import logging
import os

import aio_pika

from adapters.assembly.ffmpeg_assembler import DEFAULT_ASSEMBLY_TIMEOUT_SECONDS, FfmpegVideoAssembler
from adapters.assembly.ffprobe_inspector import MediaFormatInspector
from adapters.messaging.consumer import AssembleVideoCommandHandler
from adapters.messaging.producer import EVENTS_EXCHANGE, EVENTS_ROUTING_KEY
from adapters.persistence.db import create_pool
from adapters.persistence.inbox import InboxRepository
from adapters.persistence.outbox import OutboxRepository
from adapters.persistence.relay import OutboxRelay
from application.assemble_video import AssembleVideoUseCase

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

COMMANDS_QUEUE = "video_assembly.commands"
RABBITMQ_URL = os.environ["RABBITMQ_URL"]
READY_SENTINEL_PATH = "/tmp/ready"


async def run() -> None:
    inspector = MediaFormatInspector()
    timeout_seconds = int(os.environ.get("ASSEMBLY_TIMEOUT_SECONDS", DEFAULT_ASSEMBLY_TIMEOUT_SECONDS))
    assembler = FfmpegVideoAssembler(inspector, timeout_seconds=timeout_seconds)
    use_case = AssembleVideoUseCase(assembler)

    pool = await create_pool()
    inbox = InboxRepository(pool)
    outbox = OutboxRepository()

    connection = await aio_pika.connect_robust(RABBITMQ_URL)
    channel = await connection.channel()
    exchange = await channel.get_exchange(EVENTS_EXCHANGE)
    queue = await channel.get_queue(COMMANDS_QUEUE)

    def make_persistent_message(body: bytes) -> aio_pika.Message:
        return aio_pika.Message(body, delivery_mode=aio_pika.DeliveryMode.PERSISTENT)

    command_handler = AssembleVideoCommandHandler(use_case, pool, inbox, outbox)
    relay = OutboxRelay(pool, exchange, make_persistent_message, EVENTS_ROUTING_KEY)
    relay.start()

    consumer_tag = await queue.consume(command_handler.handle)

    with open(READY_SENTINEL_PATH, "w") as f:
        f.write("ready")
    logger.info("Video Assembly Service ready — consuming '%s'", COMMANDS_QUEUE)

    try:
        await asyncio.Future()  # run forever
    finally:
        await queue.cancel(consumer_tag)
        await relay.stop()
        await connection.close()
        await pool.close()


if __name__ == "__main__":
    asyncio.run(run())
