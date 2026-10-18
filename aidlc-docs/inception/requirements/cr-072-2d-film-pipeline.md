# CR-072 — Pipeline tạo video theo quy trình làm phim hoạt hình 2D

Trạng thái AI-DLC: **INCEPTION — Requirements Analysis đã duyệt (Creator, 2026-10-13: "ok rồi đó")**; User Stories bỏ qua; bước tiếp theo là Workflow Planning. Câu hỏi và câu trả lời: `cr-072-requirement-questions.md` (vòng 1), `cr-072-clarification-questions.md` (vòng 2), `cr-072-prototype-review.md` (sau khi xem prototype, P-1..P-13). Bước sau: Workflow Planning, chia CR con.

**Cập nhật 2026-10-17 — Tiền kỳ · Nội dung (Creator duyệt 2026-10-17: "bắt đầu code các CR cho tiền kì trước").** Creator thiết kế lại phần Tiền kỳ tới kịch bản thành sáu bước mới quanh bước Cấu hình (Ý tưởng · creative brief, Nghiên cứu và kiểm chứng, Xây dựng câu chuyện, Beat sheet, Viết kịch bản, Rà soát và chốt) và chốt giao diện bảy bước này trên mockup bản 8 (https://claude.ai/artifact/HibBJUisUocqKXDZD57cTH). Câu trả lời của Creator cùng ngày: chốt mockup bước 1–7 như bản 8; làm Tiền kỳ ngay sau U2a, khung sửa theo đơn vị dựng trong CR con đầu tiên của Tiền kỳ; chia sáu CR con; đọc giọng từng câu (`SynthesizeLines`) đưa lên bước Viết kịch bản. Những gì đổi: bảng luồng mục tiêu (21 bước, Tiền kỳ chia hai phần), FR-1, FR-2 (thay bằng FR-27..FR-32), FR-19 (chuyển sang bước Câu chuyện), FR-27..FR-32 mới, mục "Không làm trong CR này" (bỏ dòng "Nghiên cứu có nguồn thật"), tiêu chí chấp nhận 11–14, ADR-0042..0044.

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

Thanh bước chia ba giai đoạn (câu 15 B); Tiền kỳ chia hai phần, Nội dung và Hình ảnh, nên thanh bước có bốn nhóm (Creator, 2026-10-17; mockup bản 8). 21 bước; tên bước theo mockup. Thứ tự dưới đây là yêu cầu.

| Giai đoạn | # | Bước | Chỗ Creator duyệt |
|---|---|---|---|
| Tiền kỳ · Nội dung | 1 | Ý tưởng · creative brief (FR-27) | duyệt brief |
| | 2 | Cấu hình | — |
| | 3 | Nghiên cứu và kiểm chứng, tuỳ chọn (FR-28) | duyệt bảng dữ kiện, hoặc bỏ qua bước |
| | 4 | Xây dựng câu chuyện (FR-29) | duyệt câu chuyện |
| | 5 | Beat sheet (FR-30) | từng beat |
| | 6 | Viết kịch bản (FR-31) | lời thoại từng beat |
| | 7 | Rà soát và chốt kịch bản (FR-32) | xử lý từng mục rà soát, duyệt ghi chú bàn giao từng beat, chốt phiên bản |
| Tiền kỳ · Hình ảnh | 8 | Thiết kế mỹ thuật lượt 1: thế giới, bối cảnh, nhân vật, đạo cụ | một màn, duyệt hoặc làm lại từng mục |
| | 9 | Storyboard, có hành trình máy quay giữa các cảnh | từng cảnh |
| | 10 | Thiết kế mỹ thuật lượt 2: color key, khung mẫu | một màn, duyệt hoặc làm lại từng mục |
| | 11 | Giọng sơ bộ | — |
| | 12 | Animatic khung tĩnh | chốt nhịp phim |
| Sản xuất | 13–16 | Hình minh hoạ → Code → Validate → Review | Review: bắt đầu render |
| Hậu kỳ | 17–21 | Render từng cảnh (có chỉnh màu theo cảnh) → Dựng → Merge + QC → Kết quả → Publish | Dựng: làm lại một cảnh, đổi chuyển cảnh |

## Yêu cầu chức năng

**Tiền kỳ**

- **FR-1. Luồng ba giai đoạn** (câu 1 D, 15 B; cập nhật 2026-10-17). Thanh bước, tiêu đề màn, trạng thái chạy và số bước theo luồng mục tiêu (21 bước, bốn nhóm: Tiền kỳ · Nội dung, Tiền kỳ · Hình ảnh, Sản xuất, Hậu kỳ) ở cả server (orchestrator, authoring-service) và web-gui. Mỗi bước mới có trạng thái chạy, lỗi, huỷ, chạy lại như bước hiện có. Bước tuỳ chọn (Nghiên cứu) có nhãn "tuỳ chọn", đổi thành "bỏ qua" khi Creator bỏ qua.
- **FR-19. Gợi ý kiểu video** (P-2 B; cập nhật 2026-10-17). Gợi ý chạy ở bước Xây dựng câu chuyện (FR-29), không còn ở lúc nhập chủ đề: AI đọc brief và bảng dữ kiện, gợi ý 3–5 kiểu video hợp nhất trong bảng Kiểu video, mỗi kiểu có góc kể, một câu "vì sao hợp" có mã dữ kiện làm chứng, và một câu mở đầu mẫu. Nếu không kiểu nào thật sự hợp, AI đề xuất một kiểu mới kèm playbook nháp; Creator dùng thử cho video này và lưu vào bảng được. Vẫn giữ "Để AI tự chọn", "Xem tất cả kiểu", thêm "Gợi ý lại".
- **FR-2. Logline và tóm tắt** (câu 3 B). **Thay bằng FR-27..FR-32 (Creator, 2026-10-17).** Logline và tóm tắt thành hai ô của khung câu chuyện (FR-29); dàn ý thành beat sheet (FR-30); lời thoại theo beat (FR-31). Trường "Kiểm chứng" của beat (câu 20 A) thay bằng dữ kiện có mã và nguồn (FR-28). Các ý còn giữ: Creator tự viết hoặc sửa tay ô nào cũng được, ô Creator viết thì AI giữ nguyên trừ khi Creator bấm làm lại; mọi chỗ duyệt theo đơn vị (FR-17).

**Tiền kỳ · Nội dung** (Creator, 2026-10-17; giao diện theo mockup bản 8, bước 1–7)

Chung cho bảy bước: tài liệu của mỗi bước (brief, câu hỏi, dữ kiện, câu chuyện, beat, lời thoại, ghi chú bàn giao) là đơn vị lưu trong DB mà bước sau tham chiếu, không phải file sinh ra. Mỗi bước có một chỗ duyệt (gom và tối ưu sau). Mỗi đơn vị có sửa tay (trong hộp thoại), làm lại kèm ghi chú, và ô Creator viết được AI giữ nguyên (FR-17). Sửa một đơn vị thì chỉ phần phụ thuộc vào nó thành "cần làm lại", và màn hình báo trước những phần đó. Thao tác xoá, làm lại phần phía sau, hay chốt có hộp xác nhận kèm danh sách ảnh hưởng (`docs/ux-ui-design-rules.md` luật 4–6).

- **FR-27. Ý tưởng · creative brief** (bước 1). Creator nhập chủ đề hoặc câu hỏi ban đầu, dạng video (dài 16:9 hoặc Shorts 9:16), ngôn ngữ nội dung, thời lượng mục tiêu (3–4, 4–5, 5–7, 7–10 phút); ngôn ngữ hình đang bật được ghim cho video khi bấm "Tạo project và viết brief" (FR-18). AI viết brief từ đề tài: câu hỏi trung tâm cần trả lời, khán giả, sau video người xem sẽ biết / hiểu / cảm thấy gì, giọng điệu, ngoài phạm vi. Ô nào cũng sửa được; ô Creator đã sửa thì AI giữ nguyên khi làm lại. Đổi đề tài bằng "Làm lại kèm ghi chú" ở brief. Duyệt brief mới sang Cấu hình. Đổi brief đã duyệt thì câu hỏi nghiên cứu sinh lại (dữ kiện đã có giữ nguyên) và câu chuyện, beat, lời thoại đã có thành "cần làm lại".
- **FR-28. Nghiên cứu và kiểm chứng** (bước 3, tuỳ chọn).
  1. *Câu hỏi cần tra*: AI sinh từ brief, không cần duyệt; Creator sửa, xoá (có xác nhận), thêm; "Sinh lại từ brief" (xác nhận nếu sẽ thay câu Creator đã sửa).
  2. *Prompt nghiên cứu*: hệ thống dựng prompt từ brief và câu hỏi, cập nhật khi Creator sửa. Prompt có luật nguồn (chỉ dẫn nguồn đã mở và đọc; đoạn trích chép nguyên văn ngôn ngữ gốc; ưu tiên nguồn có bình duyệt, sách giáo khoa, bách khoa, viện, đại học; mỗi câu hỏi ít nhất 2 nguồn độc lập; không tìm được thì `unknown`, không bịa nguồn) và mẫu đầu ra JSON `conceptflow.research.v1` (mã dữ kiện, câu hỏi, dữ kiện bằng tiếng Việt, loại `fact` / `hypothesis` / `unknown` / `misconception`, tên nguồn, nơi đăng, năm, link, đoạn trích, độ tin, ghi chú). Creator chép prompt ra một công cụ nghiên cứu có tra web (ChatGPT, Perplexity, Gemini Deep Research…).
  3. *Dán kết quả*: Creator dán toàn bộ câu trả lời, dán nhiều lần được. Hệ thống tự tìm khối JSON, đọc từng dữ kiện, gộp theo mã (thêm hoặc cập nhật), bỏ dòng lỗi và ghi rõ lý do; mỗi lần dán có một dòng nhật ký (lúc nào, đọc được bao nhiêu, thêm và cập nhật mã nào, bỏ dòng nào). Không có khối JSON thì báo rõ, không đọc gì.
  4. *Kiểm link*: với mỗi dữ kiện có link và đoạn trích, backend tải trang và kiểm đoạn trích có trong trang không: **khớp**, **không khớp** (tải được trang nhưng không thấy đoạn trích; nêu câu gần giống nhất nếu có), **không tải được** (trang trả phí, lỗi, quá thời gian; nêu lý do), **không có nguồn**, **đang kiểm**. "Kiểm lại" từng dữ kiện. Việc tải trang chặn mọi địa chỉ nội bộ (ADR-0043).
  5. *Bảng dữ kiện*: mã, câu hỏi, dữ kiện, loại (đổi ngay trong bảng), nguồn và đoạn trích, kết quả kiểm link, "bắt buộc nói", Sửa (dữ kiện, link, đoạn trích; lưu thì kiểm lại link), Bỏ / Giữ lại (bỏ dữ kiện đang được dùng thì có xác nhận kèm danh sách beat và câu dùng nó), thêm dữ kiện tay (có link và đoạn trích thì kiểm như dữ kiện dán vào). Lọc theo: tất cả, khớp, không khớp, không tải được, không có nguồn, đã bỏ.
  6. *Độ phủ câu hỏi*: mỗi câu hỏi nên có ít nhất 2 nguồn độc lập khớp; câu hỏi chưa đủ, hoặc có dữ kiện không khớp, được gom vào một prompt bổ sung (đánh mã tiếp theo mã lớn nhất).
  7. *Duyệt bảng dữ kiện*: duyệt không cần mọi link khớp; dữ kiện chưa khớp mang nhãn ở mọi bước sau và bước Rà soát nhắc lại câu thoại dùng chúng.
  8. *Bỏ qua bước* (có xác nhận): ở bước Câu chuyện AI tự liệt kê dữ kiện cần dùng từ hiểu biết của nó, mỗi dữ kiện vẫn có mã và mang nhãn "chưa có nguồn"; beat và lời thoại tham chiếu mã như thường; bước Rà soát hiện một cảnh báo, chấp nhận là chốt được; ghi chú bàn giao mang nhãn này. Không bước nào bị chặn. "Làm nghiên cứu" quay lại bước này bất cứ lúc nào.
  9. Nguồn tra cứu đặt sau một cổng (port) để sau này cắm API tìm kiếm (Tavily, Wikipedia, Semantic Scholar) mà không đổi bảng dữ kiện; CR-072 chỉ làm cách chép prompt và dán kết quả.
- **FR-29. Xây dựng câu chuyện** (bước 4).
  1. *Góc tiếp cận · kiểu video*: gợi ý theo FR-19. Đổi góc khi câu chuyện đã duyệt thì có xác nhận: khung câu chuyện viết lại theo góc mới (ô Creator viết giữ nguyên), beat sheet và lời thoại thành "cần làm lại", bảng dữ kiện giữ nguyên.
  2. *Khung câu chuyện*: câu hỏi mở đầu, lời hứa với khán giả, chuỗi nhân quả (mỗi bước gắn mã dữ kiện), kết luận · payoff (gắn mã dữ kiện), logline (một câu), tóm tắt (một đoạn). Mỗi ô sửa tay hoặc làm lại kèm ghi chú riêng; ô Creator viết có dấu và được AI giữ nguyên.
  3. *Dữ kiện dùng trong chuyện*: chọn dữ kiện dùng; dữ kiện "bắt buộc nói" luôn dùng. Giả thuyết dùng thì lời thoại phải nói rõ "người ta cho rằng"; "chưa biết" chỉ dùng để nói khoa học chưa biết; ngộ nhận chỉ nhắc để gỡ.
  4. Một chỗ duyệt cho cả câu chuyện. Sửa một ô thì beat liên quan thành "cần làm lại"; beat không liên quan giữ nguyên.
- **FR-30. Beat sheet** (bước 5).
  1. Mỗi beat có chức năng (Hook, Đặt vấn đề, Giải thích cơ chế, Cao trào, Mở rộng hệ quả, Payoff), tên, mục tiêu · bước tiến, nội dung chính, dữ kiện dùng (gắn thêm, bỏ ra; bỏ dữ kiện đang có trong lời thoại thì có xác nhận), thời lượng mục tiêu (± 5 giây), chuyển sang beat sau. Chưa viết lời thoại.
  2. *Cấu trúc*: dải thời gian chia theo thời lượng từng beat; kiểm tổng thời lượng trong khung mục tiêu, hook không quá 20 giây, dữ kiện bắt buộc có trong beat, beat nào cũng có mục tiêu, beat cuối là payoff.
  3. Duyệt, sửa tay, làm lại kèm ghi chú từng beat; "Duyệt tất cả beat còn lại". Đổi thứ tự (xác nhận nếu lời thoại đã duyệt), thêm beat sau một beat (AI viết beat mới), xoá beat (xác nhận, nêu cả dữ kiện bắt buộc sẽ không còn trong beat nào).
  4. Sửa một beat chỉ làm lời thoại và ghi chú bàn giao của beat đó thành "cần làm lại"; đổi thời lượng thì chỉ beat đó về chờ duyệt.
- **FR-31. Viết kịch bản** (bước 6).
  1. Lời dẫn chia theo beat; mỗi câu có mã (beat.câu), thời lượng, mã dữ kiện nó dựa vào. Duyệt, sửa tay (cả các câu của beat trong một hộp thoại), làm lại kèm ghi chú theo từng beat; "Duyệt tất cả beat còn lại".
  2. *Nghe thử* bằng giọng chọn ở Cấu hình: từng câu, cả beat, cả kịch bản. Thời lượng ước tính lấy từ giọng đọc thật của từng câu, so với khung mục tiêu. Câu không đổi thì dùng lại file đã đọc; câu sửa thì đọc lại câu đó (cùng cơ chế với Giọng sơ bộ, FR-9).
  3. *Kiểm khi đọc*: câu dài (đọc một hơi quá lâu) và câu khó nghe (từ ít dùng, dễ nghe nhầm), mỗi chỗ có gợi ý của AI để áp dụng; câu mở đặt câu hỏi trong vài giây đầu; câu kết quay lại hình ảnh mở đầu.
  4. *Chỗ nối giữa hai beat*: AI kiểm câu cuối beat trước và câu đầu beat sau; chỗ nối chưa mượt có gợi ý thay câu đầu beat sau để áp dụng. Thứ tự beat vừa đổi thì chỗ nối được kiểm lại khi duyệt.
- **FR-32. Rà soát và chốt kịch bản; bộ bàn giao** (bước 7). Ba tab:
  1. *Rà soát*: máy kiểm và AI rà soát theo nhóm Độ chính xác, Nhất quán với nghiên cứu, Thời lượng, Logic và nhịp, Dễ hiểu sai, Khả năng sản xuất. Máy kiểm: dữ kiện bắt buộc có trong lời thoại; câu thoại dùng dữ kiện có đoạn trích không khớp; tổng thời lượng so với khung; câu có mốc thời gian hay con số mà chưa gắn dữ kiện; mọi dữ kiện chưa có nguồn khi bỏ qua Nghiên cứu. AI rà soát: giả thuyết bị nói như sự thật, chỗ chuyển đột ngột, chữ dễ hiểu sai, cảnh cần bối cảnh chưa có. Mỗi mục: Mở câu đó, Áp dụng gợi ý, Mở ở Nghiên cứu, Thêm vào ghi chú bàn giao, hoặc Chấp nhận kèm lý do. Ghi chú rà soát của AI và của Creator lưu cùng phiên bản.
  2. *Ghi chú bàn giao*: bảng thuật ngữ dùng chung (thuật ngữ, nói là, vẽ là, tránh; thêm được) và, cho mỗi beat, năm mục: thông tin bắt buộc truyền đạt, thuật ngữ cần giải thích, phải giữ chính xác, được phép sáng tạo, điểm chưa chắc. Mỗi mục ghi các vai AI phía sau đọc nó (Mỹ thuật, Storyboard, Giọng, Hình minh hoạ, Code, Validate). Duyệt, sửa tay, làm lại kèm ghi chú theo beat; "Duyệt tất cả ghi chú còn lại".
  3. *Bộ bàn giao và phiên bản*: điều kiện chốt (không còn mục rà soát cần xử lý, lời thoại mọi beat đã duyệt, ghi chú bàn giao mọi beat đã duyệt); bộ A Kịch bản (chủ đề, khán giả, thời lượng mục tiêu và ước tính, số câu và số beat, trình tự và chuyển tiếp, nguồn và lưu ý độ chính xác) và bộ B Ghi chú bàn giao; bảng "các bước sau đọc gì"; lịch sử phiên bản (bản, lúc, ghi chú). "Chốt phiên bản vN" đóng băng một phiên bản để các bước sau dùng. Sửa sau khi chốt ở bất kỳ bước nào từ 1 tới 7 mở bản nháp vN+1 kèm danh sách thay đổi; các bước sau vẫn dùng vN tới khi Creator chốt vN+1; chốt vN+1 có xác nhận nêu phần sẽ làm lại, và chỉ phần phụ thuộc vào chỗ đổi thành "cần làm lại" (ADR-0044). Không có gì bị khoá cứng.
  4. Các bước sau đọc bộ bàn giao từ DB, theo đúng phiên bản đã chốt, không qua file: Mỹ thuật lượt 1 (giọng điệu và ngoài phạm vi của brief, thuật ngữ, phải giữ chính xác), Storyboard (lời thoại theo beat, mục tiêu và chuyển tiếp, bắt buộc truyền đạt, được phép sáng tạo, điểm chưa chắc), Mỹ thuật lượt 2 (chức năng từng beat để chia color key), Giọng (lời thoại, thuật ngữ, cách đọc), Hình minh hoạ (phải giữ chính xác, thuật ngữ, điểm chưa chắc), Code (được phép sáng tạo), Validate (phải giữ chính xác).
  5. Không có bước "kiểm chứng lại sau khi chốt" (Creator: chưa cần).
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
- **FR-9. Giọng sơ bộ chạy sớm** (câu 7 B). Khi storyboard và lời thoại đã có, TTS chạy bằng giọng chính thức; thời lượng shot lấy từ giọng thật. Câu không đổi thì bước TTS sau dùng lại file đã có; câu sửa thì đọc lại câu đó. Một người kể. Sửa giọng theo FR-20. Việc đọc từng câu đã có từ bước Viết kịch bản (FR-31, nghe thử); bước Giọng sơ bộ dùng lại file đã đọc khi câu và cách đọc không đổi.
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
| ~~Nghiên cứu có nguồn thật~~ | Đưa vào CR-072 ở FR-28 (Creator, 2026-10-17), thay câu 20 A | — |
| Tra nguồn tự động bằng API tìm kiếm (Tavily, Wikipedia, Semantic Scholar) | CR-072 chỉ làm chép prompt và dán kết quả; cổng nguồn tra cứu để sẵn (FR-28 mục 9) | backlog `aidlc-state.md` |
| Kiểm chứng lại sau khi chốt kịch bản | Creator: chưa cần (2026-10-17) | backlog `aidlc-state.md` |
| Gom nhiều chỗ duyệt của Tiền kỳ · Nội dung vào ít lần chờ hơn | Tạm một chỗ duyệt mỗi bước, tối ưu sau (Creator, 2026-10-17) | backlog `aidlc-state.md` |
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
11. Sửa hoặc bỏ một dữ kiện ở Nghiên cứu chỉ làm beat và câu thoại dùng dữ kiện đó thành "cần làm lại" hoặc được rà soát lại; beat và câu khác giữ nguyên.
12. Bỏ qua bước Nghiên cứu vẫn đi hết luồng tới video đăng được; mọi dữ kiện mang nhãn "chưa có nguồn" từ Câu chuyện tới ghi chú bàn giao, và bước Rà soát hiện đúng một cảnh báo về việc này.
13. Dán kết quả nghiên cứu có một đoạn trích không có trong trang nguồn thì dữ kiện đó hiện "không khớp"; link trỏ tới địa chỉ nội bộ (localhost, mạng riêng, địa chỉ metadata) bị từ chối, không được tải.
14. Sửa lời thoại một beat sau khi đã chốt v1 thì các bước sau vẫn dùng v1; chốt v2 thì chỉ phần phụ thuộc vào beat đó thành "cần làm lại", và lịch sử còn cả v1, v2.

## ADR cần viết

- Luồng ba giai đoạn và bộ thiết kế mỹ thuật (FR-1, FR-3).
- Phong cách theo thế giới, thay FR7 nền sáng của CR-067 (FR-5, FR-7).
- Render theo cảnh có đoạn chồng; hợp đồng file mới (FR-14).
- Ngôn ngữ hình của kênh: dữ liệu, chỉ một bật, video giữ phiên bản đã dùng (FR-18).
- Bỏ lượt AI tự sửa code, lỗi hiện cho Creator (FR-22).
- Luồng 21 bước, Tiền kỳ chia Nội dung và Hình ảnh (FR-1; thay ADR-0034, viết ở ADR-0042).
- Nghiên cứu có nguồn: chép prompt ra công cụ ngoài, dán kết quả, backend kiểm đoạn trích, chặn địa chỉ nội bộ (FR-28; ADR-0043).
- Chốt phiên bản kịch bản và bộ bàn giao trong DB (FR-32; ADR-0044).

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
