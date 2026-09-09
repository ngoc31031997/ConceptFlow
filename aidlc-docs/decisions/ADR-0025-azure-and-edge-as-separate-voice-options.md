# ADR-0025: Azure và Edge là hai lựa chọn giọng riêng, không thay thế ngầm cho nhau

## Status
Accepted

## Date
2026-09-09

## Stage
Low-Level Design (Unit 3 — TTS Service)

## Context

CR-011 đưa Azure AI Speech vào cạnh edge-tts. Vấn đề: **hai engine phát chính xác cùng một bộ giọng neural** — `vi-VN-NamMinhNeural`, `vi-VN-HoaiMyNeural`, `en-US-JennyNeural`, `en-US-GuyNeural` — vì edge-tts thực chất là đường vào không chính thức tới cùng model Azure bán.

Chúng giống hệt nhau về **âm thanh**, nhưng khác hẳn về **bảo đảm**:

| | edge-tts | Azure F0 |
|---|---|---|
| Tài khoản | Không cần | Cần key + region |
| Chi phí | $0 | $0 tới 500k ký tự/tháng |
| ToS thương mại | Vi phạm | Hợp lệ |
| SLA | Không | Có |
| Độ ổn định đo được | 1/8 khi gọi liên tiếp | Chưa đo |

`voice_registry._BY_ID` key theo `voice_id`, nên không thể đăng ký hai voice trùng tên. Phải chọn một cách xử lý.

## Options Considered

### Option A: Azure là đường ưu tiên cho cùng 4 giọng, Edge là fallback
- What it is: danh mục vẫn 4 giọng neural. Có credential thì gọi Azure, Azure lỗi thì tự rơi về Edge.
- Strengths: GUI không đổi, không có mục nào nghe giống mục nào; tự động dùng đường tốt nhất đang có; vá đúng lỗ hổng "không còn gì để degrade sang" của ADR-0024.
- Trade-offs: Creator không thấy được đường nào đang thực sự phát; muốn ép dùng Edge (khi Azure hết quota) thì không có cách.

### Option B: Azure là các giọng riêng trong danh sách
- What it is: danh mục 8 giọng neural — 4 Edge + 4 Azure có tiền tố, nhãn `(Azure)`.
- Strengths: Creator kiểm soát trực tiếp, thấy rõ mình đang dùng đường nào; chọn Azure cho video đăng kiếm tiền và Edge cho bản nháp là chuyện làm được.
- Trade-offs: hai nhóm **nghe giống hệt nhau**, dễ gây rối khi chọn; cần tiền tố `azure:` cho voice ID để tránh trùng khóa.

### Option C: Azure thay hẳn edge-tts
- What it is: bỏ edge-tts, chỉ dùng Azure chính danh.
- Strengths: sạch về ToS, có SLA.
- Trade-offs: quay lại trạng thái không có fallback; vượt 500k ký tự/tháng là bắt đầu tính tiền; và phụ thuộc hoàn toàn vào một tài khoản cloud — đúng thứ vừa gây tắc suốt CR-009/CR-010.

## Decision

**Option B — Azure và Edge là hai lựa chọn riêng trong danh mục giọng**, phân biệt bằng tiền tố `azure:` ở `voice_id` và hậu tố `(Azure)` ở nhãn.

Creator chọn phương án này khi được hỏi trực tiếp (2026-09-09), sau khi Option A được nêu là khuyến nghị.

## Rationale

Điểm mấu chốt: hai engine khác nhau ở **bảo đảm**, không ở âm thanh — và bảo đảm là thứ chỉ Creator mới quyết được, vì nó phụ thuộc video đó dùng làm gì. Một video đăng để kiếm tiền cần đường hợp ToS có SLA; một bản nháp để xem thử thì không.

Option A giấu lựa chọn đó sau logic tự động, nghĩa là Creator không thể ép đường nào cho video nào. Option C bỏ mất lớp dự phòng vừa mới xây.

Tiền tố `azure:` là cái giá phải trả, và nó rẻ: chỉ sống trong `voice_registry` và bị strip ngay tại `AzureTTSAdapter` trước khi gửi đi.

## Consequences

- **Positive**:
  - Creator kiểm soát trực tiếp đường phát cho từng project.
  - Chọn voice Azure mà thiếu credential vẫn render được: rơi về voice Edge **cùng ngôn ngữ**, kèm cảnh báo nêu đúng biến env cần đặt.
  - Azure lỗi giữa chừng cũng rơi về Edge — lớp fallback mà ADR-0024 ghi nhận là đã mất nay có lại, ít nhất cho nhánh Azure.
- **Negative / Accepted Trade-offs**:
  - **Danh mục có 8 mục neural nghe giống hệt nhau.** Nhãn `(Azure)` là thứ duy nhất phân biệt; nếu Creator sau này thấy rối thì Option A vẫn là đường lui, và đổi được mà không đụng domain.
  - Voice ID Azure không phải tên Azure thật — mọi code chạm tới nó phải qua `azure_voice_name()`. Đã có test khoá điều này.
  - `RoutingTTSEngine` giờ mang map engine→adapter thay vì một slot; đổi lại nó không còn hard-code tên engine nào, nên engine thứ tư sẽ không cần sửa nó nữa.
- **Follow-ups**:
  - Chưa chạy với key Azure thật (key cố ý không đưa vào phiên làm việc). Cần Creator tự verify một lần.
  - Metering Azure đã tách ngưỡng 500k; nếu Creator dùng vượt, cảnh báo sẽ nổ ở 80% — nhưng không có gì chặn, đúng nguyên tắc của CR-005 FR13.5.

## Related
- Design artifact: `aidlc-docs/inception/requirements/cr-011-azure-tts-engine.md`
- ADR-0024 — edge-tts thay Piper; ADR này vá lớp fallback mà nó đánh mất.
- ADR-0023 — Google, vẫn ngủ đông; `RoutingTTSEngine` nay xử lý cả ba engine đồng nhất.
