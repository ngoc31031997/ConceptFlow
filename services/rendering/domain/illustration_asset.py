"""Luật cho một hình trong thư viện minh hoạ.

Một hình là MỘT file TSX xuất đúng một component tên PascalCase, chỉ import từ
react, remotion, bộ minh hoạ (để dùng lại Figure/Face/useBlink...) và bộ cảnh (useLayerBox
cho một nền). Hình này
về sau được Code Merger dán thẳng vào script của video, nên nó không được mang
theo import nào khác và không được khai báo gì trùng tên với khung script.
"""

from __future__ import annotations

import re
from dataclasses import dataclass

NAME_RE = re.compile(r"^[A-Z][A-Za-z0-9]{1,40}$")
ALLOWED_MODULES = {"react", "remotion", "./conceptflow-mini/illustration", "./conceptflow-mini/scene"}
IMPORT_RE = re.compile(r"^\s*import\b[^;]*?\bfrom\s*['\"]([^'\"]+)['\"]", re.M | re.S)
BARE_IMPORT_RE = re.compile(r"^\s*import\s*['\"]([^'\"]+)['\"]", re.M)
TOP_DECL_RE = re.compile(
    r"^(?:export\s+)?(?:function|const|let|var|class|type|interface)\s+([A-Za-z_]\w*)", re.M,
)

#: Tên đã có trong khung script (Code Merger) — hình không được khai báo lại.
RESERVED = {
    "React", "PALETTE", "LAYOUT", "SHOTS", "clamp", "narrations", "ShotProps", "CreatorComposition",
    "Stage", "Segments", "SAFE_MARGIN", "WIDTH", "HEIGHT", "LottieClip",
}


@dataclass(frozen=True)
class AssetIssue:
    message: str
    line: int | None = None


def _line_of(code: str, index: int) -> int:
    return code.count("\n", 0, index) + 1


def validate_asset_code(name: str, code: str) -> list[AssetIssue]:
    issues: list[AssetIssue] = []
    if not NAME_RE.match(name or ""):
        issues.append(AssetIssue(
            f"tên component '{name}' phải là PascalCase, 2–41 ký tự chữ/số (vd SchoolBus)"))
    if not code.strip():
        return issues + [AssetIssue("code trống")]
    for m in list(IMPORT_RE.finditer(code)) + list(BARE_IMPORT_RE.finditer(code)):
        if m.group(1) not in ALLOWED_MODULES:
            issues.append(AssetIssue(
                f"import '{m.group(1)}' không được phép — "
                "chỉ react, remotion, ./conceptflow-mini/illustration, ./conceptflow-mini/scene",
                _line_of(code, m.start())))
    if not re.search(rf"^export\s+function\s+{re.escape(name)}\s*\(", code, re.M):
        issues.append(AssetIssue(f"thiếu 'export function {name}(' ở đầu dòng"))
    for m in TOP_DECL_RE.finditer(code):
        declared = m.group(1)
        if declared in RESERVED or re.fullmatch(r"Shot\d+_\d+", declared):
            issues.append(AssetIssue(
                f"'{declared}' trùng tên có sẵn trong khung script", _line_of(code, m.start())))
    for bad in ("fetch(", "staticFile(", "Math.random(", "Date.now(", "<Img", "eval(", "new Function"):
        idx = code.find(bad)
        if idx >= 0:
            issues.append(AssetIssue(
                f"không dùng '{bad.rstrip('(')}' trong hình minh hoạ", _line_of(code, idx)))
    return issues
