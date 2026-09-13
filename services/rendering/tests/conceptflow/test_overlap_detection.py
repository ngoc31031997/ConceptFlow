"""Phát hiện chồng lấn hình ảnh ở lượt dry (bug report 2026-09-12).

Root cause thật: `ConceptFlowScene.caption(...)` (và `title`/`heading`/`body`)
dựng một `Text(...)` không định vị — Manim đặt nó ở tâm khung hình mặc định.
Một script quên `.to_edge(...)`/`.next_to(...)` khi khung đã có sẵn một
bảng/table sẽ chồng khít lên nhau, không đọc được, và trước bug fix này không
gì bắt được việc đó trước khi tốn TTS.

Bộ test này chứng minh `ConceptFlowScene.play()` bắt được đúng trường hợp đó
(hai mobject cùng ở tâm mặc định) và không báo động giả cho bố cục đã tách
bằng `.to_edge(...)`.
"""

import json

import pytest

pytest.importorskip("manim", reason="cần Scene thật để chạy construct()")

from manim import DOWN, UP, FadeIn, Rectangle, Text  # noqa: E402

from conceptflow import narration  # noqa: E402
from conceptflow.scene import ConceptFlowScene, _overlap_ratio  # noqa: E402


@pytest.fixture
def dry(tmp_path, monkeypatch):
    """Chạy ở chế độ dry và trả về các bản ghi mà script ghi ra."""
    marks = tmp_path / "cf_marks.jsonl"
    monkeypatch.setenv(narration.ENV_MARKS_PATH, str(marks))
    monkeypatch.setenv(narration.ENV_MODE, narration.MODE_DRY)
    narration.reset()

    def run(scene_cls):
        scene_cls().construct()
        if not marks.exists():
            return []
        return [json.loads(line) for line in marks.read_text(encoding="utf-8").splitlines() if line]

    return run


def _overlaps(records):
    return [r for r in records if r["kind"] == "overlap"]


def test_hai_mobject_cung_tam_mac_dinh_bi_bao_chong_lan(dry):
    """Đúng root cause của bug: một `Text` và một `Rectangle` (đứng cho bảng/
    table) cả hai không định vị đều rơi vào tâm khung hình mặc định của
    Manim — chồng gần như hoàn toàn."""

    class S(ConceptFlowScene):
        def construct(self):
            table = Rectangle(width=6, height=3)
            self.play(FadeIn(table))
            caption = Text("Bước 1: Quét ma trận từng ô từ trên xuống dưới.")
            self.play(FadeIn(caption))
            self.narrate("một câu bất kỳ để lượt dry có lời thoại")

    overlaps = _overlaps(dry(S))
    assert len(overlaps) == 1
    assert overlaps[0]["index"] == 0  # trước lời thoại đầu tiên (upcoming_index)
    assert "Rectangle" in overlaps[0]["description"]
    assert "Text" in overlaps[0]["description"]


def test_hai_mobject_dat_canh_tren_duoi_khong_bao(dry):
    """`.to_edge(UP)` / `.to_edge(DOWN)` là đúng cách sửa bug — không còn gì
    để báo."""

    class S(ConceptFlowScene):
        def construct(self):
            table = Rectangle(width=6, height=3).to_edge(UP)
            self.play(FadeIn(table))
            caption = Text("chú thích").to_edge(DOWN)
            self.play(FadeIn(caption))
            self.narrate("một câu bất kỳ để lượt dry có lời thoại")

    assert _overlaps(dry(S)) == []


def test_khong_bao_o_luot_render(tmp_path, monkeypatch):
    """Phép tính hình học chỉ chạy ở lượt dry (CF_MODE=dry) — lượt render thật
    không trả thêm chi phí cho một kiểm tra chỉ có ích trước TTS."""
    marks = tmp_path / "cf_marks.jsonl"
    durations = tmp_path / "cf_durations.json"
    durations.write_text("[1.0]", encoding="utf-8")
    monkeypatch.setenv(narration.ENV_MARKS_PATH, str(marks))
    monkeypatch.setenv(narration.ENV_MODE, narration.MODE_RENDER)
    monkeypatch.setenv(narration.ENV_DURATIONS_PATH, str(durations))
    narration.reset()

    class S(ConceptFlowScene):
        def construct(self):
            self.play(FadeIn(Rectangle(width=6, height=3)))
            self.play(FadeIn(Text("chồng lấn không được báo ở lượt render")))
            self.narrate("một câu bất kỳ")

    S().construct()
    records = [json.loads(line) for line in marks.read_text(encoding="utf-8").splitlines() if line]
    assert _overlaps(records) == []


def test_overlap_ratio_chia_theo_dien_tich_vat_nho_hon():
    """Đơn vị: hộp bao (left, right, top, bottom). Vật nhỏ nằm hoàn toàn
    trong vật lớn phải cho tỉ lệ 1.0 (100% diện tích vật nhỏ), không phải một
    tỉ lệ nhỏ tính theo vật lớn hay theo hợp của cả hai."""
    big = (-3.0, 3.0, 1.5, -1.5)  # rộng 6, cao 3 -> diện tích 18
    small = (-1.0, 1.0, 0.5, -0.5)  # rộng 2, cao 1 -> diện tích 2, nằm trọn trong big
    assert _overlap_ratio(big, small) == pytest.approx(1.0)


def test_overlap_ratio_khong_giao_nhau_bang_khong():
    left = (-3.0, -1.0, 1.0, -1.0)
    right = (1.0, 3.0, 1.0, -1.0)
    assert _overlap_ratio(left, right) == 0.0
