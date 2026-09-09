"""Thư viện component dựng cảnh (CR-017 FR45.2).

Sáu khuôn hình rút từ những gì các script đã sản xuất thực sự dùng
(`tests/fixtures/long_form_reference.py` và template trong `scriptTemplates.ts`),
không phải từ tưởng tượng — đúng cách giảm thiểu rủi ro mà CR-017 §Rủi ro đặt ra.
"""

from .base import Component
from .callout import Callout
from .code_panel import CodePanel
from .comparison import ComparisonSplit
from .recap import Recap
from .step_list import StepList
from .title_card import TitleCard

__all__ = [
    "Component",
    "TitleCard",
    "Callout",
    "CodePanel",
    "StepList",
    "ComparisonSplit",
    "Recap",
]
