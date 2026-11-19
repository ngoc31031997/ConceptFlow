"""Client của llm-service: cuộc gọi model (`/v1/chat`), thư viện prompt và model mặc định của bước.

Không có bộ nhớ đệm và không có bản dự phòng cục bộ: llm-service không trả lời được thì người gọi
nhận lỗi nói đúng điều đó.
"""

from __future__ import annotations

import asyncio
import json
import logging
from urllib.parse import quote

import httpx

from application.illustration_ports import ChatResult
from domain.drawer_prompts import PROMPT_OWNER, SystemPrompt
from domain.illustration import LLMError, PromptUnavailableError

logger = logging.getLogger(__name__)

PROVIDER = "llm-service"


class LLMServiceClient:
    def __init__(
        self, base_url: str, timeout_seconds: float = 0, client: httpx.AsyncClient | None = None,
    ) -> None:
        # 0 = chờ bao lâu llm-service chờ (Hive có thể mất nhiều phút với model suy luận).
        timeout = httpx.Timeout(timeout_seconds) if timeout_seconds > 0 else httpx.Timeout(None, connect=10)
        self._client = client or httpx.AsyncClient(base_url=base_url.rstrip("/"), timeout=timeout)

    async def chat(
        self, *, system: str, user: str, model: str, max_tokens: int, temperature: float,
        role: str, step: str, phase: str,
    ) -> ChatResult:
        body = {
            "provider": "hive", "system": system, "user": user, "model": model, "max_tokens": max_tokens,
            "temperature": temperature, "role": role, "step": step, "phase": phase, "expect_json": False,
        }
        try:
            async with self._client.stream("POST", "/v1/chat", json=body) as res:
                if res.status_code != 200:
                    raise self._status_error(res.status_code, (await res.aread()).decode("utf-8", "replace"))
                async for line in res.aiter_lines():
                    if not line.strip():
                        continue
                    event = self._event(line)
                    kind = event.get("type")
                    if kind == "result":
                        return ChatResult(content=str(event.get("content") or ""),
                                          call_id=str(event.get("call_id") or ""))
                    if kind == "error":
                        raise self._wire_error(event.get("error") or {})
        except httpx.TimeoutException as exc:
            raise LLMError("timeout", PROVIDER, f"call llm-service: {exc!r}") from exc
        except httpx.HTTPError as exc:
            raise LLMError("server", PROVIDER, f"call llm-service: {exc!r}") from exc
        raise LLMError("server", PROVIDER, "llm-service closed the stream without a result")

    async def get_template(self, role: str) -> str:
        """Template đang chạy của `role`: bản đang dùng, không có thì bản hệ thống."""
        res = await self._request("GET", f"/internal/v1/prompts/{quote(role, safe='')}")
        if res.status_code == 404 or res.status_code == 400:
            raise PromptUnavailableError(f'llm-service chưa có prompt "{role}"')
        self._require_ok(res)
        text = self._body(res).get("template_text")
        if not isinstance(text, str) or not text.strip():
            raise PromptUnavailableError(f'prompt "{role}" ở llm-service trống')
        return text

    async def step_default_model(self, step: str) -> str:
        """Model mặc định của bước, rỗng khi không đọc được (lúc đó llm-service tự chọn)."""
        try:
            res = await self._request("GET", "/internal/v1/step-models")
            self._require_ok(res)
            models = self._body(res).get("step_models") or {}
        except (LLMError, PromptUnavailableError) as exc:
            logger.warning("could not read the step default models (step %s): %s", step, exc)
            return ""
        return str(models.get(step) or "")

    async def push_system_prompts(self, owner: str, prompts: list[SystemPrompt]) -> None:
        """Đặt `prompts` là toàn bộ prompt hệ thống của `owner`; prompt của Creator không bị đụng."""
        body = {
            "owner": owner,
            "prompts": [{"role": p.role, "name": p.name, "template_text": p.template_text} for p in prompts],
        }
        res = await self._request("PUT", "/internal/v1/prompts/system", json=body)
        self._require_ok(res)

    async def close(self) -> None:
        await self._client.aclose()

    # --- nội bộ ---

    async def _request(self, method: str, path: str, **kwargs) -> httpx.Response:
        try:
            return await self._client.request(method, path, **kwargs)
        except httpx.TimeoutException as exc:
            raise LLMError("timeout", PROVIDER, f"call llm-service: {exc!r}") from exc
        except httpx.HTTPError as exc:
            raise LLMError("server", PROVIDER, f"call llm-service: {exc!r}") from exc

    @staticmethod
    def _require_ok(res: httpx.Response) -> None:
        if res.status_code not in (200, 204):
            raise LLMError(
                "server", PROVIDER, f"llm-service returned {res.status_code}: {res.text.strip()[:300]}")

    @staticmethod
    def _body(res: httpx.Response) -> dict:
        try:
            body = res.json()
        except ValueError as exc:
            raise LLMError("malformed", PROVIDER, f"decode response: {exc}") from exc
        if not isinstance(body, dict):
            raise LLMError("malformed", PROVIDER, "decode response: not an object")
        return body

    @staticmethod
    def _event(line: str) -> dict:
        try:
            event = json.loads(line)
        except ValueError as exc:
            raise LLMError("malformed", PROVIDER, f"unreadable event from llm-service: {exc}") from exc
        if not isinstance(event, dict):
            raise LLMError("malformed", PROVIDER, "unreadable event from llm-service")
        return event

    @staticmethod
    def _wire_error(err: dict) -> LLMError:
        return LLMError(
            str(err.get("kind") or "server"), str(err.get("provider") or PROVIDER),
            str(err.get("message") or ""), str(err.get("call_id") or ""))

    @classmethod
    def _status_error(cls, status: int, text: str) -> LLMError:
        try:
            err = json.loads(text).get("error")
        except (ValueError, AttributeError):
            err = None
        if isinstance(err, dict) and err.get("message"):
            return cls._wire_error(err)
        kind = "malformed" if status in (400, 422) else "server"
        return LLMError(kind, PROVIDER, f"llm-service returned {status}: {text.strip()[:300]}")


async def push_system_prompts_until_accepted(
    client: LLMServiceClient, prompts: list[SystemPrompt], first_pause: float = 2.0, max_pause: float = 60.0,
) -> None:
    """Đẩy prompt hệ thống lên llm-service; thử lại với quãng nghỉ tăng dần tới khi nó nhận."""
    pause = first_pause
    while True:
        try:
            await client.push_system_prompts(PROMPT_OWNER, prompts)
        except LLMError as exc:
            logger.error("could not push the system prompts to llm-service; retrying in %ss: %s", pause, exc)
            await asyncio.sleep(pause)
            pause = min(pause * 2, max_pause)
            continue
        logger.warning("system prompts pushed to llm-service: %d", len(prompts))
        return
