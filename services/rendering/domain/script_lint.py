"""Lint script video trước khi tốn một lượt `manim` (CR-017 FR46).

## Vì sao đổi từ blacklist sang whitelist

Bản trước giữ một `dict` chép tay các kwarg sai của 5 class Manim, và chính
docstring của nó thừa nhận danh sách ấy "can never be exhaustive". Manim có hàng
trăm class; một danh sách chép tay luôn chạy sau lỗi mới, và mỗi lỗi lọt lưới
tốn một lượt render mới phát hiện.

Sau CR-017, script chỉ được phép gọi bề mặt API hẹp của `conceptflow`. Luật vì
thế đảo chiều và thu về đúng một câu: **cái gì không có trong `conceptflow` thì
không hợp lệ**. Nó phủ mọi API sai, kể cả những cái chưa ai gặp bao giờ.

## Vì sao module này không import conceptflow

Chỉ import `conceptflow.api` — module dữ liệu thuần. Import cả package sẽ kéo
theo manim, mà test suite của Rendering cố tình không cần manim (README). Danh
sách tên được khoá trùng với `conceptflow.__all__` bằng một assert ngay lúc
import package, nên hai nguồn không trôi khỏi nhau.

## Hai mức nghiêm trọng

`BLOCKING` chặn render; `WARNING` chỉ báo. Đường thoát hiểm (FR45.5) là warning
chứ không phải lỗi — một thư viện chặn được cả những video tham vọng nhất thì nó
đang làm hại chứ không giúp.
"""

from __future__ import annotations

import ast
import builtins
import re
from dataclasses import dataclass

from conceptflow.api import COMPONENT_NAMES, PUBLIC_NAMES, SCENE_METHODS

BLOCKING = "blocking"
WARNING = "warning"

#: Màu viết thẳng bằng mã hex — cách phổ biến nhất để bảng màu trôi khỏi theme.
HEX_COLOR_RE = re.compile(r"^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$")

#: Cỡ chữ hợp lệ là bốn bậc trong `Theme.scale`. Giữ dưới dạng số ở đây thay vì
#: import Theme để module không phụ thuộc runtime của conceptflow.
ALLOWED_FONT_SIZES = frozenset({48, 36, 28, 20})

_BUILTINS = frozenset(dir(builtins))


@dataclass(frozen=True)
class LintIssue:
    line: int
    message: str
    severity: str = BLOCKING

    @property
    def is_blocking(self) -> bool:
        return self.severity == BLOCKING

    def __str__(self) -> str:
        prefix = "lỗi" if self.is_blocking else "cảnh báo"
        return f"{prefix} dòng {self.line}: {self.message}"


def lint_manim_script(script_content: str) -> list[LintIssue]:
    """Các vấn đề tìm thấy trong script, cả mức chặn lẫn mức cảnh báo."""
    try:
        tree = ast.parse(script_content)
    except SyntaxError as exc:
        return [
            LintIssue(
                line=exc.lineno or 0,
                message=f"script không phải Python hợp lệ: {exc.msg}",
                severity=BLOCKING,
            )
        ]

    collector = _Collector()
    collector.visit(tree)
    collector.finalize()
    return sorted(collector.issues, key=lambda i: (i.line, i.severity))


def blocking_issues(issues: list[LintIssue]) -> list[LintIssue]:
    return [issue for issue in issues if issue.is_blocking]


class _Collector(ast.NodeVisitor):
    def __init__(self) -> None:
        self.issues: list[LintIssue] = []
        #: Tên do chính script định nghĩa hoặc import — không phải API lạ.
        self.defined: set[str] = set()
        self._calls: list[ast.Call] = []

    # --- thu thập tên script tự định nghĩa -----------------------------------

    def visit_Import(self, node: ast.Import) -> None:
        for alias in node.names:
            self.defined.add(alias.asname or alias.name.split(".")[0])
            if alias.name == "manim" or alias.name.startswith("manim."):
                self._warn_escape_hatch(node.lineno, alias.name)
        self.generic_visit(node)

    def visit_ImportFrom(self, node: ast.ImportFrom) -> None:
        module = node.module or ""
        for alias in node.names:
            self.defined.add(alias.asname or alias.name)

        if module == "conceptflow" or module.startswith("conceptflow."):
            if any(alias.name == "*" for alias in node.names):
                self.defined |= set(PUBLIC_NAMES)
        elif module == "manim" or module.startswith("manim."):
            if any(alias.name == "*" for alias in node.names):
                # Star-import từ manim che khuất chính các tên của conceptflow,
                # nên whitelist mất hiệu lực hoàn toàn — đây là lỗi, không phải
                # cảnh báo như các đường thoát hiểm khác.
                self.issues.append(
                    LintIssue(
                        line=node.lineno,
                        message=(
                            "`from manim import *` che khuất API của conceptflow. "
                            "Dùng `from conceptflow import *`, và nếu thật sự cần "
                            "một API thô thì import đích danh nó."
                        ),
                        severity=BLOCKING,
                    )
                )
            else:
                self._warn_escape_hatch(
                    node.lineno, ", ".join(alias.name for alias in node.names)
                )
        self.generic_visit(node)

    def visit_FunctionDef(self, node: ast.FunctionDef) -> None:
        self.defined.add(node.name)
        self.generic_visit(node)

    def visit_AsyncFunctionDef(self, node: ast.AsyncFunctionDef) -> None:
        self.defined.add(node.name)
        self.generic_visit(node)

    def visit_ClassDef(self, node: ast.ClassDef) -> None:
        self.defined.add(node.name)
        self.generic_visit(node)

    def visit_Assign(self, node: ast.Assign) -> None:
        for target in node.targets:
            for name in _names_in_target(target):
                self.defined.add(name)
        self.generic_visit(node)

    # --- kiểm tra lời gọi -----------------------------------------------------

    def visit_Call(self, node: ast.Call) -> None:
        self._calls.append(node)
        self._check_kwargs(node)
        self.generic_visit(node)

    def visit_Constant(self, node: ast.Constant) -> None:
        if isinstance(node.value, str) and HEX_COLOR_RE.match(node.value):
            self.issues.append(
                LintIssue(
                    line=node.lineno,
                    message=(
                        f"màu {node.value!r} viết thẳng bằng mã hex. Dùng màu của "
                        "theme (`self.theme.accent`, `self.theme.series_color(i)`…) "
                        "để bảng màu của kênh không trôi theo từng video."
                    ),
                    severity=WARNING,
                )
            )
        self.generic_visit(node)

    def _check_kwargs(self, node: ast.Call) -> None:
        for kw in node.keywords:
            if kw.arg != "font_size" or not isinstance(kw.value, ast.Constant):
                continue
            value = kw.value.value
            if isinstance(value, (int, float)) and int(value) not in ALLOWED_FONT_SIZES:
                self.issues.append(
                    LintIssue(
                        line=node.lineno,
                        message=(
                            f"font_size={value} nằm ngoài thang cỡ chữ của theme "
                            f"({sorted(ALLOWED_FONT_SIZES, reverse=True)}). Dùng "
                            "`self.title/heading/body/caption` thay vì đặt cỡ tay."
                        ),
                        severity=WARNING,
                    )
                )

    def _warn_escape_hatch(self, line: int, what: str) -> None:
        self.issues.append(
            LintIssue(
                line=line,
                message=(
                    f"dùng API thô của Manim ({what}). Được phép, nhưng phần này "
                    "nằm ngoài design system nên không được theme và QC bảo vệ."
                ),
                severity=WARNING,
            )
        )

    # --- chạy sau khi đã duyệt hết cây ----------------------------------------

    def finalize(self) -> None:
        for node in self._calls:
            name = _called_name(node.func)
            if name is None:
                continue
            if _is_attribute_call(node.func):
                self._check_scene_method(node, name)
                continue
            if name in self.defined or name in PUBLIC_NAMES or name in _BUILTINS:
                continue
            self.issues.append(
                LintIssue(
                    line=node.lineno,
                    message=(
                        f"`{name}` không thuộc API của conceptflow. Component có "
                        f"sẵn: {', '.join(sorted(COMPONENT_NAMES))}. Nếu thật sự "
                        "cần API thô của Manim thì import đích danh nó."
                    ),
                    severity=BLOCKING,
                )
            )

    def _check_scene_method(self, node: ast.Call, name: str) -> None:
        """Chỉ soi `self.<gì đó>()`.

        Method trên biến khác (`group.arrange(...)`, `text.next_to(...)`) không
        kiểm được bằng phân tích tĩnh vì không biết kiểu của biến — để runtime lo.
        """
        func = node.func
        if not isinstance(func, ast.Attribute):
            return
        if not (isinstance(func.value, ast.Name) and func.value.id == "self"):
            return
        if name in SCENE_METHODS or name in _MANIM_SCENE_METHODS:
            return
        # Không báo lỗi: script hoàn toàn có thể tự định nghĩa method helper trên
        # class Scene của mình, và phân tích tĩnh ở đây không thấy được điều đó.


#: Method của `Scene` mà script vẫn cần gọi trực tiếp.
_MANIM_SCENE_METHODS = frozenset({
    "play", "wait", "add", "remove", "construct", "bring_to_front", "bring_to_back",
    "narrate", "beat", "chapter",  # CR-018/CR-019 — thêm ở bước sau
})


def _names_in_target(target: ast.expr) -> list[str]:
    if isinstance(target, ast.Name):
        return [target.id]
    if isinstance(target, (ast.Tuple, ast.List)):
        names: list[str] = []
        for element in target.elts:
            names.extend(_names_in_target(element))
        return names
    return []


def _called_name(func: ast.expr) -> str | None:
    if isinstance(func, ast.Name):
        return func.id
    if isinstance(func, ast.Attribute):
        return func.attr
    return None


def _is_attribute_call(func: ast.expr) -> bool:
    return isinstance(func, ast.Attribute)
