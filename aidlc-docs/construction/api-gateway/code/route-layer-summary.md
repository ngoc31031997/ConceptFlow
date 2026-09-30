# Route Layer Summary — Unit 9: API Gateway

> **Cập nhật (CR-055, 2026-09-30)**: Content Plugin Service (và route `GET /v1/plugins`) gỡ ở CR-020; từ CR-040 Gateway proxy thêm tới Authoring Service (`AUTHORING_SERVICE_URL`). Danh sách route thật (≈90 route) ở `services/api-gateway/src/routes/`; bảng dưới chỉ giữ các route gốc của Unit 9.

Each `routes/*.js` module exports a factory that builds an Express `Router`, wiring the routing table from `interface-contracts.md` to `proxyHandler`/`progressHandler` with the correct injected client. Routes contain no logic of their own — only endpoint definitions.

| Module | Routes | Target client |
|---|---|---|
| `src/routes/sagas.js` | `POST /v1/sagas/render`, `POST /v1/sagas/publish` | `orchestratorClient` |
| `src/routes/projects.js` | `GET /v1/projects/:id`, `POST /v1/projects/:id/retry` | `orchestratorClient` |
| `src/routes/auth.js` | `GET /v1/auth/youtube/start`, `GET /v1/auth/youtube/callback` | `publisherClient` |
| `src/routes/progress.js` | `GET /v1/progress/:id` (SSE) | `progressHandler` instance directly (not a proxy — no downstream client) |
| `src/routes/health.js` | `GET /health` | none — always `200 {status: "ok"}`, no downstream dependency |

## Testing
`tests/routes/*.test.js` build a minimal Express app per module (mounting only the router under test, plus a stub `req.requestId` middleware) and use `supertest` with a fake client (`{ request: jest.fn() }`) or fake `progressHandler`/no client for `progress`/`health`, verifying HTTP method/path mapping matches the routing table, and that the OAuth `start` route forwards a 302 with `Location` verbatim.
