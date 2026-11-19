"""Client của visual-service: bộ thông số vẽ và mô tả của một phong cách tham chiếu."""

from __future__ import annotations

import httpx

from domain.illustration import (
    ReferenceKitMissingError,
    ReferenceServiceError,
    ReferenceStyleNotFoundError,
)


class VisualServiceClient:
    def __init__(
        self, base_url: str, timeout_seconds: float = 15, client: httpx.AsyncClient | None = None,
    ) -> None:
        self._client = client or httpx.AsyncClient(base_url=base_url.rstrip("/"), timeout=timeout_seconds)

    async def get_kit(self, ref_id: str) -> dict:
        """`GET /internal/v1/reference-styles/{id}/kit` → `{ref_id, kit}`."""
        body = await self._get(f"/internal/v1/reference-styles/{ref_id}/kit")
        if body.status_code == 404:
            raise ReferenceStyleNotFoundError(ref_id)
        if body.status_code == 409:
            raise ReferenceKitMissingError(ref_id)
        kit = self._json(body).get("kit")
        if not isinstance(kit, dict):
            raise ReferenceServiceError("visual-service trả bộ thông số vẽ không đọc được")
        return kit

    async def get_style(self, ref_id: str) -> dict:
        """Mô tả của phong cách, lấy từ danh sách `GET /v1/reference-styles`."""
        body = await self._get("/v1/reference-styles")
        for style in self._json(body).get("styles") or []:
            if isinstance(style, dict) and style.get("id") == ref_id:
                return style
        raise ReferenceStyleNotFoundError(ref_id)

    async def _get(self, path: str) -> httpx.Response:
        try:
            res = await self._client.get(path)
        except httpx.HTTPError as exc:
            raise ReferenceServiceError(f"không gọi được visual-service: {exc}") from exc
        if res.status_code >= 500:
            raise ReferenceServiceError(f"visual-service trả lỗi {res.status_code}")
        return res

    @staticmethod
    def _json(res: httpx.Response) -> dict:
        if res.status_code != 200:
            raise ReferenceServiceError(f"visual-service trả {res.status_code}")
        try:
            body = res.json()
        except ValueError as exc:
            raise ReferenceServiceError("visual-service trả nội dung không đọc được") from exc
        if not isinstance(body, dict):
            raise ReferenceServiceError("visual-service trả nội dung không đọc được")
        return body

    async def close(self) -> None:
        await self._client.aclose()
