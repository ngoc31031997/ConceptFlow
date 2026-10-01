# CR-061 — Thiết kế: gỡ đường cắt clip dọc cũ

## Yêu cầu gốc (nguyên văn)

> Phần cắt clip cũ vẫn còn trong code. Việc gỡ nó thuộc CR-061, bạn bảo là mình mở => làm cái này đi

Trả lời các lựa chọn:

> A luồng còn 13 bước
> B chọn B1
> C C1 xoá hẳn

Chốt: A1 (luồng 13 bước, đổi số `project_events`), B1 (form YouTube hiện cho short), C1 (drop 4 cột).

Bối cảnh: CR-060 (Q6, Creator chọn (a)) dựng short dọc 1080×1920 từ đầu và bỏ "Cả hai" khỏi bộ chọn đầu ra; đường cắt clip từ video 16:9 còn trong code nhưng không lối nào dẫn tới, và được hẹn gỡ ở CR-061 (ADR-0031, mục Hệ quả).

## Hiện trạng

Đường cắt clip chỉ chạy khi `video_output_mode = "both"`, giá trị mà bộ chọn không còn đưa ra.

**Dữ liệu thật (đã kiểm ngày 2026-10-01):** 27/27 project là `long`; không project nào ở `generating_clips` / `failed_at_generate_clips`; `clips`, `clip_requests`, `clip_marks` đều rỗng; không có hàng `saga_steps` nào của `generate_clips`; không script nào (orchestrator lẫn authoring-service) chứa `self.clip(`; không có thư mục `/shared/*/clips`. `project_events`: 9 hàng ở `flow_step = 13`, 0 hàng ở 12 và 14. `prompts` có đúng 1 hàng `short_script` (hàng hệ thống).

**orchestrator**
- `internal/domain/flow.go:9-25` luồng 14 bước, `FlowSplit = 12 // Cắt short`, `FlowResult = 13`, `FlowPublish = 14`, `FlowStepsTotal = 14`; `FlowStateFor` (`:120-124`) ánh xạ `generating_clips`/`failed_at_generate_clips` về bước 12; `FlowStepLabel` (`:149-154`).
- `internal/domain/project.go:47-58` `StatusGeneratingClips`, `StatusFailedGenerateClips`; `:74` trong `IsInFlight`; `:118` `StepGenerateClips`; `:150` trong `FailedStatusForStep`; `:243-275` `ModeBoth`, `IsValid` nhận `both`, `WantsClips()`; `:422-451` các trường `ClipMarks`, `IntroDurationSeconds`, `ClipRequests`, `Clips`; `:485-494` `ClipResult`.
- `internal/domain/clip_rules.go` (+ `clip_rules_test.go`): `ClipPresetShort/Long`, `ValidateClipDuration`, env `CLIP_PRESET_*`.
- `internal/application/handle_step_event.go:91` `clips_generated → StepGenerateClips`; `:449-450` nhánh `onClipsGenerated`; `:543` và `:878` đọc `clip_marks` từ `script_validated`/`rendering_completed`; `:583-593` cảnh báo "không có `with self.clip`"; `:948` lưu `intro_duration_seconds`; `:962-981` `advanceAfterVideoReady` rẽ sang `generate_clips` khi `WantsClips()`; `:1033-1067` `onClipsGenerated`, `generateClipsPayload`.
- `internal/application/event_payload.go:242-` `parseClipResults`; `:288-` `buildClipRequests`.
- `internal/application/fork_project.go:154-161` xoá các trường clip khi fork.
- `internal/adapters/http/router.go:327-328, 804-885` `POST/GET /v1/projects/{id}/clips`; `dto.go:149, 244-280, 462` DTO clip; `dto.go:39, 159` chú thích `"both"`.
- `internal/adapters/postgres/db.go:176-201` cột `clip_marks`, `clip_requests`, `clips`, `intro_duration_seconds`, chú thích `video_output_mode` có `both`; `project_repository.go:34, 63, 398, 449` đọc/ghi các cột này.
- `UpdateStatus` (`project_repository.go:476-497`) ghi `flow_step`/`from_flow_step` vào `project_events` bằng số của `FlowStateFor`.

**authoring-service** (bản sao domain dùng chung luật luồng/trạng thái)
- `internal/domain/flow.go:22-25, 120-155` cùng hằng và ánh xạ bước 12.
- `internal/domain/project.go:47-58, 74, 130, 162, 259-287, 380` trạng thái/bước/`ModeBoth`/`WantsClips`.
- Trợ lý soạn script short: `internal/application/suggest_short_script.go` (+test), `adapters/http/router.go:37, 68, 222-228, 467-472, 532-558` route `POST /v1/short-script-suggestions`, `dto.go:200-216`, `adapters/llm/llm_service_client.go:339-, 391-` `SuggestShortScript`, `application/llm_provider.go:15`, `cmd/authoring/main.go:70, 127`.
- Prompt `short_script`: `domain/prompt_template.go:35, 44`, `prompt_vars.go:78-79`, `prompt_template_seeds.go:30`, `domain/prompts/short_script.txt`, `testdata/prompt_golden.json` (3 mục `short_script`).

**llm-service**: `app/main.py:62, 282-` `ShortScriptBody` + route `suggest_short_script`; `app/tasks.py:65-95, 158-` `build_short_script_prompt`, `suggest_short_script`; `tests/test_tasks.py:56-59, 98-`.

**rendering**: `conceptflow/scene.py:334-343` `ConceptFlowScene.clip()`; `conceptflow/narration.py:188-` `clip()` runtime; `adapters/rendering/manim_renderer.py:268-271, 324, 337, 573-581` `_read_clip_marks`; `domain/models.py:108-114, 141-144` `clip_marks`; `adapters/messaging/producer.py:37, 49, 64, 83, 95, 125` và `consumer.py:133, 209` gửi `clip_marks`; test `test_manim_renderer.py:393-424`, `test_producer.py`, `conceptflow/test_narration.py:108-162, 246`.

**video-assembly**: `adapters/clips/vertical_clip.py` (187 dòng); `adapters/messaging/consumer.py:37, 43, 63, 790-900, 915-929` `GenerateClipsCommandHandler`; `producer.py:30-46` (`intro_duration_seconds` trên `video_assembled`, chỉ để `generate_clips` dùng), `:89-96` `clips_generated_envelope`; `progress.py:32, 52-60` `publish_clip_progress`; `domain/clip_rules.py`; `adapters/storage/artifact_paths.py:56-60` `clip_output_path`; `main.py:21, 37, 87-92`; test `tests/adapters/test_generate_clips.py`, `tests/domain/test_clip_rules.py`, `tests/adapters/test_purge.py:88-95` (dựng thư mục `clips` giả để thử purge).

**api-gateway**: `src/handlers/clipHandler.js` (90 dòng), `src/routes/projects.js:7, 23-24, 85-87`; `tests/routes/projects.test.js:68-162`.

**web-gui**
- `src/utils/flow.ts:16-31` nhãn 14 bước (có "Cắt short"), `:34-49` mục đích từng bước (bước 12; bước 13 nói "cắt thêm clip"), `:60, 69` `FLOW_RESULT = 13`, `FLOW_PUBLISH = 14`, `:121-127` `FLOW_PHASES` (`[9,10,11,12]`, `[13,14]`), `:130` `AUTO_STEPS` có `FLOW_TTS + 3`, `:162` `step === 12` → "skipped", `:179-184` lý do "Không dùng" cho bước 12.
- `src/pages/RenderPage.tsx:24-35, 100-103` bước `generate_clips` trong tracker.
- `src/pages/ResultPage.tsx:9, 103-105, 113, 130-132` `ClipsPanel` cho `both`, phụ đề "cắt clip".
- `src/pages/PublishPage.tsx:10, 38-42, 136-145, 185, 198, 202-` `ClipsPanel` cho `short`/`both`; với `short`, form đăng YouTube **thu gọn mặc định** vì giả định cũ "short = cắt clip từ video dài, Creator đến đây vì clip".
- `src/components/CompanionProjectCard.tsx:5, 48-54` `ClipsPanel` cho `both`.
- `src/components/ClipsPanel.tsx`, `ShortScriptAssistant.tsx` (không màn nào import), `ProgressTracker.tsx:40-59` đếm "Clip", `Disclosure.tsx:19` và `styles/glass.module.css:218` chú thích nhắc `ShortScriptAssistant`.
- `src/api/client.ts:105-108` `getProjectClipUrl`, `:368-` `suggestShortScript`, `:457` role `short_script`; `src/pages/PromptSettingsPage.tsx:31` mục prompt "soạn script Shorts/TikTok".
- `src/types/index.ts:18-19, 118-131, 159-161` `"both"`, `clips`, `Clip`, `clip_index/clip_total`; `src/context/ProjectDraftContext.tsx:116-123` `VideoOutputMode` có `"both"`; `src/utils/pipelineLabels.ts:25, 40, 73, 103-110`.
- Test: `tests/components/ClipsPanel.test.tsx`, `ShortScriptAssistant.test.tsx`, `VideoOutputModePicker.test.tsx:27`.

**Tài liệu**: `docs/contracts/shared-artifacts.md:19` (`/shared/{project_id}/clips/...`), `README.md:59` (`generate_clips` của video-assembly), ADR-0031 mục Hệ quả (`:32`).

Không thuộc phạm vi dù cùng chữ "clip": `Scene.ClipPath` (đoạn render từng cảnh), clip intro/outro của kênh, "preview clips" của giọng đọc, `clipboard`, CSS `clip-path`.

## Yêu cầu

- **FR1.** Gỡ hẳn bước `generate_clips` khỏi saga: không còn trạng thái `generating_clips`/`failed_at_generate_clips`, sự kiện `clips_generated`, lệnh `generate_clips`, worker của video-assembly và mã cắt clip dọc.
- **FR2.** Luồng còn **13 bước**: … 11 Ghép video, **12 Kết quả, 13 Đăng video**. Mọi nơi đánh số (orchestrator, authoring-service, web-gui) dùng cùng số mới; lịch sử `project_events` đã ghi được đổi số theo (13→12, 14→13) để màn Nhật ký không lệch nhãn.
- **FR3.** `video_output_mode` chỉ còn `long` | `short`; API từ chối `both` (400) như mọi giá trị lạ.
- **FR4.** Gỡ đánh dấu `with self.clip(...)` khỏi kit Manim và mọi chỗ mang `clip_marks` (rendering → orchestrator).
- **FR5.** Gỡ trợ lý soạn script short cũ: component `ShortScriptAssistant`, route `POST /v1/short-script-suggestions` (authoring-service, llm-service), prompt `short_script` (seed, file, golden, mục trong màn Prompt), và hàng `short_script` trong bảng `prompts`.
- **FR6.** Gỡ API clip: `POST/GET /v1/projects/{id}/clips` (orchestrator), `GET /v1/projects/{id}/clips/{name}/{preset}` (gateway), trường `clips` trong `GET /v1/projects/{id}`.
- **FR7.** Gỡ `intro_duration_seconds` khỏi `video_assembled` và khỏi orchestrator (chỉ `generate_clips` dùng). Việc dịch timeline theo intro bên trong video-assembly (`effective_lead_in`) giữ nguyên.
- **FR8.** Màn Đăng video của project `short` hiện form YouTube như video dài (xem Phương án B).

**Tiêu chí chấp nhận**
- `grep` các tên `generate_clips|clips_generated|generating_clips|ClipsPanel|ShortScript|short_script|clip_marks|clip_requests|vertical_clip|ValidateClipDuration|WantsClips|ModeBoth|self.clip|FlowSplit|Cắt short` trên `services/` không còn kết quả (trừ chuỗi migration id và chú thích migration trong `db.go`).
- Video dài và short mới đi hết tới `ready_to_publish` với `flow_step` 12, rồi publish ở 13; thanh bước hiện 13 ô, giai đoạn "Sản xuất" = 9–11, "Hoàn tất" = 12–13.
- Màn Nhật ký của một project cũ đã tới Kết quả hiện "Kết quả" (không phải "Đăng video").
- `POST` start/wizard với `video_output_mode: "both"` trả 400.
- Toàn bộ test các service bị đụng pass; build lại các service và healthy.

**Ngoài phạm vi**
- Thêm lại một công cụ cắt clip dọc ở màn Kết quả (phương án (c) của Q6 CR-060, đã không chọn).
- Đổi gì trong cách dựng short dọc của CR-060.
- Sửa `docs/review/data-flow-review.md` (bản review đóng băng ngày 2026-09-22; ghi lại lịch sử, không phải tài liệu sống).

## Giải pháp đề xuất

Gỡ theo chiều dữ liệu chảy, từ chỗ sinh ra tới chỗ hiển thị, mỗi service một đợt, và đổi số luồng trong cùng CR. Dữ liệu thật đã sạch nên không cần giai đoạn chuyển tiếp: không có project nào giữa chừng bước 12 để giữ tương thích.

### Phương án A — đánh số luồng sau khi bỏ bước 12

- **A1 (đề xuất): đánh lại thành 13 bước** — Kết quả = 12, Đăng video = 13, `FlowStepsTotal = 13`; thêm migration có guard trong `db.go` (đúng khuôn `cr046_illustrations_flow_step`) đổi `flow_step` và `from_flow_step` của `project_events`: 13→12, 14→13, và 12→11 cho mọi hàng cũ ở 12 (hiện là 0 hàng; gán về Ghép video là bước liền trước). Ưu: số trên thanh bước liền mạch, không có ô "Không dùng" vĩnh viễn; hai lần đổi số trước (CR-046) đã đi đúng đường này. Nhược: URL `?step=13/14` cũ (nếu ai lưu) trỏ lệch một bước — chấp nhận được.
- A2: giữ 14 bước, bước 12 luôn "Không dùng". Không đụng lịch sử, nhưng để lại một ô chết trên thanh bước của mọi video và code xử lý "skipped" chỉ để che nó — trái mục tiêu của CR.

### Phương án B — form YouTube ở màn Đăng video cho `short`

`PublishPage` hiện thu gọn form YouTube với `short` vì coi short là "clip cắt ra để tải về". Từ CR-060 short là một video dọc hoàn chỉnh, nơi duy nhất để ra kênh là đăng nó; bỏ `ClipsPanel` đi thì màn này với short chỉ còn video và một nút "mở form".
- **B1 (đề xuất):** bỏ toggle, form YouTube hiện như video dài.
- B2: giữ toggle thu gọn như nay.

### Phương án C — cột DB không còn dùng

- **C1 (đề xuất):** `ALTER TABLE projects DROP COLUMN IF EXISTS clip_marks, clip_requests, clips, intro_duration_seconds` trong `db.go` (đã kiểm: toàn rỗng/0). Không để cột chết trong schema (code-standards: không dead code). Nhược: quay về image cũ thì image cũ hỏng vì thiếu cột — rollback phải đi kèm `ADD COLUMN` lại.
- C2: giữ cột, chỉ bỏ đọc/ghi. Rollback dễ, nhưng để lại 4 cột không ai dùng.

### `both` trong DB

Bỏ `ModeBoth`; `IsValid()` chỉ nhận `long`/`short` (và `""` như nay = mặc định). Thêm migration guard `UPDATE projects SET video_output_mode = 'long' WHERE video_output_mode = 'both'` (hiện 0 hàng) để một hàng sót không thể đọc lên với giá trị không hợp lệ.

### Prompt `short_script` trong DB

`SeedPrompts` chỉ upsert vai trò đang có, nên bỏ seed không xoá hàng cũ. Thêm vào `PurgeLegacyPrompts` (đã chạy mỗi lần khởi động, idempotent) câu `DELETE FROM prompts WHERE role = 'short_script'` — xoá cả hàng hệ thống lẫn hàng Creator nếu có (hiện chỉ có 1 hàng hệ thống), vì vai trò không còn hợp lệ thì hàng đó không mở/sửa/dùng được ở đâu.

### Message cũ đang nằm trong hàng đợi

Orchestrator bỏ qua trường lạ (`clip_marks`, `intro_duration_seconds`) trong `script_validated`/`rendering_completed`/`video_assembled` → không vỡ. Một `generate_clips` hay `clips_generated` sót sẽ rơi vào nhánh event/lệnh không biết của dispatcher (đã có: log + DLQ). Đã kiểm 0 saga ở bước này nên thực tế không có.

## Phạm vi

| Service | Thay đổi |
|---|---|
| orchestrator | domain (flow, project, clip_rules), application (handle_step_event, event_payload, fork_project), http (router, dto), postgres (db.go migration + drop cột, project_repository) + test |
| authoring-service | domain (flow, project, prompt_template, prompt_vars, seeds, prompts/short_script.txt, golden), application (suggest_short_script, llm_provider), http (router, dto), llm client, main.go, postgres (PurgeLegacyPrompts) + test |
| llm-service | main.py, tasks.py + test |
| rendering | conceptflow (scene, narration), manim_renderer, models, producer, consumer + test |
| video-assembly | xoá adapters/clips, domain/clip_rules; consumer, producer, progress, artifact_paths, main + test |
| api-gateway | xoá clipHandler, routes/projects.js + test |
| web-gui | flow.ts, RenderPage, ResultPage, PublishPage, CompanionProjectCard, ProgressTracker, client.ts, types, ProjectDraftContext, pipelineLabels, PromptSettingsPage, Disclosure/glass chú thích; xoá ClipsPanel, ShortScriptAssistant + test |
| Tài liệu | ADR-0031 (Hệ quả: đã gỡ), `docs/contracts/shared-artifacts.md`, `README.md` |

Hợp đồng thay đổi: bỏ lệnh `generate_clips`, sự kiện `clips_generated`; bỏ trường `clip_marks` (`script_validated`, `rendering_completed`), `intro_duration_seconds` (`video_assembled`), `clip_index/clip_total` (progress); bỏ 3 route HTTP clip và 1 route short-script (authoring) + 1 (llm-service). Không có consumer nào khác ngoài các chỗ liệt kê (đã grep toàn `services/`, `infra/rabbitmq/definitions.json` không có binding riêng cho clip).

DB: orchestrator — migration đổi số `project_events`, `both`→`long`, drop 4 cột; authoring-service — xoá hàng prompt `short_script`.

## Kế hoạch thực hiện

Mỗi bước: xoá code + test đi kèm, sửa chú thích bị sai theo `docs/code-standards-rules.md` (không ghi số CR/"đã gỡ" trong code), chạy test của service đó.

1. **rendering**
   - Xoá `ConceptFlowScene.clip` (`conceptflow/scene.py:334-343`) và `clip()` + trạng thái clip đang mở trong `conceptflow/narration.py` (từ `:188`; gỡ cả kiểm "không cho lồng" và bản ghi `kind: "clip"`).
   - `manim_renderer.py`: bỏ `clip_marks=` ở `:268-271`, `:324`, `:337`; xoá `_read_clip_marks`.
   - `domain/models.py`: bỏ hai trường `clip_marks` (`:108-114`, `:141-144`).
   - `producer.py`: bỏ tham số/trường `clip_marks` ở `script_validated_envelope` và `rendering_completed_envelope`; `consumer.py:133, 209` bỏ đối số.
   - Test: xoá `test_read_clip_marks_*` (`test_manim_renderer.py:393-424`), các test clip trong `conceptflow/test_narration.py` (`:108-162`, assert `:246`), `tests/adapters/test_producer.py` nếu chỉ thử `clip_marks` (đổi tên/viết lại nếu còn assert khác). Chạy `pytest` + `ruff`.
2. **video-assembly**
   - Xoá `adapters/clips/` (cả `__init__.py`), `domain/clip_rules.py`, `tests/adapters/test_generate_clips.py`, `tests/domain/test_clip_rules.py`.
   - `consumer.py`: xoá import `:37, 43, 63`, `GenerateClipsCommandHandler` (`:790-900`), tham số `generate_clips` của dispatcher (`:915-929`).
   - `producer.py`: xoá `clips_generated_envelope`; bỏ `intro_duration_seconds` khỏi `video_assembled_envelope` và lời gọi ở `consumer.py:267` (giữ biến `intro_duration_seconds` dùng cho `effective_lead_in`).
   - `progress.py`: xoá `GENERATE_CLIPS_STEP`, `publish_clip_progress`, sửa docstring module.
   - `artifact_paths.py`: xoá `clip_output_path`. `main.py`: xoá handler + `ClipThresholds`; sửa chú thích purge `:94` (không nhắc clips).
   - `tests/adapters/test_purge.py:88-95`: đổi thư mục con giả từ `clips` sang một thư mục con trung tính (purge xoá cả thư mục project, ý của test giữ nguyên). Sửa test producer/consumer đang assert `intro_duration_seconds` nếu có. Chạy `pytest` + `ruff`.
3. **orchestrator**
   - `domain/flow.go`: xoá `FlowSplit`; `FlowResult = 12`, `FlowPublish = 13`, `FlowStepsTotal = 13`; chú thích đầu file "13-step"; xoá 2 case clip trong `FlowStateFor`; bỏ `FlowSplit` khỏi `FlowStepLabel`.
   - `domain/project.go`: xoá `StatusGeneratingClips`, `StatusFailedGenerateClips` (và trong `IsInFlight`), `StepGenerateClips` (và case trong `FailedStatusForStep`), `ModeBoth`, `WantsClips`; `IsValid` chỉ `long`/`short`; sửa doc `VideoOutputMode`, chú thích trường `VideoOutputMode`; xoá trường `ClipMarks`, `IntroDurationSeconds`, `ClipRequests`, `Clips`, kiểu `ClipResult`; sửa chú thích "14-step" ở `ProjectSummary`.
   - Xoá `domain/clip_rules.go` + `clip_rules_test.go`.
   - `application/handle_step_event.go`: bỏ `clips_generated` khỏi map (`:91`), case `StepGenerateClips` (`:449-450`), gán `ClipMarks` (`:543`, `:878`), khối cảnh báo `:583-593`, gán `IntroDurationSeconds` (`:948`); `advanceAfterVideoReady` chỉ còn `UpdateStatus(..., StatusReadyToPublish)` — giữ tên hàm và hai lời gọi (onVideoAssembled, onQCCompleted) vì nó vẫn là điểm chung "video đã xong"; sửa chú thích `:950-959`, `:983-991`, `:1017-1023` (không nhắc generate_clips); xoá `onClipsGenerated`, `generateClipsPayload`.
   - `application/event_payload.go`: xoá `parseClipResults`, `buildClipRequests` (và helper chỉ chúng dùng).
   - `application/fork_project.go:154-161`: bỏ 4 dòng gán trường đã xoá.
   - `adapters/http/router.go`: xoá 2 route `:327-328` và `handleCreateClip`, `handleListClips`; `dto.go`: xoá `Clips` trong response, `createClipRequest`, `clipResultResponse`, `createClipResponse`, `toClipResultResponses`, dòng `Clips:` `:462`; chú thích `video_output_mode` (`:39`, `:159`) thành `"long" | "short"`; "14-step" → "13-step" (`:116`, `:188`).
   - `adapters/postgres/project_repository.go`: bỏ 4 cột khỏi SELECT (`:34`), Scan (`:63`), INSERT/UPSERT (`:398`, `:449` và mệnh đề `ON CONFLICT ... SET` tương ứng) cùng biến JSON trung gian.
   - `adapters/postgres/db.go`: bỏ các `ADD COLUMN` + chú thích của 4 cột (`:176-195`); chú thích `video_output_mode` thành `"long" | "short"`; chú thích `project_events` "13-step"; thêm cuối schema ba khối `DO $$` có guard `schema_migrations`:
     - `remove_clip_step_flow_numbers`: `UPDATE project_events SET flow_step = CASE flow_step WHEN 12 THEN 11 WHEN 13 THEN 12 WHEN 14 THEN 13 END WHERE flow_step IN (12,13,14)`; tương tự cho `from_flow_step`.
     - `remove_output_mode_both`: `UPDATE projects SET video_output_mode = 'long' WHERE video_output_mode = 'both'`.
     - (C1) `ALTER TABLE projects DROP COLUMN IF EXISTS clip_marks, DROP COLUMN IF EXISTS clip_requests, DROP COLUMN IF EXISTS clips, DROP COLUMN IF EXISTS intro_duration_seconds` (idempotent, không cần guard).
   - Test: sửa `domain/flow_test.go` (số mới, bỏ case clip), `project_inflight_test.go`, `handle_step_event_test.go` (xoá test clip; giữ/viết test "video_assembled → ready_to_publish" cho cả `long` và `short`), `router_test.go`; thêm test `IsValid` từ chối `"both"`; thêm test `FlowStateFor(StatusReadyToPublish) == {12, idle}`, `StatusPublished == {13, done}`. Chạy `go test ./...`, `go vet`, `gofmt`.
4. **authoring-service**
   - `domain/flow.go`, `domain/project.go`: cùng thay đổi như orchestrator bước 3 (hằng, trạng thái, bước, `ModeBoth`, `WantsClips`, chú thích); `flow_test.go:50` bỏ `StepGenerateClips`.
   - Xoá `application/suggest_short_script.go` + test; `llm_provider.go:15` sửa chú thích; `adapters/llm/llm_service_client.go` xoá `SuggestShortScript` (+ test), sửa chú thích port `:25`, `:339`; `adapters/http/router.go` xoá trường, route `:68`, interface `:222-228`, `WithShortScriptSuggester`, `handleSuggestShortScript`; `dto.go:200-216` xoá 2 DTO; `router_test.go` bỏ test route này; `cmd/authoring/main.go:70, 127` bỏ wiring.
   - Prompt: xoá `RoleShortScript` (`prompt_template.go:35, 44`), `shortScriptTemplate` (`prompt_vars.go:78-79`), seed `:30`, file `prompts/short_script.txt`, 3 mục `short_script` trong `testdata/prompt_golden.json`.
   - `adapters/postgres/prompt_repository.go` `PurgeLegacyPrompts`: thêm `DELETE FROM prompts WHERE role = 'short_script'`, cộng số hàng vào giá trị trả về; cập nhật doc comment ("vai trò không còn tồn tại"). Thêm test nếu repository có test với DB; nếu không, kiểm trực tiếp (bước Kiểm tra).
   - Chạy `go test ./...`, `go vet`, `gofmt`.
5. **llm-service**: xoá `ShortScriptBody`, route `suggest_short_script` (`main.py`), `build_short_script_prompt`, `suggest_short_script` (`tasks.py`) và helper chỉ chúng dùng; xoá 2 test ở `test_tasks.py`. Chạy `pytest` + `ruff`.
6. **api-gateway**: xoá `src/handlers/clipHandler.js`; `routes/projects.js` bỏ import, 3 route, 2 dòng chú thích; bỏ tham số `sharedDir` nếu chỉ clipHandler dùng (kiểm nơi gọi); `tests/routes/projects.test.js:68-162` xoá các test clip. Chạy `npm test` + lint.
7. **web-gui**
   - `utils/flow.ts`: bỏ "Cắt short" khỏi `FLOW_LABELS` và mục tương ứng của `FLOW_STEP_PURPOSE`; mục đích bước Kết quả thành "Xem lại video đã xong hoặc dựng lại với cấu hình khác."; `FLOW_RESULT = 12`, `FLOW_PUBLISH = 13`; `FLOW_PHASES` Sản xuất `[9,10,11]`, Hoàn tất `[12,13]`; `AUTO_STEPS` bỏ `FLOW_TTS + 3`; `stepStatus` bỏ dòng `step === 12` và tham số `outputMode` nếu không còn dùng (cập nhật nơi gọi); `skippedReason` chỉ còn lý do Hình minh hoạ (bỏ tham số `outputMode` nếu thừa); chú thích "14-step" → "13-step"; chú thích `flowRoute` bỏ "/split".
   - `pages/RenderPage.tsx`: bỏ `generate_clips` khỏi `SAGA_FLOW_STEP`, khỏi `PROCESS_STEPS` (nơi khai báo), bỏ lọc `longOnly`; chú thích "9–12" → "9–11".
   - `pages/ResultPage.tsx`: bỏ `ClipsPanel`, `wantsClips`; subtitle "Xem lại video hoặc dựng lại. Đăng video ở bước tiếp theo."
   - `pages/PublishPage.tsx` (B1): bỏ `ClipsPanel`, `wantsClips`, `clipsPanel`, state `youtubePublishOverride` và nhánh nút mở form cho `short`; form YouTube luôn hiện. Bố cục giữ đúng `docs/ux-ui-design-rules.md` mục 1 (xem trước bên trái, dữ liệu → nút Đăng bên phải/dưới), không đổi thứ tự.
   - `components/CompanionProjectCard.tsx`: bỏ `ClipsPanel`.
   - `components/ProgressTracker.tsx`: bỏ đếm "Clip" và `clipIndex/clipTotal` ở hook/state cấp dữ liệu (lần theo chỗ tạo hai giá trị này); chú thích `:40`.
   - Xoá `components/ClipsPanel.tsx`, `components/ShortScriptAssistant.tsx`, `tests/components/ClipsPanel.test.tsx`, `tests/components/ShortScriptAssistant.test.tsx`; xoá class CSS chỉ hai component này dùng (kiểm bằng grep tên class).
   - `api/client.ts`: xoá `getProjectClipUrl`, `SuggestShortScriptInput`, `suggestShortScript`, `"short_script"` trong union role. `pages/PromptSettingsPage.tsx:31` bỏ mục.
   - `types/index.ts`: `video_output_mode?: "long" | "short"` (2 chỗ), xoá `clips`, `Clip`, `clip_index`, `clip_total`; "flow 14 bước" → "13 bước". `context/ProjectDraftContext.tsx:116-123` `VideoOutputMode = "long" | "short"`, sửa doc. `utils/pipelineLabels.ts`: xoá nhãn `generate_clips`, `generating_clips`, mục trong map trạng thái→bước, `"generate_clips"` trong danh sách bước sản xuất; chú thích "9–12" → "9–11".
   - Chú thích nhắc `ShortScriptAssistant` ở `Disclosure.tsx:19`, `glass.module.css:218`: viết lại không nhắc component đã xoá. Các chú thích "14 bước" ở `AppShell.tsx:18, 24`, `StepRail.tsx:72`, `useStepNav.ts:36`, `AuthoringModeBar.tsx:30` → "13 bước".
   - `tests/components/VideoOutputModePicker.test.tsx:27`: giữ nếu vẫn có nghĩa (ghi chú không nhắc `self.clip`), không thì bỏ assert.
   - Sửa các test web-gui đang dùng số bước 12/13/14 hoặc `"both"` (grep `tests/` theo `FLOW_RESULT|FLOW_PUBLISH|\b1[234]\b.*step|"both"`), thêm test `stepStatus`/`FLOW_PHASES` cho 13 bước. Chạy `vitest`, `tsc --noEmit`, `eslint`, `prettier --check` trên file đã đụng.
8. **Tài liệu**: ADR-0031 mục Hệ quả `:32` thêm "Đã gỡ ở CR-061; luồng còn 13 bước."; `docs/contracts/shared-artifacts.md:19` xoá dòng clip; `README.md:59` bỏ `generate_clips`.
9. **Rà sót**: chạy grep ở Tiêu chí chấp nhận trên `services/`; mọi kết quả còn lại phải được giải thích trong báo cáo.

## Kiểm tra

- Test: rendering, video-assembly, llm-service (`pytest`, `ruff`); orchestrator, authoring-service (`go test ./...`, `go vet`, `gofmt -l`); api-gateway (`npm test`); web-gui (`vitest`, `tsc --noEmit`, `eslint`).
- Rebuild + restart: `orchestrator`, `authoring-service`, `llm-service`, `rendering`, `video-assembly`, `api-gateway`, `web-gui`; tất cả healthy.
- Kiểm trực tiếp:
  - `orchestrator-db`: `schema_migrations` có 2 id mới; `project_events` không còn hàng `flow_step` 14, 9 hàng cũ ở 13 nay ở 12; `\d projects` không còn 4 cột.
  - `authoring-service-db`: không còn hàng `prompts.role = 'short_script'`.
  - `GET /v1/projects` và một project cũ đã xong: `flow_step = 12` (ready_to_publish) / 13 (published); màn Nhật ký hiện "Kết quả".
  - `POST` start với `video_output_mode: "both"` → 400.
  - `GET /v1/projects/{id}/clips` qua gateway → 404.
  - Web: thanh bước 13 ô, không có "Cắt short"; màn Dựng video không có dòng cắt clip; màn Kết quả và Đăng video không có panel clip; Đăng video của một project short hiện form YouTube.
  - Không chạy AI/render thật (tốn token); luồng `video_assembled → ready_to_publish` được phủ bằng test.

## Rủi ro

- **Đổi số bước**: URL/bookmark cũ `?step=13|14` lệch một bước; hằng số cứng (không qua `FLOW_*`) bị sót trong web-gui → lệch nhãn. Giảm bằng grep số trong `src/` và test `FLOW_PHASES`/`stepStatus`.
- **Rollback (C1)**: image cũ cần 4 cột đã drop → phải `ADD COLUMN` lại khi quay về. Dữ liệu mất = 0 (đã kiểm rỗng).
- **Migration số bước** chạy một lần nhờ guard; nếu image cũ (14 bước) lại chạy sau migration, nó sẽ ghi sự kiện mới theo số cũ — chỉ ảnh hưởng nhãn Nhật ký.
- **Script Manim tự viết** còn `with self.clip(...)` sẽ lỗi `AttributeError` khi validate; hiện 0 script như vậy, prompt Kỹ sư không sinh lời gọi này.
- **Hợp đồng message**: bỏ trường là thay đổi không tương thích ngược về phía người nhận cũ; tất cả người nhận được build lại cùng lúc trong CR này.
