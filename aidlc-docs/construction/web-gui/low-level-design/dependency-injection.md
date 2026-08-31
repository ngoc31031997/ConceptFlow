# Dependency Injection — Unit 10: Web GUI

## Mechanism
Không dùng DI container/pattern kiểu backend (constructor injection) — React idiom hiện đại dùng **function component + hooks + module import trực tiếp** (Question 4).

## What Gets "Injected" vs Imported Directly
- **API layer**: `api/client.ts`'s function (`getPlugins`, `startRenderSaga`, ...) import trực tiếp vào `hooks/`/`components/` cần dùng — không qua constructor/factory. Test dùng `jest.mock('../api/client')` để fake.
- **Global state**: `ProjectDraftContext` cung cấp qua React's `Context.Provider` ở `App.tsx` (composition root), consume qua `useContext(ProjectDraftContext)` — đây là cơ chế "injection" gần nhất với DI trong React, nhưng dùng Context API built-in thay vì container riêng.
- **Config**: base URL API Gateway đọc từ biến môi trường build-time (`import.meta.env.VITE_API_BASE_URL`, Vite convention) — không inject, đọc trực tiếp trong `api/client.ts`.

## Composition Root
`src/main.tsx`:
1. Render `<App />` vào `#root` DOM element.

`src/App.tsx`:
1. Wrap toàn bộ route trong `<ProjectDraftContext.Provider>`.
2. Định nghĩa `react-router-dom` routes: `/` → `NewProjectPage`, `/projects/:id/render` → `RenderPage`, `/projects/:id/result` → `ResultPage`.

## Testability
- **Component test**: React Testing Library, mock `api/client.ts` module qua Jest, render component với fake context value (`<ProjectDraftContext.Provider value={fakeState}>`).
- **Hook test**: `renderHook` (React Testing Library), mock `EventSource` (cho `useSSE`) và `fetch` (cho `useProject`).

## Concurrency Model
Không áp dụng (browser single-threaded JS, không có concept concurrency cần thiết kế như backend). SSE (`EventSource`) và fetch đều bất đồng bộ qua browser event loop — React's `useEffect` xử lý lifecycle (subscribe lúc mount, cleanup lúc unmount).
