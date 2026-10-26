# ADR-0047: Chạy bước AI bằng AI ngoài qua cổng `Chat`; mỗi lượt đúng một prompt

## Status
Proposed

## Date
2026-10-26

## Stage
Construction (CR-102)

## Context
Creator muốn chạy các chỗ gọi AI ở bước 1–7 bằng một AI bên ngoài hệ thống: copy prompt, dán câu trả lời vào, để hệ thống chạy tiếp như khi tự gọi model. Mười một chỗ gọi AI ở bước 1–7 đều đi qua `LLMProviderPort.Chat` của authoring-service, mỗi chỗ tự ghép prompt từ prompt trong DB và dữ liệu các bước trước, rồi tự đọc và kiểm tra câu trả lời.

## Options Considered
### Option A: chặn ở cổng `Chat`
Một lớp bọc quanh client llm-service đọc chế độ từ `context`: bình thường gọi model; `prompt` trả prompt đã ghép rồi dừng; `paste` lấy đoạn dán làm câu trả lời.
- Strengths: một chỗ sửa thay cho mười một; prompt giống hệt bản chạy thật vì do chính use case ghép; phần đọc và kiểm tra reply không bị viết hai lần.
- Trade-offs: use case chạy tới tận lời gọi `Chat` ở chế độ `prompt`, nên phải không ghi DB trước lời gọi đó (đã kiểm cả mười một chỗ).

### Option B: endpoint dựng prompt riêng cho từng bước
- Trade-offs: ghép prompt hai lần, hai bản dễ lệch nhau.

### Option C: Creator tự dán vào ô JSON của từng bước
- Trade-offs: bỏ qua đường kiểm tra của use case.

## Decision
A.
- Request mang `X-External-AI: prompt|paste`; thân của `paste` có thêm `external_reply`. Middleware của authoring-service đặt chế độ vào `context` và, ở `prompt`, giữ lại câu trả lời của handler để trả `{"external_prompt"}` khi đã dựng được prompt (lỗi trước đó như thiếu dự án đi ra nguyên vẹn).
- Mỗi request chỉ cho đúng một lượt hỏi qua. Lượt thứ hai trả `ErrExternalAISingleTurn`; các vòng hỏi lại khi reply hỏng (`askUntilUsable`, `writeOne`) không chạy ở chế độ ngoài: báo lỗi để Creator dán lại. Nút gọi AI nhiều lượt trong một lần bấm (viết tất cả beat, viết lại cả beat sheet) không có nút AI ngoài; AI ngoài chạy từng beat.
- `paste` gọi `POST /v1/chat/external` của llm-service để đọc JSON bằng đúng hàm của `/v1/chat` và ghi `llm_calls` với `provider = "external"`; màn Nhật ký hiện nhãn "AI ngoài".
- Chế độ ngoài không cần API key.

## Consequences
- Thêm route mới ở llm-service, không đổi route cũ.
- Trường hợp có ghi DB trước lời gọi `Chat` sẽ để lại dấu vết ở chế độ `prompt`; bước mới thêm vào phải giữ quy ước "ghi sau khi có kết quả".
- Header `X-External-AI` được gateway cho phép qua CORS.
