# Infrastructure Design Plan — Unit 10: Web GUI

## Q1. Deployment / Serving Model
GUI là React SPA (Vite build). Chạy thế nào trong Docker?
- **A. (Recommended)** Multi-stage Dockerfile: `node:20-alpine` build stage (`npm run build`) → static `dist/` served bởi `nginx:alpine` stage. Container `web-gui`, port 80 nội bộ.
- B. Chạy `vite preview`/dev server trực tiếp trong container Node (không build tĩnh, không dùng nginx).

## Q2. Networking — Host Port
GUI là nơi Creator (browser) truy cập trực tiếp. Publish port nào ra host?
- **A. (Recommended)** `3000:80` (host:container) — cùng dải với các port khác đã publish (15672 RabbitMQ UI, 8080 API Gateway).
- B. `80:80`.

## Q3. API Gateway Base URL (build-time vs runtime config)
Frontend cần biết địa chỉ API Gateway (`http://localhost:8080`) để gọi REST + SSE + OAuth redirect.
- **A. (Recommended)** Build-time env var (`VITE_API_BASE_URL`), bake vào bundle lúc `npm run build`. Đơn giản, phù hợp local-single-environment hiện tại (không cần đổi giữa nhiều environment).
- B. Runtime config (fetch `/config.json` sau khi load) — linh hoạt hơn nhưng thừa phức tạp cho MVP local-only.

## Q4. Health Check
- **A. (Recommended)** nginx mặc định trả `200` cho `/` (static file tồn tại) → dùng chính route đó làm healthcheck, không cần endpoint `/health` riêng.
- B. Thêm nginx location `/health` riêng trả `200 OK`.

## Q5. Load Balancer / Scaling
- **A. (Recommended)** Không áp dụng — 1 instance duy nhất, local single-user, giống mọi unit khác.

## Q6. Storage / Database
- **A. (Recommended)** Không áp dụng — SPA hoàn toàn stateless phía server, không lưu trữ gì (kể cả localStorage là client-side, ngoài phạm vi infra).

## Q7. Messaging Infrastructure
- **A. (Recommended)** Không áp dụng — GUI không kết nối RabbitMQ trực tiếp; nhận progress qua SSE từ API Gateway (đã chốt ở LLD/NFR Design).

## Q8. Monitoring
- **A. (Recommended)** Không có stack riêng — dùng `docker compose logs` (nginx access/error log mặc định), nhất quán với các unit khác.
