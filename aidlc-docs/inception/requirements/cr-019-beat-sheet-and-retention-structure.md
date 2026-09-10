# CR-019 — Beat sheet là hợp đồng: cấu trúc video, hook và CTA (P1)

## Date
2026-09-09

## Stage
Requirements Analysis (Change Request)

## Intent Analysis
- **Request type**: New Feature — biến cấu trúc video từ lời khuyên thành ràng buộc kiểm chứng được
- **Scope estimate**: 3 unit — `rendering` (component + khai báo beat), `orchestrator` (validate ngân sách), `web-gui` (chọn format, hiển thị phân bổ)
- **Complexity estimate**: Moderate, và **phụ thuộc cứng** vào CR-017 + CR-018

## Bối cảnh — quan sát được
CR-006 §FR17 đã đặt ra yêu cầu hook và end screen từ 2026-09-07, với lý do
*"70% người xem rời trong 15 giây đầu"*. Đến nay:

- FR17 chưa được thực thi. §Quyết định #2 của chính CR-006 đã hạ nó xuống thành
  "snippet Creator tự chèn", vì bất biến `NARRATION == wait(AUTO)` khiến không
  thể tự động chèn scene.
- Không có khái niệm cấu trúc video ở bất kỳ đâu trong code. `ParsedScript` chỉ
  có `scenes`, `scene_class_name`, `chapters`. Một "scene" ở đây là **một câu
  narration**, không phải một phân đoạn nội dung.
- Ràng buộc cấu trúc duy nhất tồn tại là một câu trong prompt: *"mở đầu gây chú
  ý → giải thích khái niệm cốt lõi → ví dụ minh họa → so sánh → tổng kết"*.
  Không có gì kiểm tra câu này được tuân thủ.
- `# CHAPTER:` có tồn tại, nhưng là **nhãn mô tả** do Creator đặt tuỳ ý, không
  phải vai trò trong cấu trúc. Một video có thể có 5 chapter mà không có hook.

CR-018 gỡ bỏ nguyên nhân kỹ thuật đã chặn CR-006 FR17. CR này thực thi phần còn nợ.

## Vấn đề
1. **Không có hook, không có CTA, không có recap** — ba yếu tố quyết định
   watch-time và subscriber, đúng hai chỉ số điều kiện bật kiếm tiền mà CR-006
   nêu ở phần Intent.
2. **Thời lượng không phân bổ.** CR-016 cho Creator thấy tổng thời lượng, nhưng
   một video 10 phút với 8 phút giải thích và 4 giây mở đầu vẫn "đạt" 10 phút.
   Cái quyết định retention là **phân bổ**, không phải tổng.
3. **Mỗi video là một cấu trúc mới.** Kênh chuyên nghiệp có công thức lặp lại —
   người xem quen nhịp và ở lại lâu hơn. Ở đây mỗi lần model tự nghĩ lại từ đầu.

## Quyết định
Định nghĩa **format video là dữ liệu**, không phải câu chữ trong prompt:

```yaml
format: deep_dive_10min
target_duration: {min: 480, max: 720}
beats:
  - {id: hook,    role: hook,    duration: {min:  8, max:  15}, required: true}
  - {id: promise, role: promise, duration: {min: 10, max:  25}, required: true}
  - {id: setup,   role: context, duration: {min: 30, max:  90}}
  - {id: core,    role: explain, duration: {min: 90, max: 180}, repeat: 3}
  - {id: payoff,  role: aha,     duration: {min: 20, max:  60}, required: true}
  - {id: recap,   role: summary, duration: {min: 20, max:  45}}
  - {id: cta,     role: cta,     duration: {min: 10, max:  20}, required: true}
```

Script khai báo mình đang ở beat nào bằng `self.beat("hook")`. Ba việc trở nên
kiểm chứng được ngay lúc soạn, trước khi tốn TTS hay render:

- thiếu beat bắt buộc,
- beat vượt hoặc hụt ngân sách,
- tổng thời lượng ngoài `target_duration`.

Hook và CTA đồng thời là **component** của `conceptflow` (khả thi nhờ CR-018),
nên chúng tự mang theo narration của mình và Creator không phải dựng lại mỗi lần.

## Functional Requirements

### FR51 — Format video là dữ liệu có phiên bản
- **FR51.1**: PHẢI có định nghĩa format khai báo được: `target_duration`, danh
  sách beat với `role`, ngân sách thời lượng, cờ bắt buộc, và số lần lặp cho phép.
- **FR51.2**: PHẢI có sẵn tối thiểu hai format: một long-form (8–12 phút, nhắm
  mốc chèn quảng cáo giữa video) và một ngắn (3–5 phút).
- **FR51.3**: Format PHẢI chọn được theo từng project và lưu trên `Project`,
  cạnh `render_quality` và `content_language`.
- **FR51.4**: Thêm format mới PHẢI chỉ là thêm dữ liệu, không sửa logic.
- **FR51.5**: Creator PHẢI **nhân bản một format rồi sửa** ngay trên GUI — đổi
  ngân sách, thêm/bớt beat, đổi cờ bắt buộc — mà không đụng tới code hay phải
  chờ một lần phát hành. Cấu trúc video phụ thuộc rất nhiều vào chủ đề cụ thể,
  nên một bộ beat cố định trong mã nguồn sẽ sai ngay khi gặp chủ đề đầu tiên
  không vừa khuôn. Bộ beat ở mục Quyết định là **format khởi đầu**, không phải
  khuôn bắt buộc.
- **FR51.6**: Format PHẢI có phiên bản, và project đã sản xuất PHẢI giữ được
  format tại thời điểm nó chạy — sửa format không được làm sai lệch dàn ý hay
  chapter của video cũ.

### FR52 — Khai báo beat trong script và validate ngân sách
- **FR52.1**: `ConceptFlowScene` PHẢI cung cấp `beat(id)` đánh dấu điểm bắt đầu
  một beat; mọi narration sau đó thuộc beat ấy cho tới lời gọi `beat` kế tiếp.
- **FR52.2**: Beat PHẢI sinh ra chapter YouTube tương ứng, thay cho việc Creator
  đặt `# CHAPTER:` rời rạc. Hai cơ chế song song sẽ trôi khỏi nhau.
- **FR52.3**: Hệ thống PHẢI kiểm tra script với format đã chọn — beat bắt buộc
  còn thiếu, beat lạ không có trong format, beat sai thứ tự — và báo **trước khi
  TTS chạy**.
- **FR52.4**: Ngân sách thời lượng PHẢI được kiểm bằng ước lượng của CR-016 lúc
  soạn, và bằng số đo thật sau khi render. Sai lệch giữa hai lần PHẢI hiển thị được.
- **FR52.5**: Vi phạm ngân sách thời lượng PHẢI là **cảnh báo**, không chặn:
  ước lượng có sai số vốn có, và chặn render vì một con số ±15% sẽ làm Creator
  mất tin vào cả cơ chế. Ngược lại, **thiếu beat bắt buộc PHẢI chặn** — đó là dữ
  kiện chắc chắn, không phải ước lượng.

### FR53 — Hook, CTA, recap là component
- **FR53.1**: PHẢI có component cho hook mở đầu, CTA cuối và màn recap, mỗi cái
  tự mang narration của mình. Đây là phần CR-006 FR17 còn nợ.
- **FR53.2**: Hook PHẢI nhận nội dung từ Creator (câu hỏi/lời hứa cụ thể của
  video), KHÔNG tự sinh từ tiêu đề. CR-006 FR17.2 đề xuất lấy từ tiêu đề, nhưng
  tiêu đề được soạn ở bước publish — sau khi render — nên tại thời điểm này nó
  chưa tồn tại. Đây là cùng một lý do khiến thumbnail tự động không burn chữ
  (CR-006 §Quyết định #3).
- **FR53.3**: Màn CTA PHẢI chừa vùng an toàn cho end-screen element của YouTube
  (giữ nguyên yêu cầu CR-006 FR17.1: 15–20 giây cuối).
- **FR53.4**: Creator PHẢI bật/tắt được từng component (CR-006 FR17.2), nhưng
  tắt một beat `required` PHẢI hiện cảnh báo nêu rõ hệ quả retention.

### FR54 — Format đưa vào prompt dưới dạng ngân sách từ
- **FR54.1**: Prompt sinh script PHẢI dựng từ format đã chọn, và diễn đạt ngân
  sách **theo số từ cho từng beat**, không theo phút. Model bám ngân sách từ tốt
  hơn hẳn bám ngân sách thời gian — nó đếm được từ, không đếm được giây.
- **FR54.2**: Câu "Video dài 5-10 phút (khoảng 20-40 marker NARRATION)" trong
  `scriptPrompts.ts` PHẢI được thay bằng beat sheet sinh tự động từ FR51.
- **FR54.3**: Số từ mỗi beat PHẢI quy đổi từ ngân sách giây bằng WPM đã hiệu
  chỉnh của CR-016 FR43, không bằng hằng số riêng.

## Non-goals
- Không tự động chèn hook/CTA vào script Creator đang soạn dở. Component có sẵn,
  prompt yêu cầu dùng, validate bắt nếu thiếu — nhưng việc đặt vào script vẫn do
  script quyết định.
- Không tự viết nội dung hook. Hệ thống ép **có** hook, không ép hook **nói gì**.
- Không tối ưu format theo dữ liệu người xem — đó là CR-022.

## Rủi ro
- **Format cứng có thể phản tác dụng với một số chủ đề.** Vì vậy FR52.5 chỉ
  chặn ở dữ kiện chắc chắn. Nếu Creator liên tục phải bỏ qua cảnh báo, đó là
  tín hiệu format sai chứ không phải script sai.
- **Ngân sách bịa ra.** Các con số trong ví dụ trên là điểm khởi đầu hợp lý,
  không phải kết luận từ dữ liệu. CR-022 tồn tại để thay chúng bằng số đo thật.
- **Phụ thuộc cứng CR-017 + CR-018.** Làm CR này trước sẽ lại rơi vào đúng bế
  tắc mà CR-006 §Quyết định #2 đã gặp.

## Quyết định đã chốt (2026-09-10)

**Format khởi đầu: `visual_first_7min`, mục tiêu 6–8 phút.** Không nhắm mốc 8
phút để chèn quảng cáo giữa video: kênh chưa đạt điều kiện bật kiếm tiền (1.000
subscriber + 4.000 giờ xem), nên thứ cần tối ưu lúc này là **tỉ lệ giữ chân**.
Một video 12 phút loãng tệ hơn hẳn một video 7 phút chặt.

| Beat | Vai trò | Thời lượng | Bắt buộc |
|---|---|---|---|
| `hook` | Câu hỏi hoặc nghịch lý, hiện **bằng hình** ngay, không mở bằng chữ | 8–12s | ✓ |
| `concrete` | Một ví dụ cụ thể chạy **trước khi** có bất kỳ định nghĩa nào | 40–70s | ✓ |
| `pattern` | Rút quy luật ra từ chính ví dụ vừa xem | 60–100s | ✓ |
| `variation` | Ví dụ thứ hai, thứ ba — tăng dần độ khó | 50–90s (×2) | |
| `edge` | Chỗ dễ sai, phản ví dụ | 30–60s | |
| `recap` | Tóm tắt **bằng hình**, không phải danh sách gạch đầu dòng | 20–30s | ✓ |
| `cta` | Kêu gọi hành động | 10–15s | ✓ |

1. **`concrete` PHẢI đứng trước `pattern`.** Đây không phải sở thích trình bày mà
   là cách ép style "nhiều ví dụ minh hoạ trực quan thay vì text đơn điệu" vào
   **cấu trúc**, thay vì chỉ dặn trong prompt như hiện nay. Validate bắt được thứ
   tự này; một lời khuyên trong prompt thì không.
2. **Beat bắt buộc: `hook`, `concrete`, `pattern`, `recap`, `cta`.** `variation`
   và `edge` là tuỳ chọn.
3. **Chuyển hẳn `# CHAPTER:` sang beat** (FR52.2). Hai cơ chế song song sẽ trôi
   khỏi nhau.
4. Bộ beat trên là **điểm khởi đầu để sửa**, không phải kết luận. FR51.5 tồn tại
   chính vì lý do đó, và CR-022 sẽ thay các con số này bằng số đo thật khi kênh
   có đủ dữ liệu.

## Điều chỉnh khi triển khai (2026-09-10)

**Script không khai báo beat nào thì được cảnh báo, không bị chặn.**

FR52.5 nói beat bắt buộc thiếu thì chặn. Khi triển khai lộ ra một ca biên CR
chưa nói tới: script **chưa khai báo beat nào cả**. Áp luật nguyên văn thì mọi
script đơn giản và mọi script viết trước khi beat tồn tại đều fail — cách nhanh
nhất để Creator ghét cơ chế này thay vì dùng nó.

Luật thực thi: chưa khai báo beat nào ⇒ cảnh báo, không chặn. Khai báo **một**
beat là đã chọn dùng format ⇒ áp đủ luật. Việc chọn dùng là tất-cả-hoặc-không,
nên lỗ hổng này không dùng được để né một beat bắt buộc phiền phức.

**Kiểm beat chạy ở `orchestrator`, không ở `rendering`.** Lượt dry đã trả về
danh sách beat từ CR-020, và `VideoFormat` sống trong domain Go, nên gửi format
sang Rendering chỉ để nó kiểm hộ là thêm một đường dữ liệu không cần thiết.
Kiểm ngay khi nhận `script_validated` — vẫn là trước TTS, đúng yêu cầu FR52.3.

**Chapter lấy `beat.id` làm tiêu đề.** Làm cho nó đẹp hơn nghĩa là đoán xem
phần đó nói về cái gì — đúng thứ CR-006 đã từ chối để model làm.

## Câu hỏi cần Creator chốt
1. Format nào là format chính của kênh — độ dài mục tiêu và bộ beat cụ thể?
2. Beat nào thực sự bắt buộc? Đề xuất trong CR là hook / promise / payoff / cta;
   Creator có muốn nới hoặc siết không?
3. Chuyển hẳn `# CHAPTER:` sang beat (FR52.2), hay giữ chapter thủ công song song?

## Kiểm chứng
- Unit test: validate bắt đúng beat thiếu, beat lạ, beat sai thứ tự, ngân sách vượt.
- Unit test: prompt sinh ra chứa ngân sách từ khớp với format và WPM đã hiệu chỉnh.
- Thủ công: sản xuất hai video khác chủ đề cùng format, xác nhận nhịp mở đầu và
  kết thúc giống nhau.
- Đo: đối chiếu phân bổ beat ước lượng với phân bổ thật sau render.

## Liên quan
- CR-006 FR15/FR17 (chapters và hook/end-screen — CR này thực thi phần còn nợ)
- CR-016 (ước lượng và WPM hiệu chỉnh — nguồn số cho FR52.4 và FR54.3)
- CR-017 (component dựng hook/CTA/recap)
- CR-018 (điều kiện kỹ thuật để component mang được narration)
- CR-022 (thay ngân sách phỏng đoán bằng dữ liệu retention thật)
