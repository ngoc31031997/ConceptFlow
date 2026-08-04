# ADR-0006: Dynamic Plugin Loading for Content Plugin Service

## Status
Accepted

## Date
2026-08-04

## Stage
Application Design

## Context
Content Plugin Service cần một cơ chế nạp content-type plugin (NFR1 — Extensibility). Cần quyết định plugin được đăng ký tĩnh trong code hay nạp động từ thư mục ngoài.

## Options Considered
### Option A: Plugin nội bộ đăng ký tĩnh (static registry)
- What it is: Plugin là Python class implement interface chung, đăng ký qua config/registry cố định trong code (vd. `PLUGIN_REGISTRY = {"programming": ProgrammingPlugin()}`).
- Strengths: Đơn giản, đủ dùng khi chỉ có 1 plugin (lập trình) trong MVP, vẫn tuân thủ NFR1 vì thêm domain mới chỉ cần thêm 1 class + đăng ký, không sửa core.
- Trade-offs: Chưa hỗ trợ nạp plugin động từ file ngoài mà không sửa code.

### Option B: Plugin động (dynamic loading)
- What it is: Hệ thống quét thư mục `plugins/` và tự động nạp bất kỳ module Python nào implement đúng `ContentPluginPort`, không cần sửa code core khi thêm plugin mới.
- Strengths: Mở rộng thực sự không cần sửa/redeploy code core, kể cả không nằm trên nhánh chính — khớp chặt nhất với tinh thần "pluggable" của NFR1.
- Trade-offs: Phức tạp hơn (dynamic import, validate plugin interface tại runtime, xử lý lỗi nạp plugin hỏng); cần cân nhắc an toàn khi nạp code Python tùy ý (dù ở đây là do chính Creator viết, không phải bên thứ ba không tin cậy).

## Decision
Chọn **Option B: Plugin động (dynamic loading từ thư mục `plugins/`)**.

## Rationale
Người dùng chọn rõ ràng Option B dù được đề xuất Option A là phương án đơn giản hơn cho MVP. Quyết định được tôn trọng theo nguyên tắc User Control — Creator ưu tiên khả năng mở rộng plugin thực sự không cần sửa code core ngay từ đầu, phù hợp với định hướng dài hạn "mở rộng sang domain giáo dục khác" đã nêu ở `requirements.md`.

## Consequences
- **Positive**: Thêm domain giáo dục mới (vd. Tiếng Anh) chỉ cần thêm 1 file plugin vào thư mục `plugins/`, không cần sửa/redeploy Content Plugin Service core.
- **Negative / Accepted Trade-offs**: Content Plugin Service cần logic dynamic import + validate interface tại runtime; cần xử lý rõ ràng trường hợp plugin lỗi/không hợp lệ (không làm crash toàn service); vì chỉ Creator tự viết plugin (không phải bên thứ ba), rủi ro bảo mật từ việc nạp code động được chấp nhận ở mức độ hiện tại (phù hợp với Security Baseline extension đã tắt).
- **Follow-ups**: Low-Level Design (Content Plugin Service unit) cần thiết kế cụ thể cơ chế discovery/validation/error-handling cho dynamic plugin loading.

## Related
- Design artifact: `aidlc-docs/inception/application-design/components.md`
- Related ADRs: ADR-0002 (Hexagonal — `ContentPluginPort` là port mà mỗi plugin động phải implement)
