"""The log of every model call, in llm-service's own database.

One row per `Provider.chat`: what came back (never the prompt), what it cost and
how it ended. The same table feeds the code-step statistics and the Nhật ký
screen, so there is no second ledger to keep in step.
"""

from __future__ import annotations

import uuid
from dataclasses import dataclass, field
from datetime import datetime
from typing import Protocol

import asyncpg

SCHEMA = """
CREATE TABLE IF NOT EXISTS llm_calls (
    call_id           UUID PRIMARY KEY,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    provider          TEXT NOT NULL,
    model             TEXT NOT NULL,
    role              TEXT NOT NULL DEFAULT '',
    step              TEXT NOT NULL DEFAULT '',
    phase             TEXT NOT NULL DEFAULT '',
    project_id        TEXT,
    response          TEXT NOT NULL DEFAULT '',
    ok                BOOLEAN NOT NULL,
    error_kind        TEXT NOT NULL DEFAULT '',
    error_message     TEXT NOT NULL DEFAULT '',
    prompt_tokens     INTEGER NOT NULL DEFAULT 0,
    completion_tokens INTEGER NOT NULL DEFAULT 0,
    reasoning_tokens  INTEGER NOT NULL DEFAULT 0,
    cached_tokens     INTEGER NOT NULL DEFAULT 0,
    reasoning_chars   INTEGER NOT NULL DEFAULT 0,
    usage_reported    BOOLEAN NOT NULL DEFAULT true,
    duration_ms       INTEGER NOT NULL DEFAULT 0
);
ALTER TABLE llm_calls ALTER COLUMN response SET COMPRESSION lz4;
CREATE INDEX IF NOT EXISTS llm_calls_created_at_idx ON llm_calls (created_at DESC);
CREATE INDEX IF NOT EXISTS llm_calls_step_phase_idx ON llm_calls (step, phase, created_at);
CREATE INDEX IF NOT EXISTS llm_calls_project_idx ON llm_calls (project_id);
"""

_COLUMNS = (
    "call_id, created_at, provider, model, role, step, phase, project_id, response, ok, error_kind, "
    "error_message, prompt_tokens, completion_tokens, reasoning_tokens, cached_tokens, reasoning_chars, "
    "usage_reported, duration_ms"
)


@dataclass
class CallRecord:
    call_id: str
    provider: str
    model: str
    ok: bool
    created_at: datetime | None = None
    role: str = ""
    step: str = ""
    phase: str = ""
    project_id: str = ""
    response: str = ""
    error_kind: str = ""
    error_message: str = ""
    prompt_tokens: int = 0
    completion_tokens: int = 0
    reasoning_tokens: int = 0
    cached_tokens: int = 0
    reasoning_chars: int = 0
    usage_reported: bool = True
    duration_ms: int = 0

    def to_dict(self) -> dict:
        out = {k: getattr(self, k) for k in self.__dataclass_fields__}
        out["created_at"] = self.created_at.isoformat() if self.created_at else None
        return out


@dataclass
class CallFilter:
    project_id: str = ""
    role: str = ""
    ok: bool | None = None
    before: datetime | None = None
    limit: int = 100


@dataclass
class ModelStats:
    model: str
    calls: int = 0
    ok: int = 0
    failures: dict[str, int] = field(default_factory=dict)
    avg_completion_tokens: int = 0
    avg_duration_ms: int = 0

    def to_dict(self) -> dict:
        return {
            "model": self.model, "calls": self.calls, "ok": self.ok, "failures": self.failures,
            "avg_completion_tokens": self.avg_completion_tokens, "avg_duration_ms": self.avg_duration_ms,
        }


class CallLogPort(Protocol):
    async def record(self, rec: CallRecord) -> None: ...

    async def list(self, flt: CallFilter) -> list[CallRecord]: ...

    async def stats(self, step: str, phase: str, since: datetime) -> list[ModelStats]: ...

    async def delete_ids(self, ids: list[str]) -> int: ...

    async def delete_older_than(self, cutoff: datetime) -> int: ...

    async def delete_all(self) -> int: ...


def new_call_id() -> str:
    return str(uuid.uuid4())


class PgCallLog:
    """CallLogPort over Postgres; `start` opens the pool and creates the table."""

    def __init__(self, dsn: str) -> None:
        self._dsn = dsn
        self._pool: asyncpg.Pool | None = None

    async def start(self) -> None:
        self._pool = await asyncpg.create_pool(self._dsn, min_size=1, max_size=5)
        async with self._pool.acquire() as conn:
            await conn.execute(SCHEMA)

    async def stop(self) -> None:
        if self._pool is not None:
            await self._pool.close()

    @property
    def pool(self) -> asyncpg.Pool:
        if self._pool is None:
            raise RuntimeError("the call log is not started")
        return self._pool

    async def record(self, rec: CallRecord) -> None:
        await self.pool.execute(
            """INSERT INTO llm_calls (call_id, provider, model, role, step, phase, project_id, response, ok,
                error_kind, error_message, prompt_tokens, completion_tokens, reasoning_tokens, cached_tokens,
                reasoning_chars, usage_reported, duration_ms)
                VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)""",
            uuid.UUID(rec.call_id), rec.provider, rec.model, rec.role, rec.step, rec.phase,
            rec.project_id or None, rec.response, rec.ok, rec.error_kind, rec.error_message,
            rec.prompt_tokens, rec.completion_tokens, rec.reasoning_tokens, rec.cached_tokens,
            rec.reasoning_chars, rec.usage_reported, rec.duration_ms,
        )

    async def list(self, flt: CallFilter) -> list[CallRecord]:
        where, args = ["TRUE"], []
        if flt.project_id:
            args.append(flt.project_id)
            where.append(f"project_id = ${len(args)}")
        if flt.role:
            args.append(flt.role)
            where.append(f"role = ${len(args)}")
        if flt.ok is not None:
            args.append(flt.ok)
            where.append(f"ok = ${len(args)}")
        if flt.before is not None:
            args.append(flt.before)
            where.append(f"created_at < ${len(args)}")
        args.append(flt.limit)
        rows = await self.pool.fetch(
            f"SELECT {_COLUMNS} FROM llm_calls WHERE {' AND '.join(where)} "
            f"ORDER BY created_at DESC LIMIT ${len(args)}", *args)
        return [_record_of(r) for r in rows]

    async def stats(self, step: str, phase: str, since: datetime) -> list[ModelStats]:
        """Per model: calls, successes, failures by kind. Averages cover successful calls only: a failed
        call's duration says how long it took to fail, not what a chunk costs."""
        rows = await self.pool.fetch(
            """SELECT model, ok, error_kind, count(*) AS n,
                      COALESCE(sum(completion_tokens), 0) AS tokens, COALESCE(sum(duration_ms), 0) AS ms
               FROM llm_calls WHERE step = $1 AND phase = $2 AND created_at >= $3
               GROUP BY model, ok, error_kind""", step, phase, since)
        by_model: dict[str, ModelStats] = {}
        sums: dict[str, tuple[int, int]] = {}
        for r in rows:
            m = by_model.setdefault(r["model"], ModelStats(r["model"]))
            m.calls += r["n"]
            if r["ok"]:
                m.ok += r["n"]
                t, d = sums.get(m.model, (0, 0))
                sums[m.model] = (t + r["tokens"], d + r["ms"])
            else:
                kind = r["error_kind"] or "unknown"
                m.failures[kind] = m.failures.get(kind, 0) + r["n"]
        for name, (tokens, ms) in sums.items():
            m = by_model[name]
            m.avg_completion_tokens = tokens // m.ok
            m.avg_duration_ms = ms // m.ok
        return sorted(by_model.values(), key=lambda m: m.model)

    async def delete_ids(self, ids: list[str]) -> int:
        tag = await self.pool.execute("DELETE FROM llm_calls WHERE call_id = ANY($1::uuid[])", ids)
        return _count(tag)

    async def delete_older_than(self, cutoff: datetime) -> int:
        return _count(await self.pool.execute("DELETE FROM llm_calls WHERE created_at < $1", cutoff))

    async def delete_all(self) -> int:
        return _count(await self.pool.execute("DELETE FROM llm_calls"))


def _count(tag: str) -> int:
    """asyncpg returns the command tag, e.g. "DELETE 3"."""
    return int(tag.rsplit(" ", 1)[1])


def _record_of(row: asyncpg.Record) -> CallRecord:
    d = dict(row)
    d["call_id"] = str(d["call_id"])
    d["project_id"] = d["project_id"] or ""
    return CallRecord(**d)

