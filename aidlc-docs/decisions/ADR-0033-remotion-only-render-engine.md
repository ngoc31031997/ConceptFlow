# ADR-0033: Remotion là engine dựng video duy nhất

## Status
Accepted

## Date
2026-10-13

## Stage
Application Design (CR-071)

## Context
Hệ thống có hai engine dựng video: Manim (script Python, mặc định ban đầu) và Remotion (script TSX). Short dọc chỉ dựng được bằng Remotion (ADR-0031), và các tính năng mới nhất — câu thoại làm đơn vị thời gian của shot (ADR-0032), hình minh hoạ, kiểm tra bố cục bằng Chromium — chỉ làm cho Remotion. Nhánh Manim vẫn phải được giữ ở mọi tầng: prompt và vai trò riêng (`manim_engineer`, `manim_engineer_ai`, `manim_adjust`), pipeline sinh code riêng ở llm-service, renderer và bộ lint riêng ở rendering, intro/outro dựng bằng Manim (`render_channel_asset` → `register_channel_asset`), bốn luật QC đọc `layout_marks` theo toạ độ khung Manim, và một image rendering mang texlive, cairo/pango dev và gói `manim`.

Khi đếm dữ liệu trước khi đổi, DB orchestrator có 24 project `render_engine = 'manim'`, tất cả đang ở trạng thái `deleting`. Creator chọn không giữ chúng.

## Các phương án
- **A. Bỏ Manim hoàn toàn; xoá project Manim khỏi DB (chọn).** Không còn nhánh nào theo engine trong code; dữ liệu khớp với code.
- **B. Bỏ Manim, giữ project Manim ở chế độ chỉ xem/đăng/xoá.** Cần một giá trị "engine cũ", cổng chặn 409 ở orchestrator và authoring-service, nhãn riêng ở web-gui — code chỉ để phục vụ 24 project đang chờ xoá.
- **C. Giữ hai engine.** Mọi tính năng mới tiếp tục chỉ có ở Remotion, nhánh Manim tiếp tục phải được bảo trì và test.

## Decision
A.
- `render_engine` chỉ nhận `remotion`, ở orchestrator, authoring-service, llm-service (`/v2/code/*`) và rendering. Engine không còn được chọn: bỏ khỏi body request (`POST /v1/projects`, `POST /v1/sagas/render`, `PATCH /v1/projects/{id}/settings`), giữ trong response. Rendering nhận lệnh có `engine` khác `remotion` thì trả sự kiện thất bại với `UnsupportedEngineError`; llm-service trả 400.
- Migration orchestrator xoá các project `render_engine = 'manim'` cùng `saga_steps`, `outbox_events`, `project_events` của chúng; cột `manim_scene_class_name` đổi tên thành `scene_class_name`; cột `layout_marks` bị bỏ. authoring-service xoá các prompt vai Manim.
- Intro/outro của kênh chỉ còn đường tải file lên (`normalize_channel_asset`). Intro/outro đã đăng ký vẫn dùng được.
- QC bỏ bốn luật bố cục theo khung Manim (tràn khung, chữ đè, chữ nhỏ, tương phản); luật blocking duy nhất còn lại là chồng lời thoại. Bố cục của Remotion đã được kiểm tra ở bước Code bằng Chromium.
- Image rendering chỉ còn ffmpeg, font và Node/Chromium cho Remotion.

## Consequences
- Không chuyển được project Manim sang Remotion (script Python không dịch được); 24 project đó bị xoá khỏi DB orchestrator. File của chúng trên `/shared` và dữ liệu authoring-service theo `project_id` của chúng không được dọn tự động.
- Dữ liệu bị bỏ: prompt Manim, cột `layout_marks`, cache `/shared/.manim-media`, biến môi trường `RENDER_CACHE_ROOT`, `RENDER_MEMORY_LIMIT_GB`, `RENDER_QUALITY`, `DRY_RUN_TIMEOUT_SECONDS`, `BRAND_ASSETS_DIR` và volume `docs/brand` của rendering.
- Bước Hình minh hoạ luôn có trong chuỗi soạn; trạng thái bước "Không dùng" không còn.
- Dấu vân tay các đoạn code đã lưu vẫn tính cả `engine`, nên đoạn của project Remotion hiện có không bị coi là cũ.
