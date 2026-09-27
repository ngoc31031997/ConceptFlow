# CR-044 — Bước dựng hình minh hoạ theo từng vật, màn duyệt dạng lưới, thư viện hình dùng lại

## Date
2026-09-27

## Stage
Requirements Analysis — **Creator đã duyệt 2026-09-27** (lưu trong DB, Drive tách CR sau; bước Kỹ sư chờ mọi hình được duyệt hoặc bỏ qua). Đang code.

## Bối cảnh
CR-043 thêm 34 hình vẽ sẵn, nhưng mỗi chủ đề mới sẽ thiếu vật riêng. Creator đề xuất: sau bước Đạo diễn, dựng riêng từng vật được nhắc tới, duyệt từng ô, vẽ lại ô không ưng, sửa tay code SVG khi AI không vẽ được, và hình có animation.

## Yêu cầu chức năng
- **FR1 — Danh sách vật.** Đạo diễn in thêm `assets`: mỗi vật một mục (tên, mô tả hình, trạng thái cần diễn như `mood`/`open`/`decay`). Hệ thống đối chiếu với thư viện (hình có sẵn của CR-043 + hình đã duyệt trước đây); chỉ vật chưa có mới được đưa đi vẽ.
- **FR2 — Vẽ từng vật.** Mỗi vật một lượt gọi LLM, prompt chỉ về vật đó + luật nét vẽ của kênh + 2–3 hình mẫu làm ví dụ. Kết quả là một component React/SVG có tham số trạng thái và chuyển động tự thân (thở, chớp mắt, lắc, đập…), cùng quy ước `x, y, size, rotate, flip, scale, opacity` như bộ CR-043.
- **FR3 — Kiểm tra tự động.** Mỗi hình: biên dịch `tsc`, dựng thành PNG (và clip ngắn 2 giây để thấy animation), kiểm tra không tràn khung. Model đọc ảnh chấm "có nhận ra đây là <vật> không"; không đạt thì tự vẽ lại, tối đa 2 lần.
- **FR4 — Màn duyệt dạng lưới.** Mỗi hình một ô riêng (ảnh xem trước + clip animation khi rê chuột), trạng thái: đang vẽ / chờ duyệt / đã duyệt / lỗi. Mỗi ô có nút:
  - **Vẽ lại** (tuỳ chọn kèm ghi chú, vd "răng to hơn, mắt tròn");
  - **Sửa code** — trình soạn SVG/TSX ngay trong ô, bấm xem trước để dựng lại chỉ ô đó;
  - **Duyệt**.
  Bước Kỹ sư chỉ chạy khi mọi vật cần thiết đã được duyệt (hoặc Creator chọn bỏ qua vật đó).
- **FR5 — Thư viện dùng lại.** Hình đã duyệt được lưu (tên, thẻ, mô tả, code, phiên bản, ảnh xem trước) và có trong danh sách "hình có sẵn" cho mọi video sau. Có trang Thư viện để xem/sửa/xoá.
- **FR6 — Kỹ sư dùng hình.** Prompt Kỹ sư Remotion nhận thêm API của các hình mới trong video này; khung code import chúng.

## Thư mục thư viện (Creator yêu cầu 2026-09-27)
Mỗi hình thuộc đúng MỘT thư mục, có thêm thẻ tự do để tìm. Thư mục cố định lúc đầu (Creator thêm được):
`con-nguoi` (người, nghề nghiệp), `dong-vat`, `thuc-vat`, `co-the-suc-khoe` (bộ phận cơ thể, vi khuẩn, thuốc), `do-an-thuc-uong`, `phuong-tien` (xe, tàu, máy bay), `do-vat` (đồ gia dụng, dụng cụ, thiết bị), `cong-trinh-noi-chon` (nhà, trường, bệnh viện, cảnh), `thien-nhien-thoi-tiet` (mặt trời, mây, núi, nước), `tien-kinh-te`, `khoa-hoc-cong-nghe`, `bieu-tuong` (dấu tích, tim, bóng đèn, lấp lánh, bong bóng thoại), `boi-canh` (nền, mảng màu).
- 34 hình của CR-043 được xếp sẵn vào các thư mục này.
- Bước vẽ: LLM gán thư mục + thẻ cho hình mới; Creator đổi được ở màn duyệt.
- Đối chiếu FR1 tìm theo tên, thẻ và mô tả trong đúng thư mục trước, rồi mới toàn thư viện — tránh vẽ lại thứ đã có.
- Trang Thư viện: cây thư mục bên trái, lưới ô bên phải, ô tìm kiếm.

## Lưu trữ — đã chốt
- **Chốt:** code SVG và ảnh xem trước lưu trong Postgres của `authoring-service` (một hình chỉ vài KB–vài chục KB), không cần mạng, có phiên bản, dựng lại tức thì.
- **Google Drive cá nhân (Creator đề xuất):** hợp để **sao lưu/xuất** video thành phẩm và bộ thư viện hình, không nên là nơi lưu chính của hình (mỗi lần dựng phải tải qua mạng, phụ thuộc token hết hạn). Đã có sẵn OAuth Google (YouTube, CR-012) nên có thể dùng lại client với quyền `drive.file`. Đề xuất tách thành CR riêng sau CR này.

## Phạm vi kỹ thuật
Không thêm service mới. `authoring-service`: bảng `illustration_assets`, API. `llm-service`: bước vẽ từng vật (chạy song song). `rendering`: endpoint dựng PNG/clip một hình (tái dùng check server). `web-gui`: màn duyệt lưới + trang Thư viện. Chỉ engine Remotion.

## Rủi ro
- Vật phức tạp (xe máy, cơ quan nội tạng) có thể phải vẽ lại nhiều lần hoặc sửa tay — đây là lý do có nút Sửa code.
- Thêm thời gian cho video đầu của mỗi chủ đề; giảm dần khi thư viện lớn lên.

## Luật style và ba cách làm giàu thư viện (Creator yêu cầu 2026-09-27)
- **Luật style** `authoring-service/internal/domain/prompts/illustration_style_vi.txt`: 24 luật [S1]–[S24] (hình khối, tô bóng, màu + bảng màu kênh, khuôn mặt, khung, chuyển động, đặt tên). Hiện trên trang Thư viện, đưa nguyên văn cho AI vẽ.
- **Hình mẫu**: 3 hình chuẩn (SchoolBus, Cat, Microscope) trong `illustration_exemplars_vi.txt`, nạp vào thư viện dạng chỉ-xem "Hình mẫu". Model hiện dùng chỉ đọc chữ, nên AI tham chiếu CODE của hình mẫu (cộng tối đa 2 hình Creator đã duyệt cùng thư mục); Creator xem ẢNH của chính các hình đó.
- **Kiểm tra style tự động** (`rendering/domain/illustration_style.py`), chạy ở mọi lần xem trước/lưu/AI vẽ: gradient, filter, ảnh, chữ, thiếu Figure, Math.random/Date.now/CSS animation là LỖI chặn lưu; viền quanh khối, góc không bo, màu ngoài bảng, quá 6 màu, quá chi tiết, màu không đổi được qua prop, có mặt mà không chuyển động, chuyển động không tắt được là CẢNH BÁO (lưu vào hình, hiện ⚠ trên ô). Ba hình mẫu có test giữ ở mức không lỗi, không cảnh báo.
- **Ba cách thêm hình**: (1) Viết code; (2) Tải SVG lên — chuyển thành component ngay trong trình duyệt, mở trình sửa để xem trước, cảnh báo style hiện ngay; hình tải lên đứng yên cho tới khi thêm chuyển động; (3) AI vẽ theo mô tả — luật + linh kiện + hình mẫu vào prompt, tự sửa theo lỗi có số dòng tối đa 3 lượt, lưu "Chờ duyệt". Hình của Creator có thêm "Vẽ lại bằng AI" kèm ghi chú.
- **Chưa kiểm chứng**: chất lượng hình AI vẽ với model thật (môi trường thử không có HIVE key). Prompt vẽ là file cố định, chưa sửa được trên màn Cài đặt prompt.

## Sao lưu và khôi phục thư viện (Creator yêu cầu 2026-09-27)
Mục đích: không mất hình khi mất cơ sở dữ liệu. Trang Thư viện hình, ô "Sao lưu" bên trái.
- **Xuất** `GET /v1/admin/illustrations/export` → một file `conceptflow-thu-vien-hinh-<ngày>-<giờ>.zip`: `manifest.json` (định dạng `conceptflow-illustration-library` v1: mọi thư mục; mọi hình của Creator với tên, tiêu đề, thư mục, thẻ, mô tả, cách dùng, trạng thái duyệt, phiên bản) và `hinh/<thư-mục>/<Tên>.tsx|.png|.gif` để mở xem bằng tay. Chỉ đọc cơ sở dữ liệu, không dựng lại gì.
- Hình có sẵn (34 hình CR-043) và 3 hình mẫu **không** nằm trong file: chúng đi kèm phần mềm và được nạp lại mỗi lần khởi động.
- **Nhập** `POST /v1/admin/illustrations/import?on_conflict=skip|replace`, thân là file ZIP (tối đa 256 MB): tạo lại thư mục còn thiếu theo đúng thứ tự, đưa mỗi hình về đúng thư mục của nó, giữ trạng thái duyệt. Hình trùng tên: mặc định bỏ qua; chọn "Ghi đè hình trùng tên" thì ghi đè thành phiên bản mới của hình đang có. Không bao giờ ghi đè hình có sẵn.
- Như lúc tạo hình, mỗi hình được nhập đều phải qua kiểm tra và dựng lại của rendering (thư viện không giữ hình không dựng được với bộ linh kiện hiện tại); ảnh xem trước lưu là ảnh vừa dựng, ảnh trong file chỉ để xem. Hình không qua kiểm tra được báo lỗi kèm số dòng, các hình khác vẫn nhập tiếp. Nếu rendering hay cơ sở dữ liệu hỏng giữa chừng thì dừng, báo còn bao nhiêu hình chưa nhập; nhập lại cùng file sẽ làm tiếp (hình đã nhập được bỏ qua).

## Mốc 3 — hình minh hoạ trong luồng làm video (2026-09-27)
Cách làm khác bản đề xuất ở một điểm: không thêm bước mới vào luồng 1–13 (đánh số bước dùng chung ở orchestrator, web-gui, nhật ký). Giai đoạn hình chạy ở ĐẦU bước Code, chỉ với engine Remotion:
1. **Lập danh sách** (`planner_prompt_vi.txt`, một lượt model): đọc storyboard đối chiếu danh mục thư viện (hình có sẵn + hình đã duyệt, kèm thẻ) → `reuse` (tên phải có thật trong thư viện, nếu không bị bỏ) và `draw` (tối đa 10; tên trùng hình đã có được đổi thành dùng lại; thư mục lạ về `do-vat`). Lưu ở bảng `project_illustrations`.
2. **Vẽ** từng mục bằng AI vẽ (luật style, hình mẫu, tự sửa theo lỗi). Mục vẽ hỏng ở trạng thái "failed" kèm lý do; Creator bấm vẽ lại hoặc bỏ qua.
3. **Cổng**: bước Code dừng với lời nhắn (409, không tính là lỗi AI) khi còn mục chưa duyệt / chưa bỏ qua. Duyệt xong chạy lại bước Code — không lập lại danh sách, không vẽ lại.
4. **Kỹ sư dùng hình**: hình đã duyệt của video (rồi tới phần còn lại của thư viện, tối đa 40, hình mẫu gồm cả) được gửi sang llm-service; prompt có mục C4 liệt kê chúng, và Code Merger dán code của đúng những hình được shot gọi tới vào script (bỏ import, bỏ export) — script render không cần file ngoài. Đã kiểm tra `tsc` thật trên script ghép có hai hình thư viện: 0 lỗi.
- **Giao diện**: tab Code (Remotion) có mục "Hình minh hoạ của video": mỗi mục một ô — Vẽ / Vẽ lại, Duyệt, Sửa code / Vẽ lại bằng AI kèm ghi chú, Bỏ qua / Dùng lại; nút "Lập danh sách từ storyboard" và "Vẽ N hình còn thiếu"; dòng tóm tắt "x/y hình sẵn sàng". Tự làm mới khi bước Code đang vẽ trên server.
- **Kiểm thử**: unit test cho lập danh sách, Ensure/cổng, ForCode (không bao giờ đưa bản nháp), bước Code chờ rồi truyền hình; test Postgres thật cho bảng mới; llm-service 88 test (có dán hình vào script); chạy thật qua HTTP (authoring-service + Postgres + dựng hình thật + model giả có kịch bản): lập danh sách → vẽ (1,8 s) → chờ duyệt → duyệt → mở cổng.
- **Chưa kiểm chứng**: model thật lập danh sách và vẽ ra sao; một video Remotion chạy trọn từ storyboard tới render có dùng hình thư viện.
