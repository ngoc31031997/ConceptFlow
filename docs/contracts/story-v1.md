# Câu chuyện: API của bước 4 (`/v1`)

Bước Câu chuyện (authoring-service) gợi ý góc tiếp cận, giữ sáu ô của khung câu chuyện và danh sách dữ kiện được dùng. Mọi đường dẫn nằm dưới `/v1` (ADR-0008) qua api-gateway; chỉ thêm trường thì giữ `/v1`. Mã tương ứng: `services/authoring-service/internal/domain/story.go` (đọc câu trả lời AI, dạng gõ tay), `internal/application/story_*.go`, `internal/adapters/http/story.go`.

## Đơn vị duyệt

| Kind | Key | Nội dung (`content`) |
|---|---|---|
| `story` | `main` | `{approach_key, used_facts[]}`; duyệt cổng này là duyệt cả câu chuyện |
| `story_field` | `question` `promise` `chain` `payoff` `logline` `summary` | `{text, facts[]}`; `facts` suy ra từ `text` |
| `story_approach` | `A1`, `A2`… | `{code, name, angle, why, refs[], open, is_new, when_to_use, playbook, archetype_id}` |

`story_field` và `story_approach` dựng từ `brief:main` (`input_ref`), nên `stale` khi brief đổi. Sửa, làm lại, duyệt qua API đơn vị chung: `PUT /v1/projects/{id}/units/story_field/{key}` với `{"fields":{"text":"…"}}`, `POST …/redo` (AI viết riêng ô đó), `POST /v1/projects/{id}/units/story/main/approve`.

Dạng gõ tay của `chain` (mỗi bước một dòng, tối đa 12) và `payoff` (một dòng): cuối dòng có thể là `[F01, F02]`, mã dữ kiện đang giữ; mã khác bị từ chối `400`.

## Đường dẫn

| Phương thức | Đường dẫn | Việc |
|---|---|---|
| GET | `/v1/projects/{id}/story` | Toàn bộ màn: `status`, `approach_key`, `fields[]`, `approaches[]`, `facts[]`, `brief_approved`, `research_resolved`, `research_skipped` |
| POST | `/v1/projects/{id}/story/approaches/generate` | AI gợi ý góc, body `{"note":""}` tuỳ chọn; trả `{approaches[]}`; góc đang chọn được giữ |
| PUT | `/v1/projects/{id}/story/approach` | Chọn góc, body `{"key":"A2"}`; `""` để AI tự chọn; `204` |
| POST | `/v1/projects/{id}/story/approaches/{key}/archetype` | Lưu kiểu mới vào bảng Kiểu video |
| POST | `/v1/projects/{id}/story/generate` | AI viết các ô chưa phải của Creator, body `{"note":""}` tuỳ chọn; trả state |
| PUT | `/v1/projects/{id}/story/facts/{fact_id}` | Bật/tắt dùng dữ kiện, body `{"use":true}`; dữ kiện bắt buộc nói không tắt được; `204` |

Hai lượt AI nhận header `X-Operation-Id` và ghi nhận dưới kind `story_approach` và `story`, để trang khác theo dõi và huỷ.

## Model AI của bước

Hai lượt AI và "Làm lại" từng ô chạy trên model Creator chọn cho bước, lưu theo project ở khoá `story_frame` của `PUT /v1/projects/{id}/authoring/models` (đọc lại ở `story_frame_model` của `GET …/authoring`); rỗng là model mặc định của máy chủ. Khoá này tách khỏi `story` của bước Kịch bản.

## Mã lỗi

| Mã | Khi |
|---|---|
| `400` | Ô sửa trích mã dữ kiện không có, chuỗi quá 12 bước, duyệt khi còn ô trống, cả sáu ô đều do Creator viết |
| `409` | Brief chưa duyệt, hoặc Nghiên cứu chưa duyệt/bỏ qua, hoặc project không còn là bản nháp |
| `502` | Câu trả lời của AI không đọc được (khối JSON hỏng, thiếu ô, trích dữ kiện lạ); không lưu gì |
| `503` | Chưa cấu hình AI |

## Khi bỏ qua Nghiên cứu

Chưa có dữ kiện nào thì lượt `generate` bắt AI tự liệt kê dữ kiện từ hiểu biết của nó; mỗi dữ kiện thành một đơn vị `fact` (`F01`…) có `link.state = "none"` (hiện "chưa có nguồn"). Mã AI liệt kê trùng mã đã có (kể cả dữ kiện đã bỏ) thì cả lượt bị từ chối.
