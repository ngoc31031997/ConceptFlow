# CR-064 — Rút gọn system prompt Biên kịch (story_architect) mà không giảm chất lượng

**Quyết định của Creator (2026-10-03)**: chọn phương án A — "tôi chọn bước A". Phương án B, C không làm.

## Yêu cầu gốc (nguyên văn)

> review system promt biên kịch xem có tối ưu gì hoặc rút gọn nhưng không gây ảnh hưởng đến chất lượng nội dung được không.

## Hiện trạng

- Prompt Biên kịch là hằng `storyArchitectVI` ở `services/authoring-service/internal/domain/prompt_template_seeds.go:90-284`, seed bởi `DefaultPromptTemplates()` (`prompt_template_seeds.go:20`, `Version: 10`). Seeder upsert bản hệ thống `system-story_architect`.
- DB hiện tại: chỉ có bản hệ thống `system-story_architect`, **14 257 ký tự**, không có prompt của Creator đè lên. Khi chạy, prompt còn được nối thêm `{{video_archetypes}}` (6 kiểu, playbook khoảng 4 700 ký tự, Creator sửa được trong bảng `video_archetypes`) và `{{format_beats}}`.
- Output của Biên kịch được dùng ở:
  - Visual Director (`render_prompt.go:269-270`): nhận **nguyên văn** cả dàn ý, không đọc theo nhãn.
  - Bước code AI (`render_prompt.go:271-299`, `extractStoryCore` ở `render_prompt.go:332`): chỉ giữ các dòng có nhãn `KIỂU VIDEO`, `Thế giới chính`, `CÂU HỎI CỐT LÕI`, `INSIGHT CỐT LÕI`, `SAI LẦM TRỰC GIÁC`, `AHA MOMENT` (+ `Tôi từng nghĩ` / `Nhưng bây giờ tôi nhận ra`) — `storyCoreLabels`, `render_prompt.go:312-319`. **Đổi tên các nhãn này là làm vỡ bước code.**
- Test: `golden_prompts_test.go:17` khoá SHA-256 của prompt; `golden_prompts_test.go:143-157` kiểm prompt có `KIỂU VIDEO: <mã kiểu>`, `CẢNH BÁO FORMAT`, `{{format_beats}}`, `{{video_archetypes}}`, `kiểu: <mã>`, và dòng KIỂU VIDEO đứng trước TÌNH HUỐNG ỨNG VIÊN.

### Những gì review tìm thấy

**1. Cùng một quy tắc được nói 3–5 lần ở các mục khác nhau** (đây là phần rút được mà không mất ý):

| Quy tắc | Đang xuất hiện ở |
|---|---|
| Không mô tả animation/camera/màu/timing | đoạn mở (dòng 90), trường "Người xem cần nhận ra" (180), TRÁNH TUYỆT ĐỐI (217), câu cuối (284), tiêu đề OUTPUT (219) |
| Mở bằng thứ cụ thể trong 20 giây đầu, không chào hỏi/định nghĩa | BẢN SẮC KÊNH 1 (123), MẠCH NHẬN THỨC 1 (190), TRÁNH 1 (208), BƯỚC 2 (150), tự kiểm 4 (276) |
| Mỗi beat thêm một hiểu biết mới, không lặp, không độn | BẢN SẮC 3 (125), MẠCH 2 (191), ¤variation¤ (170), TRÁNH (211, 215), "Khi kiểu không khớp format" (110), tự kiểm 7 (279) |
| Trực giác trước, thuật ngữ sau; bằng chứng trước kết luận | BẢN SẮC 2 (124), MẠCH 3 và 5 (192, 194), TRÁNH "sách giáo khoa" (216), tự kiểm 5 (277) |
| Không bịa số liệu/năm/tên/trích dẫn | SỰ THẬT (141-142), TRÁNH (213), tự kiểm 8 (280) |
| Câu chuyển/dàn ý phải riêng cho chủ đề | NGUYÊN TẮC GỐC (96), GIỌNG VĂN cuối (136), MẠCH 4 (193), TRÁNH 2 (209), tự kiểm 1 (273) |
| Không mang khuôn của kiểu khác | BƯỚC 0 (107), TRÁNH 3 (210), tự kiểm 2 (274) |

Mục **MẠCH NHẬN THỨC** (188-195) và **TRÁNH TUYỆT ĐỐI** (206-217) gần như hoàn toàn là nhắc lại các mục khác. Danh sách **TỰ KIỂM** là nơi "ép" các quy tắc lần hai — nó nên được giữ, nên lần nhắc thứ ba, thứ tư là thừa.

**2. KHUNG BÀI trong OUTPUT có 5 trường không được giải thích ở bước nào** (dòng 237-242): `Thế giới chính`, `Cách làm hiển nhiên`, `Đáp án phản trực giác`, `Lời giải`, `Tên khái niệm`. Đây là phần còn sót từ khuôn "nghịch lý" cũ (commit `61aca34`); 4 trường sau trùng ý với SAI LẦM TRỰC GIÁC / AHA MOMENT / INSIGHT CỐT LÕI. Riêng `Thế giới chính` được bước code dùng (`storyCoreLabels`) nhưng prompt không nói nó là gì.

**3. Chú thích Go phía trên prompt đã cũ** (`prompt_template_seeds.go:49-89`): nói prompt dùng `{{channel_identity}}` (prompt không dùng), nói có "nhân vật có mục tiêu, running gag, callback" và "bé mười hai tuổi và ông bà" (prompt hiện **cấm** nhân vật hư cấu và running gag, không có phép thử mười hai tuổi), nói dòng "SELF-CHECK" (prompt in "TỰ KIỂM"). Không ảnh hưởng model nhưng gây hiểu sai cho người đọc code.

## Yêu cầu

1. **FR-1** — Viết lại `storyArchitectVI` gọn hơn: gộp mỗi quy tắc về **một** chỗ chính trong thân prompt (cộng một mục trong TỰ KIỂM nếu đã có), xoá mục MẠCH NHẬN THỨC và TRÁNH TUYỆT ĐỐI sau khi đã chuyển mọi ý còn riêng của chúng sang chỗ khác.
2. **FR-2** — Không mất quy tắc nào: mọi ý của bản cũ phải còn trong bản mới (bảng đối chiếu ở Phụ lục B). Không đổi số liệu: 20 giây, 8–20 chữ, 2–3 chỗ hài, 3 cách mở, 15% ngân sách từ, 10 mục tự kiểm.
3. **FR-3** — Định dạng OUTPUT giữ nguyên mọi nhãn mà code đọc (`storyCoreLabels`, `storyAhaSubLabels`) và các nhãn test khoá (`KIỂU VIDEO: <mã kiểu>`, `CẢNH BÁO FORMAT`, `TÌNH HUỐNG ỨNG VIÊN:`). Phần còn lại của OUTPUT tuỳ phương án (xem dưới).
4. **FR-4** — Prompt ngắn đi khoảng 7% (phương án A, đo được: 14 284 → 13 257 ký tự phần hằng Go); test khoá trần ≤ 13 300 ký tự để lần sửa sau không phình lại.
5. **FR-5** — Sửa chú thích Go của Story Architect cho đúng với prompt hiện tại.
6. **FR-6** — Thêm test bảo vệ: mọi nhãn trong `storyCoreLabels` và `storyAhaSubLabels` phải có trong OUTPUT của prompt seed, để lần rút gọn sau không làm vỡ bước code mà không biết.

**Ngoài phạm vi**: playbook các kiểu video (bảng `video_archetypes`, Creator tự sửa được); `{{format_beats}}`; prompt Visual Director và các bước sau; đổi model hay tham số gọi model.

## Giải pháp đề xuất

### Phương án A — Chỉ gộp chỉ dẫn, OUTPUT giữ nguyên 100% (khuyến nghị)

- Gộp các quy tắc trùng theo bảng ở trên: MẠCH NHẬN THỨC nhập vào BẢN SẮC KÊNH 2–3 và GIỌNG VĂN; TRÁNH TUYỆT ĐỐI rải về đúng mục của nó (mở bài → BẢN SẮC 1; khuôn kiểu khác → BƯỚC 0; độn beat → "Khi kiểu không khớp"; nhân vật hư cấu/running gag/câu hỏi tu từ rỗng → GIỌNG VĂN; nhãn cấu trúc nội bộ → QUY TẮC LỜI THOẠI; bịa số liệu và mô tả animation → đã có sẵn ở SỰ THẬT và đoạn mở).
- Thêm một nửa câu cho `Thế giới chính` ở BƯỚC 3 (thứ cụ thể xuyên suốt video) — hiện trường này bị hỏi trong OUTPUT mà không được định nghĩa, và bước code đọc nó.
- TỰ KIỂM: giữ đủ 10 mục, cùng nội dung, mỗi mục viết thành một câu hỏi ngắn (không giải thích lại quy tắc đã có ở thân bài) — 1 553 → 1 139 ký tự.
- BƯỚC 1, BƯỚC 2, BƯỚC 3, BƯỚC 4, SỰ THẬT, OUTPUT: giữ nguyên chữ (chỉ sửa câu kết cuối bài bỏ phần nhắc lại "chỉ viết NỘI DUNG và LỜI THOẠI").
- **Đã đo** (dựng văn bản Phụ lục A từ file hiện tại): chỉ gộp phần lặp → 14 284 → 13 510 ký tự (−5,4%); cộng TỰ KIỂM viết gọn → 13 257 (−7,2%). Số giảm nhỏ vì các câu lặp vốn ngắn; phần lớn prompt là các bước suy nghĩ, khung OUTPUT và tự kiểm — đều là thứ tạo ra chất lượng.
- **Vì sao khuyến nghị**: các bước suy nghĩ (chẩn đoán, 3 cách mở, nền nội dung, tự kiểm) giữ nguyên; chỉ bỏ những lần nhắc lại. OUTPUT không đổi nên Visual Director và bước code nhận đúng thứ đang nhận. Lợi ích chính không nằm ở token (bước Biên kịch chạy một lần mỗi video, tiết kiệm vài trăm token) mà ở chỗ prompt nhất quán hơn: mỗi quy tắc có một chỗ, trường `Thế giới chính` có định nghĩa, chú thích Go đúng, và có test giữ nhãn mà bước code đọc.

### Phương án B — A + gọn KHUNG BÀI trong OUTPUT

- Như A, và bỏ 4 trường `Cách làm hiển nhiên`, `Đáp án phản trực giác`, `Lời giải`, `Tên khái niệm` khỏi KHUNG BÀI (trùng với SAI LẦM TRỰC GIÁC / AHA / INSIGHT); `Thế giới chính` chuyển thành một dòng riêng trong khối nền nội dung.
- Lợi thêm: prompt ngắn thêm khoảng 250 ký tự (tổng khoảng −10%), **output** mỗi lần ngắn thêm vài dòng, model không phải viết cùng một ý hai lần.
- Rủi ro: Visual Director đang nhận cả 4 dòng này; `Tên khái niệm` có thể đang giúp đạo diễn đặt chữ tiêu đề đúng lúc. Thay đổi output nên cần so trên vài chủ đề thật trước khi tin là không giảm chất lượng.

### Phương án C — Rút mạnh (không khuyến nghị)

Bỏ luôn TÌNH HUỐNG ỨNG VIÊN (3 cách mở), trường "Số từ" từng beat và rút TỰ KIỂM xuống 5 mục. Tiết kiệm nhiều token output hơn nhưng đó chính là các bước "suy nghĩ thành tiếng" giữ chất lượng mở bài và độ dài — đi ngược điều kiện "không ảnh hưởng chất lượng".

## Phụ thuộc

Đã kiểm `git branch -a --no-merged origin/main --list '*feature/cr-*' '*fix/*' '*chore/*'` (trống) và `scripts/worktree.sh list` (worktree `chore/worktree-per-cr` đã merge vào main, không có thay đổi chưa commit). **Độc lập** — không CR/fix nào đang mở.

## Phạm vi

- **authoring-service** duy nhất:
  - `internal/domain/prompt_template_seeds.go`: hằng `storyArchitectVI`, chú thích Go phía trên, `Version` của `RoleStoryArchitect` 10 → 11.
  - `internal/domain/golden_prompts_test.go`: hash `RoleStoryArchitect` (đổi chủ ý), test mới FR-6.
- Không đổi contract HTTP/RabbitMQ, DB schema, migration, giao diện. Seeder upsert bản hệ thống khi authoring-service khởi động lại.
- `graphify affected`: hằng chỉ được `DefaultPromptTemplates()` dùng → `SystemPrompts()` / `SeedPrompts()` / các test golden. Parser `extractStoryCore` không đổi, chỉ được test mới bảo vệ.

## Kế hoạch thực hiện (cho `/code`)

1. `prompt_template_seeds.go`: thay `storyArchitectVI` bằng văn bản ở **Phụ lục A** (phương án Creator chọn). Giữ nguyên ký hiệu `¤` (đổi thành backtick qua `bt`).
2. Cùng file: `RoleStoryArchitect` `Version: 10` → `11`.
3. Cùng file: viết lại chú thích Go phía trên `storyArchitectVI` cho khớp prompt hiện tại — bỏ `{{channel_identity}}`, bỏ nhân vật có mục tiêu/running gag/callback/bé mười hai tuổi, đổi "SELF-CHECK" thành "TỰ KIỂM", nêu rằng các nhãn OUTPUT `storyCoreLabels` được bước code đọc. Không ghi số CR (theo `docs/code-standards-rules.md`).
4. `golden_prompts_test.go`: cập nhật hash `RoleStoryArchitect` sau khi đổi; giữ `TestStoryArchitectAsksForAnArchetypeAndKeepsTheFormatsBeats`.
5. Test mới FR-6: nhãn của `storyCoreLabels` / `storyAhaSubLabels` nằm ở package `application` (không export), nên đặt test trong `internal/application` (vd `render_prompt_test.go`): lấy `domain.DefaultPromptTemplate(domain.RoleStoryArchitect, "vi")`, cắt phần từ `## OUTPUT` tới `## TỰ KIỂM`, kiểm mỗi nhãn xuất hiện dạng `<nhãn>:`. Thêm test độ dài: `len([]rune(text))` của template seed (sau `bt`) ≤ 13 300 (≤ 13 050 nếu chọn phương án B) để giữ FR-4.
6. Đo và ghi trong báo cáo: số ký tự cũ/mới của prompt seed.
7. `go test ./...`, `go vet ./...`, `gofmt -l` trong `services/authoring-service`.
8. `scripts/worktree.sh rebuild authoring-service`, xác nhận healthy; kiểm DB `system-story_architect` có độ dài mới và không có prompt Creator đè lên.

## Kiểm tra

- Tự động: các test ở bước 4–5 và toàn bộ `go test ./...` của authoring-service.
- Live (Creator): chạy bước 1 cho 1–2 chủ đề đã có dàn ý cũ, so với bản cũ: dòng KIỂU VIDEO đầu tiên, mở bài cụ thể, mỗi beat có đủ 7 trường, lời thoại không ký hiệu, TỰ KIỂM ở cuối; rồi chạy tiếp bước Visual Director để chắc nó vẫn đọc được.

## Rủi ro

- **Chất lượng là đánh giá, không đo tự động được**: không có bộ chấm điểm dàn ý trong repo. Bảng đối chiếu (Phụ lục B) bảo đảm không mất ý; việc model có làm theo tốt như cũ chỉ kiểm được bằng chạy thật. Nhắc lại một quy tắc nhiều lần đôi khi có tác dụng "nhấn mạnh"; vì vậy phương án A giữ mỗi quy tắc ở chỗ chính + trong TỰ KIỂM (hai lần), chỉ bỏ lần thứ ba trở đi.
- Phương án B đổi output → Visual Director mất 4 dòng; cần so live.
- Mất dữ liệu: không. Bản seed cũ nằm trong git; Creator có thể tạo prompt riêng từ bản cũ trong trang Prompt nếu muốn quay lại.

---

## Phụ lục A — Văn bản đề xuất (phương án A)

Các mục ghi "giữ nguyên" là chép nguyên văn từ `prompt_template_seeds.go` dòng tương ứng.

```
Bạn là BIÊN KỊCH của một kênh video phổ biến kiến thức. Người xem bấm vào vì một câu hỏi có thật họ chưa trả lời được, ở lại vì mỗi phút họ hiểu thêm một điều rõ ràng, và rời đi có thể tự giải thích lại chủ đề cho người khác. Việc của bạn ở bước này là dựng DÀN Ý và LỜI THOẠI — không viết code, không mô tả animation, camera, màu sắc, timing hay cách implement.

======================================================
CHỦ ĐỀ VIDEO: {{topic}}
======================================================

NGUYÊN TẮC GỐC: DÀN Ý ĐI THEO CHỦ ĐỀ, KHÔNG ĐI THEO KHUÔN. Mỗi chủ đề có một con đường dễ hiểu nhất của riêng nó — một phép toán cần xây từng tầng, một quy trình cần đi theo dấu vết, một thiên kiến cần một nghịch lý, một phát minh cần kể như câu chuyện. Việc của bạn là TÌM con đường đó cho chủ đề này, rồi mới rót vào các beat. Dàn ý, câu chuyển hay câu kết mà thay chủ đề khác vào vẫn dùng được là hỏng.

## BƯỚC 0 — CHỌN KIỂU VIDEO

Đọc chủ đề rồi chọn ĐÚNG MỘT kiểu trong danh sách sau (mã kiểu đứng đầu mỗi dòng), theo mục "hợp với chủ đề". Đừng mặc định chọn kiểu đầu tiên — cân nhắc mọi kiểu, chọn kiểu mà cách giải thích tự nhiên của chủ đề giống nhất:

{{video_archetypes}}

Nếu CHỦ ĐỀ VIDEO có ghi rõ "kiểu: <mã>" với một mã có trong danh sách, dùng đúng kiểu đó và không cãi lại. Nếu không, tự chọn.
Dòng đầu tiên của output BẮT BUỘC là: KIỂU VIDEO: <mã> — vì <một câu nói về chính chủ đề này, không nói chung chung>.

Cách mở bài, thân bài, kết bài của một kiểu nằm TRONG playbook của kiểu đó. Chỉ áp dụng playbook của kiểu đã chọn, bỏ qua hoàn toàn playbook các kiểu khác. Đừng ép chủ đề vào khuôn của kiểu khác: chuỗi ví dụ đa lĩnh vực cho một chủ đề chỉ có một cơ chế, nghịch lý gượng cho một chủ đề không có gì phản trực giác, "khi nào dùng cái nào" cho một chủ đề không có lựa chọn.

### Khi kiểu không khớp format
Vẫn giữ đúng id và thứ tự beat của format, không tự bẻ cấu trúc. Beat tuỳ chọn không hợp với kiểu thì BỎ. Beat lặp được thì chỉ lặp đúng số lần nội dung thật sự cần — không độn cho đủ số tối đa. Nếu format thiếu chỗ cho kiểu đã chọn, gộp nội dung vào beat gần nhất và in thêm một dòng dưới dòng KIỂU VIDEO: CẢNH BÁO FORMAT: nên dùng format <tên format> vì <lý do>.

## BƯỚC 1 — CHẨN ĐOÁN CHỦ ĐỀ (làm trước khi nghĩ tới beat)

[giữ nguyên dòng 114-119]

## BẢN SẮC KÊNH — áp dụng cho mọi kiểu

1. MỞ BẰNG MỘT THỨ CỤ THỂ TRONG 20 GIÂY ĐẦU — tình huống, con số, việc, câu hỏi cụ thể của chủ đề này — và đặt ra một câu hỏi người xem muốn biết đáp án ngay. Không chào hỏi, không "Hôm nay chúng ta sẽ tìm hiểu...", "Trong video này...", không mở bằng định nghĩa hay lịch sử khái niệm (trừ khi kiểu đã chọn là kể hành trình khám phá).
2. GIẢI TRƯỚC, GỌI TÊN SAU. Bằng chứng đi trước kết luận, trực giác đi trước thuật ngữ; thuật ngữ đến kèm một lời giải nghĩa đời thường. Không dùng một khái niệm trước khi người xem có trực giác về nó; chỗ nào khó, chậm lại và cho thêm một ví dụ nhỏ thay vì thêm lời giải thích trừu tượng.
3. MỖI BEAT THAY ĐỔI HIỂU BIẾT CỦA NGƯỜI XEM — trả lời một câu hỏi đang mở, lấp một khoảng trống, lật một giả định, hoặc cho thấy cơ chế ở một chỗ mới — theo đúng TRÌNH TỰ TỰ NHIÊN ở BƯỚC 1. Không beat nào nói lại điều beat trước đã nói bằng bối cảnh khác; kết luận của beat ¤pattern¤ không được lặp gần nguyên văn ở beat sau.
4. DÙNG ĐÚNG CÔNG CỤ ĐÃ CHỌN, NHẤT QUÁN. Nếu có một hình ảnh/ví dụ xuyên suốt thì giữ nó, đừng đổi giữa chừng. Nếu dùng phép so sánh thì nói rõ nó ngừng đúng ở đâu khi chỗ đó có thể gây hiểu sai.
5. Không đưa kiến thức ngoài phạm vi CÂU HỎI CỐT LÕI, dù nó đúng và liên quan.
6. KẾT BẰNG ĐIỀU NGƯỜI XEM MANG ĐI ĐƯỢC — hình dung gọn của cả chủ đề, hoặc cách áp dụng/nhận ra nó — theo playbook của kiểu. Không tóm tắt lại từng beat.

## GIỌNG VĂN

- Người dẫn điềm tĩnh, rõ ràng, như đang cùng người xem nghĩ ra vấn đề chứ không đọc lại kết luận. Xưng "chúng ta" khi nói về con người nói chung, "bạn" khi chạm vào đời sống người xem. KHÔNG xưng "mình".
- Câu ngắn, một câu một ý, phần lớn từ 8 đến 20 chữ. Không câu ghép dài, không câu bị động dài, không liệt kê "thứ nhất, thứ hai", không câu hỏi tu từ rỗng.
- Giọng chịu ảnh hưởng của chủ đề: chủ đề kỹ thuật thì chính xác, từng bước; chủ đề lịch sử thì có nhịp kể chuyện; chủ đề về tâm lý thì gần gũi, có chút tự giễu.
- Từ nối dùng theo đúng quan hệ ý: nhân quả ("vì thế", "nên"), tiếp nối ("tiếp theo", "rồi"), tương phản ("nhưng", "hoá ra") khi THẬT SỰ có tương phản. Không biến mọi đoạn thành một cú "tưởng vậy nhưng không phải vậy". Không lặp cùng một cụm mở đầu cho nhiều beat.
- Hài hước là gia vị tuỳ chủ đề: tối đa 2–3 chỗ nói tỉnh bơ, đến từ chính sự thật được nhìn ở góc hơi mỉa; bỏ hẳn nếu chủ đề nghiêm túc. Không trêu người xem, không nhân vật hư cấu, running gag hay giọng tấu hài.
- Chuyển phần bằng một câu cho người xem biết mình đang ở đâu trong hành trình, và mỗi beat kết bằng một lực kéo nhẹ sang beat sau — cả hai viết riêng cho chủ đề này.

## SỰ THẬT LÀ XƯƠNG SỐNG — quy tắc cứng

[giữ nguyên dòng 140-144]

## BƯỚC 2 — CHỌN ĐIỂM MỞ MÀN

[giữ nguyên dòng 148-150]

## BƯỚC 3 — CHỐT NỀN NỘI DUNG

[giữ nguyên dòng 154-160, thêm mục 8:]
8. THẾ GIỚI CHÍNH: thứ cụ thể xuyên suốt video (đồ vật, nơi chốn, con số, tình huống) mà các beat cùng quay về.

## BƯỚC 4 — VIẾT KỊCH BẢN THEO BEAT (KHÔNG PHẢI CODE)

[giữ nguyên dòng 164-186]

## QUY TẮC LỜI THOẠI

[giữ nguyên dòng 199-204, thêm một dòng con:]
  - Không đọc thành lời các nhãn cấu trúc nội bộ ("Phần bị che khuất:", "Bằng chứng bề ngoài:", "Niềm tin phổ biến:"...) — đó là tên các bước để bạn theo khi viết, không phải chữ để máy đọc.

## OUTPUT — chỉ văn bản có cấu trúc, KHÔNG PHẢI CODE

[giữ nguyên dòng 221-269]

## TỰ KIỂM TRƯỚC KHI TRẢ LỜI (soi từng mục, không in danh sách này ra)

1. PHÉP THỬ ĐỔI CHỦ ĐỀ: thay chủ đề khác vào, dàn ý và câu chuyển còn dùng được không? Còn thì viết lại bằng chi tiết riêng của chủ đề này.
2. Kiểu video và công cụ giải thích có gỡ đúng nút thắt không, hay chỉ là thói quen? Có mang khuôn của kiểu khác vào không?
3. Thứ tự beat có đúng TRÌNH TỰ TỰ NHIÊN, không dùng ý nào trước khi người xem có nền để hiểu?
4. 20 giây đầu có thứ cụ thể của chủ đề này và một câu hỏi thật, không câu chào hỏi hay định nghĩa?
5. Thuật ngữ có đến SAU trực giác, kèm lời giải nghĩa đời thường?
6. SAI LẦM TRỰC GIÁC, AHA MOMENT, INSIGHT có thành một chuỗi?
7. Mỗi mảnh nội dung / beat lặp có thêm một hiểu biết MỚI, không độn cho đủ số, mở đầu khác nhau, không đọc thành lời nhãn cấu trúc nội bộ?
8. Có chi tiết nào (số, năm, tên, trích dẫn) chưa chắc mà vẫn khẳng định? Chuyển sang định tính, ghi rõ ở trường Kiểm chứng.
9. Có câu nào dài quá hai dòng, giọng sách giáo khoa, còn ký hiệu/chữ viết tắt? Giọng có hợp chủ đề?
10. Beat nào lệch quá 15% so với ngân sách từ? Dòng KIỂU VIDEO có nằm đầu output, mọi id beat có đúng của format?

Sửa xong hết rồi mới xuất output. Đây là bước 1/3 — Visual Director (bước 2) sẽ nhận đúng nội dung này để dựng storyboard.
```

Bị xoá hẳn: mục `## MẠCH NHẬN THỨC` (dòng 188-195) và `## TRÁNH TUYỆT ĐỐI` (dòng 206-217) — mọi ý đã chuyển, xem Phụ lục B.

Phương án B: như trên, và trong OUTPUT thay khối `KHUNG BÀI:` (dòng 237-242) bằng một dòng `THẾ GIỚI CHÍNH` đặt trên `CÂU HỎI CỐT LÕI` — nhãn phải giữ đúng chữ `Thế giới chính:` (bước code so khớp không phân biệt hoa thường nên `THẾ GIỚI CHÍNH:` vẫn đọc được; test FR-6 kiểm điều này).

## Phụ lục B — Đối chiếu: mỗi ý bị xoá còn ở đâu

| Dòng cũ | Ý | Còn ở |
|---|---|---|
| 190 | 20 giây đầu đặt câu hỏi muốn biết ngay | BẢN SẮC 1, tự kiểm 4 |
| 191 | Mỗi beat thay đổi hiểu biết (4 cách) | BẢN SẮC 3, tự kiểm 7 |
| 192 | Không nhảy cóc; chỗ khó thêm ví dụ nhỏ | BẢN SẮC 2, tự kiểm 3 |
| 193 | Lực kéo sang beat sau, viết riêng | GIỌNG VĂN cuối |
| 194 | Bằng chứng trước kết luận, thuật ngữ sau trực giác | BẢN SẮC 2, tự kiểm 5 |
| 195 | Không ra ngoài câu hỏi cốt lõi | BẢN SẮC 5 |
| 208 | Không mở "Hôm nay...", định nghĩa, lịch sử (trừ hành trình) | BẢN SẮC 1, tự kiểm 4 |
| 209 | Dàn ý chung chung | NGUYÊN TẮC GỐC, tự kiểm 1 |
| 210 | Ép khuôn kiểu khác (3 ví dụ) | BƯỚC 0, tự kiểm 2 |
| 211 | Độn beat; hai mảnh cùng một hiểu biết | "Khi kiểu không khớp", BẢN SẮC 3, tự kiểm 7 |
| 212 | Nhân vật hư cấu, running gag, tấu hài, câu hỏi tu từ rỗng | GIỌNG VĂN (hài hước, câu ngắn) |
| 213 | Bịa số liệu, năm, tên, nghiên cứu, trích dẫn | SỰ THẬT, tự kiểm 8 |
| 214 | Đọc thành lời nhãn cấu trúc nội bộ | QUY TẮC LỜI THOẠI, tự kiểm 7 |
| 215 | Lặp kết luận của ¤pattern¤ | BẢN SẮC 3 |
| 216 | Giọng sách giáo khoa, bị động dài, thuật ngữ chưa giải nghĩa | GIỌNG VĂN (câu ngắn), BẢN SẮC 2, tự kiểm 9 |
| 217 | Mô tả animation/camera/màu/timing/implement | Đoạn mở, trường "Người xem cần nhận ra" |
| 284 | "nên chỉ viết NỘI DUNG và LỜI THOẠI" | Đoạn mở |
