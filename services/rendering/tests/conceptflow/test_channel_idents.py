"""Sting mở/đóng kênh cố định (CR-023 FR65, FR66.5).

`importorskip` giữ đúng tính chất mà test suite của Rendering đang có: chạy
được mà không cần cài manim thật (README) — CI trong image thì luôn có.
"""

from __future__ import annotations

import pytest

pytest.importorskip("manim", reason="scene cần manim thật để dựng ImageMobject/Text")

from conceptflow.channel_idents import (  # noqa: E402
    INTRO_DURATION_SECONDS,
    OUTRO_DURATION_SECONDS,
    ChannelOutro,
    DefaultIntroSting,
)
from conceptflow.layout import Box, overlaps  # noqa: E402


def _measure_duration(scene_cls) -> float:
    """Chạy `construct()` thật (dựng mobject thật) nhưng chặn animation/wait ở
    mức không tốn thời gian render thật — chỉ cộng dồn `run_time`/`duration`
    mà script đã khai, giống cách `dry` fixture của `test_narration.py` chặn
    ở lớp `narration` thay vì chạy Manim thật tới cùng.
    """
    elapsed: list[float] = []

    class _Timed(scene_cls):
        def play(self, *args, run_time: float = 1.0, **kwargs) -> None:
            elapsed.append(run_time)

        def wait(self, duration: float = 1.0, **kwargs) -> None:
            elapsed.append(duration)

    _Timed().construct()
    return sum(elapsed)


def test_intro_dai_khoang_3_giay():
    assert _measure_duration(DefaultIntroSting) == pytest.approx(INTRO_DURATION_SECONDS)
    assert INTRO_DURATION_SECONDS == pytest.approx(3.0)


def test_outro_dai_trong_khoang_15_20_giay():
    duration = _measure_duration(ChannelOutro)
    assert 15.0 <= duration <= 20.0
    assert duration == pytest.approx(OUTRO_DURATION_SECONDS)


def test_ba_vung_an_toan_cua_outro_khong_de_len_nhau():
    """FR65.8: logo, khung 'video đề xuất', và lời mời đăng ký phải tách bạch —
    YouTube tự chèn end-screen element thật đè lên các vùng này."""
    logo = Box(*ChannelOutro.LOGO_ZONE)
    suggested = Box(*ChannelOutro.SUGGESTED_ZONE)
    subscribe = Box(*ChannelOutro.SUBSCRIBE_ZONE)

    assert not overlaps(logo, suggested)
    assert not overlaps(logo, subscribe)
    assert not overlaps(suggested, subscribe)


def test_ba_vung_an_toan_nam_trong_khung_hinh():
    from conceptflow.layout import overflow

    for zone in (ChannelOutro.LOGO_ZONE, ChannelOutro.SUGGESTED_ZONE, ChannelOutro.SUBSCRIBE_ZONE):
        assert overflow(Box(*zone)) == ()
