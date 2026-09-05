# Code Generation Summary — Unit 10: Web GUI

## Generated
`services/web-gui/` — React 18 + TypeScript SPA (Vite):
- `src/types/index.ts` — types dùng chung (Project, RenderInput, ProgressMessage, Plugin, PublishMetadata, SagaStartedResponse)
- `src/api/client.ts` — 7 hàm gọi API Gateway (Unit 9), `ApiError` + `GENERIC_CONNECTION_ERROR` cho lỗi hạ tầng (Rule 4)
- `src/context/ProjectDraftContext.tsx` — state soạn project (reducer, tách Context giá trị và Context dispatch)
- `src/hooks/useSSE.ts`, `src/hooks/useProject.ts`
- `src/components/*` — 9 leaf component (ScriptEditor, PluginSelector, VoiceLanguageSelector, BackgroundMusicPicker, ProgressTracker, ErrorBanner, VideoPlayer, YoutubeConnectButton, PublishForm), mỗi component gắn `data-testid` theo Rule 6
- `src/pages/*` — NewProjectPage, RenderPage, ResultPage (compose components, wire API/hooks, điều hướng theo Project.Status — Rule 3)
- `src/App.tsx`, `src/main.tsx` — composition root
- `Dockerfile` (multi-stage node:20-alpine build → nginx:alpine serve), `nginx.conf` (SPA fallback routing)
- Test suite: `tests/{api,hooks,context,components,pages}/` — 18 test cases (Vitest + React Testing Library)

## Deviations / Design Decisions Made During Generation
- **`project_id` generation**: LLD's sequence-flows.md không chỉ rõ ai sinh `project_id` trước request đầu tiên (`RenderInput.project_id` là bắt buộc nhưng response `startRenderSaga` chỉ trả `saga_id`). Quyết định: GUI tự sinh `project_id` bằng `crypto.randomUUID()` phía client trước khi gọi `POST /v1/sagas/render`, dùng chính ID đó để điều hướng sang `/projects/{id}/render` và mở SSE — không dùng `saga_id` cho việc điều hướng/subscribe (SSE endpoint là `/v1/progress/{projectId}`, không phải saga_id). Không thay đổi bất kỳ contract đã duyệt nào (`RenderInput.project_id` đã có sẵn trong `interface-contracts.md`).
- **Progress polling fallback interval**: `useProject`'s polling interval chọn 5s (không có trong NFR/LLD) — giá trị hợp lý cho fallback khi SSE gián đoạn, không phải cơ chế chính (SSE vẫn là nguồn cập nhật chính).

## Verification
- `npx vitest run` — 18/18 tests passed
- `npx eslint .` — 0 errors, 2 warnings (Fast Refresh cảnh báo về việc export Context cùng file với Provider component — chấp nhận được, đây là pattern LLD đã chốt, tách file sẽ over-engineer cho quy mô 1 context)
- `npx tsc -b` — type-check sạch
- `npx vite build` — build production thành công (170KB JS, gzip 55.5KB)
- `docker compose config` — hợp lệ với `web-gui` service mới thêm
