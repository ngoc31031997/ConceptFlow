# ADR-0046: Nhật ký cuộc gọi AI lưu cả prompt; Ollama luôn nhận `num_ctx` và prompt quá dài bị cắt giữa

## Status
Proposed

## Date
2026-10-25

## Stage
Requirements (CR-101)

## Context
ADR-0045 chỉ log response, không log prompt. Khi beat B7 ra tiếng Trung và viết lại vẫn ra tiếng Trung, log không cho biết model đã nhận prompt gì, nên không phân biệt được lỗi do prompt (nội dung cũ của beat bị đưa lại vào prompt) với lỗi do cửa sổ ngữ cảnh (prompt khoảng 5–6 nghìn token trên Ollama không đặt `num_ctx`, bị cắt đầu prompt trong im lặng).

Creator quyết định (2026-10-25): lưu prompt cho mọi lần gọi, xem được trên màn Nhật ký, `OLLAMA_NUM_CTX` mặc định 16384.

## Options Considered
### Option A: lưu `system_prompt` và `user_prompt` trong `llm_calls` cho mọi lần gọi
- Strengths: một nơi tra; thấy đúng cái model nhận, kể cả lần bị từ chối; không cần bật tắt.
- Trade-offs: DB lớn nhanh hơn (có nén lz4 và có nút xoá); prompt có thể chứa nội dung dự án (DB chỉ mở trên `127.0.0.1`).

### Option B: chỉ lưu prompt của lần gọi lỗi
- Trade-offs: không bắt được trường hợp reply sai nhưng `ok = true`, đúng trường hợp B7.

### Option C: bật/tắt bằng biến môi trường
- Trade-offs: lỗi xảy ra trước khi bật thì không tra được.

## Decision
A. Danh sách `GET /v1/llm/calls` không trả prompt; `GET /v1/llm/calls/{call_id}` trả đủ. Thay điểm "không lưu prompt" của ADR-0045; các điểm còn lại của ADR-0045 giữ nguyên.

Mọi cuộc gọi qua Ollama gửi `options.num_ctx`; prompt ước lượng vượt 85% cửa sổ bị cắt phần giữa (giữ đầu và đuôi) rồi chạy tiếp, có cảnh báo trong log của service. Creator chốt (2026-10-25): Ollama chỉ để test cục bộ, production dùng model lớn (DeepSeek) nên không cần báo lỗi.

## Rationale
Muốn biết vì sao một reply sai thì phải thấy prompt đi kèm. Cắt có chủ đích giữ được đầu (vai trò, ngôn ngữ) và đuôi (việc cần làm, định dạng) thay vì để Ollama cắt đầu prompt trong im lặng; bản đã cắt nằm trong log.

## Consequences
- **Positive**: tra được prompt của mọi lần gọi; prompt bị cắt thì log ghi đúng bản đã cắt; đo được độ dài prompt thật của từng bước.
- **Negative / Accepted Trade-offs**: DB lớn hơn; prompt nằm rõ trong DB; phần giữa prompt bị mất khi quá dài (chấp nhận với model test); ước lượng token theo ký tự có sai số (thận trọng: 2 ký tự một token).
- **Follow-ups**: rút gọn prompt beat nếu log cho thấy vẫn sát giới hạn; kiểm tra ngôn ngữ cho các bước khác.

## Related
- Thay một phần: ADR-0045.
- Design artifact: `aidlc-docs/inception/requirements/cr-101-prompt-trace-beat-language.md`.
