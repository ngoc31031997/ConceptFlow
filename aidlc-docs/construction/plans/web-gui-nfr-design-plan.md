# NFR Design Plan — Unit 10: Web GUI

## Execution Checklist
- [x] Thu thập câu trả lời
- [x] Tạo `nfr-design-patterns.md`
- [x] Tạo `logical-components.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: CRUD vs CQRS (BẮT BUỘC)
A) 💡 Suggested: Không áp dụng — GUI không có data store, chỉ đọc/ghi state phía client tạm thời (`ProjectDraft`, `ProgressState`) và gọi API backend (đã CRUD ở Orchestrator). Concept CRUD/CQRS không có ý nghĩa cho 1 SPA thuần client
   - ✅ Strengths: đúng bản chất
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 2: Resilience Pattern
A) 💡 Suggested: Fetch request lỗi (network/502) → hiển thị `ErrorBanner` chung (Rule 4, Functional Design), không auto-retry tự động (Creator chủ động bấm "Thử lại" — nhất quán triết lý retry-by-step của Unit 8/9). SSE dựa vào `EventSource`'s built-in reconnect (NFR Requirements Question 3)
   - ✅ Strengths: nhất quán toàn hệ thống (không auto-retry ở bất kỳ tầng nào)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3: Caching Strategy
A) 💡 Suggested: Không cache API response ở tầng ứng dụng (dữ liệu Project/progress thay đổi liên tục trong lúc Saga chạy, cache sẽ nhanh stale). Browser HTTP cache mặc định cho static asset (JS/CSS bundle) — Vite tự thêm content-hash vào filename, cache-busting tự động khi deploy version mới
   - ✅ Strengths: đơn giản, tránh stale data cho dữ liệu thay đổi liên tục
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 4: Event-Driven Design (SSE consumption)
A) 💡 Suggested: GUI là consumer THUẦN TÚY của SSE stream (qua Gateway, không trực tiếp AMQP) — không publish gì. Delivery: chấp nhận at-most-once (kế thừa quyết định Unit 9 — nếu mất 1 progress message, `useProject`'s fallback poll hoặc message tiếp theo sẽ cập nhật lại UI đúng trạng thái)
   - ✅ Strengths: nhất quán Unit 9's messaging-design.md
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 5: Security Pattern
A) 💡 Suggested: Xác nhận lại — React JSX escaping mặc định (NFR Requirements Question 4). Input file (`ScriptEditor`'s import, `BackgroundMusicPicker`) đọc client-side qua `FileReader`, không upload trực tiếp lên server nào từ GUI (script_content gửi dạng text trong JSON body, background_music_path là string path — không phải file upload thực sự ở MVP này)
   - ✅ Strengths: đơn giản, đúng scope MVP (không cần file upload service riêng)
   - ⚠️ Trade-offs: `background_music_path` yêu cầu Creator biết đường dẫn file local hợp lệ mà backend truy cập được (không phải upload) — chấp nhận được cho use case chạy local/Docker volume mount

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 6: Logical Components — Infrastructure Elements
A) 💡 Suggested: Không cần cache/circuit-breaker/rate-limiter/state management library ngoài (đã loại trừ). Thành phần: (1) React app (Vite build), (2) `api/client.ts` (fetch wrapper), (3) `EventSource` (browser native, không cần thư viện SSE riêng), (4) `ProjectDraftContext` (React built-in Context API)
   - ✅ Strengths: tối thiểu cần thiết, không thêm dependency
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A
