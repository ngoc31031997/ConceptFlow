"""The prompt library and the default model of each step, in llm-service's own database.

Every service that calls an AI keeps its system prompts next to the code that fills
them, and pushes them here at start-up (`replace_system`); the Creator's own
prompts and the choice of the active one live only here. Filling a template with
a project's data stays in the service that owns the data.
"""

from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime
from typing import Protocol

import asyncpg

from app import modelcatalog

SCHEMA = """
CREATE TABLE IF NOT EXISTS prompts (
    id            TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    role          TEXT NOT NULL,
    name          TEXT NOT NULL,
    template_text TEXT NOT NULL,
    is_system     BOOLEAN NOT NULL DEFAULT false,
    is_active     BOOLEAN NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    owner         TEXT NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX IF NOT EXISTS prompts_one_active_per_role ON prompts (role) WHERE is_active;
CREATE UNIQUE INDEX IF NOT EXISTS prompts_one_system_per_role ON prompts (role) WHERE is_system;
CREATE TABLE IF NOT EXISTS step_default_models (
    step       TEXT PRIMARY KEY,
    model      TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
"""

_COLUMNS = "id, role, name, template_text, is_system, is_active, created_at, updated_at"


class PromptNotFound(Exception):
    """No prompt has this id (or the role has no prompt at all)."""


class PromptReadOnly(Exception):
    """A system prompt cannot be edited or deleted; copy it to get one that can."""


class PromptInvalid(ValueError):
    """The request is well-formed JSON but not acceptable (blank name, unknown role, ...)."""


class PromptConflict(Exception):
    """A service tried to push a system prompt for a role another service owns."""


@dataclass
class Prompt:
    id: str
    role: str
    name: str
    template_text: str
    is_system: bool
    is_active: bool
    created_at: datetime
    updated_at: datetime

    def to_dict(self) -> dict:
        return {
            "id": self.id, "role": self.role, "name": self.name, "template_text": self.template_text,
            "is_system": self.is_system, "is_active": self.is_active,
            "created_at": _rfc3339(self.created_at), "updated_at": _rfc3339(self.updated_at),
        }


def _rfc3339(value: datetime) -> str:
    """Second precision with a `Z` or offset, the shape the GUI already reads."""
    return value.replace(microsecond=0).isoformat().replace("+00:00", "Z")


@dataclass(frozen=True)
class SystemPrompt:
    role: str
    name: str
    template_text: str


class LibraryStore(Protocol):
    """What the library rules need from storage."""

    async def get_active(self, role: str) -> Prompt | None: ...

    async def get_system(self, role: str) -> Prompt | None: ...

    async def get(self, prompt_id: str) -> Prompt | None: ...

    async def list(self, role: str) -> list[Prompt]: ...

    async def role_exists(self, role: str) -> bool: ...

    async def create(self, role: str, name: str, template_text: str) -> Prompt: ...

    async def update(self, prompt_id: str, name: str, template_text: str) -> Prompt | None: ...

    async def delete(self, prompt_id: str) -> bool: ...

    async def activate(self, prompt_id: str) -> Prompt | None: ...

    async def replace_system(self, owner: str, prompts: list[SystemPrompt]) -> tuple[int, int]: ...

    async def saved_step_models(self) -> dict[str, str]: ...

    async def save_step_models(self, models: dict[str, str]) -> None: ...


class PromptLibrary:
    """The rules of the library over a store."""

    def __init__(self, store: LibraryStore) -> None:
        self._store = store

    async def _check_role(self, role: str) -> None:
        if not await self._store.role_exists(role):
            raise PromptInvalid(f"unknown role {role!r}")

    async def active(self, role: str) -> Prompt:
        await self._check_role(role)
        found = await self._store.get_active(role)
        if found is None:
            raise PromptNotFound(role)
        return found

    async def list(self, role: str = "") -> list[Prompt]:
        if role:
            await self._check_role(role)
        return await self._store.list(role)

    async def create(self, role: str, name: str, template_text: str) -> Prompt:
        await self._check_role(role)
        name = _valid_name_and_text(name, template_text)
        return await self._store.create(role, name, template_text)

    async def copy(self, prompt_id: str) -> Prompt:
        src = await self._store.get(prompt_id)
        if src is None:
            raise PromptNotFound(prompt_id)
        return await self._store.create(src.role, "Bản sao của " + src.name, src.template_text)

    async def update(self, prompt_id: str, name: str, template_text: str) -> Prompt:
        existing = await self._store.get(prompt_id)
        if existing is None:
            raise PromptNotFound(prompt_id)
        if existing.is_system:
            raise PromptReadOnly(prompt_id)
        name = _valid_name_and_text(name, template_text)
        updated = await self._store.update(prompt_id, name, template_text)
        if updated is None:
            raise PromptNotFound(prompt_id)
        return updated

    async def activate(self, prompt_id: str) -> Prompt:
        activated = await self._store.activate(prompt_id)
        if activated is None:
            raise PromptNotFound(prompt_id)
        return activated

    async def delete(self, prompt_id: str) -> None:
        """Deleting the active prompt hands the role back to its system prompt."""
        existing = await self._store.get(prompt_id)
        if existing is None:
            raise PromptNotFound(prompt_id)
        if existing.is_system:
            raise PromptReadOnly(prompt_id)
        if not await self._store.delete(prompt_id):
            raise PromptNotFound(prompt_id)
        if existing.is_active:
            system = await self._store.get_system(existing.role)
            if system is None:
                raise PromptNotFound(existing.role)
            await self._store.activate(system.id)

    async def replace_system(self, owner: str, prompts: list[SystemPrompt]) -> dict:
        """Make `owner`'s system prompts exactly `prompts`; nobody else's rows are touched."""
        if not owner.strip():
            raise PromptInvalid("owner is required")
        roles = [p.role for p in prompts]
        if len(set(roles)) != len(roles):
            raise PromptInvalid("a role appears twice")
        for p in prompts:
            if not p.role.strip() or not p.name.strip() or not p.template_text.strip():
                raise PromptInvalid(f"system prompt {p.role!r} needs a role, a name and a template_text")
        written, removed = await self._store.replace_system(owner, prompts)
        return {"owner": owner, "written": written, "removed": removed}

    async def step_models(self) -> dict:
        """Every configurable step with its default model (the saved one, else the catalog's first)."""
        saved = await self._store.saved_step_models()
        return {"step_models": modelcatalog.with_fallback(saved), "steps": list(modelcatalog.STEPS)}

    async def set_step_model(self, step: str, model: str) -> None:
        """An unknown step or a model outside the catalog is refused, never replaced, so the
        Creator is never told one model is the default while another runs."""
        if not modelcatalog.valid_step(step):
            raise PromptInvalid(f"bước {step!r} không cấu hình được model")
        if model == "" or not modelcatalog.valid_model(model):
            raise PromptInvalid(f"model {model!r} không có trong danh sách")
        await self._store.save_step_models({step: model})

    async def set_step_models(self, models: dict[str, str]) -> None:
        for step, model in models.items():
            if not modelcatalog.valid_step(step):
                raise PromptInvalid(f"bước {step!r} không cấu hình được model")
            if model == "" or not modelcatalog.valid_model(model):
                raise PromptInvalid(f"model {model!r} không có trong danh sách")
        await self._store.save_step_models(models)


def _valid_name_and_text(name: str, template_text: str) -> str:
    name = name.strip()
    if not name:
        raise PromptInvalid("name is required")
    if not template_text.strip():
        raise PromptInvalid("template_text is required")
    return name


def _prompt_of(row: asyncpg.Record) -> Prompt:
    return Prompt(
        row["id"], row["role"], row["name"], row["template_text"], row["is_system"], row["is_active"],
        row["created_at"], row["updated_at"])


class PgLibraryStore:
    """LibraryStore over Postgres; `start` opens the pool and creates the tables."""

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
            raise RuntimeError("the prompt library is not started")
        return self._pool

    async def get_active(self, role: str) -> Prompt | None:
        """The active row, and if none is (its row was just deleted) the system row."""
        row = await self.pool.fetchrow(
            f"SELECT {_COLUMNS} FROM prompts WHERE role = $1 ORDER BY is_active DESC, is_system DESC LIMIT 1", role)
        return _prompt_of(row) if row else None

    async def get_system(self, role: str) -> Prompt | None:
        row = await self.pool.fetchrow(f"SELECT {_COLUMNS} FROM prompts WHERE role = $1 AND is_system", role)
        return _prompt_of(row) if row else None

    async def get(self, prompt_id: str) -> Prompt | None:
        row = await self.pool.fetchrow(f"SELECT {_COLUMNS} FROM prompts WHERE id = $1", prompt_id)
        return _prompt_of(row) if row else None

    async def list(self, role: str) -> list[Prompt]:
        """Grouped by role, system row first, then the Creator's rows oldest first."""
        rows = await self.pool.fetch(
            f"SELECT {_COLUMNS} FROM prompts WHERE ($1 = '' OR role = $1) "
            "ORDER BY role, is_system DESC, created_at, id", role)
        return [_prompt_of(r) for r in rows]

    async def role_exists(self, role: str) -> bool:
        """A role exists once some service has pushed a system prompt for it."""
        return bool(await self.pool.fetchval("SELECT EXISTS (SELECT 1 FROM prompts WHERE role = $1 AND is_system)", role))

    async def create(self, role: str, name: str, template_text: str) -> Prompt:
        row = await self.pool.fetchrow(
            f"INSERT INTO prompts (role, name, template_text) VALUES ($1, $2, $3) RETURNING {_COLUMNS}",
            role, name, template_text)
        return _prompt_of(row)

    async def update(self, prompt_id: str, name: str, template_text: str) -> Prompt | None:
        """Refuses system rows in SQL as well, so a caller that skipped the rules cannot edit one."""
        row = await self.pool.fetchrow(
            f"UPDATE prompts SET name = $2, template_text = $3, updated_at = now() "
            f"WHERE id = $1 AND NOT is_system RETURNING {_COLUMNS}", prompt_id, name, template_text)
        return _prompt_of(row) if row else None

    async def delete(self, prompt_id: str) -> bool:
        tag = await self.pool.execute("DELETE FROM prompts WHERE id = $1 AND NOT is_system", prompt_id)
        return tag.endswith(" 1")

    async def activate(self, prompt_id: str) -> Prompt | None:
        """Switch the role's active row in one transaction: the old one off first, because the unique
        index refuses two at once."""
        async with self.pool.acquire() as conn, conn.transaction():
            role = await conn.fetchval("SELECT role FROM prompts WHERE id = $1 FOR UPDATE", prompt_id)
            if role is None:
                return None
            await conn.execute("UPDATE prompts SET is_active = false WHERE role = $1 AND is_active AND id <> $2",
                               role, prompt_id)
            row = await conn.fetchrow(f"UPDATE prompts SET is_active = true WHERE id = $1 RETURNING {_COLUMNS}",
                                      prompt_id)
        return _prompt_of(row)

    async def replace_system(self, owner: str, prompts: list[SystemPrompt]) -> tuple[int, int]:
        """Upsert `owner`'s system rows and delete the ones it no longer lists, in one transaction.

        A role's system row becomes active only when the role has no active row yet (the first push), so a
        restart never undoes the Creator's choice."""
        async with self.pool.acquire() as conn, conn.transaction():
            for p in prompts:
                pid = "system-" + p.role
                holder = await conn.fetchval("SELECT owner FROM prompts WHERE role = $1 AND is_system", p.role)
                if holder is not None and holder not in ("", owner):
                    raise PromptConflict(f"role {p.role!r} is owned by {holder!r}")
                await conn.execute(
                    """INSERT INTO prompts (id, role, name, template_text, is_system, is_active, owner, updated_at)
                       VALUES ($1, $2, $3, $4, true,
                               NOT EXISTS (SELECT 1 FROM prompts WHERE role = $2 AND is_active), $5, now())
                       ON CONFLICT (id) DO UPDATE SET
                           name = EXCLUDED.name, template_text = EXCLUDED.template_text,
                           owner = EXCLUDED.owner, updated_at = now()""",
                    pid, p.role, p.name, p.template_text, owner)
            tag = await conn.execute(
                "DELETE FROM prompts WHERE is_system AND owner = $1 AND role <> ALL($2::text[])",
                owner, [p.role for p in prompts])
        return len(prompts), int(tag.rsplit(" ", 1)[-1])

    async def saved_step_models(self) -> dict[str, str]:
        rows = await self.pool.fetch("SELECT step, model FROM step_default_models")
        return {r["step"]: r["model"] for r in rows}

    async def save_step_models(self, models: dict[str, str]) -> None:
        async with self.pool.acquire() as conn, conn.transaction():
            for step, model in models.items():
                await conn.execute(
                    """INSERT INTO step_default_models (step, model) VALUES ($1, $2)
                       ON CONFLICT (step) DO UPDATE SET model = EXCLUDED.model, updated_at = now()""",
                    step, model)
