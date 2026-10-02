"""Composition root for the Video Assembly Service.

Plain AMQP consumer with a PostgreSQL-backed Outbox/Inbox, same composition
root shape as the TTS and Rendering services (ADR-0013). No REST endpoint —
readiness is signaled via a sentinel file (Infrastructure Design).
"""

from __future__ import annotations

import asyncio
import logging
import os
import signal

import aio_pika

from adapters.assembly.ffmpeg_assembler import DEFAULT_ASSEMBLY_TIMEOUT_SECONDS, FfmpegVideoAssembler
from adapters.messaging.cancellation import CancelAwareOutbox, listen_for_cancels
from adapters.messaging.consumer import (
    AssembleVideoCommandHandler,
    NormalizeChannelAssetCommandHandler,
    QCVideoCommandHandler,
    RegisterChannelAssetCommandHandler,
    VideoAssemblyCommandDispatcher,
)
from adapters.messaging.producer import EVENTS_EXCHANGE, EVENTS_ROUTING_KEY
from adapters.messaging.progress import PROGRESS_EXCHANGE, ProgressPublisher
from adapters.messaging.purge import PurgeProjectArtifactsCommandHandler
from adapters.persistence.channel_assets import ChannelAssetsRepository
from adapters.persistence.db import create_pool
from adapters.persistence.inbox import InboxRepository
from adapters.persistence.outbox import OutboxRepository
from adapters.persistence.relay import OutboxRelay
from adapters.storage.artifact_paths import purge_project_artifacts
from application.assemble_video import AssembleVideoUseCase
from domain.qc_rules import QCThresholds

logging.basicConfig(level=logging.WARNING)
logger = logging.getLogger(__name__)

COMMANDS_QUEUE = "video_assembly.commands"
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
    outbox = CancelAwareOutbox(OutboxRepository())
    channel_assets = ChannelAssetsRepository(pool)

    connection = await aio_pika.connect_robust(RABBITMQ_URL)
    channel = await connection.channel()
    exchange = await channel.get_exchange(EVENTS_EXCHANGE)
    progress_exchange = await channel.get_exchange(PROGRESS_EXCHANGE)
    commands_queue = await channel.get_queue(COMMANDS_QUEUE)

    def make_persistent_message(body: bytes) -> aio_pika.Message:
        return aio_pika.Message(body, delivery_mode=aio_pika.DeliveryMode.PERSISTENT)

    # assemble_video runs its ffmpeg work in a worker
    # thread (asyncio.to_thread) — ProgressPublisher needs the running loop
    # itself to marshal a publish back onto it from that thread.
    progress = ProgressPublisher(progress_exchange, asyncio.get_running_loop())

    assemble_video_handler = AssembleVideoCommandHandler(
        use_case, pool, inbox, outbox, channel_assets, progress
    )
    normalize_handler = NormalizeChannelAssetCommandHandler(pool, channel_assets, inbox, outbox)
    # Thresholds are read from the environment once, here, and
    # nowhere else. QC_ENFORCE is deliberately NOT among them — this service
    # scores and reports the real severity; whether a blocking finding actually
    # stops a publish is Orchestrator's call.
    qc_handler = QCVideoCommandHandler(pool, inbox, outbox, QCThresholds.from_env())
    register_channel_asset_handler = RegisterChannelAssetCommandHandler(pool, channel_assets, inbox, outbox)
    command_dispatcher = VideoAssemblyCommandDispatcher(
        assemble_video_handler, normalize_handler, qc_handler,
        register_channel_asset_handler,
        # Dọn final.mp4/final.srt của project bị xoá.
        purge_project_artifacts=PurgeProjectArtifactsCommandHandler(
            purge_project_artifacts, pool, inbox, outbox
        ),
    )

    relay = OutboxRelay(pool, exchange, make_persistent_message, EVENTS_ROUTING_KEY)
    relay.start()

    commands_consumer_tag = await commands_queue.consume(command_dispatcher.handle)
    # Cancel requests arrive on their own fanout, not behind the running assembly.
    await listen_for_cancels(channel)

    with open(READY_SENTINEL_PATH, "w") as f:
        f.write("ready")
    logger.info("Video Assembly Service ready — consuming '%s'", COMMANDS_QUEUE)

    # SIGTERM's default disposition terminates the process immediately,
    # unlike SIGINT (KeyboardInterrupt) — without a handler, `docker stop`
    # kills us before the `finally` below ever runs, and the AMQP connection
    # / DB pool are torn down uncleanly instead of closed.
    stop_event = asyncio.Event()
    loop = asyncio.get_running_loop()
    for sig in (signal.SIGTERM, signal.SIGINT):
        loop.add_signal_handler(sig, stop_event.set)

    try:
        await stop_event.wait()
        logger.info("Shutdown signal received — draining Video Assembly Service")
    finally:
        await commands_queue.cancel(commands_consumer_tag)
        await relay.stop()
        await connection.close()
        await pool.close()


if __name__ == "__main__":
    asyncio.run(run())
