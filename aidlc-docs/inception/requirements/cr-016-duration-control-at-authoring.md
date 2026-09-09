# CR-016 — Kiểm soát thời lượng ngay lúc soạn script (P1)

## Date
2026-09-09

## Stage
Requirements Analysis (Change Request)

## Intent Analysis
- **Request type**: Enhancement — đưa một phép tính đã tồn tại ra đúng chỗ nó có ích
- **Scope estimate**: 2 unit — `web-gui`, `orchestrator` (+ `tts` phát ra số đo)
- **Complexity estimate**: Simple. Không có thành phần mới, không đổi saga, không đổi schema message

## Bối cảnh — quan sát được
Creator không có cách nào biết video sẽ dài bao nhiêu cho đến khi bước 3
(`synthesize_speech`) chạy xong. Cụ thể, đã xác minh trong code:

- Prompt sinh script (`services/web-gui/src/components/scriptPrompts.ts`) ghi
  "Video dài 5-10 phút (khoảng 20-40 marker NARRATION)". Đây là **câu chữ trong
  một chuỗi được copy ra AI ngoài** — không có bất kỳ khâu nào trong pipeline
  đếm lại hay chặn lại.
- `validateScript` (`services/web-gui/src/utils/scriptValidation.ts`) đã đếm
  `narrationCount` và `autoWaitCount`, nhưng chỉ dùng để kiểm tra hai số **bằng
  nhau**. Nó không hề ước lượng thời lượng, dù đã có sẵn toàn bộ text narration.
- Hệ thống **đã có** hàm ước lượng: `EstimateNarrationDuration`
  (`services/orchestrator/internal/domain/narration.go`), dùng WPM 150 (en) /
  140 (vi). Nhưng nó chỉ được gọi ở nhánh **tắt TTS**
  (`handle_step_event.go::skipSynthesizeSpeech`). Ở luồng thường nó nằm im.

## Vấn đề
1. **Không control được dài ngắn.** Model trả về 12 marker hay 40 marker đều
   chạy trơn tru như nhau. Creator chỉ phát hiện video 3 phút (dưới ngưỡng chèn
   quảng cáo giữa video) sau khi đã tốn TTS và một lượt render.
2. **Vòng phản hồi sai chỗ.** Thông tin để ước lượng có sẵn ngay lúc dán script
   vào editor; hệ thống lại đợi đến bước 3 mới biết.
3. **WPM là số phỏng đoán, không bao giờ được kiểm chứng.** 140/150 hardcode từ
   đầu. Mỗi lần render là một lần đo được sự thật (word count thật, duration TTS
   thật) nhưng số đo đó bị vứt đi.

## Quyết định
Ước lượng thời lượng là **thông tin hiển thị lúc soạn**, không phải cổng chặn.
CR này không fail bất cứ thứ gì — việc ép cấu trúc là phạm vi của CR-019. Ở đây
chỉ đưa sự thật ra trước mắt Creator sớm nhất có thể.

Ước lượng dựa trên word count nên sai số vốn có (~±15% so với TTS thật). Cách
xử lý không phải là giấu nó đi mà là **hiệu chỉnh WPM bằng số đo thật** và hiển
thị kèm khoảng tin cậy.

## Functional Requirements

### FR42 — Ước lượng thời lượng realtime trong ScriptEditor
- **FR42.1**: `ScriptEditor` PHẢI hiển thị ước lượng tổng thời lượng video, cập
  nhật theo từng lần gõ, tính từ toàn bộ text trong các marker narration của
  script hiện tại.
- **FR42.2**: Công thức PHẢI khớp `EstimateNarrationDuration` phía Go — cùng
  WPM theo `content_language`, cùng sàn `minNarrationSeconds` = 1.5s. Hai công
  thức lệch nhau thì con số hiển thị lúc soạn sẽ khác con số hệ thống thực sự
  dùng khi tắt TTS, và đó là loại sai lệch không ai truy ra được.
- **FR42.3**: Ước lượng PHẢI cộng thêm thời lượng animation. Đây là phần **hệ
  thống không đọc được từ text** (nằm trong `run_time` của `self.play`, kể cả
  giá trị mặc định), nên PHẢI được nêu rõ là chưa tính, thay vì im lặng báo
  thiếu. Cách xử lý: hiển thị "≈ X phút (chưa tính animation)" và tinh chỉnh
  bằng FR43 sau lần render đầu.
- **FR42.4**: PHẢI hiển thị kèm: số marker narration, tổng số từ, và thời lượng
  ước tính của **từng** narration — để Creator thấy ngay câu nào dài bất thường.
- **FR42.5**: Ước lượng KHÔNG được chặn nút bắt đầu render trong CR này.

### FR43 — Hiệu chỉnh WPM từ số đo thật
- **FR43.1**: Sau mỗi lần `synthesize_speech` thành công, hệ thống PHẢI ghi lại
  cặp (số từ, duration thật) theo từng `voice_id`.
- **FR43.2**: WPM dùng cho ước lượng PHẢI là giá trị hiệu chỉnh theo voice khi
  đã có đủ số đo; khi chưa đủ thì rơi về hằng số theo ngôn ngữ hiện tại. Ngưỡng
  "đủ" là một quyết định cần chốt (xem Câu hỏi cần chốt).
- **FR43.3**: Hiệu chỉnh PHẢI theo **voice**, không phải theo ngôn ngữ. Hai
  giọng Azure cùng tiếng Việt đọc nhanh chậm khác nhau rõ rệt; gộp chung sẽ triệt
  tiêu chính thứ đang muốn đo.
- **FR43.4**: Sai lệch giữa ước lượng và thực tế của lần render vừa xong PHẢI
  hiển thị được cho Creator, để họ biết con số ước lượng đáng tin đến đâu.

## Non-goals
- Không chặn render vì thời lượng lệch ngân sách — CR-019 làm việc đó.
- Không tự sửa script cho đủ dài. Hệ thống báo số, Creator quyết định.
- Không ước lượng thời lượng animation bằng cách phân tích tĩnh `self.play` —
  giá trị `run_time` mặc định phụ thuộc loại animation, đoán sẽ sai nhiều hơn là
  đúng. FR43 giải quyết việc này bằng đo thật.

## Rủi ro
- **Ước lượng sai làm Creator mất tin.** Giảm thiểu bằng FR42.3 (nói rõ chưa
  tính animation) và FR43. Nếu vẫn lệch >25% sau hiệu chỉnh, cần xem lại cách
  tiếp cận thay vì tinh chỉnh hằng số.
- **Trùng lặp công thức ở hai ngôn ngữ lập trình** (Go và TypeScript). Chấp
  nhận, nhưng FR42.2 yêu cầu test khóa hai bên khớp nhau.

## Câu hỏi cần Creator chốt
1. Cần bao nhiêu mẫu đo trước khi tin WPM hiệu chỉnh của một voice — 3 video, 5
   video, hay tính trung bình có trọng số ngay từ mẫu đầu?
2. Số đo WPM lưu ở đâu: DB của `tts`, DB của `orchestrator`, hay một bảng
   `voice_calibration` riêng?

## Quyết định đã chốt (2026-09-10)
1. **Ngưỡng tin WPM hiệu chỉnh: 3 mẫu đo cho mỗi voice.** Dưới ngưỡng đó dùng
   hằng số theo ngôn ngữ. Lý do: 3 video là đủ để lộ sai lệch hệ thống của một
   giọng, trong khi vẫn cho kết quả sớm.
2. **Số đo lưu ở bảng `voice_calibration` riêng, trong database của `tts`.** Đó
   là service duy nhất biết cả text lẫn duration thật, nên đo tại chỗ tránh phải
   chuyển số liệu thô qua message chỉ để ghi lại.

## Kiểm chứng
- Unit test: công thức TS và Go cho cùng kết quả trên cùng bộ input (bảng test
  dùng chung).
- Unit test: hiệu chỉnh WPM tách theo voice, không lẫn giữa các voice.
- Thủ công: soạn một script, đối chiếu con số hiển thị với thời lượng thật sau
  render; sai lệch nằm trong khoảng đã công bố.

## Liên quan
- CR-001 (nhánh tắt TTS — nơi `EstimateNarrationDuration` đang được dùng)
- CR-008 (ngôn ngữ nội dung — nguồn của WPM theo ngôn ngữ)
- CR-019 (beat sheet — nơi con số này trở thành ràng buộc thay vì thông tin)
