"""Sao lưu thư viện hình ra một file ZIP và khôi phục, để hình của Creator qua được lúc mất database.

ZIP có `manifest.json` (mọi thư mục, và mọi hình của Creator kèm thư mục, thẻ, trạng thái, phiên bản)
và với mỗi hình là TSX cùng PNG, GIF đã lưu dưới `hinh/<thư mục>/<Tên>.*` để mở xem bằng tay. Hình có
sẵn của kit không nằm trong file: nó đi cùng image và được nạp lại mỗi lần khởi động. Hình mẫu được
giữ, có đánh dấu.

Khôi phục xếp từng hình vào đúng thư mục nó được xuất ra, tạo thư mục của Creator còn thiếu và giữ
trạng thái duyệt. Như `create`, mọi hình đi qua bộ dựng trước — thư viện không giữ hình không dựng được
với kit hiện tại — và ảnh lưu là ảnh dựng lúc này, không phải ảnh trong file.
"""

from __future__ import annotations

import asyncio
import io
import json
import logging
import zipfile
from collections.abc import Callable
from dataclasses import dataclass, field
from datetime import datetime

from application.illustration_library import IllustrationLibrary, normalize_illustration
from application.illustration_ports import RenderedIllustration
from domain.illustration import (
    EXEMPLAR_FOLDER_ID,
    MAX_EXEMPLARS,
    AlreadyExemplarError,
    BackupInvalidError,
    ExemplarFolderError,
    ExemplarLimitError,
    FolderNotFoundError,
    Illustration,
    IllustrationError,
    IllustrationFilter,
    IllustrationFolder,
    IllustrationNameTakenError,
    IllustrationStatus,
    InvalidIllustrationError,
    ReferenceKitMissingError,
    ReferenceStyleNotFoundError,
    RefStyleRequiredError,
    detail_or_medium,
    format_time,
    kind_or_figure,
    tier_for_detail,
)

logger = logging.getLogger(__name__)

BACKUP_FORMAT = "conceptflow-illustration-library"
BACKUP_FORMAT_VERSION = 1
MANIFEST_PATH = "manifest.json"
MAX_MANIFEST_BYTES = 8 << 20
MAX_CODE_BYTES = 1 << 20


@dataclass
class LibraryExport:
    zip: bytes
    folders: int
    illustrations: int


@dataclass
class ImportItem:
    name: str
    folder_id: str
    result: str  # created | replaced | skipped | failed
    reason: str = ""
    diagnostics: list[dict] = field(default_factory=list)

    def to_json(self) -> dict:
        out: dict = {"name": self.name, "folder_id": self.folder_id, "result": self.result}
        if self.reason:
            out["reason"] = self.reason
        if self.diagnostics:
            out["diagnostics"] = self.diagnostics
        return out


@dataclass
class ImportReport:
    """Kết quả một lần khôi phục. `aborted` có khi bộ dựng hay database hỏng (không phải một hình): các
    hình từ đó trở đi chưa thử (`not_processed`), nhập lại cùng file sẽ làm tiếp vì hình đã có bị bỏ qua."""

    exported_at: str = ""
    folders_created: list[str] = field(default_factory=list)
    folder_errors: list[str] = field(default_factory=list)
    items: list[ImportItem] = field(default_factory=list)
    aborted: str = ""
    not_processed: int = 0

    def to_json(self) -> dict:
        out: dict = {
            "exported_at": self.exported_at, "folders_created": self.folders_created,
            "folder_errors": self.folder_errors, "items": [i.to_json() for i in self.items],
        }
        if self.aborted:
            out["aborted"] = self.aborted
        out["not_processed"] = self.not_processed
        return out


class _ItemFailed(Exception):
    """Một hình của file không nhập được; cả lần nhập vẫn tiếp tục."""

    def __init__(self, reason: str, diagnostics: list[dict] | None = None) -> None:
        super().__init__(reason)
        self.reason = reason
        self.diagnostics = diagnostics or []


def _read_entry(zf: zipfile.ZipFile, name: str, limit: int) -> bytes:
    info = zf.getinfo(name)
    if info.file_size > limit:
        raise BackupInvalidError(f"{name} lớn quá ({info.file_size} byte)")
    try:
        with zf.open(info) as fh:
            body = fh.read(limit + 1)
    except (zipfile.BadZipFile, OSError, RuntimeError) as exc:
        raise BackupInvalidError(f"không đọc được {name}: {exc}") from exc
    if len(body) > limit:
        raise BackupInvalidError(f"{name} lớn quá")
    return body


class IllustrationBackup:
    def __init__(self, library: IllustrationLibrary, read_file: Callable[[str], bytes]) -> None:
        self._library = library
        self._read_file = read_file

    async def export(self, now: datetime) -> LibraryExport:
        """Ghi mọi thư mục và mọi hình của Creator, kèm ảnh đã lưu, ra một ZIP. Chỉ đọc; không dựng gì."""
        repo = self._library.repo
        folders = await repo.list_folders()
        rows = await repo.list_illustrations(IllustrationFilter())
        name_of = {r.id: r.name for r in rows}
        stamp = (now.year, now.month, now.day, now.hour, now.minute, now.second)
        buf = io.BytesIO()
        manifest_items: list[dict] = []
        with zipfile.ZipFile(buf, "w") as zf:
            def put(path: str, body: bytes, method: int) -> None:
                zf.writestr(zipfile.ZipInfo(path, date_time=stamp), body, compress_type=method)

            for i in rows:
                if i.builtin:
                    continue
                # Mã thư mục là slug và tên là PascalCase nên đường dẫn an toàn nguyên dạng.
                base = f"hinh/{i.folder_id}/{i.name}"
                entry: dict = {
                    "name": i.name, "title": i.title, "folder_id": i.folder_id, "tags": list(i.tags),
                    "description": i.description, "usage": i.usage, "status": i.status.value,
                    "version": i.version, "created_at": i.created_at, "updated_at": i.updated_at,
                    "kind": i.kind.value, "detail_level": i.detail_level.value, "ref_id": i.ref_id,
                }
                if i.exemplar:
                    entry["exemplar"] = True
                if name_of.get(i.source_id):
                    entry["source_name"] = name_of[i.source_id]
                if i.home_folder_id:
                    entry["home_folder_id"] = i.home_folder_id
                entry["code_file"] = base + ".tsx"
                put(entry["code_file"], i.code.encode("utf-8"), zipfile.ZIP_DEFLATED)
                png_path, gif_path = await repo.get_preview(i.id)
                # PNG và GIF đã nén sẵn; lưu nguyên đỡ tốn CPU.
                if png_path:
                    entry["png_file"] = base + ".png"
                    png = await asyncio.to_thread(self._read_file, png_path)
                    put(entry["png_file"], png, zipfile.ZIP_STORED)
                if gif_path:
                    entry["gif_file"] = base + ".gif"
                    gif = await asyncio.to_thread(self._read_file, gif_path)
                    put(entry["gif_file"], gif, zipfile.ZIP_STORED)
                manifest_items.append(entry)
            manifest = {
                "format": BACKUP_FORMAT, "format_version": BACKUP_FORMAT_VERSION,
                "exported_at": format_time(now),
                "folders": [f.to_json() for f in folders], "illustrations": manifest_items,
            }
            put(MANIFEST_PATH, json.dumps(manifest, ensure_ascii=False, indent=2).encode("utf-8"),
                zipfile.ZIP_DEFLATED)
        return LibraryExport(zip=buf.getvalue(), folders=len(folders), illustrations=len(manifest_items))

    async def import_library(self, file: bytes, replace: bool, ref_id: str = "") -> ImportReport:
        """Khôi phục một bản sao lưu do `export` tạo. `ref_id` là phong cách tham chiếu để dựng các hình
        mà file không ghi phong cách (file xuất trước khi có `ref_id`)."""
        try:
            zf = zipfile.ZipFile(io.BytesIO(file))
        except zipfile.BadZipFile as exc:
            raise BackupInvalidError(f"không phải file ZIP ({exc})") from exc
        with zf:
            if MANIFEST_PATH not in zf.namelist():
                raise BackupInvalidError(f"thiếu {MANIFEST_PATH} — đây không phải file sao lưu thư viện hình")
            raw = _read_entry(zf, MANIFEST_PATH, MAX_MANIFEST_BYTES)
            try:
                manifest = json.loads(raw)
            except ValueError as exc:
                raise BackupInvalidError(f"{MANIFEST_PATH} hỏng: {exc}") from exc
            if not isinstance(manifest, dict):
                raise BackupInvalidError(f"{MANIFEST_PATH} hỏng: không phải một object")
            if manifest.get("format") != BACKUP_FORMAT:
                raise BackupInvalidError(
                    f'định dạng "{manifest.get("format", "")}" không phải sao lưu thư viện hình')
            version = manifest.get("format_version")
            if not isinstance(version, int) or version < 1 or version > BACKUP_FORMAT_VERSION:
                raise BackupInvalidError(
                    f"phiên bản định dạng {version} chưa được hỗ trợ "
                    f"(bản này đọc tới {BACKUP_FORMAT_VERSION})")
            report = ImportReport(exported_at=str(manifest.get("exported_at") or ""))
            await self._import_folders(list(manifest.get("folders") or []), report)
            by_name = {r.name: r for r in await self._library.repo.list_illustrations(IllustrationFilter())}
            seen: set[str] = set()
            entries = [e for e in (manifest.get("illustrations") or []) if isinstance(e, dict)]
            # Hình mẫu sau cùng, để bản sao tìm thấy hình nó được sao từ.
            entries.sort(key=lambda e: bool(e.get("exemplar")))
            kits: dict[str, dict] = {}
            for n, entry in enumerate(entries):
                try:
                    item = await self._import_one(zf, entry, by_name, seen, replace, ref_id, kits)
                except Exception as exc:
                    logger.error("illustration import stopped at %r: %s", entry.get("name"), exc)
                    report.aborted = str(exc)
                    report.not_processed = len(entries) - n
                    return report
                report.items.append(item)
        return report

    async def _import_folders(self, folders: list[dict], report: ImportReport) -> None:
        """Tạo, theo thứ tự đã xuất, các thư mục mà thư viện còn thiếu. Thư mục có sẵn giữ tên hiện tại.
        Thư mục không tạo được thì báo, và các hình của nó sẽ báo "thư mục không có"."""
        have = {f.id for f in await self._library.repo.list_folders()}
        ordered = sorted(
            (f for f in folders if isinstance(f, dict)), key=lambda f: int(f.get("position") or 0))
        for f in ordered:
            folder_id = str(f.get("id") or "")
            if folder_id in have:
                continue
            try:
                made = await self._library.create_folder(IllustrationFolder(
                    id=folder_id, name=str(f.get("name") or ""), description=str(f.get("description") or "")))
            except IllustrationError as exc:
                report.folder_errors.append(f"{folder_id}: {exc}")
                continue
            have.add(made.id)
            report.folders_created.append(made.id)

    async def _import_one(
        self, zf: zipfile.ZipFile, b: dict, by_name: dict[str, Illustration], seen: set[str],
        replace: bool, default_ref: str, kits: dict[str, dict],
    ) -> ImportItem:
        """Khôi phục một hình. Lỗi không phải `IllustrationError` (bộ dựng, database) làm cả lần nhập dừng."""
        name = str(b.get("name") or "")
        item = ImportItem(name=name, folder_id=str(b.get("folder_id") or ""), result="failed")
        try:
            return await self._import_one_checked(zf, b, by_name, seen, replace, default_ref, kits, item)
        except _ItemFailed as exc:
            item.result, item.reason, item.diagnostics = "failed", exc.reason, exc.diagnostics
            return item

    async def _import_one_checked(
        self, zf: zipfile.ZipFile, b: dict, by_name: dict[str, Illustration], seen: set[str],
        replace: bool, default_ref: str, kits: dict[str, dict], item: ImportItem,
    ) -> ImportItem:
        library = self._library
        name = item.name
        if name in seen:
            raise _ItemFailed("tên này xuất hiện hai lần trong file sao lưu")
        seen.add(name)
        code_name = str(b.get("code_file") or "")
        if not code_name or code_name not in zf.namelist():
            raise _ItemFailed(f'thiếu file code "{code_name}" trong file sao lưu')
        try:
            code = _read_entry(zf, code_name, MAX_CODE_BYTES).decode("utf-8")
            ill = normalize_illustration(Illustration(
                name=name, title=str(b.get("title") or ""), folder_id=item.folder_id,
                tags=tuple(str(t) for t in (b.get("tags") or [])),
                description=str(b.get("description") or ""), usage=str(b.get("usage") or ""), code=code))
        except (IllustrationError, UnicodeDecodeError) as exc:
            raise _ItemFailed(str(exc)) from exc
        status = IllustrationStatus.APPROVED if b.get("status") == "approved" else IllustrationStatus.DRAFT
        is_exemplar = bool(b.get("exemplar"))
        home_folder = str(b.get("home_folder_id") or "")

        existing = by_name.get(ill.name)
        if existing is not None:
            if existing.builtin:
                return self._skipped(item, "trùng tên một hình có sẵn của hệ thống")
            if existing.exemplar:
                return self._skipped(item, "trùng tên một Hình mẫu (chỉ đọc)")
            if is_exemplar or not replace:
                # Hình mẫu trong file không bao giờ ghi đè hình của Creator.
                return self._skipped(item, "đã có hình cùng tên trong thư viện")
        if not is_exemplar and ill.folder_id == EXEMPLAR_FOLDER_ID:
            raise _ItemFailed(str(ExemplarFolderError()))
        if is_exemplar and home_folder == EXEMPLAR_FOLDER_ID:
            raise _ItemFailed("thư mục cũ của Hình mẫu không thể là chính thư mục Hình mẫu")
        if is_exemplar and home_folder:
            try:
                await library.folder_exists(home_folder)
            except FolderNotFoundError as exc:
                raise _ItemFailed(f'thư mục cũ "{home_folder}" của Hình mẫu không có trong thư viện') from exc
        try:
            await library.folder_exists(ill.folder_id)
        except FolderNotFoundError as exc:
            raise _ItemFailed(f'thư mục "{ill.folder_id}" không có trong thư viện') from exc

        detail = detail_or_medium(b.get("detail_level"))
        kind = kind_or_figure(b.get("kind"))
        style_ref = str(b.get("ref_id") or "") or default_ref
        if not style_ref:
            raise _ItemFailed(str(RefStyleRequiredError()))
        try:
            if style_ref not in kits:
                kits[style_ref] = await library.kit_for(style_ref)
            out = await library.render(
                style_ref, ill.name, ill.code, kind, tier_for_detail(detail), kit=kits[style_ref])
        except InvalidIllustrationError as exc:
            raise _ItemFailed(str(exc), [d.to_json() for d in exc.diagnostics]) from exc
        except (ReferenceStyleNotFoundError, ReferenceKitMissingError) as exc:
            raise _ItemFailed(str(exc)) from exc
        ill = ill.with_(kind=kind, detail_level=detail, ref_id=style_ref, warnings=tuple(out.warnings),
                        status=status)
        if is_exemplar:
            return await self._import_exemplar(ill, b, home_folder, by_name, out, item)

        repo = library.repo
        try:
            if existing is not None:
                # Phiên bản mới, để trình duyệt không hiện ảnh cũ từ cache.
                saved = await repo.update_illustration(
                    ill.with_(id=existing.id, version=existing.version + 1))
                item.result = "replaced"
            else:
                saved = await repo.create_illustration(ill.with_(version=max(_int(b.get("version")), 1)))
                item.result = "created"
        except IllustrationNameTakenError as exc:
            raise _ItemFailed("đã có hình cùng tên trong thư viện") from exc
        await library.save_preview(saved.id, saved.version, out)
        by_name[saved.name] = saved
        return item

    @staticmethod
    def _skipped(item: ImportItem, reason: str) -> ImportItem:
        item.result, item.reason = "skipped", reason
        return item

    async def _import_exemplar(
        self, ill: Illustration, b: dict, home_folder: str, by_name: dict[str, Illustration],
        out: RenderedIllustration, item: ImportItem,
    ) -> ImportItem:
        """Khôi phục một Hình mẫu khi thư mục còn chỗ. Đầy thì hình mẫu gốc về thành hình thường ở thư mục
        cũ; bản sao bị bỏ qua vì hình gốc của nó đã là một hình của thư viện."""
        library, repo = self._library, self._library.repo
        source_name = str(b.get("source_name") or "")
        version = max(_int(b.get("version")), 1)
        source = by_name.get(source_name)
        source_id = source.id if source and source_name and not source.read_only else ""
        ex = ill.with_(
            folder_id=EXEMPLAR_FOLDER_ID, exemplar=True, status=IllustrationStatus.APPROVED, version=version,
            home_folder_id=home_folder, source_id=source_id)
        item.result = "created"
        try:
            saved = await repo.create_exemplar(ex, MAX_EXEMPLARS)
        except ExemplarLimitError:
            if not ex.home_folder_id:
                item.result = "skipped"
                item.reason = (f"thư mục Hình mẫu đã đủ {MAX_EXEMPLARS} hình; đây là bản sao của "
                               f"{source_name} nên không nhập")
                return item
            plain = ill.with_(
                folder_id=ex.home_folder_id, status=IllustrationStatus.APPROVED, version=version)
            try:
                saved = await repo.create_illustration(plain)
            except IllustrationNameTakenError:
                item.result, item.reason = "failed", "đã có hình cùng tên trong thư viện"
                return item
            item.folder_id = plain.folder_id
            item.reason = f"thư mục Hình mẫu đã đủ {MAX_EXEMPLARS} hình — nhập thành hình thường"
        except AlreadyExemplarError:
            item.result, item.reason = "skipped", f"hình gốc {source_name} đã có bản Hình mẫu"
            return item
        except IllustrationNameTakenError:
            item.result, item.reason = "failed", "đã có hình cùng tên trong thư viện"
            return item
        await library.save_preview(saved.id, saved.version, out)
        by_name[saved.name] = saved
        return item


def _int(value: object) -> int:
    return value if isinstance(value, int) and not isinstance(value, bool) else 0

