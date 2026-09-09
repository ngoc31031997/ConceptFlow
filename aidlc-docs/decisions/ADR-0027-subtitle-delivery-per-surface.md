# ADR-0027: Cách giao phụ đề là thuộc tính của bề mặt phát hành, không phải một toggle toàn cục

## Status
Proposed

## Date
2026-09-09

## Stage
Low-Level Design (CR-015)

## Context
CR-001 FR9.1 cho Creator một toggle phụ đề bật/tắt. "Bật" được hiện thực hoá
bằng đúng một cách giao: ffmpeg đốt file `.ass` vào khung hình
(`ffmpeg_assembler.py` dòng 225–228).

Lúc đó hệ thống chỉ có một bề mặt phát hành (video dài trên YouTube) nên "bật
phụ đề" và "đốt phụ đề vào hình" là một. Giả định đó không còn đúng:

- YouTube nhận **caption track** — index vào search, auto-translate được, người
  xem tự tắt được. Đây mới là dạng phụ đề mà nền tảng hiểu.
- CR-007 sẽ thêm clip dọc cho Shorts/TikTok, nơi caption track **không** dùng
  được (TikTok không có đường nạp qua API, feed autoplay tắt tiếng).

Cùng một `subtitle_cues`, hai bề mặt, hai cách giao trái ngược nhau.

## Options Considered

### Option A: Giữ toggle hai trạng thái, thêm caption track vào nhánh "bật"
- What it is: bật phụ đề ⇒ vừa đốt vào hình vừa upload track.
- Strengths: không đổi mô hình, không đổi GUI, code ít nhất.
- Trade-offs: người xem bật CC thấy chữ **chồng hai lớp**. Và nó vẫn không giải
  được vấn đề gốc — chữ đốt cứng vẫn đè lên công thức Manim ở rìa khung, vẫn
  không tắt được. Chỉ là thêm một cách giao chứ không sửa cách giao sai.

### Option B: Thay burn-in bằng caption track, bỏ hẳn `.ass`
- What it is: chỉ còn một cách giao, là track.
- Strengths: mô hình đơn giản nhất; xoá được cả `subtitle_file.py` lẫn
  `SubtitleStyle`.
- Trade-offs: **chặn đường CR-007**. Clip dọc bắt buộc phải có chữ trong pixel.
  Xoá xong rồi CR-007 lại phải viết lại từ đầu. Ngoài ra `SubtitleStyle` (cỡ
  chữ, màu, hộp nền, vị trí) là lựa chọn Creator đã có — SRT không mang được
  định dạng, nên bỏ burn-in là lặng lẽ vứt luôn nhóm lựa chọn đó.

### Option C: Cách giao là một chiều riêng, mặc định suy ra từ format
- What it is: `subtitle_mode ∈ {off, track, burn_in, both}`. Long-form mặc định
  `track`, clip dọc mặc định `burn_in`. Creator override được.
- Strengths: cùng một `subtitle_cues` phục vụ cả hai bề mặt mà không bên nào
  phải thoả hiệp; `.ass` + `SubtitleStyle` giữ nguyên giá trị cho nhánh burn-in;
  CR-007 sau này chỉ cần khai mặc định của nó, không phải sửa lại kiến trúc.
- Trade-offs: bốn trạng thái thay vì hai — nhiều nhánh hơn để test, và một lựa
  chọn (`both`) mà Creator có thể chọn nhầm. Giảm thiểu bằng cảnh báo tại chỗ
  (CR-015 FR41.3) chứ không cấm, vì `both` là hợp lệ khi repost sang nền tảng
  không nhận track.

## Decision
**Option C.**

Điều quyết định là Option A và B đều ngầm coi "phụ đề" là một khái niệm duy
nhất. Thực tế nó là hai thứ khác nhau dùng chung một nguồn dữ liệu: một luồng
văn bản có timestamp mà nền tảng đọc được, và một lớp pixel vẽ lên hình. Cái
nào đúng phụ thuộc hoàn toàn vào nơi video sẽ được xem — thứ mà chỉ format phát
hành mới biết.

Đặt cách giao thành chiều riêng khiến `subtitle_cues` trở về đúng vai trò của
nó: dữ liệu nguồn, trung lập với cách hiển thị. `.ass` và `.srt` đều chỉ là
serializer.

Hệ quả trực tiếp: mọi phép biến đổi lên timeline (hiện là `lead_in`) thuộc về
tầng dựng cues, không thuộc serializer. Serializer nhận cues đã đúng và chỉ lo
định dạng. Nếu để chúng tự dịch timestamp thì hai serializer sẽ có hai bản sao
của cùng một phép tính — và bản sai sẽ là bản không ai nhìn thấy.

## Consequences
- **Video dài đổi diện mạo.** Mặc định thành `track` nên chữ không còn đốt vào
  hình; video mới khác video cũ trên cùng kênh. Creator đã chấp nhận có ý thức
  (CR-015, mục "Thay đổi hành vi đã giao").
- **`SubtitleStyle` chỉ còn nghĩa ở nhánh burn-in.** GUI phải làm mờ/ẩn nhóm
  lựa chọn này khi mode là `track`, nếu không Creator sẽ chỉnh màu chữ rồi
  không hiểu vì sao không có tác dụng.
- **CR-007 thừa hưởng sẵn.** Khi mở lại, nó chỉ khai `burn_in` là mặc định của
  preset dọc; không phải đụng vào assembly hay publisher.
- Nợ lại: chưa có cách nào để một project xuất **đồng thời** bản dài và bản dọc
  với hai mode khác nhau. Hiện mode là thuộc tính của project. Khi CR-007 sinh
  clip phái sinh, mode sẽ phải hạ xuống thành thuộc tính của từng output.

## Related
- CR-001 FR9.1/FR9.4 (toggle và `SubtitleStyle` mà ADR này mở rộng)
- CR-015 FR41 (bốn lựa chọn ở GUI)
- CR-007 FR19.4 (phụ đề trên clip dọc — bên thừa hưởng)
- ADR-0028 (scope OAuth mà nhánh `track` đòi hỏi)
