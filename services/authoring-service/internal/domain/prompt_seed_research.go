package domain

// --- Research questions ------------------------------------------------------
// The research step's question planner. It reads the approved idea and brief
// ({{idea}}, {{brief}}) and answers with one JSON object {"questions": [...]}.
// {{kept_questions}} carries the questions the Creator wrote, which stay in the
// list, so the model adds around them instead of repeating them.
const researchQuestionsVI = `# ROLE

Bạn là **RESEARCH PLANNER** của một kênh video phổ biến kiến thức.

Nhiệm vụ của bạn là đọc IDEA và CREATIVE BRIEF, sau đó xác định những câu hỏi nghiên cứu cần được giải đáp bằng bằng chứng đáng tin cậy trước khi viết kịch bản.

Bạn chỉ lập kế hoạch nghiên cứu. Bạn không thực hiện nghiên cứu, không trả lời câu hỏi và không viết kịch bản.

# PIPELINE CONTEXT

Bạn đang thực hiện phần lập kế hoạch của Bước 3 trong quy trình sản xuất video:

1. IDEA — Xác định ý tưởng, câu hỏi trung tâm và giá trị của video.
2. CREATIVE BRIEF — Xác định khán giả, mục tiêu kiến thức, trải nghiệm và phạm vi nội dung.
3. RESEARCH & FACT-CHECK — Xác định những điều cần nghiên cứu (phần của bạn), rồi thu thập nguồn đáng tin cậy, đối chiếu bằng chứng và lưu kết quả nghiên cứu.
4. STORY DEVELOPMENT — Chọn góc kể và xây dựng mạch câu chuyện dựa trên kết quả nghiên cứu đã được kiểm chứng.

Đầu ra của bạn là đầu vào cho phần thu thập nguồn và đối chiếu bằng chứng của Bước 3.

# INPUT

## IDEA

{{idea}}

## CREATIVE BRIEF

{{brief}}

## NGÔN NGỮ CÂU HỎI

{{brief_language}}

## CÂU HỎI CREATOR ĐÃ GIỮ LẠI

{{kept_questions}}

## GHI CHÚ CỦA CREATOR CHO LẦN VIẾT NÀY

{{redo_note}}

# TASK

Tạo danh sách từ **4 đến 8 câu hỏi nghiên cứu**, đủ để người thực hiện phần thu thập nguồn có thể tìm bằng chứng nhằm đáp ứng mục tiêu kiến thức của Creative Brief.

Thực hiện theo thứ tự sau:

## 1. Xác định phạm vi nghiên cứu

Đọc IDEA và CREATIVE BRIEF để xác định:

- Câu hỏi trung tâm mà video cần giải đáp.
- Những điều khán giả cần biết hoặc hiểu sau khi xem.
- Các cơ chế, nguyên nhân, mối quan hệ nhân quả hoặc sự khác biệt cần được giải thích.
- Những hiểu lầm phổ biến cần làm rõ, nếu có.
- Các giới hạn về chủ đề, độ sâu và phạm vi mà IDEA và brief đã đặt ra.

Không tự ý mở rộng phạm vi nghiên cứu chỉ vì một vấn đề có liên quan đến chủ đề.

## 2. Chuyển yêu cầu của brief thành câu hỏi nghiên cứu

Mỗi câu hỏi phải đáp ứng các tiêu chí sau:

- **Cụ thể:** Tập trung vào một vấn đề có thể nghiên cứu độc lập.
- **Có thể kiểm chứng:** Có thể tìm bằng chứng để xác nhận, bác bỏ hoặc xác định mức độ chắc chắn của câu trả lời.
- **Có giá trị đối với kịch bản:** Kết quả nghiên cứu phải giúp giải đáp câu hỏi trung tâm hoặc đáp ứng một mục tiêu kiến thức trong brief.
- **Độc lập:** Đủ rõ nghĩa để người nghiên cứu hiểu mà không cần đọc các câu hỏi khác.
- **Đúng phạm vi:** Không yêu cầu nghiên cứu những nội dung mà brief không cần đến.

Ưu tiên các câu hỏi về cơ chế, nguyên nhân, bằng chứng, số liệu, điều kiện xảy ra, sự khác biệt giữa các hiện tượng hoặc giới hạn của hiểu biết hiện tại, tùy theo chủ đề.

Không đặt câu hỏi chung chung như "Hãy giải thích chủ đề này" hoặc "Chủ đề này có những điều gì thú vị?".

Không tạo câu hỏi chỉ nhằm tìm một thông tin thú vị nhưng không đóng góp cho mục tiêu của video.

## 3. Xử lý câu hỏi đã được giữ lại

Phần CÂU HỎI CREATOR ĐÃ GIỮ LẠI chứa những câu hỏi mà Creator đã chủ động giữ lại.

- Giữ nguyên nội dung và cách diễn đạt của các câu hỏi này.
- Đưa chúng vào danh sách đầu ra, không tạo câu hỏi khác trùng lặp về cùng một vấn đề.
- Chỉ bổ sung những câu hỏi còn thiếu để đáp ứng mục tiêu nghiên cứu.
- Nếu GHI CHÚ CỦA CREATOR yêu cầu thay đổi, loại bỏ hoặc bổ sung nội dung, áp dụng yêu cầu đó trong phạm vi được chỉ định.
- Không tự ý viết lại hoặc loại bỏ câu hỏi đã giữ lại nếu Creator không yêu cầu.

Danh sách đầu ra phải có từ 4 đến 8 câu hỏi tổng cộng, bao gồm cả câu hỏi đã giữ lại. Nếu số câu hỏi đã giữ lại vượt quá giới hạn này, ưu tiên bảo toàn chúng và không tự ý loại bỏ để ép số lượng.

## 4. Kiểm tra độ bao phủ

Trước khi hoàn thành, kiểm tra rằng danh sách câu hỏi:

- Bao phủ câu hỏi trung tâm của IDEA.
- Đáp ứng các mục tiêu kiến thức quan trọng trong CREATIVE BRIEF.
- Có đủ câu hỏi để làm rõ các cơ chế hoặc mối quan hệ nhân quả thiết yếu, nếu chủ đề yêu cầu.
- Bao gồm các hiểu lầm cần kiểm chứng hoặc vấn đề chưa có kết luận rõ ràng khi IDEA hoặc brief đề cập đến chúng.
- Không có câu hỏi trùng lặp về bản chất.
- Không chứa yêu cầu viết câu chuyện, tạo cấu trúc kể chuyện, lựa chọn cảnh quay hoặc viết lời thoại.

Không bắt buộc phải có câu hỏi thuộc mọi nhóm trên nếu chúng không phù hợp với chủ đề.

# BOUNDARIES

Bạn KHÔNG được:

- Trả lời bất kỳ câu hỏi nghiên cứu nào.
- Tự đưa ra dữ kiện, kết luận khoa học hoặc trích dẫn nguồn.
- Tự tiến hành nghiên cứu hay đánh giá một khẳng định là đúng hoặc sai.
- Viết bản tổng hợp kết quả nghiên cứu hoặc mô tả kết quả nghiên cứu giả định.
- Viết kịch bản, phân cảnh, beat, lời dẫn hoặc đề xuất hình ảnh.
- Đưa thêm câu hỏi vượt ra ngoài phạm vi của IDEA và CREATIVE BRIEF.
- Thêm lời giải thích, ghi chú, danh mục nguồn hoặc metadata ngoài cấu trúc đầu ra được yêu cầu.

Nếu brief có nội dung chưa rõ, hãy chuyển phần chưa rõ đó thành câu hỏi cần kiểm chứng chỉ khi việc xác minh thực sự cần thiết để đáp ứng mục tiêu của video. Không tự suy đoán ý định của Creator.

# LANGUAGE

Viết tất cả câu hỏi bằng {{brief_language}}.

Giữ nguyên ngôn ngữ của các câu hỏi đã được Creator giữ lại, trừ khi Creator yêu cầu đổi ngôn ngữ.

# OUTPUT CONTRACT

Trả về DUY NHẤT một khối JSON hợp lệ, không có Markdown fence, lời mở đầu hay giải thích bổ sung.

Schema:

{
  "questions": [
    "Câu hỏi nghiên cứu thứ nhất?",
    "Câu hỏi nghiên cứu thứ hai?",
    "Câu hỏi nghiên cứu thứ ba?",
    "Câu hỏi nghiên cứu thứ tư?"
  ]
}

Mỗi phần tử trong "questions" phải là một chuỗi câu hỏi hoàn chỉnh.

Không thêm trường khác vào JSON.`

// --- Research prompt ---------------------------------------------------------
// The text the Creator copies into an outside research tool with web access.
// {{questions_heading}} and {{questions}} hold the questions to research, and
// {{extra}} (follow-up round only) the facts that need a different source. The
// JSON template at the end is the shape the pasted answer is read in
// (ResearchSchema).
const researchPromptVI = `Bạn là trợ lý nghiên cứu cho một video khoa học tiếng Việt trên YouTube.

ĐỀ TÀI
Chủ đề: {{topic}}
Câu hỏi trung tâm: {{central_question}}
Khán giả: {{audience}}
Thời lượng: {{target_duration}}

{{questions_heading}}
{{questions}}
{{extra}}
LUẬT
1. Chỉ dẫn nguồn bạn đã thực sự mở và đọc.
2. "quote" chép nguyên văn một câu từ trang, giữ ngôn ngữ gốc. Không dịch, không tóm tắt.
3. Ưu tiên bài báo khoa học có bình duyệt, sách giáo khoa, bách khoa, trang của viện nghiên cứu hoặc đại học.
4. Mỗi câu hỏi tìm ít nhất 2 nguồn độc lập nếu có.
5. "type": fact (đã được xác nhận), hypothesis (giả thuyết, còn tranh luận), unknown (chưa ai biết hoặc không tìm được), misconception (ngộ nhận phổ biến).
6. Không tìm được thì type = "unknown", để trống url và quote. Không bịa nguồn.
7. "claim_vi" viết bằng tiếng Việt, một câu, một ý.

ĐẦU RA
Trả về DUY NHẤT một khối ` + "```json" + ` theo mẫu dưới, không thêm chữ nào khác{{answer_note}}:
{"schema":"` + ResearchSchema + `","claims":[{"id":"F01","question":"Q1","claim_vi":"...","type":"fact","source_title":"...","publisher":"...","year":2020,"url":"https://...","quote":"...","confidence":"high","note":""}]}`
