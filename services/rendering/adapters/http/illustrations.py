"""Route của thư viện hình minh hoạ (api-gateway chuyển nguyên request tới đây).

Thân yêu cầu và trả về là JSON; lỗi là `{"error": "<thông báo>"}`. Hợp đồng ở
`docs/contracts/illustration-library-v1.md`.
"""

from __future__ import annotations

import base64
import json
import logging
from collections.abc import Callable
from datetime import datetime
from typing import Any

from fastapi import APIRouter, Request, Response
from fastapi.responses import JSONResponse
from pydantic import BaseModel, ValidationError

from application.illustration_backup import IllustrationBackup
from application.illustration_drawer import DrawRequest, IllustrationDrawer
from application.illustration_library import IllustrationLibrary
from application.illustration_ports import ReferenceSourcePort
from domain.illustration import (
    MAX_EXEMPLARS,
    AlreadyExemplarError,
    BackupInvalidError,
    DrawerDisabledError,
    ExemplarFolderError,
    ExemplarLimitError,
    FolderNotEmptyError,
    FolderNotFoundError,
    FolderReadOnlyError,
    FolderTakenError,
    Illustration,
    IllustrationError,
    IllustrationFilter,
    IllustrationFolder,
    IllustrationKind,
    IllustrationNameTakenError,
    IllustrationNotFoundError,
    IllustrationReadOnlyError,
    InvalidIllustrationError,
    LLMError,
    NotExemplarError,
    NotExemplarSourceError,
    PromptUnavailableError,
    ReferenceKitMissingError,
    ReferenceServiceError,
    ReferenceStyleNotFoundError,
    RendererUnavailableError,
    kind_or_figure,
    tier_for_detail,
)
from domain.kit import InvalidKitError

logger = logging.getLogger(__name__)

#: Trần của file sao lưu tải lên: code của mọi hình cộng PNG và GIF của từng hình.
MAX_BACKUP_BYTES = 256 << 20


def _error(status: int, message: str) -> JSONResponse:
    return JSONResponse({"error": message}, status_code=status)


def _llm_error(err: LLMError) -> JSONResponse:
    """Câu cho Creator và mã trạng thái của một cuộc gọi model hỏng."""
    cause = err.message + (f" (mã cuộc gọi {err.call_id})" if err.call_id else "")
    match err.kind:
        case "not_configured":
            return _error(503, "Chưa cấu hình HIVE_API_KEY trong .env (llm-service) nên không gọi được AI.")
        case "auth":
            return _error(502, "API key bị từ chối — kiểm tra lại HIVE_API_KEY trong .env (llm-service).")
        case "balance":
            return _error(502, "Tài khoản Hive hết số dư — nạp thêm ở dashboard Hive.")
        case "rate_limit":
            return _error(429, "Hive đang chặn vì gọi quá nhanh — chờ một lát rồi thử lại.")
        case "timeout":
            return _error(504, "AI không trả lời trong thời gian cho phép — thử lại, hoặc tăng "
                               "HIVE_TIMEOUT_SECONDS / LLM_SERVICE_TIMEOUT_SECONDS.")
        case "budget" | "truncated":
            return _error(502, "Câu trả lời bị cắt vì hết hạn mức token — tăng HIVE_MAX_OUTPUT_TOKENS rồi "
                               "chạy lại. Chi tiết nằm ở nhật ký cuộc gọi AI.")
        case "empty":
            return _error(502, "AI trả về rỗng — thử chạy lại, hoặc sửa lời prompt ở trang Prompt.")
        case "malformed":
            return _error(502, f"Kết quả AI không dùng được: {cause}")
        case "server":
            if err.provider == "llm-service":
                return _error(502, f"Không gọi được llm-service ({cause}) — kiểm tra container llm-service.")
            return _error(502, "Nhà cung cấp AI đang lỗi phía họ — thử lại sau.")
    return _error(400, str(err))


def illustration_error(exc: Exception) -> JSONResponse:
    """Trạng thái HTTP và thân lỗi cho một lỗi của thư viện hình."""
    if isinstance(exc, ExemplarLimitError | AlreadyExemplarError | NotExemplarError):
        return _error(409, str(exc))
    if isinstance(exc, ExemplarFolderError | NotExemplarSourceError):
        return _error(422, str(exc))
    if isinstance(exc, BackupInvalidError):
        return _error(400, str(exc))
    unavailable = (
        DrawerDisabledError | PromptUnavailableError | RendererUnavailableError | ReferenceServiceError)
    if isinstance(exc, unavailable):
        return _error(503, str(exc))
    if isinstance(exc, LLMError):
        return _llm_error(exc)
    if isinstance(exc, InvalidIllustrationError):
        return JSONResponse(
            {"error": str(exc), "diagnostics": [d.to_json() for d in exc.diagnostics]}, status_code=422)
    if isinstance(exc, IllustrationNotFoundError | FolderNotFoundError | ReferenceStyleNotFoundError):
        return _error(404, str(exc))
    if isinstance(exc, IllustrationReadOnlyError | FolderReadOnlyError):
        return _error(403, str(exc))
    conflict = IllustrationNameTakenError | FolderTakenError | FolderNotEmptyError | ReferenceKitMissingError
    if isinstance(exc, conflict):
        return _error(409, str(exc))
    if isinstance(exc, InvalidKitError):
        return _error(400, str(exc))
    return _error(400, str(exc))


class _Body(BaseModel):
    """Thân tolerant: trường lạ bị bỏ, trường thiếu lấy mặc định, kiểu sai là "invalid JSON body"."""


class IllustrationBody(_Body):
    name: str = ""
    title: str = ""
    folder_id: str = ""
    tags: list[str] | None = None
    description: str = ""
    usage: str = ""
    code: str = ""
    detail_level: str = ""
    kind: str = ""
    ref_id: str = ""


class FolderBody(_Body):
    id: str = ""
    name: str = ""
    description: str = ""


class TryBody(_Body):
    name: str = ""
    code: str = ""
    kind: str = ""
    detail_level: str = ""
    ref_id: str = ""


class StatusBody(_Body):
    status: str = ""


class DrawBody(_Body):
    description: str = ""
    folder_id: str = ""
    name: str = ""
    model: str = ""
    kind: str = ""
    detail_level: str = ""
    ref_id: str = ""


class RedrawBody(_Body):
    note: str = ""
    model: str = ""
    ref_id: str = ""


async def _decode(request: Request, model: type[_Body]) -> Any:
    try:
        raw = json.loads(await request.body())
        return model.model_validate(raw)
    except (ValueError, ValidationError):
        return None


def _illustration_from(body: IllustrationBody) -> Illustration:
    return Illustration(
        name=body.name, title=body.title, folder_id=body.folder_id, tags=tuple(body.tags or ()),
        description=body.description, usage=body.usage, code=body.code,
        detail_level=body.detail_level, kind=_kind(body.kind))


def _kind(text: str) -> IllustrationKind:
    return kind_or_figure(text)


def _preview_json(png: bytes, gif: bytes) -> dict:
    return {"png": base64.b64encode(png).decode("ascii"), "gif": base64.b64encode(gif).decode("ascii")}


def create_illustration_router(
    library: IllustrationLibrary, drawer: IllustrationDrawer | None, backup: IllustrationBackup,
    references: ReferenceSourcePort, clock: Callable[[], datetime] = datetime.now,
) -> APIRouter:
    router = APIRouter()
    bad_json = _error(400, "invalid JSON body")

    @router.get("/v1/illustration-folders")
    async def list_folders():
        try:
            return {"folders": [f.to_json() for f in await library.folders()]}
        except Exception as exc:
            return _failure(exc)

    @router.post("/v1/admin/illustration-folders")
    async def create_folder(request: Request):
        body = await _decode(request, FolderBody)
        if body is None:
            return bad_json
        try:
            out = await library.create_folder(IllustrationFolder(
                id=body.id, name=body.name, description=body.description))
        except Exception as exc:
            return _failure(exc)
        return JSONResponse(out.to_json(), status_code=201)

    @router.delete("/v1/admin/illustration-folders/{folder_id}")
    async def delete_folder(folder_id: str):
        try:
            await library.delete_folder(folder_id)
        except Exception as exc:
            return _failure(exc)
        return Response(status_code=204)

    @router.get("/v1/illustration-style")
    async def illustration_style(ref_id: str = ""):
        """Id các Hình mẫu (để web-gui hiện kèm ảnh) và, khi có `ref_id`, mô tả của phong cách đó."""
        try:
            exemplar_ids = [e.id for e in await library.exemplars()]
            full = await references.get_style(ref_id) if ref_id else None
            style = None if full is None else {
                key: full.get(key) for key in ("id", "name", "summary", "drawing", "layout")}
        except Exception as exc:
            return _failure(exc)
        return {"exemplar_ids": exemplar_ids, "max_exemplars": MAX_EXEMPLARS, "style": style}

    @router.get("/v1/illustrations")
    async def list_illustrations(folder: str = "", q: str = "", status: str = ""):
        try:
            rows = await library.search(IllustrationFilter(folder_id=folder, query=q, status=status))
        except Exception as exc:
            return _failure(exc)
        return {"illustrations": [r.to_json() for r in rows]}

    @router.get("/v1/illustrations/{illustration_id}")
    async def get_illustration(illustration_id: str):
        try:
            return (await library.get(illustration_id)).to_json()
        except Exception as exc:
            return _failure(exc)

    async def serve_preview(illustration_id: str, ref_id: str, kind: str):
        try:
            png, gif = await library.preview(illustration_id, ref_id)
        except Exception as exc:
            return _failure(exc)
        body, mime = (gif, "image/gif") if kind == "gif" else (png, "image/png")
        if not body:
            return _error(404, f"no {kind} preview for this illustration")
        # URL mang ?v=<phiên bản> nên cache được.
        return Response(body, media_type=mime, headers={"Cache-Control": "private, max-age=86400"})

    @router.get("/v1/illustrations/{illustration_id}/preview.png")
    async def preview_png(illustration_id: str, ref_id: str = ""):
        return await serve_preview(illustration_id, ref_id, "png")

    @router.get("/v1/illustrations/{illustration_id}/preview.gif")
    async def preview_gif(illustration_id: str, ref_id: str = ""):
        return await serve_preview(illustration_id, ref_id, "gif")

    @router.post("/v1/illustration-tries")
    async def try_illustration(request: Request):
        body = await _decode(request, TryBody)
        if body is None:
            return bad_json
        try:
            out = await library.try_render(
                body.name, body.code, _kind(body.kind), body.detail_level, body.ref_id)
        except Exception as exc:
            return _failure(exc)
        return {**_preview_json(out.png, out.gif), "warnings": [w.to_json() for w in out.warnings]}

    @router.post("/v1/admin/illustrations")
    async def create_illustration(request: Request):
        body = await _decode(request, IllustrationBody)
        if body is None:
            return bad_json
        try:
            out = await library.create(_illustration_from(body), body.ref_id)
        except Exception as exc:
            return _failure(exc)
        return JSONResponse(out.to_json(), status_code=201)

    @router.put("/v1/admin/illustrations/{illustration_id}")
    async def update_illustration(illustration_id: str, request: Request):
        body = await _decode(request, IllustrationBody)
        if body is None:
            return bad_json
        try:
            out = await library.update(illustration_id, _illustration_from(body), body.ref_id)
        except Exception as exc:
            return _failure(exc)
        return out.to_json()

    @router.post("/v1/admin/illustrations/{illustration_id}/status")
    async def set_status(illustration_id: str, request: Request):
        body = await _decode(request, StatusBody)
        if body is None:
            return bad_json
        try:
            return (await library.set_status(illustration_id, body.status)).to_json()
        except Exception as exc:
            return _failure(exc)

    @router.post("/v1/admin/illustrations/{illustration_id}/rerender")
    async def rerender(illustration_id: str, ref_id: str = ""):
        try:
            png, gif = await library.rerender(illustration_id, ref_id)
        except Exception as exc:
            return _failure(exc)
        return _preview_json(png, gif)

    @router.post("/v1/admin/illustrations/draw")
    async def draw(request: Request):
        body = await _decode(request, DrawBody)
        if body is None:
            return bad_json
        if drawer is None:
            return illustration_error(DrawerDisabledError())
        try:
            out = await drawer.draw(DrawRequest(
                description=body.description, folder_id=body.folder_id, ref_id=body.ref_id, name=body.name,
                model=body.model, kind=_kind(body.kind), tier=tier_for_detail(body.detail_level)))
        except Exception as exc:
            return _failure(exc)
        return JSONResponse(out.to_json(), status_code=201)

    @router.post("/v1/admin/illustrations/{illustration_id}/redraw")
    async def redraw(illustration_id: str, request: Request):
        body = await _decode(request, RedrawBody)
        if body is None:
            return bad_json
        if drawer is None:
            return illustration_error(DrawerDisabledError())
        try:
            return (await drawer.redraw(illustration_id, body.note, body.model, body.ref_id)).to_json()
        except Exception as exc:
            return _failure(exc)

    @router.get("/v1/admin/illustrations/export")
    async def export_library():
        """Tải cả thư viện về một file ZIP."""
        now = clock()
        try:
            out = await backup.export(now)
        except Exception as exc:
            return _failure(exc)
        name = f"conceptflow-thu-vien-hinh-{now.strftime('%Y%m%d-%H%M')}.zip"
        return Response(out.zip, media_type="application/zip", headers={
            "Content-Disposition": f'attachment; filename="{name}"', "Cache-Control": "no-store"})

    @router.post("/v1/admin/illustrations/import")
    async def import_library(request: Request, on_conflict: str = "", ref_id: str = ""):
        """Khôi phục bản sao lưu gửi dưới dạng thân ZIP thô. `?on_conflict=replace` ghi đè hình của Creator
        trùng tên; mặc định giữ hình đang có."""
        if on_conflict not in ("", "skip", "replace"):
            return _error(400, "on_conflict phải là skip hoặc replace")
        chunks: list[bytes] = []
        size = 0
        async for chunk in request.stream():
            size += len(chunk)
            if size > MAX_BACKUP_BYTES:
                return _error(413, f"file sao lưu lớn quá (tối đa {MAX_BACKUP_BYTES >> 20} MB)")
            chunks.append(chunk)
        data = b"".join(chunks)
        if not data:
            return _error(400, "chưa gửi file sao lưu")
        try:
            report = await backup.import_library(data, replace=on_conflict == "replace", ref_id=ref_id)
        except Exception as exc:
            return _failure(exc)
        return report.to_json()

    @router.delete("/v1/admin/illustrations/{illustration_id}")
    async def delete_illustration(illustration_id: str):
        try:
            await library.delete(illustration_id)
        except Exception as exc:
            return _failure(exc)
        return Response(status_code=204)

    @router.post("/v1/admin/illustrations/{illustration_id}/exemplar")
    async def make_exemplar(illustration_id: str):
        """Sao một hình đã duyệt vào Hình mẫu."""
        try:
            out = await library.make_exemplar(illustration_id)
        except Exception as exc:
            return _failure(exc)
        return JSONResponse(out.to_json(), status_code=201)

    @router.delete("/v1/admin/illustrations/{illustration_id}/exemplar")
    async def unmake_exemplar(illustration_id: str):
        """Đưa hình ra khỏi Hình mẫu: hình gốc về lại (200, hình như hiện nay), bản sao bị xoá (204)."""
        try:
            out = await library.unmake_exemplar(illustration_id)
        except Exception as exc:
            return _failure(exc)
        if out is None:
            return Response(status_code=204)
        return out.to_json()

    return router


def _failure(exc: Exception) -> JSONResponse:
    """Lỗi nghiệp vụ thành mã HTTP; lỗi không lường trước được ghi log và trả 500, không nuốt."""
    if isinstance(exc, IllustrationError | InvalidKitError):
        return illustration_error(exc)
    logger.exception("illustration route failed")
    return _error(500, f"lỗi không lường trước: {exc}")
