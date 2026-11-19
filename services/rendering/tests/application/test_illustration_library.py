"""Use case của thư viện hình: tạo, sửa, duyệt, xoá, Hình mẫu, xem trước."""

from __future__ import annotations

import pytest

from application.illustration_library import IllustrationLibrary
from domain.illustration import (
    EXEMPLAR_FOLDER_ID,
    MAX_EXEMPLARS,
    AlreadyExemplarError,
    ExemplarFolderError,
    ExemplarLimitError,
    FolderNotEmptyError,
    FolderNotFoundError,
    FolderReadOnlyError,
    Illustration,
    IllustrationError,
    IllustrationFolder,
    IllustrationReadOnlyError,
    IllustrationStatus,
    InvalidIllustrationError,
    NotExemplarError,
    NotExemplarSourceError,
    ReferenceStyleNotFoundError,
    RefStyleRequiredError,
    builtin_illustrations,
)
from tests.illustration_fakes import (
    BUS_CODE,
    REF,
    FakeFiles,
    FakeReferences,
    FakeRenderer,
    FakeRepo,
)


def make_library():
    repo, files, rend, refs = FakeRepo(), FakeFiles(), FakeRenderer(), FakeReferences()
    return IllustrationLibrary(repo, files, rend, refs), repo, files, rend, refs


def bus_row(**changes) -> Illustration:
    base = Illustration(
        id="bus",
        name="Bus",
        title="Xe buýt",
        folder_id="phuong-tien",
        code=BUS_CODE,
        usage="<Bus color /> — 320×210",
        ref_id=REF,
        status=IllustrationStatus.APPROVED,
        version=3,
    )
    return base.with_(**changes)


async def test_create_renders_first_and_stores_a_draft_with_its_preview():
    lib, repo, files, rend, refs = make_library()
    out = await lib.create(
        Illustration(name="Bus", folder_id="phuong-tien", code=BUS_CODE, tags=(" Xe ", "xe")), REF
    )
    assert (
        out.status == IllustrationStatus.DRAFT and out.version == 1 and out.has_preview and out.title == "Bus"
    )
    assert out.tags == ("xe",)
    assert out.ref_id == REF
    assert files.files[f"/shared/illustrations/{out.id}/v1.png"] == f"PNG:Bus{BUS_CODE}".encode()
    assert files.files[f"/shared/illustrations/{out.id}/v1.gif"] == b"GIF"
    assert rend.last["kit"]["ink"] and refs.kit_calls == [REF]


async def test_create_refuses_code_the_renderer_rejects_with_line_numbers():
    lib, repo, *_ = make_library()
    with pytest.raises(InvalidIllustrationError) as exc:
        await lib.create(Illustration(name="Bus", folder_id="phuong-tien", code="BROKEN"), REF)
    assert exc.value.diagnostics[0].line == 3 and "dòng 3" in str(exc.value)
    assert len(repo.rows) == len(builtin_illustrations()), "a drawing that does not render was saved"
    with pytest.raises(FolderNotFoundError):
        await lib.create(Illustration(name="Bus", folder_id="khong-co", code=BUS_CODE), REF)
    with pytest.raises(IllustrationError):
        await lib.create(Illustration(name="school_bus", folder_id="phuong-tien", code=BUS_CODE), REF)


async def test_a_drawing_needs_a_reference_style_and_the_style_must_exist():
    lib, *_ = make_library()
    with pytest.raises(RefStyleRequiredError):
        await lib.create(Illustration(name="Bus", folder_id="phuong-tien", code=BUS_CODE), "")
    with pytest.raises(ReferenceStyleNotFoundError):
        await lib.create(Illustration(name="Bus", folder_id="phuong-tien", code=BUS_CODE), "R99")


async def test_editing_code_sends_an_approved_drawing_back_to_draft_with_a_new_version():
    lib, repo, files, rend, _ = make_library()
    made = await lib.create(Illustration(name="Bus", folder_id="phuong-tien", code=BUS_CODE), REF)
    await lib.set_status(made.id, "approved")
    # Đổi thư mục giữ nguyên trạng thái đã duyệt và không dựng gì.
    calls = len(rend.calls)
    moved = await lib.update(made.id, Illustration(folder_id="do-vat"))
    assert moved.status == IllustrationStatus.APPROVED and moved.version == 1 and len(rend.calls) == calls
    edited = await lib.update(made.id, Illustration(code=BUS_CODE + "\n// bánh to hơn"))
    assert edited.status == IllustrationStatus.DRAFT and edited.version == 2
    assert repo.previews[made.id][0] == 2
    assert f"/shared/illustrations/{made.id}/v1.png" not in files.files, "the old version's image stays"


async def test_builtins_are_read_only_and_rendered_lazily_without_code():
    lib, repo, files, rend, _ = make_library()
    with pytest.raises(IllustrationReadOnlyError):
        await lib.update("builtin-Tooth", Illustration(code=BUS_CODE))
    with pytest.raises(IllustrationReadOnlyError):
        await lib.delete("builtin-Tooth")
    png, _gif = await lib.preview("builtin-Tooth", REF)
    assert png == b"PNG:Tooth" and rend.calls[0] == "Tooth|"
    await lib.preview("builtin-Tooth", REF)
    assert len(rend.calls) == 1, "stored preview rendered again"


async def test_a_builtin_without_a_style_needs_one_to_render():
    lib, *_ = make_library()
    with pytest.raises(RefStyleRequiredError):
        await lib.preview("builtin-Tooth")


async def test_system_folders_cannot_be_deleted_and_full_folders_stay():
    lib, *_ = make_library()
    with pytest.raises(FolderReadOnlyError):
        await lib.delete_folder("con-nguoi")
    with pytest.raises(IllustrationError):
        await lib.create_folder(IllustrationFolder(id="đồ chơi", name="Đồ chơi"))
    await lib.create_folder(IllustrationFolder(id="do-choi", name="Đồ chơi"))
    await lib.create(Illustration(name="Ball", folder_id="do-choi", code=BUS_CODE), REF)
    with pytest.raises(FolderNotEmptyError):
        await lib.delete_folder("do-choi")
    with pytest.raises(FolderNotFoundError):
        await lib.delete_folder("khong-co")


async def test_deleting_a_drawing_removes_its_images():
    lib, repo, files, *_ = make_library()
    made = await lib.create(Illustration(name="Bus", folder_id="phuong-tien", code=BUS_CODE), REF)
    assert any(made.id in p for p in files.files)
    await lib.delete(made.id)
    assert made.id not in repo.rows and not any(made.id in p for p in files.files)


async def test_kit_and_exemplars_cannot_be_deleted_edited_or_re_reviewed():
    lib, repo, *_ = make_library()
    repo.rows["ex"] = Illustration(
        id="ex",
        name="Cat",
        folder_id=EXEMPLAR_FOLDER_ID,
        exemplar=True,
        home_folder_id="dong-vat",
        code="export function Cat() {}",
        status=IllustrationStatus.APPROVED,
    )
    for ill_id in ("builtin-Tooth", "ex"):
        with pytest.raises(IllustrationReadOnlyError):
            await lib.delete(ill_id)
        with pytest.raises(IllustrationReadOnlyError):
            await lib.update(ill_id, Illustration(code=BUS_CODE))
        with pytest.raises(IllustrationReadOnlyError):
            await lib.set_status(ill_id, "draft")


async def test_nothing_is_saved_into_the_exemplar_folder_by_hand():
    lib, repo, *_ = make_library()
    repo.rows["bus"] = bus_row()
    with pytest.raises(ExemplarFolderError):
        await lib.create(Illustration(name="Van", folder_id=EXEMPLAR_FOLDER_ID, code=BUS_CODE), REF)
    with pytest.raises(ExemplarFolderError):
        await lib.update("bus", Illustration(folder_id=EXEMPLAR_FOLDER_ID))


async def test_make_exemplar_copies_under_a_new_name_and_renders_it():
    lib, repo, files, rend, _ = make_library()
    repo.rows["bus"] = bus_row()
    cp = await lib.make_exemplar("bus")
    assert (
        cp.name == "BusMau" and cp.folder_id == EXEMPLAR_FOLDER_ID and cp.exemplar and cp.source_id == "bus"
    )
    assert cp.status == IllustrationStatus.APPROVED and cp.title == "Xe buýt" and cp.has_preview
    assert "export function BusMau()" in cp.code and cp.usage == "<BusMau color /> — 320×210"
    assert rend.calls[-1] == f"BusMau|{cp.code}", "the copy must pass the renderer under its own name"
    assert repo.rows["bus"].code == BUS_CODE, "the source changed"
    with pytest.raises(AlreadyExemplarError):
        await lib.make_exemplar("bus")
    with pytest.raises(AlreadyExemplarError):
        await lib.make_exemplar(cp.id)
    assert [e.id for e in await lib.exemplars()] == [cp.id]


async def test_make_exemplar_refuses_drafts_kit_and_a_full_folder():
    lib, repo, *_ = make_library()
    repo.rows["draft"] = Illustration(
        id="draft", name="Van", folder_id="phuong-tien", code=BUS_CODE, status=IllustrationStatus.DRAFT
    )
    for ill_id in ("draft", "builtin-Tooth"):
        with pytest.raises(NotExemplarSourceError):
            await lib.make_exemplar(ill_id)
    for n in range(MAX_EXEMPLARS):
        repo.rows[f"ex{n}"] = Illustration(
            id=f"ex{n}", name=f"Ex{n}", folder_id=EXEMPLAR_FOLDER_ID, exemplar=True
        )
    repo.rows["bus"] = bus_row()
    with pytest.raises(ExemplarLimitError):
        await lib.make_exemplar("bus")


async def test_unmake_deletes_a_copy_and_files_an_original_back():
    lib, repo, *_ = make_library()
    repo.rows["bus"] = bus_row()
    repo.rows["cat"] = Illustration(
        id="cat",
        name="Cat",
        folder_id=EXEMPLAR_FOLDER_ID,
        exemplar=True,
        home_folder_id="dong-vat",
        code="export function Cat() {}",
        status=IllustrationStatus.APPROVED,
    )
    cp = await lib.make_exemplar("bus")
    assert await lib.unmake_exemplar(cp.id) is None
    assert cp.id not in repo.rows and "bus" in repo.rows, "the source went with the copy"
    out = await lib.unmake_exemplar("cat")
    assert out is not None and not out.exemplar and out.folder_id == "dong-vat" and not out.read_only
    with pytest.raises(NotExemplarError):
        await lib.unmake_exemplar("bus")


async def test_exemplars_are_listed_oldest_first():
    lib, repo, *_ = make_library()
    for n, name in enumerate(["SchoolBus", "Cat", "Microscope"]):
        repo.rows[f"e{n}"] = Illustration(
            id=f"e{n}",
            name=name,
            folder_id=EXEMPLAR_FOLDER_ID,
            exemplar=True,
            created_at=f"2026-09-2{n + 1}T00:00:00Z",
        )
    repo.rows["e1"] = repo.rows["e1"].with_(created_at="2026-09-20T00:00:00Z")
    assert [e.name for e in await lib.exemplars()] == ["Cat", "SchoolBus", "Microscope"]


async def test_try_render_saves_nothing():
    lib, repo, files, rend, _ = make_library()
    out = await lib.try_render("Bus", BUS_CODE, "figure", "rich", REF)
    assert out.png and len(repo.rows) == len(builtin_illustrations()) and not files.files
    assert rend.last["tier"] == "hero"
    with pytest.raises(RefStyleRequiredError):
        await lib.try_render("Bus", BUS_CODE, "figure", "", "")


async def test_rerender_stores_the_preview_of_the_current_version():
    lib, repo, files, *_ = make_library()
    made = await lib.create(Illustration(name="Bus", folder_id="phuong-tien", code=BUS_CODE), REF)
    files.files.clear()
    repo.previews.clear()
    png, gif = await lib.rerender(made.id)
    assert png and gif == b"GIF" and repo.previews[made.id][0] == 1 and files.files
