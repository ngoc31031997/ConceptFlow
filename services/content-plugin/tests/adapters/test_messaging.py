"""Unit tests for the messaging adapter: consumer + producer + idempotency.

Uses fakes for the AMQP surface (AckableMessage / PublishableExchange)
so no real RabbitMQ connection is needed.
"""

import json

import pytest

from adapters.messaging.consumer import ClassifySceneCommandHandler
from adapters.messaging.idempotency import IdempotencyStore
from adapters.messaging.producer import ScenesClassifiedEventPublisher
from application.classify_scene import ClassifyScenesBatchUseCase, ClassifySceneUseCase
from domain.models import ClassificationResult, Scene
from domain.ports import ContentPluginPort, ContentPluginRegistryPort


class FakeProgrammingPlugin(ContentPluginPort):
    @property
    def plugin_id(self) -> str:
        return "programming"

    @property
    def name(self) -> str:
        return "Lập trình"

    @property
    def supported_categories(self) -> tuple[str, ...]:
        return ("algorithm", "concept")

    def classify(self, scene: Scene) -> ClassificationResult:
        return ClassificationResult(scene.scene_index, scene.category_hint, f"{scene.category_hint}_template")


class FakeRegistry(ContentPluginRegistryPort):
    def get(self, plugin_id):
        return FakeProgrammingPlugin() if plugin_id == "programming" else None

    def list_all(self):
        return [FakeProgrammingPlugin()]


class FakeExchange:
    def __init__(self) -> None:
        self.published: list[tuple[bytes, str]] = []

    async def publish(self, message, routing_key: str) -> None:
        self.published.append((message, routing_key))


class FakeMessage:
    def __init__(self, body: bytes) -> None:
        self.body = body
        self.acked = False

    async def ack(self) -> None:
        self.acked = True


def make_envelope(
    message_id: str = "msg-1",
    plugin_id: str = "programming",
    scenes: list | None = None,
) -> bytes:
    scenes = scenes or [{"scene_index": 0, "narration_text": "text", "category_hint": "algorithm"}]
    envelope = {
        "message_id": message_id,
        "saga_id": "saga-1",
        "project_id": "project-1",
        "schema_version": "1.0",
        "timestamp": "2026-08-05T00:00:00Z",
        "payload": {"plugin_id": plugin_id, "scenes": scenes},
    }
    return json.dumps(envelope).encode("utf-8")


@pytest.fixture
def handler() -> tuple[ClassifySceneCommandHandler, FakeExchange, IdempotencyStore]:
    registry = FakeRegistry()
    batch_use_case = ClassifyScenesBatchUseCase(ClassifySceneUseCase(registry))
    exchange = FakeExchange()
    publisher = ScenesClassifiedEventPublisher(exchange, make_message=lambda body: body)
    idempotency_store = IdempotencyStore()
    command_handler = ClassifySceneCommandHandler(batch_use_case, publisher, idempotency_store)
    return command_handler, exchange, idempotency_store


@pytest.mark.asyncio
async def test_publishes_success_event_and_acks(handler) -> None:
    command_handler, exchange, _ = handler
    message = FakeMessage(make_envelope())

    await command_handler.handle(message)

    assert message.acked is True
    assert len(exchange.published) == 1
    body, routing_key = exchange.published[0]
    event = json.loads(body)
    assert routing_key == "orchestrator"
    assert event["payload"]["event_type"] == "scenes_classified"


@pytest.mark.asyncio
async def test_publishes_failure_event_for_unknown_plugin(handler) -> None:
    command_handler, exchange, _ = handler
    message = FakeMessage(make_envelope(plugin_id="unknown_plugin"))

    await command_handler.handle(message)

    assert message.acked is True
    event = json.loads(exchange.published[0][0])
    assert event["payload"]["event_type"] == "classification_failed"


@pytest.mark.asyncio
async def test_skips_reprocessing_duplicate_message_id(handler) -> None:
    command_handler, exchange, idempotency_store = handler
    idempotency_store.mark_processed("msg-1")
    message = FakeMessage(make_envelope(message_id="msg-1"))

    await command_handler.handle(message)

    assert message.acked is True
    assert len(exchange.published) == 0  # no event re-published
