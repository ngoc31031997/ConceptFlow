"""The chunked code pipeline (CR-039 FR102-FR105).

layout/cast -> chunks of N shots in parallel -> deterministic merge ->
compile check -> repair only the shots that failed -> check again.

Remotion takes two shortcuts off the critical path: a storyboard that already
carries `layout` skips the LAYOUT call, and each chunk is compiled (against
stubs for the shots it does not own) and repaired as soon as it is written,
while the other chunks are still being generated. The full-file check at the
end stays the gate.

When a chunk fails, the chunks already talking to the model are allowed to
finish so their billed output reaches the cache for the re-run, and no new chunk
starts; a dead key, an empty balance or the Creator cancelling stops everything
at once. A chunk that ran out of token budget is written again as two halves,
down to one shot (CR-048 T2).

Nothing here pretends: a chunk that cannot be produced fails the run with its
error; a repair round that changes nothing is counted; a script that still
fails the check after the last round is returned as check_failed with the
diagnostics, never as a pass.
"""

from __future__ import annotations

import asyncio
import hashlib
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
    shots_from_manim_trace,
)
from app.provider import ChatRequest, Provider
from app.storyboard import Storyboard, StoryboardError
from app.storyboard import parse as parse_storyboard

Emit = Callable[[dict], Awaitable[None]]

EXTRACT_ATTEMPTS = 2

# A chunk that failed with one of these is written again as two halves (CR-048 T2).
SPLIT_KINDS = (errors.BUDGET, errors.TRUNCATED)
# Errors that fail every other chunk the same way: the other chunks are cancelled
# at once instead of being allowed to finish.
STOP_NOW_KINDS = (errors.AUTH, errors.BALANCE, errors.NOT_CONFIGURED)


class _NotStarted(Exception):
    """A chunk (or half) that never reached the model because the run had already failed."""


@dataclass
class CodeRequest:
    engine: str  # "remotion" | "manim"
    topic: str
    storyboard: str
    system: str
    model: str = ""
    max_tokens: int = 0
    temperature: float = 0.3
    #: CR-044 — approved library drawings: {name, usage, description, code}.
    illustrations: list[dict] = field(default_factory=list)


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
    cached: bool = False
    duration_ms: int = 0

    def to_dict(self) -> dict:
        return {
            "phase": self.phase, "label": self.label, "ok": self.ok, "usage": self.usage.to_dict(),
            "error_kind": self.error_kind, "error_message": self.error_message, "cached": self.cached,
            "duration_ms": self.duration_ms,
        }


@dataclass
class CodeResult:
    code: str
    check_ok: bool
    diagnostics: list[Diagnostic]
    repair_rounds: int
    calls: list[Call]
    warnings: list[str] = field(default_factory=list)
    scene_class_name: str = ""

    def to_dict(self) -> dict:
        return {
            "code": self.code,
            "check_ok": self.check_ok,
            "diagnostics": [{"message": d.message, "line": d.line} for d in self.diagnostics],
            "repair_rounds": self.repair_rounds,
            "calls": [c.to_dict() for c in self.calls],
            "warnings": self.warnings,
            "scene_class_name": self.scene_class_name,
        }


class PipelineFailure(Exception):
    """The run could not produce code. `calls` still lists what was billed."""

    def __init__(self, message: str, calls: list[Call], error: LLMError | None = None, kind: str = "") -> None:
        super().__init__(message)
        self.message = message
        self.calls = calls
        self.error = error
        self.kind = kind or (error.kind if error else errors.MALFORMED)


class ChunkCache:
    """Successful chunk outputs of a run that later failed, kept so a re-run
    does not pay for them again. Dropped on the first fully successful run."""

    def __init__(self, max_entries: int = 512) -> None:
        self._data: dict[str, str] = {}
        self._max = max_entries

    def get(self, key: str) -> str | None:
        return self._data.get(key)

    def put(self, key: str, value: str) -> None:
        if len(self._data) >= self._max:
            self._data.pop(next(iter(self._data)))
        self._data[key] = value

    def drop(self, keys: list[str]) -> None:
        for k in keys:
            self._data.pop(k, None)


def _ms(since: float) -> int:
    return int((time.monotonic() - since) * 1000)


def _key(*parts: str) -> str:
    h = hashlib.sha256()
    for p in parts:
        h.update(p.encode())
        h.update(b"\x00")
    return h.hexdigest()


class CodePipeline:
    def __init__(
        self,
        provider: Provider,
        checker: CheckerPort,
        *,
        chunk_shots: int,
        concurrency: int,
        repair_rounds: int,
        cache: ChunkCache | None = None,
    ) -> None:
        if chunk_shots < 1 or concurrency < 1 or repair_rounds < 0:
            raise ValueError("chunk_shots and concurrency must be >= 1 and repair_rounds >= 0")
        self._p = provider
        self._checker = checker
        self._chunk_shots = chunk_shots
        self._concurrency = concurrency
        self._repair_rounds = repair_rounds
        self._cache = cache or ChunkCache()
        self._library: dict[str, str] = {}

    # -- one model call, with extraction retry ---------------------------------

    async def _ask(
        self,
        req: CodeRequest,
        phase: str,
        label: str,
        build: Callable[[str | None], str],
        parse: Callable[[str], object],
        calls: list[Call],
        cache_key: str | None = None,
    ):
        if cache_key and (hit := self._cache.get(cache_key)) is not None:
            try:
                value = parse(hit)
            except (extract.ExtractError, ValueError):
                self._cache.drop([cache_key])
            else:
                calls.append(Call(phase, label, True, Usage(), cached=True))
                return value
        problem: str | None = None
        for _attempt in range(EXTRACT_ATTEMPTS):
            began = time.monotonic()
            try:
                res = await self._p.chat(ChatRequest(
                    system=req.system, user=build(problem), model=req.model,
                    max_tokens=req.max_tokens, temperature=req.temperature))
            except LLMError as err:
                calls.append(Call(phase, label, False, err.usage, err.kind, str(err), duration_ms=_ms(began)))
                raise PipelineFailure(f"{phase} {label}: {err}", calls, error=err) from err
            try:
                value = parse(res.content)
            except (extract.ExtractError, ValueError) as exc:
                calls.append(Call(phase, label, False, res.usage, errors.MALFORMED, str(exc), duration_ms=_ms(began)))
                problem = str(exc)
                continue
            calls.append(Call(phase, label, True, res.usage, duration_ms=_ms(began)))
            if cache_key:
                self._cache.put(cache_key, res.content)
            return value
        raise PipelineFailure(
            f"{phase} {label}: the model kept returning unusable code ({problem})", calls, kind=errors.MALFORMED)

    # -- parsers --------------------------------------------------------------

    @staticmethod
    def _parse_layout(text: str) -> str:
        code = extract.strip_fence(text).strip()
        if not re.match(r"^const LAYOUT\s*=", code):
            raise extract.ExtractError("the reply must be a single `const LAYOUT = {...};` declaration")
        return code

    @staticmethod
    def _parse_cast(text: str) -> str:
        code = extract.strip_fence(text).strip("\n")
        if not re.match(r"^def setup_cast\s*\(self\)\s*:", code):
            raise extract.ExtractError("the reply must be a single `def setup_cast(self):` method at column 0")
        return code

    def _parse_shots(self, engine: str, expected: list[str], text: str) -> dict[str, str]:
        pattern = merger.REMOTION_SHOT if engine == "remotion" else merger.MANIM_SHOT
        got = extract.shot_map(extract.split_shots(extract.strip_fence(text), pattern))
        missing = [i for i in expected if i not in got]
        if missing:
            raise extract.ExtractError(f"missing shot function(s): {', '.join(missing)}")
        return {i: got[i] for i in expected}  # extras are dropped by the caller's warning path

    # -- the run ---------------------------------------------------------------

    async def run(self, req: CodeRequest, emit: Emit) -> CodeResult:
        if req.engine not in ("remotion", "manim"):
            raise PipelineFailure(f"unknown engine {req.engine!r}", [])
        try:
            sb = parse_storyboard(req.storyboard)
        except StoryboardError as exc:
            raise PipelineFailure(
                "storyboard is not valid JSON — it was written for the manual flow; run step 1b with AI "
                f"again ({exc})", [], kind=errors.MALFORMED) from exc

        calls: list[Call] = []
        warnings: list[str] = []
        used_keys: list[str] = []
        remotion = req.engine == "remotion"
        library: dict[str, str] = {}
        if remotion and req.illustrations:
            library = {i["name"]: i["code"] for i in req.illustrations if i.get("name") and i.get("code")}
            req = replace(req, system=req.system + library_section(req.illustrations))
        self._library = library
        all_shots = sb.all_shots()
        ordered = [sh.id for _, sh in all_shots]

        # 1. shared frame: LAYOUT (Remotion) / cast (Manim)
        frame_key = _key(req.engine, "frame", req.system, req.model, req.storyboard)
        used_keys.append(frame_key)
        given = merger.layout_from_storyboard(sb) if remotion else None
        if given is not None:
            await emit({"type": "phase", "phase": "layout", "source": "storyboard"})
            frame: str = given
        elif remotion:
            await emit({"type": "phase", "phase": "layout"})
            frame = await self._ask(
                req, "layout", "LAYOUT", lambda r: prompts.remotion_layout(sb, r),
                self._parse_layout, calls, frame_key)
        else:
            await emit({"type": "phase", "phase": "cast"})
            frame = await self._ask(
                req, "cast", "setup_cast", lambda r: prompts.manim_cast(sb, r),
                self._parse_cast, calls, frame_key)
        cast_names = re.findall(r"self\.(\w+)\s*=", frame) if not remotion else []

        # 2. chunks in parallel
        chunks = [ordered[i : i + self._chunk_shots] for i in range(0, len(ordered), self._chunk_shots)]
        by_id: dict[str, tuple] = {sh.id: (sc, sh) for sc, sh in all_shots}
        sem = asyncio.Semaphore(self._concurrency)
        done = 0
        results: dict[int, dict[str, str]] = {}
        # One chunk's check is the final check when there is only one chunk.
        early_check = remotion and len(chunks) > 1
        chunk_rounds = [0] * len(chunks)
        # Set once a chunk has failed: chunks already talking to the model finish
        # (their billed output reaches the cache), nothing new starts.
        stopping = False

        async def write(n: int, ids: list[str], first: bool) -> dict[str, str]:
            """One chunk's shots. A chunk the model could not finish within its
            token budget is written again as two halves, down to one shot."""
            prev = by_id[ordered[ordered.index(ids[0]) - 1]] if ids[0] != ordered[0] else None
            nxt_i = ordered.index(ids[-1]) + 1
            nxt = by_id[ordered[nxt_i]] if nxt_i < len(ordered) else None
            key = _key(req.engine, "chunk", req.system, req.model, req.storyboard, frame, ",".join(ids))
            # Remembers that this exact chunk already ran out of budget, so a
            # re-run goes straight to the halves instead of paying for it again.
            split_key = _key(key, "split")
            used_keys.extend((key, split_key))

            def build(r: str | None) -> str:
                if remotion:
                    return prompts.remotion_chunk(sb, frame, ids, prev, nxt, r)
                return prompts.manim_chunk(sb, frame, cast_names, ids, prev, nxt, r)

            async with sem:
                if stopping:
                    raise _NotStarted
                if first:
                    await emit({"type": "chunk_start", "index": n + 1, "total": len(chunks), "shots": ids})
                known_too_big = len(ids) > 1 and self._cache.get(split_key) is not None
                if not known_too_big:
                    try:
                        return await self._ask(
                            req, "chunk", f"{ids[0]}-{ids[-1]}", build,
                            lambda text: self._parse_shots(req.engine, ids, text), calls, key)
                    except PipelineFailure as exc:
                        if exc.kind not in SPLIT_KINDS or len(ids) < 2:
                            raise
                        self._cache.put(split_key, exc.kind)
            # Outside the semaphore: each half takes a slot of its own.
            half = (len(ids) + 1) // 2
            parts = [ids[:half], ids[half:]]
            await emit({"type": "chunk_split", "index": n + 1, "total": len(chunks), "shots": ids, "into": parts})
            out: dict[str, str] = {}
            for part in parts:
                out.update(await write(n, part, False))
            return out

        async def do_chunk(n: int, ids: list[str]) -> None:
            nonlocal done, stopping
            try:
                results[n] = await write(n, ids, True)
            except _NotStarted:
                return
            except BaseException:
                # Set here, not when run() sees the error: the slot this chunk
                # just freed may be handed to a waiting chunk first.
                stopping = True
                raise
            if early_check and not stopping:
                # Outside the semaphore: the check does not hold a generation slot;
                # its repair calls take one each, like any other call.
                chunk_rounds[n] = await self._settle_chunk(
                    req, sb, frame, n, len(chunks), results[n], calls, sem, emit, lambda: stopping)
            done += 1
            await emit({"type": "chunk_done", "index": n + 1, "total": len(chunks), "done": done})

        await emit({"type": "phase", "phase": "chunks", "total": len(chunks)})
        tasks = {asyncio.create_task(do_chunk(n, ids)): n for n, ids in enumerate(chunks)}
        failure: PipelineFailure | None = None
        pending = set(tasks)
        try:
            while pending:
                finished, pending = await asyncio.wait(pending, return_when=asyncio.FIRST_EXCEPTION)
                for t in sorted(finished, key=tasks.__getitem__):
                    exc = t.exception()
                    if exc is None:
                        continue
                    if not isinstance(exc, PipelineFailure) or exc.kind in STOP_NOW_KINDS:
                        # A dead key / no balance / no provider fails every other
                        # chunk the same way, and an unexpected error is not a
                        # chunk's fault: stop paying at once.
                        raise exc
                    if failure is None:
                        failure = exc
        except BaseException:
            # Also the Creator cancelling the run (CancelledError): stop at once.
            for t in tasks:
                t.cancel()
            await asyncio.gather(*tasks, return_exceptions=True)
            raise
        if failure is not None:
            # `calls` is the run's shared list, so it now also holds what the
            # chunks that were allowed to finish spent.
            raise failure

        shots: dict[str, str] = {}
        for n in range(len(chunks)):
            shots.update(results[n])

        # 3. merge + check + repair
        def merge() -> merger.Merged:
            if remotion:
                return merger.merge_remotion(sb, frame, shots, library=library)
            return merger.merge_manim(sb, req.topic, frame, shots)

        await emit({"type": "phase", "phase": "merge"})
        merged = merge()
        # Rounds spent on a chunk before the merge count against the same cap:
        # the cap bounds how many repair turns a shot can wait for.
        rounds = max(chunk_rounds, default=0)
        check: CheckResult
        while True:
            await emit({"type": "phase", "phase": "check", "round": rounds})
            check = await self._checker.check(req.engine, merged.code, merged.scene_class_name)
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
            frame = await self._repair(req, sb, remotion, frame, shots, targets, check, calls, sem)
            merged = merge()

        if check.ok:
            self._cache.drop(used_keys)
        else:
            warnings.append("the script still fails the compile check after the last repair round")
        return CodeResult(
            code=merged.code, check_ok=check.ok, diagnostics=check.diagnostics, repair_rounds=rounds,
            calls=calls, warnings=warnings, scene_class_name=merged.scene_class_name)

    async def _settle_chunk(
        self, req: CodeRequest, sb: Storyboard, frame: str, index: int, total: int,
        shots: dict[str, str], calls: list[Call], sem: asyncio.Semaphore, emit: Emit,
        stopped: Callable[[], bool] = lambda: False,
    ) -> int:
        """Compile one Remotion chunk (`shots` holds exactly its shots) against stubs
        for every other shot and repair its own failing shots, up to the repair cap. Returns the rounds
        used. What it cannot settle here (an error in LAYOUT or outside the chunk,
        the checker being unreachable) is left to the full-file check. Stops
        before another repair round once `stopped()` says the run has failed:
        repairs are not cached, so they would be paid for and thrown away."""
        rounds = 0
        while True:
            if stopped():
                return rounds
            merged = merger.merge_remotion(sb, frame, shots, stub_missing=True, library=self._library)
            try:
                check = await self._checker.check("remotion", merged.code, merged.scene_class_name)
            except CheckerUnavailable:
                return rounds
            if check.ok:
                return rounds
            targets, _ = _map_failures("remotion", merged, check)
            mine = {k: v for k, v in targets.items() if k in shots}
            if not mine or rounds >= self._repair_rounds:
                return rounds
            rounds += 1
            await emit({"type": "chunk_repair", "index": index + 1, "total": total, "round": rounds,
                        "max": self._repair_rounds, "targets": sorted(mine)})
            await self._repair(req, sb, True, frame, shots, mine, check, calls, sem)

    async def _repair(
        self, req: CodeRequest, sb: Storyboard, remotion: bool, frame: str, shots: dict[str, str],
        targets: dict[str, list[Diagnostic]], check: CheckResult, calls: list[Call],
        sem: asyncio.Semaphore,
    ) -> str:
        """Ask for a fix of each failing section, in parallel. A section whose
        fix cannot be parsed keeps its old code (the failed call is recorded)
        and will fail the next check again rather than being dropped."""
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
                    return self._parse_layout(text) if remotion else self._parse_cast(text)
                return self._parse_shots(req.engine, [key], text)[key]

            async with sem:
                try:
                    results[key] = await self._ask(req, "repair", key, build, parse, calls)
                except PipelineFailure as exc:
                    if exc.error is not None and exc.error.kind in (errors.AUTH, errors.BALANCE, errors.NOT_CONFIGURED):
                        raise
                    # unusable repair reply: keep the old code, already recorded in `calls`

        await asyncio.gather(*(fix(k, d) for k, d in targets.items()))
        for key, code in results.items():
            if key in (merger.LAYOUT_KEY, merger.CAST_KEY):
                new_frame = code
            else:
                shots[key] = code
        return new_frame


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
