# CR-101 · Lưu prompt vào nhật ký AI, sửa beat ra tiếng Trung — yêu cầu

Nhánh `feature/cr-101-prompt-trace-beat-language`. Trạng thái: yêu cầu, chờ Creator duyệt. Thay một phần quyết định của CR-100 / ADR-0045 ("không lưu prompt"); sẽ có ADR mới thay thế điểm đó.

## Bối cảnh (2026-10-25)

Beat B7 của project `055f99e2…` (hố đen) ra tiếng Trung; Creator yêu cầu viết lại bằng tiếng Việt vẫn ra tiếng Trung. Tra `llm_calls`:

- Mọi cuộc gọi `beat` đi `ollama / qwen2.5:7b` (`app_settings.default_model = ollama`, project không chọn model riêng).
- Lần viết (07:52:25) và lần viết lại (07:53:10) có `goal` tiếng Trung giống từng chữ.
- Dữ liệu đầu vào (story, 26 dữ kiện, brief) đều là tiếng Việt; prompt beat có `NGÔN NGỮ: tiếng Việt`.

Nguyên nhân tìm được (chưa đo được vì log không có prompt):

1. **Đã xác nhận trong code**: khi viết lại / viết nội dung một beat, `currentBeatsText` đưa nội dung cũ của chính beat đó vào prompt, nên model chép lại bản tiếng Trung.
2. **Nghi vấn mạnh**: llm-service không đặt `num_ctx` cho Ollama. Prompt beat của project này khoảng 13.000 ký tự tiếng Việt (ước 5–6k token), vượt cửa sổ mặc định; Ollama cắt đầu prompt (vai trò, dòng ngôn ngữ, brief) mà không báo lỗi.
3. Không có bước nào kiểm tra ngôn ngữ của câu trả lời; reply tiếng Trung vẫn được tính `ok = true` và lưu.

## Chốt của Creator (2026-10-25)

1. Lưu prompt cho **mọi** lần gọi AI.
2. Màn **Nhật ký** trên web-gui xem được prompt của từng lần gọi.
3. Gộp ba lỗi beat vào cùng CR này.
4. Số CR là CR-101.
5. `OLLAMA_NUM_CTX` mặc định 16384.

## Phạm vi

### A. llm-service: lưu prompt

- `llm_calls` thêm cột `system_prompt`, `user_prompt` (text, nén lz4 như `response`) và `prompt_chars` (int).
- Ghi ở cùng chỗ đang ghi `response` (`Provider.chat`), cho mọi đường vào: `/v1/chat`, `/v1/suggest-metadata`, `/v1/storyboard/finalize`, `/v2/code/generate` (kể cả lượt sửa JSON, lượt thử lại, lượt lỗi).
- Với Ollama ghi thêm `num_ctx` đã dùng.
- Quy tắc ghi giữ như CR-100: ghi log lỗi không làm hỏng cuộc gọi AI.
- Xoá log (chọn dòng, xoá theo 30 ngày) xoá luôn prompt; không thêm cơ chế xoá mới.
- API liệt kê log không trả prompt (để danh sách nhẹ); thêm một endpoint đọc chi tiết một `call_id` trả cả prompt và response.

### B. llm-service: cửa sổ ngữ cảnh Ollama

- Mọi cuộc gọi chat qua Ollama gửi `num_ctx` lấy từ biến môi trường mới `OLLAMA_NUM_CTX`, mặc định 16384.
- Trước khi gọi, ước lượng số token của prompt; nếu vượt 85% `num_ctx` thì cắt phần giữa (giữ đầu và đuôi) rồi chạy tiếp, không báo lỗi (Creator chốt 2026-10-25: Ollama chỉ để test, production dùng DeepSeek). Cảnh báo ghi ra log của service; `llm_calls` lưu bản đã cắt.

### C. authoring-service: bước Beat sheet

- Khi viết lại một beat hoặc viết nội dung một beat, prompt không chứa nội dung cũ (`title`, `goal`, `content`, `next`) của chính beat đó; các beat khác giữ như hiện nay. Phần Creator đã tự viết vẫn được giữ nguyên văn như luật hiện tại.
- Kiểm tra ngôn ngữ reply: nội dung là tiếng Việt hoặc tiếng Anh mà reply có chữ Hán / Kana / Hangul trong `title`, `goal`, `content`, `next` thì coi là reply hỏng (`ErrBeatReplyUnusable`), đi vào vòng viết lại sẵn có kèm lý do "viết sai ngôn ngữ"; hết lượt thì báo lỗi cho Creator, không lưu.

### D. api-gateway + web-gui: màn Nhật ký

- Gateway chuyển tiếp endpoint chi tiết mới.
- Màn Nhật ký: bấm một dòng để mở chi tiết, hiện system prompt, user prompt và response cạnh nhau, kèm `prompt_chars`, `num_ctx`, token; có nút copy từng phần. Theo `docs/ux-ui-design-rules.md`.

## Ngoài phạm vi

- Kiểm tra ngôn ngữ cho các bước khác ngoài Beat sheet (có thể làm sau, khi đã có log prompt để biết bước nào bị).
- Rút gọn prompt beat (ví dụ bỏ ghi chú của dữ kiện): chờ đo bằng log prompt rồi mới quyết.
- Đổi model mặc định trong cài đặt: việc của Creator.
- Đọc lại response đã lưu để thử lại không tốn token (vẫn chưa làm như CR-100).

## Kiểm thử

- Unit/smoke giả lập reply AI, không gọi Hive hay Ollama thật.
- Một lần đo thật trên Ollama (prompt beat của project hố đen, trước và sau khi sửa) chỉ làm khi Creator cho phép.
- Dọn dữ liệu kiểm thử (dòng `llm_calls`, beat thử) trong cùng lượt.

## Quyết định khi làm

- Ước lượng token theo ký tự (2 ký tự một token, thận trọng cho tiếng Việt có dấu); cắt giữa khi vượt 85% cửa sổ. Không gọi tokenizer của Ollama (đỡ một request mỗi lần gọi).
- Chưa kiểm chứng: Ollama bản cài trên máy có nhận `options.num_ctx` qua đường `/v1` tương thích OpenAI hay không (cần một lần gọi thật, chờ Creator cho phép).
