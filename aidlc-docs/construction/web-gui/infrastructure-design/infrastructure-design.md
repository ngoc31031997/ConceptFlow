# Infrastructure Design — Unit 10: Web GUI

## Deployment Environment
Docker container, multi-stage build: `node:20-alpine` (build stage, `npm run build`) → `nginx:alpine` (serve stage, static `dist/`). Network `backend` — chạy local, không cloud provider.

## Compute Infrastructure
1 container `web-gui`, fixed 1 instance, không auto-scaling.

## Storage Infrastructure
Không áp dụng — SPA hoàn toàn stateless phía server, không database/volume riêng.

## Database Read/Write Splitting & Sharding
Không áp dụng — không có database.

## Messaging Infrastructure
Không áp dụng — GUI không kết nối RabbitMQ trực tiếp; nhận progress qua SSE do API Gateway relay (đã chốt ở Low-Level Design).

## Networking Infrastructure
Publish port ra host: `3000:80` (host:container). Đây là entry point browser Creator dùng để truy cập GUI. API Gateway (`8080`) và RabbitMQ Management UI (`15672`, dev-only) vẫn là các port khác đã publish trong hệ thống — GUI thêm port thứ 3.

## API Gateway Base URL Configuration
Build-time env var `VITE_API_BASE_URL=http://localhost:8080` bake vào bundle production lúc `npm run build` (Vite build-time env, không phải runtime config fetch). Phù hợp scope local-single-environment hiện tại của dự án.

## Health Check
Dùng route mặc định `GET /` do nginx serve (trả `200` khi `index.html` tồn tại) làm healthcheck — không cần endpoint `/health` riêng vì không có backend logic để kiểm tra.

## Load Balancer
Không áp dụng — 1 instance duy nhất.

## API Gateway (chính unit này)
Không áp dụng — Unit 10 là GUI, không phải Gateway. Unit 10 gọi tới Unit 9 (API Gateway) qua `VITE_API_BASE_URL`.

## Monitoring Infrastructure
Không có stack riêng — `docker compose logs` (nginx access/error log mặc định), nhất quán với các unit khác.

## Shared Infrastructure
Không dùng chung network/volume nào khác ngoài `backend` (chỉ để cùng docker-compose project; GUI không cần giao tiếp nội bộ với service backend nào — mọi gọi API đi qua browser → host port 8080 của `api-gateway`, không qua network `backend`).
