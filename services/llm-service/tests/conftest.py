import json

import httpx
import pytest


def sse(*payloads: dict | str) -> str:
    lines = []
    for p in payloads:
        lines.append("data: " + (p if isinstance(p, str) else json.dumps(p)) + "\n\n")
    return "".join(lines)


def chunk(content=None, reasoning=None, finish=None, usage=None, model="test-model"):
    delta = {}
    if content is not None:
        delta["content"] = content
    if reasoning is not None:
        delta["reasoning_content"] = reasoning
    body = {
        "id": "c1", "object": "chat.completion.chunk", "created": 1, "model": model,
        "choices": [{"index": 0, "delta": delta, "finish_reason": finish}],
    }
    if usage is not None:
        body["usage"] = usage
    return body


def usage_chunk(usage, model="test-model"):
    return {"id": "c1", "object": "chat.completion.chunk", "created": 1, "model": model,
            "choices": [], "usage": usage}


def stream_response(*payloads, status=200):
    return httpx.Response(status, headers={"content-type": "text/event-stream"}, text=sse(*payloads, "[DONE]"))


@pytest.fixture
def no_sleep():
    async def _sleep(_):
        return None
    return _sleep
