# Beat sheet: API của bước 5 (`/v1`)

Bước Beat sheet (authoring-service) chia câu chuyện đã duyệt thành các beat, giữ thứ tự và thời lượng của chúng, theo hai giai đoạn: **bộ khung** (code dựng từ câu chuyện, không gọi AI: chức năng và dữ kiện của từng beat, Creator duyệt) rồi **nội dung** (AI viết từng beat một, theo thứ tự, từ bộ khung đã duyệt). Mọi đường dẫn nằm dưới `/v1` (ADR-0008) qua api-gateway; chỉ thêm trường thì giữ `/v1`. Mã tương ứng: `services/authoring-service/internal/domain/beat.go` (đọc câu trả lời AI, kiểm cấu trúc), `internal/application/beat_*.go`, `internal/adapters/http/beat.go`.

## Đơn vị duyệt

| Kind | Key | Nội dung (`content`) |
|---|---|---|
| `beat` | `b1`, `b2`… (khoá bền, không dùng lại; đổi thứ tự không đổi khoá) | `{fn, title, goal, content, facts[], dur, next, step}`; `step` là dòng của câu chuyện mà beat dựng từ đó (rỗng với beat Creator tự thêm); beat của bộ khung chưa có `goal` và `content` |
| `beat_sheet` | `main` | `{order[], next_seq, skeleton_approved, rebalanced}`: thứ tự beat, số của khoá kế tiếp, bộ khung đã duyệt chưa, và số beat bị cân lại thời lượng ở lần viết nội dung gần nhất; không qua API đơn vị chung |

`fn` là một trong `Hook`, `Đặt vấn đề`, `Giải thích cơ chế`, `Cao trào`, `Mở rộng hệ quả`, `Payoff`; `dur` là số giây (5 đến 600). Bước xong khi có beat và mọi beat đã duyệt (`status` của màn).

`beat` dựng từ `story:main` (`input_ref`). Sửa, làm lại, duyệt qua API đơn vị chung: `PUT /v1/projects/{id}/units/beat/{key}` với `{"fields":{…}}` gồm một hay nhiều trong `title`, `goal`, `content`, `next` (chữ), `fn`, `dur` (số giây, chuỗi số), `facts` (`"F01, F02"`, mã dữ kiện đang giữ); `POST …/redo` (AI viết riêng beat đó, đọc các beat còn lại và ngân sách thời lượng; dùng cả cho beat của bộ khung chưa có nội dung; chức năng và dữ kiện giữ nguyên, phần chữ Creator đã viết được thả ra, thời lượng Creator đã đặt giữ; cần bộ khung đã duyệt, và nếu đó là beat cuối chưa viết thì tổng thời lượng được đưa vào mục tiêu); `POST …/approve` (cần có mục tiêu). Trường nào đổi giá trị thì thành của Creator (`mine`) và AI giữ khi viết lại cả bảng; beat sửa xong về chờ duyệt.

## Đường dẫn

| Phương thức | Đường dẫn | Việc |
|---|---|---|
| GET | `/v1/projects/{id}/beats` | Toàn màn: `status`, `story_approved`, `phase` (`empty`, `skeleton`, `content`), `unwritten` (số beat chưa có nội dung), `rebalanced`, `beats[]` (kèm `status`, `mine[]`, `stale`, `only_must[]`: dữ kiện bắt buộc chỉ beat này giữ, `no_fact`: beat không phải Hook mà không có dữ kiện), `facts[]` (dữ kiện nghiên cứu đang giữ), `input` (`{topic, brief, brief_approved, story}`: đề tài, brief đã duyệt và câu chuyện đã duyệt đúng như prompt nhận, để Creator xem trước khi gọi AI; rỗng khi câu chuyện chưa duyệt), `checks[]` (`{ok,text}`, chỉ cảnh báo; ở giai đoạn bộ khung: beat đầu là Hook, beat cuối là Payoff, dữ kiện bắt buộc; ở giai đoạn nội dung: tổng thời lượng, hook ≤ 20 s, dữ kiện bắt buộc, mục tiêu, beat cuối là Payoff), `total_seconds`, `target_label` |
| POST | `/v1/projects/{id}/beats/skeleton` | Dựng (hay dựng lại) bộ khung từ câu chuyện đã duyệt, không gọi AI: một Hook (từ `question`), mỗi dòng của `chain` một beat mang dữ kiện của dòng đó, một Payoff (từ `payoff`). Thay các beat đang có; bộ khung dựng ra chưa duyệt; trả state. `409` khi bộ khung đã duyệt và bảng còn beat (xoá hết beat thì cờ duyệt được xoá, dựng lại được) |
| POST | `/v1/projects/{id}/beats/skeleton/approve` | Duyệt bộ khung (chức năng và dữ kiện của các beat); từ đó AI mới viết nội dung. `400` khi chưa có beat; trả state |
| POST | `/v1/projects/{id}/beats/generate` | AI viết nội dung của các beat, mỗi beat một lượt gọi theo thứ tự, body `{"note":"","only_unwritten":false}` tuỳ chọn (`only_unwritten:true` chỉ viết các beat chưa có nội dung, beat đã viết giữ nguyên); trả state. Mỗi lượt thấy bộ khung, các beat đã viết và ngân sách thời lượng; chức năng và dữ kiện giữ như đã duyệt, phần Creator đã viết giữ giá trị. Khi không còn beat nào chưa viết, code đưa tổng thời lượng vào khoảng mục tiêu (co giãn theo tỉ lệ, làm tròn 5 giây, Hook ≤ 20 giây; thời lượng Creator đã đặt và beat đã duyệt giữ nguyên; ghi số beat đã đổi vào `rebalanced`). Chỉ lưu khi mọi beat đã viết xong; mọi beat về chờ duyệt. `409` khi bộ khung chưa duyệt |
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
| `409` | Câu chuyện chưa duyệt, bộ khung chưa duyệt khi cần AI viết (cả bảng, thêm hay làm lại một beat), bộ khung đã duyệt khi dựng lại, hoặc project không còn là bản nháp |
| `502` | Câu trả lời của AI cho một beat không đọc được sau hai lần thử (JSON hỏng, thiếu mục tiêu hay nội dung, chức năng lạ, thời lượng ngoài khoảng, trích dữ kiện lạ), hoặc không có cách chia thời lượng cho tổng khớp mục tiêu; không lưu gì |
| `503` | Chưa cấu hình AI |
