# ADR-0004: API Gateway Decision

## Status
Accepted

## Date
2026-08-04

## Stage
High-Level Design

## Context
Với kiến trúc Microservices (ADR-0001), hệ thống có nhiều service backend độc lập. Cần quyết định liệu GUI gọi trực tiếp từng service hay qua một điểm vào hợp nhất.

## Options Considered
### Option A: Reverse proxy/gateway đơn giản đứng trước tất cả service
- What it is: Một gateway nhẹ (vd. FastAPI gateway service hoặc Traefik) đứng trước tất cả service; GUI chỉ gọi vào gateway, gateway định tuyến request đến đúng service.
- Strengths: GUI chỉ cần biết 1 điểm vào duy nhất; dễ thêm auth/rate-limit tập trung sau này; đơn giản hóa việc thêm/bớt service phía sau mà không đổi GUI.
- Trade-offs: Thêm 1 thành phần cần vận hành; thêm 1 điểm có thể lỗi (single point of failure) dù chạy local.

### Option B: Không có gateway, GUI gọi trực tiếp từng service
- What it is: GUI biết địa chỉ/port của từng service và gọi trực tiếp.
- Strengths: Không cần thêm thành phần trung gian.
- Trade-offs: GUI phải biết chi tiết topology của toàn bộ backend; khó thêm/đổi service mà không sửa GUI; không có điểm tập trung cho auth/routing sau này.

## Decision
Chọn **Option A: Reverse proxy/gateway đơn giản**.

## Rationale
Người dùng xác nhận rõ ràng cần một gateway (qua AskUserQuestion follow-up) sau khi làm rõ mâu thuẫn ban đầu ở High-Level Design plan. Với kiến trúc Microservices đã chọn, gateway giúp GUI không cần biết chi tiết topology backend và tạo điểm tập trung cho việc mở rộng sau này (auth, rate-limit, thêm service mới).

## Consequences
- **Positive**: GUI đơn giản hóa, chỉ gọi 1 endpoint; dễ thêm concern cross-cutting (auth, logging tập trung) sau này tại gateway.
- **Negative / Accepted Trade-offs**: Thêm 1 service cần vận hành và maintain (dù đơn giản); thêm 1 hop network cho mọi request.
- **Follow-ups**: Application Design cần định nghĩa routing table cụ thể (endpoint nào map tới service nào) tại Gateway.

## Related
- Design artifact: `aidlc-docs/inception/high-level-design/architecture-overview.md`
- Related ADRs: ADR-0001
