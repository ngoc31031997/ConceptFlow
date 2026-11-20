"""Tài sản của dự án ở bước Storyboard: lập, sửa, vẽ bằng bộ vẽ AI, duyệt và đưa vào thư viện hình.

Rendering không biết shot nào dùng tài sản: việc từ chối xoá tài sản đang được shot dùng thuộc về
visual-service, nơi giữ các shot.
"""

from __future__ import annotations

import asyncio
import re
from typing import Protocol

from application.illustration_drawer import CodeDrawRequest, IllustrationDrawer
from application.illustration_library import IllustrationLibrary
from application.illustration_ports import RenderedIllustration
from domain.illustration import (
    DrawerDisabledError,
    FolderTakenError,
    Illustration,
    IllustrationError,
    IllustrationFolder,
    IllustrationKind,
    IllustrationStatus,
    detail_for_tier,
    normalize_tags,
    rename_export,
    validate_illustration_name,
)
from domain.kit import validate_kit
from domain.project_asset import (
    PROJECT_FOLDER_ID,
    PROJECT_FOLDER_NAME,
    PROJECT_SHORT_ID_LEN,
    AssetNotFoundError,
    AssetStateError,
    AssetStatus,
    ProjectAsset,
    apply_edit,
    describe_for_drawing,
    ensure_can_approve,
    ensure_can_draw,
    normalize_asset,
    validate_asset_id,
    validate_project_id,
)


class ProjectAssetRepoPort(Protocol):
    async def list_assets(self, project_id: str) -> list[ProjectAsset]: ...

    async def get_asset(self, project_id: str, asset_id: str) -> ProjectAsset:
        """Ném AssetNotFoundError khi không có."""

    async def create_asset(self, asset: ProjectAsset) -> ProjectAsset:
        """Cấp mã `AST-<TÊN>-<số>` từ bộ đếm theo dự án rồi lưu thành `planned`."""

    async def update_asset(self, asset: ProjectAsset) -> ProjectAsset: ...

    async def delete_asset(self, project_id: str, asset_id: str) -> None: ...


class ProjectAssetFilesPort(Protocol):
    def save(self, project_id: str, asset_id: str, version: int, png: bytes) -> str:
        """Ghi PNG của một phiên bản (file tạm rồi đổi tên), trả đường dẫn."""

    def read(self, path: str) -> bytes: ...

    def remove(self, project_id: str, asset_id: str) -> None:
        """Xoá thư mục ảnh của tài sản; không có thì không phải lỗi."""


def _short_project_id(project_id: str) -> str:
    return re.sub(r"[^A-Za-z0-9]", "", project_id)[:PROJECT_SHORT_ID_LEN]


class ProjectAssets:
    def __init__(
        self, repo: ProjectAssetRepoPort, files: ProjectAssetFilesPort, library: IllustrationLibrary,
        drawer: IllustrationDrawer | None,
    ) -> None:
        self._repo = repo
        self._files = files
        self._library = library
        self._drawer = drawer

    # --- đọc, lập, sửa, xoá ---

    async def list_assets(self, project_id: str) -> list[ProjectAsset]:
        validate_project_id(project_id)
        return await self._repo.list_assets(project_id)

    async def get(self, project_id: str, asset_id: str) -> ProjectAsset:
        validate_project_id(project_id)
        validate_asset_id(asset_id)
        return await self._repo.get_asset(project_id, asset_id)

    async def preview(self, project_id: str, asset_id: str) -> bytes:
        """PNG xem trước hiện tại; ném AssetNotFoundError khi tài sản chưa có ảnh hoặc file đã mất."""
        asset = await self.get(project_id, asset_id)
        if not asset.preview_path:
            raise AssetNotFoundError()
        try:
            return await asyncio.to_thread(self._files.read, asset.preview_path)
        except FileNotFoundError as exc:
            raise AssetNotFoundError() from exc

    async def create(self, asset: ProjectAsset) -> ProjectAsset:
        """Lập tài sản mới ở `planned`; mã do máy cấp."""
        asset = normalize_asset(asset)
        await self._check_base(asset)
        return await self._repo.create_asset(asset)

    async def update(self, project_id: str, asset_id: str, edited: ProjectAsset) -> ProjectAsset:
        existing = await self.get(project_id, asset_id)
        merged = apply_edit(existing, edited.with_(project_id=project_id))
        await self._check_base(merged)
        return await self._repo.update_asset(merged)

    async def delete(self, project_id: str, asset_id: str) -> None:
        """Xoá tài sản và ảnh xem trước. Hình đã đưa vào thư viện được giữ."""
        await self.get(project_id, asset_id)
        await self._repo.delete_asset(project_id, asset_id)
        await asyncio.to_thread(self._files.remove, project_id, asset_id)

    # --- vẽ ---

    async def draw(
        self, project_id: str, asset_id: str, kit: object, ref_id: str, model: str,
    ) -> ProjectAsset:
        """Vẽ tài sản từ mô tả ghép của nó, với kit của dự án; xong thành `drawn` và tăng phiên bản."""
        return await self._draw(project_id, asset_id, kit, ref_id, model, note="", redraw=False)

    async def redraw(
        self, project_id: str, asset_id: str, kit: object, ref_id: str, model: str, note: str,
    ) -> ProjectAsset:
        """Vẽ lại từ code đang có theo ghi chú của Creator; tăng phiên bản, trạng thái về `drawn`."""
        return await self._draw(project_id, asset_id, kit, ref_id, model, note=note, redraw=True)

    async def _draw(
        self, project_id: str, asset_id: str, kit: object, ref_id: str, model: str, *, note: str,
        redraw: bool,
    ) -> ProjectAsset:
        if self._drawer is None:
            raise DrawerDisabledError()
        asset = await self.get(project_id, asset_id)
        if redraw and not asset.code.strip():
            raise AssetStateError("tài sản chưa có hình để vẽ lại — hãy vẽ trước")
        if not redraw:
            ensure_can_draw(asset)
        valid_kit = validate_kit(kit)
        base_code, base_label = await self._base_reference(asset)
        reply, out = await self._drawer.draw_code(CodeDrawRequest(
            description=describe_for_drawing(asset), name=asset.name, ref_id=ref_id, kit=valid_kit,
            kind=IllustrationKind(asset.kind), tier=asset.tier, model=model,
            current_code=asset.code if redraw else "", note=note, base_code=base_code,
            base_label=base_label))
        version = asset.version + 1
        path = await asyncio.to_thread(self._files.save, project_id, asset_id, version, out.png)
        return await self._repo.update_asset(asset.with_(
            code=reply.code, version=version, preview_path=path, status=AssetStatus.DRAWN,
            warnings=tuple(out.warnings)))

    async def _check_base(self, asset: ProjectAsset) -> None:
        """Hình gốc phải có trong thư viện và cùng loại với tài sản."""
        if not asset.base_illustration_id:
            return
        base = await self._library.get(asset.base_illustration_id)
        if base.kind.value != asset.kind:
            raise IllustrationError(
                f"hình gốc {base.name} là {base.kind.value}, tài sản là {asset.kind}: phải cùng loại")

    async def _base_reference(self, asset: ProjectAsset) -> tuple[str, str]:
        """Code và nhãn của hình gốc làm Hình mẫu; hình có sẵn của kit không có code thì không dùng được."""
        if not asset.base_illustration_id:
            return "", ""
        base = await self._library.get(asset.base_illustration_id)
        return base.code, f"{base.name} — {base.title}"

    # --- duyệt ---

    async def approve(self, project_id: str, asset_id: str, project_name: str, ref_id: str) -> ProjectAsset:
        """Duyệt một tài sản đã vẽ; nếu `to_library` thì đưa hình vào thư viện."""
        asset = await self.get(project_id, asset_id)
        ensure_can_approve(asset)
        library_id = asset.library_illustration_id
        if asset.to_library:
            library_id = await self._to_library(asset, project_name, ref_id)
        return await self._repo.update_asset(
            asset.with_(status=AssetStatus.APPROVED, library_illustration_id=library_id))

    async def unapprove(self, project_id: str, asset_id: str) -> ProjectAsset:
        """Bỏ duyệt, về `drawn`. Hình đã vào thư viện được giữ nguyên."""
        asset = await self.get(project_id, asset_id)
        if asset.status != AssetStatus.APPROVED:
            raise AssetStateError("tài sản chưa được duyệt")
        return await self._repo.update_asset(asset.with_(status=AssetStatus.DRAWN))

    async def _library_folder(self, asset: ProjectAsset) -> str:
        """Thư mục đích; thư mục mặc định "Từ dự án" được tạo khi chưa có."""
        folder_id = asset.library_folder_id or PROJECT_FOLDER_ID
        if folder_id != PROJECT_FOLDER_ID:
            await self._library.writable_folder(folder_id)
            return folder_id
        if not any(f.id == folder_id for f in await self._library.folders()):
            try:
                await self._library.create_folder(
                    IllustrationFolder(id=folder_id, name=PROJECT_FOLDER_NAME,
                                       description="hình do các dự án vẽ và duyệt"))
            except FolderTakenError:
                pass  # một lần duyệt khác vừa tạo thư mục này
        return folder_id

    async def _to_library(self, asset: ProjectAsset, project_name: str, ref_id: str) -> str:
        """Tạo (hoặc cập nhật, nếu tài sản đã đưa vào trước đó) hình thư viện từ code và ảnh của tài sản.

        Tên hàm export là `{tên}{8 ký tự đầu của mã dự án}` để không đụng hình cùng tên của dự án khác,
        còn tiêu đề là `{tên}-{8 ký tự đầu}`.
        """
        folder_id = await self._library_folder(asset)
        short = _short_project_id(asset.project_id)
        lib_name = f"{asset.name}{short}"
        validate_illustration_name(lib_name)
        code, usage = rename_export(asset.code, "", lib_name)
        png = await asyncio.to_thread(self._files.read, asset.preview_path)
        description = f"{asset.portrayal} (tài sản của dự án {project_name.strip() or asset.project_id})"
        row = Illustration(
            name=lib_name, title=f"{asset.name}-{short}", folder_id=folder_id,
            tags=normalize_tags(["du-an", asset.concept_id]), description=description, usage=usage,
            code=code, kind=IllustrationKind(asset.kind), ref_id=ref_id,
            detail_level=detail_for_tier(asset.tier), status=IllustrationStatus.APPROVED, version=1,
            warnings=asset.warnings)
        out = RenderedIllustration(ok=True, png=png)
        existing = await self._existing_library_row(asset)
        if existing is None:
            saved = await self._library.repo.create_illustration(row)
        else:
            saved = await self._library.repo.update_illustration(row.with_(
                id=existing.id, version=existing.version + 1))
        await self._library.save_preview(saved.id, saved.version, out)
        return saved.id

    async def _existing_library_row(self, asset: ProjectAsset) -> Illustration | None:
        if not asset.library_illustration_id:
            return None
        try:
            row = await self._library.get(asset.library_illustration_id)
        except IllustrationError:
            return None  # hình đã bị Creator xoá khỏi thư viện: tạo lại
        return None if row.read_only else row
