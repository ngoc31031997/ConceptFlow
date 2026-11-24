"""Thư viện hình minh hoạ: kiểu dữ liệu, luật kiểm tra và các hàng dựng sẵn.

Mỗi hình mà Kỹ sư Remotion được đặt vào một shot là một hàng của thư viện, xếp vào đúng một thư
mục để Creator và bộ chọn hình tìm lại thay vì vẽ hai lần. Hình có sẵn (`builtin`) là bộ minh hoạ
của kit: code của chúng nằm trong `remotion_project/src/conceptflow-mini/illustration.tsx` và đi
cùng image, nên hàng chỉ mang thông tin để tìm và mô tả. Hình của thư viện mang TSX riêng, do AI
hoặc tay vẽ.
"""

from __future__ import annotations

import re
from dataclasses import dataclass, replace
from datetime import UTC, datetime
from enum import StrEnum


class IllustrationStatus(StrEnum):
    DRAFT = "draft"  # đã vẽ, chờ Creator duyệt
    APPROVED = "approved"  # được đưa cho Kỹ sư Remotion


class IllustrationKind(StrEnum):
    FIGURE = "figure"  # một hình đặt trong shot (người, quả sung, xe buýt)
    BACKDROP = "backdrop"  # nơi chốn kín khung, vẽ theo các lớp sâu của Scene


class DetailLevel(StrEnum):
    RICH = "rich"  # vẽ như nhân vật chính
    MEDIUM = "medium"  # vẽ như hình phụ
    SILHOUETTE = "silhouette"  # vẽ như phông


#: Hạng của hình trong video; luật số hình con và gradient của bộ kiểm tra theo hạng.
TIER_HERO, TIER_SUPPORT, TIER_BACKGROUND = "hero", "support", "background"

EXEMPLAR_FOLDER_ID = "hinh-mau"
MAX_EXEMPLARS = 5

NAME_RE = re.compile(r"^[A-Z][A-Za-z0-9]{1,40}$")
FOLDER_ID_RE = re.compile(r"^[a-z0-9]+(-[a-z0-9]+)*$")
EXPORT_FUNCTION_RE = re.compile(r"^export function ([A-Z][A-Za-z0-9]*)(\s*\()", re.M)

#: Mức chi tiết của các hình có sẵn không phải "medium"; khớp FIGURE_DETAIL trong illustration.tsx.
BUILTIN_DETAIL = {
    "Person": DetailLevel.RICH,
    "Airplane": DetailLevel.RICH,
    "Cloud": DetailLevel.SILHOUETTE,
    "Mark": DetailLevel.SILHOUETTE,
    "Sparkle": DetailLevel.SILHOUETTE,
}


class IllustrationError(Exception):
    """Gốc của mọi lỗi nghiệp vụ của thư viện hình; thông báo hiển thị thẳng cho Creator."""


class IllustrationNotFoundError(IllustrationError):
    def __init__(self) -> None:
        super().__init__("illustration not found")


class IllustrationReadOnlyError(IllustrationError):
    def __init__(self) -> None:
        super().__init__("hình có sẵn của hệ thống và Hình mẫu chỉ đọc: không sửa, vẽ lại hay xoá được")


class IllustrationNameTakenError(IllustrationError):
    def __init__(self) -> None:
        super().__init__("illustration name is already used")


class FolderNotFoundError(IllustrationError):
    def __init__(self) -> None:
        super().__init__("illustration folder not found")


class FolderTakenError(IllustrationError):
    def __init__(self) -> None:
        super().__init__("illustration folder id is already used")


class FolderNotEmptyError(IllustrationError):
    def __init__(self) -> None:
        super().__init__("illustration folder still holds drawings")


class FolderReadOnlyError(IllustrationError):
    def __init__(self) -> None:
        super().__init__("system folders cannot be deleted")


class ExemplarFolderError(IllustrationError):
    def __init__(self) -> None:
        super().__init__('thư mục Hình mẫu chỉ nhận hình qua nút "Đặt làm mẫu"')


class ExemplarLimitError(IllustrationError):
    def __init__(self) -> None:
        super().__init__(f"thư mục Hình mẫu đã đủ {MAX_EXEMPLARS} hình — bỏ bớt một Hình mẫu trước")


class AlreadyExemplarError(IllustrationError):
    def __init__(self) -> None:
        super().__init__("hình này đã có bản Hình mẫu")


class NotExemplarSourceError(IllustrationError):
    def __init__(self) -> None:
        super().__init__("chỉ hình đã duyệt, không phải hình có sẵn của hệ thống, mới đặt làm mẫu được")


class NotExemplarError(IllustrationError):
    def __init__(self) -> None:
        super().__init__("hình này không phải Hình mẫu")


class BackupInvalidError(IllustrationError):
    """Tệp không phải bản sao lưu thư viện hình mà service này đọc được."""

    def __init__(self, reason: str) -> None:
        super().__init__(f"illustration library backup is not valid: {reason}")


class DrawerDisabledError(IllustrationError):
    def __init__(self) -> None:
        super().__init__("AI vẽ chưa được bật (thiếu kết nối model)")


class RefStyleRequiredError(IllustrationError):
    def __init__(self) -> None:
        super().__init__("chọn phong cách tham chiếu (ref_id) để vẽ hình theo bộ thông số của nó")


class RendererUnavailableError(IllustrationError):
    """Bộ dựng xem trước không chạy được (khác với code hình bị từ chối)."""


class ReferenceStyleNotFoundError(IllustrationError):
    def __init__(self, ref_id: str) -> None:
        super().__init__(f"phong cách tham chiếu {ref_id} không có trong thư viện tham chiếu")


class ReferenceKitMissingError(IllustrationError):
    def __init__(self, ref_id: str) -> None:
        super().__init__(f"phong cách tham chiếu {ref_id} chưa có bộ thông số vẽ (kit)")


class ReferenceServiceError(IllustrationError):
    """visual-service không trả lời được."""


class PromptUnavailableError(IllustrationError):
    """llm-service không có prompt của role đó hoặc không trả lời được."""


class DrawerPromptError(IllustrationError):
    """Prompt của bộ vẽ có biến mà code không biết điền."""


class LLMError(IllustrationError):
    """Một cuộc gọi model qua llm-service thất bại; `kind` cho Creator biết nên làm gì."""

    def __init__(self, kind: str, provider: str, message: str, call_id: str = "") -> None:
        self.kind, self.provider, self.message, self.call_id = kind, provider, message, call_id
        text = f"{provider}: {kind}: {message}"
        if call_id:
            text += f" (mã cuộc gọi {call_id})"
        super().__init__(text)


@dataclass(frozen=True)
class CodeFinding:
    """Một vấn đề trong code của hình: lỗi chặn lưu hoặc cảnh báo style không chặn."""

    message: str
    line: int | None = None

    def to_json(self) -> dict:
        return {"message": self.message, "line": self.line}


class InvalidIllustrationError(IllustrationError):
    """Code hình bị bộ dựng từ chối; các chẩn đoán đi kèm để trình soạn chỉ đúng dòng."""

    def __init__(self, diagnostics: list[CodeFinding], prefix: str = "") -> None:
        self.diagnostics = diagnostics
        msgs = [f"dòng {d.line}: {d.message}" if d.line is not None else d.message for d in diagnostics]
        super().__init__(prefix + "code hình minh hoạ chưa qua kiểm tra: " + "; ".join(msgs))


class DrawErrorKind(StrEnum):
    """Loại lỗi khi kiểm câu trả lời vẽ của AI, để Creator biết nên sửa gì."""

    NO_CODE = "no_code"
    WRONG_NAME = "wrong_name"
    SYNTAX = "syntax"
    RENDER = "render"
    KIT = "kit"


#: Số dòng lấy mỗi bên dòng lỗi khi trích đoạn code.
SNIPPET_RADIUS = 1
_STYLE_RULE_RE = re.compile(r"^\[[A-Z]+\d*\]")
_TS_SYNTAX_RE = re.compile(r"^TS1\d{3}:")


class DrawCheckError(IllustrationError):
    """Code AI vẽ không qua kiểm: một lỗi có cấu trúc (loại, thông báo, dòng, đoạn code quanh dòng).

    `line` là số dòng trong code đã rút; `snippet` là `null` khi không có dòng hay không có code.
    """

    def __init__(self, kind: DrawErrorKind, message: str, line: int | None = None,
                 snippet: str | None = None) -> None:
        self.kind, self.message, self.line, self.snippet = kind, message, line, snippet
        super().__init__(message)

    def to_json(self) -> dict:
        return {"kind": self.kind.value, "message": self.message, "line": self.line, "snippet": self.snippet}


def code_snippet(code: str, line: int | None) -> str | None:
    """Vài dòng quanh `line`, mỗi dòng có số, dòng lỗi đánh dấu `>`; `None` khi không gắn được dòng."""
    lines = code.splitlines()
    if line is None or not 1 <= line <= len(lines):
        return None
    first, last = max(1, line - SNIPPET_RADIUS), min(len(lines), line + SNIPPET_RADIUS)
    return "\n".join(
        f"{n}: {'> ' if n == line else '  '}{lines[n - 1]}" for n in range(first, last + 1))


def draw_check_error(code: str, diagnostics: list[CodeFinding]) -> DrawCheckError:
    """Lỗi có cấu trúc từ chẩn đoán của bộ dựng.

    Bộ dựng dừng ở bước đầu tiên hỏng, nên mọi chẩn đoán của một lần dựng cùng một bước; loại lỗi lấy
    theo dạng thông báo mà bước đó đặt: luật style của kit là `[S1] …`, lỗi tsc là `TS<mã>: …` (mã
    1xxx là lỗi cú pháp, còn lại là lỗi kiểu bộ dựng từ chối), luật tĩnh và lỗi khi dựng thành ảnh là
    lỗi dựng. Dòng của lỗi là dòng của chẩn đoán đầu tiên có số dòng.
    """
    first = diagnostics[0].message if diagnostics else ""
    if _STYLE_RULE_RE.match(first):
        kind, lead = DrawErrorKind.KIT, "Code vi phạm luật vẽ của kit"
    elif _TS_SYNTAX_RE.match(first):
        kind, lead = DrawErrorKind.SYNTAX, "Code lỗi cú pháp"
    else:
        kind, lead = DrawErrorKind.RENDER, "Bộ dựng từ chối code"
    parts = [f"dòng {d.line}: {d.message}" if d.line is not None else d.message for d in diagnostics]
    message = f"{lead} — {'; '.join(parts)}" if parts else lead
    if not message.endswith((".", "!", "?")):
        message += "."
    line = next((d.line for d in diagnostics if d.line is not None), None)
    return DrawCheckError(kind, message, line, code_snippet(code, line))


@dataclass(frozen=True)
class IllustrationFolder:
    id: str
    name: str
    description: str = ""
    position: int = 0
    is_system: bool = False

    def to_json(self) -> dict:
        return {"id": self.id, "name": self.name, "description": self.description,
                "position": self.position, "is_system": self.is_system}


@dataclass(frozen=True)
class Illustration:
    id: str = ""
    kind: IllustrationKind = IllustrationKind.FIGURE
    name: str = ""  # component được export, PascalCase
    title: str = ""
    folder_id: str = ""
    tags: tuple[str, ...] = ()
    description: str = ""  # trông ra sao và dùng khi nào
    usage: str = ""  # cách gọi một dòng: prop và tỉ lệ hộp
    code: str = ""
    #: Phong cách tham chiếu mà hình được vẽ theo bộ thông số của nó ("" = chưa gắn, ví dụ hình có sẵn).
    ref_id: str = ""
    #: Mức chi tiết: hạng mà hình được vẽ lần đầu.
    detail_level: DetailLevel = DetailLevel.MEDIUM
    builtin: bool = False
    #: Hình mẫu AI học theo, xếp ở thư mục Hình mẫu. Bản sao của "Đặt làm mẫu" mang `source_id`
    #: (trống lại khi hình gốc bị xoá); hình mẫu gốc mang `home_folder_id` để "Bỏ làm mẫu" xếp về.
    exemplar: bool = False
    source_id: str = ""
    home_folder_id: str = ""
    warnings: tuple[CodeFinding, ...] = ()
    status: IllustrationStatus = IllustrationStatus.DRAFT
    version: int = 1
    has_preview: bool = False
    created_at: str = ""
    updated_at: str = ""

    @property
    def read_only(self) -> bool:
        """Hình có sẵn và Hình mẫu không sửa, vẽ lại, duyệt lại hay xoá được."""
        return self.builtin or self.exemplar

    @property
    def is_exemplar_copy(self) -> bool:
        return self.exemplar and not self.home_folder_id

    def with_(self, **changes: object) -> Illustration:
        return replace(self, **changes)

    def to_json(self) -> dict:
        out: dict = {
            "id": self.id, "kind": self.kind.value, "name": self.name, "title": self.title,
            "folder_id": self.folder_id, "tags": list(self.tags), "description": self.description,
            "usage": self.usage,
        }
        if self.code:
            out["code"] = self.code
        out.update({
            "ref_id": self.ref_id, "detail_level": self.detail_level.value, "builtin": self.builtin,
            "exemplar": self.exemplar,
        })
        if self.source_id:
            out["source_id"] = self.source_id
        if self.home_folder_id:
            out["home_folder_id"] = self.home_folder_id
        out.update({
            "warnings": [w.to_json() for w in self.warnings], "status": self.status.value,
            "version": self.version, "has_preview": self.has_preview,
            "created_at": self.created_at, "updated_at": self.updated_at,
        })
        return out


@dataclass(frozen=True)
class IllustrationFilter:
    """Thu hẹp danh sách thư viện; trường trống khớp tất cả."""

    folder_id: str = ""
    query: str = ""  # khớp tên, tiêu đề, thẻ và mô tả
    status: str = ""


def kind_or_figure(kind: object) -> IllustrationKind:
    return IllustrationKind.BACKDROP if kind == IllustrationKind.BACKDROP else IllustrationKind.FIGURE


def valid_detail(detail: object) -> bool:
    return detail in (DetailLevel.RICH, DetailLevel.MEDIUM, DetailLevel.SILHOUETTE)


def detail_or_medium(detail: object) -> DetailLevel:
    return DetailLevel(detail) if valid_detail(detail) else DetailLevel.MEDIUM


def tier_or_support(tier: object) -> str:
    return tier if tier in (TIER_HERO, TIER_SUPPORT, TIER_BACKGROUND) else TIER_SUPPORT


def detail_for_tier(tier: str) -> DetailLevel:
    """Mức chi tiết của hình vẽ cho hạng `tier`."""
    if tier == TIER_HERO:
        return DetailLevel.RICH
    if tier == TIER_BACKGROUND:
        return DetailLevel.SILHOUETTE
    return DetailLevel.MEDIUM


def tier_for_detail(detail: object) -> str:
    """Hạng mà hình mức `detail` được kiểm theo: rich là chính, silhouette là phông, còn lại là phụ."""
    if detail == DetailLevel.RICH:
        return TIER_HERO
    if detail == DetailLevel.SILHOUETTE:
        return TIER_BACKGROUND
    return TIER_SUPPORT


def validate_illustration_name(name: str) -> None:
    if not NAME_RE.match(name):
        raise IllustrationError(
            f'tên component "{name}" phải là PascalCase, 2–41 ký tự chữ/số (vd SchoolBus)')


def validate_folder_id(folder_id: str) -> None:
    if not FOLDER_ID_RE.match(folder_id):
        raise IllustrationError(
            f'mã thư mục "{folder_id}" phải là chữ thường không dấu, nối bằng gạch ngang (vd do-choi)')


def normalize_tags(tags: list[str] | tuple[str, ...]) -> tuple[str, ...]:
    """Cắt khoảng trắng, hạ chữ thường và bỏ thẻ trùng, giữ thứ tự."""
    seen: set[str] = set()
    out: list[str] = []
    for tag in tags:
        tag = tag.strip().lower()
        if tag and tag not in seen:
            seen.add(tag)
            out.append(tag)
    return tuple(out)


def exemplar_copy_name(name: str, taken: set[str]) -> str:
    """Tên component của bản Hình mẫu: <name>Mau, rồi <name>Mau2, <name>Mau3... khi tên đã dùng."""
    n = 1
    while True:
        suffix = "Mau" + (str(n) if n > 1 else "")
        base = name[: 41 - len(suffix)]
        if base + suffix not in taken:
            return base + suffix
        n += 1


def rename_export(code: str, usage: str, to: str) -> tuple[str, str]:
    """Đổi tên component được export của code và các thẻ trong dòng cách gọi,
    để một bản sao sống cạnh hình gốc dưới tên riêng."""
    m = EXPORT_FUNCTION_RE.search(code)
    if m is None:
        raise IllustrationError('code không có dòng "export function <Tên>(" để đổi tên')
    old = m.group(1)
    code = code[: m.start(1)] + to + code[m.end(1):]
    usage = re.sub(rf"<{re.escape(old)}\b", lambda _: "<" + to, usage)
    return code, usage


def format_time(moment: datetime) -> str:
    """RFC 3339 theo UTC, chính xác tới giây."""
    return moment.astimezone(UTC).isoformat(timespec="seconds").replace("+00:00", "Z")


def system_folders() -> list[IllustrationFolder]:
    """Các kệ mà thư viện có sẵn; Creator thêm được kệ khác."""
    rows = [
        # Chỉ nhận hình qua "Đặt làm mẫu", không lưu hình vào đây bằng tay.
        (EXEMPLAR_FOLDER_ID, "Hình mẫu", "hình mẫu chuẩn AI vẽ học theo (tối đa 5)"),
        ("con-nguoi", "Con người", "người, nghề nghiệp, nhóm người"),
        ("dong-vat", "Động vật", "thú, chim, cá, côn trùng"),
        ("thuc-vat", "Thực vật", "cây, hoa, lá, chậu cây"),
        ("co-the-suc-khoe", "Cơ thể & sức khoẻ", "bộ phận cơ thể, vi khuẩn, thuốc, dụng cụ y tế"),
        ("do-an-thuc-uong", "Đồ ăn & thức uống", "món ăn, bánh kẹo, đồ uống, trái cây"),
        ("phuong-tien", "Phương tiện", "xe, tàu, máy bay, xe đạp"),
        ("do-vat", "Đồ vật", "đồ gia dụng, nội thất, dụng cụ, thiết bị"),
        ("cong-trinh-noi-chon", "Công trình & nơi chốn", "nhà, trường, bệnh viện, cửa hàng, căn phòng"),
        ("thien-nhien-thoi-tiet", "Thiên nhiên & thời tiết", "mặt trời, mây, mưa, núi, sông, biển"),
        ("tien-kinh-te", "Tiền & kinh tế", "tiền, ví, biểu đồ giá, cửa hàng, thẻ"),
        ("khoa-hoc-cong-nghe", "Khoa học & công nghệ", "máy tính, điện thoại, ống nghiệm, nguyên tử"),
        ("bieu-tuong", "Biểu tượng", "dấu tích/sai, tim, bóng đèn, lấp lánh, bong bóng thoại"),
        ("boi-canh", "Bối cảnh",
         "nền môi trường kín khung (ngoài trời, trong nhà, bên trong cơ thể, dưới nước, vũ trụ), "
         "nền màu, mảng màu"),
    ]
    return [IllustrationFolder(id=i, name=n, description=d, position=pos, is_system=True)
            for pos, (i, n, d) in enumerate(rows, start=1)]


# (tên, tiêu đề, thư mục, thẻ, mô tả, cách gọi)
_FIGURES = [
    ("Backdrop", "Nền màu phẳng", "boi-canh", "nền,sàn,phòng", "Nền màu phẳng phủ cả khung, tuỳ chọn một dải sàn.", "<Backdrop color floor floorY />",),
    ("Panel", "Mảng màu", "boi-canh", "chia đôi,mảng,cửa sổ", "Mảng màu chữ nhật: nửa khung, vạch ngăn, ô nhìn sang nơi khác. x, y là góc trên-trái.", "<Panel x y w h color radius>...</Panel>",),
    ("Person", "Người", "con-nguoi", "người,trẻ em,người già,bác sĩ", "Người nhìn thẳng, 9 dáng, 7 nét mặt, 3 lứa tuổi, 3 kiểu áo, nói được.", "<Person pose mood age outfit hairStyle talking glasses shirt pants skin hair /> — 260×420",),
    ("Tooth", "Chiếc răng", "co-the-suc-khoe", "răng,sâu răng,nha khoa", "Răng hàm có mặt, sâu dần theo decay 0→1.", "<Tooth decay mood shine face /> — 200×220",),
    ("Germ", "Vi khuẩn", "co-the-suc-khoe", "vi khuẩn,vi trùng,virus,bệnh", "Vi khuẩn lúc nhúc có gai, mặt ác/vui/buồn.", "<Germ color mood variant /> — 1:1",),
    ("OpenMouth", "Miệng há", "co-the-suc-khoe", "miệng,răng,hàm", "Miệng há thấy hai hàm răng, chỉ định răng sâu.", "<OpenMouth decayed decay lips /> — 400×260",),
    ("Drop", "Giọt chất lỏng", "thien-nhien-thoi-tiet", "giọt,nước,axit,mồ hôi", "Giọt nước/axit/mồ hôi, tuỳ chọn có mặt.", "<Drop color face /> — 140×180",),
    ("Shield", "Khiên", "bieu-tuong", "bảo vệ,an toàn,miễn dịch", "Khiên bảo vệ có dấu tích.", "<Shield color check /> — 160×190",),
    ("Heart", "Trái tim", "bieu-tuong", "tim,yêu,sức khoẻ", "Trái tim, đập nhịp được.", "<Heart color beat /> — 200×180",),
    ("Candy", "Kẹo", "do-an-thuc-uong", "kẹo,đường,ngọt", "Kẹo gói xoắn hai đầu.", "<Candy color /> — 240×110",),
    ("Lollipop", "Kẹo mút", "do-an-thuc-uong", "kẹo,đường,ngọt", "Kẹo mút xoắn trên que.", "<Lollipop color /> — 140×260",),
    ("Soda", "Ly nước ngọt", "do-an-thuc-uong", "nước ngọt,đồ uống,đường", "Ly nước ngọt có nắp và ống hút.", "<Soda color /> — 160×260",),
    ("Donut", "Bánh donut", "do-an-thuc-uong", "bánh,đường,ngọt", "Bánh donut phủ đường, rắc cốm.", "<Donut color /> — 1:1",),
    ("Apple", "Quả táo", "do-an-thuc-uong", "trái cây,táo,lành mạnh", "Quả táo có lá.", "<Apple color /> — 1:1",),
    ("Clock", "Đồng hồ treo tường", "do-vat", "thời gian,giờ,chờ đợi", "Đồng hồ vuông, kim quay theo hour/minute.", "<Clock hour minute frameColor handColor /> — 1:1",),
    ("Table", "Cái bàn", "do-vat", "bàn,nội thất,phòng", "Bàn nhìn chính diện.", "<Table color /> — 400×200",),
    ("Chair", "Cái ghế", "do-vat", "ghế,nội thất,phòng", "Ghế tựa.", "<Chair color /> — 200×260",),
    ("Window", "Cửa sổ", "cong-trinh-noi-chon", "cửa sổ,ngày,đêm,phòng", "Cửa sổ có trời ngày hoặc đêm.", "<Window time=\"day\"|\"night\" /> — 220×260",),
    ("Plant", "Chậu cây", "thuc-vat", "cây,chậu,phòng", "Chậu cây lá đung đưa.", "<Plant color pot /> — 160×240",),
    ("House", "Ngôi nhà", "cong-trinh-noi-chon", "nhà,gia đình", "Ngôi nhà nhỏ mái đỏ.", "<House color roof /> — 240×220",),
    ("Tree", "Cái cây", "thuc-vat", "cây,rừng,thiên nhiên", "Cây tán tròn.", "<Tree color /> — 180×260",),
    ("Sun", "Mặt trời", "thien-nhien-thoi-tiet", "mặt trời,nắng,ngày", "Mặt trời tia xoay chậm.", "<Sun color /> — 1:1",),
    ("Cloud", "Đám mây", "thien-nhien-thoi-tiet", "mây,trời,thời tiết", "Đám mây bông.", "<Cloud color /> — 260×140",),
    ("Lightbulb", "Bóng đèn", "bieu-tuong", "ý tưởng,hiểu ra,đèn", "Bóng đèn sáng/tắt — ý tưởng, hiểu ra.", "<Lightbulb on color /> — 160×240",),
    ("Coin", "Đồng xu", "tien-kinh-te", "tiền,xu,giá", "Đồng xu vàng.", "<Coin color /> — 1:1",),
    ("Book", "Quyển sách", "do-vat", "sách,học,kiến thức", "Quyển sách đóng.", "<Book color /> — 200×160",),
    ("Phone", "Điện thoại", "khoa-hoc-cong-nghe", "điện thoại,mạng xã hội,màn hình", "Điện thoại, màn hình sáng.", "<Phone color glow /> — 120×220",),
    ("Magnifier", "Kính lúp", "do-vat", "kính lúp,soi,tìm", "Kính lúp — soi kỹ, điều tra.", "<Magnifier color /> — 1:1",),
    ("Mark", "Dấu đúng/sai", "bieu-tuong", "đúng,sai,tích,x", "Huy hiệu tròn dấu tích hoặc dấu X.", "<Mark kind=\"check\"|\"cross\" /> — 1:1",),
    ("Sparkle", "Tia lấp lánh", "bieu-tuong", "lấp lánh,sạch,mới", "Tia bốn cánh nhấp nháy.", "<Sparkle color /> — 1:1",),
    ("Airplane", "Máy bay", "phuong-tien", "máy bay,bay,chiến tranh", "Máy bay cánh quạt nhìn ngang, dựng thành cơ cấu: cánh quạt quay, cánh tà nâng hạ, có kiểu cắt lớp và tách rời.", "<Airplane color view=\"outside\"|\"cutaway\"|\"exploded\" /> — chính — 320×120",),
    ("Bubble", "Bong bóng thoại", "bieu-tuong", "nói,thoại,suy nghĩ", "Bong bóng thoại/suy nghĩ chứa nhãn ngắn.", "<Bubble x y w h thought flip>Nhãn</Bubble>",),
    ("Toothbrush", "Bàn chải đánh răng", "co-the-suc-khoe", "bàn chải,đánh răng,vệ sinh", "Bàn chải có kem, lông quay lên hoặc xuống.", "<Toothbrush color paste bristlesDown /> — 400×100",),
    ("Toothpaste", "Kem đánh răng", "co-the-suc-khoe", "kem đánh răng,vệ sinh", "Tuýp kem đánh răng.", "<Toothpaste color /> — 300×120",),
]

# (tên, tiêu đề, thẻ, mô tả)
_BACKDROPS = [
    ("MeadowBackdrop", "Đồng cỏ ngày nắng", "ngoài trời,đồng cỏ,đồi,cây,trời", "Trời và mặt trời, đồi xa, cây hai bên, cỏ phía trước.",),
    ("RoomBackdrop", "Căn phòng", "trong nhà,phòng,cửa sổ,kệ,nhà", "Tường ấm, cửa sổ nắng, kệ sách bên cạnh, mép bàn phía trước.",),
    ("StreetBackdrop", "Đường phố", "phố,thành phố,nhà,đèn đường", "Dãy nhà có cửa sổ sáng đèn, vỉa hè, cột đèn, bụi cây phía trước.",),
    ("InsideBodyBackdrop", "Bên trong cơ thể", "cơ thể,tế bào,mạch máu,bên trong", "Mô ấm sáng từ trong, tế bào trôi, thành mạch, tế bào lớn sát ống kính.",),
    ("UnderwaterBackdrop", "Dưới nước", "biển,dưới nước,rong,san hô", "Ánh sáng từ mặt nước, đá xa, rong biển đung đưa, bọt nước.",),
    ("SpaceBackdrop", "Vũ trụ", "vũ trụ,sao,hành tinh,không gian", "Trời sao, hành tinh có vành, thiên thạch trôi gần.",),
]


def builtin_illustrations() -> list[Illustration]:
    """34 hình và 6 nền của bộ minh hoạ, xếp vào các thư mục hệ thống. `name` phải khớp một export của
    illustration.tsx hoặc backdrops.tsx."""
    rows: list[Illustration] = []
    for name, title, folder, tags, desc, usage in _FIGURES:
        rows.append(Illustration(
            id="builtin-" + name, name=name, title=title, folder_id=folder,
            tags=normalize_tags(tags.split(",")), description=desc, usage=usage, builtin=True,
            status=IllustrationStatus.APPROVED, version=1,
            detail_level=BUILTIN_DETAIL.get(name, DetailLevel.MEDIUM)))
    for name, title, tags, desc in _BACKDROPS:
        usage = "<Scene backdrop={" + name + "} backdropProps={{color, ground, light}} /> — kín khung"
        rows.append(Illustration(
            id="builtin-" + name, name=name, title=title, folder_id="boi-canh", kind=IllustrationKind.BACKDROP,
            tags=normalize_tags(tags.split(",")), description=desc, usage=usage, builtin=True,
            status=IllustrationStatus.APPROVED, version=1,
            detail_level=BUILTIN_DETAIL.get(name, DetailLevel.MEDIUM)))
    return rows
