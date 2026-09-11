# CR-007 — Low-Level Design: clip dọc 9:16 cho Shorts/TikTok

## Date
2026-09-11

## Stage
Low-Level Design (Functional/NFR/Infrastructure Design: SKIP — không hạ tầng
mới, không NFR mới ngoài timeout encode đã có)

## Phạm vi
FR19 (clip dọc phái sinh) + FR20.1 (tải về từ GUI). **FR20.2 (adapter TikTok
API) không làm** — §C1 của CR đã nêu: app TikTok phải được duyệt, và với kênh cá
nhân quy trình duyệt là rào cản thật. Đăng tay là đủ cho MVP.

---

## Correction: FR19.2 đã lỗi thời
CR-007 viết (2026-09-07) rằng Creator chọn đoạn bằng marker `# CLIP: "tên"` …
`# ENDCLIP`. **Cách đó không còn tồn tại**: CR-018 đã gỡ hẳn mọi marker comment,
vì chúng phải khớp theo số lượng và thứ tự dòng trong file — đúng lớp lỗi CR-018
sinh ra để diệt. `# NARRATION:` và `# CHAPTER:` đều đã thành method.

Nên clip cũng phải là method, cùng một cơ chế:

```python
with self.clip("ví dụ chạy thật"):
    ...            # các animation thuộc clip
```

hoặc, nếu không cần gói block, phẳng như `beat()`:

```python
self.clip_start("ví dụ chạy thật")
...
self.clip_end()
```

**Chốt: dùng context manager `with self.clip(...)`.** Lý do: một clip có điểm
đầu và điểm cuối, và cặp `start`/`end` phẳng thì quên `end` là chuyện xảy ra —
còn `with` thì Python tự đóng. Đây cũng là lần đầu conceptflow cần một API có
phạm vi, nên chọn đúng hình dạng ngay quan trọng hơn tiết kiệm một dòng.

Bản ghi ra JSONL (cùng ống `CF_MARKS_PATH`, thêm `kind` — đúng khuôn H2 của
`cr-016-024-execution-plan.md`):

```json
{"kind":"clip","name":"ví dụ chạy thật","t_start":42.5,"t_end":96.0}
```

Ghi ở **lượt render** (cần `renderer.time` thật), và cũng ghi ở **lượt dry** với
`t_start`/`t_end` bằng null để cổng duyệt dàn ý của CR-024 thấy được Creator đã
định cắt những đoạn nào trước khi tốn TTS.

FR19.2 vẫn giữ đường thứ hai: Creator nhập khoảng thời gian ở GUI. Hai đường,
một đích — xem D3.

---

## Quyết định kiến trúc

### D1 — `generate_clips` là saga step mới, worker trong video-assembly
Saga sau thay đổi:
```
... -> assemble_video -> qc_video -> generate_clips -> publish_video
```
Đặt **sau** `qc_video` chứ không song song: cắt clip từ một video vừa bị QC phát
hiện tràn khung là cắt ra hai ba clip cùng lỗi. Đặt **trước** `publish_video` để
clip có mặt lúc Creator vào màn kết quả.

Worker sống trong `video-assembly`, cùng lý do D1 của CR-021: nó chỉ cần ffmpeg
và chính file video vừa ghép. Lệnh thứ tư trên cùng queue, cùng
`VideoAssemblyCommandDispatcher` đã có.

**Không chặn publish khi cắt clip lỗi** (cùng tinh thần FR61.4 của CR-021): clip
dọc là sản phẩm phái sinh, không phải video chính. Lỗi sinh clip phát
`clips_generated` với `status="failed"` kèm lý do, project vẫn về
`ready_to_publish`.

### D2 — Preset là config, không hardcode (§C2b)
Ngưỡng độ dài do YouTube/TikTok đặt và **đã đổi nhiều lần** (Shorts từng 15s rồi
60s). Nên:

```
CLIP_PRESET_SHORT_MAX_SECONDS  = 60    # đủ điều kiện YouTube Shorts
CLIP_PRESET_LONG_MIN_SECONDS   = 60    # TikTok Creator Rewards đòi > 1 phút
CLIP_PRESET_LONG_MAX_SECONDS   = 180
```

Chốt câu hỏi mở #2 của CR: preset `long` là **60–180s**, không hẹp hơn. Hẹp hơn
thì Creator phải chọn lại đoạn thường xuyên, mà không đổi lấy được gì.

### D3 — Hai đường chọn đoạn, một đích
Cả `with self.clip(...)` lẫn khoảng thời gian nhập ở GUI đều quy về cùng một cấu
trúc trước khi tới video-assembly:

```
ClipRequest{ name, start_seconds, end_seconds, presets: ["short"|"long"] }
```

Orchestrator là nơi hợp nhất: clip từ script đọc từ bản ghi `kind="clip"` (đã
lưu cùng `layout_marks` của CR-021), clip từ GUI đọc từ request. Trùng tên thì
GUI thắng — Creator vừa nhập tay là ý định mới hơn.

### D4 — Validate độ dài TRƯỚC khi encode (FR19.6)
Kiểm ở **domain, hàm thuần**, `domain/clip_rules.py`: nhận độ dài đoạn và
preset, trả lỗi rõ ràng hoặc OK. Không im lặng cắt cụt.

Vị trí kiểm: cả hai đầu. Orchestrator kiểm lúc nhận request (để GUI báo lỗi ngay
cho Creator), video-assembly kiểm lại trước khi gọi ffmpeg (vì nó là nơi biết độ
dài thật sau khi ghép intro — CR-023 đã dịch mọi mốc đi).

FR19.7 (một đoạn xuất cả hai preset): `presets` là danh sách, không phải một giá
trị. Một đoạn 75s hợp `long` nhưng không hợp `short` ⇒ báo lỗi **cho riêng
preset đó** và vẫn xuất preset kia — không đánh đổ cả lần chạy.

### D5 — Chuyển 16:9 → 9:16 (FR19.3)
Một filtergraph, không crop:
```
[0:v]split=2[bg][fg];
[bg]scale=1080:1920:force_original_aspect_ratio=increase,crop=1080:1920,boxblur=20:2[blurred];
[fg]scale=1080:-2[scaled];
[blurred][scaled]overlay=(W-w)/2:(H-h)/2
```
Nền là chính khung gốc phóng to rồi làm mờ, video gốc căn giữa. Manim hay đặt
công thức ở rìa khung nên crop là mất nội dung — CR đã nêu đích danh.

Tái dùng `VIDEO_ENCODE_ARGS`/`AUDIO_ENCODE_ARGS`/`CONTAINER_ARGS` của
`ffmpeg_assembler.py`: clip cũng là file đem đăng, không có lý do để nó kém hơn.

### D6 — Phụ đề trên clip dọc (FR19.4)
`subtitle_file.py` hiện nhận `play_res` từ tham số (`_probe_resolution` đã có từ
CR-015) nên khung dọc không cần mở đường mới — truyền `(1080, 1920)`.

Khác biệt thật là **style**: Shorts/TikTok xem trên điện thoại, phụ đề phải to
hơn hẳn và đặt giữa khung. Nên một `SubtitleStyle` riêng cho clip dọc, không
dùng lại style của video ngang:
```
font_size: "large", position: "center"
```
Cue lấy từ `subtitle_cues` của project, **dịch về mốc 0 của clip**: cue nào nằm
ngoài `[start, end)` thì bỏ, cue trong khoảng thì trừ đi `start`. Đây là chỗ dễ
sai nhất của CR này — cùng lớp lỗi CR-002 và CR-023 FR66.2, nên test khoá chặt.

### D7 — Tải về (FR20.1)
`GET /v1/projects/{id}/clips` liệt kê clip đã sinh, `GET
/v1/projects/{id}/clips/{name}` stream file — đúng khuôn
`videoHandler`/`thumbnailServeHandler` đã có trong api-gateway, không phát minh
đường mới. Web-gui thêm danh sách clip ở `ResultPage` với nút tải.

---

## Rủi ro
- **Dịch cue về mốc 0 của clip**: lớp lỗi đã tốn CR-002 một CR và CR-023 một
  mục rủi ro riêng. Test phải assert bằng số, cho cả trường hợp cue nằm vắt qua
  biên clip.
- **Intro dịch mốc** (CR-023): `t_start`/`t_end` từ bản ghi `clip` là thời gian
  trong **video Manim**, chưa cộng intro. video-assembly phải cộng
  `intro_duration` y như `effective_lead_in` đã làm, nếu không clip cắt lệch
  đúng bằng độ dài intro.
- **Thời gian encode**: ba clip × hai preset = sáu lần encode. Dùng lại timeout
  của assembly (900s) và đo thực tế; nếu chạm trần thì tách timeout riêng chứ
  không nới timeout chung.

## Không làm
- Adapter TikTok API (FR20.2) — để pha sau.
- Không tự chọn đoạn hay hay thay Creator (không có luật nào đáng tin cho việc
  đó, và chọn sai thì Creator mất công xem lại cả ba clip).
- Không sinh clip cho video đã publish trước CR này.

## Kiểm chứng (khớp Tiêu chí nghiệm thu của CR-007)
- Unit test: `clip_rules` — đoạn 75s hợp `long`, không hợp `short`; báo lỗi rõ
  cho preset không hợp mà vẫn xuất preset hợp (FR19.6/19.7).
- Unit test: cue dịch đúng về mốc 0 của clip, cue ngoài khoảng bị bỏ, cue vắt
  qua biên xử lý đúng.
- Unit test: `t_start` có cộng `intro_duration` khi project bật intro.
- Unit test: lỗi sinh clip không chặn publish.
- Thủ công: từ video 8 phút sinh clip 1080×1920 ở cả hai preset; upload private
  clip `short` xác nhận YouTube nhận diện là Shorts; xác nhận nội dung ở rìa
  khung Manim không bị cắt.
