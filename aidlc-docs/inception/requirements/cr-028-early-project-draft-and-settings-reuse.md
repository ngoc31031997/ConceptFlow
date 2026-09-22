# CR-028 — Tạo project ngay từ bước nhập chủ đề, rollback tự do trong khi soạn, cảnh báo trùng chủ đề, tái dùng cấu hình lần trước (P1)

## Date
2026-09-22

## Stage
Requirements Analysis (Change Request)

## Nguồn gốc
Ba ghi chú Creator để lại trực tiếp trên `docs/review/data-flow-review.md` (review giai đoạn A,
mục "Rủi ro: authoring/* lưu theo project_id..."), được gộp thành một CR vì cả ba cùng chạm một
chỗ: thời điểm project thật sự "sinh ra".

## Intent Analysis
- **Request type**: New Feature + bugfix rủi ro dữ liệu mồ côi đã nêu trong review.
- **Scope estimate**: 2 unit — `orchestrator` (project được tạo sớm hơn, khoá/mở khoá sửa theo
  trạng thái, endpoint cảnh báo trùng chủ đề), `web-gui` (gọi tạo project ngay ở bước 1 thay vì
  sinh UUID client-side, banner cảnh báo trùng chủ đề, prefill cấu hình lần trước).
- **Complexity estimate**: Moderate. Không đổi Saga/kiến trúc render — chỉ đổi thời điểm hàng
  `projects` xuất hiện và nới lỏng ràng buộc sửa `project_authoring`.

## Quyết định đã chốt cùng Creator (2026-09-22)

| Chủ đề | Quyết định |
|---|---|
| Phạm vi so trùng chủ đề | So trong cùng `content_language`; chỉ **cảnh báo**, không chặn tạo |
| Rollback/sửa bước đã qua | Tự do sửa bất kỳ bước nào **trước khi saga render bắt đầu**; sau khi đã bắt đầu render, khoá toàn bộ authoring |
| Cấu hình tái dùng | Một bộ "lần cuối dùng" duy nhất, không phải nhiều preset đặt tên |

## Bối cảnh — vì sao gộp ba ý thành một

Rủi ro gốc (review Phần A): `project_authoring` lưu theo `project_id`, nhưng hàng `projects`
chỉ thật sự được `INSERT` bên trong `StartRenderSaga.Execute` (`POST /v1/sagas/render`,
[router.go:288](services/orchestrator/internal/adapters/http/router.go#L288)). Từ lúc Creator
gõ chủ đề (bước 1) tới lúc bấm "Xác nhận" (bước 7), `project_id` chỉ tồn tại trong
`localStorage` của trình duyệt — [ProjectDraftContext.tsx](services/web-gui/src/context/ProjectDraftContext.tsx).
Nếu Creator bỏ ngang, `project_authoring` (topic, story, storyboard, code, review) và
`llm_usage` đã ghi vẫn còn đó, nhưng **không có hàng `projects` nào tham chiếu tới** — không
liệt kê được, không dọn được, và tốn tiền LLM đã chi không có gì để hiện lên GUI.

Sửa gốc: tạo hàng `projects` ngay khi Creator gõ xong chủ đề (bước 1), không đợi tới
`sagas/render`. Điều này giải quyết luôn cả ba ý Creator nêu:

1. **Cảnh báo trùng chủ đề** — chỉ so được với chủ đề nếu nó nằm trong một bảng có thể query
   theo mọi project, không phải rải rác trong `localStorage` của từng trình duyệt.
2. **Rollback về mọi bước** — chỉ có ý nghĩa nếu có một `project_id` ổn định xuyên suốt từ bước 1,
   không phải một chuỗi state cục bộ bị mất khi F5.
3. **Project mồ côi** — không còn tồn tại theo định nghĩa, vì `projects` không còn "sinh ra
   muộn" nữa. Dự án dở dang vẫn hiện là một hàng `status='draft'` liệt kê được — cần thêm dọn
   dẹp định kỳ (FR83.5) chứ không còn là dữ liệu vô hình.

## Functional Requirements

### FR83 — Tạo project ngay ở bước nhập chủ đề

- **FR83.1**: Thêm `POST /v1/projects` (orchestrator) — nhận `{topic, content_language}`, tạo
  một hàng `projects` mới với `status='draft'`, `manim_scene_class_name=''`,
  `script_content=''` (script thật chưa có, sinh dần qua các bước sau), trả về `project_id` do
  **server** sinh (không phải client UUID như hiện tại — bỏ cách sinh UUID trong
  `ProjectDraftContext`).
- **FR83.2**: Bước 1 (`ScriptStepPage`, `/`) gọi `POST /v1/projects` ngay khi Creator rời khỏi
  ô nhập chủ đề (blur) hoặc bấm "Tiếp tục" lần đầu — không phải mỗi lần gõ phím. Nếu đã có
  `project_id` cho phiên hiện tại (đã tạo trước đó, Creator quay lại chỉnh chủ đề), gọi
  `PATCH /v1/projects/{id}/topic` thay vì tạo mới.
- **FR83.3**: `project_authoring.project_id` nhận foreign key `REFERENCES projects(id)
  ON DELETE CASCADE` (hiện đang cố ý không có FK vì project có thể chưa tồn tại — nay luôn tồn
  tại trước, FK áp được). Xoá project kéo theo xoá authoring state, không còn mồ côi ngược lại.
- **FR83.4**: `llm_usage.project_id` **giữ nguyên nullable, không FK** — comment hiện tại vẫn
  đúng cho `suggest-short-script` (CR-026 FR71.1), chạy trước khi có project nào.
- **FR83.5**: Không tự động dọn dẹp draft hết hạn trong CR này (quyết định 2026-09-22, Creator
  từ chối). Vì draft nay là hàng `projects` thật (`status='draft'`), nó đã liệt kê được qua
  `GET /v1/projects` và xoá được qua `DELETE /v1/projects/{id}`
  ([router.go:436](services/orchestrator/internal/adapters/http/router.go#L436)) — Creator tự
  dọn thủ công (đơn lẻ hoặc multi-delete nếu Web GUI đã/sẽ có) khi thấy cần, không cần job nền.

### FR84 — Sửa tự do trong khi soạn, khoá khi đã render

- **FR84.1**: Mọi endpoint `POST /v1/projects/{id}/authoring/*` (story/storyboard/code/review)
  cho phép ghi đè **không giới hạn số lần** khi `projects.status = 'draft'` — đúng yêu cầu "tuỳ ý
  chỉnh sửa". Không tự xoá các bước phía sau khi một bước trước đó bị sửa (không cần trường
  "stale" — vì tất cả các bước đều còn ở dạng nháp, Creator tự nhìn thấy tại tab tương ứng và tự
  quyết có sinh lại phía sau không).
- **FR84.2**: Một khi `POST /v1/sagas/render` đã chuyển `status` ra khỏi `'draft'` (ví dụ sang
  `parsing_script`), mọi `POST /v1/projects/{id}/authoring/*` tiếp theo trả `409 Conflict`
  ("authoring is locked once render has started"). `GET /v1/projects/{id}/authoring` vẫn đọc
  được (chỉ khoá ghi, không khoá đọc) để GUI hiện lại lịch sử đã soạn.
- **FR84.3**: Lưu lịch sử mỗi lần ghi đè một bước authoring (bảng mới
  `project_authoring_history`, append-only: `project_id, field_name, content, saved_at`) — để
  Creator có thể xem lại/khôi phục một bản nháp cũ của cùng một bước nếu ghi đè nhầm. GET một
  endpoint mới `GET /v1/projects/{id}/authoring/history?field=story_content` liệt kê các bản cũ.
- **FR84.4**: Web GUI bỏ điều kiện hiện tại chặn quay lại sửa bước trước (nếu có) khi
  `hasSubmitted=false`; thêm điều kiện mới: nếu `project.status !== 'draft'`, mọi tab authoring
  chuyển sang chế độ chỉ đọc kèm banner "Đã bắt đầu render — không sửa được nữa".

### FR85 — Cảnh báo trùng chủ đề

- **FR85.1**: `POST /v1/projects` (FR83.1) và `PATCH /v1/projects/{id}/topic` (FR83.2) trả thêm
  trường `similar_projects: []` trong response — danh sách project khác (khác `id`, cùng
  `content_language`) có `project_authoring.topic` khớp sau khi chuẩn hoá: lowercase, trim,
  gộp khoảng trắng liên tiếp. So khớp **chuỗi chuẩn hoá bằng nhau tuyệt đối** ở bản đầu tiên
  (không fuzzy match) — đơn giản, không dương tính giả; fuzzy/similarity matching để P2 nếu sau
  này thấy cần.
- **FR85.2**: Mỗi phần tử `similar_projects` gồm `{project_id, topic, status, created_at}` —
  đủ để GUI hiện "Chủ đề này trùng với project X (tạo lúc ...), trạng thái: ...".
- **FR85.3**: Web GUI (`ScriptStepPage`) hiện banner cảnh báo màu vàng (không phải lỗi đỏ) khi
  `similar_projects` khác rỗng, có link mở project trùng ở tab mới. Creator vẫn bấm "Tiếp tục"
  bình thường — banner không chặn submit.

### FR86 — Tái dùng cấu hình lần trước

- **FR86.1**: Khi Creator hoàn tất bước 6 (`/create/settings`) của **bất kỳ** project nào, Web
  GUI lưu toàn bộ object cấu hình (engine, quality, giọng, `ttsEnabled`, `voiceId`,
  `subtitleMode`, `subtitleStyle`, `backgroundMusicPath`, `backgroundMusicVolume`,
  `videoOutputMode`, `videoFormatId`, intro/outro) vào `localStorage` dưới một khoá cố định
  `conceptflow:last_used_settings` — **không** qua server, vì đây là tiện ích một-trình-duyệt,
  không phải dữ liệu nghiệp vụ cần đồng bộ nhiều máy (nhất quán với cách `ProjectDraftContext`
  đã dùng `localStorage` cho draft hiện tại).
- **FR86.2**: `NewProjectPage`/`ProjectDraftContext` khi khởi tạo draft mới (bước 1, sau khi có
  `project_id` từ FR83.1) đọc `conceptflow:last_used_settings` nếu có và dùng làm giá trị khởi
  tạo cho bước 6, thay vì hằng số mặc định trong `initialDraft`. Creator vẫn sửa được bình
  thường; sửa ở project này **không** ghi đè lại `last_used_settings` cho tới khi hoàn tất bước 6
  của chính project đó (đúng FR86.1) — tránh cấu hình dở dang của một project ảnh hưởng project
  khác.
- **FR86.3**: Nếu `localStorage` trống hoặc đọc lỗi (private window, bị chặn) — giữ nguyên hành
  vi hiện tại (`initialDraft` mặc định). Không có lỗi hiển thị cho Creator.

## Quyết định bổ sung (2026-09-22, vòng xác nhận thứ hai)

- **FR83.5**: bỏ hẳn khỏi CR — không có job dọn dẹp tự động, không có ngưỡng N ngày. Creator tự
  xoá project draft thủ công qua `DELETE /v1/projects/{id}` đã có sẵn (hoặc xoá hàng loạt nếu
  Web GUI bổ sung sau — không thuộc CR này).
- **FR84.3**: giữ nguyên như đề xuất — CR này chỉ lưu lịch sử + endpoint xem lại (đọc-only). Nút
  "khôi phục bản cũ" (ghi đè bằng một bản trong lịch sử) để P2, không làm trong CR này.

Không còn câu hỏi mở — CR sẵn sàng triển khai.

## Không thuộc phạm vi CR này

- Fuzzy/similarity matching cho trùng chủ đề (FR85.1 dùng exact-match chuẩn hoá).
- Nhiều preset cấu hình đặt tên (Creator đã chọn "một bộ lần cuối dùng").
- Mọi phát hiện khác của `data-flow-review.md` (QoS video-assembly, resource limit Docker VM,
  DLQ cho `orchestrator.events`, phân biệt `queued`/`in_progress`) — theo dõi riêng, không gộp
  vào CR này để giữ review tách bạch khỏi thay đổi.
