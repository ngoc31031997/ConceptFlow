# CR-050 — Review toàn bộ luồng tạo video: Story → Visual → Hình minh hoạ → Code

## Date
2026-09-29

## Cách review
- Đọc code điều phối: `authoring-service` (`generate_authoring*.go`, `authoring_chain.go`, `render_prompt.go`, `project_illustrations.go`) và `llm-service` (`pipeline/run.py`, `provider.py`, `prompts.py`).
- Đọc các prompt đang active: Visual Director AI (18 132 ký tự), Remotion Engineer AI (28 503 ký tự).
- Số liệu thật: `llm_usage` (authoring DB), `project_errors` và `project_events` (orchestrator DB), cùng storyboard và code thật của project `f7103848` (52 shot).
- Bước render video (TTS, render, ghép) **ngoài phạm vi** review này.

Mức độ: 🔴 lỗi hoặc lãng phí lớn · 🟠 nên sửa · 🟢 nhỏ, hoặc đã ổn.

## 1. Bức tranh chung (chi phí mỗi video)

| Bước | Lượt gọi | Token ra | Trong đó suy nghĩ | Thời gian model | Tỉ lệ lỗi |
|---|---|---|---|---|---|
| Story | 1 | 15–38k | 55–85% | 1–3 phút | 0/8 |
| Visual (storyboard) | 1 (+1 lượt sửa JSON nếu cần) | 22–69k | 45–80% | 2–5 phút | 1/9 |
| Hình minh hoạ | 1 lượt lập danh sách + 1 lượt mỗi hình (4 hình song song) | ~8k mỗi hình | ~80% | vài phút | 0/118 |
| **Code** | **28–74** | **0,26–1,28 triệu** | **83–94%** | **1–2,5 giờ** | **~1/3 số chunk** |

Bước Code chiếm khoảng 95% chi phí và thời gian, và gần như toàn bộ số lỗi. Ba bước trước đều rẻ và ổn định.

## 2. Phát hiện theo từng bước

### Story
- 🟢 **S1** — Một lượt gọi, ổn định (8/8), rẻ. Không cần chia nhỏ: chia theo beat dễ làm lặp ý giữa các beat, là lỗi CR-047 vừa sửa.
- 🟢 **S2** — Bước Code chỉ nhận phần cốt lõi của Story, không nhận cả dàn ý (CR-048 T3). Đã tối ưu.

### Visual (storyboard)
- 🔴 **V1 — Storyboard tả hình bằng văn xuôi. Đây là gốc của chi phí bước Code.**
  - Ví dụ: *"nhân vật đi trên vỉa hè gần trung tâm khung; phía trên đầu nở bong bóng suy nghĩ…"*.
  - Engineer ("DỊCH TỪNG SHOT, KHÔNG SÁNG TÁC", mục A của prompt) phải tự quy ra toạ độ, cỡ, thời điểm và code. Phần lớn token suy nghĩ đi vào việc này.
  - Hướng xử lý: Shot Spec (`cr-050-solution-options.md`, hướng B).
- 🟠 **V2 — Cảnh báo storyboard không chặn bước Code.**
  - Các kiểm tra rẻ của CR-048 (độ dài lời thoại, hình minh hoạ) chỉ là cảnh báo. Chuỗi tự chạy chạy thẳng sang Code, tốn khoảng 1 triệu token, kể cả khi storyboard đang có cảnh báo.
  - Đề xuất: khi có cảnh báo thì chuỗi **dừng lại chờ Creator xác nhận** trước khi sang bước tốn nhất.
- 🟢 **V3** — Một lượt lớn nhưng ổn định (8/9). Chỉ đáng chia theo cảnh nếu đi theo hướng C (đạo diễn viết luôn Shot Spec).

### Hình minh hoạ
- 🔴 **I1 — Sinh lại storyboard thì danh sách hình KHÔNG được lập lại (lỗi).**
  - `Prepare` chỉ lập danh sách khi danh sách đang rỗng (`project_illustrations.go:449-457`).
  - `clearDownstream` của Story/Visual chỉ xoá storyboard và code, không đụng danh sách hình. Danh sách chỉ bị xoá khi xoá project.
  - Giao diện không cảnh báo danh sách đã cũ.
  - Hậu quả: sau khi làm lại Visual, bước Code nhận bộ hình của **storyboard cũ**; vật mới trong storyboard mới không có hình vẽ riêng, trừ khi Creator tự nhớ bấm "Lập lại danh sách".
  - Đề xuất:
    - Lưu hash của storyboard lúc lập danh sách. Nếu storyboard đổi thì đánh dấu danh sách đã cũ.
    - Chuỗi tự lập lại danh sách; hình đã vẽ vẫn nằm trong thư viện và được dùng lại.
    - Giao diện báo rõ danh sách đã cũ.
- 🟢 **I2** — Mỗi hình một lượt, 2 lần thử, dùng lại hình đã có, 4 hình song song: 0 lỗi trên 118 lượt. Đây là mẫu tốt mà bước Code nên học theo.

### Code
- 🔴 **C1 — 83–94% đầu ra là suy nghĩ.** Video 52 shot tốn 1,28 triệu token cho khoảng 40k token code (khoảng 30 lần). Gốc là V1.
- 🔴 **C2 — Khoảng 1/3 số chunk lỗi, và một chunk lỗi làm hỏng cả bước.** Lỗi gồm `empty` 45, `server` 17 (stream đóng rỗng hoặc 500), `timeout` 5, `budget`/`truncated` 3. Hướng xử lý là lưu và chạy lại từng đoạn (câu trả lời vòng 1).
- 🔴 **C3 — Cache chunk dễ mất.**
  - Cache nằm trong RAM của `llm-service`, nên rebuild hoặc restart là mất hết.
  - Khoá cache gồm cả prompt hệ thống, mà danh sách hình thư viện được ghép **vào** prompt hệ thống (`library_section`). Duyệt thêm một hình sau khi chạy lỗi sẽ làm **mọi** chunk phải sinh lại.
- 🟠 **C4 — Số lượt sửa lỗi (149) nhiều hơn số chunk (125).** Code viết lần đầu thường không qua kiểm biên dịch hoặc kiểm bố cục. Loại lỗi nào gây nhiều lượt sửa nhất thì **chưa đo được**, vì chẩn đoán không được lưu (xem X4).
- 🟠 **C5 — Chọn model cho bước Code chưa có rào chắn.**
  - Trên cùng loại việc, glm-5.3-flash ra trung bình 97,9k token và 527 giây mỗi chunk; deepseek-v4.1-flash chỉ 28k token và 201 giây. glm tốn gấp khoảng 3,5 lần token và chậm gấp khoảng 2,6 lần.
  - llama3.2 (Ollama) từng được chọn cho bước Code và timeout cả 5 lần.
  - Đề xuất: bộ chọn model cho bước Code ghi rõ chi phí và thời gian đã đo, cảnh báo hoặc chặn các model không hợp.

### Chung cho cả luồng
- 🔴 **X1 — Chạy lại Story/Visual bị LỖI thì xoá luôn các bước sau.**
  - `clearDownstream` được gọi khi lượt AI lỗi (`generate_authoring.go:453,460,481`). Đây là hành vi có chủ ý từ commit `15a8111`.
  - Nhưng khi lỗi thì nội dung của chính bước đó **không đổi**. Ví dụ: bấm chạy lại Visual, Hive trả 500, và code của video (đáng khoảng 1 triệu token) bị xoá mất dù storyboard vẫn y nguyên.
  - Đề xuất: chỉ xoá các bước sau khi **lưu nội dung mới khác nội dung cũ**. Đường lưu đã làm việc này (`prompt_templates.go:99,141`); bỏ việc xoá khi lỗi.
  - Cần Creator quyết định, vì đây là hành vi đã được yêu cầu trước đó.
- 🟠 **X2 — Run dài không sống qua được một lần restart** (suy ra từ đọc code, chưa thấy xảy ra: 58/58 run authoring trong 10 ngày đều ghi được kết thúc).
  - Trạng thái chuỗi và tiến độ nằm trong RAM. Bước Code ghi `llm_usage` **chỉ sau khi** pipeline trả về (`generate_authoring_code.go:73-88`).
  - Nếu `authoring-service` hoặc `llm-service` bị rebuild giữa một run 1–2 giờ (chính sách rebuild sau mỗi lần sửa khiến việc này dễ xảy ra), thì run mất, và các lượt đã tính tiền **không được ghi**.
  - Đề xuất: ghi `llm_usage` cho từng lượt ngay khi lượt đó xong (llm-service đã stream sự kiện theo chunk), cùng với lưu từng đoạn (C2). Ngoài ra, kiểm tra không có run nào đang chạy trước khi rebuild.
- 🟠 **X3 — Chẩn đoán không được lưu.** Lỗi biên dịch hoặc bố cục của từng vòng sửa chỉ nằm trong log. Không thống kê được luật nào trong prompt gây nhiều lỗi nhất, nên cũng không tối ưu prompt theo số liệu được.
- 🟢 **X4 — Prompt tồn tại hai bản** (thủ công và AI) cho cùng một luật. CR-047 phải sửa cùng một ý ở 5 chỗ. Đây là chi phí bảo trì, không ảnh hưởng token.

## 3. Đề xuất thứ tự làm

| Nhóm | Nội dung | Cỡ việc | Lợi ích |
|---|---|---|---|
| **Sửa ngay** (lỗi rõ) | I1 danh sách hình cũ · X1 không xoá khi lỗi (nếu Creator đồng ý) · C5 rào chắn chọn model | Nhỏ | Chặn mất dữ liệu và code sai hình |
| **CR-050 phần 1** | C2 + C3 + X2: lưu từng đoạn vào DB, chạy lại hoặc dùng AI ngoài từng đoạn, ghi usage từng lượt, UI danh sách đoạn · V2 dừng khi storyboard có cảnh báo · X3 lưu chẩn đoán | Vừa | Hết cảnh "lỗi một chỗ, chạy lại tất cả" |
| **CR-050 phần 2** | V1 + C1 + C4: Shot Spec, spike đo trước (`cr-050-solution-options.md`) | Lớn | Nhắm vào khoảng 90% token hiện đang dùng để suy nghĩ |
