# CR-009 — Thêm Azure Neural TTS làm engine cloud song song với Google (P1, kế thừa CR-005/ADR-0023)

> **Status: SUPERSEDED (2026-09-09) — không triển khai.**
> Creator dừng cả hướng Azure lẫn Google sau khi 3 billing account Google bị đóng tự động
> và Azure vướng lỗi đăng nhập tenant. Thay bằng edge-tts: xem `cr-010-edge-tts-engine.md`
> và ADR-0024.

## Date
2026-09-08

## Stage
Requirements Analysis (Change Request — sửa đổi trực tiếp quyết định của CR-005/ADR-0023)

## Intent Analysis
- **Request type**: Enhancement / Migration (thêm 1 engine TTS mới, không đổi domain)
- **Scope estimate**: Single component (TTS Service — `services/tts/`), theo đúng ranh giới `TTSEnginePort` đã có
- **Complexity estimate**: Simple — có tiền lệ trực tiếp (`GoogleTTSAdapter`) để soi theo, không cần High-Level/Application Design mới

## Vấn đề (đã xác minh trực tiếp bởi Creator, 2026-09-08)
Google Cloud Text-to-Speech đã **gỡ bỏ free tier** cho WaveNet — Creator xác nhận trực tiếp trên Console: trang "Enable API" của Cloud Text-to-Speech không còn hiển thị hạn mức miễn phí (chỉ còn dòng giá SKU trả phí, kể cả các SKU liên quan Gemini TTS). Toàn bộ lập luận "chi phí ~$0/tháng" trong ADR-0023 (dựa trên free tier 4.000.000 ký tự WaveNet/tháng) **không còn đúng**.

Creator đã tự đăng ký và test thành công **Azure AI Speech (Neural TTS)**:
- Resource pricing tier **F0 (Free)** — 500.000 ký tự Neural/tháng, **không hết hạn** (khác `$200 credit dùng 30 ngày` và `free tier 12 tháng` mà Azure free account cũng cấp — hai loại đó KHÔNG dùng cho case này).
- Test bằng `curl` REST endpoint (`https://<region>.tts.speech.microsoft.com/cognitiveservices/v1`, SSML, voice `vi-VN-NamMinhNeural`) → **200 OK**, nghe được audio.
- Nhu cầu thực tế (~8.000 ký tự/video × ~30 video/tháng ≈ 240.000 ký tự/tháng, theo CR-005 §C6) nằm gọn dưới hạn mức 500K free — tức **$0/tháng bền vững** thay vì "gần như $0" như Google từng hứa.

## Quyết định của Creator (đã hỏi trực tiếp, 2026-09-08)
1. **Giữ cả `GoogleTTSAdapter` lẫn `AzureTTSAdapter` song song trong code** — không xoá Google. Lý do Creator chọn: không mất khả năng quay lại Google nếu sau này họ có free tier lại; `RoutingTTSEngine` vốn đã được thiết kế theo `TTSEnginePort` để thêm engine không đụng domain (đúng tinh thần ADR-0023 §"Kiến trúc đã sẵn sàng").
2. **Voice registry phải cho chọn cả giọng nam/nữ, cả tiếng Việt/tiếng Anh** — không ép một giọng mặc định cứng. Nghĩa là cả 2 giọng Azure tiếng Việt (`vi-VN-NamMinhNeural`, `vi-VN-HoaiMyNeural`) và giọng Azure tiếng Anh tương ứng đều phải đăng ký vào `voice_registry`, đúng mô hình đa-giọng đã có với Google (CR-001 FR13.2).

## Functional Requirements

### FR20 — AzureTTSAdapter (mới)
- **FR20.1**: PHẢI thêm `AzureTTSAdapter` implement `TTSEnginePort`, cắm qua interface có sẵn — không sửa domain/application, giống nguyên tắc ADR-0023 đã áp dụng cho Google.
- **FR20.2**: Credential lấy từ 2 biến env mới, **tách hoàn toàn khỏi Google và khỏi OAuth YouTube** (khác scope, khác vòng đời — cùng nguyên tắc ADR-0023): `AZURE_SPEECH_KEY`, `AZURE_SPEECH_REGION`.
- **FR20.3**: Gọi REST endpoint `https://<region>.tts.speech.microsoft.com/cognitiveservices/v1` với SSML, output `audio-24khz-...-mono` — khớp sample rate/mono convention hiện có của `GoogleTTSAdapter` để phần còn lại pipeline (ducking/loudnorm) không cần đổi.
- **FR20.4**: PHẢI raise `TTSEngineError` khi thất bại (timeout, lỗi network, lỗi key) — không tự silent-fallback bên trong adapter, để `RoutingTTSEngine` là nơi duy nhất quyết định fallback (đúng phân tách trách nhiệm hiện có giữa `GoogleTTSAdapter` và `RoutingTTSEngine`).

### FR21 — Routing đa-cloud-engine (thay đổi `RoutingTTSEngine`)
- **FR21.1**: `RoutingTTSEngine` hiện hard-code 1 slot "google" — PHẢI tổng quát hoá để nhận nhiều cloud engine (map `engine_name -> adapter`), vì giờ có 2 engine cloud (Google + Azure) cùng tồn tại thay vì 1.
- **FR21.2**: Fallback Piper khi thiếu credential hoặc lỗi mạng PHẢI áp dụng cho **cả Google lẫn Azure**, kèm cảnh báo rõ ràng theo đúng voice đang dùng — không im lặng (giữ nguyên tinh thần CR-005 FR13.3).
- **FR21.3**: Usage metering (`_record_usage`, cảnh báo 80% hạn mức) PHẢI tách theo từng engine, vì Azure free tier (500K/tháng, F0) khác hẳn ngưỡng Google cũ (4M/tháng) — không dùng chung 1 ngưỡng `FREE_TIER_CHARACTERS`.

### FR22 — Voice registry (mở rộng `voice_registry.py`)
- **FR22.1**: Thêm `ENGINE_AZURE` và 4 voice: `vi-VN-NamMinhNeural` (nam), `vi-VN-HoaiMyNeural` (nữ), cùng 2 giọng Azure tiếng Anh tương ứng nam/nữ.
- **FR22.2**: `DEFAULT_VOICE_BY_LANGUAGE` chuyển sang trỏ về voice Azure (vì Azure free tier còn dùng được, Google thì không) — nhưng voice Google **vẫn giữ trong `VOICES`** để Creator chọn thủ công nếu muốn (theo quyết định #1 ở trên).
- **FR22.3**: `fallback_voice_for` PHẢI hoạt động đúng cho voice thuộc cả 2 engine cloud, fallback về đúng Piper voice cùng ngôn ngữ (logic hiện có theo `language`, không cần đổi nhiều).

## Non-goals (out of scope cho CR-009)
- Không đổi giao diện GUI chọn giọng (đã hỗ trợ multi-voice từ CR-001).
- Không xoá `GoogleTTSAdapter` hay các Google voice khỏi registry (theo quyết định Creator).
- Không đổi cơ chế ducking/loudnorm (CR-005 FR14) — chỉ cần Azure trả về đúng format audio tương thích.

## ADR liên quan
Quyết định đổi engine mặc định (Google → Azure) là quyết định vendor/chi phí có thể tốn kém để đảo ngược sau này → cần một ADR mới thay thế ADR-0023 (không sửa/xoá ADR-0023 gốc, theo nguyên tắc "never edit a past ADR's original content"). Sẽ tạo `ADR-0024` sau khi requirements này được duyệt.

## Liên quan
- CR-005 (`cr-005-audio-quality.md`) — nơi FR13 gốc được định nghĩa; CR-009 sửa đổi trực tiếp giả định chi phí của CR-005.
- ADR-0023 — quyết định gốc chọn Google; sẽ bị supersede bởi ADR-0024.
