# Nghiên cứu và kiểm chứng: mẫu `conceptflow.research.v1` và đường ra Internet

Bước Nghiên cứu (authoring-service) đọc câu trả lời của một công cụ nghiên cứu bên ngoài mà Creator dán vào, rồi tải từng trang nguồn để kiểm đoạn trích. File này ghi hợp đồng của hai việc đó. Lý do chọn cách làm: [ADR-0043](../../aidlc-docs/decisions/ADR-0043-pasted-research-with-source-verification.md). Mã tương ứng: `services/authoring-service/internal/domain/research.go` (đọc khối JSON), `internal/domain/source_match.go` (so đoạn trích), `internal/adapters/sourcefetch/` (tải trang).

## Nơi lưu và nguồn đầu vào

- Dữ kiện lưu trong bảng `project_research` của authoring-service, một dòng một dữ kiện, khoá `(project_id, fact_id)`. Các bước sau đọc bảng này. Cột: `question_id`, `claim_vi`, `type`, `source_title`, `publisher`, `year`, `url`, `quote`, `confidence`, `note`, `must`, `keep`, `link_state`, `link_why`, `link_nearest`, `link_checked_at`, `creator_fields` (các ô Creator tự viết, lần dán sau không ghi đè), `created_at`, `updated_at`. Bước đọc dữ kiện đang giữ lọc `keep = true` và dùng `type`, `link_state` để biết dữ kiện đã được xác minh chưa.
- Câu hỏi nghiên cứu và cổng duyệt/bỏ qua của bước vẫn là review unit (`research_question`, `research`).
- Đầu vào của bước: idea đã duyệt đọc từ `project_idea` (`{{idea}}`) và brief đã duyệt đọc từ `project_brief` (`{{brief}}`). Chưa duyệt một trong hai thì sinh câu hỏi và lấy prompt trả `409`.
- Duyệt bảng chỉ cần có một dữ kiện đang giữ; nguồn chưa xác minh không chặn duyệt. Bước vẫn có thể bỏ qua.

## Mẫu JSON

Prompt nghiên cứu (vai `research_prompt`, sửa được ở Cài đặt prompt) yêu cầu công cụ trả duy nhất một khối ` ```json ` như sau:

```json
{"schema":"conceptflow.research.v1","claims":[{"id":"F01","question":"Q1","claim_vi":"...","type":"fact","source_title":"...","publisher":"...","year":2020,"url":"https://...","quote":"...","confidence":"high","note":""}]}
```

| Trường | Kiểu | Ý nghĩa |
|---|---|---|
| `id` | chuỗi | Mã dữ kiện `F01`, `F02`…; `F1`, `f01`, `F001` đều đọc thành `F01`. Bắt buộc |
| `question` | chuỗi | Mã câu hỏi nghiên cứu mà dữ kiện trả lời (`Q1`…); phải là câu hỏi đang có. Bắt buộc |
| `claim_vi` | chuỗi | Dữ kiện bằng tiếng Việt, một câu một ý. Bắt buộc |
| `type` | chuỗi | `fact` (đã xác nhận), `hypothesis` (giả thuyết còn tranh luận), `unknown` (chưa ai biết hoặc không tìm được), `misconception` (ngộ nhận phổ biến). Bắt buộc |
| `source_title`, `publisher` | chuỗi | Tên nguồn, nơi đăng |
| `year` | số hoặc chuỗi | Năm |
| `url` | chuỗi | Link trang nguồn, `http` hoặc `https`, tối đa 2048 ký tự, không kèm tài khoản. Để trống khi `type` là `unknown` |
| `quote` | chuỗi | Một câu chép nguyên văn từ trang, giữ ngôn ngữ gốc |
| `confidence`, `note` | chuỗi | Độ tin và ghi chú tuỳ ý |

## Luật đọc câu trả lời

- Tìm khối: ưu tiên mọi khối ` ``` ` có chuỗi `conceptflow.research.v1`; không có thì lấy mọi đoạn `{…}` cân ngoặc có chuỗi đó. Đọc hết các khối tìm được, một khối hỏng chỉ thành một dòng bị bỏ. Không có khối nào thì báo `400` và không đọc gì, không ghi nhật ký.
- Giới hạn: câu trả lời tối đa 1 MB; tối đa 300 dòng (từ dòng thứ 301 bị bỏ).
- Mỗi dòng không đọc được bị bỏ kèm lý do (số dòng, tiếng Việt): thiếu hoặc sai `id`, thiếu `claim_vi`, thiếu hoặc lạ `type`, `question` không có trong danh sách, `url` không dùng được. Trùng mã trong cùng lần dán: lấy dòng sau, dòng trước ghi vào lý do.
- Gộp theo mã: mã chưa có thì thêm; mã đã có thì cập nhật mọi trường chữ mà Creator chưa tự viết hoặc sửa, giữ nguyên "bắt buộc nói" và "đã bỏ". `url` hoặc `quote` đổi thì kiểm link lại; không đổi thì giữ kết quả cũ.
- Mỗi lần dán có một dòng nhật ký (số lần, lúc, mã thêm, mã cập nhật, dòng bị bỏ). Nguyên văn câu trả lời không được lưu.

## HTTP (qua api-gateway, `/v1/projects/{id}/research…`)

| Route | Việc |
|---|---|
| `GET /research` | Cả màn: `gate`, `questions`, `facts` (có `link`), `pastes`, `coverage`, `brief_approved`, `questions_stale` |
| `POST /research/questions/generate` `{replace_edited, note}` | AI sinh câu hỏi từ brief (theo dõi bằng `X-Operation-Id`); `502` khi trả lời không đọc được, `503` khi chưa có AI |
| `POST /research/questions` `{text}`, `DELETE /research/questions/{qid}` | Thêm, xoá câu hỏi. Sửa chữ qua `PUT /units/research_question/{qid}` `{fields:{text}}` |
| `GET /research/prompt`, `GET /research/follow-up-prompt` | `{prompt}`; prompt bổ sung `409` khi mọi câu hỏi đã đủ nguồn |
| `POST /research/paste` `{raw}` | `{added, updated, rejected:[{row, reason}], log}` |
| `POST /research/facts`, `PUT /research/facts/{fid}` `{type?, must?, keep?}` | Thêm tay; đổi loại, "bắt buộc nói", bỏ hoặc giữ lại. Sửa chữ, link, đoạn trích qua `PUT /units/fact/{fid}` |
| `POST /research/verify` `{fact_ids}` | Kiểm lại link, `202` (chạy nền); danh sách rỗng là mọi dữ kiện đang giữ có link |
| `PUT /research/skip` `{skipped}` | Bỏ qua bước hoặc quay lại làm |
| `POST /units/research/main/approve` | Duyệt bảng dữ kiện; `400` khi không có dữ kiện nào đang giữ, `409` khi bước đang bỏ qua |

`link.state` của một dữ kiện: `ok` (trang có đoạn trích), `bad` (tải được trang nhưng không thấy đoạn trích; kèm `nearest` là câu gần nhất nếu có), `na` (không tải được, kèm `why`), `none` (không có link), `run` (đang kiểm).

## Đường ra Internet của authoring-service

authoring-service là service duy nhất tải trang web do người ngoài đưa link. Việc này chặn địa chỉ nội bộ (SSRF):

| Hạng mục | Luật |
|---|---|
| Lược đồ và cổng | `http`, `https`; cổng 80 và 443 |
| Tên bị chặn trước khi phân giải | `localhost`, tên không có dấu chấm (tên service trong mạng Docker như `rabbitmq`), đuôi `.local`, `.internal`, `.localdomain`, `.lan` |
| Địa chỉ bị chặn | loopback, riêng (`10/8`, `172.16/12`, `192.168/16`, `fc00::/7`), link-local (`169.254/16` có địa chỉ metadata đám mây, `fe80::/10`), `0.0.0.0/8`, `100.64/10`, `192.0.0.0/24`, `198.18/15`, multicast, `240/4`, broadcast; IPv4 viết dạng IPv6 được bóc rồi kiểm. Mọi địa chỉ một tên phân giải ra đều phải công khai |
| Chống đổi địa chỉ | địa chỉ thực sự được nối được kiểm lại lúc nối, không phân giải lần hai |
| Chuyển hướng | tối đa 5; mỗi lần kiểm như một link mới |
| Thời gian, dung lượng | nối 4 giây, cả lần tải 12 giây; trang tối đa 5 MiB |
| Kiểu nội dung | `text/html`, `application/xhtml+xml`, `text/plain`; kiểu khác (PDF…) là "không tải được" |
| Đồng thời | 4 trang cùng lúc |
| Khác | không cookie, không header đăng nhập, không chạy script; trạng thái 401/402/403 là "trang trả phí hoặc cần đăng nhập" |

So đoạn trích: cả trang và đoạn trích được chuẩn hoá (bỏ thẻ và script, giải mã thực thể, Unicode NFKC, chữ thường, nháy và gạch thống nhất, bỏ dấu chú thích như `[1]`, gộp khoảng trắng). Đoạn trích có `…` hoặc `...` là danh sách các đoạn phải có đủ theo thứ tự; đoạn ngắn hơn 20 ký tự bị bỏ qua, đoạn trích không còn đoạn nào đủ dài thì không kiểm được. Không thấy thì tìm câu của trang có từ trùng nhiều nhất (từ 50% trở lên) làm `nearest`.

Cổng `ResearchSource` (tên nguồn tra cứu; bản duy nhất hiện nay là `paste`) để sau này cắm API tìm kiếm mà không đổi bảng dữ kiện.
