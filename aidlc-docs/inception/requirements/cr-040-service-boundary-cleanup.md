# CR-040 — Dọn ranh giới service: bỏ script-processing, tách authoring-service, chủ sở hữu shared_artifacts

## Date
2026-09-26

## Stage
Requirements Analysis (Change Request) — chờ Creator duyệt thiết kế trước khi code

## Intent Analysis
- **Request type**: Tái cấu trúc kiến trúc (không thêm tính năng cho Creator)
- **Scope estimate**: 8 unit — `script-processing` (XOÁ), `rendering`, `orchestrator`, `authoring-service` (MỚI), `video-assembly`, `api-gateway`, `web-gui`, `llm-service`; cộng `tts`/`publisher` ở phần đường dẫn chung
- **Complexity estimate**: Cao. Đổi saga render, thêm một service, đổi luồng xoá project. Nên làm theo từng phần độc lập (thứ tự ở cuối).

## Bối cảnh
Sau CR-018/CR-039, một số ranh giới service không còn hợp lý:
1. `script-processing` chỉ còn tìm tên class Scene / composition id bằng regex (`manim_script_parser.py`, 79 dòng) — một service, một queue, một DB chỉ cho một regex. Trong khi `rendering` vốn cần tên class trước lượt dry.
2. Orchestrator (~21k dòng Go) ôm cả saga, project, thư viện prompt, chuỗi authoring 1a/1b/1c, wizard, gợi ý metadata, usage LLM.
3. `channel_asset_rendered` từ `rendering.events` được bind thẳng vào `video-assembly` (CR-023 correction), song song với việc orchestrator cũng nghe sự kiện này — hai consumer, không ai điều phối.
4. Prompt và validate script bị nhân bản: `web-gui/src/components/scriptPrompts.ts` (536 dòng), `scriptTemplates.ts`, `utils/scriptValidation.ts` (363 dòng) song song với thư viện prompt ở orchestrator và pipeline ở `llm-service`.
5. Volume `shared_artifacts` không có chủ.
6. `llm-service` gọi `rendering /v1/check/*` đồng bộ.

## Yêu cầu chức năng

### FR110 — Gộp script-processing vào bước validate_script của rendering
- **FR110.1** `rendering` tự tìm tên class Scene (Manim) hoặc composition id (Remotion) ở đầu `ValidateScriptUseCase`, dùng lại đúng hai regex hiện có (`SCENE_CLASS_RE`, `REMOTION_COMPOSITION_RE` + `REGISTER_ROOT_RE`) và đúng thông báo lỗi tiếng Việt hiện tại.
- **FR110.2** Saga bỏ bước `parse_script`: `start_render_saga` gửi lệnh `validate_script` thẳng sang `rendering`. Event thành công của `validate_script` trả thêm `scene_class_name` và `engine` để các bước sau dùng như trước.
- **FR110.3** `/v1/check/manim` không bắt buộc `scene_class_name` nữa (rỗng thì tự tìm). `llm-service` vẫn được gửi giá trị từ Merger.
- **FR110.4** Xoá service `script-processing`: thư mục, compose, queue `script_processing.commands(.dlq)`, DB, mục trong `retry_step.go`. Saga đang dừng ở `parse_script` khi deploy: migration đánh dấu bước đó `succeeded` và chuyển sang `validate_script` (hoặc cho retry từ `validate_script`).
- **FR110.5** Test parser chuyển sang `rendering/tests`.
- *Phương án dự phòng (không chọn)*: orchestrator tự tìm bằng regex khi nhận script. Bỏ vì đưa hiểu biết về cú pháp engine vào orchestrator, đúng hướng ngược với FR111.

### FR111 — Tách authoring-service khỏi orchestrator
- **FR111.1** Service mới `authoring-service` (Go, cùng stack/hexagonal như orchestrator để chuyển code gần như nguyên trạng) sở hữu:
  - thư viện prompt (`prompt_templates*`, `prompt_template_seeds*`, `prompt_vars`, `prompts/`, `prompt_overrides`) và các golden test;
  - chuỗi authoring 1a/1b/1c (`authoring_chain`, `generate_authoring*`, `code_pipeline`, `review_outline`), wizard, `suggest_publish_metadata`, `suggest_short_script`;
  - `llm_usage`, client `llm-service`, khoá double-click, tiến độ authoring.
- **FR111.2** Orchestrator chỉ giữ **saga và project** (project, draft, fork, cancel/retry step, channel assets, project_errors).
- **FR111.3** Dữ liệu authoring (các bước 1a/1b/1c, storyboard, usage) chuyển sang DB riêng của `authoring-service` (theo ADR-0013). Khi Creator bấm chạy render, gateway lấy script đã chốt từ `authoring-service` rồi gọi orchestrator tạo saga — orchestrator không gọi ngược sang authoring.
- **FR111.4** Gateway định tuyến `/v1/authoring/*`, `/v1/prompts/*`, `/v1/usage/*` sang `authoring-service`; API công khai không đổi đường dẫn và payload.
- **FR111.5** Viết ADR mới cho ranh giới này.

### FR112 — Orchestrator điều phối channel asset
- **FR112.1** Orchestrator nhận `channel_asset_rendered`, cập nhật `channel_asset_pointers` như hiện nay, rồi phát lệnh `register_channel_asset` sang `video_assembly.commands` (qua outbox, ADR-0019).
- **FR112.2** Xoá binding `rendering.events → video-assembly` cho `channel_asset_rendered`; `ChannelAssetRenderedEventHandler` đổi thành command handler, giữ inbox idempotent.
- **FR112.3** `channel_asset_normalized` vẫn từ video-assembly về orchestrator như cũ.

### FR113 — Một nguồn duy nhất cho prompt và validate
- **FR113.1** Các prompt còn lại của web-gui (`buildGenerationSystemPrompt`, `buildAiPromptTemplate`, `buildRemotionAdjustPromptTemplate`, `buildShortScriptSystemPrompt`, beat sheet, `CHANNEL_IDENTITY`, luật ngôn ngữ narration, subtitle zone, `scriptTemplates`) chuyển thành role trong thư viện prompt (sau FR111 là của `authoring-service`). Web-gui lấy văn bản đã render qua API (nút Copy prompt gọi API thay vì ghép chuỗi ở client).
- **FR113.2** Test golden: văn bản web-gui đang tạo ra cho từng tổ hợp (ngôn ngữ × engine × format) phải trùng byte với bản render từ server trước khi xoá code client.
- **FR113.3** Validate phía client chỉ giữ mức gợi ý nhanh (strip code fence, đếm narration ước lượng, cảnh báo thiếu class/composition). Kết luận hợp lệ hay không là của `rendering` (`validate_script` / `/v1/check/*`). Xoá phần trùng với lint server trong `scriptValidation.ts`.

### FR114 — Chủ sở hữu shared_artifacts
- **FR114.1** Hiện 5 service mount volume: `tts`, `rendering`, `video-assembly`, `api-gateway` ghi được; `publisher` chỉ đọc. Mỗi service tự viết `adapters/storage/artifact_paths.py`. Gom quy ước đường dẫn về **một tài liệu hợp đồng** `docs/contracts/shared-artifacts.md` (cây thư mục, service nào ghi thư mục nào) cộng một **package Python dùng chung** cho các service Python; gateway (Node) theo tài liệu và có test so khớp.
- **FR114.2** Xoá project thành **saga xoá** do orchestrator phát: gửi `purge_project_artifacts` cho từng service ghi (tts, rendering, video-assembly), mỗi service chỉ dọn thư mục của mình; gateway chỉ dọn thư mục upload của chính nó. Bỏ `fs.rm` cả thư mục project trong `api-gateway/src/handlers/deleteProjectHandler.js` và sửa comment sai về quyền mount.
- **FR114.3** Tối thiểu (làm trước nếu saga xoá chưa kịp): **chặn xoá khi project có saga đang chạy** (409 kèm thông báo "đang render, hãy huỷ trước"). Sau khi xoá, service nhận lệnh cho project đã bị xoá thì bỏ qua, không tạo lại thư mục.

### FR115 — Kiểm tra biên dịch không tranh CPU với render (theo dõi, chưa làm)
- **FR115.1** Hiện `llm-service` gọi `/v1/check/*` đồng bộ, timeout `RENDERING_CHECK_TIMEOUT_SECONDS=2100`; check chạy chung container với render nên chung `RENDER_CPU_LIMIT`. Chấp nhận ở CR này.
- **FR115.2** Thêm số đo: thời gian chờ và thời gian chạy của mỗi lượt check, số lượt check bị timeout, số render đang chạy tại thời điểm check.
- **FR115.3** Điều kiện kích hoạt CR tiếp theo: nếu p95 thời gian chờ check vượt ngưỡng (đề xuất 60s) thì tách check sang queue riêng hoặc container `rendering-check` riêng (cùng image, không nhận lệnh render).

### FR116 — Thẻ tiến độ dùng chung cho mọi lượt gọi API chạy lâu (Creator bổ sung 2026-09-26)
- **FR116.1** Hiện chỉ bước 1a/1b/1c có thẻ tiến độ ("1b · Storyboard / AI đang viết · 14,3k ký tự · 2m05s" + thanh chạy), dựng riêng trong `AuthoringModeBar.tsx` với `useAuthoringProgress` và `formatChars`/`formatClock`. Tách phần này thành component dùng chung `OperationProgressCard` (tiêu đề, dòng phụ `pha · số lượng · thời gian`, thanh tiến độ, `role="progressbar"`) và hook `useOperationProgress(operationId)`. Bước 1a/1b/1c chuyển sang dùng component này, hiển thị không đổi.
- **FR116.2** Áp dụng cho các lượt gọi hiện chỉ có nút disabled hoặc chữ "Đang..." như sau:
  - gợi ý metadata (`PublishForm`), gợi ý short script (`ShortScriptAssistant`), gợi ý chủ đề thumbnail (`ThumbnailUpload`): hiện "AI đang suy luận / đang viết · n ký tự · thời gian" giống 1b;
  - kiểm tra biên dịch và các vòng Repair của 1c: "Đang kiểm tra · vòng k/3";
  - saga xoá project (FR114.2): "Đang dọn · k/N service";
  - tải ảnh lên: phần trăm theo byte.
- **FR116.3** Nguồn tiến độ: mỗi lượt gọi lâu trả về `operation_id` ngay, và GUI poll `GET /v1/operations/{id}` mỗi giây, cùng cách với `getAuthoringProgress`. Dữ liệu gồm `{kind, phase, reasoning_chars, content_chars, done, total, elapsed_ms, status, error}`. `authoring-service` dựng số ký tự từ sự kiện `progress` mà `llm-service` stream về (FR100.2). Orchestrator cấp tiến độ cho saga xoá.
- **FR116.4** Thao tác nào không biết tổng (ví dụ lượt suy luận) thì thanh tiến độ chạy ở chế độ không xác định, không được giả phần trăm. Có `total` thì mới hiện phần trăm.
- **FR116.5** Khi lỗi, thẻ giữ lại số ký tự và thời gian đã chạy, đồng thời hiện lỗi đã phân loại (`balance`, `budget`, `server`…) như bước authoring hiện nay. Poll bị lỗi mạng thì bỏ qua (best-effort), không làm hỏng lượt gọi.

## Ngoài phạm vi
- Không đổi nội dung prompt, luật sáng tác hay chất lượng render.
- Không đổi API công khai của gateway (trừ 409 ở FR114.3).
- Không tách check khỏi rendering (FR115 chỉ đo).

## Rủi ro và cách giảm
| Rủi ro | Giảm |
|---|---|
| Saga đang chạy khi bỏ bước `parse_script` | Migration FR110.4; deploy lúc không có saga chạy |
| Tách authoring-service làm đứt các chỗ orchestrator đang gọi trực tiếp use case authoring | Chuyển code nguyên trạng trước, đổi wiring sau; test hợp đồng gateway |
| Prompt web-gui lệch khi chuyển lên server | Golden test FR113.2 |
| File bị ghi lại sau khi xoá project | FR114.3 chặn xoá khi saga chạy + bỏ qua lệnh của project đã xoá |
| Thêm service, thêm điểm hỏng | Healthcheck compose; luồng Copy vẫn hoạt động khi `llm-service` lỗi |

## Kế hoạch triển khai (sau khi duyệt) — mỗi bước deploy được độc lập
1. FR114.3 (chặn xoá khi saga chạy) — nhỏ, giảm rủi ro mất dữ liệu ngay.
2. FR112 (channel asset qua orchestrator).
3. FR110 (bỏ script-processing).
4. FR114.1–114.2 (hợp đồng đường dẫn, saga xoá).
5. FR111 (authoring-service) + ADR.
6. FR113 (prompt web-gui về server) — sau FR111 để đặt vào đúng chỗ.
7. FR115 số đo.
7b. FR116: tách `OperationProgressCard`, rồi áp dụng dần theo FR116.2 (làm cùng FR111 vì cần `/v1/operations`).
8. Rebuild/restart các service bị ảnh hưởng (theo memory re-seed khi prompt seed đổi), E2E một project mỗi engine và một lần xoá project.

## Quyết định (Creator chốt 2026-09-26)

| # | Quyết định | Lựa chọn |
|---|---|---|
| D1 | Stack của `authoring-service` | **Go, chuyển code nguyên trạng từ orchestrator** (FR111.1). Không gộp vào `llm-service`. |
| D2 | Phạm vi xoá project | **Làm saga xoá đầy đủ trong CR này** (FR114.2). FR114.3 (chặn 409 khi saga đang chạy) vẫn làm, là điều kiện vào của saga xoá, không phải phương án thay thế. |

## Trạng thái triển khai (2026-09-26)

Đã làm và kiểm chứng bằng test đơn vị (chưa rebuild/restart, chưa E2E):
- **FR114.3** chặn xoá khi có bước đang chạy (`IsInFlight`, khoá dòng, 409).
- **FR112** orchestrator điều phối `channel_asset_rendered` → `register_channel_asset`; bỏ queue/binding cũ.
- **FR110** bỏ `script-processing`; `validate_script` tự tìm tên class; saga bắt đầu ở `validate_script`; retry `failed_at_parse_script` chạy tiếp từ `validate_script`. Thư mục service cũ được chuyển vào scratchpad backup.
- **FR114.2** saga xoá: `DELETE` trả 202, project `deleting` (ẩn khỏi danh sách), `purge_project_artifacts` tới tts/rendering/video_assembly, xoá dòng khi cả ba xác nhận; gateway chỉ xoá `thumbnail/`, `music/`.
- **FR114.1** tài liệu hợp đồng `docs/contracts/shared-artifacts.md` + test so khớp `tests/contracts/`.
  - **Lệch so với đề xuất:** không làm package Python dùng chung. Mỗi service có build context riêng nên package chung đòi đổi Dockerfile/compose của cả ba; thay bằng tài liệu + contract test.

- **FR111** tách `authoring-service` (Go, code chuyển nguyên trạng, DB riêng), xem ADR-0029. Orchestrator còn saga/project/wizard/fork/list/xoá; gateway định tuyến route authoring sang service mới; `scripts/migrate-authoring-data.sh` chuyển dữ liệu (idempotent). Đã build, chạy compose và kiểm chứng thật qua gateway: tạo draft, lưu story, prompt render, trùng topic, fork, danh sách, xoá đồng thời 3 project (saga xoá + dọn authoring), `validate_script` tự tìm tên class qua rendering thật.
  - Sửa trong lúc chạy thật: draft chưa từng render có `saga_id` rỗng nên nhiều lần xoá cùng lúc đè lên nhau; `BeginDelete` giờ cấp saga id riêng.

Chưa làm: FR113 (prompt còn lại của web-gui về server), FR115 (số đo check), FR116 (thẻ tiến độ dùng chung). Chưa kiểm chứng: lượt `generate` gọi LLM thật qua authoring-service, đường 409 khi xoá lúc đang render (chỉ có test đơn vị), `migrate-authoring-data.sh` với dữ liệu thật (DB hiện chỉ có `llm_usage`).
