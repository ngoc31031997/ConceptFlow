# CR-071 — Bỏ engine Manim, chỉ còn Remotion

Trạng thái: **thiết kế xong, chờ Creator duyệt**.

## Yêu cầu gốc (nguyên văn)

CR này mở ra từ việc đối chiếu pipeline với quy trình làm phim hoạt hình 2D (nhánh lúc đầu tên `feature/cr-071-2d-film-pipeline`; lịch sử ở `aidlc-docs/audit.md`, các mục CR-071).

> "bỏ luôn manim. chỉ còn remotion"

> "bỏ hết chỉ giữ lại cr bỏ manim, các CR còn lại cần đi qua AI-DLC rule từ lên ý tưởng cho nên implement"

Lộ trình 2D (các CR con về giọng nháp, xem trước, nhân vật, màu, SFX…) bị bỏ. Mỗi ý tưởng đó, nếu làm, sẽ đi trọn AI-DLC từ khâu ý tưởng như một yêu cầu mới. CR-071 chỉ còn việc bỏ Manim.

## Hiện trạng

Main `7df416a`. Hệ thống có hai engine dựng hình: Manim (Python, kiểu 3Blue1Brown, engine gốc theo `README.md`) và Remotion (React/TS). Short dọc chỉ dựng được bằng Remotion (`services/orchestrator/internal/domain/wizard.go:100`).

### Manim nằm ở đâu

Khoảng 110 file nguồn (không kể test) nhắc tới Manim, trong 8 service:

| Service | Số file | Chỗ chính |
|---|---|---|
| rendering | 65 | `adapters/rendering/manim_renderer.py` (`ManimScriptRenderer`, gồm `render_channel_asset` dòng 334); `adapters/rendering/engine_router.py` (chọn renderer theo engine, dòng 26-53); gói `conceptflow/` (thư viện scene Manim: `scene.py`, `narration.py`, `theme.py`, `components/*`, `channel_idents.py`); `domain/script_lint.py` (lint script Manim, import `conceptflow`); `application/render_script.py:82`, `validate_script.py:89`, `check_script.py:86` (nhánh `engine == "manim"`); `application/render_channel_asset.py`; `tools/gen_theme_reference.py`; `Dockerfile` (texlive, cairo, pango cho Manim); `requirements.txt:3` (`manim==0.18.*`) |
| authoring-service | 34 | Vai prompt `manim_engineer`, `manim_engineer_ai`, `manim_adjust` (`domain/prompt_template.go:19,28,33`); seed `manimIntroVI` → `manimOutputVI` (`domain/prompt_template_seeds.go:448-611`) và bản AI; `prompts/manim_adjust.txt`, `prompts/theme_reference_{vi,en}.txt` (chỉ prompt Manim dùng, sinh từ `conceptflow/theme.py`); `RenderEngineManim` (`domain/project.go:211`) |
| web-gui | 32 | `RenderEnginePicker.tsx`, `RenderEngineBadge.tsx`; `ManimEngineerStepPage.tsx` (màn bước Code **cho cả hai engine**, route `/create/script/code` ở `App.tsx:67`); nhánh engine ở `flow.ts`, `scriptValidation.ts`, `IllustrationsStepPage.tsx` (bước 5 "Không dùng" khi không phải Remotion)… |
| orchestrator | 21 | `RenderEngineManim`, `DefaultRenderEngine = RenderEngineManim` (`domain/project.go:199-206`); cột `render_engine TEXT NOT NULL DEFAULT 'manim'` (`adapters/postgres/db.go:189`); cột `manim_scene_class_name` (`db.go:23,53`) — Remotion cũng dùng giá trị này làm id composition (`rendering/adapters/rendering/remotion_renderer.py:241`) |
| video-assembly | 9 | `domain/qc_rules.py` (luật bố cục theo toạ độ khung Manim; bước `qc_video` đã tắt khỏi luồng chính từ CR-029) |
| llm-service | 9 | `app/pipeline/prompts.py` (`manim_chunk`, `manim_repair`, `manim_repair_many`), nhánh Manim ở `run.py`, `merger.py`, `checker.py`, `main.py` |
| api-gateway | 2 | `routes/projects.js`, `server.js` |
| tts | 1 | `adapters/messaging/cancellation.py` (chú thích) |

Tài liệu và hạ tầng: `README.md` ("Manim Educational Video Generation Tool"), `docs/contracts/shared-artifacts.md` (`/shared/.manim-media/`), biến `RENDER_CACHE_ROOT`, `RENDER_MEMORY_LIMIT_GB`, `RENDER_TIMEOUT_SECONDS` (mô tả theo Manim) trong `.env.example` và `docker-compose.yml`, các ADR nhắc Manim.

### Tên chung đang mang chữ Manim

Ba thứ Remotion đang dùng chung nhưng tên là Manim:

- `ManimScriptRendererPort` (`rendering/domain/ports.py`): port mà cả `RemotionScriptRenderer` cũng implement (`remotion_renderer.py:194`).
- `ManimEngineerStepPage` (web-gui): màn bước Code của cả hai engine.
- Cột `projects.manim_scene_class_name` (orchestrator): id composition của Remotion.

### Intro/outro của kênh

- Creator đưa intro/outro vào bằng cách **tải file lên**: `POST /v1/channel-assets/intro|outro` ở api-gateway (`services/api-gateway/src/routes/channelAssets.js`) → orchestrator gửi `normalize_channel_asset` → video-assembly chuẩn hoá.
- Nhánh **dựng intro/outro bằng Manim** (`render_channel_asset`, scene trong `conceptflow/channel_idents.py`) chỉ chạy khi có lệnh AMQP `render_channel_asset`. Không service nào gửi lệnh này (grep orchestrator, api-gateway, scripts, infra); đây là lối admin không có giao diện.
- File intro/outro đã có trong `/shared/channel-assets/` là mp4, không phụ thuộc Manim lúc ghép.

### Dữ liệu

Chưa đếm được trong DB có bao nhiêu project dùng Manim và có intro/outro nào đang đăng ký, vì Docker không chạy lúc phân tích (2026-10-12). Đếm lại khi thiết kế.

## Câu trả lời của Creator

> "1. Project đã làm bằng Manim […] a. (Khuyến nghị) Giữ ở chế độ chỉ xem […] 2. Intro/outro a. (Khuyến nghị) Bỏ nhánh dựng bằng Manim, giữ đường tải file lên […] 3. Ba tên chung mang chữ Manim a. (Khuyến nghị) Đổi tên luôn trong CR này […] 4. Luật QC bố cục ở video-assembly […] a. (Khuyến nghị) Xoá phần luật bố cục theo Manim, giữ phần đo âm lượng và các thông số video."

Chốt: **1a, 2a, 3a, 4a**.

## Thay đổi trong lúc code (2026-10-12)

> "ko cần quan tâm project cux" — trả lời câu hỏi: "Xoá hàng DB, bỏ hết code chỉ-xem (Recommended)".

Thay cho 1a: **không giữ project Manim**. T0 đếm được 24 project `render_engine = 'manim'`, tất cả ở trạng thái `deleting`. Migration orchestrator xoá các hàng này cùng `saga_steps`, `outbox_events`, `project_events` của chúng (đúng các bảng `ProjectRepository.Delete` xoá). Toàn bộ phần chỉ-xem bị bỏ: không có giá trị "engine cũ", không có lỗi 409 ngừng hỗ trợ, không có nhãn/thẻ ở web-gui. FR2 và tiêu chí chấp nhận 2 dưới đây được thay bằng: *project Manim bị xoá khỏi DB orchestrator; mọi project còn lại là Remotion*. File của 24 project trên `/shared` và dữ liệu authoring-service theo `project_id` của chúng không được dọn tự động.

## Yêu cầu

- **FR1 — Một engine.** Hệ thống chỉ dựng video bằng Remotion. Không còn chỗ nào cho Creator chọn engine; project mới luôn là Remotion.
- **FR2 — Project Manim cũ chỉ xem.** Project có `render_engine = 'manim'` vẫn hiện trong danh sách với nhãn "Manim (ngừng hỗ trợ)". Video đã xong xem được và đăng được. Mọi thao tác chạy lại đều bị từ chối, cả ở server lẫn giao diện: dựng lại, chạy lại bước, nhân bản, duyệt/chạy tiếp, sửa cấu hình, sinh nội dung bằng AI, lưu code. Xoá vẫn được.
- **FR3 — Intro/outro chỉ qua tải file lên.** Bỏ chuỗi dựng intro/outro bằng Manim (rendering `render_channel_asset` → orchestrator `register_channel_asset` → video-assembly). Đường tải file lên (`POST /v1/channel-assets/*` → `normalize_channel_asset`) giữ nguyên. Intro/outro đang đăng ký vẫn dùng được.
- **FR4 — Tên chung không mang chữ Manim.** `ManimScriptRendererPort` → `ScriptRendererPort`; `ManimEngineerStepPage` → `CodeStepPage`; cột `projects.manim_scene_class_name` → `scene_class_name` (giá trị giữ nguyên).
- **FR5 — QC chỉ còn luật không phụ thuộc Manim.** Bỏ các luật đọc `layout_marks` của Manim (tràn khung, chữ đè, chữ nhỏ, tương phản) và đường đi của `layout_marks`. Giữ: hình đứng yên, chồng lời thoại, chồng cue phụ đề, độ to (LUFS).
- **FR6 — Image và cấu hình gọn.** Image rendering không còn Manim, texlive, cairo/pango dev. Biến môi trường chỉ Manim dùng bị bỏ. Cache `/shared/.manim-media` bị xoá.
- **FR7 — Tài liệu khớp code.** README, hợp đồng `/shared`, hợp đồng message của rendering, hợp đồng `/v2/code/*` và `/v1/check/*` (giá trị `engine`), ADR mới ghi quyết định.

**Tiêu chí chấp nhận**

1. Wizard tạo project mới không có ô chọn engine; project tạo ra có `render_engine = 'remotion'`; chạy trọn tới video cuối như hôm nay.
2. Project Manim cũ (nếu DB có) hiện nhãn "Manim (ngừng hỗ trợ)"; mở được video, đăng được; không có nút Dựng lại, Nhân bản, Chạy lại; gọi thẳng API các thao tác đó trả `409` với thông điệp tiếng Việt.
3. Tải intro/outro lên vẫn chuẩn hoá và ghép vào video như cũ.
4. `docker compose exec rendering python -c "import manim"` báo không có module; image rendering nhỏ hơn trước (ghi số đo).
5. `grep -ri manim services/` (trừ `.venv`, `node_modules`) chỉ còn: giá trị cũ `"manim"` để nhận ra project cũ, các chuỗi giao diện "Manim (ngừng hỗ trợ)", migration DB và test của chúng.
6. Test của mọi service đổi đều qua; lint sạch theo `docs/code-standards-rules.md`.

**Ngoài phạm vi:** chuyển project Manim sang Remotion (script Python không chuyển được); viết intro/outro mặc định bằng Remotion; bật lại bước `qc_video`.

## Giải pháp đề xuất

### 1. Giữ giá trị `manim` như dấu "project cũ", không như engine

`render_engine` vẫn là cột của orchestrator. Giá trị `remotion` là engine duy nhất chạy được; giá trị `manim` chỉ còn để nhận ra project cũ.

- orchestrator `domain/project.go`: `RenderEngineRemotion` là engine duy nhất; hằng `RenderEngineManim` đổi tên `RenderEngineRetiredManim` (giữ giá trị `"manim"`), `DefaultRenderEngine = RenderEngineRemotion`. `IsValid()` (dùng cho input) chỉ nhận `remotion`. Thêm `(*Project).EngineRetired() bool` và lỗi `ErrEngineRetired`.
- Repository đọc `manim` từ DB giữ nguyên giá trị (không ép về mặc định), để project cũ vẫn được nhận ra.
- Migration orchestrator: `ALTER TABLE projects ALTER COLUMN render_engine SET DEFAULT 'remotion'`. Không đổi giá trị của hàng cũ.
- authoring-service `domain/project.go` (bản sao project nhận từ orchestrator qua HTTP): làm tương tự.

Lý do: xoá hẳn giá trị thì project cũ đọc lên thành Remotion, và mọi thao tác chạy lại sẽ đưa script Python cho Remotion, hỏng theo cách khó hiểu. Một dấu rõ ràng cho phép chặn sớm với thông điệp đúng.

### 2. Chặn thao tác trên project cũ ở server, ẩn ở giao diện

Server là nơi chặn thật (FR2), giao diện chỉ ẩn nút cho khỏi bấm nhầm.

- orchestrator: các use case chạy lại kiểm `project.EngineRetired()` ngay đầu và trả `ErrEngineRetired`: `StartRenderSaga` (gồm dựng lại), `RetryStep`, `ForkProject`, `ReviewOutline`, cập nhật wizard (`wizard.go`), `CreateProjectDraft` khi input có engine khác `remotion`. Router đổi `ErrEngineRetired` thành `409` với thông điệp: "Dự án này dựng bằng Manim, đã ngừng hỗ trợ: chỉ xem, đăng hoặc xoá được." Xoá (`DELETE /v1/projects/{id}`) và đăng (`StartPublishSaga`) không bị chặn.
- authoring-service: các use case sinh nội dung, sinh code, lưu code, kiểm tra kiểm `RenderEngine` của project nhận từ orchestrator và trả cùng thông điệp `409`.
- web-gui: `RenderEngineBadge` chỉ hiện cho project cũ, chữ "Manim (ngừng hỗ trợ)". `ResultPage` ẩn Dựng lại và Nhân bản. Mở một project cũ chưa xong thì `ResumeProjectPage` hiện thẻ thông báo (thông điệp trên) và nút Xoá, không đưa vào wizard. Thẻ dùng `.reveal`/`usePresence` và token Neubrutalism như mọi thẻ khác (`docs/ux-ui-design-rules.md`).

### 3. Gỡ Manim theo service

| Service | Gỡ | Đổi |
|---|---|---|
| rendering | `adapters/rendering/manim_renderer.py`, `engine_router.py`, gói `conceptflow/`, phần lint Manim của `domain/script_lint.py` (`lint_manim_script` và các luật đọc `conceptflow`; `LintIssue` và `blocking_issues` được lint Lottie của Remotion dùng nên chuyển sang module riêng `domain/lint_issue.py`), `application/render_channel_asset.py`, `tools/gen_theme_reference.py`; nhánh `engine == "manim"` trong `render_script.py`, `validate_script.py`, `check_script.py`; handler `render_channel_asset` trong `adapters/messaging/consumer.py`, envelope `channel_asset_rendered`/`channel_asset_render_failed` trong `producer.py`; `ChannelAssetRendererPort`; `manim==0.18.*` trong `requirements.txt`; texlive, cairo, pango dev trong `Dockerfile`; `compute_channel_asset_path` và đường cache Manim trong `adapters/storage/artifact_paths.py` | `ManimScriptRendererPort` → `ScriptRendererPort` (`domain/ports.py`, `remotion_renderer.py:194` và mọi chỗ gọi); `main.py` nối thẳng `RemotionScriptRenderer`; lệnh có `engine` khác `remotion` bị từ chối bằng lỗi rõ ràng (sự kiện thất bại như mọi lỗi render); phần lint Lottie trong `lottie_catalog.py` không còn nhắc Manim |
| llm-service | `prompts.manim_chunk`, `manim_cast`, `manim_repair`, `manim_repair_many`; `merger.merge_manim`, `manim_fn`, `manim_scene_class`; `checker.shots_from_manim_trace`; nhánh Manim trong `run.py` | `engine` nhận đúng `remotion`; giá trị khác trả `400` (`main.py:320`, `run.py:602`, `checker.py:76`) |
| authoring-service | Vai prompt `manim_engineer`, `manim_engineer_ai`, `manim_adjust` (`domain/prompt_template.go`), seed `manim*VI` (`prompt_template_seeds.go:448-611`) và bản AI, `prompts/manim_adjust.txt`, `prompts/theme_reference_{vi,en}.txt` và `ThemeReference` (`prompt_vars.go`); nhánh Manim trong `script_templates.go`, `code_pipeline.go`, `generate_authoring.go`, `render_prompt*.go`, `code_segment.go` | Migration: xoá các hàng `prompts` có vai Manim; chặn project cũ (mục 2); bỏ điều kiện `RenderEngine == Remotion` ở `generate_authoring_code.go:139,167,474`, `storyboard_warnings.go:34` vì luôn đúng; golden prompt cập nhật |
| orchestrator | Nhánh Manim ở `handle_step_event.go:787`, `retry_step.go`, `authoring_port.go:28` (`NeedsIllustrations` luôn đúng); `registerRenderedChannelAsset` và nhánh `channel_asset_rendered`; trường `LayoutMarks` và cột `layout_marks` | Mục 1, 2; `ManimSceneClassName` → `SceneClassName`, migration đổi tên cột `manim_scene_class_name` → `scene_class_name` (khối `DO $$` kiểm `information_schema` để chạy lại an toàn); migration xoá cột `layout_marks` (dữ liệu dẫn xuất cho bước QC đã tắt) |
| video-assembly | `RegisterChannelAssetCommandHandler` và đăng ký trong `main.py`; trong `qc_rules.py`: hằng khung Manim, tiện ích bố cục, `check_frame_overflow`, `check_text_overlap`, `check_text_too_small`, `check_low_contrast`, `font_size_to_pixels`, `contrast_ratio`; đọc `layout_marks` ở `QCVideoCommandHandler` | Giữ `check_static_frame`, `check_narration_overlap`, `check_subtitle_cue_overlap`, `check_loudness`; `BLOCKING_RULES` chỉ còn `narration_overlap`; chú thích `domain/models.py` |
| web-gui | `RenderEnginePicker.tsx`, nhánh Manim ở `flow.ts:154` (bước 5 luôn dùng), `AuthoringModeBar.tsx:52`, `VisualDirectorStepPage.tsx:102,126`, `IllustrationsStepPage.tsx:53`, `ScriptAuthoringSettingsStepPage.tsx:77`, `PipelineSettingsBar.tsx:64,92`, `ProductionSettingsPanel.tsx:91`, `scriptValidation.ts`, `authoringHint.ts`, `ProjectDraftContext.tsx:167` (`renderEngine` khỏi draft), `client.ts:1268,1297` | `ManimEngineerStepPage` → `CodeStepPage` (file, export, `data-testid` `code-step-*`, route `App.tsx:67`); `VideoFontPicker.tsx` dùng CSS module riêng thay vì `RenderEnginePicker.module.css`; `RenderEngineBadge` như mục 2; `ResumeProjectPage.tsx:22` không mặc định `manim` |
| api-gateway | Chuỗi engine trong `routes/projects.js`, `server.js` | Giá trị engine nhận đúng `remotion` |
| tts | — | Chú thích `adapters/messaging/cancellation.py` |

### 4. Biến môi trường và volume

- Bỏ `RENDER_MEMORY_LIMIT_GB`, `RENDER_CACHE_ROOT`, `DRY_RUN_TIMEOUT_SECONDS`, `RENDER_QUALITY` của rendering (chỉ `ManimScriptRenderer` đọc, `main.py:85-98`) khỏi `docker-compose.yml`, `.env.example`, `README.md`. Giữ `RENDER_TIMEOUT_SECONDS` (Remotion dùng).
- Xoá thư mục `/shared/.manim-media` một lần sau khi rebuild. Đây là cache của Manim, không còn ai đọc.

### 5. Tài liệu

- ADR-0033 "Chỉ một engine dựng hình: Remotion": bối cảnh (hai engine, short chỉ có Remotion, mọi tính năng mới từ CR-042 tới CR-068 làm cho Remotion), quyết định, hệ quả (project cũ chỉ xem, intro/outro chỉ tải lên, QC bố cục Manim bỏ). Thêm vào `aidlc-docs/decisions/README.md`.
- `README.md`: tên và mô tả dự án theo Remotion, bảng biến môi trường.
- `docs/contracts/shared-artifacts.md`: bỏ `/shared/.manim-media/` và `channel-assets/{kind}/{quality}/rendered.mp4`; `tests/contracts/test_shared_artifacts_contract.py` khớp lại.
- `aidlc-docs/construction/rendering-service/low-level-design/interface-contracts.md`: bỏ `render_channel_asset`, `channel_asset_rendered`, `channel_asset_render_failed`; `engine` chỉ `remotion`. Ghi `register_channel_asset` bị bỏ ở hợp đồng của video-assembly và orchestrator.
- `docs/contracts/authoring-llm-code-v2.md`: `engine` chỉ `remotion`.

## Phụ thuộc

Đã kiểm các nhánh chưa merge và các worktree đang mở:

| Mục | Kết quả |
|---|---|
| CR-064, CR-069, CR-070 | Creator đã đóng (bỏ hẳn) |
| `chore/worktree-per-cr` | Chỉ công cụ worktree — độc lập |
| CR-072 (`feature/cr-072-2d-film-pipeline`, ý tưởng 2D, chưa có code) | Độc lập về code; CR-072 phụ thuộc CR-071 (thiết kế chỉ cho Remotion). Cả hai cùng thêm mục vào cuối `aidlc-docs/audit.md`, xung đột nhỏ khi merge, giải quyết bằng giữ cả hai |

## Phạm vi

Service đổi: rendering, llm-service, authoring-service, orchestrator, video-assembly, web-gui, api-gateway, tts.

Hợp đồng và DB:

- Message: bỏ `render_channel_asset`, `channel_asset_rendered`, `channel_asset_render_failed`, `register_channel_asset`; `engine` trong lệnh render và các route `/v1/check/*`, `/v2/code/*` chỉ nhận `remotion`. Bên gọi duy nhất của các route này (authoring-service) không còn gửi `manim` sau CR này; giá trị cũ bị từ chối rõ ràng chứ không âm thầm.
- DB orchestrator: mặc định `render_engine` → `remotion`; đổi tên cột `manim_scene_class_name` → `scene_class_name`; xoá cột `layout_marks`.
- DB authoring-service: xoá hàng `prompts` có vai Manim.

`graphify affected` đã chạy cho `ManimScriptRenderer` (chỉ `main.py` và test của nó gọi). Các symbol còn lại (`ManimScriptRendererPort`, `RenderEngineManim`, `ManimSceneClassName`, `ManimEngineerStepPage`, `LayoutMarks`) được kiểm khi chia task.

## Kế hoạch thực hiện

Thứ tự để mỗi bước test được riêng:

1. **orchestrator**: domain (mục 1, 2), migration, repository, router `409`, gỡ nhánh Manim và `layout_marks`, gỡ chuỗi intro/outro dựng bằng Manim; test use case cho từng thao tác bị chặn và thao tác được phép (xoá, đăng).
2. **authoring-service**: chặn project cũ, gỡ vai và seed prompt Manim, migration xoá hàng prompt, bỏ điều kiện engine; golden prompt.
3. **llm-service**: gỡ prompt, merger, checker Manim; `engine` chỉ `remotion`; test `400` cho `manim`.
4. **rendering**: gỡ renderer, gói `conceptflow`, lint, intro/outro; đổi tên port; image mới; test renderer Remotion và lệnh có engine `manim` bị từ chối.
5. **video-assembly**: gỡ handler `register_channel_asset`, luật QC bố cục; test QC còn lại.
6. **api-gateway, tts**: giá trị engine, chú thích.
7. **web-gui**: gỡ chọn engine, đổi tên `CodeStepPage`, nhãn và thẻ project cũ, bước 5 luôn dùng; test component và `tsc`, `eslint`.
8. **Tài liệu**: ADR-0033, README, hợp đồng; test hợp đồng `/shared`.
9. Rebuild cả 8 service, xoá `/shared/.manim-media`, kiểm trực tiếp (mục Kiểm tra).

## Kiểm tra

- Test: `go test ./...`, `go vet`, `gofmt -l` (orchestrator, authoring-service); `pytest`, `ruff` (rendering, llm-service, video-assembly, tts, `tests/contracts`); `npm test`, `tsc --noEmit`, `eslint` (web-gui, api-gateway).
- Rebuild: `scripts/worktree.sh rebuild rendering llm-service authoring-service orchestrator video-assembly web-gui api-gateway tts`, tất cả healthy.
- Đếm trong DB: số project `render_engine = 'manim'` (theo trạng thái) và intro/outro đang đăng ký, trước và sau migration.
- Trực tiếp: tạo project mới chạy trọn tới video; mở project Manim cũ (nếu có): nhãn, xem, đăng, không có nút chạy lại, API trả `409`; tải lại intro/outro; `import manim` thất bại trong container; kích thước image trước/sau.

## Rủi ro

- **Project Manim đang chạy dở trong saga lúc deploy.** Sau rebuild, rendering từ chối lệnh `manim` thì saga đó thất bại ở bước render. Kiểm trước khi rebuild: không có project Manim nào đang ở trạng thái chạy; nếu có, báo Creator trước.
- **Xoá dữ liệu.** Hàng prompt Manim (kể cả bản Creator đã sửa), cột `layout_marks`, cache `/shared/.manim-media` mất hẳn. Không có tính năng nào còn đọc chúng.
- **Intro/outro dựng bằng Manim trước đây.** File mp4 đã đăng ký vẫn dùng được; chỉ không dựng lại được bằng Manim.
- **Docker đang tắt lúc phân tích.** Chưa đếm được project Manim và intro/outro; đếm ở bước code trước khi chạy migration.
- **Thay đổi lớn, 8 service.** Không tách nhỏ được mà không để hệ thống ở trạng thái nửa vời (ví dụ web-gui còn cho chọn Manim khi rendering đã gỡ). Giảm rủi ro bằng thứ tự ở Kế hoạch thực hiện và test từng service.
