# Code Generation Plan — Unit 10: Web GUI

## Coding Standards (đề xuất, xin xác nhận)
- TypeScript strict mode, functional components + hooks (không class component).
- Naming: `PascalCase` cho component file/tên, `camelCase` cho hàm/biến, `useXxx` cho custom hook.
- CSS Modules (`*.module.css`) per component, không dùng UI library ngoài (đã chốt Functional Design).
- ESLint (typescript-eslint + react-hooks plugin) + Prettier defaults.
- JSDoc chỉ khi cần giải thích lý do phi hiển nhiên (không doc block dài dòng).
- Test: Vitest + React Testing Library, file `*.test.tsx`/`*.test.ts` cạnh module hoặc trong `tests/`.

## Steps

1. **Project scaffolding**: `services/web-gui/` — `package.json` (Vite + React + TS + react-router-dom), `vite.config.ts`, `tsconfig.json`, `index.html`, ESLint/Prettier config.
2. **Types**: `src/types/index.ts` (Project, RenderInput, ProgressMessage, Plugin, PublishMetadata).
3. **API client layer**: `src/api/client.ts` (7 hàm theo interface-contracts.md) + `tests/api/client.test.ts` (mock `fetch`/`EventSource`).
4. **Context**: `src/context/ProjectDraftContext.tsx` (reducer + actions) + test.
5. **Hooks**: `src/hooks/useSSE.ts`, `src/hooks/useProject.ts` + tests (mock EventSource/fetch).
6. **Leaf components** (Rule 6 data-testid): `ScriptEditor`, `PluginSelector`, `VoiceLanguageSelector`, `BackgroundMusicPicker`, `ProgressTracker`, `ErrorBanner`, `VideoPlayer`, `YoutubeConnectButton`, `PublishForm` + tests cho mỗi (render + interaction + validation).
7. **Pages**: `NewProjectPage`, `RenderPage`, `ResultPage` (compose components, wire API/hooks, state-driven navigation theo Rule 3) + tests.
8. **Composition root**: `App.tsx` (routes + Context Provider), `main.tsx`.
9. **Styling**: CSS module cơ bản cho từng component (layout tối thiểu, không cần polish thẩm mỹ sâu).
10. **Documentation**: `services/web-gui/README.md`, cập nhật root `README.md`, doc summary tại `aidlc-docs/construction/web-gui/code/README.md`.
11. **Deployment artifacts**: `Dockerfile` (multi-stage), `nginx.conf`, thêm `web-gui` service vào root `docker-compose.yml` (theo deployment-architecture.md), validate `docker compose config`.

Business Logic/Repository/DB Migration steps: N/A (GUI không có business logic backend, không database — đã xác nhận ở Functional/NFR Design).
