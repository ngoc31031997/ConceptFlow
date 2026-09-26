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
	Code      string `json:"code"`        // what the model prints and the Creator types: "kiểu: B"
	Name      string `json:"name"`        // short label
	WhenToUse string `json:"when_to_use"` // which topics fit — the model chooses by this
	Playbook  string `json:"playbook"`    // how to assign the kind to the chosen format's beats
	IsSystem  bool   `json:"is_system"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// SystemVideoArchetypes are the four kinds CR-041 phase 1 shipped inside the
// prompt. Their ids are stable so the seeder can refresh wording on restart.
func SystemVideoArchetypes() []VideoArchetype {
	return []VideoArchetype{
		{
			ID: "system-A", Code: "A", Name: "Nghịch lý + chuỗi ví dụ", IsSystem: true,
			WhenToUse: "chủ đề là một lỗi tư duy, thiên kiến hay hiện tượng phản trực giác, có ví dụ đời thật ở nhiều lĩnh vực.",
			Playbook: `Xem "BẢN SẮC KÊNH": mở bằng nghịch lý có thật → giải trước, gọi tên sau → lõi ngắn → chuỗi 5–7 ví dụ đa lĩnh vực.
Gán: hook = nghịch lý; concrete = giải nghịch lý + gọi tên; pattern = lõi lý thuyết; variation = mỗi ví dụ một beat; modern = hiện tượng hôm nay; recap.
Hợp với format có beat variation lặp được. Với format ngắn không có variation: gộp 2–3 ví dụ ngắn nhất vào pattern.`,
		},
		{
			ID: "system-B", Code: "B", Name: "Một họ khái niệm, nhiều cách", IsSystem: true,
			WhenToUse: "chủ đề là 2–4 công cụ/cú pháp/cách làm cùng giải một loại việc (vd for / while / do-while; list / tuple / set).",
			Playbook: `Điều cốt lõi: KHÔNG BAO GIỜ định nghĩa các cách song song với nhau. Mỗi cách xuất hiện như câu trả lời cho điểm yếu của cách trước. Ví dụ (đừng chép, áp dụng cho chủ đề của bạn): mở bằng "in từ 1 đến 10, ba cách viết đều chạy, vậy tại sao cần tới ba?" → for hợp khi biết trước số lần lặp → khi không biết trước (đợi người dùng gõ đúng mật khẩu) for trở nên gượng, nên cần while → khi thân vòng lặp bắt buộc chạy ít nhất một lần (hiện menu rồi mới hỏi) thì cần do-while.
Gán: hook = một việc duy nhất mà mọi cách đều làm được, và câu hỏi "vậy tại sao cần nhiều cách?"; concrete = cách thứ nhất chạy trên đúng việc đó, kết ở chỗ nó trở nên gượng; pattern = bộ khung chung mà cả họ cùng có (điều luôn đúng dù chọn cách nào); variation = MỖI cách còn lại một beat, mở bằng điểm yếu của cách trước; modern = bẫy hay gặp chung cho cả họ (nếu format có beat này, nếu không thì gộp vào recap); recap = MỘT câu quyết định "khi nào dùng cái nào", thay cho bảng định nghĩa.`,
		},
		{
			ID: "system-C", Code: "C", Name: "Cơ chế theo dấu vết", IsSystem: true,
			WhenToUse: "chủ đề là một quy trình hay hệ thống chạy qua nhiều chặng (vd điều gì xảy ra từ lúc gõ địa chỉ web đến lúc trang hiện ra).",
			Playbook: `Chọn MỘT đầu vào cụ thể (một địa chỉ, một tin nhắn, một tệp) và đi theo nó qua từng chặng; mọi beat quay lại đúng đầu vào đó.
Gán: hook = hai đầu của chuỗi (đầu vào → kết quả ai cũng thấy) và câu hỏi "giữa hai đầu chuyện gì xảy ra?"; concrete = đi theo đầu vào qua cả chuỗi ở mức thô, chỉ đường đi; pattern = gọi tên các chặng và vai trò của từng chặng; variation = mỗi chặng bất ngờ nhất đào sâu một beat; modern = chuỗi hỏng ở chặng nào thì người dùng thấy gì; recap = chạy lại cả chuỗi trong một hai câu.`,
		},
		{
			ID: "system-D", Code: "D", Name: "Bài toán tiến hoá", IsSystem: true,
			WhenToUse: "chủ đề là một kỹ thuật sinh ra để cứu cách làm ngây thơ khỏi giới hạn của nó (vd bộ nhớ đệm, chỉ mục, tìm kiếm nhị phân).",
			Playbook: `Một bài toán có con số cụ thể chạy xuyên suốt. Mỗi cải tiến sinh ra từ điểm yếu của bản trước, và phải nói rõ nó đổi lấy cái gì.
Gán: hook = bài toán + cách ngây thơ; concrete = cho cách ngây thơ chạy và chỉ ra nó vỡ ở đâu, đo bằng số bước hoặc số lần; pattern = ý tưởng cốt lõi của giải pháp + cái giá phải trả; variation = MỖI vòng cải tiến hoặc biến thể một beat, mở bằng điểm yếu của bản trước; modern = khi nào KHÔNG nên dùng nó; recap.`,
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
