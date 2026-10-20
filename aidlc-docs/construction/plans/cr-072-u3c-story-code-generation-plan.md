# CR-072 U3c · Bước 4 Câu chuyện — kế hoạch code (Code Generation, Part 1)

Nguồn duy nhất cho việc code: [thiết kế](cr-072-u3c-story-design.md). Tick `[x]` ngay khi xong từng task. Chuẩn code: `docs/code-standards-rules.md`; giao diện: `docs/ux-ui-design-rules.md`. Quy ước kỹ thuật đã xác nhận: `/v1`, tiêm bằng hàm khởi tạo, `X-Operation-Id`.

Phạm vi: màn Câu chuyện theo mockup. Không đụng orchestrator, `flow.go`, bước Kịch bản đang chạy.

## authoring-service (Go)

- [x] **T1 Domain** `domain/story.go`: kind `story` / `story_field` / `story_approach`; kiểu ô (`text`, `facts[]`), góc tiếp cận, cổng (`approach_key`, `used_facts`); dạng gõ tay `câu [F01, F02]`; đọc câu trả lời AI của khung, của một ô, của danh sách góc (mã lạ, thiếu ô, JSON hỏng là lỗi, không lưu). Test: `story_test.go`.
- [x] **T2 Prompt** `domain/prompt_seed_story.go` + hai vai `story_approaches`, `story_frame` (viết một ô riêng cũng dùng `story_frame` với ghi chú ô), đăng ký trong `prompt_template.go` và `prompt_template_seeds.go`; không render bằng bộ render chung. Test seed như `prompt_seed_research_test.go`.
- [x] **T3 Use case** `application/story_desk.go`: điều kiện vào (brief đã duyệt; Nghiên cứu đã duyệt hoặc bỏ qua), `State`, gợi ý góc (giữ góc đang chọn), chọn góc, lưu kiểu mới vào bảng Kiểu video, viết khung sáu ô (giữ ô Creator; khi bỏ qua Nghiên cứu mà chưa có dữ kiện thì tạo `fact` mang `link=none`), bật/tắt dữ kiện dùng (bắt buộc không tắt được). Ghi `input_ref=brief:main` và sha. Test với AI giả.
- [x] **T4 UnitKind** `application/story_fields.go`: `story_field` (Edit nhận `text`, kiểm mã dữ kiện; Redo viết riêng ô đó; Impact; CheckApprovable) và cổng `story:main` (Approve cần đủ sáu ô; sửa/làm lại một ô đưa cổng về chờ duyệt). Test.
- [x] **T5 Tác động** `Impact` của brief thêm "khung câu chuyện cần xem lại"; người tìm dữ kiện (`RegisterFactDependents`) nêu ô câu chuyện đang trích dữ kiện. Test.
- [x] **T6 HTTP + nối dây** `adapters/http/story.go` (6 route trong thiết kế, theo dõi lượt AI kind `story_approach` / `story`), đăng ký kind trong `cmd/authoring/main.go`. Test router.

## api-gateway

- [ ] **T7** route proxy các đường `/v1/projects/:id/story...` (lượt AI qua `authoringAi`) + test `authoringRouting`.

## Hợp đồng

- [ ] **T8** `docs/contracts/story-v1.md` + dòng trong `docs/contracts/README.md`.

## web-gui

- [x] **T9 Client** `api/client.ts`: kiểu và hàm gọi (state, gợi ý góc, chọn góc, lưu kiểu, viết khung, dữ kiện dùng); `PromptRole` thêm hai vai; `PromptSettingsPage` thêm hai mục.
- [x] **T10 Luồng** `utils/flow.ts`: bước 4 có `screen: "story"`, `storyRoute`; `App.tsx` thêm route `/projects/:id/story`; nút "Sang xây dựng câu chuyện" của Nghiên cứu mở màn mới; nút "Sang beat sheet" mở trang "chưa phát triển". Cập nhật test của flow, StepRail, useStepNav.
- [x] **T11 Hook** `hooks/useStory.ts`: tải state, chạy AI (theo dõi và huỷ như `useResearch`), chọn góc, dữ kiện dùng, duyệt.
- [x] **T12 Màn** `pages/StoryStepPage.tsx` + `components/story/` (thẻ ① Góc tiếp cận, ② Khung câu chuyện với sáu ô, ③ Dữ kiện dùng, thẻ Duyệt, xác nhận đổi góc và tắt dữ kiện đang trích); Sửa tay / Làm lại từng ô qua `UnitEditModal` / `UnitRedoModal`; "Xem tất cả kiểu" mở `/settings/video-archetypes` trong tab mới. Theo mockup và luật màu nút.
- [x] **T13 Test web-gui**: màn, hook, hộp thoại; `tsc`, `eslint`, `vitest` sạch.

## Hoàn tất

- [x] **T14 Kiểm** `gofmt`, `go vet`, `go test -race ./...` (Postgres tạm, dọn sau), `npm test` api-gateway, web-gui như T13.
- [x] **T15 Rebuild** `scripts/worktree.sh rebuild authoring-service api-gateway web-gui`, xác nhận healthy; dọn dữ liệu test; cập nhật `aidlc-state.md`, `audit.md`, dashboard tiến độ.
- [x] **T16 Báo cáo** nêu rõ: chưa chạy AI thật nếu thiếu `HIVE_API_KEY`; việc nào chưa làm.

## Bổ sung (Creator chọn, 2026-10-20)

Model AI riêng cho bước: cột `story_frame_model`, khoá `story_frame` ở API model theo project, `AiModelSelect` ở thẻ ①, hộp thoại làm lại ô và làm lại cả khung.

## Chưa làm trong phạm vi này

Beat sheet và các bước sau; tác động của câu chuyện lên beat/lời thoại; orchestrator và bước Kịch bản hiện có đọc câu chuyện.
