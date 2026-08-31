# Logical Components — Unit 9: API Gateway

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
│  │  (Orch/CP/Pub)     │          │  Map<project_id, Res[]>    │    │
│  └────────┬─────────┘          └────────────┬──────────────┘    │
└───────────┼──────────────────────────────────┼───────────────────┘
            │                                  │
    ┌───────┴────────┬─────────────┐           ▼
    ▼                ▼             ▼      SSE stream tới GUI
Orchestrator   Content Plugin   Publisher
```

## Components

| Component | Type | Responsibility |
|---|---|---|
| Express Server | Adapter, inbound | Nhận REST request từ GUI, áp dụng `middleware/correlation.js`, route tới handler |
| AMQP Consumer | Adapter, inbound | Consume `progress.fanout` (exclusive queue riêng Gateway), gọi `progressHandler` |
| `proxyHandler` | Handler | Forward request nguyên trạng tới service downstream qua `httpClient`, trả response nguyên trạng |
| `progressHandler` | Handler | Đăng ký/hủy SSE connection theo `project_id`, ghi event khi nhận AMQP message |
| `httpClient` × 3 | Adapter, outbound | HTTP client tới Orchestrator/Content Plugin/Publisher, forward header (bao gồm `X-Request-ID`) |
| SSE Connection Registry | In-memory state | `Map<project_id, Response[]>` — không persist, mất khi Gateway restart (client tự reconnect) |

## Infrastructure Elements — Không áp dụng
- **Cache**: không cần.
- **Circuit Breaker**: không cần.
- **Rate Limiter**: không cần.
- **Database**: không cần (Postgres/Redis) — Gateway hoàn toàn stateless ngoại trừ in-memory SSE registry.

## Scaling Boundaries
- Fixed 1 instance (nhất quán các unit khác) — SSE connection registry chỉ đúng đắn với 1 instance duy nhất (không có shared state giữa nhiều instance).
- Node.js event loop xử lý request/SSE connection đồng thời không cần cấu hình riêng ở quy mô này.
