# CR-051 — Câu hỏi làm rõ yêu cầu

Xem phát hiện ở `cr-051-pipeline-progress-ux-audit.md`. Điền câu trả lời sau mỗi thẻ `[Answer]:`.

## Question 1
Nút "Chạy bằng AI" ở bước 3 hiện chạy liền các bước 3→4→(5)→6. Khi chuỗi đang chạy, tiến độ nên hiện thế nào?

A) Giữ chuỗi, sửa cách hiển thị: stepper và dòng trạng thái dùng đúng số và tên của luồng ("Bước 3 — Kịch bản ✓, Bước 4 — Visual (đang chạy), Bước 5 — Hình minh hoạ, Bước 6 — Code"). Nút ghi "Chạy bằng AI các bước 3–6".
   - ✅ Ưu: vẫn bấm một lần; con số khớp với thanh bước bên trái; sửa ít.
   - ⚠️ Nhược: bảng tiến độ vẫn hiện giống nhau trên cả 4 màn 3–6 (nhưng giờ đã có số đúng nên không còn lẫn).

B) 💡 Gợi ý: làm như A, và thêm trạng thái "đang chạy" ngay trên thanh bước bên trái (bước đang chạy có spinner, bước xong có dấu ✓). Trên mỗi màn, bảng tiến độ chỉ còn một dòng ngắn nói bước nào đang chạy.
   - ✅ Ưu: chỉ còn một nơi hiện tiến độ, là thanh bước vốn đã có 14 bước; hết cảm giác có một danh sách bước thứ hai.
   - ⚠️ Nhược: phải sửa thêm StepRail và tách trạng thái chạy của chuỗi (phía client) khỏi `flow_step` của server; nhiều việc hơn A.

C) Bỏ chạy chuỗi: mỗi màn 3/4/5/6 chỉ có nút chạy riêng bước của màn đó.
   - ✅ Ưu: đơn giản nhất, một màn ứng với một bước.
   - ⚠️ Nhược: mất tính năng một lần bấm (CR-030); phải bấm 3–4 lần và chờ từng bước.

X) Other (please describe after [Answer]: tag below)

[Answer]: A — Creator trả lời "1 ok"; AI hiểu là chọn phương án A (đã báo lại Creator để sửa nếu hiểu sai).

## Question 2
Với dự án Manim (không dùng Hình minh hoạ), bảng tiến độ của chuỗi nên hiện bước 5 thế nào?

A) Vẫn hiện bước 5, làm mờ, ghi "Không dùng", giống thanh bước. Như vậy luôn thấy đủ 3–6.
   - ✅ Ưu: khớp hoàn toàn với thanh bước; không còn tình trạng "4 bước mà loading chỉ hiện 3".
   - ⚠️ Nhược: có một dòng không làm gì.

B) Ẩn bước 5 nhưng vẫn giữ số thật (3, 4, 6) để không đánh số lại.
   - ✅ Ưu: gọn.
   - ⚠️ Nhược: số nhảy từ 4 sang 6, dễ bị hỏi "bước 5 đâu?".

X) Other (please describe after [Answer]: tag below)

[Answer]: A — "hiện nhưng làm mờ ghi ko dùng".

## Question 3
Phạm vi của CR này?

A) Chỉ sửa chuỗi AI bước 3–6 (F1) và cách đánh số trong tracker màn Validate (F2).

B) A, cộng thêm: thống nhất tiêu đề màn theo "Bước N — <tên trên thanh bước>" cho cả 14 bước (F3), xoá code "wizard 7 bước" không còn dùng và sửa comment cũ (F4).

C) B, cộng thêm phía server: gom định nghĩa 14 bước đang bị chép ở authoring-service và orchestrator về một chỗ dùng chung, hoặc thêm test để hai bản không lệch nhau (F6).
   - ⚠️ Nhược: đụng hai service Go, phải rebuild cả hai; hiện tại không có lỗi nào do hai bản này gây ra.

X) Other (please describe after [Answer]: tag below)

[Answer]: C.

## Question 4
Hai việc bên trong bước 7 ("Phân tích kịch bản", "Chạy thử & kiểm tra") nên đánh số thế nào?

A) Số con "7.1", "7.2".

B) Không đánh số, chỉ dùng chấm tròn hoặc dấu ✓. Tiêu đề màn đã ghi "Bước 7".

C) Cả hai dòng đều ghi số 7 (giống cách màn Render dùng số của luồng).

X) Other (please describe after [Answer]: tag below)

[Answer]: A — "đánh 7.1 7.2".

## Question 5 (vòng 2 — phát hiện F8)
Server không biết tới bước 5: dự án Remotion nháp đã xong Visual bị xếp thẳng vào bước 6 (Code), và thanh bước đánh dấu bước 5 là ✓ dù hình minh hoạ chưa duyệt xong. Có sửa trong CR này không?

A) Sửa trong CR này: authoring-service gửi thêm cờ "hình minh hoạ đã sẵn sàng" (mọi hình đã duyệt hoặc bỏ qua) trong phần tóm tắt nội bộ gửi sang orchestrator. `FlowStateFor` dùng cờ này cùng với render engine để xếp dự án Remotion vào bước 5 khi cờ chưa bật.
   - ✅ Ưu: thanh bước, danh sách dự án và nút mở lại dự án đều nói đúng; đúng tinh thần "rà soát toàn bộ quy trình server".
   - ⚠️ Nhược: đổi hợp đồng HTTP nội bộ giữa hai service (chỉ thêm trường, bản cũ vẫn đọc được); phải rebuild authoring-service và orchestrator; thêm một truy vấn lấy trạng thái hình mỗi lần tính bước cho dự án nháp.

B) Tách thành CR riêng. CR này chỉ ghi nhận F8 vào backlog.
   - ✅ Ưu: CR-051 gọn, chỉ sửa hiển thị và nhãn.
   - ⚠️ Nhược: lỗi "bước 5 đã ✓ dù chưa xong" vẫn còn cho tới CR sau.

X) Other (please describe after [Answer]: tag below)

[Answer]: A — "đúng như vậy. cần tách nó thành làm step riêng thật sự".
