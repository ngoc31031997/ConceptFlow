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
    RegisterChannelAssetCommandHandler,
    GenerateClipsCommandHandler,
    NormalizeChannelAssetCommandHandler,
    QCVideoCommandHandler,
    VideoAssemblyCommandDispatcher,
)
from adapters.messaging.cancellation import CancelAwareOutbox, listen_for_cancels
from adapters.messaging.producer import EVENTS_EXCHANGE, EVENTS_ROUTING_KEY
from adapters.messaging.progress import PROGRESS_EXCHANGE, ProgressPublisher
from adapters.persistence.channel_assets import ChannelAssetsRepository
from adapters.messaging.purge import PurgeProjectArtifactsCommandHandler
from adapters.persistence.db import create_pool
from adapters.storage.artifact_paths import purge_project_artifacts
from adapters.persistence.inbox import InboxRepository
from adapters.persistence.outbox import OutboxRepository
from adapters.persistence.relay import OutboxRelay
from application.assemble_video import AssembleVideoUseCase
from domain.clip_rules import ClipThresholds
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

    # CR-029: assemble_video/generate_clips run their ffmpeg work in a worker
    # thread (asyncio.to_thread) — ProgressPublisher needs the running loop
    # itself to marshal a publish back onto it from that thread.
    progress = ProgressPublisher(progress_exchange, asyncio.get_running_loop())

    assemble_video_handler = AssembleVideoCommandHandler(
        use_case, pool, inbox, outbox, channel_assets, progress
    )
    normalize_handler = NormalizeChannelAssetCommandHandler(pool, channel_assets, inbox, outbox)
    # CR-021 FR61.5: thresholds are read from the environment once, here, and
    # nowhere else. QC_ENFORCE is deliberately NOT among them — this service
    # scores and reports the real severity; whether a blocking finding actually
    # stops a publish is Orchestrator's call (LLD D5).
    qc_handler = QCVideoCommandHandler(pool, inbox, outbox, QCThresholds.from_env())
    # CR-007 D2/C2b: preset thresholds read from the environment once, here —
    # same convention as QC_ENFORCE-adjacent QCThresholds above.
    generate_clips_handler = GenerateClipsCommandHandler(
        pool, inbox, outbox, ClipThresholds.from_env(), progress
    )
    register_channel_asset_handler = RegisterChannelAssetCommandHandler(pool, channel_assets, inbox, outbox)
    command_dispatcher = VideoAssemblyCommandDispatcher(
        assemble_video_handler, normalize_handler, qc_handler, generate_clips_handler,
        register_channel_asset_handler,
        # CR-040 FR114.2: dọn final.mp4/final.srt/clips của project bị xoá.
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

    try:
        await asyncio.Future()  # run forever
    finally:
        await commands_queue.cancel(commands_consumer_tag)
        await relay.stop()
        await connection.close()
        await pool.close()


if __name__ == "__main__":
    asyncio.run(run())
