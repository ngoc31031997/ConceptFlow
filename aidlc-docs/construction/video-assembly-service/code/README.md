# Unit 6: Video Assembly Service — Code Summary

## Overview
Ghép animation clip (câm) + audio clip (giọng đọc) + nhạc nền tùy chọn của mỗi scene thành 1 video .mp4 hoàn chỉnh (FR5.1, FR5.2 — Stories C4, C5). Message-driven: consumer `assemble_video` (queue `video_assembly.commands`), publisher 1 event/command (`video_assembled`/`assembly_failed`) qua PostgreSQL Outbox (ADR-0013).

## Structure (Hexagonal / Ports & Adapters, ADR-0002)
```
services/video-assembly/
├── domain/            # SceneAssemblyInput, VideoAssemblyRequest, VideoAssemblyResult, errors, VideoAssemblerPort
├── application/        # AssembleVideoUseCase — zero-trust validation + idempotency check
├── adapters/
│   ├── assembly/        # MediaFormatInspector (ffprobe), FfmpegVideoAssembler (ffmpeg, implements VideoAssemblerPort)
│   ├── storage/         # artifact_paths.py — shared-volume path convention
│   ├── messaging/       # AMQP consumer/producer
│   ├── persistence/     # Inbox/Outbox/Relay (ADR-0013, copied verbatim from Unit 2/3/4/5)
│   └── logging/         # correlation.py
├── main.py             # Composition root
└── tests/
```

## Key Implementation Notes
- **No batch wrapper**: unlike Unit 2/3/4/5, `assemble_video` is already a single operation over a whole project's scenes (Low-Level Design Question 10) — `AssembleVideoUseCase` is the entire application layer.
- **Explicit scene_index**: payload carries `scenes: [{scene_index, clip_path, audio_path}]`, not parallel arrays — `FfmpegVideoAssembler` sorts and validates the sequence itself (Functional Design Question 2, discovered mid-stream and retrofitted into the approved Low-Level Design).
- **Media format pre-check**: `MediaFormatInspector` runs ffprobe on every clip before mux/concat, raising `InconsistentMediaFormatError` on mismatch (Functional Design Question 3).
- **ffmpeg pipeline**: per-scene mux (`-c:v copy`) → concat demuxer (`-c copy`) → optional background-music overlay (fixed 20% volume, `amix`, loop/shortest) — all via `subprocess`, run in a `ThreadPoolExecutor` with `ASSEMBLY_TIMEOUT_SECONDS` (default 180s).
- **Idempotency**: artifact-level (`/shared/{project_id}/video/final.mp4` existence check) + message-level (Inbox dedupe).
- **Single Outbox row/command**: unlike Unit 5, no per-scene progress events — exactly one `video_assembled`/`assembly_failed` row written in the same transaction as the Inbox mark.

## Test Coverage
- `tests/application/test_assemble_video.py`: zero-trust validation (Rule 1), idempotency (Rule 8)
- `tests/adapters/test_ffmpeg_assembler.py`: scene_index validation, media-format consistency check, ffmpeg error handling — all via mocked `subprocess.run`, no real ffmpeg/ffprobe needed
- `tests/adapters/test_messaging.py`, `test_persistence.py`, `test_relay.py`: Inbox/Outbox/consumer wiring, mirrors Unit 2/3/4/5

## Traceability
- Stories: C4, C5
- Requirements: FR5.1, FR5.2
- Design: `aidlc-docs/construction/video-assembly-service/{low-level-design,functional-design,nfr-requirements,nfr-design,infrastructure-design}/`
