"""The chunked code pipeline. See ADR-0030.

layout/cast -> chunks of N shots in parallel -> deterministic merge ->
compile check -> repair only the shots that failed -> check again.

The run is split into segments: the shared frame (LAYOUT / cast) and one
segment per chunk of shots. This service keeps none of them: the caller
(authoring-service) stores every finished segment and sends the ones it has
back with the next run. A segment whose fingerprint still matches is reused
without a model call; the others are written. Every segment result, every
billed call and every failed check is streamed as it happens, so nothing paid
for is lost when the run is cut off.

A segment that fails does not stop the others: the run ends
`incomplete` with the failed keys, and the Creator re-runs only those. A dead
key, an empty balance or the Creator cancelling still stops everything at once.
A chunk that ran out of token budget is written again as two halves, down to
one shot; every half is tried even when an earlier one failed. A segment that
still has shots missing fails with the shots that were written as its partial
content and the ids of the shots that were not: the next run writes only the
missing ones.

Remotion takes two shortcuts off the critical path: a storyboard that already
carries `layout` skips the LAYOUT call, and each chunk is compiled (against
stubs for the shots it does not own) and repaired as soon as it is written,
while the other chunks are still being generated. The full-file check at the
end stays the gate, and only runs once every segment is done.

Nothing here pretends: a segment that cannot be produced is reported failed
with its error; a repair round that changes nothing is counted; a script that
still fails the check after the last round is returned as check_failed with the
diagnostics, never as a pass.
"""

from __future__ import annotations

import asyncio
import hashlib
import json
import re
import time
from collections.abc import Awaitable, Callable
from dataclasses import dataclass, field, replace

from app import errors
from app.errors import LLMError, Usage
from app.pipeline import extract, merger, prompts
from app.pipeline.checker import (
    CheckerPort,
    CheckerUnavailable,
    CheckResult,
    Diagnostic,
    LayoutContext,
    shots_from_manim_trace,
)
from app.provider import ChatRequest, Provider
from app.storyboard import Storyboard, StoryboardError
from app.storyboard import parse as parse_storyboard

Emit = Callable[[dict], Awaitable[None]]
Record = Callable[["Call"], Awaitable[None]]

EXTRACT_ATTEMPTS = 2

# A chunk that failed with one of these is written again as two halves.
SPLIT_KINDS = (errors.BUDGET, errors.TRUNCATED)
# Errors that fail every other segment the same way: the other segments are
# cancelled at once instead of being allowed to finish.
STOP_NOW_KINDS = (errors.AUTH, errors.BALANCE, errors.NOT_CONFIGURED)

FRAME_KEY = "frame"
FRAME = "frame"
SHOTS = "shots"
# Where a segment's content came from. The caller also stores "external" and
# "manual" for what the Creator pasted or edited.
SOURCE_AI = "ai"
SOURCE_STORYBOARD = "storyboard"


@dataclass(frozen=True)
class Segment:
    """One unit the caller stores: the frame, or one chunk of shots."""

    key: str
    kind: str  # FRAME | SHOTS
    shots: tuple[str, ...]
    fingerprint: str

    def to_dict(self) -> dict:
        return {"key": self.key, "kind": self.kind, "shots": list(self.shots), "fingerprint": self.fingerprint}


@dataclass(frozen=True)
class DoneSegment:
    """A segment the caller already has: its fingerprint and its content,
    {"code": "..."} for the frame, {"shots": {"1.1": "..."}} for a chunk."""

    fingerprint: str
    content: dict


@dataclass
class CodeRequest:
    engine: str  # "remotion" | "manim"
    topic: str
    storyboard: str
    system: str
    model: str = ""
    max_tokens: int = 0
    temperature: float = 0.3
    #: Applied to every call of the run (layout/cast/chunk/repair);
    #: 0 = no limit. See ChatRequest.max_reasoning_chars.
    max_reasoning_chars: int = 0
    #: Approved library drawings: {name, usage, description, code}.
    illustrations: list[dict] = field(default_factory=list)
    #: For the layout check of a Remotion script: the strip
    #: burned-in subtitles cover ({"edge": "top"|"bottom", "px": int}, None =
    #: none) and the video's font ("" = the default).
    subtitle_band: dict | None = None
    video_font: str = ""
    #: Shots per chunk for this project; 0 = the service default.
    chunk_shots: int = 0
    #: The segments the caller already stored, by key.
    done: dict[str, DoneSegment] = field(default_factory=dict)
    #: Run only these segments; None = every missing one.
    only: set[str] | None = None

    def layout(self) -> LayoutContext | None:
        if self.engine != "remotion":
            return None
        return LayoutContext(subtitle_band=self.subtitle_band, video_font=self.video_font)


def library_section(illustrations: list[dict]) -> str:
    """The Remotion Engineer's list of library drawings beyond the built-in kit."""
    rows = [
        f"- {i['usage'] or '<' + i['name'] + ' />'} — {i.get('description', '').strip()}"
        for i in illustrations if i.get("name") and i.get("code")
    ]
    if not rows:
        return ""
    return (
        "\n\n## C4. HÌNH THƯ VIỆN ĐÃ DUYỆT CHO VIDEO NÀY — dùng như bộ minh hoạ ở mục C3\n\n"
        "Các hình dưới đây đã được Creator duyệt; khung code tự đưa chúng vào file, bạn KHÔNG import và KHÔNG "
        "viết lại chúng. Cùng quy ước x, y (tâm), size (cạnh dài), rotate, flip, scale, opacity, still. "
        "Vật nào trong \"visual\" có ở đây thì BẮT BUỘC dùng đúng component này.\n\n" + "\n".join(rows) + "\n"
    )


@dataclass
class Call:
    phase: str  # layout | cast | chunk | repair
    label: str
    ok: bool
    usage: Usage
    error_kind: str = ""
    error_message: str = ""
    duration_ms: int = 0
    segment: str = ""

    def to_dict(self) -> dict:
        return {
            "phase": self.phase, "label": self.label, "ok": self.ok, "usage": self.usage.to_dict(),
            "error_kind": self.error_kind, "error_message": self.error_message,
            "duration_ms": self.duration_ms, "segment": self.segment,
        }


def _diag_dict(d: Diagnostic) -> dict:
    return {"message": d.message, "line": d.line, "kind": d.kind, "rule": d.rule}


@dataclass
class CodeResult:
    #: "done" — every segment is done and the script was merged and checked;
    #: "incomplete" — some segment failed or was not run, nothing was merged.
    status: str
    code: str = ""
    check_ok: bool = False
    diagnostics: list[Diagnostic] = field(default_factory=list)
    repair_rounds: int = 0
    #: Every billed call of the run, also streamed one by one as `call` events.
    calls: list[Call] = field(default_factory=list)
    warnings: list[str] = field(default_factory=list)
    scene_class_name: str = ""
    #: Segments that failed in this run, and segments still without content.
    failed: list[str] = field(default_factory=list)
    missing: list[str] = field(default_factory=list)

    def to_dict(self) -> dict:
        return {
            "status": self.status,
            "code": self.code,
            "check_ok": self.check_ok,
            "diagnostics": [_diag_dict(d) for d in self.diagnostics],
            "repair_rounds": self.repair_rounds,
            "warnings": self.warnings,
            "scene_class_name": self.scene_class_name,
            "failed": self.failed,
            "missing": self.missing,
        }


@dataclass(frozen=True)
class ShotFailure:
    """Shots of a chunk the model could not write.

    `tried` lists the shot groups asked for on the way down to these shots,
    from the first call to the last (the chunk, then each half it was split
    into)."""

    ids: tuple[str, ...]
    kind: str
    message: str
    error: LLMError | None
    tried: tuple[tuple[str, ...], ...]


@dataclass
class ChunkOutcome:
    """What writing a chunk produced: the shots written, and the shots that
    failed with why."""

    shots: dict[str, str] = field(default_factory=dict)
    failures: list[ShotFailure] = field(default_factory=list)


# The Creator-facing reason for a failure kind; other kinds keep the error's
# own message.
_TRUNCATED_REASON = "câu trả lời bị cắt giữa chừng vì hết token"


def _shot_label(ids: tuple[str, ...] | list[str]) -> str:
    return f"shot {ids[0]}" if len(ids) == 1 else f"shot {ids[0]}–{ids[-1]}"


def _failure_reason(f: ShotFailure, max_reasoning_chars: int) -> str:
    if f.kind == errors.BUDGET:
        if max_reasoning_chars > 0:
            return f"model suy nghĩ quá {max_reasoning_chars} ký tự mà chưa viết được chữ nào"
        return "model dùng hết token để suy nghĩ mà chưa viết được chữ nào"
    if f.kind == errors.TRUNCATED:
        return _TRUNCATED_REASON
    return f.error.message if f.error is not None else f.message


def _tried_text(tried: tuple[tuple[str, ...], ...], segment_shots: tuple[str, ...]) -> str:
    steps = []
    for ids in tried:
        if tuple(ids) == segment_shots and len(ids) > 1:
            steps.append(f"cả đoạn {ids[0]}-{ids[-1]}")
        elif len(ids) == 1:
            steps.append(f"riêng shot {ids[0]}")
        else:
            steps.append(_shot_label(ids))
    return ", rồi ".join(steps)


def shot_failure_message(
    failures: list[ShotFailure], saved: list[str], segment_shots: tuple[str, ...], max_reasoning_chars: int,
) -> str:
    """The Creator-facing error of a segment whose shots failed: which shots,
    why, what was tried, and which shots were saved."""
    parts = []
    for f in failures:
        text = f"{_shot_label(f.ids).capitalize()}: {_failure_reason(f, max_reasoning_chars)}"
        if len(f.tried) > 1:
            text += f" (đã thử {_tried_text(f.tried, segment_shots)})"
        parts.append(text)
    msg = "; ".join(parts) + "."
    if saved:
        msg += f" Đã lưu shot {', '.join(saved)}."
    return msg


class PipelineFailure(Exception):
    """A call, a segment or the whole run failed. Raised out of the run only
    when it cannot go on at all (a dead key, no balance, an unreadable
    storyboard). `calls` stays empty: every billed call was already streamed
    as a `call` event, and must not be recorded twice."""

    def __init__(self, message: str, calls: list[Call], error: LLMError | None = None, kind: str = "") -> None:
        super().__init__(message)
        self.message = message
        self.calls = calls
        self.error = error
        self.kind = kind or (error.kind if error else errors.MALFORMED)


class SegmentNotReady(ValueError):
    """A segment's prompt needs a frame that is not written yet."""


def _ms(since: float) -> int:
    return int((time.monotonic() - since) * 1000)


def _sha(*parts: str) -> str:
    h = hashlib.sha256()
    for p in parts:
        h.update(p.encode())
        h.update(b"\x00")
    return h.hexdigest()


def _dump(obj) -> str:
    return json.dumps(obj, ensure_ascii=False, sort_keys=True)


def _storyboard(req: CodeRequest) -> Storyboard:
    try:
        return parse_storyboard(req.storyboard)
    except StoryboardError as exc:
        raise PipelineFailure(
            "storyboard is not valid JSON — it was written for the manual flow; run step 1b with AI "
            f"again ({exc})", [], kind=errors.MALFORMED) from exc


# -- the plan: segments and their fingerprints (ADR-0030) --------------------

@dataclass
class Plan:
    sb: Storyboard
    segments: list[Segment]
    ordered: list[str]
    # LAYOUT built from the storyboard itself (Remotion only), else None.
    given_frame: str | None

    @property
    def frame(self) -> Segment:
        return self.segments[0]

    @property
    def chunks(self) -> list[Segment]:
        return self.segments[1:]

    def get(self, key: str) -> Segment:
        for s in self.segments:
            if s.key == key:
                return s
        raise KeyError(key)

    def owner(self, section: str) -> str:
        """The segment a merged-file section (a shot id, or the LAYOUT/cast
        frame) belongs to."""
        if section in (merger.LAYOUT_KEY, merger.CAST_KEY):
            return FRAME_KEY
        for s in self.chunks:
            if section in s.shots:
                return s.key
        return ""


def make_plan(req: CodeRequest, chunk_shots: int) -> Plan:
    """Cut the storyboard into segments and fingerprint each one.

    A fingerprint covers what the segment's prompt is built from, so a stored
    segment is reused exactly when writing it again would ask the same thing:
    - the step's prompt (`req.system` before the library drawings are added —
      approving a drawing must not throw away every chunk);
    - the frame: the storyboard (or, for a storyboard-given LAYOUT, that LAYOUT);
    - a chunk: the frame's fingerprint, the storyboard's world and palette
      (with the PALETTE keys derived from it), and its own shots. Not the
      neighbouring shots (a continuity hint only; they would make one edited
      shot invalidate three chunks), and not the model.
    """
    sb = _storyboard(req)
    n = chunk_shots if chunk_shots > 0 else 1
    remotion = req.engine == "remotion"
    prompt_fp = _sha(req.system)
    given = merger.layout_from_storyboard(sb) if remotion else None
    if given is not None:
        frame_fp = _sha("frame-sb", given)
    else:
        frame_fp = _sha("frame", req.engine, prompt_fp, req.storyboard)
    all_shots = sb.all_shots()
    ordered = [sh.id for _, sh in all_shots]
    by_id = {sh.id: (sc, sh) for sc, sh in all_shots}
    shared = _dump({
        "hero": sb.hero, "world": sb.world,
        "palette": [p.model_dump() for p in sb.palette],
        # The keys the code is written against: a change in how a role
        # becomes a key makes every stored chunk stale instead of failing TS2551.
        "palette_keys": merger.palette_keys(sb),
    })
    segments = [Segment(FRAME_KEY, FRAME, (), frame_fp)]
    for i in range(0, len(ordered), n):
        ids = tuple(ordered[i : i + n])
        own = _dump([prompts._shot_json(*by_id[s]) for s in ids])
        segments.append(Segment(
            f"{ids[0]}-{ids[-1]}", SHOTS, ids, _sha("shots", req.engine, prompt_fp, frame_fp, shared, own)))
    return Plan(sb, segments, ordered, given)


def _neighbours(plan: Plan, ids: list[str]) -> tuple:
    by_id = {sh.id: (sc, sh) for sc, sh in plan.sb.all_shots()}
    i = plan.ordered.index(ids[0])
    j = plan.ordered.index(ids[-1]) + 1
    prev = by_id[plan.ordered[i - 1]] if i > 0 else None
    nxt = by_id[plan.ordered[j]] if j < len(plan.ordered) else None
    return prev, nxt


def _cast_names(frame: str) -> list[str]:
    return re.findall(r"self\.(\w+)\s*=", frame)


def _chunk_prompt(req: CodeRequest, plan: Plan, frame: str, ids: list[str], retry: str | None) -> str:
    prev, nxt = _neighbours(plan, ids)
    if req.engine == "remotion":
        return prompts.remotion_chunk(plan.sb, frame, ids, prev, nxt, retry)
    return prompts.manim_chunk(plan.sb, frame, _cast_names(frame), ids, prev, nxt, retry)


def _frame_prompt(req: CodeRequest, plan: Plan, retry: str | None) -> str:
    if req.engine == "remotion":
        return prompts.remotion_layout(plan.sb, retry)
    return prompts.manim_cast(plan.sb, retry)


def _system(req: CodeRequest) -> str:
    if req.engine == "remotion" and req.illustrations:
        return req.system + library_section(req.illustrations)
    return req.system


def _stored_frame(req: CodeRequest, plan: Plan) -> str | None:
    if plan.given_frame is not None:
        return plan.given_frame
    d = req.done.get(FRAME_KEY)
    if d is not None and d.fingerprint == plan.frame.fingerprint:
        code = d.content.get("code")
        if isinstance(code, str) and code.strip():
            return code
    return None


# -- parsers ----------------------------------------------------------------

def parse_layout(text: str) -> str:
    code = extract.strip_fence(text).strip()
    if not re.match(r"^const LAYOUT\s*=", code):
        raise extract.ExtractError("the reply must be a single `const LAYOUT = {...};` declaration")
    return code


def parse_cast(text: str) -> str:
    code = extract.strip_fence(text).strip("\n")
    if not re.match(r"^def setup_cast\s*\(self\)\s*:", code):
        raise extract.ExtractError("the reply must be a single `def setup_cast(self):` method at column 0")
    return code


def parse_shots(engine: str, expected: list[str], text: str) -> dict[str, str]:
    pattern = merger.REMOTION_SHOT if engine == "remotion" else merger.MANIM_SHOT
    got = extract.shot_map(extract.split_shots(extract.strip_fence(text), pattern))
    missing = [i for i in expected if i not in got]
    if missing:
        raise extract.ExtractError(f"missing shot function(s): {', '.join(missing)}")
    return {i: got[i] for i in expected}  # extras are dropped


# -- one segment outside a run: copy its prompt, check a pasted reply ---------

def segment_prompt(req: CodeRequest, key: str, default_chunk_shots: int) -> tuple[str, str]:
    """The (system, user) turn the pipeline would send for one segment, for
    the Creator to run in an outside AI. Raises KeyError for an unknown key
    and SegmentNotReady when a chunk's frame is not written yet."""
    plan = make_plan(req, req.chunk_shots or default_chunk_shots)
    seg = plan.get(key)
    if seg.kind == FRAME:
        if plan.given_frame is not None:
            raise SegmentNotReady("the LAYOUT of this video comes from its storyboard; there is nothing to write")
        return _system(req), _frame_prompt(req, plan, None)
    frame = _stored_frame(req, plan)
    if frame is None:
        raise SegmentNotReady("the frame (LAYOUT / cast) is not written yet — run or paste the frame segment first")
    return _system(req), _chunk_prompt(req, plan, frame, list(seg.shots), None)


def parse_segment(req: CodeRequest, key: str, reply: str, default_chunk_shots: int) -> tuple[str, dict]:
    """Check a reply written outside the pipeline for one segment. Returns the
    segment's (fingerprint, content); raises ValueError (ExtractError) with
    what is wrong, KeyError for an unknown key."""
    plan = make_plan(req, req.chunk_shots or default_chunk_shots)
    seg = plan.get(key)
    if seg.kind == FRAME:
        if plan.given_frame is not None:
            raise SegmentNotReady("the LAYOUT of this video comes from its storyboard; there is nothing to paste")
        code = parse_layout(reply) if req.engine == "remotion" else parse_cast(reply)
        return seg.fingerprint, {"code": code}
    return seg.fingerprint, {"shots": parse_shots(req.engine, list(seg.shots), reply)}


class CodePipeline:
    def __init__(
        self,
        provider: Provider,
        checker: CheckerPort,
        *,
        chunk_shots: int,
        concurrency: int,
        repair_rounds: int,
    ) -> None:
        if chunk_shots < 1 or concurrency < 1 or repair_rounds < 0:
            raise ValueError("chunk_shots and concurrency must be >= 1 and repair_rounds >= 0")
        self._p = provider
        self._checker = checker
        self._chunk_shots = chunk_shots
        self._concurrency = concurrency
        self._repair_rounds = repair_rounds
        self._library: dict[str, str] = {}

    # -- one model call, with extraction retry ---------------------------------

    async def _ask(
        self,
        req: CodeRequest,
        phase: str,
        label: str,
        segment: str,
        build: Callable[[str | None], str],
        parse: Callable[[str], object],
        record: Record,
    ):
        problem: str | None = None
        for _attempt in range(EXTRACT_ATTEMPTS):
            began = time.monotonic()
            try:
                res = await self._p.chat(ChatRequest(
                    system=req.system, user=build(problem), model=req.model,
                    max_tokens=req.max_tokens, temperature=req.temperature,
                    max_reasoning_chars=req.max_reasoning_chars))
            except LLMError as err:
                await record(Call(phase, label, False, err.usage, err.kind, str(err), _ms(began), segment))
                raise PipelineFailure(f"{phase} {label}: {err}", [], error=err) from err
            try:
                value = parse(res.content)
            except (extract.ExtractError, ValueError) as exc:
                await record(Call(phase, label, False, res.usage, errors.MALFORMED, str(exc), _ms(began), segment))
                problem = str(exc)
                continue
            await record(Call(phase, label, True, res.usage, duration_ms=_ms(began), segment=segment))
            return value
        raise PipelineFailure(
            f"{phase} {label}: the model kept returning unusable code ({problem})", [], kind=errors.MALFORMED)

    # -- the run ---------------------------------------------------------------

    async def run(self, req: CodeRequest, emit: Emit) -> CodeResult:
        if req.engine not in ("remotion", "manim"):
            raise PipelineFailure(f"unknown engine {req.engine!r}", [])
        plan = make_plan(req, req.chunk_shots or self._chunk_shots)
        sb = plan.sb
        await emit({"type": "plan", "segments": [s.to_dict() for s in plan.segments]})

        calls: list[Call] = []
        warnings: list[str] = []
        failed: list[str] = []
        remotion = req.engine == "remotion"

        async def record(call: Call) -> None:
            calls.append(call)
            await emit({"type": "call", **call.to_dict()})

        library: dict[str, str] = {}
        if remotion and req.illustrations:
            library = {i["name"]: i["code"] for i in req.illustrations if i.get("name") and i.get("code")}
        self._library = library
        ask_req = replace(req, system=_system(req))

        def stored(seg: Segment) -> dict | None:
            d = req.done.get(seg.key)
            return d.content if d is not None and d.fingerprint == seg.fingerprint else None

        def wanted(seg: Segment) -> bool:
            return req.only is None or seg.key in req.only

        async def segment_failed(seg: Segment, exc: PipelineFailure) -> None:
            failed.append(seg.key)
            err = exc.error.to_dict() if exc.error else {"kind": exc.kind, "message": exc.message}
            await emit({"type": "segment_failed", "key": seg.key, "failed_shots": [],
                        "error": {**err, "kind": exc.kind, "message": exc.message}})

        async def shots_failed(seg: Segment, written: dict[str, str], failures: list[ShotFailure]) -> None:
            """A chunk with shots still missing: the written shots go out as
            the segment's partial content, under its fingerprint."""
            failed.append(seg.key)
            first = failures[0]
            saved = [i for i in seg.shots if i in written]
            message = shot_failure_message(failures, saved, seg.shots, req.max_reasoning_chars)
            err = first.error.to_dict() if first.error else {}
            await emit({
                "type": "segment_failed", "key": seg.key, "fingerprint": seg.fingerprint,
                "content": {"shots": {i: written[i] for i in saved}} if saved else None,
                "failed_shots": [i for f in failures for i in f.ids],
                "error": {**err, "kind": first.kind, "message": message},
            })

        # 1. shared frame: LAYOUT (Remotion) / cast (Manim)
        frame_seg = plan.frame
        frame: str | None = None
        if plan.given_frame is not None:
            await emit({"type": "phase", "phase": "layout", "source": "storyboard"})
            frame = plan.given_frame
            if stored(frame_seg) is None:
                await emit({"type": "segment_done", "key": FRAME_KEY, "fingerprint": frame_seg.fingerprint,
                            "content": {"code": frame}, "source": SOURCE_STORYBOARD, "repaired": False})
        elif (have := stored(frame_seg)) is not None and isinstance(have.get("code"), str):
            frame = have["code"]
        elif wanted(frame_seg):
            phase, label = ("layout", "LAYOUT") if remotion else ("cast", "setup_cast")
            await emit({"type": "phase", "phase": phase})
            await emit({"type": "segment_start", "key": FRAME_KEY})
            began = time.monotonic()
            try:
                frame = await self._ask(
                    ask_req, phase, label, FRAME_KEY, lambda r: _frame_prompt(req, plan, r),
                    parse_layout if remotion else parse_cast, record)
            except PipelineFailure as exc:
                if exc.kind in STOP_NOW_KINDS:
                    raise
                await segment_failed(frame_seg, exc)
            else:
                await emit({"type": "segment_done", "key": FRAME_KEY, "fingerprint": frame_seg.fingerprint,
                            "content": {"code": frame}, "source": SOURCE_AI, "repaired": False,
                            "duration_ms": _ms(began)})
        if frame is None:
            # Nothing can be written against a frame that does not exist.
            return CodeResult(status="incomplete", calls=calls, failed=failed,
                              missing=[s.key for s in plan.segments if s.key not in failed and stored(s) is None])

        # 2. chunks in parallel
        shots: dict[str, str] = {}
        have_keys: set[str] = set()
        todo: list[Segment] = []
        # Shots a segment already has from an earlier run that failed part-way.
        partial: dict[str, dict[str, str]] = {}
        for seg in plan.chunks:
            content = stored(seg)
            got = content.get("shots") if content else None
            have = {i: got[i] for i in seg.shots if isinstance(got, dict) and isinstance(got.get(i), str)}
            if len(have) == len(seg.shots):
                shots.update(have)
                have_keys.add(seg.key)
            elif wanted(seg):
                todo.append(seg)
                partial[seg.key] = have

        sem = asyncio.Semaphore(self._concurrency)
        done = 0
        # One chunk's check is the final check when there is only one chunk.
        early_check = remotion and len(plan.chunks) > 1
        chunk_rounds: dict[str, int] = {}

        async def write(
            n: int, seg: Segment, ids: list[str], first: bool, tried: tuple[tuple[str, ...], ...],
        ) -> ChunkOutcome:
            """Some of one chunk's shots. A group the model could not finish
            within its token budget is written again as two halves, down to
            one shot; both halves are tried whatever happens to the first.
            Only the errors that stop the whole run are raised."""
            tried = (*tried, tuple(ids))
            async with sem:
                if first:
                    await emit({"type": "segment_start", "key": seg.key})
                    await emit({"type": "chunk_start", "index": n + 1, "total": len(todo), "shots": ids})
                try:
                    return ChunkOutcome(shots=await self._ask(
                        ask_req, "chunk", f"{ids[0]}-{ids[-1]}", seg.key,
                        lambda r: _chunk_prompt(req, plan, frame, ids, r),
                        lambda text: parse_shots(req.engine, ids, text), record))
                except PipelineFailure as exc:
                    if exc.kind in STOP_NOW_KINDS:
                        raise
                    if exc.kind not in SPLIT_KINDS or len(ids) < 2:
                        return ChunkOutcome(failures=[ShotFailure(tuple(ids), exc.kind, exc.message, exc.error, tried)])
            # Outside the semaphore: each half takes a slot of its own.
            half = (len(ids) + 1) // 2
            parts = [ids[:half], ids[half:]]
            await emit({"type": "chunk_split", "index": n + 1, "total": len(todo), "shots": ids, "into": parts})
            out = ChunkOutcome()
            for part in parts:
                got = await write(n, seg, part, False, tried)
                out.shots.update(got.shots)
                out.failures.extend(got.failures)
            return out

        async def do_chunk(n: int, seg: Segment) -> None:
            nonlocal done
            began = time.monotonic()
            have = partial[seg.key]
            outcome = await write(n, seg, [i for i in seg.shots if i not in have], True, ())
            written = {**have, **outcome.shots}
            if outcome.failures:
                await shots_failed(seg, written, outcome.failures)
                return
            mine = {i: written[i] for i in seg.shots}
            if early_check:
                # Outside the semaphore: the check does not hold a generation slot;
                # its repair calls take one each, like any other call.
                chunk_rounds[seg.key] = await self._settle_chunk(
                    req, ask_req, plan, frame, seg, n, len(todo), mine, record, sem, emit)
            shots.update(mine)
            have_keys.add(seg.key)
            done += 1
            await emit({"type": "segment_done", "key": seg.key, "fingerprint": seg.fingerprint,
                        "content": {"shots": mine}, "source": SOURCE_AI, "repaired": False,
                        "duration_ms": _ms(began)})
            await emit({"type": "chunk_done", "index": n + 1, "total": len(todo), "done": done})

        await emit({"type": "phase", "phase": "chunks", "total": len(todo)})
        tasks = [asyncio.create_task(do_chunk(n, seg)) for n, seg in enumerate(todo)]
        try:
            for t in asyncio.as_completed(tasks):
                await t
        except BaseException:
            # A dead key / no balance / no provider fails every other chunk the
            # same way, an unexpected error is not a chunk's fault, and the
            # Creator cancelling (CancelledError) means stop: stop paying at once.
            for t in tasks:
                t.cancel()
            await asyncio.gather(*tasks, return_exceptions=True)
            raise

        missing = [s.key for s in plan.chunks if s.key not in have_keys and s.key not in failed]
        if failed or missing:
            return CodeResult(status="incomplete", calls=calls, failed=failed, missing=missing)

        # 3. merge + check + repair
        def merge() -> merger.Merged:
            if remotion:
                return merger.merge_remotion(sb, frame, shots, library=library)
            return merger.merge_manim(sb, req.topic, frame, shots)

        await emit({"type": "phase", "phase": "merge"})
        merged = merge()
        # Rounds spent on a chunk before the merge count against the same cap:
        # the cap bounds how many repair turns a shot can wait for.
        rounds = max(chunk_rounds.values(), default=0)
        check: CheckResult
        while True:
            await emit({"type": "phase", "phase": "check", "round": rounds})
            check = await self._checker.check(req.engine, merged.code, merged.scene_class_name, req.layout())
            if not check.ok:
                await emit(_check_event("final", rounds, "", plan, merged, check))
            if check.ok or rounds >= self._repair_rounds:
                break
            targets, unmapped = _map_failures(req.engine, merged, check)
            if not targets:
                warnings.append(
                    "the check failed outside any AI-written section (the fixed frame or an unmapped "
                    "error), so there was nothing to send to the repair agent: "
                    + "; ".join(d.message for d in (unmapped or check.diagnostics))[:500])
                break
            rounds += 1
            await emit({"type": "phase", "phase": "repair", "round": rounds, "total": self._repair_rounds,
                        "targets": sorted(targets)})
            new_frame = await self._repair(ask_req, plan, remotion, frame, shots, targets, check, record, sem)
            changed = {plan.owner(k) for k in targets}
            frame = new_frame
            # The repaired sections overwrite the segments that own them.
            for key in sorted(k for k in changed if k):
                seg = plan.get(key)
                content = {"code": frame} if seg.kind == FRAME else {"shots": {i: shots[i] for i in seg.shots}}
                await emit({"type": "segment_done", "key": key, "fingerprint": seg.fingerprint,
                            "content": content, "repaired": True})
            merged = merge()

        if not check.ok:
            warnings.append("the script still fails the compile check after the last repair round")
        # The final check's own warnings (a hero drawn too small, or
        # a layout check that could not run) — never dropped.
        warnings.extend(w for w in check.warnings if w not in warnings)
        return CodeResult(
            status="done", code=merged.code, check_ok=check.ok, diagnostics=check.diagnostics,
            repair_rounds=rounds, calls=calls, warnings=warnings, scene_class_name=merged.scene_class_name)

    async def _settle_chunk(
        self, req: CodeRequest, ask_req: CodeRequest, plan: Plan, frame: str, seg: Segment, index: int,
        total: int, shots: dict[str, str], record: Record, sem: asyncio.Semaphore, emit: Emit,
    ) -> int:
        """Compile one Remotion chunk (`shots` holds exactly its shots) against stubs
        for every other shot and repair its own failing shots, up to the repair cap. Returns the rounds
        used. What it cannot settle here (an error in LAYOUT or outside the chunk,
        the checker being unreachable) is left to the full-file check."""
        rounds = 0
        while True:
            merged = merger.merge_remotion(plan.sb, frame, shots, stub_missing=True, library=self._library)
            try:
                check = await self._checker.check("remotion", merged.code, merged.scene_class_name, req.layout())
            except CheckerUnavailable:
                return rounds
            if check.ok:
                return rounds
            await emit(_check_event("chunk", rounds, seg.key, plan, merged, check))
            targets, _ = _map_failures("remotion", merged, check)
            mine = {k: v for k, v in targets.items() if k in shots}
            if not mine or rounds >= self._repair_rounds:
                return rounds
            rounds += 1
            await emit({"type": "chunk_repair", "index": index + 1, "total": total, "round": rounds,
                        "max": self._repair_rounds, "targets": sorted(mine)})
            await self._repair(ask_req, plan, True, frame, shots, mine, check, record, sem)

    async def _repair(
        self, req: CodeRequest, plan: Plan, remotion: bool, frame: str, shots: dict[str, str],
        targets: dict[str, list[Diagnostic]], check: CheckResult, record: Record,
        sem: asyncio.Semaphore,
    ) -> str:
        """Ask for a fix of each failing section, in parallel. A section whose
        fix cannot be parsed keeps its old code (the failed call is recorded)
        and will fail the next check again rather than being dropped."""
        sb = plan.sb
        new_frame = frame
        results: dict[str, str] = {}

        async def fix(key: str, diags: list[Diagnostic]) -> None:
            is_frame = key in (merger.LAYOUT_KEY, merger.CAST_KEY)
            current = frame if is_frame else shots[key]
            if remotion:
                build = lambda r: prompts.remotion_repair(sb, frame, key, current, diags, [])  # noqa: E731
            else:
                build = lambda r: prompts.manim_repair(sb, frame, key, current, diags, check.raw)  # noqa: E731

            def parse(text: str) -> str:
                if is_frame:
                    return parse_layout(text) if remotion else parse_cast(text)
                return parse_shots(req.engine, [key], text)[key]

            async with sem:
                try:
                    results[key] = await self._ask(req, "repair", key, plan.owner(key), build, parse, record)
                except PipelineFailure as exc:
                    if exc.kind in STOP_NOW_KINDS:
                        raise
                    # unusable repair reply: keep the old code, already recorded as a call

        await asyncio.gather(*(fix(k, d) for k, d in targets.items()))
        for key, code in results.items():
            if key in (merger.LAYOUT_KEY, merger.CAST_KEY):
                new_frame = code
            else:
                shots[key] = code
        return new_frame


def _check_event(
    phase: str, round_: int, segment: str, plan: Plan, merged: merger.Merged, check: CheckResult,
) -> dict:
    """A failed check, for the caller's diagnostics log: each
    diagnostic with the shot and segment its line falls in."""
    out = []
    for d in check.diagnostics:
        section = merged.shot_at(d.line) if d.line else None
        shot = section if section and section not in (merger.LAYOUT_KEY, merger.CAST_KEY) else ""
        owner = plan.owner(section) if section else ""
        out.append({**_diag_dict(d), "shot": shot, "segment": owner or segment})
    return {"type": "check", "phase": phase, "round": round_, "segment": segment, "diagnostics": out}


def _map_failures(
    engine: str, merged: merger.Merged, check: CheckResult
) -> tuple[dict[str, list[Diagnostic]], list[Diagnostic]]:
    """Group diagnostics by the section that owns the failing line. Returns
    (section -> diagnostics, diagnostics that belong to no AI-written section)."""
    targets: dict[str, list[Diagnostic]] = {}
    unmapped: list[Diagnostic] = []
    for d in check.diagnostics:
        owner = merged.shot_at(d.line) if d.line else None
        if owner is None:
            unmapped.append(d)
        else:
            targets.setdefault(owner, []).append(d)
    if engine == "manim" and check.raw:
        # A traceback names the shot function on lines that carry no diagnostic.
        for shot_id, line in shots_from_manim_trace(check.raw):
            if shot_id in merged.lines:
                targets.setdefault(shot_id, [Diagnostic(f"raised at line {line}", line)])
    return targets, unmapped
