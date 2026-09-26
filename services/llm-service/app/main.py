"""llm-service HTTP API (CR-039).

The single place that talks to an LLM provider. Streaming endpoints answer
newline-delimited JSON: zero or more `progress`/`phase` events, then exactly one
`result` or `error` event. Non-streaming endpoints answer 200 with the result,
or 502 with {"error": ...}.
"""

from __future__ import annotations

import asyncio
import json
import logging
from collections.abc import AsyncIterator, Awaitable, Callable

from fastapi import FastAPI
from fastapi.responses import JSONResponse, StreamingResponse
from pydantic import BaseModel, Field

from app import storyboard as sbm
from app import tasks
from app.config import Config
from app.errors import LLMError, Usage
from app.pipeline.checker import CheckerPort, RenderingChecker
from app.pipeline.checker import CheckerUnavailable as _CheckerUnavailable
from app.pipeline.run import ChunkCache, CodePipeline, CodeRequest, PipelineFailure
from app.provider import ChatRequest
from app.registry import Providers

logger = logging.getLogger("llm-service")


class ChatBody(BaseModel):
    provider: str = "hive"
    system: str = ""
    user: str
    model: str = ""
    max_tokens: int = Field(0, ge=0)
    temperature: float = 0.7
    json_mode: bool = False


class MetadataBody(BaseModel):
    script_content: str
    category_hint: str = ""
    language: str = "en"


class ShortScriptBody(BaseModel):
    topic: str = ""
    source_script_content: str = ""
    language: str = "en"


class StoryboardBody(BaseModel):
    content: str
    model: str = ""
    max_tokens: int = Field(0, ge=0)


class CodeBody(BaseModel):
    engine: str
    topic: str = ""
    storyboard: str
    system: str
    model: str = ""
    max_tokens: int = Field(0, ge=0)
    temperature: float = 0.3


def _line(obj: dict) -> bytes:
    return (json.dumps(obj, ensure_ascii=False) + "\n").encode()


def _error_response(err: LLMError | str, status: int = 502, **extra) -> JSONResponse:
    body = err.to_dict() if isinstance(err, LLMError) else {"kind": "malformed", "message": err}
    return JSONResponse({"error": {**body, **extra}}, status_code=status)


def _stream(work: Callable[[Callable[[dict], Awaitable[None]]], Awaitable[dict]]) -> StreamingResponse:
    """Run `work(emit)` and stream what it emits, then its returned result (or its error)."""

    async def body() -> AsyncIterator[bytes]:
        queue: asyncio.Queue[dict | None] = asyncio.Queue()

        async def emit(event: dict) -> None:
            await queue.put(event)

        async def runner() -> None:
            try:
                result = await work(emit)
                await queue.put({"type": "result", **result})
            except PipelineFailure as exc:
                err = exc.error.to_dict() if exc.error else {"kind": exc.kind, "message": exc.message}
                await queue.put({
                    "type": "error", "error": {**err, "message": exc.message, "kind": exc.kind},
                    "calls": [c.to_dict() for c in exc.calls],
                })
            except LLMError as exc:
                await queue.put({"type": "error", "error": exc.to_dict(), "calls": []})
            except _CheckerUnavailable as exc:
                await queue.put({
                    "type": "error", "error": {"kind": "server", "message": str(exc)}, "calls": [],
                })
            except Exception as exc:  # noqa: BLE001 - reported to the caller, never swallowed
                logger.exception("unexpected failure in a streamed call")
                await queue.put({"type": "error", "error": {"kind": "server", "message": f"internal error: {exc}"}, "calls": []})
            finally:
                await queue.put(None)

        task = asyncio.create_task(runner())
        try:
            while (event := await queue.get()) is not None:
                yield _line(event)
        finally:
            if not task.done():
                task.cancel()  # the caller went away

    return StreamingResponse(body(), media_type="application/x-ndjson")


def create_app(
    config: Config | None = None, providers: Providers | None = None, checker: CheckerPort | None = None
) -> FastAPI:
    config = config or Config.from_env()
    providers = providers or Providers(config)
    checker = checker or RenderingChecker(config.rendering_url, config.rendering_check_timeout)
    cache = ChunkCache()
    app = FastAPI(title="llm-service")

    @app.get("/health")
    async def health() -> dict:
        return {
            "status": "ok",
            "hive_configured": providers.hive.configured,
            "light_provider": providers.light.name,
        }

    @app.post("/v1/chat")
    async def chat(body: ChatBody):
        try:
            provider = providers.get(body.provider)
        except KeyError:
            return _error_response(f"unknown provider {body.provider!r}", status=400)

        async def work(emit):
            async def progress(reasoning: int, content: int) -> None:
                await emit({"type": "progress", "reasoning_chars": reasoning, "content_chars": content})

            res = await provider.chat(
                ChatRequest(user=body.user, system=body.system, model=body.model, max_tokens=body.max_tokens,
                            temperature=body.temperature, json_mode=body.json_mode),
                on_progress=progress,
            )
            return {"content": res.content, "usage": res.usage.to_dict(), "provider": provider.name}

        return _stream(work)

    @app.post("/v1/suggest-metadata")
    async def suggest_metadata(body: MetadataBody):
        try:
            out = await tasks.suggest_metadata(providers.light, body.script_content, body.category_hint, body.language)
        except LLMError as err:
            return _error_response(err)
        return {**out.value, "usage": out.usage.to_dict(), "provider": providers.light.name}

    @app.post("/v1/suggest-short-script")
    async def suggest_short_script(body: ShortScriptBody):
        try:
            out = await tasks.suggest_short_script(
                providers.light, body.topic, body.source_script_content, body.language)
        except LLMError as err:
            return _error_response(err)
        return {**out.value, "usage": out.usage.to_dict(), "provider": providers.light.name}

    @app.post("/v1/storyboard/finalize")
    async def storyboard_finalize(body: StoryboardBody):
        """Validate what the Visual Director returned; one LLM repair turn if it is
        not valid. Returns the canonical (re-serialised) JSON."""
        total = Usage()
        content = body.content
        for attempt in range(2):
            try:
                sb = sbm.parse(content)
            except sbm.StoryboardError as exc:
                if attempt == 1:
                    return _error_response(
                        f"the storyboard is still invalid after one repair turn: {exc}", status=422,
                        usage=total.to_dict(), problems=exc.problems)
                try:
                    res = await providers.hive.chat(ChatRequest(
                        user=sbm.fix_prompt(content, exc.problems), model=body.model,
                        max_tokens=body.max_tokens, temperature=0.0))
                except LLMError as err:
                    err.usage = total + err.usage
                    return _error_response(err)
                total = total + res.usage
                content = res.content
                continue
            return {"storyboard": sbm.dumps(sb), "usage": total.to_dict(), "repaired": attempt == 1,
                    "shots": len(sb.all_shots())}

    @app.post("/v1/code/generate")
    async def code_generate(body: CodeBody):
        pipeline = CodePipeline(
            providers.hive, checker,
            chunk_shots=config.code_chunk_shots, concurrency=config.code_chunk_concurrency,
            repair_rounds=config.code_repair_max_rounds, cache=cache)

        async def work(emit):
            res = await pipeline.run(CodeRequest(
                engine=body.engine, topic=body.topic, storyboard=body.storyboard, system=body.system,
                model=body.model, max_tokens=body.max_tokens, temperature=body.temperature), emit)
            return res.to_dict()

        return _stream(work)

    return app
