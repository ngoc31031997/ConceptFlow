# ADR-0048: llm-service sở hữu thư viện prompt, danh mục model và model mặc định theo bước

## Status
Proposed

## Date
2026-11-10

## Stage
Application Design (CR-113)

## Context
Từ bước 8, `visual-service` cũng gọi AI và cần prompt. Prompt và model mặc định đang nằm ở authoring-service (bảng `prompts`, `app_settings`), danh mục model và việc kiểm model hợp lệ cũng ở đó, trong khi llm-service mới là nơi biết một id model đi về provider nào và đã giữ `llm_calls` kèm prompt đã gửi (ADR-0045, ADR-0046). Creator muốn một nơi duy nhất quản lý prompt và mọi thứ liên quan đến việc gọi AI. Đặt prompt ở orchestrator đã bị loại vì orchestrator chỉ điều phối flow và saga.

## Options Considered
### Option A: llm-service sở hữu bảng prompt, danh mục model và model mặc định; prompt hệ thống được từng service đẩy lên lúc khởi động
- Strengths: một nơi quản lý và một trang Cài đặt cho mọi service; danh mục model nằm cạnh code định tuyến provider nên không có hai bản; thêm service mới chỉ cần đẩy bản hệ thống của mình; seed vẫn nằm cạnh code điền biến.
- Trade-offs: authoring phụ thuộc llm-service để có prompt (không có thì bước AI báo lỗi); thêm đường HTTP nội bộ; phải chuyển dữ liệu một lần.

### Option B: giữ ở authoring-service, service khác gọi sang
- Strengths: không phải chuyển dữ liệu.
- Trade-offs: mọi service AI phụ thuộc authoring; authoring giữ kiến thức về model mà nó không dùng để định tuyến.

### Option C: để ở orchestrator
- Trade-offs: bị loại, orchestrator chỉ điều phối.

## Decision
A. Chi tiết hợp đồng: [`prompt-library-v1.md`](../../docs/contracts/prompt-library-v1.md).

- Cột `owner` cho biết bản hệ thống thuộc service nào; một lần đẩy chỉ ghi và xoá bản hệ thống của `owner` đó.
- Ghép prompt (điền biến) ở lại từng service; `GET /v1/projects/{id}/prompts/{role}` ở lại authoring.
- Khi llm-service không trả lời, bước AI báo lỗi rõ; không rơi về prompt cứng trong code.
- Authoring bỏ bảng `prompts` và các key `default_model.*` của `app_settings` ngay trong CR này, sau khi script chuyển dữ liệu đã chép và kiểm.
- ADR-0045 không đổi: nó nói về nhật ký cuộc gọi, ADR này mở rộng phạm vi sở hữu của llm-service sang thư viện prompt.

## Rationale
llm-service là cổng AI chung, mọi service đã đi qua `/v1/chat`. Gom prompt, danh mục model và model mặc định vào đó đặt mọi thứ về việc gọi AI ở một chỗ, còn dữ liệu của từng dự án vẫn do service sở hữu nó điền vào template.

## Consequences
- **Positive**: visual-service (CR-112) dùng sẵn API này; không còn hai danh sách model; trang Cài đặt không đổi vì api-gateway giữ nguyên đường dẫn.
- **Negative**: llm-service ngừng chạy thì authoring không chạy được bước AI nào và không kiểm được model của video; phải chạy script chuyển dữ liệu đúng thứ tự (`copy`, `verify`, `drop-source`).
- **Neutral**: các bản của Creator thuộc role đã bỏ (xoá lúc khởi động bởi cơ chế cũ) không còn ai dọn; chúng đã được dọn ở các lần khởi động trước.
