# CR-102 · Chạy bước AI bằng AI ngoài (copy prompt, dán kết quả)

Ngày: 2026-10-26 · Nhánh: `feature/cr-102-external-ai-paste` · Trạng thái: chờ Creator duyệt yêu cầu và thiết kế

## Yêu cầu của Creator
Ở bước 1 đến bước 7, mọi chỗ gọi AI có thêm một nút. Bấm nút mở popup gồm hai phần:
1. Prompt sắp gửi cho AI, có nút copy để dán sang AI bên ngoài.
2. Ô dán kết quả AI ngoài trả về.

Bấm "Đọc kết quả" thì hệ thống chạy tiếp y như khi gọi AI, chỉ khác là câu trả lời lấy từ ô dán.

## Quyết định đã chốt (2026-10-26)
| # | Câu hỏi | Trả lời |
|---|---|---|
| Q1 | Nút gọi AI nhiều lần trong một lượt bấm | Mỗi popup đúng **một prompt**, một kết quả. Nút "chạy tất cả beat" chỉ gọi AI; AI ngoài làm theo từng beat. Không tự hỏi lại: dán sai định dạng thì báo lỗi trong popup. |
| Q2 | Ghi nhật ký AI | Có: mỗi lượt dán là một dòng `llm_calls` với `provider = "external"`, giữ prompt và kết quả dán, token để trống. |
| Q3 | Dạng prompt | Một khối ghép sẵn: system prompt, dòng trống, user prompt. |

## Yêu cầu chức năng
- **FR-1** Nút "AI ngoài" nằm cạnh từng nút gọi AI một lần ở các bước:
  | Bước | Chỗ gọi AI (use case) | Role |
  |---|---|---|
  | 1 Ý tưởng | Tạo / làm lại, Rà soát (`project_idea.go`) | `project_idea`, `project_idea_review` |
  | 2 Creative brief | Tạo / làm lại, Rà soát (`project_brief.go`) | `project_brief`, `project_brief_review` |
  | 3 Nghiên cứu | Tạo câu hỏi (`research_questions.go`) | `research_questions` |
  | 4 Câu chuyện | Tạo / làm lại (`story_desk.go`) | `story_frame` |
  | 5 Beat sheet | Tạo / làm lại từng beat (`beat_desk.go`) | `beat_sheet` |
  | 6 Viết kịch bản | Viết từng beat, gợi ý một câu (`script_writer.go`) | `script_writer` |
  | 7 Rà soát và chốt | Rà soát kịch bản, bàn giao từng beat (`script_lock_ai.go`) | các role của bước 7 |
  Danh sách chính xác của bước 5–7 (nút nào gọi đúng một lần) được kiểm lại khi code; nút gọi nhiều lần không có nút AI ngoài.
- **FR-2** Mở popup thì hệ thống ghép prompt **đúng như lúc gọi AI thật** (cùng prompt trong DB, cùng dữ liệu bước trước, cùng ghi chú Creator gõ) và hiện trong ô chỉ đọc kèm nút Copy. Mở popup không ghi gì vào DB của bước.
- **FR-3** Ô thứ hai cho dán kết quả. Nút "Đọc kết quả" gửi đoạn dán; hệ thống chạy đúng luồng của bước (đọc JSON, kiểm tra, lưu, cập nhật màn hình) như khi AI trả về đoạn đó.
- **FR-4** Bước cần JSON: lấy object JSON đầu tiên trong đoạn dán (chấp nhận có chữ hoặc khung ```json bao quanh). Không đọc được hoặc không qua kiểm tra của bước thì popup vẫn mở, giữ nguyên đoạn dán và hiện lỗi cụ thể.
- **FR-5** Mỗi lượt "Đọc kết quả" ghi một dòng `llm_calls`: `provider = "external"`, `model = "external"`, prompt, reply là đoạn dán, cùng `project_id/role/step/phase` như cuộc gọi thật. Màn hình nhật ký AI hiện nhãn "AI ngoài".
- **FR-6** Không cần API key: nút AI ngoài dùng được cả khi chưa cấu hình nhà cung cấp AI.

## Thiết kế
### Ý chính: chặn ở cổng `Chat`, không sửa logic từng bước
Mọi chỗ gọi AI ở bước 1–7 đi qua `LLMProviderPort.Chat` (`internal/application/llm_provider.go`). Thêm một decorator `externalAIProvider` bọc client llm-service, đọc chế độ từ `context`:
- Không có chế độ: gọi llm-service như hiện nay.
- `prompt`: không gọi llm-service; trả lỗi `*ExternalPromptError{Prompt}` chứa khối prompt đã ghép. Handler HTTP đổi lỗi này thành `200 {"external_prompt": "..."}`.
- `paste`: không gọi AI; gọi `POST /v1/chat/external` của llm-service với cùng trường như `/v1/chat` cộng `reply`. llm-service đọc JSON bằng đúng hàm của `/v1/chat` (bỏ lượt sửa), ghi `llm_calls`, trả `ChatResult` cùng dạng. Use case chạy tiếp như bình thường.

Lý do: một chỗ sửa thay cho 11 chỗ, prompt chắc chắn giống hệt bản chạy thật, đọc JSON không bị viết lại lần hai bằng Go.

### Đường đi của chế độ
- web-gui gửi lại **đúng request** của nút đó, thêm header `X-External-AI: prompt` hoặc `X-External-AI: paste`; ở chế độ `paste` thân request có thêm trường `external_reply`.
- api-gateway chuyển header đó sang authoring-service.
- Middleware authoring-service đặt chế độ và đoạn dán vào `context`.
- Ở chế độ `paste`, bỏ qua tự hỏi lại (`askUntilUsable` chỉ chạy một lượt) và các lượt sửa định dạng.

### Ghi DB trước khi gọi AI
Use case nào ghi trạng thái (ví dụ "đang chạy") trước khi gọi `Chat` thì ở chế độ `prompt` sẽ để lại dấu vết. Khi code, từng chỗ trong 11 chỗ được kiểm; chỗ nào có ghi trước thì chuyển việc ghi xuống sau lời gọi `Chat` hoặc bỏ qua ở chế độ `prompt`. Chỗ nào chạy nền (bước 6, 7) thì ở chế độ `paste` chạy đồng bộ trong request để popup nhận được lỗi.

### Hợp đồng
- Thêm `POST /v1/chat/external` vào `docs/contracts/llm-chat-v1.md` (chỉ thêm, không đổi route cũ).
- Ghi header `X-External-AI` và trường `external_reply` vào hợp đồng của từng bước (idea, creative-brief, research, story, beat-sheet, script, script-lock).
- ADR-0047: AI ngoài đi qua cổng `Chat` và llm-service vẫn giữ nhật ký.

### Giao diện (theo `docs/ux-ui-design-rules.md`)
- Component dùng chung `ExternalAIDialog`: ô prompt chỉ đọc + Copy, ô dán, nút "Đọc kết quả", vùng lỗi. Đặt cạnh từng nút AI qua một component nút nhỏ dùng chung.
- Màn hình nhật ký AI: nhãn "AI ngoài" cho dòng `provider = external`.

### Service bị đổi
authoring-service, llm-service, api-gateway (nếu chưa chuyển tiếp header), web-gui.

## Kiểm thử
- Unit Go: decorator ở ba chế độ; mỗi use case chế độ `prompt` trả prompt giống prompt gửi ở chế độ thường và không ghi DB; chế độ `paste` lưu kết quả như reply AI giả.
- Unit Python: `/v1/chat/external` đọc JSON, ghi `llm_calls` với `provider = external`, báo `malformed` khi không đọc được.
- Unit web-gui: dialog copy, dán, hiện lỗi.
- Kiểm tay trên stack: một bước mỗi loại, rồi xoá dữ liệu test.
- Không gọi AI thật.
