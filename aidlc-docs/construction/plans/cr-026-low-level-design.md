# CR-026 — Low-Level Design: kịch bản riêng cho bản Shorts/TikTok, hỗ trợ bằng AI

## Date
2026-09-12

## Stage
Low-Level Design (Functional/NFR/Infrastructure Design: SKIP — không thêm hạ
tầng mới, không service mới; lý do skip ghi ở D3)

---

## Quyết định kiến trúc

### D1 — Liên kết hai project bằng một field tự-trỏ, gán 2 chiều, best-effort
`Project.CompanionProjectID *string` (DB: `companion_project_id TEXT NULL`,
tự tham chiếu `projects.project_id`, không FK cứng — cùng lý do
`intro_asset_id` không FK sang bảng khác service: hai project độc lập vòng đời,
xoá một cái không được kéo lỗi sang cái kia).

Gán lúc `StartRenderSagaUseCase.Execute`: request mang thêm
`companion_project_id` (optional). Nếu có, sau khi lưu project mới, load
project kia và set `CompanionProjectID` của NÓ trỏ ngược lại project mới —
best-effort (log lỗi, không fail cả saga) giống cách `resolveChannelAssets`
đã xử lý "không có gì để gắn" ở CR-023: một video chính vẫn quý hơn một liên
kết hiển thị.

### D2 — Không có "tạo 2 project cùng lúc" ở backend — chỉ một field tái dùng
FR73.1 nhắc 2 kịch bản (tạo từ Result của video dài có sẵn / chọn "cả hai
loại" ngay từ đầu). Cả hai đi qua ĐÚNG MỘT cơ chế: gọi `POST /v1/sagas/render`
bình thường (đã có) kèm `companion_project_id` trỏ về project kia — không cần
endpoint "tạo cặp" riêng. "Chọn cả hai loại ngay từ đầu" ở tầng UI chỉ là:
submit project dài trước (như hiện nay), sau đó ngay tại RenderPage/ResultPage
của nó mở luôn UI tạo bản ngắn (D5) — không phải hai form nộp cùng lúc. Quyết
định này thu hẹp phạm vi: một điểm vào duy nhất (Result page của project dài)
cho cả 2 kịch bản, thay vì xây thêm một bước wizard soạn 2 script song song ở
Bước 1 — chưa có nhu cầu thật nào đòi hỏi phải làm cùng lúc, và tách làm hai
lượt không mất gì (project dài vẫn render bình thường trong lúc soạn bản ngắn).

### D3 — FR71.2 (lint trước khi trả) rút xuống: không thêm HTTP endpoint mới ở `rendering`
`rendering` hiện là **thuần AMQP consumer, không có REST endpoint** (comment
gốc trong `main.py`). Thêm một HTTP server chỉ để lint-only là hạ tầng mới
(SKIP header ở trên) cho một lượt kiểm tra không tốn kém gì nếu bỏ qua: FR71.3
đã bắt buộc Creator xem/sửa bản nháp AI trước khi nộp, và khi nộp thật, saga
đi qua `validate_script` — đúng lượt kiểm tra thật (dry-run Manim, không phải
đoán bằng regex) mà FR71.2 muốn có, chỉ là chạy **sau khi nộp** thay vì **trước
khi hiện cho Creator xem**. Bug 1 (2026-09-12, phiên trước) đã đảm bảo lỗi ở
`validate_script` trả đúng về Creator kèm traceback đầy đủ, không còn lọt qua
im lặng — lưới an toàn đã có sẵn, không cần lưới thứ hai. FR71.2 coi như đã
thoả mãn theo hướng này, không implement lint riêng.

### D4 — Sinh script ngắn bằng Ollama: method mới trên `OllamaClient` đã có
`SuggestShortScript(ctx, topic, language) (script string, err error)` cạnh
`Suggest` (metadata) hiện có — cùng `httpClient`, cùng `model`, cùng cách gọi
`/api/generate`. Khác biệt: `format: "json"` KHÔNG dùng ở đây (script là code
nhiều dòng, ép JSON sẽ buộc model escape newline, dễ vỡ) — gọi thường, prompt
tự yêu cầu "chỉ trả lời bằng đúng một khối code Python", rồi tự
`stripMarkdownCodeFence`-tương-đương ở phía Go (dùng lại đúng logic gỡ fence,
viết lại ngắn gọn bằng Go vì đây là service khác — không import được TS).

Endpoint mới: `POST /v1/short-script-suggestions` — không cần `project_id` có
sẵn (Creator có thể tạo bản ngắn cho một chủ đề mới toanh, không nhất thiết
xuất phát từ video dài). Input: `{topic, language, source_script_content?}` —
`source_script_content` optional, khi có (gọi từ Result page của video dài)
dùng làm ngữ cảnh chủ đề thay vì bắt Creator gõ lại topic.

### D5 — UI: một component `ShortScriptAssistant` dùng lại được ở 2 chỗ
Tái cấu trúc tối thiểu: `ScriptAssistant` hiện tại có 2 prop cần thiết
(`buildPromptFn`, `title`) để dùng chung khung sườn (topic input, nút Copy
prompt, ô dán kết quả) cho cả script dài (hiện có) và script ngắn (mới) —
không viết lại từ đầu. Điểm khác của bản ngắn: thêm nút "Soạn bằng AI nội bộ"
gọi thẳng D4, đổ kết quả vào ô dán sẵn (Creator vẫn sửa được trước khi nộp,
FR71.3).

Nơi dùng: mục mới trên `ResultPage` — "Tạo bản Shorts/TikTok riêng cho video
này" (thu gọn mặc định, giống khối "Render lại"). Nộp qua `startRenderSaga`
với `video_output_mode: "short"`, `companion_project_id: projectId`,
`script_content` lấy từ trợ lý này thay vì `project.script_content`.

### D6 — Hiện project liên kết ở Result page: card tóm tắt, không nhân đôi UI đăng bài
`GET /v1/projects/{id}` trả thêm `companion_project_id` (chỉ id, không nhúng
object project kia — FR73.2 thu hẹp: web-gui tự `useProject(companionId)` gọi
lại đúng hook đã có, không cần orchestrator embed lồng nhau). `ResultPage` khi
có `companion_project_id`: fetch thêm, hiện một `CompanionProjectCard` — video
preview/ClipsPanel rút gọn + trạng thái + link "Xem đầy đủ" sang
`/projects/{companion_id}/result`. Không nhân đôi toàn bộ `PublishForm`/
`YoutubeChannels` lên cùng trang — muốn đăng/tải clip đầy đủ thì bấm sang
trang riêng của nó (mỗi project vẫn có trang đầy đủ của mình, y như trước
CR-026).

---

## Việc cần làm

**orchestrator (Go)**
1. `domain.Project.CompanionProjectID *string`; DB migration
   `companion_project_id TEXT NULL`.
2. `StartRenderSagaInput.CompanionProjectID *string`; `Execute` set + best-effort
   gán ngược (load/save project kia, log lỗi không fail).
3. `startRenderSagaRequest`/`projectResponse` DTO thêm field.
4. `OllamaClient.SuggestShortScript` + prompt builder Go (mirror D4).
5. Endpoint `POST /v1/short-script-suggestions` + use case
   `SuggestShortScriptUseCase`.
6. Test: gán liên kết 2 chiều, best-effort khi project kia không tồn tại,
   suggestion use case (fake Ollama client).

**web-gui (TS/React)**
1. `scriptPrompts.ts`: `buildShortScriptPrompt`.
2. `ScriptAssistant` nhận `buildPromptFn`/`title`/`extraAction` (nút AI nội bộ)
   qua prop thay vì hard-code, tái dùng cho cả 2 nơi.
3. `api/client.ts`: `suggestShortScript()`, `startRenderSaga` input thêm
   `video_output_mode`/`companion_project_id` (đã có `video_output_mode` từ CR
   trước, thêm `companion_project_id`).
4. `types/index.ts`: `Project.companion_project_id?`.
5. `ResultPage`: mục "Tạo bản Shorts/TikTok riêng" (thu gọn) + `CompanionProjectCard`
   khi `companion_project_id` có giá trị.
6. Test cho từng phần trên (component + integration nhẹ ở ResultPage).

**Không đụng**: `rendering`, `video-assembly`, `generate_clips`, saga steps hiện có.
