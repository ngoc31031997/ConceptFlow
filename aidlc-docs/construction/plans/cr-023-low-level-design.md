# CR-023 — Low-Level Design: intro/outro cố định làm bản sắc kênh

## Date
2026-09-10

## Stage
Low-Level Design (Functional/NFR/Infrastructure Design: SKIP — không thêm hạ
tầng mới, không NFR mới ngoài cache; lý do skip ghi ở từng mục bên dưới)

## Phạm vi
CR-023 độc lập với CR-021 (bước 6 và 7 trong `cr-016-024-execution-plan.md`).
Không đụng `conceptflow` API công khai đã khoá ở CR-017/018, chỉ thêm scene mới
dùng component đã có (`title_card.py`, `recap.py`).

---

## Quyết định kiến trúc

### D1 — "Channel asset" là khái niệm mới, nằm ngoài Project
Intro/outro không thuộc về một project — chúng thuộc về kênh. Tạo bảng mới
`channel_assets` ở **video-assembly** (service sở hữu `lead_in`/ghép hiện nay),
không phải một field trên `projects`.

```
channel_assets
  id            uuid PK
  kind          enum('intro','outro')
  render_quality varchar   -- '720p30' | '1080p60' | '4k60'
  source_hash   varchar    -- sha256(file upload) hoặc sha256(scene_def + theme_version)
  video_path    varchar
  music_path    varchar null
  version       int        -- FR65.9, tăng dần theo kind
  created_at    timestamptz
  superseded_at timestamptz null
```
Lý do đặt ở video-assembly chứ không phải rendering: file Creator upload
(FR65.1/65.4) được chuẩn hoá bằng ffmpeg (transcode theo `RENDER_QUALITY`), đây
vốn là việc của video-assembly; outro do Manim dựng thì rendering chỉ **sinh
ra file**, còn video-assembly là nơi quyết định "asset nào đang active" và ghép.

### D2 — Không phải saga step mới; là bước con trong `assemble_video`
Intro/outro không cần TTS/render riêng theo từng project → không cần thêm
`SagaStep`. `assemble_video` (đã có) nhận thêm 2 field optional trên lệnh:
`intro_asset_id`, `outro_asset_id` (null nếu Creator tắt — FR67.1/67.2 mặc định
bật cho long-form). Orchestrator tra `channel_assets` bằng
`(kind, render_quality)` mới nhất chưa `superseded_at`, gắn vào envelope trước
khi publish `assemble_video`.

### D3 — Dựng/refresh asset là flow admin riêng, không phải per-project
Không phải saga. Một use case đơn giản, đồng bộ, gọi trực tiếp:
- Outro: gọi `POST /internal/render/channel-asset` tới rendering (tương tự lệnh
  `validate_script` đã có ở CR-020: request/response đồng bộ, không qua queue,
  vì đây không phải việc chạy trong project pipeline).
- Intro mặc định: dựng bằng Manim cùng đường trên.
- Intro từ Creator upload: video-assembly chuẩn hoá bằng ffmpeg, không gọi
  rendering.
Trigger: khi file mới upload, hoặc khi `theme.py`/scene def đổi version (CI
kiểm tra hash lúc build, giống cách `manim_renderer.py` đã cache theo hash).
FR65.6 (chỉ dựng lại khi nguồn/scene/theme đổi) thực hiện bằng so khớp
`source_hash` trước khi build.

### D4 — `conceptflow`: hai scene mới, không đổi API công khai
`conceptflow/channel_idents.py`:
- `DefaultIntroSting(ConceptFlowScene)` — 3s, logo `docs/brand/conceptflow-mark-1024.png`
  fade+scale, dùng `theme.py` nền `#080E1C`.
- `ChannelOutro(ConceptFlowScene)` — 15–20s, ghép `TitleCard` (logo + wordmark),
  `Recap`-style layout cho vùng "video đề xuất" (chỉ khung, YouTube tự chèn end
  screen thật), text lời mời đăng ký. Bố cục chừa 3 vùng an toàn (FR65.8) —
  test bằng bbox check tái dùng cơ chế QC layout đã phác ở CR-021 H2, ở đây làm
  thủ công bằng assertion toạ độ vì QC service chưa tồn tại.
Cả hai scene **không** gọi `narrate()`/`beat()` — không có lời thoại, không đi
qua lượt dry của CR-018 (đúng như CR nêu: không phụ thuộc CR-018).

### D5 — Dịch timeline: mở rộng `lead_in`, không thêm biến thứ hai
`ffmpeg_assembler.py` hiện nhận `lead_in_seconds` (dùng cho pad tổng thể).
Sửa constructor để `effective_lead_in = lead_in_seconds + intro_duration`
(intro_duration đọc từ `channel_assets`/probe ffprobe). Mọi chỗ đang dùng
`lead_in` (adelay narration, `cue.shifted_by`, `tpad`) tự động đúng — không sửa
logic dịch, chỉ sửa nguồn giá trị đầu vào (đúng tinh thần "không mở đường dịch
thứ hai" của CR).
Ghép vật lý: `concat demuxer` 3 đoạn (intro, `tpad`-ed main, outro) cùng
codec/profile — điều kiện để giữ được từ stream-copy nếu có thể; nếu không,
chấp nhận re-encode toàn bộ (đã re-encode sẵn khi có `tpad`, xem rủi ro CR).
Nhạc nền outro/intro: mix riêng, chuẩn hoá loudnorm -14 LUFS lúc build asset
(D3), không lúc assembly mỗi project (FR66.5, tốn ít lần hơn).

### D6 — `BuildChapterTimestamps` (`chapters.go:36`)
Bỏ `times[0] = 0` cứng. Thêm tham số `introDuration float64`:
- introDuration == 0: giữ nguyên hành vi cũ (không có intro → chapter[0] vẫn
  phải là 0 vì nó thật sự là đầu video).
- introDuration > 0: chèn chapter tổng hợp `{Title: "Intro", Start: 0}`, các
  chapter còn lại dùng `Start = original + introDuration` (không ép về 0).
`MinChapterSeconds` so với `totalDuration` truyền vào từ caller
(`suggest_publish_metadata.go:61`) — caller cộng thêm intro+outro duration lấy
từ `channel_assets` trước khi gọi.

### D7 — Toggle & preview
`projects` thêm 2 cột `intro_enabled bool default true`,
`outro_enabled bool default true` (FR67.1/67.2). Preview (FR67.4):
`GET /v1/channel-assets/preview?quality=1080p60` trả URL asset hiện hành —
đọc thẳng `channel_assets`, không render gì (FR67.3 áp dụng tự nhiên vì preview
project ở CR-020 không đụng bảng này).

### D8 — Upload
Theo pattern `thumbnailUploadHandler.js`/`musicUploadHandler.js`:
`POST /v1/channel-assets/intro` (video, multipart) và
`POST /v1/channel-assets/intro-music`, `/outro-music`. Validate lúc upload
(FR66.7): tỉ lệ khung 16:9, thời lượng ≤ 5s (biên rộng hơn 3s một chút để chừa
margin edit), decode được bằng ffprobe — reject kèm lý do cụ thể trong response
body, không lưu file nếu fail.

---

## Rủi ro mang từ CR sang, cách xử lý cụ thể
- **Mất stream-copy**: đo lại sau khi implement, ghi số liệu vào CR gốc, không
  chặn merge nếu mất (đã là chi phí chấp nhận được theo rủi ro đã nêu).
- **Lệch timeline**: unit test khoá cả 4 loại mốc (narration offset, `.ass`
  cue, `.srt` cue, chapter timestamp) với `introDuration` cố định giả lập,
  assert bằng snapshot số, không chỉ "không panic".

## Không làm (giữ theo CR)
- Không có UI dựng sting trong hệ thống.
- Không intro riêng theo từng project.
- Không hồi tố video đã publish.

## Kiểm chứng (khớp mục "Kiểm chứng" của CR-023)
- `chapters_test.go`: case introDuration=0 (hành vi cũ giữ nguyên) và >0.
- `ffmpeg_assembler` test: 4 loại mốc dịch đúng khi có intro.
- `channel_assets` cache test: gọi build 2 lần với cùng hash → không dựng lại.
- Thủ công: 2 video khác chủ đề, diff byte đoạn intro/outro trùng khớp; nghe
  mức âm lượng ở điểm nối.
