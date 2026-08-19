# Code Generation Plan — Unit 4: Script Processing Service

## Unit Context
- **Stories**: A2 (xem `unit-of-work-story-map.md`)
- **Dependencies**: Unit 1 (RabbitMQ) — đã hoàn tất; KHÔNG phụ thuộc Unit 2 (ADR-0012)
- **Interfaces**: AMQP consumer `parse_script` (queue `script_processing.commands`) → producer `script_parsed`/`parse_failed` (qua Outbox, ADR-0013)
- **Owned entities**: `outbox_events`/`processed_messages` (Postgres `script-processing-db`) — không có business data model khác

## Coding Standards (Step 3.5 — đã xác nhận ở Low-Level Design, nhất quán Unit 2/Unit 3)
- **Naming**: Python snake_case cho function/variable, PascalCase cho class
- **SOLID**: Bắt buộc — `ParseScriptUseCase` phụ thuộc `ScriptParserPort` abstraction, không phụ thuộc `MarkdownScriptParser` cụ thể
- **Documentation**: Python docstring (Google style), giải thích WHY
- **Linting**: `ruff`

## Steps

- [x] **Step 1 — Project Structure Setup**: Tạo `services/script-processing/` (domain/, application/, adapters/{messaging,parsing,persistence,logging}/, main.py, tests/), `requirements.txt`, `Dockerfile`
- [x] **Step 2 — Business Logic Generation (domain/)**: `models.py` (`Scene`, `ParsedScript`), `errors.py` (`ScriptSyntaxError`), `ports.py` (`ScriptParserPort`)
- [x] **Step 3 — Business Logic Generation (application/)**: `parse_script.py` (`ParseScriptUseCase`) — theo `business-logic-model.md`
- [x] **Step 4 — Business Logic Unit Testing**: `tests/domain/`, `tests/application/` — dùng `FakeScriptParser`, cover mọi business rule (Rule 1-7)
- [x] **Step 5 — Parser Adapter Generation**: `adapters/parsing/markdown_parser.py` (`MarkdownScriptParser` implements `ScriptParserPort` — heading detection, sequential numbering validation, blockquote/code-fence extraction, fail-fast)
- [x] **Step 6 — Parser Adapter Unit Testing**: `tests/adapters/test_markdown_parser.py` — cover cú pháp hợp lệ + mọi lỗi cú pháp (Rule 1-7) với `line_number`/`reason` cụ thể
- [x] **Step 7 — Persistence Adapter Generation**: `adapters/persistence/{db,inbox,outbox,relay}.py` (copy nguyên mẫu từ Unit 2/Unit 3, ADR-0013 — không có gì đặc thù cho unit này)
- [x] **Step 8 — Logging Adapter Generation**: `adapters/logging/correlation.py` (đọc `saga_id` từ envelope AMQP)
- [x] **Step 9 — Messaging Layer Generation**: `adapters/messaging/consumer.py` (`ParseScriptCommandHandler` — inbox check → parse → outbox enqueue + inbox mark → ack), `adapters/messaging/producer.py` (envelope builders `script_parsed`/`parse_failed`)
- [x] **Step 10 — Messaging Layer Unit Testing**: `tests/adapters/test_messaging.py` — mock AMQP + `FakePool` (mirror Unit 2/3's `fake_postgres.py`)
- [x] **Step 11 — Persistence/Relay Unit Testing**: `tests/adapters/test_persistence.py`, `test_relay.py` (copy nguyên mẫu từ Unit 2/3)
- [x] **Step 12 — Composition Root**: `main.py` — wiring toàn bộ (plain asyncio entrypoint, không FastAPI, giống Unit 3 sau retrofit), ghi sentinel `/tmp/ready`
- [x] **Step 13 — Documentation Generation**: Cập nhật `README.md` gốc, tạo `aidlc-docs/construction/script-processing-service/code/README.md`
- [x] **Step 14 — Deployment Artifacts**: Thêm service `script-processing` + `script-processing-db` + volume vào `docker-compose.yml` gốc, thêm `script.commands`... (đã có `script_processing.commands` từ Unit 1)

**Không áp dụng**: Repository Layer riêng ngoài Outbox/Inbox, Frontend Components, Database Migration Scripts (bootstrap qua `CREATE TABLE IF NOT EXISTS`, không migration tool), API Layer (không có REST, ADR-0012).
