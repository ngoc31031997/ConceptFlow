# Interface Contracts — Unit 4: Script Processing Service

## Script Syntax (Markdown, Question 3)
`raw_script` phải theo cú pháp Markdown sau:
- Mỗi scene bắt đầu bằng heading `## Scene N` (N là số thứ tự, dùng để validate nhưng `scene_index` thực tế trong output là 0-based theo thứ tự xuất hiện, không phải giá trị N)
- Dòng bắt đầu bằng `> ` (blockquote) ngay dưới heading là `illustration_hint`
- Đoạn text thường (không phải blockquote, không phải code fence) là `narration_text`
- Code fence (```` ``` ````, ngôn ngữ tùy chọn) nếu có là `code_snippet`
- Script phải có ít nhất 1 scene; mỗi scene phải có `narration_text` không rỗng (khớp Content Plugin Service's domain rule cho `Scene.narration_text`)

**Ví dụ hợp lệ**:
```markdown
## Scene 1
> minh họa vòng lặp for
Đây là lời thoại giải thích vòng lặp for hoạt động như thế nào.
\`\`\`python
for i in range(10):
    print(i)
\`\`\`
```

**Lỗi cú pháp** (→ `ScriptSyntaxError`, Question 8): thiếu heading `## Scene N` nào (không tìm thấy scene hợp lệ trong toàn bộ script), `narration_text` rỗng cho 1 scene, code fence không đóng.

## AMQP Consumer: command `parse_script`
Queue: `script_processing.commands` (theo `unit-of-work.md`/`component-methods.md`).

**Command payload** (envelope chuẩn, theo `messaging-design.md` Unit 1):
```json
{
  "message_id": "uuid",
  "saga_id": "uuid",
  "project_id": "string",
  "schema_version": 1,
  "timestamp": "ISO-8601",
  "payload": {
    "raw_script": "string (Markdown, xem script-syntax.md)"
  }
}
```

## AMQP Producer: event `script_parsed` / `parse_failed`
Publish tới `orchestrator.events`.

**`script_parsed`** (thành công — scenes CHƯA có `category`, sẽ được Content Plugin Service gắn ở bước Saga tiếp theo, theo Question 4 = B):
```json
{
  "message_id": "uuid",
  "saga_id": "uuid",
  "project_id": "string",
  "schema_version": 1,
  "timestamp": "ISO-8601",
  "payload": {
    "scenes": [
      {
        "scene_index": 0,
        "narration_text": "string",
        "illustration_hint": "string",
        "code_snippet": "string | null"
      }
    ]
  }
}
```

**`parse_failed`** (lỗi cú pháp — Business Rule Question 8):
```json
{
  "message_id": "uuid",
  "saga_id": "uuid",
  "project_id": "string",
  "schema_version": 1,
  "timestamp": "ISO-8601",
  "payload": {
    "error_message": "string",
    "line_number": "integer | null",
    "reason": "string"
  }
}
```

## Delivery Guarantee & Idempotency
At-least-once (kế thừa Unit 1). Manual ack sau khi xử lý xong (kể cả khi kết quả là `parse_failed` — lỗi cú pháp không tự phục hồi qua retry).

**Revision (2026-08-07, ADR-0013)**: Idempotency dedupe (`IdempotencyStore`) và việc publish event đều chuyển sang Inbox/Outbox pattern PostgreSQL-backed (chi tiết schema ở NFR Design):
- **Inbox**: bảng `processed_messages` — dedupe `message_id` bền vững (không mất khi restart, khác với `set[message_id]` in-memory trước đây).
- **Outbox**: consumer ghi event `script_parsed`/`parse_failed` vào bảng `outbox_events` trong CÙNG transaction với việc ghi Inbox (atomicity), KHÔNG publish trực tiếp lên RabbitMQ. `OutboxRelay` (background poller) publish các row chưa gửi.
- **Hệ quả**: publish event trở thành "eventually" thay vì hoàn toàn tức thời (độ trễ tối đa = chu kỳ poll của Relay, dự kiến ~1s) — chấp nhận được vì Saga vốn đã bất đồng bộ.

## Correlation ID
`saga_id` từ envelope AMQP được gắn vào mọi log line qua `adapters/logging/correlation.py`. Không có REST endpoint public cho unit này (Question 4 = B, chỉ AMQP) → N/A cho `X-Saga-ID` HTTP header / URI versioning (ADR-0008).

## Content Plugin Service Integration
**Không có tích hợp trực tiếp** ở Unit 4 (Question 4 = B) — Script Processing Service publish `script_parsed` với scene chưa có category, kết thúc trách nhiệm của unit này. Orchestrator (Unit 8, chưa phát triển) sẽ điều phối bước Saga tiếp theo (`classify_scenes` tới Content Plugin Service) sau khi nhận `script_parsed`, theo đúng thiết kế Saga ở `services.md`.
