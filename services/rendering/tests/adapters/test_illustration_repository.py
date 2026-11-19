"""Repository của thư viện hình trên PostgreSQL thật.

Chạy khi có `TEST_DATABASE_URL` trỏ tới một database trống dành riêng cho test (vd một container
postgres tạm); không có thì bị bỏ qua, vì phần còn lại của bộ test không cần database.
"""

from __future__ import annotations

import asyncio
import os

import asyncpg
import pytest

from adapters.persistence.illustration_repository import IllustrationRepository
from domain.illustration import (
    EXEMPLAR_FOLDER_ID,
    AlreadyExemplarError,
    CodeFinding,
    ExemplarLimitError,
    FolderNotFoundError,
    FolderTakenError,
    Illustration,
    IllustrationFilter,
    IllustrationFolder,
    IllustrationKind,
    IllustrationNameTakenError,
    IllustrationNotFoundError,
    IllustrationStatus,
    builtin_illustrations,
    system_folders,
)

DATABASE_URL = os.environ.get("TEST_DATABASE_URL", "")
pytestmark = pytest.mark.skipif(not DATABASE_URL, reason="TEST_DATABASE_URL không đặt")

CODE = "export function Bus() { return null; }"


@pytest.fixture
async def repo():
    from adapters.persistence.db import SCHEMA  # đọc DATABASE_URL lúc import, nên chỉ nạp khi thật sự chạy

    pool = await asyncpg.create_pool(DATABASE_URL)
    async with pool.acquire() as conn:
        await conn.execute("DROP TABLE IF EXISTS illustrations, illustration_folders CASCADE")
        await conn.execute(SCHEMA)
    repository = IllustrationRepository(pool)
    await repository.seed()
    yield repository
    async with pool.acquire() as conn:
        await conn.execute("DROP TABLE IF EXISTS illustrations, illustration_folders CASCADE")
    await pool.close()


def bus(**kw) -> Illustration:
    return Illustration(
        name="Bus",
        title="Xe buýt",
        folder_id="phuong-tien",
        tags=("xe", "bus"),
        description="d",
        usage="u",
        code=CODE,
        ref_id="R12",
        warnings=(CodeFinding("S11 đen thuần", 7),),
        **kw,
    )


async def test_seed_creates_the_system_folders_and_the_builtin_kit_and_is_repeatable(repo):
    await repo.seed()
    folders = await repo.list_folders()
    assert [f.id for f in folders] == [f.id for f in system_folders()] and all(f.is_system for f in folders)
    rows = await repo.list_illustrations(IllustrationFilter())
    assert len(rows) == len(builtin_illustrations()) and all(
        r.builtin and r.status == "approved" for r in rows
    )
    tooth = await repo.get_illustration("builtin-Tooth")
    assert (
        tooth.folder_id == "co-the-suc-khoe"
        and tooth.code == ""
        and tooth.version == 1
        and tooth.ref_id == ""
    )
    backdrop = await repo.get_illustration("builtin-MeadowBackdrop")
    assert backdrop.kind == IllustrationKind.BACKDROP and backdrop.folder_id == "boi-canh"


async def test_the_schema_bootstrap_can_run_again_on_an_existing_database(repo):
    from adapters.persistence.db import SCHEMA

    async with repo._pool.acquire() as conn:
        await conn.execute(SCHEMA)
    assert len(await repo.list_illustrations(IllustrationFilter())) == len(builtin_illustrations())


async def test_seed_leaves_a_creator_drawing_that_took_a_builtins_name(repo):
    pool = repo._pool
    await pool.execute("DELETE FROM illustrations WHERE id = 'builtin-Tooth'")
    await repo.create_illustration(bus().with_(name="Tooth", status=IllustrationStatus.DRAFT))
    await repo.seed()
    rows = [r for r in await repo.list_illustrations(IllustrationFilter()) if r.name == "Tooth"]
    assert len(rows) == 1 and not rows[0].builtin


async def test_create_round_trips_every_field_and_maps_constraint_errors(repo):
    made = await repo.create_illustration(bus(status=IllustrationStatus.DRAFT, version=1))
    got = await repo.get_illustration(made.id)
    assert got == made
    assert (got.name, got.title, got.folder_id, got.tags, got.code, got.ref_id) == (
        "Bus",
        "Xe buýt",
        "phuong-tien",
        ("xe", "bus"),
        CODE,
        "R12",
    )
    assert got.warnings == (CodeFinding("S11 đen thuần", 7),) and got.kind == IllustrationKind.FIGURE
    assert got.created_at.endswith("Z") and not got.has_preview and not got.builtin and not got.exemplar
    with pytest.raises(IllustrationNameTakenError):
        await repo.create_illustration(bus())
    with pytest.raises(FolderNotFoundError):
        await repo.create_illustration(bus().with_(name="Van", folder_id="khong-co"))
    with pytest.raises(IllustrationNotFoundError):
        await repo.get_illustration("khong-co")


async def test_update_replaces_the_row_but_never_a_builtin(repo):
    made = await repo.create_illustration(bus())
    saved = await repo.update_illustration(
        made.with_(title="Mới", version=2, status=IllustrationStatus.APPROVED, warnings=())
    )
    assert saved.title == "Mới" and saved.version == 2 and saved.warnings == ()
    with pytest.raises(IllustrationNotFoundError):
        await repo.update_illustration((await repo.get_illustration("builtin-Tooth")).with_(title="x"))


async def test_list_filters_by_folder_status_and_text_and_puts_builtins_first(repo):
    a = await repo.create_illustration(bus())
    await repo.create_illustration(
        bus().with_(
            name="Cat", title="Mèo", folder_id="dong-vat", tags=("thú",), status=IllustrationStatus.APPROVED
        )
    )
    assert [r.name for r in await repo.list_illustrations(IllustrationFilter(folder_id="dong-vat"))] == [
        "Cat"
    ]
    assert [
        r.name for r in await repo.list_illustrations(IllustrationFilter(status="approved")) if not r.builtin
    ] == ["Cat"]
    assert [r.id for r in await repo.list_illustrations(IllustrationFilter(query=" BUS "))] == [a.id]
    assert [r.name for r in await repo.list_illustrations(IllustrationFilter(query="thú"))] == ["Cat"]
    listed = await repo.list_illustrations(IllustrationFilter())
    assert [r.builtin for r in listed] == sorted((r.builtin for r in listed), reverse=True)


async def test_delete_skips_builtins_and_missing_rows(repo):
    made = await repo.create_illustration(bus())
    await repo.delete_illustration(made.id)
    await repo.delete_illustration(made.id)
    await repo.delete_illustration("builtin-Tooth")
    assert (await repo.get_illustration("builtin-Tooth")).builtin
    with pytest.raises(IllustrationNotFoundError):
        await repo.get_illustration(made.id)


async def test_folders_take_the_next_position_and_system_folders_survive_delete(repo):
    created = await repo.create_folder(IllustrationFolder(id="do-choi", name="Đồ chơi", description="d"))
    assert created.position == len(system_folders()) + 1 and not created.is_system
    with pytest.raises(FolderTakenError):
        await repo.create_folder(IllustrationFolder(id="do-choi", name="x"))
    await repo.delete_folder("con-nguoi")
    await repo.delete_folder("do-choi")
    ids = [f.id for f in await repo.list_folders()]
    assert "con-nguoi" in ids and "do-choi" not in ids


async def test_exemplar_limit_and_one_copy_per_source_are_enforced_under_a_lock(repo):
    src = await repo.create_illustration(bus())
    ex = Illustration(
        name="BusMau",
        folder_id=EXEMPLAR_FOLDER_ID,
        code=CODE,
        source_id=src.id,
        ref_id="R12",
        status=IllustrationStatus.APPROVED,
    )
    first = await repo.create_exemplar(ex, 2)
    assert first.exemplar and first.folder_id == EXEMPLAR_FOLDER_ID and first.source_id == src.id
    with pytest.raises(AlreadyExemplarError):
        await repo.create_exemplar(ex.with_(name="BusMau2"), 2)
    await repo.create_exemplar(ex.with_(name="Cat", source_id="", home_folder_id="dong-vat"), 2)
    with pytest.raises(ExemplarLimitError):
        await repo.create_exemplar(ex.with_(name="Dog", source_id=""), 2)


async def test_two_exemplars_made_at_once_cannot_pass_the_limit_together(repo):
    base = Illustration(name="X", folder_id=EXEMPLAR_FOLDER_ID, code=CODE, status=IllustrationStatus.APPROVED)
    results = await asyncio.gather(
        *(repo.create_exemplar(base.with_(name=f"Ex{n}"), 1) for n in range(4)), return_exceptions=True
    )
    assert sum(isinstance(r, Illustration) for r in results) == 1
    assert sum(isinstance(r, ExemplarLimitError) for r in results) == 3


async def test_release_files_an_original_back_and_deleting_a_source_empties_the_copys_source_id(repo):
    cat = await repo.create_exemplar(
        Illustration(name="Cat", folder_id=EXEMPLAR_FOLDER_ID, code=CODE, home_folder_id="dong-vat"), 5
    )
    assert cat.home_folder_id == "dong-vat"
    back = await repo.release_exemplar(cat.id, "dong-vat")
    assert not back.exemplar and back.home_folder_id == "" and back.folder_id == "dong-vat"
    with pytest.raises(IllustrationNotFoundError):
        await repo.release_exemplar(cat.id, "dong-vat")
    src = await repo.create_illustration(bus())
    copy = await repo.create_exemplar(
        Illustration(name="BusMau", folder_id=EXEMPLAR_FOLDER_ID, code=CODE, source_id=src.id), 5
    )
    await repo.delete_illustration(src.id)
    assert (await repo.get_illustration(copy.id)).source_id == ""


async def test_the_preview_counts_only_for_the_version_it_was_made_for(repo):
    made = await repo.create_illustration(bus().with_(version=1))
    assert await repo.get_preview(made.id) == ("", "")
    await repo.set_preview(made.id, 1, "/shared/illustrations/x/v1.png", "/shared/illustrations/x/v1.gif")
    assert await repo.get_preview(made.id) == (
        "/shared/illustrations/x/v1.png",
        "/shared/illustrations/x/v1.gif",
    )
    assert (await repo.get_illustration(made.id)).has_preview
    await repo.update_illustration(made.with_(version=2))
    assert await repo.get_preview(made.id) == ("", "")
    assert not (await repo.get_illustration(made.id)).has_preview
    with pytest.raises(IllustrationNotFoundError):
        await repo.get_preview("khong-co")
