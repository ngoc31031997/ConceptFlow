# CR-093 · Bước 1 Idea — yêu cầu

Nhánh `feature/cr-093-idea-step`. Lộ trình bảy CR: `aidlc-docs/inception/plans/cr-093-099-content-flow-roadmap.md`. Nguồn: luồng mẫu Creator dán 2026-10-24 và các câu trả lời trong `cr-093-idea-requirement-questions.md`, `cr-093-idea-clarification-questions.md`, cùng các chốt bằng lời ngày 2026-10-24 (bên dưới).

## Chốt của Creator

- Bước brief và những gì phụ thuộc vào Idea cũ: chưa làm trong CR này, hỏng thì để đó. Dạng video, thời lượng, nền tảng nhận giá trị mặc định để project vẫn tạo được.
- Đoạn Creator gõ lưu ở cột `subject`, ngôn ngữ lưu ở cột `language`, cả hai trong bảng `projects` của orchestrator. Authoring-service đọc qua API nội bộ, không ghi chéo DB.
- Không có trường chủ đề do AI viết lại: chủ đề là đoạn Creator gõ. Idea có ba trường (Creator chốt cùng lúc thay prompt viết ý tưởng, 2026-10-24).
- Idea có API riêng, bảng riêng `project_idea`; kết quả AI review ở bảng riêng `project_idea_review`.

## Hiện trạng đã kiểm trong code (đính chính tài liệu trước)

- Bước Ý tưởng đã tồn tại: màn tại route `/` (`ScriptStepPage`), AI viết ý tưởng bốn ô `question/why/known/unknown` thành đơn vị duyệt `idea:main`; bước brief đọc ý tưởng đã duyệt. Bước Cấu hình đã bỏ từ trước.
- `topic` và `language` của project nằm ở bảng `project_authoring` (authoring-service), không ở `projects`; `projects` của orchestrator chỉ có `voice_language`. Hai cột mới `subject` và `language` được thêm theo chốt trên, nên cùng một thông tin hiện nằm ở hai nơi; code cũ vẫn ghi `project_authoring` như trước.

## Luồng bước 1

**Input**: ý tưởng dạng văn bản tự do và ngôn ngữ nội dung đã chọn.

**Công việc**: AI chọn câu hỏi trung tâm và ghi các điểm gây tò mò. Chưa viết lời dẫn, chưa nghĩ hình ảnh.

**Output `idea.v1`** (hợp đồng cho bước sau; chi tiết ở `docs/contracts/idea-v1.md`): `central_question`, `curiosity_points[]`, `scope`, viết bằng ngôn ngữ đã chọn.

## Yêu cầu chức năng

| Mã | Yêu cầu | Trạng thái |
|---|---|---|
| FR-1 | Màn bước 1 mới thay màn cũ: một textbox lớn, ô chọn ngôn ngữ, nút để AI viết. Không còn ô dạng video, thời lượng, nền tảng | Làm |
| FR-2 | AI nhận văn bản, trả ba trường của `idea.v1` bằng ngôn ngữ đã chọn; bản nháp cho tới khi Creator duyệt | Làm |
| FR-3 | Creator sửa tay từng trường, làm lại kèm ghi chú, duyệt; trường Creator viết không bị AI ghi đè trừ khi Creator xoá trắng trường đó | Làm |
| FR-4 | Đoạn Creator gõ lưu ở `projects.subject`, ngôn ngữ ở `projects.language` (orchestrator) | Làm |
| FR-5 | Ba trường Idea lưu ở bảng `project_idea` của authoring-service, một dòng cho mỗi project; không lưu file `.md` | Làm |
| FR-6 | Authoring-service đọc hai giá trị trên qua `GET /internal/v1/projects/{id}/idea-input`; không ghi chéo DB | Làm |
| FR-7 | Khi đoạn gõ hoặc ngôn ngữ khác lúc AI viết ý tưởng, màn báo và để Creator quyết có làm lại không, không tự làm lại | Làm một phần: cờ `input_changed` và dải báo có; màn chỉ cho sửa đoạn gõ sau khi tạo, ngôn ngữ khoá sau khi tạo, và `projects.language` chỉ cập nhật cùng đoạn gõ, nên cờ chỉ bắt được đổi đoạn gõ |
| FR-8 | Bản Idea đã duyệt đọc được qua `ProjectIdeaUseCase.ApprovedIdea` cho bước sau; hợp đồng ở `docs/contracts/idea-v1.md` và một dòng trong `docs/contracts/README.md` | Làm; chưa có route HTTP nội bộ cho bước sau |
| FR-9 | Sửa Idea sau khi duyệt đưa nó về chờ duyệt; bước sau chỉ thấy bản đã duyệt | Làm |
| FR-10 | AI review chạy theo yêu cầu và một lần ngay sau khi AI viết; kết quả lưu ở `project_idea_review` | Làm |
| FR-11 | Review ghi nhận xét độ rõ, khả năng nghiên cứu, phạm vi, chỗ chưa rõ, gợi ý, dấu vân tay của ý tưởng lúc review | Làm |
| FR-12 | Sửa ý tưởng sau review thì review hiện là "cũ" | Làm |

### Definition of Done — Idea

1. **Máy kiểm hợp lệ** (bắt buộc để duyệt): đúng schema `idea.v1`; `central_question`, `scope` không trống; `curiosity_points` có 1 đến 10 điểm, không trống, không lặp, mỗi điểm tối đa 300 ký tự. Máy không kiểm được một điểm gây tò mò có liên quan đến chủ đề hay không; việc đó nằm ở AI review và ở Creator.
2. **AI Quality Review** (hỗ trợ, không chặn): đánh giá độ rõ, khả năng nghiên cứu, phạm vi; ghi chỗ chưa rõ và gợi ý; không thay Creator duyệt.
3. **Creator duyệt**: Creator xem Idea và review rồi chọn Duyệt; bản được duyệt lưu lại làm đầu vào cho bước sau.

Idea hoàn thành khi máy kiểm đạt và Creator duyệt.

## Ngoài phạm vi

- Bước brief, Creative brief theo luồng mẫu (CR-094), và việc brief đọc Idea mới. Hệ quả đã biết: bước brief vẫn đọc đơn vị `idea:main` cũ nên sẽ báo thiếu ý tưởng đã duyệt cho tới khi CR-094.
- Project đã có dữ liệu cũ: không chuyển đổi.
- Xuất file `.md`; thư mục chung.
- Đổi ngôn ngữ sau khi đã tạo project.
- Gỡ code Idea cũ (`IdeaUseCase`, đơn vị `idea:main`, vai prompt `idea`): còn nguyên vì bước brief và vài test của bước sau vẫn dựa vào chúng.

## Yêu cầu phi chức năng

| Mã | Yêu cầu |
|---|---|
| NFR-1 | Tôn trọng ranh giới service: authoring-service chỉ đọc dữ liệu project qua API nội bộ |
| NFR-2 | Lượt AI viết và review nhận `X-Operation-Id`, theo dõi và huỷ được như các bước khác |
| NFR-3 | Test không gọi AI thật |
| NFR-4 | Số lần làm lại ghi trong `project_idea.redo_count` |
| NFR-5 | Giao diện theo `docs/ux-ui-design-rules.md` |
| NFR-6 | Prompt viết Idea và review là prompt seed trong authoring-service (vai `project_idea`, `project_idea_review`), Creator sửa được |
