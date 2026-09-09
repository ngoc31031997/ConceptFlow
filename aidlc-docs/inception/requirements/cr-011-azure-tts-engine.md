# CR-011 — Thêm Azure AI Speech làm engine thứ ba (P1, hồi sinh CR-009)

## Date
2026-09-09

## Stage
Requirements Analysis → Code Generation (đã hoàn thành)

## Intent Analysis
- **Request type**: Enhancement (thêm engine, không đổi domain)
- **Scope estimate**: Single component (`services/tts/`), qua `TTSEnginePort` sẵn có
- **Complexity estimate**: Simple — có hai tiền lệ (`GoogleTTSAdapter`, `EdgeTTSAdapter`)

## Bối cảnh
CR-010 đưa edge-tts vào làm engine nền và gỡ Piper. Điều đó giải quyết chất lượng giọng nhưng để lại ba rủi ro đã ghi trong ADR-0024: vi phạm ToS khi dùng thương mại, không SLA, và **không còn engine nào để degrade sang**.

Creator vẫn còn Azure Speech resource F0 đã tạo và test thành công ở CR-009 (curl → 200 OK, `vi-VN-NamMinhNeural`). CR này đưa nó vào hệ thống — không thay edge-tts, mà bổ sung.

## Vấn đề thiết kế đặc thù
**Voice ID của Azure và edge-tts trùng nhau hoàn toàn** (`vi-VN-NamMinhNeural`, `vi-VN-HoaiMyNeural`, `en-US-JennyNeural`, `en-US-GuyNeural`) — vì edge-tts phát chính bộ giọng neural Azure bán. `voice_registry._BY_ID` key theo `voice_id` nên không thể đăng ký trùng.

Đã hỏi Creator (2026-09-09), Creator chọn: **chia ra 2 option riêng trong danh sách giọng** — Azure và Edge hiển thị tách bạch để tự chọn. Xem ADR-0025 cho các phương án đã cân nhắc.

## Functional Requirements

### FR26 — AzureTTSAdapter
- **FR26.1**: PHẢI thêm `AzureTTSAdapter` implement `TTSEnginePort`, không sửa domain/application. ✅
- **FR26.2**: Credential từ `AZURE_SPEECH_KEY` + `AZURE_SPEECH_REGION`, tách hoàn toàn khỏi Google và khỏi OAuth YouTube (khác scope, khác vòng đời — nguyên tắc ADR-0023). ✅
- **FR26.3**: Gọi REST `https://<region>.tts.speech.microsoft.com/cognitiveservices/v1` với SSML. Output `riff-24khz-16bit-mono-pcm` — Azure trả thẳng WAV nên **không cần ffmpeg**, khác edge-tts. ✅
- **FR26.4**: PHẢI raise `TTSEngineError` khi thất bại (401/403 sai key, 429 hết quota, lỗi mạng, timeout) để `RoutingTTSEngine` là nơi duy nhất quyết định fallback. ✅
- **FR26.5**: Lời dẫn PHẢI được XML-escape trước khi nhúng vào SSML — đây là văn bản do Creator soạn, một ký tự `&` hay `<` sẽ làm Azure từ chối cả document. ✅

### FR27 — Voice registry hai lựa chọn
- **FR27.1**: Thêm `ENGINE_AZURE` và 4 voice Azure với tiền tố `azure:` để không đụng khóa với voice Edge trùng tên. ✅
- **FR27.2**: Nhãn phân biệt rõ cho Creator: `Tiếng Việt — Nam (Azure)` vs `Tiếng Việt — Nam`. ✅
- **FR27.3**: Adapter PHẢI strip tiền tố trước khi gửi lên Azure — tiền tố là chuyện nội bộ của danh mục. ✅

### FR28 — Routing đa engine
- **FR28.1**: `RoutingTTSEngine` bỏ slot `google` hard-code, chuyển sang map `engine -> adapter` (đúng FR21.1 của CR-009 cũ). ✅
- **FR28.2**: Chọn voice Azure mà không có credential ⇒ rơi về voice Edge **cùng ngôn ngữ**, kèm cảnh báo nêu đúng biến env cần đặt — không im lặng. ✅
- **FR28.3**: Azure lỗi giữa chừng ⇒ rơi về Edge, kèm cảnh báo. Đây chính là lớp fallback mà ADR-0024 ghi nhận là đã mất. ✅
- **FR28.4**: Metering PHẢI tách theo từng engine với ngưỡng riêng: Azure 500.000 ký tự/tháng (free tier F0 thật), Google 4.000.000 (nay chỉ là mốc cảnh báo tiêu tiền). Dùng chung một bộ đếm sẽ cảnh báo sai thời điểm cho cả hai. ✅

## Non-goals
- Không bỏ edge-tts (Creator chọn giữ cả hai).
- Không đổi GUI (danh mục giọng đã do `voice_registry` sinh ra từ CR-001).
- Không đổi ducking/loudnorm (CR-005 FR14).

## Kiểm chứng
- 69/69 unit test pass, gồm 9 test riêng cho `AzureTTSAdapter` (strip tiền tố, header/format gửi đi, XML-escape, 401/429/503 → `TTSEngineError`, locale suy từ tên voice) và 5 test cho nhánh Azure trong routing.
- `build_engine()` xác minh cả hai nhánh: không có credential → `metered: (none)` kèm cảnh báo; có credential → `metered: ['azure']`.
- **Chưa test với credential Azure thật** — key chưa từng được chia sẻ vào phiên làm việc (cố ý, để không lộ). Cần Creator tự chạy một lần với key thật.

## Liên quan
- **ADR-0025** — quyết định "hai lựa chọn riêng" thay vì Azure-thay-thế-ngầm.
- ADR-0024 / CR-010 — edge-tts; CR này vá lớp fallback mà ADR-0024 ghi nhận là đã mất.
- CR-009 — bản Azure trước đó, đã Superseded; CR-011 hiện thực hoá phần lớn FR của nó.
