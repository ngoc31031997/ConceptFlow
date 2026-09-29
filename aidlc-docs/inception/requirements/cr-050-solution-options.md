# CR-050 — Thiết kế lại bước Storyboard → Code: tiết kiệm token, chạy lại được từng phần, giữ chất lượng

## Date
2026-09-29

## Trạng thái
Đề xuất để Creator chọn hướng (INCEPTION — Requirements Analysis). Chưa có code. Mọi con số "dự kiến" bên dưới là **giả thuyết cần đo** (xem Pha 0), không phải cam kết.

## 1. Token thực sự đi đâu (số liệu thật từ `llm_usage`)

Chi phí mỗi video, 7 project có bước Code chạy thật:

| Project | Story (out) | Storyboard (out) | **Code (out)** | Code: trong đó suy nghĩ | Code: số lượt gọi | Code: thời gian model |
|---|---|---|---|---|---|---|
| f7103848 (52 shot) | 30k | 56k | **1 277k** | 1 180k (92%) | 62 | 146 phút |
| 2e14928a | 15k | 22k | **1 034k** | 971k (94%) | 57 | 127 phút |
| a070963d | 38k | 69k | **956k** | 851k (89%) | 43 | 113 phút |
| d5fc2fbc | 17k | 26k | **739k** | 640k (87%) | 36 | 84 phút |
| 3035c81f | 16k | 28k | **631k** | 552k (88%) | 28 | 67 phút |

Nhận xét:
1. **Bước Code chiếm khoảng 95% token và gần như toàn bộ thời gian chờ.** Story và Storyboard gộp lại chỉ 40–100k token.
2. **Khoảng 90% đầu ra của bước Code là "suy nghĩ", không phải code.** Một chunk 3 shot trung bình ra 28k token, trong đó 24,6k là suy nghĩ và chỉ khoảng 3,4k là code. Với glm là 92,5k suy nghĩ trên 97,9k.
3. Kết quả cuối của video f7103848 là khoảng 126 KB code shot (khoảng 40k token), nhưng đã tốn **1,28 triệu token** đầu ra, tức khoảng **30 lần** lượng code thật.
4. **Số lượt sửa lỗi (repair) nhiều hơn số chunk** (149 so với 125). Code AI viết lần đầu thường không qua được kiểm biên dịch hoặc kiểm bố cục.
5. Những lỗi làm ngắt bước (`empty`, `budget`, stream đóng rỗng) đều xảy ra trong **lượt suy nghĩ rất dài**. Suy nghĩ càng dài thì càng dễ bị ngắt.

**Vì sao model phải suy nghĩ nhiều như vậy?** Storyboard mô tả hình bằng **văn xuôi**, ví dụ: *"nhân vật đi trên vỉa hè gần trung tâm khung; phía trên đầu nở bong bóng suy nghĩ chứa chiếc xe trắng kèm dấu kiểm…"*. Engineer phải tự quy đổi đoạn văn đó thành toạ độ pixel, cỡ, thời điểm và đường cong chuyển động, **đồng thời** viết TSX đúng cú pháp và đúng API của bộ kit, rồi tự kiểm vùng an toàn và dải phụ đề. Nhiều việc khó dồn vào một lượt, nên model suy nghĩ rất lâu.

**Code sinh ra lại rất "máy móc".** Đo trên 52 shot của f7103848:
- Mỗi shot là: một `Backdrop`, các component của bộ kit/thư viện (`Person` ×61, `Table`, `Car`, `Bubble`, `Hand`…) đặt theo `x/y/size`, và các chuyển động dựng từ `interpolate` (228 lần), `spring` (24), `interpolateColors` (18), easing.
- Chỉ **4/52 shot** tự vẽ SVG. 41 shot có `div` có style, chủ yếu là nhãn chữ và hiệu ứng chuyển cảnh.

Tức là phần lớn công việc là **sắp xếp và định thời các khối có sẵn**. Việc này mô tả được bằng dữ liệu, không nhất thiết phải là code tự do.

## 2. Các hướng giải pháp

### Hướng A — Giữ kiến trúc, tối ưu từng chỗ
- Lưu kết quả từng đoạn vào DB; chạy lại đúng đoạn lỗi, hoặc dùng AI ngoài cho đoạn đó (đúng câu trả lời vòng 1).
- Đoạn nhỏ hơn (2 shot), đặt phần prompt cố định lên đầu để provider cache được nhiều hơn.
- Giảm trần suy nghĩ cho chunk (hiện 60 000 ký tự), hoặc dùng chế độ ít suy nghĩ nếu model/Hive hỗ trợ (cần kiểm tra).

✅ Ít rủi ro, không đổi chất lượng hình. ⚠️ Không chạm vào gốc: model vẫn phải tự suy ra toạ độ và code từ văn xuôi. Tiết kiệm dự kiến khoảng 20–40%, vẫn cần vòng sửa lỗi biên dịch.

### Hướng B — **Shot Spec + trình biên dịch cố định** (khuyến nghị)
AI **không viết TSX nữa**. Với mỗi shot, AI viết một **bản mô tả dạng dữ liệu (JSON)** theo một bộ từ vựng cố định. Một **trình biên dịch viết tay, không dùng AI**, chuyển JSON đó thành code Remotion.

```
Story --> Storyboard (van xuoi, theo canh) --> Shot Spec (JSON, tung shot/canh) --> Trinh bien dich --> TSX --> Render
            AI, chia theo canh                  AI, chia nho, song song               khong AI, luon dung cu phap
```

Ví dụ Shot 2.2 hiện là khoảng 30 dòng TSX. Dạng Shot Spec:

```json
{
  "id": "2.2",
  "backdrop": {"color": "xanhThanhPho", "floor": "trangNga", "floorY": 0.77},
  "elements": [
    {"id": "sun",  "kit": "Sun",    "at": [0.11, 0.18], "size": 0.07},
    {"id": "car1", "kit": "Car",    "at": [0.21, 0.86], "size": 0.10, "props": {"color": "xamBiLoc"}},
    {"id": "car2", "kit": "Car",    "at": [0.43, 0.86], "size": 0.10, "props": {"color": "xamBiLoc"}},
    {"id": "hero", "kit": "Person", "at": [0.43, 0.59], "size": 0.24,
     "props": {"pose": "stand", "mood": "surprised", "shirt": "vangKhaoKhat"}}
  ],
  "motion": [
    {"target": "sun",  "move_to": [0.47, 0.18], "from": 0.0,  "to": 0.30, "ease": "inOut"},
    {"target": "car2", "color_to": "trangNga",  "from": 0.15, "to": 0.28},
    {"target": "car1", "color_to": "trangNga",  "from": 0.68, "to": 0.81},
    {"target": "hero", "rotate_to": 6,          "from": 0.20, "to": 0.55, "ease": "inOut"}
  ]
}
```

- **Từ vựng cố định**:
  - Mọi component của bộ kit và thư viện (CR-044), dùng đúng props của chúng.
  - Toạ độ tương đối 0–1 của khung.
  - Chuyển động: move, scale, rotate, fade, đổi màu, pop, slide, vẽ dần, rung, nhịp, đếm số, gõ chữ.
  - Nhãn chữ; máy quay (lia, zoom); chuyển cảnh.
- **Kiểm trước khi render, không cần AI**: JSON Schema, component/prop có thật, màu nằm trong PALETTE, vùng an toàn, dải phụ đề, cỡ tối thiểu của vật chính (CR-047 F14). Nhiều lỗi được **tự sửa cố định** (ví dụ kéo vật vào vùng an toàn). Chỉ khi thật cần mới gửi lại cho AI một thông báo lỗi ngắn và chính xác.
- **Code luôn biên dịch được**, vì do trình biên dịch sinh ra. Vòng "kiểm biên dịch → sửa" hiện chiếm 149 lượt thì gần như biến mất.
- **Lối thoát cho hình đặc biệt** (khoảng 8% shot, như 4 shot tự vẽ SVG): một phần tử `"custom"` cho AI viết TSX **chỉ cho phần tử đó**, và đi qua đường kiểm/sửa hiện có.
- **Chạy lại từng phần là tự nhiên**: mỗi shot hoặc cảnh là một bản ghi JSON nhỏ trong DB. Chạy lại đúng shot lỗi; "AI ngoài" chỉ cần dán JSON của một shot; Creator có thể **sửa tay JSON** (đổi vị trí, thời điểm) mà không cần biết code.

✅ Nhắm đúng gốc: bỏ phần khó nhất (toạ độ pixel và cú pháp) khỏi AI, nên ít suy nghĩ hơn, lượt ngắn hơn, ít bị ngắt hơn. Prompt hệ thống ngắn hơn nhiều (chỉ còn schema và từ vựng, thay cho 28 500 ký tự luật TSX).
- Dự kiến (**cần đo**): đầu ra mỗi shot từ khoảng 23k token (suy nghĩ + code) xuống vài nghìn; cả video từ khoảng 0,6–1,3 triệu token xuống khoảng 0,1–0,3 triệu; thời gian từ 1–2,5 giờ xuống vài chục phút.

⚠️ Đánh đổi:
- Chất lượng hình **bị giới hạn bởi từ vựng**: chuyển động quá lạ phải dùng lối thoát `custom`.
- Đây là việc lớn: một trình biên dịch mới, schema, prompt mới, và UI xem/sửa shot.
- Cần một ADR mới (đổi contract giữa `authoring-service` và `llm-service`, thêm một định dạng dữ liệu mới).
- Chỉ áp cho **Remotion**. Manim giữ đường hiện tại, trừ khi làm thêm.

### Hướng C — Hướng B, nhưng gộp Storyboard và Shot Spec
Đạo diễn viết **cả văn xuôi lẫn Shot Spec** trong cùng một lượt, mỗi lượt một cảnh, các cảnh chạy song song. Bỏ hẳn một lượt AI đọc lại văn xuôi.

✅ Ít lượt gọi nhất; hình được nghĩ đúng một lần. ⚠️ Mỗi lượt của đạo diễn nặng hơn, có thể làm giảm chất lượng phần sáng tạo (lời thoại, ý hình). Khó so sánh với hiện tại vì đổi hai bước cùng lúc. Có thể làm sau B khi đã có số liệu.

## 3. Lộ trình đề xuất (nếu chọn B)
- **Pha 0 — Spike đo, không đụng luồng chính**:
  - Viết schema cùng trình biên dịch cho một tập con từ vựng, đủ cho storyboard f7103848.
  - Cho AI viết Shot Spec cho 52 shot đó, render thử.
  - So với video hiện tại về token, thời gian, tỉ lệ lỗi, và **chất lượng hình, do Creator xem và chấm**.
  - Kết quả quyết định có đi tiếp hay không.
- **Pha 1 — Nền chạy lại từng phần** (cần cho mọi hướng):
  - Bảng lưu từng đoạn (shot/cảnh) trong DB của `authoring-service`, chỉ lưu kết quả cuối, ghi đè.
  - API chạy lại một đoạn, API dán kết quả từ AI ngoài.
  - UI danh sách đoạn (câu C4).
  - Áp cho Storyboard (theo cảnh) và Code.
- **Pha 2 — Shot Spec vào luồng chính cho Remotion**: đường TSX hiện tại giữ lại làm lựa chọn hoặc dự phòng trong một thời gian.
- **Pha 3 (tuỳ chọn)**: Hướng C, hoặc mở rộng cho Manim.

## 4. Ảnh hưởng tới các câu hỏi đang chờ
- C3 (số shot mỗi đoạn) **không còn cần** nếu chọn B: đơn vị tự nhiên là một cảnh, hoặc một shot khi chạy lại.
- C1, C1b, C2, C4 vẫn áp dụng; chỉ đổi "code của đoạn" thành "Shot Spec của đoạn".
