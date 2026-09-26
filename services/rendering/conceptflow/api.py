"""Danh sách tên công khai của `conceptflow`, dưới dạng dữ liệu thuần.

Tách khỏi `__init__.py` vì **lint phải đọc được danh sách này mà không import
manim**. `script_lint` chạy trong tiến trình của Rendering Service và trong test
suite — mà test suite của Rendering có một tính chất đáng giữ: nó không cần cài
manim (README nói rõ). Import `conceptflow/__init__.py` để lấy `__all__` sẽ kéo
theo cả manim và phá tính chất đó.

`tests/test_api_surface.py` khoá hai danh sách phải trùng nhau, nên không có
chuyện chúng trôi khỏi nhau trong im lặng.
"""

from __future__ import annotations

#: Nền của mọi video.
SCENE_NAMES = frozenset({"ConceptFlowScene"})

#: Component dựng cảnh (CR-017 FR45.2).
COMPONENT_NAMES = frozenset({
    "TitleCard",
    "Callout",
    "CodePanel",
    "StepList",
    "ComparisonSplit",
    "Recap",
    # Quan hệ và dữ liệu: sơ đồ, biểu đồ, đồ thị, bảng, dòng thời gian
    "FlowDiagram",
    "BarChart",
    "FunctionPlot",
    "DataTable",
    "Timeline",
    # Con số chạy được trong lúc quay — khuôn hình duy nhất có trạng thái
    "Readout",
})

#: Theme và hình học khung an toàn.
THEME_NAMES = frozenset({"Theme", "get", "register", "available"})
GEOMETRY_NAMES = frozenset({"Box", "safe_area", "overflow", "overlaps", "contrast_ratio"})

#: Vài thứ của Manim được tái xuất vì script thực sự cần: gom nhóm và chỉ hướng.
#:
#: Danh sách này cố tình ngắn. Mỗi tên thêm vào đây là một phần bề mặt Manim mở
#: lại cho script, và mở càng nhiều thì whitelist càng mất tác dụng.
REEXPORTED_MANIM_NAMES = frozenset({
    "VGroup",
    "UP",
    "DOWN",
    "LEFT",
    "RIGHT",
    "ORIGIN",
})

PUBLIC_NAMES: frozenset[str] = (
    SCENE_NAMES | COMPONENT_NAMES | THEME_NAMES | GEOMETRY_NAMES | REEXPORTED_MANIM_NAMES
)

#: Method công khai của ConceptFlowScene — bề mặt mà prompt sinh script liệt kê.
#: Lint KHÔNG dùng danh sách này để chặn `self.xxx()` (script được tự thêm
#: helper vào class của mình); `tests/conceptflow/test_components.py` khoá nó
#: trùng với method thật của scene để tài liệu không trôi khỏi code.
SCENE_METHODS = frozenset({
    "title", "heading", "body", "caption", "formula", "code",
    "stack", "row", "fit",
    "connect", "outline", "brace",
    # Hình cơ bản và số chạy, theo theme (thay cho API thô của Manim)
    "shape", "path", "readout", "count",
    "reveal", "dismiss", "swap", "emphasize", "travel", "clear_stage",
    # Camera
    "focus", "restore_view", "pace",
    # Beat dựng sẵn (CR-019 FR53)
    "hook", "hook_card", "recap", "recap_card", "call_to_action",
    "narrate", "beat", "chapter", "clip",
})
