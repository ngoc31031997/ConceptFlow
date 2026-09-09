"""Composition root for the Publisher Service.

Wires domain/application/adapters together (constructor injection, per
dependency-injection.md) and exposes both the FastAPI app (REST — OAuth
flow) and the AMQP consumer (messaging — publish_video) from one process.
"""

from __future__ import annotations

import logging
import os
from contextlib import asynccontextmanager

import aio_pika
from fastapi import FastAPI

from adapters.api.router import create_health_router, create_v1_router
from adapters.config.oauth_app_registry import FileOAuthAppRegistry
from adapters.messaging.consumer import PublishVideoCommandHandler
from adapters.messaging.producer import EVENTS_EXCHANGE, EVENTS_ROUTING_KEY
from adapters.persistence.credential_store import PostgresCredentialStore
from adapters.persistence.db import create_pool
from adapters.persistence.inbox import InboxRepository
from adapters.persistence.outbox import OutboxRepository
from adapters.persistence.relay import OutboxRelay
from adapters.youtube.oauth_flow import GoogleOAuthFlow
from adapters.youtube.oauth_state import NonceStore
from adapters.youtube.youtube_publisher import DEFAULT_UPLOAD_TIMEOUT_SECONDS, YouTubeVideoPublisher
from application.handle_oauth_callback import HandleOAuthCallbackUseCase
from application.publish_video import PublishVideoUseCase

logging.basicConfig(level=logging.WARNING)
logger = logging.getLogger(__name__)
# Root stays at WARNING so third-party libraries keep quiet, but this module's
# startup lines (which OAuth apps were found, consumer ready) are the first
# thing anyone checks when connecting a channel misbehaves — and a record that
# passes its own logger's level still reaches the root handler regardless of
# the root logger's level.
logger.setLevel(logging.INFO)

COMMANDS_QUEUE = "publisher.commands"
RABBITMQ_URL = os.environ["RABBITMQ_URL"]
DATABASE_URL = os.environ["DATABASE_URL"]


class ServiceState:
    """Tracks readiness for the /health endpoint: ready once the AMQP
    consumer + OutboxRelay are up (mirror Content Plugin Service)."""

    def __init__(self) -> None:
        self.ready = False

    def is_ready(self) -> bool:
        return self.ready


def create_app() -> FastAPI:
    state = ServiceState()

    @asynccontextmanager
    async def lifespan(app: FastAPI):
        credential_store = PostgresCredentialStore(DATABASE_URL)

        # Scanned once at startup — adding a client_secret file takes a
        # restart, which is the right trade for not stat-ing the filesystem
        # on every request (ADR-0026).
        app_registry = FileOAuthAppRegistry.from_environment()
        logger.info(
            "Loaded %d OAuth app(s): %s",
            len(app_registry.list()),
            ", ".join(app.label for app in app_registry.list()) or "none",
        )

        oauth_flow = GoogleOAuthFlow(redirect_uri=os.environ["GOOGLE_OAUTH_REDIRECT_URI"])
        nonce_store = NonceStore()
        handle_callback_use_case = HandleOAuthCallbackUseCase(
            oauth_flow, credential_store, app_registry
        )

        upload_timeout_seconds = int(
            os.environ.get("UPLOAD_TIMEOUT_SECONDS", DEFAULT_UPLOAD_TIMEOUT_SECONDS)
        )
        publisher = YouTubeVideoPublisher(
            app_registry=app_registry,
            credential_store=credential_store,
            timeout_seconds=upload_timeout_seconds,
        )
        publish_video_use_case = PublishVideoUseCase(publisher, credential_store)

        pool = await create_pool()
        inbox = InboxRepository(pool)
        outbox = OutboxRepository()

        connection = await aio_pika.connect_robust(RABBITMQ_URL)
        channel = await connection.channel()
        exchange = await channel.get_exchange(EVENTS_EXCHANGE)
        queue = await channel.get_queue(COMMANDS_QUEUE)

        def make_persistent_message(body: bytes) -> aio_pika.Message:
            return aio_pika.Message(body, delivery_mode=aio_pika.DeliveryMode.PERSISTENT)

        command_handler = PublishVideoCommandHandler(publish_video_use_case, pool, inbox, outbox)
        relay = OutboxRelay(pool, exchange, make_persistent_message, EVENTS_ROUTING_KEY)
        relay.start()

        consumer_tag = await queue.consume(command_handler.handle)
        state.ready = True
        logger.info("Publisher Service ready — consuming '%s'", COMMANDS_QUEUE)

        app.include_router(
            create_v1_router(
                oauth_flow,
                handle_callback_use_case,
                credential_store,
                app_registry,
                nonce_store,
            )
        )

        yield

        await queue.cancel(consumer_tag)
        await relay.stop()
        await connection.close()
        await pool.close()

    app = FastAPI(title="Publisher Service", lifespan=lifespan)
    app.include_router(create_health_router(state.is_ready))
    return app
