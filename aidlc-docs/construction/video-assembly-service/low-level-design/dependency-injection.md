# Dependency Injection — Unit 6: Video Assembly Service

## Mechanism
Constructor injection thủ công — không đổi so với Unit 2/3/4/5.

## What Gets Injected vs Constructed Directly
- **Injected (abstraction)**: `VideoAssemblerPort` — `AssembleVideoUseCase` nhận instance implement `VideoAssemblerPort` (`FfmpegVideoAssembler`) qua constructor. Cho phép thay công cụ ghép video sau này (không phải ffmpeg) mà không sửa `application/`.
- **Constructed directly**: `InboxRepository`/`OutboxRepository`/`OutboxRelay` (Postgres, ADR-0013, mirror Unit 2/3/4/5), `artifact_paths.py` helper thuần túy.
- **Không có registry/plugin discovery** (khác Unit 2/5) — Unit 6 chỉ có 1 chiến lược assembly cố định, không cần cơ chế mở rộng động.

## Composition Root
`main.py`:
1. Khởi tạo `FfmpegVideoAssembler(timeout_seconds=env)`.
2. Khởi tạo `AssembleVideoUseCase(assembler=ffmpeg_assembler)`.
3. Kết nối PostgreSQL, khởi tạo `InboxRepository`, `OutboxRepository`.
4. Kết nối RabbitMQ, wire `AssembleVideoCommandHandler(use_case, pool, inbox, outbox)`, đăng ký consumer cho queue `video_assembly.commands`.
5. Khởi động `OutboxRelay` như background task.
6. Ghi sentinel `/tmp/ready`.

## Wiring Diagram
```
main.py
  ├── FfmpegVideoAssembler (implements VideoAssemblerPort)
  │     └── injected into → AssembleVideoUseCase
  ├── InboxRepository, OutboxRepository (Postgres, ADR-0013)
  │     └── injected into → AssembleVideoCommandHandler (AMQP consumer)
  │           └── consumes command "assemble_video" (queue video_assembly.commands)
  │           └── ghi event "video_assembled"/"assembly_failed" (1 row/command) vào Outbox, CÙNG transaction với Inbox mark
  └── OutboxRelay (background task)
        └── poll Outbox chưa publish → publish qua producer.py → đánh dấu published_at
```
