# Module Structure — Unit 6: Video Assembly Service

## Layering (Hexagonal / Ports & Adapters — ADR-0002)

```
services/video-assembly/
├── domain/
│   ├── models.py                  # VideoAssemblyRequest, SceneAssemblyInput, VideoAssemblyResult (value objects)
│   ├── errors.py                   # MissingArtifactError, InvalidSceneIndexError, InconsistentMediaFormatError, AssemblyEngineError (last 2 added at Functional Design, Question 2/3)
│   └── ports.py                    # VideoAssemblerPort
├── application/
│   └── assemble_video.py           # AssembleVideoUseCase (single command, idempotency check, no batch/fail-fast logic — Question 10)
├── adapters/
│   ├── messaging/
│   │   ├── consumer.py              # AMQP consumer for assemble_video (queue video_assembly.commands)
│   │   └── producer.py              # Envelope builders: video_assembled, assembly_failed
│   ├── persistence/
│   │   ├── db.py                     # PostgreSQL pool + schema bootstrap (ADR-0013, copied verbatim)
│   │   ├── inbox.py                  # InboxRepository
│   │   ├── outbox.py                 # OutboxRepository
│   │   └── relay.py                  # OutboxRelay
│   ├── assembly/
│   │   ├── ffprobe_inspector.py      # MediaFormatInspector — ffprobe pre-check per clip, raises InconsistentMediaFormatError (Functional Design Question 3)
│   │   └── ffmpeg_assembler.py       # FfmpegVideoAssembler implements VideoAssemblerPort — threadpool + timeout, subprocess ffmpeg CLI (Question 3); sorts scenes by scene_index before mux/concat (Functional Design Question 2)
│   ├── storage/
│   │   └── artifact_paths.py         # Shared-volume path convention: final video + _tmp/ intermediate mux files (mirror Unit 3/5's artifact_paths.py)
│   └── logging/
│       └── correlation.py            # saga_id injection vào log context
├── main.py                          # Composition root — wiring, AMQP consumer + OutboxRelay startup, /tmp/ready sentinel
└── tests/
    ├── domain/
    ├── application/
    └── adapters/
```

## Dependency Direction
`adapters/` → `application/` → `domain/`. `domain/` không import ffmpeg/aio-pika/asyncpg. `application/` chỉ phụ thuộc `domain/` qua `VideoAssemblerPort` (abstraction) — không biết chi tiết ffmpeg cụ thể. `adapters/assembly/ffmpeg_assembler.py` implement `domain/ports.py::VideoAssemblerPort`.

Khác với Unit 5 (có `AnimationTemplatePort` + registry cho dynamic plugin discovery), Unit 6 không cần cơ chế discovery — chỉ có 1 chiến lược ghép cố định (mux từng scene → concat → overlay nhạc nền, Question 4), nên chỉ có 1 port duy nhất.

## Module Responsibilities

| Module | Responsibility |
|---|---|
| `domain/models.py` | `VideoAssemblyRequest` (`project_id`, `scenes: list[SceneAssemblyInput]`, `background_music_path: str \| None`), `SceneAssemblyInput` (`scene_index`, `clip_path`, `audio_path`), `VideoAssemblyResult` (`video_path`) — **Revision (Functional Design Question 2)**: `scene_clip_paths`/`scene_audio_paths` (2 mảng song song) thay bằng `scenes: list[SceneAssemblyInput]` (1 mảng object có `scene_index` tường minh) |
| `domain/errors.py` | `MissingArtifactError` (1 trong `clip_path`/`audio_path`/`background_music_path` không tồn tại trên shared volume), `InvalidSceneIndexError` (thiếu/trùng `scene_index` trong dãy liên tục từ 0 — **Revision, Functional Design Question 2**), `InconsistentMediaFormatError` (ffprobe pre-check phát hiện codec/resolution/framerate không đồng nhất giữa các clip — **Revision, Functional Design Question 3**), `AssemblyEngineError` (ffmpeg exit code khác 0, hoặc timeout) |
| `domain/ports.py` | `VideoAssemblerPort` (abstract: `assemble(request, output_path) -> None`) |
| `application/assemble_video.py` | `AssembleVideoUseCase(assembler: VideoAssemblerPort)` — validate `scenes` không rỗng, mỗi `SceneAssemblyInput` có `clip_path`/`audio_path` không rỗng, tính đường dẫn output cuối cùng, kiểm tra idempotency (file tồn tại), gọi `assembler.assemble(...)` nếu chưa có — KHÔNG có batch/fail-fast logic (1 command = 1 lời gọi duy nhất, Question 10) |
| `adapters/assembly/ffprobe_inspector.py` | `MediaFormatInspector` — chạy `ffprobe` trên mỗi `clip_path` lấy codec/resolution/framerate; so sánh giữa các scene, không đồng nhất → raise `InconsistentMediaFormatError` (**Revision, Functional Design Question 3**); chạy TRƯỚC bước mux, trong cùng `ThreadPoolExecutor` |
| `adapters/assembly/ffmpeg_assembler.py` | `FfmpegVideoAssembler` — implement `VideoAssemblerPort`; sort `scenes` theo `scene_index` tăng dần, validate dãy liên tục từ 0 (không thiếu/trùng) → `InvalidSceneIndexError` nếu sai (**Revision, Functional Design Question 2**); gọi `MediaFormatInspector` pre-check; chạy 3 bước ffmpeg tuần tự trong `ThreadPoolExecutor` (Question 6): (1) per-scene mux animation+audio → `_tmp/scene_{i}_muxed.mp4` (`-map 0:v -map 1:a -c:v copy -c:a aac`), (2) concat demuxer nối các file muxed theo `scene_index` đã sort (`-c copy`), (3) nếu có `background_music_path`: overlay `amix` với `volume=0.2` trên nhạc nền + `-stream_loop -1 -shortest` nếu ngắn hơn video (Question 4, 5); dọn `_tmp/` sau khi thành công (Question 7); timeout đọc từ env var `ASSEMBLY_TIMEOUT_SECONDS` (mặc định 180s, bao gồm cả bước ffprobe pre-check) |
| `adapters/storage/artifact_paths.py` | Đường dẫn quy ước `/shared/{project_id}/video/final.mp4` (output), `/shared/{project_id}/video/_tmp/scene_{i}_muxed.mp4` (trung gian); kiểm tra tồn tại (idempotency) |
| `adapters/messaging/consumer.py` | Consume `assemble_video`; publish `video_assembled`/`assembly_failed` (1 event duy nhất/command, cùng transaction với Inbox mark — không có progress event per-scene, khác Unit 5) |
| `adapters/logging/correlation.py` | Gắn `saga_id` vào log context |
