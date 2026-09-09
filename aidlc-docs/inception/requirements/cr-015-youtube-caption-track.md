# CR-015 — Caption track cho YouTube, burn-in theo format (P1)

## Date
2026-09-09

## Stage
Requirements Analysis

## Intent Analysis
- **Request type**: Feature + đổi mặc định hành vi đã giao (không đổi domain model của phụ đề)
- **Scope estimate**: 3 unit — `video-assembly`, `publisher`, `web-gui`; cộng `orchestrator` để chuyển thêm một đường dẫn artifact
- **Complexity estimate**: Moderate. Phần logic đơn giản (cues đã có sẵn, chỉ đổi format xuất); phần khó nằm ở **đổi OAuth scope**, kéo theo re-consent toàn bộ kênh đã nối

## Bối cảnh — quan sát được
Creator mở video đã publish trên YouTube: nút CC mờ, hiện "Subtitles/closed
captions unavailable".

Không phải bug. Hệ thống chưa bao giờ gửi caption track lên YouTube:

- `video-assembly` render cues thành file `.ass` rồi ffmpeg **đốt thẳng vào khung
  hình** (`ffmpeg_assembler.py` dòng 225–228). Chữ trở thành pixel; YouTube không
  có cách nào biết đó là phụ đề.
- `publisher` chỉ gọi `videos().insert` và `thumbnails().set`
  (`youtube_publisher.py` dòng 124). Không có `captions().insert`.

Auto-caption của YouTube không lấp được chỗ này: nó phụ thuộc nhận diện giọng
nói, chạy trễ hàng chục phút, và không đảm bảo cho mọi ngôn ngữ — trong khi hệ
thống **đã có sẵn transcript chính xác từng từ** (chính là `subtitle_cues`, do
Orchestrator sinh với timing đo từ audio thật, CR-002).

## Vấn đề
Burn-in là lựa chọn **sai mặc định cho video dài trên YouTube**, vì ba lý do độc
lập:

1. **Mất SEO.** YouTube index nội dung caption vào search. Pixel không đóng góp
   gì. Với kênh giáo dục, search là kênh traffic chính — đây là thiệt hại lớn
   nhất và cũng là thứ không nhìn thấy được.
2. **Mất người xem quốc tế.** Caption track cho phép YouTube auto-translate.
   Chữ đốt cứng thì vĩnh viễn một ngôn ngữ, kể cả khi CR-008 đã cho phép sản xuất
   nhiều ngôn ngữ.
3. **Đè lên nội dung.** Manim thường đặt công thức và chú thích ở rìa khung —
   chính lý do CR-007 FR19.3 từ chối crop khi chuyển 9:16. Phụ đề đốt cứng ở đáy
   khung che mất phần đó, và người xem **không tắt được**.

Ngược lại, với clip dọc Shorts/TikTok thì burn-in mới là đúng: feed autoplay tắt
tiếng, người xem lướt trong vài giây, và TikTok không có đường nạp caption track
đàng hoàng qua API.

Tức là đây không phải câu hỏi "cái nào tốt hơn" mà là "cái nào hợp với bề mặt
phát hành nào".

## Quyết định
Phụ đề có **mặc định theo format phát hành**, Creator override được:

| Format | Mặc định | Vì sao |
|---|---|---|
| Long-form 16:9 → YouTube | caption track, **không** burn-in | SEO, auto-translate, người xem tự tắt được, khung hình sạch |
| Dọc 9:16 → Shorts/TikTok | burn-in, không caption track | autoplay tắt tiếng; TikTok không nhận caption track |

**Phạm vi CR này chỉ là hàng long-form.** CR-007 đang hoãn, code clip dọc chưa
tồn tại; hàng thứ hai của bảng là ràng buộc ghi sẵn cho CR-007 khi nó được làm,
không phải việc phải code bây giờ.

## Thay đổi hành vi đã giao — có ý thức
Video dài hiện đang burn-in. Sau CR này thì thôi. Video mới sẽ trông khác video
cũ trên cùng một kênh. Creator đã cân nhắc và chấp nhận (2026-09-09): lợi ích
SEO và auto-translate lớn hơn tính nhất quán thị giác của backlog.

## Functional Requirements

### FR38 — Xuất caption track ở Video Assembly
- **FR38.1**: Khi Creator chọn chế độ có caption track, `video-assembly` PHẢI ghi
  thêm một file `.srt` từ chính `subtitle_cues` đang dùng cho `.ass` — cùng một
  nguồn dữ liệu, chỉ khác format serialize. KHÔNG được sinh cues riêng cho SRT.
- **FR38.2**: Timestamp trong `.srt` PHẢI cộng cùng lượng dịch `lead_in` mà
  `_write_subtitles` đang áp cho `.ass` (`ffmpeg_assembler.py` dòng 145). Bỏ sót
  bước này thì caption lệch đúng bằng lead-in trên toàn bộ video, và **không ai
  phát hiện ra** vì bản burn-in vẫn đúng.
- **FR38.3**: `SubtitleStyle` KHÔNG áp dụng cho `.srt` — SRT không mang định
  dạng, và YouTube tự quyết định cách hiển thị. Đây là lý do `.ass` vẫn phải tồn
  tại cho nhánh burn-in chứ không thay được bằng SRT.
- **FR38.4**: Đường dẫn `.srt` PHẢI được phát ra trong event hoàn tất assembly,
  lưu vào project (cột mới, cạnh `video_path`), và chuyển tiếp vào payload Saga
  Publish — cùng đường mà `thumbnail_path` đang đi.

### FR39 — Upload caption track ở Publisher
- **FR39.1**: `PublishRequest` PHẢI nhận thêm `caption_path: str | None`.
- **FR39.2**: Sau khi `videos().insert` thành công, PHẢI gọi `captions().insert`
  với `snippet.videoId`, `snippet.language`, `snippet.name = ""` (track mặc
  định), `media_body` là file `.srt`.
- **FR39.3**: `snippet.language` PHẢI là mã BCP-47 lấy từ `ContentLanguage` của
  project (CR-008), KHÔNG hardcode. Sai mã ngôn ngữ thì YouTube auto-translate
  dịch sai nguồn — hỏng đúng thứ CR này muốn đạt được.
- **FR39.4**: Lỗi upload caption PHẢI được log chứ KHÔNG raise, y như
  `thumbnails().set`: video đã lên rồi, hỏng caption không đáng làm hỏng cả lần
  publish. Nhưng khác thumbnail ở chỗ **PHẢI báo lên GUI** — thumbnail hỏng thì
  Creator nhìn thấy ngay trên YouTube, caption hỏng thì im lặng hoàn toàn.

### FR40 — OAuth scope và re-consent
- **FR40.1**: `oauth_flow.py` PHẢI thêm scope
  `https://www.googleapis.com/auth/youtube.force-ssl` — `captions.insert` bắt
  buộc scope này, `youtube.upload` không đủ.
- **FR40.2**: Credential đã lưu trước CR này KHÔNG có scope mới. Hệ thống PHẢI
  phát hiện được điều đó **trước khi upload** và báo cho Creator biết kênh nào
  cần nối lại — không để phát hiện muộn bằng một lỗi 403 sau khi video đã lên.
- **FR40.3**: Kênh chưa re-consent PHẢI vẫn publish được video bình thường, chỉ
  bỏ qua bước caption kèm cảnh báo. Không được chặn publish.

### FR41 — Lựa chọn của Creator ở GUI
- **FR41.1**: Toggle phụ đề bật/tắt hiện tại (CR-001 FR9.1) PHẢI thành bốn lựa
  chọn: `tắt` | `caption track` | `burn-in` | `cả hai`.
- **FR41.2**: Mặc định cho project long-form PHẢI là `caption track`.
- **FR41.3**: Chọn `cả hai` PHẢI kèm cảnh báo tại chỗ: người xem bật CC sẽ thấy
  chữ chồng hai lớp. Vẫn cho chọn — hợp lệ khi Creator repost sang nền tảng
  không nhận caption track — nhưng phải nói rõ hệ quả.

## Non-goals
- Không làm phụ đề đa ngôn ngữ do ta tự dịch. Auto-translate của YouTube đã đủ,
  và dịch thuật là việc riêng.
- Không đụng `subtitle_file.py` ở nhánh `.ass` — burn-in giữ nguyên hành vi.
- Không làm gì cho clip dọc. Ràng buộc đã ghi ở mục Quyết định, thực thi khi
  CR-007 được mở lại.
- Không tự động re-consent hộ Creator. Chỉ phát hiện và hướng dẫn.

## Rủi ro
- **Google verification.** `youtube.force-ssl` rộng hơn `youtube.upload` đáng kể.
  Nếu app đang ở chế độ Testing thì không sao; nếu đã published thì có thể phải
  xin verify lại. CẦN xác nhận trạng thái app trước khi bắt tay code.
- **Re-consent là phiền phức một lần.** Chấp nhận, nhưng chỉ chấp nhận **một
  lần** — đó là lý do không tách CR này thành hai đợt.

## Kiểm chứng
- Unit test: SRT serialize đúng format và đúng lead-in offset; `captions().insert`
  được gọi với đúng language code; lỗi caption không làm hỏng publish; credential
  thiếu scope bị phát hiện trước khi upload.
- E2E: publish private một video có caption, mở trên YouTube, xác nhận nút CC bật
  được và nội dung khớp narration.

## Liên quan
- CR-001 (toggle phụ đề, SubtitleStyle — thứ FR41 mở rộng)
- CR-002 (timing cues đo từ audio thật — nguồn dữ liệu của caption track)
- CR-008 (ngôn ngữ nội dung — nguồn của FR39.3)
- CR-007 (clip dọc — nơi ràng buộc burn-in ở bảng Quyết định sẽ được thực thi)
- CR-012 / ADR-0026 (mô hình nhiều OAuth client — bối cảnh của FR40)
