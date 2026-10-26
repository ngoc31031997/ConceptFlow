import json
from datetime import UTC, datetime, timedelta

import httpx
import pytest
import respx

from app import errors
from app.errors import LLMError
from app.jsonreply import first_json_object
from app.main import create_app
from app.provider import ChatRequest, Provider
from app.registry import Providers
from tests.conftest import chunk, stream_response, usage_chunk
from tests.test_api import OkChecker, config, events

BASE = "https://hive.test/api/v3"
URL = BASE + "/chat/completions"
HIVE = "https://hive.test/v3/chat/completions"


def make(no_sleep, call_log, retries=2):
    return Provider("hive", BASE, "k", "default-model", timeout=5, max_retries=retries, sleep=no_sleep,
                    call_log=call_log)


@respx.mock
async def test_a_call_is_logged_with_its_origin_response_and_cost(no_sleep, call_log):
    respx.post(URL).mock(return_value=stream_response(
        chunk("hello", finish="stop"), usage_chunk({"prompt_tokens": 7, "completion_tokens": 3}, model="m1")))
    res = await make(no_sleep, call_log).chat(
        ChatRequest(user="SECRET PROMPT", system="SECRET SYSTEM", project_id="p1", role="story_frame",
                    step="story", phase="write"))
    [row] = call_log.rows
    assert res.call_id == row.call_id
    assert (row.project_id, row.role, row.step, row.phase) == ("p1", "story_frame", "story", "write")
    assert row.ok and row.response == "hello" and row.model == "m1" and row.provider == "hive"
    assert (row.prompt_tokens, row.completion_tokens) == (7, 3)
    assert (row.system_prompt, row.user_prompt) == ("SECRET SYSTEM", "SECRET PROMPT")
    assert row.prompt_chars == len("SECRET SYSTEM") + len("SECRET PROMPT")
    assert "SECRET" not in json.dumps(row.to_dict())
    assert row.to_dict(with_prompt=True)["system_prompt"] == "SECRET SYSTEM"


@respx.mock
async def test_the_detail_endpoint_returns_the_prompt_the_list_leaves_out(call_log):
    respx.post(HIVE).mock(return_value=reply("hi"))
    client = app_client(call_log)
    await client.post("/v1/chat", json={"system": "SYS", "user": "USR", "step": "beat"})
    [row] = (await client.get("/v1/llm/calls")).json()["calls"]
    assert "system_prompt" not in row and row["prompt_chars"] == 6
    detail = (await client.get(f"/v1/llm/calls/{row['call_id']}")).json()
    assert (detail["system_prompt"], detail["user_prompt"], detail["response"]) == ("SYS", "USR", "hi")
    assert (await client.get("/v1/llm/calls/not-a-call")).status_code == 404


@respx.mock
async def test_ollama_gets_its_context_window_and_the_log_records_it(no_sleep, call_log):
    route = respx.post(URL).mock(return_value=stream_response(chunk("ok", finish="stop"), usage_chunk({})))
    provider = Provider("ollama", BASE, "ollama", "qwen", timeout=5, max_retries=0, sleep=no_sleep,
                        call_log=call_log, num_ctx=8192)
    await provider.chat(ChatRequest(user="u"))
    assert json.loads(route.calls.last.request.content)["options"] == {"num_ctx": 8192}
    assert call_log.rows[0].num_ctx == 8192


@respx.mock
async def test_a_prompt_the_window_cannot_hold_is_cut_in_the_middle_and_logged_as_sent(no_sleep, call_log):
    route = respx.post(URL).mock(return_value=stream_response(chunk("ok", finish="stop"), usage_chunk({})))
    provider = Provider("ollama", BASE, "ollama", "qwen", timeout=5, max_retries=2, sleep=no_sleep,
                        call_log=call_log, num_ctx=1000)
    system = "ROLE " + "m" * 3000 + " TASK"
    res = await provider.chat(ChatRequest(system=system, user="go"))
    assert res.content == "ok"
    sent = json.loads(route.calls.last.request.content)["messages"][0]["content"]
    assert sent.startswith("ROLE ") and sent.endswith(" TASK") and "bị cắt" in sent
    assert len(sent) + 2 <= int(1000 * 0.85) * 2
    [row] = call_log.rows
    assert row.ok and row.system_prompt == sent and row.prompt_chars == len(sent) + 2


@respx.mock
async def test_a_prompt_that_fits_the_window_goes_out(no_sleep, call_log):
    respx.post(URL).mock(return_value=stream_response(chunk("ok", finish="stop"), usage_chunk({})))
    provider = Provider("ollama", BASE, "ollama", "qwen", timeout=5, max_retries=0, sleep=no_sleep,
                        call_log=call_log, num_ctx=1000)
    assert (await provider.chat(ChatRequest(user="y" * 1600))).content == "ok"


@respx.mock
async def test_a_failed_call_is_logged_with_its_kind_and_the_partial_text(no_sleep, call_log):
    respx.post(URL).mock(return_value=stream_response(
        chunk("half an ans"), chunk(finish="length"), usage_chunk({"prompt_tokens": 5, "completion_tokens": 9})))
    with pytest.raises(LLMError) as exc:
        await make(no_sleep, call_log).chat(ChatRequest(user="u", role="beat_sheet"))
    [row] = call_log.rows
    assert exc.value.call_id == row.call_id
    assert not row.ok and row.error_kind == errors.TRUNCATED and row.response == "half an ans"
    assert row.completion_tokens == 9


@respx.mock
async def test_retries_are_one_logged_call(no_sleep, call_log):
    respx.post(URL).mock(side_effect=[
        httpx.Response(500, json={"message": "boom"}),
        stream_response(chunk("ok", finish="stop"), usage_chunk({})),
    ])
    await make(no_sleep, call_log).chat(ChatRequest(user="u"))
    assert [r.ok for r in call_log.rows] == [True]


@respx.mock
async def test_a_log_that_cannot_be_written_does_not_fail_the_call(no_sleep, call_log, caplog):
    respx.post(URL).mock(return_value=stream_response(chunk("hi", finish="stop"), usage_chunk({})))
    call_log.fail = True
    res = await make(no_sleep, call_log).chat(ChatRequest(user="u"))
    assert res.content == "hi"
    assert "could not write llm call" in caplog.text


@pytest.mark.parametrize("text,ok", [
    ('{"a": 1}', True),
    ('```json\n{"a": 1}\n```', True),
    ('{"a": 1}\n\nGhi chú: dùng {x} và }', True),
    ('Đây là kết quả:\n{"a": {"b": 2}} xong', True),
    ("không có gì ở đây", False),
    ('{"a": 1,}', False),
    ('Ghi chú {x} rồi {"a": 1}', False),
])
def test_first_json_object(text, ok):
    obj, problem = first_json_object(text)
    assert (obj is not None) is ok
    assert (problem == "") is ok


def app_client(call_log):
    cfg = config()
    app = create_app(cfg, Providers(cfg, call_log), OkChecker(), call_log)
    return httpx.AsyncClient(transport=httpx.ASGITransport(app=app), base_url="http://svc")


def reply(text, **usage):
    return stream_response(chunk(text, finish="stop"), usage_chunk({"prompt_tokens": 1, "completion_tokens": 1, **usage}))


@respx.mock
async def test_chat_returns_the_call_id_and_logs_where_it_came_from(call_log):
    respx.post(HIVE).mock(return_value=reply("hi"))
    r = await app_client(call_log).post("/v1/chat", json={
        "user": "U", "project_id": "p9", "role": "idea", "step": "idea"})
    result = events(r)[-1]
    assert result["call_id"] == call_log.rows[0].call_id
    assert (call_log.rows[0].project_id, call_log.rows[0].role) == ("p9", "idea")
    assert "json" not in result


@respx.mock
async def test_expect_json_ignores_what_follows_the_object(call_log):
    respx.post(HIVE).mock(return_value=reply('{"title": "T"}\n\nGhi chú: có thêm } ở đây'))
    ev = events(await app_client(call_log).post("/v1/chat", json={"user": "U", "expect_json": True}))
    assert ev[-1]["type"] == "result" and ev[-1]["json"] == {"title": "T"}
    assert len(call_log.rows) == 1


@respx.mock
async def test_expect_json_asks_the_model_to_fix_an_unreadable_reply_once(call_log):
    route = respx.post(HIVE).mock(side_effect=[reply('{"title": "T",}'), reply('{"title": "T"}')])
    ev = events(await app_client(call_log).post("/v1/chat", json={
        "user": "U", "expect_json": True, "role": "story_frame"}))
    result = ev[-1]
    assert result["json"] == {"title": "T"} and result["usage"]["completion_tokens"] == 2
    assert [(r.phase, r.ok, r.role) for r in call_log.rows] == [
        ("", True, "story_frame"), ("json_repair", True, "story_frame")]
    assert result["call_id"] == call_log.rows[1].call_id
    assert '{"title": "T",}' in json.loads(route.calls.last.request.content)["messages"][-1]["content"]


@respx.mock
async def test_expect_json_that_stays_unreadable_is_a_malformed_error_with_the_call_id(call_log):
    respx.post(HIVE).mock(side_effect=[reply("không phải json"), reply("vẫn không phải json")])
    ev = events(await app_client(call_log).post("/v1/chat", json={"user": "U", "expect_json": True}))
    err = ev[-1]["error"]
    assert ev[-1]["type"] == "error" and err["kind"] == errors.MALFORMED
    assert err["call_id"] == call_log.rows[1].call_id and err["partial"] == "vẫn không phải json"
    assert err["usage"]["completion_tokens"] == 2


@respx.mock
async def test_stats_calls_and_deletes(call_log):
    respx.post(HIVE).mock(return_value=reply("hi"))
    client = app_client(call_log)
    for _ in range(2):
        await client.post("/v1/chat", json={"user": "U", "step": "code", "phase": "chunk", "project_id": "p1"})
    await client.post("/v1/chat", json={"user": "U", "step": "story", "project_id": "p2"})

    stats = (await client.get("/v1/llm/stats", params={"step": "code", "phase": "chunk"})).json()
    [model] = stats.values()
    assert model["calls"] == 2 and model["ok"] == 2 and model["avg_completion_tokens"] == 1

    page = (await client.get("/v1/llm/calls", params={"limit": 2})).json()
    assert len(page["calls"]) == 2 and page["next_before"] is not None
    only = (await client.get("/v1/llm/calls", params={"project_id": "p2"})).json()
    assert [c["step"] for c in only["calls"]] == ["story"] and only["next_before"] is None

    ids = [c["call_id"] for c in only["calls"]]
    assert (await client.post("/v1/llm/calls/delete", json={"ids": ids})).json() == {"deleted": 1}
    call_log.rows[0].created_at = datetime.now(UTC) - timedelta(days=31)
    assert (await client.post("/v1/llm/calls/delete", json={"older_than_days": 30})).json() == {"deleted": 1}
    assert (await client.post("/v1/llm/calls/delete", json={"all": True})).json() == {"deleted": 1}
    assert call_log.rows == []


@pytest.mark.parametrize("body", [{}, {"ids": [], "all": True}, {"all": False}, {"older_than_days": -1}])
async def test_delete_needs_exactly_one_clear_choice(call_log, body):
    r = await app_client(call_log).post("/v1/llm/calls/delete", json=body)
    assert r.status_code == 422


def test_the_database_url_is_required(monkeypatch):
    from app.config import Config

    monkeypatch.delenv("LLM_DATABASE_URL")
    with pytest.raises(ValueError, match="LLM_DATABASE_URL"):
        Config.from_env()


async def test_a_pasted_reply_is_logged_as_an_external_call_and_its_json_is_read(call_log):
    r = await app_client(call_log).post("/v1/chat/external", json={
        "system": "S", "user": "U", "reply": 'Đây:\n```json\n{"a": 1}\n```', "expect_json": True,
        "project_id": "p1", "role": "story_frame", "step": "story"})
    body = r.json()
    assert r.status_code == 200 and body["json"] == {"a": 1} and body["provider"] == "external"
    [row] = call_log.rows
    assert (row.call_id, row.provider, row.ok, row.system_prompt, row.user_prompt) == (
        body["call_id"], "external", True, "S", "U")
    assert row.role == "story_frame" and row.project_id == "p1" and not row.usage_reported


async def test_a_pasted_reply_without_json_is_a_malformed_error_and_still_logged(call_log):
    r = await app_client(call_log).post("/v1/chat/external", json={"user": "U", "reply": "xin chào", "expect_json": True})
    err = r.json()["error"]
    assert r.status_code == 422 and err["kind"] == "malformed" and err["call_id"]
    [row] = call_log.rows
    assert not row.ok and row.error_kind == "malformed" and row.response == "xin chào"


async def test_a_pasted_reply_without_expect_json_comes_back_as_text(call_log):
    body = (await app_client(call_log).post("/v1/chat/external", json={"user": "U", "reply": "văn bản"})).json()
    assert body["content"] == "văn bản" and "json" not in body
