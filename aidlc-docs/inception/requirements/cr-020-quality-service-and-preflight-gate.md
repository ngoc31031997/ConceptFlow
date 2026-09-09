# CR-020 — Quality Service và cổng kiểm tra trước khi tốn TTS (P1)

## Date
2026-09-09

## Stage
Requirements Analysis (Change Request)

## Intent Analysis
- **Request type**: Refactoring + New Feature — tái dụng một service đã chết, thêm một cổng kiểm tra vào Saga
- **Scope estimate**: 3 unit — `content-plugin` (đổi vai), `orchestrator` (đổi thứ tự bước), `web-gui` (chế độ preview)
- **Complexity estimate**: Moderate. Không thêm service mới; phần khó là đổi thứ tự bước trong Saga đang chạy

## Bối cảnh — quan sát được
**Bước `classify_scenes` là một bước chết.** Đã xác minh:

- `handle_step_event.go::onScriptParsed` đánh dấu `StepClassifyScenes` là
  `completed` và phát progress ngay, **không dispatch command nào**. Comment
  ngay trên hàm nói rõ lý do: script Manim tự mang animation nên không cần
  template.
- `services/content-plugin/` vẫn tồn tại đầy đủ: ~825 dòng, một container, một
  Postgres riêng (`content-plugin-db`), một queue. Không ai gửi lệnh tới.

**Và pipeline đang tiêu tiền trước khi kiểm tra.** Thứ tự Saga hiện tại:

```
parse_script -> classify_scenes(no-op) -> synthesize_speech -> render_scenes -> assemble_video
```

Lỗi Manim thật chỉ lộ ở `render_scenes`, tức là **sau khi** TTS đã chạy xong và
đã tiêu quota Azure (tier F0: 500.000 ký tự/tháng). `script_lint.py` chạy trước
render nhưng chỉ phủ 5 class trong một blacklist tự nhận là không bao giờ đầy đủ.

Ngoài ra không có đường nào rẻ để xem thử: `RENDER_QUALITY` thấp nhất là
`720p30`, và TTS luôn chạy trừ khi tắt hẳn narration.

## Vấn đề
1. **Một container và một database chạy không việc**, đồng thời đúng chỗ trống
   đó là nơi hợp lý nhất để đặt khâu kiểm tra chất lượng.
2. **Fail muộn và đắt.** Một dấu ngoặc sai làm hỏng cả lượt TTS.
3. **Không có vòng lặp rẻ.** Muốn xem thử bố cục một cảnh, Creator phải trả giá
   một lượt sản xuất đầy đủ.

## Quyết định
Đổi `content-plugin` thành **`quality-service`**, giữ nguyên khung hexagonal,
Inbox/Outbox và database đã có. Bước `classify_scenes` trong Saga đổi vai thành
`validate_script` và **được đặt trước `synthesize_speech`** — đúng vị trí nó
đang đứng, chỉ khác là lần này nó thực sự làm việc.

```
parse_script -> validate_script -> synthesize_speech -> render_scenes -> assemble_video -> qc_video
```

Số bước của Saga không tăng ở CR này (bước 2 đổi vai). `qc_video` là phạm vi
CR-021, ghi ở đây để thấy đích đến.

`validate_script` chạy **lượt dry của CR-018** — cùng một lượt, không thêm chi
phí. Đó là lý do hai CR nên làm chung một đợt.

## Functional Requirements

### FR55 — Quality Service thay chỗ Content Plugin
- **FR55.1**: `content-plugin` PHẢI được đổi vai thành `quality-service`, giữ
  lại hạ tầng đã có (queue, Inbox/Outbox, database, khung hexagonal).
- **FR55.2**: Toàn bộ mã plugin phân loại scene PHẢI được gỡ. Nó chưa từng chạy
  từ khi chuyển sang chuẩn script Manim, và giữ lại là giữ một đường dẫn chết
  mà người đọc code sau này sẽ tưởng là còn dùng.
- **FR55.3**: Endpoint `/v1/plugins` mà API Gateway đang proxy PHẢI được gỡ
  hoặc thay thế; `web-gui` PHẢI bỏ phần chọn plugin tương ứng.
- **FR55.4**: `ADR-0006` (dynamic plugin loading) và `ADR-0012` (content plugin
  integration via orchestrator) PHẢI được đánh dấu superseded, không xoá.

### FR56 — Cổng validate trước TTS
- **FR56.1**: Bước `validate_script` PHẢI chạy **trước** `synthesize_speech` và
  PHẢI chặn Saga khi phát hiện lỗi mức blocking.
- **FR56.2**: Validate PHẢI gồm tối thiểu:
  - **compile thật** — script chạy được đến hết, qua lượt dry của CR-018. Đây là
    thứ thay thế blacklist: nó bắt **mọi** lỗi API, không chỉ 5 class đã biết;
  - **lint whitelist** của CR-017 FR46;
  - **kiểm tra beat** của CR-019 FR52.3;
  - **lint narration**: câu quá dài (ước lượng vượt ngưỡng), và ký hiệu mà TTS
    sẽ đọc sai (`O(n²)`, `arr[i]`, `!=`) — loại lỗi chỉ nghe ra sau khi đã render.
- **FR56.3**: Kết quả validate PHẢI phân biệt rõ **blocking** và **cảnh báo**.
  Cảnh báo không được chặn Saga.
- **FR56.4**: Thông báo lỗi PHẢI nêu số dòng và cách sửa cụ thể, theo đúng chất
  lượng thông báo mà `script_lint.py` đang đạt được.
- **FR56.5**: Kết quả validate PHẢI hiện ở GUI **ngay khi Creator bấm render**,
  không phải sau vài phút chờ.
- **FR56.6**: `script_lint.py` PHẢI chuyển sang `quality-service`. Giữ hai nơi
  cùng lint là giữ hai nguồn sự thật.

### FR57 — Chế độ preview
- **FR57.1**: PHẢI có chế độ preview: độ phân giải và framerate thấp hơn mọi
  giá trị `RENDER_QUALITY` hiện có, **không chạy TTS** (dùng ước lượng thời
  lượng theo nhánh đã có sẵn của CR-001), không ghép nhạc nền, không phụ đề.
- **FR57.2**: Preview PHẢI đi qua đúng đường code của render thật để những gì
  nhìn thấy là thật, chỉ khác tham số. Một đường riêng sẽ trôi khỏi đường chính.
- **FR57.3**: Kết quả preview PHẢI được đánh dấu rõ ràng là preview và KHÔNG
  được publish lên YouTube.
- **FR57.4**: Preview KHÔNG được ghi đè video thật của project.
- **FR57.5**: Preview PHẢI dùng chung `media_dir` cache với render thật, để lượt
  render thật sau đó tái dùng được segment đã dựng.

## Non-goals
- Không thêm service mới. CR này giảm số thành phần, không tăng.
- Không kiểm tra nội dung hình ảnh — đó là CR-021.
- Không giữ lại khả năng phân loại scene bằng plugin. Nếu sau này cần, đó là một
  CR mới với bối cảnh mới.

## Rủi ro
- **Đổi thứ tự bước trong Saga đang chạy.** `SagaStep` được lưu theo `step_name`;
  project dở dang lúc triển khai sẽ mang tên bước cũ. Cần chiến lược migrate rõ
  ràng, hoặc chỉ triển khai khi không còn Saga đang chạy.
- **FR57.5 mâu thuẫn tiềm tàng với FR57.1**: preview ở độ phân giải khác sẽ sinh
  segment cache khác, nên lợi ích tái dùng cache có thể bằng không. Cần đo trước
  khi cam kết; nếu không tái dùng được thì bỏ FR57.5 chứ không hạ chất lượng
  preview.
- **Gỡ `/v1/plugins` là thay đổi phá vỡ** với bất kỳ thứ gì đang gọi nó. Trong
  phạm vi hệ thống này chỉ có `web-gui`, cần xác nhận lại.

## Quyết định đã chốt (2026-09-10)
1. **Gỡ plugin phân loại là an toàn** — đã xác nhận không có project nào dùng.
   Toàn bộ phần bị gỡ là 39 dòng map hai chuỗi (`algorithm` →
   `algorithm_visualization`, `concept` → `concept_illustration`) sang một hệ
   template không còn tồn tại từ khi chuyển sang chuẩn script Manim. Hạ tầng
   (queue, database, Inbox/Outbox, khung hexagonal) **được giữ lại** và tái dụng
   cho `quality-service`, đúng như FR55.1.
2. **Preview là một lựa chọn trong `RenderQualityPicker`**, không phải nút riêng
   — nó nằm trên cùng một trục quyết định với 720p30/1080p60/4k60.
3. **Ngưỡng "câu narration quá dài": 20 giây** ước lượng.

## Câu hỏi cần Creator chốt
1. Có project nào đang dùng plugin phân loại scene mà tôi chưa thấy không, hay
   gỡ thẳng là an toàn?
2. Preview nên là một nút riêng, hay một lựa chọn trong bộ chọn chất lượng
   hiện có (`RenderQualityPicker`)?
3. Ngưỡng "câu narration quá dài" ở FR56.2 nên đặt bao nhiêu giây?

## Kiểm chứng
- Unit test: script lỗi API bị chặn ở `validate_script`, và `tts.commands`
  KHÔNG nhận được lệnh nào.
- Unit test: cảnh báo không chặn Saga; blocking thì chặn.
- E2E: preview cho ra video xem được, không publish được, không đè video thật.
- Đo: xác nhận quota TTS không bị tiêu khi script sai.

## Liên quan
- ADR-0006, ADR-0012 (bị superseded bởi FR55.4)
- CR-001 (nhánh tắt TTS — cơ sở cho preview FR57.1)
- CR-017 (lint whitelist)
- CR-018 (lượt dry dùng chung — nên làm cùng đợt)
- CR-019 (kiểm tra beat)
- CR-021 (bước `qc_video`, sống trong cùng service này)
- ADR sẽ cần: "Quality Service thay Content Plugin; cổng validate đặt trước TTS"
