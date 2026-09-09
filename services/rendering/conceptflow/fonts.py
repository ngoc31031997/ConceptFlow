"""Kiểm tra font đã có mặt thật chưa (CR-017 §Quyết định #3).

Manim/Pango **im lặng thay bằng font khác** khi font yêu cầu không tồn tại. Với
một kênh mà bản sắc nằm ở chữ, hỏng kiểu đó không ai nhận ra cho tới khi nhìn
video đã publish — nên phải hỏi thẳng fontconfig thay vì tin rằng Dockerfile đã
cài đúng.

`VIETNAMESE_PROBE` là chuỗi dùng để xác nhận font mang đủ dấu tiếng Việt. Chọn
các tổ hợp khó nhất (dấu chồng dấu, nguyên âm có móc), vì một font "hỗ trợ
Latin Extended" vẫn có thể thiếu đúng những chữ này.
"""

from __future__ import annotations

import subprocess

#: Không tin mô tả trên Google Fonts — render chuỗi này rồi nhìn.
VIETNAMESE_PROBE = "ỗ ự ằ ẩ ợ ữ ẵ ọ ườ nghiêng"


def installed_families() -> set[str]:
    """Các họ font fontconfig đang thấy. Rỗng nếu không có `fc-list`."""
    try:
        result = subprocess.run(
            ["fc-list", "--format", "%{family[0]}\\n"],
            capture_output=True, text=True, timeout=10, check=False,
        )
    except (OSError, subprocess.SubprocessError):
        return set()
    if result.returncode != 0:
        return set()
    return {line.strip() for line in result.stdout.splitlines() if line.strip()}


def missing(*families: str) -> tuple[str, ...]:
    """Những font trong `families` mà hệ thống không có.

    Trả về tuple rỗng khi không kiểm tra được (không có `fc-list`), thay vì báo
    thiếu hết: đây là công cụ chẩn đoán, không phải cổng chặn — làm hỏng lượt
    render vì thiếu `fc-list` sẽ tệ hơn nhiều so với việc render bằng font thay thế.
    """
    present = installed_families()
    if not present:
        return ()
    return tuple(f for f in families if f not in present)
