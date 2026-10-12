# CR-071 — Bỏ engine Manim: danh sách task

Thiết kế đã duyệt: `cr-071-remove-manim-design.md` (commit `4d86ad1`). Mỗi task làm và test được riêng; làm theo thứ tự. Comment trong code theo `docs/code-standards-rules.md` §1: nói code làm gì, không nhắc CR, không "trước đây/không còn".

Quy ước dùng trong các task:

- **Project cũ** = project có `render_engine = 'manim'` trong DB orchestrator.
- **Thông điệp ngừng hỗ trợ** (một chuỗi, dùng y hệt ở orchestrator, authoring-service, web-gui): `Dự án này dựng bằng Manim, đã ngừng hỗ trợ: chỉ xem, đăng hoặc xoá được.`
- Python: chạy test bằng môi trường của service (`.venv` của service nếu có, hoặc trong container của service); `ruff check .` sạch.

---

### T0 — Đếm dữ liệu trước khi đổi
- [ ] Xong

- **Service**: orchestrator, video-assembly (DB), không đổi code.
- **Thay đổi**: Bật Docker nếu đang tắt (`docker compose up -d` ở bản checkout chính). Chạy và ghi kết quả vào audit:
  - `SELECT render_engine, status, count(*) FROM projects GROUP BY 1,2 ORDER BY 1,2;` (DB orchestrator)
  - `SELECT kind, render_quality, asset_id, version FROM channel_asset_pointers;` (DB orchestrator)
  - Có project cũ nào ở trạng thái đang chạy (`validating_script`, `synthesizing_speech`, `rendering`, `assembling_video`, `publishing`…) không.
- **Test**: —
- **Xong khi**: số liệu ghi trong audit. **Nếu có project cũ đang chạy: dừng, báo Creator trước khi làm tiếp** (thiết kế, Rủi ro 1).
- **Phụ thuộc**: —

---

## orchestrator

### T1 — Engine: Remotion là engine duy nhất, `manim` là dấu project cũ
- [ ] Xong

- **Service**: orchestrator
- **File**: `internal/domain/project.go:190-216` (`RenderEngine`, hằng, `DefaultRenderEngine`, `IsValid`), `:337-340` (doc trường `RenderEngine`); `internal/domain/errors.go` (lỗi mới).
- **Thay đổi**:
  - `RenderEngineManim` → `RenderEngineRetiredManim RenderEngine = "manim"`; doc: giá trị chỉ còn trên project dựng bằng Manim, không chạy được nữa (`See ADR-0033.`).
  - `DefaultRenderEngine = RenderEngineRemotion`.
  - `IsValid()` (kiểm input): chỉ `RenderEngineRemotion`.
  - Thêm `IsKnown()`: `remotion` hoặc `manim` (dùng khi đọc/ghi DB).
  - Thêm `func (p *Project) EngineRetired() bool { return p.RenderEngine == RenderEngineRetiredManim }`.
  - `errors.go`: `ErrEngineRetired = errors.New("Dự án này dựng bằng Manim, đã ngừng hỗ trợ: chỉ xem, đăng hoặc xoá được.")`.
  - Doc kiểu `RenderEngine` và trường `Project.RenderEngine` viết lại theo một engine.
- **Test**: `internal/domain/project_test.go` (hoặc file test domain đang có cho engine): `IsValid` chỉ nhận `remotion`; `IsKnown` nhận cả hai; `EngineRetired` đúng/sai. `go test ./internal/domain/...`
- **Xong khi**: FR1, FR2 (nền).
- **Phụ thuộc**: T0

### T2 — Migration và repository
- [ ] Xong

- **Service**: orchestrator
- **File**: `internal/adapters/postgres/db.go` (`CREATE TABLE projects` dòng 23; dòng 53; dòng 170-174 `layout_marks`; dòng 186-189 `render_engine`; cuối chuỗi `schema`), `internal/adapters/postgres/project_repository.go` (`Get` dòng 28, 33, 54, 70-74, 106; `List` dòng 179; `Save` dòng 289-292, 340-345, 354-400; `SaveRenderEngine` dòng 520), `internal/domain/project.go:291` (`ManimSceneClassName`), `:395` (`LayoutMarks`).
- **Thay đổi**:
  - `CREATE TABLE`: `manim_scene_class_name` → `scene_class_name`. Bỏ dòng `ADD COLUMN IF NOT EXISTS manim_scene_class_name` (53) và `ADD COLUMN IF NOT EXISTS layout_marks` (174) cùng comment của chúng.
  - Dòng 189 giữ `ADD COLUMN IF NOT EXISTS render_engine TEXT NOT NULL DEFAULT 'remotion'`; comment: engine dựng video, `'manim'` chỉ còn ở project cũ.
  - Cuối `schema`, thêm:
    ```sql
    -- Project dựng bằng Manim giữ giá trị 'manim'; project mới mặc định Remotion.
    ALTER TABLE projects ALTER COLUMN render_engine SET DEFAULT 'remotion';

    -- Tên trung lập cho id class/composition mà rendering chạy.
    DO $$
    BEGIN
        IF EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'projects' AND column_name = 'manim_scene_class_name')
           AND NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'projects' AND column_name = 'scene_class_name') THEN
            ALTER TABLE projects RENAME COLUMN manim_scene_class_name TO scene_class_name;
        END IF;
    END $$;

    ALTER TABLE projects DROP COLUMN IF EXISTS layout_marks;
    ```
  - `Project.ManimSceneClassName` → `SceneClassName` (doc: id class Manim hoặc composition Remotion mà rendering chạy; với project mới là composition). Bỏ trường `LayoutMarks`.
  - Repository: mọi SQL `manim_scene_class_name` → `scene_class_name`; bỏ `layout_marks` khỏi SELECT/INSERT/UPDATE và biến `layoutMarksJSON`. Đọc/ghi engine dùng `IsKnown()` thay `IsValid()` (giữ `manim` của project cũ; rỗng/lạ → `DefaultRenderEngine`).
  - Xoá `SaveRenderEngine` (repository và port `ProjectDraftPort` ở `internal/application/create_project_draft.go:25-31`): engine không còn chọn được.
- **Test**: test repository/adapters đang có (`internal/adapters/postgres/*_test.go` nếu chạy được với DB test; nếu không, test đọc chuỗi `schema`) cập nhật tên cột; thêm ca: `schema` chứa khối rename và `DROP COLUMN IF EXISTS layout_marks`. Sửa mọi chỗ biên dịch hỏng do đổi tên (`go build ./...`). `go test ./... && go vet ./... && gofmt -l .`
- **Xong khi**: FR4 (cột), FR5 (bỏ `layout_marks`); schema chạy lại nhiều lần không lỗi.
- **Phụ thuộc**: T1

### T3 — Chặn thao tác chạy lại trên project cũ
- [ ] Xong

- **Service**: orchestrator
- **File** và chỗ chèn kiểm tra (ngay sau khi nạp project):
  - `internal/application/start_render_saga.go:88` `Execute`: trước khi tạo `project`, `existing, err := uc.repo.Get(ctx, input.ProjectID)`; `ErrProjectNotFound` → tiếp tục (project mới); lỗi khác → trả; `existing.EngineRetired()` → `ErrEngineRetired`. Bỏ trường `RenderEngine` khỏi `StartRenderSagaInput`; project mới luôn `domain.RenderEngineRemotion`.
  - `internal/application/retry_step.go:72` `Execute`: sau `uc.repo.Get`, nếu `project.EngineRetired()` và bước lỗi không phải `domain.StepPublishVideo` → `ErrEngineRetired` (đăng lại video đã xong vẫn được).
  - `internal/application/fork_project.go:77` sau `uc.repo.Get(ctx, sourceID)`: `src.EngineRetired()` → `ErrEngineRetired`.
  - `internal/application/review_outline.go:54` `Approve`, `:74` `EditNarration`, `:136` `Reject`: sau `uc.repo.Get`, `project.EngineRetired()` → `domain.ErrEngineRetired`.
  - `internal/application/wizard.go:33` `PatchWizardSettingsUseCase.Execute`: nạp project (đã nạp ở dòng ~84) trước mọi ghi; `EngineRetired()` → `ErrEngineRetired`. Bỏ `RenderEngine` khỏi `domain.WizardSettingsPatch` và nhánh kiểm dòng 37-39, 94-99.
  - `internal/application/create_project_draft.go:80` `CreateProjectDraftUseCase.Execute`: khi `projectExists`, `uc.repo.Get` → `EngineRetired()` → `ErrEngineRetired`. Bỏ `CreateProjectDraftInput.RenderEngine` và khối dòng 127-134; project mới `RenderEngine: domain.RenderEngineRemotion`.
  - `create_project_draft.go:197` `UpdateProjectTopicUseCase.Execute`: thêm `uc.repo.Get` (port đã có `Get`) → `EngineRetired()` → `ErrEngineRetired`.
  - `internal/adapters/http/router.go:787` `writeUseCaseError`: thêm `case errors.Is(err, domain.ErrEngineRetired): writeError(w, http.StatusConflict, err.Error())`. `handleReviewDecision` (dòng 896) và `handleEditNarration` (dòng 934): thêm case `ErrEngineRetired` → 409 trước `default`.
  - `router.go:449-451`, `:379`, `:457`, `:966` và `adapters/http/dto.go:34,240,282`: bỏ `render_engine` khỏi body request (giữ ở response `dto.go:136,179,335,390`).
  - `router.go:1008` `handleInternalProjectStatus`: trả thêm `"render_engine"` (cho T7).
- **Không chặn**: `DELETE /v1/projects/{id}`, `POST /v1/sagas/publish`, `POST /v1/projects/{id}/cancel`, mọi `GET`.
- **Test**: thêm ca trong test của từng use case (`start_render_saga_test.go`, `retry_step_test.go`, `fork_project_test.go`, `review_outline_test.go`, `wizard_test.go`, `create_project_draft_test.go`): project cũ → `ErrEngineRetired`, không có lệnh nào được publish, không ghi gì; `retry` của project cũ lỗi ở `publish_video` → chạy. `router_test.go`: `ErrEngineRetired` → 409 với đúng thông điệp; `/internal/v1/projects/{id}/status` có `render_engine`. `go test ./...`
- **Xong khi**: FR2 phía server, tiêu chí 2 (409).
- **Phụ thuộc**: T1, T2

### T4 — Gỡ nhánh Manim, `layout_marks` và intro/outro dựng bằng Manim
- [ ] Xong

- **Service**: orchestrator
- **File**:
  - `internal/application/handle_step_event.go`: `onScriptParsed` (492-515) và `onScriptValidated` (522-528) dùng `SceneClassName`; dispatch render (776-796): bỏ `if engine == domain.RenderEngineRemotion`, luôn gửi `video_font`; `engine` gửi đúng `project.RenderEngine` (project cũ còn chạy dở sẽ bị rendering từ chối, T10). Dòng 228-235: chỉ `channel_asset_normalized` đi vào `handleChannelAssetProjection`; trong hàm đó (300-316) bỏ nhánh `channel_asset_rendered`, xoá `registerRenderedChannelAsset` (1198-1222), sửa doc. Dòng 848 (gán `LayoutMarks`) xoá; lệnh `qc_video` (985-1003) bỏ `layout_marks`.
  - `internal/application/retry_step.go:136-165`, `review_outline.go:125`: `SceneClassName`; bỏ thay engine mặc định khi `!IsValid()` (dùng `project.RenderEngine`).
  - `internal/application/fork_project.go:147,154`: `SceneClassName = ""`, bỏ dòng `LayoutMarks`.
  - `internal/application/authoring_port.go:25-28`: `NeedsIllustrations: engine != domain.RenderEngineRetiredManim`.
  - `internal/application/event_payload.go:184`: sửa comment nhắc `layout_marks`.
  - `internal/adapters/amqp/consumer.go:323-326`: comment chỉ còn `channel_asset_normalized`.
  - `internal/domain/narration.go:189`, `start_render_saga.go:41`: comment không nhắc Manim.
- **Test**: `handle_step_event_test.go`: bỏ ca `channel_asset_rendered` → `register_channel_asset`; thêm ca `channel_asset_rendered` bị bỏ qua (không publish gì); dispatch `render_scenes` luôn có `video_font`; `qc_video` không có `layout_marks`. `go test ./... && go vet ./... && gofmt -l .`; `grep -rni manim internal/` chỉ còn `RenderEngineRetiredManim`, migration, thông điệp lỗi và test.
- **Xong khi**: FR3 (orchestrator), FR5 (orchestrator).
- **Phụ thuộc**: T2, T3

---

## authoring-service

### T5 — Engine và chặn project cũ
- [ ] Xong

- **Service**: authoring-service
- **File**: `internal/domain/project.go:203-228`, `internal/domain/errors.go`, `internal/adapters/http/router.go:119` (`writeUseCaseError`), `:1052` (`writeGenerateError`), `:1142` (`writeSegmentError`), `internal/adapters/http/dto.go:33-34,177-179,221,260-262`.
- **Thay đổi**:
  - Như T1 cho bản sao domain: `RenderEngineRetiredManim`, `DefaultRenderEngine = RenderEngineRemotion`, `IsValid` chỉ `remotion`, `IsKnown`, `(*Project).EngineRetired()`, `ErrEngineRetired` (cùng thông điệp).
  - Ba hàm ghi lỗi: thêm case `ErrEngineRetired` → 409 với `err.Error()`.
  - `dto.go`: bỏ `render_engine` khỏi request; comment response viết theo một engine.
- **Test**: domain test như T1; router test: mỗi hàm ghi lỗi đổi `ErrEngineRetired` thành 409. `go test ./...`
- **Xong khi**: FR2 (nền authoring).
- **Phụ thuộc**: T3 (orchestrator trả `render_engine` ở endpoint status)

### T6 — Cổng sửa và sinh nội dung chặn project cũ
- [ ] Xong

- **Service**: authoring-service
- **File**: `internal/application/prompt_templates.go:21-23` (`AuthoringLockPort`), `:41-58` (`checkAuthoringUnlocked`); `internal/adapters/orchestrator/client.go:80-89`; `internal/application/generate_authoring.go:400`, `generate_authoring_segments.go:39,107,160`, `render_prompt.go:158`.
- **Thay đổi**:
  - `AuthoringLockPort`: thêm `GetLock(ctx, projectID) (domain.ProjectLock, error)`; `domain.ProjectLock{Status ProjectStatus; RenderEngine RenderEngine}` (file mới `internal/domain/project_lock.go` hoặc trong `project.go`). `checkAuthoringUnlocked` dùng `GetLock`: `RenderEngine == RenderEngineRetiredManim` → `ErrEngineRetired` trước kiểm trạng thái.
  - `client.go`: `GetLock` gọi `/internal/v1/projects/{id}/status`, đọc `status` và `render_engine` (rỗng → `DefaultRenderEngine`). `GetStatus` giữ cho `illustrations.go:446`.
  - Năm chỗ nạp project bằng `uc.projects.Get`: thêm `if project.EngineRetired() { return …, domain.ErrEngineRetired }`.
  - Mọi fake của `AuthoringLockPort` trong test implement `GetLock`.
- **Test**: `prompt_templates_test.go`: lưu story/storyboard/code của project cũ → `ErrEngineRetired`; `generate_authoring_test.go`, `generate_authoring_segments_test.go`, `render_prompt_test.go`: project cũ → `ErrEngineRetired`, không gọi llm-service; `adapters/orchestrator/client_test.go`: `GetLock` đọc đủ hai trường. `go test ./...`
- **Xong khi**: FR2 phía authoring, tiêu chí 2.
- **Phụ thuộc**: T5

### T7 — Gỡ prompt và nhánh Manim
- [ ] Xong

- **Service**: authoring-service
- **File**:
  - `internal/domain/prompt_template.go:7,19,28,33,46-48`: xoá `RoleManimEngineer`, `RoleManimEngineerAI`, `RoleManimAdjust` khỏi hằng và `IsRenderablePromptRole`; sửa doc dòng 7.
  - `internal/domain/prompt_template_seeds.go`: xoá `manimIntroVI` → `manimOutputVI` (448-611) và hằng ghép `manimEngineerVI`; dòng 22, 28 trong `DefaultPromptTemplates`; `withThemeReference` nếu không còn ai dùng.
  - `internal/domain/prompt_template_seeds_ai.go`: xoá `manim*AIVI` (194-~260) và `manimEngineerAIVI`; dòng 25 ở `DefaultPromptTemplates`; comment dòng 8.
  - `internal/domain/prompt_vars.go:27-35,72-73,107-…`: xoá embed `prompts/theme_reference_{vi,en}.txt`, `prompts/manim_adjust.txt`, `ThemeReference` và biến `{{theme_reference}}`/`manimAdjustTemplate`. Xoá ba file `.txt` đó.
  - `internal/application/render_prompt.go:109-145`: `RoleFor(step string)` và `AIRoleFor(step string)` bỏ tham số engine, bước `code` → vai Remotion; `:272,275` bỏ vai Manim; `:431-436` xoá `engineOf`, chỗ gọi dòng 209 truyền `"remotion"`. Chỗ gọi: `generate_authoring.go:409`, `generate_authoring_segments.go:111`.
  - `internal/application/render_prompt_input.go:41,51,72-75,84-88`: xoá `manimScriptPlaceholder`, `engine` luôn `remotion`; `isRemotionRole` chỉ còn nếu nơi khác cần.
  - `internal/application/generate_authoring_code.go:139,167,474`, `storyboard_warnings.go:34`: bỏ điều kiện `RenderEngine == RenderEngineRemotion` (luôn đúng sau T6).
  - `internal/application/code_pipeline.go:51`: `Engine` luôn `"remotion"`; comment.
  - Comment nhắc Manim: `script_templates.go:9`, `code_segment.go:10`, `narration.go:246`, `generate_authoring.go:31`, `prompt_templates.go:144,150`, `adapters/http/router.go:722`, `adapters/postgres/prompt_template_repository.go:182-197`, `config/config.go:82`.
  - `internal/adapters/postgres/db.go`, cuối `schema`: `DELETE FROM prompts WHERE role IN ('manim_engineer', 'manim_engineer_ai', 'manim_adjust');` (comment: các vai này không còn trong pipeline).
- **Test**: golden prompt (`internal/domain/testdata/prompt_golden.json`, `prompt_golden_test.go`, `golden_prompts_test.go`): bỏ mục Manim, sinh lại golden theo cách test đang làm; `prompt_templates_test.go`, `render_prompt_test.go`: `RoleFor("code")`/`AIRoleFor("code")` → vai Remotion; `IsRenderablePromptRole("manim_engineer") == false`. `go test ./... && go vet ./... && gofmt -l .`; `grep -rni manim internal/` chỉ còn `RenderEngineRetiredManim`, thông điệp, `DELETE` migration và test.
- **Xong khi**: FR1 phía authoring, FR6 (prompt Manim xoá).
- **Phụ thuộc**: T6

---

## llm-service

### T8 — Chỉ nhận engine `remotion`, gỡ pipeline Manim
- [ ] Xong

- **Service**: llm-service
- **File**:
  - `app/pipeline/prompts.py`: xoá `manim_cast` (240-262, kể cả mẫu `setup_cast`), `manim_chunk` (263-305), `manim_repair_many` (306-346), `manim_repair` (347-hết).
  - `app/pipeline/merger.py`: xoá `MANIM_SHOT` (20), `manim_fn` (47-50), `manim_scene_class` (346-349), `merge_manim` (350-hết).
  - `app/pipeline/checker.py`: dòng 76 chỉ nhận `"remotion"`; xoá `_MANIM_SHOT_IN_TRACE` (102) và `shots_from_manim_trace` (106-…).
  - `app/pipeline/run.py`: bỏ import `shots_from_manim_trace` (57); `CodeRequest.engine` (107) giữ trường, comment "luôn remotion"; dòng 134, 401, 446-454, 458, 491, 506, 541, 611, 786, 918-937, 1011-1013 bỏ nhánh Manim (`_cast_names` 440, `parse_cast` 483 xoá nếu không còn ai gọi). Dòng 602-603: `if req.engine != "remotion": raise PipelineFailure(f"unknown engine {req.engine!r}", [])`. **Giữ `req.engine` trong các `_sha(...)` (dòng 407, 427)** để dấu vân tay của các đoạn đã lưu không đổi.
  - `app/main.py:320-321`: chỉ nhận `"remotion"` (400 khác).
- **Test**: `tests/test_pipeline.py`, `tests/test_api.py`, test merger/checker/prompts: xoá ca Manim; thêm ca `engine="manim"` → 400 ở `/v2/code/generate` và `PipelineFailure` ở pipeline; ca kiểm dấu vân tay của một plan Remotion giống trước khi sửa (so một giá trị cố định tính trên main). `pytest`, `ruff check .`
- **Xong khi**: FR1 phía llm-service, FR7 (hợp đồng `engine`).
- **Phụ thuộc**: —

---

## rendering

### T9 — Đổi tên port, tách `LintIssue`
- [ ] Xong

- **Service**: rendering
- **File**: `domain/ports.py:4,20-23,61-76`, `adapters/rendering/remotion_renderer.py:1-25,44,194`, `domain/script_lint.py`, file mới `domain/lint_issue.py`, `domain/lottie_catalog.py:244`, `application/check_script.py`, `validate_script.py:29-30`, `render_script.py:22-23`.
- **Thay đổi**:
  - `ManimScriptRendererPort` → `ScriptRendererPort` (doc: dựng và chạy thử một script Remotion). Mọi chỗ import đổi theo.
  - Chuyển `LintIssue` và `blocking_issues` từ `domain/script_lint.py` sang `domain/lint_issue.py` (docstring module). `lottie_catalog.py` và các use case import từ module mới.
  - Docstring `remotion_renderer.py` không so sánh với Manim.
- **Test**: `tests/domain/test_lottie_catalog.py` và test use case chạy qua sau đổi import. `pytest`
- **Xong khi**: FR4 (port).
- **Phụ thuộc**: —

### T10 — Gỡ Manim khỏi rendering
- [ ] Xong

- **Service**: rendering
- **File**:
  - Xoá: `adapters/rendering/manim_renderer.py`, `adapters/rendering/engine_router.py`, thư mục `conceptflow/` (gói Python Manim; **không đụng** `remotion_project/src/conceptflow-mini/`), `domain/script_lint.py`, `application/render_channel_asset.py`, `tools/gen_theme_reference.py`.
  - `domain/errors.py:15-29`: xoá `InvalidManimApiUsageError`; thêm `UnsupportedEngineError(ValueError)` ("engine {engine!r} không còn được hỗ trợ; chỉ có remotion"). Sửa docstring lỗi render dòng 15.
  - `domain/models.py`: docstring module; `engine: str = "remotion"` (49); bỏ comment Manim (45-63 phần nói về Manim, giữ phần NFC nếu vẫn đúng cho Remotion, viết lại không nhắc Manim).
  - `domain/script_locator.py`: chỉ tìm composition Remotion; `LocatedScene.engine` bỏ hoặc luôn `remotion`.
  - `domain/ports.py`: xoá `ChannelAssetRendererPort` (61-76).
  - `application/render_script.py:1-4,20-23,32,82-90`, `validate_script.py:7-15,29-30,40,48,63,88-95`, `check_script.py:8,46,86,101`: bỏ lint Manim và nhánh `engine == "manim"`; nhận `engine != "remotion"` → `UnsupportedEngineError`.
  - `adapters/messaging/consumer.py:2-4,101,147,191,230,269-280,301`: `payload.get("engine", "remotion")`; xoá handler `render_channel_asset` và đăng ký của nó; `UnsupportedEngineError` đi đường lỗi như mọi lỗi render (sự kiện `*_failed` mang thông điệp).
  - `adapters/messaging/producer.py:2,77`: mặc định `engine="remotion"`; xoá `channel_asset_rendered_envelope`, `channel_asset_render_failed_envelope`.
  - `adapters/messaging/progress.py:43`, `cancellation.py:53`: docstring không nhắc Manim.
  - `adapters/http/check_server.py:4,122-124`: xoá route `/v1/check/manim`.
  - `adapters/storage/artifact_paths.py:1,28,104`: xoá `compute_channel_asset_path` và đường cache Manim; purge chỉ xoá `video/rendered.mp4`, `video/timing.json` của project.
  - `main.py:38-51,85-105,120-123`: bỏ `ManimScriptRenderer`, `EngineRouterRenderer`, `RenderChannelAssetUseCase`, các biến `DRY_RUN_TIMEOUT_SECONDS`, `RENDER_MEMORY_LIMIT_GB`, `RENDER_CACHE_ROOT`, `RENDER_QUALITY`; `renderer = RemotionScriptRenderer(timeout_seconds=timeout_seconds)`; comment `prefetch_count=1` viết lại theo một lượt render Remotion mỗi lần.
  - `requirements.txt:3`: xoá `manim==0.18.*`.
  - `Dockerfile:1-20`: khối đầu chỉ còn `ffmpeg`; bỏ `libcairo2-dev`, `libpango1.0-dev`, `pkg-config`, `build-essential`, texlive, `tipa`, `dvisvgm` và comment texlive. Comment khối font (21-30) viết theo Chromium/fontconfig. Comment khối Node (49-55) bỏ chuyện giữ hai engine.
- **Test**:
  - Xoá: `tests/adapters/test_manim_renderer.py`, `test_engine_router.py`, `tests/conceptflow/` (cả thư mục), `tests/domain/test_script_lint.py`, `tests/fixtures/conceptflow_template.py` (nếu không còn ai dùng).
  - Sửa: `test_check_server.py` (route Manim → 404), `test_messaging.py` (bỏ `render_channel_asset`; lệnh `engine="manim"` → sự kiện thất bại có thông điệp `UnsupportedEngineError`), `test_purge.py` (không còn cache Manim), `test_remotion_renderer.py`, `tests/application/test_*.py`, `tests/domain/test_models.py`, `test_script_locator.py`, `tests/conftest.py`.
  - `pytest`, `ruff check .`; `grep -rni manim --exclude-dir=node_modules .` chỉ còn `UnsupportedEngineError`/test của nó.
- **Xong khi**: FR1, FR3 (rendering), FR6 (image), tiêu chí 4.
- **Phụ thuộc**: T9

---

## video-assembly

### T11 — Gỡ `register_channel_asset` và luật QC bố cục
- [ ] Xong

- **Service**: video-assembly
- **File**:
  - `adapters/messaging/consumer.py:275-363`: xoá `RegisterChannelAssetCommandHandler`; `:801-806` bỏ tham số và đăng ký `register_channel_asset`. `:690` docstring; `:749-772` `_score`: bỏ đọc `layout_marks` và nhánh trả `not_scored` khi thiếu; gọi `evaluate_all` không có `layout_marks`.
  - `main.py:23,83-86`: bỏ import, khởi tạo và truyền handler.
  - `domain/qc_rules.py`: xoá khung Manim (20-41: `FRAME_WIDTH`, `FRAME_HEIGHT`, `THEME_SAFE_MARGIN`, `THEME_BACKGROUND`, `TEXT_CLASSES`), tiện ích bố cục (186-276), `check_frame_overflow`, `_overflow_of`, `_area`, `_intersection_area`, `check_text_overlap`, `font_size_to_pixels`, `check_text_too_small`, `_too_small_in_mark`, `_parse_hex_color`, `_relative_luminance`, `contrast_ratio`, `check_low_contrast` (278-559). `QCThresholds` bỏ `safe_margin`, `frame_overflow_tolerance`, `text_overlap_min_ratio`, `min_text_pixel_height`, `min_contrast_ratio`, `background_color` và env tương ứng. `BLOCKING_RULES = frozenset({"narration_overlap"})`. `evaluate_all` bỏ tham số `layout_marks` và bốn lời gọi. Giữ `QUALITY_FRAME_SIZE`/`QUALITY_FRAME_RATE` nếu `check_publish_attributes` còn dùng.
  - `adapters/storage/artifact_paths.py:32`: docstring không nhắc bản Manim.
  - `domain/models.py:3`: docstring module.
  - `docker-compose.yml:232-236`: xoá các dòng comment `QC_MIN_TEXT_PIXEL_HEIGHT`, `QC_MIN_CONTRAST_RATIO`, `QC_TEXT_OVERLAP_MIN_RATIO`, `QC_SAFE_MARGIN` (và các biến QC bố cục khác nếu có quanh đó).
- **Test**: test QC (`tests/domain/test_qc_rules.py` hoặc tương đương): xoá ca bốn luật bố cục; ca `evaluate_all` không còn tham số `layout_marks`; ca `_score` chấm được khi không có `layout_marks`. Test messaging: bỏ ca `register_channel_asset`; lệnh `register_channel_asset` tới thì bị bỏ qua như lệnh lạ (theo cách dispatcher đang xử lý event_type lạ). `pytest`, `ruff check .`
- **Xong khi**: FR3 (video-assembly), FR5.
- **Phụ thuộc**: —

---

## api-gateway, tts

### T12 — Chú thích
- [ ] Xong

- **Service**: api-gateway, tts
- **File**: `services/api-gateway/src/server.js:60`, `src/routes/projects.js:85`; `services/tts/adapters/messaging/cancellation.py` (dòng nhắc Manim).
- **Thay đổi**: comment nói về code của bước Code (Remotion), không nhắc Manim.
- **Test**: `npm test` (api-gateway), `pytest` (tts).
- **Xong khi**: tiêu chí 5.
- **Phụ thuộc**: —

---

## web-gui

### T13 — Gỡ chọn engine
- [ ] Xong

- **Service**: web-gui
- **File**:
  - Xoá `src/components/RenderEnginePicker.tsx` (và test của nó). `src/components/VideoFontPicker.tsx:4`: chuyển các class đang dùng từ `RenderEnginePicker.module.css` sang `VideoFontPicker.module.css` mới (chép đúng các class được dùng), rồi xoá `RenderEnginePicker.module.css`.
  - `src/components/PipelineSettingsBar.tsx:3,14,45,64,92`: bỏ prop `renderEngine`/`onEngineChange`, nhãn engine và picker.
  - `src/pages/ScriptAuthoringSettingsStepPage.tsx:7,77` và chỗ render picker: bỏ; mọi nhánh `isRemotion` thành nhánh Remotion.
  - `src/context/ProjectDraftContext.tsx:53,110,167,251,291,341`: bỏ `renderEngine` khỏi `ProjectDraft`, mặc định, action và payload gửi server.
  - `src/api/client.ts:1268,1282,1297,1316`: `createProjectDraft` bỏ tham số `renderEngine`; bỏ `renderEngine` khỏi kiểu cài đặt wizard và bảng tên trường.
  - `src/hooks/useAuthoringMode.ts:74`, `src/pages/ScriptStepPage.tsx:50`: bỏ đối số engine.
  - `src/utils/flow.ts:11-14,153-154`: bước 5 luôn dùng, trừ project cũ (`stepStatus(..., renderEngine)` trả `"skipped"` khi `renderEngine === "manim"`); comment theo đó.
  - `src/components/AuthoringModeBar.tsx:51-52`: `authoringChainSteps()` luôn `["story", "storyboard", "illustrations", "code"]` (bỏ tham số; cập nhật chỗ gọi).
  - `src/pages/VisualDirectorStepPage.tsx:102,126,142`, `IllustrationsStepPage.tsx:21,53,74`, `ProductionSettingsPanel.tsx:91`, `hooks/useStepNav.ts:49,59`: bỏ nhánh Manim.
  - `src/utils/scriptValidation.ts`, `src/utils/authoringHint.ts`, `src/hooks/useLlmStatus.ts`, `src/components/ProgressTracker.tsx`: bỏ nhánh/chuỗi Manim.
  - `src/pages/PromptSettingsPage.tsx:20,26,29`: xoá ba vai Manim.
  - `src/types/index.ts:12-13,113-114,201`: `render_engine?: "remotion" | "manim"`, doc: `"manim"` chỉ có ở project cũ, không chạy lại được.
- **Test**: cập nhật test của các component/page trên (vitest); xoá ca chọn engine. `npm test`, `npx tsc -b`, `npm run lint`.
- **Xong khi**: FR1 phía giao diện, tiêu chí 1.
- **Phụ thuộc**: T3 (API không còn nhận `render_engine`)

### T14 — Đổi tên `CodeStepPage`
- [ ] Xong

- **Service**: web-gui
- **File**: `src/pages/ManimEngineerStepPage.tsx` → `src/pages/CodeStepPage.tsx` (và CSS module/test đi kèm nếu có cùng tên); `src/App.tsx:20,67`; `src/hooks/useLlmStatus.ts:5` (comment).
- **Thay đổi**: export `CodeStepPage`; `data-testid` `manim-engineer-step-page` → `code-step-page`, `manim-engineer-prompt` → `code-step-prompt`, `manim-engineer-copy` → `code-step-copy`; dòng 29-37: vai luôn `remotion_engineer`, bỏ biến `isRemotion`; dòng 127 `startRenderSaga` không gửi `render_engine`.
- **Test**: đổi tên file test và các selector theo `data-testid` mới. `npm test`, `npx tsc -b`, `npm run lint`.
- **Xong khi**: FR4 (màn).
- **Phụ thuộc**: T13

### T15 — Project cũ: nhãn, thông báo, ẩn thao tác chạy lại
- [ ] Xong

- **Service**: web-gui
- **File**: file mới `src/utils/retiredEngine.ts`; `src/components/RenderEngineBadge.tsx`; `src/pages/VideoListPage.tsx:7,323`; `src/components/StatusStrip.tsx:60-…`; `src/pages/ResultPage.tsx:47-70,188`; `src/pages/ValidatePage.tsx:119`, `RenderPage.tsx:114`; `src/hooks/useOutlineReview.ts:48`; `src/pages/ResumeProjectPage.tsx:22,69-…`.
- **Thay đổi**:
  - `retiredEngine.ts`: `export const RETIRED_ENGINE_MESSAGE = "Dự án này dựng bằng Manim, đã ngừng hỗ trợ: chỉ xem, đăng hoặc xoá được.";` và `export function isRetiredEngine(renderEngine?: string): boolean { return renderEngine === "manim"; }` (JSDoc).
  - `RenderEngineBadge`: chỉ render khi `isRetiredEngine`; chữ "Manim (ngừng hỗ trợ)", `title={RETIRED_ENGINE_MESSAGE}`, class `glass.badgeNeutral`. JSDoc viết lại. `VideoListPage` giữ chỗ gọi.
  - `StatusStrip`: project cũ → nhánh đầu tiên: pill "Ngừng hỗ trợ" (`styles.warn`), message `RETIRED_ENGINE_MESSAGE`, actions chỉ `back`; không render `ForkDialog`, nút Chạy tiếp, Nhân bản.
  - `ResultPage`: project cũ → ẩn khối Dựng lại (nút `rerender-submit` và chọn chất lượng); bỏ `render_engine: project.render_engine` khỏi `startRenderSaga` (dòng 66).
  - `ValidatePage`, `RenderPage`: ẩn nút chạy lại khi project cũ. `PublishPage` giữ nút thử lại đăng.
  - `useOutlineReview`: project cũ → không gọi `approveOutline`/`rejectOutline`; trả lỗi `RETIRED_ENGINE_MESSAGE` cho màn hiện.
  - `ResumeProjectPage`: sau `getProject`, nếu `isRetiredEngine(project.render_engine)`: có video (`project.video_path`) → `navigate` tới màn Kết quả; không có → hiện thẻ (`glass` card, `.reveal` + `usePresence` theo `docs/ux-ui-design-rules.md` §3) gồm `RETIRED_ENGINE_MESSAGE` và nút Xoá (gọi hàm xoá project đang có trong `client.ts`, xác nhận trước như danh sách video đang làm), không nạp draft. Dòng 22: `renderEngine` bỏ (T13).
- **Test**: `RenderEngineBadge.test.tsx` (project cũ có nhãn, project mới không); `StatusStrip.test.tsx` (project cũ không có nút Chạy tiếp/Nhân bản, có thông điệp); `ResultPage.test.tsx` (project cũ không có `rerender-submit`); `ResumeProjectPage.test.tsx` (project cũ chưa xong hiện thẻ và nút Xoá; đã xong chuyển tới Kết quả). `npm test`, `npx tsc -b`, `npm run lint`.
- **Xong khi**: FR2 phía giao diện, tiêu chí 2.
- **Phụ thuộc**: T13, T14

---

## Tài liệu và hợp đồng

### T16 — ADR-0033, README, hợp đồng, biến môi trường
- [ ] Xong

- **Service**: tài liệu, `docker-compose.yml`, `.env.example`, `tests/contracts`
- **File**:
  - Mới `aidlc-docs/decisions/ADR-0033-remotion-only-render-engine.md` (theo mẫu ADR-0032): Bối cảnh (hai engine; short chỉ Remotion; tính năng mới chỉ làm cho Remotion; image rendering mang texlive), Quyết định (một engine Remotion; `manim` chỉ đánh dấu project cũ, chỉ xem/đăng/xoá; intro/outro chỉ tải lên; luật QC bố cục theo Manim bỏ), Hệ quả (dữ liệu bị xoá: prompt Manim, `layout_marks`, cache `.manim-media`; không chuyển project cũ sang Remotion). Thêm dòng vào `aidlc-docs/decisions/README.md`.
  - `README.md`: tên và mô tả dự án theo Remotion; bảng biến môi trường bỏ `RENDER_MEMORY_LIMIT_GB`, `RENDER_CACHE_ROOT`, `RENDER_QUALITY`, `DRY_RUN_TIMEOUT_SECONDS` (nếu có), sửa mô tả `RENDER_TIMEOUT_SECONDS` (Remotion).
  - `docker-compose.yml:128-144` (service rendering): bỏ `DRY_RUN_TIMEOUT_SECONDS`, `RENDER_MEMORY_LIMIT_GB`, `RENDER_CACHE_ROOT`, `RENDER_QUALITY` và comment Manim; `.env.example` tương ứng.
  - `docs/contracts/shared-artifacts.md`: bỏ dòng `/shared/.manim-media/{project_id}/` và `/shared/channel-assets/{kind}/{quality}/rendered.mp4`; `tests/contracts/test_shared_artifacts_contract.py` khớp lại.
  - `docs/contracts/authoring-llm-code-v2.md`: `engine` chỉ `"remotion"`; giá trị khác → 400.
  - `aidlc-docs/construction/rendering-service/low-level-design/interface-contracts.md`: bỏ `render_channel_asset`, `channel_asset_rendered`, `channel_asset_render_failed`, route `/v1/check/manim`; `engine` chỉ `remotion`, giá trị khác → sự kiện thất bại.
  - Hợp đồng của orchestrator và video-assembly (`aidlc-docs/construction/{orchestrator-service,video-assembly-service}/low-level-design/interface-contracts.md`): bỏ `register_channel_asset`; `render_engine` không còn ở request; `/internal/v1/projects/{id}/status` có `render_engine`; lỗi 409 ngừng hỗ trợ.
  - `docs/contracts/README.md` nếu bảng có nhắc các mục trên.
- **Test**: `pytest tests/contracts` (ở gốc repo, môi trường đang dùng cho test hợp đồng).
- **Xong khi**: FR7.
- **Phụ thuộc**: T4, T8, T10, T11

---

## Sau cùng

1. **Rebuild** (từ worktree này): `scripts/worktree.sh rebuild orchestrator authoring-service llm-service rendering video-assembly web-gui api-gateway tts`, tất cả healthy. Ghi kích thước image rendering trước/sau (`docker image ls`).
2. **Xoá cache**: `docker compose exec rendering rm -rf /shared/.manim-media` (ở bản checkout chính) sau khi rendering mới đã chạy.
3. **Kiểm dữ liệu sau migration**: `render_engine` của project cũ vẫn `manim`; cột `scene_class_name` có giá trị cũ; `layout_marks` không còn; bảng `prompts` không còn vai Manim; `channel_asset_pointers` như trước (T0).
4. **Kiểm trực tiếp** (tiêu chí 1-5 của thiết kế):
   - Tạo project mới: không có ô chọn engine; chạy trọn tới video.
   - Project cũ (nếu có): nhãn "Manim (ngừng hỗ trợ)"; mở video, đăng được; không có nút chạy lại; `curl -X POST …/v1/projects/{id}/retry` (qua api-gateway) → 409 với thông điệp ngừng hỗ trợ.
   - Tải lại một intro/outro: chuẩn hoá và ghép như cũ.
   - `docker compose exec rendering python -c "import manim"` → `ModuleNotFoundError`.
   - `grep -rni manim services/ --exclude-dir=.venv --exclude-dir=node_modules` chỉ còn các chỗ được phép (tiêu chí 5).
5. Báo cáo kết quả từng mục; mục nào không chạy được thì ghi rõ là chưa chạy.
