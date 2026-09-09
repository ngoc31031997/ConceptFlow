# CR-012 — Nhiều tài khoản YouTube trên nhiều OAuth client (P1)

## Date
2026-09-09

## Stage
Requirements Analysis → Low-Level Design (chờ Creator duyệt trước khi Code Generation)

## Intent Analysis
- **Request type**: Enhancement + bug fix (một lỗi ghi đè dữ liệu đã tồn tại sẵn)
- **Scope estimate**: Cross-service — `publisher` (chính), `orchestrator`, `api-gateway`, `web-gui`
- **Complexity estimate**: Moderate — đổi khoá chính của bảng credential, cần migration

## Bối cảnh
Creator bổ sung một file `client_secret_770292878083-…json` thuộc GCP project `concer-508105`,
khác project của cặp client_id/secret đang nằm trong `.env` (`129036727441-…`), và hỏi liệu có thể
nối nhiều tài khoản Google, mỗi tài khoản một file client_secret.

## Làm rõ giả định của Creator
Giả định "1 file client_secret = 1 tài khoản" là **không đúng**. Một OAuth client cấp được refresh
token cho bao nhiêu tài khoản Google cũng được — mỗi lần consent là một credential riêng.

Lý do thật sự để cần nhiều client_secret là **quota**: quota YouTube Data API tính theo GCP
project, mặc định 10.000 unit/ngày, mà `videos.insert` tốn 1.600 unit → **~6 video/ngày cho toàn
hệ thống**, bất kể nối bao nhiêu kênh. Chỉ khi trải các kênh qua nhiều GCP project thì trần đó mới
được nhân lên.

Nên CR này tách thành **hai tầng độc lập**, không gộp làm một:

| Tầng | Thực thể | Là gì | Nguồn dữ liệu |
|---|---|---|---|
| 1 | **OAuth app** | 1 file `client_secret*.json` = 1 GCP project = 1 rổ quota | quét thư mục `secrets/` |
| 2 | **Tài khoản** | 1 kênh YouTube đã consent, ghi nhớ app nào đã cấp nó | bảng `youtube_accounts` |

## Ba khiếm khuyết hiện có mà CR này phải sửa

**D1 — Ghi đè mất credential (bug thật, không chỉ là thiếu tính năng).**
`adapters/persistence/db.py` chốt cứng `oauth_credentials … CHECK (id = 1)`, và
`PostgresCredentialStore.save()` `INSERT … VALUES (1, …) ON CONFLICT (id) DO UPDATE`. Nối kênh thứ
hai sẽ **ghi đè im lặng** lên kênh thứ nhất — Creator mất quyền đăng lên kênh cũ mà không có cảnh
báo nào.

**D2 — `redirect_uri_mismatch` chắc chắn xảy ra với file mới.**
File Creator đưa chỉ khai `redirect_uris: ["http://localhost:3000/"]`, trong khi luồng thật callback
về `http://localhost:3000/oauth/youtube/callback` (`web-gui/src/App.tsx:23`). Google so khớp
redirect URI chính xác từng ký tự nên sẽ từ chối. Hiện hệ thống không hề kiểm tra điều này trước —
Creator chỉ gặp một trang lỗi của Google, không có gợi ý phải sửa gì.

**D3 — Client secret nằm sai chỗ.**
File client_secret đang ở repo root và `.gitignore` không loại trừ nó (`git check-ignore` xác nhận).
Một lệnh `git add .` là secret vào lịch sử Git vĩnh viễn.

## Functional Requirements

### FR29 — Danh mục OAuth app nạp từ thư mục
- **FR29.1**: PHẢI có `OAuthAppRegistryPort` ở `domain/ports.py`; application layer chỉ phụ thuộc
  port này, không đọc file trực tiếp (ADR-0002).
- **FR29.2**: `FileOAuthAppRegistry` quét `GOOGLE_OAUTH_CLIENT_SECRETS_DIR` tìm mọi `*.json`, đọc
  được cả khoá `web` lẫn `installed`, khoá theo `client_id`. Thả thêm file là có thêm app, không
  cần sửa code hay biến env.
- **FR29.3**: PHẢI giữ tương thích ngược — nếu thư mục trống/không tồn tại mà ba biến
  `GOOGLE_OAUTH_CLIENT_ID/SECRET/REDIRECT_URI` có giá trị thì tổng hợp thành đúng một app. Cấu hình
  `.env` hiện tại của Creator vẫn chạy nguyên trạng.
- **FR29.4**: File hỏng (JSON sai, thiếu `client_id`) PHẢI bị bỏ qua kèm log cảnh báo nêu tên file,
  không được làm service chết lúc khởi động — một file thừa trong `secrets/` không đáng đánh sập
  cả Publisher.
- **FR29.5**: Registry PHẢI **không bao giờ** để `client_secret` lọt ra REST response hay log.

### FR30 — Kiểm tra redirect URI trước khi đẩy Creator sang Google (sửa D2)
- **FR30.1**: `GET /v1/auth/youtube/apps` PHẢI trả cờ `redirect_ok` cho từng app = liệu
  `GOOGLE_OAUTH_REDIRECT_URI` có nằm trong `redirect_uris` của file đó không.
- **FR30.2**: `/start` với app có `redirect_ok=false` PHẢI trả 400 kèm thông điệp nêu **đúng chuỗi
  URI cần thêm** vào Google Cloud Console, thay vì redirect sang một trang lỗi của Google.
- **FR30.3**: GUI PHẢI hiện cảnh báo đó tại chỗ chọn app, không để Creator bấm rồi mới biết.

### FR31 — Lưu nhiều tài khoản (sửa D1)
- **FR31.1**: Bảng mới `youtube_accounts`, khoá chính `channel_id`, thêm cột `client_id` (app nào
  cấp), `channel_title`, `is_default`, `created_at`, `updated_at`.
- **FR31.2**: PHẢI có unique partial index bảo đảm nhiều nhất một hàng `is_default = TRUE`.
- **FR31.3**: Bootstrap PHẢI di trú hàng `oauth_credentials.id = 1` đang có sang bảng mới khi bảng
  mới còn rỗng, lấy `client_id` từ env cũ, đặt `is_default = TRUE`. Creator không phải nối lại kênh
  đang dùng. Bảng cũ được giữ nguyên, không DROP.
- **FR31.4**: `CredentialStorePort` mở rộng: `get(channel_id | None)` (None ⇒ tài khoản mặc định),
  `list()`, `save()`, `delete(channel_id)`, `set_default(channel_id)`.
- **FR31.5**: Consent lại một kênh đã có PHẢI cập nhật đúng hàng của kênh đó (upsert theo
  `channel_id`), không đẻ hàng mới, không đụng các kênh khác.
- **FR31.6**: Google chỉ trả `refresh_token` ở lần consent đầu. Khi exchange không có refresh token
  mà kênh đó đã có sẵn trong DB, PHẢI giữ lại refresh token cũ thay vì ghi đè bằng NULL.

### FR32 — Đăng video lên đúng kênh đã chọn
- **FR32.1**: `PublishRequest` thêm `channel_id: str | None`; None ⇒ dùng tài khoản mặc định (giữ
  nguyên hành vi cũ cho project đã tạo trước CR này).
- **FR32.2**: `channel_id` PHẢI xuyên suốt: web-gui → api-gateway → orchestrator (`dto.go`,
  `router.go`, `start_publish_saga.go`, `domain/project.go`, cột `youtube_channel_id`) → payload
  AMQP → `consumer.py` → `PublishRequest`.
- **FR32.3**: `channel_id` trỏ tới kênh không tồn tại PHẢI raise `MissingCredentialError` với thông
  điệp nêu rõ kênh nào, không được âm thầm rơi về kênh mặc định — đăng nhầm kênh là hỏng không sửa
  được về mặt đối ngoại.
- **FR32.4**: `YouTubeVideoPublisher` PHẢI refresh token bằng đúng cặp client_id/secret của app đã
  cấp credential đó (tra registry theo `credential.client_id`), không dùng một cặp toàn cục. Dùng
  sai client sẽ khiến refresh thất bại với `invalid_client`.

### FR33 — State parameter mang được app + project
- **FR33.1**: `state` hiện chỉ chứa `project_id`. Nay PHẢI mã hoá base64url-JSON gồm `client_id`,
  `project_id`, và một nonce ngẫu nhiên, để callback biết dùng app nào mà exchange.
- **FR33.2**: Nonce PHẢI được sinh ở `/start`, lưu tạm trong bộ nhớ có TTL, và kiểm ở `/callback` —
  chặn CSRF nối một kênh lạ vào hệ thống của Creator.
- **FR33.3**: State cũ (chuỗi project_id trần) PHẢI vẫn đọc được, để một luồng OAuth đang dở dang
  lúc deploy không gãy.

### FR36 — Cho phép chọn tài khoản/kênh ở mỗi lần consent
- **FR36.1**: `authorization_url` PHẢI dùng `prompt="select_account consent"` thay vì `"consent"`.
  `"consent"` chỉ ép hiện màn hình đồng ý, không ép hiện màn hình chọn tài khoản — Google tái dùng
  phiên đang đăng nhập, nên Creator **không nối được kênh thứ hai** trừ khi đăng xuất Google.
- **FR36.2**: Một tài khoản Google sở hữu được nhiều kênh (kênh cá nhân + các Brand Account). Quan
  hệ thật là 1 app → N tài khoản Google → M kênh, và token gắn với **đúng một kênh** chọn lúc
  consent. Khoá theo `channel_id` (FR31.1) là thứ xử lý đúng cả hai trường hợp: hai kênh cùng một
  tài khoản Google, hay hai tài khoản Google khác nhau.
- **FR36.3**: `_fetch_channel_id` PHẢI lấy thêm `part="snippet"` để có `channel_title` — Creator
  cần thấy tên kênh để phân biệt, `channel_id` là chuỗi không đọc được.
- **FR36.4**: **Ràng buộc không vượt qua được**: token gắn với đúng một kênh (kênh chọn lúc
  consent), và `channels.list(mine=true)` chỉ trả về kênh đó — không có cách nào nối một tài khoản
  Google rồi liệt kê mọi kênh của nó để chọn sau. (`onBehalfOfContentOwner` chỉ dành cho tài khoản
  YouTube CMS/đối tác.) Do đó **mỗi kênh = một lần consent**, và bước chọn kênh nằm trên màn hình
  chooser của Google. Việc chọn kênh trong GUI chỉ áp dụng ở **lúc đăng video**, chọn giữa các kênh
  đã nối.

### FR34 — Quản lý tài khoản trong GUI
- **FR34.1**: `YoutubeConnectButton` (boolean đã/chưa kết nối) chuyển thành bảng kênh đã nối: tên
  kênh, app/GCP project sở hữu, nhãn "mặc định", nút xoá.
- **FR34.2**: Nút "Thêm kênh" cho chọn app rồi mới sang Google. Đúng một app ⇒ bỏ qua bước chọn.
  GUI PHẢI nói rõ rằng **việc chọn kênh diễn ra trên màn hình của Google**, và mỗi kênh cần một lần
  consent riêng — nếu không Creator sẽ chờ một danh sách kênh không bao giờ hiện ra (xem FR36.4).
- **FR34.3**: Màn đăng video PHẢI cho chọn kênh, mặc định chọn sẵn tài khoản `is_default`.
- **FR34.4**: PHẢI hiện quota theo app ở dạng thông tin ("mỗi GCP project ~6 video/ngày") để Creator
  hiểu vì sao nên trải kênh qua nhiều app — đây chính là câu hỏi ban đầu của Creator.

### FR35 — Vị trí file secret (sửa D3)
- **FR35.1**: PHẢI tạo `secrets/` ở repo root, thêm vào `.gitignore`, kèm `secrets/README.md` giải
  thích cách thả file (README được commit, file secret thì không).
- **FR35.2**: `docker-compose.yml` PHẢI mount `./secrets` read-only vào container publisher.
- **FR35.3**: PHẢI di dời file client_secret đang nằm ở repo root vào `secrets/`.
- **FR35.4**: KHÔNG được ghi bất kỳ giá trị client secret nào vào file được commit.

## Non-goals
- Không tự động chia tải/round-robin giữa các app — Creator chọn kênh thủ công.
- Không mã hoá token khi lưu (giữ nguyên threat model local-only của ADR-0016).
- Không đụng luồng TTS, render, hay Saga ngoài việc thêm một trường `channel_id`.
- Không đăng đồng thời một video lên nhiều kênh — một lệnh publish, một kênh.

## Rủi ro đã biết
- **Secret đã lộ**: client secret của `770292878083-…` đã xuất hiện dưới dạng plaintext trong hội
  thoại. Creator đã chọn xoay (rotate) nó trên Google Cloud Console. CR này cố tình không ghi giá
  trị secret nào vào repo (FR35.4).
- **Trần quota vẫn còn**: nhiều app nâng trần lên ~6 video/ngày mỗi app, không phải bỏ trần. Muốn
  cao hơn nữa phải xin Google duyệt quota.

## Liên quan
- ADR-0026 (quyết định kiến trúc hai tầng của CR này)
- ADR-0016 (lưu token dạng plaintext, threat model local-only)
- ADR-0002 (ports & adapters)
