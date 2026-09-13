"""Domain value objects for the Rendering Service (Manim-script input mode).

Rendering executes the Creator's own Manim script twice per project (CR-018):
a dry pass that collects the narration lines in the order they actually run,
and — once those lines have been synthesized — a real pass where each
`self.narrate(...)` waits for exactly as long as its audio.
"""

from __future__ import annotations

import unicodedata
from dataclasses import dataclass, field


@dataclass(frozen=True)
class NarrationSegment:
    """One narration line's timing, in scene_index (i.e. the order the dry pass
    saw them run) — the i-th segment's duration_seconds is how long the i-th
    `self.narrate(...)` call holds the animation.

    audio_path is None when the Creator disabled narration (CR-001):
    duration_seconds is then an estimate from the narration text rather than
    a real audio file's length, but Rendering never reads the audio itself
    either way — only Video Assembly does.
    """

    scene_index: int
    duration_seconds: float
    audio_path: str | None = None


@dataclass(frozen=True)
class ScriptRenderRequest:
    """Input to whole-script rendering. Zero-trust validated by
    RenderScriptUseCase — the service never trusts upstream data."""

    project_id: str
    script_content: str
    scene_class_name: str
    narration_segments: list[NarrationSegment]
    # CR-004 FR12.6: the Creator picks this per project (a fast 720p30 draft to
    # check the content, then a 1080p60 pass for upload). None means "use the
    # service default", which covers projects created before the field existed.
    render_quality: str | None = None

    def __post_init__(self) -> None:
        # A browser clipboard on macOS hands out Vietnamese diacritics
        # decomposed (NFD: base letter + combining marks as separate
        # codepoints) rather than precomposed (NFC: one codepoint per
        # letter). Manim's Text mobject builds one submobject per codepoint
        # via Pango glyph shaping, which can merge a decomposed sequence into
        # fewer glyphs than codepoints — the mismatch then crashes
        # `_gen_chars` with `IndexError: list index out of range`. Normalizing
        # to NFC here, once, before the script ever reaches Manim, means every
        # accented codepoint is the same one Pango shapes 1:1.
        object.__setattr__(self, "script_content", unicodedata.normalize("NFC", self.script_content))


@dataclass(frozen=True)
class OverlapWarning:
    """Một cặp mobject chồng lấn hình học, phát hiện ở lượt dry (bug report
    2026-09-12: một `self.caption(...)` không định vị đã chồng khít lên một
    bảng đang hiện, cả hai không đọc được).

    Cùng hình dạng hai-trường như `LintIssue` (`script_lint.py`) để cách xử lý
    warning nhất quán trên toàn service — chỉ khác `line` (vị trí trong file
    nguồn) thay bằng `narration_index` (lời thoại sắp chạy khi phát hiện chồng
    lấn — không có ý nghĩa "dòng file" nào ở đây, vì phát hiện diễn ra ở
    runtime giữa hai lời thoại).
    """

    narration_index: int
    description: str

    def __str__(self) -> str:
        return f"chồng lấn hình ảnh tại lời thoại #{self.narration_index}: {self.description}"


@dataclass(frozen=True)
class DryRunResult:
    """What the dry pass learned by running the script without rendering it.

    `narrations` is in **runtime order**, not file order. That distinction is
    the whole point of CR-018: a `self.narrate(...)` inside a loop or a helper
    contributes exactly as many lines as it really produces, which reading
    `# NARRATION:` comments out of the source could never get right.

    `beats` and `chapters` carry (narration_index, value): each marker attaches
    to the narration line that follows it, so its timestamp is the real offset
    the render pass measures rather than an estimate (CR-006 FR15).
    """

    narrations: list[str]
    #: CR-024 FR68.5 — mô tả ngắn khung hình tại mỗi lời thoại, cùng thứ tự với
    #: `narrations`. Dùng cho màn duyệt dàn ý, không ảnh hưởng gì tới render.
    visuals: list[str] = field(default_factory=list)
    beats: list[tuple[int, str]] = field(default_factory=list)
    chapters: list[tuple[int, str]] = field(default_factory=list)
    # Bug report (2026-09-12): `with self.clip(...)` đã ghi ra marks file từ
    # lượt dry rồi (script chạy y hệt, chỉ không render hình) — chỉ là trước
    # đây dry_run() không đọc lại. Hệ quả: một project chọn video_output_mode
    # short/both mà script quên đánh dấu self.clip() phải render xong (tốn cả
    # TTS) mới biết "Chưa có clip nào". Đọc ở đây để Orchestrator cảnh báo
    # ngay tại màn duyệt dàn ý, trước khi TTS chạy.
    clip_marks: list[dict] = field(default_factory=list)
    # Bug report (2026-09-12): CR-024's outline review gate is the only screen
    # a Creator sees between writing a script and paying for TTS/render — so
    # any script-authoring mistake a static lint or a human reviewer could
    # miss (an unpositioned Text landing on top of an existing visual) must
    # surface here too, non-blocking like `warnings` in ValidationResult.
    layout_warnings: list[OverlapWarning] = field(default_factory=list)


@dataclass(frozen=True)
class ScriptRenderResult:
    """Where the silent video landed, plus the timing Video Assembly needs to
    line narration up with it (CR-002 FR3.5).

    wait_offsets[i] is the second, measured from the start of the video, at
    which the i-th `self.narrate(...)` begins waiting — i.e. where narration
    segment i must start playing. It is NOT the running sum of narration durations: the
    animation between narrations pushes every later segment further out.
    """

    video_path: str
    wait_offsets: list[float] = field(default_factory=list)
    video_duration_seconds: float = 0.0
    # CR-021 FR58: một bản ghi cho mỗi mốc narration, giữ nguyên shape script
    # đã ghi ra ({"kind","index","t","mobjects"}). Best-effort, nên rỗng là
    # trạng thái hợp lệ — QC khi đó chỉ chấm được phần audio.
    layout_marks: list[dict] = field(default_factory=list)
    # CR-007 FR19.2: một bản ghi cho mỗi `with self.clip(...)` script mở ra,
    # giữ nguyên shape script đã ghi ({"kind","name","index","t_start","t_end"}).
    # Best-effort như layout_marks — rỗng nghĩa là script không cắt clip nào.
    clip_marks: list[dict] = field(default_factory=list)


@dataclass(frozen=True)
class ChannelAssetRenderRequest:
    """Input to building a channel-wide intro/outro (CR-023 FR65, D3/D4).

    Unlike `ScriptRenderRequest` this carries no `script_content` — the scene
    to run is one of `conceptflow.channel_idents`' two fixed classes, selected
    by `kind`, not an arbitrary Creator script. There is also no
    `narration_segments`: neither scene calls `self.narrate(...)`, so the
    two-pass dry/real machinery CR-018 built for narrated scripts does not
    apply here (D4 — deliberately independent of CR-018).
    """

    kind: str  # "intro" | "outro"
    render_quality: str | None = None


@dataclass(frozen=True)
class ChannelAssetRenderResult:
    """Where the rendered channel asset landed, plus its real duration.

    video-assembly (D1) is the one that decides which asset is "active" and
    stores this alongside `source_hash`/`version` in its own `channel_assets`
    table — Rendering only ever produces the file.
    """

    video_path: str
    video_duration_seconds: float = 0.0
    render_quality: str = ""
