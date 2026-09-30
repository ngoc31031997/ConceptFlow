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
docker compose ps   # chờ các service báo healthy
```
Cổng mở ra host: Web GUI http://localhost:3000, API Gateway http://localhost:8080, RabbitMQ Management UI http://localhost:15672 (`RABBITMQ_USER`/`RABBITMQ_PASS`), Grafana http://localhost:3001 (`GRAFANA_USER`/`GRAFANA_PASS`). Mọi service khác chỉ nằm trong network `backend`; xem log bằng `docker compose logs <service>`.

| Service | Ngôn ngữ | Vai trò | Giao tiếp | DB riêng (ADR-0013) |
|---|---|---|---|---|
| `web-gui` | React 18 + TypeScript (Vite, nginx) | Giao diện Creator: wizard tạo project, soạn kịch bản cùng AI, duyệt, render, xem video, đăng YouTube, thư viện prompt/hình | Gọi API Gateway (`VITE_API_BASE_URL`, build-time) | — |
| `api-gateway` | Node.js/Express (ADR-0020) | Điểm vào duy nhất: proxy REST tới orchestrator, authoring-service, publisher; SSE tiến độ từ `progress.fanout` (ADR-0017); upload thumbnail/nhạc vào `shared_artifacts` | REST + SSE | — |
| `orchestrator` | Go (ADR-0018) | Saga coordinator và chủ sở hữu project: tạo/fork/xoá project, huỷ/retry bước, channel asset | REST nội bộ; RabbitMQ command/event (Outbox cho command, ADR-0019) | `orchestrator-db` |
| `authoring-service` | Go (ADR-0029) | Thư viện prompt, chuỗi soạn kịch bản (câu chuyện → storyboard → code), wizard, gợi ý metadata/short, thư viện hình minh hoạ, nhật ký dùng LLM | REST nội bộ; gọi orchestrator, llm-service, rendering | `authoring-service-db` |
| `llm-service` | Python (CR-039) | Nơi duy nhất gọi mô hình ngôn ngữ (Hive qua OpenAI SDK, hoặc Ollama); pipeline sinh code chia đoạn | REST nội bộ; gọi rendering để kiểm biên dịch | — |
| `ollama` | image `ollama/ollama` | Mô hình local cho llm-service khi chọn provider `ollama` (`ollama-pull` tải model một lần) | HTTP nội bộ | — |
| `rendering` | Python (Manim, Remotion) | Bước `validate_script` và `render_scenes`, render channel asset; HTTP nội bộ `/v1/check/*`, `/v1/illustrations/preview` | RabbitMQ `rendering.commands` + HTTP nội bộ | `rendering-db` |
| `tts` | Python (Edge, Azure — ADR-0024, ADR-0025) | Bước `synthesize_speech` | RabbitMQ `tts.commands` (ADR-0014) | `tts-db` |
| `video-assembly` | Python (ffmpeg) | Bước `assemble_video`, `qc_video`, `generate_clips`, chuẩn hoá channel asset | RabbitMQ `video_assembly.commands` | `video-assembly-db` |
| `publisher` | Python (YouTube Data API — ADR-0016, ADR-0026) | OAuth YouTube (nhiều app, nhiều kênh), bước `publish_video` | REST nội bộ + RabbitMQ `publisher.commands` | `publisher-db` |

Hạ tầng: `rabbitmq` (topology ở `infra/rabbitmq/definitions.json`), volume `shared_artifacts` (hợp đồng ở `docs/contracts/shared-artifacts.md`), log tập trung Loki + Promtail + Grafana (datasource Loki auto-provision; query LogQL vd. `{container="orchestrator"}`). Mục lục mọi hợp đồng giữa service: `docs/contracts/README.md`.

## Running Tests
Python (`tts`, `rendering`, `video-assembly`, `publisher`, `llm-service`), Python 3.12:
```bash
cd services/<service>
python3.12 -m venv .venv && .venv/bin/pip install -r requirements-dev.txt
.venv/bin/python -m pytest -q
```
Rendering Service's `requirements.txt` bao gồm `manim` (native dependencies: ffmpeg, cairo, pango); test suite dùng fake cho tương tác Manim thật. Video Assembly và Publisher cũng mock ffmpeg/Google API/psycopg2 nên không cần cài ffmpeg hay credential thật.

Go (`orchestrator`, `authoring-service`):
```bash
cd services/<service>
go vet ./... && go test ./...
```

Node (`api-gateway` — Jest, `web-gui` — Vitest):
```bash
cd services/<service>
npm install && npx eslint . && npm test
```

Test hợp đồng chéo service ở gốc repo (volume `shared_artifacts`, luồng bước, các bản chép mã hạ tầng Python):
```bash
services/tts/.venv/bin/python -m pytest tests/contracts -q
```

Hướng dẫn build/test chi tiết: `aidlc-docs/construction/build-and-test/`.

## Project Structure
```
.
├── docker-compose.yml        # Toàn bộ service + hạ tầng
├── .env.example              # Mẫu biến môi trường
├── Makefile                  # make graph / make graph-hooks (graphify)
├── services/
│   ├── web-gui/              # React/TypeScript — src/{pages,components,hooks,api,context,utils,types,styles}
│   ├── api-gateway/          # Node/Express, phân lớp — src/{routes,handlers,clients,middleware,config}
│   ├── orchestrator/         # Go, hexagonal — cmd/orchestrator + internal/{domain,application,adapters,config}
│   ├── authoring-service/    # Go, hexagonal — cmd/authoring + internal/{domain,application,adapters,config}
│   ├── llm-service/          # Python, layout phẳng — app/
│   ├── rendering/            # Python, hexagonal — domain/ application/ adapters/; conceptflow/ (thư viện chạy trong script Manim), remotion_project/, tools/
│   ├── tts/                  # Python, hexagonal — domain/ application/ adapters/
│   ├── video-assembly/       # Python, hexagonal — domain/ application/ adapters/
│   └── publisher/            # Python, hexagonal — domain/ application/ adapters/
├── tests/                    # contracts/ (test hợp đồng chéo service), fixtures/, benchmark_render.py
├── infra/                    # rabbitmq/ (topology), observability/ (Loki, Promtail, Grafana)
├── docs/                     # contracts/, setup/ (lấy credential), brand/, review/, agentic/, ux-ui-design-rules.md
├── scripts/                  # graph.sh (graphify)
├── data/shared_artifacts/    # Nơi bind-mount video ra host (git-ignored)
├── secrets/                  # OAuth client YouTube (git-ignored, xem secrets/README.md)
├── aidlc-docs/               # Tài liệu AI-DLC: requirements, design, ADR, audit trail
├── .ai-dlc/                  # Quy tắc quy trình AI-DLC
└── .claude/skills/           # /cr, /code, /deliver
```

## CI/CD
Không có CI. Lớp kiểm tra tự động (`make setup/build/check`, GitHub Actions) đã gỡ ngày 2026-09-30; mỗi thay đổi được kiểm bằng test của từng service và test hợp đồng ở trên.
