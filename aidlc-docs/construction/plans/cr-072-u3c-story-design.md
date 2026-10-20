# CR-072 U3c · Bước 4 Câu chuyện — thiết kế

Nhánh `feature/cr-072-u3c-story` (worktree riêng, từ main `a06abaf`). Nguồn: FR-29, FR-19, FR-28 mục 8; `unit-of-work.md` mục U3c; mockup bản "Giao diện đã chốt" (màn `story`). Khuôn: bước Ý tưởng (brief) và bước Nghiên cứu đã có trong code. Trạng thái: **chờ Creator trả lời câu hỏi và duyệt thiết kế**; chưa có dòng code nào được commit.

## Phụ thuộc

- Phụ thuộc U3a và U3b: cả hai đã vào main. U2a, U1 cũng đã vào main.
- Không còn nhánh nào chưa merge (`git branch --no-merged main` trống), nên không trùng với CR nào đang mở.
- Chặn U3d (Beat sheet đọc `story`).

## Phạm vi

Creator chốt: làm đúng màn Câu chuyện theo mockup, không đụng orchestrator (`flow.go`) hay bước Kịch bản đang chạy; chỗ nào chưa rõ thì hỏi Creator. Sáu ô lưu thành sáu đơn vị riêng; làm lại một ô chỉ viết riêng ô đó; sửa hay làm lại một ô thì câu chuyện về chờ duyệt.

## Thiết kế chức năng

### Thực thể (đơn vị duyệt, ADR-0035; bảng `review_units` đã có)

| Kind | Key | Nội dung |
|---|---|---|
| `story` | `main` | cổng của bước: `approach_key` (góc đang chọn, "" = AI tự chọn), `used_facts[]`; trạng thái duyệt của cả câu chuyện |
| `story_field` | `question` `promise` `chain` `payoff` `logline` `summary` | `text`, `facts[]` (chain: mỗi dòng một bước, kèm mã dữ kiện) |
| `story_approach` | `A1`… | `code`, `name`, `angle`, `why`, `refs[]`, `open`, `is_new`, `when_to_use`, `playbook`, `archetype_id` |
| `fact` (đã có) | `F01`… | khi bỏ qua Nghiên cứu: AI liệt kê, `link=none` ("chưa có nguồn") |

- `story_field` dựng từ `brief:main` (`input_ref`/`input_sha`) nên đổi brief thì `stale`.
- Ô Creator sửa tay vào `creator_fields` của đơn vị đó; AI không ghi đè (FR-17).

### Luồng nghiệp vụ

1. Điều kiện vào: brief đã duyệt; Nghiên cứu đã duyệt hoặc bỏ qua.
2. **Gợi ý góc tiếp cận** (AI, vai `story_approaches`): 4–5 góc từ bảng Kiểu video, hoặc một kiểu mới kèm playbook nháp; mỗi góc có góc kể, lý do kèm mã dữ kiện, câu mở mẫu. "Gợi ý lại" giữ góc đang chọn. "Lưu kiểu mới vào bảng Kiểu video" tạo `VideoArchetype`. "Xem tất cả kiểu" mở trang Kiểu video.
3. **Chọn góc** hoặc "Để AI tự chọn". Khung đã có và đã duyệt thì hỏi xác nhận (UI rule 5), rồi AI viết lại các ô không phải Creator viết.
4. **Viết khung** (AI, vai `story_frame`): sáu ô một lượt, kèm dữ kiện dùng. Khi bỏ qua Nghiên cứu và chưa có dữ kiện, AI liệt kê dữ kiện F01… từ hiểu biết của nó.
5. Mỗi ô: Sửa tay (hộp thoại), Làm lại kèm ghi chú (AI viết riêng ô đó, đọc các ô còn lại).
6. **Dữ kiện dùng**: dữ kiện bắt buộc luôn dùng và không tắt được. Tắt dữ kiện mà chuỗi hoặc payoff đang trích thì hỏi xác nhận, câu chuyện về chờ duyệt.
7. **Duyệt câu chuyện**: một chỗ duyệt cho cả bước. Sửa hay làm lại một ô khi đã duyệt thì câu chuyện về chờ duyệt.

### Quy tắc

- Ô trống không duyệt được; chuỗi có 1–12 bước.
- Mã dữ kiện được trích phải có trong bảng dữ kiện (hoặc trong danh sách AI vừa liệt kê). Mã lạ trong câu trả lời của AI là lỗi, không lưu gì. Mã lạ trong ô Creator gõ bị từ chối kèm tên mã.
- Dạng gõ tay của chuỗi và payoff: mỗi bước một dòng, cuối dòng `[F01, F02]` nếu có dữ kiện.
- Giả thuyết dùng thì ô nói thận trọng; "chưa biết" chỉ dùng để nói khoa học chưa biết; ngộ nhận chỉ nhắc để gỡ (nằm trong prompt).
- Dữ kiện đã bỏ (`keep=false`) không tính là dùng.
- Cần AI mà chưa cấu hình: báo rõ, Creator tự viết từng ô.
- Tác động: `Impact` của brief thêm "Khung câu chuyện cần xem lại" khi đã có khung; `Impact` của ô câu chuyện hiện rỗng tới khi Beat sheet có dữ liệu. Dữ kiện đang được khung trích thì danh sách tác động của dữ kiện (đã có `RegisterFactDependents`) nêu "Câu chuyện: ô …".

### Giao diện (frontend-components)

Trang `/projects/:id/story`, bước 4 thành bước có màn riêng (`screen: "story"`); dải trạng thái; thẻ ① Góc tiếp cận · kiểu video, ② Khung câu chuyện (sáu ô, mỗi ô nhãn "✎ Bạn viết", Sửa tay, Làm lại kèm ghi chú), ③ Dữ kiện dùng trong chuyện (bảng có ô chọn, nhãn loại, "bắt buộc", cảnh báo theo loại), thẻ Duyệt; nút "Sang beat sheet →" mở trang "chưa phát triển" cho tới U3d. Dùng lại `UnitEditModal`, `UnitRedoModal`, `UnitReviewBar`, `ImpactConfirm`, `AiOperationCard`, `AiModelSelect` (bước `story`). Nút theo luật màu: "Gợi ý lại" tím, "Đổi góc" cam (khi đã duyệt), "Duyệt" vàng.

## Thiết kế mức thấp

### Module (authoring-service, hexagonal như U3b)

| Lớp | Tệp mới |
|---|---|
| domain | `story.go` (ô, góc, đọc câu trả lời AI, dạng gõ tay `text [F01]`), `prompt_seed_story.go` (hai vai `story_approaches`, `story_frame`) |
| application | `story_desk.go` (state, chọn góc, viết khung, dữ kiện dùng), `story_fields.go` (UnitKind của ô và cổng), `story_approaches.go` (gợi ý, lưu kiểu mới) |
| adapters/http | `story.go` (route) |
| cmd | nối dây trong `main.go` |

Phụ thuộc đi vào trong: adapters → application → domain. Tiêm bằng hàm khởi tạo (constructor injection) như các use case khác; `main.go` là nơi nối dây. Dùng lại `ReviewUnitPort`, `LLMProviderPort`, `PromptActivePort`, `VideoArchetypesUseCase`.

### Hợp đồng HTTP (qua api-gateway, tiền tố `/v1/projects/{id}`)

| Phương thức | Đường dẫn | Việc |
|---|---|---|
| GET | `/story` | toàn bộ màn: cổng, các ô, các góc, dữ kiện (kèm `used`, `cited`), điều kiện vào |
| POST | `/story/approaches/generate` | AI gợi ý góc (theo dõi bằng `X-Operation-Id`, kind `story_approach`) |
| PUT | `/story/approach` | chọn góc (`key` hoặc "") |
| POST | `/story/approaches/{key}/archetype` | lưu kiểu mới vào bảng Kiểu video |
| POST | `/story/generate` | AI viết khung (kind `story`) |
| PUT | `/story/facts/{fact_id}` | bật/tắt dùng một dữ kiện |
| GET/PUT/POST | `/units/story_field/{key}` (+ `/redo`, `/impact`), `/units/story/main` (`/approve`, `/unapprove`) | API đơn vị chung |

Phiên bản: tiền tố `/v1`, chỉ thêm trường (không đổi tên). Truy vết: `X-Operation-Id` như các lượt AI khác. Tài liệu hợp đồng: `docs/contracts/story-v1.md`.

### Trình tự chính

1. Mở màn: `GET /story`. Nếu thiếu điều kiện vào, màn báo "Duyệt brief / làm hoặc bỏ qua Nghiên cứu trước".
2. Gợi ý góc → lưu `story_approach` (giữ góc đang chọn) → màn tải lại.
3. Viết khung: đọc brief + dữ kiện + góc + ô Creator → AI → kiểm câu trả lời → lưu sáu `story_field` (giữ ô Creator) + (nếu có) tạo `fact` → cổng về chờ duyệt.
4. Đổi góc khi đã có khung: chọn góc → (UI xác nhận) → viết khung.

### Kiểm thử

Go: đọc câu trả lời AI (đúng, thiếu ô, mã lạ, hỏng JSON), dạng gõ tay, use case với AI giả (theo quy ước dự án), kho lưu với Postgres tạm, HTTP. Gateway: route. Web-gui: vitest cho màn, hook, các hộp thoại. Dọn dữ liệu test sau khi chạy.

## Chưa làm (nói rõ)

- Beat sheet và các bước sau: nút "Sang beat sheet" mở trang "chưa phát triển".
- Tác động của câu chuyện lên beat/lời thoại: chưa có dữ liệu để liệt kê.
- Chưa chạy được AI thật trên máy này nếu không có `HIVE_API_KEY`; phần AI chỉ kiểm bằng test giả lập câu trả lời.
