# Dependency Injection — Unit 4: Script Processing Service

**Revision (2026-08-07, ADR-0013)**: `IdempotencyStore` (in-memory) thay bằng `InboxRepository`/`OutboxRepository` (PostgreSQL-backed).

## Mechanism
Constructor injection thủ công, nhất quán với Unit 2/Unit 3.

## What Gets Injected vs Constructed Directly
- **Injected (abstraction)**: `ScriptParserPort` — `ParseScriptUseCase` nhận instance implement `ScriptParserPort` qua constructor, cho phép thay `MarkdownScriptParser` bằng parser cú pháp khác sau này mà không sửa `application/`.
- **Constructed directly**: `InboxRepository`, `OutboxRepository`, `OutboxRelay` (helper hạ tầng gắn trực tiếp với PostgreSQL, không cần abstraction thay thế — chỉ có 1 implementation, tương tự cách `artifact_paths.py` được xử lý ở Unit 3).

## Composition Root
`main.py`:
1. Khởi tạo `MarkdownScriptParser()`.
2. Khởi tạo `ParseScriptUseCase(parser=markdown_parser)`.
3. Kết nối PostgreSQL (`adapters/persistence/db.py`), bootstrap schema (`outbox_events`, `processed_messages`).
4. Khởi tạo `InboxRepository(db)`, `OutboxRepository(db)`.
5. Kết nối RabbitMQ (`aio_pika.connect_robust`).
6. Wire `ParseScriptCommandHandler(use_case, inbox, outbox, db)`, đăng ký consumer cho queue `script_processing.commands`.
7. Khởi động `OutboxRelay(outbox, producer, db)` như background task (poll định kỳ, publish + đánh dấu `published_at`).

## Wiring Diagram
```
main.py
  ├── MarkdownScriptParser (implements ScriptParserPort)
  │     └── injected into → ParseScriptUseCase
  ├── InboxRepository, OutboxRepository (Postgres, ADR-0013)
  │     └── injected into → ParseScriptCommandHandler (AMQP consumer)
  │           └── consumes command "parse_script"
  │           └── ghi event "script_parsed"/"parse_failed" vào Outbox (không publish trực tiếp)
  └── OutboxRelay (background task)
        └── poll Outbox chưa publish → publish qua producer.py → đánh dấu published_at
```
