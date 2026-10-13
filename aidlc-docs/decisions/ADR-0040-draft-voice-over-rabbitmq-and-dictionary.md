# ADR-0040: Giọng sơ bộ qua RabbitMQ ngoài saga render; từ điển phát âm ở authoring-service

## Status
Accepted

## Date
2026-10-13

## Stage
Application Design (CR-072)

## Context
Giọng phải có trước animatic (FR-9), tức là trước saga render, trong lúc project còn là bản nháp. Creator muốn sửa cách đọc từng câu và có từ điển phát âm dùng chung (FR-20, P-6 A). tts hiện chỉ nhận lệnh qua RabbitMQ, dùng Azure SSML, và lưu file theo nội dung câu. AD-6 và AD-7 hỏi cách làm.

## Options Considered
### Option A: orchestrator gửi lệnh `synthesize_lines` qua RabbitMQ, ngoài saga
- What it is: authoring-service báo orchestrator khi có câu cần đọc; orchestrator gửi lệnh, nhận kết quả, chuyển về authoring-service.
- Strengths: tts giữ một kiểu giao tiếp; có sẵn hàng đợi, thử lại, tiến độ.
- Trade-offs: thêm một cặp lệnh và sự kiện.

### Option B: authoring-service gọi tts qua HTTP
- Trade-offs: tts thêm một kiểu giao tiếp và một đường lỗi mới.

### Option C: từ điển ở authoring-service, câu gửi đi đã kèm cách đọc
- What it is: authoring-service tra từ điển, gửi chữ, "đọc là", tốc độ, khoảng nghỉ; tts chỉ dựng SSML.
- Strengths: tts không cần biết từ điển; khoá cache gồm cả cách đọc, nên đổi từ điển thì đúng câu bị ảnh hưởng tự đọc lại.

### Option D: từ điển ở tts
- Trade-offs: cache phải tính cả phiên bản từ điển; giao diện phải gọi thêm một service.

## Decision
A (AD-6 A) và C (AD-7 A).

## Rationale
Giữ tts chỉ nhận lệnh qua RabbitMQ thì không phải thêm cơ chế chịu lỗi mới. Đặt từ điển cạnh các cài đặt kênh khác ở authoring-service, và đưa cách đọc vào nội dung lệnh, giữ cho tts không có trạng thái và cache luôn đúng.

## Consequences
- **Positive**: sửa một câu chỉ đọc lại câu đó; bước TTS của saga dùng lại file đã có.
- **Negative / Accepted Trade-offs**: khoá tên file giọng đổi (thêm cách đọc); orchestrator có thêm luồng lệnh ngoài saga.
