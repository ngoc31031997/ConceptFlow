#!/usr/bin/env python3
"""Move the prompt library and the default model of each step from authoring-db to llm-db.

Run once, by hand, in this order:

    copy         copy the Creator's prompts (is_system = false, active or not) and the
                 default models into llm-db, keeping id, is_active and the timestamps
    verify       compare the two databases; exits 1 and lists every difference
    drop-source  verify once more, then drop `prompts` and the `default_model.*` rows
                 of `app_settings` from authoring-db (asks for --yes)

Needs llm-service to have started once on the new build (it creates the tables and
receives authoring's system prompts) and the `asyncpg` package. Connection strings come
from AUTHORING_DATABASE_URL and LLM_DATABASE_URL.

System prompts are not copied: authoring pushes its own to llm-service at start-up.
"""

from __future__ import annotations

import argparse
import asyncio
import os
import sys

import asyncpg

KEY_PREFIX = "default_model."
COLUMNS = "id, role, name, template_text, is_system, is_active, created_at, updated_at"


def _dsn(name: str) -> str:
    value = os.environ.get(name, "")
    if not value:
        sys.exit(f"{name} is not set")
    return value


async def _connect() -> tuple[asyncpg.Connection, asyncpg.Connection]:
    return await asyncpg.connect(_dsn("AUTHORING_DATABASE_URL")), await asyncpg.connect(_dsn("LLM_DATABASE_URL"))


async def _source_prompts(src: asyncpg.Connection) -> list[asyncpg.Record]:
    return await src.fetch(f"SELECT {COLUMNS} FROM prompts ORDER BY role, created_at, id")


async def _source_models(src: asyncpg.Connection) -> dict[str, str]:
    rows = await src.fetch("SELECT key, value FROM app_settings WHERE key LIKE $1", KEY_PREFIX + "%")
    return {r["key"][len(KEY_PREFIX):]: r["value"] for r in rows}


async def copy() -> None:
    src, dst = await _connect()
    try:
        creator = [r for r in await _source_prompts(src) if not r["is_system"]]
        models = await _source_models(src)
        copied = skipped = 0
        async with dst.transaction():
            for r in creator:
                if await dst.fetchval("SELECT 1 FROM prompts WHERE id = $1", r["id"]):
                    skipped += 1
                    continue
                if r["is_active"]:
                    # The role's system row took the active slot when it was first pushed.
                    await dst.execute("UPDATE prompts SET is_active = false WHERE role = $1 AND is_active", r["role"])
                await dst.execute(
                    f"INSERT INTO prompts ({COLUMNS}, owner) VALUES ($1, $2, $3, $4, false, $5, $6, $7, '')",
                    r["id"], r["role"], r["name"], r["template_text"], r["is_active"], r["created_at"], r["updated_at"])
                copied += 1
            for step, model in models.items():
                await dst.execute(
                    """INSERT INTO step_default_models (step, model) VALUES ($1, $2)
                       ON CONFLICT (step) DO UPDATE SET model = EXCLUDED.model, updated_at = now()""", step, model)
        print(f"copied {copied} Creator prompts ({skipped} already there), {len(models)} default models")
    finally:
        await src.close()
        await dst.close()


def _running(rows: list[asyncpg.Record]) -> dict[str, str]:
    """The row each role runs on: the active one, else the system one."""
    best: dict[str, asyncpg.Record] = {}
    for r in rows:
        cur = best.get(r["role"])
        if cur is None or (r["is_active"], r["is_system"]) > (cur["is_active"], cur["is_system"]):
            best[r["role"]] = r
    return {role: r["id"] for role, r in best.items()}


async def _differences(src: asyncpg.Connection, dst: asyncpg.Connection) -> list[str]:
    problems: list[str] = []
    source = await _source_prompts(src)
    creator = {r["id"]: r for r in source if not r["is_system"]}
    dest = {r["id"]: r for r in await dst.fetch(f"SELECT {COLUMNS} FROM prompts")}
    print(f"Creator prompts: authoring-db {len(creator)}, llm-db {sum(1 for r in dest.values() if not r['is_system'])}")
    for pid, r in creator.items():
        d = dest.get(pid)
        if d is None:
            problems.append(f"prompt {pid} ({r['role']}) is missing in llm-db")
            continue
        for col in ("role", "name", "template_text", "is_system", "is_active", "created_at", "updated_at"):
            if r[col] != d[col]:
                problems.append(f"prompt {pid}: {col} differs")
    src_run, dst_run = _running(source), _running(list(dest.values()))
    source_by_id = {r["id"]: r for r in source}
    for role, pid in src_run.items():
        if source_by_id[pid]["is_system"] and role not in dst_run:
            # A system row no service ships any more: nobody pushes it, so there is nothing to carry over.
            print(f"NOTE: role {role} has only a system row in authoring-db that no service ships; not carried over")
            continue
        if dst_run.get(role) != pid:
            problems.append(f"role {role}: runs on {pid} in authoring-db but {dst_run.get(role)} in llm-db")
    print(f"Roles with a prompt: authoring-db {len(src_run)}, llm-db {len(dst_run)}")
    src_models = await _source_models(src)
    dst_models = {r["step"]: r["model"] for r in await dst.fetch("SELECT step, model FROM step_default_models")}
    print(f"Default models: authoring-db {len(src_models)}, llm-db {len(dst_models)}")
    for step, model in src_models.items():
        if dst_models.get(step) != model:
            problems.append(f"default model of {step}: {model!r} in authoring-db, {dst_models.get(step)!r} in llm-db")
    return problems


async def verify() -> int:
    src, dst = await _connect()
    try:
        problems = await _differences(src, dst)
    finally:
        await src.close()
        await dst.close()
    for p in problems:
        print("DIFFERENT:", p)
    print("OK: nothing differs" if not problems else f"{len(problems)} difference(s)")
    return 1 if problems else 0


async def drop_source(yes: bool) -> int:
    if not yes:
        print("drop-source deletes data from authoring-db; run again with --yes", file=sys.stderr)
        return 2
    src, dst = await _connect()
    try:
        problems = await _differences(src, dst)
        if problems:
            for p in problems:
                print("DIFFERENT:", p)
            print("not dropping: fix the differences first", file=sys.stderr)
            return 1
        async with src.transaction():
            await src.execute("DROP TABLE IF EXISTS prompts")
            tag = await src.execute("DELETE FROM app_settings WHERE key LIKE $1", KEY_PREFIX + "%")
        print(f"dropped table prompts; app_settings: {tag}")
        return 0
    finally:
        await src.close()
        await dst.close()


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = parser.add_subparsers(dest="cmd", required=True)
    sub.add_parser("copy")
    sub.add_parser("verify")
    d = sub.add_parser("drop-source")
    d.add_argument("--yes", action="store_true")
    args = parser.parse_args()
    if args.cmd == "copy":
        asyncio.run(copy())
        return 0
    if args.cmd == "verify":
        return asyncio.run(verify())
    return asyncio.run(drop_source(args.yes))


if __name__ == "__main__":
    sys.exit(main())
