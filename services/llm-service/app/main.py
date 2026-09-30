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
from typing import Literal

from fastapi import FastAPI
from fastapi.responses import JSONResponse, StreamingResponse
from pydantic import BaseModel, Field

from app import storyboard as sbm
from app import tasks
from app.config import Config
from app.errors import LLMError, Usage
from app.pipeline.checker import CheckerPort, RenderingChecker
from app.pipeline.checker import CheckerUnavailable as _CheckerUnavailable
from app.pipeline.extract import ExtractError
from app.pipeline.run import (
    Call,
    CodePipeline,
    CodeRequest,
    DoneSegment,
    PipelineFailure,
    SegmentNotReady,
    make_plan,
    parse_segment,
    segment_prompt,
)
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
    stream: bool = False  # CR-040 FR116: emit `progress` events like /v1/chat


class ShortScriptBody(BaseModel):
    topic: str = ""
    source_script_content: str = ""
    language: str = "en"
    stream: bool = False


class StoryboardBody(BaseModel):
    content: str
    model: str = ""
    max_tokens: int = Field(0, ge=0)


class IllustrationIn(BaseModel):
    """CR-044 — one approved library drawing the code step may use."""
    name: str
    usage: str = ""
    description: str = ""
    code: str


class SubtitleBandIn(BaseModel):
    """CR-048 T6b — the strip burned-in subtitles cover, from the frame edge."""
    edge: Literal["top", "bottom"]
    px: int = Field(gt=0, lt=1080)


class SegmentIn(BaseModel):
    """CR-050 — a segment the caller already stored (ADR-0030)."""
    key: str
    fingerprint: str
    content: dict


class CodeBody(BaseModel):
    illustrations: list[IllustrationIn] = []
    engine: str
    topic: str = ""
    storyboard: str
    system: str
    model: str = ""
    max_tokens: int = Field(0, ge=0)
    temperature: float = 0.3
    # CR-048 T6b — passed to the Rendering layout check (Remotion only).
    # No band = nothing burned into the frame; no font = the video default.
    subtitle_band: SubtitleBandIn | None = None
    video_font: str = ""
    # CR-050 FR-7 — shots per chunk for this project; 0 = CODE_CHUNK_SHOTS.
    chunk_shots: int = Field(0, ge=0, le=10)
    # CR-050 — what the caller already has, and (FR-4) the only segments to run.
    segments: list[SegmentIn] = []
    only: list[str] | None = None

    def request(self, model: str, max_reasoning_chars: int) -> CodeRequest:
        return CodeRequest(
            engine=self.engine, topic=self.topic, storyboard=self.storyboard, system=self.system,
            model=model, max_tokens=self.max_tokens, temperature=self.temperature,
            max_reasoning_chars=max_reasoning_chars,
            illustrations=[i.model_dump() for i in self.illustrations],
            subtitle_band=self.subtitle_band.model_dump() if self.subtitle_band else None,
            video_font=self.video_font, chunk_shots=self.chunk_shots,
            done={s.key: DoneSegment(s.fingerprint, s.content) for s in self.segments},
            only=set(self.only) if self.only is not None else None)


class SegmentPromptBody(CodeBody):
    key: str


class SegmentParseBody(CodeBody):
    key: str
    reply: str


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
                diag = exc.error.diag if exc.error else ""
                logger.warning("pipeline failure: %s\n%s", exc.message, diag)
                await queue.put({
                    "type": "error", "error": {**err, "message": exc.message, "kind": exc.kind},
                    "calls": [c.to_dict() for c in exc.calls],
                })
            except LLMError as exc:
                logger.warning("llm call failed: %s\n%s", exc, exc.diag)
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


async def _suggestion(stream: bool, run):
    """One-shot JSON by default; with `stream` the same work is streamed as
    `progress` events plus the final result, so the GUI can show a live card."""
    if not stream:
        try:
            return await run()
        except LLMError as err:
            logger.warning("llm call failed: %s\n%s", err, err.diag)
            return _error_response(err)

    async def work(emit):
        async def progress(reasoning: int, content: int) -> None:
            await emit({"type": "progress", "reasoning_chars": reasoning, "content_chars": content})

        return await run(progress)

    return _stream(work)


def create_app(
    config: Config | None = None, providers: Providers | None = None, checker: CheckerPort | None = None
) -> FastAPI:
    config = config or Config.from_env()
    providers = providers or Providers(config)
    checker = checker or RenderingChecker(config.rendering_url, config.rendering_check_timeout)
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
        provider, model = providers.for_model(provider, body.model)

        async def work(emit):
            async def progress(reasoning: int, content: int) -> None:
                await emit({"type": "progress", "reasoning_chars": reasoning, "content_chars": content})

            res = await provider.chat(
                ChatRequest(user=body.user, system=body.system, model=model, max_tokens=body.max_tokens,
                            temperature=body.temperature, json_mode=body.json_mode,
                            max_reasoning_chars=config.chat_max_reasoning_chars),
                on_progress=progress,
            )
            return {"content": res.content, "usage": res.usage.to_dict(), "provider": provider.name}

        return _stream(work)

    @app.post("/v1/suggest-metadata")
    async def suggest_metadata(body: MetadataBody):
        async def run(on_progress=None):
            out = await tasks.suggest_metadata(
                providers.light, body.script_content, body.category_hint, body.language, on_progress)
            return {**out.value, "usage": out.usage.to_dict(), "provider": providers.light.name}

        return await _suggestion(body.stream, run)

    @app.post("/v1/suggest-short-script")
    async def suggest_short_script(body: ShortScriptBody):
        async def run(on_progress=None):
            out = await tasks.suggest_short_script(
                providers.light, body.topic, body.source_script_content, body.language, on_progress)
            return {**out.value, "usage": out.usage.to_dict(), "provider": providers.light.name}

        return await _suggestion(body.stream, run)

    @app.post("/v1/storyboard/finalize")
    async def storyboard_finalize(body: StoryboardBody):
        """Validate what the Visual Director returned; one LLM repair turn if it is
        not valid. Returns the canonical (re-serialised) JSON."""
        total = Usage()
        content = body.content
        provider, model = providers.for_model(providers.hive, body.model)
        for attempt in range(2):
            try:
                sb = sbm.parse(content)
            except sbm.StoryboardError as exc:
                if attempt == 1:
                    return _error_response(
                        f"the storyboard is still invalid after one repair turn: {exc}", status=422,
                        usage=total.to_dict(), problems=exc.problems)
                try:
                    res = await provider.chat(ChatRequest(
                        user=sbm.fix_prompt(content, exc.problems), model=model,
                        max_tokens=body.max_tokens, temperature=0.0))
                except LLMError as err:
                    err.usage = total + err.usage
                    logger.warning("llm call failed: %s\n%s", err, err.diag)
                    return _error_response(err)
                total = total + res.usage
                content = res.content
                continue
            return {"storyboard": sbm.dumps(sb), "usage": total.to_dict(), "repaired": attempt == 1,
                    "shots": len(sb.all_shots())}

    # Events of a v2 run a v1 caller does not know; its calls come back in `calls`.
    v2_only_events = {"plan", "segment_start", "segment_done", "segment_failed", "call", "check"}

    @app.post("/v1/code/generate")
    async def code_generate_v1(body: CodeBody):
        """The pre-CR-050 contract, kept so llm-service and authoring-service
        deploy independently (ADR-0030); removing it is on the backlog. Runs
        the whole step with nothing stored: progress events only, every billed
        call listed at the end, and the first failed segment fails the run with
        the calls so far — as before. What v1 no longer does is keep finished
        chunks in memory for a re-run: a v1 re-run pays for them again."""
        if body.engine not in ("remotion", "manim"):
            return _error_response(f"unknown engine {body.engine!r}", status=400)
        provider, model = providers.for_model(providers.hive, body.model)
        pipeline = CodePipeline(
            provider, checker,
            chunk_shots=config.code_chunk_shots, concurrency=config.code_chunk_concurrency,
            repair_rounds=config.code_repair_max_rounds)

        async def work(emit):
            calls: list[Call] = []
            first_failure: dict | None = None

            async def v1_emit(event: dict) -> None:
                nonlocal first_failure
                if event["type"] == "call":
                    calls.append(Call(
                        event["phase"], event["label"], event["ok"], Usage(**event["usage"]), event["error_kind"],
                        event["error_message"], event["duration_ms"], event["segment"]))
                if event["type"] == "segment_failed" and first_failure is None:
                    first_failure = event["error"]
                if event["type"] not in v2_only_events:
                    await emit(event)

            req = body.request(model, config.code_max_reasoning_chars)
            req.done, req.only = {}, None
            try:
                res = await pipeline.run(req, v1_emit)
            except PipelineFailure as exc:
                exc.calls = calls  # v1 reports what was billed on the error itself
                raise
            if res.status != "done":
                err = first_failure or {"kind": "malformed", "message": "the code step did not finish"}
                raise PipelineFailure(err.get("message", ""), calls, kind=err.get("kind", "malformed"))
            out = res.to_dict()
            for key in ("status", "failed", "missing"):
                out.pop(key)
            out["calls"] = [c.to_dict() for c in calls]
            return out

        return _stream(work)

    @app.post("/v2/code/plan")
    async def code_plan(body: CodeBody):
        """CR-050: how this storyboard is cut into segments, so authoring-service
        never keeps its own copy of the rule. Fingerprints included; no model call."""
        try:
            plan = make_plan(body.request("", 0), body.chunk_shots or config.code_chunk_shots)
        except PipelineFailure as exc:
            return _error_response(exc.message, status=422)
        segments = [s.to_dict() for s in plan.segments]
        if plan.given_frame is not None:
            segments[0]["source"] = "storyboard"
        return {"segments": segments}

    @app.post("/v2/code/generate")
    async def code_generate(body: CodeBody):
        """CR-050 / ADR-0030: runs the missing segments (or `only`), streaming
        each segment's result, each billed call and each failed check."""
        if body.engine not in ("remotion", "manim"):
            return _error_response(f"unknown engine {body.engine!r}", status=400)
        provider, model = providers.for_model(providers.hive, body.model)
        pipeline = CodePipeline(
            provider, checker,
            chunk_shots=config.code_chunk_shots, concurrency=config.code_chunk_concurrency,
            repair_rounds=config.code_repair_max_rounds)

        async def work(emit):
            res = await pipeline.run(body.request(model, config.code_max_reasoning_chars), emit)
            return res.to_dict()

        return _stream(work)

    def _segment_error(exc: Exception) -> JSONResponse:
        if isinstance(exc, PipelineFailure):
            return _error_response(exc.message, status=422)
        if isinstance(exc, KeyError):
            return _error_response(f"unknown segment {exc.args[0]!r}", status=404)
        if isinstance(exc, SegmentNotReady):
            return _error_response(str(exc), status=409)
        return _error_response(str(exc), status=422)

    @app.post("/v2/code/segment-prompt")
    async def code_segment_prompt(body: SegmentPromptBody):
        """CR-050 FR-5: the exact turn one segment would be asked, for an outside AI."""
        _, model = providers.for_model(providers.hive, body.model)
        try:
            system, user = segment_prompt(
                body.request(model, config.code_max_reasoning_chars), body.key, config.code_chunk_shots)
        except (PipelineFailure, KeyError, SegmentNotReady) as exc:
            return _segment_error(exc)
        return {"system": system, "user": user}

    @app.post("/v2/code/segment-parse")
    async def code_segment_parse(body: SegmentParseBody):
        """CR-050 FR-5: check a reply written outside the pipeline for one segment."""
        _, model = providers.for_model(providers.hive, body.model)
        try:
            fingerprint, content = parse_segment(
                body.request(model, config.code_max_reasoning_chars), body.key, body.reply,
                config.code_chunk_shots)
        except (PipelineFailure, KeyError, SegmentNotReady, ExtractError, ValueError) as exc:
            return _segment_error(exc)
        return {"fingerprint": fingerprint, "content": content}

    return app
