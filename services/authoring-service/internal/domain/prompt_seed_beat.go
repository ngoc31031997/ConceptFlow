package domain

import (
	"fmt"
	"strings"
)

// --- Beat sheet --------------------------------------------------------------
// The beat sheet step's writer. One prompt serves three jobs, told apart by
// {{task}}: write the whole sheet, write one beat again, or write a new beat
// between two. It answers with one JSON object {"beats": [...]}.
// {{current_beats}} lists the stored beats with their keys and {{kept_beats}}
// the parts the Creator wrote, which are copied back verbatim after the reply
// is read.
const beatSheetVI = `Bạn là BIÊN KỊCH của một kênh video phổ biến kiến thức. Việc của bạn ở bước này là chia câu chuyện thành các BEAT: mỗi beat có một chức năng và đẩy người xem tiến một bước. Chưa viết lời thoại.

======================================================
ĐỀ TÀI: {{topic}}
NGÔN NGỮ: {{brief_language}}
THỜI LƯỢNG MỤC TIÊU: {{target}}
======================================================

## CREATIVE BRIEF
{{brief}}

## KHUNG CÂU CHUYỆN ĐÃ DUYỆT
{{story}}

## BẢNG DỮ KIỆN (mã, loại, nội dung)
{{facts}}

## CÁC BEAT HIỆN CÓ (theo thứ tự, kèm khoá)
{{current_beats}}

## PHẦN CREATOR ĐÃ VIẾT (giữ nguyên từng chữ)
{{kept_beats}}

## GHI CHÚ CỦA CREATOR CHO LẦN VIẾT NÀY
{{redo_note}}

## VIỆC CẦN LÀM
{{task}}

## MỖI BEAT GỒM
- "key": khoá của beat hiện có mà bạn viết lại (b1, b2...); để "" cho beat mới;
- "fn": chức năng, đúng một trong: Hook, Đặt vấn đề, Giải thích cơ chế, Cao trào, Mở rộng hệ quả, Payoff;
- "title": tên ngắn của beat;
- "goal": mục tiêu, tức bước tiến của beat với người xem, một câu;
- "content": nội dung chính của beat, 1 đến 2 câu, KHÔNG phải lời thoại;
- "facts": mã các dữ kiện beat dùng, chỉ dùng mã có trong bảng dữ kiện;
- "dur": thời lượng mục tiêu của beat, số giây nguyên, từ 5 trở lên;
- "next": một câu nói beat này nối sang beat sau thế nào; để "" ở beat cuối.

## LUẬT
- Beat đầu là Hook và không quá 20 giây; beat cuối là Payoff trả lời câu hỏi mở đầu của khung câu chuyện.
- Tổng "dur" của cả bảng nằm trong thời lượng mục tiêu.
- Mọi dữ kiện "bắt buộc nói" phải nằm trong ít nhất một beat. Dữ kiện là giả thuyết thì content nói thận trọng.
- Không bịa dữ kiện, con số hay tên ngoài bảng dữ kiện và khung câu chuyện. Viết bằng {{brief_language}}.
- Phần Creator đã viết thì chép lại nguyên văn; beat Creator viết thì giữ nguyên khoá.

## ĐẦU RA
Trả về DUY NHẤT một khối ` + "```json" + ` theo mẫu dưới, không thêm chữ nào khác. Trong nội dung KHÔNG dùng dấu nháy kép thẳng ("): dùng dấu « » hoặc nháy đơn, và không xuống dòng giữa một ô.
{"beats":[{"key":"","fn":"Hook","title":"...","goal":"...","content":"...","facts":["F01"],"dur":15,"next":"..."}]}`

// Tasks of the beat sheet prompt, the values of {{task}}.
const (
	beatTaskWriteVI = "Viết nội dung của ĐÚNG MỘT beat có khoá %s (beat thứ %d trong %d beat). Chức năng và dữ kiện của beat đã được chốt, giữ nguyên: chức năng %s; %s. Bước câu chuyện beat này kể: %s\n" +
		"Viết \"title\", \"goal\", \"content\", \"next\" và ước lượng \"dur\" (số giây) theo lượng nội dung beat cần, đọc các beat khác để mạch chuyện liền và không lặp ý. " +
		"Ngân sách thời lượng cả video: từ %s đến %s; các beat đã viết dùng %s; còn %d beat chưa viết, kể cả beat này. " +
		"Trả về một danh sách có đúng một beat, mang khoá %s, với \"fn\" và \"facts\" đúng như đã chốt."
	beatTaskRedoVI    = "Viết lại ĐÚNG MỘT beat có khoá %s, đọc các beat khác để khớp. Trả về một danh sách có đúng một beat, mang khoá %s."
	beatTaskAddVI     = "Viết MỘT beat mới đặt giữa beat %s và beat %s, để mạch chuyện liền. Trả về một danh sách có đúng một beat với key \"\"."
	beatTaskAddLastVI = "Viết MỘT beat mới đặt cuối sau beat %s. Trả về một danh sách có đúng một beat với key \"\"."
)

// BeatTaskWrite is {{task}} when the content of the beat with key is written
// from the approved skeleton: b is the beat as the skeleton fixed it, no and
// total place it in the sheet, lo and hi bound the whole video in seconds, used
// is the length of the beats already written and remaining counts the beats
// still to write, this one included.
func BeatTaskWrite(key string, no, total int, b Beat, lo, hi, used, remaining int) string {
	facts := "chưa có dữ kiện: để facts rỗng, đừng bịa"
	if len(b.Facts) > 0 {
		facts = "dữ kiện " + strings.Join(b.Facts, ", ")
	}
	step := b.Step
	if strings.TrimSpace(step) == "" {
		step = "(beat do Creator thêm: tự viết từ khung câu chuyện)"
	}
	return fmt.Sprintf(beatTaskWriteVI, key, no, total, b.Fn, facts, step, FormatSeconds(lo), FormatSeconds(hi), FormatSeconds(used), remaining, key)
}

// BeatTaskRedo is {{task}} when the beat with key is written again.
func BeatTaskRedo(key string) string { return fmt.Sprintf(beatTaskRedoVI, key, key) }

// BeatTaskAdd is {{task}} when a new beat goes after the beat with key after;
// before is the key of the beat that follows it, or "" when it goes last.
func BeatTaskAdd(after, before string) string {
	if before == "" {
		return fmt.Sprintf(beatTaskAddLastVI, after)
	}
	return fmt.Sprintf(beatTaskAddVI, after, before)
}
