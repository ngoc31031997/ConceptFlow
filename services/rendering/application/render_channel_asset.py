"""RenderChannelAssetUseCase — dựng intro/outro cố định của kênh (CR-023 FR65).

Đồng bộ, không qua queue riêng, giống cách `validate_script` (CR-020) đã tách
khỏi lượt render thật: đây không phải việc chạy trong pipeline của một project,
mà là flow admin gọi khi asset mới cần dựng (D3 —
`cr-023-low-level-design.md`). Không có lượt dry, không có `narration_segments`
để đối chiếu — hai scene ở `conceptflow.channel_idents` không gọi
`self.narrate(...)`, nên máy hai lượt của CR-018 không áp dụng ở đây.
"""

from __future__ import annotations

from dataclasses import dataclass

from adapters.storage.artifact_paths import compute_channel_asset_path, ensure_parent_dir
from domain.models import ChannelAssetRenderRequest, ChannelAssetRenderResult
from domain.ports import ChannelAssetRendererPort

#: CR-023 chỉ có hai loại asset kênh — mở rộng thêm loại thứ ba (nếu có) là
#: việc của một CR sau, không phải mở khoá kind tuỳ ý ngay từ đầu.
VALID_KINDS = frozenset({"intro", "outro"})


class ChannelAssetRenderError(Exception):
    """Yêu cầu dựng asset không hợp lệ, hoặc Manim thất bại lúc dựng."""


@dataclass(frozen=True)
class ChannelAssetRenderResponse:
    """Hình dạng trả về, soi gương `ValidationResult` của `validate_script.py`:
    một dataclass đóng gói kết quả, để chỗ gọi (D3's admin flow / handler
    messaging) không phải biết chi tiết bên trong `ManimScriptRenderer`."""

    video_path: str
    video_duration_seconds: float
    render_quality: str


class RenderChannelAssetUseCase:
    def __init__(self, renderer: ChannelAssetRendererPort) -> None:
        self._renderer = renderer

    def render(self, kind: str, render_quality: str | None = None) -> ChannelAssetRenderResponse:
        if kind not in VALID_KINDS:
            raise ChannelAssetRenderError(
                f"kind không hợp lệ: {kind!r}; phải là một trong {sorted(VALID_KINDS)}"
            )

        # Resolved before naming the output path: FR65.5 caches one asset per
        # quality, and the fallback for an absent/unknown quality lives in the
        # renderer, not here.
        resolved_quality = self._renderer.resolve_render_quality(render_quality)
        request = ChannelAssetRenderRequest(kind=kind, render_quality=resolved_quality)
        output_path = compute_channel_asset_path(kind, resolved_quality)
        ensure_parent_dir(output_path)
        try:
            result: ChannelAssetRenderResult = self._renderer.render_channel_asset(
                request, output_path
            )
        except ValueError as exc:
            raise ChannelAssetRenderError(str(exc)) from exc

        return ChannelAssetRenderResponse(
            video_path=result.video_path,
            video_duration_seconds=result.video_duration_seconds,
            render_quality=resolved_quality,
        )
