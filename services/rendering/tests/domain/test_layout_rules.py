"""Luật bố cục trên số đo thật của layout probe.

`remotion_project/layout_probe_samples/*.layout.json` là output thật của probe
(`node layout_probe.mjs --out-dir layout_probe_samples ...`) cho các script mẫu
do Code Merger thật của llm-service sinh ra; `*.lines.json` là `Merged.lines`
của chúng — khoảng dòng mà `_map_failures` dùng để gán lỗi về shot.
"""

from __future__ import annotations

import json
from pathlib import Path

import pytest

from domain.layout_rules import (
    HERO_SIZE,
    MIN_FONT,
    NOT_SETTLED,
    RUNTIME_ERROR,
    SAFE_AREA,
    SUBTITLE_ZONE,
    TEXT_OVERFLOW,
    TEXT_OVERLAP,
    SubtitleBand,
    evaluate,
)

SAMPLES = Path(__file__).parents[2] / "remotion_project" / "layout_probe_samples"


def load(name: str) -> tuple[dict, dict[str, list[int]]]:
    probe = json.loads((SAMPLES / f"{name}.layout.json").read_text(encoding="utf-8"))
    lines = json.loads((SAMPLES / f"{name}.lines.json").read_text(encoding="utf-8"))
    return probe, lines


def owner(lines: dict[str, list[int]], line: int) -> str | None:
    """Merged.shot_at của llm-service."""
    return next((sid for sid, (a, b) in lines.items() if a <= line <= b), None)


# --- trên các script mẫu -------------------------------------------------------------


def test_clean_script_has_no_finding():
    probe, _ = load("clean")
    assert evaluate(probe) == []


def test_the_three_planted_faults_block_on_the_right_shots():
    probe, lines = load("problems")
    found = evaluate(probe)
    assert all(f.blocking for f in found)
    got = sorted((owner(lines, f.line), f.rule, f.line) for f in found)
    assert got == [
        ("1.2", TEXT_OVERFLOW, 62),   # nhãn 'Lớp men răng bảo vệ' rộng 502px trong width 360
        ("1.3", SAFE_AREA, 77),       # Tooth mép phải 1891
        ("1.3", SAFE_AREA, 78),       # Germ mép phải 1830 (cũng thật sự vượt)
        ("2.1", TEXT_OVERLAP, 96),    # hai nhãn đè nhau
    ]
    by_rule = {f.rule: f.message for f in found}
    assert by_rule[TEXT_OVERFLOW].startswith("Shot 1.2, frame ")
    assert "nhãn 'Lớp men răng bảo vệ' tràn khung chữ (rộng 502px > width 360px)" in by_rule[TEXT_OVERFLOW]
    assert "giao nhau 456×56px" in by_rule[TEXT_OVERLAP] and "(dòng 99)" in by_rule[TEXT_OVERLAP]
    tooth = next(f for f in found if f.line == 77)
    assert tooth.message == "Shot 1.3, mọi frame đo: hình Tooth ra ngoài vùng an toàn (phải x=1891 > 1824)"


def test_spring_overshoot_between_the_old_four_samples_is_caught():
    # 0/50/85/100% bỏ sót; mặt trời spring({damping: 6}) vọt ra ở frame 10.
    probe, lines = load("overshoot")
    [f] = evaluate(probe)
    assert f.blocking and f.rule == SAFE_AREA and owner(lines, f.line) == "1.1"
    assert f.frames == ["7%"]
    assert "hình Sun ra ngoài vùng an toàn (trên y=94 < 96, phải x=1846 > 1824)" in f.message


def test_subtitle_band_uses_the_strip_from_the_frame_edge():
    probe, lines = load("clean")
    found = evaluate(probe, SubtitleBand("bottom", 240))
    assert found and all(f.rule == SUBTITLE_ZONE and f.blocking for f in found)
    panel = next(f for f in found if owner(lines, f.line) == "2.3")
    assert "lấn vùng phụ đề ở mép dưới khung (mép dưới y=960 > 840)" in panel.message
    # Phụ đề ở mép trên: giờ tới lượt tiêu đề của shot 1.1 (y=128) bị lấn.
    top = evaluate(probe, SubtitleBand("top", 200))
    assert any(owner(lines, f.line) == "1.1" and "mép trên y=128 < 200" in f.message for f in top)


# --- từng luật, trên số đo dựng tay ----------------------------------------------------


def rect(x, y, w, h):
    return {"x": x, "y": y, "w": w, "h": h}


def text(line, label, box, **over):
    e = {"kind": "text", "tag": "div", "line": line, "text": label, "rect": box, "text_rect": box,
         "opacity": 1, "lines": 1, "font_size": 48, "font_size_rendered": 48, "scroll_width": round(box["w"]),
         "client_width": round(box["w"]), "scroll_height": round(box["h"]), "client_height": round(box["h"]),
         "text_parent": None}
    e.update(over)
    return e


def kit(line, name, box, **over):
    return {"kind": "kit", "component": name, "line": line, "rect": box, "opacity": 1, **over}


BIG = kit(12, "Tooth", rect(700, 300, 420, 420))  # đủ lớn để không có cảnh báo L14

PCTS = [0, 5 / 149, 10 / 149, 0.5, 127 / 149, 1]


def probe_of(*shots_elements, pcts=PCTS, lines=(10,), shots_line=None):
    """shots_elements[i] = hàm pct -> danh sách phần tử của shot i ở mẫu đó."""
    shots = []
    for i, fn in enumerate(shots_elements):
        samples = []
        for p in pcts:
            got = fn(p)
            sample = {"pct": p, "frame": round(p * 149), "elements": got if isinstance(got, list) else []}
            if isinstance(got, dict):
                sample.update(got)
            samples.append(sample)
        shots.append({"index": i, "id": f"1.{i + 1}", "component": f"Shot1_{i + 1}", "line": lines[i],
                      "samples": samples})
    return {"version": 1, "composition": {"width": 1920, "height": 1080, "fps": 30},
            "shots_line": shots_line, "shots": shots}


def rules(found):
    return sorted((f.rule, f.blocking) for f in found)


def test_an_object_that_slides_in_from_outside_the_frame_is_judged_only_once_settled():
    def slide(p):  # từ ngoài mép trái vào tới x=400, đi qua dải lề 0..96 ở frame 5
        x = -300 if p == 0 else 40 if p < 0.05 else 400
        return [BIG, kit(14, "Germ", rect(x, 400, 200, 200))]

    assert evaluate(probe_of(slide)) == []

    def slide_but_stays_out(p):
        return [BIG, kit(14, "Germ", rect(-300 if p < 0.5 else 40, 400, 200, 200))]

    [f] = evaluate(probe_of(slide_but_stays_out))
    assert f.rule == SAFE_AREA and f.frames == ["85%", "100%"]


def test_an_object_that_never_leaves_the_frame_is_judged_at_every_sample():
    [f] = evaluate(
        probe_of(lambda p: [BIG, kit(14, "Sun", rect(1700, 200, 150 if p == 5 / 149 else 100, 100))])
    )
    assert f.rule == SAFE_AREA and f.frames == ["3%"] and "phải x=1850 > 1824" in f.message


def test_full_frame_backdrops_and_objects_entirely_off_frame_are_ignored():
    backdrop = {"kind": "shape", "tag": "div", "line": 11, "rect": rect(0, 0, 1920, 1080), "opacity": 1,
                "full_frame": True}
    hidden = kit(15, "Cloud", rect(2000, 100, 200, 100))
    assert evaluate(probe_of(lambda p: [backdrop, BIG, hidden])) == []


def test_top_subtitle_band():
    found = evaluate(
        probe_of(lambda p: [BIG, text(13, "Tiêu đề", rect(400, 150, 800, 70))]), SubtitleBand("top", 200)
    )
    [f] = found
    assert f.rule == SUBTITLE_ZONE and "mép trên y=150 < 200" in f.message


def test_text_overflow_horizontal_and_vertical():
    wide = text(13, "Một nhãn quá dài", rect(400, 400, 360, 60), scroll_width=412, client_width=360)
    tall = text(14, "Ba dòng", rect(400, 600, 360, 60), scroll_height=200, client_height=60)
    diacritics = text(15, "Tiêu đề", rect(400, 800, 360, 72), scroll_height=74, client_height=72)
    found = evaluate(probe_of(lambda p: [BIG, wide, tall, diacritics]))
    assert rules(found) == [(TEXT_OVERFLOW, True), (TEXT_OVERFLOW, True)]
    msgs = [f.message for f in sorted(found, key=lambda f: f.line)]
    assert "nhãn 'Một nhãn quá dài' tràn khung chữ (rộng 412px > width 360px)" in msgs[0]
    assert "tràn khung chữ theo chiều cao (cao 200px > height 60px)" in msgs[1]


def test_tiny_font_counts_only_once_the_shot_has_settled():
    def grow(p):  # bong bóng phóng lên: chữ 20px lúc đầu, 40px khi yên
        return [BIG, text(13, "Xin chào", rect(300, 300, 300, 60), font_size_rendered=20 if p < 0.5 else 40)]

    assert evaluate(probe_of(grow)) == []
    [f] = evaluate(
        probe_of(lambda p: [BIG, text(13, "chú thích", rect(300, 300, 300, 40), font_size_rendered=24)])
    )
    assert f.rule == MIN_FONT and f.blocking and f.frames == ["85%", "100%"] and "cỡ 24px < 32px" in f.message


def test_text_over_text_needs_both_blocks_clearly_visible_and_not_nested():
    a = text(13, "Nhãn A", rect(300, 300, 400, 60))
    b = text(16, "Nhãn B", rect(500, 320, 400, 60))
    [f] = evaluate(probe_of(lambda p: [BIG, a, b]))
    assert f.rule == TEXT_OVERLAP and f.line == 13 and "giao nhau 200×40px" in f.message

    fading = dict(b, opacity=0.3)
    assert evaluate(probe_of(lambda p: [BIG, a, fading])) == []
    nested = dict(b, text_parent=1)  # b nằm trong a (a là phần tử thứ 1 của mẫu)
    assert evaluate(probe_of(lambda p: [BIG, a, nested])) == []


def test_hero_size_is_only_a_warning_and_text_only_shots_are_not_judged():
    small = kit(14, "Apple", rect(800, 400, 200, 200))
    [f] = evaluate(probe_of(lambda p: [small]))
    assert f.rule == HERO_SIZE and not f.blocking
    assert f.message == ("Shot 1.1: vật lớn nhất (hình Apple) chỉ chiếm 19% chiều khung (< 30%) — "
                         "khung dễ thành nền trống với vật nhỏ lọt thỏm")
    assert evaluate(probe_of(lambda p: [text(13, "Chương 2", rect(400, 400, 1000, 100))])) == []


def test_a_shot_that_throws_blocks_on_its_declaration_line_and_pending_assets_warn():
    def boom(p):
        return {"error": "Error: boom at 75\n    at Shot1_1 (eval)"} if p >= 0.5 else [BIG]

    [f] = evaluate(probe_of(boom, lines=(40,)))
    assert f.rule == RUNTIME_ERROR and f.blocking and f.line == 40
    assert "shot ném lỗi khi chạy: Error: boom at 75" in f.message and f.frames == ["50%", "85%", "100%"]

    [w] = evaluate(probe_of(lambda p: {"elements": [BIG], "pending_delay_render": 1} if p == 0 else [BIG]))
    assert w.rule == NOT_SETTLED and not w.blocking


def test_the_line_is_the_offending_tag_inside_the_shot_else_the_shot_declaration():
    # Hai shot: 1.1 khai báo ở dòng 40, 1.2 ở dòng 60, `const SHOTS` ở dòng 80.
    inside = kit(45, "Tooth", rect(1700, 300, 420, 420))
    from_library = kit(20, "SchoolBus", rect(1700, 300, 420, 420))  # hình thư viện dán phía trên các shot
    probe = probe_of(lambda p: [inside], lambda p: [from_library], lines=(40, 60), shots_line=80)
    found = sorted(evaluate(probe), key=lambda f: f.shot)
    assert [(f.shot, f.line, f.element_line) for f in found] == [("1.1", 45, 45), ("1.2", 60, 20)]
    assert "(vẽ ở dòng 20)" in found[1].message


def test_long_frame_lists_are_shortened():
    [f] = evaluate(probe_of(lambda p: [BIG, kit(14, "Sun", rect(1700, 200, 150 if p > 0 else 100, 100))]))
    assert f.message.startswith("Shot 1.1, frame 3%…100% (5/6 mẫu):")


@pytest.mark.parametrize("edge,px", [("left", 200), ("bottom", 0), ("top", 1080)])
def test_a_nonsense_subtitle_band_is_refused(edge, px):
    with pytest.raises(ValueError):
        SubtitleBand(edge, px)
