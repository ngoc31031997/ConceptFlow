"""Composition root for the TTS Service.

Revision (ADR-0014, ADR-0013): TTS Service is no longer a FastAPI/REST
app — it's a plain AMQP consumer (command synthesize_speech, queue
tts.commands) with a PostgreSQL-backed Outbox/Inbox, mirroring Content
Plugin Service's composition root shape.

Readiness is signaled via a sentinel file (Infrastructure Design) since
there's no HTTP endpoint left to serve a /health check.
"""

from __future__ import annotations

import asyncio
import logging
import os

import aio_pika

from adapters.messaging.consumer import SynthesizeSpeechCommandHandler
from adapters.messaging.producer import EVENTS_EXCHANGE, EVENTS_ROUTING_KEY
from adapters.persistence.db import create_pool
from adapters.persistence.inbox import InboxRepository
from adapters.persistence.outbox import OutboxRepository
from adapters.persistence.relay import OutboxRelay
from adapters.tts_engines import azure_adapter
from adapters.tts_engines.edge_adapter import EdgeTTSAdapter
from adapters.tts_engines.routing_engine import RoutingTTSEngine
from adapters.tts_engines.voice_registry import ENGINE_AZURE, ENGINE_GOOGLE
from adapters.tts_engines.voice_samples import generate_missing_samples
from application.synthesize_speech import SynthesizeSpeechUseCase
from application.synthesize_speech_batch import SynthesizeSpeechBatchUseCase
from domain.ports import TTSEnginePort

logging.basicConfig(level=logging.WARNING)
logger = logging.getLogger(__name__)

COMMANDS_QUEUE = "tts.commands"
RABBITMQ_URL = os.environ["RABBITMQ_URL"]
READY_SENTINEL_PATH = "/tmp/ready"


def build_engine() -> RoutingTTSEngine:
    """Edge for every voice; Azure and Google only when their credentials are
    present (ADR-0024, CR-011, ADR-0023).

    Edge is always constructed: it owns the default voices and is also where a
    failed metered call degrades to.
    """
    edge = EdgeTTSAdapter()
    metered: dict[str, TTSEnginePort] = {}

    if azure_adapter.is_configured():
        metered[ENGINE_AZURE] = azure_adapter.AzureTTSAdapter()
    else:
        logger.warning(
            "%s/%s are not set — the Azure voices in the catalogue will fall back "
            "to the equivalent Edge voice (CR-011).",
            azure_adapter.KEY_ENV_VAR,
            azure_adapter.REGION_ENV_VAR,
        )

    if os.environ.get("GOOGLE_APPLICATION_CREDENTIALS"):
        try:
            from adapters.tts_engines.google_adapter import GoogleTTSAdapter

            metered[ENGINE_GOOGLE] = GoogleTTSAdapter()
        except ImportError:
            # The library is optional so the image still builds without it.
            # Warn rather than fail — Edge covers every voice.
            logger.warning(
                "GOOGLE_APPLICATION_CREDENTIALS is set but google-cloud-texttospeech "
                "is not installed; the Google voices will use the Edge engine."
            )

    return RoutingTTSEngine(edge=edge, metered=metered)


async def run() -> None:
    engine = build_engine()
    batch_use_case = SynthesizeSpeechBatchUseCase(SynthesizeSpeechUseCase(engine))
    generate_missing_samples(engine)

    pool = await create_pool()
    inbox = InboxRepository(pool)
    outbox = OutboxRepository()

    connection = await aio_pika.connect_robust(RABBITMQ_URL)
    channel = await connection.channel()
    exchange = await channel.get_exchange(EVENTS_EXCHANGE)
    queue = await channel.get_queue(COMMANDS_QUEUE)

    def make_persistent_message(body: bytes) -> aio_pika.Message:
        return aio_pika.Message(body, delivery_mode=aio_pika.DeliveryMode.PERSISTENT)

    command_handler = SynthesizeSpeechCommandHandler(batch_use_case, pool, inbox, outbox)
    relay = OutboxRelay(pool, exchange, make_persistent_message, EVENTS_ROUTING_KEY)
    relay.start()

    consumer_tag = await queue.consume(command_handler.handle)

    with open(READY_SENTINEL_PATH, "w") as f:
        f.write("ready")
    logger.info("TTS Service ready — consuming '%s'", COMMANDS_QUEUE)

    try:
        await asyncio.Future()  # run forever
    finally:
        await queue.cancel(consumer_tag)
        await relay.stop()
        await connection.close()
        await pool.close()


if __name__ == "__main__":
    asyncio.run(run())
