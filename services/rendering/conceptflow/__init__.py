"""ConceptFlow — bề mặt API mà script video được phép dùng (CR-017).

Script của Creator mở đầu bằng `from conceptflow import *`, không phải
`from manim import *`. Danh sách `__all__` dưới đây **chính là whitelist** mà
lint (CR-017 FR46.1) đối chiếu: mọi thứ ngoài đây bị coi là API thô của Manim.

Đường thoát hiểm (FR45.5): `from manim import ...` vẫn dùng được khi component
không đủ diễn đạt. Lint cảnh báo chứ không chặn — thư viện chặn được cả những
video tham vọng nhất thì nó đang làm hại chứ không giúp.
"""

from manim import DOWN, LEFT, ORIGIN, RIGHT, UP, VGroup

from .components import (
    Callout,
    CodePanel,
    ComparisonSplit,
    Recap,
    StepList,
    TitleCard,
)
from .api import PUBLIC_NAMES
from .layout import Box, contrast_ratio, overflow, overlaps, safe_area
from .scene import ConceptFlowScene
from .theme import Theme, available, get, register

__all__ = [
    # Nền của mọi video
    "ConceptFlowScene",
    # Component dựng cảnh
    "TitleCard",
    "Callout",
    "CodePanel",
    "StepList",
    "ComparisonSplit",
    "Recap",
    # Theme
    "Theme",
    "get",
    "register",
    "available",
    # Hình học khung an toàn (cũng dùng cho QC — CR-021)
    "Box",
    "safe_area",
    "overflow",
    "overlaps",
    "contrast_ratio",
    # Tái xuất từ Manim — xem api.REEXPORTED_MANIM_NAMES
    "VGroup",
    "UP",
    "DOWN",
    "LEFT",
    "RIGHT",
    "ORIGIN",
]

# Khoá hai nguồn sự thật phải trùng nhau ngay lúc import, chứ không đợi test:
# lint đọc `api.PUBLIC_NAMES`, script đọc `__all__`. Lệch nhau thì lint sẽ từ
# chối đúng thứ mà thư viện vừa cấp cho script.
assert set(__all__) == set(PUBLIC_NAMES), (
    "conceptflow.__all__ và conceptflow.api.PUBLIC_NAMES đã lệch nhau: "
    f"{set(__all__) ^ set(PUBLIC_NAMES)}"
)
