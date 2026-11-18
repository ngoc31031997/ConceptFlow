# CR-115 · Seed nháp: thư viện Nhịp và Chuyển cảnh

Bản nháp để Creator duyệt trước khi đưa vào code. Số giây là **gợi ý trong dữ liệu**, không phải luật trong code; Creator sửa được ở màn thư viện.
Ví dụ phim chỉ ghi cảnh mình chắc chắn; ô `—` là không có ví dụ chắc chắn (không bịa).

## A. Nhịp (`pacing_patterns`)

| id | name | summary | when_to_use | avoid_when | shot_length | motion | audio_relation | film_examples |
|---|---|---|---|---|---|---|---|---|
| P01 | Hé lộ chậm (slow reveal) | Giấu chủ thể, mở dần từng phần | Mở đầu, giới thiệu đối tượng lớn hoặc bí ẩn | Khán giả đã biết đối tượng | dài, 6–12 s | camera trôi chậm, lộ dần | lời ít, nhạc nền nền | Jaws (1975): cá mập gần như không lộ diện nửa đầu phim |
| P02 | Giải thích đều (steady) | Mỗi hình một ý, đổi hình theo câu | Đoạn giải thích cơ chế chính | Cao trào, cần cảm xúc | vừa, 3–6 s | chuyển động nhẹ, có chủ đích | hình theo sát lời | — |
| P03 | Dồn nhịp (quick build) | Hình ngắn dần, tăng tốc về một điểm | Dẫn tới phát hiện, kết luận | Nội dung cần đọc kỹ | ngắn dần, 4 → 1 s | tăng tốc | nhạc tăng, lời dồn | Rocky (1976): đoạn tập luyện |
| P04 | Đối chiếu (comparison) | Hai thứ cạnh nhau hoặc luân phiên | So sánh hai góc nhìn, trước–sau | Chỉ có một đối tượng | vừa, 3–5 s mỗi bên | đối xứng, nhịp đều hai bên | lời nêu điểm khác | The Godfather (1972): cắt đan lễ rửa tội và các vụ ám sát |
| P05 | Lặng để nhấn (pause emphasis) | Dừng lời, giữ hình | Ngay sau một ý quan trọng hoặc gây sốc | Đoạn đã chậm sẵn | dài, 2–4 s im | gần đứng yên | khoảng lặng hoặc chỉ âm nền | — |
| P06 | Cao trào (climax) | Mật độ hình và âm cao nhất video | Beat Cao trào | Phần mở đầu | ngắn, 1–3 s | mạnh, nhiều lớp | nhạc đỉnh | — |
| P07 | Lắng xuống (resolve) | Hạ nhịp sau cao trào, khép ý | Payoff, kết | Giữa phần giải thích | dài, 5–10 s | chậm lại | nhạc giảm, lời chậm | — |
| P08 | Khoảng thở (breathing room) | Hình đẹp, không thông tin mới | Giữa hai khối kiến thức nặng | Video ngắn dưới 60 s | dài, 3–6 s | trôi nhẹ | nhạc, không lời | — |
| P09 | Cắt theo phách (rhythmic) | Đổi hình đúng phách nhạc | Montage liệt kê, tóm tắt | Đoạn cần lời rõ | theo phách, 0.5–2 s | đồng bộ phách | nhạc dẫn | — |
| P10 | Giữ căng (tension hold) | Một hình dài, không cắt | Chờ đợi, quá trình liên tục | Thông tin dày | rất dài, 10–30 s | liên tục, một hướng | âm nền tăng dần | Children of Men (2006): cảnh tấn công trong xe, một cú máy |
| P11 | Tăng tiến (crescendo) | Quy mô, số lượng hoặc tốc độ tăng theo bậc | Thang đo, độ lớn tăng dần | Ý giảm dần | vừa → ngắn | mở rộng dần | nhạc dâng | Powers of Ten (Eames, 1977): mỗi 10 s tăng 10 lần |
| P12 | Thu hẹp (ramp down) | Từ toàn cảnh về một chi tiết | Đi từ khái quát vào cụ thể | Đang ở chi tiết | vừa → dài | thu vào | lời chậm lại | — |
| P13 | Gọi lại (callback) | Lặp lại hình hoặc nhịp của đoạn trước | Payoff nhắc Hook | Chưa gieo gì trước đó | như đoạn gốc | như đoạn gốc | lặp motif âm | — |
| P14 | Tóm tắt nhanh (recap) | Lướt lại các hình chính | Trước kết luận | Video dưới 90 s | ngắn, 1–2 s | nhanh, đều | lời tóm tắt | — |
| P15 | Đếm ngược (countdown) | Mốc thời gian hoặc số tiến về một điểm | Trình tự thời gian, quá trình có hạn | Không có mốc | ngắn dần | hướng tới một điểm | tiếng tích tắc, nhạc nén | — |
| P16 | Ngắt đột ngột (staccato) | Chuỗi cắt ngắn, dứt khoát, có quãng nghỉ | Liệt kê sai lầm, phản bác | Cảm xúc nhẹ nhàng | ngắn, 1–2 s | dừng–đi | lời ngắt câu | — |

## B. Chuyển cảnh (`transition_types`)

| id | name | summary | mechanics | meaning | when_to_use | avoid_when | needs_motif | code_feasibility | film_examples |
|---|---|---|---|---|---|---|---|---|---|
| T01 | Cắt thẳng (cut) | Đổi hình tức thì | 0 s | liền mạch, trung tính | mặc định | — | không | dễ | — |
| T02 | Match cut | Cắt giữa hai hình có cùng hình dạng hoặc chuyển động | giữ hình dạng hoặc vị trí, đổi chủ thể | nối hai ý bằng hình giống nhau | nối hai khái niệm, hai thời điểm | hai hình không có điểm chung | có | vừa | 2001: A Space Odyssey (1968): khúc xương → vệ tinh; Lawrence of Arabia (1962): que diêm → mặt trời sa mạc |
| T03 | Graphic match | Khớp đường nét hoặc bố cục, không cần cùng chủ thể | khớp khung tròn, đường chéo… | liên tưởng thị giác | đổi đối tượng có hình gần nhau | — | có | vừa | Psycho (1960): lỗ thoát nước → mắt |
| T04 | Hoà tan (dissolve) | Hình trước mờ dần vào hình sau | chồng 0.5–2 s | thời gian trôi, liên tưởng mềm | đổi thời điểm, nối mềm | cần nhịp nhanh | không | dễ | — |
| T05 | Mờ về đen (fade to black) | Tắt hình về đen rồi mở | 0.5–1.5 s mỗi chiều | kết một chương | giữa hai phần lớn | giữa câu đang nói | không | dễ | — |
| T06 | Cắt về đen (cut to black) | Đen tức thì | 0 s, giữ đen 1–3 s | đột ngột, bỏ lửng | sau ý gây sốc, kết bỏ ngỏ | dùng nhiều lần | không | dễ | The Sopranos (2007), tập cuối |
| T07 | Smash cut | Cắt mạnh từ yên sang ồn hoặc ngược lại | 0 s, đổi âm lượng gấp | bất ngờ, hài hoặc sốc | phá kỳ vọng | đoạn trang nghiêm cần mềm | không | dễ | — |
| T08 | Jump cut | Cắt cùng khung, bỏ một đoạn thời gian | cùng góc, nhảy thời gian | rút gọn, gấp gáp | rút gọn quá trình dài | cần cảm giác liên tục | không | dễ | Breathless (Godard, 1960) |
| T09 | Wipe | Hình sau quét qua thay hình trước | đường quét 0.3–1 s | chuyển chương, nhẹ nhàng | đổi địa điểm, chủ đề | phim cần nghiêm | không | dễ | Seven Samurai (1954), Star Wars (1977) |
| T10 | Iris | Vòng tròn đóng/mở | 0.5–1 s | tập trung vào một điểm, cổ điển | khoá vào chi tiết | — | không | dễ | phim câm (dùng phổ biến) |
| T11 | Morph | Hình này biến dạng liên tục thành hình kia | nội suy hình 1–3 s | biến đổi, cùng bản chất | quá trình biến đổi | hai thứ không liên quan | có | khó | Black or White (Michael Jackson, 1991): khuôn mặt biến đổi |
| T12 | Zoom vào (zoom in / push in) | Đi sâu vào một chi tiết, ra hình mới | phóng to tới khi chi tiết thành cảnh mới | đi vào bên trong | từ vĩ mô vào vi mô | — | không | vừa | Powers of Ten (1977), phần đi vào bàn tay |
| T13 | Zoom ra (zoom out / pull back) | Lùi ra thấy bối cảnh lớn hơn | thu nhỏ liên tục | lộ quy mô | so quy mô, lộ toàn cảnh | — | không | vừa | Powers of Ten (1977), phần đi ra vũ trụ |
| T14 | Lia nhanh (whip pan) | Lia nhoè nối hai cảnh | nhoè chuyển động 0.2–0.5 s | năng lượng, thời gian trôi nhanh | montage, đổi nơi nhanh | đoạn trầm | không | vừa | Citizen Kane (1941): đoạn bữa sáng |
| T15 | Lia sang (pan / slide) | Camera trượt sang khung kề bên | trượt 0.5–1.5 s | không gian liên tục | hai thứ cạnh nhau | — | không | dễ | — |
| T16 | Cắt đan (cross-cutting) | Luân phiên hai mạch song song | luân phiên nhiều lần | đồng thời, đối chiếu | so sánh song song | một mạch | không | dễ | The Godfather (1972): lễ rửa tội; Inception (2010): các tầng giấc mơ |
| T17 | J-cut | Âm của cảnh sau vào trước hình | âm dẫn trước 0.5–2 s | dẫn dắt | báo trước cảnh sau | — | không | dễ | — |
| T18 | L-cut | Âm cảnh trước còn kéo sang hình sau | âm kéo dài 0.5–2 s | liền mạch cảm xúc | lời còn dở khi đổi hình | — | không | dễ | — |
| T19 | Cắt theo chuyển động (match on action) | Cắt giữa một chuyển động, tiếp tục ở hình sau | cắt giữa động tác | liền mạch | đổi góc cùng hành động | — | có | vừa | — |
| T20 | Xuyên qua vật (through object) | Đi xuyên một vật hoặc mặt phẳng sang cảnh mới | camera xuyên qua trong 0.5–1 s | đi vào bên trong | xuyên lớp, vào bên trong | — | có | vừa | — |
| T21 | Lấp khung (masking / object wipe) | Vật đi qua che kín khung rồi lộ cảnh mới | vật che 0.3–0.8 s | liền mạch giấu đường cắt | đổi cảnh khó nối | — | không | vừa | Rope (Hitchcock, 1948): các cú cắt giấu sau lưng nhân vật |
| T22 | Đóng băng hình (freeze frame) | Dừng hình ở khung cuối | giữ 1–3 s | nhấn một khoảnh khắc | chú thích, kết | dùng nhiều lần | không | dễ | The 400 Blows (1959): cảnh cuối |
| T23 | Chia khung (split screen) | Chia khung, hình mới vào một nửa | trượt chia 0.5–1 s | đồng thời, so sánh | hai góc nhìn | màn nhỏ, hình nhiều chi tiết | không | dễ | — |
| T24 | Biến thành sơ đồ (diagram morph) | Hình thực tả biến thành sơ đồ hoặc biểu tượng | nét vẽ còn lại, chi tiết rơi đi 1–2 s | trừu tượng hoá | từ hiện tượng tới mô hình | — | có | vừa | — |
| T25 | Lật giấy (flip / page turn) | Lật như trang sách hoặc thẻ | 0.5–1 s | chương mới, liệt kê | danh sách, các bước | phim nghiêm | không | dễ | — |
| T26 | Nhoè lấy nét (focus pull / blur) | Mờ nét hình trước, rõ nét hình sau | 0.5–1.5 s | chuyển chú ý | đổi lớp tiền cảnh và hậu cảnh | — | không | dễ | — |
| T27 | Chớp sáng (flash) | Loé trắng nối cảnh | 0.1–0.3 s | va chạm, kí ức, năng lượng | vụ nổ, sự kiện mạnh | người xem nhạy sáng; dùng nhiều | không | dễ | — |

## Câu hỏi khi duyệt
- Thêm, bớt hoặc gộp mục nào?
- Có giữ thêm loại cho video khoa học không (đã thêm T24 Biến thành sơ đồ, P11 Tăng tiến)?
