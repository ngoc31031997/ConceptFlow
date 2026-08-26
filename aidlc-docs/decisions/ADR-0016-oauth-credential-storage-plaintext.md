# ADR-0016: OAuth Credential Storage — Plaintext for MVP

## Status
Accepted

## Date
2026-08-24

## Stage
Low-Level Design (Unit 7: Publisher Service)

## Context
Story E1 requires YouTube OAuth authentication once, with the resulting `access_token`/`refresh_token` persisted so the Creator never has to re-authenticate. These tokens are sensitive — they grant upload access to the Creator's YouTube channel. Low-Level Design needed to decide how they are stored at rest in the Publisher Service's `oauth_credentials` table.

## Options Considered
### Option A: Plaintext in PostgreSQL (Chosen)
- What it is: Store `access_token`/`refresh_token` as plain text columns in `oauth_credentials`, no additional encryption layer.
- Strengths: Simplest implementation, no key management surface, no rotation mechanism needed.
- Trade-offs: If the Postgres data volume is copied/backed up/exfiltrated, the tokens are readable directly.

### Option B: Application-level encryption (Fernet symmetric encryption)
- What it is: Encrypt `access_token`/`refresh_token` with Fernet before writing, key read from `CREDENTIAL_ENCRYPTION_KEY` env var.
- Strengths: Defense in depth — a leaked Postgres backup alone doesn't expose usable tokens.
- Trade-offs: Adds key management complexity (where the key itself lives, how it's rotated if compromised) disproportionate to the actual threat model at this project's scale.

## Decision
Chọn **Option A: Plaintext** cho MVP.

## Rationale
Hệ thống chạy hoàn toàn local trên máy cá nhân của chính Creator (single-user, không multi-tenant, không public-facing — theo `technology-direction.md`'s "chỉ 1 người dùng, chạy local" constraint). Postgres container chỉ accessible trong Docker network nội bộ, không expose ra host. Trong threat model này, nếu máy bị compromise tới mức kẻ tấn công đọc được Postgres data volume, encryption key (nếu lưu cùng máy, vd. env var) cũng bị lộ theo — mã hóa không giảm rủi ro thực tế mà chỉ thêm bề mặt quản lý key (nơi lưu key, cơ chế rotation nếu lộ) không tương xứng với lợi ích ở quy mô MVP hiện tại.

## Consequences
- **Positive**: Đơn giản, không cần quản lý encryption key, giảm bề mặt lỗi cấu hình.
- **Negative / Accepted Trade-offs**: Nếu dự án mở rộng sang multi-user/cloud-hosted sau này, threat model thay đổi căn bản (nhiều Creator, backup có thể rời khỏi máy cá nhân) — bắt buộc phải bổ sung mã hóa tại thời điểm đó.
- **Follow-ups**: Khi/nếu chuyển sang multi-user hoặc triển khai không phải local-only, revisit ADR này và implement Option B (hoặc dùng secret manager của cloud provider thay vì tự quản lý key).

## Related
- Design artifact: `aidlc-docs/construction/publisher-service/low-level-design/module-structure.md` (`adapters/persistence/credential_store.py`)
- Related ADRs: Consistent with the "single-user, local-only" scope established across ADR-0001/ADR-0003 (`technology-direction.md`)
