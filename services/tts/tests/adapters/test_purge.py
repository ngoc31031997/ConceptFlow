"""purge_project_artifacts (CR-040 FR114.2)."""

from __future__ import annotations

import json
import os

import pytest

from adapters.messaging.purge import PurgeProjectArtifactsCommandHandler
from adapters.persistence.inbox import InboxRepository
from adapters.persistence.outbox import OutboxRepository
from adapters.storage import artifact_paths
from tests.adapters.fake_postgres import FakePool


class _Msg:
    def __init__(self, body: bytes) -> None:
        self.body = body
        self.acked = False

    async def ack(self) -> None:
        self.acked = True


def _envelope(project_id="proj-1", message_id="m1") -> bytes:
    return json.dumps({"message_id": message_id, "saga_id": "s1", "project_id": project_id,
                       "event_type": "purge_project_artifacts", "payload": {}}).encode()


def _handler(purge):
    pool = FakePool()
    return PurgeProjectArtifactsCommandHandler(purge, pool, InboxRepository(pool), OutboxRepository()), pool


@pytest.mark.asyncio
async def test_purges_and_reports_success():
    calls = []
    handler, pool = _handler(calls.append)
    msg = _Msg(_envelope())

    await handler.handle(msg)

    assert calls == ["proj-1"] and msg.acked
    event = next(iter(pool.store.outbox_events.values()))
    assert event["event_type"] == "artifacts_purged"
    assert event["payload"]["payload"]["service"] == "tts"


@pytest.mark.asyncio
async def test_duplicate_message_is_purged_once():
    calls = []
    handler, pool = _handler(calls.append)

    await handler.handle(_Msg(_envelope()))
    await handler.handle(_Msg(_envelope()))

    assert calls == ["proj-1"] and len(pool.store.outbox_events) == 1


@pytest.mark.asyncio
async def test_failure_is_reported_not_raised():
    def boom(_):
        raise OSError("disk gone")

    handler, pool = _handler(boom)
    msg = _Msg(_envelope())

    await handler.handle(msg)

    assert msg.acked
    event = next(iter(pool.store.outbox_events.values()))
    assert event["event_type"] == "purge_failed"
    assert "disk gone" in event["payload"]["payload"]["error_message"]


@pytest.mark.parametrize("bad", ["", "..", "a/b", "."])
def test_unsafe_project_id_is_refused(bad):
    with pytest.raises(ValueError):
        artifact_paths.purge_project_artifacts(bad)


def test_purge_removes_only_audio(tmp_path, monkeypatch):
    monkeypatch.setattr(artifact_paths, "SHARED_VOLUME_ROOT", str(tmp_path))
    (tmp_path / "p" / "audio").mkdir(parents=True)
    (tmp_path / "p" / "audio" / "0.wav").write_bytes(b"x")
    (tmp_path / "p" / "video").mkdir()
    (tmp_path / "p" / "video" / "final.mp4").write_bytes(b"x")

    artifact_paths.purge_project_artifacts("p")
    artifact_paths.purge_project_artifacts("p")  # idempotent

    assert not (tmp_path / "p" / "audio").exists()
    assert (tmp_path / "p" / "video" / "final.mp4").exists()
