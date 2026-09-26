"""Danh mục animation Lottie cho engine Remotion (CR-038).

Creator chọn clip từ các kho miễn phí và bỏ file vào `remotion_project/public/lottie/`;
LLM không sinh, không sửa file Lottie — nó chỉ CHỌN theo id. Module này là nguồn sự
thật duy nhất cho ba việc: kiểm manifest, sinh khối catalog nhúng vào prompt, và lint
id trong script Remotion.

Chỉ clip `approved` mới vào prompt và mới được phép dùng. Clip mới luôn là `candidate`
để Creator xem trước (tools/lottie_catalog.py gallery) rồi mới duyệt.
"""

from __future__ import annotations

import json
import re
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

CANDIDATE = "candidate"
APPROVED = "approved"
STATUSES = frozenset({CANDIDATE, APPROVED})

#: Giấy phép được nhận. Thêm giấy phép mới là một quyết định pháp lý của Creator,
#: không phải của công cụ — nên nó nằm ở đây, trong code review, không trong manifest.
ALLOWED_LICENSES = frozenset({"CC0", "CC-BY", "Lottie Simple License", "MIT"})
#: Giấy phép bắt buộc ghi công.
ATTRIBUTION_LICENSES = frozenset({"CC-BY"})

#: Mỗi file nhúng vào image Docker và được tải lúc render; giữ file gọn.
MAX_ASSET_BYTES = 1_500_000

DATE_RE = re.compile(r"^\d{4}-\d{2}-\d{2}$")
ID_RE = re.compile(r"^[a-z][a-z0-9]*(?:[._-][a-z0-9]+)*$")
#: `<LottieClip id="cat.idle" .../>` — chỉ nhận id là chuỗi literal; id động thì
#: không lint được nên để lỗi lộ lúc render (LottieClip ném lỗi rõ ràng).
CLIP_ID_RE = re.compile(r"<LottieClip\b[^>]*?\bid\s*=\s*(?:\"([^\"]+)\"|'([^']+)'|\{\s*[\"']([^\"']+)[\"']\s*\})")


class CatalogError(ValueError):
    """Manifest hỏng hoặc không đọc được."""


@dataclass(frozen=True)
class LottieAsset:
    id: str
    title: str
    description: str
    license: str
    source_url: str
    author: str = ""
    attribution: str = ""
    #: Ngày Creator tự kiểm giấy phép RIÊNG của clip này (YYYY-MM-DD). Giấy phép của
    #: kho không đủ: LottieFiles ghi rõ mỗi animation có giấy phép riêng và tác giả có
    #: thể thêm hạn chế. Bắt buộc với clip `approved`.
    license_checked: str = ""
    tags: tuple[str, ...] = ()
    status: str = CANDIDATE
    loop: bool = True
    #: Màu gốc (hex) mà `LottieClip.colors` có thể đổi.
    palette: tuple[str, ...] = ()
    #: Ghi chú tuỳ ý cho Creator (không vào prompt).
    notes: str = field(default="", compare=False)

    @property
    def is_approved(self) -> bool:
        return self.status == APPROVED


def load_manifest(path: Path) -> list[LottieAsset]:
    """Đọc manifest; thiếu file nghĩa là danh mục rỗng, không phải lỗi."""
    if not path.exists():
        return []
    try:
        raw = json.loads(path.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        raise CatalogError(f"{path}: JSON không hợp lệ: {exc}") from exc
    entries = raw.get("assets") if isinstance(raw, dict) else None
    if not isinstance(entries, list):
        raise CatalogError(f"{path}: cần khoá 'assets' là một mảng")

    assets: list[LottieAsset] = []
    for index, entry in enumerate(entries):
        if not isinstance(entry, dict):
            raise CatalogError(f"{path}: assets[{index}] phải là object")
        missing = [k for k in ("id", "title", "description", "license", "source_url") if not entry.get(k)]
        if missing:
            raise CatalogError(f"{path}: assets[{index}] ({entry.get('id', '?')}) thiếu {', '.join(missing)}")
        assets.append(
            LottieAsset(
                id=entry["id"],
                title=entry["title"],
                description=entry["description"],
                license=entry["license"],
                source_url=entry["source_url"],
                author=entry.get("author", ""),
                attribution=entry.get("attribution", ""),
                license_checked=entry.get("license_checked", ""),
                tags=tuple(entry.get("tags", ())),
                status=entry.get("status", CANDIDATE),
                loop=bool(entry.get("loop", True)),
                palette=tuple(c.upper() for c in entry.get("palette", ())),
                notes=entry.get("notes", ""),
            )
        )
    return assets


def validate(assets: list[LottieAsset], public_dir: Path) -> list[str]:
    """Trả về danh sách vấn đề (rỗng nghĩa là hợp lệ)."""
    problems: list[str] = []
    seen: set[str] = set()
    for asset in assets:
        tag = f"[{asset.id}]"
        if not ID_RE.match(asset.id):
            problems.append(f"{tag} id phải dạng 'nhom.ten' chữ thường/số")
        if asset.id in seen:
            problems.append(f"{tag} id trùng")
        seen.add(asset.id)
        if asset.status not in STATUSES:
            problems.append(f"{tag} status '{asset.status}' không hợp lệ (candidate|approved)")
        if asset.license not in ALLOWED_LICENSES:
            problems.append(
                f"{tag} giấy phép '{asset.license}' chưa được nhận "
                f"(cho phép: {', '.join(sorted(ALLOWED_LICENSES))})"
            )
        if asset.license in ATTRIBUTION_LICENSES and not asset.attribution:
            problems.append(f"{tag} giấy phép {asset.license} bắt buộc có 'attribution'")

        if asset.is_approved and not DATE_RE.match(asset.license_checked):
            problems.append(
                f"{tag} clip đã duyệt cần 'license_checked' (YYYY-MM-DD): ngày bạn đã đọc "
                "giấy phép của CHÍNH clip này trên trang nguồn"
            )

        file_path = public_dir / f"{asset.id}.json"
        if not file_path.exists():
            problems.append(f"{tag} thiếu file {file_path}")
            continue
        if file_path.stat().st_size > MAX_ASSET_BYTES:
            problems.append(f"{tag} file quá {MAX_ASSET_BYTES // 1000} KB — dùng bản nhẹ hơn")
        try:
            data = json.loads(file_path.read_text(encoding="utf-8"))
        except json.JSONDecodeError:
            problems.append(f"{tag} file không phải JSON hợp lệ (dotLottie .lottie chưa hỗ trợ)")
            continue
        if not all(k in data for k in ("v", "fr", "ip", "op", "layers")):
            problems.append(f"{tag} không giống file Lottie (thiếu v/fr/ip/op/layers)")
    return problems


def approved(assets: list[LottieAsset]) -> list[LottieAsset]:
    return [a for a in assets if a.is_approved]


def lottie_info(data: dict[str, Any]) -> dict[str, Any]:
    """Thông số Creator cần khi viết manifest: kích thước, thời lượng, màu gốc."""
    fps = float(data.get("fr", 0)) or 1.0
    frames = float(data.get("op", 0)) - float(data.get("ip", 0))
    return {
        "width": data.get("w"),
        "height": data.get("h"),
        "fps": fps,
        "frames": frames,
        "seconds": round(frames / fps, 2),
        "palette": sorted(extract_palette(data)),
    }


def extract_palette(data: Any) -> set[str]:
    """Mọi màu fill/stroke tĩnh hoặc keyframe trong file, dạng '#RRGGBB'."""
    colors: set[str] = set()

    def add(components: Any) -> None:
        if isinstance(components, list) and len(components) >= 3 and all(
            isinstance(v, (int, float)) for v in components[:3]
        ):
            r, g, b = (max(0, min(255, round(v * 255))) for v in components[:3])
            colors.add(f"#{r:02X}{g:02X}{b:02X}")

    def walk(node: Any) -> None:
        if isinstance(node, dict):
            if node.get("ty") in ("fl", "st") and isinstance(node.get("c"), dict):
                k = node["c"].get("k")
                if isinstance(k, list) and k and isinstance(k[0], dict):
                    for keyframe in k:
                        add(keyframe.get("s"))
                else:
                    add(k)
            for value in node.values():
                walk(value)
        elif isinstance(node, list):
            for value in node:
                walk(value)

    walk(data)
    return colors


def render_prompt_block(assets: list[LottieAsset], language: str = "vi") -> str:
    """Khối `{{lottie_catalog}}` cho prompt. Rỗng vẫn phải nói rõ, để LLM không bịa id."""
    ready = approved(assets)
    if language != "vi":
        raise CatalogError(f"chưa có bản catalog cho ngôn ngữ '{language}'")
    if not ready:
        return (
            "Chưa có clip Lottie nào được duyệt. Không dùng ¤LottieClip¤; "
            "mọi hình vẽ bằng JSX/SVG/CSS như thường lệ."
        )
    lines = [
        "Đây là các clip hoạt hình DỰNG SẴN (Lottie) mà kênh có. Dùng khi một clip khớp "
        "với shot — đặc biệt nhân vật, biểu cảm, vật thể có chi tiết mà SVG tự vẽ khó đẹp. "
        "Không bắt buộc: nếu không clip nào hợp, vẽ bằng SVG như bình thường. "
        "TUYỆT ĐỐI không bịa id ngoài danh sách này.",
        "",
    ]
    for asset in sorted(ready, key=lambda a: a.id):
        line = f"- ¤{asset.id}¤ — {asset.description}"
        details = []
        if asset.tags:
            details.append("tag: " + ", ".join(asset.tags))
        details.append("lặp mặc định" if asset.loop else "chạy một lần")
        if asset.palette:
            details.append("màu đổi được: " + " ".join(asset.palette))
        lines.append(f"{line} ({'; '.join(details)})")
    return "\n".join(lines)


def credits(assets: list[LottieAsset]) -> list[str]:
    """Dòng ghi công cho các clip đã duyệt cần ghi công (dán vào mô tả video)."""
    return [
        f"{a.title} — {a.attribution} ({a.license}) {a.source_url}"
        for a in sorted(approved(assets), key=lambda a: a.id)
        if a.license in ATTRIBUTION_LICENSES
    ]


def lint_lottie_ids(script: str, approved_ids: set[str]) -> list[tuple[int, str]]:
    """Id không có trong danh mục đã duyệt, dạng (số dòng, thông điệp) — rẻ hơn nhiều so
    với chết lúc render. Trả tuple thay vì `LintIssue` để module này không kéo theo
    `conceptflow` (và manim); use case bọc lại thành LintIssue."""
    issues: list[tuple[int, str]] = []
    for match in CLIP_ID_RE.finditer(script):
        clip_id = next(g for g in match.groups() if g)
        if clip_id not in approved_ids:
            line = script.count("\n", 0, match.start()) + 1
            known = f" (có: {', '.join(sorted(approved_ids))})" if approved_ids else " (danh mục đang rỗng)"
            issues.append((line, f"LottieClip id '{clip_id}' không có trong danh mục đã duyệt{known}"))
    return issues
