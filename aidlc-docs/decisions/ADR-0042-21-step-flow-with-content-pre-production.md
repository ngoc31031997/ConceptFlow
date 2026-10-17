# ADR-0042: Luồng 21 bước, Tiền kỳ chia Nội dung và Hình ảnh

## Status
Accepted

## Date
2026-10-17

## Stage
Application Design (CR-072, cập nhật Tiền kỳ · Nội dung)

## Context
ADR-0034 chốt luồng 17 bước chia ba giai đoạn, trong đó Tiền kỳ tới kịch bản chỉ có Khởi tạo, Cấu hình và một bước Kịch bản (logline, tóm tắt, dàn ý, lời thoại). Ngày 2026-10-17 Creator thiết kế lại phần này thành sáu bước quanh Cấu hình: Ý tưởng · creative brief, Nghiên cứu và kiểm chứng (tuỳ chọn), Xây dựng câu chuyện, Beat sheet, Viết kịch bản, Rà soát và chốt, và chốt giao diện trên mockup bản 8. Mockup gom bảy bước này thành nhóm "Tiền kỳ · Nội dung" và năm bước từ Mỹ thuật lượt 1 tới Animatic thành "Tiền kỳ · Hình ảnh". Luồng thành 21 bước.

## Options Considered
### Option A: Gộp sáu việc mới vào bước Kịch bản hiện có, giữ 17 bước
- What it is: Kịch bản thành một màn có tab brief, nghiên cứu, câu chuyện, beat, lời thoại, rà soát.
- Strengths: không đổi số bước; ít đổi ba bản định nghĩa luồng.
- Trade-offs: một bước có sáu chỗ duyệt; thanh bước không cho thấy luồng đang dừng ở đâu; trái giao diện Creator đã chốt.

### Option B: Mỗi việc một bước, 21 bước, ba giai đoạn với Tiền kỳ chia hai nhóm
- What it is: bảy bước Tiền kỳ · Nội dung (1–7), năm bước Tiền kỳ · Hình ảnh (8–12), Sản xuất (13–16), Hậu kỳ (17–21). Nghiên cứu là bước tuỳ chọn, bỏ qua được mà không chặn bước nào.
- Strengths: đúng mockup đã chốt; mỗi chỗ duyệt có một bước riêng trên thanh bước; mỗi CR con thêm đúng một bước.
- Trade-offs: luồng dài hơn; ba bản định nghĩa luồng đổi thêm năm lần trong CR-072; cần khái niệm bước tuỳ chọn ở cả ba bản.

### Option C: Bốn giai đoạn ngang hàng (Nội dung, Hình ảnh, Sản xuất, Hậu kỳ)
- What it is: tách Tiền kỳ thành hai giai đoạn riêng.
- Strengths: đơn giản hoá thanh bước.
- Trade-offs: bỏ cách gọi ba giai đoạn của quy trình làm phim mà Creator chọn (câu 15 B).

## Decision
B. Giai đoạn vẫn là ba; Tiền kỳ có hai nhóm, thanh bước hiện bốn nhóm với tên "Tiền kỳ · Nội dung", "Tiền kỳ · Hình ảnh", "Sản xuất", "Hậu kỳ". Ba bản định nghĩa luồng và test hợp đồng giữ như ADR-0034 (AD-9 A). Quyết định bộ mỹ thuật duyệt hai lượt của ADR-0034 giữ nguyên: lượt 1 sau bước Rà soát và chốt, lượt 2 sau Storyboard.

## Rationale
Creator chốt giao diện bảy bước trước khi làm (UI-first) và muốn thấy từng chỗ dừng. Một bước một chỗ duyệt khớp khung sửa theo đơn vị (ADR-0035): bước đi tiếp khi mọi đơn vị của nó đã duyệt. Giữ ba giai đoạn để tên trên thanh bước vẫn là ngôn ngữ làm phim; hai nhóm của Tiền kỳ cho thấy chỗ chuyển từ chữ sang hình.

## Consequences
- **Positive**: Creator thấy rõ nội dung đã chốt trước khi bước nào vẽ hình; bước sau đọc bản kịch bản đã chốt (ADR-0044).
- **Negative / Accepted Trade-offs**: 21 bước; ba bản định nghĩa luồng đổi ở U2a, từng CR con U3a..U3f, U4a, U5, U6, U8a, U8b; bước tuỳ chọn cần thêm trạng thái "bỏ qua".
- **Follow-ups**: gom bớt chỗ duyệt của Tiền kỳ · Nội dung sau khi đo tỉ lệ duyệt ngay lần đầu (backlog).

## Related
- Supersedes: ADR-0034 (phần luồng 17 bước; phần bộ mỹ thuật hai lượt giữ nguyên ở đây).
- Design artifact: `aidlc-docs/inception/application-design/cr-072/application-design.md`, `unit-of-work.md`.
- Yêu cầu: `aidlc-docs/inception/requirements/cr-072-2d-film-pipeline.md` FR-1, FR-27..FR-32.
