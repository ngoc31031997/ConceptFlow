# CR-094 · Bước 2 Creative brief — yêu cầu và thiết kế

Nhánh `feature/cr-094-creative-brief`. Lộ trình: `aidlc-docs/inception/plans/cr-093-099-content-flow-roadmap.md`. Nguồn: luồng mẫu bước 2 và chốt của Creator ngày 2026-10-24 (bên dưới).

## Chốt của Creator (2026-10-24)

- Brief đọc dữ liệu bước 1 đã tạo: chủ đề (`projects.subject`), câu hỏi trung tâm, danh sách điểm gây tò mò, phạm vi (`project_idea` bản đã duyệt).
- AI sinh năm trường: Audience, Duration, Tone, Learning objectives, Avoid. AI trả JSON, lưu vào bảng mới `project_brief`, hiện lên màn để Creator xem, sửa tay, nhờ AI làm lại kèm ghi chú, rồi duyệt trước khi sang bước sau.
- Duration: AI đề xuất số giây cụ thể; màn hiển thị dạng giờ:phút:giây. Video short giới hạn 1–2 phút.
- Phần cũ của màn brief (ô chọn thời lượng, ô chọn nền tảng, sáu ô biết/hiểu/cảm thấy/ngoài phạm vi) bỏ khỏi màn.
- Có AI review brief trước khi duyệt, giống bước Idea.
- Không xử lý bước 3 trở đi trong CR này.

## Luồng bước 2

**Input**: `idea.v1` đã duyệt + chủ đề + ngôn ngữ + dạng video (long/short) của project.

**Output `creative-brief.v1`** (chi tiết ở `docs/contracts/creative-brief-v1.md`):

| Trường | Kiểu | Ràng buộc máy kiểm |
|---|---|---|
| `audience` | string | không trống |
| `duration_seconds` | integer | short: 60–120; long: 121–1800 (giả định, xem câu hỏi mở) |
| `tone` | string | không trống |
| `learning_objectives` | string[] | 1–6 mục, không trống, không lặp, mỗi mục ≤ 300 ký tự |
| `avoid` | string[] | 1–8 mục, không trống, không lặp, mỗi mục ≤ 300 ký tự |

Dạng `.md` tương ứng (chỉ để đọc, không lưu file):

```
# Creative Brief
Audience: Khán giả phổ thông từ cấp 2 trở lên.
Duration: 4–5 phút.            ← màn hiện 00:04:30 từ duration_seconds = 270
Tone: Tò mò, dễ hiểu, chính xác; không hù dọa quá mức.
Learning objectives:
1. …
Avoid:
- …
```

## Yêu cầu chức năng

| Mã | Yêu cầu |
|---|---|
| FR-1 | Màn bước 2 viết lại: khối chỉ đọc tóm tắt bước 1 (chủ đề, câu hỏi trung tâm, điểm gây tò mò, phạm vi), nút để AI viết brief, thẻ brief năm trường. Không còn ô thời lượng, nền tảng, sáu ô cũ |
| FR-2 | Nút AI viết chỉ bật khi Idea bước 1 đã duyệt; nếu chưa, màn báo và dẫn về bước 1 |
| FR-3 | AI nhận bốn phần của bước 1 + ngôn ngữ + dạng video, trả JSON năm trường bằng ngôn ngữ đã chọn; lưu `project_brief`, trạng thái chờ duyệt |
| FR-4 | Creator sửa tay từng trường (danh sách sửa theo dòng; Duration nhập dạng giờ:phút:giây, lưu lại thành giây). Trường Creator viết không bị AI ghi đè khi làm lại, trừ khi Creator xoá trắng trường đó |
| FR-5 | Làm lại kèm ghi chú; `redo_count` tăng mỗi lần |
| FR-6 | AI review chạy một lần ngay sau khi AI viết và khi Creator bấm; nhận xét bốn mặt: khán giả, thời lượng so với phạm vi, mục tiêu học tập (cụ thể, nằm trong phạm vi Idea), danh sách tránh (bắt được hiểu lầm hay gặp, nói điều chưa chắc như sự thật); kèm chỗ chưa rõ và gợi ý. Lưu `project_brief_review`. Review chỉ để tham khảo, không chặn duyệt |
| FR-7 | Sửa brief sau review thì review hiện "cũ" |
| FR-8 | Duyệt chỉ được khi máy kiểm đạt. Sửa sau khi duyệt đưa brief về chờ duyệt |
| FR-9 | Khi Idea bước 1 đổi sau lúc AI viết brief (lưu `idea_hash`), màn báo brief không còn khớp Idea và để Creator quyết có làm lại không, không tự làm lại |
| FR-10 | Bản đã duyệt đọc được qua `ProjectBriefUseCase.ApprovedBrief` cho bước sau; hợp đồng ở `docs/contracts/creative-brief-v1.md` và một dòng trong `docs/contracts/README.md` |

### Definition of Done — Brief

1. **Máy kiểm** (bắt buộc để duyệt): đúng schema, các ràng buộc trong bảng trên.
2. **AI review** (hỗ trợ, không chặn).
3. **Creator duyệt**: nhóm biết video dành cho ai, muốn người xem hiểu gì và dài bao lâu.

## Thiết kế

### authoring-service

- **Domain** `internal/domain/project_brief.go`: `ProjectBrief` (năm trường), `Normalized`, `Validate(mode)`, `Hash`, `ParseProjectBriefReply`, `StoredBrief` (status, creator_fields, redo_count, idea_hash, language, approved_at), `BriefReview` + `ParseBriefReviewReply` (cùng dạng `IdeaAspect` good/fair/weak). Giới hạn Duration theo dạng video là luật domain.
- **Bảng** (thêm vào `db.go`, `CREATE TABLE IF NOT EXISTS`, giống `project_idea`):
  - `project_brief(project_id PK, audience TEXT, duration_seconds INT, tone TEXT, learning_objectives JSONB, avoid JSONB, creator_fields JSONB, status TEXT, redo_count INT, language TEXT, idea_hash TEXT, approved_at TIMESTAMPTZ, updated_at TIMESTAMPTZ)`
  - `project_brief_review(project_id PK, review JSONB, brief_hash TEXT, created_at TIMESTAMPTZ)`
- **Repository** `project_brief_repository.go`.
- **Use case** `internal/application/project_brief.go`: `State`, `Generate(note)`, `Edit(patch)`, `Redo`, `Approve`, `Unapprove`, `Review`, `ApprovedBrief`. Đọc Idea qua `ProjectIdeaUseCase.ApprovedIdea` (không đọc bảng `project_idea` trực tiếp), chủ đề và ngôn ngữ qua `IdeaInputPort` sẵn có, dạng video qua cổng project đang dùng. Khoá authoring, `X-Operation-Id`, ghi usage giống bước Idea.
- **Prompt seed** hai vai mới `project_brief`, `project_brief_review` (Creator sửa được); prompt yêu cầu trả JSON đúng năm trường, ghi rõ khoảng giây theo dạng video. Theo ghi nhớ: rebuild authoring-service để seeder upsert.
- **HTTP** `/v1/projects/{id}/creative-brief`: `GET`, `POST /generate`, `PATCH`, `POST /approve`, `POST /unapprove`, `POST /review`. Route `/brief` cũ để nguyên trong code nhưng màn không dùng nữa.

### api-gateway

Proxy các route mới; `generate` và `review` đi client timeout dài như route Idea.

### web-gui

- `CreativeBriefStepPage.tsx` viết lại; thành phần mới `components/brief/ProjectBriefCard`, `BriefEditModal`, `BriefRedoModal`, hook `useProjectBrief`, hàm `formatDuration`/`parseDuration` (giây ↔ `h:mm:ss`, dạng `m:ss` khi dưới một giờ).
- `CreativeBriefCard.tsx` cũ bỏ khỏi màn.
- Theo `docs/ux-ui-design-rules.md`.

### Code cũ để nguyên (theo chốt "không quan tâm bước sau")

`CreativeBriefUseCase`, đơn vị `brief:main`, `brief_platform`, `TargetDuration` còn nguyên để các bước sau vẫn build được. Hệ quả: bước 3 trở đi vẫn đọc `brief:main` cũ nên sẽ báo thiếu brief đã duyệt cho tới CR-095.

## Kiểm thử

Test domain (parse, validate theo dạng video, hash), use case (generate giữ trường Creator, idea chưa duyệt bị chặn, review cũ, sửa sau duyệt), repository, HTTP, gateway routing, trang web. AI được giả lập trong mọi test, không gọi model thật. Dọn dữ liệu test sau khi chạy.

## Câu hỏi mở

1. Duration video long: mình giả định 121–1800 giây (2–30 phút). Bạn muốn khoảng khác không?
2. Giới hạn số mục: mục tiêu học tập 1–6, danh sách tránh 1–8. Có ổn không?
