# Code Generation Plan — Unit 9: API Gateway

## Unit Context
- **Stories**: A1 (soạn script), B1 (chọn plugin), C1 (khởi chạy render), C6 (theo dõi tiến trình — SSE), E1 (xác thực YouTube OAuth) — Gateway là lớp routing chung, không business logic riêng
- **Dependencies**: RabbitMQ (Unit 1), Orchestrator Service (Unit 8), Content Plugin Service (Unit 2), Publisher Service (Unit 7) — cả 4 đã Code Generation xong
- **Database entities owned**: Không có — Gateway hoàn toàn stateless
- **Service boundary**: Reverse-proxy/SSE bridge thuần túy — routing REST + AMQP-to-SSE cho progress, KHÔNG chứa business logic

## Code Location
- **Application code**: `services/api-gateway/` (workspace root)
- **Documentation**: `aidlc-docs/construction/api-gateway/code/`

## Step 3.5: Coding Standards (Confirmed từ Low-Level Design — không cần hỏi lại)
- **Naming convention**: JavaScript idiomatic — `camelCase` cho biến/hàm, `PascalCase` không cần thiết (không dùng class, chỉ factory function theo `dependency-injection.md`)
- **SOLID**: Áp dụng ở mức phù hợp quy mô — Dependency Inversion qua factory injection (`proxyHandler(client)`, `progressHandler(amqpClient)`), Single Responsibility rõ ràng theo module-structure.md (`routes` chỉ định nghĩa endpoint, `handlers` chứa logic, `clients` chứa I/O)
- **Documentation style**: JSDoc trên mọi exported function, giải thích WHY cho quyết định non-obvious (vd. tại sao ack AMQP message dù ghi SSE thất bại — trỏ `messaging-design.md`)
- **Linting/formatting**: ESLint (config mặc định `eslint:recommended`) + Prettier — chưa có config Node.js nào khác trong repo (unit Node.js đầu tiên)
- **DI mechanism**: Constructor injection thủ công qua factory function (dependency-injection.md) — không dùng DI container

## Execution Steps

### Step 1: Project Structure Setup
- [x] Tạo `services/api-gateway/` theo `module-structure.md`: `src/{routes,handlers,clients,middleware,config}/`, `tests/`
- [x] `package.json` (Node 20, dependencies: `express`, `amqplib`, `pino`, `uuid`; devDependencies: `jest`, `supertest`, `eslint`, `prettier`)

**Story**: N/A (infrastructure setup)

### Step 2: Supporting Modules Generation
- [x] `src/config/config.js` — env var loading (`ORCHESTRATOR_URL`, `CONTENT_PLUGIN_URL`, `PUBLISHER_URL`, `RABBITMQ_URL`, `PORT`)
- [x] `src/middleware/correlation.js` — sinh/forward `X-Request-ID` (Question 6, LLD)

**Story**: N/A (cross-cutting)

### Step 3: Client Layer Generation
- [x] `src/clients/httpClient.js` — generic HTTP client wrapper (Node's built-in `fetch`), forward header, timeout 30s (`AbortController`, NFR Requirements Question 2)
- [x] `src/clients/amqpClient.js` — RabbitMQ consumer cho `progress.fanout` (exclusive queue), auto-reconnect backoff 5s (NFR Requirements Question 4)

**Story**: N/A (infrastructure clients)

### Step 4: Client Layer Unit Testing
- [x] `tests/clients/httpClient.test.js` — mock `fetch`, test forward header/timeout/502 mapping
- [x] `tests/clients/amqpClient.test.js` — mock `amqplib`, test reconnect logic

### Step 5: Handler Layer Generation
- [x] `src/handlers/proxyHandler.js` — forward request nguyên trạng, map lỗi kết nối → 502 (LLD Flow 1, 4)
- [x] `src/handlers/progressHandler.js` — SSE connection registry (`Map<project_id, Response[]>`), subscribe/unsubscribe/fan-out (LLD Flow 3, NFR Design's ack-regardless-of-write-outcome)

**Story**: A1, B1, C1, C6, E1 (routing logic)

### Step 6: Handler Layer Unit Testing
- [x] `tests/handlers/proxyHandler.test.js` — fake client, test passthrough + error mapping
- [x] `tests/handlers/progressHandler.test.js` — fake response objects, test subscribe/fan-out/cleanup-on-close

### Step 7: Business/Handler Layer Summary
- [x] Tạo `aidlc-docs/construction/api-gateway/code/handler-layer-summary.md`

### Step 8: Route Layer Generation
- [x] `src/routes/plugins.js`, `src/routes/sagas.js`, `src/routes/projects.js`, `src/routes/auth.js`, `src/routes/progress.js`, `src/routes/health.js` (routing table đầy đủ, interface-contracts.md)

**Story**: A1, B1, C1, C6, E1 (REST surface)

### Step 9: Route Layer Unit Testing
- [x] `tests/routes/*.test.js` — `supertest` + fake handler, verify method/path mapping đúng routing table

### Step 10: Route Layer Summary
- [x] Tạo `aidlc-docs/construction/api-gateway/code/route-layer-summary.md`

### Step 11: Composition Root
- [x] `src/server.js` — wire toàn bộ theo `dependency-injection.md`'s Composition Root (6 bước)

### Step 12: Documentation Generation
- [x] Tạo `services/api-gateway/README.md` (per-unit — overview, prerequisites Node 20+, cách chạy `npm start`/Docker, env var, cách chạy test `npm test`)
- [x] Cập nhật root `README.md` — thêm API Gateway vào Project Structure/service list

### Step 13: Deployment Artifacts Generation
- [x] `services/api-gateway/Dockerfile` (theo `deployment-architecture.md`)
- [x] Thêm service `api-gateway` vào root `docker-compose.yml` (port `8080:8080` publish ra host — theo `deployment-architecture.md`'s reference)

---

## Traceability Summary
| Story | Covered by |
|---|---|
| A1 (soạn script) | Step 5, 8 (proxy tới Orchestrator qua GUI's future calls) |
| B1 (chọn plugin) | Step 5, 8 (`/v1/plugins` proxy) |
| C1 (khởi chạy render) | Step 5, 8 (`/v1/sagas/render` proxy) |
| C6 (theo dõi tiến trình) | Step 5, 8 (`/v1/progress/{id}` SSE) |
| E1 (xác thực YouTube) | Step 5, 8 (`/v1/auth/youtube/*` proxy) |

**Tổng**: 13 bước. Plan này là single source of truth cho Code Generation — không tạo logic ngoài những gì Low-Level Design/NFR Requirements/NFR Design/Infrastructure Design đã duyệt.
