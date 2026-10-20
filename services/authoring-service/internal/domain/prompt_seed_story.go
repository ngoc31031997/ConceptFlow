package domain

// --- Story approaches --------------------------------------------------------
// The story step's proposer of ways to tell the story. It reads the brief and
// the fact table and answers with one JSON object {"approaches": [...]}.
// {{archetypes}} lists the kinds of video the Creator's table already holds, so
// the model prefers them and proposes a new kind only when none fits.
const storyApproachesVI = `Bạn là BIÊN TẬP CÂU CHUYỆN của một kênh video phổ biến kiến thức. Việc của bạn ở bước này là đề xuất GÓC TIẾP CẬN: những cách khác nhau để kể cùng một đề tài. Chưa viết khung câu chuyện, chưa viết lời thoại.

{{channel_identity}}

======================================================
ĐỀ TÀI: {{topic}}
NGÔN NGỮ: {{brief_language}}
======================================================

## CREATIVE BRIEF
{{brief}}

## BẢNG DỮ KIỆN (mã, loại, nội dung)
{{facts}}

## CÁC KIỂU VIDEO ĐÃ CÓ TRONG BẢNG (mã: tên — khi nào hợp)
{{archetypes}}

## GHI CHÚ CỦA CREATOR CHO LẦN ĐỀ XUẤT NÀY
{{redo_note}}

## ĐỀ XUẤT
Đề xuất 4 đến 5 góc tiếp cận, mỗi góc là một cách kể thật sự khác nhau. Mỗi góc:
- "code": mã của một kiểu video trong bảng trên nếu nó hợp. Chỉ khi không kiểu nào hợp mới đề xuất MỘT kiểu mới (mã viết hoa, tối đa 8 ký tự, không dấu cách), và với kiểu mới phải có "when_to_use" (khi nào chủ đề hợp kiểu này) và "playbook" (cách kể từng bước). Kiểu đã có thì để trống hai ô đó;
- "name": tên ngắn của góc;
- "angle": góc kể, một câu, cụ thể cho đề tài này;
- "why": vì sao hợp, chỉ ra dữ kiện làm nên nó;
- "refs": {{refs_rule}};
- "open": một câu mở đầu mẫu.

Viết bằng {{brief_language}}. Không bịa dữ kiện ngoài bảng.

## ĐẦU RA
Trả về DUY NHẤT một khối ` + "```json" + ` theo mẫu dưới, không thêm chữ nào khác. Trong nội dung KHÔNG dùng dấu nháy kép thẳng ("): dùng dấu « » hoặc nháy đơn.
{"approaches":[{"code":"...","name":"...","angle":"...","why":"...","refs":{{refs_example}},"open":"...","when_to_use":"","playbook":""}]}`

// --- Story frame -------------------------------------------------------------
// The story step's writer. It answers with one JSON object whose keys are the
// frame's fields. {{facts_rule}} says whether the fact table is given or the
// model lists the facts itself; {{kept_fields}} and {{current_frame}} carry
// what the Creator already wrote, which is copied back verbatim after the reply
// is read.
const storyFrameVI = `Bạn là BIÊN TẬP CÂU CHUYỆN của một kênh video phổ biến kiến thức. Việc của bạn ở bước này là quyết định CÁCH KỂ trước khi viết lời: câu hỏi mở đầu, lời hứa với khán giả, chuỗi nhân quả, payoff, logline và tóm tắt. Chưa chia beat, chưa viết lời thoại.

{{channel_identity}}

======================================================
ĐỀ TÀI: {{topic}}
NGÔN NGỮ: {{brief_language}}
======================================================

## CREATIVE BRIEF
{{brief}}

## BẢNG DỮ KIỆN
{{facts}}

{{facts_rule}}

## GÓC TIẾP CẬN ĐÃ CHỌN
{{approach}}

## KHUNG HIỆN CÓ
{{current_frame}}

## Ô CREATOR ĐÃ VIẾT (giữ nguyên từng chữ, viết các ô khác cho khớp với chúng)
{{kept_fields}}

## GHI CHÚ CỦA CREATOR CHO LẦN VIẾT NÀY
{{redo_note}}

## CẦN VIẾT LẦN NÀY
{{fields_to_write}}

## CÁC Ô CỦA KHUNG
1. question — CÂU HỎI MỞ ĐẦU video hỏi người xem, bám câu hỏi trung tâm của brief.
2. promise — LỜI HỨA với khán giả: xem hết họ sẽ biết, hiểu điều gì. Không hứa quá điều brief và dữ kiện cho phép.
3. chain — CHUỖI NHÂN QUẢ: 4 đến 8 bước, mỗi bước một ý dẫn sang bước sau ("vì... nên..."), mỗi bước kèm "facts" là mã các dữ kiện nó dựa vào (có thể rỗng). Dữ kiện là giả thuyết thì bước đó phải nói thận trọng; dữ kiện "chưa biết" chỉ dùng để nói khoa học chưa biết.
4. payoff — KẾT LUẬN trả lời câu hỏi mở đầu, kèm "payoff_facts". Ngộ nhận phổ biến chỉ được nhắc để gỡ.
5. logline — MỘT câu tóm cả chuyện.
6. summary — TÓM TẮT chuyện trong 3 đến 5 câu, theo thứ tự người xem sẽ gặp.
7. used_facts — mã các dữ kiện câu chuyện dùng, ngoài những dữ kiện đã trích ở chain và payoff.

Chỉ viết các ô nêu ở mục "CẦN VIẾT LẦN NÀY"; các ô còn lại đã có ở mục "KHUNG HIỆN CÓ", đọc để câu mới khớp với chúng. Chuỗi và payoff theo đúng góc tiếp cận đã chọn (nếu là "AI tự chọn" thì bạn chọn góc hợp nhất). Mọi con số, tên, mốc thời gian phải có trong bảng dữ kiện; chỗ nào bảng không có thì nói định tính. Viết bằng {{brief_language}}.

Nếu có ghi chú của Creator, ghi chú ƯU TIÊN HƠN khung hiện có: khung hiện có chỉ là bản nháp cũ để sửa. Làm theo ghi chú rồi viết lại mọi ô bị ảnh hưởng cho khớp nhau. Ô Creator đã viết thì vẫn chép lại nguyên văn.

## ĐẦU RA
Trả về DUY NHẤT một khối ` + "```json" + ` theo mẫu dưới, không thêm chữ nào khác; chỉ gồm các khoá của ô cần viết (và "used_facts", "facts" khi luật dữ kiện đòi). Trong nội dung KHÔNG dùng dấu nháy kép thẳng ("): dùng dấu « » hoặc nháy đơn, và không xuống dòng giữa một ô. Mẫu đầy đủ:
{"question":"...","promise":"...","chain":[{"text":"...","facts":["F01"]}],"payoff":"...","payoff_facts":["F02"],"logline":"...","summary":"...","used_facts":["F03"],"facts":{{facts_example}}}`

// storyFactsGivenVI is {{facts_rule}} when the fact table is given.
const storyFactsGivenVI = `## LUẬT VỀ DỮ KIỆN
Chỉ trích dữ kiện bằng mã có trong bảng trên. Để "facts" là danh sách rỗng [].`

// storyFactsListVI is {{facts_rule}} when the research step was skipped and the
// model lists the facts itself.
const storyFactsListVI = `## LUẬT VỀ DỮ KIỆN
Bước Nghiên cứu đã bỏ qua nên chưa có bảng dữ kiện. Hãy tự liệt kê các dữ kiện câu chuyện cần dùng từ hiểu biết của bạn, ở trường "facts": mỗi dữ kiện {"id":"F01","claim_vi":"...","type":"fact","note":""}, mã F01, F02... theo thứ tự, "type" là một trong fact (đã được xác nhận rộng rãi), hypothesis (giả thuyết), misconception (ngộ nhận phổ biến), unknown (khoa học chưa biết). Chỉ ghi điều bạn chắc; không bịa con số, tên người hay mốc thời gian. Rồi trích các dữ kiện đó bằng mã ở chain, payoff_facts, used_facts. Các dữ kiện này sẽ mang nhãn "chưa có nguồn".`

// StoryFactsRule is {{facts_rule}} of the frame prompt: how the model treats
// facts. With listMode false the fact table is given and only cited; with
// listMode true the research step was skipped and the model lists the facts itself.
func StoryFactsRule(listMode bool) string {
	if listMode {
		return storyFactsListVI
	}
	return storyFactsGivenVI
}

// StoryRefsRule is {{refs_rule}} of the approaches prompt: how the model cites
// facts. With facts it cites only codes of the table; without (the research
// step was skipped and none was listed yet) it cites none.
func StoryRefsRule(hasFacts bool) string {
	if hasFacts {
		return "mã các dữ kiện đó, chỉ dùng mã có trong bảng dữ kiện"
	}
	return "để [] vì chưa có bảng dữ kiện (bước Nghiên cứu đã bỏ qua); nêu lý do bằng lời ở \"why\", tuyệt đối không bịa mã dữ kiện"
}

// StoryRefsExample is {{refs_example}} of the approaches prompt: the "refs" of
// the sample answer, which must not show a fact code when there is no fact to cite.
func StoryRefsExample(hasFacts bool) string {
	if hasFacts {
		return `["F01"]`
	}
	return "[]"
}

// StoryFactsExample is {{facts_example}} of the frame prompt: the "facts" of the
// sample answer, which lists a fact only when the model has to list them itself.
func StoryFactsExample(listMode bool) string {
	if listMode {
		return `[{"id":"F01","claim_vi":"...","type":"fact","note":""},{"id":"F02","claim_vi":"...","type":"hypothesis","note":""}]`
	}
	return "[]"
}
