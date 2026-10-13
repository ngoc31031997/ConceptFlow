# CR-072 — Câu hỏi làm rõ, vòng 2

Câu trả lời vòng 1 nằm trong `cr-072-requirement-questions.md`. File này có ý kiến của AI cho các câu Creator nhờ xem thêm (6, 12, 19, 21), và vài câu nảy ra từ câu trả lời (2, 18). Điền sau mỗi thẻ `[Answer]:`, hoặc trả lời ngắn trong chat.

Các câu đã rõ, không hỏi lại: 1 (B + D, và video đẹp hơn), 3 B, 4 C, 5 A, 7 B, 8 C, 9 C, 10 B, 11 B, 13 C, 14 B, 15 B, 16 (AI đề xuất thứ tự), 17 C, 20 A, 22 B, 23 A (đưa vào backlog).

Hai chỗ AI sẽ tự xử lý ở bước thiết kế, ghi ra để Creator biết (Creator 2026-10-13: "đúng", "ok"):
- **Câu 11 B (render từng cảnh) và câu 22 B (máy quay liền mạch) va nhau**: chỗ nối hai cảnh dùng chung một vật, nên mỗi cảnh phải render thêm một đoạn chồng ở đầu và cuối để Merge ghép khớp. Sửa một cảnh có thể kéo theo render lại đoạn nối với cảnh bên cạnh.
- **Câu 3 và câu 14 B**: có thêm nhiều chỗ duyệt, nhưng Creator không muốn phải sửa nhiều. Mỗi chỗ duyệt sẽ có "làm lại mục này kèm ghi chú" thay vì sửa tay, và AI tự kiểm theo luật trước khi đưa Creator xem. Đây sẽ là một yêu cầu phi chức năng của CR.

---

## Question C1 — Màu (tiếp câu 6)

> Creator: "nếu dùng color script và style guide theo palette thì có làm cho video trông nhàm chán và thiếu sáng tạo không? có đề xuất nào khác không?"

**Ý kiến của AI.** Video nhàm không phải vì có bảng màu, mà vì một bộ màu được rải đều từ đầu đến cuối. Hiện storyboard đã có `palette` riêng cho từng video do Đạo diễn AI chọn, nhưng cả video dùng chung bộ đó, cộng thêm luật nền màu đặc sáng cho cảnh giải thích ý (FR7 của CR-067). Color script ở xưởng phim (ví dụ Pixar) là **kế hoạch thay đổi màu theo câu chuyện**: mỗi phần phim đổi độ ấm và độ sáng, cao trào có tương phản mạnh nhất, đoạn kết dịu lại. Phim vẫn đồng nhất nhờ cùng một thế giới, cùng logic ánh sáng và cùng cách vẽ, không nhờ cùng một bộ màu.

Đề xuất "hành trình màu":
1. **Bảng màu của thế giới** lấy từ chủ đề (khớp câu 19): lòng quả sung → đỏ tía, hồng thịt, vàng hạt; Ai Cập → cát vàng, vàng kim, xanh lapis; biển → xanh ngọc tối dần theo độ sâu; vũ trụ → xanh đêm, tím, các điểm sáng.
2. **Một color key cho mỗi phần của dàn ý** (mở, khám phá, cao trào, kết), gồm độ ấm/lạnh, độ sáng, màu nhấn và nguồn sáng. Creator xem cả dải màu trên một hàng.
3. **Một khoảnh khắc phá bảng màu**: cảnh cao trào được dùng một màu nằm ngoài bảng để tạo ấn tượng.
4. **Đổi ánh sáng thay vì đổi màu**: giờ trong ngày, nguồn sáng, đèn trong cảnh. Có khác biệt mà không mất đồng nhất.
5. **Kiểm tra tự động**: chữ đủ tương phản; hai cảnh liền nhau không quá giống (tránh nhàm) và không lệch khỏi bảng màu thế giới (tránh rời rạc).

A) Hành trình màu như trên (AI đề xuất).
   - ✅ Ưu: vừa đa dạng vừa đồng nhất; câu 13 C (chỉnh màu theo cảnh) dùng luôn các color key này.
   - ⚠️ Nhược: Đạo diễn AI phải lập thêm color key cho từng phần; cần thêm luật kiểm tra mới.

B) Mỗi cảnh một bảng màu riêng, AI chọn tự do (câu 6 C cũ).
   - ✅ Ưu: tự do nhất.
   - ⚠️ Nhược: dễ thành mỗi cảnh một phim; khó kiểm tra tương phản và độ đồng nhất.

C) Giữ một bảng màu cho cả video như hiện tại.

X) Other (please describe after [Answer]: tag below)

[Answer]: A — Creator 2026-10-13: "câu 6 chốt như bạn đề xuất".

## Question C2 — Hiệu ứng âm thanh (tiếp câu 12)

> Creator: "thư viện sfx ở đâu mà có sẵn vậy"

**Ý kiến của AI.** Chưa có sẵn trong ConceptFlow, phải tuyển. Các nguồn dùng được cho video YouTube có kiếm tiền:
- **Miễn phí, dùng thương mại được**: Kenney (kenney.nl, CC0, nhiều âm giao diện, pop, click, va chạm); Freesound (freesound.org, mỗi âm một giấy phép, lọc CC0 thì không cần ghi công, có API); Sonniss GDC Game Audio Bundle (gói miễn phí phát hằng năm, rất lớn, dùng thương mại không cần ghi công, không được bán lại âm rời); YouTube Audio Library (có mục hiệu ứng âm thanh, tải tay).
- **Trả phí theo thuê bao**: Epidemic Sound, Artlist (có cả SFX và nhạc). Điều khoản giấy phép phải đọc kỹ trước khi dùng.
- **AI tạo SFX từ mô tả chữ**: ElevenLabs Sound Effects có API, trả phí theo lượt.

Với câu 19 (phong cách theo chủ đề) nên tuyển thêm **âm nền theo thế giới**: rừng, biển, gió sa mạc, tiếng ù vũ trụ, bên trong cơ thể. Âm nền tạo cảm giác "đang ở trong thế giới đó" rất rõ.

A) Thư viện tự tuyển, lưu cùng hệ thống: khoảng 150–250 âm CC0 (Kenney, Freesound CC0, Sonniss), gắn thẻ theo sự kiện (whoosh, pop, click, ding, va chạm, dâng lên) và theo thế giới (âm nền). Mỗi file ghi nguồn và giấy phép. AI chọn theo thẻ. Creator nghe duyệt danh sách ban đầu một lần.
   - ✅ Ưu: không tốn phí, không phụ thuộc dịch vụ ngoài, giấy phép rõ.
   - ⚠️ Nhược: phải có người nghe và chọn lúc đầu; âm hiếm có thể thiếu.

B) Như A, thêm AI tạo SFX bằng API khi thư viện không có âm phù hợp.
   - ✅ Ưu: không bao giờ thiếu âm.
   - ⚠️ Nhược: cần khoá API, tốn phí; âm tạo ra phải nghe lại.

C) Dùng thư viện của một dịch vụ thuê bao (Epidemic Sound hoặc Artlist); Creator tải về rồi hệ thống gắn thẻ.
   - ⚠️ Nhược: phí hằng tháng; giấy phép phụ thuộc thuê bao.

X) Other (please describe after [Answer]: tag below)

Đi kèm: câu 12 còn chọn giữa B (AI tự gắn) và C (Creator chỉnh từng SFX). Với câu 14 B, AI đề xuất: **AI tự gắn, Creator xem ở màn Dựng & âm thanh, bật tắt hoặc đổi từng SFX** (tức là 12 C).

[Answer]: Đưa SFX vào backlog — Creator 2026-10-13: "câu 12 sfx đưa vào backlog". Không làm SFX và âm nền trong CR này; câu 12 cũng chuyển thành backlog.

## Question C3 — Phong cách theo chủ đề (tiếp câu 19)

> Creator: "phong cách hình là tham khảo không có khuôn khổ nào, nó phải được quyết định dựa trên chủ đề của video…"

**Ý kiến của AI.** Đồng ý, và hệ thống đã đi một nửa đường này: storyboard có `world`, mỗi cảnh có `setting`, bảng màu riêng cho từng video, luật nền nói "nền là một NƠI CHỐN". Bộ kit có 6 nền (InsideBody, Meadow, Room, Space, Street, Underwater), cộng các nền trong thư viện. Ba chỗ đang chặn:
- FR7 của CR-067 bắt cảnh giải thích ý dùng nền màu đặc sáng (xem C4).
- Luật hình cố định ánh sáng từ trái-trên và màu mặc định theo bảng màu kênh.
- Luật "mọi hình như do cùng một hoạ sĩ vẽ".

Câu cần chốt: **giữa các video, cái gì giữ nguyên để người xem vẫn nhận ra kênh?**

A) Giữ **ngôn ngữ hình** chung (vector phẳng, bo tròn, không viền, cách vẽ nhân vật); đổi **thế giới** theo chủ đề: bối cảnh, bảng màu, ánh sáng, chất liệu bề mặt, âm nền.
   - ✅ Ưu: kênh có chữ ký riêng giống Kurzgesagt; thư viện hình hiện có vẫn dùng lại được.
   - ⚠️ Nhược: không đổi được cả kiểu vẽ (ví dụ vẽ hẳn kiểu tranh tường Ai Cập).

B) 💡 Gợi ý: như A, và cho phép **lớp chất liệu theo chủ đề** ở nền và đạo cụ: Ai Cập có vân giấy cói và hoạ tiết phù điêu; vũ trụ glow mạnh; dưới biển có vệt sáng gợn nước (caustics); bên trong quả sung mềm, ướt, ánh sáng xuyên qua vỏ. Nhân vật giữ ngôn ngữ chung.
   - ✅ Ưu: mỗi chủ đề có cảm giác riêng rõ hơn A mà vẫn là một kênh.
   - ⚠️ Nhược: luật hình phải có thêm phần "chất liệu thế giới"; vẽ nền tốn hơn.

C) Tự do hoàn toàn: mỗi video một kiểu vẽ riêng.
   - ✅ Ưu: sáng tạo nhất.
   - ⚠️ Nhược: không có chữ ký kênh; thư viện hình khó dùng lại nên mỗi video phải vẽ lại nhiều; khó kiểm tra tự động.

X) Other (please describe after [Answer]: tag below)

[Answer]: B — Creator 2026-10-13: "câu 19 chốt như bạn đề xuất" (AI đề xuất B: giữ ngôn ngữ hình chung, đổi thế giới theo chủ đề, thêm lớp chất liệu theo chủ đề).

## Question C4 — Cảnh giải thích ý (số liệu, sơ đồ, khái niệm)

Theo C3, video sống trong thế giới của chủ đề. Còn cảnh giải thích ý trừu tượng (đồ thị, con số, so sánh) thì sao? Hiện FR7 của CR-067 cho chúng một nền màu đặc sáng bão hoà, đổi màu khi đổi ý.

A) Đặt luôn trong thế giới, như Kurzgesagt: con số và sơ đồ là vật trong cảnh (đếm hạt trong quả sung, đồ thị vẽ trên vách lăng mộ, so sánh kích thước bằng các thiên thể).
   - ✅ Ưu: phim liền mạch, không "nhảy ra slide"; hợp câu 22 B.
   - ⚠️ Nhược: Đạo diễn AI và Kỹ sư AI phải nghĩ ra cách đặt ý vào cảnh, khó hơn; thay FR7 của CR-067 (cần một ADR).

B) Giữ nền màu đặc, nhưng màu lấy từ bảng màu của thế giới thay vì bộ màu sáng cố định.
   - ✅ Ưu: dễ làm, ý trừu tượng đọc rõ.
   - ⚠️ Nhược: vẫn có cảm giác chuyển sang slide giữa phim.

C) AI chọn cho từng cảnh: ý gắn được với thế giới thì làm như A, không gắn được thì làm như B.

X) Other (please describe after [Answer]: tag below)

[Answer]: A — AI hiểu từ "câu 19 chốt như bạn đề xuất", vì trong chat AI gộp C4 vào phần câu 19 và đề xuất "đặt luôn trong thế giới thay vì nền màu đặc". Creator xác nhận 2026-10-13: "đúng".

## Question C5 — Bộ thiết kế mỹ thuật kiểu xưởng phim (tiếp câu 21)

> Creator: "B nhưng detail hơn làm giống một xưởng phim thật sự"

**Ý kiến của AI.** Ở xưởng phim, giai đoạn visual development tạo ra một **bộ thiết kế mỹ thuật** (art bible) mà cả đội bám theo. Ứng vào ConceptFlow, mỗi mục là chữ hoặc ảnh tĩnh render bằng Remotion (`renderStill`, đang dùng cho ảnh xem trước hình minh hoạ):

1. **Bản mô tả thế giới**: nơi chốn, thời đại, chất liệu, logic ánh sáng, bảng màu thế giới, âm nền; kèm một bảng ghép hình tham khảo (moodboard) từ thư viện có sẵn.
2. **Bản thiết kế bối cảnh**: mỗi nơi chốn chính một nền 4 lớp, xem ở khung ngang và khung dọc, kèm biến thể ánh sáng nếu câu chuyện cần (ngày/đêm, trước/sau).
3. **Bản thiết kế nhân vật** (câu 4 C): model sheet trước/nghiêng/sau, biểu cảm, tư thế tay, nhân vật đặt trên từng nền chính để kiểm độ nổi.
4. **Bản thiết kế đạo cụ**: các vật quan trọng của câu chuyện.
5. **Color key**: mỗi phần của dàn ý một khung màu (hành trình màu ở C1).
6. **Khung mẫu (style frames)**: 3–5 khung hoàn chỉnh của các khoảnh khắc chính (mở, cao trào, kết), đúng chất lượng của video cuối.

Thứ tự phụ thuộc: kịch bản → (1) → (2), (3), (4) → storyboard → (5) → (6) → giọng sơ bộ → animatic. Mục 1–4 chỉ cần kịch bản; mục 5–6 cần storyboard để biết khoảnh khắc nào quan trọng.

A) Đủ 6 mục, mỗi mục duyệt riêng theo thứ tự trên.
   - ✅ Ưu: chốt kỹ từng thứ, đúng kiểu xưởng phim.
   - ⚠️ Nhược: khoảng 6 lần duyệt trước khi tới animatic.

B) Đủ 6 mục, chia hai lượt duyệt: lượt 1 duyệt mục 1–4 trên một màn (mỗi mục có "duyệt" hoặc "làm lại kèm ghi chú"); lượt 2 duyệt mục 5–6 sau storyboard.
   - ✅ Ưu: vẫn đủ như xưởng phim nhưng ít lần chờ; khớp câu 3 (ít phải sửa).
   - ⚠️ Nhược: một màn duyệt chứa nhiều thứ.

C) Gọn: chỉ bối cảnh (2), nhân vật (3) và khung mẫu (6).
   - ⚠️ Nhược: không còn "chi tiết như xưởng phim thật" như Creator muốn.

X) Other (please describe after [Answer]: tag below)

[Answer]: B — Creator 2026-10-13: "câu 21 ok chốt như bạn đề xuất" (đủ 6 mục). Cách duyệt AI hiểu là B (hai lượt duyệt, khớp câu 3); Creator xác nhận 2026-10-13: "đúng".

## Question C6 — Giọng nói trong phim (tiếp câu 2)

Câu 2 mở rộng thành "video chuyên nghiệp về nhiều chủ đề, trông như phim hoạt hình ngắn", và câu 5 đã bỏ nhép miệng. Nhân vật có nói không?

A) Chỉ một giọng dẫn kể chuyện; nhân vật diễn bằng hình, như Kurzgesagt.
   - ✅ Ưu: hợp với việc không nhép miệng; giữ được cơ chế TTS và thời lượng shot hiện tại.
   - ⚠️ Nhược: khó làm dạng tiểu phẩm có đối thoại.

B) Giọng dẫn là chính; thỉnh thoảng nhân vật nói một câu ngắn bằng giọng riêng (TTS nhiều giọng). Không nhép miệng: lúc nói thì đổi biểu cảm, nhún người hoặc hiện bóng thoại.
   - ✅ Ưu: phim có hơi thở hoạt hình hơn mà chi phí vừa phải.
   - ⚠️ Nhược: câu thoại (`Line`) phải có thêm người nói; TTS chọn giọng theo nhân vật.

C) Hội thoại đầy đủ giữa nhiều nhân vật.
   - ⚠️ Nhược: không nhép miệng thì cảnh đối thoại dài trông thiếu; việc nhiều nhất.

X) Other (please describe after [Answer]: tag below)

[Answer]: A — Creator 2026-10-13: "tạm thời để 1 người kể". Nhân vật có giọng riêng đưa vào backlog.

## Question C7 — Xoá dữ liệu cũ (tiếp câu 18)

> Creator: "xoá toàn bộ data cũ"

Xoá không lấy lại được, nên cần chốt phạm vi. Mặc định AI sẽ dump DB và nén thư mục `/shared` vào một file sao lưu trước khi xoá, trừ khi Creator nói không cần. Video đã đăng YouTube không bị đụng tới.

A) Xoá project, video và file của project trong `/shared`, cùng dữ liệu authoring của các project đó. **Giữ** thư viện hình và nền, prompt và luật style, kiểu video (archetype), cấu hình kênh và kết nối YouTube.
   - ✅ Ưu: bắt đầu sạch mà không mất công sức đã làm cho thư viện và prompt.

B) Như A, xoá thêm thư viện hình và nền để vẽ lại theo luật mới.
   - ⚠️ Nhược: mất toàn bộ thư viện (lúc CR-067 có khoảng 265 hình và 18 nền); phải vẽ lại từ đầu.

C) Xoá tất cả, bắt đầu từ DB trống (chỉ còn dữ liệu hệ thống tự gieo khi khởi động).

X) Other (please describe after [Answer]: tag below)

Thời điểm xoá: khi CR con đầu tiên đổi dạng dữ liệu (storyboard, bảng nhân vật) vào main.

[Answer]: X — Creator 2026-10-13: "xoá hết toàn bộ project, prompt, luật style để làm lại từ đầu, thư viện hình tôi đã backup ra file zip nên xoá được luôn". Xoá: mọi project và file của chúng, prompt (cả bản Creator sửa), luật style, thư viện hình và nền. Creator không nhắc tới kiểu video (archetype), cấu hình kênh và kết nối YouTube nên giữ. AI vẫn dump DB và nén `/shared` trước khi xoá.
