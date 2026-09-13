"""Lời thoại khai báo bằng lời gọi runtime, không phải comment (CR-018).

## Vì sao đổi

Trước đây lời thoại là comment `# NARRATION: "..."` và điểm chờ là
`self.wait(AUTO)`, hai thứ tách rời mà Rendering phải khớp **theo số lượng và
theo thứ tự dòng trong file**. Toàn bộ chuỗi hạn chế phía sau bắt nguồn từ đó:
`wait(AUTO)` không được nằm trong vòng lặp hay nhánh điều kiện (vì sẽ chạy khác
số lần), nên lời thoại không đóng gói được vào hàm, nên hook/CTA không thể là
component — đúng lý do CR-006 §Quyết định #2 đã phải lùi FR17 xuống thành snippet.

`self.narrate("...")` gộp hai thứ làm một lời gọi. Danh sách lời thoại lấy theo
**thứ tự chạy thật** ở một lượt dry, nên số lượng và thứ tự tự khớp theo cấu
trúc — không còn gì để lệch.

## Hai lượt

- `CF_MODE=dry`   — ghi lời thoại ra JSONL rồi đi tiếp, không chờ. Chạy trước
  TTS, nên script sai bị chặn trước khi tiêu quota giọng đọc (CR-020 FR56).
- `CF_MODE=render` — tra thời lượng audio thật theo thứ tự, ghi mốc bắt đầu
  chờ, rồi chờ đúng bằng thời lượng đó.

Mốc bắt đầu (`kind="mark"`) giữ nguyên ngữ nghĩa CR-002: nó là thời điểm **bắt
đầu** khoảng chờ, tức là chỗ Video Assembly phải đặt đoạn audio. Đọc
`scene.renderer.time` TRƯỚC khi chờ, không phải sau.

Kênh liên lạc duy nhất ra khỏi subprocess vẫn là file JSONL trỏ bởi
`CF_MARKS_PATH`, y như trước — CR này chỉ thêm loại bản ghi, không mở thêm ống.
"""

from __future__ import annotations

import contextlib
import json
import os

MODE_DRY = "dry"
MODE_RENDER = "render"

ENV_MODE = "CF_MODE"
ENV_MARKS_PATH = "CF_MARKS_PATH"
ENV_DURATIONS_PATH = "CF_DURATIONS_PATH"


class NarrationError(RuntimeError):
    """Script gọi narrate nhiều lần hơn số thời lượng được cấp, hoặc môi
    trường thiếu thứ runtime cần."""


class _Recorder:
    """Trạng thái của một lượt chạy: đếm tới đâu và ghi vào đâu.

    Một instance cho mỗi tiến trình. Manim chạy đúng một scene trong một
    subprocess, nên không cần khoá đồng bộ.
    """

    def __init__(self) -> None:
        self._index = 0
        self._durations: list[float] | None = None
        # Tên của clip đang mở, hoặc None. Dùng để chặn `clip()` lồng nhau —
        # cùng một _recorder singleton nên state này thấy được từ mọi lời gọi
        # `clip()`, kể cả xuyên qua `self.narrate()` bên trong nó.
        self.open_clip: str | None = None

    @property
    def mode(self) -> str:
        return os.environ.get(ENV_MODE, MODE_RENDER)

    @property
    def marks_path(self) -> str:
        path = os.environ.get(ENV_MARKS_PATH)
        if not path:
            raise NarrationError(
                f"thiếu biến môi trường {ENV_MARKS_PATH} — script này phải được "
                "chạy qua Rendering Service, không chạy trực tiếp bằng `manim`"
            )
        return path

    def durations(self) -> list[float]:
        if self._durations is None:
            path = os.environ.get(ENV_DURATIONS_PATH)
            if not path or not os.path.isfile(path):
                raise NarrationError(
                    f"thiếu file thời lượng ({ENV_DURATIONS_PATH}) cho lượt render"
                )
            with open(path, encoding="utf-8") as f:
                self._durations = [float(value) for value in json.load(f)]
        return self._durations

    def write(self, record: dict) -> None:
        with open(self.marks_path, "a", encoding="utf-8") as f:
            f.write(json.dumps(record, ensure_ascii=False) + "\n")

    def next_index(self) -> int:
        index = self._index
        self._index += 1
        return index

    @property
    def upcoming_index(self) -> int:
        """Thứ tự của lời thoại **sắp tới**.

        `beat()` và `chapter()` gắn vào đây: một beat mở ra tại đúng lời thoại
        giới thiệu nó, nên timestamp của nó là mốc thật mà lượt render đo được
        chứ không phải ước lượng (giữ nguyên cách CR-006 FR15 đã làm).
        """
        return self._index


_recorder = _Recorder()


def reset() -> None:
    """Dùng trong test. Tiến trình render thật chỉ chạy một scene rồi thoát."""
    global _recorder
    _recorder = _Recorder()


def narrate(scene, text: str) -> None:
    """Phát một đoạn lời thoại tại đúng vị trí này trong dòng chảy của scene."""
    cleaned = text.strip()
    if not cleaned:
        raise NarrationError("narrate() nhận chuỗi rỗng")

    index = _recorder.next_index()

    if _recorder.mode == MODE_DRY:
        _recorder.write({
            "kind": "narration",
            "index": index,
            "text": cleaned,
            # CR-024 FR68.5: cái gì đang trên màn hình lúc câu này được nói.
            # Với một kênh đặt trọng tâm vào ví dụ trực quan, duyệt dàn ý mà chỉ
            # đọc được lời thoại là duyệt đúng nửa ít quan trọng hơn.
            "visual": _describe_stage(scene),
        })
        return

    durations = _recorder.durations()
    if index >= len(durations):
        raise NarrationError(
            f"script gọi narrate() lần thứ {index + 1} nhưng chỉ có "
            f"{len(durations)} đoạn audio. Lượt dry và lượt render đang chạy "
            "khác nhau — script có phần không tất định (random, thời gian thực)?"
        )

    # Đọc mốc TRƯỚC khi chờ: đây là thời điểm bắt đầu, thứ Video Assembly cần.
    now = scene.renderer.time
    _recorder.write({"kind": "mark", "index": index, "t": now})
    # CR-021 FR58.1: bố cục tại đúng mốc đó, để QC chấm được tràn khung /
    # chồng lấn / chữ nhỏ / tương phản mà không phải xem lại từng khung hình.
    # Chỉ ở lượt render: lượt dry là cổng chặn trước TTS, chỗ Creator đang chờ.
    _recorder.write({"kind": "layout", "index": index, "t": now,
                     "mobjects": _describe_layout(scene)})
    scene.wait(durations[index])


@contextlib.contextmanager
def clip(scene, name: str):
    """Đánh dấu một đoạn của scene là clip dọc phái sinh (CR-007 FR19.2).

    Ghi **một** bản ghi `kind="clip"` duy nhất, lúc `__exit__`, mang cả
    `t_start` lẫn `t_end`: gộp hai mốc vào một bản ghi thay vì ghi mở/đóng
    riêng vì `manim_renderer.py` chỉ cần đọc mỗi clip đúng một lần — hai bản
    ghi rời sẽ bắt phía Python phải tự ghép cặp theo `name`+`index`, thêm một
    chỗ có thể lệch mà một bản ghi duy nhất không có.

    `index` lấy tại `__enter__` (`upcoming_index`, giống `beat()`/`chapter()`):
    đó là lời thoại **sắp** chạy khi clip mở ra, bất kể `self.narrate()` bên
    trong block chạy bao nhiêu lần.

    Không cho lồng nhau: cắt một clip nằm trong một clip khác không có nghĩa
    rõ ràng (clip nào chứa clip nào khi xuất ra?), nên đây là lỗi ngay khi mở,
    không phải thứ âm thầm nhận lấy nghĩa nào đó.
    """
    cleaned = str(name).strip()
    if not cleaned:
        raise NarrationError("clip() nhận tên rỗng")
    if _recorder.open_clip is not None:
        raise NarrationError(
            f"clip() lồng nhau: đang ở trong clip {_recorder.open_clip!r} thì "
            f"mở thêm clip {cleaned!r} — cắt clip lồng nhau không có nghĩa rõ "
            "ràng, đóng clip trước bằng cách thoát khỏi khối `with` của nó"
        )

    index = _recorder.upcoming_index
    t_start = scene.renderer.time if _recorder.mode == MODE_RENDER else None
    _recorder.open_clip = cleaned
    try:
        yield
    finally:
        _recorder.open_clip = None
        t_end = scene.renderer.time if _recorder.mode == MODE_RENDER else None
        _recorder.write({
            "kind": "clip",
            "name": cleaned,
            "index": index,
            "t_start": t_start,
            "t_end": t_end,
        })


def is_dry_run() -> bool:
    """True trong lượt dry (CF_MODE=dry).

    Bug report (2026-09-12): phát hiện chồng lấn hình ảnh (`ConceptFlowScene`
    overlap check) chỉ chạy được ở đây — lượt render thật không cần trả tiền
    thêm cho một phép tính hình học chỉ có ích trước khi tốn TTS.
    """
    return _recorder.mode == MODE_DRY


def record_overlap(scene, description: str) -> None:
    """Ghi một cảnh báo chồng lấn hình ảnh phát hiện ở lượt dry.

    Bug report (2026-09-12): một `self.caption(...)` không định vị (rơi vào
    tâm khung hình mặc định của Manim) đã chồng khít lên một bảng/table đang
    hiện, cả hai không đọc được. `index` dùng `upcoming_index` — cùng quy ước
    với `beat()`/`chapter()` — vì chồng lấn được phát hiện GIỮA hai lời thoại,
    nên "lời thoại đang tới" là điểm quy chiếu duy nhất có ý nghĩa với người
    duyệt dàn ý.
    """
    _recorder.write({
        "kind": "overlap",
        "index": _recorder.upcoming_index,
        "description": str(description).strip(),
    })


def beat(scene, beat_id: str) -> None:
    """Đánh dấu mở đầu một beat trong beat sheet (CR-019)."""
    _recorder.write(
        {"kind": "beat", "index": _recorder.upcoming_index, "id": str(beat_id).strip()}
    )


def chapter(scene, title: str) -> None:
    """Đánh dấu mở đầu một chapter YouTube (CR-006 FR15, thay `# CHAPTER:`)."""
    cleaned = str(title).strip()
    if cleaned:
        _recorder.write(
            {"kind": "chapter", "index": _recorder.upcoming_index, "title": cleaned}
        )


def _describe_stage(scene) -> str:
    """Tóm tắt khung hình hiện tại thành một chuỗi đọc được.

    Đếm theo tên class chứ không mô tả nội dung: mô tả nội dung nghĩa là đoán
    xem hình đang nói gì, và đoán sai còn tệ hơn không nói. "Text×2, Arrow" là
    thứ kiểm chứng được và đủ để Creator nhận ra một beat chỉ toàn chữ.

    Best-effort tuyệt đối: đây là dữ liệu cho màn duyệt, không phải sản phẩm,
    nên mọi lỗi ở đây phải im lặng thay vì làm hỏng lượt dry.
    """
    try:
        counts: dict[str, int] = {}
        for mobject in getattr(scene, "mobjects", []):
            name = type(mobject).__name__
            counts[name] = counts.get(name, 0) + 1
        if not counts:
            return "khung trống"
        return ", ".join(
            name if count == 1 else f"{name}×{count}"
            for name, count in sorted(counts.items(), key=lambda kv: (-kv[1], kv[0]))
        )
    except Exception:  # noqa: BLE001 — xem docstring
        return ""


def _describe_layout(scene) -> list[dict]:
    """Bố cục khung hình hiện tại, dạng máy chấm được (CR-021 FR58.1).

    Mỗi mobject thành một bản ghi: tên class, hộp bao theo toạ độ Manim
    (trái/phải/trên/dưới), màu hex và cỡ chữ. Đó đúng là bốn thứ luật QC ở
    `video-assembly` cần — tràn khung và chồng lấn đọc bbox, chữ quá nhỏ đọc
    `font_size`, tương phản đọc `color`. Không cố đoán ý nghĩa hình, y như
    `_describe_stage`: chấm điểm là việc của luật, không phải của chỗ thu số.

    Best-effort tuyệt đối (FR58.2): đây là dữ liệu cho QC, không phải sản
    phẩm, nên một mobject lạ hay một API Manim đổi kiểu phải làm mất dữ liệu
    chứ không được làm hỏng lượt render đã tốn hàng phút.
    """
    try:
        described: list[dict] = []
        for mobject in getattr(scene, "mobjects", []):
            try:
                described.append({
                    "cls": type(mobject).__name__,
                    "bbox": [
                        float(mobject.get_left()[0]),
                        float(mobject.get_right()[0]),
                        float(mobject.get_top()[1]),
                        float(mobject.get_bottom()[1]),
                    ],
                    "color": _hex_color(mobject),
                    # Chỉ mobject chữ mới có; None là câu trả lời đúng cho
                    # phần còn lại, không phải một con số bịa ra.
                    "font_size": _optional_float(getattr(mobject, "font_size", None)),
                })
            except Exception:  # noqa: BLE001 — bỏ qua đúng một mobject, giữ phần còn lại
                continue
        return described
    except Exception:  # noqa: BLE001 — xem docstring
        return []


def _hex_color(mobject) -> str | None:
    """Màu của mobject dạng `#RRGGBB`, hoặc None nếu không đọc được.

    `Text` trong Manim 0.18 là một group các glyph: màu thật nằm ở glyph, còn
    `Text.get_color()` luôn trả `#000000` kể cả khi chữ đang vàng. Lấy màu tô
    của glyph đầu tiên cho nhóm có submobject, nếu không thì `get_color()` —
    nếu không thế thì luật tương phản (FR59.4) sẽ chấm sai mọi dòng chữ.
    """
    try:
        leaf = mobject
        while getattr(leaf, "submobjects", None):
            leaf = leaf.submobjects[0]
        color = leaf.get_fill_color() if leaf is not mobject else mobject.get_color()
    except Exception:  # noqa: BLE001
        return None
    # Manim 0.18 trả ManimColor có `.to_hex()`; các phiên bản/kiểu khác trả
    # thẳng chuỗi. Cả hai đều chấp nhận được, thứ QC cần chỉ là chuỗi hex.
    to_hex = getattr(color, "to_hex", None)
    if callable(to_hex):
        try:
            value = to_hex()
        except Exception:  # noqa: BLE001
            return None
    else:
        value = color
    if not isinstance(value, str):
        return None
    value = value.strip().upper()
    # `to_hex()` có thể kèm kênh alpha (#RRGGBBAA); QC chấm tương phản trên RGB.
    if len(value) == 9 and value.startswith("#"):
        value = value[:7]
    return value


def _optional_float(value) -> float | None:
    if value is None:
        return None
    try:
        return float(value)
    except (TypeError, ValueError):
        return None
