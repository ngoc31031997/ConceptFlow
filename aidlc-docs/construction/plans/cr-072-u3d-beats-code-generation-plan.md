# CR-072 U3d · Bước 5 Beat sheet — kế hoạch code

Nguồn duy nhất cho việc code: [thiết kế](cr-072-u3d-beats-design.md) (sau khi Creator duyệt). Tick `[x]` ngay khi xong từng task. Chuẩn: `docs/code-standards-rules.md`, `docs/ux-ui-design-rules.md`.

## authoring-service (Go)
- [x] **T1 Domain** `beat.go`: kind `beat` / `beat_sheet`, đọc câu trả lời AI (bảng, một beat), kiểm cấu trúc. Test.
- [x] **T2 Prompt** `prompt_seed_beat.go` vai `beat_sheet`, đăng ký seed. Test seed.
- [x] **T3 Use case** `beat_desk.go`: điều kiện vào, State, viết bảng, thêm/xoá/đổi thứ tự, gắn dữ kiện, đổi thời lượng/chức năng. Test với AI giả.
- [x] **T4 UnitKind** `beat_units.go`: Edit, Redo, Impact, CheckApprovable cho `beat`; cổng `beat_sheet`. Test.
- [x] **T5 Tác động** `Impact` của câu chuyện thêm "Beat sheet cần xem lại"; dữ kiện bị bỏ khỏi câu chuyện thì bỏ khỏi beat. Test.
- [x] **T6 HTTP + nối dây** `adapters/http/beat.go`, kind AI, `main.go`, cột `beat_sheet_model`. Test router.

## api-gateway
- [x] **T7** route proxy `/v1/projects/:id/beats...` + test.

## Hợp đồng
- [x] **T8** `docs/contracts/beats-v1.md` + dòng trong README.

## web-gui
- [x] **T9 Client** `api/client.ts`: kiểu và hàm gọi; `PromptRole` thêm `beat_sheet`; `PromptSettingsPage` thêm mục.
- [x] **T10 Luồng** `utils/flow.ts` bước 5 `screen: "beats"`; `App.tsx` route; nút "Sang beat sheet" của Câu chuyện mở màn mới; "Sang viết kịch bản" mở trang "chưa phát triển". Cập nhật test flow.
- [x] **T11 Hook** `hooks/useBeats.ts`.
- [x] **T12 Màn** `pages/BeatsStepPage.tsx` + `components/beats/` theo mockup.
- [x] **T13 Test web-gui**; `tsc`, `eslint`, `vitest` sạch.

## Hoàn tất
- [x] **T14 Kiểm** `gofmt`, `go vet`, `go test -race ./...` (Postgres tạm, dọn sau), npm test api-gateway và web-gui.
- [x] **T15 Rebuild** `scripts/worktree.sh rebuild authoring-service api-gateway web-gui`, xác nhận healthy; dọn dữ liệu test; cập nhật `aidlc-state.md`, `audit.md`.
- [x] **T16 Báo cáo** nêu rõ việc chưa làm và phần chưa chạy AI thật.

## Ghi chú khi code

- Cổng `beat_sheet` chỉ giữ thứ tự; không đăng ký làm kind của API đơn vị chung. Bước xong khi mọi beat đã duyệt (`status` của `GET /beats`).
- Sửa chức năng, thời lượng, dữ kiện đi qua `PUT /units/beat/{key}` (cùng đường với sửa chữ), nên không có route PATCH riêng.
- Dữ kiện gắn vào beat chọn trong mọi dữ kiện đang giữ (như mockup), không giới hạn ở dữ kiện câu chuyện đang dùng.
- Menu bước: câu chuyện và beat sheet có trạng thái "xong" theo duyệt thật (`ProjectFlowContext` đọc `GET /story`, `GET /beats`).
- "Sang viết kịch bản" mở `coming/6`, chuyển sang xem trước bước Kịch bản đang chạy vì bước 6 có bước chạy sẵn.
- Chưa làm: tác động của beat lên lời thoại và ghi chú bàn giao; hộp xác nhận đổi thứ tự khi lời thoại đã duyệt; `stale` của beat khi sửa ô khung câu chuyện (chỉ đổi cổng câu chuyện mới đánh dấu); AI thật chưa chạy.
