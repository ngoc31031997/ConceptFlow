# CR-023 — Intro/outro cố định làm bản sắc kênh (P1)

## Date
2026-09-09

## Stage
Requirements Analysis (Change Request)

## Intent Analysis
- **Request type**: New Feature — thêm phần khung cố định bao quanh mọi video của kênh
- **Scope estimate**: 3 unit — `rendering` (dựng và cache asset), `video-assembly` (ghép + dịch timeline), `orchestrator` (chapters, bật/tắt); `web-gui` (xem trước)
- **Complexity estimate**: Moderate. Việc ghép thì đơn giản; phần khó là **mọi mốc thời gian trong hệ thống đều dịch đi**, đúng lớp lỗi mà CR-002 tồn tại để diệt

## Phân biệt với CR-019 — đây KHÔNG phải hook/CTA
Hai thứ khác nhau và cần cả hai:

| | CR-019 (hook / CTA) | CR-023 (intro / outro) |
|---|---|---|
| Nội dung | Đổi theo từng video | **Y hệt nhau ở mọi video** |
| Mục đích | Giữ chân trong 15 giây đầu, kêu gọi hành động | Nhận diện kênh |
| Có lời thoại | Có, do Creator viết | Không, hoặc chỉ nhạc hiệu cố định |
| Dựng lại mỗi video | Có | **Không — dựng một lần, dùng lại** |

Thứ tự trong video hoàn chỉnh:
```
[intro sting] [hook] [promise] ... [payoff] [recap] [CTA] [outro sting]
 <- CR-023 ->  <----------------- CR-019 -----------------> <- CR-023 ->
```

## Bối cảnh — quan sát được
Kênh **đã có bộ nhận diện**, chỉ là nó chưa bao giờ đi vào video:

- `docs/brand/` chứa `conceptflow-mark-1024.png`, `conceptflow-avatar-800.png`,
  `banner-a-logo-2560x1440.png`, `banner-b-wordmark-2560x1440.png`.
- `docs/brand/make-banner.py` khai báo bảng màu thật đang dùng cho banner:
  nền `rgb(8,14,28)`, chữ `rgb(242,247,255)`, nhấn `rgb(143,199,239)`.
- Không có tham chiếu nào tới `docs/brand/` từ bất kỳ service nào. Bản sắc dừng
  lại ở trang kênh; video thì mở ra bằng bất cứ thứ gì model nghĩ ra hôm đó.

Về mặt kỹ thuật, hệ thống **đã có sẵn khuôn mẫu để dịch timeline an toàn**:
`ASSEMBLY_LEAD_IN_SECONDS` trong `ffmpeg_assembler.py` dịch video, narration và
phụ đề **cùng một lượng**, với comment nói rõ vì sao: dịch riêng audio là cách
tái tạo đúng lỗi desync mà CR-002 đã xoá. Intro về bản chất là một lead-in có
nội dung, nên phải đi qua đúng cơ chế đó.

## Vấn đề
1. **Video không có dấu hiệu nhận diện nào.** Người xem lướt qua không biết đây
   là kênh nào; các video của cùng kênh không liên kết với nhau về thị giác.
2. **Bộ nhận diện đã trả tiền nhưng không dùng.** Logo, wordmark, bảng màu đều
   có sẵn trong repo.
3. **Nếu làm sai cách, đây là chỗ dễ phát sinh chi phí vô ích**: dựng lại intro
   trong từng lượt render Manim là trả tiền nhiều lần cho một đoạn phim không
   bao giờ đổi, và không đảm bảo hai video có intro giống hệt nhau.

## Ràng buộc từ chính logo hiện có
Logo (`conceptflow-mark-1024.png`) là **ảnh 3D photorealistic** — torus thủy
ngân có phản chiếu kim loại và glow, 1024×1024 raster, **không có bản vector
SVG** trong `docs/brand/`.

Điều này quyết định thiết kế của CR: Manim animate vector và hình học, không
animate chất liệu 3D. Với ảnh này Manim chỉ làm được fade, phóng to thu nhỏ,
xoay 2D, dịch chuyển, cộng các phần tử vector vẽ thêm xung quanh. Nó **không**
làm được xoay 3D thật với phản chiếu đổi theo góc, ánh sáng động trên bề mặt kim
loại, hay glow thể tích. Ép Manim dựng sting cho logo này sẽ cho ra thứ trông rẻ
hơn chính cái logo — nên đường upload ở FR65.1 không phải tiện ích thêm thắt mà
là cách duy nhất để intro đạt chất lượng ngang bộ nhận diện.

## Quyết định
Intro và outro là **asset dựng sẵn, ghép ở khâu assembly** — không phải scene
nằm trong script của Creator.

Ba lý do:
- **Đồng bộ tuyệt đối**: cùng một file thì mọi video giống nhau đến từng frame.
  Nếu để mỗi script tự dựng, chúng sẽ trôi khỏi nhau.
- **Chi phí**: dựng một lần, dùng cho mọi video.
- **Không đụng vào bất biến của script**: intro không mang narration nên không
  chạm tới cơ chế đếm/đo offset. Nghĩa là **CR này không phụ thuộc CR-018** và
  có thể làm ngay.

Asset được cache theo từng mức `RENDER_QUALITY` (`720p30` / `1080p60` / `4k60`)
vì ghép hai đoạn phim khác thông số là không ghép được. Intro đến từ file Creator
upload (có bản Manim mặc định để chạy ngay); outro dựng bằng Manim với theme của
CR-017.

## Functional Requirements

### FR65 — Nguồn của asset intro/outro
- **FR65.1**: Intro PHẢI nhận được **file video do Creator upload**, và hệ thống
  PHẢI chuẩn hoá nó về đúng thông số (độ phân giải, framerate, codec) tương ứng
  với từng `RENDER_QUALITY` trước khi ghép.
- **FR65.2**: PHẢI có **một intro dựng bằng Manim làm mặc định**, để pipeline
  chạy được ngay khi chưa có file upload nào.
- **FR65.3**: Outro PHẢI được dựng bằng Manim (không cần đường upload): nội dung
  của nó chủ yếu là chữ, logo tĩnh và bố cục — thứ Manim làm tốt.
- **FR65.4**: Nhạc intro/outro PHẢI là **file riêng, upload và thay được độc
  lập** với phần hình. Ghép ở khâu assembly, không nhúng cứng vào asset hình.
- **FR65.5**: Asset PHẢI được chuẩn hoá **một lần** và cache theo từng
  `RENDER_QUALITY`, không xử lý lại trong mỗi lượt render project.
- **FR65.6**: Cache PHẢI tự dựng lại khi file nguồn, định nghĩa scene hoặc theme
  đổi, và KHÔNG được dựng lại vì bất kỳ lý do nào khác.
- **FR65.7**: Intro PHẢI ngắn. Mốc tham chiếu là 3 giây: đây là phần người xem
  phải trả giá ở đúng khoảng thời gian mà CR-006 xác định là 70% người xem rời
  bỏ. Intro dài là cách đánh đổi retention lấy nhận diện, và tỉ lệ đó rất xấu.
- **FR65.8**: Outro PHẢI dài 15–20 giây và chừa vùng an toàn cho end-screen
  element của YouTube (giữ nguyên ràng buộc CR-006 FR17.1). Creator đã chọn hiện
  **cả ba**: logo, ô video đề xuất, và lời mời đăng ký — nên bố cục PHẢI chừa
  chỗ cho ba loại element đó cùng lúc, không để chúng đè lên logo hay chữ.
- **FR65.9**: Asset PHẢI có phiên bản. Đổi intro không được làm mất khả năng
  truy vết video cũ đã dùng phiên bản nào.

### FR66 — Ghép và dịch timeline
- **FR66.1**: `video-assembly` PHẢI ghép intro trước và outro sau video chính.
- **FR66.2**: Toàn bộ mốc thời gian PHẢI dịch đi đúng bằng thời lượng intro —
  offset narration, cue phụ đề (cả `.ass` và `.srt`), và timestamp chapter. PHẢI
  dùng lại đúng cơ chế dịch một-lần-duy-nhất mà `lead_in` đang dùng, không thêm
  đường dịch thứ hai. Đây là yêu cầu quan trọng nhất của CR: bỏ sót một trong
  các mốc trên thì lệch nằm im cho tới khi có người xem phát hiện.
- **FR66.3**: `BuildChapterTimestamps` PHẢI được sửa cho đúng. Hiện nó ép
  `times[0] = 0` (`chapters.go`) với lập luận "chapter đầu phải mở màn video".
  Khi có intro, chapter đầu tiên thật sự bắt đầu **sau** intro, nên ép về 0 sẽ
  khiến chapter đầu nói dối. Cách xử lý: chapter `00:00` là chính intro, các
  chapter còn lại giữ timestamp thật đã dịch.
- **FR66.4**: Kiểm tra `MinChapterSeconds` PHẢI tính trên tổng thời lượng sau
  khi ghép, không phải thời lượng video chính.
- **FR66.5**: Mức âm lượng của nhạc hiệu intro/outro PHẢI khớp với phần thân đã
  chuẩn hoá về -14 LUFS (CR-005). Ghép một đoạn to hơn hoặc nhỏ hơn hẳn là lỗi
  nghe thấy ngay ở giây đầu tiên. Vì nhạc là file Creator tự upload (FR65.4), hệ
  thống PHẢI tự chuẩn hoá nó chứ không giả định file đã đúng mức.
- **FR66.6**: Nhạc nền của phần thân KHÔNG được chạy đè lên intro/outro.
- **FR66.7**: File upload không hợp lệ (sai tỉ lệ khung, quá dài, không đọc được)
  PHẢI bị từ chối **lúc upload** kèm lý do cụ thể, không phải lúc ghép.

### FR67 — Lựa chọn của Creator
- **FR67.1**: Creator PHẢI bật/tắt được intro và outro **độc lập** với nhau,
  theo từng project.
- **FR67.2**: Mặc định PHẢI là bật cả hai cho project long-form.
- **FR67.3**: Chế độ preview (CR-020 FR57) PHẢI bỏ qua intro/outro — chúng
  không đổi, nên xem lại mỗi lần là lãng phí thời gian của Creator.
- **FR67.4**: PHẢI xem trước được intro/outro ở GUI mà không cần render một
  project nào.

## Non-goals
- Không làm hook/CTA — đó là CR-019.
- Không cho phép mỗi project một intro riêng. Điều đó mâu thuẫn trực tiếp với
  mục đích của CR: intro là **của kênh**, không phải của video. Bật/tắt theo
  project thì được (FR67.1), chọn intro khác theo project thì không — file
  upload ở FR65.1 thay intro cho **toàn kênh**.
- Không dựng lại intro cho video đã publish khi bản sắc đổi.
- Không làm công cụ dựng hay sửa sting trong hệ thống. Hệ thống nhận file đã
  dựng xong (FR65.1) và cung cấp một bản Manim mặc định (FR65.2); việc dựng
  sting chất lượng cao diễn ra ở công cụ chuyên dụng bên ngoài.

## Rủi ro
- **Mất đường stream-copy.** `ffmpeg_assembler` hiện dùng `-c:v copy` khi không
  burn phụ đề và không pad — đo được 0.1s so với 24.8s khi phải encode lại. Ghép
  intro nhiều khả năng buộc encode lại toàn bộ. Cần xác nhận liệu concat demuxer
  với thông số khớp hoàn toàn có giữ được stream-copy không; nếu không thì chấp
  nhận chi phí, nhưng phải biết trước chứ không phát hiện sau.
- **Dịch timeline là lớp lỗi nguy hiểm nhất của hệ thống này.** CR-002 đã tốn
  một CR để sửa đúng loại lỗi đó (đo được 61.6s lệch). FR66.2 phải có test khoá
  chặt cả bốn loại mốc.
- **Intro làm tụt retention.** Có thật và đo được sau CR-022. Nếu số liệu cho
  thấy tụt, câu trả lời là rút ngắn intro, không phải bỏ nhận diện.

## Quyết định đã chốt (2026-09-10)
1. **Intro làm theo cả hai đường**: một bản dựng bằng Manim làm mặc định để
   pipeline chạy được ngay, cộng một khe nhận file video upload để thay bằng
   sting dựng ngoài sau. Không phải chờ, và không phải làm lại thiết kế khi thay.
2. **Cách dựng sting thật do Creator chọn ngoài pipeline.** Hai đường phù hợp
   nhất với một logo vốn là ảnh AI 3D: image-to-video AI (Runway/Kling/Sora) cho
   chính ảnh torus đó chuyển động, hoặc dựng 3D bằng Blender/After Effects.
   Vector hoá logo thành SVG để Manim vẽ nét **không** được khuyến nghị — nó xoá
   mất chất liệu thủy ngân, tức là tạo ra một logo khác.
3. **Intro dài 3 giây**, ưu tiên retention.
4. **Nhạc: chưa chọn, upload sau.** Hệ thống phải sẵn khe nhận file (FR65.4) và
   tự chuẩn hoá mức âm lượng (FR66.5). Nguồn nên ưu tiên theo mức an toàn bản
   quyền: YouTube Audio Library (Content ID không đánh vì YouTube sở hữu) →
   Pixabay Music (CC0) → Uppbeat (phải ghi nguồn). Từ khoá tìm cho sting 3 giây:
   `logo sting`, `ident`, `logo reveal`.
5. **Outro hiện cả ba**: logo, ô video đề xuất, lời mời đăng ký (FR65.8).
6. **Màu**: intro/outro giữ bảng màu brand. Kèm điều chỉnh đã ghi ở CR-017 —
   nền `#080E1C` dùng chung cho cả thân video, để chỗ nối không giật màu, còn
   màu nhấn của thân vẫn là Catppuccin Mocha.

## Câu hỏi cần Creator chốt
Đã chốt toàn bộ — xem mục Quyết định đã chốt. Còn một việc cần Creator làm chứ
không phải trả lời: cung cấp file sting intro khi đã dựng xong (pipeline chạy
được bằng bản Manim mặc định trong lúc chờ).

## Kiểm chứng
- Unit test: dịch timeline đúng cho cả bốn loại mốc (offset narration, cue
  `.ass`, cue `.srt`, timestamp chapter) khi bật intro.
- Unit test: chapter đầu là intro tại `00:00`, các chapter sau giữ giá trị thật
  đã dịch; `MinChapterSeconds` tính trên tổng thời lượng.
- Unit test: asset không bị dựng lại khi định nghĩa không đổi.
- Thủ công: xuất hai video khác chủ đề, xác nhận intro/outro giống nhau đến
  từng frame; nghe kiểm mức âm lượng ở điểm nối.

## Liên quan
- CR-002 (đồng bộ timeline — lớp lỗi mà FR66 phải tránh tái phát)
- CR-005 (chuẩn hoá -14 LUFS — nguồn ràng buộc FR66.5)
- CR-006 FR15/FR17 (chapters và end-screen)
- CR-017 (theme và component để dựng asset)
- CR-019 (hook/CTA — phần bù, không trùng)
- CR-020 (preview bỏ qua intro/outro)
- CR-022 (đo tác động của intro lên retention)
