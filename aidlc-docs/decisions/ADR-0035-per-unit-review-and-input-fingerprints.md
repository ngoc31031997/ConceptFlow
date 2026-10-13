# ADR-0035: Khung sửa theo đơn vị và dấu đầu vào để chỉ làm lại phần bị ảnh hưởng

## Status
Accepted

## Date
2026-10-13

## Stage
Application Design (CR-072)

## Context
Hiện đổi dàn ý hay hình ảnh thì phần sau của cả video bị xoá (hộp xác nhận ở `ScriptOutlineStepPage.tsx`, `VisualDirectorStepPage.tsx`), và mọi lượt chạy lại tính theo cả bước. Creator muốn sửa đúng một mục, cảnh, shot hay câu, chỉ phần phụ thuộc phải làm lại (P-1 A, FR-17), và muốn mỗi chỗ duyệt ít phải sửa (câu 3). AD-1 và AD-2 hỏi ai giữ trạng thái và cách biết phần nào bị ảnh hưởng.

## Options Considered
### Option A: authoring-service và orchestrator cùng một mô hình đơn vị, dấu đầu vào trên từng đơn vị
- What it is: authoring-service giữ đơn vị tiền kỳ và sản xuất, orchestrator giữ đơn vị hậu kỳ. Mỗi đơn vị ghi hash của các đơn vị nó được dựng từ; đầu vào đổi thì đơn vị thành `stale`.
- Strengths: đúng ranh giới ADR-0029; đúng cả khi Creator sửa tay; không cần luật xoá cho từng thao tác; mở rộng cách CR-050 đã làm cho danh sách hình (`illustrations_storyboard_sha`).
- Trade-offs: mô hình có ở hai service; mỗi loại đơn vị phải khai báo đầu vào.

### Option B: một service mới giữ trạng thái duyệt và ghi chú
- What it is: service riêng cho duyệt.
- Strengths: một chỗ duy nhất.
- Trade-offs: thêm service để chạy và deploy; nội dung và trạng thái nằm hai nơi.

### Option C: luật xoá viết tay theo từng thao tác
- What it is: như hiện nay, mỗi thao tác sửa có danh sách những gì bị xoá.
- Strengths: không cần mô hình mới.
- Trade-offs: dễ sót; chính cách này đang làm đổi dàn ý thì xoá cả video.

## Decision
A (AD-1 A, AD-2 A).

## Rationale
Dấu đầu vào biến câu hỏi "cái gì phải làm lại" thành phép so sánh hash, nên không phụ thuộc thao tác nào đã gây ra thay đổi. Cách này đã chạy đúng cho danh sách hình ở CR-050. Giữ đơn vị ở service đang giữ nội dung tránh được việc đồng bộ hai nơi.

## Consequences
- **Positive**: sửa một cảnh chỉ làm lại cảnh đó; có thể hiện "sẽ làm lại" trước khi lưu; đếm được tỉ lệ duyệt ngay lần đầu (NFR-2).
- **Negative / Accepted Trade-offs**: mỗi loại đơn vị cần một `UnitGenerator` khai báo đầu vào; mô hình đơn vị ở orchestrator và authoring-service phải giữ khớp bằng test.
