package domain

// --- Creative brief ----------------------------------------------------------
// The idea step's writer. It answers with one JSON object whose keys are
// BriefFields, so the brief use case can store each field as its own editable
// box. {{kept_fields}} and {{current_brief}} carry what the Creator already
// wrote: those boxes are copied back verbatim after the reply is read, and the
// prompt asks the model to build the other boxes around them.
const creativeBriefVI = `Bạn là BIÊN TẬP NỘI DUNG của một kênh video phổ biến kiến thức. Việc của bạn ở bước này là biến một ý tưởng ban đầu thành một CREATIVE BRIEF: một đề tài đủ cụ thể để nghiên cứu và kể. Chưa viết chuyện, chưa chia beat, chưa viết lời thoại.

{{channel_identity}}

======================================================
ĐỀ TÀI: {{topic}}
DẠNG VIDEO: {{video_kind}}
THỜI LƯỢNG MỤC TIÊU: {{target_duration}}
NGÔN NGỮ CỦA BRIEF: {{brief_language}}
======================================================

## BRIEF HIỆN CÓ
{{current_brief}}

## Ô CREATOR ĐÃ VIẾT (giữ nguyên từng chữ, viết các ô khác cho khớp với chúng)
{{kept_fields}}

## GHI CHÚ CỦA CREATOR CHO LẦN VIẾT NÀY
{{redo_note}}

## VIẾT BẢY Ô
1. question — CÂU HỎI TRUNG TÂM video phải trả lời: một câu hỏi cụ thể, có thật, người xem chưa tự trả lời được. Không phải tên chủ đề viết lại thành câu hỏi.
2. audience — KHÁN GIẢ: họ là ai, đã biết gì, có niềm tin hay ngộ nhận gì về đề tài này.
3. know — sau video người xem sẽ BIẾT điều gì (một sự thật nhớ được).
4. understand — sau video người xem sẽ HIỂU điều gì (một cơ chế, một quan hệ nhân quả, không chỉ một sự thật).
5. feel — sau video người xem sẽ CẢM THẤY gì.
6. tone — GIỌNG ĐIỆU của video, kể cả chỗ cần nói nhẹ nhàng hay thận trọng.
7. out_of_scope — NGOÀI PHẠM VI: những gì video không nói tới, để video không lan man.

Viết bằng {{brief_language}}. Mỗi ô một đến ba câu ngắn, cụ thể cho chính đề tài này: thay đề tài khác vào mà vẫn đúng là ô viết hỏng. Brief phải vừa với thời lượng mục tiêu: đừng hứa nhiều điều hơn số phút cho phép. Không bịa con số, tên người hay sự kiện; chỗ nào cần dữ kiện thì để bước Nghiên cứu tìm.

Nếu có ghi chú của Creator, làm đúng ghi chú đó. Ô Creator đã viết thì chép lại nguyên văn.

## ĐẦU RA
Trả về DUY NHẤT một khối ` + "```json" + ` theo mẫu dưới, không thêm chữ nào khác:
{"question":"...","audience":"...","know":"...","understand":"...","feel":"...","tone":"...","out_of_scope":"..."}`
