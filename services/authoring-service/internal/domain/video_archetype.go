package domain

import (
	"fmt"
	"strings"
)

// VideoArchetype is one kind of video the Story Architect can be told to make
// (CR-041). The prompt used to carry a single skeleton; now the kinds are rows
// the Creator can add to, and {{video_archetypes}} expands to whatever rows
// exist. System rows ship in the binary and are read-only, like system prompts.
type VideoArchetype struct {
	ID        string `json:"id"`
	Code      string `json:"code"`        // what the model prints and the Creator types: "kiểu: SO-SÁNH"
	Name      string `json:"name"`        // short label
	WhenToUse string `json:"when_to_use"` // which topics fit — the model chooses by this
	Playbook  string `json:"playbook"`    // how to assign the kind to the chosen format's beats
	// RecommendedFormatID is the video format this kind fits best; the wizard
	// suggests it when the Creator picks the kind. "" means no suggestion.
	RecommendedFormatID string `json:"recommended_format_id"`
	IsSystem            bool   `json:"is_system"`
	CreatedAt           string `json:"created_at"`
	UpdatedAt           string `json:"updated_at"`
}

// SystemVideoArchetypes are the four kinds CR-041 phase 1 shipped inside the
// prompt, plus XÂY-TẦNG and HÀNH-TRÌNH so abstract concepts and discovery
// stories stop being forced into one of the four. Their ids are stable so the seeder can refresh wording on restart.
func SystemVideoArchetypes() []VideoArchetype {
	return []VideoArchetype{
		{
			ID: "system-A", Code: "NGHỊCH-LÝ", RecommendedFormatID: "case_study_essay_8min", Name: "Nghịch lý + chuỗi ví dụ", IsSystem: true,
			WhenToUse: "chủ đề là một lỗi tư duy, thiên kiến hay hiện tượng phản trực giác, có ví dụ đời thật ở NHIỀU LĨNH VỰC THẬT SỰ KHÁC NHAU (không phải cùng một vật/hệ thống chỉ đổi bối cảnh). KHÔNG chọn kiểu này nếu chủ đề thực chất là một cơ chế/quy trình đơn nhất dạng \"X thực sự là gì\" hay \"X hoạt động thế nào\" mà mọi ví dụ đều quay lại đúng một vật — chủ đề đó hợp DẤU-VẾT hơn.",
			Playbook: `Mở bằng một nghịch lý có thật: một tình huống được ghi chép rộng rãi (sự kiện lịch sử, nghiên cứu nổi tiếng, hiện tượng ai cũng từng thấy), dựng "cách làm hiển nhiên" rồi lật bằng một đáp án nghe vô lý. Giải nghịch lý bằng chính chi tiết của tình huống rồi mới gọi tên khái niệm.
Lõi lý thuyết ngắn, bằng lời thường: định nghĩa → vì sao xảy ra → neo lại tình huống mở màn → vì sao khó nhận ra → cách xử lý.
Thân bài: 3–7 ví dụ (đúng số ví dụ THẬT SỰ khác nhau, không độn), mỗi ví dụ ở một lĩnh vực THẬT SỰ khác (đời sống, tự nhiên, kinh tế, lịch sử, khoa học, truyền thông...), không hai ví dụ liền nhau cùng lĩnh vực, sắp từ gần gũi đến tinh vi, mỗi ví dụ cho thấy MỘT góc khác của cơ chế. Khuôn Ý của một ví dụ (để nghĩ, không đọc thành lời): niềm tin phổ biến → bằng chứng bề ngoài → cú lật → phần bị che khuất → kết luận.
Gán: hook = nghịch lý, dừng ở đáp án vô lý; concrete = giải nghịch lý + gọi tên ở câu cuối; pattern = lõi lý thuyết; variation = mỗi ví dụ một beat; modern = hiện tượng được thời đại hôm nay (mạng xã hội, quảng cáo, AI...) khuếch đại ra sao; recap = thừa nhận giới hạn, lời khuyên thực tế, câu chốt tỉnh bơ quay về chủ đề.
Hợp với format có beat variation lặp được. Với format ngắn không có variation: gộp 2–3 ví dụ ngắn nhất vào pattern.`,
		},
		{
			ID: "system-B", Code: "SO-SÁNH", RecommendedFormatID: "visual_first_7min", Name: "Một họ khái niệm, nhiều cách", IsSystem: true,
			WhenToUse: "chủ đề là 2–4 công cụ/cú pháp/cách làm cùng giải một loại việc (vd for / while / do-while; list / tuple / set).",
			Playbook: `Điều cốt lõi: KHÔNG BAO GIỜ định nghĩa các cách song song với nhau. Mỗi cách xuất hiện như câu trả lời cho điểm yếu của cách trước. Ví dụ (đừng chép, áp dụng cho chủ đề của bạn): mở bằng "in từ 1 đến 10, ba cách viết đều chạy, vậy tại sao cần tới ba?" → for hợp khi biết trước số lần lặp → khi không biết trước (đợi người dùng gõ đúng mật khẩu) for trở nên gượng, nên cần while → khi thân vòng lặp bắt buộc chạy ít nhất một lần (hiện menu rồi mới hỏi) thì cần do-while.
Gán: hook = một việc duy nhất mà mọi cách đều làm được, và câu hỏi "vậy tại sao cần nhiều cách?"; concrete = cách thứ nhất chạy trên đúng việc đó, kết ở chỗ nó trở nên gượng; pattern = bộ khung chung mà cả họ cùng có (điều luôn đúng dù chọn cách nào); variation = MỖI cách còn lại một beat, mở bằng điểm yếu của cách trước; modern = bẫy hay gặp chung cho cả họ (nếu format có beat này, nếu không thì gộp vào recap); recap = MỘT câu quyết định "khi nào dùng cái nào", thay cho bảng định nghĩa.`,
		},
		{
			ID: "system-C", Code: "DẤU-VẾT", RecommendedFormatID: "visual_first_7min", Name: "Cơ chế theo dấu vết", IsSystem: true,
			WhenToUse: "chủ đề là một quy trình hay hệ thống chạy qua nhiều chặng (vd điều gì xảy ra từ lúc gõ địa chỉ web đến lúc trang hiện ra) — KỂ CẢ khi chủ đề chỉ có MỘT thế giới/vật xuyên suốt và câu hỏi dạng \"X thực sự là gì\" hay \"X hoạt động thế nào\" (vd sâu răng thực sự là gì, vì sao ngủ lại cần thiết): không cần nhiều lĩnh vực, mỗi chặng là một bước của CHÍNH cơ chế đó, không phải một ví dụ ở lĩnh vực khác.",
			Playbook: `Chọn MỘT đầu vào cụ thể (một địa chỉ, một tin nhắn, một tệp) và đi theo nó qua từng chặng; mọi beat quay lại đúng đầu vào đó.
Gán: hook = hai đầu của chuỗi (đầu vào → kết quả ai cũng thấy) và câu hỏi "giữa hai đầu chuyện gì xảy ra?"; concrete = đi theo đầu vào qua cả chuỗi ở mức thô, chỉ đường đi; pattern = gọi tên các chặng và vai trò của từng chặng; variation = mỗi chặng bất ngờ nhất đào sâu một beat; modern = chuỗi hỏng ở chặng nào thì người dùng thấy gì; recap = chạy lại cả chuỗi trong một hai câu.`,
		},
		{
			ID: "system-D", Code: "TIẾN-HOÁ", RecommendedFormatID: "visual_first_7min", Name: "Bài toán tiến hoá", IsSystem: true,
			WhenToUse: "chủ đề là một kỹ thuật sinh ra để cứu cách làm ngây thơ khỏi giới hạn của nó (vd bộ nhớ đệm, chỉ mục, tìm kiếm nhị phân).",
			Playbook: `Một bài toán có con số cụ thể chạy xuyên suốt. Mỗi cải tiến sinh ra từ điểm yếu của bản trước, và phải nói rõ nó đổi lấy cái gì.
Gán: hook = bài toán + cách ngây thơ; concrete = cho cách ngây thơ chạy và chỉ ra nó vỡ ở đâu, đo bằng số bước hoặc số lần; pattern = ý tưởng cốt lõi của giải pháp + cái giá phải trả; variation = MỖI vòng cải tiến hoặc biến thể một beat, mở bằng điểm yếu của bản trước; modern = khi nào KHÔNG nên dùng nó; recap.`,
		},
		{
			ID: "system-E", Code: "XÂY-TẦNG", RecommendedFormatID: "visual_first_7min", Name: "Xây trực giác từng tầng", IsSystem: true,
			WhenToUse: "chủ đề là một khái niệm trừu tượng (toán, vật lý, lập trình, kinh tế học...) mà người xem hiểu sai hoặc thấy khó vì thiếu nền, và cách dễ hiểu nhất là đi từ trường hợp đơn giản nhất rồi thêm dần từng lớp (vd đạo hàm, đệ quy, lãi kép, entropy, xác suất có điều kiện).",
			Playbook: `Bắt đầu từ trường hợp nhỏ nhất, đơn giản nhất mà người xem tự làm được trong đầu. Mỗi tầng chỉ thêm MỘT điều mới lên tầng trước và được kiểm bằng một ví dụ nhỏ có con số hoặc hình cụ thể. Khái niệm đầy đủ chỉ được gọi tên khi người xem đã tự đi tới nó.
Gán: hook = một câu hỏi/bài toán cụ thể mà khái niệm trả lời được, nói rõ vì sao cách nghĩ thông thường bí; concrete = tầng đầu tiên — trường hợp đơn giản nhất chạy trọn vẹn; pattern = quy luật rút ra từ tầng đầu, gọi tên khái niệm; variation = MỖI tầng tiếp theo một beat, mỗi tầng thêm một lớp khó hơn đúng một bậc; edge/modern = chỗ trực giác vừa xây bắt đầu sai, hoặc khái niệm xuất hiện ở đâu trong đời thật; recap = quay lại câu hỏi mở màn và trả lời nó bằng trực giác vừa xây.`,
		},
		{
			ID: "system-F", Code: "HÀNH-TRÌNH", RecommendedFormatID: "visual_first_7min", Name: "Câu chuyện khám phá", IsSystem: true,
			WhenToUse: "chủ đề dễ hiểu nhất khi kể lại con người đã tìm ra/giải quyết nó như thế nào — một phát minh, một khám phá khoa học, một sự kiện lịch sử, một ý tưởng thay đổi thế giới (vd vì sao ta biết Trái Đất quay, ai nghĩ ra số không, vắc-xin ra đời thế nào).",
			Playbook: `Kể theo thời gian nhưng mỗi mốc là một bước HIỂU, không phải một dòng niên biểu: người thời đó bí ở đâu, họ thử gì, cái gì sai, manh mối nào mở ra bước tiếp. Người xem hiểu khái niệm bằng cách đi lại đúng con đường đó. Chỉ dùng chi tiết lịch sử chắc chắn; không chắc thì kể định tính.
Gán: hook = vấn đề hoặc bí ẩn mà người thời đó đối mặt, đặt sao cho người xem cũng thấy bí; concrete = nỗ lực/quan sát đầu tiên và vì sao nó chưa đủ; pattern = ý tưởng then chốt gỡ được bí ẩn, gọi tên khái niệm; variation = MỖI mốc tiếp theo một beat — một bằng chứng, một phản bác, một bước hoàn thiện; edge/modern = khái niệm đó đang sống trong đời ta hôm nay thế nào hoặc câu hỏi nào vẫn còn mở; recap = một câu cho thấy vì sao con đường đó dẫn tới hiểu biết ta có bây giờ.`,
		},
	}
}

// BuildVideoArchetypesSection renders {{video_archetypes}}: the menu the model
// chooses from, then one playbook per kind. With no rows it says so plainly —
// a prompt that silently lost its kinds would still look valid.
func BuildVideoArchetypesSection(list []VideoArchetype) string {
	if len(list) == 0 {
		return "(chưa có kiểu video nào được cấu hình — dùng khuôn chung của BẢN SẮC KÊNH và in KIỂU VIDEO: không có)"
	}
	var menu, books strings.Builder
	for _, a := range list {
		fmt.Fprintf(&menu, "- %s — %s: %s\n", a.Code, strings.ToUpper(a.Name), strings.TrimSpace(a.WhenToUse))
		fmt.Fprintf(&books, "\n### Playbook %s — %s\n%s\n", a.Code, a.Name, strings.TrimSpace(a.Playbook))
	}
	return strings.TrimRight(menu.String(), "\n") + "\n" + books.String()
}
