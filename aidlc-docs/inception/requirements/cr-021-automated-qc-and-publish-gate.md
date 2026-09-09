# CR-021 — Chấm chất lượng video tự động và cổng chặn publish (P2)

## Date
2026-09-09

## Stage
Requirements Analysis (Change Request)

## Intent Analysis
- **Request type**: New Feature — kiểm tra artifact đầu ra, thứ hệ thống hiện không làm chút nào
- **Scope estimate**: 3 unit — `quality-service` (bước QC mới), `rendering` (xuất dữ liệu bố cục), `web-gui` (báo cáo QC + chặn publish)
- **Complexity estimate**: Moderate. Cơ chế thu dữ liệu đã có sẵn; phần khó là chọn ngưỡng chấm điểm

## Bối cảnh — quan sát được
Hệ thống hiện **không có bất kỳ kiểm tra nào lên nội dung của video đã dựng**.
`manim` trả về exit code 0 nghĩa là "thành công", kể cả khi:

- chữ tràn ra ngoài khung,
- hai object đè lên nhau,
- chữ tối trên nền tối,
- hình đứng yên nhiều giây trong lúc đang thuyết minh.

Prompt có dặn *"bố cục nằm gọn trong khung an toàn 16:9, không để chữ/hình tràn
hoặc chồng lấp"* — lại là một câu chữ không ai kiểm chứng.

**Nhưng cơ chế để kiểm tra thì đã tồn tại sẵn.** `manim_renderer.py::MARK_PREAMBLE`
chèn hàm `_cf_mark` chạy **bên trong** scene tại mỗi mốc narration, hiện chỉ ghi
`scene.renderer.time`. Đúng tại điểm đó, `scene.mobjects` đang nằm trong tầm tay.

Phía âm thanh, đã xác minh hai điểm:

- `LOUDNORM_FILTER` chạy **một lượt** (`ffmpeg_assembler.py` dòng 254). Chế độ
  một lượt là dynamic normalization, kết quả thực tế thường lệch khỏi đích
  -14 LUFS. Không có bước nào đo lại.
- Chồng lấn narration giữa video không được kiểm. `_target_duration` đã xử lý
  đúng trường hợp narration **cuối** chạy quá animation (lấy max, `tpad` giữ
  khung cuối), nhưng nếu một đoạn audio dài hơn khoảng chờ dành cho nó ở **giữa**
  video, `amix` sẽ trộn nó với đoạn kế tiếp — hai giọng nói chồng lên nhau.
  Rủi ro này cao nhất ở nhánh tắt TTS, nơi thời lượng là **ước lượng** chứ không
  phải số đo.

## Vấn đề
Creator là cơ chế QC duy nhất, và họ phải xem lại toàn bộ video mỗi lần. Với một
kênh sản xuất đều đặn, đó là khâu không mở rộng được và là khâu đầu tiên bị bỏ
qua khi vội — đúng lúc rủi ro cao nhất.

## Quyết định
Thêm bước `qc_video` sau `assemble_video`, chạy trong `quality-service` (CR-020).
Chấm điểm theo **hai nguồn dữ liệu, đều tất định, không cần AI**:

1. **Bố cục** — mở rộng `_cf_mark` để ghi thêm bounding box, màu và cỡ chữ của
   các mobject đang hiện tại mỗi mốc narration. Từ đó kiểm tràn khung, chồng
   lấn, chữ quá nhỏ, tương phản kém — bằng số học thuần, chạy trong mili giây.
2. **Âm thanh và đóng gói** — đo lại bằng ffmpeg sau khi ghép.

Kết quả là một **báo cáo QC** gắn vào project, hiện ở `ResultPage` trước nút
publish. Lỗi mức blocking chặn publish; cảnh báo thì không.

Việc chấm frame bằng mô hình thị giác (VLM qua Ollama) **không thuộc phạm vi CR
này** — nó không tất định, và giá trị chưa được chứng minh. Chỉ cân nhắc sau khi
lớp số học đã chạy và cho thấy nó bỏ sót cái gì.

## Functional Requirements

### FR58 — Thu dữ liệu bố cục lúc render
- **FR58.1**: Preamble mà `rendering` chèn vào script PHẢI ghi thêm, tại mỗi mốc
  narration: bounding box, màu và cỡ chữ của các mobject đang hiển thị.
- **FR58.2**: Việc thu dữ liệu PHẢI là best-effort — lỗi khi ghi KHÔNG được làm
  hỏng lượt render. Đây là dữ liệu QC, không phải sản phẩm.
- **FR58.3**: Chi phí thêm PHẢI không đáng kể so với thời gian render. Nếu vượt
  quá vài phần trăm, phải giảm tần suất lấy mẫu chứ không bỏ tính năng.
- **FR58.4**: Dữ liệu PHẢI ghi cùng chỗ với `cf_marks.jsonl` hiện nay, đi qua
  đúng kênh đã có (`CF_MARKS_PATH`), không mở thêm đường ra khỏi subprocess.

### FR59 — Chấm bố cục
- **FR59.1**: PHẢI phát hiện **tràn khung**: bounding box vượt ra ngoài khung
  Manim, hoặc lấn vào safe margin do theme của CR-017 định nghĩa.
- **FR59.2**: PHẢI phát hiện **chồng lấn** giữa các mobject dạng chữ.
- **FR59.3**: PHẢI phát hiện **chữ quá nhỏ**: cỡ chữ quy đổi ra pixel ở độ phân
  giải xuất thấp hơn ngưỡng đọc được trên màn hình điện thoại.
- **FR59.4**: PHẢI phát hiện **tương phản kém** giữa màu chữ và màu nền.
- **FR59.5**: PHẢI phát hiện **hình chết**: khoảng giữa hai mốc narration dài
  hơn ngưỡng mà không có animation nào chạy.
- **FR59.6**: Mỗi phát hiện PHẢI kèm timestamp trong video, để Creator tua thẳng
  tới chỗ đó thay vì tự dò.

### FR60 — Chấm âm thanh và đóng gói
- **FR60.1**: PHẢI đo LUFS thật của video đã ghép và báo sai lệch so với đích
  -14. Nếu sai lệch vượt ngưỡng chấp nhận được, `loudnorm` PHẢI chuyển sang hai
  lượt (đo trước, nạp `measured_*` vào lượt sau).
- **FR60.2**: PHẢI phát hiện **chồng lấn narration**: một đoạn audio kéo dài quá
  điểm bắt đầu của đoạn kế tiếp. Đây là lỗi nghe thấy được ngay và hiện không
  có gì bắt.
- **FR60.3**: PHẢI phát hiện clipping.
- **FR60.4**: PHẢI kiểm cue phụ đề chồng thời gian nhau.
- **FR60.5**: PHẢI kiểm các thuộc tính bắt buộc để phát hành: độ phân giải,
  framerate, `yuv420p`, `+faststart`. Chúng đã được đặt đúng trong
  `VIDEO_ENCODE_ARGS`/`CONTAINER_ARGS`, nhưng nhánh `-c:v copy` bỏ qua bước
  encode nên không có gì đảm bảo file cuối vẫn đạt.

### FR61 — Báo cáo QC và cổng publish
- **FR61.1**: Kết quả QC PHẢI được lưu cùng project dưới dạng dữ liệu đọc được
  bằng máy, không chỉ là dòng log.
- **FR61.2**: `ResultPage` PHẢI hiển thị báo cáo trước nút publish, nhóm theo
  mức độ, mỗi mục có timestamp bấm được để tua tới đúng chỗ.
- **FR61.3**: Lỗi mức blocking PHẢI chặn Saga Publish. Creator PHẢI bỏ qua được
  bằng một hành động có ý thức, và lần bỏ qua đó PHẢI được ghi lại.
- **FR61.4**: QC thất bại vì lý do kỹ thuật (thiếu dữ liệu, ffmpeg lỗi) KHÔNG
  được chặn publish — chỉ báo là không chấm được. Một cổng hỏng không được biến
  thành cổng khoá.
- **FR61.5**: Ngưỡng chấm PHẢI cấu hình được, không hardcode.

## Non-goals
- Không chấm frame bằng mô hình thị giác trong CR này.
- Không tự sửa lỗi bố cục. Hệ thống báo, Creator sửa script.
- Không chấm chất lượng **nội dung** (giải thích hay hay dở). Ngoài tầm.
- Không đụng vào cơ chế đo offset của CR-002 — chỉ thêm dữ liệu bên cạnh.

## Rủi ro
- **Báo động giả.** Đây là rủi ro làm hỏng cả tính năng: nếu QC kêu ở những video
  vốn ổn, Creator sẽ bỏ qua báo cáo và cổng chặn thành hình thức. Nên khởi đầu
  với ngưỡng nới rộng, chạy ở chế độ chỉ-báo trên vài video thật, siết dần theo
  số liệu. FR61.5 tồn tại vì lý do này.
- **Bounding box của Manim không phải cái mắt nhìn thấy.** Nhóm `VGroup`, phần
  trong suốt, mobject đã `FadeOut` nhưng chưa remove đều có thể gây hiểu nhầm.
  Cần đối chiếu với frame thật khi hiệu chỉnh ngưỡng.
- **Chuyển loudnorm sang hai lượt làm chậm assembly** thêm một lượt đọc toàn
  file. Chỉ làm khi FR60.1 chứng minh sai lệch đủ lớn.

## Quyết định đã chốt (2026-09-10)
1. **Chặn publish**: tràn khung (FR59.1) và chồng lấn narration (FR60.2). Mọi
   phát hiện còn lại là cảnh báo.
2. **Ngưỡng "chữ quá nhỏ"**: tính theo màn hình điện thoại ~5,5 inch, video xem
   toàn màn hình.
3. **Giai đoạn đầu chạy ở chế độ chỉ-báo, không chặn gì.** Bật cổng chặn sau khi
   đã hiệu chỉnh ngưỡng trên các video thật — báo động giả là rủi ro làm hỏng cả
   tính năng, và không có cách nào hiệu chỉnh ngưỡng bằng suy luận.

## Câu hỏi cần Creator chốt
1. Lỗi nào đáng **chặn publish**, lỗi nào chỉ cảnh báo? Đề xuất chặn: tràn khung
   và chồng lấn narration. Còn lại cảnh báo.
2. Ngưỡng "chữ quá nhỏ" tính theo màn hình điện thoại ở kích thước nào?
3. Có muốn giai đoạn đầu chạy QC ở chế độ chỉ-báo (không chặn gì) để hiệu chỉnh
   ngưỡng trước không?

## Kiểm chứng
- Unit test: từng luật chấm điểm, với bộ dữ liệu bố cục dựng sẵn cho cả trường
  hợp đạt và không đạt.
- Unit test: QC lỗi kỹ thuật không chặn publish (FR61.4).
- Thủ công: dựng một script cố tình có chữ tràn khung và một script có narration
  chồng lấn; xác nhận QC bắt được cả hai.
- Đối chiếu: chạy QC trên các video đã sản xuất, xem tỉ lệ báo động giả trước
  khi bật cổng chặn.

## Liên quan
- CR-002 (`_cf_mark` — cơ chế được mở rộng ở FR58)
- CR-004 (chất lượng encode — FR60.5 kiểm lại điều CR đó đặt ra)
- CR-005 (loudnorm và ducking — FR60.1 kiểm lại)
- CR-017 (safe margin và thang cỡ chữ — nguồn ngưỡng cho FR59)
- CR-020 (`quality-service` — nơi bước `qc_video` sống)
