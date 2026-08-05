# Interface Contracts — Unit 2: Content Plugin Service

## REST API (versioned per ADR-0008)

### `GET /v1/plugins`
- **Called by**: API Gateway (Unit 9)
- **Response 200**:
```json
{
  "plugins": [
    { "plugin_id": "programming", "name": "Lập trình", "supported_categories": ["algorithm", "concept"] }
  ]
}
```
- **Correlation ID**: Header `X-Request-ID` — nếu Gateway không gửi, service tự sinh UUID mới; luôn có trong response header và mọi log line liên quan đến request đó.

## AMQP Interface (theo `component-methods.md`, Application Design)

### Consumer: command `classify_scenes` (queue `content_plugin.commands`)
```json
{
  "message_id": "uuid",
  "saga_id": "uuid",
  "project_id": "uuid",
  "schema_version": "1.0",
  "timestamp": "ISO8601",
  "payload": {
    "plugin_id": "programming",
    "scenes": [{ "scene_index": 0, "narration_text": "...", "illustration_hint": "..." }]
  }
}
```
- **Correlation ID**: `saga_id` trong envelope — mọi log line xử lý message này include `saga_id`.
- **Idempotency**: kiểm tra `message_id` đã xử lý chưa (in-memory dedupe set, TTL 24h khớp message TTL của Unit 1) trước khi classify lại.

### Producer: event `scenes_classified` (→ `orchestrator.events`, routing key `orchestrator`)
```json
{
  "message_id": "uuid",
  "saga_id": "uuid",
  "project_id": "uuid",
  "schema_version": "1.0",
  "timestamp": "ISO8601",
  "payload": {
    "event_type": "scenes_classified",
    "scenes": [{ "scene_index": 0, "category": "algorithm", "animation_template_id": "sorting_visualization" }]
  }
}
```

### Producer: event `classification_failed` (lỗi)
```json
{
  "...envelope...": "...",
  "payload": {
    "event_type": "classification_failed",
    "error_message": "Plugin 'unknown_plugin' not found"
  }
}
```

## API Versioning & Deprecation (ADR-0008)
- Prefix `/v1/` áp dụng cho mọi REST endpoint.
- Breaking change tương lai → `/v2/...`, giữ `/v1/` tối thiểu 1 chu kỳ phát triển song song.
- AMQP message versioning: field `schema_version` trong envelope, additive-only trong cùng version (theo `messaging-design.md`, Unit 1).
