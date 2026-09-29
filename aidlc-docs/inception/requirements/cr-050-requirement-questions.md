# CR-050 — Câu hỏi làm rõ yêu cầu

Trả lời bằng cách ghi chữ cái (và ghi chú nếu cần) sau mỗi thẻ `[Answer]:`. Bối cảnh và số liệu: `cr-050-llm-call-chunking-resume.md`.

## Câu 1
Phạm vi CR này gồm những bước nào?

A) Chỉ bước Code (khuyến nghị) — bước này chiếm gần như toàn bộ lỗi (70/196 lượt chunk lỗi).
   - ✅ Ưu: tập trung đúng chỗ đau, ít rủi ro.
   - ⚠️ Đánh đổi: Storyboard vẫn là một lượt gọi lớn (1/9 lượt lỗi).

B) Bước Code + Storyboard (chia storyboard theo cảnh).
   - ✅ Ưu: storyboard cũng chạy lại được theo cảnh.
   - ⚠️ Đánh đổi: các cảnh viết riêng dễ mất mạch liền (nhân vật, bố cục, nhịp); cần thêm bước ghép và kiểm, việc tăng gấp đôi.

C) Bước Code, và thêm tự thử lại lỗi thoáng qua (Câu 2) cho mọi lượt gọi đơn (Story, Storyboard, gợi ý metadata), nhưng không chia nhỏ các bước đó.
   - ✅ Ưu: mọi bước đều bền hơn mà không phải đổi cấu trúc.
   - ⚠️ Đánh đổi: phạm vi rộng hơn A một chút; một lượt Story/Storyboard lỗi thật vẫn phải chạy lại cả lượt.

X) Other (please describe after [Answer]: tag below)

[Answer]: B + các lượt gọi đơn — "cả storyboard/các lượt gọi đơn" (làm rõ ở cr-050-clarification-questions.md, Câu C1)

## Câu 2
Khi một đoạn gặp lỗi thoáng qua (`empty`, stream đóng mà không gửi gì, 500, rớt kết nối), hệ thống xử lý thế nào?

A) Tự thử lại ngay đoạn đó, tối đa N lần (ví dụ 2), có chờ giữa các lần; hết lượt mới tính là lỗi (khuyến nghị).
   - ✅ Ưu: đa số lỗi hiện tại là loại này, nên phần lớn run sẽ tự xong. Mỗi lần thử lại chỉ tốn đúng một đoạn.
   - ⚠️ Đánh đổi: một đoạn xấu có thể làm bước kéo dài thêm vài phút. Một lượt `empty` thường không bị tính tiền (0 token), nhưng lượt 500 giữa chừng có thể bị tính.

B) Không tự thử lại; chỉ đánh dấu lỗi và để Creator bấm chạy lại.
   - ✅ Ưu: Creator kiểm soát hoàn toàn chi phí.
   - ⚠️ Đánh đổi: gần như run nào cũng cần bấm tay.

C) 💡 Đề xuất: tự thử lại như A, và lần thử cuối chuyển sang một model dự phòng Creator chọn trước (ví dụ deepseek lỗi → glm).
   - ✅ Ưu: vượt được sự cố kéo dài của một model.
   - ⚠️ Đánh đổi: một video có code do hai model viết; cần thêm ô cấu hình model dự phòng.

X) Other (please describe after [Answer]: tag below)

[Answer]: B — "ko cần , user sẽ tự chạy lại trên giao diện"

## Câu 3
Khi một đoạn lỗi hẳn (đã hết lượt thử lại), các đoạn khác thì sao?

A) Chạy xong mọi đoạn còn lại, rồi kết thúc bước với danh sách đoạn lỗi (khuyến nghị). Lần chạy lại chỉ còn các đoạn đó.
   - ✅ Ưu: một lần chạy làm được nhiều nhất có thể; lần sau rất ngắn.
   - ⚠️ Đánh đổi: nếu lỗi là do prompt/storyboard sai (mọi đoạn sẽ lỗi như nhau), sẽ tốn thêm vài lượt trước khi dừng.

B) Giữ như hiện nay: đoạn đang chạy chạy xong, đoạn chưa bắt đầu thì không chạy.
   - ✅ Ưu: dừng tốn tiền sớm khi có vấn đề chung.
   - ⚠️ Đánh đổi: phải chạy lại nhiều vòng mới đủ các đoạn.

C) Như A, nhưng dừng sớm nếu quá K đoạn lỗi hẳn (ví dụ quá một nửa), vì khi đó nhiều khả năng là vấn đề chung.
   - ✅ Ưu: cân bằng giữa A và B.
   - ⚠️ Đánh đổi: thêm một ngưỡng cần chọn.

X) Other (please describe after [Answer]: tag below)

[Answer]: A, cộng thêm cách chạy riêng đoạn lỗi — "chạy tiếp nhưng phải có cách để chạy riêng đoạn đó băng ai ngoài hoặc retry"

## Câu 4
Kết quả các đoạn đã xong được lưu ở đâu để không mất khi restart/rebuild?

A) Lưu trong DB của `authoring-service`, theo project (khuyến nghị). `llm-service` trả về từng đoạn xong ngay khi có; lần chạy lại gửi kèm các đoạn đã có, và `llm-service` chỉ gọi model cho phần thiếu.
   - ✅ Ưu: `llm-service` vẫn không có trạng thái (đúng ADR/CR-039); dữ liệu gắn với project, xoá theo project; Creator xem được.
   - ⚠️ Đánh đổi: thêm một bảng và mở rộng contract HTTP giữa `authoring-service` ↔ `llm-service`.

B) Cho `llm-service` một kho riêng (file trên volume hoặc SQLite), giữ nguyên cách cache hiện nay nhưng bền.
   - ✅ Ưu: sửa ít chỗ nhất.
   - ⚠️ Đánh đổi: `llm-service` có trạng thái; cache không gắn với project (khó xoá, khó hiển thị); cần volume.

C) Giữ cache trong RAM như hiện nay; chỉ thêm tự thử lại và chạy tiếp (Câu 2/3).
   - ✅ Ưu: ít việc nhất.
   - ⚠️ Đánh đổi: rebuild/restart vẫn làm mất các đoạn đã xong.

X) Other (please describe after [Answer]: tag below)

[Answer]: A — "db"

## Câu 5
Khi chạy lại, những gì làm các đoạn đã xong **không** được dùng lại nữa?

A) Chỉ khi storyboard thay đổi (hoặc Creator chủ động bấm "Sinh lại toàn bộ"). Đổi model hay sửa prompt Engineer vẫn dùng lại các đoạn đã xong (khuyến nghị).
   - ✅ Ưu: đổi model sau khi lỗi (việc đã xảy ra 27–28/09) không phải trả tiền lại cho các đoạn đã có.
   - ⚠️ Đánh đổi: một video có thể có các đoạn do hai model khác nhau viết. Các đoạn cũ không theo prompt mới, trừ khi bấm sinh lại toàn bộ.

B) Khi storyboard, model hoặc prompt Engineer đổi (như khoá cache hiện nay).
   - ✅ Ưu: code luôn đồng nhất một model và một prompt.
   - ⚠️ Đánh đổi: đổi model = sinh lại từ đầu.

X) Other (please describe after [Answer]: tag below)

[Answer]: X — "đổi promt" (làm rõ ở Câu C2)

## Câu 6
Kích thước mỗi đoạn (hiện 3 shot, ≈ 38k token ra, ≈ 4 phút mỗi lượt)?

A) Giữ 3 shot; chỉ dựa vào thử lại, lưu bền và chạy tiếp.
   - ✅ Ưu: không đổi chất lượng hay chi phí prompt.
   - ⚠️ Đánh đổi: mỗi lượt lỗi vẫn đắt và dài.

B) Giảm xuống 2 shot mỗi đoạn (khuyến nghị để thử), vẫn chạy song song.
   - ✅ Ưu: lượt ngắn hơn, ít bị ngắt hơn; mỗi lần chạy lại rẻ hơn.
   - ⚠️ Đánh đổi: nhiều lượt hơn, nên phần prompt chung (storyboard + luật, ≈ 20k token vào) bị gửi nhiều lần hơn; chi phí vào tăng khoảng 1,5 lần.

C) 1 shot mỗi đoạn.
   - ✅ Ưu: lỗi nhỏ nhất, chạy lại chính xác nhất.
   - ⚠️ Đánh đổi: chi phí vào tăng khoảng 3 lần; khó giữ mạch chuyển giữa các shot.

D) 💡 Đề xuất: giữ số shot là cấu hình (`CODE_CHUNK_SHOTS`) như hiện nay, và chọn giá trị mặc định sau khi đo một video thật với 2 và 3.
   - ✅ Ưu: quyết định theo số liệu.
   - ⚠️ Đánh đổi: tốn thêm một lần chạy đo.

X) Other (please describe after [Answer]: tag below)

[Answer]: (hỏi lại) — "chưa hiểu kích thước mỗi đoạn là sao" → Câu C3

## Câu 7
GUI hiển thị và thao tác thế nào ở bước Code?

A) Danh sách đoạn (shot a–b) với trạng thái chờ / đang chạy / xong / lỗi (kèm lý do). Nút "Chạy lại phần còn thiếu" và nút "Sinh lại toàn bộ" (khuyến nghị).
   - ✅ Ưu: Creator thấy rõ đoạn nào lỗi, và chủ động chọn chạy tiếp hay làm lại từ đầu.
   - ⚠️ Đánh đổi: thêm việc ở web-gui và API.

B) Giữ giao diện hiện nay; nút "Chạy lại" tự động chỉ chạy phần còn thiếu, và thông báo lỗi ghi rõ đoạn lỗi.
   - ✅ Ưu: ít việc ở GUI.
   - ⚠️ Đánh đổi: không thấy tiến độ từng đoạn; không có cách ép sinh lại toàn bộ trừ khi sửa storyboard.

X) Other (please describe after [Answer]: tag below)

[Answer]: (hỏi lại) — "giao diện bước code thì như thế nào" → Câu C4

## Câu 8
Các lượt sửa (repair) sau kiểm biên dịch có cần lưu và dùng lại không?

A) Có: lưu code đã sửa của từng đoạn như kết quả mới của đoạn đó, để lần chạy lại bắt đầu từ bản đã sửa (khuyến nghị).
   - ✅ Ưu: không mất công sửa khi bước bị ngắt ở giai đoạn kiểm/sửa.
   - ⚠️ Đánh đổi: cần phân biệt bản gốc và bản đã sửa khi Creator muốn làm lại.

B) Không: chỉ lưu bản gốc model viết; lượt sửa luôn chạy lại.
   - ✅ Ưu: đơn giản hơn.
   - ⚠️ Đánh đổi: tốn thêm lượt sửa (rẻ: TB 14s, ~1,8k token) sau mỗi lần ngắt.

X) Other (please describe after [Answer]: tag below)

[Answer]: X — "không cần. ghi đè lên chỉ cần lưu một kêt quả cuối cùng"
