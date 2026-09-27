# CR-043 — Minh hoạ phẳng kiểu tranh kể chuyện: hình phải minh hoạ đúng cái lời thoại đang nói

## Date
2026-09-27

## Stage
Code Generation (engine Remotion). Chờ Creator chạy lại một video thật để duyệt.

## Yêu cầu của Creator
Hình minh hoạ hiện tại xấu, và không liên quan tới nội dung: video chủ đề "sâu răng" có lời thoại nói về răng, vi khuẩn, đường, nhưng hình là các khối trừu tượng. Muốn hình giống phong cách kênh "Vẽ Chuyện" (video "Thiên kiến sống sót là gì?"): tranh vector phẳng, người có nét mặt, đồ vật, căn phòng nền vàng ấm, màn chia đôi hai thế giới. Ít nhất, hình phải minh hoạ đúng điều đang nói.

## Nguyên nhân gốc (đối chiếu với code)
1. **Prompt Đạo diễn cấm đúng thứ cần vẽ.** Luật 18 (`prompt_template_seeds.go`, `visualDirectorHeadVI`) chỉ cho "hình cơ bản, đàn chấm, lưới, đồ thị, mũi tên..." và bảo tránh "nhân vật hữu cơ"; mục tự kiểm 14 bắt "diễn đạt lại bằng hình cơ bản". Một chiếc răng hay một con vi khuẩn bị đổi thành hình tròn có nhãn trước khi tới bước dựng. Ví dụ nhân vật chính trong luật 1 cũng là "một hình vuông → vỡ thành lưới".
2. **Nền bị khoá.** "Nền video CỐ ĐỊNH #080E1C", và Kỹ sư Remotion bị cấm tô nền cho khung — không thể có căn phòng vàng hay màn chia đôi.
3. **Kỹ sư Remotion phải tự vẽ mọi hình từ `<path>`.** Không có bộ hình nào; model vẽ một chiếc răng từ toạ độ ngay lúc viết code thì hoặc trừu tượng hoặc xấu. Bộ Lottie (CR-038) mới chỉ có 10 biểu cảm của một con mèo.

## Thay đổi
### 1. Bộ minh hoạ phẳng `conceptflow-mini/illustration.tsx` (rendering, engine Remotion)
34 component vẽ tay bằng SVG, cùng một nét: phẳng, không viền, mảng bóng nhẹ, dáng tròn, mắt chấm.
- Bối cảnh: `Backdrop` (nền màu phẳng + dải sàn), `Panel` (mảng màu — màn chia đôi, vạch ngăn).
- Người: `Person` — 9 dáng (đứng, ngồi ghế đẩu, vẫy, chỉ, chống cằm, mừng, ôm má, nhún vai, đi), 7 nét mặt, trẻ em/người lớn/người già, áo sơ mi/áo blouse/váy, 4 kiểu tóc, miệng mấp máy khi nói, tự chớp mắt và thở.
- Cơ thể & sức khoẻ: `Tooth` (decay 0→1: trắng → đốm nâu → lỗ sâu → nứt, nét mặt tự đổi theo), `Germ`, `OpenMouth`, `Drop`, `Shield`, `Heart`.
- Đồ ăn: `Candy`, `Lollipop`, `Soda`, `Donut`, `Apple`.
- Đồ vật/bối cảnh: `Toothbrush`, `Toothpaste`, `Clock`, `Table`, `Chair`, `Window`, `Plant`, `House`, `Tree`, `Sun`, `Cloud`, `Lightbulb`, `Coin`, `Book`, `Phone`, `Magnifier`, `Mark`, `Sparkle`, `Airplane`, `Bubble`.
- Quy ước vị trí giống `LottieClip`/`LAYOUT`: `x`, `y` là tâm, `size` là cạnh dài; `rotate`, `flip`, `scale`, `opacity` để shot tự nội suy.
- Xem trước: `node render_gallery.mjs sheets <dir>` (hai trang tổng hợp) và `node render_gallery.mjs demo out.mp4` (mẫu "sâu răng" 4 shot, `src/gallery/tooth-decay-demo.tsx`).

### 2. Prompt Đạo diễn (thủ công v10, AI v4) — vẫn không nhắc tên API của engine nào
- Luật 18 viết lại: phim vẽ theo lối tranh phẳng kể chuyện; vật liệu dựng tốt nay gồm người (dáng, nét mặt), cơ thể & sức khoẻ, đồ ăn, đồ vật & bối cảnh; hình học chỉ dành cho ý tự nó trừu tượng.
- Luật 19 mới — **MINH HOẠ ĐÚNG CÁI ĐANG NÓI**, kèm phép thử "tắt tiếng chỉ nhìn hình, có đoán được thoại nói về cái gì không". Thêm mục tự kiểm tương ứng ở cả hai bản.
- Nền: mặc định vẫn `#080E1C`, nhưng cảnh được có nền màu phẳng riêng và chia nhiều mảng màu; màu nền là một vai trò trong BẢNG MÀU.
- Luật 1 và 14: ví dụ nhân vật chính cụ thể (chiếc răng), diễn xuất bằng nét mặt và dáng.

### 3. Prompt Kỹ sư Remotion (thủ công v6, AI v2)
- Mục C3 mới chứa API bộ minh hoạ (`{{illustration_kit}}`, nhúng lúc seed từ `prompts/illustration_kit_vi.txt`): vật nào có trong bộ thì BẮT BUỘC dùng component, kèm một shot ví dụ.
- Mục C: cho phép `Backdrop`/`Panel`; mục B: màu mặc định bên trong hình của bộ không tính là màu tự bịa.
- Luồng AI: khung code của Code Merger (`llm-service/app/pipeline/merger.py`, `ILLUSTRATION_KIT`) import sẵn cả bộ.

### 4. Prompt Kỹ sư Manim (thủ công v8, AI v3)
Manim không có bộ hình này. Thêm một dòng dịch: vật cụ thể → hình gợi dáng từ `self.shape(...)` + nhãn tên vật, để người xem vẫn biết hình là gì; nền màu riêng của cảnh → bỏ qua. **Muốn có phong cách tranh phẳng thì chọn engine Remotion.**

## Kiểm thử
- `authoring-service`: `go test ./...` xanh. Hash của 3 prompt thủ công trong `golden_prompts_test.go` được chốt lại có chủ đích; thêm `illustration_kit_test.go`.
- `llm-service`: 86/86 test xanh, `ruff` sạch.
- `rendering`: `tests/domain/test_illustration_kit.py` giữ ba nơi khớp nhau (export của `illustration.tsx`, tài liệu trong prompt, danh sách import của merger) và chạy `tsc` thật trên một shot dùng bộ minh hoạ. `tsc --noEmit` cả `remotion_project` sạch. 150 test chạy được đều xanh; 7 file test cần `manim` không nạp được trong môi trường kiểm thử này (không cài được `manimpango`), nhưng không có dòng Python nào của rendering bị sửa.

## Chưa làm / giới hạn
- Chưa chạy lại một video thật qua cả pipeline (cần stack Docker + LLM của Creator). Việc cần làm: rebuild `rendering`, `authoring-service`, `llm-service` rồi tạo lại video "sâu răng" với engine **Remotion**.
- Bộ hình phủ tốt chủ đề sức khoẻ răng miệng và đời sống thường ngày; chủ đề khác (xe cộ, động vật, nghề nghiệp...) sẽ rơi vào nhánh "ghép từ vài khối". Mở rộng bộ theo chủ đề Creator làm tiếp theo.
- Prompt đã lưu trong DB: system prompt được ghi đè khi service khởi động lại, nhưng nếu Creator đang dùng một bản prompt tự sửa (không phải bản hệ thống) thì bản đó không nhận thay đổi này.
