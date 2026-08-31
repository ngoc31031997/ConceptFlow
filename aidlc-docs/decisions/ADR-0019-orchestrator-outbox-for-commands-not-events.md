# ADR-0019: Orchestrator's Outbox Publishes Commands, Not Events

## Status
Accepted

## Date
2026-08-31

## Stage
NFR Design (Unit 8: Orchestrator Service)

## Context
Units 2–7 use the Transactional Outbox pattern to guarantee at-least-once publication of **events** they emit as the result of completing their business step (e.g. `script_parsed`, `video_published`). Orchestrator Service is structurally different: it does not produce a result event for external consumption after each step — instead, upon receiving an event, it decides the next action and must reliably dispatch a **command** to the next service. NFR Design needed to confirm whether the same Outbox mechanism applies, and if so, to what payload type, since reusing the pattern with a different semantic could confuse future implementers expecting "Outbox = outgoing event" as in every other unit.

## Options Considered
### Option A: Outbox for commands, Inbox for incoming events (Chosen)
- What it is: `outbox_events` table queues commands to be sent (same technical mechanism — write in the same DB transaction as the state update, a background relay polls and publishes); `processed_messages` (Inbox) dedupes incoming events by `message_id`.
- Strengths: preserves the crash-safety guarantee that matters here — a command must not be lost or duplicated between the state transition and the publish; reuses a well-understood technical pattern already proven in Units 2–7; keeps Inbox and Outbox as two independent, single-purpose mechanisms (dedupe-in vs. guarantee-out).
- Trade-offs: the table name `outbox_events` is technically storing commands, not events — a naming mismatch that must be documented clearly (this ADR + `messaging-design.md`) so Code Generation and future maintainers aren't misled by the Unit 2–7 convention.

### Option B: Rename the table to `outbox_commands` for clarity
- What it is: Same mechanism as Option A but rename the table to avoid the naming mismatch.
- Strengths: self-documenting, avoids the "outbox_events holds commands" confusion entirely.
- Trade-offs: diverges from the column/table naming already implemented in Low-Level Design's `adapters/postgres/outbox.go` (commit `c60a06a`) and the schema referenced throughout `interface-contracts.md`/`module-structure.md` — would require reopening and re-approving already-completed LLD artifacts for a purely cosmetic gain.

## Decision
Chọn **Option A**: giữ nguyên tên bảng/mechanism `outbox_events` như LLD đã dùng, nhưng ghi nhận rõ ràng bằng ADR + tài liệu rằng ở Unit 8, Outbox chứa COMMAND (không phải event như các unit khác).

## Rationale
Cơ chế kỹ thuật (transactional write + polling relay) giống hệt nhau bất kể payload là command hay event — sự khác biệt chỉ ở tầng ý nghĩa nghiệp vụ (semantic), không phải kiến trúc. Đổi tên bảng chỉ để "đúng nghĩa" sẽ buộc phải sửa lại Low-Level Design đã duyệt (module-structure.md, interface-contracts.md) mà không mang lại lợi ích kỹ thuật nào — documentation rõ ràng (ADR này + `messaging-design.md`) đủ để tránh nhầm lẫn khi Code Generation.

## Consequences
- **Positive**: Không cần rework LLD đã duyệt; cơ chế crash-safety cho command dispatch được đảm bảo giống hệt cách các unit khác đảm bảo cho event.
- **Negative / Accepted Trade-offs**: Tên bảng `outbox_events` gây hiểu lầm nếu đọc code Unit 8 mà không biết ADR này — giảm thiểu bằng comment trong code (Code Generation stage) trỏ về ADR-0019.
- **Follow-ups**: Code Generation cần thêm comment ngắn ở `outbox.go`/schema migration trỏ tới ADR-0019 để giải thích semantic khác biệt.

## Related
- Design artifact: `aidlc-docs/construction/orchestrator-service/nfr-design/messaging-design.md`
- Related ADRs: ADR-0013 (Inbox/Outbox pattern gốc, Unit 2), ADR-0007 (Saga orchestration-based)
