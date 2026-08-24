# NFR Design Patterns — Unit 6: Video Assembly Service

## CRUD vs CQRS
CRUD đơn giản trên `outbox_events`/`processed_messages` (bảng kỹ thuật, ADR-0013) — không phải business data model, không phải CQRS.

## Resilience Pattern
Không retry nội bộ. ffmpeg lỗi/timeout (sau `ASSEMBLY_TIMEOUT_SECONDS`) → `AssemblyEngineError` → `assembly_failed` ngay, để Orchestrator quyết định retry (transient). Lỗi validation (zero-trust — `MissingArtifactError`/`InvalidSceneIndexError`/`InconsistentMediaFormatError`) là permanent tại thời điểm đó — raise ngay, không retry tự động (input sai cần Orchestrator/upstream sửa trước khi retry có ý nghĩa).

## Idempotency Pattern
2 tầng, mirror Unit 3/5:
- **Message-level**: Inbox (`processed_messages`) dedupe `message_id` — 1 command = 1 message_id, 1 event kết quả.
- **Artifact-level**: kiểm tra file `final.mp4` tồn tại tại `/shared/{project_id}/video/final.mp4` trước khi assembly lại (Functional Design Rule 8).

Không cần lock/race-condition handling — Orchestrator gọi tuần tự theo saga, cùng lý do đã chấp nhận ở Unit 3/5.

## Saga Pattern
Participant trực tiếp, bước "Assemble Video" (sau "Render Scenes") trong Saga Render Pipeline. Không có compensating action — idempotent theo `project_id`; animation/audio clip input KHÔNG bị xoá dù assembly lỗi (Functional Design Rule 9), Orchestrator retry command nếu cần.

## Event-Driven Design
Consumer `assemble_video` (queue `video_assembly.commands`), publish đúng 1 trong 2 event: `video_assembled` HOẶC `assembly_failed` — **KHÁC BIỆT so với Unit 5**: đúng 1 Outbox row/command (giống Unit 2/3/4, không phải nhiều row per-scene, vì `assemble_video` không có khái niệm progress trung gian, Low-Level Design Question 10).

## Inbox/Outbox Pattern
PostgreSQL-backed (`video-assembly-db`, ADR-0013), cùng kiến trúc Unit 2/3/4 — Outbox row (event kết quả) và Inbox mark (`processed_messages`) ghi CÙNG 1 transaction duy nhất (khác Unit 5's per-scene commit riêng, vì ở đây chỉ có 1 event/command nên không cần publish sớm giữa chừng). `OutboxRelay` publish bất đồng bộ (poll định kỳ).

## Security Pattern
Zero-trust validation trong domain/application layer (Functional Design Rule 1, 2, 3) — validate file tồn tại, `scene_index` hợp lệ (dãy liên tục từ 0), và media format đồng nhất (ffprobe pre-check) trước khi chạy ffmpeg. Không auth/rate-limit riêng — chỉ Orchestrator gửi command qua RabbitMQ nội bộ.
