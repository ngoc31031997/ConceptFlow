import asyncio
import json
import re

import pytest

from app import errors
from app.errors import LLMError, Usage
from app.pipeline.checker import CheckResult, Diagnostic
from app.pipeline.run import ChunkCache, CodePipeline, CodeRequest, PipelineFailure
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
        self.raw_for_manim = raw_for_manim

    async def check(self, engine, code, scene_class_name):
        self.codes.append(code)
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


def pipeline(provider, checker, chunk=10, rounds=3, cache=None):
    return CodePipeline(provider, checker, chunk_shots=chunk, concurrency=3, repair_rounds=rounds, cache=cache)


def req(sb, engine="remotion"):
    return CodeRequest(engine=engine, topic="Chủ đề", storyboard=sb, system="SYS")


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


async def test_failed_run_keeps_finished_chunks_in_cache_and_a_rerun_does_not_pay_again():
    cache = ChunkCache()
    prov1 = FakeProvider(broken={"1.1"}, fail_repair=True)
    res1 = await pipeline(prov1, FakeChecker(), chunk=1, rounds=1, cache=cache).run(req(storyboard(3)), emit_none)
    assert not res1.check_ok
    prov2 = FakeProvider()
    res2 = await pipeline(prov2, FakeChecker(), chunk=1, rounds=1, cache=cache).run(req(storyboard(3)), emit_none)
    # layout + 3 chunks come from the cache (free); the cached 1.1 is still broken,
    # so the re-run pays for exactly one thing: repairing it.
    assert sum(1 for c in res2.calls if c.cached) == 4
    assert res2.check_ok and res2.repair_rounds == 1
    paid = [c for c in res2.calls if not c.cached]
    assert [c.phase for c in paid] == ["repair"]
    assert [r for r in prov2.calls if "VIẾT CODE CHO SHOT" in r.user or "LAYOUT (bước 1/2" in r.user] == []


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
    assert not any(ev.get("phase") == "repair" for ev in events)  # nothing was left for the final loop
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
        async def check(self, engine, code, scene_class_name):
            if "return null;\n}" in code and code.count("function Shot") != code.count("// Shot"):
                raise CheckerUnavailable("rendering is restarting")  # only the stubbed per-chunk files
            return await super().check(engine, code, scene_class_name)

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


async def test_a_failing_chunk_lets_running_chunks_finish_into_the_cache_and_a_rerun_pays_only_for_it():
    cache = ChunkCache()
    others_in, release = asyncio.Event(), asyncio.Event()
    arrived = []

    def gate(ids):
        if ids == ["1.2"]:
            return others_in  # fails only once chunks 1 and 3 are talking to the model
        arrived.append(ids)
        if len(arrived) == 2:
            others_in.set()
        return release

    def fail(ids):
        if ids == ["1.2"]:
            release.set()  # chunks 1 and 3 are still waiting on the model when this one fails
            return llm_err(errors.SERVER)

    prov1 = Scripted(fail=fail, gate=gate)
    with pytest.raises(PipelineFailure) as e:
        await pipeline(prov1, FakeChecker(), chunk=1, cache=cache).run(req(storyboard_with_layout(3)), emit_none)
    assert e.value.kind == errors.SERVER
    assert sorted(prov1.finished) == [["1.1"], ["1.3"]]  # chunks 1 and 3 were not thrown away
    by_label = {c.label: c for c in e.value.calls}
    assert by_label["1.1-1.1"].ok and by_label["1.3-1.3"].ok  # ...and their cost is reported
    assert not by_label["1.2-1.2"].ok and by_label["1.2-1.2"].error_kind == errors.SERVER

    prov2 = Scripted()
    res = await pipeline(prov2, FakeChecker(), chunk=1, cache=cache).run(req(storyboard_with_layout(3)), emit_none)
    assert res.check_ok
    assert chunk_turns(prov2) == [["1.2"]]  # the re-run calls the model for chunk 2 only
    assert sorted(c.label for c in res.calls if c.cached) == ["1.1-1.1", "1.3-1.3"]


async def test_chunks_still_waiting_for_a_slot_do_not_start_after_a_failure():
    release = asyncio.Event()

    def fail(ids):
        if ids == ["1.2"]:
            release.set()
            return llm_err(errors.SERVER)

    prov = Scripted(fail=fail, gate=lambda ids: release if ids == ["1.1"] else None)
    p = CodePipeline(prov, FakeChecker(), chunk_shots=1, concurrency=2, repair_rounds=1)
    events = []

    async def emit(ev):
        events.append(ev)

    with pytest.raises(PipelineFailure):
        await p.run(req(storyboard_with_layout(4)), emit)
    assert sorted(chunk_turns(prov)) == [["1.1"], ["1.2"]]  # 1.3 and 1.4 never reached the model
    assert prov.finished == [["1.1"]]
    assert sorted(ev["index"] for ev in events if ev["type"] == "chunk_start") == [1, 2]


async def test_the_first_failure_is_raised_even_when_a_later_chunk_also_fails():
    second_in, release = asyncio.Event(), asyncio.Event()

    def gate(ids):
        if ids == ["1.1"]:
            return second_in
        second_in.set()
        return release

    def fail(ids):
        if ids == ["1.1"]:
            release.set()
            return llm_err(errors.SERVER)
        if ids == ["1.2"]:
            return llm_err(errors.TIMEOUT)

    prov = Scripted(fail=fail, gate=gate)
    with pytest.raises(PipelineFailure) as e:
        await pipeline(prov, FakeChecker(), chunk=1).run(req(storyboard_with_layout(2)), emit_none)
    assert e.value.kind == errors.SERVER
    assert sorted(c.error_kind for c in e.value.calls) == [errors.SERVER, errors.TIMEOUT]


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


async def test_a_one_shot_chunk_over_budget_fails_the_run_with_budget():
    prov = Scripted(fail=lambda ids: llm_err(errors.BUDGET) if "1.2" in ids else None)
    with pytest.raises(PipelineFailure) as e:
        await pipeline(prov, FakeChecker(), chunk=2).run(req(storyboard_with_layout(2)), emit_none)
    assert e.value.kind == errors.BUDGET
    assert [(c.label, c.ok) for c in e.value.calls] == [("1.1-1.2", False), ("1.1-1.1", True), ("1.2-1.2", False)]


async def test_other_error_kinds_are_not_split():
    prov = Scripted(fail=lambda ids: llm_err(errors.EMPTY))
    with pytest.raises(PipelineFailure) as e:
        await pipeline(prov, FakeChecker(), chunk=4).run(req(storyboard_with_layout(4)), emit_none)
    assert e.value.kind == errors.EMPTY and chunk_turns(prov) == [["1.1", "1.2", "1.3", "1.4"]]


async def test_a_rerun_goes_straight_to_the_halves_of_a_chunk_that_was_over_budget():
    cache = ChunkCache()
    prov1 = Scripted(fail=lambda ids: llm_err(errors.BUDGET) if len(ids) == 4 or ids == ["1.5"] else None)
    with pytest.raises(PipelineFailure):
        await pipeline(prov1, FakeChecker(), chunk=4, cache=cache).run(req(storyboard_with_layout(5)), emit_none)
    prov2 = Scripted()
    res = await pipeline(prov2, FakeChecker(), chunk=4, cache=cache).run(req(storyboard_with_layout(5)), emit_none)
    assert res.check_ok
    assert chunk_turns(prov2) == [["1.5"]]  # neither the 4-shot chunk nor its cached halves are paid again
    # a fully successful run drops the split marker with the rest of its cache
    prov3 = Scripted()
    await pipeline(prov3, FakeChecker(), chunk=4, cache=cache).run(req(storyboard_with_layout(5)), emit_none)
    assert sorted(chunk_turns(prov3)) == [["1.1", "1.2", "1.3", "1.4"], ["1.5"]]


async def test_a_dead_key_cancels_the_other_chunks_at_once():
    never = asyncio.Event()
    prov = Scripted(fail=lambda ids: llm_err(errors.AUTH) if ids == ["1.2"] else None,
                    gate=lambda ids: never if ids != ["1.2"] else None)
    with pytest.raises(PipelineFailure) as e:
        await asyncio.wait_for(
            pipeline(prov, FakeChecker(), chunk=1).run(req(storyboard_with_layout(3)), emit_none), timeout=2)
    assert e.value.kind == errors.AUTH
    assert prov.finished == [] and [c.label for c in e.value.calls] == ["1.2-1.2"]


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
