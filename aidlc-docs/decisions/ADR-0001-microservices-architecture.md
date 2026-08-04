# ADR-0001: Microservices Architecture (Macro Decomposition)

## Status
Accepted

## Date
2026-08-04

## Stage
High-Level Design

## Context
Hệ thống cần được tổ chức macro-level. Ứng dụng chạy local trên máy cá nhân qua Docker, phục vụ 1 người dùng duy nhất, không có yêu cầu render song song/scale (theo requirements.md NFR2, NFR3).

## Options Considered
### Option A: Modular Monolith
- What it is: Một backend service duy nhất chứa các module tách biệt rõ ràng (Plugin System, Script Parser, Renderer, TTS, Video Assembler, YouTube Publisher), cùng chạy trong 1 process/container.
- Strengths: Đơn giản triển khai, dễ debug, không cần quản lý nhiều service, phù hợp quy mô dự án cá nhân.
- Trade-offs: Khó scale ngang nếu sau này cần xử lý nhiều video song song trên nhiều máy; ranh giới plugin dễ bị xói mòn nếu không kỷ luật.

### Option B: Microservices
- What it is: Mỗi service (Render, TTS, Publisher, Plugin, Script Processing, Video Assembly) chạy như container độc lập, giao tiếp qua HTTP, cùng trên 1 máy qua docker-compose.
- Strengths: Mỗi service scale/deploy độc lập, cô lập lỗi tốt, ranh giới rõ ràng, dễ tách máy riêng sau này nếu cần.
- Trade-offs: Thêm chi phí vận hành (network giữa container, nhiều Dockerfile, cần API Gateway) so với 1 người dùng chạy local.

## Decision
Chọn **Option B: Microservices**.

## Rationale
Người dùng xác nhận rõ ràng ưu tiên Microservices dù được đề xuất Modular Monolith làm phương án đơn giản hơn. Sau khi làm rõ mâu thuẫn với ràng buộc "single-container Docker local" (xem clarification trong `high-level-design-clarification-questions.md`), người dùng khẳng định đây là lựa chọn có chủ đích: chấp nhận thêm độ phức tạp vận hành (nhiều container qua docker-compose, cần API Gateway) để có ranh giới service rõ ràng ngay từ đầu và dễ tách máy sau này. Quyết định được tôn trọng theo nguyên tắc User Control của AI-DLC.

## Consequences
- **Positive**: Ranh giới service rõ ràng ngay từ đầu, mỗi service có thể được implement/test độc lập ở Construction Phase (Units Generation sẽ map 1 unit ≈ 1 service); dễ tách sang nhiều máy/scale độc lập nếu nhu cầu thay đổi sau này.
- **Negative / Accepted Trade-offs**: Cần thêm API Gateway (xem ADR-0004); nhiều Dockerfile/container cần quản lý trong docker-compose; giao tiếp qua network (dù local) thay vì function call trực tiếp, tăng độ phức tạp debug so với monolith.
- **Follow-ups**: Application Design cần xác định rõ contract API giữa các service; Infrastructure Design (per-unit) cần thiết kế docker-compose cụ thể.

## Related
- Design artifact: `aidlc-docs/inception/high-level-design/architecture-overview.md`
- Related ADRs: ADR-0004 (API Gateway), ADR-0005 (Orchestration pattern)
