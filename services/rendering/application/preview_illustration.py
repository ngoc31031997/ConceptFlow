"""Kiểm tra rồi dựng xem trước một hình của thư viện minh hoạ.

Thứ tự: luật tĩnh (tên, import, trùng tên) → tsc thật → dựng PNG/GIF. Hình
hỏng ở bước nào thì trả lỗi có số dòng của bước đó, để nút "Sửa code" hay
lượt vẽ lại của LLM biết sửa chỗ nào.
"""

from __future__ import annotations

from dataclasses import dataclass, field

from adapters.rendering.illustration_previewer import IllustrationPreviewer
from adapters.rendering.typescript_checker import TypeScriptChecker
from domain.illustration_asset import validate_asset_code
from domain.illustration_style import check_style


@dataclass
class IllustrationPreview:
    ok: bool
    diagnostics: list[dict] = field(default_factory=list)
    png: str = ""
    gif: str = ""
    #: Vi phạm luật style không chặn lưu (luật S.. trong illustration_style_vi.txt).
    warnings: list[dict] = field(default_factory=list)


class PreviewIllustrationUseCase:
    def __init__(self, typescript: TypeScriptChecker, previewer: IllustrationPreviewer) -> None:
        self._ts = typescript
        self._preview = previewer

    def run(
        self, name: str, code: str = "", props: dict | None = None, gif: bool = True,
    ) -> IllustrationPreview:
        if code:
            issues = validate_asset_code(name, code)
            if issues:
                return IllustrationPreview(False, [{"message": i.message, "line": i.line} for i in issues])
            style_errors, style_warnings = check_style(code)
            warnings = [{"message": f.text(), "line": f.line} for f in style_warnings]
            if style_errors:
                found = [{"message": f.text(), "line": f.line} for f in style_errors]
                return IllustrationPreview(False, found, warnings=warnings)
            diags = self._ts.check(code)
            if diags:
                found = [{"message": f"{d.code}: {d.message}", "line": d.line} for d in diags]
                return IllustrationPreview(False, found)
        else:
            warnings = []  # hình có sẵn của bộ minh hoạ: đã duyệt tay
        out = self._preview.preview(name, code, props, gif)
        if not out.ok:
            return IllustrationPreview(False, [{"message": out.error, "line": None}], warnings=warnings)
        return IllustrationPreview(True, png=out.png, gif=out.gif, warnings=warnings)
