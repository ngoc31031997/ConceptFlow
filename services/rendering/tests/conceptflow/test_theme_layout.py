"""Theme và hình học khung an toàn — phần thuần Python của design system.

Không import manim, đúng như `theme.py`/`layout.py`. Đây là chỗ đáng test nhất
của CR-017: bảng màu và luật "thế nào là tràn khung" là thứ CR-021 sẽ dùng lại
để chấm QC, nên chúng phải đúng trước khi có bất kỳ khung hình nào được dựng.
"""

import pytest

from conceptflow import theme as theme_module
from conceptflow.layout import (
    Box,
    contrast_ratio,
    fit_scale,
    overflow,
    overlaps,
    safe_area,
)


def test_theme_mac_dinh_dung_nen_brand():
    t = theme_module.get(None)
    assert t.background == theme_module.BRAND_BG


def test_ten_theme_la_thi_roi_ve_mac_dinh():
    """Không raise: render bằng theme mặc định vẫn ra video xem được, còn hỏng
    cả lượt render vì gõ sai tên theme thì không."""
    assert theme_module.get("khong-ton-tai") is theme_module.DEFAULT


def test_derive_giu_moi_thu_tru_cai_duoc_ghi_de():
    base = theme_module.DEFAULT
    child = base.derive("kenh-en", accent=theme_module.MOCHA_TEAL)
    assert child.accent == theme_module.MOCHA_TEAL
    assert child.background == base.background
    assert child.fonts == base.fonts


def test_series_color_quay_vong():
    t = theme_module.DEFAULT
    assert t.series_color(0) == t.series_color(len(t.series))


@pytest.mark.parametrize("side,box", [
    ("trái", Box(-8, -5, -1, 1)),
    ("phải", Box(5, 8, -1, 1)),
    ("trên", Box(-1, 1, 2, 5)),
    ("dưới", Box(-1, 1, -5, -2)),
])
def test_overflow_neu_ten_canh_tran(side, box):
    assert side in overflow(box)


def test_khong_tran_khi_nam_gon():
    assert overflow(Box(-1, 1, -1, 1)) == ()


def test_fit_scale_khong_bao_gio_phong_to():
    """Phóng to sẽ khiến cùng một đoạn chữ hiện ở cỡ khác nhau tuỳ độ dài, và
    thang cỡ chữ mất hết ý nghĩa."""
    assert fit_scale(Box(-0.5, 0.5, -0.5, 0.5)) == 1.0


def test_fit_scale_co_vua_that_khong_chi_vua_khit():
    """Hồi quy: co về đúng mép rồi sai số dấu phẩy động đẩy ra ngoài, khiến
    overflow() báo tràn đúng thứ fit_scale() vừa bảo là vừa."""
    area = safe_area()
    tall = Box(-1, 1, -area.height, area.height)
    factor = fit_scale(tall)
    scaled = Box(
        tall.left * factor, tall.right * factor,
        tall.bottom * factor, tall.top * factor,
    )
    assert overflow(scaled) == ()


def test_overlaps():
    assert overlaps(Box(0, 2, 0, 1), Box(1, 3, 0, 1))
    assert not overlaps(Box(0, 1, 0, 1), Box(2, 3, 0, 1))


def test_bang_mau_du_tuong_phan_tren_nen():
    """Chữ trên video còn bị nén và bị xem trên màn hình kém, nên nhắm cao hơn
    mức AA (4.5) của WCAG."""
    t = theme_module.DEFAULT
    assert contrast_ratio(t.ink, t.background) > 7.0
    assert contrast_ratio(t.muted, t.background) > 4.5
    for index in range(len(t.series)):
        assert contrast_ratio(t.series_color(index), t.background) > 4.5
