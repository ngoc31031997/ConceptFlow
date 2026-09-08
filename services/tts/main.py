"""Composition root for the TTS Service.

Revision (ADR-0014, ADR-0013): TTS Service is no longer a FastAPI/REST
app — it's a plain AMQP consumer (command synthesize_speech, queue
tts.commands) with a PostgreSQL-backed Outbox/Inbox, mirroring Content
Plugin Service's composition root shape.

Voice models are still loaded once at startup (PiperTTSAdapter
construction) and kept in memory for the lifetime of the process.
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
from adapters.tts_engines.piper_adapter import PiperTTSAdapter
from adapters.tts_engines.routing_engine import RoutingTTSEngine
from adapters.tts_engines.voice_registry import ENGINE_PIPER, VOICES, voices_for_engine
from adapters.tts_engines.voice_samples import generate_missing_samples
from application.synthesize_speech import SynthesizeSpeechUseCase
from application.synthesize_speech_batch import SynthesizeSpeechBatchUseCase

logging.basicConfig(level=logging.WARNING)
logger = logging.getLogger(__name__)

COMMANDS_QUEUE = "tts.commands"
RABBITMQ_URL = os.environ["RABBITMQ_URL"]
READY_SENTINEL_PATH = "/tmp/ready"


def build_engine() -> RoutingTTSEngine:
    """Google when credentials are present, Piper otherwise (ADR-0023).

    Piper is always constructed: it is the fallback path, so the service must
    be able to speak even with no network or credentials at all.
    """
    piper = PiperTTSAdapter(
        voice_ids=[voice.voice_id for voice in voices_for_engine(ENGINE_PIPER)]
    )

    google = None
    if os.environ.get("GOOGLE_APPLICATION_CREDENTIALS"):
        try:
            from adapters.tts_engines.google_adapter import GoogleTTSAdapter

            google = GoogleTTSAdapter()
        except ImportError:
            # The library is optional so the image still builds and runs
            # offline-only. Warn rather than fail — Piper covers it.
            logger.warning(
                "GOOGLE_APPLICATION_CREDENTIALS is set but google-cloud-texttospeech "
                "is not installed; every voice will use the offline engine."
            )
    else:
        logger.warning(
            "GOOGLE_APPLICATION_CREDENTIALS is not set — narration will use the "
            "offline Piper voices, which are noticeably lower quality and are the "
            "largest monetization risk in this pipeline (ADR-0023)."
        )

    return RoutingTTSEngine(piper=piper, google=google)


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
