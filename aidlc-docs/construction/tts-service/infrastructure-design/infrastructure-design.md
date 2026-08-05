# Infrastructure Design — Unit 3: TTS Service

## Deployment Environment
Docker container (custom image, base `python:3.12-slim` + system dependencies cần thiết cho Piper qua `apt-get`), trong docker-compose, cùng docker network `backend`.

## Storage Infrastructure — Shared Volume (lần đầu áp dụng trong hệ thống)
Named volume `shared_artifacts` (Docker managed volume, không phải bind mount) mount vào `/shared` trong container. Volume này sẽ được mount tương tự vào Rendering Service (Unit 5) khi phát triển unit đó, để đọc lại `audio_path` sinh ra từ TTS Service.

## Compute Infrastructure — Voice Model Bundling
Dockerfile tải voice model Piper (`.onnx` + `.onnx.json`) cho `vi` và `en` từ Piper's official model repository trong build stage (`RUN curl`), lưu vào `/app/voices/` trong image. Build cần network access (chỉ lúc build, không lúc runtime).

## Networking
Port 8000 (FastAPI), chỉ nội bộ docker network `backend`, không map ra host.

## Health Check
`GET /health` — trả 200 nếu FastAPI app sẵn sàng VÀ voice model đã load thành công vào in-process cache; trả 503 nếu model chưa load xong. `healthcheck` trong docker-compose dùng endpoint này. Service phụ thuộc (Rendering Service, Unit 5) dùng `depends_on: condition: service_healthy`.

## Not Applicable
- **Load Balancer**: N/A — 1 instance cố định
- **API Gateway**: N/A — chỉ Rendering Service gọi nội bộ, không expose ra ngoài
- **Database Read/Write Splitting / Sharding**: N/A — không có database

## Scaling
1 instance cố định, không auto-scaling.

## Monitoring
Structured logging ra stdout (`docker logs`), bao gồm `saga_id` (từ header `X-Saga-ID`) trong mọi log line. Không cần APM/metrics platform riêng cho MVP local.
