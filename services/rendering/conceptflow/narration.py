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
    _recorder.write({"kind": "mark", "index": index, "t": scene.renderer.time})
    scene.wait(durations[index])


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
