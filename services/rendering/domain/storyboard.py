"""Khung hình và bản chốt của bước Storyboard: từ vựng đóng, yêu cầu dựng khung, lỗi nghiệp vụ.

Một khung ghép nhiều tài sản đã duyệt lên lưới vị trí rồi vẽ chữ ở dải dưới. Kích thước từng hình do bộ
dựng quyết (nó đo hộp thật của hình, xem `host.tsx`), người gọi chỉ nói đặt hình nào ở đâu. Hai từ vựng
đóng `POSITIONS` và `CAMERA_VIEWS` giống hệt bảng trong host. Xem ADR-0051.
"""

from __future__ import annotations

import re
from dataclasses import dataclass, field

from domain.illustration import IllustrationError
from domain.kit import InvalidKitError, validate_kit

POSITIONS = (
    "upper_left", "upper_center", "upper_right",
    "center_left", "center", "center_right",
    "lower_left", "lower_center", "lower_right",
    "split_left", "split_right", "full",
)
CAMERA_VIEWS = ("wide", "medium", "close", "comparison")
ROLES = ("subject", "main")
FRAME_WIDTH = 1280
FRAME_HEIGHT = 720
MAX_TEXT_LINES = 6
MAX_TEXT_PX = 200
SHOT_ID_RE = re.compile(r"^SH-\d{3,}$")
ID_RE = re.compile(r"^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$")
SPEC_HASH_RE = re.compile(r"^[A-Za-z0-9]{8,128}$")
COLOR_RE = re.compile(r"^#(?:[0-9A-Fa-f]{3}|[0-9A-Fa-f]{6})$")


class StoryboardError(IllustrationError):
    """Gốc của lỗi nghiệp vụ của khung và bản chốt; thông báo hiển thị thẳng cho Creator."""


class InvalidFrameRequestError(StoryboardError):
    """Yêu cầu dựng khung sai hình dạng hoặc ngoài từ vựng đóng (400)."""


class InvalidDocumentError(StoryboardError):
    """Tài liệu bản chốt sai hình dạng (400)."""


class AssetsNotReadyError(StoryboardError):
    """Tài sản của khung không có hoặc chưa duyệt (409); `problems` liệt kê từng mã và lý do."""

    def __init__(self, problems: list[dict[str, str]]) -> None:
        self.problems = problems
        listed = ", ".join(f"{p['asset_id']} ({p['reason']})" for p in problems)
        super().__init__(f"tài sản chưa dùng được cho khung: {listed}")


class FrameRenderError(StoryboardError):
    """Bộ dựng khung không chạy được hoặc dựng hỏng (503); không có ảnh giả."""


class FrameNotFoundError(StoryboardError):
    """Shot chưa có khung đã vẽ (404)."""

    def __init__(self) -> None:
        super().__init__("frame not found")


class VersionNotFoundError(StoryboardError):
    """Chưa có bản chốt hoặc không có số phiên bản đó (404)."""

    def __init__(self) -> None:
        super().__init__("storyboard version not found")


class MissingFramesError(StoryboardError):
    """Bản chốt tham chiếu khung chưa vẽ (409); `shot_ids` liệt kê các shot thiếu ảnh."""

    def __init__(self, shot_ids: list[str]) -> None:
        self.shot_ids = shot_ids
        super().__init__(f"thiếu ảnh khung của: {', '.join(shot_ids)}")


class VersionConflictError(StoryboardError):
    """Số phiên bản đã có người ghi (409)."""

    def __init__(self, version: int) -> None:
        super().__init__(f"storyboard version {version} already exists")


@dataclass(frozen=True)
class Placement:
    asset_id: str
    role: str
    position: str


@dataclass(frozen=True)
class TextLine:
    text: str
    min_px: int


@dataclass(frozen=True)
class FrameRequest:
    """Yêu cầu dựng khung của một shot, đã kiểm hình dạng và từ vựng đóng."""

    project_id: str
    shot_id: str
    unit_id: str
    beat_id: str
    spec_hash: str
    kit: dict
    placements: tuple[Placement, ...]
    camera_view: str
    background_color: str
    background_asset_id: str = ""
    text: tuple[TextLine, ...] = field(default_factory=tuple)
    text_color: str = "#FFFFFF"

    @property
    def asset_ids(self) -> list[str]:
        """Mọi mã tài sản của khung (nền rồi các hình), không lặp, giữ thứ tự."""
        ids = ([self.background_asset_id] if self.background_asset_id else []) + [
            p.asset_id for p in self.placements]
        return list(dict.fromkeys(ids))


def _require(condition: bool, message: str) -> None:
    if not condition:
        raise InvalidFrameRequestError(message)


def build_frame_request(
    project_id: str, shot_id: str, unit_id: str, beat_id: str, spec_hash: str, kit: object,
    placements: list[dict], camera_view: str, background: dict, text: list[dict], text_color: str,
    size: dict | None = None,
) -> FrameRequest:
    """Kiểm yêu cầu rồi dựng `FrameRequest`.

    Sai hình dạng, kit hay từ vựng đóng thì ném InvalidFrameRequestError.
    """
    _require(bool(SHOT_ID_RE.match(shot_id)), f"mã shot không hợp lệ: {shot_id!r}")
    _require(bool(ID_RE.match(unit_id)), f"mã unit không hợp lệ: {unit_id!r}")
    _require(bool(ID_RE.match(beat_id)), f"mã beat không hợp lệ: {beat_id!r}")
    _require(bool(SPEC_HASH_RE.match(spec_hash)), "spec_hash phải là chuỗi chữ-số 8-128 ký tự")
    _require(
        size is None or (size.get("w") == FRAME_WIDTH and size.get("h") == FRAME_HEIGHT),
        f"khung chỉ dựng cỡ {FRAME_WIDTH}x{FRAME_HEIGHT}")
    _require(camera_view in CAMERA_VIEWS, f"camera_view {camera_view!r} ngoài từ vựng đóng {CAMERA_VIEWS}")
    _require(len(placements) >= 1, "khung cần ít nhất một placement")
    parsed = []
    for item in placements:
        _require(item.get("position") in POSITIONS, f"position {item.get('position')!r} ngoài từ vựng đóng")
        _require(item.get("role") in ROLES, f"role {item.get('role')!r} ngoài từ vựng đóng {ROLES}")
        _require(bool(item.get("asset_id")), "placement thiếu asset_id")
        parsed.append(Placement(item["asset_id"], item["role"], item["position"]))
    _require(len({p.position for p in parsed}) == len(parsed), "hai placement không được chung một position")
    color = background.get("color") or ""
    _require(bool(COLOR_RE.match(color)), f"màu nền không hợp lệ: {color!r}")
    _require(bool(COLOR_RE.match(text_color)), f"màu chữ không hợp lệ: {text_color!r}")
    _require(len(text) <= MAX_TEXT_LINES, f"tối đa {MAX_TEXT_LINES} dòng chữ")
    lines = []
    for line in text:
        px = line.get("min_px")
        _require(bool(line.get("text")), "dòng chữ rỗng")
        _require(isinstance(px, int) and not isinstance(px, bool) and 1 <= px <= MAX_TEXT_PX,
                 f"min_px phải là số nguyên 1..{MAX_TEXT_PX}")
        lines.append(TextLine(line["text"], px))
    try:
        checked_kit = validate_kit(kit)
    except InvalidKitError as exc:
        raise InvalidFrameRequestError(str(exc)) from exc
    return FrameRequest(
        project_id=project_id, shot_id=shot_id, unit_id=unit_id, beat_id=beat_id, spec_hash=spec_hash,
        kit=checked_kit, placements=tuple(parsed), camera_view=camera_view, background_color=color,
        background_asset_id=background.get("asset_id") or "", text=tuple(lines), text_color=text_color)


@dataclass(frozen=True)
class ShotFrame:
    """Bản ghi khung đã vẽ gần nhất của một shot (`frame_path` tương đối so với `/shared/{project_id}`)."""

    project_id: str
    shot_id: str
    unit_id: str
    beat_id: str
    frame_path: str | None
    frame_hash: str
    spec_hash: str
    drawn_at: str
    error: str = ""

    def to_json(self) -> dict:
        return {
            "project_id": self.project_id, "shot_id": self.shot_id, "unit_id": self.unit_id,
            "beat_id": self.beat_id, "frame_path": self.frame_path, "frame_hash": self.frame_hash,
            "spec_hash": self.spec_hash, "drawn_at": self.drawn_at, "error": self.error,
        }


@dataclass(frozen=True)
class StoryboardVersion:
    """Bản chốt bất biến: tài liệu `storyboard.json` kèm số phiên bản bước 10 mà nó dựa vào."""

    project_id: str
    version: int
    document: dict
    sequence_version: int
    note: str
    locked_at: str

    def to_json(self) -> dict:
        return {
            "project_id": self.project_id, "version": self.version, "sequence_version": self.sequence_version,
            "note": self.note, "locked_at": self.locked_at, "document": self.document,
        }


def frame_relative_path(shot_id: str) -> str:
    """Đường dẫn khung đang làm, tương đối so với `/shared/{project_id}`."""
    return f"storyboard/{shot_id}.png"


def version_frame_path(version: int, shot_id: str) -> str:
    """Đường dẫn khung của một bản chốt, tương đối so với `/shared/{project_id}`."""
    return f"storyboard/v{version}/{shot_id}.png"


def document_shot_ids(document: object) -> list[str]:
    """Mã shot của tài liệu chốt theo thứ tự; tài liệu hoặc shot sai hình dạng ném InvalidDocumentError."""
    if not isinstance(document, dict):
        raise InvalidDocumentError("document phải là đối tượng JSON")
    shots = document.get("shots")
    if not isinstance(shots, list) or not shots:
        raise InvalidDocumentError("document.shots phải là danh sách có ít nhất một shot")
    ids = []
    for shot in shots:
        shot_id = shot.get("shot_id") if isinstance(shot, dict) else None
        if not isinstance(shot_id, str) or not SHOT_ID_RE.match(shot_id):
            raise InvalidDocumentError(f"shot_id không hợp lệ trong document: {shot_id!r}")
        ids.append(shot_id)
    if len(set(ids)) != len(ids):
        raise InvalidDocumentError("document có shot_id lặp")
    return ids
