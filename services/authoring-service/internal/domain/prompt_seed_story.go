package domain

// --- Story structure ---------------------------------------------------------
// The story step's writer. It reads the approved brief and the kept facts of
// the research step and answers with one JSON object holding the story's three
// fields. {{research}} is the fact table, or says there is none when the
// research step was skipped; {{current_story}} and {{kept_fields}} carry what
// is already stored and what the Creator wrote, which is copied back verbatim
// after the reply is read.
const storyFrameVI = `Bạn là BIÊN TẬP CÂU CHUYỆN của một kênh video phổ biến kiến thức. Việc của bạn ở bước này: chọn góc kể và xây mạch từ câu hỏi đến lời giải. Chưa chia beat, chưa viết lời thoại.

======================================================
NGÔN NGỮ: {{brief_language}}
======================================================

## CREATIVE BRIEF
{{brief}}

## BẢNG DỮ KIỆN (mã [loại · lưu ý]: nội dung)
{{research}}

## CÂU CHUYỆN HIỆN CÓ
{{current_story}}

## Ô CREATOR ĐÃ VIẾT (giữ nguyên từng chữ, viết các ô khác cho khớp với chúng)
{{kept_fields}}

## GHI CHÚ CỦA CREATOR CHO LẦN VIẾT NÀY
{{redo_note}}

## CÁC Ô CẦN VIẾT
1. central_question — CÂU HỎI TRUNG TÂM của video, một câu, đúng với tò mò mà brief nêu và người xem mục tiêu của brief.
2. core_answer — CÂU TRẢ LỜI CỐT LÕI: điều video sẽ chứng minh, một đến hai câu, trả lời đúng central_question. Không hứa quá điều dữ kiện cho phép.
3. narrative_arc — MẠCH KỂ từ câu hỏi đến lời giải, gồm các nhịp theo thứ tự người xem sẽ gặp:
   - "beat": tên ngắn của vai trò nhịp trong cách kể (ví dụ: Hook, Đặt bối cảnh, Hai góc nhìn, Giải thích, Payoff). Tự đặt tên cho hợp câu chuyện này;
   - "text": nhịp này nói gì, một đến hai câu, và nó dẫn sang nhịp sau thế nào;
   - "facts": mã các dữ kiện nhịp này dựa vào, lấy từ bảng dữ kiện (có thể rỗng).

## LUẬT
- SỐ NHỊP DO BẠN QUYẾT theo lượng và độ nặng của dữ kiện, từ 3 đến 12 nhịp: ít dữ kiện thì mạch gọn, nhiều dữ kiện thì tách nhịp, không gộp nhiều ý nặng vào một nhịp, không chèn nhịp độn cho đủ số.
- Nhịp đầu là hook dẫn thẳng vào central_question. Nhịp cuối là payoff, trả lời central_question bằng core_answer. Các nhịp giữa nối nhau như một đường dây: mỗi nhịp trả lời điều nhịp trước làm người xem thắc mắc. Danh sách thông tin xếp cạnh nhau không phải là câu chuyện.
- Chỉ trích dữ kiện bằng mã có trong bảng. Dữ kiện loại hypothesis phải nói thận trọng; misconception chỉ nhắc để gỡ; unknown chỉ dùng để nói khoa học chưa biết; dữ kiện «bắt buộc nói» phải xuất hiện trong ít nhất một nhịp.
- Mọi con số, tên, mốc thời gian phải có trong bảng dữ kiện; chỗ nào bảng không có thì nói định tính.
- Nếu bảng dữ kiện trống (bước Nghiên cứu đã bỏ qua): để "facts" rỗng ở mọi nhịp, chỉ nói điều định tính chắc chắn, không bịa con số, tên người hay mốc thời gian.
- Tuân theo "tránh" và mục tiêu học của brief; đúng giọng và độ dài mục tiêu của brief.
- Nếu có ghi chú của Creator, ghi chú ƯU TIÊN HƠN câu chuyện hiện có: câu chuyện hiện có chỉ là bản nháp cũ để sửa. Làm theo ghi chú rồi viết lại mọi ô bị ảnh hưởng cho khớp nhau. Ô Creator đã viết thì chép lại nguyên văn.
- Viết bằng {{brief_language}}.

## ĐẦU RA
Trả về DUY NHẤT một khối ` + "```json" + ` theo mẫu dưới, không thêm chữ nào khác, luôn đủ ba khoá. Trong nội dung KHÔNG dùng dấu nháy kép thẳng ("): dùng dấu « » hoặc nháy đơn, và không xuống dòng giữa một ô. Mẫu:
{"central_question":"...","core_answer":"...","narrative_arc":[{"beat":"...","text":"...","facts":["F01"]},{"beat":"...","text":"...","facts":[]}]}`
