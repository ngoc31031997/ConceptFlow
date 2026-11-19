"""Bộ dựng của thư viện hình: gọi thẳng use case xem trước trong cùng tiến trình.

Dùng chung hàng đợi một-yêu-cầu-một-lúc với `POST /v1/illustrations/preview`, vì bộ dựng chỉ mở một
tab trình duyệt.
"""

from __future__ import annotations

import asyncio
import base64

from adapters.rendering.illustration_previewer import IllustrationPreviewError
from adapters.rendering.typescript_checker import TypeScriptCheckError
from application.illustration_ports import RenderedIllustration
from application.preview_illustration import PreviewIllustrationUseCase
from domain.illustration import CodeFinding, RendererUnavailableError


def _findings(items: list[dict]) -> list[CodeFinding]:
    return [CodeFinding(message=i["message"], line=i.get("line")) for i in items]


class InProcessIllustrationRenderer:
    def __init__(self, preview: PreviewIllustrationUseCase, gate: asyncio.Semaphore) -> None:
        self._preview = preview
        self._gate = gate

    async def render(self, kit: dict, name: str, code: str, kind: str, tier: str) -> RenderedIllustration:
        async with self._gate:
            try:
                out = await asyncio.to_thread(
                    self._preview.run, kit, name, code, None, True, kind, tier)
            except (IllustrationPreviewError, TypeScriptCheckError) as exc:
                raise RendererUnavailableError(f"dựng xem trước: {exc}") from exc
        if not out.ok:
            return RenderedIllustration(False, _findings(out.diagnostics), _findings(out.warnings))
        return RenderedIllustration(
            True, warnings=_findings(out.warnings), png=base64.b64decode(out.png),
            gif=base64.b64decode(out.gif) if out.gif else b"")
