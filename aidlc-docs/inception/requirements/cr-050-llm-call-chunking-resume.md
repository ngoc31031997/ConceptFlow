# CR-050 — Luồng tạo video chạy lại được từng phần, Shot Spec cho Remotion, và các sửa lỗi từ review

## Date
2026-09-29

## Stage
INCEPTION — Requirements Analysis (Comprehensive: nhiều service, đổi contract, cần ADR). Nhánh `feature/cr-050-llm-call-chunking-resume`, tạo từ `main` @ `1fe7e44`. **Chờ Creator duyệt requirements.**

## Yêu cầu của Creator (nguyên văn)
1. "kiểm tra lại các tác vụ có gọi api llm xem có hợp lý hay chưa, hiện nay ở đoạn code hay bị ngắt và lỗi, ở đoạn đó có thể chia nhỏ ra để chạy song song hoặc tuần tự để khi có lỗi chỉ cần chạy lại đúng đoạn lỗi ko phải toàn bộ như hiện tại"
2. "có lẽ cần nghiêm túc suy nghĩ 1 solution tốt hơn cho bước chuản bị scirpt và render code này sao cho tiết kiệm token và tối ưu hơn nhưng vẫn đảm bảo chất lượng video"
3. "thử revew lại toàn bộ quá trình tạo video từ story, visual, render hinh minh hoạ đến code. xem còn sửa chữa và tối ưu được gì không?"

## Tài liệu liên quan
- Số liệu và phân tích: `cr-050-pipeline-review.md`, `cr-050-solution-options.md`.
- Câu trả lời: `cr-050-requirement-questions.md` (vòng 1), `cr-050-clarification-questions.md` (vòng 2), câu trả lời trong chat ngày 29/09.

## Phân tích ý định
- **Loại**: Enhancement + Refactoring (pipeline sinh code) + Bug fix (I1, X1).
- **Phạm vi**: Multiple Components: `llm-service`, `authoring-service`, `web-gui`, và `rendering` (render Shot Spec thành TSX, xem thiết kế).
- **Độ phức tạp**: Complex. Cần ADR mới cho Shot Spec và contract mới giữa `authoring-service` và `llm-service`.

## Quyết định của Creator

| Câu | Quyết định |
|---|---|
| D1 | Làm **cả** phần lưu và chạy lại từng đoạn **và** Shot Spec ("làm shot spec luôn"). |
| R1 | Lượt AI lỗi **không** xoá các bước sau; chỉ xoá khi lưu nội dung mới khác nội dung cũ. |
| R2 | Các lỗi từ review (I1, C5, X1) gộp vào CR-050. |
| Q2 | Không tự thử lại; Creator chạy lại trên giao diện. |
| Q3 | Một đoạn lỗi thì các đoạn khác vẫn chạy tiếp. Đoạn lỗi chạy riêng được bằng AI, hoặc bằng AI ngoài (sao chép prompt, dán kết quả). |
| Q4 | Lưu từng đoạn trong DB. |
| Q8 | Mỗi đoạn chỉ lưu một kết quả cuối cùng; lượt sửa ghi đè lên. |
| C1 | Story và các gợi ý giữ một lượt gọi; Storyboard chia theo cảnh; Code chia theo đoạn. |
| C1b | Storyboard: viết **khung chung** trước, sau đó các cảnh dựa trên khung đó. |
| C2 | Các đoạn đã xong **chỉ** bị bỏ khi **prompt** của bước đổi. Đổi model thì vẫn dùng lại. |
| C3 / C3b | Số shot mỗi đoạn là cài đặt Creator chỉnh được, mặc định **3**. |
| C4 | Giao diện liệt kê đầy đủ các đoạn, mỗi đoạn có trạng thái và nút riêng. |

## ⚠️ Hai điểm tôi phải diễn giải — cần Creator xác nhận khi duyệt
1. **C2 ("chỉ khi prompt đổi") đi cùng R1 (lưu nội dung mới khác cũ thì xoá bước sau).**
   - Nếu Storyboard được sửa rồi lưu, R1 sẽ xoá Code. Còn nếu giữ nguyên mọi đoạn Code, code sẽ không khớp storyboard mới.
   - **Đề xuất**: đoạn Code được gắn với **nội dung các shot của nó**. Sửa storyboard thì chỉ những đoạn có shot thật sự thay đổi mới phải sinh lại; đoạn có shot không đổi được giữ. Tương tự, Storyboard được gắn với từng cảnh của Story.
   - Như vậy: đổi model thì giữ hết; đổi prompt thì bỏ hết; sửa nội dung thì chỉ bỏ phần bị sửa.
2. **Shot Spec chỉ áp cho Remotion.** Manim vẫn được dùng nhiều (14 project Manim, 13 project Remotion).
   - **Đề xuất**: Manim giữ đường sinh code hiện tại nhưng được lưu và chạy lại từng đoạn (FR-1 đến FR-6). Shot Spec cho Manim để sang CR sau.

## Yêu cầu chức năng

### A. Lưu và chạy lại từng phần (Storyboard và Code, cả Remotion lẫn Manim)
- **FR-1 — Đoạn là đơn vị lưu trữ.**
  - Storyboard: một đoạn "khung chung" và mỗi cảnh một đoạn.
  - Code: một đoạn khung (LAYOUT/cast) và mỗi nhóm N shot một đoạn.
  - Mỗi đoạn có trạng thái (chờ / đang chạy / xong / lỗi kèm lý do), nội dung cuối cùng, nguồn (AI / AI ngoài / sửa tay) và dấu vân tay đầu vào (prompt + nội dung phần nó phụ thuộc).
  - Lưu trong DB của `authoring-service`.
- **FR-2 — Chạy tiếp khi một đoạn lỗi.** Các đoạn khác vẫn chạy. Bước kết thúc với danh sách đoạn lỗi. Ba lỗi vẫn dừng ngay: key chết, hết số dư, Creator huỷ.
- **FR-3 — Không tự thử lại.** Một lượt lỗi được ghi là đoạn lỗi; Creator quyết định chạy lại.
- **FR-4 — Chạy lại đúng phần cần.**
  - "Chạy các đoạn còn thiếu" chỉ gọi model cho đoạn lỗi hoặc chưa chạy.
  - "Chạy lại đoạn này" gọi model cho một đoạn.
  - "Sinh lại toàn bộ" bỏ mọi đoạn và sinh lại từ đầu (có hỏi xác nhận).
- **FR-5 — AI ngoài cho từng đoạn.** Sao chép đúng prompt của một đoạn; dán kết quả về. Hệ thống kiểm định dạng (đủ shot, đúng schema) rồi lưu như đoạn đã xong.
- **FR-6 — Khi nào bỏ đoạn đã xong.**
  - Prompt của bước đổi: bỏ toàn bộ đoạn của bước đó.
  - Nội dung đầu vào của một đoạn đổi: bỏ đúng đoạn đó (xem điểm diễn giải 1).
  - Đổi model: giữ.
- **FR-7 — Số shot mỗi đoạn** của bước Code là cài đặt Creator chỉnh được, mặc định 3.
- **FR-8 — Ghép và kiểm.** Khi mọi đoạn đã xong, hệ thống tự ghép, kiểm, và sửa các shot lỗi như hiện nay. Kết quả sửa ghi đè lên đoạn chứa shot đó.
- **FR-9 — Giao diện (Storyboard và Code)**: danh sách đầy đủ các đoạn với trạng thái, lý do lỗi, thời gian, và các nút ở FR-4/FR-5, theo `docs/ux-ui-design-rules.md`. Đoạn đã xong có thể thu gọn.

### B. Storyboard theo cảnh
- **FR-10 — Khung chung trước.** Một lượt viết khung chung (hero, world, palette, layout, và quy ước hình dùng lại). Sau đó mỗi cảnh một lượt, chạy song song, nhận khung chung, Story và cảnh liền trước/liền sau.
- **FR-11** — Ghép các cảnh thành storyboard JSON như hiện nay (qua `storyboard/finalize`). Các cảnh báo CR-048 chạy trên bản ghép.

### C. Shot Spec cho Remotion
- **FR-12 — Định dạng Shot Spec.** Mỗi shot là JSON theo một schema cố định:
  - phần tử (component của bộ kit hoặc thư viện, props của nó, vị trí và cỡ tương đối);
  - chuyển động (loại, thời điểm theo tỉ lệ thời lượng, easing);
  - nhãn chữ, máy quay, chuyển cảnh;
  - phần tử `custom` là lối thoát cho hình không diễn tả được, do AI viết TSX riêng cho phần tử đó.
- **FR-13 — Trình biên dịch cố định**, không dùng AI, chuyển Shot Spec thành TSX Remotion. Cùng một spec thì luôn ra cùng một code.
- **FR-14 — Kiểm trên spec trước khi render**: schema, component và prop có thật, màu thuộc PALETTE, vùng an toàn, dải phụ đề, cỡ tối thiểu của vật chính. Lỗi nào sửa được một cách cố định thì tự sửa (ví dụ kéo vật vào vùng an toàn) và báo cho Creator; lỗi còn lại gửi lại cho AI một thông báo ngắn, chính xác.
- **FR-15 — Pha 0 (spike đo) là cổng.** Trước khi Shot Spec thành đường mặc định cho Remotion:
  - Chạy trên storyboard thật `f7103848` (52 shot).
  - Đo token, thời gian, tỉ lệ lỗi, và **Creator xem, chấm chất lượng hình** so với video hiện tại.
  - Nếu chất lượng không đạt, Creator quyết định dừng hoặc chỉnh; đường TSX hiện tại vẫn giữ.
- **FR-16** — Đường TSX hiện tại cho Remotion vẫn chọn được (theo project) cho đến khi Creator quyết định bỏ.

### D. Sửa lỗi từ review
- **FR-17 (I1)** — Danh sách hình minh hoạ ghi lại dấu vân tay của storyboard lúc lập. Khi storyboard đổi:
  - Danh sách được đánh dấu là cũ và giao diện báo rõ.
  - Chuỗi tự chạy lập lại danh sách; hình đã vẽ vẫn được dùng lại từ thư viện.
  - Bước Code không chạy với danh sách cũ.
- **FR-18 (X1)** — Lượt AI lỗi ở Story/Storyboard không xoá gì. Việc xoá các bước sau chỉ xảy ra khi lưu nội dung mới khác nội dung cũ.
- **FR-19 (C5)** — Bộ chọn model cho bước Code hiện số liệu đã đo (token và thời gian trung bình mỗi đoạn) và cảnh báo model đã có lỗi timeout hoặc budget ở bước này. Provider Ollama bị chặn cho bước Code.
- **FR-20 (V2)** — Storyboard có cảnh báo thì chuỗi tự chạy dừng lại chờ Creator xác nhận trước khi sang Hình minh hoạ và Code.
- **FR-21 (X2)** — Mỗi lượt gọi model của Storyboard và Code được ghi `llm_usage` ngay khi lượt đó xong, không chờ cả bước.
- **FR-22 (X3)** — Chẩn đoán của mỗi lần kiểm (loại lỗi, luật vi phạm, shot) được lưu để thống kê được.

## Yêu cầu phi chức năng
- **NFR-1 — Không giả thành công.** Bước chỉ "xong" khi mọi đoạn xong và bản ghép qua cổng kiểm. Đoạn không kiểm được thì báo là lỗi, không bao giờ báo đạt.
- **NFR-2** — Mọi lượt gọi đã tính tiền đều có một dòng `llm_usage`, kể cả khi run bị ngắt giữa chừng.
- **NFR-3** — Rebuild hoặc restart `llm-service`/`authoring-service` không làm mất đoạn đã xong. Run đang chạy thì các đoạn dở được đánh dấu lỗi ("bị ngắt"), để Creator chạy tiếp.
- **NFR-4** — Không đổi contract RabbitMQ. Contract HTTP `authoring-service` ↔ `llm-service` đổi có version và được ghi trong `docs/contracts/`.
- **NFR-5 — Đo được.** Đạt khi so với số liệu hiện tại (`cr-050-pipeline-review.md`):
  - (a) lỗi ở một đoạn không bao giờ bắt sinh lại đoạn đã xong;
  - (b) với Shot Spec, token đầu ra bước Code **giảm ít nhất 50%** trên video đo ở Pha 0, và Creator chấm chất lượng không kém hơn.
  - Mức 50% là đề xuất, Creator chỉnh được khi duyệt.
- **NFR-6** — Project cũ vẫn mở và render được. Code đã lưu của project cũ không bị đổi.

## Ngoài phạm vi
- Shot Spec cho Manim (CR sau).
- Hướng C: đạo diễn viết Shot Spec trực tiếp.
- Chia nhỏ Story và các gợi ý metadata / kịch bản ngắn.
- Tự thử lại.
- Gộp hai bản prompt thủ công và AI (X4).
- Các bước TTS, render, ghép video.

## Kiến trúc cần quyết ở giai đoạn Design
- ADR mới: Shot Spec (định dạng, nơi đặt trình biên dịch, versioning schema), lưu đoạn trong `authoring-service`, contract HTTP mới cho chạy theo đoạn.
- Nơi đặt trình biên dịch Shot Spec → TSX: `llm-service` (cạnh merger hiện tại) hay `rendering` (nơi có bộ kit TypeScript và checker).

## Kiểm thử (dự kiến)
- Unit: trình biên dịch Shot Spec (golden TSX cho các spec mẫu), kiểm spec, dấu vân tay và luật bỏ đoạn (FR-6), không xoá khi lỗi (FR-18), danh sách hình cũ (FR-17).
- Integration: chạy theo đoạn với provider giả trả lỗi ở một đoạn, rồi chạy lại chỉ đoạn đó; restart giữa run.
- Pha 0: đo thật trên `f7103848` với Hive (tốn token thật; chạy khi Creator đồng ý).
- `make check`, rebuild các service bị đổi, kiểm tra trực tiếp trên giao diện.

## Workflow tiếp theo (đề xuất)
Workflow Planning → Application Design (ADR) → chia unit. Dự kiến làm theo thứ tự:
1. FR-17..22 (sửa lỗi, nhỏ);
2. FR-1..9 (lưu và chạy lại từng đoạn cho Code);
3. FR-10..11 (Storyboard theo cảnh);
4. Pha 0 Shot Spec (FR-12..15);
5. Shot Spec vào luồng chính (FR-16).
