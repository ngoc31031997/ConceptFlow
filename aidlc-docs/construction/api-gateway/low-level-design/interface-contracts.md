# Interface Contracts — Unit 9: API Gateway

## Routing Table (GUI-facing REST/SSE, tất cả prefix `/v1/`)

| Path | Method | Proxy tới | Ghi chú |
|---|---|---|---|
| `/v1/plugins` | GET | Content Plugin Service (`GET /v1/plugins`) | passthrough |
| `/v1/sagas/render` | POST | Orchestrator Service (`POST /v1/sagas/render`) | passthrough |
| `/v1/sagas/publish` | POST | Orchestrator Service (`POST /v1/sagas/publish`) | passthrough |
| `/v1/projects/{project_id}` | GET | Orchestrator Service (`GET /v1/projects/{project_id}`) | passthrough |
| `/v1/projects/{project_id}/retry` | POST | Orchestrator Service (`POST /v1/projects/{project_id}/retry`) | passthrough |
| `/v1/auth/youtube/start` | GET | Publisher Service (`GET /v1/auth/youtube/start`) | passthrough, forward `302` redirect nguyên trạng |
| `/v1/auth/youtube/callback` | GET | Publisher Service (`GET /v1/auth/youtube/callback`) | passthrough |
| `/v1/progress/{project_id}` | GET (SSE) | Gateway tự xử lý — KHÔNG proxy | AMQP consumer (`progress.fanout`) fan-out theo `project_id` |
| `/health` | GET | Gateway tự xử lý | `200 {status: "ok"}`, không phụ thuộc downstream |

## Passthrough Behavior (Question 3, 7)
Mọi endpoint passthrough forward NGUYÊN TRẠNG: request method, headers (bao gồm `X-Request-ID`), body, query params. Response: status code + body nguyên trạng từ service downstream — Gateway KHÔNG transform/validate lại payload (validation là trách nhiệm service downstream, zero-trust đã xử lý ở từng unit).

## SSE Endpoint: `GET /v1/progress/{project_id}`
- **Response**: `200`, `Content-Type: text/event-stream`, giữ kết nối mở.
- **Event format**: `data: <JSON của ProgressMessage>\n\n` (ADR-0017's format, nguyên trạng từ Orchestrator — Gateway không transform).
- **Kết thúc**: Client tự đóng kết nối (đóng tab, hoặc logic GUI dừng theo dõi sau khi thấy `status: "completed"`/`"failed"` cho bước cuối); Gateway không tự đóng.

## API Versioning (Question 5)
Không tự định nghĩa version riêng ở Gateway — forward nguyên trạng prefix `/v1/` đã có ở service downstream (URI versioning, nhất quán Unit 2/7/8). SSE endpoint theo cùng convention.

## Correlation ID Propagation (Question 6, BẮT BUỘC — Gateway là entry point)
- **Inbound**: Nếu GUI request có header `X-Request-ID`, giữ nguyên. Nếu KHÔNG có, Gateway SINH MỚI (UUID v4).
- **Outbound**: Forward `X-Request-ID` nguyên trạng tới service downstream trong mọi request proxy.
- **Logging**: Mọi log request/response ở Gateway kèm `X-Request-ID`.
- **SSE**: `project_id` là correlation chính (không có request/response 1-lần để gắn `X-Request-ID`).

## Error Contract
Lỗi từ service downstream (4xx/5xx) forward nguyên trạng (status + body JSON). Lỗi mạng/timeout khi gọi downstream (Gateway không kết nối được) → Gateway trả `502 Bad Gateway` với body `{ "error": "upstream_unavailable", "service": "<tên service>" }`.

## Internal Client Contract (clients/httpClient.js)
```js
// Pseudocode interface (JSDoc)
/**
 * @typedef {Object} HttpClient
 * @property {(req: ProxyRequest) => Promise<ProxyResponse>} request
 */
```
