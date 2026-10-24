# CR-093 · Bước 1 Idea — câu hỏi yêu cầu

Lộ trình bảy CR: `aidlc-docs/inception/plans/cr-093-099-content-flow-roadmap.md`. CR này chỉ làm bước 1.

## Mục tiêu theo luồng mẫu

- **Input**: câu hỏi thô của Creator, ví dụ "Chuyện gì xảy ra nếu ta rơi vào hố đen? Ta có bị kéo dài như sợi mì không? Người bên ngoài có thấy ta biến mất không?"
- **Công việc**: chọn câu hỏi trung tâm; ghi các điểm gây tò mò; chưa viết lời dẫn, chưa nghĩ hình ảnh.
- **Output `idea.v1`** (tương ứng `idea.md`): chủ đề, câu hỏi trung tâm, các điểm gây tò mò, phạm vi.
- **Hoàn thành khi**: có câu hỏi đủ rõ để bắt đầu nghiên cứu.

## Hiện tại

Bước 1 "Ý tưởng · creative brief" (`creative_brief.go`, đơn vị `brief:main`): Creator nhập chủ đề, dạng video, ngôn ngữ, thời lượng, nền tảng; AI viết brief sáu ô (khán giả, biết, hiểu, cảm thấy, giọng điệu, ngoài phạm vi); duyệt brief thì sang Cấu hình. Câu chuyện, Nghiên cứu, Beat đọc `brief:main`.

Vui lòng điền chữ cái sau mỗi `[Answer]:`.

## Question 1
`idea.v1` gồm những trường nào?

A) Đúng mẫu: `topic` (chủ đề), `central_question` (câu hỏi trung tâm), `curiosity_points[]` (điểm gây tò mò), `scope` (phạm vi)
   - ✅ Strengths: đúng mẫu, gọn
   - ⚠️ Trade-offs: không giữ lại câu hỏi thô Creator gõ

B) Như A, thêm `raw_input` (câu hỏi thô, chỉ đọc) để bước sau và lần "làm lại" biết ý ban đầu
   - ✅ Strengths: truy được ý gốc; AI làm lại không trôi khỏi ý Creator
   - ⚠️ Trade-offs: thêm một trường ngoài mẫu

X) Other (please describe after [Answer]: tag below)

[Answer]: bước này sủa lại input chỉ còn là textbox lớn (tương tự giao diện của chatgpt hay google vậy, user chỉ cần nhập ý tưởng dạng text), AI nhận input và trả về các trường
và các trường này sẽ được lưu theo project chứ không lưu md vào db nhé
ta sẽ lưu như sau
topic
central_question
curiosity_points []
scope


## Question 2
Ai viết bốn trường của Idea?

A) Creator gõ câu hỏi thô; AI viết nháp bốn trường; Creator sửa từng trường, làm lại kèm ghi chú, rồi duyệt (giống các bước khác)
   - ✅ Strengths: nhanh, đồng bộ với khung đơn vị (ADR-0035)
   - ⚠️ Trade-offs: thêm một vai AI và prompt mới

B) Creator tự điền cả bốn trường, không có AI
   - ✅ Strengths: đơn giản; ý tưởng hoàn toàn của Creator
   - ⚠️ Trade-offs: chậm khi chỉ có một câu hỏi thô

C) Creator tự điền, có nút "AI gợi ý" cho từng trường khi cần
   - ✅ Strengths: Creator chủ động, AI chỉ hỗ trợ
   - ⚠️ Trade-offs: nhiều lượt AI nhỏ

X) Other (please describe after [Answer]: tag below)

[Answer]: A 

## Question 3
Các ô đang có ở màn bước 1 (dạng video, ngôn ngữ nội dung, thời lượng mục tiêu, nền tảng) để ở đâu? Mẫu đặt thời lượng ở Creative brief.

A) Giữ ở bước 1, nhập cùng câu hỏi thô khi tạo project (cần để tạo project), nhưng không thuộc `idea.v1`; brief đọc chúng từ project
   - ✅ Strengths: tạo project như hiện nay; `idea.v1` đúng mẫu
   - ⚠️ Trade-offs: màn Idea có ô không thuộc Idea

B) Bước 1 chỉ có câu hỏi thô; các ô này chuyển sang Creative brief (CR-094)
   - ✅ Strengths: đúng mẫu nhất
   - ⚠️ Trade-offs: tạo project trước khi biết dạng video, phải đổi cách orchestrator tạo project; trong lúc chờ CR-094, các ô này cần chỗ tạm

X) Other (please describe after [Answer]: tag below)

[Answer]: chưa cần quan tâm tại CR này

## Question 4
Điều kiện "có câu hỏi đủ rõ để bắt đầu nghiên cứu" kiểm thế nào trước khi cho duyệt?

A) Máy kiểm: câu hỏi trung tâm không trống, là một câu hỏi (kết thúc bằng "?"), có ít nhất 1 điểm gây tò mò, phạm vi không trống; còn "đủ rõ" thì Creator quyết khi duyệt
   - ✅ Strengths: đơn giản, không phụ thuộc AI
   - ⚠️ Trade-offs: không bắt được câu hỏi mơ hồ

B) Như A, thêm AI chấm độ rõ và nêu chỗ mơ hồ (cảnh báo, không chặn duyệt)
   - ✅ Strengths: giúp Creator thấy chỗ yếu trước khi sang nghiên cứu
   - ⚠️ Trade-offs: thêm lượt AI; cảnh báo có thể sai

X) Other (please describe after [Answer]: tag below)

[Answer]: B nhưng tôi bổ sung thêm 
### Definition of Done — IDEA

**1. Machine Validation**
- `output` tuân thủ schema đã định nghĩa.
- Chủ đề, câu hỏi trung tâm, các điểm gây tò mò và phạm vi đều có nội dung hợp lệ.
- Có ít nhất một điểm gây tò mò liên quan đến chủ đề.

**2. Quality Review**
- AI đã hoàn thành đánh giá độ rõ ràng, khả năng nghiên cứu và phạm vi.
- Các vấn đề chưa rõ và gợi ý cải thiện được ghi nhận.
- AI review là bước hỗ trợ ra quyết định, không tự động thay thế Creator Approval.

**3. Creator Approval**
- Creator đã xem nội dung IDEA và kết quả review.
- Creator chọn Approve.
- Phiên bản `output` được duyệt được lưu lại và chuyển làm input cho bước Research.

**Completion Condition:** IDEA được coi là hoàn thành khi Machine Validation đạt và Creator Approval được xác nhận. AI Quality Review cung cấp thông tin hỗ trợ trước khi duyệt, nhưng không phải điều kiện chặn độc lập.

## Question 5
Trong lúc chưa có CR-094, brief hiện tại nối với Idea thế nào?

A) Idea là bước 1 riêng; brief hiện tại thành bước 2 tạm, AI viết brief đọc `idea.v1` đã duyệt; Cấu hình lùi một bước. CR-094 sau đó đổi brief theo mẫu
   - ✅ Strengths: luồng chạy được ngay, mỗi bước một chỗ duyệt
   - ⚠️ Trade-offs: luồng thêm một bước; ba bản định nghĩa luồng và thanh bước đổi trong CR này

B) Idea và brief cùng ở màn bước 1 (hai phần, duyệt Idea mới viết brief); CR-094 mới tách thành bước 2
   - ✅ Strengths: CR này không đổi số bước
   - ⚠️ Trade-offs: chưa đúng mẫu cho tới CR-094

X) Other (please describe after [Answer]: tag below)

[Answer]: chưa cần quan tâm bước brief cứ làm idea, sau đó implment lại brief sau

## Question 6
Hợp đồng `idea.v1` được trao qua đâu?

A) Lưu là đơn vị duyệt `idea:main` trong authoring-service; tài liệu `docs/contracts/idea-v1.md`; bước sau đọc qua một hàm/route đọc "Idea đã duyệt", không đọc bảng trực tiếp
   - ✅ Strengths: giữ khung đơn vị, phụ thuộc, "cần làm lại" (ADR-0035)
   - ⚠️ Trade-offs: hợp đồng là JSON trong DB, không phải file

B) Như A, thêm nút "Tải `idea.md`" xuất đúng định dạng mẫu
   - ✅ Strengths: có file mang ra ngoài như mẫu
   - ⚠️ Trade-offs: thêm định dạng xuất phải giữ khớp hợp đồng

C) Như B, và khi duyệt thì ghi `idea.md` vào thư mục chung của project (`/shared/{id}/content/`)
   - ✅ Strengths: bộ file đầy đủ theo mẫu, tích dần qua bảy bước
   - ⚠️ Trade-offs: file và DB có thể lệch khi sửa sau duyệt; cần ghi đè khi duyệt lại

X) Other (please describe after [Answer]: tag below)

[Answer]: tạo 1 bảng riêng project_idea để lưu data vào table, khi cần thì thì query để lấy lên giao diện hoặc đưa cho brief

## Question 7
Project đang có brief ở bước 1 xử lý thế nào?

A) Tự tạo `idea:main` từ dữ liệu cũ (chủ đề → topic, câu hỏi trung tâm lấy từ brief nếu có) và đánh dấu đã duyệt nếu brief đã duyệt
   - ✅ Strengths: project cũ không phải làm lại
   - ⚠️ Trade-offs: Idea dựng lại có thể thiếu điểm gây tò mò

B) Project cũ có Idea trống, chờ Creator viết; brief cũ giữ nguyên, không bị đánh "cần làm lại"
   - ✅ Strengths: không đoán dữ liệu
   - ⚠️ Trade-offs: project cũ đứng ở bước 1 cho tới khi viết Idea

C) Dữ liệu hiện có là dữ liệu thử, không cần chuyển
   - ✅ Strengths: đơn giản nhất
   - ⚠️ Trade-offs: project cũ hỏng luồng

X) Other (please describe after [Answer]: tag below)

[Answer]: không cần quan tâm cứ tập trung làm idea, sửa brief sau

## Question 8
Ngôn ngữ của các trường Idea?

A) Theo ngôn ngữ nội dung của project
   - ✅ Strengths: giống các bước khác
   - ⚠️ Trade-offs: cần biết ngôn ngữ trước khi viết Idea (liên quan câu 3)

B) Luôn tiếng Việt (ngôn ngữ làm việc của Creator)
   - ✅ Strengths: không phụ thuộc câu 3
   - ⚠️ Trade-offs: video tiếng Anh thì Idea vẫn tiếng Việt

X) Other (please describe after [Answer]: tag below)

[Answer]: bổ sung thêm trường chọn ngôn ngữ để AI viết brief theo ngôn ngữ được chọn
