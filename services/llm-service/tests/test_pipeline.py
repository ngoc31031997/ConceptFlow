import asyncio
import json
import re

import pytest

from app import errors
from app.errors import LLMError, Usage
from app.pipeline import merger
from app.pipeline.checker import CheckResult, Diagnostic
from app.pipeline.run import (
    CodePipeline,
    CodeRequest,
    DoneSegment,
    PipelineFailure,
    SegmentNotReady,
    make_plan,
    parse_segment,
    segment_prompt,
)
from app.provider import ChatRequest, ChatResult


def storyboard(n_shots: int) -> str:
    shots = [{"id": f"1.{i}", "visual": f"v{i}", "narration": f"Câu {i}."} for i in range(1, n_shots + 1)]
    return json.dumps({"hero": "h", "palette": [{"role": "accent", "hex": "#F5B841"}],
                       "scenes": [{"id": "hook", "shots": shots}]})


def tsx(i, body="return null;"):
    a, b = i.split(".")
    return f"// Shot {i}\nfunction Shot{a}_{b}({{duration}}: ShotProps) {{\n  {body}\n}}"


class FakeProvider:
    """Answers by reading which task the user turn asks for."""

    name = "hive"

    def __init__(self, engine="remotion", broken=None, fail_repair=False, bad_first_chunk=False):
        self.engine = engine
        self.calls: list[ChatRequest] = []
        self.broken = broken or set()  # shot ids whose FIRST version has a compile error
        self.fail_repair = fail_repair
        self.bad_first_chunk = bad_first_chunk
        self._chunk_calls = 0

    async def chat(self, req, on_progress=None):
        self.calls.append(req)
        u = req.user
        usage = Usage(model="m", prompt_tokens=10, completion_tokens=5)
        if "LAYOUT (bước 1/2" in u:
            return ChatResult("```tsx\nconst LAYOUT = {hero: {x: 960, y: 480, size: 320}};\n```", usage)
        if "CAST (bước 1/2" in u:
            return ChatResult("```python\ndef setup_cast(self):\n    self.hero = self.shape('square')\n```", usage)
        if m := re.search(r"VIẾT CODE CHO SHOT ([\d.]+) → ([\d.]+)", u):
            self._chunk_calls += 1
            if self.bad_first_chunk and self._chunk_calls == 1:
                return ChatResult("here is some prose with no code", usage)
            lo, hi = m.group(1), m.group(2)
            ids = [f"1.{i}" for i in range(int(lo.split(".")[1]), int(hi.split(".")[1]) + 1)]
            if self.engine == "remotion":
                return ChatResult("```tsx\n" + "\n\n".join(
                    tsx(i, "BROKEN" if i in self.broken else "return null;") for i in ids) + "\n```", usage)
            return ChatResult("```python\n" + "\n\n".join(
                f"def shot_{i.replace('.', '_')}(self):\n    self.narrate('{'BROKEN' if i in self.broken else 'ok'}')"
                for i in ids) + "\n```", usage)
        if "SỬA LỖI" in u:
            if self.fail_repair:
                return ChatResult("no code here", usage)
            sid = re.search(r"trong shot (\d+\.\d+)", u).group(1)
            if self.engine == "remotion":
                return ChatResult("```tsx\n" + tsx(sid) + "\n```", usage)
            return ChatResult(f"```python\ndef shot_{sid.replace('.', '_')}(self):\n    self.narrate('fixed')\n```", usage)
        raise AssertionError("unrecognised prompt: " + u[:80])


class FakeChecker:
    """Fails while any 'BROKEN' marker is in the merged code, reporting its line."""

    def __init__(self, raw_for_manim=False):
        self.codes: list[str] = []
        self.layouts: list = []
        self.raw_for_manim = raw_for_manim

    async def check(self, engine, code, scene_class_name, layout=None):
        self.codes.append(code)
        self.layouts.append(layout)
        diags = [Diagnostic("Cannot find name 'BROKEN'", i)
                 for i, ln in enumerate(code.splitlines(), 1) if "BROKEN" in ln]
        raw = ""
        if self.raw_for_manim and diags:
            raw = "\n".join(f'  File "s.py", line {d.line}, in {_fn(code, d.line)}' for d in diags)
            diags = []
        return CheckResult(ok=not diags and "BROKEN" not in code, diagnostics=diags, raw=raw)


def _fn(code, line):
    for i in range(line - 1, -1, -1):
        m = re.match(r"\s*def (\w+)", code.splitlines()[i])
        if m:
            return m.group(1)


async def emit_none(_):
    return None


def pipeline(provider, checker, chunk=10, rounds=3):
    return CodePipeline(provider, checker, chunk_shots=chunk, concurrency=3, repair_rounds=rounds)


class Events:
    """Collects what a run streams; `done()` is what the caller would have
    stored, ready to send back with the next run."""

    def __init__(self):
        self.all: list[dict] = []

    async def __call__(self, ev):
        self.all.append(ev)

    def of(self, kind):
        return [e for e in self.all if e["type"] == kind]

    def done(self):
        out = {}
        for e in self.of("segment_done"):
            prev = out.get(e["key"])
            out[e["key"]] = DoneSegment(e.get("fingerprint") or prev.fingerprint, e["content"])
        return out


def req(sb, engine="remotion"):
    return CodeRequest(engine=engine, topic="Chủ đề", storyboard=sb, system="SYS")


@pytest.mark.parametrize("engine", ["remotion", "manim"])
async def test_every_call_of_the_run_carries_the_reasoning_limit(engine):
    # CR-048 T1: layout/cast, chunks and repairs all get the request's limit.
    prov, chk = FakeProvider(engine=engine, broken={"1.2"}), FakeChecker()
    r = CodeRequest(engine=engine, topic="Chủ đề", storyboard=storyboard(4), system="SYS", max_reasoning_chars=1234)
    res = await pipeline(prov, chk, chunk=2).run(r, emit_none)
    assert {c.phase for c in res.calls} >= {"chunk", "repair"} and {c.phase for c in res.calls} & {"layout", "cast"}
    assert prov.calls and all(c.max_reasoning_chars == 1234 for c in prov.calls)


async def test_happy_path_splits_into_chunks_and_records_every_call():
    prov, chk = FakeProvider(), FakeChecker()
    res = await pipeline(prov, chk, chunk=10).run(req(storyboard(25)), emit_none)
    assert res.check_ok and res.repair_rounds == 0
    assert [c.phase for c in res.calls].count("chunk") == 3 and [c.phase for c in res.calls][0] == "layout"
    assert all(c.usage.prompt_tokens == 10 for c in res.calls)
    assert "const SHOTS: React.FC<ShotProps>[] = [Shot1_1," in res.code and "Shot1_25]" in res.code
    assert res.code.count('"Câu ') == 25  # narrations come from the storyboard, one per shot
    assert all(r.system == "SYS" for r in prov.calls)


async def test_chunk_turn_carries_the_previous_shot_for_seamless_transitions():
    prov = FakeProvider()
    await pipeline(prov, FakeChecker(), chunk=2).run(req(storyboard(4)), emit_none)
    second = next(c.user for c in prov.calls if "VIẾT CODE CHO SHOT 1.3 → 1.4" in c.user)
    assert '"shot": "1.2"' in second and "Câu 2." in second


async def test_repair_only_touches_the_failing_shot_and_then_passes():
    prov, chk = FakeProvider(broken={"1.3"}), FakeChecker()
    res = await pipeline(prov, chk).run(req(storyboard(5)), emit_none)
    assert res.check_ok and res.repair_rounds == 1
    repairs = [c for c in prov.calls if "SỬA LỖI" in c.user]
    assert len(repairs) == 1 and "trong shot 1.3" in repairs[0].user
    assert [c.label for c in res.calls if c.phase == "repair"] == ["1.3"]
    assert len(chk.codes) == 2 and "BROKEN" not in res.code


async def test_repair_gives_up_after_max_rounds_and_reports_check_failed_not_pass():
    prov, chk = FakeProvider(broken={"1.2"}, fail_repair=True), FakeChecker()
    res = await pipeline(prov, chk, rounds=2).run(req(storyboard(3)), emit_none)
    assert not res.check_ok and res.repair_rounds == 2
    assert res.diagnostics and "BROKEN" in res.code
    assert any("still fails" in w for w in res.warnings)
    assert sum(1 for c in res.calls if c.phase == "repair" and not c.ok) >= 2  # failed repairs are recorded


async def test_zero_repair_rounds_means_check_only():
    res = await pipeline(FakeProvider(broken={"1.1"}), FakeChecker(), rounds=0).run(req(storyboard(2)), emit_none)
    assert not res.check_ok and res.repair_rounds == 0


async def test_unusable_chunk_reply_is_retried_once_with_the_problem_named():
    prov = FakeProvider(bad_first_chunk=True)
    res = await pipeline(prov, FakeChecker()).run(req(storyboard(2)), emit_none)
    assert res.check_ok
    retry = [c.user for c in prov.calls if "LẦN TRƯỚC BẠN TRẢ LỜI SAI" in c.user]
    assert len(retry) == 1 and "no shot function" in retry[0]
    failed = [c for c in res.calls if not c.ok]
    assert len(failed) == 1 and failed[0].error_kind == errors.MALFORMED


async def test_a_dead_key_fails_the_run_with_the_provider_error_and_no_retry():
    class Dead(FakeProvider):
        async def chat(self, req, on_progress=None):
            raise LLMError(errors.AUTH, "hive", "key rejected", Usage())

    with pytest.raises(PipelineFailure) as e:
        await pipeline(Dead(), FakeChecker()).run(req(storyboard(2)), emit_none)
    assert e.value.kind == errors.AUTH and e.value.error is not None


async def test_a_storyboard_that_is_prose_is_refused_with_an_actionable_message():
    with pytest.raises(PipelineFailure) as e:
        await pipeline(FakeProvider(), FakeChecker()).run(req("CẢNH 1 — hook\n1.1 | MÁY: a"), emit_none)
    assert "run step 1b with AI" in e.value.message and e.value.calls == []


async def test_a_rerun_with_every_segment_stored_pays_for_no_writing_turn():
    ev = Events()
    prov1 = FakeProvider(broken={"1.1"}, fail_repair=True)
    res1 = await pipeline(prov1, FakeChecker(), chunk=1, rounds=1).run(req(storyboard(3)), ev)
    assert res1.status == "done" and not res1.check_ok
    prov2 = FakeProvider()
    r = req(storyboard(3))
    r.done = ev.done()
    res2 = await pipeline(prov2, FakeChecker(), chunk=1, rounds=1).run(r, emit_none)
    # layout + 3 chunks come from the caller; the stored 1.1 is still broken, so
    # the re-run pays for exactly one thing: repairing it.
    assert res2.check_ok and res2.repair_rounds == 1
    assert [c.phase for c in res2.calls] == ["repair"]
    assert [x for x in prov2.calls if "VIẾT CODE CHO SHOT" in x.user or "LAYOUT (bước 1/2" in x.user] == []


async def test_manim_pipeline_uses_cast_and_maps_tracebacks_to_shots():
    prov, chk = FakeProvider(engine="manim", broken={"1.2"}), FakeChecker(raw_for_manim=True)
    res = await pipeline(prov, chk).run(req(storyboard(3), "manim"), emit_none)
    assert res.check_ok and res.repair_rounds == 1
    assert res.calls[0].phase == "cast" and res.scene_class_name == "ChuDeScene"
    assert "class ChuDeScene(ConceptFlowScene):" in res.code and "self.narrate('fixed')" in res.code


# --- storyboard layout (B) and per-chunk check (C) --------------------------------

def storyboard_with_layout(n_shots: int, layout=None) -> str:
    data = json.loads(storyboard(n_shots))
    data["layout"] = layout if layout is not None else {"hero": {"x": 960, "y": 480, "size": 320}}
    return json.dumps(data)


async def test_a_storyboard_layout_replaces_the_layout_call():
    prov, chk = FakeProvider(), FakeChecker()
    events = []

    async def emit(ev):
        events.append(ev)

    res = await pipeline(prov, chk, chunk=10).run(req(storyboard_with_layout(5)), emit)
    assert res.check_ok
    assert not any("LAYOUT (bước 1/2" in c.user for c in prov.calls)
    assert [c.phase for c in res.calls] == ["chunk"]
    assert "const LAYOUT = {\n  hero: {x: 960, y: 480, size: 320},\n};" in res.code
    chunk_turn = next(c.user for c in prov.calls if "VIẾT CODE CHO SHOT" in c.user)
    assert "hero: {x: 960, y: 480, size: 320}" in chunk_turn
    assert {"type": "phase", "phase": "layout", "source": "storyboard"} in events


async def test_manim_still_writes_its_cast_when_the_storyboard_has_a_layout():
    prov = FakeProvider(engine="manim")
    res = await pipeline(prov, FakeChecker()).run(req(storyboard_with_layout(2), engine="manim"), emit_none)
    assert res.check_ok and [c.phase for c in res.calls][0] == "cast"


async def test_each_remotion_chunk_is_checked_against_stubs_and_repaired_before_the_merge():
    prov, chk = FakeProvider(broken={"1.4"}), FakeChecker()
    events = []

    async def emit(ev):
        events.append(ev)

    res = await pipeline(prov, chk, chunk=3).run(req(storyboard(7)), emit)
    assert res.check_ok and res.repair_rounds == 1
    # 3 chunk checks + the recheck of the repaired chunk + the final full-file check
    assert len(chk.codes) == 5
    early = chk.codes[:4]
    assert all(c.count("function Shot") == 7 for c in early)  # every shot present, as code or stub
    assert all(c.count("// Shot 1.") < 7 for c in early)  # ...and the ones this chunk does not own are stubs
    repairs = [c for c in prov.calls if "SỬA LỖI" in c.user]
    assert len(repairs) == 1 and "trong shot 1.4" in repairs[0].user
    assert any(ev.get("type") == "chunk_repair" and ev["index"] == 2 and ev["targets"] == ["1.4"] for ev in events)
    assert not any(ev["type"] == "phase" and ev["phase"] == "repair" for ev in events)  # nothing left for the final loop
    assert "BROKEN" not in res.code


async def test_a_chunk_is_repaired_while_a_later_chunk_is_still_being_written():
    import asyncio

    release = asyncio.Event()

    class SlowSecondChunk(FakeProvider):
        async def chat(self, req, on_progress=None):
            if "VIẾT CODE CHO SHOT 1.3 → 1.4" in req.user:
                await asyncio.wait_for(release.wait(), timeout=5)  # only a repair of chunk 1 releases it
            if "SỬA LỖI" in req.user:
                release.set()
            return await super().chat(req, on_progress)

    prov = SlowSecondChunk(broken={"1.1"})
    res = await pipeline(prov, FakeChecker(), chunk=2).run(req(storyboard(4)), emit_none)
    assert res.check_ok and res.repair_rounds == 1


async def test_rounds_spent_on_a_chunk_count_against_the_same_cap():
    prov, chk = FakeProvider(broken={"1.2"}, fail_repair=True), FakeChecker()
    res = await pipeline(prov, chk, chunk=2, rounds=2).run(req(storyboard(4)), emit_none)
    assert not res.check_ok and res.repair_rounds == 2
    assert sum(1 for c in prov.calls if "SỬA LỖI" in c.user) == 2 * 2  # 2 rounds x 2 extraction attempts
    assert any("still fails" in w for w in res.warnings)


async def test_an_unreachable_checker_during_chunks_leaves_the_decision_to_the_final_check():
    from app.pipeline.checker import CheckerUnavailable

    class FlakyChecker(FakeChecker):
        async def check(self, engine, code, scene_class_name, layout=None):
            if "return null;\n}" in code and code.count("function Shot") != code.count("// Shot"):
                raise CheckerUnavailable("rendering is restarting")  # only the stubbed per-chunk files
            return await super().check(engine, code, scene_class_name, layout)

    prov, chk = FakeProvider(broken={"1.3"}), FlakyChecker()
    res = await pipeline(prov, chk, chunk=2).run(req(storyboard(4)), emit_none)
    assert res.check_ok and res.repair_rounds == 1
    assert len(chk.codes) == 2  # only full-file checks reached the checker


async def test_one_chunk_is_checked_once():
    chk = FakeChecker()
    res = await pipeline(FakeProvider(), chk, chunk=10).run(req(storyboard(4)), emit_none)
    assert res.check_ok and len(chk.codes) == 1


BUS = """import React from 'react';
import {useCurrentFrame} from 'remotion';
import {Figure, type FigureProps} from './conceptflow-mini/illustration';

export function SchoolBus({color = '#FFC72C', ...fig}: FigureProps & {color?: string}) {
  return <Figure {...fig} size={fig.size ?? 320} vw={320} vh={210}><rect width={10} height={10} rx={4} fill={color} /></Figure>;
}
"""


class UsesBus(FakeProvider):
    async def chat(self, req, on_progress=None):
        res = await super().chat(req, on_progress)
        return ChatResult(res.content.replace("return null;", "return <SchoolBus x={960} y={540} />;"), res.usage)


async def test_library_drawings_reach_the_prompt_and_the_ones_used_are_pasted_into_the_script():
    provider, checker = UsesBus(), FakeChecker()
    r = req(storyboard(2))
    r.illustrations = [
        {"name": "SchoolBus", "usage": "<SchoolBus color /> — 320×210", "description": "Xe buýt vàng", "code": BUS},
        {"name": "Cat", "usage": "<Cat /> — 220×240", "description": "Con mèo", "code": "export function Cat() { return null; }"},
    ]
    res = await pipeline(provider, checker).run(r, emit_none)
    assert all("## C4. HÌNH THƯ VIỆN" in c.system and "<SchoolBus color /> — 320×210" in c.system for c in provider.calls)
    code = res.code
    assert "// Hình thư viện: SchoolBus\nfunction SchoolBus(" in code
    assert "export function SchoolBus" not in code and "function Cat(" not in code  # only what is used
    assert code.count("from './conceptflow-mini/illustration'") == 2  # the frame's own imports only
    assert code.index("function SchoolBus(") < code.index("function Shot1_1(")


async def test_manim_ignores_library_drawings():
    provider, checker = FakeProvider(engine="manim"), FakeChecker()
    r = req(storyboard(1), engine="manim")
    r.illustrations = [{"name": "SchoolBus", "usage": "", "description": "", "code": BUS}]
    await pipeline(provider, checker).run(r, emit_none)
    assert all("C4." not in c.system for c in provider.calls)


# --- CR-048 T2: a failing chunk does not throw away the others; split on budget ----

def chunk_ids(user: str) -> list[str] | None:
    m = re.search(r"VIẾT CODE CHO SHOT ([\d.]+) → ([\d.]+)", user)
    if not m:
        return None
    lo, hi = int(m.group(1).split(".")[1]), int(m.group(2).split(".")[1])
    return [f"1.{i}" for i in range(lo, hi + 1)]


class Scripted(FakeProvider):
    """`fail(ids)` returns the LLMError a chunk call of those shots raises, or None.
    `gate(ids)` returns an asyncio.Event the call waits for first, or None."""

    def __init__(self, fail=None, gate=None, **kw):
        super().__init__(**kw)
        self.fail = fail or (lambda ids: None)
        self.gate = gate or (lambda ids: None)
        self.finished: list[list[str]] = []

    async def chat(self, req, on_progress=None):
        ids = chunk_ids(req.user)
        if ids is not None:
            if (ev := self.gate(ids)) is not None:
                await ev.wait()
                await asyncio.sleep(0.02)  # still streaming for a moment after the gate opens
            if (err := self.fail(ids)) is not None:
                self.calls.append(req)
                raise err
        res = await super().chat(req, on_progress)
        if ids is not None:
            self.finished.append(ids)
        return res


def llm_err(kind):
    return LLMError(kind, "hive", f"{kind} failure", Usage(model="m", prompt_tokens=7, completion_tokens=3))


def chunk_turns(prov):
    return [ids for c in prov.calls if (ids := chunk_ids(c.user)) is not None]


async def test_a_failing_chunk_does_not_stop_the_others_and_a_rerun_pays_only_for_it():
    # CR-050 FR-2: every other chunk is written and handed back; the run ends incomplete.
    prov1 = Scripted(fail=lambda ids: llm_err(errors.SERVER) if ids == ["1.2"] else None)
    ev = Events()
    res1 = await pipeline(prov1, FakeChecker(), chunk=1).run(req(storyboard_with_layout(4)), ev)
    assert res1.status == "incomplete" and res1.failed == ["1.2-1.2"] and res1.missing == []
    assert res1.code == "" and sorted(prov1.finished) == [["1.1"], ["1.3"], ["1.4"]]
    assert sorted(e["key"] for e in ev.of("segment_done")) == ["1.1-1.1", "1.3-1.3", "1.4-1.4", "frame"]
    [failed] = ev.of("segment_failed")
    assert failed["key"] == "1.2-1.2" and failed["error"]["kind"] == errors.SERVER
    assert not ev.of("check")  # nothing is merged while a segment is missing

    prov2 = Scripted()
    r = req(storyboard_with_layout(4))
    r.done = ev.done()
    res2 = await pipeline(prov2, FakeChecker(), chunk=1).run(r, emit_none)
    assert res2.status == "done" and res2.check_ok
    assert chunk_turns(prov2) == [["1.2"]]  # the re-run calls the model for chunk 2 only
    assert res2.code.count("function Shot1_") == 4


async def test_chunks_waiting_for_a_slot_still_start_after_a_failure():
    prov = Scripted(fail=lambda ids: llm_err(errors.SERVER) if ids == ["1.2"] else None)
    p = CodePipeline(prov, FakeChecker(), chunk_shots=1, concurrency=2, repair_rounds=1)
    ev = Events()
    res = await p.run(req(storyboard_with_layout(4)), ev)
    assert sorted(chunk_turns(prov)) == [["1.1"], ["1.2"], ["1.3"], ["1.4"]]
    assert res.failed == ["1.2-1.2"]
    assert sorted(e["index"] for e in ev.of("chunk_start")) == [1, 2, 3, 4]


async def test_every_failed_chunk_is_reported():
    prov = Scripted(fail=lambda ids: {"1.1": llm_err(errors.SERVER), "1.2": llm_err(errors.TIMEOUT)}.get(ids[0]))
    ev = Events()
    res = await pipeline(prov, FakeChecker(), chunk=1).run(req(storyboard_with_layout(3)), ev)
    assert sorted(res.failed) == ["1.1-1.1", "1.2-1.2"]
    assert sorted(e["error"]["kind"] for e in ev.of("segment_failed")) == [errors.SERVER, errors.TIMEOUT]
    assert sorted(c.error_kind for c in res.calls if not c.ok) == [errors.SERVER, errors.TIMEOUT]


async def test_a_chunk_over_budget_is_split_in_two_and_the_run_succeeds():
    prov = Scripted(fail=lambda ids: llm_err(errors.BUDGET) if len(ids) == 5 else None)
    events = []

    async def emit(ev):
        events.append(ev)

    res = await pipeline(prov, FakeChecker(), chunk=5).run(req(storyboard_with_layout(7)), emit)
    assert res.check_ok and "function Shot1_5(" in res.code and "BROKEN" not in res.code
    assert sorted(chunk_turns(prov)) == sorted([
        ["1.1", "1.2", "1.3", "1.4", "1.5"], ["1.6", "1.7"], ["1.1", "1.2", "1.3"], ["1.4", "1.5"]])
    chunk_calls = [(c.label, c.ok, c.error_kind) for c in res.calls if c.phase == "chunk"]
    assert ("1.1-1.5", False, errors.BUDGET) in chunk_calls  # the failed call stays billed
    assert ("1.1-1.3", True, "") in chunk_calls and ("1.4-1.5", True, "") in chunk_calls
    assert next(c for c in res.calls if c.label == "1.1-1.5").usage.prompt_tokens == 7
    split = [ev for ev in events if ev["type"] == "chunk_split"]
    assert split == [{"type": "chunk_split", "index": 1, "total": 2, "shots": ["1.1", "1.2", "1.3", "1.4", "1.5"],
                      "into": [["1.1", "1.2", "1.3"], ["1.4", "1.5"]]}]
    assert sorted(ev["index"] for ev in events if ev["type"] == "chunk_start") == [1, 2]  # halves are not new chunks
    assert sorted(ev["done"] for ev in events if ev["type"] == "chunk_done") == [1, 2]
    # each half carries its own neighbours: the first half leads into 1.4, the second follows 1.3
    first = next(c.user for c in prov.calls if "VIẾT CODE CHO SHOT 1.1 → 1.3" in c.user)
    second = next(c.user for c in prov.calls if "VIẾT CODE CHO SHOT 1.4 → 1.5" in c.user)
    before = lambda u: u.split("SHOT NGAY TRƯỚC lô này")[1].split("SHOT NGAY SAU lô này")[0]  # noqa: E731
    after = lambda u: u.split("SHOT NGAY SAU lô này")[1]  # noqa: E731
    assert "đây là lô đầu tiên" in before(first) and '"shot": "1.4"' in after(first)
    assert '"shot": "1.3"' in before(second) and '"shot": "1.6"' in after(second)


async def test_splitting_recurses_down_to_one_shot_on_truncation():
    prov = Scripted(fail=lambda ids: llm_err(errors.TRUNCATED) if len(ids) > 1 else None)
    res = await pipeline(prov, FakeChecker(), chunk=3).run(req(storyboard_with_layout(3)), emit_none)
    assert res.check_ok
    assert chunk_turns(prov) == [["1.1", "1.2", "1.3"], ["1.1", "1.2"], ["1.1"], ["1.2"], ["1.3"]]
    assert [(c.label, c.ok) for c in res.calls] == [
        ("1.1-1.3", False), ("1.1-1.2", False), ("1.1-1.1", True), ("1.2-1.2", True), ("1.3-1.3", True)]


async def test_a_one_shot_chunk_over_budget_fails_its_segment_with_budget():
    prov = Scripted(fail=lambda ids: llm_err(errors.BUDGET) if "1.2" in ids else None)
    ev = Events()
    res = await pipeline(prov, FakeChecker(), chunk=2).run(req(storyboard_with_layout(2)), ev)
    assert res.status == "incomplete" and res.failed == ["1.1-1.2"]
    assert ev.of("segment_failed")[0]["error"]["kind"] == errors.BUDGET
    assert [(c.label, c.ok) for c in res.calls] == [("1.1-1.2", False), ("1.1-1.1", True), ("1.2-1.2", False)]


async def test_other_error_kinds_are_not_split():
    prov = Scripted(fail=lambda ids: llm_err(errors.EMPTY))
    res = await pipeline(prov, FakeChecker(), chunk=4).run(req(storyboard_with_layout(4)), emit_none)
    assert res.failed == ["1.1-1.4"] and chunk_turns(prov) == [["1.1", "1.2", "1.3", "1.4"]]


async def test_a_dead_key_cancels_the_other_chunks_at_once():
    never = asyncio.Event()
    prov = Scripted(fail=lambda ids: llm_err(errors.AUTH) if ids == ["1.2"] else None,
                    gate=lambda ids: never if ids != ["1.2"] else None)
    ev = Events()
    with pytest.raises(PipelineFailure) as e:
        await asyncio.wait_for(
            pipeline(prov, FakeChecker(), chunk=1).run(req(storyboard_with_layout(3)), ev), timeout=2)
    assert e.value.kind == errors.AUTH
    assert prov.finished == [] and [c["label"] for c in ev.of("call")] == ["1.2-1.2"]
    assert e.value.calls == []  # already streamed; must not be recorded twice


async def test_cancelling_the_run_cancels_every_chunk_at_once():
    never = asyncio.Event()
    started = asyncio.Event()

    def gate(ids):
        started.set()
        return never

    prov = Scripted(gate=gate)
    run = asyncio.create_task(pipeline(prov, FakeChecker(), chunk=1).run(req(storyboard_with_layout(3)), emit_none))
    await asyncio.wait_for(started.wait(), timeout=2)
    run.cancel()
    with pytest.raises(asyncio.CancelledError):
        await asyncio.wait_for(run, timeout=2)
    assert prov.finished == []
    assert [t for t in asyncio.all_tasks() if t is not asyncio.current_task() and not t.done()] == []


# --- CR-048 T6b: layout diagnostics, context and warnings ------------------------------


class LayoutChecker(FakeChecker):
    """Passes tsc, but reports a layout fault on Shot1_2's declaration line until
    the shot has been repaired once; always warns about a small hero."""

    async def check(self, engine, code, scene_class_name, layout=None):
        self.codes.append(code)
        self.layouts.append(layout)
        warn = ["Bố cục: Shot 1.1: vật lớn nhất (hình Apple) chỉ chiếm 19% chiều khung (dòng 20)"]
        at = next((i for i, ln in enumerate(code.splitlines(), 1) if ln.startswith("function Shot1_2(")), None)
        if at is None or "// repaired" in code:
            return CheckResult(ok=True, warnings=warn)
        return CheckResult(ok=False, warnings=warn, diagnostics=[Diagnostic(
            "Shot 1.2, frame 85%: nhãn 'Vi khuẩn axit' tràn khung chữ (rộng 412px > width 360px)", at, "layout")])


class LayoutRepairProvider(FakeProvider):
    async def chat(self, req, on_progress=None):
        if "SỬA LỖI" in req.user:
            self.calls.append(req)
            return ChatResult("```tsx\n// repaired\n" + tsx("1.2") + "\n```",
                              Usage(model="m", prompt_tokens=10, completion_tokens=5))
        return await super().chat(req, on_progress)


async def test_a_layout_diagnostic_is_repaired_on_its_shot_with_a_layout_prompt():
    prov, chk = LayoutRepairProvider(), LayoutChecker()
    r = CodeRequest(engine="remotion", topic="t", storyboard=storyboard(3), system="SYS",
                    subtitle_band={"edge": "bottom", "px": 240}, video_font="Montserrat")
    res = await pipeline(prov, chk).run(r, emit_none)
    assert res.check_ok and res.repair_rounds == 1
    [repair] = [c for c in prov.calls if "SỬA LỖI" in c.user]
    assert "SỬA LỖI BỐ CỤC trong shot 1.2" in repair.user and "Lỗi bố cục — đo trên hình thật" in repair.user
    assert "Lỗi trình biên dịch TypeScript" not in repair.user
    assert "tràn khung chữ (rộng 412px > width 360px)" in repair.user
    # the layout context reaches every check, and the final check's warning is kept
    assert chk.layouts and all(
        lc.subtitle_band == {"edge": "bottom", "px": 240} and lc.video_font == "Montserrat" for lc in chk.layouts)
    assert res.warnings == ["Bố cục: Shot 1.1: vật lớn nhất (hình Apple) chỉ chiếm 19% chiều khung (dòng 20)"]
    assert res.to_dict()["diagnostics"] == []


async def test_layout_context_is_not_sent_for_manim():
    chk = FakeChecker()
    await pipeline(FakeProvider(engine="manim"), chk).run(req(storyboard(2), engine="manim"), emit_none)
    assert chk.layouts == [None]


async def test_a_layout_check_that_could_not_run_surfaces_as_a_warning_of_the_run():
    class Unchecked(FakeChecker):
        async def check(self, engine, code, scene_class_name, layout=None):
            res = await super().check(engine, code, scene_class_name, layout)
            res.warnings = ["Bố cục: KHÔNG kiểm tra được — đo bố cục quá 60s (đo). Kết quả biên dịch vẫn giữ nguyên."]
            return res

    res = await pipeline(FakeProvider(), Unchecked()).run(req(storyboard(2)), emit_none)
    assert res.check_ok and any("KHÔNG kiểm tra được" in w for w in res.warnings)


def test_repair_prompt_lists_compile_and_layout_errors_apart():
    from app.pipeline import prompts
    from app.storyboard import parse

    sb = parse(storyboard(2))
    text = prompts.remotion_repair(sb, "const LAYOUT = {};", "1.2", tsx("1.2"), [
        Diagnostic("TS2304: Cannot find name 'x'.", 30),
        Diagnostic("Shot 1.2, frame 7%: hình Sun ra ngoài vùng an toàn (trên y=94 < 96)", 31, "layout"),
    ], [])
    assert "SỬA LỖI BIÊN DỊCH VÀ BỐ CỤC trong shot 1.2" in text
    compile_at, layout_at = text.index("Lỗi trình biên dịch TypeScript"), text.index("Lỗi bố cục")
    assert compile_at < text.index("- dòng 30: TS2304") < layout_at < text.index("- dòng 31: Shot 1.2, frame 7%")
    only_tsc = prompts.remotion_repair(sb, "const LAYOUT = {};", "1.2", tsx("1.2"), [Diagnostic("TS1005", 3)], [])
    assert "SỬA LỖI BIÊN DỊCH trong shot 1.2" in only_tsc and "Lỗi bố cục" not in only_tsc


# --- CR-050 Unit 2: segments, fingerprints, streamed calls and checks ---------------

def fps(r: CodeRequest, chunk=2) -> dict[str, str]:
    return {s.key: s.fingerprint for s in make_plan(r, chunk).segments}


def test_the_plan_cuts_a_frame_and_chunks_of_the_requested_size():
    plan = make_plan(req(storyboard(5)), 2)
    assert [(s.key, s.kind, list(s.shots)) for s in plan.segments] == [
        ("frame", "frame", []), ("1.1-1.2", "shots", ["1.1", "1.2"]),
        ("1.3-1.4", "shots", ["1.3", "1.4"]), ("1.5-1.5", "shots", ["1.5"])]


def test_fingerprints_follow_the_prompt_and_the_inputs_but_not_the_model_or_the_library():
    base = fps(req(storyboard(4)))
    other_model = req(storyboard(4))
    other_model.model = "glm"
    assert fps(other_model) == base  # CR-050 C2
    with_drawings = req(storyboard(4))
    with_drawings.illustrations = [{"name": "Cat", "usage": "", "description": "", "code": "export function Cat() {}"}]
    assert fps(with_drawings) == base  # approving a drawing keeps every chunk (review C3)
    new_prompt = req(storyboard(4))
    new_prompt.system = "SYS v2"
    assert all(fps(new_prompt)[k] != v for k, v in base.items())


def test_editing_one_shot_invalidates_only_its_chunk_when_the_layout_comes_from_the_storyboard():
    data = json.loads(storyboard_with_layout(6))
    before = fps(req(json.dumps(data)))
    data["scenes"][0]["shots"][3]["visual"] = "khác hẳn"  # shot 1.4
    after = fps(req(json.dumps(data)))
    assert {k for k in before if before[k] != after[k]} == {"1.3-1.4"}


def test_a_change_in_the_palette_keys_makes_every_chunk_stale_but_not_the_frame(monkeypatch):
    # CR-056: code written against `connguoi` must be written again, not reused
    # into a file whose PALETTE now says `conNguoi`.
    r = req(storyboard_with_layout(4))
    before = fps(r)
    real = merger.palette_keys
    monkeypatch.setattr(merger, "palette_keys", lambda sb: {k: v.lower() for k, v in real(sb).items()} | {"x": "y"})
    after = fps(r)
    assert after["frame"] == before["frame"]
    assert all(after[k] != before[k] for k in before if k != "frame")


def test_an_ai_frame_depends_on_the_whole_storyboard_so_every_chunk_follows_it():
    data = json.loads(storyboard(4))
    before = fps(req(json.dumps(data)))
    data["scenes"][0]["shots"][0]["visual"] = "khác"
    after = fps(req(json.dumps(data)))
    assert all(before[k] != after[k] for k in before)


async def test_stored_segments_with_a_matching_fingerprint_are_not_written_again():
    ev = Events()
    await pipeline(FakeProvider(), FakeChecker(), chunk=2).run(req(storyboard(4)), ev)
    r = req(storyboard(4))
    r.model = "another-model"
    r.done = ev.done()
    prov = FakeProvider()
    ev2 = Events()
    res = await pipeline(prov, FakeChecker(), chunk=2).run(r, ev2)
    assert res.status == "done" and res.check_ok and prov.calls == [] and not ev2.of("segment_done")


async def test_a_stale_stored_segment_is_written_again():
    ev = Events()
    await pipeline(FakeProvider(), FakeChecker(), chunk=2).run(req(storyboard_with_layout(4)), ev)
    done = ev.done()
    done["1.3-1.4"] = DoneSegment("an-old-fingerprint", done["1.3-1.4"].content)
    r = req(storyboard_with_layout(4))
    r.done = done
    prov = FakeProvider()
    await pipeline(prov, FakeChecker(), chunk=2).run(r, emit_none)
    assert chunk_turns(prov) == [["1.3", "1.4"]]


async def test_only_runs_the_named_segment_and_merges_when_nothing_else_is_missing():
    prov1 = Scripted(fail=lambda ids: llm_err(errors.TIMEOUT) if ids[0] in ("1.1", "1.3") else None)
    ev = Events()
    await pipeline(prov1, FakeChecker(), chunk=2).run(req(storyboard_with_layout(6)), ev)
    r = req(storyboard_with_layout(6))
    r.done, r.only = ev.done(), {"1.1-1.2"}
    prov2 = Scripted()
    res = await pipeline(prov2, FakeChecker(), chunk=2).run(r, emit_none)
    assert chunk_turns(prov2) == [["1.1", "1.2"]]
    assert res.status == "incomplete" and res.failed == [] and res.missing == ["1.3-1.4"]
    r.done, r.only = {**r.done, "1.1-1.2": DoneSegment(fps(r)["1.1-1.2"], {"shots": {
        "1.1": tsx("1.1"), "1.2": tsx("1.2")}})}, {"1.3-1.4"}
    res = await pipeline(Scripted(), FakeChecker(), chunk=2).run(r, emit_none)
    assert res.status == "done" and res.check_ok


async def test_a_failed_frame_leaves_every_chunk_missing():
    class NoLayout(FakeProvider):
        async def chat(self, req, on_progress=None):
            if "LAYOUT (bước 1/2" in req.user:
                self.calls.append(req)
                raise llm_err(errors.TIMEOUT)
            return await super().chat(req, on_progress)

    prov = NoLayout()
    ev = Events()
    res = await pipeline(prov, FakeChecker(), chunk=2).run(req(storyboard(4)), ev)
    assert res.status == "incomplete" and res.failed == ["frame"] and res.missing == ["1.1-1.2", "1.3-1.4"]
    assert chunk_turns(prov) == []


async def test_every_billed_call_is_streamed_once_with_its_segment():
    ev = Events()
    res = await pipeline(FakeProvider(broken={"1.3"}), FakeChecker(), chunk=2).run(req(storyboard(4)), ev)
    calls = ev.of("call")
    assert len(calls) == len(res.calls) and [c["label"] for c in calls] == [c.label for c in res.calls]
    assert {(c["phase"], c["segment"]) for c in calls} == {
        ("layout", "frame"), ("chunk", "1.1-1.2"), ("chunk", "1.3-1.4"), ("repair", "1.3-1.4")}
    assert all(c["usage"]["prompt_tokens"] == 10 for c in calls)


async def test_repaired_shots_overwrite_their_segment_and_every_failed_check_is_reported():
    ev = Events()
    res = await pipeline(FakeProvider(broken={"1.3"}), FakeChecker(), chunk=10).run(req(storyboard(4)), ev)
    assert res.check_ok
    [check] = ev.of("check")
    assert check["phase"] == "final" and check["round"] == 0
    [d] = check["diagnostics"]
    assert d["shot"] == "1.3" and d["segment"] == "1.1-1.4" and d["kind"] == "compile" and d["rule"] == ""
    repaired = [e for e in ev.of("segment_done") if e["repaired"]]
    assert [e["key"] for e in repaired] == ["1.1-1.4"]
    assert "BROKEN" not in repaired[0]["content"]["shots"]["1.3"] and len(repaired[0]["content"]["shots"]) == 4


async def test_early_chunk_checks_are_reported_against_their_segment():
    ev = Events()
    await pipeline(FakeProvider(broken={"1.4"}), FakeChecker(), chunk=3).run(req(storyboard(7)), ev)
    [check] = ev.of("check")
    assert check["phase"] == "chunk" and check["segment"] == "1.4-1.6"
    assert [d["shot"] for d in check["diagnostics"]] == ["1.4"]


def test_a_segment_prompt_is_what_the_pipeline_would_send():
    r = req(storyboard(4))
    r.illustrations = [{"name": "Cat", "usage": "<Cat />", "description": "mèo", "code": "export function Cat() {}"}]
    system, user = segment_prompt(r, "frame", 2)
    assert "## C4. HÌNH THƯ VIỆN" in system and "LAYOUT (bước 1/2" in user
    with pytest.raises(SegmentNotReady):
        segment_prompt(r, "1.3-1.4", 2)
    r.done = {"frame": DoneSegment(fps(r)["frame"], {"code": "const LAYOUT = {hero: {x: 1, y: 2, size: 3}};"})}
    _, user = segment_prompt(r, "1.3-1.4", 2)
    assert "VIẾT CODE CHO SHOT 1.3 → 1.4" in user and "hero: {x: 1, y: 2, size: 3}" in user
    with pytest.raises(KeyError):
        segment_prompt(r, "9.9-9.9", 2)


def test_a_pasted_reply_is_checked_and_fingerprinted():
    r = req(storyboard(4))
    fp, content = parse_segment(r, "1.1-1.2", "```tsx\n" + tsx("1.1") + "\n\n" + tsx("1.2") + "\n```", 2)
    assert fp == fps(r)["1.1-1.2"] and sorted(content["shots"]) == ["1.1", "1.2"]
    with pytest.raises(ValueError, match="missing shot function"):
        parse_segment(r, "1.1-1.2", "```tsx\n" + tsx("1.1") + "\n```", 2)
    fp, content = parse_segment(r, "frame", "const LAYOUT = {a: {x: 1}};", 2)
    assert content == {"code": "const LAYOUT = {a: {x: 1}};"}
    with pytest.raises(SegmentNotReady):
        parse_segment(req(storyboard_with_layout(2)), "frame", "const LAYOUT = {};", 2)
