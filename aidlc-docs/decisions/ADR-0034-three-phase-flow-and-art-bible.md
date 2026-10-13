# ADR-0034: Luồng ba giai đoạn và bộ thiết kế mỹ thuật hai lượt

## Status
Accepted

## Date
2026-10-13

## Stage
Application Design (CR-072)

## Context
Luồng 13 bước đi thẳng từ kịch bản vào hình minh hoạ và code. Creator chỉ thấy nhân vật, màu và nhịp phim sau khi render. CR-072 muốn luồng đi theo quy trình làm phim hoạt hình 2D, và chốt cái nhìn của video trước khi tốn thời gian sản xuất (câu 1, 15, 21; C5; AD-9).

## Options Considered
### Option A: Giữ 13 bước, thêm khâu mới vào màn có sẵn
- What it is: nhân vật và color script nằm trong màn Visual, animatic nằm trong Review.
- Strengths: ít đổi orchestrator và web-gui.
- Trade-offs: màn Visual và Review quá tải; không còn thấy được tiền kỳ, sản xuất, hậu kỳ.

### Option B: 17 bước chia ba giai đoạn, có bộ mỹ thuật duyệt hai lượt
- What it is: thêm Mỹ thuật lượt 1 (thế giới, bối cảnh, nhân vật, đạo cụ) sau Kịch bản; Mỹ thuật lượt 2 (color key, khung mẫu) sau Storyboard; thêm Giọng sơ bộ, Animatic, Dựng.
- Strengths: đúng thứ tự xưởng phim; mỗi chỗ chốt có màn riêng.
- Trade-offs: đổi số bước ở ba nơi; thêm chỗ duyệt.

### Option C (định nghĩa luồng): một nguồn duy nhất qua API thay cho ba bản
- What it is: orchestrator trả định nghĩa luồng, authoring-service và web-gui đọc.
- Strengths: không thể lệch.
- Trade-offs: web-gui phải nạp trước khi vẽ thanh bước; authoring-service phụ thuộc thêm orchestrator.

## Decision
B, với định nghĩa luồng giữ ba bản (orchestrator, authoring-service, web-gui) cùng test hợp đồng như CR-051 (AD-9 A). Bộ mỹ thuật duyệt hai lượt: mục 1–4 sau kịch bản, mục 5–6 sau storyboard.

## Rationale
Creator chọn tổ chức lại theo ba giai đoạn (câu 1 D, 15 B) và muốn bộ mỹ thuật chi tiết như xưởng phim (câu 21) nhưng ít lần chờ (câu 3). Hai lượt là số lần duyệt nhỏ nhất mà vẫn chốt thế giới trước storyboard, và chốt màu cùng khung mẫu sau khi biết khoảnh khắc nào quan trọng. Ba bản định nghĩa luồng đang chạy ổn và có test bắt lệch; một nguồn qua API thêm phụ thuộc mà không giải quyết vấn đề nào đang có.

## Consequences
- **Positive**: Creator thấy và chốt thế giới, nhân vật, màu, nhịp phim trước khi AI viết code và render.
- **Negative / Accepted Trade-offs**: luồng dài hơn (17 bước); đổi luồng vẫn phải sửa ba chỗ.
