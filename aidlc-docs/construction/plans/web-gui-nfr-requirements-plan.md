# NFR Requirements Plan — Unit 10: Web GUI

## Unit Context
SPA React/TypeScript, giao tiếp qua API Gateway (Unit 9), luồng người dùng tuyến tính 1 Creator/máy cá nhân.

## Execution Checklist
- [x] Thu thập câu trả lời
- [x] Tạo `nfr-requirements.md`
- [x] Tạo `tech-stack-decisions.md`
- [x] Tạo ADR cho lựa chọn ngôn ngữ/framework
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Tech Stack — Xác nhận React + Vite + TypeScript (BẮT BUỘC)
A) 💡 Suggested: **React 18** + **TypeScript 5** + **Vite** (đã chốt ở `technology-direction.md`/ADR-0003 cho React, LLD Question 8 cho Vite). Package manager: `npm` (nhất quán, không cần `yarn`/`pnpm` cho 1 project nhỏ)
   - ✅ Strengths: nhất quán quyết định Inception, Vite là build tool hiện đại chuẩn cho React/TS
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 2: Performance — Bundle Size / Loading
A) 💡 Suggested: Không cần code-splitting/lazy-loading phức tạp — chỉ 3 page, tổng bundle nhỏ (không dùng UI library ngoài, LLD Question 5). Vite's default production build (tree-shaking, minify) đủ cho quy mô này
   - ✅ Strengths: đơn giản, không over-engineer cho app nhỏ
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3: Availability & Reliability — SSE Reconnect (Browser-side)
A) 💡 Suggested: Dựa vào hành vi mặc định của `EventSource` — browser TỰ ĐỘNG reconnect khi kết nối SSE bị đứt (built-in behavior của Web API, không cần code thêm). `useSSE` hook chỉ cần cleanup đúng lúc unmount (tránh leak), không cần tự viết reconnect logic thủ công
   - ✅ Strengths: tận dụng browser built-in behavior, giảm code không cần thiết
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 4: Security — XSS/Content Handling
A) 💡 Suggested: React's default JSX escaping đủ để chống XSS cơ bản (không dùng `dangerouslySetInnerHTML` ở bất kỳ đâu — script content, error message đều render qua JSX text node, tự động escape). Không cần CSP header riêng (ngoài phạm vi GUI — Gateway/server config nếu cần)
   - ✅ Strengths: đơn giản, React's default behavior đã đủ an toàn cho scope này
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 5: Usability — Accessibility (a11y) Baseline
A) 💡 Suggested: Baseline tối thiểu — mọi `<input>`/`<button>` có `label`/`aria-label` rõ ràng, form field lỗi có `aria-invalid`/`aria-describedby` trỏ tới thông báo lỗi. KHÔNG cần audit a11y toàn diện (WCAG AA đầy đủ) — ngoài phạm vi MVP cá nhân, nhưng baseline cơ bản không tốn thêm effort đáng kể
   - ✅ Strengths: baseline hợp lý không tốn nhiều effort, tốt cho 1 người dùng có thể dùng screen reader/keyboard navigation
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 6: Maintainability — Testing Tooling
A) 💡 Suggested: **Vitest** (tích hợp sẵn với Vite, nhanh hơn Jest cho project dùng Vite) + **React Testing Library** cho component test, khớp `dependency-injection.md`'s Testability đã mô tả (mock module `api/client.ts`, `renderHook` cho custom hook)
   - ✅ Strengths: tích hợp tốt với Vite (không cần cấu hình Babel/transform riêng như Jest), chuẩn cộng đồng React hiện tại
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A
