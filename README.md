# ConceptFlow — Manim Educational Video Generation Tool

## Project Overview
ConceptFlow là một pipeline sản xuất video giáo dục hoàn chỉnh, dùng [Manim](https://www.manim.community/) làm animation engine lõi, cho phép tạo video dạy học (ban đầu tập trung vào lập trình — thuật toán, cấu trúc dữ liệu, khái niệm lập trình) theo phong cách trực quan kiểu 3Blue1Brown. Pipeline đi từ script/markdown → animation + giọng đọc TTS → video hoàn chỉnh → tự động đăng YouTube, chạy hoàn toàn local qua Docker.

Kiến trúc: Microservices + Saga Orchestration qua RabbitMQ. Xem `aidlc-docs/inception/high-level-design/` và `aidlc-docs/inception/application-design/` để biết chi tiết thiết kế, và `aidlc-docs/decisions/` cho các Architecture Decision Records (ADR).

## Prerequisites
- Docker >= 24.x và Docker Compose >= v2
- (Cho phát triển từng service riêng lẻ sau này) Python >= 3.11, Node.js >= 20.x

## Installation
```bash
cp .env.example .env
# Chỉnh sửa .env với giá trị thật (RABBITMQ_USER, RABBITMQ_PASS, v.v.)
```

## Configuration
Biến môi trường cấu hình qua file `.env` (xem `.env.example` cho danh sách đầy đủ và giá trị mẫu — không commit giá trị thật vào git).

**Cách lấy từng credential**: xem [`docs/setup/`](docs/setup/) — hướng dẫn từng bước cho OAuth client YouTube, key Azure Speech và key Google Cloud TTS, kèm cách kiểm tra và cách xoay key khi lộ.

| Biến | Mô tả |
|---|---|
| `RABBITMQ_USER` | Username đăng nhập RabbitMQ (thay thế `guest` mặc định) |
| `RABBITMQ_PASS` | Password RabbitMQ |
| `POSTGRES_USER` | Username cho mọi PostgreSQL instance (database-per-service, ADR-0013) |
| `POSTGRES_PASS` | Password PostgreSQL |
| *(thư mục `secrets/`)* | Từ CR-012, OAuth client YouTube **không** khai trong `.env` nữa: thả file `client_secret*.json` tải từ Google Cloud Console vào `secrets/`. 1 file = 1 GCP project = 1 rổ quota (~6 video/ngày). Xem [`docs/setup/youtube-oauth-client.md`](docs/setup/youtube-oauth-client.md) |
| `GOOGLE_OAUTH_CLIENT_ID` | Fallback cho cấu hình cũ trước CR-012, chỉ dùng khi `secrets/` không có file nào. Để trống khi đã dùng `secrets/` |
| `GOOGLE_OAUTH_CLIENT_SECRET` | Fallback tương ứng |
| `RENDER_TIMEOUT_SECONDS` | Trần wall-clock cho 1 lần render Manim (mặc định 1800). Đo được: video 10 phút @1080p60 mất ~276s, nên đây là ~6.5× biên an toàn |
| `RENDER_MEMORY_LIMIT_GB` | Trần address-space của tiến trình render (mặc định 4). KHÔNG đặt vượt RAM của Docker VM |
| `AZURE_SPEECH_KEY` / `AZURE_SPEECH_REGION` | Key + region của Azure Speech resource (CR-011) — xem [`docs/setup/azure-speech-key.md`](docs/setup/azure-speech-key.md). Region viết dạng mã (`southeastasia`), không phải tên hiển thị. Bỏ trống → các giọng `(Azure)` trong danh mục tự rơi về giọng Edge y hệt. Tier F0: 500.000 ký tự neural/tháng, miễn phí, không hết hạn |
| `GOOGLE_TTS_CREDENTIALS_FILE` | Đường dẫn **trên máy host** tới service-account JSON của Google Cloud TTS (ADR-0023) — xem [`docs/setup/google-cloud-tts-key.md`](docs/setup/google-cloud-tts-key.md). Không bắt buộc: bỏ trống thì mọi giọng dùng engine Edge (ADR-0024), vốn không cần tài khoản. Google đã bỏ free tier nên nhánh này hiện để không (CR-010) |
| `ASSEMBLY_LEAD_IN_SECONDS` / `ASSEMBLY_TAIL_SECONDS` | Khoảng lặng đầu/cuối video (mặc định 0). Bật lên sẽ ép re-encode toàn bộ video — đo được chậm hơn ~250 lần so với stream-copy |
| `RENDER_QUALITY` | Chất lượng render mặc định khi project không chỉ định: `720p30` \| `1080p60` \| `4k60` (mặc định `1080p60`). Creator chọn theo từng project trên GUI |
| `RENDER_CACHE_ROOT` | Nơi giữ `media_dir` theo từng project để Manim tái dùng cache (mặc định `/shared/.manim-media`). Đặt rỗng để tắt cache. Đo được: render lại nhanh gấp ~5 lần |
| `ASSEMBLY_TIMEOUT_SECONDS` | Trần wall-clock cho 1 lần ghép video bằng ffmpeg (mặc định 900) |
| `GOOGLE_OAUTH_REDIRECT_URI` | Redirect URI dùng chung cho **mọi** OAuth client: `http://localhost:3000/oauth/youtube/callback`. Mọi client trong `secrets/` phải khai đúng chuỗi này trong Authorized redirect URIs, Google so khớp từng ký tự |

## Running the Project
```bash
docker compose up -d
```
- RabbitMQ Management UI: http://localhost:15672 (đăng nhập bằng `RABBITMQ_USER`/`RABBITMQ_PASS`)
- Content Plugin Service: nội bộ (`content-plugin:8000` trong docker network), không expose ra host — dùng `docker compose logs content-plugin` hoặc `docker exec` để kiểm tra. DB riêng: `content-plugin-db` (Postgres, Inbox/Outbox — ADR-0013)
- TTS Service: message-driven qua RabbitMQ (queue `tts.commands`), không có port HTTP nào (ADR-0014) — dùng `docker compose logs tts`. DB riêng: `tts-db` (Postgres, Inbox/Outbox — ADR-0013)
- Script Processing Service: message-driven qua RabbitMQ (queue `script_processing.commands`), không có port HTTP nào — dùng `docker compose logs script-processing`. DB riêng: `script-processing-db` (Postgres, Inbox/Outbox — ADR-0013)
- Rendering Service: message-driven qua RabbitMQ (queue `rendering.commands`), sinh animation Manim, không có port HTTP nào — dùng `docker compose logs rendering`. DB riêng: `rendering-db` (Postgres, Inbox/Outbox — ADR-0013). Lưu animation clip vào volume `shared_artifacts` (dùng chung với TTS Service)
- Video Assembly Service: message-driven qua RabbitMQ (queue `video_assembly.commands`), ghép animation + audio + nhạc nền (ffmpeg), không có port HTTP nào — dùng `docker compose logs video-assembly`. DB riêng: `video-assembly-db` (Postgres, Inbox/Outbox — ADR-0013). Đọc animation/audio clip và ghi video hoàn chỉnh vào volume `shared_artifacts` (dùng chung với TTS/Rendering Service)
- Publisher Service: REST (`/v1/auth/youtube/{start,callback}`, OAuth flow) + message-driven qua RabbitMQ (queue `publisher.commands`), đăng video lên YouTube — nội bộ (`publisher:8000`), không expose ra host (được API Gateway proxy tới khi Unit 9 hoàn thành) — dùng `docker compose logs publisher`. DB riêng: `publisher-db` (Postgres, Inbox/Outbox + `oauth_credentials` — ADR-0013, ADR-0016). Đọc video hoàn chỉnh (read-only) từ volume `shared_artifacts`. **Yêu cầu**: đăng ký Google OAuth Client trước khi dùng tính năng đăng video (xem `GOOGLE_OAUTH_*` ở mục Configuration)
- Orchestrator Service: Saga orchestrator (Go, không phải Python — ADR-0018) điều phối Render Saga (5 bước) + Publish Saga (1 bước) qua REST (`POST /v1/sagas/render`, `POST /v1/sagas/publish`, `GET /v1/projects/{id}`, `POST /v1/projects/{id}/retry`) + message-driven qua RabbitMQ (`orchestrator.events` + 6 `*.commands.dlq`) — nội bộ (`orchestrator:8000`), không expose ra host, được API Gateway proxy tới — dùng `docker compose logs orchestrator`. DB riêng: `orchestrator-db` (Postgres, Inbox cho event nhận vào + Outbox cho command gửi đi — ADR-0013, ADR-0019)
- API Gateway (Unit 9): reverse-proxy + AMQP-to-SSE bridge (Node.js/Express — ADR-0020), stateless, entry point duy nhất cho Web GUI. Proxy nguyên trạng REST tới Content Plugin/Orchestrator/Publisher (`/v1/plugins`, `/v1/sagas/render`, `/v1/sagas/publish`, `/v1/projects/{id}`, `/v1/projects/{id}/retry`, `/v1/auth/youtube/{start,callback}`), tự xử lý `GET /v1/progress/{id}` (SSE, consume `progress.fanout` từ RabbitMQ) và `GET /health` (không phụ thuộc downstream). Publish port ra host: `8080:8080` — dùng `docker compose logs api-gateway`. Không có database riêng (hoàn toàn stateless ngoại trừ in-memory SSE connection registry).
- Web GUI (Unit 10): React 18 + TypeScript SPA (Vite build, serve tĩnh qua nginx). Giao diện Creator: soạn script + chọn plugin/ngôn ngữ/nhạc nền (`NewProjectPage`), theo dõi tiến trình render qua SSE (`RenderPage`), xem video + kết nối YouTube + đăng (`ResultPage`). Gọi API Gateway qua `VITE_API_BASE_URL` (build-time env). Publish port ra host: `3000:80` — dùng `docker compose logs web-gui`. Không có database, hoàn toàn stateless phía server.
- Centralized Logging (Grafana + Loki + Promtail): Promtail tự phát hiện toàn bộ container qua Docker socket và gửi log tới Loki (không cần sửa code service nào). Grafana UI tại http://localhost:3001 (đăng nhập bằng `GRAFANA_USER`/`GRAFANA_PASS`), datasource Loki đã auto-provision sẵn — vào Explore, query LogQL vd. `{container="orchestrator"}` để xem log 1 service, hoặc `{container=~".+"}` để xem tất cả. Loki không expose port ra host (chỉ truy cập qua Grafana). Xem `aidlc-docs/construction/observability/code/README.md` để biết chi tiết.

## Running Tests
Mỗi service có test suite riêng (pytest). Ví dụ cho Content Plugin Service:
```bash
cd services/content-plugin
pip install -r requirements-dev.txt
pytest -q
```
Tương tự cho TTS Service, Script Processing Service, và Rendering Service:
```bash
cd services/tts && pip install -r requirements-dev.txt && pytest -q
cd services/script-processing && pip install -r requirements-dev.txt && pytest -q
cd services/rendering && pip install -r requirements-dev.txt && pytest -q
cd services/video-assembly && pip install -r requirements-dev.txt && pytest -q
cd services/publisher && pip install -r requirements-dev.txt && pytest -q
```
Rendering Service's `requirements.txt` bao gồm `manim` (native dependencies: ffmpeg, cairo, pango) — nếu chỉ chạy unit test (không cần render Manim thật), có thể bỏ qua `manim` khi cài cục bộ vì test suite dùng fake/mock cho toàn bộ tương tác Manim thật (`_render_to_file` được monkeypatch trong test, không import `manim` khi chạy `pytest`).
Video Assembly Service's test suite tương tự không cần cài `ffmpeg` cục bộ — mọi tương tác `subprocess.run`/ffmpeg/ffprobe được mock trong test.
Publisher Service's test suite không cần Google OAuth Client thật hay kết nối mạng — mọi tương tác `google-api-python-client`/`google-auth-oauthlib`/`psycopg2` được mock trong test.
Toàn bộ service Python yêu cầu Python 3.12 (dùng `from datetime import UTC` và union type `X | Y` không cần `from __future__ import annotations` cho runtime — chạy test suite trên Python < 3.12 sẽ lỗi import).

Orchestrator Service (Go, không dùng pytest):
```bash
cd services/orchestrator
go mod tidy
go build ./...
go vet ./...
go test ./...
```

API Gateway (Node.js, không dùng pytest):
```bash
cd services/api-gateway
npm install
npx eslint .
npm test
```

Web GUI (React/TypeScript, Vitest):
```bash
cd services/web-gui
npm install
npx eslint .
npm test
```

Hướng dẫn test tổng hợp toàn hệ thống sẽ được bổ sung ở giai đoạn Build and Test (`aidlc-docs/construction/build-and-test/`, sau khi tất cả unit hoàn thành).

## Project Structure
```
.
├── docker-compose.yml       # Định nghĩa toàn bộ service (bắt đầu với RabbitMQ)
├── .env.example              # Mẫu biến môi trường
├── infra/
│   └── rabbitmq/              # Cấu hình topology RabbitMQ (exchange/queue/DLQ)
├── services/
│   ├── content-plugin/         # Content Plugin Service (Python/FastAPI, Hexagonal)
│   │                             # domain/ → application/ → adapters/{api,messaging,persistence,plugins}/
│   ├── tts/                     # TTS Service (Python, Hexagonal, Edge TTS engine, message-driven — ADR-0014)
│   │                             # domain/ → application/ → adapters/{messaging,persistence,tts_engines,storage,logging}/
│   ├── script-processing/       # Script Processing Service (Python, Hexagonal, Markdown parser — ADR-0011)
│   │                             # domain/ → application/ → adapters/{messaging,persistence,parsing,logging}/
│   ├── rendering/                # Rendering Service (Python, Hexagonal, Manim engine, dynamic templates — ADR-0015)
│   │                             # domain/ → application/ → adapters/{messaging,persistence,rendering,storage,logging}/
│   ├── video-assembly/           # Video Assembly Service (Python, Hexagonal, ffmpeg/ffprobe)
│   │                             # domain/ → application/ → adapters/{messaging,persistence,assembly,storage,logging}/
│   ├── publisher/                # Publisher Service (Python/FastAPI, Hexagonal, YouTube Data API — ADR-0016)
│   │                             # domain/ → application/ → adapters/{api,messaging,persistence,youtube,logging}/
│   ├── orchestrator/             # Orchestrator Service (Go, Hexagonal, Saga coordinator — ADR-0018)
│   │                             # cmd/orchestrator/ (composition root) + internal/domain → application → adapters/{http,amqp,postgres,logging}/
│   ├── api-gateway/              # API Gateway (Node.js/Express, layered — ADR-0020)
│   │                             # src/{routes,handlers,clients,middleware,config}/ — reverse-proxy + AMQP-to-SSE bridge, không có domain logic riêng
│   └── web-gui/                  # Web GUI (React 18/TypeScript, Vite, feature-based)
│                                 # src/{pages,components,hooks,api,context,types}/ — SPA, không có backend logic
├── shared/                    # Schema/type dùng chung giữa service (nếu cần)
└── aidlc-docs/                 # Toàn bộ tài liệu AI-DLC (requirements, design, ADR, audit trail)
```

## CI/CD
Chưa thiết lập — sẽ được cấu hình ở giai đoạn Build and Test (`aidlc-docs/construction/build-and-test/ci-cd-integration-instructions.md`, sau khi tất cả unit hoàn thành Construction Phase).
