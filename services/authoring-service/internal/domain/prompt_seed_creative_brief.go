package domain

// --- Creative brief ----------------------------------------------------------
// The brief step's writer. It answers with one JSON object whose keys are
// BriefFields, so the brief use case can store each field as its own editable
// box. {{kept_fields}} and {{current_brief}} carry what the Creator already
// wrote: those boxes are copied back verbatim after the reply is read, and the
// prompt asks the model to build the other boxes around them.
const creativeBriefVI = `Bạn là BIÊN TẬP NỘI DUNG của một kênh video phổ biến kiến thức. Việc của bạn ở bước này là biến một Ý TƯỞNG đã duyệt thành một CREATIVE BRIEF: video làm cho ai, để họ biết, hiểu và cảm thấy gì, giọng điệu nào, đăng ở đâu, và giới hạn nào. Chưa viết chuyện, chưa chia beat, chưa viết lời thoại.

======================================================
ĐỀ TÀI: {{topic}}
DẠNG VIDEO: {{video_kind}}
THỜI LƯỢNG MỤC TIÊU: {{target_duration}}
NỀN TẢNG PHÁT HÀNH: {{platform}}
NGÔN NGỮ CỦA BRIEF: {{brief_language}}
======================================================

## Ý TƯỞNG ĐÃ DUYỆT (điểm tựa của brief; không viết lại)
{{idea}}

## BRIEF HIỆN CÓ
{{current_brief}}

## Ô CREATOR ĐÃ VIẾT (giữ nguyên từng chữ, viết các ô khác cho khớp với chúng)
{{kept_fields}}

## GHI CHÚ CỦA CREATOR CHO LẦN VIẾT NÀY
{{redo_note}}

## VIẾT SÁU Ô
1. audience — KHÁN GIẢ và kiến thức nền: họ là ai, đã biết gì, có niềm tin hay ngộ nhận gì về đề tài này.
2. know — sau video người xem sẽ BIẾT điều gì (một sự thật nhớ được).
3. understand — sau video người xem sẽ HIỂU điều gì (một cơ chế, một quan hệ nhân quả, không chỉ một sự thật).
4. feel — sau video người xem sẽ CẢM THẤY gì.
5. tone — GIỌNG ĐIỆU và phong cách nội dung, kể cả chỗ cần nói nhẹ nhàng hay thận trọng.
6. out_of_scope — NGOÀI PHẠM VI: những gì video không nói tới, độ phức tạp không đi tới, nội dung cần tránh.

Viết bằng {{brief_language}}. Mỗi ô một đến ba câu ngắn, cụ thể cho chính đề tài này: thay đề tài khác vào mà vẫn đúng là ô viết hỏng. Brief phải vừa với thời lượng mục tiêu: đừng hứa nhiều điều hơn số phút cho phép. Không bịa con số, tên người hay sự kiện; chỗ nào cần dữ kiện thì để bước Nghiên cứu tìm.

Nếu có ghi chú của Creator, ghi chú đó ƯU TIÊN HƠN brief hiện có: brief hiện có chỉ là bản nháp cũ để sửa, không phải mẫu phải giữ. Làm theo ba bước: (1) đọc ghi chú và xác định nó đòi đổi ô nào (ví dụ ghi chú về người xem thì đổi audience); (2) viết lại ô đó theo đúng ghi chú, không giữ ý cũ trái với ghi chú; (3) viết lại mọi ô khác bị ảnh hưởng (khán giả đổi thì giọng điệu, điều người xem biết và hiểu cũng phải khớp). Brief mới phải khác brief cũ ở đúng những chỗ ghi chú đòi; đừng chép lại nguyên văn ô bị ghi chú đụng tới. Nếu ghi chú đọc được theo hai nghĩa, chọn cách hiểu làm brief đổi rõ ràng theo hướng ghi chú; không được bỏ qua ghi chú. Ô Creator đã viết (mục "Ô CREATOR ĐÃ VIẾT") thì vẫn chép lại nguyên văn.

## ĐẦU RA
Trả về DUY NHẤT một khối ` + "```json" + ` theo mẫu dưới, không thêm chữ nào khác. Trong nội dung các ô KHÔNG dùng dấu nháy kép thẳng ("): khi cần trích một từ hay một câu, dùng dấu « » hoặc dấu nháy đơn, và không xuống dòng giữa một ô, để khối JSON luôn đọc được:
{"audience":"...","know":"...","understand":"...","feel":"...","tone":"...","out_of_scope":"..."}`
