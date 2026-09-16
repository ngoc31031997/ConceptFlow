"""Domain value objects for the Script Processing Service.

Sau CR-018 service này thu hẹp còn đúng một việc: tìm tên class Scene mà
Rendering phải chạy. `Scene` và `Chapter` đã rời khỏi đây — chúng do lượt dry
sinh ra theo thứ tự chạy thật, không còn đọc được từ text script.
"""

from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class ParsedScript:
    """Service này không giữ trạng thái gì — không lưu lại raw_script.

    scene_class_name giữ tên định danh Rendering phải chạy: tên class Manim
    Scene cho engine "manim", hoặc `id` của `<Composition>` cho engine
    "remotion" (Root.tsx tra `id` đó trong `selectComposition`/`renderMedia`
    — xem services/rendering/adapters/rendering/remotion_renderer.py).
    """

    scene_class_name: str
    # "manim" (mặc định) | "remotion" — engine nào sinh ra định danh trên.
    engine: str = "manim"
