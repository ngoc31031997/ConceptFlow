# ADR-0051: Bước 11 Storyboard chia việc giữa visual-service và rendering

## Status
Accepted

## Date
2026-11-20

## Stage
Inception / Application Design (CR-117)

## Context
ADR-0049 giao visual-service sở hữu các bước tiền kỳ hình ảnh: gọi AI, đơn vị duyệt (`review_units`), rà soát, chốt phiên bản. Bước 11 cần thêm hai thứ mà visual-service không có: vẽ ảnh khung (Remotion, Chromium) và vẽ tài sản dự án bằng bộ vẽ AI (thư viện hình đã chuyển sang rendering ở ADR-0050). Creator muốn dữ liệu output của bước 11 (tài sản, shot, khung, bản chốt) nằm ở DB rendering.

## Options Considered
### A. visual-service điều phối bản nháp; rendering giữ tài sản, khung và bản chốt
- Bản nháp shot (nội dung Creator đang sửa) ở `review_units` của visual-service như bước 8–10, nên dùng lại toàn bộ khung desk (duyệt, rà soát, stale, chốt). Rendering giữ `project_assets`, `storyboard_shots` (bản ghi khung: đường dẫn ảnh, hash) và `storyboard_versions` (tài liệu chốt).
- Strengths: dùng lại khung đã chạy ở bước 8–10; rendering chỉ có thứ nó vốn làm (vẽ, lưu file).
- Trade-offs: nội dung shot nằm ở hai nơi (bản nháp ở visual-service, bản ghi khung và bản chốt ở rendering); phải gọi HTTP giữa hai service khi vẽ khung và khi chốt.

### B. Mọi trạng thái ở rendering; visual-service chỉ gọi AI
- Strengths: một nơi giữ dữ liệu bước 11.
- Trade-offs: phải viết lại khung duyệt, stale, rà soát, chốt ở Python trong rendering; hai chỗ làm cùng việc cho các bước tiền kỳ.

### C. Mọi thứ ở visual-service (vẽ khung gọi rendering, nhưng lưu ảnh và bảng ở visual-service)
- Trade-offs: trái ý Creator (output ở rendering); visual-service giữ file ảnh.

## Decision
A.
- visual-service: viết tài sản (AI), viết shot theo từng unit (AI), rà soát (AI), duyệt, stale, chốt; bản nháp ở `review_units` (`storyboard_unit`, `storyboard_review`).
- rendering: bảng `project_assets`, `storyboard_shots`, `storyboard_versions`; vẽ tài sản bằng bộ vẽ AI có sẵn; dựng khung bằng composition Remotion `storyboard-frame` (không gọi AI); ghi `/shared/{project_id}/storyboard/`.
- Khung vẽ khi shot được tạo hoặc sửa bố cục; bản ghi khung (`storyboard_shots`) là kết quả đã vẽ, không phải bản sửa. Tài liệu chốt do visual-service dựng và gửi cho rendering lưu; bước sau đọc từ rendering.
- visual-service gọi rendering qua HTTP nội bộ (`RENDERING_URL`); không có đường ngược.

## Consequences
- visual-service thêm client rendering và cấu hình `RENDERING_URL`.
- Lỗi khung (vẽ thất bại) chỉ ảnh hưởng một shot, không chặn bản nháp.
- Tài sản chưa duyệt không được dùng ở shot; xoá tài sản đang được shot dùng bị từ chối.
- Bước 12 "Thiết kế tài nguyên" trên bản đồ luồng trùng ý với phần tài sản của bước này; để một CR sau quyết định gộp hay giữ riêng.
