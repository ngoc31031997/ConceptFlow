# CR-072 — Pipeline tạo video theo quy trình làm phim hoạt hình 2D

Trạng thái AI-DLC: **INCEPTION — Requirements Analysis đã duyệt (Creator, 2026-10-13: "ok rồi đó")**; User Stories bỏ qua; bước tiếp theo là Workflow Planning. Câu hỏi và câu trả lời: `cr-072-requirement-questions.md` (vòng 1), `cr-072-clarification-questions.md` (vòng 2), `cr-072-prototype-review.md` (sau khi xem prototype, P-1..P-13). Bước sau: Workflow Planning, chia CR con.

## Ý tưởng gốc (nguyên văn)

> "đưa toàn bộ phàn này nào nhánh làm phim 2d nhé"

> Có. Quy trình làm phim hoạt hình 2D thường chia làm 3 giai đoạn lớn:
>
> 1. Tiền kỳ (Pre-production)
> Ý tưởng và kịch bản: viết logline, tóm tắt, rồi kịch bản chi tiết gồm lời thoại và mô tả hành động.
> Thiết kế nhân vật (character design): vẽ model sheet (turnaround trước, nghiêng, sau), bảng biểu cảm, bảng tư thế tay và khẩu hình (mouth chart) để lip-sync.
> Thiết kế bối cảnh và đạo cụ: gồm background layout và props.
> Color script và style guide: quy định bảng màu theo từng cảnh, ánh sáng, mood và quy tắc phong cách để cả đội vẽ đồng nhất.
> Storyboard: vẽ phác từng shot với góc máy, chuyển động camera, ghi chú thoại và thời lượng.
> Thu âm sơ bộ (scratch voice): thường thu thoại trước khi làm hoạt hình để có timing.
> Animatic: ghép storyboard với thoại và nhạc tạm thành video thô để chốt nhịp phim và thời lượng từng shot.
> 2. Sản xuất (Production)
> Layout: dựng bố cục chính xác cho từng shot, gồm vị trí nhân vật, camera và các lớp cảnh.
> Key animation: họa sĩ chính vẽ các khung hình chủ chốt (key pose, extreme).
> In-between (tweening): vẽ các khung trung gian giữa các key để chuyển động mượt. Nhịp phổ biến là "on 2s" (12 hình/giây) hoặc "on 1s" (24 hình/giây).
> Clean-up: đi lại nét sạch, đúng model.
> Tô màu (ink & paint): tô theo bảng màu của nhân vật.
> Vẽ background: vẽ hoàn chỉnh nền cảnh.
> Cách làm thay thế là rigging (cut-out animation): dựng nhân vật thành các bộ phận có khớp (tay, đầu, miệng…) trong Toon Boom Harmony, Spine, Moho hoặc After Effects (Duik), rồi diễn hoạt bằng cách xoay và di chuyển bộ phận thay vì vẽ lại từng frame. Cách này rẻ và nhanh, hay dùng cho video explainer và phim TV.
> 3. Hậu kỳ (Post-production)
> Compositing: ghép nhân vật, background, hiệu ứng (khói, ánh sáng, bóng) và chuyển động camera.
> VFX 2D: thêm lửa, nước, tia sáng và các hiệu ứng tương tự.
> Dựng phim (editing): chốt cắt cảnh và chuyển cảnh.
> Âm thanh: thu thoại chính thức, lip-sync, thêm SFX (foley), nhạc nền, rồi mix.
> Chỉnh màu và xuất bản: color grading cuối, render và xuất theo định dạng phát hành.

Các chỉ dẫn trước đó của Creator về ý tưởng này (2026-10-12):

> "tôi đi qua pipiline chi tiết như như làm phim 2d gồm cá bước […] làm 1 cr để đi qua hết các quy trinh này"

> "pipeline nhiều lắm mà […] sao plan ít vậy?"

> "rồi từ plan này kiểm tra lại xem có bước nào không cần với ý tưởng của chúng ta không?"

> "bỏ luôn manim. chỉ còn remotion"

> "bỏ hết chỉ giữ lại cr bỏ manim, các CR còn lại cần đi qua AI-DLC rule từ lên ý tưởng cho nên implement"

## Phân tích yêu cầu

- **Yêu cầu của Creator**: tổ chức lại luồng làm video theo quy trình làm phim hoạt hình 2D (tiền kỳ, sản xuất, hậu kỳ); Creator thấy và chốt sớm trước khi tốn thời gian render; video trông giống phim hoạt hình hơn.
- **Loại**: Enhancement lớn kèm tính năng mới (bước mới, dữ liệu mới, luật hình mới), có xoá dữ liệu cũ.
- **Phạm vi**: toàn hệ thống: orchestrator, authoring-service, llm-service, rendering, tts, video-assembly, api-gateway, web-gui, cùng DB và `/shared`.
- **Độ phức tạp**: Complex. Độ sâu tài liệu: Comprehensive. Sẽ chia thành nhiều CR con ở Workflow Planning (câu 16: AI đề xuất thứ tự).

## Mục tiêu sản phẩm (câu 1, 2)

Một người làm, hệ thống làm phần lớn, Creator xem và duyệt ở các mốc. Video chuyên nghiệp về nhiều chủ đề và kiến thức, không chỉ video giải thích, **trông như một phim hoạt hình ngắn**. Đăng YouTube, video dài và Shorts như hiện nay. Một người kể (câu C6). Chỉ dùng Remotion (CR-071, ADR-0033).

## Luồng mục tiêu

Thanh bước chia ba giai đoạn (câu 15 B). Tên và số bước chốt ở Application Design; thứ tự dưới đây là yêu cầu.

| Giai đoạn | Bước | Chỗ Creator duyệt |
|---|---|---|
| Tiền kỳ | Khởi tạo → Cấu hình | — |
| | Kịch bản: logline → tóm tắt → dàn ý → kịch bản | duyệt logline và tóm tắt; duyệt kịch bản như hiện nay |
| | Thiết kế mỹ thuật lượt 1: thế giới, bối cảnh, nhân vật, đạo cụ | một màn, duyệt hoặc làm lại từng mục |
| | Storyboard, có hành trình máy quay giữa các cảnh | như hiện nay |
| | Thiết kế mỹ thuật lượt 2: color key, khung mẫu | một màn, duyệt hoặc làm lại từng mục |
| | Giọng sơ bộ (TTS chạy sớm) | — |
| | Animatic khung tĩnh | chốt nhịp phim |
| Sản xuất | Hình minh hoạ → Code → Validate → Review | Review: bắt đầu render |
| Hậu kỳ | Render từng cảnh (có chỉnh màu theo cảnh) → Dựng → Merge + QC → Kết quả → Publish | Dựng: làm lại một cảnh, đổi chuyển cảnh |

## Yêu cầu chức năng

**Tiền kỳ**

- **FR-1. Luồng ba giai đoạn** (câu 1 D, 15 B). Thanh bước, tiêu đề màn, trạng thái chạy và số bước theo luồng mục tiêu ở cả server (orchestrator, authoring-service) và web-gui. Mỗi bước mới có trạng thái chạy, lỗi, huỷ, chạy lại như bước hiện có.
- **FR-19. Gợi ý kiểu video** (P-2 B). Khi Creator nhập chủ đề, AI gợi ý 3–5 kiểu video hợp nhất trong bảng Kiểu video, mỗi kiểu có một câu "vì sao hợp" và một câu mở đầu mẫu. Nếu không kiểu nào thật sự hợp, AI đề xuất một kiểu mới kèm playbook nháp; Creator dùng thử cho video này và lưu vào bảng được. Vẫn giữ "Để AI tự chọn" và "Xem tất cả kiểu".
- **FR-2. Logline và tóm tắt** (câu 3 B). AI viết logline (một câu) và tóm tắt (một đoạn) trước dàn ý; Creator duyệt hoặc cho làm lại kèm ghi chú; dàn ý viết theo logline đã duyệt. Đổi logline thì dàn ý viết lại. Trường "Kiểm chứng" của từng beat giữ như hiện nay (câu 20 A). Creator có thể tự viết logline và tóm tắt (ô có chữ thì AI dùng chữ của Creator, ô trống thì AI viết) hoặc sửa tay bản của AI. Dàn ý sửa tay hoặc làm lại từng beat kèm ghi chú; lời thoại duyệt, làm lại kèm ghi chú, sửa tay theo từng cảnh và từng câu; kịch bản được duyệt khi mọi cảnh đã duyệt (FR-17).
- **FR-3. Bộ thiết kế mỹ thuật** (câu 21 B, C5). Sáu mục, AI làm, mỗi mục là chữ hoặc ảnh tĩnh render bằng Remotion (`renderStill`):
  1. Mô tả thế giới: nơi chốn, thời đại, chất liệu, logic ánh sáng, bảng màu thế giới; kèm bảng ghép hình tham khảo.
  2. Thiết kế bối cảnh: mỗi nơi chốn chính một nền 4 lớp, ở khung ngang và dọc, kèm biến thể ánh sáng khi câu chuyện cần.
  3. Thiết kế nhân vật: model sheet trước/nghiêng/sau, biểu cảm, tư thế tay, nhân vật đặt trên từng nền chính.
  4. Thiết kế đạo cụ: các vật quan trọng của câu chuyện.
  5. Color key: mỗi phần của dàn ý một khung màu (FR-6).
  6. Khung mẫu: 3–5 khung hoàn chỉnh của các khoảnh khắc chính, đúng chất lượng video cuối.

  Duyệt hai lượt (C5 B, Creator xác nhận 2026-10-13): mục 1–4 sau kịch bản, mục 5–6 sau storyboard. Mỗi mục có "duyệt" hoặc "làm lại kèm ghi chú". Các bước sau (storyboard, hình minh hoạ, code) bám theo bộ đã duyệt.

  Bảng tham khảo của mục 1 không vào video: mỗi ô bỏ, thay hoặc làm lại kèm ghi chú được; Creator tải ảnh tham khảo của mình lên được (P-5 A): model vẽ đọc được ảnh thì dùng làm gợi ý, không thì ảnh chỉ để Creator đối chiếu (kiểm ở thiết kế). Mỗi hình của mục 2–4 sửa được bằng một trong bốn cách: làm lại kèm ghi chú, sửa mô tả rồi vẽ lại, tải SVG lên, thay bằng hình trong thư viện. Sửa mục gốc thì mọi shot dùng nó đổi theo, màn hình báo trước các shot bị ảnh hưởng. Mỗi color key và mỗi khung mẫu là một mục riêng; color key sửa tay được bằng bộ chọn màu (độ ấm, độ sáng, màu nhấn, nguồn sáng), AI kiểm lại tương phản ngay sau khi đổi.
- **FR-4. Nhân vật cấp video và thư viện nhân vật kênh** (câu 4 C). Nhân vật khai báo một lần cho cả video. Shot chỉ tham chiếu nhân vật kèm biểu cảm và tư thế, không đặt lại dáng, tuổi, áo. Nhân vật đã duyệt lưu được vào thư viện kênh và dùng lại ở video sau. Không nhép miệng: miệng giữ hình cố định theo biểu cảm (câu 5 A).
- **FR-5. Phong cách theo chủ đề** (câu 19, C3 B). Mỗi video có một thế giới theo chủ đề (lòng quả sung, vũ trụ, lăng mộ Ai Cập, rừng, đáy biển…): bối cảnh, bảng màu, ánh sáng, chất liệu bề mặt. Giữ chung giữa các video là **ngôn ngữ hình đang bật** của kênh (FR-18): cách dựng khối, nét, cách vẽ nhân vật. Cho phép lớp chất liệu theo chủ đề ở nền và đạo cụ (vân giấy cói, glow vũ trụ, vệt sáng gợn nước…). Ánh sáng theo nguồn sáng của thế giới, trừ khi ngôn ngữ hình quy định khác. Luật hình và luật nền viết lại theo yêu cầu này (FR-15).
- **FR-6. Hành trình màu** (câu 6, C1 A). Bảng màu thế giới lấy từ chủ đề. Mỗi phần của dàn ý có một color key (ấm/lạnh, độ sáng, màu nhấn, nguồn sáng). Được một khoảnh khắc phá bảng màu ở cao trào. Ưu tiên tạo khác biệt bằng ánh sáng. Kiểm tra tự động: chữ đủ tương phản; hai cảnh liền nhau không quá giống và không lệch khỏi bảng màu thế giới. Creator xem cả dải màu của video trên một hàng.
- **FR-7. Cảnh giải thích ý đặt trong thế giới** (C4 A, Creator xác nhận 2026-10-13). Con số, sơ đồ, so sánh là vật trong cảnh của thế giới, không chuyển sang nền màu đặc. Thay FR7 "nền sáng bão hoà mặc định" của CR-067; cần một ADR.
- **FR-8. Hành trình máy quay liền mạch** (câu 22 B). Storyboard ghi mỗi cảnh nối sang cảnh sau bằng gì (zoom xuyên vào vật, kéo ra tỉ lệ lớn hơn, vật biến hình, máy quay lia tiếp) và vật nối dùng chung; code hai cảnh dùng chung vật nối đó.
- **FR-9. Giọng sơ bộ chạy sớm** (câu 7 B). Khi storyboard và lời thoại đã có, TTS chạy bằng giọng chính thức; thời lượng shot lấy từ giọng thật. Câu không đổi thì bước TTS sau dùng lại file đã có; câu sửa thì đọc lại câu đó. Một người kể. Sửa giọng theo FR-20.
- **FR-20. Sửa giọng từng câu và từ điển phát âm** (P-6 A). Mỗi câu ở bước Giọng sơ bộ sửa được theo hai cách: sửa chữ câu thoại (đổi cả phụ đề), hoặc giữ chữ và sửa cách đọc (ô "đọc là", tốc độ, khoảng nghỉ trước/sau câu; phụ đề vẫn hiện chữ gốc; làm bằng SSML của Azure). "Đọc lại" chỉ đọc lại câu đó, animatic của shot tự cập nhật. Từ điển phát âm dùng chung cho kênh: một từ sửa cách đọc một lần, mọi video sau đọc đúng.
- **FR-10. Animatic khung tĩnh** (câu 8 C). Trước Hình minh hoạ và Code: mỗi shot một khung tĩnh dựng từ bộ thiết kế mỹ thuật (nền, nhân vật, mô tả shot), giữ đúng thời lượng câu, ghép với giọng sơ bộ và phụ đề thành video thô. Creator chốt nhịp. Sửa trên từng shot (P-7 B): đổi thời lượng (giữ hình lâu hơn hoặc ngắn hơn, thêm khoảng nghỉ trước/sau câu); sửa câu thoại ngay tại đây (đồng bộ ngược về storyboard, đọc lại câu đó, dựng lại animatic của shot đó); đổi khung tĩnh kèm ghi chú. Thêm ghi chú nhịp cho cả cảnh ("cảnh 3 chậm quá"): AI đề xuất cắt hoặc gộp câu, Creator xem rồi mới áp dụng. Không có nút "làm lại" chung cho cả animatic.

**Sản xuất**

- **FR-11. Diễn hoạt theo key pose** (câu 9 C). Rig nhân vật diễn theo tư thế chính có giữ dáng, ease in/out mạnh, vượt đà, thay vì trôi đều. Không làm kiểu "on 2s".
- **FR-12. Bộ VFX viết bằng Remotion** (câu 10 B). Các hiệu ứng có tên (hạt, khói, tia sáng, lấp lánh, mưa, glow…), AI gọi theo tên, màu lấy từ bảng màu của video.
- **FR-13. Hình minh hoạ theo bộ thiết kế mỹ thuật**. Bước Hình minh hoạ vẽ phần còn thiếu cho các shot, theo thế giới, bối cảnh, nhân vật và đạo cụ đã duyệt. Thư viện bắt đầu trống (FR-16). Giữ như hiện nay: vẽ song song, mỗi hình duyệt, sửa, bỏ, sửa prompt; thêm "làm lại kèm ghi chú". Hình đến từ bộ thiết kế mỹ thuật hiện là đã duyệt; muốn sửa thì sửa ở mục gốc.
- **FR-22. Code theo cảnh, lỗi theo shot** (P-8). Đoạn code là một cảnh, mỗi shot có trạng thái riêng. Khi shot lỗi, AI **không tự sửa**: shot hiện lỗi kèm thông báo cụ thể (shot nào, lỗi gì), Creator chọn chạy lại, chạy lại kèm ghi chú, sửa tay hoặc dán kết quả AI ngoài. Autofix máy móc (không phải AI) của CR-068 giữ lại, và shot nào được autofix sửa thì ghi rõ đã sửa gì (P-13 A). Xem nhanh từng shot: bấm một shot để render nhanh 480p riêng shot đó ngay ở bước Code. Sửa code một shot chỉ làm cảnh chứa nó phải render lại.
- **FR-23. Validate đúng chỗ**. Mỗi phát hiện ghi rõ shot hoặc mục nào, luật nào, sai thế nào, gợi ý sửa; nút "Mở chỗ đó" (nhảy tới đúng shot hoặc mục), "AI sửa chỗ này" (AI chỉ chạy khi Creator bấm, lấy phát hiện làm ghi chú), "Bỏ qua cảnh báo" (chỉ với cảnh báo, ghi lại để hiện ở Review). Sửa xong chỉ kiểm lại shot hoặc cảnh đó.
- **FR-24. Review đủ thông tin**. Ngoài tóm tắt video: các chỗ đã duyệt (lúc nào, bao nhiêu mục phải làm lại), cảnh báo đã bỏ qua, thời lượng từng phần so với khung của dạng video, ước tính thời gian và dung lượng render, chi phí token theo bước và ước tính phần còn lại, tỉ lệ duyệt ngay lần đầu, danh sách hình và nhân vật sẽ vào thư viện kênh.

**Hậu kỳ**

- **FR-14. Render từng cảnh và dựng** (câu 11 B). Mỗi cảnh render thành một file riêng, các cảnh render song song. Chỗ nối liền mạch (FR-8) render thêm một đoạn chồng ở đầu và cuối để Merge ghép khớp. Mỗi cảnh báo tiến độ theo số khung đã render. Một cảnh lỗi không dừng các cảnh khác; cảnh lỗi hiện thông báo dễ hiểu (shot nào, khung thứ mấy, lỗi gì); lỗi tạm thời (hết bộ nhớ, quá thời gian) tự chạy lại một lần (P-9 A); lỗi do code có nút "Sửa shot lỗi" mở đúng shot ở bước Code; mỗi cảnh huỷ và chạy lại được riêng. Màn Dựng theo FR-25. Chỉnh màu theo cảnh làm trong Remotion, theo color key (câu 13 C). Đổi hợp đồng file `docs/contracts/shared-artifacts.md`; cần một ADR.
- **FR-25. Dựng theo ghi chú mốc thời gian** (P-10 A). Creator xem cảnh, dừng ở một mốc và ghi chú; ghi chú gắn vào shot đang chạy ở mốc đó, một cảnh ghi được nhiều chỗ. "Làm lại cảnh theo ghi chú": AI đề xuất sửa ở tầng nào cho từng ghi chú (code, hình, hay nội dung storyboard/câu thoại) kèm danh sách sẽ làm lại; Creator đồng ý thì AI chạy; xong chỉ cảnh đó và đoạn nối liền kề render lại. Có "Chỉ render lại" (giữ code). Chỗ cắt đổi chuyển cảnh ngay, không render lại.
- **FR-26. Merge có tiến độ, QC chỉ cảnh báo** (P-11 B). Thanh tiến độ theo chặng: ghép cảnh → giọng → nhạc → âm lượng → QC (video-assembly đã gửi tiến độ ffmpeg). QC giữ 6 kiểm hiện có (hình đứng quá lâu, giọng chồng nhau, phụ đề chồng nhau, âm lượng, vỡ tiếng, thuộc tính file), thêm **chỗ nối khớp** (khung cuối cảnh trước và khung đầu cảnh sau không nhảy) và **không có khung đen hay khung trống**. Mọi kết quả QC chỉ là cảnh báo, Creator quyết; `QC_ENFORCE` giữ tắt.
- **FR-21. Thumbnail bằng Remotion** (P-12 A). Mặc định Remotion dựng 3 phương án từ hình của chính video (khung mẫu, nhân vật, bối cảnh) với chữ tiêu đề lớn theo vài khuôn bố cục, 1280×720; Creator sửa chữ và chọn một. Phương án phụ: AI viết prompt cho công cụ tạo ảnh bên ngoài rồi Creator tải ảnh lên, như hiện nay; prompt này phải sinh từ ngôn ngữ hình, thế giới và bảng màu của video để ảnh cùng phong cách.

**Sửa và chạy lại (áp cho mọi bước)**

- **FR-17. Sửa đúng chỗ, chỉ làm lại phần phụ thuộc** (P-1 A). Mọi kết quả của AI sửa được ở đơn vị nhỏ nhất (một mục, một cảnh, một shot, một câu) bằng sửa tay, làm lại kèm ghi chú, hoặc gom ghi chú trên nhiều đơn vị rồi cho AI sửa một lượt. AI không ghi đè phần Creator tự viết, trừ khi Creator bấm làm lại. Khi làm lại một đơn vị, AI được xem các đơn vị bên cạnh để giữ mạch. Sửa một đơn vị chỉ làm lại phần phụ thuộc vào nó (ví dụ sửa cảnh 3 thì chỉ giọng, animatic, hình, code, render của cảnh 3), và màn hình báo trước sẽ làm lại những gì. Một bước được duyệt khi mọi đơn vị đã duyệt; có "Duyệt tất cả mục còn lại". Bảng đơn vị sửa của từng bước: `cr-072-prototype-review.md` mục 0.

**Dữ liệu, prompt, luật**

- **FR-18. Ngôn ngữ hình của kênh** (P-3, P-4 A). Trang cài đặt "Ngôn ngữ hình", cách quản lý như bảng Kiểu video. Mỗi ngôn ngữ gồm tên, mô tả, luật hình, luật nền, cách vẽ người và con vật (rig cơ sở), và bộ ảnh mẫu render cùng một bộ cảnh thử để so công bằng. Chỉ một ngôn ngữ được bật tại một thời điểm; video mới dùng ngôn ngữ đang bật; video đã tạo giữ ngôn ngữ và phiên bản đã dùng. Ngôn ngữ hệ thống chỉ đọc, Creator sao chép ra để sửa. Bốn ngôn ngữ hệ thống mặc định (P-3): **Sách tranh tươi** (tham khảo Vẽ Chuyện), **Khoa học phát sáng** (tham khảo Kurzgesagt), **Giấy cắt lớp**, **Đường nét tối giản**. Tên và luật không nhắc tên kênh, phần "tham khảo" chỉ là ghi chú cho Creator. Trước CR con đầu tiên, AI dựng bộ ảnh mẫu cho cả bốn, Creator so rồi bật một (P-4 A).

- **FR-15. Prompt và luật viết lại từ đầu** (C7). Prompt hệ thống của mọi vai (Đạo diễn, Kỹ sư, người vẽ, các bước mới) viết lại cho luồng mới theo FR-2 đến FR-26. Luật hình và luật nền viết riêng cho từng ngôn ngữ hình hệ thống (FR-18); prompt đọc luật của ngôn ngữ đang bật.
- **FR-16. Xoá dữ liệu cũ** (câu 18, C7). Xoá mọi project và file của chúng trong `/shared`, dữ liệu authoring theo project, prompt (cả bản Creator sửa), luật style, thư viện hình và nền (Creator đã sao lưu thư viện ra zip). Giữ kiểu video (archetype), cấu hình kênh và kết nối YouTube. Trước khi xoá: dump DB và nén `/shared` vào một file sao lưu. Thời điểm: khi CR con đầu tiên đổi dạng dữ liệu vào main. Video đã đăng YouTube không bị đụng tới.

## Yêu cầu phi chức năng

- **NFR-1. Chất lượng trước, chi phí sau** (câu 17 C). Chấp nhận tăng token và thời gian render. Mỗi bước vẫn ghi token và thời gian như hiện nay để Creator thấy chi phí.
- **NFR-2. Ít phải sửa** (câu 3, 14 B). Trước khi đưa Creator duyệt, AI tự kiểm kết quả theo luật. Với lỗi code, hệ thống không cho AI tự sửa; lỗi hiện cho Creator quyết (FR-22). Mỗi chỗ duyệt có "làm lại kèm ghi chú" cho từng mục thay vì sửa tay. Ghi tỉ lệ được duyệt ngay lần đầu cho từng loại mục, để đo và tối ưu prompt.
- **NFR-3. Không nuốt lỗi**. Bước mới lỗi thì báo rõ và chạy lại được; không có đường "giả thành công".
- **NFR-4. Hai khung hình**. Bộ thiết kế mỹ thuật, animatic và video render đúng ở 16:9 và 9:16.
- **NFR-5. Giao diện** theo `docs/ux-ui-design-rules.md` (thứ tự trên-xuống, trái-sang-phải; Neubrutalism; trượt ra/vào).
- **NFR-6. Code** theo `docs/code-standards-rules.md`.
- **NFR-7. Ngôn ngữ nội dung** tiếng Việt và tiếng Anh như hiện nay.

## Không làm trong CR này

| Việc | Lý do | Ghi vào |
|---|---|---|
| Nhép miệng (lip-sync, mouth chart) | Creator bỏ (câu 5) | — |
| Kiểu chuyển động "on 2s" | Chọn key pose (câu 9 C) | — |
| Nghiên cứu có nguồn thật | Giữ trường "Kiểm chứng" (câu 20 A) | — |
| SFX và âm nền theo thế giới | Creator đưa vào backlog (câu 12, C2); nguồn đã tìm: Kenney, Freesound CC0, Sonniss GDC, YouTube Audio Library | backlog `aidlc-state.md` |
| Nhạc theo mood hoặc nhạc AI riêng mỗi video | Creator đưa vào backlog (câu 23) | backlog `aidlc-state.md` |
| Nhân vật có giọng riêng | Tạm thời một người kể (C6) | backlog `aidlc-state.md` |
| Lượt AI tự sửa code khi shot lỗi (repair của CR-063); autofix máy móc của CR-068 vẫn giữ (P-13 A) | Creator muốn thấy lỗi và tự chọn cách sửa (P-8) | — |
| QC chặn đăng | Giữ chỉ cảnh báo (P-11 B) | — |

## Tiêu chí chấp nhận (cấp CR, chi tiết ở từng CR con)

1. Một project mới đi hết luồng mục tiêu tới video đăng được, Creator duyệt ở đúng các chỗ trong bảng luồng.
2. Hai video khác chủ đề (ví dụ vũ trụ và lòng quả sung) cho hai thế giới khác hẳn (bối cảnh, màu, ánh sáng, chất liệu), nhưng nhìn vẫn ra cùng một kênh.
3. Creator xem animatic có giọng trước khi bước Code chạy.
4. Một nhân vật giữ đúng model trên mọi shot của video.
5. Làm lại một cảnh ở màn Dựng không render lại cả video; chỗ nối liền mạch không bị giật.
6. Dải màu của video cho thấy color key đổi theo từng phần của dàn ý.
7. Sửa một cảnh ở storyboard chỉ làm lại giọng, animatic, hình, code và render của cảnh đó; các cảnh khác giữ nguyên.
8. Đổi ngôn ngữ hình đang bật không làm đổi video đã tạo; video mới dùng ngôn ngữ mới.
9. Shot lỗi ở bước Code hiện đúng shot và thông báo lỗi, không bị AI tự sửa; một cảnh lỗi khi render không dừng các cảnh khác.
10. Một từ sửa trong từ điển phát âm được đọc đúng ở video sau.

## ADR cần viết

- Luồng ba giai đoạn và bộ thiết kế mỹ thuật (FR-1, FR-3).
- Phong cách theo thế giới, thay FR7 nền sáng của CR-067 (FR-5, FR-7).
- Render theo cảnh có đoạn chồng; hợp đồng file mới (FR-14).
- Ngôn ngữ hình của kênh: dữ liệu, chỉ một bật, video giữ phiên bản đã dùng (FR-18).
- Bỏ lượt AI tự sửa code, lỗi hiện cho Creator (FR-22).

## Phụ lục A — Đối chiếu từng khâu ban đầu (2026-10-12)

Đối chiếu từng khâu với main `7df416a`, làm trong phiên 2026-10-12. Đây là đầu vào đã dùng để hỏi Creator; yêu cầu đã chốt nằm ở các mục trên.

Ký hiệu: **✅ có**, **◐ một phần**, **✗ chưa có**, **≠ hệ thống làm khác cách** (diễn hoạt bằng code Remotion kiểu rig, không vẽ tay từng khung).

| Khâu | Hiện tại | |
|---|---|---|
| Ý tưởng, kịch bản | Bước 1 Ý tưởng, bước 3 Kịch bản (dàn ý có câu hỏi cốt lõi, beat). Không có logline/tóm tắt tách riêng; câu thoại có `say`/`show` | ◐ |
| Thiết kế nhân vật | Rig `Person` có dáng, nét mặt, tuổi, áo, `talking` (`services/rendering/remotion_project/src/conceptflow-mini/illustration.tsx:258-279`), `rig.tsx`. Storyboard chỉ có `hero: str` (`services/llm-service/app/storyboard.py:142`); thuộc tính nhân vật đặt lại ở từng shot | ◐ |
| Bảng tay | `ReachingHand`: `open`, `point`, `grip`, `pinch`, `count` (CR-067) | ✅ |
| Mouth chart | Miệng chỉ có độ mở, nhép theo sóng sin (`illustration.tsx:347`) | bỏ (Creator) |
| Bối cảnh, đạo cụ | `Scene.setting` (`storyboard.py:131-133`), `backdrops.tsx`, bước 5 Hình minh hoạ | ✅ |
| Color script, style guide | `palette` một bộ cho cả video (`storyboard.py:144`), `mood` chữ tự do theo cảnh, luật style trong DB | ◐ |
| Storyboard | Bước 4: `Shot.camera`, `visual`, `lines[{say, show}]`; chỉ là chữ | ◐ |
| Thu âm sơ bộ | Mỗi câu một clip TTS, chạy sau Duyệt nội dung (bước 9); TTS dùng lại file đã có (`services/tts/application/synthesize_speech.py:38`) | ◐ |
| Animatic | Không có; lần đầu thấy hình + giọng là sau TTS, render và ghép. Có mức `480p15` | ✗ |
| Layout | `Storyboard.layout` (`storyboard.py:145-148`), luật layout tự động | ✅ |
| Key animation, in-between, clean-up, tô màu | Chuyển động nội suy bằng code; hình SVG theo palette | ≠ |
| Vẽ background | `Scene` nhiều lớp, parallax, nền kit và thư viện | ✅ |
| Rigging | `rig.tsx`, `Person`, `ReachingHand`; backlog avatar Mướp (CR-038) | ◐ |
| Compositing, camera | `Scene` nhiều lớp, `CameraMove` (`scene.tsx:26-32`) | ✅ |
| VFX 2D | Clip Lottie (`lottie.tsx`) | ◐ |
| Dựng phim | `Scene.transition_in`; dựng lại chỉ được cả video (`services/web-gui/src/pages/ResultPage.tsx:47`); rendering xuất một `rendered.mp4` cho cả video (`docs/contracts/shared-artifacts.md`) | ◐ |
| Thoại chính thức | Một giọng dẫn cho cả video; `Line` không có người nói | ◐ |
| Lip-sync | Sóng sin cố định | bỏ (Creator) |
| SFX | Không có; backlog CR-042 GĐ4 (`aidlc-docs/aidlc-state.md:115`) | ✗ |
| Nhạc nền, mix | Ducking sidechain, `loudnorm` −14 LUFS (`services/video-assembly/adapters/assembly/ffmpeg_assembler.py:53-68`) | ✅ |
| Chỉnh màu | Không có bước grading | ✗ |
| Xuất bản | 16:9, 9:16, nhiều mức chất lượng, đăng YouTube | ✅ |

Ý tưởng sản phẩm dùng khi xét khâu nào cần (cần Creator xác nhận ở Requirements Analysis): một người làm, AI làm phần lớn; video giải thích kiến thức đăng YouTube (dài và Shorts); một giọng dẫn, hình đổi theo từng câu thoại; flat vector (`README.md`, `aidlc-docs/inception/requirements/requirements.md` §1, CR-067).

## Phụ thuộc

- CR-071 (bỏ Manim) đã vào main (`7f327d0`); nhánh này đã lên commit đó.
- Ngày 2026-10-13 không còn nhánh CR hay fix nào chưa merge vào main, nên không có việc trùng.
- Backlog avatar Mướp (CR-038) gần với thư viện nhân vật kênh (FR-4); xét ở Application Design.

## Bước AI-DLC tiếp theo

1. **User Stories**: đề xuất bỏ qua. Creator là người dùng duy nhất, và các màn đã mô tả theo bước trong luồng mục tiêu.
2. **Workflow Planning**: chia CR con, thứ tự (câu 16: AI đề xuất), độ sâu từng phần, sơ đồ luồng.
3. **Application Design / Units Generation**: chốt tên và số bước, dữ liệu mới (logline, bộ thiết kế mỹ thuật, nhân vật, color key, hành trình máy quay), API, sự kiện; viết các ADR ở trên.
4. **Construction** cho từng CR con: thiết kế → task → code → build và test, mỗi CR con một nhánh.
