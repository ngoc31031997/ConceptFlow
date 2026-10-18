# ADR-0044: Chốt phiên bản kịch bản; bước sau chỉ đọc bản đã chốt và bộ bàn giao trong DB

## Status
Accepted

## Date
2026-10-17

## Stage
Application Design (CR-072, cập nhật Tiền kỳ · Nội dung)

## Context
Bước Rà soát và chốt (FR-32) kết thúc Tiền kỳ · Nội dung bằng một phiên bản kịch bản kèm ghi chú bàn giao cho các vai AI phía sau. Creator quyết định (2026-10-17): tài liệu là đơn vị trong DB, không phải file; "chốt" là đóng băng một phiên bản, sửa sau đó tạo phiên bản mới và đánh dấu phần phụ thuộc là "cần làm lại"; không có gì bị khoá cứng. Khung đơn vị (ADR-0035) cho mỗi đơn vị một trạng thái và dấu đầu vào, nhưng chưa nói các bước sau đọc nội dung đang sửa hay một bản cố định.

## Options Considered
### Option A: Ảnh chụp bất biến của mọi đơn vị nội dung khi chốt; bước sau chỉ đọc ảnh chụp
- What it is: chốt tạo `script_version` (số phiên bản, lúc chốt, ghi chú, nội dung đã chuẩn hoá của brief, dữ kiện đang dùng, câu chuyện, beat, lời thoại, ghi chú bàn giao, thuật ngữ, lý do chấp nhận mục rà soát), có hash cho từng phần. Đơn vị ở bước 8 trở đi ghi dấu đầu vào trỏ tới phần của phiên bản (ví dụ lời thoại beat 4 của v2), không trỏ tới đơn vị đang sửa.
- Strengths: Creator sửa thoải mái ở bước 1–7 mà không làm rung các bước sau; chốt bản mới chỉ làm phần đổi thành "cần làm lại" bằng phép so hash của ADR-0035; có lịch sử v1, v2…
- Trade-offs: lưu thêm một bản nội dung mỗi lần chốt; prompt các bước sau đọc từ phiên bản, không từ đơn vị.

### Option B: Bước sau đọc thẳng đơn vị đang sửa; cờ "đã chốt" trên từng đơn vị
- What it is: không có ảnh chụp; sửa một đơn vị đã chốt thì gỡ cờ.
- Strengths: ít dữ liệu.
- Trade-offs: mỗi lần sửa nhỏ ở bước 1–7 làm các bước sau thấy nội dung chưa duyệt; không có "các bước sau vẫn dùng v1 tới khi chốt v2".

### Option C: Xuất bộ bàn giao thành file trên `/shared`
- What it is: chốt ghi một file Markdown hoặc JSON; bước sau đọc file.
- Strengths: dễ xem bằng mắt.
- Trade-offs: trái quyết định "đơn vị trong DB" của Creator; chữ nhỏ cần tra cứu và tính hash thuộc về DB theo ADR-0041.

## Decision
A. Bản nháp là trạng thái đơn vị hiện tại so với phiên bản chốt gần nhất; danh sách thay đổi của bản nháp tính bằng so hash từng phần. Chốt vN+1 chỉ được khi đủ điều kiện chốt (FR-32), có xác nhận nêu phần sẽ làm lại. Bộ bàn giao là một phần của phiên bản; mỗi vai AI phía sau đọc đúng các mục FR-32 mục 4 chỉ định. Khi một bước phía sau chưa theo đơn vị (bước có sẵn trước khi CR con của nó vào main), chốt bản mới đánh cả bước đó "cần làm lại" thay vì từng phần.

## Rationale
Đóng băng bằng ảnh chụp cho đúng hành vi Creator chọn: sửa sau khi chốt không khoá ai, và bước sau không thấy nội dung chưa chốt. Dấu đầu vào trỏ vào phần của phiên bản làm cho việc chốt bản mới đi qua đúng cơ chế "cần làm lại" đã có, không cần luật riêng.

## Consequences
- **Positive**: lịch sử phiên bản; chỉ phần phụ thuộc chỗ đổi phải làm lại; prompt các bước sau có một nguồn đọc ổn định.
- **Negative / Accepted Trade-offs**: thêm bảng phiên bản ở authoring-service; prompt của Storyboard, Hình minh hoạ, Code đổi chỗ đọc kịch bản sang phiên bản đã chốt ngay ở U3f.
- **Follow-ups**: kiểm chứng lại sau khi chốt (backlog).

## Related
- Mở rộng: ADR-0035 (dấu đầu vào), ADR-0041 (chữ nhỏ nằm trong DB).
- Design artifact: `aidlc-docs/inception/application-design/cr-072/components.md`, `component-methods.md`, `services.md`.
- Yêu cầu: `aidlc-docs/inception/requirements/cr-072-2d-film-pipeline.md` FR-32, tiêu chí chấp nhận 14.
