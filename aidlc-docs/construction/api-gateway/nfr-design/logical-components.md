# Logical Components — Unit 9: API Gateway

> **Cập nhật (CR-055, 2026-09-30)**: Content Plugin Service (và route `GET /v1/plugins`) gỡ ở CR-020; từ CR-040 Gateway proxy thêm tới Authoring Service (`AUTHORING_SERVICE_URL`). Danh sách route thật (≈90 route) ở `services/api-gateway/src/routes/`; bảng dưới chỉ giữ các route gốc của Unit 9.

## Component Diagram (logical, technology-agnostic)

```
┌──────────────────────────────────────────────────────────┐
│                       API Gateway                          │
│                                                              │
│  ┌────────────────┐        ┌───────────────────────────┐   │
│  │  Express Server  │        │   AMQP Consumer            │   │
│  │  (routes)         │        │   (progress.fanout)         │   │
│  └────────┬─────────┘        └───────────┬─────────────────┘   │
│           │                              │                     │
│           ▼                              ▼                     │
│  ┌─────────────────┐          ┌──────────────────────────┐    │
│  │  proxyHandler     │          │  progressHandler           │    │
│  └────────┬─────────┘          └────────────┬──────────────┘    │
│           │                                 │                   │
│           ▼                                 ▼                   │
│  ┌─────────────────┐          ┌──────────────────────────┐    │
│  │  httpClient × 3    │          │  SSE Connection Registry   │    │
│  │  (Orch/Auth/Pub)   │          │  Map<project_id, Res[]>    │    │
│  └────────┬─────────┘          └────────────┬──────────────┘    │
└───────────┼──────────────────────────────────┼───────────────────┘
            │                                  │
    ┌───────┴────────┬─────────────┐           ▼
    ▼                ▼             ▼      SSE stream tới GUI
Orchestrator   Authoring Svc    Publisher
```

## Components

| Component | Type | Responsibility |
|---|---|---|
| Express Server | Adapter, inbound | Nhận REST request từ GUI, áp dụng `middleware/correlation.js`, route tới handler |
| AMQP Consumer | Adapter, inbound | Consume `progress.fanout` (exclusive queue riêng Gateway), gọi `progressHandler` |
| `proxyHandler` | Handler | Forward request nguyên trạng tới service downstream qua `httpClient`, trả response nguyên trạng |
| `progressHandler` | Handler | Đăng ký/hủy SSE connection theo `project_id`, ghi event khi nhận AMQP message |
| `httpClient` × 3 | Adapter, outbound | HTTP client tới Orchestrator/Authoring Service/Publisher, forward header (bao gồm `X-Request-ID`) |
| SSE Connection Registry | In-memory state | `Map<project_id, Response[]>` — không persist, mất khi Gateway restart (client tự reconnect) |

## Infrastructure Elements — Không áp dụng
- **Cache**: không cần.
- **Circuit Breaker**: không cần.
- **Rate Limiter**: không cần.
- **Database**: không cần (Postgres/Redis) — Gateway hoàn toàn stateless ngoại trừ in-memory SSE registry.

## Scaling Boundaries
- Fixed 1 instance (nhất quán các unit khác) — SSE connection registry chỉ đúng đắn với 1 instance duy nhất (không có shared state giữa nhiều instance).
- Node.js event loop xử lý request/SSE connection đồng thời không cần cấu hình riêng ở quy mô này.
