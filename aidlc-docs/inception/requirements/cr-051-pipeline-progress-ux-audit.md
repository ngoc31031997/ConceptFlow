# CR-051 — Rà soát quy trình server + giao diện: chỗ gây nhầm lẫn về bước và tiến độ

## Date
2026-09-29

## Stage
Requirements Analysis — **đã được Creator duyệt** (vòng 2, 2026-09-29). Bước tiếp theo: Functional Design (`aidlc-docs/construction/web-gui/cr-051-functional-design.md`).

## Yêu cầu của Creator (nguyên văn)
"kiểm tra lại giúp tôi toàn bộ quy trình server và giao diện xem có gây confuse gì ko, ví dụ bước 3 4 5 6 là 4 bước riêng nhưng loading lại hiện 3 bước và hiện chung"

## Phân tích ý định
- **Loại yêu cầu**: Enhancement (UX nhất quán), bắt đầu bằng một đợt rà soát.
- **Độ rõ**: ví dụ cụ thể thì rõ; phạm vi "toàn bộ" cần chốt (xem câu hỏi 3).
- **Phạm vi ước tính**: chủ yếu `web-gui`. Phía server không thấy lệch (xem F6).
- **Độ phức tạp**: Simple → Moderate, tuỳ phương án ở câu hỏi 1.
- **Độ sâu**: Standard.

## Cách rà soát
Đọc code, chưa xem trên UI chạy thật. `graphify` không có trên PATH của phiên này (có ở `~/.local/bin`), nên phạm vi được xác định bằng cách đọc trực tiếp các file bên dưới.

Nguồn sự thật về số bước: luồng 14 bước (`FlowInit=1 … FlowPublish=14`). Luồng này được định nghĩa trùng nhau ở `services/authoring-service/internal/domain/flow.go` và `services/orchestrator/internal/domain/flow.go`, và phía web ở `services/web-gui/src/utils/flow.ts` (`FLOW_LABELS`, `FLOW_PHASES`). Thanh bước (`StepRail`) vẽ đúng 14 bước này.

## Phát hiện

### F1 — Chuỗi AI của bước 3–6 hiện thành "3 bước" đánh số riêng (đúng ví dụ Creator nêu)
- Nút AI trên màn **Bước 3 — Kịch bản** (`ScriptOutlineStepPage` → `AuthoringModeBar`, `steps = authoringChainSteps(renderEngine)`) chạy **cả chuỗi** `story → storyboard → [illustrations] → code` trong một lần bấm.
- Với dự án Manim, `authoringChainSteps` bỏ `illustrations` nên chuỗi chỉ còn **3 mục**. Trong khi đó thanh bước vẫn hiện 4 bước riêng 3/4/5/6, bước 5 ghi "Không dùng". Dự án Remotion thì chuỗi có 4 mục.
- Khi đang chạy:
  - dòng trạng thái ghi `Bước ${currentIndex+1}/${steps.length} — …`, tức "Bước 1/3 — Kịch bản", "Bước 2/3 — Visual"… Cách đánh số này không khớp với số 3–6 trên thanh bước;
  - nút ghi "Chạy cả 3 bước bằng AI";
  - bảng tiến độ (`authoring-run-panel`) là một stepper riêng, không có số, và hiện **giống hệt nhau trên cả 4 màn** 3/4/5/6;
  - tiêu đề thẻ là "Cách làm bước 3", dù lựa chọn này áp dụng cho cả các bước 3–6.
- Tham chiếu: `services/web-gui/src/components/AuthoringModeBar.tsx` (STEP_LABELS, `authoringChainSteps`, `runLabel`, dòng trạng thái đang chạy, run panel, "Cách làm bước 3").

### F2 — Màn Validate (bước 7) đánh số hai việc bên trong là "1" và "2"
- `ValidatePage` gọi `ProgressTracker` với `steps={VALIDATE_STEPS}` nhưng **không truyền `stepNumbers`**. Vì vậy chấm tròn hiện số **1** "Phân tích kịch bản" và **2** "Chạy thử & kiểm tra". Người xem dễ đọc thành bước 1 và bước 2 của luồng, trong khi cả hai đều thuộc bước 7.
- `RenderPage` đã truyền `stepNumbers={SAGA_FLOW_STEP}` nên không bị lỗi này. Hai màn cùng dùng một component nhưng hiện số theo hai cách khác nhau.

### F3 — Tên bước trên tiêu đề màn khác với tên trên thanh bước
- Bước 1: tiêu đề màn là "Bước 1 — Ý tưởng", thanh bước ghi "Khởi tạo".
- Bước 2: tiêu đề là "Bước 2 — Cấu hình Remotion/Manim", thanh bước ghi "Cấu hình". Lệch nhẹ, chấp nhận được.
- Bước 7/8 ("Đang kiểm tra kịch bản", "Duyệt dàn ý trước khi sản xuất"), 13 ("Xem kết quả") và 14 ("Đăng video") **không có số bước** trên tiêu đề. Trong khi đó 1–6 và 9–12 (`RenderPage`: `Bước ${n} — ${label}`) thì có. Cách đặt tiêu đề không thống nhất giữa các màn.

### F4 — Code cũ của mô hình "wizard 7 bước" vẫn còn
- `pipelineLabels.ts`: `WIZARD_STEP_NAMES` (7 tên: Ý tưởng, Cấu hình, Script, Validate, Xử lý, Kết quả, Đăng) và `wizardStepLabel()` không còn chỗ nào dùng. Các comment `projectPhase`, "Bước 4 của 7" (`ValidatePage`), "Bước 5 của 7" (`RenderPage`), "Bước 5/6" (`ResultPage`, `PublishPage`), "tab 1a/1b/1c" vẫn mô tả mô hình cũ.
- Creator không nhìn thấy những chỗ này. Nhưng chúng dễ làm người sửa code sau này (người hoặc agent) hiểu sai số bước và tái tạo lại đúng lỗi F1/F2.

### F5 — Một màn hình cho nhiều bước (ghi nhận, có thể giữ nguyên)
- Bước 7 và 8 dùng chung `/validate`; bước 9–12 dùng chung `/render`. Mỗi lần xem, màn `/render` chỉ hiện một bước và tiêu đề có đúng số bước, nên nhất quán. Màn `/validate` là chỗ lẫn: xem F2, F3.

### F6 — Phía server: không thấy lệch
- `flow.go` của authoring-service và orchestrator hiện **giống hệt nhau** (144 dòng, cùng 14 hằng số). Điều này đúng ở thời điểm rà soát. Rủi ro nằm ở chỗ định nghĩa bị sao chép sang hai nơi: sửa một bên mà quên bên kia là lệch. Đây là nợ kỹ thuật, không phải lỗi UX hiện tại.
- Tên các bước saga (`parse_script`, `validate_script`, `synthesize_speech`, `render_scenes`, `assemble_video`, `generate_clips`) khớp với `STEP_LABELS`/`SAGA_FLOW_STEP` phía web.

### F7 — Nhãn bước phía server khác nhãn trên thanh bước (phát hiện ở vòng 2)
- `FlowStepLabel` (Go, ở cả hai service) dùng "Render hoạt hình" và "Cắt video short". Web `FLOW_LABELS` dùng "Render" và "Cắt short".
- Server gửi `step_label` trong nhật ký sự kiện, và trang Nhật ký (`JournalPage`) hiện nguyên nhãn đó. Kết quả là cùng một bước có hai tên, tuỳ bạn đang xem trang nào.

### F8 — Server xếp dự án Remotion nháp thẳng vào bước 6, bỏ qua bước 5 (phát hiện ở vòng 2)
- `FlowStateFor` (ở cả hai service): khi dự án nháp đã có storyboard thì trả về `FlowCode` (6). Hàm này không bao giờ trả về `FlowIllustrations` (5), vì `AuthoredContent` và phần tóm tắt nội bộ authoring → orchestrator (`AuthoringSummary`: topic/story/storyboard/code) không có thông tin nào về hình minh hoạ.
- Hệ quả (suy ra từ code, chưa xem trên UI thật): với dự án Remotion đã xong Visual nhưng hình minh hoạ chưa duyệt hết, thanh bước đánh dấu bước 5 là ✓ xong và đứng ở bước 6. Trong khi đó bước Code vẫn bị chặn cho tới khi mọi hình đã duyệt hoặc bỏ qua. Danh sách dự án ("Bước 6 — Code") và nút "mở lại dự án" cũng lệch theo.
- Comment của `FlowStateFor` vẫn ghi "first of 3/4/5", là mô hình trước CR-046.

## Câu trả lời của Creator (vòng 1)
- Q1: **A**. Creator trả lời "1 ok", AI hiểu là A và đã báo lại để Creator sửa nếu hiểu sai.
- Q2: **A**, hiện bước 5 làm mờ, ghi "Không dùng".
- Q3: **C**, gồm cả phía server.
- Q4: **A**, đánh số "7.1" và "7.2".
- Câu hỏi thêm cho F8 nằm ở `cr-051-questions.md`, Question 5.

## Câu trả lời của Creator (vòng 2)
- Xác nhận Q1 = A, F7 sửa theo tên trên thanh bước.
- Q5 = **A**: "đúng như vậy. cần tách nó thành làm step riêng thật sự". Hình minh hoạ phải là một bước riêng thật sự cả ở phía server (vị trí dự án ở bước 5), không chỉ trên giao diện.

### F9 — Màn mở lại dự án chặn trần ở bước 5 (phát hiện khi thiết kế F8)
- `ResumeProjectPage`: `step = stepParam || (flowStep > 5 ? 5 : flowStep)`. Con số 5 là bước Code của luồng 13 bước trước CR-046.
- Trong luồng 14 bước, dự án nháp đang ở bước Code (6) khi mở lại bị đưa sang `authoringRoute(5)`, tức màn Hình minh hoạ. Dự án Manim cũng bị, dù với Manim bước này là "Không dùng". Lỗi này thuộc "nút mở lại dự án lệch" nên đưa vào CR (FR9).

## Yêu cầu chức năng
- **FR1** — Khi chuỗi AI chạy, mọi chỗ hiển thị tiến độ đều dùng **số và tên bước của luồng 14 bước** ("Bước 4 — Visual"), không dùng số thứ tự trong chuỗi ("2/3"). Cách trình bày cụ thể tuỳ câu hỏi 1–2.
- **FR2** — Tiêu đề/chú thích của lựa chọn cách làm nói đúng phạm vi áp dụng (bước 3–6), không ghi "bước 3".
- **FR3** — Tracker của màn Validate không hiện số làm người xem tưởng là bước 1/2 của luồng (câu hỏi 4).
- **FR4** — Tên bước trên tiêu đề màn khớp với `FLOW_LABELS`. Việc hiện số bước trên tiêu đề áp dụng thống nhất cho mọi màn (câu hỏi 3).
- **FR5** — Xoá `WIZARD_STEP_NAMES`/`wizardStepLabel` (đã không còn dùng) và sửa các comment còn mô tả wizard 7 bước (câu hỏi 3).
- **FR6** — (Q3-C) Một test hợp đồng ở `tests/contracts/` kiểm rằng hai bản `flow.go` (authoring-service, orchestrator) và `flow.ts` (web) có cùng 14 số bước và cùng tên bước. Test này chạy trong `make check`. Không gom thành một module Go dùng chung, vì mỗi service là một Go module riêng với build context Docker riêng (ADR-0001).
- **FR7** — (F7) `FlowStepLabel` phía server dùng đúng tên trên thanh bước. FR6 kiểm luôn điều này để hai bên không lệch lại.
- **FR8** — (F8, Q5-A) Dự án Remotion nháp đã có storyboard nhưng hình minh hoạ chưa sẵn sàng (chưa lập danh sách, hoặc còn hình chưa duyệt/chưa bỏ qua) thì server xếp ở bước 5. Điều kiện "sẵn sàng" dùng đúng quy tắc chặn bước Code (`ProjectIllustrationsUseCase.Gate` + `ProjectIllustration.Ready`). Dự án Manim không bao giờ đứng ở bước 5.
- **FR9** — (F9) Mở lại dự án nháp đưa tới đúng bước server báo (tối đa là bước 6 — Code), không chặn trần ở 5.

## Phạm vi kỹ thuật dự kiến
- `services/web-gui/src/components/AuthoringModeBar.tsx` (+ `.module.css`)
- `services/web-gui/src/pages/{ValidatePage,ScriptStepPage,ResultPage,PublishPage}.tsx`
- `services/web-gui/src/utils/{pipelineLabels,flow}.ts`
- Test: `services/web-gui/tests/components/*`, `tests/pages/*` có kiểm các chuỗi chữ bị đổi
- Server (Q3-C): `services/{authoring-service,orchestrator}/internal/domain/flow.go` (+ `flow_test.go`), `tests/contracts/test_flow_steps_contract.py` (mới). Nếu làm F8: thêm một trường vào tóm tắt nội bộ authoring → orchestrator (thêm trường mới, bản cũ vẫn đọc được), sửa `docs/contracts/` nếu hợp đồng này có tài liệu ở đó.
- Không đổi schema DB và không đổi message RabbitMQ.
- Phải tuân `docs/ux-ui-design-rules.md` ở bước thiết kế

## Kiểm thử (dự kiến)
- `make check` (web-gui: vitest + tsc + lint)
- Rebuild và restart `web-gui`, xem trực tiếp: màn 3 đang chạy chuỗi AI (Manim và Remotion), màn 7 đang chạy, màn 8
