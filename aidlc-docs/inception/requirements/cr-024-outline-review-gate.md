# CR-024 — Cổng duyệt dàn ý trước khi sản xuất (P1)

## Date
2026-09-09

## Stage
Requirements Analysis (Change Request)

## Intent Analysis
- **Request type**: New Feature — thêm một điểm dừng có chủ đích vào Saga đang chạy tự động từ đầu đến cuối
- **Scope estimate**: 3 unit — `orchestrator` (trạng thái chờ + endpoint duyệt), `quality-service` (dựng dàn ý), `web-gui` (màn duyệt)
- **Complexity estimate**: Moderate–Complex. Kỹ thuật không khó; phần khó là **Saga lần đầu tiên phải dừng lại chờ người**, kéo theo trạng thái, khôi phục và idempotency

## Bối cảnh — quan sát được
Render Saga hiện chạy **một mạch, không có điểm dừng nào**. Đã xác minh trong
`handle_step_event.go`: mỗi bước thành công lập tức dispatch bước kế tiếp. Chín
trạng thái happy-path trong `ProjectStatus` đều là trạng thái *đang làm* hoặc
*đã xong* — không có trạng thái nào nghĩa là *đang chờ người*.

Hệ quả: lần đầu tiên Creator biết video nói gì là lúc **xem video đã dựng xong**.
Trước đó nội dung chỉ tồn tại dưới dạng chuỗi ký tự nằm rải rác trong một file
Python do AI ngoài viết ra, không có cách nào đọc lướt.

Điểm dừng duy nhất đang tồn tại nằm **sau** toàn bộ chi phí: `ready_to_publish`,
tức là Creator duyệt **sản phẩm**, không duyệt **kế hoạch**.

## Vấn đề
1. **Sửa nội dung đắt hơn hàng chục lần mức cần thiết.** Phát hiện dàn ý sai
   trọng tâm ở phút thứ 6 của video đã dựng đồng nghĩa với việc bỏ cả lượt TTS,
   lượt render và lượt ghép.
2. **Nội dung không đọc được trước khi sản xuất.** Narration nằm lẫn trong code
   Manim; muốn biết video sẽ nói gì phải tự đọc script và tự ghép lại trong đầu.
3. **Creator không kiểm soát được thứ quan trọng nhất.** Hệ thống đang kiểm soát
   rất chặt phần kỹ thuật (đồng bộ, encode, loudness) nhưng buông hoàn toàn phần
   quyết định chất lượng video: nó nói gì, theo thứ tự nào, có đúng trọng tâm không.

## Quyết định
Thêm trạng thái **chờ duyệt** vào Render Saga, đặt **ngay sau `validate_script`
và trước `synthesize_speech`** — đúng ranh giới giữa phần rẻ và phần đắt.

Vị trí này không phải lựa chọn tuỳ ý mà là điểm duy nhất thoả mãn cả hai điều kiện:

- **Dữ liệu đã đủ**: lượt dry của CR-018 vừa chạy xong ở bước validate, nên đã
  có danh sách narration theo đúng thứ tự chạy thật, các beat (CR-019), các
  chapter, và thời lượng ước lượng từng đoạn (CR-016).
- **Chi phí chưa phát sinh**: chưa gọi TTS, chưa render, chưa ghép.

Cái Creator nhìn thấy là **dàn ý video**, không phải code: các beat theo thứ tự,
lời thoại từng đoạn, thời lượng ước tính từng beat và tổng, các chapter sẽ sinh
ra, cùng cảnh báo từ validate.

## Functional Requirements

### FR68 — Dàn ý đọc được
- **FR68.1**: Sau `validate_script`, hệ thống PHẢI dựng được **dàn ý** gồm: các
  beat theo thứ tự, toàn văn lời thoại từng đoạn, thời lượng ước tính của từng
  đoạn và từng beat, tổng thời lượng, và danh sách chapter sẽ sinh ra.
- **FR68.2**: Dàn ý PHẢI dựng từ **kết quả chạy thật** (lượt dry), không từ việc
  đọc text script. Đây là điều kiện để dàn ý phản ánh đúng video sẽ ra, kể cả khi
  narration nằm trong vòng lặp hay hàm helper (CR-018 FR48.3).
- **FR68.3**: Dàn ý PHẢI hiển thị kèm mọi cảnh báo của `validate_script`
  (CR-020 FR56.3) và mọi vi phạm ngân sách beat (CR-019 FR52.5), tại đúng vị trí
  phát sinh — Creator duyệt nội dung và duyệt cảnh báo trong cùng một lần nhìn.
- **FR68.4**: Dàn ý PHẢI đọc được như một dàn ý, không phải như một bản dump kỹ
  thuật. Không hiện code, không hiện tên class, không hiện đường dẫn artifact.
- **FR68.5**: Dàn ý PHẢI mô tả **cái gì xuất hiện trên màn hình** ở từng beat,
  không chỉ lời thoại. Với một kênh đặt trọng tâm vào ví dụ minh hoạ trực quan,
  duyệt mà chỉ đọc được lời thoại là duyệt một nửa video — và đúng nửa ít quan
  trọng hơn. Mô tả này lấy từ các đối tượng mà lượt dry ghi nhận (CR-021 FR58),
  không phải từ một lời mô tả do model tự viết.

### FR69 — Điểm dừng trong Saga
- **FR69.1**: `ProjectStatus` PHẢI có trạng thái mới nghĩa là *đang chờ Creator
  duyệt*. Nó KHÔNG được biểu diễn bằng một trạng thái `failed_at_*` nào — đây là
  đường đi bình thường, không phải lỗi.
- **FR69.2**: PHẢI có endpoint để duyệt và tiếp tục Saga từ đúng chỗ đã dừng.
- **FR69.3**: PHẢI có đường **từ chối**: Creator quay lại sửa script, và Saga
  kết thúc sạch sẽ chứ không nằm treo.
- **FR69.4**: Duyệt hai lần PHẢI là thao tác idempotent — lần thứ hai không được
  sinh ra một lượt TTS thứ hai. Bảo vệ ở cấp bước đã có sẵn (`SagaStep` bỏ qua
  event khi trạng thái không còn `in_progress`); cơ chế duyệt PHẢI dùng đúng lớp
  bảo vệ đó thay vì tự dựng lớp mới.
- **FR69.5**: Trạng thái chờ PHẢI hiển thị trên SSE progress là *đang chờ bạn*,
  phân biệt rõ với *đang xử lý*. Một Saga dừng mà giao diện trông như đang chạy
  là cách chắc chắn để Creator ngồi đợi vô ích.
- **FR69.6**: Project nằm ở trạng thái chờ quá lâu PHẢI xử lý được — tối thiểu
  là hiện rõ ở danh sách project để không bị bỏ quên. Có tự huỷ sau một khoảng
  thời gian hay không là quyết định cần chốt.
- **FR69.7**: Cổng duyệt PHẢI bỏ qua được, cho những lần chạy mà Creator đã biết
  rõ mình muốn gì (ví dụ render lại đúng script vừa duyệt ở chất lượng cao hơn).
  Một cổng không bỏ qua được sẽ biến thành thao tác bấm cho xong, và mất hết giá trị.

### FR70 — Sửa nội dung ngay tại màn duyệt
- **FR70.1**: Creator PHẢI sửa được lời thoại ngay trong màn duyệt, không phải
  quay về trình soạn code để tìm đúng dòng.
- **FR70.2**: Nội dung sửa PHẢI được ghi ngược vào script nguồn, đúng chuỗi
  narration tương ứng, không đụng tới bất kỳ phần nào khác của script. Khả thi vì
  sau CR-018 mỗi narration là một chuỗi ký tự tại một vị trí xác định trong mã
  nguồn (`self.narrate("...")`).
- **FR70.3**: Sau khi sửa, thời lượng ước tính PHẢI cập nhật lại ngay (CR-016).
- **FR70.4**: Sửa lời thoại PHẢI kích hoạt validate lại trước khi cho duyệt —
  một câu sửa xong vẫn có thể vi phạm ngân sách beat hoặc chứa ký hiệu TTS đọc sai.
- **FR70.5**: Việc sửa ở màn duyệt CHỈ áp dụng cho lời thoại. Sửa animation vẫn
  phải quay về trình soạn script — đó là code, không phải nội dung.

## Non-goals
- Không để AI tự phê duyệt hay tự viết lại nội dung. Cổng này tồn tại để đưa
  quyết định về cho con người, tự động hoá nó là tự mâu thuẫn.
- Không làm quy trình duyệt nhiều người (reviewer, phê duyệt nhiều cấp). Đây là
  công cụ một Creator.
- Không cho sửa cấu trúc beat ở màn duyệt — thêm/bớt beat là sửa script.
- Không lưu lịch sử phiên bản dàn ý. Nếu cần, đó là CR khác.

## Rủi ro
- **Saga dừng chờ người là năng lực hệ thống chưa từng có.** Kéo theo: khôi phục
  sau khi restart service, Inbox/Outbox trong lúc chờ, và ngữ nghĩa của
  `SagaStep.in_progress` khi "đang chờ" chứ không phải "đang chạy". Cần thiết kế
  cẩn thận ở bước low-level design, không suy diễn từ CR này.
- **Ma sát quy trình.** Thêm một lần bấm vào mọi lần sản xuất. FR69.7 là van xả;
  nếu Creator vẫn thấy phiền thì nên cân nhắc bật cổng theo lựa chọn thay vì mặc định.
- **Ghi ngược vào script là thao tác sửa mã nguồn tự động** (FR70.2). Sửa nhầm
  vị trí sẽ làm hỏng script của Creator. Cần test chặt, và nên giữ bản sao script
  trước khi ghi.
- **FR70 phụ thuộc CR-018.** Với chuẩn cũ (`# NARRATION` comment) thì vẫn map
  được theo dòng, nhưng kém tin cậy hơn nhiều. Nếu CR-018 chưa xong, nên triển
  khai FR68+FR69 trước và để FR70 ở chế độ chỉ-đọc.

## Quyết định đã chốt (2026-09-10)
1. **Cổng duyệt mặc định bật** cho mọi project, có nút bỏ qua (FR69.7).
2. **FR70 (sửa lời thoại ngay tại màn duyệt) làm ngay trong đợt đầu**, không lùi
   xuống chế độ chỉ-đọc. Creator yêu cầu làm triệt để từ đầu, và việc này rẻ hơn
   dự tính ban đầu: sau khi CR-018 bỏ chuẩn cũ, mỗi narration là một chuỗi ký tự
   tại một vị trí xác định trong mã nguồn, nên ghi ngược là thao tác sửa một
   literal chứ không phải suy luận vị trí.
3. **Project treo ở trạng thái chờ KHÔNG tự huỷ**, chỉ được đánh dấu nổi bật
   trong danh sách project. Tự huỷ công việc của Creator vì họ bận vài ngày là
   hành vi sai.
4. **Dàn ý phải chi tiết**, gồm cả mô tả hình ảnh từng beat (FR68.5).

## Điều chỉnh khi triển khai (2026-09-10)

**FR70.2 (ghi ngược lời thoại vào script) từ chối khi câu đó không truy được về
đúng một chỗ trong mã nguồn.**

CR giả định mỗi dòng dàn ý ứng với một literal trong script. Không phải lúc nào
cũng vậy, và chính CR-018 là lý do: lời thoại giờ được phép nằm trong vòng lặp
và trong hàm, nên **một literal có thể sinh ra ba dòng dàn ý**, còn lời thoại
dựng bằng f-string thì không xuất hiện nguyên văn ở đâu trong mã nguồn cả.

Cách xử lý: chỉ ghi ngược khi câu cũ xuất hiện **đúng một lần** trong script.
Không thì trả 422 kèm lý do cụ thể, và GUI hiện nguyên văn lý do đó. Đoán xem
Creator muốn sửa lần xuất hiện nào rồi ghi nhầm chỗ là làm hỏng script của họ —
không sửa được tại chỗ chỉ là bất tiện nhỏ.

Chuỗi thay thế cũng bị từ chối nếu chứa dấu nháy, xuống dòng hay dấu chéo ngược:
nó được ghép thẳng vào một string literal Python, nên những ký tự đó sẽ đóng
literal sớm và làm script không còn parse được.

**Điểm dừng dùng sentinel `errAwaitingReview`.** `handleSuccess` phát một
progress "completed" sau mỗi bước thành công, và nó ghi đè lên "awaiting_review"
vừa phát. Dùng lại đúng cơ chế sentinel mà `errAggregationFailed` đã có sẵn
trong cùng file, thay vì thêm một nhánh điều kiện thứ hai.

**FR68.5 lấy dữ liệu từ lượt dry, không chờ CR-021.** CR ghi rằng mô tả hình ảnh
lấy từ dữ liệu bố cục của CR-021 FR58. Thực tế lượt dry đã chạy scene rồi, nên
đếm tên class của các mobject đang hiển thị là đủ cho màn duyệt (`Text×2, Arrow`)
và không phải chờ CR-021. Đếm tên class chứ không mô tả nội dung: mô tả nội dung
nghĩa là đoán hình đang nói gì, và đoán sai còn tệ hơn không nói.

## Câu hỏi cần Creator chốt
1. Cổng duyệt nên **mặc định bật** cho mọi project, hay là một lựa chọn khi tạo
   project?
2. FR70 (sửa lời thoại tại chỗ) làm ngay cùng đợt, hay v1 chỉ cần chỉ-đọc và
   "quay lại sửa script"?
3. Project treo ở trạng thái chờ có nên tự huỷ sau một khoảng thời gian không?
   Nếu có thì bao lâu?
4. Ngoài lời thoại, dàn ý còn cần hiện gì nữa để bạn đủ tự tin bấm duyệt?

## Kiểm chứng
- Unit test: duyệt hai lần chỉ sinh một lượt TTS.
- Unit test: từ chối làm Saga kết thúc sạch, không để lại step treo.
- Unit test: sửa lời thoại ghi đúng chuỗi trong script, các phần khác byte-identical.
- E2E: script sai trọng tâm bị chặn ở màn duyệt, xác nhận `tts.commands` không
  nhận lệnh nào.
- Thủ công: đọc dàn ý của một video chưa từng xem và trả lời được "video này nói
  gì, theo thứ tự nào" — đó là tiêu chí nghiệm thu thật của FR68.4.

## Liên quan
- CR-016 (ước lượng thời lượng hiển thị trong dàn ý)
- CR-018 (lượt dry sinh dữ liệu dàn ý; narration là chuỗi định vị được để sửa)
- CR-019 (beat là khung xương của dàn ý)
- CR-020 (cổng duyệt đặt ngay sau `validate_script`, dùng chung dữ liệu)
- CR-021 (cổng duyệt **sản phẩm** ở cuối — cặp đôi với cổng duyệt **kế hoạch** ở đây)
- ADR sẽ cần: "Render Saga có điểm dừng chờ người duyệt"
