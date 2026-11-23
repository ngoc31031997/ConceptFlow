# ADR-0052: Lượt chạy AI có xem trước, một nút ba kênh (mọi bước)

**Trạng thái:** Proposed · **Ngày:** 2026-11-23

## Bối cảnh

Mỗi lượt AI của bước 11 có ba cách chạy: model của hệ thống (API), extension AI Bridge, và Creator tự chép prompt rồi dán câu trả lời. Ba cách này nằm thành ba nút ngang nhau trong mỗi hộp thoại. Chạy bằng API hay extension thì kết quả được lưu ngay, Creator không xem được trước khi ghi đè. Creator muốn có một kênh mặc định đặt trong Cài đặt, một nút chạy, và được xem kết quả của mọi kênh trước khi lưu.

## Quyết định

1. llm-service giữ kênh mặc định (`api`, `extension`, `manual`) cạnh model mặc định: một giá trị cho mọi bước và giá trị riêng cho từng bước (bảng `ai_channel_defaults`, route `/v1/settings/ai-channel…`).
2. Service sở hữu bước có **lượt chạy AI** (bảng `ai_runs`; authoring-service cho bước 1–7, visual-service cho bước 8–11; hợp đồng `docs/contracts/ai-runs-v1.md`). Lượt chạy giữ chính request AI của bước (method, path, body, dựng ở web-gui bằng chế độ `describe` đã có) và phát lại nó bên trong service qua router, nên không có route hay bảng việc riêng cho từng bước. Khi bắt đầu, nó dựng prompt và lưu lượt. Câu trả lời đến từ model (chạy nền trong service), từ extension (mục `kind: "prompt"` gửi `{raw}` về `…/ai-runs/{id}/reply`) hoặc từ ô dán; cả ba đi qua cùng route đó.
3. Xem trước dùng lại đúng luồng ghi: hai chế độ mới của `ExternalAIRun`, `preview` (câu trả lời được dán) và `capture` (gọi model rồi giữ câu trả lời), chạy luồng như thường nhưng dừng ở `stopForPreview`, ngay sau bước đọc và kiểm tra, trước mọi lần ghi. Áp dụng chạy lại luồng đó ở chế độ `paste` với câu trả lời đã giữ. Không có bộ kiểm tra thứ hai.
4. Chạy hàng loạt (mọi unit, mọi tài sản chưa có chi tiết) vẫn lưu ngay từng mục, vì mỗi mục đọc kết quả **đã lưu** của mục liền trước.

## Hệ quả

- Không phải sửa extension: loại mục `kind: "prompt"` đã có.
- Câu trả lời dán tay hoặc từ extension được gửi qua `/v1/chat/external` hai lần (một lần khi xem trước, một lần khi áp dụng), nên nhật ký gọi AI có hai dòng cho lượt đó.
- Nếu bước đã đổi giữa lúc xem trước và lúc áp dụng (ví dụ Creator sửa một shot) mà câu trả lời không còn khớp, áp dụng bị từ chối, lượt chuyển sang `unreadable` kèm lý do, và không có gì được ghi.
- Lượt đang gọi model khi visual-service khởi động lại sẽ bị đánh dấu `failed` lúc khởi động.
- Mọi bước, kể cả các bước làm sau, dùng `AiRunDialog` và gọi `stopForPreview` trong luồng của mình; luồng chưa có điểm dừng thì lượt báo lỗi thay vì lưu không qua xem trước.
- Kênh mặc định có danh sách bước riêng theo màn hình (`idea` … `visual_storyboard`), khác khoá model mặc định.
