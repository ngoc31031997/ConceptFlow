# CR-015 — Kế hoạch thực hiện: caption track YouTube

## Date
2026-09-09

## Stage
Low-Level Design → Code Generation

## Phạm vi
4 unit. Thứ tự dưới đây là thứ tự phụ thuộc dữ liệu: `.srt` phải tồn tại trước
khi có gì đó để chuyển đi, và phải chuyển tới nơi trước khi upload được.

---

## Bước 1 — `video-assembly`: sinh `.srt` (FR38)

**Sửa**
- `domain/models.py` — thêm `subtitle_mode: str = "burn_in"` vào
  `VideoAssemblyRequest` (`off | track | burn_in | both`). Mặc định `burn_in`
  giữ tương thích ngược cho message đang bay trong queue lúc deploy.
- `adapters/assembly/srt_file.py` (**mới**) — `write_srt_file(cues, lead_in, path)`.
  Tách file riêng khỏi `subtitle_file.py`: hai format không chia sẻ gì ngoài
  danh sách cues, và `subtitle_file.py` đang mang toàn bộ logic PlayRes/scale
  vốn vô nghĩa với SRT.
- `adapters/assembly/ffmpeg_assembler.py` — `_write_subtitles` chỉ chạy khi mode
  ∈ {`burn_in`, `both`}; thêm nhánh ghi `.srt` khi mode ∈ {`track`, `both`}.
- `adapters/messaging/consumer.py` — đọc `subtitle_mode` từ payload; phát
  `caption_path` trong event hoàn tất.

**`lead_in` — khử bằng thiết kế, không bằng test (FR38.2)**
Cách hiển nhiên là để mỗi serializer tự cộng `lead_in` như `_write_subtitles`
đang làm (dòng 145). KHÔNG làm vậy. Hai nơi cùng tính một phép dịch là hai nơi
có thể lệch nhau, và khi lệch thì burn-in vẫn đúng còn caption sai đều toàn
video — không test nào hiện có bắt được.

Thay vào đó, dịch **một lần** ở tầng trên rồi đưa cues đã dịch cho cả hai:

```
shifted = [c.shifted_by(lead_in) for c in cues]   # đúng một nơi tính
write_subtitle_file(shifted, style, ass_path)     # không biết lead_in là gì
write_srt_file(shifted, srt_path)                 # cũng vậy
```

- `SubtitleCue.shifted_by(seconds)` là method mới trên dataclass (đang frozen —
  trả về instance mới).
- `write_subtitle_file` **bỏ** tham số `lead_in`; chữ ký không còn nhận nó nữa
  thì không ai cộng nhầm lần hai được.
- Hai serializer không thể lệch nhau vì không cái nào tự tính.

Test còn lại chỉ cần một cái ở tầng `shifted_by`, thay vì hai cái song song
phải tự trông nhau.

**Lưu ý**: khi mode là `track`, video không còn phải re-encode vì phụ đề. Comment
ở dòng 228–235 nói rõ burn-in là lý do duy nhất phải re-encode trong nhiều
trường hợp — bỏ nhánh đó đi thì render nhanh hơn đáng kể. Kiểm tra lại điều kiện
`-c:v copy` để thực sự hưởng lợi, đừng để nó vẫn re-encode vô ích.

---

## Bước 2 — `orchestrator`: chuyển `caption_path` đi (FR38.4)

Đi đúng đường `thumbnail_path` đang đi, không phát minh đường mới:
- `internal/adapters/postgres/db.go` — `ALTER TABLE projects ADD COLUMN IF NOT
  EXISTS caption_path TEXT;` (theo đúng khuôn các cột đã thêm ở dòng 51–53).
- `project_repository.go` — thêm cột vào SELECT/UPSERT.
- `handle_step_event.go::onVideoAssembled` — đọc `caption_path` từ payload, lưu.
- `start_publish_saga.go` — nhét vào payload publish, cạnh `thumbnail_path`
  (dòng 118), và đính kèm mã ngôn ngữ BCP-47 của project cho FR39.3.

---

## Bước 3 — `publisher`: upload track + scope (FR39, FR40)

**Sửa**
- `domain/models.py` — `PublishRequest`: thêm `caption_path: str | None`,
  `caption_language: str | None`. `OAuthCredential`: thêm `scopes: tuple[str, ...] = ()`
  (rỗng ⇒ credential cũ ⇒ coi như chỉ có `youtube.upload`, ADR-0028).
- `adapters/youtube/oauth_flow.py` — thêm `YOUTUBE_FORCE_SSL_SCOPE` vào danh
  sách xin consent; lưu scope Google **thực sự cấp** (đọc từ response), không
  lưu scope ta đã xin — người dùng có thể bỏ tick.
- `adapters/youtube/youtube_publisher.py` — sau `videos().insert` (dòng 124),
  thêm `_insert_caption`, bọc try/except như `thumbnails().set` (dòng 126–134).
- Tầng lưu credential — persist `scopes`.

**Kiểm tra scope trước, không phải sau (FR40.2)**
Đối chiếu `credential.scopes` **trước khi** bắt đầu upload. Nếu thiếu: bỏ qua
caption, ghi log, và trả cờ trong `PublishResult` để GUI hiển thị. Không raise.

**Khác thumbnail ở chỗ báo lỗi (FR39.4)**
Thumbnail hỏng thì Creator mở YouTube là thấy. Caption hỏng thì im lặng tuyệt
đối — nên `PublishResult` phải mang được trạng thái caption
(`uploaded | skipped_no_scope | failed`), không chỉ log.

---

## Bước 4 — `web-gui`: bốn lựa chọn + cảnh báo (FR41)

- Toggle phụ đề → 4 lựa chọn, mặc định `track`.
- Nhóm `SubtitleStyle` (cỡ chữ, màu, hộp nền, vị trí) **ẩn/mờ khi mode = `track`**
  — SRT không mang định dạng, để nguyên là mời Creator chỉnh một thứ vô tác dụng
  (ADR-0027, mục Consequences).
- Chọn `both` ⇒ cảnh báo tại chỗ về chữ chồng hai lớp. Cảnh báo, không cấm.
- Màn hình chọn kênh: kênh thiếu scope hiện "chưa nối lại — video sẽ không có
  phụ đề" **trước** khi publish.
- `ResultPage`: hiện trạng thái caption trả về từ bước 3.

---

## Test

| Bước | Test bắt buộc |
|---|---|
| 1 | SRT đúng format; `shifted_by` đúng (một test, không phải hai); mode `off`/`track` không sinh `.ass`; mode `track` không re-encode |
| 2 | `caption_path` sống sót qua vòng lưu → đọc → payload publish |
| 3 | `captions().insert` đúng language code; caption hỏng không làm hỏng publish; credential thiếu scope bị chặn trước upload; scope lưu là scope Google cấp |
| 4 | mặc định `track`; `SubtitleStyle` ẩn khi `track`; cảnh báo hiện khi `both` |

**E2E**: publish private một video có caption → mở YouTube → nút CC bật được,
nội dung khớp narration, không lệch thời gian.

---

## Thứ tự triển khai
Bước 1–2 độc lập với OAuth, làm được ngay. Bước 3 cần Creator nối lại ít nhất
một kênh với scope mới để verify. Bước 4 phụ thuộc kiểu dữ liệu của bước 3.

## Liên quan
- CR-015 (yêu cầu)
- ADR-0027 (mode theo bề mặt phát hành), ADR-0028 (scope + suy giảm êm)
