# ADR-0049: `visual-service` sở hữu tiền kỳ hình ảnh (bước 8–14)

## Status
Proposed

## Date
2026-11-10

## Stage
Inception / Construction (CR-112)

## Context
Flow có 23 bước. Bước 1–7 (Tiền kỳ · Nội dung) nằm trong authoring-service. Creator muốn authoring chỉ lo viết kịch bản. Bước 8–14 (Tiền kỳ · Hình ảnh) làm việc khác hẳn: đặt quy tắc hình, phương án hình, storyboard, tài nguyên, chuyển động, animatic. Bước 8 là Ngôn ngữ hình ảnh (Visual Concept). Bước Phân rã kịch bản cũ bị bỏ, vì dữ liệu của nó đã có ở bước 4–7.

Bước 8 cần đọc bản chốt của bước 1–7, gọi AI, có đơn vị duyệt, có rà soát và chốt phiên bản. Các bước 9–14 sẽ đọc bản chốt của bước 8.

## Options Considered
### Option A: service mới `visual-service`
- Strengths: ranh giới rõ (nội dung và hình ảnh); bước 9–14 có chỗ để thêm vào; authoring không phình thêm.
- Trade-offs: thêm một service và một DB; phải làm lại phần khung đơn vị duyệt, AI ngoài và chốt phiên bản đang có ở authoring; dữ liệu bước 1–7 phải đọc qua HTTP.

### Option B: thêm bước 8 vào authoring
- Strengths: dùng lại sẵn mọi khung.
- Trade-offs: trái ý Creator; authoring ôm cả nội dung lẫn hình ảnh.

### Option C: đặt ở rendering-service
- Trade-offs: rendering dựng video bằng Remotion, không có DB cho dữ liệu dự án và không gọi AI; trộn hai việc khác nhau.

## Decision
A.
- `visual-service` viết bằng Go, kiến trúc hexagonal như authoring. Có Postgres riêng (`visual-db`).
- **Đọc bước 1–7:** authoring thêm route nội bộ chỉ đọc `GET /internal/v1/authoring/{id}/script-versions/latest`. Route trả nguyên bản kịch bản chốt mới nhất của bước 7; bản này đã chụp brief, dữ kiện, câu chuyện, beat, lời thoại, ghi chú bàn giao và thuật ngữ, mỗi phần kèm hash. visual-service không chép các dữ liệu này, chỉ lưu hash của từng phần.
- **AI:** gọi llm-service (`/v1/chat`, `/v1/chat/external`). Prompt và model mặc định lấy từ thư viện prompt ở llm-service (CR-113, ADR-0048). Model riêng từng video lưu ở visual-service. AI ngoài và DeepSeek chạy qua header `X-External-AI` như ADR-0047; middleware được làm lại trong visual-service.
- **Xoá video:** visual-service không giữ file nên không phải một `PurgeTarget`. Giống authoring: ở cuối saga xoá, orchestrator gọi `DELETE /internal/v1/visual/{id}`, cố gắng một lần, lỗi thì ghi log.
- **Bản chốt** của mỗi bước hình ảnh là một tài liệu JSON bất biến có phiên bản. Bước sau đọc qua route nội bộ của visual-service, theo phần mà bước đó cần.
- orchestrator `flow.go` không đổi trong CR-112.

## Consequences
- CR-112 phụ thuộc CR-113 (thư viện prompt ở llm-service).
- Đổi ở các service khác: authoring thêm một route đọc; orchestrator gọi thêm một route xoá cuối saga; api-gateway thêm proxy; docker compose thêm `visual-service` và `visual-db`.
- Khi authoring không trả lời thì bước 8 báo lỗi rõ, không dùng dữ liệu cũ.
- Khung đơn vị duyệt tồn tại ở hai service. Nếu sau này trùng lặp quá nhiều thì tách thành thư viện Go dùng chung (chưa làm).
