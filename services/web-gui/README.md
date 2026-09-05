# Web GUI

React + TypeScript SPA (Vite) — giao diện Creator để soạn script, theo dõi render, xem video và đăng lên YouTube.

## Development

```bash
npm install
npm run dev       # http://localhost:5173
npm test          # Vitest + React Testing Library
npm run lint
npm run build
```

Cấu hình `VITE_API_BASE_URL` (mặc định `http://localhost:8080`, xem `.env.example`) — build-time env, trỏ tới API Gateway (Unit 9).

## Cấu trúc

```
src/
├── pages/       # NewProjectPage, RenderPage, ResultPage
├── components/  # UI components tái sử dụng
├── hooks/       # useSSE (progress), useProject (fetch + poll)
├── api/         # client.ts — toàn bộ lời gọi API Gateway
├── context/     # ProjectDraftContext (state soạn project trước submit)
└── types/       # Type dùng chung
```

## Docker

```bash
docker compose up --build web-gui   # http://localhost:3000
```

Xem chi tiết thiết kế tại `aidlc-docs/construction/web-gui/`.
