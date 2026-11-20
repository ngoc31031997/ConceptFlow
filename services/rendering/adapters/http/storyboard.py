"""Route nội bộ của Storyboard: `/internal/v1/projects/{project_id}/storyboard…` (visual-service gọi).

Thân yêu cầu và trả về là JSON; lỗi là `{"error": "<thông báo>"}`.
Hợp đồng ở `docs/contracts/storyboard-v1.md`.
"""

from __future__ import annotations

import json
import logging
from typing import Any

from fastapi import APIRouter, Request, Response
from fastapi.responses import JSONResponse
from pydantic import BaseModel, ValidationError

from application.storyboard import Storyboard
from domain.illustration import IllustrationError
from domain.project_asset import validate_project_id
from domain.storyboard import (
    AssetsNotReadyError,
    FrameNotFoundError,
    FrameRenderError,
    InvalidDocumentError,
    InvalidFrameRequestError,
    MissingFramesError,
    VersionConflictError,
    VersionNotFoundError,
    build_frame_request,
)

logger = logging.getLogger(__name__)


class _Body(BaseModel):
    """Thân tolerant: trường lạ bị bỏ, trường thiếu lấy mặc định, kiểu sai là "invalid JSON body"."""


class FrameBody(_Body):
    shot_id: str = ""
    unit_id: str = ""
    beat_id: str = ""
    spec_hash: str = ""
    kit: dict | None = None
    size: dict | None = None
    background: dict = {}
    placements: list[dict] = []
    camera_view: str = ""
    on_screen_text: list[dict] = []
    text_color: str = "#FFFFFF"


class VersionBody(_Body):
    document: Any = None
    sequence_version: int = 0
    note: str = ""


def _error(status: int, message: str, **extra: Any) -> JSONResponse:
    return JSONResponse({"error": message, **extra}, status_code=status)


def _failure(exc: Exception) -> JSONResponse:
    """Lỗi nghiệp vụ thành mã HTTP; lỗi không lường trước được ghi log và trả 500, không nuốt."""
    if isinstance(exc, AssetsNotReadyError):
        return _error(409, str(exc), assets=exc.problems)
    if isinstance(exc, MissingFramesError):
        return _error(409, str(exc), missing_frames=exc.shot_ids)
    if isinstance(exc, VersionConflictError):
        return _error(409, str(exc))
    if isinstance(exc, FrameRenderError):
        return _error(503, str(exc))
    if isinstance(exc, FrameNotFoundError | VersionNotFoundError):
        return _error(404, str(exc))
    if isinstance(exc, InvalidFrameRequestError | InvalidDocumentError | IllustrationError):
        return _error(400, str(exc))
    logger.exception("storyboard route failed")
    return _error(500, f"lỗi không lường trước: {exc}")


async def _decode(request: Request, model: type[_Body]) -> Any:
    try:
        return model.model_validate(json.loads(await request.body() or b"{}"))
    except (ValueError, ValidationError):
        return None


def create_storyboard_router(storyboard: Storyboard) -> APIRouter:
    router = APIRouter()
    bad_json = _error(400, "invalid JSON body")
    base = "/internal/v1/projects/{project_id}/storyboard"

    @router.post(base + "/frames")
    async def draw_frame(project_id: str, request: Request):
        body = await _decode(request, FrameBody)
        if body is None:
            return bad_json
        try:
            validate_project_id(project_id)
            frame = build_frame_request(
                project_id, body.shot_id, body.unit_id, body.beat_id, body.spec_hash, body.kit,
                body.placements, body.camera_view, body.background, body.on_screen_text, body.text_color,
                body.size)
            return (await storyboard.draw_frame(frame)).to_json()
        except Exception as exc:
            return _failure(exc)

    @router.get(base + "/frames/{shot_id}")
    async def read_frame(project_id: str, shot_id: str):
        try:
            validate_project_id(project_id)
            return Response(await storyboard.read_frame(project_id, shot_id), media_type="image/png")
        except Exception as exc:
            return _failure(exc)

    @router.post(base + "/versions")
    async def lock_version(project_id: str, request: Request):
        body = await _decode(request, VersionBody)
        if body is None:
            return bad_json
        try:
            validate_project_id(project_id)
            version = await storyboard.lock(project_id, body.document, body.sequence_version, body.note)
            return JSONResponse(version.to_json(), status_code=201)
        except Exception as exc:
            return _failure(exc)

    @router.get(base + "/versions/latest")
    async def latest_version(project_id: str):
        try:
            validate_project_id(project_id)
            return (await storyboard.latest_version(project_id)).to_json()
        except Exception as exc:
            return _failure(exc)

    @router.get(base + "/versions/{number}")
    async def get_version(project_id: str, number: int):
        try:
            validate_project_id(project_id)
            return (await storyboard.get_version(project_id, number)).to_json()
        except Exception as exc:
            return _failure(exc)

    @router.delete(base)
    async def delete_storyboard(project_id: str):
        try:
            validate_project_id(project_id)
            await storyboard.delete_project(project_id)
        except Exception as exc:
            return _failure(exc)
        return Response(status_code=204)

    return router
