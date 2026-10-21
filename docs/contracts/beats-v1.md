# Beat sheet: API của bước 5 (`/v1`)

Bước Beat sheet (authoring-service) chia câu chuyện đã duyệt thành các beat, giữ thứ tự và thời lượng của chúng. Mọi đường dẫn nằm dưới `/v1` (ADR-0008) qua api-gateway; chỉ thêm trường thì giữ `/v1`. Mã tương ứng: `services/authoring-service/internal/domain/beat.go` (đọc câu trả lời AI, kiểm cấu trúc), `internal/application/beat_*.go`, `internal/adapters/http/beat.go`.

## Đơn vị duyệt

| Kind | Key | Nội dung (`content`) |
|---|---|---|
| `beat` | `b1`, `b2`… (khoá bền, không dùng lại; đổi thứ tự không đổi khoá) | `{fn, title, goal, content, facts[], dur, next}` |
| `beat_sheet` | `main` | `{order[], next_seq}`: chỉ giữ thứ tự beat và số của khoá kế tiếp; không qua API đơn vị chung |

`fn` là một trong `Hook`, `Đặt vấn đề`, `Giải thích cơ chế`, `Cao trào`, `Mở rộng hệ quả`, `Payoff`; `dur` là số giây (5 đến 600). Bước xong khi có beat và mọi beat đã duyệt (`status` của màn).

`beat` dựng từ `story:main` (`input_ref`). Sửa, làm lại, duyệt qua API đơn vị chung: `PUT /v1/projects/{id}/units/beat/{key}` với `{"fields":{…}}` gồm một hay nhiều trong `title`, `goal`, `content`, `next` (chữ), `fn`, `dur` (số giây, chuỗi số), `facts` (`"F01, F02"`, mã dữ kiện đang giữ); `POST …/redo` (AI viết riêng beat đó, đọc các beat còn lại; beat thôi là của Creator); `POST …/approve` (cần có mục tiêu). Trường nào đổi giá trị thì thành của Creator (`mine`) và AI giữ khi viết lại cả bảng; beat sửa xong về chờ duyệt.

## Đường dẫn

| Phương thức | Đường dẫn | Việc |
|---|---|---|
| GET | `/v1/projects/{id}/beats` | Toàn màn: `status`, `story_approved`, `beats[]` (kèm `status`, `mine[]`, `stale`, `only_must[]`: dữ kiện bắt buộc chỉ beat này giữ), `facts[]` (dữ kiện đang giữ), `checks[]` (`{ok,text}`: tổng thời lượng, hook ≤ 20 s, dữ kiện bắt buộc, mục tiêu, beat cuối là Payoff; chỉ cảnh báo), `total_seconds`, `target_label` |
| POST | `/v1/projects/{id}/beats/generate` | AI viết cả bảng, body `{"note":""}` tuỳ chọn; trả state. Beat Creator đã viết phần nào giữ khoá và phần đó; beat khác bị thay; mọi beat về chờ duyệt |
| POST | `/v1/projects/{id}/beats` | Thêm beat sau `after` (`""` khi bảng trống), body `{"after":"b1","blank":false,"note":""}`; `blank:true` thêm beat trống, không cần AI; trả state |
| PUT | `/v1/projects/{id}/beats/order` | Đổi thứ tự, body `{"order":["b2","b1"]}` nêu mỗi beat đúng một lần; trả state |
| DELETE | `/v1/projects/{id}/beats/{key}` | Xoá beat; trả state |

Lượt AI nhận header `X-Operation-Id` và ghi nhận dưới kind `beat_sheet` (cả bảng) và `beat` (thêm, làm lại một beat), để trang khác theo dõi và huỷ.

## Model AI của bước

Các lượt AI chạy trên model Creator chọn cho bước, lưu theo project ở khoá `beat_sheet` của `PUT /v1/projects/{id}/authoring/models` (đọc lại ở `beat_sheet_model` của `GET …/authoring`); rỗng là model mặc định của máy chủ. Mặc định theo bước đặt ở Cài đặt · Model AI, khoá `beat_sheet`.

## Mã lỗi

| Mã | Khi |
|---|---|
| `400` | Sửa trường lạ, chức năng lạ, thời lượng ngoài khoảng, mã dữ kiện không có, tên trống; duyệt beat không mục tiêu; thứ tự không nêu đủ beat; bảng đã đủ 20 beat; thêm beat không nêu `after` khi bảng đã có beat |
| `404` | Beat không có |
| `409` | Câu chuyện chưa duyệt, hoặc project không còn là bản nháp |
| `502` | Câu trả lời của AI không đọc được (JSON hỏng, thiếu mục tiêu hay nội dung, chức năng lạ, thời lượng ngoài khoảng, trích dữ kiện lạ, bỏ mất beat Creator đã viết); không lưu gì |
| `503` | Chưa cấu hình AI |
