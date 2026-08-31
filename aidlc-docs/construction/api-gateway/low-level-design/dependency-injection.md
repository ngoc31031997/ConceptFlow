# Dependency Injection — Unit 9: API Gateway

## Mechanism
Constructor injection thủ công qua factory function (Node.js idiom) — không dùng DI container (`InversifyJS`, `tsyringe`). Nhất quán tinh thần constructor injection thủ công đã dùng ở Unit 8 (Go), chuyển sang idiom Node.js/Express.

## What Gets Injected vs Constructed Directly
- **Injected**: `httpClient` (3 instance, mỗi instance gắn 1 base URL — Orchestrator/ContentPlugin/Publisher) truyền vào `proxyHandler` factory qua tham số; `amqpClient` (kết nối RabbitMQ) truyền vào `progressHandler` factory.
- **Constructed trực tiếp**: Express `app`, route router objects — đây là framework infrastructure, không phải business abstraction cần test độc lập.

## Composition Root
`src/server.js`:
1. Load config từ env var (`config/config.js`).
2. Khởi tạo `httpClient` cho 3 target: `orchestratorClient`, `contentPluginClient`, `publisherClient`.
3. Kết nối RabbitMQ (`amqpClient`), declare exclusive queue bind vào `progress.fanout`.
4. Khởi tạo Express `app`, đăng ký `middleware/correlation.js`.
5. Đăng ký route: `routes/plugins.js(contentPluginClient)`, `routes/sagas.js(orchestratorClient)`, `routes/projects.js(orchestratorClient)`, `routes/auth.js(publisherClient)`, `routes/progress.js(amqpClient)`, `routes/health.js()`.
6. Start HTTP server (`app.listen`), start AMQP consumer loop.

## Wiring Diagram
```
server.js
  ├── httpClient(ORCHESTRATOR_URL) ──┐
  ├── httpClient(CONTENT_PLUGIN_URL) ─┤── injected into → routes/*.js → proxyHandler
  ├── httpClient(PUBLISHER_URL) ─────┘
  │
  └── amqpClient(RABBITMQ_URL) ── injected into → routes/progress.js → progressHandler
```

## Testability
`proxyHandler`/`progressHandler` nhận client qua tham số factory (`proxyHandler(client)`) — unit test dùng fake client (object có method `.request()` trả response giả), không cần service downstream/RabbitMQ thật.

## Concurrency Model
Node.js single-threaded event loop — mỗi request xử lý bất đồng bộ (`async`/`await`), không cần đồng bộ hóa thủ công. SSE connection map (`Map<project_id, Response[]>`) an toàn vì event loop đơn luồng (không có race condition giữa các request JS, khác Go/Orchestrator cần transaction-level locking).
