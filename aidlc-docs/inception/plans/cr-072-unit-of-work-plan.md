# CR-072 — Kế hoạch Units Generation

Trạng thái: **đã duyệt** (Creator, 2026-10-13: "tôi duyệt hết rồi"). U-1 A, U-2 A, U-3.1 A, U-3.2 A, U-4.1 A, U-5 B; tài liệu ở `aidlc-docs/inception/application-design/cr-072/`.

## Date
2026-10-13

Đầu vào: yêu cầu `aidlc-docs/inception/requirements/cr-072-2d-film-pipeline.md` (FR-1..FR-26, NFR-1..NFR-7, tiêu chí chấp nhận 1–10); kế hoạch `aidlc-docs/inception/plans/cr-072-workflow-plan.md` (9 CR con U1..U9, W-1..W-3 A); thiết kế `aidlc-docs/inception/application-design/cr-072/` (thành phần, giao diện, đồ thị đơn vị, ADR-0034..0041).

Units Generation chốt **ranh giới, thứ tự và tiêu chí xong của từng CR con**. Workflow Planning đã chia 9 CR con và ghi: "XL lớn hơn, có thể tách tiếp ở Units Generation". Kế hoạch này xét việc tách đó và những chỗ ranh giới còn mở.

## Kế hoạch

- [x] Creator trả lời U-1..U-5 bên dưới
- [x] Rà câu trả lời, hỏi thêm nếu còn mơ hồ hoặc mâu thuẫn (vòng 1 hỏi lại U-3, U-4; vòng 2 U-3.1, U-3.2, U-4.1 rõ)
- [x] `application-design/cr-072/unit-of-work.md`: từng CR con có phạm vi, FR, service và thành phần đụng tới, thứ nó giao cho CR con sau, tiêu chí xong, cỡ, các stage Construction phải chạy
- [x] `application-design/cr-072/unit-of-work-dependency.md`: ma trận phụ thuộc, sơ đồ thứ tự, chỗ làm song song được, chỗ hai CR con đụng cùng file
- [x] `application-design/cr-072/unit-of-work-story-map.md`: FR-1..FR-26, NFR-1..NFR-7 và tiêu chí chấp nhận 1–10 gán vào CR con (bước User Stories đã bỏ, nên đối chiếu theo yêu cầu thay cho story)
- [x] Kiểm ranh giới và phụ thuộc: đồ thị không có vòng (mọi phụ thuộc chỉ về CR con làm trước); mỗi CR con có tiêu chí "project mới chạy trọn luồng tới video"
- [x] Kiểm mọi FR, NFR và tiêu chí chấp nhận đều có CR con nhận; mọi thành phần trong `components.md` có CR con làm
- [x] Cập nhật `aidlc-state.md`, `audit.md`; trình Creator duyệt

Tài liệu đặt trong `application-design/cr-072/`, cạnh thiết kế của CR-072, để không đè lên `unit-of-work*.md` của 10 unit ban đầu. Bước "chiến lược tổ chức code" chỉ áp cho dự án greenfield nên bỏ: code của mỗi CR con nằm trong thư mục service hiện có.

## Điểm AI suy ra, không cần hỏi

Các điểm dưới đây theo thẳng từ kế hoạch đã duyệt hoặc từ code. Creator thấy sai thì ghi vào đây.

1. **Luồng lớn dần theo CR con.** Kế hoạch đã duyệt ghi "bước chưa làm thì giữ như hiện nay, không có bước giả". Luồng hiện có 13 bước (`services/web-gui/src/utils/flow.ts` `FLOW_LABELS`). U2 chỉ chia 13 bước đó vào ba giai đoạn. Mỗi CR con sau thêm bước của nó vào cả ba bản định nghĩa luồng (orchestrator, authoring-service, web-gui) cùng test hợp đồng: Mỹ thuật lượt 1 ở U4, Mỹ thuật lượt 2 ở U5, Giọng sơ bộ và Animatic ở U6, Dựng ở U8. Luồng đủ 17 bước khi U8 xong.
2. **U3 không phải chờ cổng chọn ngôn ngữ.** Kịch bản không đọc luật hình. Cổng chỉ chặn CR con đầu tiên vẽ theo ngôn ngữ (U4), nên Creator so ảnh mẫu trong lúc U3 chạy.
3. **U7 không phải chờ U6.** Theo ADR-0032, code của shot nhận thời lượng làm props lúc render, không cần biết thời lượng lúc sinh code. U6 và U7 làm song song được như kế hoạch đã ghi. Cạnh `voice_line → code_shot` trong đồ thị đơn vị do CR con vào main sau đăng ký.
4. **Rig key pose và VFX (FR-11, FR-12) chỉ cần kit của U1.** Phần này nằm trong kit Remotion và prompt của Kỹ sư Remotion, không đọc storyboard mới. Nếu tách riêng (U-1 A), nó chạy được ngay sau cổng, song song với tiền kỳ.
5. **Mỗi CR con đi qua `/cr` → `/code` → `/deliver`.** Bản thiết kế của `/cr` là Functional Design của CR con. NFR Requirements và NFR Design chỉ chạy cho CR con có xem nhanh shot và CR con có render theo cảnh (kế hoạch đã duyệt). Infrastructure Design bỏ.
6. **Tiêu chí xong chung cho mọi CR con:** test của các service đụng tới xanh; build lại và khởi động lại đúng các service đã đổi, báo healthy; một project mới chạy trọn luồng tới video; Creator chạy một video thật để đo tỉ lệ duyệt ngay lần đầu (NFR-2); mọi bước mới có trạng thái lỗi và chạy lại, không có đường giả thành công (NFR-3).

## Đánh giá các nhóm câu hỏi

| Nhóm | Hỏi? | Lý do |
|---|---|---|
| Gom yêu cầu vào CR con | Có: U-1, U-4 | Bốn CR con cỡ XL; khung đơn vị của U2 cần chọn bước áp đầu tiên |
| Phụ thuộc và tích hợp | Có: U-2 | Mỗi lần một CR con vào main, dạng dữ liệu đổi trong khi có thể còn project làm dở |
| Người làm và phối hợp | Có: U-5 | Creator duyệt mọi thiết kế; nhiều agent có thể chạy cùng lúc trong các worktree |
| Kỹ thuật (mở rộng, triển khai khác nhau) | Có: U-3 | Bốn kit Remotion: làm sâu cả bốn hay chỉ kit được bật quyết định khối lượng của nhiều CR con. Triển khai không khác nhau giữa các CR con: CR con là lát dọc qua 8 service có sẵn, triển khai bằng build lại service đã đổi. Số worker render để NFR Design của CR con render theo cảnh |
| Miền nghiệp vụ | Không | Ranh giới đã chọn ở Workflow Planning: chia theo bước của luồng (lát dọc), không chia theo service. Ranh giới service theo ADR-0029, giữ nguyên ở Application Design |
| Tổ chức code | Không | Không phải greenfield; monorepo và thư mục service đã có |

---

## Câu hỏi

## Question U-1 — Tách các CR con lớn

Workflow Planning đánh U2, U4, U7, U8 cỡ XL (lớn hơn khoảng hai CR thường). CR con lớn thì bản thiết kế và danh sách task dài, Creator duyệt khó, và một lần vào main đổi nhiều thứ cùng lúc. CR con nhỏ thì nhiều lượt duyệt hơn và nhiều lần chạy video thật hơn (tốn token).

Bảng tách AI đề xuất (giữ số cũ, thêm chữ cái, để thiết kế và ADR đã duyệt vẫn chỉ đúng CR con):

| CR con | Phạm vi | FR | Phụ thuộc | Cỡ |
|---|---|---|---|---|
| U1 | Ngôn ngữ hình: kho ngôn ngữ, 4 kit, API ảnh tĩnh `/v1/stills`, phát file chung ở api-gateway, ảnh mẫu, trang cài đặt, luật hình và nền theo ngôn ngữ; xoá dữ liệu cũ | FR-18, FR-15 (luật), FR-16 | — | L (giữ) |
| U2a | Luồng 3 giai đoạn: chia 13 bước hiện có vào Tiền kỳ, Sản xuất, Hậu kỳ; thanh bước, tiêu đề màn | FR-1 | — | M |
| U2b | Khung sửa theo đơn vị: lưu đơn vị, dấu đầu vào, làm lại kèm ghi chú, gom ghi chú, "sẽ làm lại", bộ thành phần đơn vị ở web-gui; áp lên bước chọn ở U-4 | FR-17, NFR-2 (đếm lần làm lại) | U2a | L |
| U3 | Kịch bản mới: logline, tóm tắt, beat, lời thoại theo cảnh; gợi ý kiểu video | FR-2, FR-19 | U2b | M (giữ) |
| U4a | Mỹ thuật lượt 1 phần thế giới: mô tả thế giới và bảng tham khảo (ảnh tải lên), bối cảnh nền 4 lớp ở hai khung hình, đạo cụ; phong cách theo thế giới (ADR-0037); bốn cách sửa hình (làm lại kèm ghi chú, sửa mô tả, tải SVG, thay từ thư viện) | FR-3 (mục 1, 2, 4), FR-5 | U1 + cổng, U2b, U3 | L |
| U4b | Nhân vật: nhân vật cấp video, model sheet, biểu cảm, tư thế tay, nhân vật trên từng nền chính; thư viện nhân vật kênh (trang cài đặt) | FR-3 (mục 3), FR-4 | U4a | L |
| U4c | Hình minh hoạ theo bộ mỹ thuật: chỉ vẽ phần còn thiếu; hình của bộ mỹ thuật hiện là đã duyệt; làm lại kèm ghi chú | FR-13 | U4b | M |
| U5 | Storyboard mới và lượt 2: hành trình máy quay, cảnh giải thích ý trong thế giới, color key và dải màu, kiểm tương phản, khung mẫu | FR-3 (mục 5, 6), FR-6, FR-7, FR-8 | U4b | L (giữ) |
| U6 | Giọng sơ bộ, sửa giọng từng câu, từ điển phát âm, animatic khung tĩnh | FR-9, FR-10, FR-20 | U5 | L (giữ) |
| U7a | Code theo cảnh, lỗi theo shot, bỏ lượt AI tự sửa (giữ autofix máy móc), xem nhanh từng shot | FR-22 | U5 | L, có NFR |
| U7b | Rig key pose, bộ VFX có tên | FR-11, FR-12 | U1 + cổng | M |
| U7c | Validate đúng chỗ, Review đủ thông tin | FR-23, FR-24 | U7a | M |
| U8a | Render từng cảnh có đoạn chồng và chỉnh màu theo color key; Merge theo cảnh có tiến độ theo chặng; QC chỗ nối và khung đen | FR-14, FR-26 | U5, U7a | L, có NFR |
| U8b | Dựng theo ghi chú mốc thời gian, đổi chuyển cảnh ở chỗ cắt | FR-25 | U8a | M |
| U9 | Thumbnail bằng Remotion, prompt AI ngoài theo style video; rà lại toàn bộ prompt | FR-21, FR-15 (phần còn lại) | U5 | M (giữ) |

A) Tách bốn CR con XL như bảng trên, giữ nguyên năm CR con còn lại: 15 CR con (AI đề xuất).
   - ✅ Ưu: không còn CR con XL; mỗi phần tách ra có ý nghĩa riêng và chạy được trọn luồng khi vào main; U7b chạy sớm song song với tiền kỳ.
   - ⚠️ Nhược: 15 lượt `/cr` (mỗi lượt hai lần duyệt) và 15 lần chạy video thật thay vì 9.

B) Tách thêm cả ba CR con cỡ L: U1 thành U1a (ảnh tĩnh và phát file; bản xem trước hình hiện có chuyển sang `/v1/stills`) và U1b (ngôn ngữ hình, kit, xoá dữ liệu); U5 thành U5a (storyboard) và U5b (color key, khung mẫu); U6 thành U6a (giọng, từ điển) và U6b (animatic). Tổng 18 CR con.
   - ✅ Ưu: mọi CR con cỡ M trở xuống.
   - ⚠️ Nhược: 18 vòng duyệt và 18 lần chạy video thật.

C) Giữ 9 CR con như Workflow Planning.
   - ✅ Ưu: ít vòng duyệt nhất.
   - ⚠️ Nhược: U2, U4, U7, U8 mỗi cái một bản thiết kế rất dài; một lần vào main đổi nhiều service cùng lúc, khó quay lui.

X) Other (please describe after [Answer]: tag below)

[Answer]: A

## Question U-2 — Project đang làm dở khi một CR con vào main

Dữ liệu cũ bị xoá khi U1 vào main (W-2 A). Từ đó tới khi U8 xong, mỗi lần một CR con vào main thì luồng đổi (thêm bước, đổi dạng storyboard). Một project tạo trước đó và đang ở giữa luồng có thể thiếu dữ liệu của bước mới (ví dụ chưa có logline khi U3 vào main).

A) Không viết code chuyển đổi cho project dở dang. Trước khi chạy `/deliver` một CR con, Creator làm xong hoặc xoá project đang dở. Project đã có video vẫn mở được Kết quả và Đăng video; mỗi CR con kiểm điều này trong test (AI đề xuất).
   - ✅ Ưu: không có code chỉ dùng một lần; sau U1 chỉ còn ít project vì dữ liệu cũ đã xoá.
   - ⚠️ Nhược: Creator phải dọn project dở trước mỗi lần một CR con vào main.

B) Mỗi CR con viết chuyển đổi cho project dở dang, điền dữ liệu cho bước mới (ví dụ AI viết bù logline khi mở lại project).
   - ✅ Ưu: không mất việc đang làm.
   - ⚠️ Nhược: mỗi CR con thêm code và test chuyển đổi; dữ liệu điền bù có thể lệch ý Creator.

C) Project ghim phiên bản luồng lúc tạo; project cũ đi hết theo luồng cũ.
   - ⚠️ Nhược: phải giữ code của luồng cũ chạy song song tới khi mọi project cũ xong.

X) Other (please describe after [Answer]: tag below)

[Answer]:A

## Question U-3 — Bốn kit ngôn ngữ hình làm sâu tới đâu

U1 dựng bốn kit (Sách tranh tươi, Khoa học phát sáng, Giấy cắt lớp, Đường nét tối giản) đủ để render ảnh mẫu, rồi Creator bật một (P-4 A). Các CR con sau thêm phần nặng vào kit: model sheet nhân vật (U4b), rig key pose và VFX (U7b), các composition ảnh tĩnh (U4a, U5, U6, U9). Kế hoạch đã duyệt ghi "các CR con sau viết prompt theo ngôn ngữ đã bật", nhưng chưa nói ba kit còn lại có được làm đủ không.

A) U1 làm cả bốn kit ở mức dựng được ảnh mẫu (rig cơ sở, hình nền tảng, luật hình và nền). Từ U4 trở đi, phần nặng chỉ làm cho kit của ngôn ngữ đã bật. Ba ngôn ngữ còn lại ghi "chỉ có ảnh mẫu" và không bật được cho tới khi một CR riêng làm đủ kit đó. Bản sao Creator tạo từ ngôn ngữ đã bật thì bật được, vì dùng chung kit (AI đề xuất).
   - ✅ Ưu: phần nặng làm một lần thay vì bốn lần; tiêu chí chấp nhận 8 (đổi ngôn ngữ không đổi video đã tạo) vẫn kiểm được bằng bản sao.
   - ⚠️ Nhược: muốn chuyển hẳn sang ngôn ngữ khác về sau thì cần một CR.

B) Mọi CR con làm cho cả bốn kit.
   - ✅ Ưu: bật ngôn ngữ nào cũng dùng được ngay.
   - ⚠️ Nhược: U4b, U7b và mọi composition ảnh tĩnh nặng gấp khoảng bốn lần; ba kit không dùng vẫn phải bảo trì (nhược điểm đã ghi trong ADR-0036).

C) Sau cổng, xoá ba ngôn ngữ không được chọn.
   - ⚠️ Nhược: mất bộ so sánh; trái FR-18 (bốn ngôn ngữ hệ thống).

X) Other (please describe after [Answer]: tag below)

[Answer]: vậy ưu tiên làm khoa học phát sáng trước nhưng làm detail các phần nặng luôn. 

## Question U-4 — Khung sửa theo đơn vị áp lên bước nào trước

Kế hoạch đã duyệt ghi khung đơn vị "áp ngay cho Kịch bản và Storyboard hiện có". Khi xét kỹ: U3 viết lại Kịch bản ngay sau U2 (logline, beat, lời thoại theo cảnh), và U5 viết lại Storyboard. Phần áp khung lên Kịch bản hiện tại sẽ bị U3 thay gần như ngay.

A) Khung đơn vị áp lên Storyboard hiện có, vốn đã chia theo cảnh từ CR-050: mỗi cảnh một đơn vị `board_scene` có duyệt, ghi chú, làm lại kèm ghi chú, sửa tay; phần phụ thuộc đang có của cảnh đó được đánh "cần làm lại" (chi tiết ở Functional Design). Kịch bản vào khung ở U3, khi được viết lại (AI đề xuất).
   - ✅ Ưu: khung chạy thật trên luồng ngay khi vào main; không làm phần sẽ bị bỏ.
   - ⚠️ Nhược: Kịch bản giữ cách duyệt cũ thêm một CR con.

B) Như kế hoạch: áp cả Kịch bản hiện có và Storyboard.
   - ⚠️ Nhược: phần Kịch bản làm ở U2 bị U3 viết lại ngay sau.

C) Chỉ dựng khung và test, chưa áp lên bước nào; U3 là bước đầu tiên dùng khung.
   - ⚠️ Nhược: khung vào main mà chưa chạy trên luồng thật; tiêu chí "chạy trọn luồng" không kiểm được gì của khung.

X) Other (please describe after [Answer]: tag below)

[Answer]: ưu tiên làm theo kế hoạch đã duyệt hơn codebase hiện tại

## Question U-5 — Bao nhiêu CR con mở cùng lúc

Mỗi CR con một nhánh và một worktree, nên nhiều agent chạy cùng lúc được. Creator duyệt mọi thiết kế, danh sách task và kết quả, nên việc duyệt là chỗ nghẽn. Nhiều CR con cùng đụng authoring-service và web-gui.

Chỗ đồ thị cho phép song song: U1 cạnh U2a, U2b, U3; U7b sau cổng, cạnh U4a..U6; U4c cạnh U5; U6 cạnh U7a; U9 cạnh U6..U8.

A) Tối đa hai CR con mở cùng lúc: một trên đường chính (U2a → U2b → U3 → U4a → U4b → U5 → …) và một ở nhánh phụ (U1, rồi U7b, U4c, U9 khi tới lượt). Khi mở CR con, `/cr` báo chỗ hai CR con đụng cùng file (AI đề xuất).
   - ✅ Ưu: Creator không phải duyệt quá hai thiết kế một lúc; U1 và cổng chọn ngôn ngữ không chặn đường chính.
   - ⚠️ Nhược: chậm hơn so với mở hết mọi chỗ song song.

B) Tuần tự hoàn toàn, mỗi lúc một CR con.
   - ✅ Ưu: không bao giờ trùng file.
   - ⚠️ Nhược: lâu nhất; thời gian Creator so ảnh mẫu chặn cả luồng.

C) Mở mọi chỗ đồ thị cho phép.
   - ⚠️ Nhược: có lúc ba, bốn nhánh cùng sửa authoring-service và web-gui; Creator duyệt nhiều thứ cùng lúc.

X) Other (please describe after [Answer]: tag below)

[Answer]: B

---

## Rà câu trả lời vòng 1

| Câu | Trả lời | Rà |
|---|---|---|
| U-1 | A | Rõ: 15 CR con theo bảng ở U-1 (bảng có thể đổi theo U-4.1) |
| U-2 | A | Rõ: không có code chuyển đổi cho project dở; Creator làm xong hoặc xoá project dở trước mỗi lần `/deliver` |
| U-3 | "vậy ưu tiên làm khoa học phát sáng trước nhưng làm detail các phần nặng luôn" | Hiểu được nhiều cách: có còn cổng so ảnh mẫu không; "phần nặng" làm cho riêng Khoa học phát sáng hay cho cả bốn; làm ở U1 hay ở các CR con sau. Hỏi lại ở U-3.1, U-3.2 |
| U-4 | "ưu tiên làm theo kế hoạch đã duyệt hơn codebase hiện tại" | Có thể là đúng chữ kế hoạch (áp khung lên Kịch bản và Storyboard hiện có, tức U-4 B), hoặc là bám đích của luồng mới, không áp lên bước cũ nào. Hỏi lại ở U-4.1 |
| U-5 | B | Rõ: mỗi lúc một CR con |

**Thứ tự khi làm tuần tự (U-5 B), AI suy ra:** U1 → U2a → U2b → U3 → U4a → U4b → U4c → U5 → U6 → U7a → U7b → U7c → U8a → U8b → U9. U1 làm đầu tiên vì xoá dữ liệu cũ đi cùng U1 (W-2 A). Nếu giữ cổng so ảnh mẫu (U-3.1 B), Creator so trong lúc U2a..U3 chạy, nên cổng không chặn luồng. Creator thấy cần đổi thứ tự thì ghi vào đây.

## Câu hỏi làm rõ

## Question U-3.1 — Còn cổng so ảnh mẫu không

Kế hoạch cũ (P-4 A): U1 dựng ảnh mẫu của cả bốn ngôn ngữ, Creator so rồi bật một; CR con vẽ đầu tiên (U4a) chờ tới lúc đó.

A) Khoa học phát sáng là ngôn ngữ được bật khi U1 vào main; bỏ cổng chờ Creator so. U1 vẫn dựng ảnh mẫu của cả bốn để Creator xem và so về sau (AI hiểu từ "ưu tiên làm khoa học phát sáng trước").
   - ✅ Ưu: không có bước chờ giữa U1 và U4a.
   - ⚠️ Nhược: không còn bước chính thức xem cả bốn trước khi chốt.

B) Giữ cổng. Creator xem ảnh mẫu cả bốn rồi mới bật chính thức; "ưu tiên" chỉ là trong U1 làm kit Khoa học phát sáng trước.

X) Other (please describe after [Answer]: tag below)

[Answer]:A

## Question U-3.2 — "Làm detail các phần nặng" cho ngôn ngữ nào, lúc nào

Phần nặng là: model sheet, biểu cảm, tư thế nhân vật (U4b); bối cảnh, đạo cụ, bảng tham khảo ở dạng ảnh tĩnh (U4a); rig key pose và bộ VFX (U7b); composition ảnh tĩnh của color key, khung mẫu (U5), animatic (U6), thumbnail (U9).

A) Phần nặng làm đầy đủ chi tiết, chỉ cho Khoa học phát sáng, ở đúng CR con của nó. Ba ngôn ngữ còn lại dừng ở ảnh mẫu; làm đủ ở CR riêng về sau. Đây là U-3 A với ngôn ngữ đã chọn.
   - ✅ Ưu: tới video đầu tiên theo luồng mới nhanh nhất.
   - ⚠️ Nhược: ba ngôn ngữ còn lại chưa dùng cho video được.

B) Phần nặng làm cho cả bốn ngôn ngữ, Khoa học phát sáng trước: mỗi CR con làm cho Khoa học phát sáng; thêm một CR con cuối (U10) làm phần nặng cho ba ngôn ngữ còn lại, sau khi luồng mới chạy xong. Nếu ý của Creator là cả bốn đều phải làm đủ, AI đề xuất cách này.
   - ✅ Ưu: cuối cùng bật ngôn ngữ nào cũng dùng được; luồng mới không bị chậm lại.
   - ⚠️ Nhược: thêm một CR con cỡ L–XL; U10 phải đi lại mọi composition và rig cho ba kit.

C) Phần nặng làm cho cả bốn ngay trong từng CR con, Khoa học phát sáng làm trước trong mỗi CR con.
   - ✅ Ưu: không có CR con dồn cuối.
   - ⚠️ Nhược: mỗi CR con nặng gần gấp bốn; ba kit chưa dùng phải bảo trì ngay từ đầu.

D) Dồn phần nặng của Khoa học phát sáng lên U1: rig nhân vật đủ biểu cảm và tư thế, key pose, bộ VFX làm ngay trong U1, để ảnh mẫu đã ở chất lượng cuối; U7b chỉ còn đưa key pose và VFX vào prompt của Kỹ sư. Ba ngôn ngữ còn lại ở mức ảnh mẫu.
   - ✅ Ưu: thấy chất lượng cuối của Khoa học phát sáng sớm nhất.
   - ⚠️ Nhược: U1 thành XL; nhân vật thiết kế trước khi có bộ mỹ thuật (U4a, U4b) nên có thể phải sửa lại.

X) Other (please describe after [Answer]: tag below)

[Answer]:A

## Question U-4.1 — "Theo kế hoạch đã duyệt" nghĩa là gì ở U2b

A) Đúng chữ kế hoạch đã duyệt: U2b áp khung đơn vị lên cả Kịch bản và Storyboard hiện có (U-4 B). U3 sau đó chuyển Kịch bản mới vào khung; phần Kịch bản cũ trong khung được thay (AI hiểu từ câu trả lời).
   - ✅ Ưu: khung chạy thật trên hai bước ngay khi U2b vào main.
   - ⚠️ Nhược: phần Kịch bản làm ở U2b bị U3 thay ngay sau.

B) Bám đích của luồng mới, không áp khung lên bước cũ nào: gộp U2b vào U3, để khung chạy thật ngay trên Kịch bản mới; Storyboard vào khung ở U5. Bảng còn 14 CR con.
   - ✅ Ưu: không làm phần sẽ bị thay; khung vẫn chạy thật khi vào main.
   - ⚠️ Nhược: U3 thành cỡ L–XL.

C) U2b chỉ dựng khung và test (U-4 C); U3 là bước đầu tiên dùng khung.
   - ⚠️ Nhược: khung vào main mà chưa chạy trên luồng thật.

X) Other (please describe after [Answer]: tag below)

[Answer]: A
