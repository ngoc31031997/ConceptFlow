# ADR-0045: llm-service giữ DB riêng và là nơi duy nhất ghi và trả JSON của cuộc gọi AI

## Status
Proposed

## Date
2026-10-25

## Stage
Application Design (CR-100)

## Context
Khi AI trả về khối JSON hỏng ở một bước (ví dụ `invalid character '}' after top-level value` ở bước câu chuyện), Creator không xem lại được AI đã trả gì: authoring-service chỉ ghi số token (`llm_usage`), không ghi nội dung. Việc cắt khối JSON nằm trong domain của authoring-service (`jsonObjectIn`) và lặp lại ở nhiều bước. llm-service là nơi duy nhất nói chuyện với model nhưng không giữ dấu vết nào của cuộc gọi.

Creator quyết định (2026-10-25): llm-service có DB riêng, log mọi cuộc gọi (chỉ response, không prompt), trả JSON sạch; authoring-service không lưu gì về LLM; thống kê chunk đi thẳng từ gateway tới llm-service.

## Options Considered
### Option A: llm-service có DB riêng, ghi mọi cuộc gọi tại `Provider.chat`, trả JSON sạch; authoring-service bỏ `llm_usage`
- What it is: bảng `llm_calls` trong `llm-db` (cổng `127.0.0.1:5435`); `expect_json` cắt khối JSON đầu tiên và cho model sửa một lượt; endpoint stats và nhật ký đọc/xoá do gateway chuyển tới.
- Strengths: một điểm ghi duy nhất cho mọi đường gọi model (kể cả pipeline code); số liệu và nội dung ở cùng chỗ; authoring-service chỉ còn kiểm tra nghiệp vụ; Creator xem được trong DBeaver và trên màn hình Nhật ký.
- Trade-offs: thêm một container Postgres và phụ thuộc `asyncpg`; hợp đồng `/v1/chat` đổi; số liệu cũ của `llm_usage` bị bỏ; xoá log cũng xoá dữ liệu thống kê chunk.

### Option B: authoring-service ghi reply khi parse lỗi, giữ `llm_usage`
- What it is: log thêm vào bảng sẵn có của authoring-service, chỉ khi parse lỗi.
- Strengths: ít thay đổi nhất, không thêm container.
- Trade-offs: không thấy các cuộc gọi thành công mà kết quả kém; không phủ các endpoint llm-service tự gọi model; `jsonObjectIn` vẫn lặp ở nhiều bước; trái quyết định của Creator.

### Option C: llm-service chỉ ghi file log
- What it is: ghi `response` vào log của container.
- Strengths: không cần DB.
- Trade-offs: không lọc/xoá được trên giao diện, không có thống kê; trái quyết định của Creator.

## Decision
A.

## Rationale
`Provider.chat` là chỗ duy nhất mọi cuộc gọi model đi qua, nên ghi ở đó phủ được tất cả, và llm-service không cần biết schema từng bước để cắt JSON. Kiểm tra nghiệp vụ (dữ kiện, ô Creator đã điền) cần dữ liệu dự án nên ở lại authoring-service.

## Consequences
- **Positive**: tra được response của mọi lần gọi theo `call_id`; sửa lỗi `}` thừa một lần cho mọi bước; thống kê và nhật ký ở một nơi; chuẩn bị cho việc đọc lại response đã lưu.
- **Negative / Accepted Trade-offs**: thêm `llm-db`; mất lịch sử `llm_usage`; xoá dòng nhật ký làm mất số liệu thống kê tương ứng; nội dung response lưu rõ trong DB (chỉ mở trên `127.0.0.1`); ghi log lỗi không chặn cuộc gọi AI nên có thể mất dòng khi DB sập (có ghi vào log của service).
- **Follow-ups**: đọc lại response đã lưu để thử lại không tốn token; tác vụ nền xoá theo ngày nếu Creator cần.

## Related
- Mở rộng: cr-039 (hợp đồng `/v1/chat`).
- Design artifact: `aidlc-docs/inception/requirements/cr-100-llm-call-log.md`.
