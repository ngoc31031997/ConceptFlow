"""The two light tasks that used to live in the orchestrator's ollama_client.go
(CR-014 suggest-metadata, CR-026 suggest-short-script). Prompts and guard rails
are carried over unchanged; only the transport moved.
"""

from __future__ import annotations

import json
import re
from dataclasses import dataclass, field

from app import errors
from app.errors import LLMError, Usage
from app.provider import ChatRequest, Provider

ENGLISH_NAMES = {"vi": "Vietnamese", "en": "English"}
DEFAULT_LANGUAGE = "en"  # mirrors domain.defaultLanguage in the orchestrator

MAX_TITLE_LENGTH = 100  # YouTube counts characters, not bytes.
# Ollama defaults num_ctx to 2048 tokens; a ten-minute script overflows it and
# the model then returns a well-formed but empty object (CR-014, measured).
# A title and description are drawn from the opening, where the topic is stated.
MAX_SCRIPT_CHARS = 4000
SUGGEST_MAX_ATTEMPTS = 2


def language_name(language: str) -> str:
    return ENGLISH_NAMES.get(language, ENGLISH_NAMES[DEFAULT_LANGUAGE])


def truncate_script(script: str) -> str:
    return script[:MAX_SCRIPT_CHARS]  # str slicing is by code point, never mid-rune


def truncate_title(title: str) -> str:
    return title[:MAX_TITLE_LENGTH]


def normalize_tags(raw) -> list[str]:
    """Never returns None: the API contract says tags is an array and the GUI
    crashes on `tags.join` the moment a model omits the key."""
    if isinstance(raw, list):
        return [str(t) for t in raw]
    if isinstance(raw, str):
        return [p.strip() for p in raw.split(",") if p.strip()]
    return []


def build_suggest_prompt(script_content: str, category_hint: str, language: str) -> str:
    script_content = truncate_script(script_content)
    name = language_name(language)
    topic = json.dumps(category_hint, ensure_ascii=False)
    return f"""You are a YouTube SEO expert. Based on the video script below (topic: {topic}), produce:
- "title": a compelling, SEO-friendly title, at most {MAX_TITLE_LENGTH} characters, written in {name}.
- "description": a 2-4 sentence SEO-friendly description with relevant keywords, written in {name}.
- "tags": an array of 5-10 relevant keywords, written in {name}, with no "#" characters.

Every piece of text you return must be written in {name}.

Return exactly one JSON object with exactly the three keys "title", "description" and "tags". Do not add any explanation.

Script:
{script_content}"""


def build_short_script_prompt(topic: str, source_script_content: str, language: str) -> str:
    name = language_name(language)
    context = topic
    if source_script_content.strip():
        context = (
            f"{topic}\n\nExisting long-form script on this topic, for context only — do not summarize or "
            f"transform its code, just reuse the topic and key idea it teaches:\n"
            f"{truncate_script(source_script_content)}"
        )
    return f"""You are writing a SHORT-FORM educational video script (Manim Community Edition v0.18) for YouTube Shorts/TikTok — 30 to 60 seconds of narration, ONE idea, no filler, hook in the first 2 seconds. This is NOT a trimmed-down version of a longer video; it must stand completely on its own.

Topic:
{context}

Hard rules (the render pipeline rejects anything that breaks these):
1. Start with exactly: from conceptflow import *
2. Exactly one class, inheriting ConceptFlowScene, name ending in "Scene":
   class <TopicName>Scene(ConceptFlowScene):
       def construct(self):
           ...
3. The ENTIRE body of construct() must be wrapped in exactly one:
   with self.clip("short"):
       ...
   This is mandatory, not optional — it is how the pipeline recognizes this as a Shorts/TikTok clip.
4. Every line of narration is a call, not a comment: self.narrate("...")
5. Do not set colors, font sizes, or backgrounds by hand — the design system (ConceptFlowScene) handles all of that.
6. Available components: TitleCard, Callout, CodePanel, StepList, ComparisonSplit, Recap, FlowDiagram, BarChart, FunctionPlot, DataTable, Timeline. Available scene methods: self.narrate("...", animation...) (animations passed in run WHILE the line is spoken, so the picture keeps moving), self.hook("question", animation...) (no title card: build the opening visual first), self.recap(narration="...") (returns to the wide shot with the hero in its final state, no bullet list), self.call_to_action(...), self.title/heading/body/caption/formula/code(...), self.stack/row/fit(...), self.connect/outline(...), self.reveal/dismiss/swap/emphasize/clear_stage(...).
7. All narration text must be written in {name}.

Before answering, check: does construct() start with a single with self.clip("short"): wrapping everything else? Is there exactly one Scene class? Is every spoken line a self.narrate(...) call, not a comment?

Respond with ONLY one Python code block (wrapped in ```python ... ```), no explanation before or after it."""


_FENCE = re.compile(r"```[a-zA-Z0-9]*\r?\n(.*?)\r?\n?```", re.DOTALL)


def strip_code_fence(script: str) -> str:
    m = _FENCE.search(script.strip())
    return m.group(1) if m else script


@dataclass
class TaskOutcome:
    value: dict
    usage: Usage = field(default_factory=Usage)


async def suggest_metadata(
    provider: Provider, script_content: str, category_hint: str, language: str, on_progress=None
) -> TaskOutcome:
    """A small local model asked only for "valid JSON" sometimes returns a
    well-formed object with an empty title. That parses fine, so it is checked
    here and retried once; a title that is still empty is an error, not blank
    fields the Creator might publish as-is."""
    prompt = build_suggest_prompt(script_content, category_hint, language)
    last: LLMError | None = None
    total = Usage()
    for _ in range(SUGGEST_MAX_ATTEMPTS):
        try:
            res = await provider.chat(ChatRequest(user=prompt, json_mode=True), on_progress)
        except LLMError as err:
            last = err
            total = total + err.usage
            if err.kind in (errors.AUTH, errors.BALANCE, errors.NOT_CONFIGURED):
                break
            continue
        total = total + res.usage
        try:
            data = json.loads(res.content)
        except json.JSONDecodeError as exc:
            last = LLMError(errors.MALFORMED, provider.name, f"parse model output as JSON: {exc}", res.usage)
            continue
        if not isinstance(data, dict):
            last = LLMError(errors.MALFORMED, provider.name, "model output is not a JSON object", res.usage)
            continue
        title = truncate_title(str(data.get("title") or "").strip())
        if not title:
            last = LLMError(errors.EMPTY, provider.name, "model returned no title", res.usage)
            continue
        return TaskOutcome(
            {
                "title": title,
                "description": str(data.get("description") or "").strip(),
                "tags": normalize_tags(data.get("tags")),
            },
            total,
        )
    assert last is not None
    last.usage = total
    raise last


async def suggest_short_script(
    provider: Provider, topic: str, source_script_content: str, language: str, on_progress=None
) -> TaskOutcome:
    # No JSON mode: the output is multi-line Python, and forcing JSON would make
    # the model escape every newline and quote — an easier way to get invalid
    # Python back than plain text is.
    prompt = build_short_script_prompt(topic, source_script_content, language)
    last: LLMError | None = None
    total = Usage()
    for _ in range(SUGGEST_MAX_ATTEMPTS):
        try:
            res = await provider.chat(ChatRequest(user=prompt), on_progress)
        except LLMError as err:
            last = err
            total = total + err.usage
            if err.kind in (errors.AUTH, errors.BALANCE, errors.NOT_CONFIGURED):
                break
            continue
        total = total + res.usage
        script = strip_code_fence(res.content).strip()
        if not script:
            last = LLMError(errors.EMPTY, provider.name, "model returned an empty script", res.usage)
            continue
        return TaskOutcome({"script": script}, total)
    assert last is not None
    last.usage = total
    raise last
