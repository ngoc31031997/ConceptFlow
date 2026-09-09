# Hướng dẫn lấy credential

Mỗi file là một quy trình đầy đủ, đọc là làm được, không cần hỏi lại.

| Hướng dẫn | Dùng cho | Bắt buộc? | Chi phí |
|---|---|---|---|
| [youtube-oauth-client.md](youtube-oauth-client.md) | Đăng video lên YouTube | **Có**, nếu muốn đăng tự động | Miễn phí (giới hạn ~6 video/ngày mỗi client) |
| [azure-speech-key.md](azure-speech-key.md) | Giọng đọc Azure | Không | Miễn phí — F0: 500k ký tự/tháng |
| [google-cloud-tts-key.md](google-cloud-tts-key.md) | Giọng đọc Google WaveNet | Không | **Tính tiền mọi ký tự** — không khuyến nghị |

## Những thứ KHÔNG cần key

- **Giọng đọc Edge** — engine giọng mặc định (ADR-0024). Không tài khoản, không key,
  không cấu hình. Đây là lý do hệ thống chạy được ngay khi chưa có credential nào.
- **LLM (Ollama)** — chạy local trong container `ollama`, mô hình đặt bằng
  `OLLAMA_MODEL` trong `.env` (mặc định `llama3.2`). Không gọi API bên ngoài,
  không API key.
- **RabbitMQ / PostgreSQL / Grafana** — user/password do bạn tự đặt trong `.env`,
  không phải đi xin ở đâu.

## Nguyên tắc chung

- File secret **không bao giờ** vào Git. `secrets/*.json` và `.env` đã được
  `.gitignore` — kiểm tra bằng `git check-ignore -v <đường dẫn file>` nếu nghi ngờ.
- Đổi `.env` hoặc thêm file secret rồi thì phải **restart container** liên quan;
  các service đọc cấu hình lúc khởi động.
- Secret lỡ dán vào chat, ảnh chụp màn hình, hay commit thì coi như đã lộ — xoay
  (rotate) nó, đừng chỉ xoá đi chỗ hiển thị. Mỗi hướng dẫn đều có mục xoay key.

## Tổng hợp biến trong `.env`

| Biến | Hướng dẫn |
|---|---|
| `GOOGLE_OAUTH_REDIRECT_URI` | [youtube-oauth-client.md](youtube-oauth-client.md) |
| *(file `secrets/client_secret*.json`)* | [youtube-oauth-client.md](youtube-oauth-client.md) |
| `AZURE_SPEECH_KEY`, `AZURE_SPEECH_REGION` | [azure-speech-key.md](azure-speech-key.md) |
| `GOOGLE_TTS_CREDENTIALS_FILE` | [google-cloud-tts-key.md](google-cloud-tts-key.md) |
| `RABBITMQ_USER/PASS`, `POSTGRES_USER/PASS`, `GRAFANA_USER/PASS` | tự đặt, xem `.env.example` |
| `OLLAMA_MODEL` | tuỳ chọn, mặc định `llama3.2` |
