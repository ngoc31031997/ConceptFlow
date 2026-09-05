# Logical Components — Unit 10: Web GUI

## Component Diagram (logical, technology-agnostic)

```
┌───────────────────────────────────────────────┐
│                  Web GUI (SPA)                   │
│                                                    │
│  ┌──────────────┐   ┌───────────────────────┐    │
│  │  React App     │   │  ProjectDraftContext    │    │
│  │  (Vite build)  │   │  (Context + useReducer) │    │
│  └──────┬───────┘   └───────────┬───────────────┘    │
│         │                       │                    │
│         ▼                       ▼                    │
│  ┌──────────────┐   ┌───────────────────────┐    │
│  │  Pages/Components │   │  api/client.ts (fetch)  │    │
│  └──────┬───────┘   └───────────┬───────────────┘    │
│         │                       │                    │
│         ▼                       ▼                    │
│  ┌──────────────┐   ┌───────────────────────┐    │
│  │  EventSource   │   │  fetch (HTTP)            │    │
│  │  (SSE, native) │   │                           │    │
│  └──────┬───────┘   └───────────┬───────────────┘    │
└─────────┼───────────────────────┼─────────────────────┘
          │                       │
          ▼                       ▼
     API Gateway (Unit 9, port 8080)
```

## Components

| Component | Type | Responsibility |
|---|---|---|
| React App | Runtime | SPA render toàn bộ UI, routing (`react-router-dom`) |
| `ProjectDraftContext` | State container | Local state soạn project trước khi submit (React Context + useReducer) |
| Pages/Components | UI | Xem `frontend-components.md` (Functional Design) — chi tiết đầy đủ |
| `api/client.ts` | Adapter, outbound | Wrapper `fetch`, các hàm gọi API Gateway |
| `EventSource` | Browser native API | Kết nối SSE tới `GET /v1/progress/{id}` (qua Gateway), browser tự reconnect |

## Infrastructure Elements — Không áp dụng
- **Cache**: không cần.
- **Circuit Breaker**: không cần.
- **Rate Limiter**: không cần.
- **State management library ngoài**: không cần (React Context đủ).
- **Backend/Database**: không áp dụng — GUI là static SPA, phục vụ qua static file server (Nginx hoặc tương đương — xác nhận cụ thể ở Infrastructure Design).

## Scaling Boundaries
Không áp dụng — SPA client-side chạy hoàn toàn trong browser Creator, không có "instance" server cần scale (chỉ static file serving).
