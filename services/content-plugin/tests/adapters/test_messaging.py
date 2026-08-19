"""Unit tests for the messaging adapter: consumer + Inbox/Outbox (ADR-0013).

Uses fakes for the AMQP surface (AckableMessage) and a FakePool standing
in for asyncpg (tests/adapters/fake_postgres.py) so no real RabbitMQ or
PostgreSQL connection is needed.
"""

import json

import pytest

from adapters.messaging.consumer import ClassifySceneCommandHandler
from adapters.persistence.inbox import InboxRepository
from adapters.persistence.outbox import OutboxRepository
from application.classify_scene import ClassifyScenesBatchUseCase, ClassifySceneUseCase
from domain.models import ClassificationResult, Scene
from domain.ports import ContentPluginPort, ContentPluginRegistryPort
from tests.adapters.fake_postgres import FakePool


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
def handler() -> tuple[ClassifySceneCommandHandler, FakePool]:
    registry = FakeRegistry()
    batch_use_case = ClassifyScenesBatchUseCase(ClassifySceneUseCase(registry))
    pool = FakePool()
    inbox = InboxRepository(pool)
    outbox = OutboxRepository()
    command_handler = ClassifySceneCommandHandler(batch_use_case, pool, inbox, outbox)
    return command_handler, pool


@pytest.mark.asyncio
async def test_enqueues_success_event_to_outbox_and_acks(handler) -> None:
    command_handler, pool = handler
    message = FakeMessage(make_envelope())

    await command_handler.handle(message)

    assert message.acked is True
    assert len(pool.store.outbox_events) == 1
    event = next(iter(pool.store.outbox_events.values()))
    assert event["event_type"] == "scenes_classified"
    assert event["payload"]["payload"]["event_type"] == "scenes_classified"
    assert event["published_at"] is None  # OutboxRelay hasn't run yet


@pytest.mark.asyncio
async def test_enqueues_failure_event_for_unknown_plugin(handler) -> None:
    command_handler, pool = handler
    message = FakeMessage(make_envelope(plugin_id="unknown_plugin"))

    await command_handler.handle(message)

    assert message.acked is True
    event = next(iter(pool.store.outbox_events.values()))
    assert event["event_type"] == "classification_failed"


@pytest.mark.asyncio
async def test_marks_message_processed_in_inbox(handler) -> None:
    command_handler, pool = handler
    message = FakeMessage(make_envelope(message_id="msg-1"))

    await command_handler.handle(message)

    assert "msg-1" in pool.store.processed_message_ids


@pytest.mark.asyncio
async def test_skips_reprocessing_duplicate_message_id(handler) -> None:
    command_handler, pool = handler
    pool.store.processed_message_ids.add("msg-1")
    message = FakeMessage(make_envelope(message_id="msg-1"))

    await command_handler.handle(message)

    assert message.acked is True
    assert len(pool.store.outbox_events) == 0  # no event re-enqueued
