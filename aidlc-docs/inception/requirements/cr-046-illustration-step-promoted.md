# CR-046 — Đưa bước "Hình minh hoạ" thành bước 6 có số riêng (đảo ngược FR9 của CR-045)

## Date
2026-09-27

## Stage
Requirements Analysis — Creator (chủ dự án) đã yêu cầu và duyệt trực tiếp trong phiên làm việc.

## Bối cảnh
CR-045 (cùng ngày) tách bước Hình minh hoạ thành một tab riêng trong wizard soạn thảo (`/create/script/illustrations`), giữa Visual và Code, nhưng FR9 của CR-045 cố tình **không** cho nó một số bước riêng trong luồng 1–13 dùng chung giữa orchestrator/authoring-service/web-gui — để tránh phải đổi luồng dùng chung đó. Lượt chạy của bước này được ghi nhật ký dưới số bước của Code (`flow_step=5`), phân biệt bằng `source="illustrations"`. Trên thanh bước (StepRail) bên trái, bước này hoàn toàn không hiện — Creator chỉ vào được tab qua nút "Tiếp tục" ở Visual, nút "Quay lại" ở Code, chuỗi chạy AI, hoặc một ghi chú trên tab Code.

Sau khi dùng thử, Creator thấy cách làm này khiến bước Hình minh hoạ trông "vô hình": không có mục nào trên thanh bước bên trái đại diện cho nó, không đếm được là dự án đã đi tới đâu trong toàn bộ hành trình, dễ nhầm tưởng nó là một phần của bước Code. Creator yêu cầu thăng bước này thành một mục có số, hiện trên thanh bước, **dù biết rõ điều này đảo ngược quyết định đã duyệt ở FR9 của CR-045**.

## Quyết định của Creator
Đảo ngược FR9 của CR-045: bước Hình minh hoạ trở thành bước 6 có số riêng trong luồng dùng chung, hiện trên thanh bước như mọi bước khác — nhưng hiện ở trạng thái "Không dùng" (vô hiệu hoá, không bấm được) khi dự án không dùng Remotion (vì bước này chỉ áp dụng cho Remotion).

## Yêu cầu chức năng

- **FR1 — Luồng 13 bước thành luồng 14 bước.** Bước Hình minh hoạ nhận số 6. Mọi bước từ Validate trở đi lùi lại một số.
- **FR2 — Thanh bước bên trái (StepRail) hiện bước 6** với nhãn "Hình minh hoạ", cùng nhóm giai đoạn "Soạn" với Kịch bản/Visual/Code (theo đúng vị trí của nó trong wizard soạn thảo).
- **FR3 — Trạng thái "Không dùng".** Khi dự án không dùng Remotion (`renderEngine !== "remotion"`), bước 6 hiện trạng thái "skipped" với nhãn "Không dùng", không bấm được — giống cách bước "Cắt short" đã hiện "skipped" cho video không làm bản dọc.
- **FR4 — Nhật ký ghi thẳng số bước mới.** Lượt chạy của bước Hình minh hoạ ghi `flow_step=6` trực tiếp, không cần workaround `source="illustrations"` dưới `flow_step=5` của CR-045 nữa. Dữ liệu lịch sử (ghi trước CR-046, có `flow_step=5` và `source="illustrations"`) vẫn được đọc đúng — web-gui giữ khả năng nhận diện qua `source` cho dữ liệu cũ, song song với việc đọc `flow_step=6` cho dữ liệu mới.
- **FR5 — Điều hướng, route, trang tạo bản mới (fork), huỷ bước (cancel), lọc theo bước** trên web-gui đều cập nhật theo số bước mới; trang `IllustrationsStepPage` (`currentStep`) báo đúng số 6.
- **FR6 — Dữ liệu lịch sử trên server** (`project_events.flow_step`, `project_events.from_flow_step`) được dịch số (+1 cho các bước ≥ 6) qua một migration một lần, để nhật ký cũ không đọc sai bước sau khi đổi số.

## Bảng đánh số lại (áp dụng cho mọi bước ≥ 6)

| Bước | Số cũ (CR-045) | Số mới (CR-046) |
|---|---|---|
| Khởi tạo | 1 | 1 |
| Cấu hình | 2 | 2 |
| Kịch bản | 3 | 3 |
| Visual | 4 | 4 |
| Code | 5 | 5 |
| **Hình minh hoạ** | *(không có số)* | **6** |
| Validate | 6 | 7 |
| Review | 7 | 8 |
| TTS | 8 | 9 |
| Render | 9 | 10 |
| Merge | 10 | 11 |
| Cắt short | 11 | 12 |
| Kết quả | 12 | 13 |
| Publish | 13 | 14 |

## Đảo ngược so với CR-045
**Đây là đảo ngược trực tiếp FR9 của CR-045.** FR9 nói: "lượt chạy bước Hình minh hoạ ghi `source = "illustrations"` dưới số bước 5 (Code) — không thêm số bước mới vào luồng 1–13 dùng chung với orchestrator." CR-046 làm ngược lại: thêm số bước mới (6), đổi luồng dùng chung thành 14 bước. Toàn bộ các quyết định khác của CR-045 (FR1–FR8: tab riêng trong wizard, không giới hạn số hình, vẽ song song, tiến độ từng hình, xoá hình nháp, cảnh báo style dễ đọc) giữ nguyên, không đổi.

## Phạm vi kỹ thuật
- `orchestrator` và `authoring-service` (Go): hằng số `FlowIllustrations`/`FLOW_ILLUSTRATIONS` = 6, đánh số lại các hằng `Flow*` còn lại, migration dữ liệu `project_events`.
- `web-gui`: `utils/flow.ts` (FLOW_LABELS, các hằng FLOW_*, `stepStatus` nhận thêm `renderEngine` để quyết định "skipped", `authoringRoute`/`flowRoute`, `FLOW_PHASES`, `FORK_STEPS`), và mọi nơi đọc/so sánh số bước (StepRail, StatusStrip, ForkDialog, useStepNav, VideoListPage, JournalPage, AuthoringModeBar, IllustrationsStepPage, RenderPage).
- Không thêm service mới.

## Kiểm thử
- web-gui (vitest): cập nhật số bước cứng trong test của StepRail, AppShell, StatusStrip, JournalPage, VideoListPage cho khớp luồng 14 bước; `npm run build` (tsc + vite build) sạch.
- Go: (thực hiện bởi một phiên làm việc song song trên cùng nhánh, khác file) — xem `aidlc-docs/audit.md` cho chi tiết phần backend.

## User Confirmation
Creator đã xác nhận rõ ràng, biết đây là đảo ngược thiết kế đã duyệt của CR-045, và vẫn muốn thực hiện.
