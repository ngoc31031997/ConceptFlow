"""Bản sắc kênh cố định: intro/outro (CR-023 FR65, FR66.5).

Hai scene ở đây **khác loại** với mọi script Creator viết: chúng không có lời
thoại, không gọi `self.narrate()`/`self.beat()`, và vì vậy không đi qua lượt
dry của CR-018 (script sẽ không sinh ra một dòng lời thoại nào để lượt dry thu
thập). Chúng được dựng một lần, không phải mỗi project (D3 của
`cr-023-low-level-design.md`), qua `RenderChannelAssetUseCase` chứ không qua
saga `render_scenes`.

`DefaultIntroSting` và `ChannelOutro` vẫn kế thừa `ConceptFlowScene` để lấy nền
và font theo `theme.py` — bản sắc kênh chỉ nên có một nguồn sự thật, dù là
intro/outro hay nội dung.
"""

from __future__ import annotations

import os
from pathlib import Path

from manim import DOWN, RIGHT, Group, ImageMobject, Rectangle, Text

from .components import Recap, TitleCard
from .scene import ConceptFlowScene

#: Thư mục chứa bộ nhận diện. `docs/brand/` KHÔNG vào được image: build context
#: của rendering là `./services/rendering` (docker-compose.yml), nên mọi thứ
#: ngoài thư mục đó nằm ngoài tầm với của `COPY` — đúng ràng buộc H1 mà kế
#: hoạch CR-016..024 đã ghi. Cách đi vào là một bind mount read-only, còn biến
#: này để lúc dev chạy thẳng từ checkout vẫn tìm được cùng file.
_REPO_ROOT = Path(__file__).resolve().parents[3]
BRAND_ASSETS_DIR = Path(os.environ.get("BRAND_ASSETS_DIR") or (_REPO_ROOT / "docs" / "brand"))

#: Logo bộ nhận diện (`docs/brand/make-banner.py` sinh ra file này).
LOGO_MARK_PATH = BRAND_ASSETS_DIR / "conceptflow-mark-1024.png"

#: D4: 3s là đủ để logo "đóng dấu" mà không làm mất giây xem đầu tiên.
INTRO_DURATION_SECONDS = 3.0

#: D4: 15–20s, chọn 18s — đủ chỗ cho end-screen (element YouTube giữ tối thiểu
#: 5s) mà không kéo dài quá mức so với biên trên của CR.
OUTRO_DURATION_SECONDS = 18.0


class DefaultIntroSting(ConceptFlowScene):
    """Sting mở kênh: logo fade+scale trên nền brand (FR65.2 mặc định).

    Không dùng `self.reveal()`/`self.dismiss()` của `ConceptFlowScene`: những
    animation đó lấy `run_time` từ `theme.pacing`, vốn được hiệu chỉnh cho
    khuôn hình có chữ đọc được — sting logo cần đúng 3s cố định bất kể theme,
    vì đó là thời lượng mà D5 cộng thẳng vào `lead_in` của video-assembly.
    """

    def construct(self) -> None:
        logo = ImageMobject(str(LOGO_MARK_PATH))
        logo.scale_to_fit_height(3.2)
        logo.scale(0.6)
        logo.set_opacity(0.0)

        fade_in = 1.1
        hold = 0.9
        fade_out = 1.0
        assert abs((fade_in + hold + fade_out) - INTRO_DURATION_SECONDS) < 1e-9

        self.add(logo)
        self.play(logo.animate.set_opacity(1.0).scale(1 / 0.6), run_time=fade_in)
        self.wait(hold)
        self.play(logo.animate.set_opacity(0.0), run_time=fade_out)


class ChannelOutro(ConceptFlowScene):
    """Outro cố định: logo + wordmark, khung "video đề xuất", lời mời đăng ký.

    FR65.8 buộc chừa 3 vùng an toàn không đè lên nhau cho end-screen element
    của YouTube (YouTube tự vẽ đè video/subscribe thật lên đây — Rendering chỉ
    phải KHÔNG đặt chữ/hình của mình vào chỗ YouTube sẽ chèn):

      (a) LOGO_ZONE       — góc trên-trái: logo + wordmark kênh (TitleCard).
      (b) SUGGESTED_ZONE  — nửa dưới-phải: khung trống hình vuông 16:9's
                             "suggested video" card của YouTube neo vào.
      (c) SUBSCRIBE_ZONE  — nửa dưới-trái: text mời đăng ký (Recap-style).

    Toạ độ tính theo hệ Manim (gốc ở tâm khung, y hướng lên, đơn vị của
    `theme.FRAME_WIDTH`/`FRAME_HEIGHT`). Viết tường minh bằng số thay vì suy ra
    từ layout tự động, vì test bbox (`test_channel_idents.py`) cần khẳng định
    ba vùng này không giao nhau — nếu toạ độ ẩn trong `arrange()`, một thay đổi
    nội dung nhỏ (đổi chữ dài hơn) có thể âm thầm làm chúng đè lên nhau.
    """

    #: Bốn cạnh (left, right, bottom, top) của từng vùng an toàn, đơn vị Manim.
    #: Khung an toàn tổng thể (theme.SAFE_MARGIN) rộng khoảng
    #: [-6.5..6.5] x [-3.4..3.4]; ba vùng dưới đây chia khung làm ba dải không
    #: chồng lấn, mỗi dải còn margin nội bộ để chữ/hình không chạm biên.
    LOGO_ZONE = (-6.4, -0.3, 1.2, 3.3)
    SUGGESTED_ZONE = (0.3, 6.4, -0.4, 3.3)
    SUBSCRIBE_ZONE = (-6.4, 6.4, -3.3, -0.6)

    def construct(self) -> None:
        t = self.theme

        logo = ImageMobject(str(LOGO_MARK_PATH)).scale_to_fit_height(1.6)
        wordmark = Text("ConceptFlow", font=t.fonts.display, font_size=t.scale.h2, color=t.ink)
        logo_group = Group(logo, wordmark).arrange(RIGHT, buff=0.3)
        logo_group.move_to(_zone_center(self.LOGO_ZONE))

        suggested_box = Rectangle(
            width=(self.SUGGESTED_ZONE[1] - self.SUGGESTED_ZONE[0]) - 0.6,
            height=(self.SUGGESTED_ZONE[3] - self.SUGGESTED_ZONE[2]) - 0.6,
            color=t.muted,
            stroke_width=2,
        )
        suggested_box.move_to(_zone_center(self.SUGGESTED_ZONE))

        subscribe = TitleCard("Đăng ký kênh để xem tiếp", theme=t)
        subscribe.move_to(_zone_center(self.SUBSCRIBE_ZONE))

        recap_hint = Recap(["Cảm ơn đã xem hết video"], title="", theme=t)
        recap_hint.next_to(subscribe, DOWN, buff=0.25)

        self.add(logo_group, suggested_box, subscribe)

        remaining = OUTRO_DURATION_SECONDS - 1.0
        self.play(logo_group.animate.set_opacity(1.0), run_time=0.6)
        self.play(suggested_box.animate.set_stroke(opacity=1.0), run_time=0.4)
        self.wait(remaining)


def _zone_center(zone: tuple[float, float, float, float]) -> tuple[float, float, float]:
    left, right, bottom, top = zone
    return ((left + right) / 2, (bottom + top) / 2, 0)
