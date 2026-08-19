# Script Processing Service — Code Summary

**Revision (2026-08-07)**: added `Scene.code_language` — discovered missing while designing Unit 5 (Rendering
Service): Story B3 requires the code snippet's programming language for correct syntax highlight, and the
Markdown code fence (```` ```python ````) already carries it, but the parser was discarding it. Fixed in
`markdown_parser.py` (captures the fence's language annotation) and `producer.py` (includes it in the
`script_parsed` event payload).

## Structure
Implements the module structure from `low-level-design/module-structure.md` (Hexagonal / Ports & Adapters), following the exact shape established by Content Plugin Service and TTS Service (post-retrofit):

```
services/script-processing/
├── domain/            # Scene, ParsedScript, ScriptSyntaxError, ScriptParserPort
├── application/         # ParseScriptUseCase (thin — delegates to the parser port)
├── adapters/
│   ├── parsing/          # MarkdownScriptParser implements ScriptParserPort (ADR-0011)
│   ├── messaging/         # consumer.py (parse_script command), producer.py (envelope builders)
│   ├── persistence/       # db.py, inbox.py, outbox.py, relay.py (ADR-0013, copied verbatim from Unit 2/3 — no service-specific logic)
│   └── logging/           # correlation.py — saga_id from AMQP envelope
├── main.py               # Composition root — plain asyncio entrypoint, writes /tmp/ready sentinel
└── tests/
```

## No REST Layer
Per ADR-0012, this service does not call Content Plugin Service directly and has no inbound REST endpoint — it is pure AMQP consumer/producer, identical in shape to TTS Service post-retrofit.

## Key Behaviors Implemented
- **Markdown parsing** (business-logic-model.md): scans for `## Scene N` headings, validates strictly
  sequential numbering starting at 1, extracts `narration_text` (mandatory), `illustration_hint` (optional,
  first blockquote line), and `code_snippet` (optional, at most one fenced code block per scene).
- **Fail-fast** (Business Rule 7): the parser raises `ScriptSyntaxError(line_number, reason)` at the first
  violation encountered, scene-by-scene in document order.
- **Outbox/Inbox** (ADR-0013): consumer writes `script_parsed`/`parse_failed` to `outbox_events` in the same
  DB transaction as marking the Inbox; `OutboxRelay` (background polling task, ~1s) is the only thing that
  actually publishes to RabbitMQ.
- **No artifact-level idempotency**: unlike TTS Service, there is no file/side-effect to check for reuse —
  parsing is pure computation, so only message-level (Inbox) dedupe applies.

## Tests
- `tests/domain/` — value object sanity checks.
- `tests/application/test_parse_script.py` — thin `ParseScriptUseCase` delegation tests against a
  `FakeScriptParser`.
- `tests/adapters/test_markdown_parser.py` — full grammar coverage: valid script, multi-scene, optional
  fields, content-before-first-heading, no-headings error, non-sequential numbering error, empty-narration
  error, multiple-code-fence error, unterminated-fence error, fail-fast ordering.
- `tests/adapters/test_messaging.py` — consumer + Inbox/Outbox via `FakePool`.
- `tests/adapters/test_persistence.py`, `test_relay.py` — copied verbatim from Unit 2/3 (identical shape).

28 tests passing, `ruff check` clean — verified under Python 3.12 via Docker. `main.py` import-sanity-checked
with `DATABASE_URL`/`RABBITMQ_URL` env vars set.
