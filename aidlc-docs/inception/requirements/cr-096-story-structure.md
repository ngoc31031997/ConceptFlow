# CR-096 · Bước 4 Story development — yêu cầu và thiết kế

Nhánh `feature/cr-096-story-structure`. Lộ trình: `aidlc-docs/inception/plans/cr-093-099-content-flow-roadmap.md`. Nguồn: luồng mẫu bước 4 (`docs/document/animation_pipeline_realistic_io_example.md`) và chốt của Creator ngày 2026-10-24 (bên dưới).

## Chốt của Creator (2026-10-24)

1. Bước 4 đọc output của bước 2 (`project_brief`, bản đã duyệt) và bước 3 (`project_research`, các dòng `keep = true`), đúng như luồng mẫu (`creative-brief.md`, `research-dossier.md`).
2. Prompt của bước chỉ nạp input từ hai bước này.
3. Màn hình và luồng của bước sửa theo các trường mới.
4. Kết quả chỉ lưu đúng các trường của `story-structure.md` trong ví dụ, ở bảng mới `project_story`.
5. Số nhịp của Narrative arc không cố định: AI quyết định theo dữ kiện đầu vào.
6. Bỏ bước chọn góc tiếp cận.

## Hiện trạng (đối chiếu code)

- Bước 4 đọc brief từ review unit `brief` và ý tưởng theo kiểu cũ (`loadApprovedBrief`, `story_desk.go`), đọc dữ kiện từ review unit `fact` (`loadKeptFacts`). Sau CR-095, dự án mới lưu dữ kiện ở `project_research`, nên bước 4 không thấy dữ kiện nào và rơi vào chế độ "bỏ qua nghiên cứu": AI tự liệt kê dữ kiện chưa có nguồn.
- Đầu ra là 6 unit `story_field` (`question`, `promise`, `chain`, `payoff`, `logline`, `summary`), một gate unit `story` (`approach_key`, `used_facts`) và các unit `story_approach` (A1…), với hai prompt: `story_approaches` và `story_architect` (`prompt_seed_story.go`).
- Bước 5 Beat sheet đọc các unit `story_field` (`beat_writer.go`: `storyFields`, `storyFieldsText`; dùng `question`, `chain`, `payoff`).

## Hợp đồng `story-structure.v1` · bảng `project_story`

Một dòng một dự án; khoá `project_id`.

| Cột | Kiểu | Ý nghĩa (ví dụ) |
|---|---|---|
| `project_id` | text, khoá | |
| `central_question` | text, bắt buộc | Central question |
| `core_answer` | text, bắt buộc | Core answer |
| `narrative_arc` | jsonb, bắt buộc | Narrative arc: mảng có thứ tự `[{"beat":"Hook","text":"...","facts":["F01"]}]`, ít nhất 3 nhịp, số nhịp do AI quyết |
| `creator_fields` | jsonb | các trường Creator đã tự viết (`central_question`, `core_answer`, `narrative_arc`), lần viết lại không ghi đè |
| `brief_hash`, `research_hash` | text | băm của brief và dữ kiện lúc viết, để báo khung đã cũ |
| `status` | text | `pending` / `approved` |
| `created_at`, `updated_at` | timestamptz | |

`facts` của mỗi nhịp là mã trong `project_research`. Nó là chỗ bám để bước sau trích dữ kiện, ứng với "đường dây kể chuyện" của luồng mẫu; có thể rỗng. Mã không có trong bảng thì bị từ chối.

**Hoàn thành khi** (điều kiện duyệt): có đủ ba trường; arc có ít nhất 3 nhịp, nhịp đầu dẫn vào câu hỏi và nhịp cuối trả lời nó. Máy chỉ kiểm được số trường và số nhịp; Creator xác nhận câu chuyện có mạch, không chỉ là danh sách thông tin.

## Prompt `story_architect` (viết lại)

- Input chỉ còn: `{{brief}}` (5 trường của bước 2: audience, duration, tone, learning objectives, avoid), `{{research}}` (các dòng giữ lại của `project_research`: mã, loại, dữ kiện, độ tin), `{{brief_language}}`, `{{current_story}}`, `{{kept_fields}}`, `{{redo_note}}`. Bỏ `{{topic}}`, `{{approach}}`, `{{facts_rule}}`, `{{facts_example}}`. Prompt không có `{{idea}}`: brief đã được viết từ ý tưởng đã duyệt (CR-094).
- Việc của AI: chọn góc kể và xây mạch từ câu hỏi đến lời giải. Số nhịp tuỳ lượng dữ kiện; nhịp đầu là hook, nhịp cuối là payoff; dữ kiện `hypothesis` phải nói thận trọng, `misconception` chỉ nhắc để gỡ, số liệu chỉ lấy từ bảng.
- Đầu ra: `{"central_question":"...","core_answer":"...","narrative_arc":[{"beat":"...","text":"...","facts":["F01"]}]}`.
- Bỏ prompt `story_approaches` (seed và vai).
- Seed nằm ở authoring-service và seeder upsert: sau khi đổi phải rebuild và restart authoring-service, và kiểm Creator không có prompt tuỳ chỉnh đang active đè lên bản seed.

## Màn hình bước 4

- Bỏ thẻ góc tiếp cận (`ApproachCard`) và thẻ chọn dữ kiện dùng (`FactsUseCard`).
- Thẻ khung (`FrameCard`) hiện ba trường: Câu hỏi trung tâm, Câu trả lời cốt lõi, Mạch kể (danh sách nhịp, mỗi nhịp tên nhịp, nội dung và mã dữ kiện). Sửa tay, viết lại có ghi chú, duyệt như hiện nay.
- Nút "Xem đầu vào của AI" hiện brief và bảng dữ kiện mà AI đọc, giống bước 3.

## Bước 5 vẫn chạy được (lớp chuyển đổi)

Bước 5 chưa có CR mới (CR-097), nên `beat_writer.go` đọc `project_story` đã duyệt qua một hàm chuyển đổi: `question` ← `central_question`, `chain` ← các nhịp của arc (kèm mã dữ kiện), `payoff` ← `core_answer`. Điều kiện "story đã duyệt" của bước 5 đọc `project_story.status`.

## Việc cần làm (sau khi duyệt)

1. Migration tạo `project_story`; repository và cổng lưu trữ.
2. `StoryDeskUseCase`: đọc `project_brief` và `project_research`; viết, sửa, viết lại và duyệt trên `project_story`; bỏ phần góc tiếp cận, gate `used_facts` và chế độ AI tự liệt kê dữ kiện.
3. Domain: kiểu `StoryStructure`, parse đầu ra của AI, kiểm mã dữ kiện.
4. Prompt seed `story_architect` mới; bỏ `story_approaches`.
5. Lớp chuyển đổi cho bước 5; xoá `project_story` khi xoá phần authoring của dự án.
6. Web GUI: `StoryStepPage`, `FrameCard`, client API, hook, dialog xem đầu vào.
7. `docs/contracts/story-structure-v1.md` và một dòng trong `docs/contracts/README.md`; test (AI giả).
8. Rebuild authoring-service và web-gui.

## Chốt thêm của Creator (2026-10-24)

- **Cột phụ** `creator_fields`, `brief_hash`, `research_hash`, `status` được giữ.
- **Bước 3 bị bỏ qua** (CR-095 vẫn cho bỏ qua): AI viết câu chuyện không trích mã dữ kiện và chỉ nói định tính; không còn tự liệt kê dữ kiện.
- **Dự án cũ:** chỉ áp dụng cho dự án mới, giống CR-095; không migrate unit `story_field` cũ.

## Ngoài phạm vi

Thiết kế lại bước 5 Beat sheet (CR-097) và các bước sau.
