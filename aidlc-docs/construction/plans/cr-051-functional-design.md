# CR-051 — Functional Design: số bước và tiến độ nhất quán, Hình minh hoạ là bước 5 thật

## Date
2026-09-29

## Stage
Functional Design, **chờ Creator duyệt**. Yêu cầu nằm ở `aidlc-docs/inception/requirements/cr-051-pipeline-progress-ux-audit.md` (FR1–FR9, đã duyệt).

## Nguyên tắc chung
Một nguồn sự thật cho số bước và tên bước: luồng 14 bước. Tên bước là tên trên thanh bước (`FLOW_LABELS` ở web, `FlowStepLabel` ở server).
- Mọi nơi Creator nhìn thấy (tiêu đề màn, tiến độ chuỗi AI, tracker, nhật ký) đều dùng cặp "Bước N — <tên>".
- Không nơi nào còn đánh số riêng kiểu "1/3" hay "1, 2".
- Việc con bên trong một bước dùng số con "N.k".

## Đơn vị 1 — orchestrator + authoring-service (FR6, FR7, FR8)

### D1. Thêm cờ `illustrations_ready` vào tóm tắt nội bộ (FR8)
- **Hợp đồng**: `GET /internal/v1/authoring/summaries?ids=` (ADR-0029). Mỗi phần tử có thêm trường boolean `illustrations_ready`. Trường này chỉ được thêm vào, không đổi hay xoá trường cũ.
- **authoring-service**:
  - `application.AuthoringSummary` thêm `IllustrationsReady bool \`json:"illustrations_ready"\``.
  - `PromptTemplateRepository.Summaries` tính cờ này trong cùng câu SQL, không thêm vòng gọi. Dự án "sẵn sàng" khi thoả cả hai điều kiện:
    1. đã lập danh sách hình: `illustrations_planned_at IS NOT NULL` hoặc đã có dòng trong `project_illustrations`;
    2. không có dòng nào chưa sẵn sàng. Một dòng sẵn sàng khi `state = 'skipped'`, hoặc khi `state IN ('reused','drawn')` và hình trong thư viện tồn tại, đồng thời là hình có sẵn (`builtin`) hoặc đã duyệt (`status = 'approved'`).
  - Điều kiện 2 dịch đúng từng dòng của `ProjectIllustration.Ready()`, và cả hai điều kiện cùng khớp với `ProjectIllustrationsUseCase.Gate` (quy tắc đang chặn bước Code). Có test repository/integration đối chiếu SQL với `Ready()` trên cùng dữ liệu, nếu hạ tầng test Postgres có sẵn. Nếu không có, tôi sẽ nói rõ trong báo cáo.
  - Cờ này không phụ thuộc engine. Authoring tính cho mọi dự án, việc Manim có dùng cờ hay không là do orchestrator quyết định.
- **orchestrator**:
  - `application.AuthoringSummary` thêm `IllustrationsReady *bool \`json:"illustrations_ready"\``. Con trỏ `nil` nghĩa là authoring-service bản cũ chưa gửi trường này. Khi đó coi là sẵn sàng, tức giữ đúng hành vi hiện tại, để khi deploy lệch nhau thì dự án không bị kéo lùi về bước 5.

### D2. `FlowStateFor` biết tới bước 5 (FR8)
- `AuthoredContent` thêm hai trường `NeedsIllustrations` (dự án dùng Remotion) và `Illustrations` (đã sẵn sàng).
- Nhánh dự án nháp đang ở bước soạn, theo thứ tự:
  1. có Code → 6 (Code);
  2. có Storyboard và (`!NeedsIllustrations` hoặc `Illustrations`) → 6 (Code);
  3. có Storyboard → **5 (Hình minh hoạ)**;
  4. có Story → 4 (Visual);
  5. còn lại → 3 (Kịch bản).
- Luật 1 đặt trước luật 3 để dự án cũ (Remotion có code nhưng chưa từng lập danh sách hình, từ trước CR-044) không bị kéo lùi về bước 5.
- Nơi gọi trong orchestrator điền `NeedsIllustrations = RenderEngine == remotion`, lấy `Illustrations` từ D1. Hai nơi cần sửa là `router.flowFor` và `projectListReader` ở `cmd/orchestrator/main.go`. Các nơi gọi với `AuthoredContent{}` (dự án không ở trạng thái nháp, hoặc nhật ký sự kiện) giữ nguyên, vì nhánh nháp không dùng tới.
- `run_state` của bước 5 là `idle` giống 3/4/6. Việc hiện "đang chạy" trên thanh bước là phương án B, Creator không chọn.
- Sửa comment "first of 3/4/5" thành mô tả 3/4/5/6.

### D3. Nhãn bước server trùng thanh bước (FR7)
- `FlowStepLabel`: `FlowRender` đổi thành "Render", `FlowSplit` đổi thành "Cắt short".
- `step_label` của nhật ký được tính lúc đọc (`project_repository.go`), nên sự kiện cũ cũng hiện tên mới. Không cần migrate.

### D4. Test hợp đồng cho định nghĩa luồng (FR6)
- Không gom `flow.go` thành module Go dùng chung, vì mỗi service là một Go module riêng với build context Docker riêng (ADR-0001, ADR-0029). Thay vào đó thêm `tests/contracts/test_flow_steps_contract.py`, chỉ dùng stdlib và pytest, chạy trong `make check` như contract test hiện có. Test kiểm:
  1. `services/authoring-service/internal/domain/flow.go` và `services/orchestrator/internal/domain/flow.go` **giống hệt từng byte**. Hai file đang giống hệt nhau, nên đây là cách chặn lệch đơn giản và chặt nhất.
  2. 14 hằng `Flow*` và `FlowStepLabel` đọc từ Go khớp theo thứ tự với `FLOW_LABELS` trong `services/web-gui/src/utils/flow.ts`, và `FLOW_ILLUSTRATIONS/FLOW_CODE/FLOW_VALIDATE/FLOW_REVIEW/FLOW_TTS/FLOW_RESULT/FLOW_PUBLISH` khớp số với Go.
- `scripts/check.sh` ở chế độ `changed` hiện chỉ chạy contract test khi `tests/contracts/` hoặc `docs/contracts/` đổi. Tôi thêm ba đường dẫn trên vào điều kiện kích hoạt, để sửa một trong ba file là chạy test này. File `scripts/**` thuộc diện `ask`, nên sẽ có hỏi bạn khi sửa.

## Đơn vị 2 — web-gui (FR1–FR5, FR9)

### D5. Chuỗi AI bước 3–6 (FR1, FR2; Q1-A, Q2-A)
Trong `AuthoringModeBar.tsx`:
- Thêm bảng `STEP_FLOW: Record<AuthoringStep, number> = { story: 3, storyboard: 4, illustrations: 5, code: 6 }`, lấy số từ hằng của `utils/flow.ts`, và hàm `stepName(step)` trả "Bước 4 — Visual".
- **Nút**: chuỗi hiện "Chạy bằng AI các bước 3–6". Chạy một bước hiện như cũ: "Chạy <what> bằng AI".
- **Dòng trạng thái khi đang chạy**: "Đang chạy Bước 4 — Visual. Có thể mất vài phút. Vui lòng không đóng trang." Đang chạy ở màn khác: "Đang chạy ở Bước 4 — Visual. Vui lòng chờ hoàn tất."
- **Stepper** (`authoring-run-panel`) luôn vẽ đủ 4 dòng 3/4/5/6, mỗi dòng có chấm tròn mang số thật và tên bước.
  - Với dự án Manim, dòng 5 hiện `data-state="skipped"`, làm mờ, ghi "Không dùng", giống thanh bước. Dòng này không nằm trong `run.steps`, nên cách tính done/running/pending vẫn dựa trên danh sách bước thực chạy.
  - Style của dòng skipped dùng lại cách làm mờ của `StepRail` (token theme, không hard-code màu).
- **Tiêu đề thẻ và `aria-label`**: đổi "Cách làm bước 3" thành "Cách làm các bước 3–6".
- **"Lần chạy AI gần nhất"**: nhãn dòng đổi thành "Bước N — tên".
- `authoringChainSteps` giữ nguyên (Manim vẫn không chạy illustrations). Chỉ phần hiển thị thay đổi.

### D6. Tracker màn Validate đánh số 7.1, 7.2 (FR3; Q4-A)
- `ProgressTracker.stepNumbers` đổi kiểu thành `Record<string, number | string>`.
- `pipelineLabels.ts` thêm `VALIDATE_SUBSTEP_NUMBERS = { parse_script: "7.1", validate_script: "7.2" }`, lấy từ `FLOW_VALIDATE`. `ValidatePage` truyền giá trị này vào cả hai chỗ vẽ tracker.

### D7. Tiêu đề màn "Bước N — <tên trên thanh bước>" (FR4)
| Màn | Hiện tại | Mới |
|---|---|---|
| ScriptStepPage | Bước 1 — Ý tưởng | Bước 1 — Khởi tạo |
| ScriptAuthoringSettingsStepPage | Bước 2 — Cấu hình Remotion/Manim | Bước 2 — Cấu hình (engine chuyển sang đầu subtitle: "Remotion · …") |
| ValidatePage (đang chạy/lỗi/huỷ) | Đang kiểm tra kịch bản / Kịch bản không chạy được / Đã hủy kiểm tra | Bước 7 — Validate. Câu trạng thái chuyển xuống subtitle, bản lỗi và bản huỷ mở đầu bằng câu cũ. |
| ValidatePage (chờ duyệt/xem lại bước 8) | Duyệt dàn ý trước khi sản xuất | Bước 8 — Review. Subtitle: "Duyệt dàn ý trước khi sản xuất. …" |
| ResultPage | Xem kết quả | Bước 13 — Kết quả |
| PublishPage | Đăng video | Bước 14 — Publish |
- Tiêu đề lấy từ một hàm chung `flowTitle(step)` trong `utils/flow.ts`, dùng `FLOW_LABELS`, không gõ tay chuỗi. Các màn 3–6 và `RenderPage` cũng chuyển sang dùng hàm này.

### D8. Dọn mô hình "wizard 7 bước" (FR5)
- Xoá `WIZARD_STEP_NAMES` và `wizardStepLabel` (không còn chỗ nào gọi).
- Sửa comment còn mô tả mô hình cũ trong các file bị đụng: "Bước 4 của 7", "Bước 5 của 7", "Bước 5/6", "tab 1a/1b/1c", "Bước 1b trước" (thông báo lỗi server `chưa có storyboard — hãy chạy bước 1b trước` đổi thành "…hãy chạy Bước 4 — Visual trước", vì Creator nhìn thấy câu này).

### D9. Mở lại dự án (FR9)
- `ResumeProjectPage`: `step = stepParam || Math.min(flowStep, FLOW_CODE)`, với `FLOW_CODE = 6` lấy từ `utils/flow.ts`.

## Ngoài phạm vi (ghi rõ)
- Trạng thái "đang chạy" trên thanh bước cho chuỗi AI (Q1-B, Creator không chọn).
- Làm lại dự án từ bước 5 (`FORK_STEPS` loại bước 5) vì chưa sao chép được hình đã vẽ. Giữ nguyên như CR-046.

## Kiểm thử
- **Go** (cả hai service): cập nhật `flow_test.go` với các case mới: Remotion + storyboard + chưa sẵn sàng → 5; Remotion + sẵn sàng → 6; Manim + storyboard → 6; có code nhưng chưa sẵn sàng → 6; nhãn mới. Test JSON của client orchestrator: thiếu `illustrations_ready` thì coi là sẵn sàng.
- **authoring-service**: test cho `Summaries` (có sẵn / không có sẵn / đã bỏ qua / chưa lập danh sách).
- **Contract**: `tests/contracts/test_flow_steps_contract.py`.
- **web-gui** (vitest): cập nhật `ScriptOutlineStepPage`, `ValidatePage`, `ScriptStepPage`, `RenderPage`, `IllustrationsStepPage`, `JournalPage`, `ProgressTracker`. Thêm case: stepper Manim có dòng 5 "Không dùng"; tracker hiện 7.1/7.2; resume flow_step 6 → màn Code.
- `make check`, sau đó rebuild và restart `authoring-service`, `orchestrator`, `web-gui` và xác nhận healthy.
- Kiểm tay trên giao diện thật: dự án Remotion ở bước 5, chuỗi AI Manim và Remotion, màn Validate. Nếu không làm được, sẽ ghi rõ là chưa làm.
