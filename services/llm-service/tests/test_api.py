import json

import httpx
import pytest
import respx

from app.config import Config
from app.main import create_app
from app.pipeline.checker import CheckResult
from app.registry import Providers
from tests.conftest import chunk, stream_response, usage_chunk
from tests.test_pipeline import storyboard, storyboard_with_layout


def config(**over):
    base = dict(
        hive_api_key="k", hive_base_url="https://hive.test/v3", hive_model="deepseek", hive_timeout=5,
        hive_max_retries=0, hive_rate_per_second=1000, ollama_url="http://ollama.test:11434",
        ollama_model="llama", ollama_timeout=5, light_provider="ollama", code_chunk_shots=10,
        code_chunk_concurrency=2, code_repair_max_rounds=1, rendering_url="http://rendering.test", rendering_check_timeout=5,
        code_max_reasoning_chars=60000, chat_max_reasoning_chars=0)
    base.update(over)
    return Config(**base)


class OkChecker:
    def __init__(self, warnings=None):
        self.layouts = []
        self.warnings = warnings or []

    async def check(self, engine, code, scene_class_name, layout=None):
        self.layouts.append(layout)
        return CheckResult(ok=True, warnings=list(self.warnings))


@pytest.fixture
def client():
    cfg = config()
    app = create_app(cfg, Providers(cfg), OkChecker())
    return httpx.AsyncClient(transport=httpx.ASGITransport(app=app), base_url="http://svc")


def events(resp):
    return [json.loads(ln) for ln in resp.text.splitlines() if ln.strip()]


async def test_health(client):
    r = await client.get("/health")
    assert r.json() == {"status": "ok", "hive_configured": True, "light_provider": "ollama"}


@respx.mock
async def test_chat_streams_progress_then_a_single_result(client):
    respx.post("https://hive.test/v3/chat/completions").mock(return_value=stream_response(
        chunk(reasoning="abc"), chunk("hi", finish="stop"), usage_chunk({"prompt_tokens": 3, "completion_tokens": 4})))
    r = await client.post("/v1/chat", json={"system": "S", "user": "U"})
    ev = events(r)
    assert [e["type"] for e in ev] == ["progress", "progress", "result"]
    assert ev[-1]["content"] == "hi" and ev[-1]["usage"]["completion_tokens"] == 4 and ev[-1]["provider"] == "hive"


@respx.mock
async def test_chat_reports_the_error_kind_and_billed_usage(client):
    respx.post("https://hive.test/v3/chat/completions").mock(return_value=httpx.Response(405, json={"message": "no credit"}))
    ev = events(await client.post("/v1/chat", json={"user": "U"}))
    assert ev[-1]["type"] == "error" and ev[-1]["error"]["kind"] == "balance"


async def test_unknown_provider_is_a_400(client):
    r = await client.post("/v1/chat", json={"user": "U", "provider": "nope"})
    assert r.status_code == 400


@respx.mock
async def test_suggest_metadata_goes_to_the_light_provider_ollama(client):
    route = respx.post("http://ollama.test:11434/v1/chat/completions").mock(return_value=stream_response(
        chunk('{"title":"T","description":"d","tags":["a"]}', finish="stop"), usage_chunk({})))
    r = await client.post("/v1/suggest-metadata", json={"script_content": "s", "category_hint": "c", "language": "vi"})
    assert r.status_code == 200 and r.json()["title"] == "T" and r.json()["provider"] == "ollama"
    assert route.called


@respx.mock
async def test_suggest_metadata_can_stream_progress(client):
    respx.post("http://ollama.test:11434/v1/chat/completions").mock(return_value=stream_response(
        chunk('{"title":"T","description":"d","tags":["a"]}', finish="stop"), usage_chunk({})))
    r = await client.post("/v1/suggest-metadata", json={"script_content": "s", "language": "vi", "stream": True})
    ev = events(r)
    assert any(e["type"] == "progress" and e["content_chars"] > 0 for e in ev)
    assert ev[-1]["type"] == "result" and ev[-1]["title"] == "T"


@respx.mock
async def test_suggest_stream_reports_the_error_kind(client):
    respx.post("http://ollama.test:11434/v1/chat/completions").mock(return_value=httpx.Response(500, json={}))
    r = await client.post("/v1/suggest-metadata", json={"script_content": "s", "stream": True})
    ev = events(r)
    assert ev[-1]["type"] == "error" and ev[-1]["error"]["kind"] == "server"


@respx.mock
async def test_suggest_error_is_a_502_with_the_kind(client):
    respx.post("http://ollama.test:11434/v1/chat/completions").mock(return_value=httpx.Response(500, json={}))
    r = await client.post("/v1/suggest-metadata", json={"script_content": "s"})
    assert r.status_code == 502 and r.json()["error"]["kind"] == "server"


@respx.mock
async def test_storyboard_finalize_accepts_valid_json_without_calling_the_model(client):
    route = respx.post("https://hive.test/v3/chat/completions")
    r = await client.post("/v1/storyboard/finalize", json={"content": storyboard(2)})
    assert r.status_code == 200 and r.json()["shots"] == 2 and not r.json()["repaired"] and not route.called


@respx.mock
async def test_storyboard_finalize_repairs_once_and_bills_it(client):
    respx.post("https://hive.test/v3/chat/completions").mock(return_value=stream_response(
        chunk(storyboard(1), finish="stop"), usage_chunk({"prompt_tokens": 7, "completion_tokens": 9})))
    r = await client.post("/v1/storyboard/finalize", json={"content": "this is prose"})
    assert r.status_code == 200 and r.json()["repaired"] and r.json()["usage"]["prompt_tokens"] == 7


@respx.mock
async def test_storyboard_finalize_fails_with_422_when_repair_is_still_invalid(client):
    respx.post("https://hive.test/v3/chat/completions").mock(return_value=stream_response(
        chunk("still prose", finish="stop"), usage_chunk({"prompt_tokens": 1, "completion_tokens": 1})))
    r = await client.post("/v1/storyboard/finalize", json={"content": "prose"})
    assert r.status_code == 422 and r.json()["error"]["problems"]


async def test_a_frame_that_is_neither_landscape_nor_portrait_is_refused(client):
    r = await client.post("/v1/storyboard/finalize", json={
        "content": storyboard(1), "frame": {"width": 1280, "height": 720}})
    assert r.status_code == 422
    r = await client.post("/v2/code/plan", json={
        "engine": "remotion", "storyboard": storyboard(1), "system": "S", "frame": {"width": 1080, "height": 1920}})
    assert r.status_code == 200


async def test_code_generate_streams_phases_and_the_final_code(client):
    with respx.mock:
        respx.post("https://hive.test/v3/chat/completions").mock(side_effect=lambda req: _fake_hive(req))
        r = await client.post("/v2/code/generate", json={
            "engine": "remotion", "topic": "t", "storyboard": storyboard(2), "system": "SYS"})
    ev = events(r)
    assert ev[-1]["type"] == "result" and ev[-1]["check_ok"] is True
    assert {"layout", "chunks", "merge", "check"} <= {e["phase"] for e in ev if e["type"] == "phase"}
    assert ev[-1]["status"] == "done" and "const SHOTS" in ev[-1]["code"]
    # Each billed call is its own event, not a list at the end
    assert [e["phase"] for e in ev if e["type"] == "call"] == ["layout", "chunk"] and "calls" not in ev[-1]
    assert [e["key"] for e in ev if e["type"] == "segment_done"] == ["frame", "1.1-1.2"]
    assert ev[0]["type"] == "plan" and [s["key"] for s in ev[0]["segments"]] == ["frame", "1.1-1.2"]


async def test_the_v1_code_route_is_gone(client):
    # The code step is /v2 only.
    r = await client.post("/v1/code/generate", json={
        "engine": "remotion", "topic": "t", "storyboard": storyboard(2), "system": "SYS"})
    assert r.status_code == 404


async def test_code_plan_is_the_cut_without_a_model_call(client):
    with respx.mock:
        route = respx.post("https://hive.test/v3/chat/completions")
        r = await client.post("/v2/code/plan", json={
            "engine": "remotion", "storyboard": storyboard_with_layout(5), "system": "", "chunk_shots": 2})
    assert r.status_code == 200 and route.call_count == 0
    segs = r.json()["segments"]
    assert [(s["key"], s["kind"], s["shots"]) for s in segs] == [
        ("frame", "frame", []), ("1.1-1.2", "shots", ["1.1", "1.2"]), ("1.3-1.4", "shots", ["1.3", "1.4"]),
        ("1.5-1.5", "shots", ["1.5"])]
    assert segs[0]["source"] == "storyboard" and "source" not in segs[1]
    manim = await client.post("/v2/code/plan", json={"engine": "manim", "storyboard": storyboard(2), "system": ""})
    assert "source" not in manim.json()["segments"][0]
    bad = await client.post("/v2/code/plan", json={"engine": "remotion", "storyboard": "CẢNH 1", "system": ""})
    assert bad.status_code == 422


async def test_code_generate_runs_only_what_is_missing(client):
    with respx.mock:
        respx.post("https://hive.test/v3/chat/completions").mock(side_effect=lambda req: _fake_hive(req))
        first = events(await client.post("/v2/code/generate", json={
            "engine": "remotion", "topic": "t", "storyboard": storyboard(2), "system": "SYS"}))
        stored = [{"key": e["key"], "fingerprint": e["fingerprint"], "content": e["content"]}
                  for e in first if e["type"] == "segment_done"]
        route = respx.post("https://hive.test/v3/chat/completions").mock(side_effect=lambda req: _fake_hive(req))
        before = route.call_count
        again = events(await client.post("/v2/code/generate", json={
            "engine": "remotion", "topic": "t", "storyboard": storyboard(2), "system": "SYS",
            "model": "another", "segments": stored}))
    assert again[-1]["status"] == "done" and route.call_count == before


async def test_segment_prompt_and_parse(client):
    base = {"engine": "remotion", "topic": "t", "storyboard": storyboard(2), "system": "SYS", "chunk_shots": 1}
    r = await client.post("/v2/code/segment-prompt", json={**base, "key": "frame"})
    assert r.status_code == 200 and r.json()["system"] == "SYS" and "LAYOUT (bước 1/2" in r.json()["user"]
    r = await client.post("/v2/code/segment-prompt", json={**base, "key": "1.2-1.2"})
    assert r.status_code == 409  # no frame yet
    r = await client.post("/v2/code/segment-prompt", json={**base, "key": "7.7-7.7"})
    assert r.status_code == 404
    r = await client.post("/v2/code/segment-parse", json={**base, "key": "frame", "reply": "prose"})
    assert r.status_code == 422 and "const LAYOUT" in r.json()["error"]["message"]
    r = await client.post("/v2/code/segment-parse", json={**base, "key": "frame", "reply": "const LAYOUT = {};"})
    assert r.status_code == 200 and r.json()["content"] == {"code": "const LAYOUT = {};"}
    fp = r.json()["fingerprint"]
    r = await client.post("/v2/code/segment-prompt", json={
        **base, "key": "1.2-1.2", "segments": [{"key": "frame", "fingerprint": fp, "content": {"code": "const LAYOUT = {};"}}]})
    assert r.status_code == 200 and "VIẾT CODE CHO SHOT 1.2 → 1.2" in r.json()["user"]


async def test_code_generate_hands_the_subtitle_band_and_font_to_the_layout_check_and_returns_its_warnings():
    cfg = config()
    checker = OkChecker(warnings=["Bố cục: Shot 1.1: vật lớn nhất (hình Apple) chỉ chiếm 19% chiều khung"])
    app = create_app(cfg, Providers(cfg), checker)
    c = httpx.AsyncClient(transport=httpx.ASGITransport(app=app), base_url="http://svc")
    with respx.mock:
        respx.post("https://hive.test/v3/chat/completions").mock(side_effect=lambda req: _fake_hive(req))
        r = await c.post("/v2/code/generate", json={
            "engine": "remotion", "topic": "t", "storyboard": storyboard(2), "system": "SYS",
            "subtitle_band": {"edge": "bottom", "px": 240}, "video_font": "Montserrat"})
    ev = events(r)
    assert ev[-1]["type"] == "result" and ev[-1]["check_ok"] is True
    assert ev[-1]["warnings"] == ["Bố cục: Shot 1.1: vật lớn nhất (hình Apple) chỉ chiếm 19% chiều khung"]
    assert checker.layouts and all(
        lc.subtitle_band == {"edge": "bottom", "px": 240} and lc.video_font == "Montserrat" for lc in checker.layouts)


async def test_code_generate_rejects_a_nonsense_subtitle_band(client):
    r = await client.post("/v2/code/generate", json={
        "engine": "remotion", "topic": "t", "storyboard": storyboard(2), "system": "SYS",
        "subtitle_band": {"edge": "left", "px": 240}})
    assert r.status_code == 422


async def test_code_generate_with_a_prose_storyboard_is_an_error_event(client):
    r = await client.post("/v2/code/generate", json={
        "engine": "remotion", "topic": "t", "storyboard": "CẢNH 1", "system": "SYS"})
    ev = events(r)
    assert ev[-1]["type"] == "error" and "step 1b with AI" in ev[-1]["error"]["message"]


def _only_reasoning(chars: int):
    return stream_response(*[chunk(reasoning="r" * 1000) for _ in range(chars // 1000)],
                           chunk("", finish="length"), usage_chunk({"completion_tokens": 50}))


def _client(cfg):
    return httpx.AsyncClient(
        transport=httpx.ASGITransport(app=create_app(cfg, Providers(cfg), OkChecker())), base_url="http://svc")


async def test_code_generate_stops_a_call_that_only_reasons_past_the_code_limit():
    with respx.mock:
        route = respx.post("https://hive.test/v3/chat/completions").mock(return_value=_only_reasoning(5000))
        r = await _client(config(code_max_reasoning_chars=2000)).post("/v2/code/generate", json={
            "engine": "remotion", "topic": "t", "storyboard": storyboard(2), "system": "SYS"})
    ev = events(r)
    # A failed segment is reported and the run ends incomplete, not as an error.
    assert ev[-1]["type"] == "result" and ev[-1]["status"] == "incomplete" and ev[-1]["failed"] == ["frame"]
    [failed] = [e for e in ev if e["type"] == "segment_failed"]
    assert failed["error"]["kind"] == "budget" and "suy nghĩ quá 2000 ký tự" in failed["error"]["message"]
    [call] = [e for e in ev if e["type"] == "call"]
    assert call["phase"] == "layout" and call["error_kind"] == "budget"
    # The stream was cut, so no usage came back; what was counted is sent
    assert call["usage"]["usage_reported"] is False and call["usage"]["reasoning_chars"] > 2000
    assert route.call_count == 1


@respx.mock
async def test_chat_uses_its_own_limit_off_by_default():
    respx.post("https://hive.test/v3/chat/completions").mock(side_effect=lambda _req: stream_response(
        *[chunk(reasoning="r" * 1000) for _ in range(5)], chunk("hi", finish="stop"), usage_chunk({})))
    # the code limit does not apply to /v1/chat
    ev = events(await _client(config(code_max_reasoning_chars=2000)).post("/v1/chat", json={"user": "U"}))
    assert ev[-1]["type"] == "result" and ev[-1]["content"] == "hi"

    respx.post("https://hive.test/v3/chat/completions").mock(return_value=_only_reasoning(5000))
    ev = events(await _client(config(chat_max_reasoning_chars=2000)).post("/v1/chat", json={"user": "U"}))
    assert ev[-1]["type"] == "error" and ev[-1]["error"]["kind"] == "budget"
    assert "suy nghĩ quá 2000 ký tự" in ev[-1]["error"]["message"]


def test_reasoning_limits_from_env(monkeypatch):
    from app.config import Config

    monkeypatch.delenv("CODE_MAX_REASONING_CHARS", raising=False)
    monkeypatch.delenv("CHAT_MAX_REASONING_CHARS", raising=False)
    cfg = Config.from_env()
    assert (cfg.code_max_reasoning_chars, cfg.chat_max_reasoning_chars) == (60000, 0)

    monkeypatch.setenv("CODE_MAX_REASONING_CHARS", "0")
    monkeypatch.setenv("CHAT_MAX_REASONING_CHARS", "90000")
    cfg = Config.from_env()
    assert (cfg.code_max_reasoning_chars, cfg.chat_max_reasoning_chars) == (0, 90000)

    monkeypatch.setenv("CODE_MAX_REASONING_CHARS", "-1")
    with pytest.raises(ValueError, match="CODE_MAX_REASONING_CHARS"):
        Config.from_env()


def _fake_hive(req):
    user = json.loads(req.content)["messages"][-1]["content"]
    if "LAYOUT (bước 1/2" in user:
        text = "const LAYOUT = {hero: {x: 1, y: 2, size: 3}};"
    else:
        text = "\n\n".join(
            f"function Shot1_{i}({{duration}}: ShotProps) {{\n  return null;\n}}" for i in (1, 2))
    return stream_response(chunk(text, finish="stop"), usage_chunk({"prompt_tokens": 1, "completion_tokens": 1}))


def test_for_model_routes_ollama_ids():
    from app.config import Config
    from app.registry import Providers

    cfg = Config.from_env()
    p = Providers(cfg)
    assert p.for_model(p.hive, "ollama") == (p.ollama, cfg.ollama_model)
    assert p.for_model(p.hive, "ollama/qwen2.5") == (p.ollama, "qwen2.5")
    assert p.for_model(p.hive, "zai-org/glm-5.3-flash") == (p.hive, "zai-org/glm-5.3-flash")


@respx.mock
async def test_rendering_checker_sends_the_layout_context_and_reads_kinds_and_warnings():
    from app.pipeline.checker import LayoutContext, RenderingChecker

    route = respx.post("http://rendering.test/v1/check/remotion").mock(return_value=httpx.Response(200, json={
        "ok": False, "raw": "",
        "diagnostics": [{"message": "TS2304: x", "line": 3, "kind": "compile"},
                        {"message": "Shot 1.2, frame 85%: nhãn tràn", "line": 62, "kind": "layout", "rule": "text_overflow"},
                        {"message": "old rendering, no kind", "line": 4}],
        "warnings": ["Bố cục: Shot 1.3: vật lớn nhất nhỏ"]}))
    chk = RenderingChecker("http://rendering.test", 5)
    res = await chk.check("remotion", "code", "creator",
                          LayoutContext(subtitle_band={"edge": "top", "px": 200}, video_font="Montserrat"))
    assert json.loads(route.calls[0].request.content) == {
        "code": "code", "scene_class_name": "creator",
        "subtitle_band": {"edge": "top", "px": 200}, "video_font": "Montserrat"}
    assert [(d.line, d.kind) for d in res.diagnostics] == [(3, "compile"), (62, "layout"), (4, "compile")]
    assert [d.rule for d in res.diagnostics] == ["", "text_overflow", ""]
    assert res.warnings == ["Bố cục: Shot 1.3: vật lớn nhất nhỏ"]

    manim = respx.post("http://rendering.test/v1/check/manim").mock(return_value=httpx.Response(200, json={
        "ok": True, "diagnostics": [], "raw": ""}))
    await chk.check("remotion", "code", "creator", LayoutContext())
    res = await chk.check("manim", "code", "XScene")
    assert json.loads(route.calls[1].request.content) == {"code": "code", "scene_class_name": "creator"}
    assert json.loads(manim.calls[0].request.content) == {"code": "code", "scene_class_name": "XScene"}
    assert res.ok and res.warnings == []  # an answer without warnings (older rendering) reads as none
