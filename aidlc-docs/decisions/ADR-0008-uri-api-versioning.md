# ADR-0008: URI-based API Versioning (System-wide)

## Status
Accepted

## Date
2026-08-05

## Stage
Low-Level Design (Unit 2: Content Plugin Service — decision applies system-wide)

## Context
Khi thiết kế Low-Level Design cho Content Plugin Service, cần xác định chiến lược API versioning cho REST endpoint. Ban đầu đề xuất chưa cần versioning (vì chỉ có 1 client nội bộ), nhưng người dùng yêu cầu áp dụng versioning ngay từ đầu.

## Options Considered
### Option A: URI versioning (`/v1/...`)
- What it is: Thêm tiền tố version vào đường dẫn, ví dụ `GET /v1/plugins`.
- Strengths: Rõ ràng, dễ thấy trên log/browser, dễ cache theo version, dễ route tới đúng version khi có nhiều version chạy song song.
- Trade-offs: Khi có breaking change phải thêm `/v2/...` và Gateway cần biết route tới đúng version.

### Option B: Header-based versioning
- What it is: URI không đổi, version nằm trong header (`Accept: application/vnd.api+json;version=2`).
- Strengths: URI ổn định.
- Trade-offs: Version ít hiển thị/khó thấy hơn khi debug trực tiếp qua browser/log đơn giản.

### Option C: Payload/field-level, chỉ thêm field mới
- What it is: Không dùng version số, quy ước schema chỉ thêm field, không xóa/đổi tên.
- Strengths: Tránh nhân bản endpoint theo version.
- Trade-offs: Giới hạn khả năng thay đổi cấu trúc khi thật sự cần breaking change.

## Decision
Chọn **Option A: URI versioning (`/v1/...`)**, áp dụng **ngay từ đầu cho mọi REST endpoint của mọi service** trong hệ thống (Content Plugin Service, và các unit REST khác sẽ thiết kế sau: API Gateway, Publisher OAuth endpoints, v.v.), không đợi đến khi cần breaking change.

**Deprecation policy**: Version cũ được giữ tối thiểu 1 chu kỳ phát triển sau khi version mới ra mắt. Vì hệ thống hiện chỉ có client nội bộ (GUI/Gateway) do chính chủ dự án kiểm soát, thông báo deprecation qua changelog nội bộ (`aidlc-docs/`), không cần cơ chế thông báo tự động tới bên thứ ba.

## Rationale
Người dùng chủ động yêu cầu versioning ngay từ đầu thay vì trì hoãn — quyết định ưu tiên khả năng tiến hóa API rõ ràng hơn ngay cả khi chi phí ban đầu (thêm 1 cấp `/v1/`) là nhỏ. URI versioning được chọn cụ thể vì tính hiển thị/dễ debug cao hơn header-based, phù hợp với quy mô dự án cá nhân nơi chính người phát triển sẽ là người debug thường xuyên nhất.

## Consequences
- **Positive**: Toàn bộ REST endpoint hệ thống nhất quán về versioning ngay từ MVP; dễ giới thiệu breaking change sau này (`/v2/...`) mà không phá vỡ client hiện có.
- **Negative / Accepted Trade-offs**: Mọi route REST phải có tiền tố `/v1/` — cần áp dụng nhất quán khi thiết kế các unit REST tiếp theo (API Gateway, Publisher Service OAuth endpoints); tăng nhẹ độ dài đường dẫn.
- **Follow-ups**: Áp dụng `/v1/` prefix khi thiết kế Low-Level Design của API Gateway (Unit 9) và mọi endpoint REST khác trong hệ thống; cập nhật `component-methods.md` (Application Design) để phản ánh tiền tố `/v1/` khi các unit đó được thiết kế.

## Related
- Design artifact: `aidlc-docs/construction/content-plugin-service/low-level-design/interface-contracts.md`
- Related ADRs: Không có (quyết định độc lập ở Low-Level Design)
