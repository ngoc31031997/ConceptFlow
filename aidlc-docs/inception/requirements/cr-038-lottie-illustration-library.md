# CR-038 — Thư viện minh hoạ Lottie cho engine Remotion (P2)

## Date
2026-09-25

## Stage
Requirements Analysis (Change Request) — chờ Creator duyệt thiết kế trước khi chốt

## Intent Analysis
- **Request type**: Tính năng mới — làm giàu khả năng minh hoạ của engine Remotion
- **Scope estimate**: 3 unit — `rendering` (primitive `LottieClip`, catalog, công cụ), `orchestrator` (nhúng catalog vào prompt Visual Director + Remotion Engineer), `web-gui` (không đổi ở CR này)
- **Complexity estimate**: Moderate. Không đổi saga, không đổi hợp đồng giữa các service.

## Bối cảnh
Engine Remotion hiện chỉ dựng được từ JSX/SVG/CSS do LLM viết. Kịch bản cần
nhân vật có hồn (mèo phản ứng, con trỏ nhăn mặt) hoặc vật thể chi tiết thì LLM
không vẽ nổi, và LLM cũng **không nên** sinh Lottie (file JSON animation quá
phức tạp, dễ hỏng, không nhất quán). Hướng đi: tái dùng animation Lottie từ các
kho miễn phí, được Creator tuyển chọn thủ công, rồi để LLM chỉ **chọn theo tên**.

## Ngoài phạm vi
- Không cần tương tác (state machine, input) — chỉ phát animation. Vì vậy Lottie, không phải Rive.
- Không dùng cho engine Manim.
- Không tự động tải hay cào asset từ kho nào; Creator chọn và bỏ file vào.
- LLM không sinh, không sửa nội dung file Lottie.

## Quyết định

### 1. Asset và manifest
- File Lottie đặt ở `rendering/remotion_project/public/lottie/<id>.json`.
- Danh mục `rendering/remotion_project/lottie/manifest.json`, mỗi asset có: `id` (quy ước `nhóm.tên`, ví dụ `cat.thinking`), `title`, `description` (LLM đọc để chọn), `tags`, `source_url`, `author`, `license`, `attribution`, `status` (`candidate` | `approved`), `loop` (mặc định), `palette` (các màu gốc có thể đổi màu).
- **Chỉ asset `approved` mới vào catalog của prompt và mới render được.** Asset mới luôn vào `candidate` để Creator xem trước rồi mới duyệt.
- **Cổng giấy phép (đã đối chiếu với LottieFiles Help, 2026-09-25):** Lottie Simple License cho dùng thương mại, không bắt buộc ghi công; nhưng **mỗi animation có giấy phép riêng và tác giả có thể thêm hạn chế**. Vì vậy clip `approved` bắt buộc có `license_checked` (ngày Creator đọc giấy phép của chính clip đó). Nguồn ngoài LottieFiles (IconScout, LottieIcon…) chưa được kiểm chứng ở đây — Creator đọc điều khoản trước khi dùng.
   `license` phải thuộc danh sách cho phép (mặc định: CC0, CC-BY, Lottie Simple License). Giấy phép cần ghi công (CC-BY) bắt buộc có `attribution`; công cụ liệt kê danh sách ghi công để dán vào mô tả video.

### 2. Primitive `LottieClip` (rendering/remotion_project/src/conceptflow-mini/lottie.tsx)
- Props: `id`, `x`/`y`/`size` (hoặc `width`/`height`), `loop`, `playbackRate`, `startFrame` (trễ bắt đầu), `colors` (bản đồ màu gốc → khoá PALETTE).
- Phát theo frame của Remotion (`@remotion/lottie`, chính thức, xác định, render song song an toàn). Tải file qua `staticFile`, giữ khung hình đầu bằng `delayRender` cho tới khi tải xong.
- **Đổi màu** là bước tuỳ biến tất định (duyệt cây JSON, thay màu fill/stroke), không dùng LLM. Nhờ đó clip tuân thủ luật "mọi màu từ PALETTE" của Remotion Engineer.
- Id không có trong catalog: ném lỗi rõ ràng, không im lặng bỏ qua.

### 3. Đưa catalog vào prompt
- Theo đúng tiền lệ `theme_reference`: công cụ sinh file text từ manifest → `orchestrator/internal/domain/prompts/lottie_catalog_vi.txt` → `go:embed` → thay `{{lottie_catalog}}` lúc seed.
- Chỉ **Remotion Engineer** nhận catalog (mục C2 trong prompt), kèm cú pháp `LottieClip`. **Visual Director cố ý không đổi**: nó là một đạo diễn cho mọi engine, Manim không dùng được clip. Engineer thay bằng clip chỉ khi HÌNH của shot mô tả đúng chủ thể của clip (ví dụ shot ghi "chú mèo nghiêng đầu" và có `cat.thinking`); không thêm clip trang trí.
- Catalog chỉ có `id`, mô tả, tag, màu có thể đổi. Khi catalog rỗng, khối prompt tự ghi "chưa có clip nào" để LLM quay về SVG như hiện tại.
- Lottie là **tuỳ chọn**: LLM dùng khi clip phù hợp, vẫn được vẽ bằng SVG. Không ép dùng để tránh bóp sáng tạo.

### 4. Kiểm tra sớm
- Lint nhẹ cho script Remotion: mọi `<LottieClip id="…">` phải nằm trong catalog đã duyệt. Sai id thì báo lỗi ở bước validate (rẻ) thay vì chết lúc render.

### 5. Công cụ cho Creator (rendering/tools/lottie_catalog.py)
- `validate`: kiểm tra manifest (trường bắt buộc, giấy phép, file tồn tại, JSON hợp lệ Lottie).
- `gallery`: sinh trang HTML xem trước tất cả clip (dùng lottie-web), lọc theo `status`, hiển thị nguồn và giấy phép — **đây là chỗ Creator duyệt asset**.
- `prompt`: sinh `lottie_catalog_vi.txt`.
- `credits`: in danh sách ghi công.

### 6. Avatar mèo mướp (Mướp) — bộ biểu cảm và hành động
- **Nguồn:** clip "Bad Cat" do Creator cung cấp, lưu ở `remotion_project/lottie/source/bad-cat.lottie`. `tools/build_avatar.py` dựng các biến thể từ đó, tất định và chạy lại được (không dùng LLM).
- **Rig chung:** thêm viền cho đuôi, chân, ly (bản gốc chỉ thân có viền nên đuôi chìm trên nền tối); ba sọc mướp trên trán ở lớp riêng nằm trên mắt; đuôi nằm dưới thân và có miếng che ở gốc đuôi để thân và đuôi liền một khối; một lớp null làm cha để nảy, rung, thở cả nhân vật.
- **Mười clip `cat.*`:** `idle` (mặc định), `smug`, `surprised`, `angry`, `happy`, `sleep`, `thinking`, `sad`, `look` (đảo mắt), `push` (đẩy ly, gag đặc trưng). Biểu cảm chỉ chỉnh mí mắt, con ngươi, vòng trắng của mắt; đạo cụ (chấm than, dấu hỏi, chữ z, giọt mồ hôi, trái tim, dấu giận) là các lớp hình thêm vào.
- **Viền theo nền:** viền mảnh đổi màu bằng `colors` (thay #FFFFFF); viền dày kiểu miếng dán bằng `outline` của `LottieClip` (chống chìm trên nền bất kỳ).
- **Trạng thái:** tất cả `candidate`. Tác giả và giấy phép riêng của clip gốc chưa xác minh (file chỉ ghi "LottieFiles", tên công cụ tạo file), và bộ này là bản CHỈNH SỬA của clip đó. Không clip nào vào prompt hay được dùng trong video cho tới khi Creator xác nhận giấy phép cho phép chỉnh sửa và phân phối lại, điền `license_checked`, đổi `status` sang `approved`.

## Đánh đổi đã cân nhắc
- **Nhúng tĩnh JSON vào bundle** (import): đơn giản nhưng mọi render đều mang toàn bộ asset. Chọn `staticFile` để chỉ tải cái đang dùng.
- **Rive**: mạnh hơn nhưng có trạng thái, khó render xác định song song; không cần tương tác nên bỏ.
- **Để LLM sinh Lottie**: đã loại (xem Bối cảnh).

## Rủi ro
- Nhất quán phong cách giữa các clip từ nhiều tác giả → gallery cho phép Creator loại clip lệch phong cách; ưu tiên tuyển theo pack.
- Giấy phép → cổng giấy phép + `credits`.
- Dung lượng Docker image tăng theo số asset → giới hạn kích thước mỗi file khi validate.

## Backlog: ghép avatar vào video (làm sau, ngoài phạm vi CR này)
Ghép Mướp vào video thật chưa làm. Cần quyết định và làm:
1. **Vai trò:** linh vật phản ứng ở góc khung, hay nhân vật chính trong cảnh (đưa vào Bước 2 của Story Architect)?
2. **Ai chọn trạng thái theo từng đoạn thoại:** Visual Director ghi dòng `AVATAR` cho mỗi shot, hay một bước phân loại cảm xúc của lời thoại? Kèm quy tắc chuyển trạng thái (idle → surprised) để không giật.
3. **Hai engine:** Remotion dựng thẳng bằng `LottieClip`. Manim thì không nhúng được; cần một bước ghép phủ (overlay) ở `video-assembly`, ví dụ render avatar riêng có nền trong suốt rồi ghép bằng ffmpeg.
4. **Bố cục:** vị trí, kích thước, vùng an toàn, tránh vùng phụ đề, viền tự chọn theo độ sáng của nền.
5. **Điều kiện tiên quyết:** duyệt giấy phép clip gốc (mục 6), rồi catalog vào prompt của Remotion Engineer và (nếu chọn) của Visual Director.
6. **Kiểm thử:** render một video ngắn có avatar đổi ít nhất ba trạng thái, kiểm tra không lệch nhịp với lời thoại.

## Tiêu chí chấp nhận
1. Manifest rỗng: hệ thống hoạt động y như trước, prompt ghi "chưa có clip".
2. Thêm một asset `candidate` → gallery hiện nó; không xuất hiện trong prompt.
3. Chuyển `approved` → sinh lại catalog → prompt có nó → script dùng `LottieClip` render ra mp4 đúng, frame ổn định giữa các lần render.
4. `colors` đổi được màu; id sai bị lint chặn.
