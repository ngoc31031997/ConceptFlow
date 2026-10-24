# Ý tưởng: API của bước 1 (`/v1`)

Bước Ý tưởng (authoring-service) biến đoạn Creator gõ thành ba trường có cấu trúc, cho AI đánh giá và để Creator duyệt. Mọi đường dẫn nằm dưới `/v1` (ADR-0008) qua api-gateway; chỉ thêm trường thì giữ `/v1`. Mã tương ứng: `services/authoring-service/internal/domain/project_idea.go`, `internal/application/project_idea.go`, `internal/adapters/http/review_units.go` (nhóm route `/idea`), `internal/adapters/postgres/project_idea_repository.go`.

Idea không dùng khung đơn vị duyệt chung (`/units/...`): nó có API và bảng riêng.

## Hợp đồng `idea.v1`: đầu ra cho bước sau

| Trường | Kiểu | Ghi chú |
|---|---|---|
| `central_question` | chuỗi | câu hỏi trung tâm |
| `curiosity_points` | danh sách chuỗi | 1 đến 10 điểm, mỗi điểm tối đa 300 ký tự, không trống, không lặp |
| `scope` | chuỗi | phạm vi |

Ba trường viết bằng ngôn ngữ nội dung của project. Bước sau chỉ đọc **bản đã duyệt**: hàm `ProjectIdeaUseCase.ApprovedIdea(projectID)` trả `(idea, ok)`, `ok` là false khi chưa có ý tưởng hoặc ý tưởng đang chờ duyệt. Chưa có route HTTP nội bộ cho việc này; bước nối vào (Creative brief) sẽ thêm khi cần.

## Dữ liệu và nơi lưu

| Dữ liệu | Nơi lưu |
|---|---|
| Đoạn Creator gõ (`subject`) và ngôn ngữ nội dung (`language`) | bảng `projects` của orchestrator, hai cột `subject` và `language`; ghi bằng `POST /v1/projects` và `PATCH /v1/projects/{id}/topic` |
| Ba trường của ý tưởng, trường do Creator viết, trạng thái, số lần làm lại, ngôn ngữ và dấu vân tay của đoạn gõ lúc AI viết | bảng `project_idea` của authoring-service, một dòng cho mỗi project |
| Kết quả AI đánh giá | bảng `project_idea_review`, tách khỏi `project_idea` |

authoring-service không đọc bảng của orchestrator; nó gọi `GET /internal/v1/projects/{id}/idea-input` (orchestrator) và nhận `{"subject": "…", "language": "vi"|"en"}`, `404` khi project không có. Hai cột này nằm cạnh `projects.voice_language` và `project_authoring.topic/.language`, vốn vẫn được ghi như trước; không bộ nào đọc `project_authoring` thay cho chúng ở bước này.

## Làm sạch chữ Creator gõ

AI trả lời bằng JSON và hay chép dấu nháy kép thẳng hoặc dấu gạch chéo ngược từ đầu vào vào chuỗi JSON, làm câu trả lời không đọc được. Vì vậy mọi chữ Creator gõ ở bước này được làm sạch ở API (và cả ở web-gui khi gõ):

- `"` thành nháy cong: `“` ở đầu chuỗi hoặc sau khoảng trắng, dấu mở, gạch ngang; `”` ở chỗ còn lại.
- `\` thành `/`; ký tự điều khiển bỏ đi (giữ xuống dòng và tab); chuỗi được cắt khoảng trắng hai đầu.
- orchestrator làm sạch `topic` của `POST /v1/projects` và `PATCH /v1/projects/{id}/topic` trước khi lưu vào `subject` (và `project_authoring`); `400` nếu chữ rỗng sau khi làm sạch.
- authoring-service làm sạch các ô của `PUT /v1/projects/{id}/idea`, `note` của `generate`, và đoạn đọc từ orchestrator (dữ liệu lưu trước khi có luật này) trước khi đưa vào prompt. `input_changed` so sánh chữ đã làm sạch nên không báo sai.

## Đường dẫn

| Phương thức | Đường dẫn | Việc |
|---|---|---|
| GET | `/v1/projects/{id}/idea` | Toàn màn: `idea` (null khi chưa có), `subject`, `language`, `input_changed`, `review` (null khi chưa có), `review_stale` |
| POST | `/v1/projects/{id}/idea/generate` | AI viết ý tưởng, body `{"note":""}` tuỳ chọn; giữ trường Creator đã viết; ý tưởng về chờ duyệt; trả ý tưởng |
| PUT | `/v1/projects/{id}/idea` | Sửa tay, body gồm trường nào đổi trong `central_question`, `curiosity_points` (danh sách), `scope`; trường đổi thành "Bạn viết", trường xoá trắng trả về cho AI; mọi thay đổi đưa ý tưởng về chờ duyệt |
| POST | `/v1/projects/{id}/idea/approve` | Creator duyệt; cần đủ ba trường |
| POST | `/v1/projects/{id}/idea/unapprove` | Bỏ duyệt |
| POST | `/v1/projects/{id}/idea/review` | AI đọc ý tưởng; trả đánh giá |

Hai lượt AI (`generate`, `review`) nhận header `X-Operation-Id` và ghi nhận dưới kind `idea` và `idea_review`, để trang khác theo dõi và huỷ. Chúng đi qua `authoringAi` (timeout dài) ở api-gateway.

`input_changed` là true khi đoạn gõ (hoặc ngôn ngữ) lúc này khác lúc AI viết ý tưởng. `review_stale` là true khi ý tưởng đã đổi sau đánh giá.

## Đánh giá của AI

Ba mục `clarity` (độ rõ), `researchability` (khả năng nghiên cứu), `scope_fit` (phạm vi), mỗi mục `{"rating": "good"|"fair"|"weak", "comment": "…"}`, cùng `unclear[]` (chỗ chưa rõ), `suggestions[]` (gợi ý), `idea_hash`, `created_at`. Đánh giá chỉ để Creator tham khảo: nó không đổi ý tưởng, không đổi trạng thái duyệt và không chặn duyệt.

## Model AI và prompt

Cả viết và đánh giá chạy trên model Creator chọn cho khoá `brief` (`PUT /v1/projects/{id}/authoring/models`). Prompt là hai vai `project_idea` và `project_idea_review` trong thư viện prompt (gieo nếu chưa có; Creator sửa được). Prompt `project_idea` nhận `{{topic}}` (đoạn Creator gõ), `{{idea_language}}`, `{{current_idea}}`, `{{kept_fields}}`, `{{redo_note}}`; lần làm lại dùng cùng prompt, kèm ý tưởng cũ và ghi chú của Creator. Vai `idea` cũ vẫn nằm trong thư viện nhưng bước này không dùng.

## Mã lỗi

| Mã | Khi |
|---|---|
| `400` | Duyệt khi còn trường trống, danh sách quá 10 điểm, body hỏng, project chưa có đoạn gõ |
| `404` | Duyệt, bỏ duyệt hoặc đánh giá khi chưa có ý tưởng |
| `409` | Project không còn là bản nháp |
| `502` | Câu trả lời của AI không đọc được hoặc thiếu trường; không lưu gì |
| `503` | Chưa cấu hình AI (Creator tự viết ý tưởng) |
