# CR-018 — `self.narrate()` và render hai lượt thay cho marker comment (P1)

## Date
2026-09-09

## Stage
Requirements Analysis (Change Request)

## Intent Analysis
- **Request type**: Migration — đổi cách narration được khai báo và phát hiện
- **Scope estimate**: 3 unit — `script-processing`, `rendering`, `web-gui`; `orchestrator` không đổi hình dạng dữ liệu
- **Complexity estimate**: Complex. Đây là thay đổi kiến trúc lớn nhất trong nhóm CR này, và là thay đổi phá vỡ tương thích với mọi script đã viết

## Bối cảnh — quan sát được
Narration hiện là **comment tĩnh**, và mọi hạn chế phía sau đều bắt nguồn từ đó.
Chuỗi nhân quả, đã xác minh trong code:

1. `manim_script_parser.py` quét text tìm `# NARRATION: "..."` bằng regex, theo
   thứ tự dòng trong file. Nó không bao giờ chạy script.
2. Vì parser chỉ đọc text, `manim_renderer.py::_patch_auto_waits` buộc phải yêu
   cầu **số `self.wait(AUTO)` khớp tuyệt đối** số marker, nếu không thì
   `AnimationEngineError`.
3. Vì phải khớp tuyệt đối theo thứ tự file, `_read_wait_offsets` phải bác bỏ
   mọi script mà một `wait(AUTO)` chạy khác một lần — docstring ghi rõ: *"nó
   không thể nằm trong vòng lặp hay điều kiện"*.
4. Vì narration không được nằm trong hàm hay vòng lặp, **hook và CTA không thể
   là component**. CR-006 §Quyết định #2 đã phải lùi FR17 xuống thành "snippet
   Creator tự chèn", và nêu đúng lý do: tự chèn scene *"là cách dễ nhất phá vỡ
   bất biến này"*.
5. Và vì Creator/AI phải tự đếm tay, prompt phải dành hẳn một mục tự kiểm tra
   ("đánh số thứ tự 1, 2, 3... rồi đếm riêng số `self.wait(AUTO)`"), còn
   `scriptValidation.ts` phải tồn tại chỉ để bắt lỗi lệch đếm — được prompt mô tả
   là **lỗi thường gặp nhất**.

Toàn bộ năm điểm trên là hệ quả của một lựa chọn duy nhất: **đếm text thay vì
chạy code**.

## Vấn đề
Bất biến "số marker == số wait" không bảo vệ điều gì có giá trị tự thân. Nó là
cái giá phải trả cho việc parser không dám chạy script. Cái giá đó gồm:

- Lớp lỗi phổ biến nhất của hệ thống, phải phòng bằng ba lớp kiểm tra rời rạc
  (prompt, GUI, renderer) mà vẫn lọt.
- Không thể đóng gói narration vào component ⇒ không thể ép cấu trúc video
  (hook/CTA/recap) bằng code, chỉ có thể khuyên bằng chữ.
- Narration không được nằm trong vòng lặp, dù đó là cách tự nhiên để thuyết minh
  một thuật toán lặp — đúng thể loại nội dung mà kênh này nhắm tới.

## Quyết định
Thay marker comment + `self.wait(AUTO)` bằng **một lời gọi duy nhất**:

```python
self.narrate("Ba phần của vòng lặp for: khởi tạo, điều kiện, bước nhảy.")
```

và đổi render thành **hai lượt**:

| Lượt | Việc | Chi phí |
|---|---|---|
| **1 — dry** | Chạy scene với `narrate()` = ghi text ra JSONL rồi `wait(0)`, không encode video | Rẻ; và đây chính là bước dry-run compile mà CR-020 cần — gộp làm một |
| *(giữa)* | TTS trên danh sách thu được | Không đổi |
| **2 — thật** | Render bình thường, `narrate()` tra duration theo thứ tự và `wait()` đúng số giây, đồng thời ghi mark như `_cf_mark` hiện nay | Không đổi |

Điểm mấu chốt: danh sách narration lấy theo **thứ tự chạy thật**, không phải thứ
tự dòng trong file. Bất biến "khớp tuyệt đối" biến mất vì nó không còn cần thiết
— hai lượt chạy cùng một mã, nên số lượng và thứ tự tự khớp theo cấu trúc.

## Được gì
- Lớp lỗi "lệch số lượng" **không còn tồn tại về mặt cấu trúc**. Gỡ được mục tự
  kiểm tra trong prompt, gỡ được phần đếm trong `scriptValidation.ts`, gỡ được
  nhánh lỗi trong `_patch_auto_waits`.
- Narration được phép nằm trong `for`, trong `if`, trong hàm helper ⇒ **hook,
  CTA, recap trở thành component thật** (CR-019 dựa hẳn vào điều này).
- Không còn phải sinh mã bằng thay thế chuỗi. Hiện `_patch_auto_waits` biến
  `self.wait(AUTO)` thành `(_cf_mark(self, i), self.wait(D))` — script lưu trong
  DB **không phải Python hợp lệ** cho tới khi được vá. Sau CR này script là mã
  chạy được nguyên trạng.

## Functional Requirements

### FR48 — API `narrate`
- **FR48.1**: `ConceptFlowScene` PHẢI cung cấp `narrate(text: str)`: phát ra một
  đoạn narration tại đúng vị trí đó trong dòng chảy của scene.
- **FR48.2**: Ở lượt thật, `narrate` PHẢI dừng scene đúng bằng thời lượng audio
  thật của đoạn đó, và ghi lại offset bắt đầu — giữ nguyên đảm bảo đồng bộ mà
  CR-002 đã thiết lập (offset là thời điểm **bắt đầu** chờ).
- **FR48.3**: `narrate` PHẢI dùng được bên trong vòng lặp, nhánh điều kiện và
  hàm helper. Đây là mục đích của CR, không phải hiệu ứng phụ.
- **FR48.4**: `# CHAPTER: "..."` PHẢI có API tương ứng (`self.chapter("...")`)
  gắn vào narration kế tiếp, giữ nguyên ngữ nghĩa CR-006 FR15.
- **FR48.5**: `self.wait(số cụ thể)` PHẢI tiếp tục dùng được cho khoảng lặng
  không lời thoại. Không đụng tới.

### FR49 — Render hai lượt
- **FR49.1**: `rendering` PHẢI chạy lượt dry trước khi TTS, thu danh sách
  narration theo thứ tự chạy thật, và phát ra như kết quả của bước phân tích script.
- **FR49.2**: Lượt dry KHÔNG được encode video và PHẢI có giới hạn thời gian
  riêng, ngắn hơn nhiều so với `RENDER_TIMEOUT_SECONDS` — một script treo phải
  lộ ra ở lượt rẻ, không phải lượt đắt.
- **FR49.3**: Lượt dry PHẢI chịu **cùng** mức cách ly như lượt thật: subprocess,
  môi trường đã lược bỏ credential, giới hạn RLIMIT_AS. Đây là mã chưa được kiểm
  chứng chạy sớm hơn trong pipeline, không phải an toàn hơn.
- **FR49.4**: Nếu lượt dry và lượt thật cho số narration khác nhau (script không
  tất định — dùng random, thời gian thực...), render PHẢI fail với thông báo nêu
  đúng nguyên nhân đó. Đây là bất biến thay thế cho bất biến cũ, và nó bắt đúng
  loại script thực sự nguy hiểm.
- **FR49.5**: Duration truyền vào lượt thật PHẢI được làm tròn về bội số của
  `1/fps`. Manim đánh cache theo hash nội dung từng segment; duration lẻ tới
  micro-giây khiến mọi segment phía sau một chỉnh sửa nhỏ đều đổi hash và mất
  cache — đúng thứ `CACHE_ROOT` sinh ra để tránh (đo được: render lại nhanh gấp ~5 lần).

## Quyết định đã chốt (2026-09-10)
**FR50 (công cụ chuyển đổi + cửa sổ song song hai chuẩn) đã được gỡ bỏ khỏi CR
này.** Kênh chưa xuất bản video nào và không có kho script cũ cần giữ chạy được,
nên toàn bộ phần tương thích ngược là công sức bảo vệ một thứ không tồn tại.
Chuyển đứt điểm sang `self.narrate()`: bỏ luôn đường xử lý `# NARRATION` +
`self.wait(AUTO)` thay vì duy trì song song.

Đây là thay đổi lớn nhất về mặt rủi ro của CR: phần "phá vỡ tương thích" ở mục
Rủi ro không còn áp dụng, và khối lượng giảm khoảng một phần ba. **Cửa sổ này sẽ
đóng lại** — mỗi video xuất bản theo chuẩn cũ đều làm chi phí chuyển đổi tăng lên.

## Non-goals
- Không giữ tương thích với chuẩn `# NARRATION` + `self.wait(AUTO)`. Xem mục
  Quyết định đã chốt.
- Không đổi hình dạng event giữa các service. `narration_segments`, offset,
  chapters giữ nguyên schema — đây là thay đổi bên trong `rendering` và
  `script-processing`.
- Không chạy script trong sandbox mạnh hơn (seccomp, container-per-render). Mức
  cách ly giữ nguyên như hiện nay, chỉ áp thêm cho lượt dry.
- Không bỏ `_cf_mark`. Cơ chế đo offset của CR-002 vẫn đúng và vẫn dùng.

## Rủi ro
- **Model chưa quen API mới.** Giống rủi ro của CR-017 và cộng dồn với nó. Nên
  đo chất lượng script sinh ra sau khi cả hai CR xong, không đo riêng lẻ.
- **Lượt dry không tất định** với script dùng `random`. FR49.4 biến việc này
  thành lỗi rõ ràng thay vì lệch âm thầm, nhưng cần nói trước trong tài liệu.
- **Tăng tổng thời gian pipeline** thêm một lượt chạy scene. Bù lại CR-020 cần
  đúng lượt đó, nên chi phí ròng gần bằng không nếu hai CR làm cùng nhau.

## Câu hỏi cần Creator chốt
1. ~~Chấp nhận phá vỡ tương thích ở mức nào?~~ → Đã chốt: chuyển đứt điểm.
2. ~~Có bao nhiêu script cũ cần giữ chạy được?~~ → Đã chốt: không có.
3. CR-018 và CR-020 làm chung một đợt (chia sẻ lượt dry) hay tách? → Đề xuất:
   **chung**.

## Kiểm chứng
- Unit test: `narrate` trong vòng lặp / trong hàm helper cho ra đúng số narration
  và đúng thứ tự.
- Unit test: lượt dry và lượt thật lệch số narration ⇒ lỗi nêu đúng nguyên nhân.
- Unit test: duration làm tròn theo fps; cache Manim trúng khi chỉ sửa narration cuối.
- E2E: một script chuẩn mới cho ra video có narration khớp hình đúng như chuẩn
  cũ (so sánh offset với bản render trước khi đổi).
- Regression: script chuẩn cũ vẫn render đúng qua đường tương thích.

## Liên quan
- CR-002 (offset đo thật — cơ chế được giữ nguyên)
- CR-006 §Quyết định #2 (lý do hook/CTA phải lùi thành snippet — CR này gỡ bỏ lý do đó)
- CR-017 (`ConceptFlowScene` là nơi `narrate` sống)
- CR-019 (người hưởng lợi trực tiếp: hook/CTA thành component)
- CR-020 (dùng chung lượt dry)
- ADR sẽ cần: "Narration khai báo bằng lời gọi runtime, phát hiện bằng render hai lượt"
