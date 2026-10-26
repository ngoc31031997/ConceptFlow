# Chạy bước AI bằng AI ngoài: `external-ai.v1`

Quyết định: [ADR-0047](../../aidlc-docs/decisions/ADR-0047-external-ai-through-the-chat-gate.md). Mã: `services/authoring-service/internal/application/external_ai.go`, `internal/adapters/http/external_ai.go`, `services/web-gui/src/components/ExternalAiDialog.tsx`.

Áp dụng cho mọi route của authoring-service gọi AI đúng một lượt, không đổi đường dẫn hay thân yêu cầu của route đó.

| Header `X-External-AI` | Thân yêu cầu | Trả về |
|---|---|---|
| (không có) | như cũ | như cũ: gọi model |
| `prompt` | như cũ | `200 {"external_prompt": "<system>\n\n<user>"}`; không gọi model, không ghi gì. Lỗi xảy ra trước khi dựng được prompt (thiếu dự án, bước chưa mở) đi ra nguyên vẹn |
| `paste` | như cũ, thêm `external_reply` (chuỗi, bắt buộc, tối đa 4 MB) | như route cũ khi chạy thật: bước đọc `external_reply` thay câu trả lời của model, kiểm tra và lưu |
| giá trị khác | | `400` |

Lỗi của `paste`:
- `400` khi `external_reply` rỗng.
- `422`/`502` theo cách route cũ báo câu trả lời không dùng được (reply không có JSON, thiếu ô, dữ kiện lạ). Không có vòng hỏi lại: Creator sửa hoặc dán lại.
- `400` `AI ngoài chỉ chạy được một prompt mỗi lượt` khi route cần hơn một lượt hỏi (viết tất cả beat, bàn giao tất cả beat). Gọi lại theo từng beat.

Route có nút AI ngoài ở web-gui: `idea/generate`, `idea/review`, `creative-brief/generate`, `creative-brief/review`, `research/questions/generate`, `story/generate`, `beats` (thêm beat bằng AI), `beats/{key}/redo`, `units/{kind}/{key}/redo` (kịch bản và bàn giao), `script/generate` (một beat), `script/suggest`, `script-lock/review`, `script-lock/handoff/generate` (một beat).

CORS của api-gateway cho phép header `X-External-AI`.
