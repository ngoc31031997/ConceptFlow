"""Sao lưu và khôi phục thư viện hình (ZIP)."""

from __future__ import annotations

import io
import json
import zipfile
from datetime import UTC, datetime

import pytest

from application.illustration_backup import BACKUP_FORMAT, IllustrationBackup
from application.illustration_library import IllustrationLibrary
from domain.illustration import (
    EXEMPLAR_FOLDER_ID,
    MAX_EXEMPLARS,
    BackupInvalidError,
    Illustration,
    IllustrationFolder,
    IllustrationStatus,
    system_folders,
)
from tests.illustration_fakes import BUS_CODE, REF, FakeFiles, FakeReferences, FakeRenderer, FakeRepo

NOW = datetime(2026, 9, 27, 8, 30, 0, tzinfo=UTC)
BALL_CODE = "export function Ball() { return null; }"


def make():
    repo, files, rend = FakeRepo(), FakeFiles(), FakeRenderer()
    library = IllustrationLibrary(repo, files, rend, FakeReferences())
    return library, IllustrationBackup(library, files.read), repo, rend


async def library_with_drawings():
    library, backup, repo, rend = make()
    await library.create_folder(IllustrationFolder(id="do-choi", name="Đồ chơi", description="bóng, búp bê"))
    ball = await library.create(
        Illustration(name="Ball", title="Quả bóng", folder_id="do-choi", code=BALL_CODE, tags=("bóng",)), REF
    )
    await library.set_status(ball.id, "approved")
    await library.create(Illustration(name="Bus", folder_id="phuong-tien", code=BUS_CODE), REF)
    return library, backup, repo, rend


def zip_files(blob: bytes) -> dict[str, bytes]:
    with zipfile.ZipFile(io.BytesIO(blob)) as zf:
        return {n: zf.read(n) for n in zf.namelist()}


def make_zip(files: dict[str, str]) -> bytes:
    buf = io.BytesIO()
    with zipfile.ZipFile(buf, "w") as zf:
        for name, body in files.items():
            zf.writestr(name, body)
    return buf.getvalue()


def by_name(repo: FakeRepo, name: str) -> Illustration | None:
    return next((r for r in repo.rows.values() if r.name == name), None)


async def test_export_writes_creator_drawings_with_code_and_images_by_folder():
    _, backup, *_ = await library_with_drawings()
    out = await backup.export(NOW)
    assert out.illustrations == 2 and out.folders == len(system_folders()) + 1
    files = zip_files(out.zip)
    for want in (
        "manifest.json",
        "hinh/do-choi/Ball.tsx",
        "hinh/do-choi/Ball.png",
        "hinh/do-choi/Ball.gif",
        "hinh/phuong-tien/Bus.tsx",
    ):
        assert want in files, f"{want} missing from the backup: {sorted(files)}"
    assert files["hinh/phuong-tien/Bus.tsx"].decode() == BUS_CODE
    assert not any("Tooth" in n for n in files), "built-ins ship with the service"
    manifest = json.loads(files["manifest.json"])
    assert manifest["format"] == BACKUP_FORMAT and manifest["format_version"] == 1
    assert manifest["exported_at"] == "2026-09-27T08:30:00Z"
    ball = next(i for i in manifest["illustrations"] if i["name"] == "Ball")
    assert ball["ref_id"] == REF and ball["kind"] == "figure" and ball["detail_level"] == "medium"


async def test_import_restores_a_lost_library_into_the_same_folders():
    _, backup, *_ = await library_with_drawings()
    blob = (await backup.export(NOW)).zip
    library, fresh_backup, repo, rend = make()
    report = await fresh_backup.import_library(blob, replace=False)
    assert not report.aborted and len(report.items) == 2 and report.folders_created == ["do-choi"]
    ball, bus = by_name(repo, "Ball"), by_name(repo, "Bus")
    assert ball.folder_id == "do-choi" and ball.status == IllustrationStatus.APPROVED
    assert ball.title == "Quả bóng" and ball.tags == ("bóng",) and ball.ref_id == REF
    assert bus.folder_id == "phuong-tien" and bus.status == IllustrationStatus.DRAFT and bus.code == BUS_CODE
    # Mọi hình đi qua bộ dựng, ảnh lưu là ảnh dựng lúc này.
    assert len(rend.calls) == 2 and repo.previews[ball.id][0] == ball.version
    folder = next(f for f in repo.folders if f.id == "do-choi")
    assert folder.name == "Đồ chơi" and folder.description == "bóng, búp bê"


async def test_import_skips_existing_names_unless_asked_to_replace():
    library, backup, repo, _ = await library_with_drawings()
    blob = (await backup.export(NOW)).zip
    before = by_name(repo, "Bus")
    await library.update(before.id, Illustration(code=BUS_CODE + "\n// sửa sau khi sao lưu"))

    report = await backup.import_library(blob, replace=False)
    assert all(it.result == "skipped" for it in report.items), report.items
    assert report.folders_created == []

    report = await backup.import_library(blob, replace=True)
    bus = by_name(repo, "Bus")
    result = next(it.result for it in report.items if it.name == "Bus")
    assert (
        result == "replaced" and bus.id == before.id and bus.code == BUS_CODE and bus.version == 3
    ), "replace must restore the backup's code as a new version of the same row"


async def test_import_never_overwrites_a_builtin_and_reports_broken_drawings():
    manifest = json.dumps(
        {
            "format": BACKUP_FORMAT,
            "format_version": 1,
            "folders": [],
            "illustrations": [
                {
                    "name": "Tooth",
                    "folder_id": "co-the-suc-khoe",
                    "status": "approved",
                    "version": 1,
                    "code_file": "a.tsx",
                },
                {
                    "name": "Broken",
                    "folder_id": "do-vat",
                    "status": "approved",
                    "version": 1,
                    "code_file": "b.tsx",
                },
                {"name": "Lost", "folder_id": "khong-co", "version": 1, "code_file": "c.tsx"},
                {"name": "NoCode", "folder_id": "do-vat", "version": 1, "code_file": "missing.tsx"},
                {
                    "name": "Kite",
                    "folder_id": "do-vat",
                    "status": "weird",
                    "version": 4,
                    "code_file": "d.tsx",
                },
            ],
        }
    )
    blob = make_zip(
        {
            "manifest.json": manifest,
            "a.tsx": "export function Tooth() {}",
            "b.tsx": "BROKEN",
            "c.tsx": "export function Lost() {}",
            "d.tsx": "export function Kite() {}",
        }
    )
    _, backup, repo, _ = make()
    report = await backup.import_library(blob, replace=True, ref_id=REF)
    want = ["skipped", "failed", "failed", "failed", "created"]
    for n, it in enumerate(report.items):
        assert it.result == want[n], f"item {n} {it.name}: got {it.result} ({it.reason}), want {want[n]}"
    assert report.items[1].diagnostics and "dòng 3" in report.items[1].reason
    assert repo.rows["builtin-Tooth"].code == "", "built-in overwritten"
    kite = by_name(repo, "Kite")
    assert (
        kite.status == IllustrationStatus.DRAFT and kite.version == 4
    ), "unknown status must come back as a draft"


async def test_a_drawing_without_a_reference_style_fails_when_none_is_given():
    manifest = json.dumps(
        {
            "format": BACKUP_FORMAT,
            "format_version": 1,
            "folders": [],
            "illustrations": [{"name": "Kite", "folder_id": "do-vat", "version": 1, "code_file": "d.tsx"}],
        }
    )
    _, backup, repo, _ = make()
    blob = make_zip({"manifest.json": manifest, "d.tsx": "export function Kite() {}"})
    report = await backup.import_library(blob, replace=False)
    assert report.items[0].result == "failed" and "ref_id" in report.items[0].reason
    assert by_name(repo, "Kite") is None
    report = await backup.import_library(blob, replace=False, ref_id=REF)
    assert report.items[0].result == "created"


async def test_import_stops_when_the_renderer_is_down_and_says_what_was_left():
    _, backup, *_ = await library_with_drawings()
    blob = (await backup.export(NOW)).zip
    _, fresh_backup, repo, rend = make()
    rend.unreachable = True
    report = await fresh_backup.import_library(blob, replace=False)
    assert "rendering unreachable" in report.aborted and report.not_processed == 2 and len(rend.calls) == 1
    assert by_name(repo, "Ball") is None, "a drawing that was not rendered was saved"


@pytest.mark.parametrize(
    "name, blob",
    [
        ("not a zip", b"hello"),
        ("no manifest", make_zip({"x.txt": "x"})),
        ("broken manifest", make_zip({"manifest.json": "{"})),
        ("other format", make_zip({"manifest.json": '{"format":"something","format_version":1}'})),
        (
            "newer format",
            make_zip({"manifest.json": json.dumps({"format": BACKUP_FORMAT, "format_version": 9})}),
        ),
    ],
)
async def test_import_refuses_files_that_are_not_a_library_backup(name, blob):
    _, backup, *_ = make()
    with pytest.raises(BackupInvalidError):
        await backup.import_library(blob, replace=False)


async def test_the_backup_keeps_the_exemplars_and_restores_them_as_exemplars():
    library, backup, repo, _ = await library_with_drawings()
    repo.rows["cat"] = Illustration(
        id="cat",
        name="Cat",
        title="Con mèo",
        folder_id=EXEMPLAR_FOLDER_ID,
        exemplar=True,
        home_folder_id="dong-vat",
        code="export function Cat() {}",
        status=IllustrationStatus.APPROVED,
        version=2,
        ref_id=REF,
    )
    ball = by_name(repo, "Ball")
    cp = await library.make_exemplar(ball.id)
    blob = (await backup.export(NOW)).zip
    manifest = json.loads(zip_files(blob)["manifest.json"])
    marked = {i["name"]: i for i in manifest["illustrations"]}
    assert marked["Cat"]["exemplar"] and marked["Cat"]["home_folder_id"] == "dong-vat"
    assert marked[cp.name]["exemplar"] and marked[cp.name]["source_name"] == "Ball"

    _, fresh_backup, fresh, _ = make()
    report = await fresh_backup.import_library(blob, replace=False)
    assert not report.aborted
    cat, copy, ball2 = by_name(fresh, "Cat"), by_name(fresh, cp.name), by_name(fresh, "Ball")
    assert cat.exemplar and cat.home_folder_id == "dong-vat" and cat.folder_id == EXEMPLAR_FOLDER_ID
    assert copy.exemplar and copy.source_id == ball2.id, "copy must find its source again"


async def test_import_into_a_full_exemplar_folder_keeps_originals_as_drawings_and_skips_copies():
    library, backup, repo, _ = await library_with_drawings()
    repo.rows["cat"] = Illustration(
        id="cat",
        name="Cat",
        folder_id=EXEMPLAR_FOLDER_ID,
        exemplar=True,
        home_folder_id="dong-vat",
        code="export function Cat() {}",
        status=IllustrationStatus.APPROVED,
        ref_id=REF,
    )
    cp = await library.make_exemplar(by_name(repo, "Ball").id)
    blob = (await backup.export(NOW)).zip

    _, full_backup, full, _ = make()
    for n in range(MAX_EXEMPLARS):
        full.rows[f"x{n}"] = Illustration(
            id=f"x{n}", name=f"Full{n}", folder_id=EXEMPLAR_FOLDER_ID, exemplar=True
        )
    report = await full_backup.import_library(blob, replace=False)
    results = {it.name: it for it in report.items}
    cat_item = results["Cat"]
    assert (
        cat_item.result == "created" and cat_item.folder_id == "dong-vat" and "hình thường" in cat_item.reason
    )
    cat = by_name(full, "Cat")
    assert not cat.exemplar and cat.folder_id == "dong-vat"
    copy_item = results[cp.name]
    assert copy_item.result == "skipped" and "Ball" in copy_item.reason


async def test_import_refuses_an_exemplar_whose_home_is_the_exemplar_folder():
    manifest = json.dumps(
        {
            "format": BACKUP_FORMAT,
            "format_version": 1,
            "folders": [],
            "illustrations": [
                {
                    "name": "Loop",
                    "title": "x",
                    "folder_id": "hinh-mau",
                    "status": "approved",
                    "version": 1,
                    "code_file": "hinh/hinh-mau/Loop.tsx",
                    "exemplar": True,
                    "home_folder_id": "hinh-mau",
                }
            ],
        }
    )
    blob = make_zip({"manifest.json": manifest, "hinh/hinh-mau/Loop.tsx": "export function Loop() {}"})
    _, backup, *_ = make()
    report = await backup.import_library(blob, replace=False, ref_id=REF)
    assert len(report.items) == 1 and report.items[0].result == "failed"


async def test_import_refuses_an_ordinary_drawing_filed_in_the_exemplar_folder():
    manifest = json.dumps(
        {
            "format": BACKUP_FORMAT,
            "format_version": 1,
            "folders": [],
            "illustrations": [
                {
                    "name": "Sneaky",
                    "title": "x",
                    "folder_id": "hinh-mau",
                    "status": "approved",
                    "version": 1,
                    "code_file": "hinh/hinh-mau/Sneaky.tsx",
                }
            ],
        }
    )
    blob = make_zip({"manifest.json": manifest, "hinh/hinh-mau/Sneaky.tsx": "export function Sneaky() {}"})
    _, backup, repo, _ = make()
    report = await backup.import_library(blob, replace=False, ref_id=REF)
    assert len(report.items) == 1 and report.items[0].result == "failed"
    assert by_name(repo, "Sneaky") is None, "an ordinary drawing got into the Hình mẫu folder"
