# CR-050 — Câu hỏi làm rõ (vòng 2)

Câu trả lời vòng 1: `cr-050-requirement-questions.md`. Những điểm đã chốt:
- Không tự thử lại. Khi lỗi, Creator tự chạy lại trên giao diện.
- Một đoạn lỗi thì các đoạn khác vẫn chạy tiếp. Đoạn lỗi chạy riêng được: bấm chạy lại bằng AI, hoặc dùng AI ngoài (sao chép prompt của đúng đoạn đó, dán kết quả về, giống chế độ "Tự làm với ChatGPT, Claude…" hiện có).
- Các đoạn đã xong được lưu trong DB.
- Mỗi đoạn chỉ lưu một kết quả cuối cùng. Lượt sửa lỗi biên dịch ghi đè lên kết quả đó.

## Câu D1 (quyết định trước các câu còn lại)
Chọn hướng cho bước Storyboard → Code. Phân tích đầy đủ và số liệu: `cr-050-solution-options.md`.

A) Hướng A — giữ kiến trúc hiện tại: lưu từng đoạn, chạy lại đoạn lỗi, đoạn nhỏ hơn, giảm trần suy nghĩ.
   - ✅ Ưu: ít rủi ro, không đổi chất lượng hình.
   - ⚠️ Đánh đổi: tiết kiệm khoảng 20–40%; AI vẫn suy nghĩ khoảng 90% đầu ra và vẫn có vòng sửa lỗi biên dịch.

B) Hướng B — Shot Spec (JSON) + trình biên dịch cố định; làm Pha 0 (spike đo trên video f7103848) trước, rồi mới quyết định đưa vào luồng chính (khuyến nghị).
   - ✅ Ưu: nhắm đúng gốc tốn token; code luôn biên dịch được; chạy lại và sửa tay từng shot dễ.
   - ⚠️ Đánh đổi: việc lớn và cần ADR mới; chất lượng bị giới hạn bởi từ vựng (có lối thoát `custom`); chỉ áp cho Remotion.

C) Hướng C — như B, nhưng đạo diễn viết luôn Shot Spec theo từng cảnh (gộp Storyboard với Code).
   - ✅ Ưu: ít lượt gọi nhất.
   - ⚠️ Đánh đổi: rủi ro chất lượng phần sáng tạo; đổi hai bước cùng lúc nên khó so sánh.

D) Làm A (Pha 1: lưu và chạy lại từng đoạn) ngay, đồng thời làm spike B song song.
   - ✅ Ưu: giảm đau ngay mà vẫn tìm giải pháp gốc.
   - ⚠️ Đánh đổi: phần UI và bảng lưu của A cần thiết kế sao cho dùng lại được cho B.

X) Other (please describe after [Answer]: tag below)

[Answer]: A → đổi trong chat: "d1 làm shot spec luôn" (làm cả A và Shot Spec)

## Câu R1 (từ review toàn luồng — `cr-050-pipeline-review.md`)
Hiện nay, khi chạy lại Story/Visual mà lượt AI **lỗi**, các bước sau (storyboard, code) bị xoá, dù nội dung của bước đó không đổi (hành vi có chủ ý từ commit `15a8111`). Giữ hay đổi?

A) Đổi: chỉ xoá các bước sau khi **lưu nội dung mới khác nội dung cũ**; lượt lỗi không xoá gì (khuyến nghị).
   - ✅ Ưu: một lỗi Hive không xoá mất code đáng khoảng 1 triệu token.
   - ⚠️ Đánh đổi: sau một lượt lỗi, các bước sau vẫn là kết quả cũ (và vẫn khớp với nội dung cũ, vì nội dung đó không đổi).

B) Giữ như hiện nay.
   - ✅ Ưu: không đổi hành vi quen thuộc.
   - ⚠️ Đánh đổi: có thể mất các bước sau vì một lỗi thoáng qua.

X) Other (please describe after [Answer]: tag below)

[Answer]: A

## Câu R2
Các lỗi rõ ràng từ review (I1: danh sách hình không lập lại khi storyboard đổi; C5: rào chắn chọn model cho bước Code; X1 nếu R1 = A) làm ở đâu?

A) Tách thành các bản sửa nhỏ, làm ngay (mỗi cái một nhánh `fix/…` theo /fix-bug), trước phần lớn của CR-050 (khuyến nghị).
   - ✅ Ưu: chặn mất dữ liệu và sai hình ngay; review và merge nhanh.
   - ⚠️ Đánh đổi: thêm vài nhánh cần theo dõi.

B) Gộp vào CR-050.
   - ✅ Ưu: một chỗ theo dõi.
   - ⚠️ Đánh đổi: phải chờ CR-050 (việc lớn) xong mới có.

X) Other (please describe after [Answer]: tag below)

[Answer]: B

## Câu C1
Anh/chị trả lời câu 1 là "cả storyboard/các lượt gọi đơn". Các lượt gọi đơn là **Story**, **gợi ý metadata** và **gợi ý kịch bản ngắn**. Các lượt này nên đổi thế nào?

A) Chỉ chia nhỏ **Storyboard** (mỗi cảnh một đoạn) và **Code** (mỗi nhóm shot một đoạn). Story và các gợi ý vẫn là một lượt gọi, vì chúng ngắn và gần như không lỗi (Story 8/8 ok). Khi lỗi thì hiện rõ lý do, kèm nút chạy lại và nút dùng AI ngoài (khuyến nghị).
   - ✅ Ưu: chia nhỏ đúng chỗ có ích; Story giữ mạch kể liền một khối.
   - ⚠️ Đánh đổi: một lượt Story lỗi vẫn phải chạy lại cả Story (khoảng 2 phút).

B) Chia nhỏ cả **Story** theo beat, như Storyboard.
   - ✅ Ưu: mọi bước đều chạy lại được theo phần.
   - ⚠️ Đánh đổi: các beat viết riêng dễ lặp ý và mất mạch. CR-047 vừa phải sửa lỗi lặp kết luận giữa các beat.

X) Other (please describe after [Answer]: tag below)

[Answer]: A

## Câu C1b
Chia Storyboard theo cảnh nghĩa là mỗi cảnh (beat) của Story do một lượt gọi riêng viết các shot của cảnh đó, rồi hệ thống ghép lại và kiểm JSON như hiện nay. Làm cách nào để các cảnh vẫn nhất quán (nhân vật, màu, bố cục, nhịp)?

A) Hai bước: một lượt ngắn viết **khung chung** trước (nhân vật, bố cục, quy ước hình dùng lại), sau đó các cảnh viết song song dựa trên khung đó (khuyến nghị). Bước Code đang làm đúng như vậy với LAYOUT.
   - ✅ Ưu: các cảnh bám cùng một khung; khung lỗi thì chỉ chạy lại khung.
   - ⚠️ Đánh đổi: thêm một lượt gọi; phải viết prompt mới cho khung và cho từng cảnh.

B) Mỗi cảnh chỉ nhận toàn bộ Story cùng cảnh liền trước và liền sau, không có khung chung.
   - ✅ Ưu: đơn giản hơn.
   - ⚠️ Đánh đổi: dễ lệch nhân vật và bố cục giữa các cảnh.

X) Other (please describe after [Answer]: tag below)

[Answer]: A — "c1b có"

## Câu C2
Anh/chị trả lời câu 5 là "đổi prompt". Tôi hiểu như sau, anh/chị xác nhận giúp:

A) Các đoạn đã xong **bị bỏ và phải sinh lại** khi prompt của bước đó đổi (ví dụ đổi prompt Engineer đang dùng, hoặc sửa nội dung prompt), hoặc khi đầu vào của bước đổi (Story đổi thì Storyboard sinh lại; Storyboard đổi thì Code sinh lại). **Đổi model thì vẫn dùng lại** các đoạn đã xong.

B) Chỉ khi prompt đổi mới bỏ các đoạn đã xong. Đầu vào đổi (ví dụ sửa Storyboard) thì vẫn giữ các đoạn đã có. ⚠️ Nếu làm vậy, code có thể không khớp storyboard mới.

X) Other (please describe after [Answer]: tag below)

[Answer]: B — "c2 chỉ khi promt đổi"

## Câu C3
**"Kích thước đoạn" là gì**: bước Code không viết cả video trong một lượt. Hệ thống chia danh sách shot của storyboard thành từng nhóm, mỗi nhóm là một **đoạn**, và mỗi đoạn là một lượt gọi AI.

Ví dụ: video có 30 shot.
- 3 shot mỗi đoạn (hiện nay): 10 đoạn. Đoạn 4 lỗi thì chỉ phải chạy lại shot 10–12. Mỗi đoạn mất khoảng 4 phút, AI viết khoảng 38k token.
- 2 shot mỗi đoạn: 15 đoạn. Mỗi đoạn ngắn hơn (khoảng 2–3 phút), ít bị ngắt hơn, lỗi thì chạy lại ít hơn. Tổng chi phí phần đầu vào tăng khoảng 1,5 lần, vì mỗi đoạn đều phải gửi kèm luật và storyboard (khoảng 20k token).
- 1 shot mỗi đoạn: 30 đoạn. Chạy lại chính xác nhất. Chi phí đầu vào tăng khoảng 3 lần, và khó giữ chuyển cảnh mượt giữa các shot.

Chọn số shot mỗi đoạn:

A) Giữ 3 shot.

B) 2 shot (khuyến nghị).

C) 1 shot.

D) Để thành cài đặt Creator chỉnh được trong giao diện (mặc định 2 hoặc 3).

X) Other (please describe after [Answer]: tag below)

[Answer]: D

## Câu C3b
C3 = D: số shot mỗi đoạn là cài đặt Creator chỉnh được. Giá trị mặc định là bao nhiêu?

A) 3 — giữ như hiện nay, không đổi chi phí hay hành vi của project cũ.

B) 2 — lượt ngắn hơn, ít bị ngắt hơn; chi phí đầu vào tăng khoảng 1,5 lần.

X) Other (please describe after [Answer]: tag below)

[Answer]: A — "c3b mặc định là 3"

## Câu C4
**Giao diện bước Code hiện nay**: một thanh chạy AI, một dòng tiến độ "Đang viết code: 4/10 phần · 3:12", và ô soạn thảo code. Khi lỗi thì chỉ có một thông báo lỗi và nút chạy lại cả bước.

**Đề xuất mới.** Bản vẽ dưới đây cho bước Code; Storyboard dùng cùng mẫu, mỗi dòng là một cảnh:

```
+--------------------------------------------------------------------+
| Code - 10 doan (3 shot/doan)            8 xong - 1 loi - 1 cho     |
+--------------------------------------------------------------------+
| #1  shot 1.1-1.3   [XONG]                              [Xem]       |
| #2  shot 1.4-2.2   [XONG]                              [Xem]       |
| #3  shot 2.3-2.5   [LOI] model tra ve rong                         |
|                    [Chay lai bang AI] [AI ngoai: Sao chep prompt]  |
|                    [Dan ket qua...]                                |
| #4  shot 3.1-3.3   [DANG CHAY] 1:40                                |
| ...                                                                |
+--------------------------------------------------------------------+
| [Chay cac doan con thieu]                  [Sinh lai toan bo]      |
+--------------------------------------------------------------------+
| (Khi du doan: ghep - kiem bien dich - sua loi, roi hien code)      |
+--------------------------------------------------------------------+
```

- "Chạy các đoạn còn thiếu": chỉ chạy những đoạn đang lỗi hoặc chưa chạy.
- "Sinh lại toàn bộ": bỏ mọi đoạn đã lưu, sinh lại từ đầu (có hỏi xác nhận).
- "AI ngoài": sao chép prompt của **đúng đoạn đó**, anh/chị dán vào ChatGPT/Claude/Gemini, rồi dán kết quả về ô "Dán kết quả". Hệ thống kiểm định dạng (đủ các shot của đoạn) rồi lưu như đoạn đã xong.
- Khi mọi đoạn xong, hệ thống tự ghép, kiểm biên dịch và sửa lỗi như hiện nay.

Chọn cách hiển thị:

A) Như bản vẽ: liệt kê mọi đoạn, mỗi đoạn có trạng thái và nút riêng (khuyến nghị).
   - ✅ Ưu: thấy toàn cảnh, thao tác từng đoạn ngay tại chỗ.
   - ⚠️ Đánh đổi: danh sách dài khi video có nhiều đoạn (có thể thu gọn các đoạn đã xong).

B) Gọn: chỉ một dòng tổng ("8 xong - 1 lỗi - 1 chờ") và chỉ liệt kê các **đoạn lỗi** kèm nút; đoạn xong ẩn đi.
   - ✅ Ưu: gọn, ít rối.
   - ⚠️ Đánh đổi: không xem nhanh được từng đoạn đã xong.

X) Other (please describe after [Answer]: tag below)

[Answer]: A
