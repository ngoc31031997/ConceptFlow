# CR-072 — Câu hỏi làm rõ yêu cầu

Ý tưởng, chỉ dẫn trước đó và bảng đối chiếu từng khâu nằm ở `cr-072-2d-film-pipeline.md`. Điền câu trả lời sau mỗi thẻ `[Answer]:`; trả lời ngắn kiểu "1 B, 2 A…" trong chat cũng được, AI sẽ chép vào đây.

**Nên trả lời Question 19 (phong cách hình) trước**: nó đổi đáp án hợp lý của các câu 6, 9, 10, 12, 13 và 17. Các câu 20–23 hỏi về quy trình vĩ mô để làm ra một video kiểu Kurzgesagt.

Luồng hiện tại (main `7f327d0`, chỉ còn Remotion): 1 Khởi tạo → 2 Cấu hình → 3 Kịch bản → 4 Visual → 5 Hình minh hoạ → 6 Code → 7 Validate → 8 Review → 9 TTS → 10 Render → 11 Merge + QC → 12 Kết quả → 13 Publish.

---

## A. Mục tiêu và ý tưởng sản phẩm

## Question 1
Mục tiêu chính khi đưa quy trình làm phim 2D vào ConceptFlow là gì?

A) Chất lượng hình: video trông giống phim hoạt hình hơn (nhân vật nhất quán, miệng nhép đúng, có tiếng động, màu theo cảnh).
   - ✅ Ưu: người xem thấy khác biệt ngay.
   - ⚠️ Nhược: luồng làm việc của Creator gần như giữ nguyên; lỗi nhịp phim vẫn chỉ thấy sau render.

B) Quy trình: Creator thấy và chốt sớm hơn (nhân vật, màu, nhịp phim) trước khi tốn thời gian render, giống cách xưởng phim chốt animatic trước khi sản xuất.
   - ✅ Ưu: ít render lại cả video; lỗi nhịp và thời lượng lộ ra sớm.
   - ⚠️ Nhược: thêm khâu và màn duyệt; hình cuối không đẹp hơn nếu chỉ làm phần này.

C) Cả A và B, chia thành nhiều CR con theo thứ tự ưu tiên (xem Question 16).

D) Tổ chức lại luồng theo ba giai đoạn tiền kỳ / sản xuất / hậu kỳ cho dễ hiểu, không thêm tính năng lớn.
   - ✅ Ưu: ít rủi ro.
   - ⚠️ Nhược: chủ yếu đổi tên và nhóm bước, video không khác.

X) Other (please describe after [Answer]: tag below)

[Answer]: B và D, tôi muốn tổ chức lại luồng theo sát flow làm phim hoạt hình gồm các giai đoạn tiền kỳ/ sản xuất / hậu kì nhưng cũng muốn creator thấy và chốt sớm hơn trước khi tốn thời gian render. ngoài ra tôi cũng muốn chất lượng video tốt hơn giống phim hoạt hình hơn
## Question 2
Ý tưởng sản phẩm dùng để xét khâu nào cần: **một người làm, AI làm phần lớn; video giải thích kiến thức đăng YouTube (dài và Shorts); một giọng dẫn, hình đổi theo từng câu thoại; flat vector**. Ý tưởng này còn đúng không?

A) Đúng, giữ nguyên. Khâu nào không hợp ý tưởng này thì bỏ.

B) Đúng, nhưng muốn có nhân vật nói trên hình: người dẫn hoặc linh vật kênh xuất hiện và nói lời dẫn (vẫn một giọng).
   - ⚠️ Nhược: kéo theo lip-sync (Question 5) và nhân vật cố định cho cả video (Question 4).

C) Mở rộng sang có hội thoại: nhiều nhân vật, mỗi người một giọng, câu thoại có người nói.
   - ✅ Ưu: làm được video dạng tiểu phẩm, kể chuyện.
   - ⚠️ Nhược: đổi kịch bản (`Line` thêm người nói), TTS nhiều giọng, lip-sync từng nhân vật, nhiều việc nhất.

X) Other (please describe after [Answer]: tag below)

[Answer]: ý tưởng vẫn là một người làm, hệ thống làm phần lớn nhưng cần create review và duyệt lại. không gói gọn trong video giải thích kiến thức mà cố gắng đi theo con đường làm video chuyên nghiệp về nhiều chủ đề, kiến thức đa dạng nào đó nhưng trông giống môt bộ phim hoạt hình ngắn

---

## B. Tiền kỳ

## Question 3
Logline và tóm tắt: hiện bước 3 Kịch bản đi thẳng vào dàn ý (câu hỏi cốt lõi, các beat). Có cần thêm logline (một câu) và tóm tắt (một đoạn) để Creator chốt trước dàn ý không?

A) Không cần, dàn ý hiện tại đã có câu hỏi cốt lõi, đủ dùng.

B) Có: AI viết logline và tóm tắt trước, Creator sửa hoặc duyệt, rồi AI viết dàn ý theo đó.
   - ✅ Ưu: chốt hướng video bằng một câu trước khi AI viết dài; dễ đổi hướng sớm.
   - ⚠️ Nhược: thêm một lượt gọi AI và một chỗ duyệt.

C) Có, nhưng chỉ hiển thị (AI tự rút từ dàn ý), không phải bước duyệt riêng.
   - ✅ Ưu: không thêm thao tác.
   - ⚠️ Nhược: không giúp chốt hướng sớm.

X) Other (please describe after [Answer]: tag below)

[Answer]: B. hệ thống làm nhưng creator phải được review và duyệt nhưng thiết kế làm sao để không phải chỉnh sửa nhiều. nghĩa là phải tối ưu câu trả lời từ AI 

## Question 4
Thiết kế nhân vật: storyboard hiện chỉ có `hero` là chữ; dáng, tuổi, áo, nét mặt được đặt lại ở từng shot nên nhân vật có thể lệch giữa các shot. Nên làm thế nào?

A) Giữ như hiện tại.

B) Bảng nhân vật cấp video (character sheet): AI đề xuất từng nhân vật một lần (dáng, tuổi, áo, màu, nét mặt mặc định, các biểu cảm cho phép), Creator xem hình mẫu và duyệt; mọi shot chỉ tham chiếu tên nhân vật và chọn biểu cảm, tư thế.
   - ✅ Ưu: nhân vật đồng nhất cả video, giống model sheet của xưởng phim; AI viết shot ngắn hơn.
   - ⚠️ Nhược: đổi dữ liệu storyboard và prompt; cần màn xem nhân vật.

C) Như B, và lưu nhân vật thành thư viện dùng lại qua nhiều video (linh vật kênh; nối với backlog avatar Mướp của CR-038).
   - ✅ Ưu: kênh có nhân vật quen thuộc.
   - ⚠️ Nhược: thêm bảng dữ liệu và màn quản lý thư viện.

X) Other (please describe after [Answer]: tag below)

[Answer]: C

## Question 5
Khẩu hình và lip-sync: miệng nhân vật hiện chỉ mở đóng theo sóng sin cố định, không theo giọng thật.

A) Không cần (nếu nhân vật ít khi nói trên hình).

B) Nhép theo độ lớn giọng thật: đọc âm lượng của file TTS theo từng khung hình, miệng mở to khi giọng to, đóng khi lặng.
   - ✅ Ưu: rẻ, không cần công cụ mới; miệng dừng đúng lúc ngừng nói.
   - ⚠️ Nhược: không phân biệt được âm (o, a, m…).

C) 💡 Gợi ý: mouth chart thật (viseme): một công cụ phân tích âm như Rhubarb Lip Sync đọc file giọng và trả ra hình miệng theo thời gian (khoảng 6–9 hình miệng chuẩn), rig miệng vẽ theo bảng đó.
   - ✅ Ưu: giống phim hoạt hình thật nhất.
   - ⚠️ Nhược: thêm công cụ vào container; Rhubarb làm tốt với tiếng Anh, tiếng Việt chưa chắc, phải thử trước; vẽ thêm 6–9 hình miệng cho rig.

X) Other (please describe after [Answer]: tag below)

[Answer]: A — Creator 2026-10-13: "vậy thì bỏ phần nhép miệng đi". Không làm lip-sync và mouth chart; miệng nhân vật giữ hình cố định theo biểu cảm.

## Question 6
Color script và style guide: hiện cả video dùng một `palette`, còn `mood` của từng cảnh là chữ tự do.

A) Giữ một palette cho cả video.

B) Color script có cấu trúc: AI gán cho mỗi cảnh một mood trong danh sách cố định (ví dụ ấm, lạnh, căng thẳng, kết luận), mỗi mood là một biến thể của palette chung (sáng/tối, độ ấm, màu nhấn). Creator xem dải màu của cả video trên một hàng.
   - ✅ Ưu: màu đổi theo cảm xúc nhưng vẫn cùng một bộ; nhìn một hàng là thấy nhịp màu của phim.
   - ⚠️ Nhược: phải thiết kế các biến thể và kiểm tra độ tương phản chữ.

C) Mỗi cảnh một palette riêng do AI chọn.
   - ✅ Ưu: tự do nhất.
   - ⚠️ Nhược: dễ mất đồng nhất, khó kiểm soát tương phản.

X) Other (please describe after [Answer]: tag below)

[Answer]: chỗ này cần investigate thêm. nếu dùng color script và style guide theo pallete thì có làm cho video trông nhàm chán và thiếu sáng tạo không? có đề xuất nào khác không? → vòng 2 (C1): A, hành trình màu.

## Question 7
Thu âm sơ bộ (scratch voice): hiện TTS chạy ở bước 9, sau Review. Muốn có animatic (Question 8) thì giọng phải có trước. Nên làm thế nào?

A) Giữ TTS sau Review.

B) Chạy TTS sớm, ngay sau khi kịch bản và storyboard xong, bằng giọng chính thức. Câu nào không đổi thì bước 9 dùng lại file đã có (TTS đã có sẵn cơ chế dùng lại).
   - ✅ Ưu: thời lượng shot lấy từ giọng thật ngay từ đầu; không tốn thêm khi câu không đổi.
   - ⚠️ Nhược: câu nào sửa sau đó phải đọc lại.

C) Hai lượt: giọng tạm (nhanh, rẻ) để làm animatic, giọng chính thức ở bước 9.
   - ✅ Ưu: sửa kịch bản thoải mái không tốn giọng chính thức.
   - ⚠️ Nhược: thời lượng giọng tạm và giọng thật lệch nhau, nhịp đã chốt có thể sai.

X) Other (please describe after [Answer]: tag below)

[Answer]: B

## Question 8
Animatic: hiện lần đầu Creator thấy hình cùng giọng là sau Render và Merge. Có cần xem nhịp phim sớm hơn không?

A) Không cần.

B) Animatic thô: sau Validate, render cả video ở mức thấp (480p15) ghép với giọng, Creator xem và chốt nhịp ở màn Review trước khi render chính thức.
   - ✅ Ưu: thấy chuyển động thật; dùng lại mức `480p15` đã có.
   - ⚠️ Nhược: vẫn cần Code chạy xong mới có; mất thêm một lượt render thấp.

C) 💡 Gợi ý: animatic dạng khung tĩnh: ngay sau Visual (trước Hình minh hoạ và Code), dựng mỗi shot thành một khung tĩnh vẽ sơ (hoặc chỉ chữ mô tả shot) giữ đúng thời lượng câu thoại, ghép với giọng thành video thô.
   - ✅ Ưu: chốt nhịp và thời lượng trước khi tốn AI viết code và vẽ hình; đúng nghĩa animatic của xưởng phim.
   - ⚠️ Nhược: không thấy chuyển động; cần TTS chạy sớm (Question 7 B hoặc C).

D) Cả C (sớm, khung tĩnh) và B (trước render, chuyển động thật).

X) Other (please describe after [Answer]: tag below)

[Answer]: C

---

## C. Sản xuất

## Question 9
Cảm giác hoạt hình của chuyển động: hiện chuyển động là nội suy mượt bằng code ở tốc độ khung hình đầy đủ.

A) Giữ chuyển động mượt như hiện tại.

B) 💡 Gợi ý: kiểu "on 2s": chuyển động của nhân vật giữ mỗi tư thế 2 khung (12 hình/giây) như phim vẽ tay, còn camera và chữ vẫn mượt.
   - ✅ Ưu: trông giống hoạt hình 2D hơn mà không phải vẽ thêm; là tuỳ chọn trong Cấu hình.
   - ⚠️ Nhược: một số người xem thấy giật; phải áp đúng chỗ (chỉ rig nhân vật).

C) Có key pose rõ: rig nhân vật diễn theo tư thế chính (key pose) có giữ dáng, ease in/out mạnh, thay vì trôi đều.
   - ✅ Ưu: diễn xuất có nhịp, giống key animation.
   - ⚠️ Nhược: đổi rig và luật viết code của AI.

D) Cả B và C.

X) Other (please describe after [Answer]: tag below)

[Answer]: C

## Question 10
VFX 2D (lửa, nước, khói, tia sáng): hiện chỉ có clip Lottie.

A) Lottie là đủ.

B) Thêm bộ hiệu ứng viết bằng Remotion (hạt, khói, tia sáng, lấp lánh, mưa) mà AI gọi theo tên, giống cách gọi rig.
   - ✅ Ưu: hiệu ứng theo màu palette, không phụ thuộc file Lottie bên ngoài.
   - ⚠️ Nhược: phải viết và kiểm thử từng hiệu ứng.

X) Other (please describe after [Answer]: tag below)

[Answer]: B

---

## D. Hậu kỳ

## Question 11
Dựng phim: hiện Render xuất một `rendered.mp4` cho cả video; muốn sửa một cảnh phải dựng lại cả video.

A) Giữ dựng cả video.

B) Render và lưu theo từng cảnh, Merge ghép các cảnh. Creator dựng lại được một cảnh, đổi chuyển cảnh giữa hai cảnh mà không render lại cả video.
   - ✅ Ưu: sửa nhanh, giống khâu dựng phim; render song song được.
   - ⚠️ Nhược: đổi hợp đồng file giữa rendering và video-assembly; nhạc nền và chuyển cảnh nối giữa các file phải xử lý cẩn thận.

X) Other (please describe after [Answer]: tag below)

[Answer]: B

## Question 12
Hiệu ứng âm thanh (SFX / foley): hiện chưa có (backlog CR-042 GĐ4).

A) Không cần.

B) Thư viện SFX có sẵn (whoosh, pop, click, ding…) và AI gắn SFX vào sự kiện trong shot (chuyển cảnh, chữ hiện, số đếm); Merge trộn vào cùng giọng và nhạc.
   - ✅ Ưu: video sống động hơn nhiều với chi phí nhỏ.
   - ⚠️ Nhược: cần nguồn SFX có giấy phép dùng trên YouTube; phải giữ mức âm lượng không lấn giọng.

C) Như B, và Creator bật tắt, đổi từng SFX trên giao diện.
   - ⚠️ Nhược: thêm màn chỉnh âm thanh.

X) Other (please describe after [Answer]: tag below)

[Answer]: thư viện sfx ở đâu mà có sẵn vậy → vòng 2 (C2): đưa SFX vào backlog.

## Question 13
Chỉnh màu cuối (color grading): hiện không có.

A) Không cần (palette đã quyết định màu).

B) Một bộ lọc màu cho cả video chọn theo preset (ví dụ ấm, điện ảnh, tươi), áp ở bước Merge bằng ffmpeg.
   - ✅ Ưu: đơn giản, một chỗ.
   - ⚠️ Nhược: áp đều cả video, không theo cảnh.

C) Grading theo cảnh trong Remotion, đi theo color script (Question 6).
   - ✅ Ưu: khớp mood từng cảnh.
   - ⚠️ Nhược: phụ thuộc Question 6 B.

X) Other (please describe after [Answer]: tag below)

[Answer]: C

---

## E. Quy trình, phạm vi và ràng buộc

## Question 14
Cổng duyệt và mức tự động cho các khâu mới (nhân vật, color script, animatic…):

A) Khâu mới chạy tự động, chỉ giữ các chỗ duyệt hiện có.
   - ✅ Ưu: không thêm thao tác.
   - ⚠️ Nhược: phát hiện lỗi muộn.

B) Mỗi khâu tiền kỳ mới có chỗ duyệt riêng (Creator xem, sửa hoặc chạy lại trước khi đi tiếp).
   - ✅ Ưu: chốt từng thứ một, giống xưởng phim.
   - ⚠️ Nhược: nhiều lần bấm hơn cho một video.

C) Có công tắc trong Cấu hình: "dừng ở từng khâu để duyệt" hoặc "AI chạy liền, chỉ dừng ở Review".
   - ✅ Ưu: video nhanh thì chạy liền, video quan trọng thì duyệt kỹ.
   - ⚠️ Nhược: hai đường chạy phải test cả hai.

X) Other (please describe after [Answer]: tag below)

[Answer]: B

## Question 15
Thanh bước và cách chia việc:

A) Giữ 13 bước; khâu mới nằm trong màn đã có (ví dụ nhân vật và color script trong màn Visual, animatic trong màn Review).

B) Nhóm thanh bước thành ba giai đoạn Tiền kỳ / Sản xuất / Hậu kỳ, thêm bước khi cần.
   - ⚠️ Nhược: đổi thứ tự và số bước ở orchestrator, authoring-service, web-gui; phải theo `docs/ux-ui-design-rules.md`.

X) Other (please describe after [Answer]: tag below)

[Answer]: B

## Question 16
Thứ tự làm. AI sẽ đề xuất chia CR con và thứ tự ở bước Workflow Planning / Units Generation để Creator duyệt. Nếu đã có ưu tiên, ghi 3 khâu muốn có trước nhất (ví dụ "nhân vật, SFX, animatic"); bỏ trống thì AI đề xuất.

[Answer]: để AI đề xuất thứ tự làm

## Question 17
Chi phí và thời gian: các khâu mới có thể thêm lượt gọi AI, lượt TTS, lượt render.

A) Không được tăng đáng kể thời gian và token cho một video; khâu nào tốn thì làm tuỳ chọn.

B) Chấp nhận tăng vừa phải (khoảng dưới 30%) nếu video đẹp hơn rõ.

C) Chất lượng trước, chi phí sau.

X) Other (please describe after [Answer]: tag below)

[Answer]: C

## Question 18
Project đã có (làm trước CR này):

A) Khâu mới chỉ áp dụng cho project tạo sau khi CR vào main; project cũ vẫn mở và chạy như cũ.
   - ✅ Ưu: không cần chuyển dữ liệu.
   - ⚠️ Nhược: code phải chạy được cả dữ liệu cũ (ví dụ storyboard không có bảng nhân vật).

B) Chuyển dữ liệu project cũ sang dạng mới.
   - ⚠️ Nhược: viết migration cho storyboard đã lưu; rủi ro hỏng project cũ.

C) Không cần giữ project cũ.

X) Other (please describe after [Answer]: tag below)

[Answer]: xoá toàn bộ data cũ 

---

## F. Phong cách hình

## Question 19
Phong cách đồ hoạ tham khảo cho video. Luật style hiện tại (`illustration_style_vi.txt`, `backdrop_style_vi.txt`) đã có nhiều điểm giống Kurzgesagt – In a Nutshell: vector phẳng, không viền, bo tròn, bóng mềm bằng gradient, nền 4 lớp parallax. Khác lớn nhất: CR-067 đặt **nền sáng bão hoà là mặc định, nền tối phải có lý do**, còn Kurzgesagt chủ yếu dùng nền xanh tím đậm với vật phát sáng.

A) Giữ phong cách hiện tại (nền sáng bão hoà theo ý, kiểu sách tranh, CR-067).
   - ✅ Ưu: không đổi luật, không đổi 265 hình và 18 nền trong thư viện.
   - ⚠️ Nhược: không ra chất Kurzgesagt.

B) Chuyển cả kênh sang hướng kiểu Kurzgesagt: nền đậm (xanh đêm, tím than) làm mặc định, màu nhấn bão hoà phát sáng (glow, bloom), ánh sáng theo nguồn sáng trong cảnh thay vì cố định trái-trên, viền sáng (rim light) phía nguồn sáng, nền dày chi tiết nhỏ chuyển động (hạt, sao, tế bào), máy quay zoom xuyên tỉ lệ, chuyển động mượt có ease mạnh và vượt đà, gần như mọi chuyển động đều có SFX.
   - ✅ Ưu: chất "phim khoa học" rõ, hợp video giải thích dài.
   - ⚠️ Nhược: đảo ngược FR7 của CR-067; phải sửa luật S1, S6, S10, B4 và vẽ lại hoặc kiểm lại thư viện hình trên nền tối; glow (`feGaussianBlur`) và nhiều hạt làm render Remotion chậm hơn rõ.

C) 💡 Gợi ý: bộ style chọn được theo video trong Cấu hình, ví dụ "Sáng tươi" (hiện tại) và "Vũ trụ phát sáng" (hướng Kurzgesagt). Mỗi bộ có luật style, palette, nền mặc định và preset nhạc/SFX riêng trong DB (luật style đã nằm trong DB từ CR-067).
   - ✅ Ưu: không bỏ hướng đang có; chọn theo chủ đề (vũ trụ, cơ thể, vi mô → phát sáng; đời sống, trẻ em → sáng tươi).
   - ⚠️ Nhược: hai bộ luật phải giữ và kiểm thử song song; thư viện hình phải đọc rõ trên cả hai loại nền.

X) Other (please describe after [Answer]: tag below)

Lưu ý dù chọn B hay C: chỉ học phong cách, không dùng lại con chim linh vật, logo, tên Kurzgesagt hay hình của họ trong video, tiêu đề, thumbnail.

[Answer]: phong cách hình là thao khảo không có khuôn khổ nào, nó phải được quyết định dựa trên chủ đề của video, ví dụ chủ đề bên trong quả sung thì phải làm sao để giống cả giác bên trong quả sung, chủ đề vũ trụ thì phải nền tối v.v chủ đề vua ai cập thì bối cảnh lăng bộ, sa mạc, chủ đề động vật thì trong rừng, chủ đề cá thì dứoi biển v.v.

---

## G. Quy trình vĩ mô để làm ra một video kiểu Kurzgesagt

Một xưởng kiểu Kurzgesagt làm một video trong nhiều tháng, theo thứ tự: chọn câu hỏi lớn → nghiên cứu với nguồn và chuyên gia → kịch bản viết lại nhiều vòng và được kiểm chứng → thu giọng → thiết kế thế giới của video (khung mẫu) → storyboard và animatic → dựng hình và diễn hoạt → nhạc riêng và sound design → phát hành kèm danh sách nguồn. Đối chiếu với ConceptFlow, phần chưa có nằm ở các câu dưới (giọng trước, animatic và SFX đã hỏi ở câu 7, 8, 12).

## Question 20
Nghiên cứu và kiểm chứng: bước 3 hiện chỉ có trường "Kiểm chứng" do AI tự ghi mức chắc chắn cho từng beat, không có nguồn thật.

A) Giữ như hiện tại.

B) Thêm bước Nghiên cứu trước Kịch bản: AI liệt kê các khẳng định video sẽ nói (số liệu, cơ chế, mốc lịch sử); Creator dán hoặc đính kèm nguồn (link, PDF, ghi chú) cho từng khẳng định; AI viết kịch bản chỉ từ các khẳng định đã có nguồn; mô tả YouTube tự có danh sách nguồn.
   - ✅ Ưu: video giải thích đáng tin, đúng tinh thần "có nguồn" của Kurzgesagt; không cần dịch vụ mới.
   - ⚠️ Nhược: Creator phải tự tìm nguồn; thêm một màn và một chỗ duyệt.

C) 💡 Gợi ý: như B, nhưng AI tự tìm nguồn bằng một API tìm kiếm web và đọc trang nguồn, Creator chỉ duyệt hoặc bỏ từng nguồn.
   - ✅ Ưu: nhanh, ít việc tay.
   - ⚠️ Nhược: cần khoá API và tốn phí theo lượt; AI vẫn có thể trích sai nên Creator vẫn phải duyệt; thêm một dịch vụ bên ngoài.

X) Other (please describe after [Answer]: tag below)

[Answer]: A

## Question 21
Khung mẫu (style frames): xưởng phim vẽ hoàn chỉnh vài khung của các cảnh chính để chốt "thế giới" của video trước khi làm hàng loạt. Hiện ConceptFlow vẽ hình minh hoạ rồi viết code cho mọi shot, lần đầu thấy cảnh hoàn chỉnh là sau render.

A) Không cần.

B) Sau storyboard, AI dựng 2–3 khung hoàn chỉnh (nền, nhân vật, màu, ánh sáng) cho các cảnh quan trọng nhất; Creator duyệt rồi mới vẽ và code phần còn lại theo đúng khung đó.
   - ✅ Ưu: chốt cái nhìn sớm và rẻ; các shot sau có mẫu để bám, video đồng nhất hơn.
   - ⚠️ Nhược: thêm một lượt vẽ và một chỗ duyệt; khung mẫu cần render ảnh tĩnh từ Remotion.

X) Other (please describe after [Answer]: tag below)

[Answer]: B nhưng detail hơn làm giống một xưởng phim thật sự 

## Question 22
Một thế giới liền mạch: video kiểu Kurzgesagt đi như một hành trình máy quay liên tục (zoom vào một tế bào thành cảnh sau, kéo ra thành Trái Đất, vật này biến thành vật kia). Hiện mỗi cảnh là một `Scene` riêng, nối bằng `transition_in`.

A) Giữ cảnh rời nối bằng chuyển cảnh.

B) Storyboard lập "hành trình máy quay" cho cả video: mỗi cảnh ghi nó nối sang cảnh sau bằng gì (zoom xuyên vào vật, kéo ra tỉ lệ lớn hơn, vật biến hình, máy quay lia tiếp); code dùng chung vật nối giữa hai cảnh.
   - ✅ Ưu: chất "một thế giới" rõ nhất của phong cách này.
   - ⚠️ Nhược: hai cảnh liền nhau phụ thuộc nhau, khó sửa riêng một cảnh; nếu chọn render từng cảnh (câu 11 B) thì phải render thêm phần chồng ở chỗ nối.

X) Other (please describe after [Answer]: tag below)

[Answer]: B

## Question 23
Nhạc nền: hiện Creator tải một file nhạc cho project; video-assembly hạ nhạc khi có giọng (ducking).

A) Giữ như hiện tại.

B) Thư viện nhạc chia theo mood; AI chọn bài và đổi bài hoặc đổi đoạn ở các mốc lớn của video theo color script (câu 6) và dàn ý.
   - ✅ Ưu: nhạc đi theo cảm xúc từng phần; không cần dịch vụ mới.
   - ⚠️ Nhược: cần nguồn nhạc có giấy phép dùng trên YouTube; nối hai bài cho mượt phải xử lý ở Merge.

C) Nhạc riêng cho từng video bằng một dịch vụ AI tạo nhạc.
   - ✅ Ưu: gần cách Kurzgesagt có nhạc riêng mỗi video.
   - ⚠️ Nhược: dịch vụ ngoài, tốn phí; phải kiểm điều khoản bản quyền của dịch vụ khi đăng YouTube.

X) Other (please describe after [Answer]: tag below)

[Answer]: A nhưng đưa ý tưởng này vào backlog có thể làm sau.
