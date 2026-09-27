# CR-044 — Bước dựng hình minh hoạ theo từng vật, màn duyệt dạng lưới, thư viện hình dùng lại

## Date
2026-09-27

## Stage
Requirements Analysis — **chờ Creator duyệt**, chưa code.

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

## Lưu trữ — quyết định cần Creator chốt
- **Đề xuất:** code SVG và ảnh xem trước lưu trong Postgres của `authoring-service` (một hình chỉ vài KB–vài chục KB), không cần mạng, có phiên bản, dựng lại tức thì.
- **Google Drive cá nhân (Creator đề xuất):** hợp để **sao lưu/xuất** video thành phẩm và bộ thư viện hình, không nên là nơi lưu chính của hình (mỗi lần dựng phải tải qua mạng, phụ thuộc token hết hạn). Đã có sẵn OAuth Google (YouTube, CR-012) nên có thể dùng lại client với quyền `drive.file`. Đề xuất tách thành CR riêng sau CR này.

## Phạm vi kỹ thuật
Không thêm service mới. `authoring-service`: bảng `illustration_assets`, API. `llm-service`: bước vẽ từng vật (chạy song song). `rendering`: endpoint dựng PNG/clip một hình (tái dùng check server). `web-gui`: màn duyệt lưới + trang Thư viện. Chỉ engine Remotion.

## Rủi ro
- Vật phức tạp (xe máy, cơ quan nội tạng) có thể phải vẽ lại nhiều lần hoặc sửa tay — đây là lý do có nút Sửa code.
- Thêm thời gian cho video đầu của mỗi chủ đề; giảm dần khi thư viện lớn lên.
