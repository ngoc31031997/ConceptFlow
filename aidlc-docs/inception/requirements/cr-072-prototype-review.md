# CR-072 — Trao đổi sau khi xem prototype

Prototype: https://claude.ai/artifact/HibBJUisUocqKXDZD57cTH. File này trả lời từng câu Creator hỏi khi xem prototype (2026-10-13). Với mỗi bước: hệ thống hiện làm được gì (đã kiểm trong code), đề xuất của AI, và câu hỏi cần chốt (`P-n`). Điền sau mỗi thẻ `[Answer]:` hoặc trả lời ngắn trong chat. Khi chốt xong, AI cập nhật tài liệu yêu cầu `cr-072-2d-film-pipeline.md` và prototype.

---

## 0. Nguyên tắc chung: sửa đúng chỗ, chỉ làm lại phần bị ảnh hưởng

Phần lớn câu hỏi của Creator cùng một ý: **thấy một chỗ chưa ổn thì sửa được đúng chỗ đó, không phải làm lại cả bước**. AI đề xuất đưa thành một yêu cầu chung (FR-17 mới) cho mọi bước:

1. **Sửa ở đơn vị nhỏ nhất**: một mục, một cảnh, một shot, một câu. Mỗi đơn vị có ba cách sửa:
   - **Sửa tay**: Creator gõ trực tiếp. AI không ghi đè phần Creator viết, trừ khi Creator bấm làm lại.
   - **Làm lại kèm ghi chú**: Creator viết điều chưa ổn, AI làm lại đúng đơn vị đó. Khi làm lại, AI được xem các đơn vị bên cạnh (cảnh trước, cảnh sau) để giữ mạch.
   - **Gom ghi chú**: ghi chú lên nhiều đơn vị rồi bấm "AI sửa theo các ghi chú" một lần, thay vì chạy từng cái.
2. **Chỉ làm lại phần phụ thuộc**: sửa cảnh 3 thì chỉ giọng, animatic, hình, code và render của cảnh 3 phải làm lại; các cảnh khác giữ nguyên. Trước khi chạy, màn hình báo rõ sẽ làm lại những gì. Hiện nay đổi dàn ý hay hình ảnh sẽ xoá phần sau của **cả video** (hộp xác nhận ở `ScriptOutlineStepPage.tsx`, `VisualDirectorStepPage.tsx`); nguyên tắc này thu hẹp phạm vi đó lại.
3. **Duyệt theo đơn vị**: một bước được duyệt khi mọi đơn vị của nó đã duyệt. Có nút "Duyệt tất cả mục còn lại" cho lúc Creator đã ưng hết.

| Bước | Đơn vị sửa | Sửa tay | Làm lại kèm ghi chú | Kéo theo làm lại |
|---|---|---|---|---|
| Kịch bản | logline, tóm tắt, từng beat, lời thoại từng cảnh, từng câu | có | có | lời thoại của beat/cảnh đó trở xuống |
| Mỹ thuật lượt 1 | từng mục: thế giới, mỗi bối cảnh, mỗi nhân vật, mỗi đạo cụ | mô tả, màu; SVG tải lên | có | các shot dùng mục đó |
| Storyboard | từng cảnh, từng shot, từng trường | có | có | giọng, animatic, hình, code, render của cảnh đó |
| Mỹ thuật lượt 2 | từng color key, từng khung mẫu | màu của color key | có | màu của các cảnh thuộc phần đó |
| Giọng sơ bộ | từng câu | chữ, cách đọc | đọc lại | animatic của shot đó |
| Animatic | từng shot | thời lượng | khung tĩnh | không |
| Hình minh hoạ | từng hình | prompt, SVG | có | code của các shot dùng hình |
| Code | từng shot | code | có | render của cảnh đó |
| Render, Dựng | từng cảnh | chuyển cảnh ở chỗ cắt | có (ghi chú theo mốc thời gian) | Merge |

## Question P-1 — Nguyên tắc chung

A) Đồng ý với ba ý trên (FR-17).

B) Đồng ý, nhưng bỏ "gom ghi chú", chỉ làm lại từng đơn vị một.

X) Other (please describe after [Answer]: tag below)

[Answer]: A 

---

## Bước 1 — Khởi tạo: kiểu video

> Creator: "mục 4 Kiểu video có thể cho thêm nhiều kiểu video hơn không hoặc để AI tự define hoặc gợi ý 4–5 mục rồi user chọn được không."

**Hiện nay:** bảng Kiểu video có 6 kiểu hệ thống (NGHỊCH-LÝ, SO-SÁNH… cùng XÂY-TẦNG, HÀNH-TRÌNH). Creator thêm kiểu mới ở mục "Kiểu video" trong cài đặt. Khi tạo video, Creator chọn một kiểu hoặc "Để AI tự chọn" (`VideoArchetypePicker.tsx`). Prototype vẽ 4 nút cố định nên trông như ít kiểu.

**Đề xuất:** khi Creator nhập chủ đề, AI đọc chủ đề rồi **gợi ý 3–5 kiểu hợp nhất**. Mỗi gợi ý có một câu "vì sao hợp" và một câu mở đầu mẫu theo kiểu đó, để Creator hình dung trước. Nếu không kiểu nào trong bảng thật sự hợp, AI được **đề xuất một kiểu mới** kèm cách triển khai (playbook) nháp; Creator dùng thử cho video này, ưng thì lưu vào bảng Kiểu video. Vẫn giữ "Để AI tự chọn" và "Xem tất cả kiểu".

## Question P-2

A) AI gợi ý 3–5 kiểu từ bảng, Creator chọn.

B) Như A, và AI được đề xuất kiểu mới chưa có trong bảng, lưu vào bảng được (AI đề xuất).

C) Giữ như hiện tại (chọn từ bảng hoặc để AI tự chọn).

X) Other (please describe after [Answer]: tag below)

[Answer]: B 

---

## Bước 3 — Kịch bản

> Creator: "logline và tóm tắt: nếu bỏ duyệt thì có cho creator tự viết không? / dàn ý: có cho user chỉnh sửa hoặc AI gen lại nếu creator từ chối duyệt không? / lời thoại: có cho làm lại theo ghi chú của từng cảnh không…"

**Hiện nay:** dàn ý và storyboard hiện trong ô sửa được ("Kết quả của AI hiện ở đây để bạn chỉnh sửa, rồi bấm Tiếp tục"), chạy lại AI được, nhưng chạy lại cả bước. Chưa có logline và tóm tắt.

**Đề xuất:**
- **Logline và tóm tắt**: là hai ô sửa được. Creator có thể tự viết ngay từ đầu (ô có chữ thì AI dùng chữ của Creator, ô trống thì AI viết), sửa tay bản AI viết, hoặc "Làm lại kèm ghi chú". Nút "Bỏ duyệt" trong prototype chỉ là mở khoá để sửa tiếp.
- **Dàn ý**: mỗi beat sửa tay được, làm lại một beat kèm ghi chú được, hoặc làm lại cả dàn ý kèm ghi chú. Đổi một beat thì chỉ lời thoại của beat đó viết lại.
- **Lời thoại**: mỗi cảnh có "Duyệt cảnh", "Làm lại cảnh kèm ghi chú", và sửa tay từng câu. Cảnh đã duyệt giữ nguyên khi cảnh khác làm lại. Kịch bản được duyệt khi mọi cảnh đã duyệt.

Không cần chốt thêm nếu Creator đồng ý P-1.

---

## Bước 4 — Mỹ thuật lượt 1

### Ngôn ngữ hình của kênh

> Creator: "ngôn ngữ hình: cần làm rõ thêm, tôi muốn tham khảo thêm từ 2 kênh Vẽ Chuyện và Kurzgesagt hoặc một số ngôn ngữ hình khác để chọn trước khi chốt, và ngôn ngữ kênh phải được đưa vào cài đặt để có thể thay đổi sau này. Sau này có nhiều ngôn ngữ kênh có thể bật/tắt, miễn sao chỉ có 1 ngôn ngữ được bật tại một thời điểm."

**Hiện nay:** luật style hình (`illustration_style_vi.txt`) và luật nền (`backdrop_style_vi.txt`) là một bộ duy nhất, nằm trong DB từ CR-067, Creator sửa được trong Cài đặt prompt. Theo quyết định của CR-066/CR-067, prompt không nhắc tên kênh nào, chỉ mô tả lối vẽ.

**Đề xuất:** thêm khái niệm **Ngôn ngữ hình của kênh**, quản lý ở một trang cài đặt riêng, giống cách bảng Kiểu video đang làm.
- Mỗi ngôn ngữ hình gồm: tên, mô tả ngắn, luật hình (S), luật nền (B), cách vẽ người và con vật (bộ rig cơ sở), và **bộ ảnh mẫu**. Ảnh mẫu là cùng một bộ cảnh thử (ví dụ lòng quả sung, vũ trụ, căn bếp) render bằng chính luật đó, để so các ngôn ngữ một cách công bằng.
- **Chỉ một ngôn ngữ được bật tại một thời điểm.** Video mới dùng ngôn ngữ đang bật. Video cũ giữ ngôn ngữ (và phiên bản) đã dùng khi tạo, để đổi ngôn ngữ không làm hỏng video đang dở.
- Ngôn ngữ hệ thống chỉ đọc; Creator sao chép ra để sửa, như prompt và kiểu video hiện nay.
- Thế giới theo chủ đề (C3 B) vẫn giữ: ngôn ngữ hình là phần chung của kênh, thế giới đổi theo từng video.

Bộ ngôn ngữ khởi đầu để Creator so sánh trước khi chốt. Tên trong hệ thống không dùng tên kênh, chỉ ghi "tham khảo" trong phần mô tả:

| Ngôn ngữ | Tham khảo | Đặc điểm |
|---|---|---|
| Sách tranh tươi | Vẽ Chuyện (hướng hiện tại của CR-043/CR-067) | Nền màu ấm sáng, người có nét mặt rõ, khối tròn, sáng trái-trên, ít hiệu ứng |
| Khoa học phát sáng | Kurzgesagt – In a Nutshell | Nền đậm, glow, ánh sáng theo nguồn, viền sáng, nhân vật tối giản mắt chấm, chi tiết nền dày |
| 💡 Giấy cắt lớp | kiểu cut-out giấy | Các lớp giấy có bóng đổ mềm giữa lớp, vân giấy, mép hơi răng cưa; hợp chủ đề lịch sử, thiên nhiên |
| 💡 Đường nét tối giản | kiểu infographic nét mảnh | Nét mảnh một màu, mảng màu ít, nhiều khoảng trắng; hợp chủ đề số liệu, công nghệ |

## Question P-3 — Bộ ngôn ngữ hình khởi đầu

A) Làm 2 ngôn ngữ: Sách tranh tươi và Khoa học phát sáng.

B) Làm 4 ngôn ngữ như bảng trên (AI đề xuất làm trước 2 cái đầu, 2 cái sau làm ở CR con riêng).

C) Làm 2 cái đầu, và Creator tự thêm ngôn ngữ khác qua trang cài đặt (viết luật, AI render ảnh mẫu).

X) Other (please describe after [Answer]: tag below)

[Answer]: làm default cho tôi đủ 4 kiểu

## Question P-4 — Chốt ngôn ngữ trước khi làm gì?

A) Trước khi làm CR con đầu tiên: AI dựng bộ ảnh mẫu cho các ngôn ngữ ở P-3, Creator so rồi bật một cái. Mọi luật và prompt viết lại (FR-15) theo ngôn ngữ được bật.

B) Làm luồng mới trước với ngôn ngữ hiện tại, thêm trang ngôn ngữ hình sau.

X) Other (please describe after [Answer]: tag below)

[Answer]: A

### Bảng tham khảo và cách sửa từng hình

> Creator: "bảng tham khảo có phải là hình ảnh được dùng trong video sau này không, nếu có 1 hình tôi không vừa ý thì làm sao để chỉnh sửa hình ảnh đó?"

**Trả lời:** không. **Bảng tham khảo** (moodboard) chỉ để chốt cảm giác của thế giới, không vào video. Trong prototype nó mượn tạm hình bối cảnh nên dễ nhầm. Thứ **vào video** là các mục bối cảnh, nhân vật, đạo cụ ở các tab bên cạnh.

**Đề xuất cách sửa một hình** (bối cảnh, nhân vật, đạo cụ, và cả ở bước Hình minh hoạ):
1. Làm lại kèm ghi chú ("cửa đáy quả khít hơn, ít ánh sáng hơn").
2. Sửa mô tả (prompt) của hình rồi vẽ lại, như bước Hình minh hoạ hiện nay.
3. Tải lên một file SVG tự vẽ; luật hình hiện đã cho phép hình SVG tải lên.
4. Chọn một hình khác trong thư viện thay vào.

Sửa một mục ở đây là sửa luôn nguồn chung: mọi shot dùng hình đó đều đổi theo, và màn hình báo trước sẽ ảnh hưởng những shot nào.

Với bảng tham khảo: mỗi ô bỏ được, thay được, hoặc làm lại kèm ghi chú.

## Question P-5 — Ảnh tham khảo do Creator tải lên

Creator có muốn tải ảnh tham khảo của mình (chụp màn hình một video, một bức tranh) vào bảng tham khảo không? Lưu ý: AI vẽ hiện viết SVG bằng chữ. Để AI "nhìn" được ảnh thì model phải đọc được ảnh; AI sẽ kiểm việc này ở bước thiết kế.

A) Có. Nếu model đọc được ảnh thì AI dùng ảnh làm gợi ý; nếu không thì ảnh chỉ để Creator đối chiếu.

B) Không cần, bảng tham khảo chỉ gồm hình AI dựng.

X) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## Bước 5 — Storyboard

> Creator: "có thể duyệt từng bước ở đây, nếu có chỗ nào không vừa ý thì làm sao sửa lại từng cảnh mà không cần sửa lại toàn bộ / mô tả từng cảnh là ý tưởng tốt nhưng nếu muốn thay đổi hoặc góp ý thì làm sao?"

**Hiện nay:** từ CR-050, storyboard đã được AI viết theo từng cảnh, nhưng giao diện duyệt và chạy lại vẫn tính cả bước.

**Đề xuất:**
- Mỗi cảnh có "Duyệt cảnh" và "Làm lại cảnh kèm ghi chú". Storyboard được duyệt khi đủ 6 cảnh.
- Mọi trường của cảnh và shot **sửa tay ngay tại chỗ**: bối cảnh, color key, cách nối sang cảnh sau, và trong từng shot là máy quay, hình, nhân vật (chọn từ danh sách biểu cảm và tư thế đã duyệt), câu thoại.
- **Góp ý**: mỗi cảnh và mỗi shot có ô ghi chú. Creator ghi nhiều chỗ rồi bấm "AI sửa theo các ghi chú" một lần (gom ghi chú, P-1).
- Sửa câu thoại ở đây thì câu đó được đọc lại ở bước Giọng sơ bộ. Đổi cách nối giữa hai cảnh thì cả hai cảnh được đánh dấu cần làm lại phần code ở chỗ nối.

Không cần chốt thêm nếu Creator đồng ý P-1.

---

## Bước 6 — Mỹ thuật lượt 2

> Creator: "nếu các color key và khung mẫu không ổn có cho sửa từng khung không?"

**Có.** Mỗi color key và mỗi khung mẫu là một mục riêng, có Duyệt và Làm lại kèm ghi chú. Color key còn sửa tay được: chọn màu nhấn, độ ấm, độ sáng, nguồn sáng bằng bộ chọn màu; AI kiểm lại tương phản ngay sau khi đổi. Prototype đang gộp cả 5 color key thành một mục; bản thật sẽ tách thành 5.

---

## Bước 7 — Giọng sơ bộ

> Creator: "nếu có shot nào đọc sai thì cho user gõ tay rồi render lại đúng shot đó được không?"

**Hiện nay:** TTS dùng Azure, gửi SSML. File giọng lưu theo nội dung câu, nên sửa một câu thì chỉ câu đó đọc lại (`synthesize_speech.py`).

**Đề xuất:** mỗi câu có hai cách sửa:
1. **Sửa chữ câu thoại**: đổi luôn phụ đề.
2. **Sửa cách đọc, giữ nguyên chữ**: ô "đọc là" (ví dụ "ficin" đọc là "phi-xin", "2 mm" đọc là "hai mi-li-mét"), kèm chỉnh tốc độ và khoảng nghỉ trước/sau câu. Phụ đề vẫn hiện chữ gốc. Làm bằng SSML của Azure (thẻ thay thế cách đọc); chi tiết kiểm ở thiết kế.

Rồi bấm "Đọc lại", chỉ câu đó đọc lại, và animatic của shot đó tự cập nhật.

## Question P-6 — Từ điển phát âm của kênh

A) Có thêm từ điển phát âm dùng chung cho kênh: một từ sửa cách đọc một lần, mọi video sau đọc đúng (AI đề xuất).

B) Chỉ sửa cách đọc theo từng câu.

X) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## Bước 8 — Animatic

> Creator: "làm lại ở bước này có nghĩa là gì nhỉ, và sửa câu thoại của shot nữa?"

**Trả lời:** nút "Làm lại kèm ghi chú" cho cả animatic trong prototype là chưa rõ. Animatic được ghép từ những gì đã có (khung tĩnh dựng từ bộ mỹ thuật, giọng sơ bộ), nên "làm lại" nên tách thành các việc cụ thể **trên từng shot**:
- **Đổi thời lượng shot**: giữ hình lâu hơn hoặc ngắn hơn, thêm khoảng nghỉ trước hoặc sau câu.
- **Sửa câu thoại của shot**: sửa ngay tại đây (đồng bộ ngược về storyboard), câu đó được đọc lại, animatic cập nhật đúng shot đó.
- **Đổi khung tĩnh của shot**: làm lại kèm ghi chú ("ong nhỏ hơn, đặt lệch trái").
- **Ghi chú nhịp cho cả cảnh** ("cảnh 3 chậm quá"): AI đề xuất cắt bớt hoặc gộp câu. Creator xem đề xuất rồi mới áp dụng.

"Chốt nhịp phim" là chỗ duyệt duy nhất của bước này.

## Question P-7 — Animatic được sửa đến đâu?

A) Ba việc đầu: thời lượng, câu thoại, khung tĩnh của từng shot.

B) Như A, cộng ghi chú nhịp cho cả cảnh, AI đề xuất cách sửa (AI đề xuất).

C) Như B, cộng tách, gộp và đổi thứ tự shot ngay trong animatic.

X) Other (please describe after [Answer]: tag below)

[Answer]: B

---

## Bước 9 — Hình minh hoạ

> Creator: "ở bước này vẫn như cũ cho phép prompt và chỉnh sửa từng hình ảnh đúng không?"

**Đúng.** Giữ như hiện nay (vẽ song song, mỗi hình duyệt, sửa hoặc bỏ, sửa prompt; thư viện xoá và sửa hình mẫu từ CR-052), thêm "Làm lại kèm ghi chú". Khác một điểm: hình đến từ bộ mỹ thuật hiện là "đã duyệt". Muốn sửa thì sửa ở mục gốc, và màn hình báo những shot nào dùng hình đó.

---

## Bước 10 — Code

> Creator: "đang chia theo từng cảnh, mỗi cảnh lại có shot; nếu lỗi ở cảnh hoặc ở từng shot thì cơ chế chỉnh sửa và render lại sẽ như thế nào?"

**Hiện nay** (`CodeSegmentsPanel.tsx`, CR-059, CR-063, CR-068): code chia theo đoạn vài shot. Một đoạn lỗi không làm mất đoạn khác. Mỗi đoạn có "Chạy lại đoạn này", "Chạy lại shot X, Y" (chỉ shot lỗi), dán kết quả AI ngoài, hoặc sửa tay. Hệ thống tự sửa lỗi code máy móc (autofix) và có một lượt sửa bằng AI trước khi báo lỗi.

**Đề xuất cho luồng mới:**
- Đoạn code = một cảnh (khớp với render từng cảnh); mỗi shot trong cảnh có trạng thái riêng.
- Khi shot lỗi: autofix → AI sửa một lượt → vẫn lỗi thì shot hiện **lỗi kèm thông báo dễ hiểu** (shot nào, lỗi gì). Creator chọn: chạy lại shot, làm lại shot kèm ghi chú, sửa tay, hoặc dán AI ngoài.
- 💡 **Xem nhanh từng shot**: bấm một shot để xem bản render nhanh 480p của riêng shot đó, ngay ở bước Code. Thấy sai chuyển động thì sửa luôn, không phải đợi render cả cảnh.
- Sửa code một shot chỉ làm cảnh đó phải render lại.

## Question P-8

A) Đồng ý, có xem nhanh từng shot (AI đề xuất).

B) Đồng ý, nhưng không cần xem nhanh từng shot.

X) Other (please describe after [Answer]: tag below)

[Answer]: khi shot lỗi AI không tự sửa mà hiện lỗi kèm thông báo cụ thể để creator chạy lại hoặc chạy lại kèm ghi chú, sửa tay hoặc dán AI ngoài, ý tưởng xem nhanh từng shot và render shot chỉ phải render cảnh rất oke

---

## Bước 11 — Validate

> Creator: "nếu có chỗ validate không ổn hoặc lỗi thì có cơ chế nào để sửa lại đúng chỗ đang bị lỗi không?"

**Hiện nay:** Validate phân tích code và chạy thử. Cảnh báo bố cục không chặn (CR-062), lỗi code máy móc được tự sửa (CR-068).

**Đề xuất:** mỗi phát hiện ghi rõ **shot hoặc mục nào, luật nào, sai thế nào, gợi ý sửa**, kèm các nút:
- **Mở chỗ đó**: nhảy tới đúng shot ở bước Code, hoặc đúng mục ở bước Mỹ thuật.
- **AI sửa chỗ này**: AI làm lại đúng shot đó, lấy phát hiện làm ghi chú.
- **Bỏ qua cảnh báo**: chỉ áp cho cảnh báo, có ghi lại để hiện ở Review. Lỗi thì phải sửa.

Sửa xong, chỉ shot hoặc cảnh đó được kiểm lại, không kiểm lại cả video.

---

## Bước 12 — Review: thông tin bổ sung

> Creator: "ok rồi nhưng nếu có thêm thông tin nào bạn nghĩ cần thiết thì cứ bổ sung nhé."

AI đề xuất bổ sung:
1. **Các chỗ đã duyệt**: bước nào, duyệt lúc nào, bao nhiêu mục phải làm lại. 
2. **Cảnh báo đã bỏ qua** ở Validate, bấm vào xem được.
3. **Thời lượng từng phần** so với khung của dạng video (ví dụ hook dài quá 20 giây).
4. **Ước tính render**: thời gian, dung lượng file, số cảnh chạy song song.
5. **Chi phí**: token tới giờ theo từng bước, và ước tính phần còn lại.
6. **Tỉ lệ duyệt ngay lần đầu** của video này (NFR-2), để biết prompt nào cần tối ưu.
7. **Danh sách hình mới** sẽ vào thư viện kênh, và nhân vật sẽ lưu vào thư viện nhân vật.

---

## Bước 13 — Render từng cảnh

> Creator: "chính xác là làm gì ở đoạn này, cũng không thấy cơ chế nếu lỗi thì sẽ xử lý như thế nào?"

**Bước này làm gì:** Remotion render code của từng cảnh thành một file video ở chất lượng đã chọn (ví dụ 1080p30). Trong lúc render:
- Màu được chỉnh theo color key của cảnh đó (câu 13 C).
- Chỗ nối liền mạch được render thêm một đoạn chồng ngắn ở đầu hoặc cuối để Merge ghép khớp.
- Nhiều cảnh chạy song song, giới hạn theo số nhân CPU của máy.
- Mỗi cảnh báo tiến độ theo số khung đã render trên tổng số khung.

Giọng, nhạc và ghép chưa làm ở bước này; đó là việc của Merge.

**Hiện nay khi lỗi:** cả bước render lỗi một lần (`rendering_failed`) và Creator chạy lại cả bước.

**Đề xuất khi lỗi:**
- Một cảnh lỗi **không dừng các cảnh khác**.
- Cảnh lỗi hiện thông báo dễ hiểu: shot nào, khung thứ mấy, lỗi gì (lỗi code lúc chạy, hết bộ nhớ, quá thời gian).
- Lỗi tạm thời (hết bộ nhớ, quá thời gian) được **tự chạy lại một lần**.
- Lỗi do code thì có nút **"Sửa shot lỗi"** mở đúng shot đó ở bước Code (với thông báo lỗi làm ghi chú cho AI). Sửa xong, chỉ cảnh đó render lại.
- Mỗi cảnh huỷ được và chạy lại được riêng.

## Question P-9

A) Đồng ý, có tự chạy lại một lần cho lỗi tạm thời (AI đề xuất).

B) Đồng ý, nhưng không tự chạy lại; lỗi nào cũng chờ Creator bấm.

X) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## Bước 14 — Dựng: làm lại cảnh theo góp ý

> Creator: "thấy có cho option làm lại cảnh nhưng nếu không cho creator góp ý thì làm sao biết chỉnh sửa theo hướng nào nhỉ, cần làm rõ."

**Đồng ý, nút "Làm lại cảnh" trơn là thiếu.** Đề xuất:
- **Ghi chú theo mốc thời gian**: Creator xem cảnh, dừng ở 0:12 và ghi "ong bò nhanh quá". Ghi chú tự gắn vào shot đang chạy ở mốc đó (shot 4.2). Ghi được nhiều chỗ trong một cảnh.
- Khi bấm "Làm lại cảnh theo ghi chú", AI đọc từng ghi chú và đề xuất **sửa ở tầng nào**: chỉ code (chuyển động, vị trí, tốc độ), hình (một hình minh hoạ), hay nội dung (storyboard của shot, câu thoại). Kèm danh sách sẽ làm lại gì. Creator đồng ý thì AI chạy; xong chỉ cảnh đó (và đoạn nối liền kề) render lại.
- Vẫn có "Chỉ render lại" (giữ nguyên code), dùng khi lỗi lúc render không do code.

## Question P-10

A) Ghi chú theo mốc thời gian, AI đề xuất tầng sửa, Creator đồng ý rồi mới chạy (AI đề xuất).

B) Ghi chú theo mốc thời gian, Creator tự chọn tầng sửa (code, hình, nội dung).

C) Một ô ghi chú cho cả cảnh, AI tự quyết và chạy luôn.

X) Other (please describe after [Answer]: tag below)

[Answer]: A

---

## Bước 15 — Merge + QC

> Creator: "QC ở đây cụ thể là làm việc gì / có thể hiển thị thanh tiến độ ở đây không."

**Merge làm gì:** ghép các cảnh (khớp đoạn chồng ở chỗ nối liền mạch, áp chuyển cảnh ở chỗ cắt), đặt giọng từng câu đúng mốc, trộn nhạc nền có hạ nhạc khi có giọng, chuẩn hoá âm lượng −14 LUFS, xuất file.

**QC hiện có** (`services/video-assembly/domain/qc_rules.py`), chạy sau khi ghép:

| Kiểm | Ý nghĩa | Mức |
|---|---|---|
| Hình đứng quá lâu | khoảng giữa hai câu dài quá ngưỡng | cảnh báo |
| Giọng chồng nhau | hai câu nói đè lên nhau | chặn |
| Phụ đề chồng nhau | hai dòng phụ đề hiện cùng lúc | cảnh báo |
| Âm lượng | độ to so với mục tiêu −14 LUFS | cảnh báo |
| Vỡ tiếng | đỉnh âm vượt ngưỡng | cảnh báo |
| Thuộc tính file | độ phân giải, số hình/giây, định dạng, phát nhanh trên web | cảnh báo |

Việc lỗi "chặn" có thật sự chặn đăng hay không do cờ `QC_ENFORCE` của orchestrator, hiện đang tắt. AI đề xuất thêm hai kiểm cho luồng mới: **chỗ nối khớp** (khung cuối cảnh trước và khung đầu cảnh sau không nhảy) và **không có khung đen hay khung trống**.

**Thanh tiến độ: có.** video-assembly đã gửi tiến độ ffmpeg trong lúc chạy. Màn này sẽ hiện thanh tiến độ theo từng chặng: ghép cảnh → giọng → nhạc → âm lượng → QC. Prototype đang vẽ thành danh sách tĩnh, bản sau sẽ sửa.

## Question P-11 — Bật chặn của QC?

A) Bật chặn với lỗi "chặn" (giọng chồng nhau, chỗ nối không khớp): phải sửa mới đăng được.

B) Giữ như hiện tại: chỉ cảnh báo, Creator quyết.

X) Other (please describe after [Answer]: tag below)

[Answer]: B nhưng bỏ sung thêm chỗ nối khớp (khung cuối cảnh trước và khung đầu cảnh sau không nhảy) và không có khung đen hay khung trống.

---

## Bước 17 — Publish: thumbnail

> Creator: "có thể render thumbnail bằng Remotion luôn không hay nên dùng AI render?"

**Hiện nay:** AI viết một prompt để Creator mang sang công cụ tạo ảnh bên ngoài (Midjourney, DALL-E…) (`thumbnail_design.txt`), rồi Creator tải ảnh thumbnail lên.

| | Render bằng Remotion | AI tạo ảnh |
|---|---|---|
| Cách làm | Ghép từ chính hình của video (khung mẫu, nhân vật, bối cảnh) với chữ tiêu đề lớn, theo vài khuôn bố cục; `renderStill` đã có sẵn | Gửi prompt sang dịch vụ tạo ảnh |
| Ưu | Cùng ngôn ngữ hình với video; chữ tiếng Việt chuẩn và sửa được; không tốn phí; làm nhiều phương án để thử | Ảnh có thể "bắt mắt" hơn, chi tiết hơn |
| Nhược | Bị giới hạn bởi bộ hình của video | Lệch phong cách với video; chữ tiếng Việt hay sai dấu; tốn phí; thêm dịch vụ ngoài |

## Question P-12

A) Remotion làm thumbnail mặc định: 3 phương án từ hình của video, Creator sửa chữ, chọn một. Vẫn giữ cách cũ (prompt cho AI ngoài, tải ảnh lên) làm phương án phụ (AI đề xuất).

B) Chỉ Remotion.

C) Gọi dịch vụ AI tạo ảnh trực tiếp từ hệ thống.

X) Other (please describe after [Answer]: tag below)

[Answer]: A nhưng nếu dùng AI ngoài tả lênh thì vẫn phải đảm bảo promt sinh ra tuân thủ style của video

---

## Question P-13 — Autofix máy móc khi code lỗi (tiếp P-8)

> Creator (P-8): "khi shot lỗi AI không tự sửa mà hiện lỗi kèm thông báo cụ thể để creator chạy lại hoặc chạy lại kèm ghi chú, sửa tay hoặc dán AI ngoài…"

Hiện có hai cơ chế tự sửa khi code lỗi:
1. **Lượt AI tự sửa** (repair, CR-063): gửi lỗi cho model để viết lại shot. Theo P-8, cơ chế này **bỏ**.
2. **Autofix máy móc** (CR-068): không phải AI. Là các luật cố định, sửa những lỗi gõ quen thuộc của code AI viết, ví dụ tên màu `PALETTE` viết sai, thiếu import. Nhanh, không tốn token, kết quả lúc nào cũng như nhau.

A) Giữ autofix máy móc, chỉ bỏ lượt AI tự sửa. Shot nào autofix có sửa thì vẫn ghi rõ đã sửa gì, để Creator thấy (AI đề xuất).
   - ✅ Ưu: Creator không phải bấm cho những lỗi gõ vặt; không tốn token.
   - ⚠️ Nhược: vẫn có một bước tự sửa code, dù có ghi lại.

B) Bỏ cả hai: mọi lỗi đều hiện cho Creator.
   - ✅ Ưu: không có gì tự đổi code.
   - ⚠️ Nhược: nhiều lỗi vặt phải bấm "chạy lại" hoặc sửa tay.

X) Other (please describe after [Answer]: tag below)

[Answer]: A — AI hiểu từ "ok rồi đó" (2026-10-13), sau tin nhắn AI đề xuất A; Creator chưa ghi riêng câu này, đã báo lại để sửa nếu hiểu sai.

---

## Sau khi chốt

AI sẽ:
1. Cập nhật `cr-072-2d-film-pipeline.md`, dự kiến thêm các yêu cầu:
   - FR-17: sửa đúng chỗ, chỉ làm lại phần phụ thuộc.
   - FR-18: ngôn ngữ hình của kênh trong cài đặt.
   - FR-19: gợi ý kiểu video.
   - FR-20: sửa cách đọc, từ điển phát âm.
   - FR-21: thumbnail.
   - Sửa FR-14 (lỗi khi render từng cảnh), và thêm phần Validate, Dựng, QC theo câu trả lời.
2. Cập nhật prototype theo các điểm trên.
3. Trình lại tài liệu yêu cầu để Creator duyệt.
