# CR-072 U3c · Câu hỏi thiết kế

Điền chữ cái sau `[Answer]:`.

## Question 1
Phạm vi của U3c. `unit-of-work.md` ghi U3c còn đụng orchestrator (`flow.go`, luồng 15 bước) và việc bước Kịch bản hiện có đọc câu chuyện đã duyệt. Ở U3b bạn đã thu hẹp về đúng màn trong mockup.

A) Chỉ màn Câu chuyện như mockup, giống U3b — ✅ nhỏ, không đụng luồng đang chạy, ⚠️ bước Kịch bản hiện có chưa đọc câu chuyện; "Sang beat sheet" mở trang "chưa phát triển".

B) Cả phạm vi trong unit-of-work — ✅ đúng kế hoạch gốc, ⚠️ đổi số bước ở orchestrator (cần migration `project_events`) và sửa bước Kịch bản đang chạy; rủi ro cao, có thể hỏng luồng hiện tại.

C) Other (mô tả sau [Answer]:)

[Answer]: C — Làm theo giao diện mockup đã chốt, không đụng orchestrator hay bước Kịch bản đang chạy. Chỗ nào chưa rõ thì hỏi Creator. (Creator: "u3b là cái gì tôi đã bảo xoá hết rồi mà. làm theo như giao diện chỗ nào chưa hiểu thì hỏi tôi")

## Question 2
Cách lưu sáu ô của khung. Mockup cho mỗi ô một nút Sửa tay và Làm lại kèm ghi chú riêng, cả khung một chỗ duyệt.

A) Mỗi ô một đơn vị `story_field`, thêm cổng `story:main` — ✅ khớp mockup và FR-29.2, ô Creator viết giữ riêng từng ô, làm lại từng ô qua API đơn vị chung, ⚠️ nhiều đơn vị hơn (6 + cổng).

B) Một đơn vị `story:main` chứa cả sáu ô — ✅ ít đơn vị, ⚠️ làm lại một ô phải viết thêm đường riêng ngoài API đơn vị chung; đánh dấu "Bạn viết" theo ô khó hơn.

C) Other (mô tả sau [Answer]:)

[Answer]: A — lưu thành 6 đơn vị riêng.

## Question 3
"Làm lại kèm ghi chú" một ô: AI viết gì?

A) Chỉ ô đó, đọc năm ô còn lại làm ngữ cảnh — ✅ nhanh, đúng ý "làm lại ô này", ⚠️ các ô khác có thể lệch nhẹ so với ô mới; Creator tự làm lại ô lệch.

B) Ô đó rồi viết lại mọi ô không phải Creator viết cho khớp — ✅ khung luôn nhất quán, ⚠️ chậm hơn, làm mất ô AI viết mà Creator đã ưng.

C) Other (mô tả sau [Answer]:)

[Answer]: A — viết riêng ô đó.

## Question 4
Sửa hay làm lại một ô khi câu chuyện đã duyệt.

A) Câu chuyện về chờ duyệt (như mockup) — ✅ an toàn, phải xem lại, ⚠️ thêm một lần bấm duyệt.

B) Giữ đã duyệt trừ khi ô bị xoá trắng (như brief) — ✅ ít bấm, ⚠️ có thể duyệt nhầm nội dung đã đổi.

C) Other (mô tả sau [Answer]:)

[Answer]: A — sửa một ô thì câu chuyện về chờ duyệt.

## Question 5
Quy ước kỹ thuật cho U3c (mục bắt buộc của Low-Level Design): URI `/v1`, chỉ thêm trường; tiêm phụ thuộc bằng hàm khởi tạo, nối dây ở `main.go`; truy vết bằng `X-Operation-Id` như U3a/U3b; kiểu mã Go/TS theo `docs/code-standards-rules.md`.

A) Giữ đúng như vậy — ✅ nhất quán với phần đã có.

B) Other (mô tả sau [Answer]:)

[Answer]: A — giữ như vậy; thêm quy tắc đánh version /v1 vào thiết kế hệ thống (integration-boundaries.md, theo ADR-0008).
