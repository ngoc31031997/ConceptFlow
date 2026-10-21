package domain

import "fmt"

// --- Script writing ----------------------------------------------------------
// The script step's writer. One prompt serves four jobs, told apart by {{task}}:
// write the dialogue of a beat, write it again, suggest a replacement for one
// flagged line, or suggest a replacement for the first line of a beat so it
// joins the beat before it. {{beat}} is the beat the job is about; {{previous}}
// and {{following}} are its neighbours' dialogue; {{current_lines}} is the
// dialogue the beat has now, with its line ids.
const scriptWriterVI = `Bạn là BIÊN KỊCH LỜI DẪN của một kênh video phổ biến kiến thức. Việc của bạn ở bước này là viết lời thoại (lời dẫn đọc thành tiếng) cho từng BEAT của video.

======================================================
ĐỀ TÀI: {{topic}}
NGÔN NGỮ: {{brief_language}}
THỜI LƯỢNG MỤC TIÊU CẢ VIDEO: {{target}}
======================================================

## CREATIVE BRIEF
{{brief}}

## KHUNG CÂU CHUYỆN ĐÃ DUYỆT
{{story}}

## BẢNG DỮ KIỆN (mã, loại, nội dung)
{{facts}}

## BEAT CẦN VIẾT
{{beat}}

## LỜI THOẠI CỦA BEAT LIỀN TRƯỚC
{{previous}}

## LỜI THOẠI CỦA BEAT LIỀN SAU
{{following}}

## LỜI THOẠI HIỆN CÓ CỦA BEAT NÀY (kèm mã câu)
{{current_lines}}

## GHI CHÚ CỦA CREATOR CHO LẦN VIẾT NÀY
{{redo_note}}

## VIỆC CẦN LÀM
{{task}}

## LUẬT VIẾT LỜI ĐỂ ĐỌC
- Viết để NGHE, không phải để đọc: câu ngắn, mỗi câu một ý, không quá 28 từ, không dùng ngoặc, ký hiệu (%, /, &), chữ viết tắt hay số dài; viết số bằng chữ khi có thể.
- Chuyển tiếp tự nhiên: câu đầu của beat nối được với câu cuối của beat liền trước, không lặp lại cùng một cách mở câu.
- Đúng nhịp: độ dài lời thoại của beat vừa với thời lượng mục tiêu của beat (đọc khoảng 2,3 từ mỗi giây).
- Đủ rõ mà không dài dòng. Chỉ nói điều thuộc mục tiêu và nội dung của beat.
- Dữ kiện là giả thuyết thì nói thận trọng (“có thể”, “nhiều nhà nghiên cứu cho rằng”). Không bịa dữ kiện, con số hay tên ngoài bảng dữ kiện và khung câu chuyện. Viết bằng {{brief_language}}.

## ĐẦU RA
Trả về DUY NHẤT một khối ` + "```json" + ` theo mẫu của việc cần làm, không thêm chữ nào khác. Trong nội dung KHÔNG dùng dấu nháy kép thẳng ("): dùng dấu « » hoặc nháy đơn, và không xuống dòng giữa một ô.`

// Tasks of the script writer prompt, the values of {{task}}.
const (
	scriptTaskWriteVI = "Viết lời thoại cho beat trên. Trả về {\"lines\":[{\"text\":\"...\",\"facts\":[\"F01\"]}]}: các câu theo thứ tự đọc; \"facts\" là mã dữ kiện câu đó nói, chỉ dùng mã có trong bảng dữ kiện, để [] nếu câu không dùng dữ kiện."
	scriptTaskRedoVI  = "Viết lại TOÀN BỘ lời thoại của beat trên theo ghi chú của Creator. Trả về {\"lines\":[{\"text\":\"...\",\"facts\":[\"F01\"]}]} như khi viết mới; \"facts\" chỉ dùng mã có trong bảng dữ kiện."
	scriptTaskLineVI  = "Gợi ý câu thay cho câu %s (xem lời thoại hiện có) vì câu đó bị cờ: %s. Giữ ý và các dữ kiện của câu. Trả về {\"text\":\"...\"} với đúng một câu thay thế."
	scriptTaskJoinVI  = "Gợi ý câu thay cho CÂU ĐẦU (%s) của beat này để nó nối mượt với câu cuối của beat liền trước, vì: %s. Giữ ý và các dữ kiện của câu. Trả về {\"text\":\"...\"} với đúng một câu thay thế."
)

// ScriptTaskWrite is {{task}} when the dialogue of a beat is written.
func ScriptTaskWrite() string { return scriptTaskWriteVI }

// ScriptTaskRedo is {{task}} when the dialogue of a beat is written again.
func ScriptTaskRedo() string { return scriptTaskRedoVI }

// ScriptTaskLine is {{task}} when line id is flagged with reason.
func ScriptTaskLine(id, reason string) string { return fmt.Sprintf(scriptTaskLineVI, id, reason) }

// ScriptTaskJoin is {{task}} when the first line (id) of a beat is to join the beat before it.
func ScriptTaskJoin(id, reason string) string { return fmt.Sprintf(scriptTaskJoinVI, id, reason) }
