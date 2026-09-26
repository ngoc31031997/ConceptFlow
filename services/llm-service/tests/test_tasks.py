import json

import httpx
import pytest
import respx

from app import errors, tasks
from app.errors import LLMError
from app.provider import Provider
from tests.conftest import chunk, stream_response, usage_chunk

BASE = "http://ollama.test:11434/v1"
URL = BASE + "/chat/completions"


def make(no_sleep):
    return Provider("ollama", BASE, "ollama", "test-model", timeout=5, max_retries=0, sleep=no_sleep)


def reply(text):
    return stream_response(chunk(text, finish="stop"), usage_chunk({}))


def test_prompt_names_the_projects_language_and_carries_script_and_topic():
    vi = tasks.build_suggest_prompt("SCRIPT-BODY", "topic-x", "vi")
    en = tasks.build_suggest_prompt("SCRIPT-BODY", "topic-x", "en")
    assert "written in Vietnamese" in vi and "written in English" not in vi
    assert "written in English" in en
    assert "SCRIPT-BODY" in vi and "topic-x" in vi


def test_prompt_caps_the_script_by_characters_not_bytes():
    script = "ế" * (tasks.MAX_SCRIPT_CHARS + 500)
    p = tasks.build_suggest_prompt(script, "t", "vi")
    assert p.count("ế") == tasks.MAX_SCRIPT_CHARS


def test_truncate_title_counts_characters():
    t = "ệ" * 150
    assert len(tasks.truncate_title(t)) == tasks.MAX_TITLE_LENGTH
    assert tasks.truncate_title("ngắn") == "ngắn"


def test_normalize_tags_never_returns_none():
    assert tasks.normalize_tags(None) == []
    assert tasks.normalize_tags(["a", "b"]) == ["a", "b"]
    assert tasks.normalize_tags("a, b , ,c") == ["a", "b", "c"]
    assert tasks.normalize_tags([]) == []


def test_strip_code_fence():
    assert tasks.strip_code_fence("```python\nx = 1\n```") == "x = 1"
    assert tasks.strip_code_fence("x = 1") == "x = 1"


def test_short_script_prompt_requires_clip_wrapper_and_uses_source_as_context_only():
    p = tasks.build_short_script_prompt("Topic", "LONG-SOURCE", "vi")
    assert 'with self.clip("short")' in p and "for context only" in p and "LONG-SOURCE" in p
    assert "for context only" not in tasks.build_short_script_prompt("Topic", "", "vi")


@respx.mock
async def test_suggest_metadata_retries_when_title_empty_and_uses_json_mode(no_sleep):
    route = respx.post(URL).mock(side_effect=[
        reply('{"title":"","description":"d","tags":null}'),
        reply('{"title":"Sắp xếp nổi bọt","description":"d","tags":["a"]}'),
    ])
    out = await tasks.suggest_metadata(make(no_sleep), "script", "topic", "vi")
    assert out.value["title"] == "Sắp xếp nổi bọt" and out.value["tags"] == ["a"]
    assert route.call_count == 2
    assert json.loads(route.calls.last.request.content)["response_format"] == {"type": "json_object"}


@respx.mock
async def test_suggest_metadata_fails_loudly_when_every_attempt_unusable(no_sleep):
    route = respx.post(URL).mock(return_value=reply('{"title":"   ","description":"d","tags":null}'))
    with pytest.raises(LLMError) as e:
        await tasks.suggest_metadata(make(no_sleep), "s", "t", "vi")
    assert e.value.kind == errors.EMPTY and route.call_count == tasks.SUGGEST_MAX_ATTEMPTS


@respx.mock
async def test_suggest_metadata_returns_empty_tags_when_omitted(no_sleep):
    respx.post(URL).mock(return_value=reply('{"title":"T","description":"d"}'))
    out = await tasks.suggest_metadata(make(no_sleep), "s", "t", "vi")
    assert out.value["tags"] == []


@respx.mock
async def test_suggest_metadata_does_not_retry_a_dead_key(no_sleep):
    route = respx.post(URL).mock(return_value=httpx.Response(401, json={}))
    with pytest.raises(LLMError) as e:
        await tasks.suggest_metadata(make(no_sleep), "s", "t", "vi")
    assert e.value.kind == errors.AUTH and route.call_count == 1


@respx.mock
async def test_short_script_strips_fence_retries_empty_and_does_not_force_json(no_sleep):
    route = respx.post(URL).mock(side_effect=[reply("   "), reply("```python\nfrom conceptflow import *\n```")])
    out = await tasks.suggest_short_script(make(no_sleep), "topic", "", "vi")
    assert out.value["script"] == "from conceptflow import *"
    assert route.call_count == 2
    assert "response_format" not in json.loads(route.calls.last.request.content)


def test_unknown_language_falls_back_to_english_like_the_orchestrator_domain():
    assert tasks.language_name("") == "English" and tasks.language_name("fr") == "English"
