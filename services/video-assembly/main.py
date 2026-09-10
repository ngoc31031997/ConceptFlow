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
from adapters.messaging.consumer import (
    AssembleVideoCommandHandler,
    ChannelAssetRenderedEventHandler,
    NormalizeChannelAssetCommandHandler,
    VideoAssemblyCommandDispatcher,
)
from adapters.messaging.producer import EVENTS_EXCHANGE, EVENTS_ROUTING_KEY
from adapters.persistence.channel_assets import ChannelAssetsRepository
from adapters.persistence.db import create_pool
from adapters.persistence.inbox import InboxRepository
from adapters.persistence.outbox import OutboxRepository
from adapters.persistence.relay import OutboxRelay
from application.assemble_video import AssembleVideoUseCase

logging.basicConfig(level=logging.WARNING)
logger = logging.getLogger(__name__)

COMMANDS_QUEUE = "video_assembly.commands"
# CR-023 correction: rendering.events' channel_asset_rendered fans out here
# via events.direct/"orchestrator" (infra/rabbitmq/definitions.json) — this
# queue exists solely so video-assembly, not just Orchestrator, gets a copy.
CHANNEL_ASSET_EVENTS_QUEUE = "video_assembly.channel_asset_events"
RABBITMQ_URL = os.environ["RABBITMQ_URL"]
READY_SENTINEL_PATH = "/tmp/ready"


async def run() -> None:
    timeout_seconds = int(os.environ.get("ASSEMBLY_TIMEOUT_SECONDS", DEFAULT_ASSEMBLY_TIMEOUT_SECONDS))
    assembler = FfmpegVideoAssembler(
        timeout_seconds=timeout_seconds,
        lead_in_seconds=float(os.environ.get("ASSEMBLY_LEAD_IN_SECONDS", 0.0)),
        tail_seconds=float(os.environ.get("ASSEMBLY_TAIL_SECONDS", 0.0)),
    )
    use_case = AssembleVideoUseCase(assembler)

    pool = await create_pool()
    inbox = InboxRepository(pool)
    outbox = OutboxRepository()
    channel_assets = ChannelAssetsRepository(pool)

    connection = await aio_pika.connect_robust(RABBITMQ_URL)
    channel = await connection.channel()
    exchange = await channel.get_exchange(EVENTS_EXCHANGE)
    commands_queue = await channel.get_queue(COMMANDS_QUEUE)
    channel_asset_events_queue = await channel.get_queue(CHANNEL_ASSET_EVENTS_QUEUE)

    def make_persistent_message(body: bytes) -> aio_pika.Message:
        return aio_pika.Message(body, delivery_mode=aio_pika.DeliveryMode.PERSISTENT)

    assemble_video_handler = AssembleVideoCommandHandler(use_case, pool, inbox, outbox, channel_assets)
    normalize_handler = NormalizeChannelAssetCommandHandler(pool, channel_assets, inbox, outbox)
    command_dispatcher = VideoAssemblyCommandDispatcher(assemble_video_handler, normalize_handler)
    channel_asset_rendered_handler = ChannelAssetRenderedEventHandler(pool, channel_assets, inbox, outbox)

    relay = OutboxRelay(pool, exchange, make_persistent_message, EVENTS_ROUTING_KEY)
    relay.start()

    commands_consumer_tag = await commands_queue.consume(command_dispatcher.handle)
    channel_asset_events_consumer_tag = await channel_asset_events_queue.consume(
        channel_asset_rendered_handler.handle
    )

    with open(READY_SENTINEL_PATH, "w") as f:
        f.write("ready")
    logger.info(
        "Video Assembly Service ready — consuming '%s' and '%s'",
        COMMANDS_QUEUE,
        CHANNEL_ASSET_EVENTS_QUEUE,
    )

    try:
        await asyncio.Future()  # run forever
    finally:
        await commands_queue.cancel(commands_consumer_tag)
        await channel_asset_events_queue.cancel(channel_asset_events_consumer_tag)
        await relay.stop()
        await connection.close()
        await pool.close()


if __name__ == "__main__":
    asyncio.run(run())
