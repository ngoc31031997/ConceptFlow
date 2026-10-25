"""PgCallLog against a real Postgres. Runs only when LLM_TEST_DATABASE_URL points at a
throwaway database; the table is emptied before and after."""

import os
import uuid
from datetime import UTC, datetime, timedelta

import pytest

from app.calllog import CallFilter, CallRecord, PgCallLog

DSN = os.environ.get("LLM_TEST_DATABASE_URL", "")

pytestmark = pytest.mark.skipif(not DSN, reason="LLM_TEST_DATABASE_URL is not set")


@pytest.fixture
async def log():
    pg = PgCallLog(DSN)
    await pg.start()
    await pg.delete_all()
    yield pg
    await pg.delete_all()
    await pg.stop()


def row(**over) -> CallRecord:
    base = dict(call_id=str(uuid.uuid4()), provider="hive", model="m", ok=True, step="code", phase="chunk",
                project_id="p1", response="hello", completion_tokens=10, duration_ms=100)
    base.update(over)
    return CallRecord(**base)


async def test_a_row_round_trips(log):
    rec = row(role="beat_sheet", error_kind="", reasoning_chars=5, usage_reported=False)
    await log.record(rec)
    [got] = await log.list(CallFilter())
    assert got.call_id == rec.call_id and got.response == "hello" and got.role == "beat_sheet"
    assert got.project_id == "p1" and got.usage_reported is False and got.created_at is not None


async def test_a_call_without_a_project_reads_back_as_empty(log):
    await log.record(row(project_id=""))
    assert (await log.list(CallFilter()))[0].project_id == ""


async def test_list_filters_and_pages_newest_first(log):
    for i in range(3):
        await log.record(row(project_id=f"p{i}", ok=i != 1))
    assert [r.project_id for r in await log.list(CallFilter(limit=2))] == ["p2", "p1"]
    assert [r.project_id for r in await log.list(CallFilter(ok=False))] == ["p1"]
    page = await log.list(CallFilter(limit=1))
    assert [r.project_id for r in await log.list(CallFilter(before=page[0].created_at, limit=5))] == ["p1", "p0"]


async def test_stats_average_the_successful_calls_and_count_failures_by_kind(log):
    await log.record(row(completion_tokens=10, duration_ms=100))
    await log.record(row(completion_tokens=30, duration_ms=300))
    await log.record(row(ok=False, error_kind="truncated", completion_tokens=999, duration_ms=9999))
    await log.record(row(step="story", phase=""))
    since = datetime.now(UTC) - timedelta(days=30)
    [stat] = await log.stats("code", "chunk", since)
    assert (stat.calls, stat.ok, stat.failures) == (3, 2, {"truncated": 1})
    assert (stat.avg_completion_tokens, stat.avg_duration_ms) == (20, 200)
    assert await log.stats("code", "chunk", datetime.now(UTC) + timedelta(days=1)) == []


async def test_deletes(log):
    keep, drop = row(), row()
    await log.record(keep)
    await log.record(drop)
    assert await log.delete_ids([drop.call_id]) == 1
    assert await log.delete_older_than(datetime.now(UTC) + timedelta(days=1)) == 1
    await log.record(row())
    assert await log.delete_all() == 1
    assert await log.delete_ids([]) == 0
