# CR-001 — Tuỳ chọn giọng đọc TTS / Phụ đề & Chọn giọng đọc có nghe thử

## Date
2026-09-06

## Stage
Requirements Analysis (Change Request trên requirements.md đã được duyệt)

## Intent Analysis
Creator dán script Manim ở trang "Tạo video mới" và muốn tự quyết định video có giọng đọc TTS hay không, có phụ đề hay không, và nếu bật TTS thì được chọn giọng đọc cụ thể sau khi nghe thử. Hiện tại hệ thống luôn bắt buộc sinh TTS (FR4.1), mỗi ngôn ngữ chỉ có đúng 1 model giọng (`vi.onnx` / `en.onnx`), và **chưa có phụ đề ở bất kỳ đâu trong pipeline**.

## Quyết định đã chốt với người dùng
1. Khi tắt TTS: video **câm** (không giọng đọc), vẫn có phụ đề (nếu bật) và nhạc nền (nếu chọn). Cho phép tắt cả hai.
2. Số giọng đọc: **4 giọng tổng cộng** — Việt (nam/nữ) + Anh (nam/nữ), ưu tiên **độ chính xác phát âm/chính tả** hơn số lượng. Xem bảng chốt bên dưới (thay cho đề xuất "4+/ngôn ngữ" ban đầu — không khả thi cho tiếng Việt, xem C1).
3. Nghe thử: dùng **file audio mẫu tĩnh** (câu chào cố định) sinh sẵn lúc build/deploy, không gọi TTS realtime.
4. Style phụ đề (font, cỡ chữ, màu, nền mờ, vị trí): **cho người dùng tuỳ chỉnh**, không cố định.

## Functional Requirements (bổ sung)

### FR4 — Text-to-Speech (sửa đổi)
- **FR4.1 (sửa)**: TTS trở thành **tuỳ chọn**. Người dùng bật/tắt giọng đọc cho từng video ở bước tạo video.
- **FR4.4 (mới)**: Khi bật TTS, người dùng PHẢI chọn được một giọng đọc cụ thể trong danh sách giọng của ngôn ngữ đang chọn (không còn cố định 1 giọng/ngôn ngữ).
- **FR4.5 (mới)**: Mỗi giọng trong danh sách PHẢI có nút nghe thử phát file audio mẫu tĩnh; panel nghe thử hiển thị tên giọng, giới tính/phong cách, và mức chất lượng (x_low/low/medium/high).
- **FR4.6 (mới)**: Khi tắt TTS, hệ thống KHÔNG sinh audio giọng đọc và KHÔNG gửi job tới TTS Service.

### FR9 — Phụ đề (mới)
- **FR9.1**: Người dùng bật/tắt phụ đề cho từng video, độc lập với công tắc TTS.
- **FR9.2**: Khi bật, phụ đề hiển thị ở phần dưới khung hình, lấy nội dung từ các dòng `# NARRATION` trong script.
- **FR9.3**: Timing phụ đề:
  - Khi **bật TTS**: khớp với thời lượng audio thực tế của từng đoạn narration.
  - Khi **tắt TTS**: dùng thời lượng **ước lượng theo số ký tự/từ** của narration (cùng giá trị được dùng để thay `self.wait(AUTO)`), đảm bảo phụ đề và animation vẫn khớp nhau.
- **FR9.4 (mới)**: Người dùng tuỳ chỉnh được **style phụ đề**: cỡ chữ, màu chữ, nền mờ (bật/tắt + độ mờ), vị trí (dưới/trên khung hình). Panel style hiển thị preview trực quan trước khi render. Giá trị mặc định hợp lý (chữ trắng, nền đen mờ 60%, cỡ vừa, dưới khung hình) nếu người dùng không chỉnh.

### FR3/FR5 — Ảnh hưởng dây chuyền
- **FR3.4 (mới)**: Rendering Service hiện thay mỗi `self.wait(AUTO)` bằng thời lượng audio TTS. Khi tắt TTS, PHẢI thay bằng **thời lượng ước lượng** từ text narration (không được để render lỗi vì thiếu audio duration).
- **FR5.4 (mới)**: Video Assembly PHẢI ghép được video không có track giọng đọc (chỉ nhạc nền, hoặc hoàn toàn câm) và PHẢI burn-in phụ đề khi được bật.

## Non-Functional / Ràng buộc

- **C1 — Số giọng tiếng Việt bị giới hạn bởi upstream**: kho `rhasspy/piper-voices` chỉ có **3** giọng tiếng Việt (`vais1000-medium`, `25hours_single-low`, `vivos-x_low`), không đủ 4+. Tiếng Anh có hàng chục giọng nên sẽ chọn 4–6 giọng tốt nhất. → Đề xuất: lấy **cả 3 giọng vi** + **4–6 giọng en**, và nếu muốn thêm giọng tiếng Việt chất lượng cao hơn thì cần engine khác (xem C2).
- **C2 — Mở rộng ngoài Piper (không thuộc phạm vi CR này)**: nếu cần giọng tiếng Việt tự nhiên hơn, các engine local free khác: **Coqui TTS / XTTS-v2** (đa ngôn ngữ, voice cloning, nặng, cần GPU để nhanh), **Kokoro TTS** (82M params, rất nhẹ và tự nhiên, chưa hỗ trợ tiếng Việt chính thức), **F5-TTS**, **viXTTS** (bản fine-tune tiếng Việt của XTTS-v2). ADR-0010 đã thiết kế `TTSEnginePort` để thêm adapter mà không sửa domain — nên đây là bước mở rộng sau, không làm trong CR này.
- **C3 — Dung lượng model**: mỗi model Piper medium ~60MB, low/x_low ~20MB. Với ~9 giọng, tổng ~300–400MB tải về khi build image.
- **C4 — Tương thích ngược**: `RenderInput` hiện có `voice_language`; các trường mới (`tts_enabled`, `voice_id`, `subtitles_enabled`) phải có giá trị mặc định để project cũ và API client cũ không vỡ.

## Danh sách giọng CHỐT (Piper, local, miễn phí, license MIT/CC — 4 giọng, ưu tiên độ chính xác)

| Ngôn ngữ | Voice ID | Giới tính | Chất lượng | Lý do chọn |
|---|---|---|---|---|
| vi | `vi_VN-vais1000-medium` | Nữ | **medium** (cao nhất có cho vi) | Dataset thu âm phòng thu (VAIS), phát âm/chính tả chuẩn nhất trong các giọng vi hiện có |
| vi | `vi_VN-vivos-x_low` | Nam | x_low | **Giọng nam duy nhất có sẵn cho tiếng Việt** trong kho Piper chính thức — chấp nhận chất lượng thấp hơn vì không có lựa chọn nam nào khác (xem C1) |
| en | `en_US-lessac-medium` | Nữ | medium | Dataset đọc sách rõ ràng, ít lỗi phát âm, ổn định |
| en | `en_US-ryan-high` | Nam | **high** | Chất lượng/độ chính xác cao nhất trong các giọng nam tiếng Anh của Piper |

**Lưu ý về giọng nam tiếng Việt**: kho `rhasspy/piper-voices` hiện KHÔNG có giọng nam tiếng Việt nào ở mức medium/high — `vivos-x_low` là lựa chọn duy nhất và có chất lượng phát âm thấp hơn hẳn 3 giọng còn lại. Nếu độ chính xác của giọng nam tiếng Việt là yêu cầu bắt buộc, cần engine khác (viXTTS/Coqui, xem C2) — nằm ngoài phạm vi CR này vì tốn thêm effort đáng kể (GPU, model lớn hơn).

## Phạm vi tác động (unit)
- **Web GUI**: 2 công tắc mới + panel chọn/nghe thử giọng, `ProjectDraftContext`, `RenderInput`.
- **API Gateway**: mở rộng payload start-saga.
- **Orchestrator**: bỏ qua bước TTS khi tắt; truyền duration ước lượng xuống Rendering.
- **TTS Service**: voice registry theo `voice_id` thay vì theo language; sinh sẵn audio mẫu; API liệt kê giọng.
- **Script Processing**: hàm ước lượng thời lượng narration; xuất cue phụ đề.
- **Rendering**: fallback duration khi không có TTS.
- **Video Assembly**: burn-in phụ đề; ghép video không có track giọng đọc.

## Câu hỏi đã chốt
1. ~~Style phụ đề~~ → **cho người dùng tuỳ chỉnh** (FR9.4).
2. ~~Số/chọn giọng~~ → **4 giọng cố định** theo bảng trên, ưu tiên độ chính xác.

## Câu hỏi còn mở (không chặn Design, có thể quyết trong Low-Level Design)
1. Tốc độ ước lượng khi tắt TTS — mặc định ~150 từ/phút (en) và ~140 từ/phút (vi); không cho người dùng chỉnh (giữ đơn giản cho MVP), chỉ Rendering/Script Processing dùng nội bộ.
