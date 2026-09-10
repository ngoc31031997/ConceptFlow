# CR-021 — Low-Level Design: chấm chất lượng video tự động và cổng publish

## Date
2026-09-10

## Stage
Low-Level Design (Functional/NFR/Infrastructure Design: SKIP — không hạ tầng
mới sau quyết định D1, không NFR mới; lý do ghi tại từng mục)

## Phạm vi
Bước cuối của đợt CR-016..024. Phụ thuộc: dữ liệu `layout` cần cơ chế mark của
CR-002/018 (đã có), và bước này đứng sau `assemble_video` của CR-023 (đã có).

---

## Quyết định kiến trúc

### D1 — `qc_video` là saga step riêng, nhưng worker sống trong video-assembly
CR-021 §Quyết định viết "chạy trong `quality-service` (CR-020)". Không dựng
service đó. Lý do là chính lý do CR-020 đã dùng để **gỡ** content-plugin, và nó
áp gần như nguyên văn ở đây: một service nghĩa là một container, một Postgres,
một queue phải vận hành; QC chỉ cần `ffmpeg`/`ffprobe` và chính file video vừa
ghép — cả hai đã nằm sẵn trong `video-assembly`. Dựng image thứ hai chỉ để chạy
ffprobe là cái giá không đáng, y như CR-020 đã kết luận với `validate_script`.

Cái **không** bị gộp: `qc_video` vẫn là một `StepName` riêng, một
`ProjectStatus` riêng, một lệnh riêng trên queue. Một service chạy nhiều lệnh
là khuôn đã có — `rendering` đang chạy ba (`validate_script`, `render_scenes`,
`render_channel_asset`) trên một queue. Ranh giới saga giữ nguyên như CR-021
mô tả; chỉ có địa chỉ của worker là khác.

Khi nào nên tách ra thật: lúc QC mọc thêm chấm frame bằng VLM (Non-goal của CR
này). Lúc đó nó có phụ thuộc nặng riêng — đúng tiêu chí đã tách `rendering` vì
Manim/texlive — và việc tách là một CR riêng, không phải nợ ngầm.

### D2 — Saga sau thay đổi
```
parse_script -> validate_script -> [CHỜ DUYỆT] -> synthesize_speech
             -> render_scenes -> assemble_video -> qc_video -> publish_video
```
`project.go` thêm `StepQCVideo`, `StatusRunningQC`, `StatusFailedQCVideo` vào
đúng ba chỗ đã có khuôn (`StepName` consts, `ProjectStatus` consts,
`FailedStatusForStep`). `StatusReadyToPublish` chuyển sang được đặt bởi
`qc_completed` thay vì `video_assembled` — đây là thay đổi hành vi duy nhất với
saga hiện có, và là chỗ cần test khoá chặt.

FR61.4 (QC hỏng không được thành cổng khoá): `qc_video` **không có nhánh
failed**. Lỗi kỹ thuật (thiếu marks, ffprobe lỗi) vẫn phát `qc_completed` với
`status="not_scored"` kèm lý do, project vẫn `ready_to_publish`.
`StatusFailedQCVideo` tồn tại cho đúng khuôn `FailedStatusForStep` nhưng chỉ
dùng khi chính message hỏng (không parse được envelope) — cùng ngữ nghĩa các
bước khác.

### D3 — Thu dữ liệu bố cục (FR58)
`conceptflow/narration.py::narrate()` ở lượt **render** hiện ghi
`{"kind":"mark"}`. Thêm ngay cạnh đó một bản ghi `{"kind":"layout"}` cho cùng
index: với mỗi mobject trong `scene.mobjects`, ghi bbox (trái/phải/trên/dưới
theo toạ độ Manim), màu chữ (hex), cỡ chữ, và tên class.

- Cùng file, cùng `CF_MARKS_PATH`, chỉ thêm `kind` (FR58.4 — không mở ống mới).
- Bọc trong `try/except` im lặng y như `_describe_stage` đã làm (FR58.2).
- Chỉ chạy ở `MODE_RENDER`, không chạy ở lượt dry: lượt dry là cổng chặn trước
  TTS, thêm việc vào đó là làm chậm đúng chỗ Creator đang chờ.
- Chi phí (FR58.3): một vòng lặp trên `scene.mobjects` mỗi mốc narration, cùng
  bậc với `_describe_stage` đang chạy sẵn ở lượt dry. Nếu đo thấy vượt vài phần
  trăm thời gian render thì giảm tần suất lấy mẫu, không bỏ tính năng.

`manim_renderer.py::_parse_marks` (dòng ~628) thêm nhánh gom `kind="layout"`,
và `ScriptRenderResult` mang danh sách đó ra ngoài để `rendering` đẩy kèm sự
kiện `rendering_completed` → orchestrator lưu → truyền vào lệnh `qc_video`.

### D4 — Luật chấm, tách khỏi I/O
`services/video-assembly/domain/qc_rules.py` — hàm thuần, nhận dữ liệu, trả
danh sách `QCFinding(rule, severity, message, timestamp_seconds)`. Không đọc
file, không gọi ffmpeg. Đây là chỗ toàn bộ unit test của FR59/FR60 trỏ vào.

| Luật | FR | Nguồn dữ liệu |
|---|---|---|
| tràn khung / lấn safe margin | 59.1 | layout bbox + `theme.py` safe margin |
| chồng lấn giữa mobject chữ | 59.2 | layout bbox |
| chữ quá nhỏ | 59.3 | cỡ chữ + `render_quality` |
| tương phản kém | 59.4 | màu chữ + màu nền theme |
| hình chết | 59.5 | khoảng cách giữa hai mốc, không animation |
| lệch LUFS | 60.1 | `ffmpeg -af loudnorm print_format=json` trên file cuối |
| chồng lấn narration | 60.2 | `narration_segments` + thời lượng audio thật |
| clipping | 60.3 | `astats` peak level |
| cue phụ đề chồng nhau | 60.4 | `subtitle_cues` |
| thuộc tính phát hành | 60.5 | `ffprobe` trên file cuối |

Ngưỡng (FR61.5) nằm trong một dataclass `QCThresholds` đọc từ biến môi trường,
không hằng số rải trong luật. Ngưỡng "chữ quá nhỏ" quy chiếu màn hình 5,5 inch
xem toàn màn hình (Quyết định #2 của CR).

### D5 — Chế độ chỉ-báo trước (Quyết định #3 của CR)
`QC_ENFORCE=false` là mặc định: mọi phát hiện đều ghi `severity` thật vào báo
cáo, nhưng không phát hiện nào chặn publish. Bật `QC_ENFORCE=true` sau khi đã
hiệu chỉnh ngưỡng trên video thật thì tràn khung (59.1) và chồng lấn narration
(60.2) thành blocking (Quyết định #1). Cổng chặn sống ở orchestrator lúc khởi
Saga Publish, không ở web-gui — nút bấm không phải nơi giữ luật.

FR61.3 (bỏ qua có ý thức): endpoint publish nhận `acknowledge_qc: true`, và lần
bỏ qua đó ghi vào bảng `qc_reports` (`overridden_at`, `overridden_findings`) —
tái dùng đúng khuôn idempotent của cổng duyệt dàn ý CR-024 đã dựng.

### D6 — Lưu báo cáo (FR61.1)
Bảng `qc_reports` ở orchestrator (nơi web-gui đọc mọi thứ về project):
`project_id`, `status` (`passed`/`has_findings`/`not_scored`), `findings` JSONB,
`created_at`, `overridden_at`. JSONB chứ không phải text: FR61.1 đòi dữ liệu
đọc được bằng máy, và web-gui cần lọc theo severity.

### D7 — web-gui (FR61.2)
`ResultPage` thêm khối báo cáo trước nút publish, nhóm theo severity, mỗi mục
có timestamp bấm được để tua `<video>` tới đúng giây. Không dựng màn mới.

---

## Rủi ro và cách xử lý
- **Báo động giả** (rủi ro làm hỏng cả tính năng): D5 chính là câu trả lời —
  chạy chỉ-báo, đo tỉ lệ trên video đã sản xuất, siết dần bằng biến môi trường.
- **Bbox Manim không phải cái mắt thấy**: luật chồng lấn chỉ áp cho mobject
  dạng chữ (`Text`/`MarkupText`/`Tex`), bỏ qua `VGroup` và mobject opacity 0 —
  ba nguồn hiểu nhầm mà CR đã nêu đích danh.
- **Loudnorm hai lượt**: FR60.1 chỉ *đo và báo* ở CR này. Chuyển sang hai lượt
  là thay đổi assembly có chi phí, chỉ làm khi số đo chứng minh sai lệch đủ lớn
  — đúng như CR viết.

## Không làm (giữ theo CR)
- Không chấm frame bằng VLM.
- Không tự sửa lỗi bố cục.
- Không chấm chất lượng nội dung.
- Không đụng cơ chế đo offset của CR-002 — chỉ thêm bản ghi bên cạnh.

## Kiểm chứng
- Unit test từng luật trong `qc_rules.py`, cả case đạt và không đạt.
- Test: `qc_video` gặp lỗi kỹ thuật → `not_scored`, project vẫn
  `ready_to_publish` (FR61.4).
- Test: `QC_ENFORCE=false` → phát hiện blocking vẫn không chặn publish.
- Test: `ready_to_publish` giờ do `qc_completed` đặt, không phải
  `video_assembled`.
- Thủ công: một script cố tình tràn khung, một script narration chồng lấn.
