# Module Structure — Unit 9: API Gateway

## Layering (Layered đơn giản — không phải Hexagonal, Question 1)
Gateway không có domain logic — chỉ routing/proxy/SSE fan-out, nên không cần ports/adapters đầy đủ như Unit 2–8.

```
services/api-gateway/
├── src/
│   ├── server.js              # Composition root — Express app, wiring clients, start HTTP server + AMQP consumer
│   ├── routes/
│   │   ├── plugins.js          # GET /v1/plugins
│   │   ├── sagas.js            # POST /v1/sagas/render, POST /v1/sagas/publish
│   │   ├── projects.js         # GET /v1/projects/:id, POST /v1/projects/:id/retry
│   │   ├── auth.js             # GET /v1/auth/youtube/start, GET /v1/auth/youtube/callback
│   │   ├── progress.js         # GET /v1/progress/:id (SSE)
│   │   └── health.js           # GET /health
│   ├── handlers/
│   │   ├── proxyHandler.js     # Generic passthrough proxy handler (dùng chung cho plugins/sagas/projects/auth)
│   │   └── progressHandler.js  # SSE connection lifecycle + fan-out lookup
│   ├── clients/
│   │   ├── httpClient.js       # Generic HTTP client wrapper (fetch/axios) tới service downstream, forward headers
│   │   └── amqpClient.js       # RabbitMQ consumer cho progress.fanout
│   ├── middleware/
│   │   └── correlation.js      # Sinh/forward X-Request-ID (Question 6)
│   └── config/
│       └── config.js           # env var loading (service URLs, RABBITMQ_URL)
├── tests/
│   ├── routes/
│   ├── handlers/
│   └── clients/
├── package.json
└── Dockerfile
```

## Dependency Direction
`routes/` → `handlers/` → `clients/`. `routes/` chỉ định nghĩa endpoint + gọi handler tương ứng, không chứa logic. `handlers/` chứa logic proxy/SSE, gọi `clients/` để thực hiện I/O thực tế. `clients/` không phụ thuộc ngược lại `handlers/`/`routes/`. `middleware/correlation.js` áp dụng ở tầng Express app (trước mọi route).

## Module Responsibilities

| Module | Responsibility |
|---|---|
| `server.js` | Composition root: tạo Express app, khởi tạo `httpClient` (3 base URL: Orchestrator/ContentPlugin/Publisher), khởi tạo `amqpClient`, đăng ký middleware + route, start HTTP server + AMQP consumer |
| `routes/plugins.js` | `GET /v1/plugins` → `proxyHandler` với target = Content Plugin Service |
| `routes/sagas.js` | `POST /v1/sagas/render`, `POST /v1/sagas/publish` → `proxyHandler` với target = Orchestrator Service |
| `routes/projects.js` | `GET /v1/projects/:id`, `POST /v1/projects/:id/retry` → `proxyHandler` với target = Orchestrator Service |
| `routes/auth.js` | `GET /v1/auth/youtube/start`, `GET /v1/auth/youtube/callback` → `proxyHandler` với target = Publisher Service (bao gồm forward `302` redirect nguyên trạng) |
| `routes/progress.js` | `GET /v1/progress/:id` → `progressHandler` (SSE, KHÔNG qua `proxyHandler` — không phải HTTP proxy, Question 3) |
| `routes/health.js` | `GET /health` → trả `200 {status: "ok"}`, không proxy |
| `handlers/proxyHandler.js` | Nhận request, forward method/headers (bao gồm `X-Request-ID`)/body tới target service qua `httpClient`, trả nguyên trạng status code + body (Question 7, Flow 4: lỗi downstream) |
| `handlers/progressHandler.js` | Đăng ký SSE connection vào `Map<project_id, Response[]>` (Question 4), set SSE header (`Content-Type: text/event-stream`), xóa khỏi map khi `req.on('close')` |
| `clients/httpClient.js` | Wrapper HTTP client (Node's `fetch`, built-in từ Node 18+), forward header, timeout mặc định |
| `clients/amqpClient.js` | Consume `progress.fanout` (exclusive queue riêng của Gateway), gọi callback `progressHandler` khi nhận message |
| `middleware/correlation.js` | Sinh `X-Request-ID` (UUID) nếu request chưa có header này; set vào `req` để mọi handler/log dùng chung |
| `config/config.js` | Đọc env var: `ORCHESTRATOR_URL`, `CONTENT_PLUGIN_URL`, `PUBLISHER_URL`, `RABBITMQ_URL` |
