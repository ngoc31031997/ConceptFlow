# Code Generation Plan — Unit 6: Video Assembly Service

## Unit Context
- **Stories**: C4, C5 (xem `unit-of-work-story-map.md`)
- **Dependencies**: Unit 1 (RabbitMQ) — đã hoàn tất; KHÔNG phụ thuộc trực tiếp unit nào khác
- **Interfaces**: AMQP consumer `assemble_video` (queue `video_assembly.commands`) → 1 event/command (`video_assembled`/`assembly_failed`) qua Outbox (ADR-0013)
- **Owned entities**: `outbox_events`/`processed_messages` (Postgres `video-assembly-db`); video output + file trung gian trên shared volume

## Coding Standards (nhất quán Unit 2/3/4/5)
- **Naming**: Python snake_case cho function/variable, PascalCase cho class
- **SOLID**: Bắt buộc — `AssembleVideoUseCase` phụ thuộc `VideoAssemblerPort` abstraction
- **Documentation**: Python docstring (Google style), giải thích WHY
- **Linting**: `ruff`

## Steps

- [x] **Step 1 — Project Structure Setup**: Tạo `services/video-assembly/` (domain/, application/, adapters/{messaging,persistence,assembly,storage,logging}/, main.py, tests/), `requirements.txt`, `requirements-dev.txt`, `pyproject.toml`, `Dockerfile`
- [x] **Step 2 — Business Logic Generation (domain/)**: `models.py` (`SceneAssemblyInput`, `VideoAssemblyRequest`, `VideoAssemblyResult`), `errors.py` (`MissingArtifactError`, `InvalidSceneIndexError`, `InconsistentMediaFormatError`, `AssemblyEngineError`), `ports.py` (`VideoAssemblerPort`)
- [x] **Step 3 — Business Logic Generation (application/)**: `assemble_video.py` (`AssembleVideoUseCase` — idempotency check, zero-trust validate, delegate to assembler; no batch wrapper needed — 1 command = 1 call, Low-Level Design Question 10)
- [x] **Step 4 — Business Logic Unit Testing**: `tests/domain/`, `tests/application/test_assemble_video.py` — dùng `FakeVideoAssembler`, cover Business Rule 1 (zero trust) + idempotency (Rule 8)
- [x] **Step 5 — Storage Adapter Generation**: `adapters/storage/artifact_paths.py` (đường dẫn `/shared/{project_id}/video/final.mp4` + `_tmp/`, idempotency check, cleanup helper)
- [x] **Step 6 — Media Format Inspector Generation**: `adapters/assembly/ffprobe_inspector.py` (`MediaFormatInspector` — ffprobe subprocess, so sánh codec/resolution/framerate, raise `InconsistentMediaFormatError`)
- [x] **Step 7 — ffmpeg Assembler Adapter Generation**: `adapters/assembly/ffmpeg_assembler.py` (`FfmpegVideoAssembler` implements `VideoAssemblerPort` — sort+validate scene_index, gọi `MediaFormatInspector`, threadpool + `ASSEMBLY_TIMEOUT_SECONDS`, mux→concat→overlay nhạc nền, cleanup `_tmp/`)
- [x] **Step 8 — Assembler/Inspector Unit Testing**: `tests/adapters/test_ffmpeg_assembler.py` — mock `subprocess.run`, cover sort/validate scene_index, không gọi ffmpeg thật
- [x] **Step 9 — Persistence Adapter Generation**: `adapters/persistence/{db,inbox,outbox,relay}.py` (copy nguyên mẫu Unit 2/3/4/5, ADR-0013)
- [x] **Step 10 — Logging Adapter Generation**: `adapters/logging/correlation.py`
- [x] **Step 11 — Messaging Layer Generation**: `adapters/messaging/consumer.py` (`AssembleVideoCommandHandler` — inbox check → use case → 1 event + inbox mark cùng transaction → ack), `adapters/messaging/producer.py` (2 envelope builders: `video_assembled`, `assembly_failed`)
- [x] **Step 12 — Messaging Layer Unit Testing**: `tests/adapters/test_messaging.py` — mock AMQP + `FakePool`
- [x] **Step 13 — Persistence/Relay Unit Testing**: `tests/adapters/test_persistence.py`, `test_relay.py` (copy nguyên mẫu Unit 2/3/4/5)
- [x] **Step 14 — Composition Root**: `main.py` — wiring toàn bộ (plain asyncio, `/tmp/ready` sentinel)
- [x] **Step 15 — Documentation Generation**: Cập nhật `README.md` gốc, tạo `aidlc-docs/construction/video-assembly-service/code/README.md`
- [x] **Step 16 — Deployment Artifacts**: Thêm service `video-assembly` + `video-assembly-db` + volume vào `docker-compose.yml` gốc (`shared_artifacts` đã có, chỉ mount thêm)

**Không áp dụng**: Repository Layer riêng ngoài Outbox/Inbox, Frontend Components, Database Migration Scripts, API Layer (không có REST).

**Note về ffmpeg thật**: Unit test dùng mock `subprocess.run`/`FakeVideoAssembler` nên KHÔNG cần môi trường có ffmpeg cài đặt để chạy test suite. Việc chạy ffmpeg thật (integration/manual test) sẽ cần môi trường Docker build đầy đủ theo `Dockerfile`, ngoài phạm vi test tự động ở bước này.
