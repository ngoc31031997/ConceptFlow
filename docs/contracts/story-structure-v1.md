# Cấu trúc câu chuyện: `story-structure.v1` (bước Câu chuyện)

Bước Câu chuyện (authoring-service) viết cấu trúc câu chuyện từ brief đã duyệt và các dữ kiện đang giữ của bước Nghiên cứu. Mã tương ứng: `services/authoring-service/internal/domain/project_story.go` (kiểu, kiểm tra, đọc câu trả lời của AI), `internal/application/story_desk.go`, `internal/adapters/http/story.go`, `internal/adapters/postgres/project_story_repository.go`.

## Nơi lưu và nguồn đầu vào

- Cấu trúc lưu trong bảng `project_story`, một dòng một dự án, khoá `project_id`. Cột: `central_question`, `core_answer`, `narrative_arc` (JSON), `creator_fields` (JSON, các ô Creator tự viết, lần viết lại không ghi đè), `brief_hash`, `research_hash` (băm của brief đã duyệt và của các dữ kiện đang giữ lúc viết), `status` (`pending` hoặc `approved`), `created_at`, `updated_at`.
- Đầu vào: brief đã duyệt đọc từ `project_brief` (`{{brief}}`) và các dòng `keep = true` của `project_research` (`{{research}}`). Bước mở khi brief đã duyệt và bước Nghiên cứu đã duyệt hoặc bỏ qua; chưa thì viết câu chuyện trả `409`. Bước Nghiên cứu bỏ qua thì không có dữ kiện: AI chỉ nói định tính và không trích mã dữ kiện.
- Chỉ áp dụng cho dự án mới: câu chuyện cũ dạng đơn vị `story`, `story_field`, `story_approach` không được chuyển sang.

## Các trường

| Trường | Kiểu | Ý nghĩa |
|---|---|---|
| `central_question` | chuỗi, bắt buộc | Câu hỏi trung tâm của video |
| `core_answer` | chuỗi, bắt buộc | Câu trả lời cốt lõi video sẽ chứng minh |
| `narrative_arc` | mảng nhịp, bắt buộc | Các nhịp theo thứ tự người xem gặp; số nhịp do AI quyết theo dữ kiện, từ 3 đến 12 |

Một nhịp: `{"beat": "Hook", "text": "…", "facts": ["F01"]}`. `beat` là tên vai trò của nhịp (tối đa 60 ký tự), `text` là nhịp nói gì, `facts` là mã dữ kiện trong `project_research` mà nhịp dựa vào (có thể rỗng). Mã không có trong bảng dữ kiện đang giữ thì bị từ chối, cả khi AI viết lẫn khi Creator sửa tay.

## Điều kiện bước xong

Duyệt chỉ cần máy kiểm qua: đủ ba trường, mạch có từ 3 đến 12 nhịp, mỗi nhịp có tên và nội dung. Câu chuyện có đường dây rõ ràng, không chỉ là danh sách thông tin, do Creator xác nhận khi duyệt. Sửa bất kỳ ô nào của câu chuyện đã duyệt đưa nó về `pending`.

## HTTP (qua api-gateway, `/v1/projects/{id}/story…`)

| Route | Việc |
|---|---|
| `GET /story` | Cả màn: `story` (`null` khi chưa có; gồm `central_question`, `core_answer`, `narrative_arc`, `creator_fields`, `status`, `updated_at`), `facts` (dữ kiện đang giữ kèm `must`, `cited`, `link`), `brief` (brief đã duyệt hoặc `null`), `brief_approved`, `research_resolved`, `research_skipped`, `brief_stale`, `research_stale` (brief hoặc dữ kiện đã đổi so với lúc câu chuyện được viết) |
| `POST /story/generate` `{note}` | AI viết câu chuyện, giữ các ô Creator đã viết (theo dõi bằng `X-Operation-Id`); `409` khi bước chưa mở, `400` khi cả ba ô đều của Creator, `502` khi trả lời không đọc được, `503` khi chưa có AI |
| `PUT /story` `{central_question?, core_answer?, narrative_arc?}` | Sửa tay, chỉ trường có gửi; ô đổi giá trị thành của Creator, ô xoá trắng trả về cho AI; tạo câu chuyện nếu chưa có; `400` khi mã dữ kiện lạ hoặc quá 12 nhịp |
| `POST /story/approve`, `POST /story/unapprove` | Duyệt (`400` khi chưa qua máy kiểm, `404` khi chưa có câu chuyện), bỏ duyệt |

## Bước đọc nó

- Bước Beat sheet đọc câu chuyện đã duyệt qua một lớp chuyển đổi cho tới khi bước đó có hợp đồng mới: `question` ← `central_question`, `chain` ← các nhịp (một nhịp một dòng, kèm mã dữ kiện), `payoff` ← `core_answer`. Beat tham chiếu câu chuyện bằng `story:main` trong `input_ref`; kho đơn vị tính băm của tham chiếu này từ `project_story`, nên beat viết từ câu chuyện đã đổi hiện là cũ.
- Bước Nghiên cứu hỏi bước này nhịp nào đang trích một dữ kiện trước khi Creator bỏ hoặc sửa dữ kiện đó.

## Prompt

Vai `story_frame` (sửa được ở Cài đặt prompt) nhận `{{brief}}`, `{{research}}`, `{{brief_language}}`, `{{current_story}}`, `{{kept_fields}}`, `{{redo_note}}` và trả một khối JSON `{"central_question", "core_answer", "narrative_arc":[{"beat","text","facts"}]}`. Vai `story_approaches` đã bỏ cùng bước chọn góc tiếp cận.
