# Lấy OAuth client cho YouTube (đăng video)

Dùng cho: Publisher Service đăng video lên kênh YouTube của bạn (CR-012, ADR-0026).

**Bắt buộc** nếu muốn đăng video tự động. Kết quả cần: một file
`client_secret*.json` thả vào thư mục `secrets/` ở gốc repo.

---

## Khái niệm phải nắm trước, không sẽ làm thừa việc

| | |
|---|---|
| **OAuth client** (file JSON) | = 1 GCP project = **1 rổ quota** |
| **Kênh** | 1 kênh YouTube đã cấp quyền, lưu trong DB |

- Một client nối được **bao nhiêu kênh cũng được**. Không cần mỗi kênh một file.
- Thêm file chỉ để **thêm quota**. Quota tính theo GCP project: 10.000 unit/ngày,
  mỗi lần upload tốn 1.600 → **~6 video/ngày cho mỗi file**, dùng chung cho mọi
  kênh nối qua nó.
- Mỗi kênh vẫn cần **cấp quyền riêng một lần**, kể cả hai kênh cùng một tài khoản
  Google. Token YouTube gắn cứng vào đúng một kênh — kênh bạn chọn trên màn hình
  của Google. Không có cách nào nối một tài khoản rồi liệt kê các kênh còn lại.

---

## 1. Tạo GCP project

1. Vào https://console.cloud.google.com
2. Thanh trên cùng, bấm ô chọn project → **NEW PROJECT**
3. Đặt tên (ví dụ `conceptflow-2`) → **CREATE**
4. Chọn đúng project vừa tạo trước khi làm bước sau

## 2. Bật YouTube Data API v3

1. **APIs & Services** → **Library**
2. Tìm **YouTube Data API v3** → mở → **ENABLE**

Quên bước này thì OAuth vẫn chạy nhưng upload sẽ lỗi 403 `accessNotConfigured`.

## 3. OAuth consent screen

1. **APIs & Services** → **OAuth consent screen**
2. User Type: **External** → **CREATE**
3. Điền App name, User support email, Developer contact email → **SAVE AND CONTINUE**
4. **Scopes** → **ADD OR REMOVE SCOPES** → thêm hai scope:
   ```
   https://www.googleapis.com/auth/youtube.upload
   https://www.googleapis.com/auth/youtube.readonly
   ```
   → **UPDATE** → **SAVE AND CONTINUE**
5. **Test users** → **ADD USERS** → thêm **mọi tài khoản Google** bạn định nối kênh

> ⚠️ **Bước 5 là chỗ hay tắc nhất.** Khi app còn ở trạng thái *Testing*, chỉ email
> nằm trong danh sách Test users mới cấp quyền được. Tài khoản khác sẽ bị Google
> chặn ngay ở màn hình consent với thông báo mơ hồ.

> ⚠️ **Trạng thái *Testing* làm refresh token hết hạn sau 7 ngày.** Nghĩa là cứ
> khoảng một tuần bạn phải nối lại kênh. Muốn token sống lâu thì phải bấm
> **PUBLISH APP** để chuyển sang *In production*. Với scope `youtube.upload`,
> Google yêu cầu xác minh (verification) trước khi cho publish — chấp nhận nối
> lại hàng tuần cũng là một lựa chọn hợp lý nếu bạn chỉ dùng cá nhân.

## 4. Tạo OAuth client ID

1. **APIs & Services** → **Credentials** → **+ CREATE CREDENTIALS** → **OAuth client ID**
2. Application type: **Web application**
3. Name: tuỳ ý
4. **Authorized redirect URIs** → **+ ADD URI** → dán **chính xác**:

   ```
   http://localhost:3000/oauth/youtube/callback
   ```

   Google so khớp **từng ký tự**. Thừa/thiếu dấu `/` ở cuối là mismatch. Chuỗi này
   phải khớp `GOOGLE_OAUTH_REDIRECT_URI` trong `.env`.
5. **CREATE** → cửa sổ hiện ra, bấm **DOWNLOAD JSON**

## 5. Thả file vào repo

```bash
mv ~/Downloads/client_secret_*.json <repo>/secrets/
```

Tên file không quan trọng, kể cả có ` (1)` hay dấu cách — Publisher quét mọi
`*.json` trong thư mục. Thư mục này đã được `.gitignore`, file secret không vào Git.

## 6. Áp dụng và kiểm tra

```bash
docker compose up -d --build publisher
docker compose logs publisher | grep "Loaded"
```

Phải thấy: `Loaded 1 OAuth app(s): <tên GCP project>`

Kiểm nhanh redirect URI có khớp không, không cần mở trình duyệt:

```bash
curl -s http://localhost:8080/v1/auth/youtube/apps | python3 -m json.tool
```

`"redirect_ok": true` là xong. `false` nghĩa là bước 4 chưa đúng — quay lại
Console sửa, **tải lại file JSON**, thay file cũ trong `secrets/`. Sửa tay
`redirect_uris` trong file JSON **không có tác dụng**: Google kiểm theo dữ liệu
phía họ, không theo file của bạn.

## 7. Nối kênh

Mở web-gui → vào một project ở màn kết quả → **Thêm kênh** → Google hiện màn hình
chọn tài khoản và kênh → chọn → quay về, kênh xuất hiện trong danh sách.

Lặp lại cho từng kênh. Khi đăng video, chọn kênh bằng radio trong danh sách đó.

---

## Lỗi thường gặp

| Thông báo | Nguyên nhân |
|---|---|
| `redirect_uri_mismatch` | bước 4 sai, hoặc sửa file JSON thay vì sửa trên Console |
| Google chặn ở màn consent | tài khoản chưa có trong **Test users** (bước 3.5) |
| `accessNotConfigured` / 403 khi upload | chưa bật YouTube Data API v3 (bước 2) |
| `quotaExceeded` | hết 10.000 unit/ngày (~6 video). Đợi sang ngày, hoặc thêm một file client_secret từ GCP project khác |
| Nối lại kênh sau ~7 ngày | app còn ở trạng thái *Testing* — xem cảnh báo ở bước 3 |
| `invalid_client` khi đăng | file client_secret đã cấp kênh đó không còn trong `secrets/`. Khôi phục file, hoặc nối lại kênh |

## Thêm quota sau này

Lặp lại từ bước 1 với một GCP project **mới**, thả file JSON thứ hai vào
`secrets/`, restart publisher. GUI sẽ hỏi chọn client mỗi lần bấm "Thêm kênh".
