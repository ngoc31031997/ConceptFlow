import json

import httpx
import pytest
import respx

from app import errors
from app.errors import LLMError
from app.provider import ChatRequest, Provider
from tests.conftest import chunk, stream_response, usage_chunk

BASE = "https://hive.test/api/v3"
URL = BASE + "/chat/completions"


def make(no_sleep, key="k", retries=2, **kw):
    return Provider("hive", BASE, key, "default-model", timeout=5, max_retries=retries, sleep=no_sleep, **kw)


@respx.mock
async def test_sends_bearer_key_stream_and_both_messages(no_sleep):
    route = respx.post(URL).mock(return_value=stream_response(chunk("hi", finish="stop"), usage_chunk({})))
    res = await make(no_sleep).chat(ChatRequest(system="SYS", user="USR", max_tokens=99))
    req = route.calls.last.request
    body = json.loads(req.content)
    assert req.headers["authorization"] == "Bearer k"
    assert req.headers["accept"] == "text/event-stream"
    assert body["stream"] is True and body["max_tokens"] == 99
    assert [m["role"] for m in body["messages"]] == ["system", "user"]
    assert res.content == "hi"


@respx.mock
async def test_omits_system_message_and_max_tokens_when_unset(no_sleep):
    route = respx.post(URL).mock(return_value=stream_response(chunk("hi", finish="stop"), usage_chunk({})))
    await make(no_sleep).chat(ChatRequest(user="only user"))
    body = json.loads(route.calls.last.request.content)
    assert [m["role"] for m in body["messages"]] == ["user"]
    assert "max_tokens" not in body


@respx.mock
async def test_per_call_model_overrides_default(no_sleep):
    route = respx.post(URL).mock(return_value=stream_response(chunk("hi", finish="stop"), usage_chunk({})))
    await make(no_sleep).chat(ChatRequest(user="u", model="zai-org/glm-5.3-flash"))
    assert json.loads(route.calls.last.request.content)["model"] == "zai-org/glm-5.3-flash"


@respx.mock
async def test_json_mode_sets_response_format(no_sleep):
    route = respx.post(URL).mock(return_value=stream_response(chunk("{}", finish="stop"), usage_chunk({})))
    await make(no_sleep).chat(ChatRequest(user="u", json_mode=True))
    assert json.loads(route.calls.last.request.content)["response_format"] == {"type": "json_object"}


@respx.mock
async def test_reads_both_usage_shapes(no_sleep):
    glm = {"prompt_tokens": 10, "completion_tokens": 20, "reasoning_tokens": 7, "prompt_tokens_details": None}
    respx.post(URL).mock(return_value=stream_response(chunk("a", finish="stop"), usage_chunk(glm)))
    r = await make(no_sleep).chat(ChatRequest(user="u"))
    assert (r.usage.prompt_tokens, r.usage.completion_tokens, r.usage.reasoning_tokens, r.usage.cached_tokens) == (10, 20, 7, 0)

    ds = {"prompt_tokens": 30, "completion_tokens": 40, "completion_tokens_details": {"reasoning_tokens": 9},
          "prompt_tokens_details": {"cached_tokens": 12}}
    respx.post(URL).mock(return_value=stream_response(chunk("a", finish="stop"), usage_chunk(ds)))
    r = await make(no_sleep).chat(ChatRequest(user="u"))
    assert (r.usage.prompt_tokens, r.usage.completion_tokens, r.usage.reasoning_tokens, r.usage.cached_tokens) == (30, 40, 9, 12)


@respx.mock
async def test_progress_reports_running_sizes(no_sleep):
    respx.post(URL).mock(return_value=stream_response(
        chunk(reasoning="think"), chunk("ab"), chunk("cd", finish="stop"), usage_chunk({})))
    seen = []
    await make(no_sleep).chat(ChatRequest(user="u"), on_progress=lambda r, c: seen.append((r, c)))
    assert seen == [(5, 0), (5, 2), (5, 4)]


@respx.mock
async def test_trims_whitespace_around_answer(no_sleep):
    respx.post(URL).mock(return_value=stream_response(chunk("  \n hello \n", finish="stop"), usage_chunk({})))
    assert (await make(no_sleep).chat(ChatRequest(user="u"))).content == "hello"


@respx.mock
async def test_empty_content_is_not_always_the_same_failure(no_sleep):
    # clean stop, nothing written → EMPTY
    respx.post(URL).mock(return_value=stream_response(chunk("", finish="stop"), usage_chunk({"completion_tokens": 3})))
    with pytest.raises(LLMError) as e:
        await make(no_sleep).chat(ChatRequest(user="u", max_tokens=1000))
    assert e.value.kind == errors.EMPTY

    # DeepSeek reports finish=stop although max_tokens ended it mid-reasoning → BUDGET
    respx.post(URL).mock(return_value=stream_response(
        chunk(reasoning="x"), chunk("", finish="stop"),
        usage_chunk({"prompt_tokens": 5, "completion_tokens": 100, "completion_tokens_details": {"reasoning_tokens": 100}})))
    with pytest.raises(LLMError) as e:
        await make(no_sleep).chat(ChatRequest(user="u", max_tokens=100))
    assert e.value.kind == errors.BUDGET
    assert e.value.usage.reasoning_tokens == 100


@respx.mock
async def test_truncated_keeps_partial_and_billed_usage(no_sleep):
    respx.post(URL).mock(return_value=stream_response(
        chunk("half an ans"), chunk("", finish="length"), usage_chunk({"prompt_tokens": 8, "completion_tokens": 50})))
    with pytest.raises(LLMError) as e:
        await make(no_sleep).chat(ChatRequest(user="u", max_tokens=50))
    assert e.value.kind == errors.TRUNCATED
    assert e.value.partial == "half an ans"
    assert e.value.usage.completion_tokens == 50
    assert "finish_reason=length" in e.value.diag


@pytest.mark.parametrize("status,kind", [
    (401, errors.AUTH), (403, errors.AUTH), (402, errors.BALANCE), (405, errors.BALANCE),
    (429, errors.RATE_LIMIT), (500, errors.SERVER), (503, errors.SERVER), (400, errors.MALFORMED),
])
@respx.mock
async def test_status_to_error_kind(no_sleep, status, kind):
    respx.post(URL).mock(return_value=httpx.Response(status, json={"message": "nope"}))
    with pytest.raises(LLMError) as e:
        await make(no_sleep, retries=0).chat(ChatRequest(user="u"))
    assert e.value.kind == kind
    assert f"http={status}" in e.value.diag and "nope" in e.value.diag


@respx.mock
async def test_retries_only_what_retrying_can_fix(no_sleep):
    route = respx.post(URL).mock(return_value=httpx.Response(429, json={}))
    with pytest.raises(LLMError):
        await make(no_sleep, retries=2).chat(ChatRequest(user="u"))
    assert route.call_count == 3

    route.reset()
    route.mock(return_value=httpx.Response(401, json={}))
    with pytest.raises(LLMError):
        await make(no_sleep, retries=2).chat(ChatRequest(user="u"))
    assert route.call_count == 1


@respx.mock
async def test_recovers_when_a_retry_succeeds(no_sleep):
    route = respx.post(URL).mock(side_effect=[
        httpx.Response(503, json={}),
        stream_response(chunk("ok", finish="stop"), usage_chunk({})),
    ])
    assert (await make(no_sleep).chat(ChatRequest(user="u"))).content == "ok"
    assert route.call_count == 2


async def test_without_a_key_fails_fast_and_says_why(no_sleep):
    with pytest.raises(LLMError) as e:
        await make(no_sleep, key="").chat(ChatRequest(user="u"))
    assert e.value.kind == errors.NOT_CONFIGURED


@respx.mock
async def test_error_inside_stream_is_surfaced(no_sleep):
    respx.post(URL).mock(return_value=stream_response(
        chunk("part"), {"error": {"message": "upstream exploded", "type": "server_error"}}))
    with pytest.raises(LLMError) as e:
        await make(no_sleep, retries=0).chat(ChatRequest(user="u"))
    assert e.value.kind == errors.SERVER
    assert "upstream exploded" in e.value.diag or "upstream exploded" in e.value.message


@respx.mock
async def test_connection_failure_is_server_and_timeout_is_timeout(no_sleep):
    respx.post(URL).mock(side_effect=httpx.ConnectError("boom"))
    with pytest.raises(LLMError) as e:
        await make(no_sleep, retries=0).chat(ChatRequest(user="u"))
    assert e.value.kind == errors.SERVER

    respx.post(URL).mock(side_effect=httpx.ReadTimeout("slow"))
    with pytest.raises(LLMError) as e:
        await make(no_sleep, retries=0).chat(ChatRequest(user="u"))
    assert e.value.kind == errors.TIMEOUT
