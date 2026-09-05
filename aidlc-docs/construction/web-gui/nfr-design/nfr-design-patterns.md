# NFR Design Patterns — Unit 10: Web GUI

## Data Access Pattern: Không áp dụng (No CRUD/CQRS)
GUI không có data store — chỉ client-side state tạm thời (`ProjectDraft`, `ProgressState`) và API call tới backend đã CRUD ở Orchestrator.

## Resilience Pattern
- Fetch lỗi (network/502) → `ErrorBanner` chung, không auto-retry (Creator chủ động bấm "Thử lại" — nhất quán retry-by-step triết lý toàn hệ thống).
- SSE: dựa vào `EventSource`'s built-in browser reconnect, không tự viết logic.

## Caching Strategy
- Không cache API response (dữ liệu Project/progress thay đổi liên tục).
- Static asset: browser HTTP cache mặc định, Vite tự thêm content-hash filename (cache-busting tự động).

## Event-Driven Design
GUI là SSE consumer thuần túy (qua Gateway), không publish gì. Delivery at-most-once (kế thừa Unit 9) — chấp nhận mất progress message lẻ tẻ, `useProject`'s fallback poll hoặc message tiếp theo tự sửa lại UI.

## Security Pattern
- React JSX escaping mặc định chống XSS.
- File input (`ScriptEditor` import, `BackgroundMusicPicker`) đọc client-side qua `FileReader`, KHÔNG có file-upload service — `background_music_path` là string path Creator tự nhập (yêu cầu file đã tồn tại ở nơi backend truy cập được, vd. Docker volume mount local).

## Logical Components
Xem `logical-components.md`.

## Không áp dụng
- **Caching layer riêng**: không cần.
- **Circuit Breaker**: không cần — GUI chỉ gọi 1 backend duy nhất (Gateway).
- **Rate Limiter**: không cần.
- **State management library ngoài** (Redux/Zustand): không cần — React Context đủ.
