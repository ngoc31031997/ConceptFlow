"""ValidateScriptUseCase — cổng kiểm tra trước khi tốn TTS (CR-020 FR56).

Chạy hai việc, theo thứ tự rẻ trước:

1. **Lint** (CR-017 FR46) — phân tích tĩnh, mili-giây. Bắt API ngoài design
   system, màu hex viết thẳng, cỡ chữ đặt tay.
2. **Lượt dry** (CR-018 FR49.1) — thực sự chạy script qua Manim. Đây là thứ
   thay thế danh sách blacklist: nó bắt **mọi** lỗi API, kể cả những cái chưa
   ai gặp bao giờ, vì nó không đoán mà chạy thật.

Đồng thời lượt dry trả về danh sách lời thoại theo đúng thứ tự chạy — thứ mà
Saga cần để gọi TTS.

Vì sao bước này sống ở Rendering chứ không ở một Quality Service riêng: lượt dry
cần Manim, và Manim kéo theo cả texlive. Dựng image thứ hai nặng như vậy chỉ để
chạy một lệnh là cái giá không đáng. Bước chấm chất lượng **video** (CR-021)
thì ngược lại — nó chỉ cần ffmpeg và file JSONL, nên sẽ sống ở service riêng.
"""

from __future__ import annotations

import logging
from dataclasses import dataclass, field

from domain.models import DryRunResult, ScriptRenderRequest
from domain.ports import ManimScriptRendererPort
from domain.script_lint import LintIssue, blocking_issues, lint_manim_script

logger = logging.getLogger(__name__)


@dataclass(frozen=True)
class ValidationResult:
    """Kết quả kiểm tra một script.

    `warnings` không chặn Saga (FR56.3): đường thoát hiểm ra API thô của Manim
    là hợp lệ, chỉ là phần đó không được theme và QC bảo vệ.
    """

    dry_run: DryRunResult
    warnings: list[LintIssue] = field(default_factory=list)


class ScriptValidationError(Exception):
    """Script không qua được cổng. `issues` rỗng nghĩa là lượt dry đã chết."""

    def __init__(self, message: str, issues: list[LintIssue] | None = None) -> None:
        super().__init__(message)
        self.issues = issues or []


class ValidateScriptUseCase:
    def __init__(self, renderer: ManimScriptRendererPort) -> None:
        self._renderer = renderer

    def validate(self, request: ScriptRenderRequest) -> ValidationResult:
        if not request.script_content.strip():
            raise ScriptValidationError("script_content rỗng")
        if not request.scene_class_name:
            raise ScriptValidationError("thiếu scene_class_name")

        issues = lint_manim_script(request.script_content)
        blocking = blocking_issues(issues)
        if blocking:
            # Dừng ở đây: chạy lượt dry cho một script đã biết là sai chỉ tốn
            # thêm một subprocess để nhận cùng câu trả lời.
            raise ScriptValidationError(
                "; ".join(str(issue) for issue in blocking), issues=blocking
            )

        warnings = [issue for issue in issues if not issue.is_blocking]
        for issue in warnings:
            logger.warning("lint script (%s): %s", request.project_id, issue)

        return ValidationResult(dry_run=self._renderer.dry_run(request), warnings=warnings)
