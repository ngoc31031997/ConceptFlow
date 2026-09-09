# ADR-0024: Edge Read Aloud thay Piper làm engine giọng đọc nền

## Status
Accepted

## Date
2026-09-09

## Stage
Low-Level Design (Unit 3 — TTS Service)

## Context

ADR-0023 chọn Google Cloud TTS làm engine mặc định với lập luận cốt lõi: **free tier WaveNet 4 triệu ký tự/tháng** phủ hết nhu cầu (~240.000 ký tự/tháng), nên chi phí thực tế là $0. Piper ở lại làm fallback offline.

Ngày 2026-09-08, Creator xác minh trực tiếp trên Google Cloud Console: **free tier đó không còn tồn tại**. Điều này làm sụp tiền đề chính của ADR-0023 — nhánh Google giờ tính tiền từ ký tự đầu tiên.

Ba lần thử mở billing account trên Google (và một vòng thử Azure Neural TTS trước đó) đều bị chặn ở tầng hạ tầng tài khoản: billing account bị đóng tự động ngay khi enable, cuối cùng Google yêu cầu prepayment ₫800.000 để gỡ. Creator quyết định **không đi tiếp đường cloud có tài khoản** ở thời điểm này.

Trong khi đó vấn đề gốc mà ADR-0023 sinh ra để giải quyết vẫn nguyên vẹn: **giọng Piper là rủi ro monetization lớn nhất của pipeline**. Giọng nam tiếng Việt duy nhất Piper phát hành là `vi_VN-vivos-x_low` ở mức chất lượng `x_low` (CR-001 §C1), đúng loại "giọng đọc máy" mà chính sách inauthentic content của YouTube nhắm tới.

## Options Considered

### Option A: Giữ Piper, chấp nhận chất lượng thấp
- What it is: quay về trạng thái trước ADR-0023, bỏ hẳn tham vọng giọng chất lượng cao.
- Strengths: chạy hoàn toàn offline, miễn phí thật, không phụ thuộc ai.
- Trade-offs: không giải quyết rủi ro monetization — đúng vấn đề đã khiến ADR-0023 ra đời.

### Option B: Trả tiền Google/Azure
- What it is: dùng cloud TTS chính danh, chấp nhận chi phí ~$4/tháng và khống chế bằng spend cap.
- Strengths: hợp pháp rõ ràng, có SLA, chất lượng cao.
- Trade-offs: bị chặn ở tầng tài khoản (billing account tự đóng ×3, yêu cầu prepayment ₫800.000); Azure thì vướng lỗi đăng nhập tenant. Không khả thi ở thời điểm quyết định.

### Option C: Edge Read Aloud (edge-tts) thay Piper
- What it is: dùng chính các giọng neural mà Azure bán, qua endpoint Edge dùng cho tính năng đọc to, không cần tài khoản/key.
- Strengths: chất lượng ngang Azure Neural, có cả giọng nam (`vi-VN-NamMinhNeural`) lẫn nữ (`vi-VN-HoaiMyNeural`) tiếng Việt; không tài khoản, không key, không chi phí.
- Trade-offs: **vi phạm ToS của Microsoft khi dùng thương mại**; endpoint không chính thức, không SLA, có thể bị chặn bất kỳ lúc nào; cần mạng nên mất khả năng chạy offline.

### Option D: Model mã nguồn mở tự host (VibeVoice, CosyVoice, GPT-SoVITS)
- What it is: chạy model TTS hiện đại ngay trong hạ tầng của mình.
- Strengths: miễn phí thật, không ToS, không phụ thuộc dịch vụ ngoài, giữ được nguyên tắc offline.
- Trade-offs: cần GPU để không quá chậm; chất lượng tiếng Việt chưa kiểm chứng; tích hợp nặng hơn nhiều.

## Decision

**Dùng Edge Read Aloud (`edge-tts`) làm engine nền, thay thế hoàn toàn Piper.** Piper bị gỡ khỏi codebase và khỏi image. Google giữ nguyên dạng adapter ngủ đông (chỉ chạy nếu có `GOOGLE_APPLICATION_CREDENTIALS`).

Creator được cảnh báo rõ về rủi ro ToS và độ ổn định trước khi chốt, và chọn Option C với hiểu biết đó.

## Rationale

Option A không giải quyết vấn đề. Option B bất khả thi ở tầng tài khoản tại thời điểm này. Option D là phương án đúng về lâu dài nhưng chi phí tích hợp lớn hơn nhiều so với giá trị nó mang lại ngay bây giờ.

Option C là phương án duy nhất **giải quyết được rủi ro monetization ngay lập tức mà không cần tài khoản nào**, và nó tái sử dụng đúng `TTSEnginePort` mà ADR-0002/ADR-0010 đã dựng sẵn — thay engine không đụng domain hay application layer.

## Consequences

- **Positive**: Giọng tiếng Việt lên hạng neural cho cả nam lẫn nữ, xoá rủi ro monetization lớn nhất của pipeline mà không tốn đồng nào và không cần tài khoản cloud.
- **Negative / Accepted Trade-offs**:
  - **Mất hoàn toàn khả năng chạy offline.** Đây là hệ quả có ý thức: ADR-0023 đã hạ nguyên tắc local-only xuống "Piper là fallback", ADR này bỏ nốt lớp đó. Không còn engine nào chạy được khi mất mạng.
  - **Không còn engine để degrade sang.** Khi Edge lỗi, không có gì đỡ — nên retry trong adapter là thành phần chịu lực, không phải tiện ích.
  - **Rủi ro ToS cho mục đích thương mại.** Microsoft không tài liệu hoá endpoint này như API công khai. Đây là rủi ro Creator chấp nhận có ý thức.
  - **Không có SLA.** Đo được trên máy Creator: gọi liên tiếp theo kiểu render nhiều scene chỉ đạt **1/8 thành công**; giãn cách 3s vẫn chỉ 2/5. Với retry có backoff (4 lần, backoff 1.5s×n) đạt **8/8**. Đây là lý do retry nằm trong `EdgeTTSAdapter`.
  - **Endpoint treo request, không chỉ từ chối.** Creator đối chiếu issue/discussion của repo edge-tts (2026-09-09): không có con số rate-limit chính thức nào được công bố, nhưng 403/503 và request treo lâu khi gọi liên tục là hiện tượng có thật đã được ghi nhận. Vì vậy mỗi lần gọi bị chặn trần riêng (`connect_timeout` 10s, `receive_timeout` 30s) — nếu không, một request treo sẽ ăn hết trần tổng và **retry không bao giờ chạy**, đúng thứ mà adapter này sinh ra để chống. Trần tổng (`SYNTHESIS_TIMEOUT_SECONDS`) được tính ra từ các con số đó thay vì đặt tay, và có test khoá bất biến này.
  - **Thêm phụ thuộc ffmpeg trong image TTS**, vì Edge trả MP3 còn pipeline đọc WAV.
- **Follow-ups**:
  - Nếu Microsoft siết endpoint (đã có dấu hiệu: token chống-abuse ngắn hạn, chặn dải IP datacenter), pipeline sẽ mất giọng đọc hoàn toàn. Option D là đường lui đã phân tích sẵn.
  - Nếu chạy stack trên VPS/cloud thay vì máy cá nhân, cần kiểm tra lại ngay: dải IP datacenter là thứ Microsoft chặn trước tiên.

## Related
- Design artifact: `aidlc-docs/inception/requirements/cr-010-edge-tts-engine.md`
- ADR-0010 — chọn Piper cho MVP. **Bị ADR này thay thế**: Piper không còn trong hệ thống.
- ADR-0023 — chọn Google Cloud TTS. Vẫn còn hiệu lực ở phần "Google là adapter tuỳ chọn", nhưng tiền đề free tier của nó đã sai và Piper-làm-fallback bị ADR này thay bằng Edge.
- ADR-0002 — Hexagonal: `TTSEnginePort` là thứ khiến lần thay engine này không chạm domain.
