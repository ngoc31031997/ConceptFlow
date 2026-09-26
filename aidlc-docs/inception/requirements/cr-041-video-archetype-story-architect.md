# CR-041 — Kiểu video (archetype) cho prompt Biên kịch, chọn ngầm rồi chọn tường minh

## Date
2026-09-26

## Stage
Requirements Analysis (Change Request). **Giai đoạn 1 đã code xong** (chỉ đổi prompt, không đổi hợp đồng). **Giai đoạn 2 chờ Creator duyệt thiết kế trước khi code.**

## Intent Analysis
- **Request type**: Mở rộng chất lượng nội dung (một khuôn không hợp mọi chủ đề)
- **Scope estimate**: GĐ1 — `authoring-service` (một seed prompt). GĐ2 — `authoring-service`, `llm-service`, `orchestrator` (lưu lựa chọn), `web-gui` (bước Cấu hình).
- **Complexity estimate**: GĐ1 thấp. GĐ2 trung bình (một trường mới trên project + một tác vụ LLM nhẹ + một biến prompt).

## Bối cảnh
Prompt Biên kịch (`story_architect`) đến v6 là **một khuôn duy nhất**: nghịch lý có thật → chuỗi 5–7 ví dụ đa lĩnh vực. Khuôn này hợp chủ đề "lỗi tư duy", nhưng ép nhầm với chủ đề kiểu "for / while / do-while": model sẽ định nghĩa ba vòng lặp song song, đúng thứ Creator không muốn ("nhiều định nghĩa na ná nhau").

Kiểu B (họ khái niệm) trả lời được lo ngại đó bằng một nguyên tắc: **không bao giờ định nghĩa các cách song song; mỗi cách xuất hiện như câu trả lời cho điểm yếu của cách trước**, và kết bằng một câu quyết định "khi nào dùng cái nào" thay cho bảng định nghĩa.

## Các kiểu video

| Kiểu | Hợp với chủ đề | Xương sống |
|---|---|---|
| A — Nghịch lý + chuỗi ví dụ | lỗi tư duy, thiên kiến, hiện tượng phản trực giác | nghịch lý thật → giải trước, gọi tên sau → 5–7 ví dụ đa lĩnh vực |
| B — Họ khái niệm, nhiều cách | 2–4 công cụ cùng giải một loại việc (for/while/do-while) | một việc chung → mỗi cách là lời đáp cho điểm yếu của cách trước → câu quyết định |
| C — Cơ chế theo dấu vết | quy trình/hệ thống qua nhiều chặng (gõ URL → trang hiện ra) | một đầu vào cụ thể đi qua từng chặng |
| D — Bài toán tiến hoá | kỹ thuật sinh ra để cứu cách ngây thơ (cache, index, tìm nhị phân) | cách ngây thơ → vỡ ở đâu → cải tiến kèm cái giá |

Đã cân nhắc và **không tách riêng**: "so sánh hai phe" (gộp vào B — cùng nguyên tắc điểm yếu → cách tiếp theo) và "gỡ lỗi/đập tan hiểu lầm" (gộp vào A và C). Thêm kiểu thứ năm về sau là thêm một playbook, không phải đổi kiến trúc; đây là lý do GĐ2 tách playbook thành dữ liệu.

## Vì sao không để AI tự chọn giữa nhiều system prompt
1. **Format được chọn trước khi viết script.** Wizard chọn format ở bước 2 (Cấu hình), prompt Biên kịch chạy ở bước 3. `{{format_beats}}` ép dàn ý dùng đúng id beat của format; `validate_script` chặn render nếu thiếu beat bắt buộc. AI tự đổi sang kiểu không vừa format thì dàn ý vỡ hoặc bị chặn.
2. **Không có định tuyến.** Mỗi vai chỉ có một prompt đang kích hoạt (`prompts.is_active`). Nhiều prompt + AI chọn nghĩa là thêm code phân loại, thêm trường lưu, sửa `render_prompt`.
3. **Trùng lặp.** 70–80% prompt là phần dùng chung (lời thoại, độ chính xác, khuôn output, tự kiểm). Tách 4 prompt đầy đủ thì mỗi lần sửa một quy tắc phải sửa 4 chỗ.
4. **Chọn ngầm thì sai cũng ngầm.** Vì vậy model phải in `KIỂU VIDEO: B — vì …` ở dòng đầu output.

## Lưu ý về format hiện có
Các beat trong bản mô tả ban đầu (`edge`) thuộc `visual_first_7min`, format này **không còn nằm trong `BuiltinFormats()`**. Hai format đang có: `case_study_essay_8min` (hook, concrete, pattern, variation ×7, modern, recap, cta) và `quick_explainer_3min` (không có `variation`/`modern`). Playbook được viết theo các beat này: "bẫy hay gặp" của kiểu B đi vào `modern` chứ không phải `edge`.

## Giai đoạn 1 — prompt khung (ĐÃ LÀM, không đổi code ngoài seed)

### FR120 — Prompt Biên kịch v7
- **FR120.1** Giữ phần dùng chung: bản sắc kênh (viết lại để không mặc định là kiểu A), giọng văn, quy tắc sự thật, quy tắc lời thoại, khuôn output, tự kiểm.
- **FR120.2** Thêm **Bước 0**: chọn A/B/C/D, in `KIỂU VIDEO: <kiểu> — vì …` ở **dòng đầu** output.
- **FR120.3** Mỗi kiểu có một playbook ngắn, nói cách gán vào các beat của format đang chọn.
- **FR120.4** Kiểu không khớp format: giữ đúng id/thứ tự beat và in thêm `CẢNH BÁO FORMAT: nên dùng format X vì …`, không tự bẻ cấu trúc.
- **FR120.5** Creator ép kiểu bằng cách ghi "kiểu: B" vào chủ đề.
- **FR120.6** Mọi nhãn output cũ giữ nguyên (KHUNG BÀI, DANH SÁCH VÍ DỤ, BEAT …), nên Visual Director không đổi hợp đồng. Tự kiểm thêm mục 11 (kiểu đúng, id beat đúng, kiểu B không định nghĩa song song).

### Đã thực hiện
- `prompt_template_seeds.go`: `storyArchitectVI` v6 → v7.
- `golden_prompts_test.go`: hash `story_architect` được **cố ý** cập nhật (test này canh việc tách prompt không làm đổi bản thủ công; ở đây đổi là chủ đích) + test mới kiểm dòng `KIỂU VIDEO`, `CẢNH BÁO FORMAT`, 4 playbook, `{{format_beats}}`.
- `go test ./...` của authoring-service xanh. Đã build lại và khởi động lại `authoring-service`; hàng `system-story_architect` là hàng đang active và trả về nội dung v7.

### Tiêu chí để sang GĐ2 (Creator chấm bằng chạy thật)
Chạy ít nhất 6 chủ đề (2 mỗi kiểu B/C/D hoặc hỗn hợp), trên cả hai format, và xem:
- dòng `KIỂU VIDEO` có đúng chủ đề không (tỉ lệ chọn sai cần thấp, ví dụ ≤ 1/6);
- id beat có luôn đúng format, `validate_script` không chặn;
- kiểu B có thật sự không định nghĩa song song;
- chủ đề ghi "kiểu: X" có được tuân theo không.
Nếu model hay chọn sai kiểu hoặc bỏ dòng đầu, đó là lý do làm GĐ2; nếu chạy ổn thì GĐ2 vẫn có giá trị (giảm token, bỏ đoán) nhưng có thể hoãn.

## FR123 — Bảng kiểu video do Creator quản lý (ĐÃ LÀM, Creator yêu cầu sau GĐ1)
Giữ A–D làm dữ liệu hệ thống và cho Creator tự thêm kiểu, nên phần "playbook thành dữ liệu" của FR121.2 được làm trước.
- **FR123.1** Bảng `video_archetypes` (authoring DB): `code` (duy nhất, không phân biệt hoa/thường, ≤ 8 ký tự), `name`, `when_to_use`, `playbook`, `is_system`. A–D là dòng hệ thống (`system-A`…), được nạp lại mỗi lần khởi động như prompt hệ thống; chỉ xem và copy.
- **FR123.2** API: `GET /v1/video-archetypes`; `POST /v1/admin/video-archetypes`, `POST …/{id}/copy`, `PUT …/{id}`, `DELETE …/{id}` (sửa/xoá dòng hệ thống trả 403, trùng mã trả 409). Mã để trống thì lấy chữ cái còn trống kế tiếp (E, F…). Gateway proxy các route này sang authoring-service.
- **FR123.3** Prompt Biên kịch v8 dùng biến `{{video_archetypes}}` (thực đơn kiểu + một playbook mỗi dòng) thay cho bốn playbook cứng; thêm dòng vào bảng là lượt Biên kịch kế tiếp thấy ngay, không cần build lại. Cả hai đường render (theo project và `prompt-renders`) điền biến này.
- **FR123.4** web-gui: mục "Kiểu video" ở menu trái, trang `/settings/video-archetypes` (danh sách, thêm, sửa, copy, xoá) cùng quy ước với "Cài đặt prompt".
- Còn lại của GĐ2 (chọn kiểu ở bước Cấu hình, nút AI gợi ý, biến `{{video_archetype}}` chỉ chèn một playbook) vẫn chờ duyệt; nó sẽ đọc từ bảng này.

## Giai đoạn 2 — kiểu video thành lựa chọn ở bước Cấu hình (CHỜ DUYỆT)

### Mục tiêu
Kiểu video đi cùng format và do **Creator quyết định cuối**, giống cổng duyệt dàn ý (CR-024). AI chỉ gợi ý.

### FR121 — Dữ liệu & biến prompt
- **FR121.1** Project có thêm trường `video_archetype` (`auto | A | B | C | D`, mặc định `auto`) — lưu ở orchestrator cùng các lựa chọn Cấu hình khác (`format_id`, `voice_id`…).
- **FR121.2** Playbook tách khỏi prompt thành **dữ liệu** (bốn khối văn bản trong authoring-service, cùng kiểu với `channel_identity`). Prompt Biên kịch có biến mới `{{video_archetype}}`; bộ render thay bằng **đúng một playbook** của kiểu đã chọn. Vì thế prompt ngắn đi (chỉ mang một playbook thay vì bốn) và không còn phải đoán.
- **FR121.3** `auto` giữ hành vi GĐ1: `{{video_archetype}}` mở ra Bước 0 + cả bốn playbook, model tự chọn và in dòng `KIỂU VIDEO`. Nhờ vậy project cũ và prompt do Creator tự viết không vỡ.
- **FR121.4** Việc thay biến nằm trong authoring-service, nơi `render_prompt` đã ở sau CR-040. (Bản mô tả ban đầu nói "Orchestrator chèn"; sau CR-040 orchestrator không còn render prompt.) `RenderInput` thêm `VideoArchetype`.

### FR122 — Gợi ý bằng AI
- **FR122.1** Nút "AI gợi ý" ở bước Cấu hình đọc chủ đề rồi trả `{archetype, reason, suggested_format_id}`.
- **FR122.2** Chạy qua `llm-service` (Ollama, tác vụ nhẹ), theo mẫu `POST /v1/suggest-metadata`: JSON mode, retry 2 lần, giới hạn độ dài chủ đề. Endpoint mới `POST /v1/suggest-video-archetype`.
- **FR122.3** Bảng kiểu → format hợp lệ nằm ở authoring-service (dữ liệu, có test): ví dụ A → `case_study_essay_8min`; kiểu cần beat `variation` không được gợi ý cùng `quick_explainer_3min`. Gợi ý của model chỉ là đề xuất; nếu nó ghép kiểu với format không hợp lệ, bảng thắng và lý do được ghi lại.
- **FR122.4** Creator xác nhận hoặc đổi. Đổi kiểu **không** tự đổi format (Creator có thể đã chọn format cố ý); nếu cặp không khớp, wizard hiện cảnh báo giống dòng `CẢNH BÁO FORMAT` ở GĐ1.
- **FR122.5** Nếu `llm-service` lỗi/không có, nút báo lỗi rõ ràng và Creator vẫn chọn tay hoặc để `auto`; không có gợi ý giả.

### Ngoài phạm vi
- Định tuyến nhiều system prompt / một lượt LLM phân loại trước khi chạy Biên kịch.
- Playbook riêng cho Visual Director/Engineer theo kiểu (chỉ Biên kịch đổi trong CR này).
- Kiểu do Creator tự định nghĩa (dữ liệu playbook đã tách nên có thể làm sau).

### Kiểm thử dự kiến (GĐ2)
- `render_prompt`: mỗi kiểu chỉ chèn đúng playbook của nó; `auto` chèn Bước 0 + bốn playbook; không còn `{{video_archetype}}` sót lại (test placeholder hiện có).
- Bảng kiểu → format: mỗi format builtin đều có ít nhất một kiểu hợp lệ.
- `llm-service`: JSON hỏng → retry → lỗi rõ; kiểu ngoài A–D bị loại.
- web-gui: chọn tay, gợi ý, đổi lại, cảnh báo không khớp, project cũ hiển thị `auto`.

## Rủi ro
- Model nhỏ có thể bỏ dòng `KIỂU VIDEO` hoặc chọn sai (GĐ1) — đã có mục tự kiểm 11 và tiêu chí chấm ở trên; GĐ2 loại rủi ro này khi Creator chọn tường minh.
- Prompt Creator tự viết (`is_system = false`) che seed hệ thống sẽ không thấy v7; bắt buộc kiểm hàng active sau mỗi lần đổi seed.

## Quyết định cần Creator chốt trước GĐ2
1. Bốn kiểu A–D có đủ, hay thêm kiểu (gỡ lỗi code? so sánh hai phe?) trước khi cố định dữ liệu?
2. Đổi kiểu có nên **tự đề xuất** đổi format hay chỉ cảnh báo (đề xuất hiện là chỉ cảnh báo).
3. Gợi ý bằng Ollama hay Hive (đề xuất: Ollama, vì là tác vụ nhẹ và không tốn hạn mức).
