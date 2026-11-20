"""Repository Storyboard trên PostgreSQL thật (cần `TEST_DATABASE_URL`, xem test_illustration_repository)."""

from __future__ import annotations

import os

import asyncpg
import pytest

from adapters.persistence.project_asset_repository import ProjectAssetRepository
from adapters.persistence.storyboard_repository import StoryboardRepository
from domain.project_asset import ProjectAsset
from domain.storyboard import ShotFrame, StoryboardVersion, VersionConflictError

DATABASE_URL = os.environ.get("TEST_DATABASE_URL", "")
pytestmark = pytest.mark.skipif(not DATABASE_URL, reason="TEST_DATABASE_URL không đặt")
TABLES = "project_assets, project_asset_counters, storyboard_shots, storyboard_versions"


def shot(project="p1", shot_id="SH-001", **kw) -> ShotFrame:
    fields = {"unit_id": "VP-001", "beat_id": "B1", "frame_path": "storyboard/SH-001.png",
              "frame_hash": "f" * 64, "spec_hash": "s" * 64, "drawn_at": ""}
    fields.update(kw)
    return ShotFrame(project_id=project, shot_id=shot_id, **fields)


def version(project="p1", number=1, **kw) -> StoryboardVersion:
    fields = {"document": {"shots": [{"shot_id": "SH-001", "frame_path": "storyboard/v1/SH-001.png"}]},
              "sequence_version": 2, "note": "n", "locked_at": ""}
    fields.update(kw)
    return StoryboardVersion(project_id=project, version=number, **fields)


@pytest.fixture
async def pool():
    from adapters.persistence.db import SCHEMA

    pool = await asyncpg.create_pool(DATABASE_URL)
    async with pool.acquire() as conn:
        await conn.execute(f"DROP TABLE IF EXISTS {TABLES}")
        await conn.execute(SCHEMA)
    yield pool
    async with pool.acquire() as conn:
        await conn.execute(f"DROP TABLE IF EXISTS {TABLES}")
    await pool.close()


async def test_upsert_shot_replaces_the_row_and_keeps_the_error_with_no_frame(pool):
    repo = StoryboardRepository(pool)
    assert await repo.get_shot("p1", "SH-001") is None
    first = await repo.upsert_shot(shot())
    assert first.frame_path == "storyboard/SH-001.png" and first.drawn_at.endswith("Z") and first.error == ""
    failed = await repo.upsert_shot(
        shot(frame_path=None, frame_hash="", spec_hash="t" * 64, error="Chromium"))
    assert failed.frame_path is None and failed.error == "Chromium"
    got = await repo.get_shot("p1", "SH-001")
    assert got == failed
    assert await repo.get_shot("p2", "SH-001") is None, "rows are per project"


async def test_versions_are_numbered_per_project_insert_only_and_a_duplicate_is_a_conflict(pool):
    repo = StoryboardRepository(pool)
    assert await repo.next_version("p1") == 1 and await repo.latest_version("p1") is None
    stored = await repo.insert_version(version())
    assert stored.version == 1 and stored.locked_at.endswith("Z")
    assert stored.document["shots"][0]["frame_path"] == "storyboard/v1/SH-001.png"
    assert await repo.next_version("p1") == 2 and await repo.next_version("p2") == 1
    with pytest.raises(VersionConflictError):
        await repo.insert_version(version(note="again"))
    await repo.insert_version(version(number=2, sequence_version=5))
    assert (await repo.latest_version("p1")).sequence_version == 5
    assert (await repo.get_version("p1", 1)).note == "n"
    assert await repo.get_version("p1", 9) is None


async def test_delete_project_removes_its_assets_counters_shots_and_versions_only(pool):
    repo = StoryboardRepository(pool)
    assets = ProjectAssetRepository(pool)
    for project in ("p1", "p2"):
        await assets.create_asset(
            ProjectAsset(project_id=project, concept_id="C1", name="Space", portrayal="x"))
        await repo.upsert_shot(shot(project=project))
        await repo.insert_version(version(project=project))
    await repo.delete_project("p1")
    assert await assets.list_assets("p1") == [] and await repo.get_shot("p1", "SH-001") is None
    assert await repo.latest_version("p1") is None
    counters = await pool.fetchval("SELECT count(*) FROM project_asset_counters WHERE project_id = 'p1'")
    assert counters == 0
    assert len(await assets.list_assets("p2")) == 1 and await repo.latest_version("p2") is not None
    await repo.delete_project("p1")  # idempotent
