# Functional Design Plan — Unit 4: Script Processing Service

## Unit Context
- **Stories**: A2 (phân tích script thành cấu trúc scene, FR2.2)
- **Scope**: Parse Markdown script (ADR-0011) thành danh sách `Scene`; KHÔNG gọi Content Plugin Service (ADR-0012) — chỉ trách nhiệm parse + validate cú pháp

## Execution Checklist
- [ ] Thu thập câu trả lời
- [ ] Tạo `business-logic-model.md`
- [ ] Tạo `business-rules.md`
- [ ] Tạo `domain-entities.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Business Logic — `scene_index` Assignment
Heading `## Scene N` trong cú pháp (ADR-0011) có số N do Creator viết. `scene_index` trong output là 0-based theo thứ tự xuất hiện (đã ghi chú ở `interface-contracts.md`). Số N trong heading dùng để làm gì?

A) 💡 Suggested: N trong `## Scene N` CHỈ mang tính hiển thị/đọc cho Creator (không ràng buộc phải đúng thứ tự tăng dần, không cần khớp `scene_index`) — parser hoàn toàn bỏ qua giá trị N, chỉ dùng vị trí xuất hiện thực tế của heading để gán `scene_index` (0-based). Không validate N có hợp lệ/tăng dần hay không
   - ✅ Strengths: đơn giản nhất, không tạo thêm ràng buộc cú pháp không cần thiết (Creator không cần lo đánh số đúng, có thể copy-paste/xóa scene tự do)
   - ⚠️ Trade-offs: nếu Creator đánh số sai (vd. viết "Scene 1" 2 lần), không có cảnh báo — chấp nhận được vì N chỉ là nhãn hiển thị

B) Other (please describe after [Answer]: tag below)

[Answer]: B — N trong `## Scene N` PHẢI tăng dần liên tục bắt đầu từ 1 (1, 2, 3, ... — không được nhảy số như 1, 3). Vi phạm → `ScriptSyntaxError(line_number, reason)` tại heading sai. `scene_index` (0-based, output) vẫn tính theo vị trí xuất hiện = N - 1 khi hợp lệ.

### Question 2: Business Rule — `illustration_hint` có bắt buộc không?
`interface-contracts.md` mô tả dòng `> ...` là `illustration_hint`, nhưng chưa xác nhận có bắt buộc mỗi scene phải có hay không.

A) 💡 Suggested: **Optional** — nếu scene không có dòng blockquote `>`, `illustration_hint` = `null`/rỗng, KHÔNG raise `ScriptSyntaxError`. Chỉ `narration_text` là bắt buộc không rỗng (đã xác nhận ở LLD)
   - ✅ Strengths: linh hoạt — không phải scene nào cũng cần gợi ý minh họa riêng (vd. scene giới thiệu chung chung), khớp với Content Plugin Service's domain rule (chỉ `narration_text` bắt buộc, theo Unit 2's `business-rules.md`)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3: Business Logic — Nhiều `code_snippet` trong 1 scene?
Nếu Creator viết nhiều hơn 1 code fence trong cùng 1 scene, xử lý thế nào?

A) 💡 Suggested: Chỉ lấy code fence ĐẦU TIÊN làm `code_snippet` của scene đó; code fence thứ 2 trở đi bị bỏ qua (không raise lỗi, không gộp). Scene schema (`component-methods.md`) chỉ có 1 field `code_snippet` (không phải list), nên đây là giới hạn tự nhiên của contract hiện tại
   - ✅ Strengths: đơn giản, không cần mở rộng contract; trường hợp nhiều code block trong 1 lời thoại là hiếm ở quy mô MVP (mỗi scene nên tập trung 1 đoạn code)
   - ⚠️ Trade-offs: nếu Creator thực sự cần nhiều code block trong 1 scene, phải tách thành nhiều scene — chấp nhận được, đúng tinh thần "1 scene = 1 đơn vị nội dung"

B) Other (please describe after [Answer]: tag below)

[Answer]: Raise lỗi cú pháp nếu có ≥ 2 code fence trong 1 scene — `ScriptSyntaxError(line_number, reason)` tại code fence thứ 2. Creator phải tách thành nhiều scene nếu cần nhiều đoạn code.

### Question 4: Business Scenario — Text ngoài heading đầu tiên (trước `## Scene 1`)
Nếu script có nội dung TRƯỚC heading `## Scene 1` đầu tiên (vd. tiêu đề tổng, ghi chú), xử lý thế nào?

A) 💡 Suggested: Bỏ qua hoàn toàn nội dung trước heading đầu tiên — không phải lỗi cú pháp, chỉ đơn giản không được đưa vào bất kỳ scene nào. Nếu KHÔNG có heading `## Scene N` nào trong toàn bộ script → raise `ScriptSyntaxError` (không tìm thấy scene hợp lệ, đã xác nhận ở LLD)
   - ✅ Strengths: cho phép Creator viết ghi chú tự do ở đầu file (vd. `# Tên project`) mà không lo lỗi cú pháp
   - ⚠️ Trade-offs: nội dung đó bị mất thầm lặng — chấp nhận được vì đây không phải nội dung nghiệp vụ (không map vào Scene)

B) Other (please describe after [Answer]: tag below)

[Answer]:A

### Question 5: Domain Entity — Có cần giữ lại `raw_script` gốc không?
`ParsedScript` hiện chỉ có `scenes: list[Scene]` (theo LLD). Có cần thêm field lưu lại text gốc hoặc metadata khác không?

A) 💡 Suggested: KHÔNG — `ParsedScript` chỉ chứa `scenes`, không lưu `raw_script` gốc (script gốc do GUI/Orchestrator quản lý theo Story A1, không phải trách nhiệm Script Processing Service — đã xác nhận N/A ở NFR Requirements Question 9 "Stateless")
   - ✅ Strengths: đúng ranh giới trách nhiệm, entity gọn
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A
