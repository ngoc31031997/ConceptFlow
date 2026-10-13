# CR-072 — Đối chiếu yêu cầu với CR con

## Date
2026-10-13

Bước User Stories đã bỏ (Workflow Planning), nên tài liệu này đối chiếu theo yêu cầu: FR-1..FR-26, NFR-1..NFR-7 và tiêu chí chấp nhận 1–10 trong `aidlc-docs/inception/requirements/cr-072-2d-film-pipeline.md`. Định nghĩa CR con: [`unit-of-work.md`](unit-of-work.md).

## Yêu cầu chức năng

| FR | Nội dung | CR con | Ghi chú |
|---|---|---|---|
| FR-1 | Luồng ba giai đoạn | U2a; U4a, U5, U6, U8b | U2a chia giai đoạn và đổi tên bước; bốn CR con sau thêm bước của mình |
| FR-2 | Logline, tóm tắt, dàn ý, lời thoại theo cảnh | U3 | |
| FR-3 | Bộ thiết kế mỹ thuật | U4a (mục 1, 2, 4), U4b (mục 3), U5 (mục 5, 6) | Bốn cách sửa hình: U4a, dùng lại ở U4b |
| FR-4 | Nhân vật cấp video, thư viện nhân vật | U4b | Bỏ nhép miệng cũng ở U4b |
| FR-5 | Phong cách theo thế giới | U4a | ADR-0037 |
| FR-6 | Hành trình màu | U5 | |
| FR-7 | Cảnh giải thích ý trong thế giới | U5 | Thay FR7 của CR-067 (ADR-0037) |
| FR-8 | Máy quay liền mạch | U5 (storyboard), U7a (code dùng chung vật nối), U8a (đoạn chồng khi render và ghép) | |
| FR-9 | Giọng sơ bộ chạy sớm | U6 | ADR-0040 |
| FR-10 | Animatic khung tĩnh | U6 | ADR-0039 |
| FR-11 | Key pose | U7b | Chỉ kit Khoa học phát sáng (U-3.2 A) |
| FR-12 | VFX Remotion | U7b | Chỉ kit Khoa học phát sáng (U-3.2 A) |
| FR-13 | Hình minh hoạ theo bộ mỹ thuật | U4c | |
| FR-14 | Render từng cảnh | U8a | ADR-0038 |
| FR-15 | Prompt và luật viết lại | U1 (luật hình, luật nền); mỗi CR con viết lại prompt của bước nó đổi; U9 rà toàn bộ | |
| FR-16 | Xoá dữ liệu cũ | U1 | Chạy khi U1 vào main, sau sao lưu (W-2 A) |
| FR-17 | Sửa đúng chỗ, chỉ làm lại phần phụ thuộc | U2b (khung); mọi CR con đăng ký loại đơn vị của mình | ADR-0035 |
| FR-18 | Ngôn ngữ hình của kênh | U1 | Khoa học phát sáng bật khi U1 vào main (U-3.1 A); ba ngôn ngữ còn lại chỉ có ảnh mẫu (U-3.2 A) |
| FR-19 | Gợi ý kiểu video | U3 | |
| FR-20 | Sửa giọng từng câu, từ điển phát âm | U6 | |
| FR-21 | Thumbnail Remotion | U9 | |
| FR-22 | Code theo cảnh, lỗi theo shot | U7a | |
| FR-23 | Validate đúng chỗ | U7c | |
| FR-24 | Review đủ thông tin | U7c | |
| FR-25 | Dựng theo ghi chú mốc thời gian | U8b | |
| FR-26 | Merge có tiến độ, QC chỉ cảnh báo | U8a | |

## Yêu cầu phi chức năng

| NFR | Nội dung | CR con |
|---|---|---|
| NFR-1 | Chất lượng trước, chi phí sau; ghi token và thời gian theo bước | Mọi CR con có bước gọi AI (tiêu chí xong chung); U7c hiện chi phí ở Review |
| NFR-2 | Ít phải sửa; đo tỉ lệ duyệt ngay lần đầu | U2b (đếm lần làm lại), U7c (hiện trên Review); Creator đo sau mỗi CR con bằng một video thật |
| NFR-3 | Không nuốt lỗi | Mọi CR con (tiêu chí xong chung) |
| NFR-4 | Hai khung hình 16:9 và 9:16 | U4a, U4b, U5, U6, U7b, U8a |
| NFR-5 | Giao diện theo `docs/ux-ui-design-rules.md` | Mọi CR con có màn hình |
| NFR-6 | Code theo `docs/code-standards-rules.md` | Mọi CR con |
| NFR-7 | Nội dung tiếng Việt và tiếng Anh | Mọi CR con có prompt |

NFR Requirements và NFR Design làm riêng ở U7a (thời gian xem nhanh một shot) và U8a (số cảnh render song song, đoạn chồng, lỗi tạm thời).

## Tiêu chí chấp nhận

Cột "Kiểm ở" là CR con mà khi xong thì tiêu chí kiểm được trọn vẹn.

| # | Tiêu chí | Góp phần | Kiểm ở |
|---|---|---|---|
| 1 | Một project mới đi hết luồng mục tiêu tới video đăng được, Creator duyệt đúng các chỗ trong bảng luồng | mọi CR con | U9 |
| 2 | Hai chủ đề khác nhau cho hai thế giới khác hẳn, vẫn cùng một kênh | U1, U4a, U5 | U5 |
| 3 | Creator xem animatic có giọng trước khi bước Code chạy | U6 | U6 |
| 4 | Một nhân vật giữ đúng model trên mọi shot | U4b, U7a | U7a |
| 5 | Làm lại một cảnh ở màn Dựng không render lại cả video; chỗ nối liền mạch không giật | U5, U8a, U8b | U8b |
| 6 | Dải màu cho thấy color key đổi theo từng phần dàn ý | U5 | U5 |
| 7 | Sửa một cảnh storyboard chỉ làm lại giọng, animatic, hình, code, render của cảnh đó | U2b, U4c, U5, U6, U7a, U8a | U8a |
| 8 | Đổi ngôn ngữ hình đang bật không đổi video đã tạo; video mới dùng ngôn ngữ mới | U1 | U1 (đổi sang một bản sao của Khoa học phát sáng, vì ba ngôn ngữ còn lại chưa bật được) |
| 9 | Shot lỗi ở Code hiện đúng shot và lỗi, không bị AI tự sửa; một cảnh lỗi khi render không dừng cảnh khác | U7a, U8a | U8a |
| 10 | Một từ sửa trong từ điển phát âm được đọc đúng ở video sau | U6 | U6 |

## Kiểm đủ

- Mỗi FR-1..FR-26 có ít nhất một CR con nhận.
- Mỗi NFR-1..NFR-7 có CR con nhận hoặc nằm trong tiêu chí xong chung.
- Mỗi tiêu chí chấp nhận 1–10 có một CR con kiểm được trọn vẹn.
- Việc "Không làm trong CR này" của yêu cầu (nhép miệng, "on 2s", nghiên cứu có nguồn, SFX, nhạc theo mood, nhân vật có giọng riêng, lượt AI tự sửa code, QC chặn đăng) không CR con nào nhận.
- Hai việc mới để lại sau CR-072 (làm đủ ba kit còn lại, bỏ `render_scenes`) ghi ở cuối `unit-of-work.md` và trong backlog của `aidlc-state.md`.
