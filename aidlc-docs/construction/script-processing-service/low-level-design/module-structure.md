# Module Structure — Unit 4: Script Processing Service

**Revision (2026-08-07, ADR-0013)**: `idempotency.py` (in-memory) thay bằng `adapters/persistence/` (PostgreSQL — Inbox/Outbox pattern system-wide retrofit). Xem `nfr-design/` khi unit này tới NFR Design để biết schema đầy đủ; module structure dưới đây đã phản ánh vị trí các module liên quan.

## Layering (Hexagonal / Ports & Adapters — ADR-0002)

```
services/script-processing/
├── domain/
│   ├── models.py               # Scene, ParsedScript (value objects)
│   ├── errors.py                # ScriptSyntaxError(line_number, reason)
│   └── ports.py                 # ScriptParserPort (abstract interface)
├── application/
│   └── parse_script.py          # ParseScriptUseCase
├── adapters/
│   ├── messaging/
│   │   ├── consumer.py           # AMQP consumer cho command parse_script
│   │   └── producer.py           # AMQP producer — KHÔNG publish trực tiếp; ghi vào Outbox (ADR-0013)
│   ├── parsing/
│   │   └── markdown_parser.py    # MarkdownScriptParser implements ScriptParserPort
│   ├── persistence/
│   │   ├── db.py                  # SQLAlchemy engine/session, bootstrap schema (Postgres, ADR-0013)
│   │   ├── inbox.py               # InboxRepository — durable dedupe (processed_messages table), thay IdempotencyStore in-memory
│   │   ├── outbox.py              # OutboxRepository — ghi event chờ publish (outbox_events table)
│   │   └── relay.py               # OutboxRelay — polling background task, publish event chưa gửi, đánh dấu published_at
│   └── logging/
│       └── correlation.py        # saga_id injection vào log context
├── main.py                      # Composition root — wiring, AMQP consumer + OutboxRelay startup
└── tests/
    ├── domain/
    ├── application/
    └── adapters/
```

## Dependency Direction
`adapters/` → `application/` → `domain/`. `domain/` không import FastAPI/aio-pika/markdown library. `application/` chỉ phụ thuộc `domain/` qua `ScriptParserPort` (abstraction) — không biết chi tiết cú pháp Markdown cụ thể nào. `adapters/parsing/markdown_parser.py` implement `domain/ports.py::ScriptParserPort` — cho phép đổi cú pháp script sau này (vd. thêm hỗ trợ YAML) mà không sửa `application/`.

## Module Responsibilities

| Module | Responsibility |
|---|---|
| `domain/models.py` | `Scene` (`scene_index`, `narration_text`, `illustration_hint`, `code_snippet?`), `ParsedScript` (`scenes: list[Scene]`) |
| `domain/errors.py` | `ScriptSyntaxError(line_number, reason)` — lỗi cú pháp có vị trí cụ thể (Business Rule, Question 8) |
| `domain/ports.py` | `ScriptParserPort` (abstract: `parse(raw_script: str) -> ParsedScript`, raise `ScriptSyntaxError`) |
| `application/parse_script.py` | `ParseScriptUseCase(parser: ScriptParserPort)` — gọi parser, trả `ParsedScript` hoặc để lỗi propagate lên consumer |
| `adapters/parsing/markdown_parser.py` | `MarkdownScriptParser` — parse cú pháp Markdown với scene delimiter (`## Scene N`, blockquote `>` cho illustration hint, code fence cho code_snippet) |
| `adapters/messaging/consumer.py` | Consume `parse_script` (queue `script_processing.commands`); trong 1 DB transaction: kiểm tra `InboxRepository` (đã xử lý `message_id` chưa), gọi `ParseScriptUseCase`, ghi kết quả vào `OutboxRepository` (event `script_parsed`/`parse_failed`), ghi `message_id` vào Inbox — commit transaction, ack message |
| `adapters/messaging/producer.py` | Publish thực tế lên RabbitMQ — chỉ được gọi bởi `OutboxRelay`, KHÔNG gọi trực tiếp từ `consumer.py` (đảm bảo atomicity giữa xử lý + ghi outbox) |
| `adapters/persistence/db.py` | Kết nối PostgreSQL (`script-processing-db`, ADR-0013), bootstrap bảng `outbox_events`/`processed_messages` lúc khởi động |
| `adapters/persistence/inbox.py` | `InboxRepository` — `has_processed(message_id) -> bool`, `mark_processed(message_id)` (bảng `processed_messages`) |
| `adapters/persistence/outbox.py` | `OutboxRepository` — `enqueue(aggregate_id, event_type, payload)` (bảng `outbox_events`, `published_at = NULL`) |
| `adapters/persistence/relay.py` | `OutboxRelay` — background task poll bảng `outbox_events` định kỳ, publish row `published_at IS NULL` qua `producer.py`, đánh dấu `published_at` |
| `adapters/logging/correlation.py` | Gắn `saga_id` (từ message envelope) vào mọi log line |
