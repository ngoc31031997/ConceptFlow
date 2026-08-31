# Low-Level Design Plan — Unit 10: Web GUI

## Unit Context
- **Scope**: FR6.1, FR6.2 — toàn bộ giao diện Creator (Epic A-F)
- **Stories chịu trách nhiệm chính**: A1 (soạn script), B1 (chọn plugin), B4 (chọn ngôn ngữ giọng đọc), C1 (khởi chạy render), C5 (cấu hình nhạc nền), C6 (theo dõi tiến trình — SSE), D1 (xem trước video), E1 (trigger OAuth — Gateway xử lý flow), E2 (cấu hình metadata publish)
- **Depends on**: Unit 9 (API Gateway) — MỌI giao tiếp backend qua Gateway, không gọi trực tiếp service nào khác
- **Tech stack (đã chốt ở ADR-0003/technology-direction.md)**: TypeScript/React

## Execution Checklist
- [x] Thu thập câu trả lời
- [x] Tạo `module-structure.md`
- [x] Tạo `dependency-injection.md`
- [x] Tạo `interface-contracts.md`
- [x] Tạo `sequence-flows.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Layering & Component Architecture (BẮT BUỘC)
A) 💡 Suggested: **Feature-based folder structure** (không phải Atomic Design đầy đủ — quá phức tạp cho quy mô 1 luồng người dùng tuyến tính): `pages/` (1 page/route chính — NewProject, Render, Result, Publish), `components/` (UI tái sử dụng — Button, ProgressBar, VideoPlayer...), `hooks/` (custom hook — `useSSE`, `useApi`), `api/` (API client layer, tách biệt hoàn toàn UI), `types/` (TypeScript type dùng chung). Dependency direction: `pages` → `components`/`hooks` → `api`, không ngược lại
   - ✅ Strengths: đơn giản, đúng quy mô 1 luồng người dùng tuyến tính (soạn → cấu hình → render → xem → đăng), dễ điều hướng cho 1 người phát triển
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 2: State Management
A) 💡 Suggested: **React Context + `useReducer`** cho state toàn cục của 1 project đang soạn (script, plugin đã chọn, ngôn ngữ, nhạc nền, metadata publish) — KHÔNG dùng Redux/Zustand/thư viện ngoài (quy mô nhỏ, 1 luồng tuyến tính không cần state management phức tạp). State server (progress SSE, project status) giữ riêng ở component-level qua custom hook (`useSSE`, `useProjectStatus`), không đẩy vào global context (tránh re-render không cần thiết)
   - ✅ Strengths: không thêm dependency, đủ cho quy mô 1 luồng người dùng, React built-in
   - ⚠️ Trade-offs: nếu app phức tạp hơn sau này (nhiều project cùng lúc trong 1 phiên UI) sẽ cần đánh giá lại — ngoài phạm vi hiện tại

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3: Routing
A) 💡 Suggested: `react-router-dom` (v6), route theo luồng tuyến tính: `/` (New Project — soạn script + chọn plugin + ngôn ngữ + nhạc nền), `/projects/:id/render` (theo dõi tiến trình, SSE), `/projects/:id/result` (xem trước video + form metadata + nút đăng)
   - ✅ Strengths: chuẩn cộng đồng React cho SPA routing, khớp đúng luồng người dùng tuyến tính (Epic A→B→C→D→E)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 4: API Client Layer / Dependency Injection
A) 💡 Suggested: `api/client.ts` — wrapper `fetch` với base URL từ biến môi trường build-time (`VITE_API_BASE_URL`, trỏ tới API Gateway). Không cần DI container — các hàm API (`createProject`, `getPluginList`, `startRenderSaga`, ...) là pure function nhận tham số, import trực tiếp vào component/hook (đủ đơn giản, không cần constructor injection như backend — React component không phải class cần test độc lập theo kiểu OOP)
   - ✅ Strengths: đơn giản, đúng idiom React/frontend hiện đại (function components + hooks, không cần DI pattern của backend)
   - ⚠️ Trade-offs: test component cần mock module (`jest.mock('../api/client')`) thay vì inject fake — chấp nhận được, đây là pattern chuẩn test React

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 5: Interface Contracts — API Client Methods (khớp API Gateway's routing table)
A) 💡 Suggested: `api/client.ts` expose các hàm khớp 1-1 với API Gateway's routing table (Unit 9's interface-contracts.md): `getPlugins()`, `startRenderSaga(input)`, `getProject(id)`, `retryProject(id)`, `startPublishSaga(id, metadata)`, `getYoutubeAuthStartUrl()` (trả URL để redirect, không phải fetch — vì đây là redirect flow, không phải JSON API), `subscribeProgress(projectId, onMessage)` (dùng `EventSource` browser API cho SSE, không phải `fetch`)
   - ✅ Strengths: khớp đầy đủ Gateway's routing table, không thiếu hàm nào cho story đã gán Unit 10
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 6: Sequence Flows cần thiết kế
A) 💡 Suggested: 5 flow chính — (1) Soạn script + chọn plugin/ngôn ngữ/nhạc nền → submit `POST /v1/sagas/render` (Epic A/B/C1), (2) Theo dõi tiến trình qua SSE (`EventSource`, cập nhật UI theo `ProgressMessage`, Story C6), (3) Xem kết quả + phát video (`<video>` tag, `GET /v1/projects/{id}`'s `video_path`, Story D1), (4) OAuth flow (redirect tới Gateway's `/v1/auth/youtube/start`, browser tự xử lý redirect chain, GUI chỉ cần link/button, Story E1), (5) Cấu hình metadata + submit `POST /v1/sagas/publish` (Story E2/E3)
   - ✅ Strengths: bao phủ đủ luồng người dùng chính theo đúng thứ tự Epic A→E
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 7: Error Handling & Retry UI (Story C6/E3's error display, LLD's error contract)
A) 💡 Suggested: Khi `Project.Status` (qua `GET /v1/projects/{id}` hoặc SSE) là `failed_at_<step>`, hiển thị banner lỗi với `error_message` + nút "Thử lại" gọi `POST /v1/projects/{id}/retry` (Gateway proxy tới Orchestrator). Lỗi mạng/Gateway `502` hiển thị thông báo chung "Không thể kết nối máy chủ, thử lại sau" — không phân biệt chi tiết service nào lỗi (GUI không cần biết kiến trúc nội bộ)
   - ✅ Strengths: khớp đúng cơ chế retry-by-step đã thiết kế ở Unit 8, UX đơn giản rõ ràng
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 8: Build Tooling
A) 💡 Suggested: **Vite** (không phải Create React App, đã deprecated) — build tool nhanh, chuẩn cộng đồng React/TypeScript hiện tại, dev server có HMR nhanh, cấu hình tối thiểu
   - ✅ Strengths: chuẩn hiện đại, nhanh, cấu hình đơn giản
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A
