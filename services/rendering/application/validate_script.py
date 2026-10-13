"""ValidateScriptUseCase — cổng kiểm tra trước khi tốn TTS.

Chạy hai việc, theo thứ tự rẻ trước:

1. **Lint** — phân tích tĩnh, mili-giây: mọi id clip Lottie phải có trong
   catalog đã duyệt.
2. **Lượt dry** — đọc composition để biết danh sách lời thoại theo đúng thứ tự
   chạy, thứ mà Saga cần để gọi TTS.

Bước này sống ở Rendering vì lượt dry cần chính toolchain dựng hình.
"""

from __future__ import annotations

import dataclasses
import logging
from collections.abc import Callable
from dataclasses import dataclass, field

from domain.errors import UnsupportedEngineError
from domain.lint_issue import LintIssue, blocking_issues
from domain.lottie_catalog import lint_lottie_ids
from domain.models import DryRunResult, ScriptRenderRequest
from domain.ports import ScriptRendererPort
from domain.script_locator import SceneNotFoundError, locate_scene

logger = logging.getLogger(__name__)


@dataclass(frozen=True)
class ValidationResult:
    """Kết quả kiểm tra một script. `warnings` không chặn Saga."""

    dry_run: DryRunResult
    # The composition the saga will render, found here (before the dry pass
    # needs it) instead of by a separate parse step.
    scene_class_name: str = ""
    engine: str = "remotion"
    warnings: list[LintIssue] = field(default_factory=list)


class ScriptValidationError(Exception):
    """Script không qua được cổng. `issues` rỗng nghĩa là lượt dry đã chết."""

    def __init__(self, message: str, issues: list[LintIssue] | None = None) -> None:
        super().__init__(message)
        self.issues = issues or []


class ValidateScriptUseCase:
    def __init__(
        self,
        renderer: ScriptRendererPort,
        approved_lottie_ids: Callable[[], set[str]] | None = None,
    ) -> None:
        self._renderer = renderer
        # Ids of the approved Lottie catalog. None disables the check
        # (tests that never touch the catalog), an empty set means "no clip is
        # allowed" — which is exactly what an empty catalog should enforce.
        self._approved_lottie_ids = approved_lottie_ids

    def validate(self, request: ScriptRenderRequest) -> ValidationResult:
        if request.engine != "remotion":
            raise UnsupportedEngineError(request.engine)
        if not request.script_content.strip():
            raise ScriptValidationError("script_content rỗng")
        if not request.scene_class_name:
            try:
                located = locate_scene(request.script_content)
            except SceneNotFoundError as exc:
                raise ScriptValidationError(str(exc)) from exc
            request = dataclasses.replace(request, scene_class_name=located.scene_class_name)

        issues: list[LintIssue] = []
        if self._approved_lottie_ids is not None:
            issues += [
                LintIssue(line=line, message=message)
                for line, message in lint_lottie_ids(request.script_content, self._approved_lottie_ids())
            ]
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

        return ValidationResult(
            dry_run=self._renderer.dry_run(request),
            scene_class_name=request.scene_class_name,
            engine=request.engine,
            warnings=warnings,
        )
