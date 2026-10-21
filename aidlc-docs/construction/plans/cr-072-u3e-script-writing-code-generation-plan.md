# CR-072 U3e · Bước 6 Viết kịch bản — kế hoạch code

Nguồn duy nhất cho việc code: [thiết kế](cr-072-u3e-script-writing-design.md) (sau khi Creator duyệt và trả lời mục "Cần Creator quyết"). Tick `[x]` ngay khi xong từng task. Chuẩn: `docs/code-standards-rules.md`, `docs/ux-ui-design-rules.md`.

## authoring-service (Go)
- [x] **T1 Domain** `script.go`: kind `script_beat`, câu và mã câu, đọc câu trả lời AI, cờ câu dài/khó nghe, kiểm câu mở/kết, kiểm chỗ nối. Test.
- [x] **T2 Prompt** `prompt_seed_script.go` vai `script_writer` (viết, làm lại, gợi ý câu, gợi ý chỗ nối), đăng ký seed. Test seed.
- [x] **T3 Use case** `script_desk.go`: điều kiện vào, State, viết một beat, viết các beat còn lại, áp gợi ý, sửa câu. Test với AI giả.
- [x] **T4 UnitKind** `script_units.go`: Edit, Redo, Impact, CheckApprovable cho `script_beat`. Test.
- [x] **T5 Tác động** `Impact` của beat thêm "Lời thoại cần viết lại"; hộp xác nhận đổi thứ tự khi lời thoại đã duyệt. Test.
- [x] **T6 Đọc giọng** `voice_lines.go` + `adapters/postgres/voice_line_repository.go` + migration `voice_lines`: hàng chờ đọc, nhận kết quả, ước tính thời lượng từ giọng thật. Test.
- [x] **T7 HTTP + nối dây** `adapters/http/script.go` (gồm các route nội bộ của đọc giọng), kind AI, `main.go`; model AI dùng khoá `story` có sẵn. Test router.

## orchestrator (Go)
- [x] **T8** `DraftVoiceUseCase` (application/draft_voice.go): lệnh nội bộ → lấy câu chờ đọc → `synthesize_lines` → nhận `lines_synthesized` → gửi kết quả về authoring-service; khai báo exchange/queue, consumer. Test với cổng giả. `flow.go` không đụng.

## tts (Python)
- [x] **T9** `SynthesizeLines` + consumer `synthesize_lines` + producer `lines_synthesized`, file `/shared/{id}/audio/{line}_{cache_key}.wav`, `cache_hit`. Test với engine giả.

## api-gateway
- [x] **T10** route proxy `/v1/projects/:id/script...`; route `GET /v1/projects/:id/audio/:file` bằng `sharedFileHandler` + test (tên sai, ra ngoài thư mục, Range).

## Hợp đồng
- [x] **T11** `docs/contracts/script-v1.md`, `shared-artifacts.md`, dòng trong README; hợp đồng RabbitMQ `synthesize_lines`/`lines_synthesized` ở tts `interface-contracts.md`.

## web-gui
- [x] **T12 Client** `api/client.ts`: kiểu và hàm gọi; `PromptRole` thêm `script_writer`; `PromptSettingsPage` thêm mục.
- [x] **T13 Luồng** `utils/flow.ts` bước 6 `screen: "script"`; `App.tsx` route; "Sang viết kịch bản" của Beat sheet mở màn mới; "Sang rà soát và chốt" mở trang "chưa phát triển". Cập nhật test flow và `ProjectFlowContext`.
- [x] **T14 Hook** `hooks/useScript.ts` (state, thăm dò, nghe thử).
- [x] **T15 Màn** `pages/ScriptWriteStepPage.tsx` + `components/script/` theo mockup.
- [x] **T16 Test web-gui**; `tsc`, `eslint`, `vitest` sạch.

## Hoàn tất
- [x] **T17 Kiểm** `gofmt`, `go vet`, `go test -race ./...` (authoring, orchestrator; Postgres tạm), `pytest` tts, jest api-gateway, web-gui; dọn dữ liệu test.
- [x] **T18 Rebuild** `scripts/worktree.sh rebuild authoring-service orchestrator tts api-gateway web-gui`, xác nhận healthy; cập nhật `aidlc-state.md`, `audit.md`.
- [x] **T19 Báo cáo** nêu rõ việc chưa làm và phần chưa chạy AI/giọng thật.

## Ghi chú khi code

- Model AI của bước dùng khoá `story` có sẵn (không thêm cột `script_writer_model`); xem mục "Khác với thiết kế lúc duyệt" của thiết kế.
- T6 có test kho Postgres chạy thật trên một Postgres tạm (đã xoá sau khi chạy); T8 thêm cả định tuyến trong consumer: câu trả lời `lines_synthesized`/`lines_synthesis_failed` và lệnh `synthesize_lines` bị dead-letter không đi vào saga.
- Chưa làm (nói rõ): chưa chạy AI thật và chưa nghe giọng thật trên stack (local không có `HIVE_API_KEY`; đọc giọng Edge cần mạng ngoài) — đường AI và đường đọc chỉ kiểm bằng test giả lập; ô sửa dữ kiện từng câu trên màn; từ điển phát âm, "đọc là", tốc độ, khoảng nghỉ (U6); bộ bàn giao và chốt (U3f).
- Bước Kịch bản cũ (`flow.go`, màn `ScriptOutlineStepPage`) không còn đường vào từ menu; "Sang rà soát và chốt" mở trang "chưa phát triển" tới U3f, nên một video mới chưa đi tiếp được sau bước 6 cho tới khi U3f và các bước sau có.
- Ngoài phạm vi, đã có sẵn: `tests/contracts` có hai test đỏ không do CR này (`flow.go` hai bản khác nhau; `clip_output_path` của video-assembly).
