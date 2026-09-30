# CR-053 — Rà soát và xoá file không dùng

## 1. Yêu cầu gốc (nguyên văn)

> review lại toàn bộ dự án xem file nào không dùng thì remove hết giúp tôi nhé

Trả lời của Creator cho mục 4 (nguyên văn):

> 1 chọn a
> 2 chạy rồi xoá đi
> 3 xoá đi
> ok làm đi

→ Giữ toàn bộ nhóm B; xoá `scripts/migrate-authoring-data.sh`; xoá nhóm C.

## 2. Hiện trạng — cách rà và kết quả

### 2.1 Cách rà

graphify (graph tại `1952533`, khớp HEAD) không bắt được phần lớn import Python dạng `from adapters.persistence.db import ...`, nên danh sách "file mồ côi" của graph (≈110 file) có rất nhiều báo động giả. Vì vậy mỗi loại file được kiểm lại bằng cách riêng:

| Loại | Cách kiểm |
|---|---|
| Python | Tìm `import`/`from ... import` theo đường dẫn module đầy đủ (tính từ gốc service), tách người dùng là test và không phải test. |
| TS/JS | Phân giải mọi import tương đối (`import`, `export from`, `import()`, `require`) ra file thật. |
| Go | `go list` tìm package không ai import; với từng file, xem các symbol cấp cao có được file khác (không phải `_test.go`) cùng service dùng không. |
| Asset / file dữ liệu | Tìm tên file trong code, Dockerfile, compose, `go:embed`. |
| Công cụ chạy tay | Tìm lệnh gọi trong code, Dockerfile, Makefile, docs. |

Entrypoint (`main.go`, `main.py`, `server.js`, `main.tsx`), file cấu hình (`vite.config.ts`, `.eslintrc.*`, `vite-env.d.ts`, `pyproject.toml`), `__init__.py` và file được nạp bằng tên (`render.mjs`, `tscheck.mjs`, `layout_check.mjs`, `illustration_preview.mjs`, `src/illustration-preview/host.tsx`, `src/layout-probe/*`, `templates/starter_script_*.py` qua `go:embed`, `public/lottie/cat.*.json` qua `lottie/manifest.json`) được loại khỏi danh sách vì đều có đường gọi thật.

### 2.2 Nhóm A — code chết thật (không đường gọi nào trong runtime)

| File | Bằng chứng |
|---|---|
| `services/web-gui/src/components/ScriptEditor.tsx` (256 dòng) | `ScriptEditor` (`ScriptEditor.tsx:74`) chỉ được `tests/components/ScriptEditor.test.tsx` import. Màn Kỹ sư dùng `ManimEngineerStepPage.tsx` thay thế. |
| `services/web-gui/src/components/ScriptEditor.module.css` | Chỉ `ScriptEditor.tsx:6` dùng. |
| `services/web-gui/tests/components/ScriptEditor.test.tsx` | Test của component chết. |
| `services/web-gui/src/hooks/useRequireScript.ts` | `useRequireScript` (`useRequireScript.ts:21`) chỉ test import; không trang nào gọi từ CR-030. |
| `services/web-gui/tests/hooks/useRequireScript.test.tsx` | Test của hook chết. |
| `services/web-gui/src/hooks/useVoiceCalibration.ts` | Không file nào import `useVoiceCalibration` / `wordsPerMinuteFor` (từ CR-016). |
| `services/web-gui/public/favicon-48.png` | `index.html:6-8` và `site.webmanifest` không nhắc tới; chỉ dùng `favicon.ico`, `-16`, `-32`, `icon-192/512`, `apple-touch-icon`. |
| `services/rendering/conceptflow/fonts.py` | `installed_families`, `missing`, `VIETNAMESE_PROBE` không được gọi ở đâu (kể cả test) từ CR-017. |
| `services/authoring-service/internal/adapters/logging/correlation.go` | Package `authoring/internal/adapters/logging` không được import (`go list`). |
| `services/orchestrator/internal/adapters/logging/correlation.go` | Package `orchestrator/internal/adapters/logging` không được import. |
| `services/authoring-service/internal/application/idgen.go` | `newUUID` chỉ được dùng trong orchestrator (bản của orchestrator ở `orchestrator/internal/application/idgen.go`); bản authoring là bản sao sót lại sau CR-040. |
| `services/authoring-service/internal/domain/topic.go` | `NormalizeTopic` chỉ orchestrator gọi (`create_project_draft.go`), dùng bản của orchestrator. |
| `services/authoring-service/internal/domain/clip_rules.go` + `clip_rules_test.go` | `ValidateClipDuration`, `ClipPreset*` chỉ orchestrator dùng (bản của orchestrator); bản authoring chỉ có test của chính nó. |
| `services/authoring-service/internal/domain/script_edit.go` + `script_edit_test.go` | `ReplaceNarrationLiteral`, `ErrNarrationNotEditable` chỉ orchestrator dùng (bản của orchestrator). |
| `services/orchestrator/internal/domain/testdata/` (4 file: `beats_en_default.txt`, `beats_vi_calibrated.txt`, `beats_vi_default.txt`, `format.json`) | Không test nào của orchestrator đọc `testdata`; test golden đã chuyển sang authoring-service (`authoring-service/internal/domain/testdata/`, nội dung giống hệt). |

Kéo theo khi xoá nhóm A: `fetchVoiceCalibration` (`services/web-gui/src/api/client.ts:101`) chỉ có `useVoiceCalibration.ts` gọi, nên cũng thành code chết.

### 2.3 Nhóm B — công cụ chạy tay (không nằm trong runtime, nhưng có mục đích)

| File | Dùng để |
|---|---|
| `services/rendering/remotion_project/render_gallery.mjs`, `src/gallery/index.tsx`, `src/gallery/tooth-decay-demo.tsx` | Dựng bảng xem thử component Remotion bằng mắt (`node render_gallery.mjs sheets <outDir>`). |
| `services/rendering/remotion_project/layout_probe.mjs`, `layout_probe_samples/build_samples.py` + `*.tsx` mẫu | CLI đo bố cục; sinh lại `layout_probe_samples/*.layout.json` mà `tests/domain/test_layout_rules.py` và `tests/adapters/test_layout_checker.py` đang đọc. |
| `services/rendering/tools/gen_theme_reference.py` | Sinh lại khối theme trong prompt (`authoring-service/internal/domain/prompt_vars.go:32` ghi "GENERATED ... never edit by hand"). |
| `services/rendering/tools/lottie_catalog.py`, `tools/build_avatar.py` | Sinh catalog Lottie / dựng avatar mèo (`lottie/manifest.json` ghi nguồn). |
| `tests/benchmark_render.py`, `tests/fixtures/long_form_reference.py` | Benchmark render video dài (CR-002/007). |
| `docs/brand/make-banner.py` | Dựng lại banner/logo kênh (nhắc trong 3 tài liệu brand và `conceptflow/theme.py`). |
| `scripts/migrate-authoring-data.sh` | Chép dữ liệu authoring một lần từ DB orchestrator sang DB authoring (CR-040 FR111). |

### 2.4 Nhóm C — tài liệu rời trong `docs/` không ai tham chiếu

| File | Ghi chú |
|---|---|
| `docs/story_architect_tu_lieu.txt` | Bản nháp prompt Story Architect (commit `31df8f6`); nội dung thật đã nằm trong `prompt_template_seeds.go`. |
| `docs/format_case_study_essay.json` | Bản nháp format `case_study_essay_8min`; nội dung thật nằm trong `builtin_formats.go` của authoring-service và orchestrator. |

### 2.5 Không đụng tới

- `aidlc-docs/**`, `.ai-dlc/**`, `docs/contracts`, `docs/setup`, `docs/review`, `docs/brand/*.md`, ADR: lịch sử quy trình AI-DLC và tài liệu vận hành, được code và CLAUDE.md nhắc tới.
- Endpoint `GET /v1/voice-calibration` (orchestrator `router.go:310`, api-gateway `routes/projects.js:68`): sau khi xoá `useVoiceCalibration` thì web-gui không còn gọi, nhưng xoá endpoint là thay đổi API, không phải xoá file — để CR riêng nếu cần.
- Symbol chết bên trong file vẫn còn dùng (hàm export lẻ không ai gọi): ngoài phạm vi "file không dùng".

## 3. Yêu cầu

- **FR1**: Xoá toàn bộ file nhóm A (mục 2.2) và hàm `fetchVoiceCalibration` trong `client.ts`.
- **FR2**: Xoá `scripts/migrate-authoring-data.sh` và 2 file nhóm C (`docs/story_architect_tu_lieu.txt`, `docs/format_case_study_essay.json`). Giữ nguyên toàn bộ nhóm B.
- **FR3**: Không thay đổi hành vi: mọi service build được, toàn bộ test hiện có (trừ test của chính file bị xoá) vẫn pass.

**Tiêu chí chấp nhận**: `go build ./... && go test ./...` pass ở authoring-service và orchestrator; `pytest` pass ở rendering; `tsc --noEmit` + `vitest run` pass ở web-gui; các container bị ảnh hưởng rebuild xong và healthy; `git grep` không còn tham chiếu tới file đã xoá ngoài `aidlc-docs/`.

**Ngoài phạm vi**: tài liệu AI-DLC, endpoint `/v1/voice-calibration`, symbol chết lẻ trong file còn dùng.

## 4. Câu hỏi cho Creator (đã trả lời: 1a, 2a, 3a)

1. **Nhóm B — công cụ chạy tay** (gallery, layout probe, tools/*.py, benchmark, make-banner):
   - a) Giữ lại hết (**đề xuất**): không nằm trong runtime nhưng là cách duy nhất để sinh lại file generated (theme prompt, catalog Lottie, `*.layout.json` mà test đọc, banner). Xoá thì lần sau phải viết lại.
   - b) Xoá gallery (`render_gallery.mjs` + `src/gallery/`) và benchmark (`tests/benchmark_render.py` + `tests/fixtures/long_form_reference.py`) — hai thứ này không sinh ra file nào đang dùng; giữ phần còn lại.
   - c) Xoá hết nhóm B.
2. **`scripts/migrate-authoring-data.sh`** — script chuyển dữ liệu một lần của CR-040. Anh/chị đã chạy nó trên máy thật chưa?
   - a) Đã chạy xong → xoá (**đề xuất nếu đã chạy**).
   - b) Chưa chạy / không chắc → giữ.
3. **Nhóm C — `docs/story_architect_tu_lieu.txt`, `docs/format_case_study_essay.json`**:
   - a) Xoá (**đề xuất**): là bản nháp, nội dung thật đã nằm trong seed của authoring-service.
   - b) Giữ làm tài liệu tham khảo.

## 5. Giải pháp đề xuất

Chỉ xoá file (và một hàm kéo theo), không sửa logic. Nhóm A xoá luôn vì đã chứng minh không có đường gọi. Nhóm B/C theo câu trả lời mục 4. Kiểm chứng bằng build + test đầy đủ của từng service bị đụng, vì trình biên dịch/test là bằng chứng mạnh hơn phân tích tĩnh (đặc biệt với Go: xoá file trùng tên trong cùng package sẽ lộ ngay nếu còn ai dùng).

## 6. Phạm vi

- **web-gui**: 5 file + 1 ảnh public + `fetchVoiceCalibration` trong `src/api/client.ts`.
- **rendering**: `conceptflow/fonts.py`.
- **authoring-service**: `adapters/logging/correlation.go`, `application/idgen.go`, `domain/topic.go`, `domain/clip_rules.go` + test, `domain/script_edit.go` + test.
- **orchestrator**: `adapters/logging/correlation.go`, `domain/testdata/` (4 file).
- **repo gốc**: `docs/story_architect_tu_lieu.txt`, `docs/format_case_study_essay.json`, `scripts/migrate-authoring-data.sh`, `.claude/settings.json` (bỏ quyền của script).
- Không đổi contract, DB, migration. `graphify affected` không cần cho symbol nào vì không symbol nào còn người gọi.

## 7. Kế hoạch thực hiện (cho `/code`)

1. `git rm` các file nhóm A (mục 2.2), gồm cả thư mục `services/orchestrator/internal/domain/testdata/`.
2. Sửa `services/web-gui/src/api/client.ts`: xoá hàm `fetchVoiceCalibration` (dòng 101 tới hết thân hàm) và chú thích đi kèm nếu có.
3. `git rm scripts/migrate-authoring-data.sh docs/story_architect_tu_lieu.txt docs/format_case_study_essay.json`; bỏ dòng quyền chạy `migrate-authoring-data.sh` trong `.claude/settings.json`. Không đụng nhóm B.
4. `git grep` tên từng file đã xoá (trừ `aidlc-docs/`) để chắc không còn tham chiếu.
5. Chạy kiểm tra (mục 8). Nếu có gì fail vì file vừa xoá: dừng, khôi phục file đó và báo lại, không vá vòng.
6. Rebuild và khởi động lại service bị đụng.

## 8. Kiểm tra

- authoring-service, orchestrator: `go vet ./... && go test ./...`.
- rendering: `pytest` (qua container hoặc venv như các CR trước).
- web-gui: `npx tsc --noEmit && npx vitest run`.
- Rebuild: `docker compose build authoring-service orchestrator rendering web-gui && docker compose up -d authoring-service orchestrator rendering web-gui`; xác nhận healthy.
- Kiểm trực tiếp: mở web-gui, đi qua wizard tới bước Kỹ sư và trang Kết quả; favicon vẫn hiện.

## 9. Rủi ro

- **Tham chiếu động bị bỏ sót** (file được nạp bằng chuỗi ghép lúc chạy): đã kiểm tên file trong toàn repo; build + test + chạy thử là lớp chặn cuối. Mọi thứ nằm trong git nên khôi phục được bằng `git revert`.
- **Không mất dữ liệu**: không đụng DB, volume, hay file generated đang được đọc.
- **Script migrate**: Creator xác nhận đã chạy; nếu cần lại thì lấy từ lịch sử git (commit `1ad8c2c`).
