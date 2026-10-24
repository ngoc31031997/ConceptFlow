# Creative brief: API của bước 2 (`/v1`)

Bước Creative brief (authoring-service) đọc ý tưởng đã duyệt ở bước 1, nhờ AI viết năm trường định hướng sáng tạo, cho AI đánh giá và để Creator duyệt. Mọi đường dẫn nằm dưới `/v1` (ADR-0008) qua api-gateway; chỉ thêm trường thì giữ `/v1`. Mã tương ứng: `services/authoring-service/internal/domain/project_brief.go`, `internal/application/project_brief.go`, `internal/adapters/http/project_brief.go`, `internal/adapters/postgres/project_brief_repository.go`.

Brief không dùng khung đơn vị duyệt chung (`/units/...`): nó có API và bảng riêng, như Idea.

## Hợp đồng `creative-brief.v1`: đầu ra cho bước sau

| Trường | Kiểu | Ghi chú |
|---|---|---|
| `audience` | chuỗi | khán giả mục tiêu, không trống |
| `duration_seconds` | số nguyên | thời lượng mục tiêu, tính bằng giây. Video dài: 121 đến 1800; video ngắn: 60 đến 120. Màn hình hiện dạng `m:ss` hoặc `h:mm:ss` |
| `tone` | chuỗi | giọng điệu, không trống |
| `learning_objectives` | danh sách chuỗi | 1 đến 6 dòng, mỗi dòng tối đa 300 ký tự, không trống, không lặp |
| `avoid` | danh sách chuỗi | 1 đến 8 dòng, mỗi dòng tối đa 300 ký tự, không trống, không lặp. Là cách trình bày sai hay dễ gây hiểu lầm mà video phải tránh, không phải danh sách chủ đề bỏ qua |

Các trường chữ viết bằng ngôn ngữ nội dung của project. Bước sau chỉ đọc **bản đã duyệt**: hàm `ProjectBriefUseCase.ApprovedBrief(projectID)` trả `(brief, ok)`, `ok` là false khi chưa có brief hoặc brief đang chờ duyệt. Chưa có route HTTP nội bộ cho việc này.

Dạng `.md` tương ứng chỉ để đọc, không lưu thành file:

```
# Creative Brief
Audience: …
Duration: 4:30
Tone: …
Learning objectives:
1. …
Avoid:
- …
```

## Đầu vào

- Ý tưởng đã duyệt (`idea.v1`), đọc qua `ProjectIdeaUseCase.ApprovedIdea`: câu hỏi trung tâm, danh sách điểm gây tò mò, phạm vi.
- Đoạn Creator gõ (chủ đề) và ngôn ngữ nội dung, đọc qua `GET /internal/v1/projects/{id}/idea-input` của orchestrator như bước 1.
- Dạng video của project (dài hoặc ngắn), quyết định khoảng `duration_seconds`.

Chưa có ý tưởng đã duyệt thì AI không viết brief (`409`).

## Dữ liệu và nơi lưu

| Dữ liệu | Nơi lưu |
|---|---|
| Năm trường của brief, trường do Creator viết, trạng thái, số lần làm lại, ngôn ngữ và dấu vân tay của ý tưởng đã duyệt lúc AI viết | bảng `project_brief` của authoring-service, một dòng cho mỗi project |
| Kết quả AI đánh giá | bảng `project_brief_review`, tách khỏi `project_brief` |

Cả hai bảng bị xoá cùng project.

## Đường dẫn

| Phương thức | Đường dẫn | Việc |
|---|---|---|
| GET | `/v1/projects/{id}/creative-brief` | Toàn màn: `brief` (null khi chưa có), `idea` (ý tưởng đã duyệt, null khi chưa duyệt), `subject`, `language`, `video_mode`, `min_duration_seconds`, `max_duration_seconds`, `idea_changed`, `review` (null khi chưa có), `review_stale` |
| POST | `/v1/projects/{id}/creative-brief/generate` | AI viết brief, body `{"note":""}` tuỳ chọn; giữ trường Creator đã viết; brief về chờ duyệt; trả brief |
| PUT | `/v1/projects/{id}/creative-brief` | Sửa tay, body gồm trường nào đổi trong `audience`, `duration_seconds`, `tone`, `learning_objectives`, `avoid`; trường đổi thành "Bạn viết", trường xoá trắng trả về cho AI; mọi thay đổi đưa brief về chờ duyệt |
| POST | `/v1/projects/{id}/creative-brief/approve` | Creator duyệt; cần đủ năm trường và thời lượng trong khoảng của dạng video |
| POST | `/v1/projects/{id}/creative-brief/unapprove` | Bỏ duyệt |
| POST | `/v1/projects/{id}/creative-brief/review` | AI đọc brief; trả đánh giá |

Hai lượt AI (`generate`, `review`) nhận header `X-Operation-Id` và ghi nhận dưới kind `brief` và `brief_review`. Chúng đi qua `authoringAi` (timeout dài) ở api-gateway.

`idea_changed` là true khi ý tưởng đã duyệt lúc này khác ý tưởng lúc AI viết brief; brief không tự làm lại. `review_stale` là true khi brief đã đổi sau đánh giá.

## Đánh giá của AI

Bốn mục `audience_fit` (khán giả), `duration_fit` (thời lượng so với phạm vi ý tưởng), `objectives` (mục tiêu học tập), `avoid_list` (danh sách cần tránh), mỗi mục `{"rating": "good"|"fair"|"weak", "comment": "…"}`, cùng `unclear[]`, `suggestions[]`, `brief_hash`, `created_at`. Đánh giá chỉ để Creator tham khảo: nó không đổi brief, không đổi trạng thái duyệt và không chặn duyệt. Máy kiểm schema và các giới hạn ở bảng trên mới là điều kiện bắt buộc để duyệt.

## Model AI và prompt

Viết và đánh giá chạy trên model Creator chọn cho khoá `brief` (`PUT /v1/projects/{id}/authoring/models`). Prompt là hai vai `project_brief` và `project_brief_review` trong thư viện prompt (gieo nếu chưa có; Creator sửa được). Prompt `project_brief` nhận `{{topic}}`, `{{idea}}`, `{{video_form}}`, `{{duration_range}}`, `{{brief_language}}`, `{{current_brief}}`, `{{kept_fields}}`, `{{redo_note}}`. Vai `creative_brief` cũ vẫn nằm trong thư viện nhưng bước này không dùng.

## Mã lỗi

| Mã | Khi |
|---|---|
| `400` | Duyệt khi còn trường trống hoặc thời lượng ngoài khoảng, danh sách quá dài, body hỏng |
| `404` | Duyệt, bỏ duyệt hoặc đánh giá khi chưa có brief |
| `409` | Project không còn là bản nháp, hoặc ý tưởng ở bước 1 chưa duyệt |
| `502` | Câu trả lời của AI không đọc được, thiếu trường hoặc thời lượng ngoài khoảng; không lưu gì |
| `503` | Chưa cấu hình AI (Creator tự viết brief) |
