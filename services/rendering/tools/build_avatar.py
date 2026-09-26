"""Dựng bộ avatar mèo `cat.*` từ clip gốc (CR-038).

    python tools/build_avatar.py            sinh mọi biến thể vào remotion_project/public/lottie/
    python tools/build_avatar.py --manifest cập nhật luôn remotion_project/lottie/manifest.json
    python tools/build_avatar.py idle sleep chỉ dựng một vài trạng thái

Đầu vào là remotion_project/lottie/source/bad-cat.lottie (clip do Creator cung cấp). Mọi biến thể
đều xuất phát từ cùng một "rig": thêm viền cho đuôi/chân/ly, ba sọc mướp, đuôi nằm dưới thân và
một miếng che ở gốc đuôi (để thân và đuôi liền một khối). Biểu cảm chỉ chỉnh mí mắt, con ngươi, vòng
trắng của mắt; chuyển động thân nhờ một lớp null làm cha; "đạo cụ" (dấu chấm than, chữ z, giọt mồ
hôi...) là các lớp hình thêm vào. Không có gì do LLM sinh: chạy lại luôn ra cùng một kết quả.

Manifest: clip mới luôn là `candidate`; chạy lại KHÔNG đụng vào status/giấy phép Creator đã điền.
"""

from __future__ import annotations

import argparse
import copy
import json
import sys
import zipfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))

from domain import lottie_catalog as catalog  # noqa: E402

PROJECT = ROOT / "remotion_project"
SOURCE = PROJECT / "lottie" / "source" / "bad-cat.lottie"
PUBLIC_DIR = PROJECT / "public" / "lottie"
MANIFEST = PROJECT / "lottie" / "manifest.json"

WHITE = [1, 1, 1, 1]
YELLOW = "#FFD215"
PINK = "#FF78D6"
RED = "#FF4D4D"
BLUE = "#8FD3FF"
SOFT = "#EDEDED"
PIVOT = [190, 305, 0]  # đáy thân, tâm co giãn/nảy của cả nhân vật
NULL_IND = 50


# --- phần tử Lottie --------------------------------------------------------------------------------

def rgba(hex_color: str) -> list[float]:
    n = int(hex_color.lstrip("#"), 16)
    return [((n >> 16) & 255) / 255, ((n >> 8) & 255) / 255, (n & 255) / 255, 1]


def static(value):
    return {"a": 0, "k": value}


def keys(points):
    """[(t, value), ...] -> thuộc tính có keyframe, dễ vào dễ ra."""
    out = []
    for n, (t, value) in enumerate(points):
        s = value if isinstance(value, list) else [value]
        frame = {"t": t, "s": s}
        if n < len(points) - 1:
            d = len(s)
            frame["i"] = {"x": [0.42] * d, "y": [1] * d}
            frame["o"] = {"x": [0.58] * d, "y": [0] * d}
        out.append(frame)
    return {"a": 1, "k": out}


def prop(value):
    if isinstance(value, list) and value and isinstance(value[0], tuple):
        return keys(value)
    return static(value)


def transform(p=(0, 0), s=(100, 100), r=0, o=100):
    return {"ty": "tr", "p": static(list(p)), "a": static([0, 0]), "s": static(list(s)), "r": static(r),
            "o": static(o), "sk": static(0), "sa": static(0), "nm": "Transform"}


def path(points, closed=False):
    zeros = [[0, 0]] * len(points)
    return {"ty": "sh", "nm": "path",
            "ks": static({"c": closed, "v": [list(p) for p in points], "i": zeros, "o": zeros})}


def arch(cx, cy, half_w, rise):
    """Nét cong hình ∩ dùng cho mắt cười."""
    return {"ty": "sh", "nm": "arch", "ks": static({
        "c": False,
        "v": [[cx - half_w, cy], [cx, cy - rise], [cx + half_w, cy]],
        "i": [[0, 0], [-half_w * 0.6, 0], [0, -rise * 0.9]],
        "o": [[0, rise * 0.9], [half_w * 0.6, 0], [0, 0]]})}


def ellipse(cx, cy, w, h):
    return {"ty": "el", "nm": "ellipse", "p": static([cx, cy]), "s": static([w, h])}


def fill(hex_color):
    return {"ty": "fl", "nm": "fill", "c": static(rgba(hex_color)), "o": static(100), "r": 1}


def stroke(hex_color, width):
    return {"ty": "st", "nm": "stroke", "c": static(rgba(hex_color)), "o": static(100), "w": static(width),
            "lc": 2, "lj": 2}


def group(name, items):
    return {"ty": "gr", "nm": name, "it": items + [transform()]}


def shape_layer(ind, name, shapes, op, p=(0, 0, 0), s=(100, 100, 100), o=100, parent=None):
    layer = {
        "ddd": 0, "ind": ind, "ty": 4, "nm": name, "sr": 1,
        "ks": {"o": prop(o), "r": static(0), "p": prop(list(p) if not isinstance(p, list) else p),
               "a": static([0, 0, 0]), "s": prop(list(s) if not isinstance(s, list) else s)},
        "ao": 0, "shapes": shapes, "ip": 0, "op": op, "st": 0, "bm": 0,
    }
    if parent is not None:
        layer["parent"] = parent
    return layer


# --- rig --------------------------------------------------------------------------------------------

def load_base() -> dict:
    with zipfile.ZipFile(SOURCE) as z:
        name = next(n for n in z.namelist() if n.startswith("animations/") and n.endswith(".json"))
        return json.loads(z.read(name))


def layer_of(d: dict, name: str) -> dict:
    return next(layer for layer in d["layers"] if layer["nm"] == name)


def build_rig() -> dict:
    d = load_base()
    layers = {layer["nm"]: layer for layer in d["layers"]}

    # Ba sọc mướp trên trán (toạ độ của lớp thân: đỉnh đầu y≈-58, mắt y≈-25).
    stripes = group("Stripes", [
        path([(-21, -52), (-21, -38)]), path([(-37, -51), (-34, -39)]), path([(-5, -51), (-8, -39)]),
        stroke(SOFT, 3.2)])
    # Lớp riêng nằm trên mắt: mí xếch/sụp của biểu cảm không được che mất sọc.
    d["layers"].insert(0, shape_layer(30, "Stripes", [stripes], d["op"], p=[190, 183, 0]))

    # Viền cho chân và ly, giống viền mảnh của thân (stroke đặt trước fill để nằm trên nền tô).
    for name, width in (("Lapa", 2), ("cup", 3)):
        items = layers[name]["shapes"][0]["it"]
        rim = stroke("#FFFFFF", width)
        rim["nm"] = "Rim"
        items.insert(next(i for i, s in enumerate(items) if s["ty"] == "fl"), rim)

    # Đuôi: nét đen rộng 20 -> thêm nét trắng rộng 24 nằm dưới, rồi đưa cả lớp xuống dưới thân.
    tail_group = layers["tale"]["shapes"][0]
    under = copy.deepcopy(tail_group)
    under["nm"] = "Rim under"
    for s in under["it"]:
        if s["ty"] == "st":
            s["c"]["k"] = WHITE
            s["w"]["k"] = 24
    layers["tale"]["shapes"].append(under)
    d["layers"].remove(layers["tale"])
    d["layers"].insert(d["layers"].index(layers["body"]) + 1, layers["tale"])

    # Viền mép phải của thân cắt ngang gốc đuôi: che đoạn đó bằng một miếng đen nằm trong lòng đuôi.
    patch = group("Tail joint patch", [
        path([(58, 100.8), (69.5, 100.8), (69.5, 120.4), (58, 120.4)], closed=True), fill("#000000")])
    layers["body"]["shapes"].insert(0, patch)
    return d


# --- mắt --------------------------------------------------------------------------------------------

def eye_group(layer: dict, name: str) -> dict:
    return next(s for s in layer["shapes"] if s["ty"] == "gr" and s["nm"] == name)


def group_tr(g: dict) -> dict:
    return next(i for i in g["it"] if i["ty"] == "tr")


LID_P = [-1.375, -42.688]
PUPIL_P = [-0.812, -25.125]


def blink_keys(base_sy: float, at: int, op: int):
    """Chớp một lần bắt đầu ở khung `at` (mí đi xuống rồi lên lại), giữ nguyên độ mở nền."""
    return [(0, [100, base_sy]), (at, [100, base_sy]), (at + 14, [100, 200]), (at + 26, [100, base_sy]),
            (op, [100, base_sy])]


def set_eyes(d: dict, *, op: int, lid_tilt=0, lid_dy=0, lid_sx=100, lid_sy=100, blink_at=None,
             pupil=(0, 0), pupil_scale=100, white_scale=100, happy=False, look=None, hide_lid=False):
    """`lid_tilt` là độ nghiêng VÀO trong: dương thì đuôi mắt cao hơn (giận), âm thì đuôi mắt thấp (buồn)."""
    for name, sign in (("eye 2", 1), ("eye", -1)):
        layer = layer_of(d, name)
        lid_tr = group_tr(eye_group(layer, "Ellipse 3"))
        pupil_tr = group_tr(eye_group(layer, "Ellipse 1"))
        white_tr = group_tr(eye_group(layer, "Ellipse 2"))

        lid_tr["p"] = static([LID_P[0], LID_P[1] + lid_dy])
        lid_tr["r"] = static(lid_tilt * sign)
        lid_tr["s"] = keys(blink_keys(lid_sy, blink_at, op)) if blink_at is not None else static([lid_sx, lid_sy])
        if blink_at is not None and lid_sx != 100:
            for frame in lid_tr["s"]["k"]:
                frame["s"][0] = lid_sx
        if hide_lid:  # mắt mở to: bỏ hẳn mí (dời mí lên sẽ đè mất viền đỉnh đầu)
            lid_tr["o"] = static(0)
        pupil_tr["s"] = static([pupil_scale, pupil_scale])
        white_tr["s"] = static([white_scale, white_scale])
        if look:
            pupil_tr["p"] = keys([(t, [PUPIL_P[0] + dx, PUPIL_P[1] + dy]) for t, dx, dy in look])
        else:
            pupil_tr["p"] = static([PUPIL_P[0] + pupil[0], PUPIL_P[1] + pupil[1]])

        if happy:  # mắt cười ^ ^: ẩn vòng trắng, con ngươi, mí; vẽ một nét cong
            for g in (eye_group(layer, "Ellipse 1"), eye_group(layer, "Ellipse 2"), eye_group(layer, "Ellipse 3")):
                group_tr(g)["o"] = static(0)
            layer["shapes"].insert(0, group("Happy eye", [arch(-0.75, -20, 11, 13), stroke("#FFFFFF", 5)]))


# --- đuôi, thân -------------------------------------------------------------------------------------

def set_tail(d: dict, *, op: int, period: int = 62, flat: bool = False):
    tail = layer_of(d, "tale")
    sh = next(i for i in tail["shapes"][0]["it"] if i["ty"] == "sh")
    base = sh["ks"]["k"]
    if flat:  # đuôi nằm yên: lấy dáng ở khung 19 (thẳng dọc mặt bàn)
        flat_frame = next(f for f in base if f["t"] == 19)
        sh["ks"] = {"a": 0, "k": copy.deepcopy(flat_frame["s"][0])}
        for g in tail["shapes"][1:]:
            for item in g["it"]:
                if item["ty"] == "sh":
                    item["ks"] = {"a": 0, "k": copy.deepcopy(flat_frame["s"][0])}
        return
    cycles = max(1, round(op / period))
    frames = []
    for k in range(cycles):
        for f in base[:-1]:
            nf = copy.deepcopy(f)
            nf["t"] = round(k * period + f["t"] * period / 62)
            frames.append(nf)
    last = copy.deepcopy(base[-1])
    last["t"] = cycles * period
    frames.append(last)
    for item_group in tail["shapes"]:
        for item in item_group["it"]:
            if item["ty"] == "sh":
                item["ks"] = {"a": 1, "k": copy.deepcopy(frames)}


def add_mover(d: dict, *, op: int, p=None, s=None):
    """Lớp null làm cha của mọi phần thân để nảy, rung, thở cả nhân vật một lượt."""
    for layer in d["layers"]:
        layer["parent"] = NULL_IND
    null = {
        "ddd": 0, "ind": NULL_IND, "ty": 3, "nm": "Mover", "sr": 1,
        "ks": {"o": static(100), "r": static(0), "p": prop(p if p else list(PIVOT)),
               "a": static(list(PIVOT)), "s": prop(s if s else [100, 100, 100])},
        "ao": 0, "ip": 0, "op": op, "st": 0, "bm": 0,
    }
    d["layers"].append(null)


def at(dx=0, dy=0):
    return [PIVOT[0] + dx, PIVOT[1] + dy, 0]


def finish(d: dict, op: int):
    d["op"] = op
    for layer in d["layers"]:
        layer["op"] = op
    return d


# --- đạo cụ -----------------------------------------------------------------------------------------

def pops(ind, name, shapes, op, at_xy, times):
    """Nảy lên rồi lặp: `times` = [(t_bắt_đầu, t_kết_thúc)]; ngoài khoảng đó thì thu về 0."""
    pts = [(0, [0, 0, 100])]
    for a, b in times:
        pts += [(a, [0, 0, 100]), (a + 6, [140, 140, 100]), (a + 12, [100, 100, 100]), (b - 8, [100, 100, 100]),
                (b, [0, 0, 100])]
    pts.append((op, [0, 0, 100]))
    return shape_layer(ind, name, shapes, op, p=[at_xy[0], at_xy[1], 0], s=pts)


def floating(ind, name, shapes, op, p0, p1, start, dur, s0=0.6, s1=1.1):
    """Trôi từ p0 tới p1 trong `dur` khung, mờ vào rồi mờ ra."""
    def scale(v):
        return [v * 100, v * 100, 100]
    pos = [(0, [p0[0], p0[1], 0]), (start, [p0[0], p0[1], 0]), (start + dur, [p1[0], p1[1], 0]),
           (op, [p1[0], p1[1], 0])]
    sc = [(0, scale(s0)), (start, scale(s0)), (start + dur, scale(s1)), (op, scale(s1))]
    alpha = [(0, 0), (start, 0), (start + dur * 0.25, 100), (start + dur * 0.75, 100), (start + dur, 0), (op, 0)]
    return shape_layer(ind, name, shapes, op, p=pos, s=sc, o=alpha)


def glyph_exclaim():
    return [group("bar", [path([(0, -14), (0, -2)]), stroke(YELLOW, 5)]),
            group("dot", [ellipse(0, 5, 5.5, 5.5), fill(YELLOW)])]


def glyph_question():
    hook = [(-6, -8), (-4, -12), (0, -14), (4, -12), (6, -8), (4, -4), (0, -1), (0, 3)]
    return [group("hook", [path(hook), stroke(YELLOW, 3.4)]), group("dot", [ellipse(0, 9, 4.6, 4.6), fill(YELLOW)])]


def glyph_z():
    return [group("z", [path([(-4, -4), (4, -4), (-4, 4), (4, 4)]), stroke(SOFT, 2.4)])]


def glyph_heart():
    return [group("heart", [ellipse(-3.6, -2, 8, 8), ellipse(3.6, -2, 8, 8),
                            path([(-7.2, 0.5), (7.2, 0.5), (0, 8.5)], closed=True), fill(PINK)])]


def glyph_drop():
    return [group("drop", [path([(0, -8), (3.4, -1), (-3.4, -1)], closed=True), ellipse(0, 1.2, 7, 7), fill(BLUE)])]


def glyph_anger():
    corners = [[(-8, -3), (-3, -3), (-3, -8)], [(8, -3), (3, -3), (3, -8)],
               [(-8, 3), (-3, 3), (-3, 8)], [(8, 3), (3, 3), (3, 8)]]
    return [group("anger", [path(c) for c in corners] + [stroke(RED, 2.8)])]


# --- các trạng thái ---------------------------------------------------------------------------------

def drop_props(d, names=("Lapa", "cup")):
    d["layers"] = [layer for layer in d["layers"] if layer["nm"] not in names]


def state_idle(rig):
    d = copy.deepcopy(rig); drop_props(d); op = 124
    set_eyes(d, op=op, blink_at=70); set_tail(d, op=op)
    add_mover(d, op=op)
    return finish(d, op)


def state_smug(rig):
    d = copy.deepcopy(rig); drop_props(d); op = 124
    set_eyes(d, op=op, lid_dy=1.5, lid_sy=122, pupil=(1.5, 1.5), blink_at=64)
    set_tail(d, op=op, period=62)
    add_mover(d, op=op, s=[(0, [100, 100, 100]), (62, [100, 101.2, 100]), (124, [100, 100, 100])])
    return finish(d, op)


def state_surprised(rig):
    d = copy.deepcopy(rig); drop_props(d); op = 65
    set_eyes(d, op=op, hide_lid=True, white_scale=120, pupil_scale=55)
    set_tail(d, op=op)
    add_mover(d, op=op, p=[(0, at()), (5, at(0, -12)), (12, at()), (18, at(0, -3)), (24, at()), (op, at())])
    d["layers"].append(pops(20, "Exclaim", glyph_exclaim(), op, (236, 108), [(0, 62)]))
    return finish(d, op)


def state_angry(rig):
    d = copy.deepcopy(rig); drop_props(d); op = 66
    set_eyes(d, op=op, lid_tilt=22, lid_dy=0, lid_sx=130, lid_sy=128, pupil_scale=85, pupil=(0, 1))
    set_tail(d, op=op, period=33)
    shake = [(t, at(1.6 if (t // 3) % 2 == 0 else -1.6, 0)) for t in range(0, op, 3)] + [(op, at(1.6, 0))]
    add_mover(d, op=op, p=shake)
    pulse = [(0, [130, 130, 100]), (16, [160, 160, 100]), (33, [130, 130, 100]), (49, [160, 160, 100]),
             (op, [130, 130, 100])]
    d["layers"].append(shape_layer(20, "Anger", glyph_anger(), op, p=[229, 122, 0], s=pulse))
    return finish(d, op)


def state_happy(rig):
    d = copy.deepcopy(rig); drop_props(d); op = 64
    set_eyes(d, op=op, happy=True)
    set_tail(d, op=op, period=32)
    bob = [(0, at())]
    for t in range(8, op + 1, 8):
        bob.append((t, at(0, -7 if (t // 8) % 2 == 1 else 0)))
    add_mover(d, op=op, p=bob)
    d["layers"].append(floating(20, "Heart A", glyph_heart(), op, (228, 122), (240, 80), 0, 52, 1.0, 1.7))
    d["layers"].append(floating(21, "Heart B", glyph_heart(), op, (218, 116), (206, 78), 24, 40, 0.8, 1.4))
    return finish(d, op)


def state_sleep(rig):
    d = copy.deepcopy(rig); drop_props(d); op = 124
    set_eyes(d, op=op, lid_sy=200)
    set_tail(d, op=op, flat=True)
    add_mover(d, op=op, s=[(0, [100, 100, 100]), (62, [100, 102.6, 100]), (124, [100, 100, 100])])
    for n, start in enumerate((0, 22, 44)):
        d["layers"].append(floating(20 + n, f"Z {n + 1}", glyph_z(), op, (232, 118), (250, 76), start, 62,
                                    1.3 + 0.4 * n, 1.9 + 0.5 * n))
    return finish(d, op)


def state_thinking(rig):
    d = copy.deepcopy(rig); drop_props(d); op = 124
    set_eyes(d, op=op, lid_sy=115, pupil=(3.5, -4.5), blink_at=84)
    set_tail(d, op=op)
    add_mover(d, op=op)
    bob = [(0, [236, 112, 0]), (31, [236, 107, 0]), (62, [236, 112, 0]), (93, [236, 107, 0]), (124, [236, 112, 0])]
    d["layers"].append(shape_layer(20, "Question", glyph_question(), op, p=bob))
    return finish(d, op)


def state_sad(rig):
    d = copy.deepcopy(rig); drop_props(d); op = 124
    set_eyes(d, op=op, lid_tilt=-24, lid_dy=0, lid_sx=122, lid_sy=118, pupil=(0, 3), pupil_scale=90)
    set_tail(d, op=op, flat=True)
    add_mover(d, op=op, s=[(0, [100, 98, 100]), (62, [100, 96.6, 100]), (124, [100, 98, 100])])
    for n, start in enumerate((0, 62)):
        d["layers"].append(floating(20 + n, f"Drop {n + 1}", glyph_drop(), op, (204, 138), (206, 176), start, 56,
                                    1.4, 1.7))
    return finish(d, op)


def state_look(rig):
    d = copy.deepcopy(rig); drop_props(d); op = 124
    look = [(0, 0, 0), (14, -5, 0), (44, -5, 0), (58, 5, 0), (92, 5, 0), (106, 0, 0), (op, 0, 0)]
    set_eyes(d, op=op, look=look, lid_sy=110)
    set_tail(d, op=op)
    add_mover(d, op=op)
    return finish(d, op)


def state_push(rig):
    d = copy.deepcopy(rig); op = 65
    add_mover(d, op=op)
    return finish(d, op)


STATES = {
    "idle": (state_idle, "Mèo ngồi lườm người xem, đuôi vẫy chậm, thỉnh thoảng chớp mắt. Trạng thái mặc định.",
             ["avatar", "mặc định", "lườm"]),
    "smug": (state_smug, "Mèo nheo mắt tự mãn, ánh mắt liếc về phía người xem. Dùng sau khi nhân vật làm chuyện dại dột.",
             ["avatar", "tự mãn", "gian"]),
    "surprised": (state_surprised, "Mèo giật mình, mắt tròn to, nảy lên và hiện dấu chấm than. Dùng ở cú xoay hoặc khoảnh khắc Aha.",
                  ["avatar", "giật mình", "bất ngờ"]),
    "angry": (state_angry, "Mèo tức giận, mắt xếch, cả người rung, hiện dấu giận đỏ. Dùng khi kết quả sai hoặc bị làm phiền.",
              ["avatar", "giận", "bực"]),
    "happy": (state_happy, "Mèo vui, mắt cười, nảy nhẹ, đuôi vẫy nhanh, bay ra trái tim. Dùng khi hiểu ra hoặc kết thúc tốt.",
              ["avatar", "vui", "ăn mừng"]),
    "sleep": (state_sleep, "Mèo ngủ, thở đều, chữ z bay lên. Dùng khi chờ, khi lời thoại kéo dài hoặc khi nội dung tẻ nhạt.",
              ["avatar", "ngủ", "chờ"]),
    "thinking": (state_thinking, "Mèo ngước mắt suy nghĩ, dấu hỏi nhấp nhô trên đầu. Dùng khi đặt câu hỏi cho người xem.",
                 ["avatar", "suy nghĩ", "câu hỏi"]),
    "sad": (state_sad, "Mèo buồn, mắt sụp, cúi xuống, giọt mồ hôi rơi. Dùng khi thất bại hoặc tình huống xấu đi.",
            ["avatar", "buồn", "thất bại"]),
    "look": (state_look, "Mèo đảo mắt trái rồi phải rồi về giữa. Dùng khi kể chuyện có hai vế hoặc khi nghi ngờ.",
             ["avatar", "đảo mắt", "nghi ngờ"]),
    "push": (state_push, "Mèo dùng chân đẩy cái ly khỏi mép bàn, mắt lườm. Gag đặc trưng: cách làm sai leo thang.",
             ["avatar", "hành động", "gag", "đẩy ly"]),
}


#: Khung clip là hình vuông 400x400 nhưng mèo không nằm giữa khung; `LottieClip` đặt theo tâm khung,
#: nên LLM cần biết để không đặt mèo lệch chỗ.
PLACEMENT_NOTE = (" Khung vuông; mèo nằm nửa dưới, hơi lệch trái, đuôi kéo sang phải"
                  " (x, y là tâm khung chứ không phải tâm mèo; size 500-700 là cỡ vừa).")


def sync_manifest(built: dict[str, dict]) -> None:
    raw = json.loads(MANIFEST.read_text(encoding="utf-8")) if MANIFEST.exists() else {"assets": []}
    by_id = {a["id"]: a for a in raw["assets"]}
    for state, data in built.items():
        _, description, tags = STATES[state]
        entry = by_id.setdefault(f"cat.{state}", {
            # Chưa biết tác giả và giấy phép riêng của clip gốc -> ứng viên, chờ Creator xác minh.
            "id": f"cat.{state}", "title": f"Mèo {state}", "license": "Lottie Simple License",
            "source_url": "Creator cung cấp: clip Bad Cat (LottieFiles), chưa rõ trang nguồn",
            "author": "Chưa rõ", "status": catalog.CANDIDATE, "license_checked": ""})
        entry["description"] = description + PLACEMENT_NOTE
        entry["tags"] = tags
        entry["loop"] = True
        entry["palette"] = sorted(catalog.extract_palette(data))
        entry["notes"] = "Dựng từ lottie/source/bad-cat.lottie bằng tools/build_avatar.py."
    raw["assets"] = sorted(by_id.values(), key=lambda a: a["id"])
    MANIFEST.write_text(json.dumps(raw, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("states", nargs="*", help=f"mặc định: tất cả ({', '.join(STATES)})")
    parser.add_argument("--manifest", action="store_true", help="cập nhật manifest.json")
    args = parser.parse_args()
    wanted = args.states or list(STATES)
    unknown = [s for s in wanted if s not in STATES]
    if unknown:
        parser.error(f"trạng thái không có: {', '.join(unknown)}")

    rig = build_rig()
    PUBLIC_DIR.mkdir(parents=True, exist_ok=True)
    built = {}
    for state in wanted:
        data = STATES[state][0](rig)
        (PUBLIC_DIR / f"cat.{state}.json").write_text(json.dumps(data, separators=(",", ":")), encoding="utf-8")
        built[state] = data
        info = catalog.lottie_info(data)
        print(f"cat.{state}: {info['seconds']}s, {info['frames']:.0f} khung")
    if args.manifest:
        sync_manifest(built)
        print("manifest cập nhật:", MANIFEST)
    return 0


if __name__ == "__main__":
    sys.exit(main())
