"""Luật bố cục của một script Remotion, áp lên số đo thật.

Số đo đến từ layout probe (`remotion_project/layout_check.mjs`): hộp bao DOM
của mọi chữ, `<svg>`, hình của bộ minh hoạ, clip Lottie và khối màu mà từng
shot vẽ ra, tại một loạt thời điểm của shot. Module này chỉ đọc JSON đó — không
trình duyệt, không Node — nên test được bằng JSON mẫu.

Luật lấy từ mục F của prompt Remotion Engineer (L1, L4, L5, L14):

  chặn (vào vòng sửa lỗi)                    | chỉ cảnh báo
  -------------------------------------------+-------------------------------
  vùng an toàn của khung (domain/frame.py)   | vật lớn nhất của shot < 40%
  vùng phụ đề in lên hình                     |   chiều khung (L14)
  hai khối chữ đè nhau                        | còn tài nguyên chưa tải xong
  chữ tràn khung (scrollWidth > clientWidth+1)|   (số đo có thể thiếu vật)
  chữ nhỏ hơn 32px                            | lấn vùng an toàn ≤ 1% cạnh ngắn
  shot ném lỗi khi chạy ở một frame           |   của khung
                                              | vật phụ bị máy quay (zoom, lia)
                                              |   đẩy ra ngoài vùng an toàn

Khung (ngang hay dọc) lấy từ `composition` của chính lần đo. Lớp nền và ánh
sáng của `Scene`, màu của `Backdrop` và `Panel` (`data-cf-layer`) không có
trong số đo: harness bỏ qua chúng. Khối không phải chữ cắt ngang trọn vùng an
toàn theo một chiều (dải sáng rộng hơn khung, vạch chia cao bằng khung) là
trang trí của khung: không xét vùng an toàn, vùng phụ đề, cũng không được
chọn làm vật trọng tâm.

Vài điều không hiển nhiên, đều vì hoạt hình:

* **Vật trượt vào/ra khỏi khung** là ngoại lệ duy nhất của L1. Code không đọc
  được kịch bản, nên vật nào có lúc nằm (một phần) ngoài khung hình ở một mẫu
  bất kỳ được coi là vật trượt: với nó, vùng an toàn/vùng phụ đề chỉ xét ở các
  mẫu đã yên (≥ 85% shot). Vật đã nằm trọn trong khung rồi kết thúc ở ngoài
  khung là vật trượt ra: các mẫu đã yên của nó là lúc nó đang trượt ra, nên
  không xét. Vật không bao giờ ra khỏi khung thì xét ở mọi mẫu — nhờ vậy
  spring vọt lố qua mép vùng an toàn vẫn bị bắt.
* **Máy quay** zoom hay lia thì cắt bớt vật ở mép khung: đó là cú máy, nên vật
  phụ bị đẩy ra ngoài vùng an toàn chỉ là cảnh báo. Vật trọng tâm (vật lớn
  nhất của shot) vẫn phải nằm trong vùng an toàn (L8).
* **Cỡ chữ** xét ở các mẫu đã yên: bong bóng thoại đang phóng lên có chữ 25px
  ở giữa chuyển động là bình thường (T6a, mục 8.5). Cỡ là cỡ HIỂN THỊ (sau
  scale), vì đó là cái người xem thấy.
* **Chữ đè chữ** chỉ tính khi cả hai khối đang hiện rõ (độ mờ ≥ 0.6), để hai
  nhãn hoà vào nhau (một mờ đi, một hiện lên) không bị coi là đè.
"""

from __future__ import annotations

from dataclasses import dataclass, field

from domain.frame import MAX_HEIGHT, SafeArea, frame_for

# Nửa pixel: hộp bao được làm tròn 0.1px, và 96.0 chạm mép thì vẫn là trong.
EDGE_TOLERANCE = 0.5
MIN_FONT_PX = 32
MIN_HERO_FRACTION = 0.40
SETTLED_PCT = 0.85
OVERLAP_MIN_OPACITY = 0.6
# Chữ lấn một pixel ra ngoài hộp của chính nó chưa phải là tràn.
OVERFLOW_TOLERANCE_PX = 1
# Lấn vùng an toàn tới chừng này (phần của cạnh ngắn khung) mắt khó thấy: cảnh báo.
SOFT_EXCESS_FRACTION = 0.01

SAFE_AREA = "safe_area"
SUBTITLE_ZONE = "subtitle_zone"
TEXT_OVERLAP = "text_overlap"
TEXT_OVERFLOW = "text_overflow"
MIN_FONT = "min_font"
RUNTIME_ERROR = "runtime_error"
HERO_SIZE = "hero_size"
NOT_SETTLED = "not_settled"

BLOCKING_RULES = frozenset({SAFE_AREA, SUBTITLE_ZONE, TEXT_OVERLAP, TEXT_OVERFLOW, MIN_FONT, RUNTIME_ERROR})


@dataclass(frozen=True)
class SubtitleBand:
    """Dải phụ đề in lên hình, đo từ mép khung (cùng số với `subtitleBandPx`
    của authoring-service, domain/narration.go)."""

    edge: str  # "top" | "bottom"
    px: int

    def __post_init__(self) -> None:
        if self.edge not in ("top", "bottom"):
            raise ValueError(f"subtitle band edge must be top or bottom, got {self.edge!r}")
        if not 0 < self.px < MAX_HEIGHT:
            raise ValueError(f"subtitle band height must be between 0 and {MAX_HEIGHT} px, got {self.px}")


@dataclass
class LayoutFinding:
    rule: str
    shot: str  # "1.2", hoặc "#3" khi script không đặt tên shot theo ShotN_M
    line: int | None  # dòng gán cho chẩn đoán (luôn nằm trong phần của shot nếu biết)
    element_line: int | None
    detail: str
    frames: list[str] = field(default_factory=list)
    total_samples: int = 0
    score: float = 0.0  # độ nặng, để giữ lại số đo tệ nhất khi gộp các mẫu
    # Vì sao một vi phạm của luật chặn chỉ là cảnh báo; "" = chặn như luật.
    soft_reason: str = ""

    @property
    def blocking(self) -> bool:
        return self.rule in BLOCKING_RULES and not self.soft_reason

    @property
    def message(self) -> str:
        detail = f"{self.detail} — {self.soft_reason}" if self.soft_reason else self.detail
        if not self.frames:  # cả shot (kích thước vật trọng tâm)
            return f"Shot {self.shot}: {detail}"
        return f"Shot {self.shot}, {_frames_label(self.frames, self.total_samples)}: {detail}"


def _pct_label(pct: float) -> str:
    return f"{round(pct * 100)}%"


def _frames_label(frames: list[str], total: int) -> str:
    if total and len(frames) == total:
        return "mọi frame đo"
    if len(frames) <= 4:
        return "frame " + "/".join(frames)
    return f"frame {frames[0]}…{frames[-1]} ({len(frames)}/{total} mẫu)"


def _quote(text: str) -> str:
    text = text or ""
    return "'" + (text if len(text) <= 40 else text[:39] + "…") + "'"


def _describe(e: dict) -> str:
    kind = e.get("kind")
    if kind == "text":
        return f"chữ {_quote(e.get('text', ''))}"
    if kind == "kit":
        return f"hình {e.get('component') or '?'}"
    if kind == "svg":
        return "khối <svg>"
    return f"khối <{e.get('tag') or '?'}>"


def _box(e: dict) -> dict:
    return (e.get("text_rect") or e["rect"]) if e.get("kind") == "text" else e["rect"]


def _identity(e: dict) -> tuple:
    return (e.get("kind"), e.get("line"), e.get("text") or e.get("component") or e.get("tag") or "")


class _Frame:
    def __init__(self, width: float, height: float) -> None:
        self.w, self.h = width, height
        self.safe: SafeArea = frame_for(width, height).safe

    def beyond(self, b: dict) -> bool:
        """Có phần nào của hộp nằm ngoài khung hình."""
        t = EDGE_TOLERANCE
        return b["x"] < -t or b["y"] < -t or b["x"] + b["w"] > self.w + t or b["y"] + b["h"] > self.h + t

    def outside(self, b: dict) -> bool:
        """Cả hộp nằm ngoài khung hình (không ai nhìn thấy)."""
        return b["x"] + b["w"] <= 0 or b["y"] + b["h"] <= 0 or b["x"] >= self.w or b["y"] >= self.h

    def spans(self, b: dict) -> bool:
        """Hộp vượt qua cả hai mép đối diện của vùng an toàn, theo chiều ngang
        hoặc chiều dọc: một dải, một vạch chia của khung, không phải một vật."""
        t, safe = EDGE_TOLERANCE, self.safe
        across = b["x"] < safe.left - t and b["x"] + b["w"] > safe.right + t
        down = b["y"] < safe.top - t and b["y"] + b["h"] > safe.bottom + t
        return across or down

    @property
    def soft_excess_px(self) -> float:
        return SOFT_EXCESS_FRACTION * min(self.w, self.h)


def _safe_area_excess(b: dict, safe: SafeArea) -> tuple[list[str], float]:
    t = EDGE_TOLERANCE
    parts: list[str] = []
    worst = 0.0
    right, bottom = b["x"] + b["w"], b["y"] + b["h"]
    if b["x"] < safe.left - t:
        parts.append(f"trái x={round(b['x'])} < {safe.left}")
        worst = max(worst, safe.left - b["x"])
    if b["y"] < safe.top - t:
        parts.append(f"trên y={round(b['y'])} < {safe.top}")
        worst = max(worst, safe.top - b["y"])
    if right > safe.right + t:
        parts.append(f"phải x={round(right)} > {safe.right}")
        worst = max(worst, right - safe.right)
    if bottom > safe.bottom + t:
        parts.append(f"dưới y={round(bottom)} > {safe.bottom}")
        worst = max(worst, bottom - safe.bottom)
    return parts, worst


def _intersect(a: dict, b: dict) -> tuple[float, float] | None:
    w = min(a["x"] + a["w"], b["x"] + b["w"]) - max(a["x"], b["x"])
    h = min(a["y"] + a["h"], b["y"] + b["h"]) - max(a["y"], b["y"])
    return (w, h) if w > 1 and h > 1 else None


def _shot_ranges(probe: dict) -> dict[int, tuple[int, int]]:
    """Dòng đầu–cuối của từng hàm shot trong file đã ghép: từ dòng khai báo của
    nó tới trước dòng khai báo kế tiếp (hoặc trước `const SHOTS`)."""
    declared = sorted((s["line"], s["index"]) for s in probe.get("shots", []) if s.get("line"))
    end_all = probe.get("shots_line") or 10**9
    out: dict[int, tuple[int, int]] = {}
    for n, (line, index) in enumerate(declared):
        end = declared[n + 1][0] - 1 if n + 1 < len(declared) else end_all - 1
        out[index] = (line, max(line, end))
    return out


class _Shot:
    """Một shot của lần đo: tên, và dòng nào của file đã ghép thuộc về nó."""

    def __init__(self, shot: dict, rng: tuple[int, int] | None) -> None:
        self.name = shot.get("id") or f"#{shot['index'] + 1}"
        self.line = shot.get("line")
        self.range = rng
        self.total = len(shot.get("samples", []))

    def line_for(self, element_line: int | None) -> int | None:
        """Dòng của thẻ JSX gây lỗi nếu nó nằm trong hàm shot; nếu không (vật vẽ
        từ hình thư viện dán phía trên, lỗi runtime) thì dòng khai báo hàm shot.
        Cả hai đều rơi vào khoảng `Merged.lines` của shot, nên llm-service gán
        đúng shot cho vòng sửa lỗi."""
        if element_line is not None and self.range and self.range[0] <= element_line <= self.range[1]:
            return element_line
        return self.line or element_line

    def finding(self, rule: str, element_line: int | None, detail: str, score: float = 0.0) -> LayoutFinding:
        line = self.line_for(element_line)
        if element_line is not None and line != element_line:
            detail = f"{detail} (vẽ ở dòng {element_line})"
        return LayoutFinding(
            rule, self.name, line, element_line, detail, total_samples=self.total, score=score)


class _Collector:
    def __init__(self) -> None:
        self.found: dict[tuple, LayoutFinding] = {}

    def add(self, key: tuple, finding: LayoutFinding, pct: float) -> None:
        label = _pct_label(pct)
        prev = self.found.get(key)
        if prev is None:
            finding.frames = [label]
            self.found[key] = finding
            return
        if label not in prev.frames:
            prev.frames.append(label)
        # Chặn ở một mẫu là chặn cả vi phạm.
        soft = prev.soft_reason and finding.soft_reason
        if finding.score > prev.score:
            prev.detail, prev.score, prev.soft_reason = finding.detail, finding.score, finding.soft_reason
        if not soft:
            prev.soft_reason = ""


def evaluate(probe: dict, subtitle_band: SubtitleBand | None = None) -> list[LayoutFinding]:
    """Mọi vi phạm của một lần đo, mỗi (shot, luật, vật) một lần, kèm danh sách frame."""
    comp = probe.get("composition") or {}
    frame = _Frame(float(comp.get("width") or 1920), float(comp.get("height") or 1080))
    ranges = _shot_ranges(probe)
    out: list[LayoutFinding] = []
    for shot in probe.get("shots", []):
        out.extend(_evaluate_shot(shot, _Shot(shot, ranges.get(shot["index"])), frame, subtitle_band))
    return out


def _evaluate_shot(
    shot: dict, ctx: _Shot, frame: _Frame, band: SubtitleBand | None,
) -> list[LayoutFinding]:
    samples = shot.get("samples", [])

    def judged(e: dict) -> bool:
        """Phần tử có được xét không: không phủ trọn khung, còn thấy được."""
        return not e.get("full_frame") and not frame.outside(_box(e))

    # Lượt 1: vật trượt, vật trượt ra (nằm trọn trong khung rồi mới ra), vật trọng tâm.
    sliding: set[tuple] = set()
    first_inside: dict[tuple, int] = {}
    last_beyond: dict[tuple, int] = {}
    last_seen: dict[tuple, int] = {}
    largest, hero = -1.0, None
    for i, s in enumerate(samples):
        for e in s.get("elements", []):
            if e.get("full_frame"):
                continue
            ident = _identity(e)
            last_seen[ident] = i
            if frame.beyond(_box(e)):
                sliding.add(ident)
                last_beyond[ident] = i
            else:
                first_inside.setdefault(ident, i)
            if e.get("kind") != "text" and judged(e) and not frame.spans(_box(e)):
                size = max(e["rect"]["w"] / frame.w, e["rect"]["h"] / frame.h)
                if size > largest:
                    largest, hero = size, e
    hero_ident = _identity(hero) if hero is not None else None
    sliding_out = {
        ident for ident in sliding
        if ident in first_inside and first_inside[ident] < last_beyond[ident] == last_seen[ident]
    }

    # Lượt 2: chấm từng mẫu.
    col = _Collector()
    for s in samples:
        pct = float(s.get("pct", 0))
        settled = pct >= SETTLED_PCT - 1e-9
        if s.get("error"):
            first = str(s["error"]).split("\n")[0][:200]
            detail = f"shot ném lỗi khi chạy: {first}"
            col.add((RUNTIME_ERROR, first), ctx.finding(RUNTIME_ERROR, None, detail), pct)
        if s.get("pending_delay_render"):
            col.add((NOT_SETTLED,), ctx.finding(
                NOT_SETTLED, None,
                f"còn {s['pending_delay_render']} tài nguyên (font, Lottie) chưa tải xong sau 5 giây — "
                "số đo có thể thiếu vật"), pct)
        for e in s.get("elements", []):
            if not judged(e):
                continue
            ident = _identity(e)
            # Vật bị máy quay đẩy ra khỏi khung không phải vật trượt ra: luật máy quay xét nó.
            leaving = ident in sliding_out and not e.get("camera_moved")
            placed = ident not in sliding or (settled and not leaving)
            if placed and (e.get("kind") == "text" or not frame.spans(_box(e))):
                _check_placement(e, ctx, frame, band, col, pct, is_hero=ident == hero_ident)
            if e.get("kind") == "text":
                _check_text(e, ctx, settled, col, pct)
        _check_overlaps(s.get("elements", []), ctx, frame, col, pct)

    found = list(col.found.values())
    # Shot chỉ có chữ (thẻ tiêu đề) không có vật trọng tâm để đo.
    if hero is not None and largest < MIN_HERO_FRACTION:
        found.append(ctx.finding(
            HERO_SIZE, hero.get("line"),
            f"vật lớn nhất ({_describe(hero)}) chỉ chiếm {round(largest * 100)}% chiều khung "
            f"(< {round(MIN_HERO_FRACTION * 100)}%) — khung dễ thành nền trống với vật nhỏ lọt thỏm"))
    return found


def _check_placement(
    e: dict, ctx: _Shot, frame: _Frame, band: SubtitleBand | None, col: _Collector, pct: float,
    is_hero: bool = False,
) -> None:
    """Vùng an toàn và vùng phụ đề.

    Lấn vùng an toàn chỉ là cảnh báo khi lấn rất ít, hoặc khi máy quay đang
    zoom/lia đẩy một vật không phải vật trọng tâm ra mép."""
    box, ident, line = _box(e), _identity(e), e.get("line")
    parts, worst = _safe_area_excess(box, frame.safe)
    if parts:
        detail = f"{_describe(e)} ra ngoài vùng an toàn ({', '.join(parts)})"
        finding = ctx.finding(SAFE_AREA, line, detail, worst)
        if e.get("camera_moved") and not is_hero:
            finding.soft_reason = "máy quay zoom/lia đẩy vật phụ ra mép, chỉ cảnh báo"
        elif worst <= frame.soft_excess_px:
            finding.soft_reason = f"lấn nhẹ ≤ {round(frame.soft_excess_px)}px, chỉ cảnh báo"
        col.add((SAFE_AREA, ident), finding, pct)
    if band is None:
        return
    if band.edge == "top":
        over = band.px - box["y"]
        where = f"mép trên y={round(box['y'])} < {band.px}"
    else:
        limit = frame.h - band.px
        over = box["y"] + box["h"] - limit
        where = f"mép dưới y={round(box['y'] + box['h'])} > {round(limit)}"
    if over > EDGE_TOLERANCE:
        side = "trên" if band.edge == "top" else "dưới"
        detail = f"{_describe(e)} lấn vùng phụ đề ở mép {side} khung ({where})"
        col.add((SUBTITLE_ZONE, ident), ctx.finding(SUBTITLE_ZONE, line, detail, over), pct)


def _check_text(e: dict, ctx: _Shot, settled: bool, col: _Collector, pct: float) -> None:
    """Chữ tràn khung và chữ quá nhỏ."""
    ident, line, label = _identity(e), e.get("line"), _quote(e.get("text", ""))
    sw, cw = e.get("scroll_width"), e.get("client_width")
    sh, ch = e.get("scroll_height"), e.get("client_height")
    fs = float(e.get("font_size") or 0)
    if cw and sw is not None and sw > cw + OVERFLOW_TOLERANCE_PX:
        detail = f"nhãn {label} tràn khung chữ (rộng {sw}px > width {cw}px)"
        col.add((TEXT_OVERFLOW, ident), ctx.finding(TEXT_OVERFLOW, line, detail, sw - cw), pct)
    elif ch and sh is not None and sh > ch + fs / 2:
        # Nửa dòng chứ không phải 1px: lineHeight 1.2 với dấu tiếng Việt đã làm
        # scrollHeight lớn hơn chiều cao tự nhiên vài px.
        detail = f"nhãn {label} tràn khung chữ theo chiều cao (cao {sh}px > height {ch}px)"
        col.add((TEXT_OVERFLOW, ident), ctx.finding(TEXT_OVERFLOW, line, detail, sh - ch), pct)
    rendered = e.get("font_size_rendered")
    if settled and rendered is not None and rendered < MIN_FONT_PX:
        detail = f"chữ {label} cỡ {rendered:g}px < {MIN_FONT_PX}px"
        col.add((MIN_FONT, ident), ctx.finding(MIN_FONT, line, detail, MIN_FONT_PX - rendered), pct)


def _check_overlaps(elements: list[dict], ctx: _Shot, frame: _Frame, col: _Collector, pct: float) -> None:
    """Hai khối chữ cùng đang hiện rõ mà đè lên nhau."""
    texts = [
        (i, e) for i, e in enumerate(elements)
        if e.get("kind") == "text" and float(e.get("opacity", 1)) >= OVERLAP_MIN_OPACITY
        and not frame.outside(_box(e))
    ]
    for a in range(len(texts)):
        for b in range(a + 1, len(texts)):
            (ia, ea), (ib, eb) = texts[a], texts[b]
            if ea.get("text_parent") == ib or eb.get("text_parent") == ia:
                continue  # chữ lồng trong chữ: một khối
            hit = _intersect(_box(ea), _box(eb))
            if hit is None:
                continue
            w, h = hit
            detail = (
                f"nhãn {_quote(ea.get('text', ''))} (dòng {ea.get('line')}) đè lên nhãn "
                f"{_quote(eb.get('text', ''))} (dòng {eb.get('line')}) — giao nhau {round(w)}×{round(h)}px"
            )
            key = (TEXT_OVERLAP, _identity(ea), _identity(eb))
            col.add(key, ctx.finding(TEXT_OVERLAP, ea.get("line"), detail, w * h), pct)
