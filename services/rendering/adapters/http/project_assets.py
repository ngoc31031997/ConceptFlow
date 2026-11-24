"""Route nội bộ của tài sản dự án: `/internal/v1/projects/{project_id}/assets…` (visual-service gọi).

Thân yêu cầu và trả về là JSON; lỗi là `{"error": "<thông báo>"}` với mã như route thư viện hình,
riêng code vẽ không qua kiểm là `422 {"error", "detail": {kind, message, line, snippet}}`.
Hợp đồng ở `docs/contracts/storyboard-v1.md`.
"""

from __future__ import annotations

import base64
import json
import logging
from typing import Any

from fastapi import APIRouter, Request, Response
from fastapi.responses import JSONResponse
from pydantic import BaseModel, ValidationError

from adapters.http.illustrations import illustration_error
from application.project_assets import ProjectAssets
from domain.illustration import DrawCheckError, DrawErrorKind, IllustrationError
from domain.kit import InvalidKitError
from domain.project_asset import AssetNoDetailError, AssetNotFoundError, AssetStateError, ProjectAsset

logger = logging.getLogger(__name__)


class _Body(BaseModel):
    """Thân tolerant: trường lạ bị bỏ, trường thiếu lấy mặc định, kiểu sai là "invalid JSON body"."""


class AssetBody(_Body):
    concept_id: str = ""
    name: str = ""
    kind: str = "figure"
    tier: str = "support"
    portrayal: str = ""
    must_not_imply: list[str] | None = None
    palette_tokens: list[str] | None = None
    expressions: list[str] | None = None
    base_illustration_id: str = ""
    units: list[str] | None = None
    to_library: bool = True
    library_folder_id: str = ""


class DrawBody(_Body):
    #: Kit của dự án; visual-service gửi kèm mỗi lần.
    kit: dict | None = None
    #: Phong cách tham chiếu của dự án: luật vẽ của nó vào prompt của bộ vẽ.
    ref_id: str = ""


class DrawErrorBody(_Body):
    kind: str = ""
    message: str = ""
    line: int | None = None
    snippet: str | None = None


class FixErrorBody(_Body):
    #: Code hỏng vừa nhận; rỗng khi câu trả lời không có code.
    code: str = ""
    error: DrawErrorBody | None = None


class DrawPromptBody(DrawBody):
    note: str = ""
    redraw: bool = False
    fix_error: FixErrorBody | None = None


class DrawCheckBody(DrawBody):
    #: Nguyên văn câu trả lời của AI.
    reply: str = ""


class DrawApplyBody(DrawBody):
    code: str = ""


class ApproveBody(_Body):
    project_name: str = ""
    ref_id: str = ""


def _error(status: int, message: str) -> JSONResponse:
    return JSONResponse({"error": message}, status_code=status)


def _failure(exc: Exception) -> JSONResponse:
    """Lỗi nghiệp vụ thành mã HTTP; lỗi không lường trước được ghi log và trả 500, không nuốt."""
    if isinstance(exc, DrawCheckError):
        return JSONResponse({"error": exc.message, "detail": exc.to_json()}, status_code=422)
    if isinstance(exc, AssetNotFoundError):
        return _error(404, str(exc))
    if isinstance(exc, AssetNoDetailError):
        return JSONResponse({"error": str(exc), "code": exc.code}, status_code=409)
    if isinstance(exc, AssetStateError):
        return _error(409, str(exc))
    if isinstance(exc, IllustrationError | InvalidKitError):
        return illustration_error(exc)
    logger.exception("project asset route failed")
    return _error(500, f"lỗi không lường trước: {exc}")


async def _decode(request: Request, model: type[_Body]) -> Any:
    try:
        return model.model_validate(json.loads(await request.body() or b"{}"))
    except (ValueError, ValidationError):
        return None


def _asset_from(project_id: str, body: AssetBody) -> ProjectAsset:
    return ProjectAsset(
        project_id=project_id, concept_id=body.concept_id, name=body.name, kind=body.kind, tier=body.tier,
        portrayal=body.portrayal, must_not_imply=tuple(body.must_not_imply or ()),
        palette_tokens=tuple(body.palette_tokens or ()), expressions=tuple(body.expressions or ()),
        base_illustration_id=body.base_illustration_id, units=tuple(body.units or ()),
        to_library=body.to_library, library_folder_id=body.library_folder_id)


def _fix_error(body: FixErrorBody | None) -> tuple[str, DrawCheckError | None]:
    """Code hỏng và lỗi có cấu trúc của "gửi lại kèm lỗi"; lỗi thiếu loại hợp lệ hay thông báo là 400."""
    if body is None:
        return "", None
    err = body.error
    kinds = {k.value for k in DrawErrorKind}
    if err is None or err.kind not in kinds or not err.message.strip():
        raise IllustrationError(
            "fix_error cần error có kind (" + ", ".join(sorted(kinds)) + ") và message")
    return body.code, DrawCheckError(DrawErrorKind(err.kind), err.message, err.line, err.snippet)


def create_project_asset_router(assets: ProjectAssets) -> APIRouter:
    router = APIRouter()
    bad_json = _error(400, "invalid JSON body")
    base = "/internal/v1/projects/{project_id}/assets"

    @router.get(base)
    async def list_assets(project_id: str):
        try:
            return {"assets": [a.to_json() for a in await assets.list_assets(project_id)]}
        except Exception as exc:
            return _failure(exc)

    @router.post(base)
    async def create_asset(project_id: str, request: Request):
        body = await _decode(request, AssetBody)
        if body is None:
            return bad_json
        try:
            return JSONResponse(
                (await assets.create(_asset_from(project_id, body))).to_json(), status_code=201)
        except Exception as exc:
            return _failure(exc)

    @router.get(base + "/{asset_id}")
    async def get_asset(project_id: str, asset_id: str):
        try:
            return (await assets.get(project_id, asset_id)).to_json()
        except Exception as exc:
            return _failure(exc)

    @router.get(base + "/{asset_id}/preview")
    async def preview_asset(project_id: str, asset_id: str):
        try:
            png = await assets.preview(project_id, asset_id)
        except Exception as exc:
            return _failure(exc)
        return Response(png, media_type="image/png", headers={"Cache-Control": "private, no-cache"})

    @router.put(base + "/{asset_id}")
    async def update_asset(project_id: str, asset_id: str, request: Request):
        body = await _decode(request, AssetBody)
        if body is None:
            return bad_json
        try:
            return (await assets.update(project_id, asset_id, _asset_from(project_id, body))).to_json()
        except Exception as exc:
            return _failure(exc)

    @router.delete(base + "/{asset_id}")
    async def delete_asset(project_id: str, asset_id: str):
        try:
            await assets.delete(project_id, asset_id)
        except Exception as exc:
            return _failure(exc)
        return Response(status_code=204)

    @router.post(base + "/{asset_id}/draw-prompt")
    async def draw_prompt(project_id: str, asset_id: str, request: Request):
        body = await _decode(request, DrawPromptBody)
        if body is None:
            return bad_json
        try:
            fix_code, fix_error = _fix_error(body.fix_error)
            system, user = await assets.draw_prompt(
                project_id, asset_id, body.kit, body.ref_id, note=body.note, redraw=body.redraw,
                fix_code=fix_code, fix_error=fix_error)
        except Exception as exc:
            return _failure(exc)
        return {"system": system, "user": user}

    @router.post(base + "/{asset_id}/draw-check")
    async def draw_check(project_id: str, asset_id: str, request: Request):
        body = await _decode(request, DrawCheckBody)
        if body is None:
            return bad_json
        try:
            checked = await assets.draw_check(project_id, asset_id, body.kit, body.ref_id, body.reply)
        except Exception as exc:
            return _failure(exc)
        return {
            "code": checked.code,
            "png_base64": base64.b64encode(checked.rendered.png).decode("ascii"),
            "warnings": [w.to_json() for w in checked.rendered.warnings],
        }

    @router.post(base + "/{asset_id}/draw-apply")
    async def draw_apply(project_id: str, asset_id: str, request: Request):
        body = await _decode(request, DrawApplyBody)
        if body is None:
            return bad_json
        try:
            return (await assets.draw_apply(
                project_id, asset_id, body.kit, body.ref_id, body.code)).to_json()
        except Exception as exc:
            return _failure(exc)

    @router.post(base + "/{asset_id}/approve")
    async def approve_asset(project_id: str, asset_id: str, request: Request):
        body = await _decode(request, ApproveBody)
        if body is None:
            return bad_json
        try:
            return (await assets.approve(project_id, asset_id, body.project_name, body.ref_id)).to_json()
        except Exception as exc:
            return _failure(exc)

    @router.post(base + "/{asset_id}/unapprove")
    async def unapprove_asset(project_id: str, asset_id: str):
        try:
            return (await assets.unapprove(project_id, asset_id)).to_json()
        except Exception as exc:
            return _failure(exc)

    return router
