# CR-072-U3a — Ý tưởng · creative brief, khung sửa theo đơn vị

Trạng thái: **đang code theo yêu cầu của Creator, chưa duyệt thiết kế riêng** (2026-10-17). Nhánh `feature/cr-072-u3a-idea-brief`, từ main `e113794`.

## Yêu cầu gốc (nguyên văn)

> "bỏ hết các quyết định trước đó cứ implment bước Tiền kỳ đã chốt trước" (Creator, 2026-10-17)

Hiểu là: bỏ thứ tự "U2a trước" và các vòng chờ duyệt thiết kế, task; làm ngay các bước Tiền kỳ đã chốt trên mockup, theo thứ tự bước. Giữ yêu cầu FR-27..FR-32, ADR-0042..0044 và giao diện mockup bản 9 vì đó là định nghĩa của các màn đã chốt. U2a tạm dừng (thiết kế vòng 2 vẫn trong worktree của nó). Mỗi CR con vẫn một nhánh, một worktree; chưa commit tới khi Creator duyệt.

Phạm vi: `unit-of-work.md` mục U3a; FR-27, FR-17 (phần khung), NFR-2 (đếm lần làm lại).

## Hiện trạng

- Bước 1 (`services/web-gui/src/pages/ScriptStepPage.tsx`) chỉ có ô chủ đề; "Tiếp tục" gọi `POST /v1/projects` (orchestrator `CreateProjectDraftUseCase`) rồi sang bước 2. Ngôn ngữ nội dung và dạng video (dài / Shorts) chọn ở bước 2 (`ScriptAuthoringSettingsStepPage.tsx`, `ContentLanguagePicker`, `VideoOutputModePicker`).
- authoring-service giữ `project_authoring` (topic, story, storyboard, code, mode, models); prompt theo vai ở bảng `prompts`, seed khi khởi động; gọi AI qua `LLMProviderPort`, ghi `llm_usage`, theo dõi tiến độ bằng `X-Operation-Id` + `/v1/operations/{id}`.
- Vai Story Architect (`story_architect`) viết dàn ý từ `{{topic}}`; chưa có brief.
- Chưa có đơn vị duyệt nào; mỗi bước duyệt cả bước.

## Quyết định

1. **Khung đơn vị** (ADR-0035) ở authoring-service: bảng `review_units` (project, kind, key, status `pending|approved`, nội dung JSON, các ô Creator đã viết, số lần làm lại). Trạng thái "AI đang làm lại" chỉ sống ở màn hình trong lúc request mở; `stale` thêm ở U3b cùng `DependencyTracker`. Một registry loại đơn vị; HTTP chung `/v1/projects/{id}/units/{kind}/{key}` (đọc, sửa tay, làm lại kèm ghi chú, duyệt, bỏ duyệt, ảnh hưởng). Loại đầu tiên: `brief`, key `main`.
   - Phần theo dõi phụ thuộc giữa các đơn vị (`DependencyTracker`) chưa làm ở U3a vì chưa có loại đơn vị nào phụ thuộc vào brief; U3b thêm khi `research_question` dựng từ brief. Trong U3a, phần phụ thuộc duy nhất là bước Kịch bản hiện có: lúc lưu dàn ý, server ghi dấu (md5) của brief đã duyệt; brief đổi thì màn Kịch bản báo "viết theo brief cũ".
2. **Brief do AI viết** bằng vai mới `creative_brief` (prompt seed trong DB, sửa được ở Cài đặt prompt), trả JSON bảy ô: câu hỏi trung tâm, khán giả, biết, hiểu, cảm thấy, giọng điệu, ngoài phạm vi. Ô Creator đã sửa được giữ nguyên khi làm lại (gửi kèm prompt và ghi đè lại sau khi AI trả). AI trả thiếu ô thì lượt đó lỗi, không lưu nửa vời. LLM chưa cấu hình thì Creator tự điền ô và duyệt.
3. **Thời lượng mục tiêu** lưu ở `project_authoring.target_duration` (`3-4`, `4-5`, `5-7`, `7-10` phút; Shorts luôn "dưới 60 giây"). Chọn trước khi tạo project; sau đó chỉ đọc, như đề tài (mockup: "Đổi đề tài thì dùng Làm lại kèm ghi chú ở brief").
4. **Màn Ý tưởng** thay màn bước 1: ① Đề tài (chủ đề, dạng video, ngôn ngữ nội dung, thời lượng mục tiêu) + "Tạo project và viết brief"; ② Creative brief với thẻ đơn vị (Sửa tay, Làm lại kèm ghi chú, Duyệt brief / Bỏ duyệt). "Sang cấu hình" chỉ bật khi brief đã duyệt. Ngôn ngữ nội dung và dạng video rời bước 2. Ô Ngôn ngữ hình do U1 thêm; chọn kiểu video vẫn ở bước 2 tới U3c.
5. **Story Architect đọc brief đã duyệt** qua biến `{{creative_brief}}` (seed `story_architect` thêm một mục). Prompt Creator tự viết không có biến này thì brief không vào prompt: màn Kịch bản không đổi gì, nhưng `GET /v1/projects/{id}/brief` báo `story_prompt_reads_brief=false` để màn báo rõ.
6. **Sửa tay brief đã duyệt** giữ trạng thái đã duyệt (Creator tự viết); **làm lại** đưa về chờ duyệt. Xoá trắng một ô thì ô đó trả lại cho AI (không còn là "Bạn viết"), lần làm lại sau AI viết ô đó; brief đã duyệt mà có ô trống thì về chờ duyệt. Brief còn ô trống không duyệt được; lỗi gọi tên ô bằng nhãn trên màn.
   - Các ô brief hiện dạng chữ, sửa trong modal "Sửa tay" (luật giao diện 4). Mockup bản 9 vẽ bảy ô là textarea sửa thẳng trên trang; làm theo luật 4 thay vì mockup ở điểm này.
   - Bố cục theo mockup (Creator, 2026-10-18): tiêu đề "Ý tưởng · creative brief", dòng "Bước này:", dải trạng thái của bước; thẻ "① Đề tài" có bốn ô đánh số; thẻ "② Creative brief" với chip trạng thái; "Sang cấu hình →" ở cuối nội dung, không còn thanh cố định ở đáy màn. Popup hay không theo luật thiết kế (Creator: "popup hay không thuộc vào rule designer").
   - Sau khi tạo project, các ô của ① khoá lại (chủ đề, dạng video, ngôn ngữ). Thời lượng còn đổi được tới khi có brief (project cũ chưa có brief, hoặc lượt AI đầu lỗi). Cảnh báo chủ đề trùng chuyển từ bước 2 sang ① ngay sau khi tạo.
   - Brief chạy trên model đã chọn cho bước Kịch bản (cùng loại việc viết). Lúc tạo project, màn lưu bộ model lần trước Creator dùng lên project; bước 2 vẫn đổi được. Model `ollama` đưa lượt chạy về Ollama local (dùng cho test). Cả hai, khi Kịch bản đã có, hiện trong modal danh sách ảnh hưởng do server trả (luật giao diện 5); nút màu cam khi có phần phụ thuộc, tím/vàng khi không (luật 6).
7. **Modal dùng chung**: tách vỏ hộp thoại từ `ConfirmModal` (portal, giữ focus, Esc, khoá cuộn) thành `components/ui/Dialog.tsx` theo luật 4; thêm biến thể nút `impact`, `run`, `view` và token `--impact`, `--impact-ink`, `--danger-ink`, `--run-ink` theo luật 6.
   - `ConfirmModal` dùng lại vỏ này (thêm nút ×, mờ dần khi đóng). Nút đỏ dùng `--danger-ink`.
   - Thanh chủ đề ở bước 2, 3 đổi "Sửa" thành "Xem ở Ý tưởng" (màu xem), vì đề tài không còn sửa sau khi tạo project.
8. Sửa brief chỉ khi project còn sửa được (`checkAuthoringUnlocked`, như các bước soạn khác).
9. **Màn Kịch bản** gửi `project_id` khi render prompt (đường chép tay), và hiện ba lời nhắc: chưa có brief đã duyệt; dàn ý viết theo brief cũ; prompt Story Architect đang bật không có `{{creative_brief}}`.

## Khung chung theo mockup (phần web-gui của U2a, gộp vào nhánh này)

Creator, 2026-10-18: "làm phần này luôn đi bro" (menu trái, thanh bước), "và cả phần menu bên trong sidebar menu bước nữa". Làm theo thiết kế U2a vòng 2, trừ script xoá dữ liệu cũ (vẫn ở U2a):
- 13 bước hiện có xếp vào bốn nhóm của mockup, mỗi nhóm một màu: Tiền kỳ · Nội dung (1–3), Tiền kỳ · Hình ảnh (4), Sản xuất (5–8), Hậu kỳ (9–13). Tám bước chưa có của luồng 21 bước không hiện; CR con nào làm bước đó thì chèn vào nhóm của nó.
- Tên bước theo mockup ở web-gui và hai file `flow.go`: Storyboard, Validate, Review, Render, Merge + QC, Publish; test hợp đồng luồng xanh.
- Thanh bước như mockup: các nhóm luôn mở, số trong vòng tròn, ◆ ở bước Creator duyệt (1, 3, 4, 8), bước đang xem tô vàng.
- Mọi màn: tiêu đề là tên bước trần (dòng trên đã ghi "nhóm · Bước n/13") và dòng "Bước này:" lấy từ mô tả của bước.
- Menu trái: "▶ Tiếp tục" kèm dòng "nhóm · bước", "Bản đồ luồng" (trang mới `/flow`), nhóm "Cài đặt". Ba mục "mới" của mockup (Ngôn ngữ hình, Thư viện nhân vật, Từ điển phát âm) thêm khi CR con làm trang của chúng.

## Chọn model AI (Creator, 2026-10-18)

"toàn bộ những phần nào có call AI đều phải cho lựa chọn hoặc set model default ngay lúc chọn đó", rồi "chọn model đi theo project và trong setting cũng cho setting model default":
- Mỗi project giữ model của từng chỗ gọi AI (`brief`, `publish`, `story`, `storyboard`, `code`); chọn ở nút AI là lưu lên project ngay.
- Model mặc định của app đặt ở Cài đặt · Model AI mặc định (`GET/PUT /v1/settings/ai-model`, bảng `app_settings`). Chỗ nào để trống thì chạy bằng mặc định này, không thì mặc định của máy chủ (`HIVE_MODEL`). Chỉ model dùng được cho mọi bước (kể cả code và vẽ hình) được chọn làm mặc định; model Ollama chọn riêng tại từng nút.
- Project mới bắt đầu để trống (không mang theo model lần trước), nên luôn theo mặc định cho tới khi Creator đổi.
- Gợi ý metadata ở Publish nhận `model` (llm-service `POST /v1/suggest-metadata`), vẽ và vẽ lại hình thư viện gửi `model` (nhớ theo trình duyệt vì không thuộc project nào).

## API (authoring-service, qua api-gateway)

- `GET /v1/projects/{id}/brief` → `{unit, target_duration, story_stale, story_prompt_reads_brief}`.
- `PUT /v1/projects/{id}/brief/target-duration` `{target_duration}`.
- `POST /v1/projects/{id}/brief/generate` `{note?}` (theo dõi bằng `X-Operation-Id`) → đơn vị brief.
- `GET /v1/projects/{id}/units/{kind}/{key}`, `PUT` (sửa tay `{fields}`), `POST …/redo` `{note}`, `POST …/approve`, `POST …/unapprove`, `GET …/impact` → `{items: [string]}`.

## Kiểm tra

- Go: `go test ./...` authoring-service; web-gui: `npx vitest run`, `npx tsc --noEmit`, `npx eslint`; api-gateway: `npm test`.
- Build lại: authoring-service, api-gateway, web-gui.
- Trên app: tạo project ở bước 1, brief hiện đủ bảy ô; sửa một ô rồi làm lại kèm ghi chú, ô đó giữ nguyên; duyệt mới sang Cấu hình; viết dàn ý ở Kịch bản, quay lại sửa brief thì modal báo Kịch bản sẽ cần viết lại và màn Kịch bản hiện báo "viết theo brief cũ".

## Rủi ro

- Prompt `story_architect` Creator đã sửa và đang bật sẽ không có `{{creative_brief}}`; màn báo rõ thay vì im lặng.
- Ngôn ngữ và dạng video không còn đổi được ở bước 2 sau khi tạo project; muốn đổi thì tạo video mới.
