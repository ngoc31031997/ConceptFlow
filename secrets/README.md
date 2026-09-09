# secrets/

Thư mục chứa file `client_secret*.json` tải từ Google Cloud Console.
File README này được commit; **mọi file `.json` trong đây thì không** (xem `.gitignore`).

> Hướng dẫn đầy đủ từng bước (bật API, consent screen, test users, lỗi thường
> gặp): [`docs/setup/youtube-oauth-client.md`](../docs/setup/youtube-oauth-client.md).
> Tóm tắt bên dưới.

## Thêm một OAuth app (= thêm một rổ quota)

1. Google Cloud Console → tạo (hoặc chọn) một **GCP project**
2. **APIs & Services → Library** → bật **YouTube Data API v3**
3. **Credentials → Create credentials → OAuth client ID** → *Web application*
4. Ở **Authorized redirect URIs**, thêm **chính xác** chuỗi này:

       http://localhost:3000/oauth/youtube/callback

   Google so khớp từng ký tự. Thiếu bước này là gặp `redirect_uri_mismatch`.
5. Tải file JSON về, thả thẳng vào thư mục này. Không cần sửa code hay biến env —
   Publisher quét thư mục lúc khởi động và tự nhận app mới.

## Vì sao lại cần nhiều file?

Quota YouTube Data API tính **theo GCP project**: 10.000 unit/ngày, mà mỗi lần upload
video tốn 1.600 unit → **~6 video/ngày cho mỗi app**, dùng chung cho mọi kênh nối qua app đó.

Thêm file = thêm rổ quota. Nối thêm kênh vào cùng một app thì **không** thêm quota.

## App và kênh là hai thứ khác nhau

- **App** (file ở đây) = 1 GCP project = 1 rổ quota
- **Kênh** = 1 kênh YouTube đã consent, lưu trong bảng `youtube_accounts` của publisher-db

Một app nối được bao nhiêu kênh cũng được. Mỗi kênh cần **một lần consent riêng** — kể cả
hai kênh thuộc cùng một tài khoản Google — vì token YouTube gắn cứng vào đúng một kênh,
là kênh bạn chọn trên màn hình chooser của Google.

Xem `aidlc-docs/decisions/ADR-0026-two-tier-oauth-apps-and-accounts.md`.
