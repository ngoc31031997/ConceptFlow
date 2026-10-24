# CR-093 · Bước 1 Idea — câu hỏi làm rõ

Đã ghi nhận từ câu trả lời trước: ô nhập là một textbox lớn; AI trả `topic`, `central_question`, `curiosity_points[]`, `scope`; AI viết nháp, Creator sửa, làm lại, duyệt; AI review độ rõ, khả năng nghiên cứu, phạm vi (cảnh báo, không chặn); điều kiện hoàn thành gồm máy kiểm hợp lệ và Creator duyệt; lưu ở bảng riêng `project_idea`; có ô chọn ngôn ngữ; không đụng brief, dạng video, thời lượng, nền tảng, project cũ.

Còn 5 điểm cần làm rõ.

## Question 1
Câu 1 nói "AI nhận input và trả về các trường", còn `idea.v1` chỉ có 4 trường. Văn bản Creator gõ vào textbox có được lưu không?

A) Có. Lưu thêm cột `raw_input` trong `project_idea`, chỉ đọc, không thuộc 4 trường của hợp đồng `idea.v1`. Dùng cho "Làm lại" và cho Creator xem lại ý gốc
   - ✅ Strengths: "Làm lại" bám ý gốc; không mất văn bản Creator đã gõ
   - ⚠️ Trade-offs: thêm một cột ngoài hợp đồng

B) Không lưu. Textbox chỉ là đầu vào cho lượt AI đầu; sau đó 4 trường là nguồn duy nhất
   - ✅ Strengths: đúng nguyên văn câu trả lời
   - ⚠️ Trade-offs: "Làm lại toàn bộ" không còn ý gốc để viết lại, chỉ còn 4 trường hiện có và ghi chú

X) Other (please describe after [Answer]: tag below)

[Answer]: 

## Question 2
Màn bước 1 hiện nay là màn tạo project (chủ đề, dạng video, ngôn ngữ, thời lượng, nền tảng) rồi AI viết brief. Bạn nói chưa quan tâm các ô đó và brief. CR này đặt màn Idea ở đâu?

A) Màn Idea mới thay phần nhập chủ đề của màn tạo project; các ô còn lại (dạng video, thời lượng, nền tảng) giữ nguyên ở nơi chúng đang nằm, chưa đổi trong CR này. Duyệt Idea thì sang bước tiếp theo hiện có
   - ✅ Strengths: tối thiểu thay đổi, đúng "sửa brief sau"
   - ⚠️ Trade-offs: màn tạo project phải tách được phần chủ đề khỏi các ô còn lại; cần xem kỹ code màn hiện tại

B) Màn Idea là bước riêng đứng trước màn hiện tại; màn hiện tại giữ nguyên, kể cả ô chủ đề
   - ✅ Strengths: không đụng màn cũ
   - ⚠️ Trade-offs: Creator nhập chủ đề hai lần cho tới khi CR-094 dọn

X) Other (please describe after [Answer]: tag below)

[Answer]: 

## Question 3
Câu 8: ô chọn ngôn ngữ nằm trên màn Idea. Hai việc khác nhau cần phân biệt: ngôn ngữ mà 4 trường Idea được viết, và ngôn ngữ mà brief (và các bước sau) được viết.

A) Một ô duy nhất: ngôn ngữ chọn ở màn Idea áp cho cả Idea và được lưu làm ngôn ngữ nội dung của project; brief và các bước sau đọc nó
   - ✅ Strengths: một nguồn, không lệch
   - ⚠️ Trade-offs: phải ghi vào ngôn ngữ nội dung của project ở orchestrator hoặc có một nguồn tương đương

B) Ô ngôn ngữ chỉ lưu trong `project_idea` (cột `language`); Idea luôn viết bằng tiếng Việt; brief đọc cột này để biết viết bằng ngôn ngữ nào
   - ✅ Strengths: không đụng cài đặt project hiện có
   - ⚠️ Trade-offs: hai nơi cùng nói về ngôn ngữ cho tới khi CR-094 gộp lại

C) Ô ngôn ngữ chỉ lưu trong `project_idea`; Idea viết bằng ngôn ngữ đã chọn; brief đọc cột này
   - ✅ Strengths: Idea cùng ngôn ngữ với video
   - ⚠️ Trade-offs: như B về hai nơi; Creator phải đọc Idea bằng ngôn ngữ họ có thể không quen

X) Other (please describe after [Answer]: tag below)

[Answer]: 

## Question 4
Bạn muốn dùng khung đơn vị (duyệt, làm lại, "cần làm lại" khi đầu vào đổi) cho Idea, hay một bảng và luồng riêng? Câu 6 chọn bảng `project_idea`.

A) Bảng `project_idea` là nơi lưu dữ liệu; Idea vẫn đăng ký là một đơn vị duyệt trong khung để dùng duyệt, làm lại, trạng thái; trạng thái và nội dung đọc từ bảng riêng, không lưu hai bản
   - ✅ Strengths: dùng lại hộp xác nhận, làm lại kèm ghi chú, thanh bước ↻; một nơi dữ liệu
   - ⚠️ Trade-offs: phải kiểm xem khung cho phép kho lưu riêng không; nếu không cho thì cần mở rộng khung

B) Bảng `project_idea` và API riêng của Idea (lưu, AI viết, review, duyệt), không dùng khung đơn vị
   - ✅ Strengths: đơn giản, độc lập
   - ⚠️ Trade-offs: viết lại duyệt, làm lại, đếm lần làm lại; bước sau không tự biết khi Idea đổi

X) Other (please describe after [Answer]: tag below)

[Answer]: 

## Question 5
Kết quả AI review (độ rõ, khả năng nghiên cứu, phạm vi, chỗ chưa rõ, gợi ý cải thiện) có lưu không?

A) Lưu trong `project_idea` (cột JSON `review`) cho bản đang hiển thị; chạy lại review thay bản cũ; sửa trường nào sau review thì review đánh dấu "cũ"
   - ✅ Strengths: Creator mở lại vẫn thấy; biết review đã cũ
   - ⚠️ Trade-offs: thêm một cột JSON

B) Không lưu, chạy lại mỗi lần mở màn
   - ✅ Strengths: không thêm cột
   - ⚠️ Trade-offs: tốn một lượt AI mỗi lần mở; kết quả thay đổi giữa các lần

X) Other (please describe after [Answer]: tag below)

[Answer]: 
