"""Storyboard ở rendering: dựng khung của một shot, chốt bản, đọc bản chốt và xoá cả dự án.

Dựng khung đặt các tài sản đã duyệt lên lưới vị trí qua bộ dựng xem trước (cùng hàng đợi một yêu cầu một
lúc với xem trước hình). Khoá cache là `spec_hash` do người gọi tính: cùng `spec_hash` và file còn thì trả
khung cũ, không vẽ lại. Xem ADR-0051.
"""

from __future__ import annotations

import asyncio
import hashlib
from dataclasses import dataclass
from typing import Protocol

from adapters.rendering.illustration_previewer import IllustrationPreviewError
from domain.project_asset import AssetNotFoundError, AssetStatus, ProjectAsset
from domain.storyboard import (
    AssetsNotReadyError,
    FrameNotFoundError,
    FrameRenderError,
    FrameRequest,
    InvalidFrameRequestError,
    MissingFramesError,
    ShotFrame,
    StoryboardVersion,
    VersionConflictError,
    VersionNotFoundError,
    document_shot_ids,
    frame_relative_path,
    version_frame_path,
)


class StoryboardRepoPort(Protocol):
    async def get_shot(self, project_id: str, shot_id: str) -> ShotFrame | None: ...

    async def upsert_shot(self, shot: ShotFrame) -> ShotFrame: ...

    async def next_version(self, project_id: str) -> int: ...

    async def insert_version(self, version: StoryboardVersion) -> StoryboardVersion:
        """Ném VersionConflictError khi số phiên bản đã có."""

    async def latest_version(self, project_id: str) -> StoryboardVersion | None: ...

    async def get_version(self, project_id: str, version: int) -> StoryboardVersion | None: ...

    async def delete_project(self, project_id: str) -> None: ...


class StoryboardAssetsPort(Protocol):
    async def get_asset(self, project_id: str, asset_id: str) -> ProjectAsset:
        """Ném AssetNotFoundError khi không có."""


class StoryboardFilesPort(Protocol):
    def begin_frame(self, project_id: str, shot_id: str) -> str: ...

    def commit_frame(self, project_id: str, shot_id: str, tmp_path: str) -> str: ...

    def discard(self, tmp_path: str) -> None: ...

    def exists(self, project_id: str, relative: str) -> bool: ...

    def read(self, project_id: str, relative: str) -> bytes: ...

    def copy_to_version(self, project_id: str, shot_id: str, version: int) -> str: ...

    def version_exists(self, project_id: str, version: int) -> bool: ...

    def remove_version(self, project_id: str, version: int) -> None: ...

    def write_document(self, project_id: str, document: dict) -> None: ...

    def remove_project(self, project_id: str) -> None: ...


class FrameRendererPort(Protocol):
    def frame(self, spec: dict, output: str) -> None:
        """Dựng khung ra file PNG `output`; ném IllustrationPreviewError khi bộ dựng không chạy được."""


@dataclass(frozen=True)
class FrameResult:
    """Kết quả dựng khung: bản ghi shot và có lấy từ cache không."""

    shot: ShotFrame
    cache_hit: bool

    def to_json(self) -> dict:
        return {**self.shot.to_json(), "cache_hit": self.cache_hit}


class Storyboard:
    def __init__(
        self, repo: StoryboardRepoPort, assets: StoryboardAssetsPort, files: StoryboardFilesPort,
        renderer: FrameRendererPort, gate: asyncio.Semaphore,
    ) -> None:
        self._repo = repo
        self._assets = assets
        self._files = files
        self._renderer = renderer
        self._gate = gate

    async def draw_frame(self, request: FrameRequest) -> FrameResult:
        """Dựng (hoặc lấy từ cache) khung của một shot.

        Tài sản thiếu hay chưa duyệt → AssetsNotReadyError, không ghi gì. Bộ dựng hỏng → bản ghi shot mang
        lỗi và không có khung, rồi ném FrameRenderError.
        """
        assets = await self._approved_assets(request)
        previous = await self._repo.get_shot(request.project_id, request.shot_id)
        if (previous and previous.spec_hash == request.spec_hash and previous.frame_path
                and self._files.exists(request.project_id, previous.frame_path)):
            return FrameResult(previous, cache_hit=True)
        tmp = self._files.begin_frame(request.project_id, request.shot_id)
        try:
            async with self._gate:
                await asyncio.to_thread(self._renderer.frame, _preview_spec(request, assets), tmp)
        except IllustrationPreviewError as exc:
            self._files.discard(tmp)
            await self._record_failure(request, str(exc))
            raise FrameRenderError(f"dựng khung {request.shot_id}: {exc}") from exc
        relative = self._files.commit_frame(request.project_id, request.shot_id, tmp)
        digest = hashlib.sha256(self._files.read(request.project_id, relative)).hexdigest()
        shot = await self._repo.upsert_shot(ShotFrame(
            project_id=request.project_id, shot_id=request.shot_id, unit_id=request.unit_id,
            beat_id=request.beat_id, frame_path=relative, frame_hash=digest, spec_hash=request.spec_hash,
            drawn_at=""))
        return FrameResult(shot, cache_hit=False)

    async def read_frame(self, project_id: str, shot_id: str) -> bytes:
        """PNG của khung đã vẽ gần nhất; chưa vẽ hoặc file mất → FrameNotFoundError."""
        shot = await self._repo.get_shot(project_id, shot_id)
        if shot is None or not shot.frame_path or not self._files.exists(project_id, shot.frame_path):
            raise FrameNotFoundError()
        return self._files.read(project_id, shot.frame_path)

    async def lock(
        self, project_id: str, document: object, sequence_version: int, note: str,
    ) -> StoryboardVersion:
        """Ghi bản chốt mới: sao khung từng shot sang `v{n}/`, sửa `frame_path` trong tài liệu, lưu và ghi
        `storyboard.json`.

        Khung của shot nào chưa có file → MissingFramesError (không ghi gì).
        """
        shot_ids = document_shot_ids(document)
        assert isinstance(document, dict)
        missing = [s for s in shot_ids if not self._files.exists(project_id, frame_relative_path(s))]
        if missing:
            raise MissingFramesError(missing)
        number = await self._repo.next_version(project_id)
        if self._files.version_exists(project_id, number):
            raise VersionConflictError(number)
        try:
            for shot_id in shot_ids:
                self._files.copy_to_version(project_id, shot_id, number)
            stored = {**document, "shots": [
                {**shot, "frame_path": version_frame_path(number, shot["shot_id"])}
                for shot in document["shots"]]}
            version = await self._repo.insert_version(StoryboardVersion(
                project_id=project_id, version=number, document=stored, sequence_version=sequence_version,
                note=note, locked_at=""))
        except VersionConflictError:
            # The folder now belongs to whoever took this number; leave it alone.
            raise
        except BaseException:
            self._files.remove_version(project_id, number)
            raise
        self._files.write_document(project_id, version.document)
        return version

    async def latest_version(self, project_id: str) -> StoryboardVersion:
        version = await self._repo.latest_version(project_id)
        if version is None:
            raise VersionNotFoundError()
        return version

    async def get_version(self, project_id: str, number: int) -> StoryboardVersion:
        version = await self._repo.get_version(project_id, number)
        if version is None:
            raise VersionNotFoundError()
        return version

    async def delete_project(self, project_id: str) -> None:
        """Xoá tài sản, khung, bản chốt của dự án và thư mục `/shared/{project_id}/storyboard`.

        Chạy lại vẫn đúng (không có gì để xoá thì không phải lỗi).
        """
        await self._repo.delete_project(project_id)
        self._files.remove_project(project_id)

    async def _approved_assets(self, request: FrameRequest) -> dict[str, ProjectAsset]:
        found: dict[str, ProjectAsset] = {}
        problems: list[dict[str, str]] = []
        for asset_id in request.asset_ids:
            try:
                asset = await self._assets.get_asset(request.project_id, asset_id)
            except AssetNotFoundError:
                problems.append({"asset_id": asset_id, "reason": "không tìm thấy"})
                continue
            if asset.status is not AssetStatus.APPROVED or not asset.code:
                problems.append({"asset_id": asset_id, "reason": f"chưa duyệt ({asset.status.value})"})
                continue
            found[asset_id] = asset
        if problems:
            raise AssetsNotReadyError(problems)
        background = found.get(request.background_asset_id)
        if background is not None and background.kind != "backdrop":
            raise InvalidFrameRequestError(f"nền {background.asset_id} không phải tài sản backdrop")
        return found

    async def _record_failure(self, request: FrameRequest, message: str) -> None:
        await self._repo.upsert_shot(ShotFrame(
            project_id=request.project_id, shot_id=request.shot_id, unit_id=request.unit_id,
            beat_id=request.beat_id, frame_path=None, frame_hash="", spec_hash=request.spec_hash,
            drawn_at="", error=message))


def _preview_spec(request: FrameRequest, assets: dict[str, ProjectAsset]) -> dict:
    """Thân lệnh `frame` của bộ dựng: code của từng tài sản, vị trí, hạng (từ tài sản), chữ và nền."""
    background: dict = {"color": request.background_color}
    if request.background_asset_id:
        backdrop = assets[request.background_asset_id]
        background["backdrop"] = {"code": backdrop.code, "name": backdrop.name}
    return {
        "kit": request.kit,
        "background": background,
        "placements": [
            {"code": assets[p.asset_id].code, "name": assets[p.asset_id].name, "position": p.position,
             "tier": assets[p.asset_id].tier}
            for p in request.placements],
        "camera_view": request.camera_view,
        "on_screen_text": [{"text": line.text, "min_px": line.min_px} for line in request.text],
        "text_color": request.text_color,
    }
