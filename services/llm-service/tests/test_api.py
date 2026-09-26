import json

import httpx
import pytest
import respx

from app.config import Config
from app.main import create_app
from app.pipeline.checker import CheckResult
from app.registry import Providers
from tests.conftest import chunk, stream_response, usage_chunk
from tests.test_pipeline import storyboard


def config(**over):
    base = dict(
        hive_api_key="k", hive_base_url="https://hive.test/v3", hive_model="deepseek", hive_timeout=5,
        hive_max_retries=0, hive_rate_per_second=1000, ollama_url="http://ollama.test:11434",
        ollama_model="llama", ollama_timeout=5, light_provider="ollama", code_chunk_shots=10,
        code_chunk_concurrency=2, code_repair_max_rounds=1, rendering_url="http://rendering.test", rendering_check_timeout=5)
    base.update(over)
    return Config(**base)


class OkChecker:
    async def check(self, engine, code, scene_class_name):
        return CheckResult(ok=True)


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
    r = await client.post("/v1/suggest-short-script", json={"topic": "t", "stream": True})
    ev = events(r)
    assert ev[-1]["type"] == "error" and ev[-1]["error"]["kind"] == "server"


@respx.mock
async def test_suggest_error_is_a_502_with_the_kind(client):
    respx.post("http://ollama.test:11434/v1/chat/completions").mock(return_value=httpx.Response(500, json={}))
    r = await client.post("/v1/suggest-short-script", json={"topic": "t"})
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


async def test_code_generate_streams_phases_and_the_final_code(client):
    with respx.mock:
        respx.post("https://hive.test/v3/chat/completions").mock(side_effect=lambda req: _fake_hive(req))
        r = await client.post("/v1/code/generate", json={
            "engine": "remotion", "topic": "t", "storyboard": storyboard(2), "system": "SYS"})
    ev = events(r)
    assert ev[-1]["type"] == "result" and ev[-1]["check_ok"] is True
    assert {"layout", "chunks", "merge", "check"} <= {e["phase"] for e in ev if e["type"] == "phase"}
    assert "const SHOTS" in ev[-1]["code"] and len(ev[-1]["calls"]) == 2


async def test_code_generate_with_a_prose_storyboard_is_an_error_event(client):
    r = await client.post("/v1/code/generate", json={
        "engine": "remotion", "topic": "t", "storyboard": "CẢNH 1", "system": "SYS"})
    ev = events(r)
    assert ev[-1]["type"] == "error" and "step 1b with AI" in ev[-1]["error"]["message"]


def _fake_hive(req):
    user = json.loads(req.content)["messages"][-1]["content"]
    if "LAYOUT (bước 1/2" in user:
        text = "const LAYOUT = {hero: {x: 1, y: 2, size: 3}};"
    else:
        text = "\n\n".join(
            f"function Shot1_{i}({{duration}}: ShotProps) {{\n  return null;\n}}" for i in (1, 2))
    return stream_response(chunk(text, finish="stop"), usage_chunk({"prompt_tokens": 1, "completion_tokens": 1}))
