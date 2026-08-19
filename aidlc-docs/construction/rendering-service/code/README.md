# Rendering Service — Code Summary

## Structure
Implements the module structure from `low-level-design/module-structure.md` (Hexagonal / Ports & Adapters), following the shape established by Unit 2/3/4:

```
services/rendering/
├── domain/               # SceneRenderRequest/Result, errors, AnimationRendererPort, AnimationTemplatePort
├── application/
│   ├── render_scene.py               # RenderSceneUseCase — zero-trust validation + idempotency
│   └── render_scenes_batch.py        # RenderScenesBatchUseCase — fail-fast, on_scene_start/on_scene_rendered callbacks
├── adapters/
│   ├── rendering/
│   │   ├── registry.py                 # AnimationTemplateRegistry.discover() — dynamic plugin loading (ADR-0015)
│   │   ├── manim_renderer.py           # ManimAnimationRenderer implements AnimationRendererPort
│   │   └── templates/
│   │       ├── _code_display.py         # shared Code-mobject helper (Business Rule 3-4), not a template itself
│   │       ├── concept_illustration.py  # ConceptIllustrationTemplate
│   │       └── algorithm_visualization.py  # AlgorithmVisualizationTemplate
│   ├── messaging/         # consumer.py (render_scenes), producer.py (4 envelope builders)
│   ├── persistence/       # db.py, inbox.py, outbox.py, relay.py (ADR-0013, copied verbatim from Unit 2/3/4)
│   ├── storage/           # artifact_paths.py — shared-volume convention + ffprobe-based duration reading
│   └── logging/           # correlation.py
├── main.py               # Composition root — plain asyncio entrypoint, writes /tmp/ready sentinel
└── tests/
```

## Key Design Point: Manim Rendering Must Not Block the Event Loop
`RenderScenesBatchUseCase.execute()` can legitimately run for minutes (Manim rendering, up to
`RENDER_TIMEOUT_SECONDS` per scene, itself run in a `ThreadPoolExecutor` inside `ManimAnimationRenderer`).
Calling it directly from `consumer.py`'s `async def handle()` would block the asyncio event loop for that
entire duration — starving RabbitMQ heartbeats and `OutboxRelay`. `consumer.py` therefore runs the batch via
`asyncio.to_thread()`, with the batch's `on_scene_start`/`on_scene_rendered` callbacks bridging back into the
event loop via `asyncio.run_coroutine_threadsafe()` to perform their per-scene Outbox writes.

**Note**: this same pattern (a synchronous, internally-threadpooled use case called directly from an async
`consumer.handle()`) already exists in TTS Service — there it blocks the event loop for up to 60s per call.
That's a smaller window than Rendering Service's up to 300s+ per scene batch, so it wasn't flagged as an
issue at the time, but the same `asyncio.to_thread()` fix would apply there too if it becomes a problem
(out of scope for this unit — noted here for awareness, not changed).

## Manim Integration Details
- **Media output**: Manim writes through its own `media_dir`/`output_file` config, not to an arbitrary path —
  each render uses an isolated temp `media_dir` (`tempfile.mkdtemp`), and the resulting `.mp4` is located
  (`_find_rendered_file`, walks the temp dir for the first `.mp4`) and moved to the caller's `output_path`.
- **Duration measurement**: read via `ffprobe` (bundled with the `ffmpeg` system dependency already required
  by Manim) — `adapters/storage/artifact_paths.py::read_duration_seconds`.
- **Code display** (Story B3): `_code_display.py` builds a Manim `Code` mobject positioned in the left column,
  falling back to Pygments' plain-text lexer when `code_language` is missing/unrecognized (Business Rule 4) —
  shared by both templates so the placement rule isn't duplicated.
- **Duration matching** (Business Rule 2): each template computes its own animation content, then pads with
  `self.wait(remaining)` to approach the target `duration_seconds` — never cutting content short if the
  natural animation runs longer.
- **MVP content scope**: both templates render generic, working Manim scenes (title/narration text +
  optional code) rather than bespoke per-algorithm/per-concept animation art — the architecturally
  significant part of this unit is the pluggable template mechanism (ADR-0015) and duration-matching logic,
  not hand-crafted animation content for every possible programming topic.

## Tests
- `tests/domain/`, `tests/application/` — business rules against `FakeAnimationRenderer` (no real Manim).
- `tests/adapters/test_registry.py` — registry `get`/`list_all` against fakes constructed directly (not
  `discover()` against the real `templates/` package, which requires Manim to be importable — see below).
- `tests/adapters/test_manim_renderer.py` — `ManimAnimationRenderer`'s template lookup, timeout, and error
  mapping, with `_render_to_file` monkeypatched out entirely (so `manim` is never imported during tests).
- `tests/adapters/test_messaging.py` — consumer + Inbox/Outbox, verifies per-scene events land as separate
  Outbox rows (not batched into one transaction).
- `tests/adapters/test_persistence.py`, `test_relay.py` — copied verbatim from Unit 2/3/4.

30 tests passing, `ruff check` clean — verified under Python 3.12 via Docker (`python:3.12-slim` + `aio-pika`/
`asyncpg`/`pytest`, no `manim` installed — confirms the test suite genuinely doesn't require it). Non-Manim
modules (`consumer.py`, `producer.py`, `registry.py`, `manim_renderer.py`, both `application/` use cases)
import-sanity-checked the same way.

**Full production image** (with real `manim` + `ffmpeg`/`cairo`/`pango`, per `Dockerfile`) is the only way to
exercise `AnimationTemplateRegistry.discover()` against the real templates and an actual Manim render — this
was built and verified separately (see audit.md for the outcome recorded at Unit 5 completion).
