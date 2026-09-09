# CR-022 — Đóng vòng lặp bằng dữ liệu retention thật (P3)

## Date
2026-09-09 (hoãn 2026-09-10)

## Trạng thái
**HOÃN — chưa triển khai.**

## Stage
Requirements Analysis (Change Request)

## Intent Analysis
- **Request type**: New Feature — đưa dữ liệu người xem ngược trở lại khâu sản xuất
- **Scope estimate**: 3 unit — `publisher` (đọc Analytics), `orchestrator` (lưu và đối chiếu), `web-gui` (hiển thị)
- **Complexity estimate**: Moderate. Phụ thuộc bên ngoài (YouTube Analytics API, OAuth scope) nhiều hơn phụ thuộc nội tại

## Bối cảnh — quan sát được
Pipeline hiện là một chiều: script vào, video ra, publish xong là hết. Không có
gì chảy ngược lại.

- `publisher` chỉ gọi `videos().insert`, `thumbnails().set` và (sau CR-015)
  `captions().insert`. Không đọc lại bất cứ dữ liệu gì từ YouTube.
- Sau CR-019, hệ thống biết **chính xác từng beat bắt đầu ở giây thứ mấy** —
  timestamp thật, đo từ render (CR-002), không phải ước lượng.
- Ngân sách thời lượng trong CR-019 FR51 là **phỏng đoán hợp lý**, do chính CR
  đó thừa nhận ở mục Rủi ro: *"các con số là điểm khởi đầu, không phải kết luận
  từ dữ liệu"*.

## Vấn đề
Mọi quyết định về cấu trúc video — hook dài bao nhiêu, bao nhiêu beat giải
thích, CTA đặt ở đâu — đang dựa vào cảm tính và thông lệ chung. Trong khi
YouTube **đo sẵn** đường cong giữ chân người xem cho từng video, và hệ thống đã
có đủ dữ liệu để map đường cong đó lên từng beat.

Không có vòng lặp này thì CR-019 mãi mãi dừng ở mức "cấu trúc nhất quán". Có nó
thì thành "cấu trúc nhất quán **và đúng**".

## Quyết định
Kéo `audienceWatchRatio` từ YouTube Analytics API về theo từng video đã publish,
map lên beat sheet bằng timestamp thật, và trả lời đúng một câu hỏi:

> Người xem rời bỏ hoặc tua qua ở **beat nào**?

Đây là CR ưu tiên thấp nhất nhóm — nó chỉ có giá trị khi đã có đủ video publish
theo cùng một format, tức là sau khi CR-019 chạy được một thời gian.

## Functional Requirements

### FR62 — Thu dữ liệu Analytics
- **FR62.1**: Hệ thống PHẢI lấy được đường cong giữ chân người xem
  (`audienceWatchRatio` theo `elapsedVideoTimeRatio`) cho từng video đã publish.
- **FR62.2**: Việc lấy dữ liệu PHẢI có độ trễ hợp lý sau khi publish — số liệu
  của một video mới đăng chưa có ý nghĩa thống kê. Ngưỡng cụ thể cần chốt.
- **FR62.3**: PHẢI xử lý được trường hợp video chưa đủ lượt xem để YouTube trả
  dữ liệu, và nói rõ điều đó thay vì hiển thị đường cong rỗng như thể là sự thật.
- **FR62.4**: Lỗi khi gọi Analytics KHÔNG được ảnh hưởng tới bất kỳ luồng sản
  xuất nào. Đây là tính năng quan sát, chạy hoàn toàn ngoài Saga.

### FR63 — Đối chiếu retention với beat
- **FR63.1**: Đường cong PHẢI được map lên các beat bằng timestamp thật của
  chúng, và quy ra tỉ lệ giữ chân **theo từng beat**.
- **FR63.2**: PHẢI xác định được điểm rời bỏ lớn nhất và cho biết nó rơi vào beat nào.
- **FR63.3**: Khi đã có nhiều video cùng format, PHẢI tổng hợp được theo beat để
  thấy mẫu hình lặp lại — một video kém có thể do nội dung, mười video cùng tụt
  ở một beat là do cấu trúc.
- **FR63.4**: PHẢI so sánh được ngân sách thời lượng trong format với thời lượng
  thật và retention thật của beat đó — đây là dữ liệu để sửa CR-019 FR51.

### FR64 — Hiển thị
- **FR64.1**: PHẢI có màn hình hiển thị đường cong retention chồng lên dải beat
  của video, để nhìn ra ngay chỗ tụt rơi vào đâu.
- **FR64.2**: PHẢI hiển thị được tổng hợp theo format (FR63.3), không chỉ theo
  từng video.
- **FR64.3**: KHÔNG tự động sửa format. Hệ thống trình bày bằng chứng, Creator
  quyết định.

## Non-goals
- Không tự điều chỉnh beat sheet theo dữ liệu. Tự động hoá vòng này khi chưa
  hiểu dữ liệu là cách nhanh nhất để tối ưu vào nhiễu.
- Không dựng lại toàn bộ dashboard analytics của YouTube. Chỉ đúng một câu hỏi
  ở mục Quyết định.
- Không thu thập dữ liệu người xem nào khác ngoài số liệu tổng hợp mà API trả về.

## Rủi ro
- **Cần thêm OAuth scope** (`yt-analytics.readonly`), kéo theo re-consent toàn
  bộ kênh đã nối. Đây đúng là loại phiền phức mà CR-015 §Rủi ro đã trải qua với
  `force-ssl`. **Nếu CR-015 chưa triển khai thì nên gộp hai lần đổi scope làm
  một** — re-consent một lần thay vì hai.
- **Cỡ mẫu nhỏ.** Một kênh mới không đủ lượt xem để đường cong có ý nghĩa. CR
  này chỉ đáng làm sau khi có lưu lượng thật; làm sớm sẽ ra kết luận sai.
- **Tối ưu vào nhiễu.** Giảm thiểu bằng FR63.3 (yêu cầu mẫu hình lặp lại) và
  FR64.3 (không tự động hoá).

## Quyết định đã chốt (2026-09-10)
**CR này hoãn vô thời hạn.** Kênh chưa xuất bản video nào, nên không có dữ liệu
retention để phân tích — mọi kết luận rút ra lúc này sẽ là nhiễu chứ không phải
tín hiệu.

Điều kiện mở lại: khoảng **10 video đã publish theo cùng một format** và đủ lượt
xem để YouTube trả về đường cong `audienceWatchRatio` có ý nghĩa thống kê.

Ghi chú về scope: CR-015 **đã triển khai xong** (đã xác minh trong code —
`youtube_publisher.py` gọi `captions().insert`, `oauth_flow.py` khai
`youtube.force-ssl`, có cột lưu scope để phát hiện kênh nối trước khi đổi).
Nghĩa là không còn cơ hội gộp hai lần đổi scope làm một: CR này khi được mở lại
sẽ cần **lần re-consent thứ hai** cho `yt-analytics.readonly`. Chi phí đó hiện
rất thấp vì mới có ít kênh được nối, và sẽ tăng dần theo thời gian.

## Câu hỏi cần Creator chốt
1. Kênh đã có đủ lượt xem để dữ liệu này có ý nghĩa chưa? Nếu chưa, CR này nên
   để lại backlog cho tới khi đủ.
2. Có gộp việc đổi scope với CR-015 không?
3. Lấy dữ liệu sau publish bao lâu — 7 ngày, 28 ngày?

## Kiểm chứng
- Unit test: map đường cong lên beat đúng theo timestamp; video thiếu dữ liệu
  được xử lý rõ ràng.
- Unit test: lỗi Analytics không lan sang luồng sản xuất.
- Thủ công: đối chiếu số liệu hiển thị với YouTube Studio trên cùng một video.

## Liên quan
- CR-002 (timestamp thật — điều kiện để map chính xác)
- CR-012 / ADR-0026 (mô hình nhiều OAuth client — bối cảnh đổi scope)
- CR-015 / ADR-0028 (lần đổi scope trước; nên gộp nếu chưa triển khai)
- CR-019 (beat sheet — đối tượng được dữ liệu này hiệu chỉnh)
