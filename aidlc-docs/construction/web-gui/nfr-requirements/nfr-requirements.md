# NFR Requirements — Unit 10: Web GUI

## Performance
- Không cần code-splitting/lazy-loading — chỉ 3 page, bundle nhỏ. Vite production build (tree-shaking, minify) mặc định đủ.

## Availability & Reliability
- SSE reconnect dựa vào hành vi built-in của `EventSource` (browser tự động reconnect khi mất kết nối) — không cần tự viết reconnect logic.
- `useSSE` hook cleanup đúng lúc component unmount, tránh memory leak/kết nối treo.

## Security
- React JSX escaping mặc định đủ chống XSS cơ bản — không dùng `dangerouslySetInnerHTML`.
- Không cần CSP header riêng ở tầng GUI (ngoài phạm vi — thuộc Gateway/server config nếu cần).

## Usability
- Accessibility baseline: `label`/`aria-label` cho mọi input/button, `aria-invalid`/`aria-describedby` cho field lỗi. Không audit WCAG AA đầy đủ.

## Maintainability
- Testing: Vitest + React Testing Library (tích hợp sẵn Vite).
- Mock `api/client.ts` module cho component/hook test (nhất quán `dependency-injection.md`).

## Scalability
- Không áp dụng — SPA client-side, không có "tải" phía server để scale (chỉ 1 browser/1 Creator dùng tại 1 thời điểm).

## Caching
- Không áp dụng thêm — Vite build output là static asset, browser cache mặc định đủ. Không cache API response (dữ liệu Project/progress thay đổi liên tục).
