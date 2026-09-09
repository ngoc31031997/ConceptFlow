# CR-010 — Edge TTS thay Piper làm engine giọng đọc nền (P1, thay thế CR-009)

## Date
2026-09-09

## Stage
Requirements Analysis → Code Generation (đã hoàn thành)

## Intent Analysis
- **Request type**: Migration (thay engine TTS nền, không đổi domain)
- **Scope estimate**: Single component (`services/tts/`), qua đúng `TTSEnginePort` sẵn có
- **Complexity estimate**: Simple — có tiền lệ trực tiếp là `GoogleTTSAdapter`

## Bối cảnh: vì sao CR-009 (Azure) bị bỏ

CR-009 định thêm Azure Neural TTS song song Google. Creator đã tạo Azure account, test `curl` thành công (200 OK, `vi-VN-NamMinhNeural`), nhưng sau đó đổi hướng vì:

1. **Google mất free tier** (xác minh trực tiếp trên Console 2026-09-08) — tiền đề chi phí $0 của ADR-0023 sai.
2. **Cả hai đường cloud đều tắc ở tầng tài khoản**: Azure vướng lỗi đăng nhập `AADSTS50020`; Google thì **3 billing account liên tiếp bị đóng tự động** ngay khi enable billing, cuối cùng yêu cầu prepayment ₫800.000.
3. Creator quyết định bỏ qua cả Google lẫn Azure ở thời điểm này, và chuyển sang edge-tts.

→ **CR-009 chuyển trạng thái Superseded**, không triển khai.

## Vấn đề gốc vẫn còn nguyên
Giọng Piper là rủi ro monetization lớn nhất của pipeline: `vi_VN-vivos-x_low` là giọng nam tiếng Việt duy nhất Piper có, ở mức `x_low` (CR-001 §C1, CR-005 §FR13).

## Quyết định của Creator (2026-09-09)
Được cảnh báo trước về rủi ro ToS và độ ổn định của edge-tts, Creator vẫn chọn:
- **edge-tts thay hoàn toàn Piper** (không giữ Piper làm fallback offline).
- Chấp nhận mất khả năng chạy offline.

## Functional Requirements

### FR23 — EdgeTTSAdapter
- **FR23.1**: PHẢI thêm `EdgeTTSAdapter` implement `TTSEnginePort`, không sửa domain/application. ✅
- **FR23.2**: Không cần credential/env var nào — đây là điểm khác biệt cốt lõi so với Google/Azure. ✅
- **FR23.3**: Output PHẢI là WAV 24 kHz mono để phần còn lại pipeline (ducking, loudnorm, `wave`) không phải đổi. Edge trả MP3 ⇒ transcode bằng ffmpeg. ✅
- **FR23.4**: PHẢI raise `TTSEngineError` khi thất bại, để `RoutingTTSEngine` là nơi duy nhất quyết định fallback. ✅
- **FR23.5**: PHẢI retry có backoff khi endpoint từ chối. **Bắt buộc, không phải tuỳ chọn**: đo thực tế cho thấy gọi liên tiếp kiểu render nhiều scene chỉ đạt 1/8 thành công; có retry đạt 8/8. Vì không còn Piper để degrade sang, một lần từ chối tạm thời sẽ giết cả render. ✅
- **FR23.6**: Mỗi lần gọi PHẢI có trần thời gian riêng (`connect_timeout`/`receive_timeout`), vì endpoint còn **treo** request chứ không chỉ từ chối. Nếu không, một request treo ăn hết trần tổng và retry không chạy lần nào. Trần tổng phải được suy ra từ ngân sách retry, không đặt tay. ✅

### FR24 — Gỡ Piper
- **FR24.1**: Xoá `PiperTTSAdapter`, binary Piper và 4 model `.onnx` khỏi image. ✅
- **FR24.2**: `voice_registry` thay voice Piper bằng 4 voice Edge (vi/en × nam/nữ). ✅
- **FR24.3**: `RoutingTTSEngine` đổi engine nền từ Piper sang Edge; nhánh Google giữ nguyên, fallback của nó giờ trỏ về Edge. ✅

### FR25 — Google ngủ đông
- **FR25.1**: Giữ `GoogleTTSAdapter` và các voice Google trong registry, chỉ hoạt động khi có `GOOGLE_APPLICATION_CREDENTIALS`. Không xoá — credential là thứ duy nhất nó còn thiếu. ✅

## Kết quả đo được (máy Creator, 2026-09-09)
| Kịch bản | Trước retry | Sau retry |
|---|---|---|
| 8 scene gọi liên tiếp | 1/8 thành công | **8/8 thành công**, 17.3s |
| 5 lần giãn cách 3s | 2/5 | — |
| 5 lần có retry+backoff | — | 5/5 |

## Non-goals
- Không đổi GUI chọn giọng (đã multi-voice từ CR-001).
- Không đổi ducking/loudnorm (CR-005 FR14).
- Không xoá `GoogleTTSAdapter`.

## ADR liên quan
- **ADR-0024** — quyết định chính, thay thế ADR-0010.
- CR-009 (`cr-009-azure-tts-engine.md`) — Superseded bởi CR này.
