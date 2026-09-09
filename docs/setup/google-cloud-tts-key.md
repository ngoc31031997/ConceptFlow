# Lấy key Google Cloud Text-to-Speech

Dùng cho: giọng WaveNet của Google trong danh mục giọng (ADR-0023).

> **Không khuyến nghị dùng.** Google đã bỏ free tier WaveNet mà nhánh này được
> chọn vì nó (CR-010), nên **mọi ký tự gửi lên đều bị tính tiền**. Nhánh code vẫn
> còn nhưng ở trạng thái ngủ. Muốn giọng chất lượng miễn phí thì dùng Edge (không
> cần gì cả) hoặc Azure F0 (xem [azure-speech-key.md](azure-speech-key.md)).

Cần: `GOOGLE_TTS_CREDENTIALS_FILE` trong `.env` — đường dẫn tuyệt đối **trên máy
host** tới file service-account JSON.

---

## 1. Bật API

1. https://console.cloud.google.com → chọn (hoặc tạo) project
2. **APIs & Services** → **Library** → tìm **Cloud Text-to-Speech API** → **ENABLE**
3. API này đòi **billing account**. Chưa có thì Console sẽ bắt liên kết thẻ ở bước này.

## 2. Tạo service account

Khác với YouTube: đây là **service account** (máy tự gọi), không phải OAuth client
(người bấm đồng ý). Đừng nhầm hai loại — file JSON của chúng khác nhau.

1. **IAM & Admin** → **Service Accounts** → **+ CREATE SERVICE ACCOUNT**
2. Đặt tên, ví dụ `conceptflow-tts` → **CREATE AND CONTINUE**
3. Role: không cần cấp gì thêm để gọi TTS → **CONTINUE** → **DONE**

## 3. Tạo key JSON

1. Bấm vào service account vừa tạo
2. Tab **KEYS** → **ADD KEY** → **Create new key** → chọn **JSON** → **CREATE**
3. File tự tải về. **Đây là lần duy nhất tải được** — Google không cho xem lại.

## 4. Cất file và khai báo

Để **ngoài repo** cho chắc:

```bash
mkdir -p ~/secrets
mv ~/Downloads/<project>-<hash>.json ~/secrets/google-tts.json
chmod 600 ~/secrets/google-tts.json
```

Trong `.env`:

```
GOOGLE_TTS_CREDENTIALS_FILE=/Users/<tên bạn>/secrets/google-tts.json
```

Phải là **đường dẫn tuyệt đối**, không dùng `~`. Docker Compose mount file này
read-only vào container tại `/run/secrets/google-tts.json`, và biến
`GOOGLE_APPLICATION_CREDENTIALS` bên trong container trỏ tới đó.

## 5. Áp dụng

```bash
docker compose up -d tts
```

## 6. Kiểm tra

```bash
docker compose exec tts printenv GOOGLE_APPLICATION_CREDENTIALS
docker compose exec tts ls -l /run/secrets/google-tts.json
```

Cả hai phải có kết quả. Nếu biến rỗng nghĩa là `.env` chưa đặt — Compose cố ý để
biến rỗng khi không cấu hình, và mount `/dev/null` vào chỗ đó, để `docker compose`
vẫn validate được khi bạn không dùng Google.

Sau đó mở danh sách giọng trong ứng dụng: các giọng WaveNet chỉ hiện khi
credential hợp lệ.

## Bỏ dùng

Xoá (hoặc comment) dòng `GOOGLE_TTS_CREDENTIALS_FILE` trong `.env` rồi
`docker compose up -d tts`. Giọng Google biến khỏi danh mục, project cũ đã lưu
voice_id Google sẽ tự rơi về giọng Edge cùng ngôn ngữ.

Nên xoá luôn key trên Console: **Service Accounts** → service account →
**KEYS** → xoá key. Key còn sống là còn có thể bị tiêu tiền.
