# ADR-0026: Tách OAuth app và tài khoản YouTube thành hai tầng độc lập

## Status
Proposed

## Date
2026-09-09

## Stage
Low-Level Design

## Context
Creator muốn nối nhiều tài khoản Google, và giả định cách làm là "mỗi tài khoản một file
client_secret". Đồng thời `oauth_credentials` hiện chốt cứng `CHECK (id = 1)` nên chỉ chứa được
đúng một credential — nối kênh thứ hai sẽ ghi đè mất kênh thứ nhất (CR-012 D1).

Hai sự thật kỹ thuật định hình quyết định này:
1. Một OAuth client cấp được refresh token cho **bao nhiêu tài khoản Google cũng được**. Số client
   không hề giới hạn số tài khoản.
2. Quota YouTube Data API tính **theo GCP project**, mặc định 10.000 unit/ngày; `videos.insert` tốn
   1.600 unit → ~6 video/ngày mỗi project, dùng chung cho mọi kênh nối qua project đó.

Nghĩa là số client_secret quyết định **thông lượng**, còn số credential quyết định **nối được mấy
kênh**. Hai trục khác nhau.

## Options Considered

### Option A: 1 file client_secret = 1 tài khoản (đúng nguyên văn yêu cầu)
- What it is: mỗi kênh YouTube bắt buộc phải có một file client_secret và một GCP project riêng.
- Strengths: mô hình đơn giản một-một, dễ hình dung; mỗi kênh tự nhiên có rổ quota riêng.
- Trade-offs: buộc Creator tạo một GCP project + OAuth client + màn hình consent cho **mỗi** kênh,
  kể cả khi chỉ muốn nối hai kênh của cùng một người và không cần thêm quota. Đây là công việc thủ
  công nặng nhất trong toàn bộ luồng, và mô hình này bắt làm nó không cần thiết. Nó cũng mã hoá một
  ràng buộc không có thật vào schema, nên sau này gỡ ra là phải migrate lần nữa.

### Option B: Nhiều credential trên đúng một OAuth client
- What it is: giữ một file client_secret, bảng credential khoá theo `channel_id`.
- Strengths: thay đổi nhỏ nhất; sửa được đúng bug ghi đè; Creator không phải dựng thêm GCP project.
- Trade-offs: mọi kênh dùng chung một rổ quota ~6 video/ngày. Chính là trần mà Creator sẽ đụng ngay
  khi bắt đầu chạy nhiều kênh — tức là giải quyết được vấn đề đã nêu mà bỏ sót vấn đề sắp tới.

### Option C: Hai tầng — danh mục app quét từ thư mục, credential khoá theo kênh
- What it is: `secrets/` chứa N file client_secret (tầng 1, rổ quota); bảng `youtube_accounts` chứa
  M kênh, mỗi kênh ghi nhớ `client_id` của app đã cấp nó (tầng 2). N và M độc lập.
- Strengths: bao trùm cả Option B (N=1) lẫn Option A (N=M) mà không ép ai vào trường hợp nào; thêm
  quota = thả thêm file, không sửa code; nối thêm kênh = một lần consent. Việc lưu `client_id` theo
  từng kênh là **bắt buộc về mặt kỹ thuật** dù chọn gì: refresh token chỉ refresh được bằng đúng
  cặp client_id/secret đã cấp nó, nên một cặp toàn cục sẽ hỏng ngay khi có app thứ hai.
- Trade-offs: nhiều khái niệm hơn cho Creator ("app" và "tài khoản" là hai thứ khác nhau); GUI phải
  hỏi chọn app khi có >1; nhiều mã hơn Option B.

## Decision
Option C — hai tầng độc lập: danh mục OAuth app nạp từ thư mục `secrets/`, và bảng
`youtube_accounts` khoá theo `channel_id` có cột `client_id` trỏ ngược về app.

## Rationale
Creator hỏi "nối nhiều tài khoản", nhưng nhu cầu đứng sau là chạy nhiều kênh thật sự — mà nhu cầu
đó đụng trần quota ~6 video/ngày ngay khi bắt đầu dùng. Option B sửa được câu hỏi chữ mà không sửa
được nhu cầu; Option A sửa được nhu cầu nhưng bắt trả giá thiết lập thủ công ngay cả trong trường
hợp không cần.

Option C thắng vì phần thêm so với B là nhỏ và phần lớn **không tránh được**: đã có khả năng nhiều
app thì việc lưu `client_id` theo từng credential là điều kiện đúng đắn của refresh token, không
phải chi phí phát sinh. Chi phí thật chỉ còn là màn hình chọn app trong GUI, và nó được ẩn đi khi
chỉ có một app (CR-012 FR34.2) — nên Creator dùng ở quy mô nhỏ không phải trả gì.

Chọn nạp từ **thư mục** thay vì biến env đánh số (`GOOGLE_OAUTH_CLIENT_ID_2`, `_3`…) vì file
client_secret là thứ Google Cloud Console phát ra nguyên khối; bắt Creator tách nó ra thành từng
biến env là một bước chép tay thủ công dễ sai, và không có giới hạn số app nào cần tồn tại trong
cấu hình.

## Consequences
- Đổi khoá chính bảng credential ⇒ cần migration (CR-012 FR31.3), chạy tự động lúc bootstrap.
- `channel_id` phải xuyên qua 4 service (web-gui → gateway → orchestrator → publisher).
- `YouTubeVideoPublisher` không còn nhận client_id/secret ở constructor mà tra registry theo
  credential — đây là hệ quả trực tiếp và bắt buộc của nhiều app.
- Thư mục `secrets/` phải được gitignore; đặt file secret sai chỗ là rò rỉ thật (CR-012 D3).
- Vẫn còn trần ~6 video/ngày **mỗi app**. ADR này nâng trần, không bỏ trần.

## Related
- ADR-0016 (token lưu plaintext — không đổi)
- ADR-0002 (ports & adapters — `OAuthAppRegistryPort` theo đúng khuôn này)
- CR-012
