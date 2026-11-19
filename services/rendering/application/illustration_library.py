"""Thư viện hình minh hoạ: thư mục, hình, duyệt, Hình mẫu, xem trước.

Mỗi hình có một ảnh PNG và một GIF ngắn do chính rendering dựng, Creator duyệt trước khi Kỹ sư
Remotion được dùng. Hình được vẽ theo bộ thông số (kit) của một phong cách tham chiếu: `ref_id`
chọn phong cách, kit lấy từ visual-service mỗi lần dựng.
"""

from __future__ import annotations

import asyncio

from application.illustration_ports import (
    IllustrationRendererPort,
    IllustrationRepoPort,
    PreviewFilesPort,
    ReferenceSourcePort,
    RenderedIllustration,
)
from domain.illustration import (
    EXEMPLAR_FOLDER_ID,
    MAX_EXEMPLARS,
    AlreadyExemplarError,
    DetailLevel,
    ExemplarFolderError,
    ExemplarLimitError,
    FolderNotEmptyError,
    FolderNotFoundError,
    FolderReadOnlyError,
    Illustration,
    IllustrationError,
    IllustrationFilter,
    IllustrationFolder,
    IllustrationKind,
    IllustrationReadOnlyError,
    IllustrationStatus,
    InvalidIllustrationError,
    NotExemplarError,
    NotExemplarSourceError,
    RefStyleRequiredError,
    detail_or_medium,
    exemplar_copy_name,
    kind_or_figure,
    normalize_tags,
    rename_export,
    tier_for_detail,
    tier_or_support,
    validate_folder_id,
    validate_illustration_name,
)


def normalize_illustration(i: Illustration) -> Illustration:
    """Cắt khoảng trắng, chuẩn hoá thẻ, kiểm tên và code; tiêu đề trống lấy tên."""
    name, title = i.name.strip(), i.title.strip()
    validate_illustration_name(name)
    if not i.code.strip():
        raise IllustrationError("code là bắt buộc")
    return i.with_(
        name=name, title=title or name, description=i.description.strip(), usage=i.usage.strip(),
        tags=normalize_tags(i.tags))


class IllustrationLibrary:
    def __init__(
        self, repo: IllustrationRepoPort, files: PreviewFilesPort, renderer: IllustrationRendererPort,
        references: ReferenceSourcePort,
    ) -> None:
        self.repo = repo
        self._files = files
        self._renderer = renderer
        self._references = references

    # --- thư mục ---

    async def folders(self) -> list[IllustrationFolder]:
        return await self.repo.list_folders()

    async def create_folder(self, folder: IllustrationFolder) -> IllustrationFolder:
        folder_id, name = folder.id.strip(), folder.name.strip()
        validate_folder_id(folder_id)
        if not name:
            raise IllustrationError("tên thư mục là bắt buộc")
        return await self.repo.create_folder(IllustrationFolder(
            id=folder_id, name=name, description=folder.description, is_system=False))

    async def delete_folder(self, folder_id: str) -> None:
        """Xoá một thư mục rỗng của Creator."""
        for folder in await self.repo.list_folders():
            if folder.id != folder_id:
                continue
            if folder.is_system:
                raise FolderReadOnlyError()
            if await self.repo.list_illustrations(IllustrationFilter(folder_id=folder_id)):
                raise FolderNotEmptyError()
            await self.repo.delete_folder(folder_id)
            return
        raise FolderNotFoundError()

    async def folder_exists(self, folder_id: str) -> None:
        if not any(f.id == folder_id for f in await self.repo.list_folders()):
            raise FolderNotFoundError()

    async def writable_folder(self, folder_id: str) -> None:
        """Thư mục lưu hình vào được: có thật và không phải Hình mẫu, chỉ "Đặt làm mẫu" mới điền vào đó."""
        if folder_id == EXEMPLAR_FOLDER_ID:
            raise ExemplarFolderError()
        await self.folder_exists(folder_id)

    # --- đọc ---

    async def search(self, flt: IllustrationFilter) -> list[Illustration]:
        return await self.repo.list_illustrations(flt)

    async def get(self, illustration_id: str) -> Illustration:
        return await self.repo.get_illustration(illustration_id)

    async def exemplars(self) -> list[Illustration]:
        """Hình mẫu, cũ nhất trước: thứ tự bộ vẽ AI đọc chúng."""
        rows = await self.repo.list_illustrations(IllustrationFilter(folder_id=EXEMPLAR_FOLDER_ID))
        return sorted((r for r in rows if r.exemplar), key=lambda r: (r.created_at, r.name))

    # --- dựng ---

    async def kit_for(self, ref_id: str) -> dict:
        if not ref_id:
            raise RefStyleRequiredError()
        return await self._references.get_kit(ref_id)

    async def render(
        self, ref_id: str, name: str, code: str, kind: IllustrationKind, tier: str,
        kit: dict | None = None,
    ) -> RenderedIllustration:
        """Kiểm tra và dựng `code` thành hình hay nền (`kind`); bị từ chối là InvalidIllustrationError."""
        if kit is None:
            kit = await self.kit_for(ref_id)
        out = await self._renderer.render(kit, name, code, kind_or_figure(kind).value, tier_or_support(tier))
        if not out.ok:
            raise InvalidIllustrationError(out.diagnostics)
        return out

    async def save_preview(self, illustration_id: str, version: int, out: RenderedIllustration) -> None:
        png_path, gif_path = await asyncio.to_thread(
            self._files.save, illustration_id, version, out.png, out.gif)
        await self.repo.set_preview(illustration_id, version, png_path, gif_path)

    async def try_render(
        self, name: str, code: str, kind: IllustrationKind, detail: DetailLevel | str, ref_id: str,
    ) -> RenderedIllustration:
        """Dựng thử không lưu gì — nút "Xem trước" của trình soạn."""
        name = name.strip()
        validate_illustration_name(name)
        return await self.render(ref_id, name, code, kind, tier_for_detail(detail))

    # --- tạo, sửa, duyệt, xoá ---

    async def create(self, i: Illustration, ref_id: str) -> Illustration:
        """Lưu hình mới thành bản nháp. Code phải qua bộ dựng trước: thư viện không giữ hình hỏng."""
        i = normalize_illustration(i)
        await self.writable_folder(i.folder_id)
        detail = detail_or_medium(i.detail_level)
        out = await self.render(ref_id, i.name, i.code, i.kind, tier_for_detail(detail))
        saved = await self.repo.create_illustration(i.with_(
            kind=kind_or_figure(i.kind), detail_level=detail, ref_id=ref_id, builtin=False, exemplar=False,
            status=IllustrationStatus.DRAFT, version=1, source_id="", home_folder_id="",
            warnings=tuple(out.warnings)))
        await self.save_preview(saved.id, saved.version, out)
        return saved.with_(has_preview=True)

    async def update(self, illustration_id: str, i: Illustration, ref_id: str = "") -> Illustration:
        """Sửa một hình. Đổi code về lại bản nháp với phiên bản mới, vì thứ Creator đã duyệt không còn
        là thứ sẽ dựng."""
        existing = await self.repo.get_illustration(illustration_id)
        if existing.read_only:
            raise IllustrationReadOnlyError()
        code = i.code if i.code.strip() else existing.code
        i = i.with_(code=code, folder_id=i.folder_id or existing.folder_id, name=i.name or existing.name)
        i = normalize_illustration(i)
        await self.writable_folder(i.folder_id)
        # Phong cách và mức chi tiết của hình chốt ở lần vẽ đầu.
        i = i.with_(
            id=illustration_id, builtin=False, exemplar=False, ref_id=existing.ref_id,
            detail_level=existing.detail_level, status=existing.status, version=existing.version,
            warnings=existing.warnings, source_id=existing.source_id, home_folder_id=existing.home_folder_id,
            kind=existing.kind)
        code_changed = i.code != existing.code or i.name != existing.name
        out: RenderedIllustration | None = None
        if code_changed:
            if not existing.ref_id:
                i = i.with_(ref_id=ref_id)
            out = await self.render(
                i.ref_id, i.name, i.code, i.kind, tier_for_detail(i.detail_level))
            i = i.with_(status=IllustrationStatus.DRAFT, version=existing.version + 1,
                        warnings=tuple(out.warnings))
        saved = await self.repo.update_illustration(i)
        if out is not None:
            await self.save_preview(illustration_id, saved.version, out)
            saved = saved.with_(has_preview=True)
        return saved

    async def set_status(self, illustration_id: str, status: str) -> Illustration:
        """Duyệt một bản nháp, hoặc trả hình đã duyệt về nháp."""
        existing = await self.repo.get_illustration(illustration_id)
        if existing.read_only:
            raise IllustrationReadOnlyError()
        if status not in (IllustrationStatus.DRAFT, IllustrationStatus.APPROVED):
            raise IllustrationError(f'trạng thái "{status}" không hợp lệ')
        return await self.repo.update_illustration(existing.with_(status=IllustrationStatus(status)))

    async def delete(self, illustration_id: str) -> None:
        existing = await self.repo.get_illustration(illustration_id)
        if existing.read_only:
            raise IllustrationReadOnlyError()
        await self._remove(existing)

    async def _remove(self, illustration: Illustration) -> None:
        await self.repo.delete_illustration(illustration.id)
        await asyncio.to_thread(self._files.remove, illustration.id)

    # --- Hình mẫu ---

    async def make_exemplar(self, illustration_id: str) -> Illustration:
        """Sao một hình đã duyệt vào thư mục Hình mẫu dưới tên riêng (<Tên>Mau), dựng như mọi hình
        trước khi giữ."""
        src = await self.repo.get_illustration(illustration_id)
        if src.exemplar:
            raise AlreadyExemplarError()
        if src.builtin or src.status != IllustrationStatus.APPROVED or not src.code.strip():
            raise NotExemplarSourceError()
        taken: set[str] = set()
        exemplars = 0
        for row in await self.repo.list_illustrations(IllustrationFilter()):
            taken.add(row.name)
            if row.exemplar:
                exemplars += 1
                if row.source_id == src.id:
                    raise AlreadyExemplarError()
        if exemplars >= MAX_EXEMPLARS:
            raise ExemplarLimitError()
        name = exemplar_copy_name(src.name, taken)
        code, usage = rename_export(src.code, src.usage, name)
        out = await self.render(src.ref_id, name, code, src.kind, tier_for_detail(src.detail_level))
        saved = await self.repo.create_exemplar(Illustration(
            name=name, title=src.title, folder_id=EXEMPLAR_FOLDER_ID, tags=src.tags,
            description=src.description, usage=usage, code=code, exemplar=True, source_id=src.id,
            kind=src.kind, ref_id=src.ref_id, detail_level=src.detail_level,
            status=IllustrationStatus.APPROVED, version=1, warnings=tuple(out.warnings)), MAX_EXEMPLARS)
        await self.save_preview(saved.id, saved.version, out)
        return saved.with_(has_preview=True)

    async def unmake_exemplar(self, illustration_id: str) -> Illustration | None:
        """Đưa một hình ra khỏi Hình mẫu. Bản sao bị xoá (hình gốc vẫn ở thư viện, trả None); hình mẫu
        gốc về thư mục cũ thành hình thường và được trả lại."""
        ill = await self.repo.get_illustration(illustration_id)
        if not ill.exemplar:
            raise NotExemplarError()
        if not ill.home_folder_id:
            await self._remove(ill)
            return None
        try:
            await self.folder_exists(ill.home_folder_id)
        except FolderNotFoundError as exc:
            raise IllustrationError(
                f'thư mục cũ "{ill.home_folder_id}" của hình mẫu không còn: {exc}') from exc
        return await self.repo.release_exemplar(illustration_id, ill.home_folder_id)

    # --- xem trước ---

    async def preview(self, illustration_id: str, ref_id: str = "") -> tuple[bytes, bytes]:
        """PNG và GIF đã lưu; chưa có cho phiên bản hiện tại thì dựng trước (hình có sẵn dựng lúc xem
        đầu tiên). `ref_id` chỉ dùng cho hình chưa gắn phong cách."""
        png_path, gif_path = await self.repo.get_preview(illustration_id)
        if png_path:
            png = await asyncio.to_thread(self._files.read, png_path)
            gif = await asyncio.to_thread(self._files.read, gif_path) if gif_path else b""
            return png, gif
        return await self.rerender(illustration_id, ref_id)

    async def rerender(self, illustration_id: str, ref_id: str = "") -> tuple[bytes, bytes]:
        """Dựng lại ảnh của phiên bản hiện tại và lưu."""
        existing = await self.repo.get_illustration(illustration_id)
        # Hình có sẵn của kit không có code (code đi cùng image); Hình mẫu mang code riêng.
        out = await self.render(
            existing.ref_id or ref_id, existing.name, existing.code, existing.kind,
            tier_for_detail(existing.detail_level))
        await self.save_preview(illustration_id, existing.version, out)
        return out.png, out.gif

