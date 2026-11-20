"""Repository tài sản dự án trên PostgreSQL thật (cần `TEST_DATABASE_URL`, xem test_illustration_repository)
và kho ảnh trên đĩa."""

from __future__ import annotations

import os

import asyncpg
import pytest

from adapters.persistence.project_asset_repository import ProjectAssetRepository
from adapters.storage.project_asset_files import ProjectAssetFiles
from domain.illustration import CodeFinding, IllustrationError
from domain.project_asset import AssetNotFoundError, AssetStatus, ProjectAsset

DATABASE_URL = os.environ.get("TEST_DATABASE_URL", "")
needs_db = pytest.mark.skipif(not DATABASE_URL, reason="TEST_DATABASE_URL không đặt")


def asset(**kw) -> ProjectAsset:
    return ProjectAsset(
        project_id="p1", concept_id="C1", name="Space", portrayal="Vũ trụ", must_not_imply=("ban ngày",),
        palette_tokens=("navy",), expressions=("tĩnh",), units=("VP-001", "VP-002"), **kw)


@pytest.fixture
async def repo():
    from adapters.persistence.db import SCHEMA

    pool = await asyncpg.create_pool(DATABASE_URL)
    drop = "DROP TABLE IF EXISTS project_assets, project_asset_counters"
    async with pool.acquire() as conn:
        await conn.execute(drop)
        await conn.execute(SCHEMA)
    yield ProjectAssetRepository(pool)
    async with pool.acquire() as conn:
        await conn.execute(drop)
    await pool.close()


@needs_db
async def test_create_issues_per_project_codes_and_round_trips_every_field(repo):
    first = await repo.create_asset(asset())
    second = await repo.create_asset(asset())
    other = await repo.create_asset(asset().with_(project_id="p2"))
    codes = (first.asset_id, second.asset_id, other.asset_id)
    assert codes == ("AST-SPACE-001", "AST-SPACE-002", "AST-SPACE-001")
    changed = second.with_(
        code="x", version=3, preview_path="/p.png", status=AssetStatus.DRAWN,
        warnings=(CodeFinding("w", 4),), library_illustration_id="i1")
    saved = await repo.update_asset(changed)
    assert await repo.get_asset("p1", second.asset_id) == saved
    assert saved.warnings == (CodeFinding("w", 4),) and saved.must_not_imply == ("ban ngày",)
    assert saved.units == ("VP-001", "VP-002") and first.units == ("VP-001", "VP-002")
    moved = await repo.update_asset(saved.with_(units=("VP-003",)))
    assert moved.units == ("VP-003",)
    listed = await repo.create_asset(asset().with_(project_id="p3", units=(), portrayal=""))
    assert listed.units == () and not listed.has_detail
    assert [a.asset_id for a in await repo.list_assets("p1")] == ["AST-SPACE-001", "AST-SPACE-002"]


@needs_db
async def test_a_deleted_code_is_never_issued_again(repo):
    await repo.create_asset(asset())
    last = await repo.create_asset(asset())
    await repo.delete_asset("p1", last.asset_id)
    assert (await repo.create_asset(asset())).asset_id == "AST-SPACE-003"


@needs_db
async def test_missing_rows_raise_not_found_and_the_bootstrap_is_repeatable(repo):
    from adapters.persistence.db import SCHEMA

    with pytest.raises(AssetNotFoundError):
        await repo.get_asset("p1", "AST-SPACE-001")
    with pytest.raises(AssetNotFoundError):
        await repo.update_asset(asset().with_(asset_id="AST-SPACE-001"))
    async with repo._pool.acquire() as conn:
        await conn.execute(SCHEMA)


def test_the_preview_is_written_under_the_project_asset_folder_and_old_versions_are_removed(tmp_path):
    files = ProjectAssetFiles(tmp_path)
    first = files.save("p1", "AST-SPACE-001", 1, b"one")
    second = files.save("p1", "AST-SPACE-001", 2, b"two")
    assert second == str(tmp_path / "p1" / "storyboard" / "assets" / "AST-SPACE-001" / "v2.png")
    assert not os.path.exists(first) and files.read(second) == b"two"
    assert not [p for p in os.listdir(os.path.dirname(second)) if p.endswith(".tmp")]
    files.remove("p1", "AST-SPACE-001")
    assert not os.path.exists(second)


async def test_a_preview_path_outside_the_root_is_never_read(tmp_path):
    from application.project_assets import ProjectAssets
    from tests.project_asset_fakes import FakeAssetRepo

    repo = FakeAssetRepo()
    made = await repo.create_asset(asset())
    await repo.update_asset(made.with_(preview_path="/etc/hosts"))
    use_case = ProjectAssets(repo, ProjectAssetFiles(tmp_path), None, None)
    with pytest.raises(ValueError):
        await use_case.preview("p1", made.asset_id)


def test_the_files_adapter_refuses_paths_that_escape_its_root(tmp_path):
    files = ProjectAssetFiles(tmp_path)
    with pytest.raises(IllustrationError):
        files.save("../x", "AST-SPACE-001", 1, b"")
    with pytest.raises(ValueError):
        files.read("/etc/hosts")
