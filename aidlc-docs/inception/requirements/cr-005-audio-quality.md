# CR-005 — Chất lượng âm thanh & giọng đọc (P1, quan trọng nhất cho monetization)

## Date
2026-09-07

## Stage
Requirements Analysis (Change Request — tiếp nối C2 của CR-001, nay đưa vào phạm vi)

## Intent Analysis
Giọng đọc là yếu tố **rủi ro cao nhất** khi xét bật kiếm tiền. Chính sách YouTube Partner Program về *Inauthentic / mass-produced content* nhắm thẳng vào video "giọng đọc máy + nội dung sinh tự động". Piper — đặc biệt các model `x_low`/`medium` tiếng Việt — nghe rõ là máy.

## Vấn đề (đã xác minh trong code)

1. **Giọng Piper chất lượng thấp** — `services/tts/adapters/tts_engines/voice_registry.py::VOICES` có `vi_VN-vivos-x_low` (chất lượng **x_low**, giọng nam tiếng Việt duy nhất) và `vi_VN-vais1000-medium`. CR-001 §C1/C2 đã ghi nhận giới hạn này và cố ý để ngoài phạm vi — CR này đưa nó vào.
2. **Nhạc nền cố định, không ducking** — `ffmpeg_assembler.py::BACKGROUND_MUSIC_VOLUME = 0.2` hardcode; `amix` trộn phẳng ⇒ nhạc đè lời khi to, và ngắt quãng khi không có narration.
3. **Không chuẩn hoá loudness** — không có `loudnorm` ở đâu trong pipeline. YouTube normalize về ~−14 LUFS; nguồn quá nhỏ ⇒ video nghe yếu hơn hẳn các kênh khác.
4. **`-shortest` rủi ro** — `ffmpeg_assembler.py::_run_pipeline` kết thúc bằng `-shortest`; sau khi CR-002 đặt narration đúng offset, câu kết có thể bị **cắt cụt**.
5. **Không có khoảng lặng đầu/cuối** — narration bắt đầu ngay frame 0, nghe gấp gáp.

## Functional Requirements

### FR13 — Engine giọng đọc chất lượng cao (mới)
- **FR13.1**: PHẢI thêm ít nhất một adapter TTS chất lượng cao ngoài Piper, cắm qua `TTSEnginePort` sẵn có (ADR-0010 đã thiết kế đúng cho việc này — không sửa domain).
- **FR13.2**: Creator chọn engine + giọng ở GUI; danh mục giọng (`voice_registry`) PHẢI mang thêm trường `engine`.
- **FR13.3**: API key/service account PHẢI lấy từ env, và PHẢI có fallback về Piper khi thiếu key hoặc lỗi mạng (kèm cảnh báo rõ ở GUI, không im lặng).
- **FR13.4**: Giữ nguyên `voice_id` + file nghe thử tĩnh (CR-001 FR4.5) cho mọi engine.

### FR14 — Trộn âm chuyên nghiệp (mới)
- **FR14.1**: Nhạc nền PHẢI tự động hạ khi có narration (sidechain ducking), thay vì volume phẳng.
- **FR14.2**: Mức volume nhạc nền PHẢI chỉnh được ở GUI (mặc định giữ 0.2).
- **FR14.3**: Track audio cuối PHẢI được chuẩn hoá `loudnorm=I=-14:TP=-1.5:LRA=11`.
- **FR14.4**: PHẢI thay `-shortest` bằng cơ chế giữ trọn audio (pad video bằng `tpad` nếu cần) — không bao giờ cắt mất câu kết.
- **FR14.5**: PHẢI chèn được khoảng lặng đầu/cuối cấu hình được (mặc định 0.5s / 1.5s).

## Ràng buộc / Lựa chọn engine

| Engine | Chất lượng tiếng Việt | Chi phí | Ghi chú |
|---|---|---|---|
| **Piper** (hiện tại) | Thấp–TB | Miễn phí, local | Giữ làm fallback/nháp |
| **Google Cloud TTS (Neural2/Studio)** ✅ **ĐÃ CHỌN** | Rất tốt | Trả phí theo ký tự, **có free tier hàng tháng** | Dễ tích hợp nhất, đã có sẵn hệ Google trong Publisher |
| **ElevenLabs** | Rất tốt, tự nhiên nhất | Trả phí | Chất lượng cao nhất, nhưng đắt cho video dài |
| **Azure Neural TTS** | Rất tốt | Trả phí | Nhiều giọng Việt |
| **viXTTS / F5-TTS** (local) | Tốt | Miễn phí, cần GPU | Nặng, chậm nếu không có GPU |

- **C1 — ĐÃ CHỐT (2026-09-07)**: dùng **Google Cloud Text-to-Speech**, khai thác free tier hàng tháng. Piper giữ lại làm engine fallback/nháp.
- **C2**: Chuyển sang engine cloud phá vỡ nguyên tắc "chạy hoàn toàn local qua Docker" ghi ở README — **cần ADR mới** (`ADR-0023-cloud-tts-engine.md`).
- **C3**: Ducking (`sidechaincompress`) yêu cầu track narration liên tục làm sidechain — **phụ thuộc CR-002** (offset đúng) mới có track narration hợp lệ.
- **C4**: Chất lượng giọng chỉ giảm rủi ro monetization, **không loại bỏ**. Yếu tố quyết định vẫn là giá trị nội dung và bình luận/phân tích gốc của Creator.

## Phạm vi tác động
- **TTS** (Unit 3): adapter engine mới, `voice_registry`, sinh file nghe thử.
- **Video Assembly** (Unit 6): ducking, loudnorm, tpad, padding.
- **Orchestrator** (Unit 8) + **Web GUI** (Unit 10) + **API Gateway** (Unit 9): chọn engine, volume nhạc.
- **ADR mới**: engine TTS cloud vs local-only.

## Tiêu chí nghiệm thu
1. Đo LUFS bản xuất bằng `ffmpeg -af ebur128`: −14 ±1 LUFS.
2. Nghe thử: nhạc nền tự hạ khi có lời, tự lên khi hết lời.
3. Câu narration cuối không bị cắt.
4. So sánh mù giọng mới vs Piper — giọng mới rõ ràng tự nhiên hơn.

## Quyết định đã chốt với Creator (2026-09-07)

**Engine: Google Cloud Text-to-Speech, dùng trong hạn mức free tier.**

Hệ quả và ràng buộc bổ sung:

- **C5 — Hạn mức free tier (tra cứu 2026-09-07)**: free tier của Google TTS không hết hạn, mức theo hạng giọng:

  | Hạng | Free/tháng | Giá sau đó | Giọng vi-VN |
  |---|---|---|---|
  | **WaveNet** ✅ chọn | **4.000.000 ký tự** | $4/1M | `vi-VN-Wavenet-A/B/C` |
  | Standard | 4.000.000 | $4/1M | `vi-VN-Standard-A/B/C/D` |
  | Neural2 | 1.000.000 | $16/1M | — |
  | Chirp 3: HD | 1.000.000 | $30/1M | có hỗ trợ vi-VN |

  Đầu 2026 WaveNet giảm giá xuống ngang Standard ⇒ **không còn lý do dùng Standard** (cùng hạn mức, cùng giá, chất lượng cao hơn).
  Con số PHẢI được xác minh lại trên trang pricing của Google tại thời điểm implement.
- **C6 — Ước lượng tiêu thụ**: narration cho 1 video 10 phút ≈ 1.300–1.500 từ ≈ **~8.000 ký tự**. Đây là con số cần dùng để đối chiếu với hạn mức free tier và chốt hạng giọng ở Low-Level Design.
- **C7 — Đếm ký tự & cảnh báo hạn mức**: hệ thống PHẢI đếm ký tự đã tổng hợp theo tháng và cảnh báo Creator ở GUI khi sắp chạm hạn mức. **Mức ưu tiên đã hạ** sau khi có số liệu C5/C6: 4M ký tự ≈ **500 video 10 phút/tháng**; nhịp đăng 1 video/ngày chỉ dùng ~6% hạn mức, và chi phí vượt hạn mức là **~3 cent/video 10 phút**. Do đó FR13.5 là tính năng "để yên tâm", KHÔNG phải hàng rào chống phát sinh chi phí — không được để nó chặn tiến độ CR-005.
- **C8 — Chống lãng phí quota**: PHẢI cache audio theo hash `(text, voice_id, engine)` trên `shared_artifacts` — render lại một project chỉ sửa vài câu KHÔNG được tổng hợp lại toàn bộ narration. Đây là **FR13.6 (mới)**, và cũng ăn khớp với mục tiêu tăng tốc render lại của CR-003 FR11.5.
- **C9 — Giọng tiếng Việt**: `vi-VN` có nhiều hạng giọng (Standard/WaveNet/Neural2 tuỳ thời điểm). Danh sách giọng thực tế PHẢI lấy động từ API `voices.list` khi build danh mục, KHÔNG hardcode voice id — tránh lặp lại vấn đề của CR-001 §C1.
- **C10 — Credentials**: service account key qua env (`GOOGLE_APPLICATION_CREDENTIALS`), thêm vào `.env.example` và README. Đây là credential thứ hai của dự án sau Google OAuth của Publisher — **không dùng chung**, khác scope hoàn toàn.

### FR13 — bổ sung
- **FR13.5 (mới)**: Đếm và hiển thị số ký tự TTS đã dùng trong tháng; cảnh báo khi sắp chạm hạn mức free tier.
- **FR13.6 (mới)**: Cache kết quả tổng hợp theo hash `(text, voice_id, engine)`; chỉ tổng hợp lại phần narration thực sự thay đổi.

## Câu hỏi còn mở
1. ~~Chọn hạng giọng nào~~ → **CHỐT: WaveNet** (`vi-VN-Wavenet-*`), theo bảng C5. Vẫn nghe thử giọng thật ở LLD để chọn giọng mặc định trong A/B/C.
2. Danh sách tra cứu chỉ thấy **3 giọng** WaveNet vi-VN (A/B/C) — chưa rõ có đủ cả nam lẫn nữ không. Không ảnh hưởng thiết kế vì C9 đã yêu cầu lấy danh sách động từ `voices.list`; xác nhận khi implement.
3. Hành vi khi vượt hạn mức: chặn cứng, hay tự động rơi về Piper? (Đề xuất: chặn + hỏi Creator, tránh im lặng xuất ra video giọng kém.)

## Phương án local dự phòng (nếu sau này muốn bỏ hẳn cloud)
**VietTTS** (https://github.com/dangvansam/viet-tts) — mã nguồn mở, chuyên tiếng Việt, có voice cloning, chạy qua Docker, chất lượng tốt hơn Piper rõ rệt; cần GPU để không quá chậm. Cắm qua `TTSEnginePort` y hệt Google/Piper nên chi phí chuyển đổi thấp. Thay thế cho các phương án local đã liệt kê ở CR-001 §C2.
