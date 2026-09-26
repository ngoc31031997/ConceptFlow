"""One OpenAI-compatible provider (Hive or Ollama), called through the
`openai` SDK.

Replaces the orchestrator's hand-written hive_client.go while keeping the
behaviour that file had been measured and fixed into (CR-027 D11-D13):
error classification, both usage shapes, mid-stream error chunks, the
"budget" vs "empty" distinction and retry only for 429/5xx.
"""

from __future__ import annotations

import asyncio
import logging
import random
import time
from collections.abc import Awaitable, Callable
from dataclasses import dataclass

import openai
from openai import AsyncOpenAI

from app import errors
from app.errors import LLMError, Usage
from app.limiter import RateLimiter

logger = logging.getLogger(__name__)

ProgressFn = Callable[[int, int], Awaitable[None] | None]


@dataclass
class ChatRequest:
    user: str
    system: str = ""
    model: str = ""  # "" = the provider's default model
    max_tokens: int = 0  # 0 = no cap sent; the provider applies its own ceiling
    temperature: float = 0.7
    json_mode: bool = False


@dataclass
class ChatResult:
    content: str
    usage: Usage


def _usage_from(raw: dict | None, model: str) -> Usage:
    """Accepts BOTH shapes the two documented Hive models return (CR-027 D12):

    glm-5.3-flash        reasoning_tokens at the top level
    deepseek-v4.1-flash  reasoning_tokens nested in completion_tokens_details,
                         plus prompt_tokens_details.cached_tokens

    Reading only one shape silently records zero for the other model.
    """
    raw = raw or {}
    reasoning = raw.get("reasoning_tokens") or 0
    if not reasoning:
        reasoning = (raw.get("completion_tokens_details") or {}).get("reasoning_tokens") or 0
    cached = (raw.get("prompt_tokens_details") or {}).get("cached_tokens") or 0
    return Usage(
        model=model,
        prompt_tokens=raw.get("prompt_tokens") or 0,
        completion_tokens=raw.get("completion_tokens") or 0,
        reasoning_tokens=reasoning,
        cached_tokens=cached,
    )


def kind_for_status(status: int) -> str:
    if status in (401, 403):
        return errors.AUTH
    # Hive documents 405 as "Out of Balance"; 402 is the conventional code.
    if status in (402, 405):
        return errors.BALANCE
    if status == 429:
        return errors.RATE_LIMIT
    if status >= 500:
        return errors.SERVER
    return errors.MALFORMED


def _backoff(attempt: int) -> float:
    return float(1 << (attempt - 1)) + random.random() * 0.25


class Provider:
    def __init__(
        self,
        name: str,
        base_url: str,
        api_key: str,
        default_model: str,
        *,
        timeout: float | None,
        max_retries: int,
        limiter: RateLimiter | None = None,
        extra_headers: dict[str, str] | None = None,
        sleep: Callable[[float], Awaitable[None]] = asyncio.sleep,
    ) -> None:
        self.name = name
        self.default_model = default_model
        self._api_key = api_key
        self._max_retries = max_retries
        self._limiter = limiter
        self._extra_headers = extra_headers or {}
        self._sleep = sleep
        # max_retries=0: retrying is done here so that the shared limiter sees
        # every attempt and only the failures worth retrying are retried.
        self._client = AsyncOpenAI(
            base_url=base_url, api_key=api_key or "unset", timeout=timeout, max_retries=0
        )

    @property
    def configured(self) -> bool:
        return bool(self._api_key)

    async def chat(self, req: ChatRequest, on_progress: ProgressFn | None = None) -> ChatResult:
        if not self.configured:
            raise LLMError(errors.NOT_CONFIGURED, self.name, f"{self.name.upper()}_API_KEY is not set")
        last: LLMError | None = None
        for attempt in range(self._max_retries + 1):
            if attempt:
                await self._sleep(_backoff(attempt))
            try:
                return await self._once(req, on_progress)
            except LLMError as err:
                last = err
                if err.kind not in errors.RETRYABLE:
                    break
        assert last is not None
        raise last

    async def _once(self, req: ChatRequest, on_progress: ProgressFn | None) -> ChatResult:
        model = req.model or self.default_model
        started = time.monotonic()
        diag: list[str] = [
            f"request: model={model} max_tokens={req.max_tokens} temperature={req.temperature:g} "
            f"system_chars={len(req.system)} user_chars={len(req.user)}"
        ]

        def fail(kind: str, message: str, usage: Usage | None = None, partial: str = "") -> LLMError:
            diag.append(f"elapsed: {time.monotonic() - started:.3f}s")
            return LLMError(kind, self.name, message, usage or Usage(model=model), partial, "\n".join(diag))

        messages = []
        if req.system.strip():
            messages.append({"role": "system", "content": req.system})
        messages.append({"role": "user", "content": req.user})
        kwargs: dict = {
            "model": model,
            "messages": messages,
            "temperature": req.temperature,
            "stream": True,
            "extra_headers": {"Accept": "text/event-stream", **self._extra_headers},
        }
        if req.max_tokens > 0:
            kwargs["max_tokens"] = req.max_tokens
        if req.json_mode:
            kwargs["response_format"] = {"type": "json_object"}

        if self._limiter:
            await self._limiter.acquire()

        content: list[str] = []
        reasoning_chars = 0
        content_chars = 0
        finish = ""
        raw_usage: dict | None = None
        resp_model = ""
        chunks = 0
        try:
            stream = await self._client.chat.completions.create(**kwargs)
            async for chunk in stream:
                chunks += 1
                if chunk.model:
                    resp_model = chunk.model
                if chunk.usage is not None:
                    raw_usage = chunk.usage.model_dump()
                for choice in chunk.choices:
                    delta = choice.delta
                    if delta.content:
                        content.append(delta.content)
                        content_chars += len(delta.content)
                    reasoning = (delta.model_extra or {}).get("reasoning_content")
                    if reasoning:
                        reasoning_chars += len(reasoning)
                    if choice.finish_reason:
                        finish = choice.finish_reason
                if on_progress and chunk.choices:
                    res = on_progress(reasoning_chars, content_chars)
                    if res is not None:
                        await res
        except openai.APITimeoutError as exc:
            raise fail(errors.TIMEOUT, f"call {self.name}: {exc}") from exc
        except openai.APIConnectionError as exc:
            raise fail(errors.SERVER, f"call {self.name}: {exc}") from exc
        except openai.APIStatusError as exc:
            diag.append(f"response: http={exc.status_code} request_id={exc.request_id!r}")
            diag.append(f"body: {str(exc.body)[:4000]}")
            raise fail(kind_for_status(exc.status_code), f"{self.name} returned {exc.status_code}: {exc.message}") from exc
        except openai.APIError as exc:
            # An API that fails mid-stream answers 200 and then sends
            # {"error": ...} as a data chunk; the SDK raises it as APIError.
            diag.append(f"stream_error: {str(exc.body)[:2000]}")
            raise fail(errors.SERVER, f"{self.name} sent an error inside the stream: {exc.message}") from exc
        except TimeoutError as exc:
            raise fail(errors.TIMEOUT, f"call {self.name}: timed out") from exc

        text = "".join(content).strip()
        usage = _usage_from(raw_usage, resp_model or model)
        diag.append(
            f"stream: chunks={chunks} reasoning_chars={reasoning_chars} content_chars={content_chars} "
            f"finish_reason={finish}"
        )

        # D13: "empty" and "ran out of room" are different problems, and a
        # reasoning model turns the second into the first. DeepSeek reports
        # finish_reason "stop" even when max_tokens ended the reply
        # mid-reasoning, so an empty answer that used the whole budget is a
        # budget problem, not an "empty" one.
        hit_cap = req.max_tokens > 0 and usage.completion_tokens >= req.max_tokens
        if finish == "length" or (not text and hit_cap):
            if not text:
                raise fail(
                    errors.BUDGET,
                    "the whole token budget went on reasoning before any answer was written "
                    f"({usage.reasoning_tokens} reasoning tokens)",
                    usage,
                )
            raise fail(errors.TRUNCATED, "response was cut off at max_tokens", usage, partial=text)
        if not text:
            raise fail(errors.EMPTY, "model returned empty content", usage)
        return ChatResult(content=text, usage=usage)
