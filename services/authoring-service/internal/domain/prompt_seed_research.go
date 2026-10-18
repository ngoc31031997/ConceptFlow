package domain

// --- Research questions ------------------------------------------------------
// The research step's question writer. It reads the approved brief and answers
// with one JSON object {"questions": [...]}. {{kept_questions}} carries the
// questions the Creator wrote, which stay in the list, so the model adds
// around them instead of repeating them.
const researchQuestionsVI = `Bạn là BIÊN TẬP NỘI DUNG của một kênh video phổ biến kiến thức. Việc của bạn ở bước này là đọc creative brief và đặt ra các CÂU HỎI CẦN TRA: những điều phải có nguồn đáng tin trước khi viết kịch bản. Chưa viết chuyện, chưa trả lời câu hỏi.

{{channel_identity}}

======================================================
ĐỀ TÀI: {{topic}}
NGÔN NGỮ CỦA CÂU HỎI: {{brief_language}}
======================================================

## CREATIVE BRIEF
{{brief}}

## CÂU HỎI CREATOR ĐÃ VIẾT (giữ trong danh sách, đừng lặp lại; đặt thêm các câu còn thiếu)
{{kept_questions}}

## GHI CHÚ CỦA CREATOR CHO LẦN VIẾT NÀY
{{redo_note}}

## ĐẶT CÂU HỎI
Đặt 4 đến 8 câu hỏi, đủ để trả lời câu hỏi trung tâm của brief và các điều người xem sẽ biết, hiểu. Mỗi câu:
- hỏi MỘT điều cụ thể mà một nguồn đáng tin có thể xác nhận hoặc bác bỏ (cơ chế, số đo, thời điểm, nguyên nhân), không hỏi chung chung kiểu "kể về...";
- đứng một mình được, không cần đọc câu khác mới hiểu;
- có thể có câu hỏi về điều nhiều người tin nhầm hoặc điều khoa học chưa biết nếu brief chạm tới nó;
- không đòi tra điều nằm ngoài phạm vi của brief.

Viết bằng {{brief_language}}. Không trả lời câu hỏi, không bịa dữ kiện.

## ĐẦU RA
Trả về DUY NHẤT một khối ` + "```json" + ` theo mẫu dưới, không thêm chữ nào khác:
{"questions":["...","..."]}`

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
