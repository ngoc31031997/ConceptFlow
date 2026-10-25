import json
from datetime import UTC, datetime

import httpx
import pytest

from app.calllog import CallFilter, CallRecord, ModelStats


@pytest.fixture(autouse=True)
def _database_url(monkeypatch):
    monkeypatch.setenv("LLM_DATABASE_URL", "postgresql://test/test")


class MemoryCallLog:
    """CallLogPort kept in a list, so the API and provider tests run without a database."""

    def __init__(self) -> None:
        self.rows: list[CallRecord] = []
        self.fail = False

    async def record(self, rec: CallRecord) -> None:
        if self.fail:
            raise ConnectionError("the call log is down")
        rec.created_at = rec.created_at or datetime.now(UTC)
        self.rows.append(rec)

    async def list(self, flt: CallFilter) -> list[CallRecord]:
        rows = [
            r for r in self.rows
            if (not flt.project_id or r.project_id == flt.project_id)
            and (not flt.role or r.role == flt.role)
            and (flt.ok is None or r.ok == flt.ok)
            and (flt.before is None or r.created_at < flt.before)
        ]
        return sorted(rows, key=lambda r: r.created_at, reverse=True)[: flt.limit]

    async def stats(self, step: str, phase: str, since: datetime) -> list[ModelStats]:
        out: dict[str, ModelStats] = {}
        for r in self.rows:
            if (r.step, r.phase) != (step, phase) or r.created_at < since:
                continue
            m = out.setdefault(r.model, ModelStats(r.model))
            m.calls += 1
            if r.ok:
                m.ok += 1
                m.avg_completion_tokens += r.completion_tokens
                m.avg_duration_ms += r.duration_ms
            else:
                m.failures[r.error_kind] = m.failures.get(r.error_kind, 0) + 1
        for m in out.values():
            if m.ok:
                m.avg_completion_tokens //= m.ok
                m.avg_duration_ms //= m.ok
        return list(out.values())

    async def delete_ids(self, ids: list[str]) -> int:
        return self._drop(lambda r: r.call_id in ids)

    async def delete_older_than(self, cutoff: datetime) -> int:
        return self._drop(lambda r: r.created_at < cutoff)

    async def delete_all(self) -> int:
        return self._drop(lambda r: True)

    def _drop(self, pick) -> int:
        kept = [r for r in self.rows if not pick(r)]
        n, self.rows = len(self.rows) - len(kept), kept
        return n


@pytest.fixture
def call_log() -> MemoryCallLog:
    return MemoryCallLog()


def sse(*payloads: dict | str) -> str:
    lines = []
    for p in payloads:
        lines.append("data: " + (p if isinstance(p, str) else json.dumps(p)) + "\n\n")
    return "".join(lines)


def chunk(content=None, reasoning=None, finish=None, usage=None, model="test-model"):
    delta = {}
    if content is not None:
        delta["content"] = content
    if reasoning is not None:
        delta["reasoning_content"] = reasoning
    body = {
        "id": "c1", "object": "chat.completion.chunk", "created": 1, "model": model,
        "choices": [{"index": 0, "delta": delta, "finish_reason": finish}],
    }
    if usage is not None:
        body["usage"] = usage
    return body


def usage_chunk(usage, model="test-model"):
    return {"id": "c1", "object": "chat.completion.chunk", "created": 1, "model": model,
            "choices": [], "usage": usage}


def stream_response(*payloads, status=200):
    return httpx.Response(status, headers={"content-type": "text/event-stream"}, text=sse(*payloads, "[DONE]"))


@pytest.fixture
def no_sleep():
    async def _sleep(_):
        return None
    return _sleep
