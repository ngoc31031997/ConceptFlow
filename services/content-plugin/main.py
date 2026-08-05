"""Composition root for the Content Plugin Service.

Wires domain/application/adapters together (constructor injection,
per dependency-injection.md) and exposes both the FastAPI app (REST)
and the AMQP consumer (messaging) from one process.
"""

from __future__ import annotations

import logging
import os
from contextlib import asynccontextmanager

import aio_pika
from fastapi import FastAPI

from adapters.api.router import create_health_router, create_v1_router
from adapters.messaging.consumer import ClassifySceneCommandHandler
from adapters.messaging.idempotency import IdempotencyStore
from adapters.messaging.producer import EVENTS_EXCHANGE, ScenesClassifiedEventPublisher
from adapters.plugins.registry import ContentPluginRegistry
from application.classify_scene import ClassifyScenesBatchUseCase, ClassifySceneUseCase
from application.list_plugins import ListPluginsUseCase

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

COMMANDS_QUEUE = "content_plugin.commands"
RABBITMQ_URL = os.environ["RABBITMQ_URL"]


class ServiceState:
    """Tracks readiness for the /health endpoint (Infrastructure Design,
    Question 3): ready only after plugin discovery has completed."""

    def __init__(self) -> None:
        self.plugins_discovered = False

    def is_ready(self) -> bool:
        return self.plugins_discovered


def create_app() -> FastAPI:
    state = ServiceState()

    @asynccontextmanager
    async def lifespan(app: FastAPI):
        registry = ContentPluginRegistry.discover()
        state.plugins_discovered = True

        connection = await aio_pika.connect_robust(RABBITMQ_URL)
        channel = await connection.channel()
        exchange = await channel.get_exchange(EVENTS_EXCHANGE)
        queue = await channel.get_queue(COMMANDS_QUEUE)

        def make_persistent_message(body: bytes) -> aio_pika.Message:
            return aio_pika.Message(body, delivery_mode=aio_pika.DeliveryMode.PERSISTENT)

        batch_use_case = ClassifyScenesBatchUseCase(ClassifySceneUseCase(registry))
        publisher = ScenesClassifiedEventPublisher(exchange, make_message=make_persistent_message)
        command_handler = ClassifySceneCommandHandler(batch_use_case, publisher, IdempotencyStore())

        consumer_tag = await queue.consume(command_handler.handle)
        logger.info("Content Plugin Service ready — consuming '%s'", COMMANDS_QUEUE)

        app.include_router(create_v1_router(ListPluginsUseCase(registry)))

        yield

        await queue.cancel(consumer_tag)
        await connection.close()

    app = FastAPI(title="Content Plugin Service", lifespan=lifespan)
    app.include_router(create_health_router(state.is_ready))
    return app
