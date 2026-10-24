# CR-095 · Bước 3 Research & fact-check — yêu cầu và thiết kế

Nhánh `feature/cr-095-research-dossier`. Lộ trình: `aidlc-docs/inception/plans/cr-093-099-content-flow-roadmap.md`. Nguồn: luồng mẫu bước 3 (`docs/document/animation_pipeline_realistic_io_example.md`) và chốt của Creator ngày 2026-10-24 (bên dưới).

## Chốt của Creator (2026-10-24)

1. Sau khi kết quả nghiên cứu về (Creator dán JSON `conceptflow.research.v1`), lưu vào bảng mới `project_research`, mỗi dữ kiện một dòng, để các bước sau tham chiếu.
2. Prompt tạo câu hỏi nghiên cứu (vai `research_questions`) dùng bản Creator dán (RESEARCH PLANNER), sửa input nhận từ bước 1 và 2 để chỉ trả đúng dữ liệu cần thiết như ví dụ.
3. Cho duyệt bảng dữ kiện kể cả khi nguồn chưa xác minh: giữ nguyên điều kiện duyệt hiện tại (có ít nhất một dữ kiện đang giữ).
4. Cho phép bỏ qua cả bước: giữ nguyên.
5. Chỉ áp dụng cho dự án mới, không migrate dữ kiện cũ.
6. Không xuất file `research-dossier.md` / `source-register.csv`; chỉ lưu xuống DB để các bước sau dùng lại.
7. `{{idea}}` của prompt là output bước 1 đọc từ bảng `project_idea` (bản đã duyệt); `{{brief}}` là output bước 2 đọc từ bảng `project_brief`.

## Hiện trạng (đối chiếu code)

- Dữ kiện đang là review unit loại `fact` (khoá `F01`…), câu hỏi là unit `research_question` (`Q1`…), cổng bước là unit `research`.
- Bước sau đọc dữ kiện qua `ListUnits(UnitKindFact)` (`story_desk.go`, `story_fields.go`) và `FactDependentsFinder`.
- Prompt `research_questions` nhận `{{topic}}`, `{{brief}}`, `{{brief_language}}`, `{{kept_questions}}`, `{{redo_note}}`; chưa nhận idea riêng.

## Bảng `project_research`

Một dòng một dữ kiện; khoá `(project_id, fact_id)`.

| Cột | Ý nghĩa |
|---|---|
| `project_id`, `fact_id` | khoá (`F01`…) |
| `question_id` | mã câu hỏi nghiên cứu mà dữ kiện trả lời (`Q1`…) |
| `claim_vi`, `type` | dữ kiện và loại (`fact`, `hypothesis`, `unknown`, `misconception`) |
| `source_title`, `publisher`, `year`, `url`, `quote` | nguồn |
| `confidence`, `note` | độ tin, ghi chú |
| `must`, `keep` | "bắt buộc nói", đang giữ hay đã bỏ |
| `link_state`, `link_why`, `link_nearest`, `link_checked_at` | kết quả kiểm link |
| `creator_fields` | các trường Creator đã tự viết, lần dán sau không ghi đè |
| `created_at`, `updated_at` | |

Bước sau chỉ đọc dòng có `keep = true`; mỗi dòng mang `type` và `link_state` để bước sau tự quyết có dùng dữ kiện chưa xác minh hay không (duyệt vẫn không chặn).

## Prompt `research_questions`

- Thay nội dung seed bằng bản Creator dán, thêm `{{idea}}` (bản đã duyệt của bước 1) bên cạnh `{{brief}}`.
- Input rút gọn đúng dữ liệu cần: `idea` (chủ đề, câu hỏi trung tâm, điểm gây tò mò, phạm vi), `brief` (audience, duration, tone, learning objectives, avoid), `brief_language`, `kept_questions`, `redo_note`. Bỏ `{{topic}}` nếu `idea` đã chứa chủ đề.
- Sửa số bước trong phần Pipeline context cho khớp luồng mẫu (Research & fact-check là bước 3, câu hỏi nghiên cứu là phần lập kế hoạch của bước 3).
- Đầu ra giữ `{"questions":[…]}` nên `ParseResearchQuestionsReply` không đổi.
- Prompt seed nằm ở authoring-service và seeder upsert: sau khi đổi phải rebuild và restart authoring-service, và kiểm Creator không có prompt tuỳ chỉnh đang active đè lên bản seed.

## Việc cần làm (sau khi duyệt)

1. Migration tạo `project_research`; repository và cổng lưu trữ mới.
2. Dán kết quả (`POST /research/paste`), sửa dữ kiện, kiểm link ghi vào `project_research` thay vì unit `fact`.
4. Cập nhật prompt seed `research_questions`, truyền `idea` trong `GenerateQuestions`.
5. Cập nhật `docs/contracts/research-v1.md` và `docs/contracts/README.md`; test (AI giả).
6. Rebuild authoring-service (và dịch vụ khác nếu đổi).

## Ngoài phạm vi

Bước 4 Story (`story_desk.go`, `story_fields.go`, `FactDependentsFinder`) không đổi trong CR này; chuyển nó sang đọc `project_research` (kể cả dữ kiện AI Story tự liệt kê) thuộc CR-096.

**Hệ quả đã biết:** sau CR-095, dự án mới lưu dữ kiện ở `project_research` còn bước Story vẫn đọc dữ kiện từ review unit `fact`, nên với dự án mới bước Story sẽ không thấy dữ kiện nào cho tới khi CR-096 xong.
