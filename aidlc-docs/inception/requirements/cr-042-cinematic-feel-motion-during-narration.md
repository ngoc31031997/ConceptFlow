# CR-042 — Cảm giác "phim" thay vì "slide": hình chuyển động trong lúc đọc thoại, bỏ thẻ tiêu đề, bổ sung quy tắc đạo diễn, SFX

## Date
2026-09-26

## Stage
Requirements Analysis (Change Request). **GĐ1–GĐ3 đã code xong (2026-09-26). GĐ4 (SFX/nhạc) nằm trong backlog.**

## Intent Analysis
- **Request type**: Cải thiện chất lượng video (giữ chân người xem) từ một bản review prompt Visual Director + bước dựng phía sau.
- **Scope estimate**: GĐ1 — `rendering` (engine Manim). GĐ2 — `authoring-service` (seed prompt Kỹ sư Manim, Visual Director), `rendering` (`recap()`). GĐ3 — cùng seed Visual Director. GĐ4 — `rendering`, `video-assembly` (CR riêng khi tới lượt).
- **Complexity estimate**: GĐ1 trung bình (đổi hợp đồng `narrate`, hai lượt render). GĐ2/GĐ3 thấp (prompt + một component). GĐ4 cao.

## Kết luận của review
Prompt Visual Director đã viết theo tư duy làm phim khá tốt. Cảm giác "slide" chủ yếu đến từ **engine Manim và prompt Kỹ sư Manim**, nên chỉ sửa prompt đạo diễn không đủ.

## Nguyên nhân gốc (đã đối chiếu với code)

### 1. Khung hình đứng yên trong lúc đọc thoại
`rendering/conceptflow/narration.py:156` — `narrate()` kết thúc bằng `scene.wait(durations[index])`. Suốt thời gian giọng đọc một câu, hình hoàn toàn đứng yên; animation chỉ chạy giữa các câu. Nhịp "hình động → đứng yên nghe nói → hình động" chính là nhịp slide. Quy tắc 6 của đạo diễn đòi "hình luôn sống", còn prompt Kỹ sư chỉ chữa cháy bằng cách tách câu thoại thành nhiều `narrate()` ngắn, mỗi mảnh vẫn là một khung đứng yên.

Engine Remotion không có vấn đề này (mỗi shot có thời lượng riêng, animation chạy theo frame).

### 2. Component dựng sẵn kéo video về slide
`scene.py:344` `hook()` hiện `TitleCard` chứa câu hỏi rồi xoá; `scene.py:358` `recap()` hiện bảng gạch đầu dòng. Prompt Kỹ sư dặn "DÙNG CHÚNG thay vì tự dựng lại", nên 10 giây đầu luôn là trang tiêu đề dù đạo diễn đã viết cảnh mở màn sống. Thư viện (Callout, StepList, ComparisonSplit, DataTable, Recap) phần lớn là khối trình bày kiểu slide; Kỹ sư sẽ với tới chúng trước.

### 3. Prompt Visual Director thiếu quy tắc về nhịp và diễn xuất
Đã tốt, giữ nguyên: phân biệt phim/slide, nhân vật chính bằng hình, match cut, bảng màu mang nghĩa, "Ý nghĩa bất biến", lời thoại nói ý nghĩa không tả hình, danh sách tự kiểm. Còn thiếu: nhịp thị giác, cho người xem đoán trước, diễn xuất bằng chuyển động, chuyển động nền hợp lệ (quy tắc 5 đang cấm mọi "chuyển động trang trí" nên model để hình đứng yên), callback khung mở/kết, khuôn lặp cho chuỗi ví dụ, cấm hook là thẻ tiêu đề. Ngoài ra đạo diễn cố ý "không biết engine" nên có thể viết thứ không dựng nổi (3D, hạt, nhân vật hữu cơ) rồi Kỹ sư hạ cấp thành component slide.

### 4. Thiếu âm thanh
`video-assembly` chỉ trộn giọng đọc + nhạc nền: không SFX, nhạc không theo cảm xúc. (Nhận định về các kênh Kurzgesagt/3Blue1Brown/Vox… là hiểu biết chung, không phải số đo.)

## Thứ tự thực hiện (theo tác động)

### Giai đoạn 1 — `narrate` chạy animation trong lúc đọc thoại (tác động lớn nhất)
**FR123**
- **FR123.1** `narrate(text, *animations)`: các animation kèm theo chạy **đồng thời** với giọng đọc, `run_time` kéo dài đúng bằng thời lượng câu thoại (ví dụ `self.narrate("Một nửa khả năng vừa biến mất.", right_half.animate.fade(0.9))`). Áp dụng ở cả `scene.narrate` và `narration.narrate`.
- **FR123.2** Lượt dry run chưa có audio dùng thời lượng ước lượng (cùng cách hiện đang ước lượng); lượt thật dùng thời lượng đo được. Hai lượt phải cho cùng cấu trúc animation.
- **FR123.3** Câu không kèm animation: mặc định máy quay/khung đẩy vào rất chậm (ambient drift) để không chết khung. Có cờ tắt cho cảnh không hợp (ví dụ khung đã chiếm toàn màn hình).
- **FR123.4** Tương thích ngược: `narrate(text)` một tham số vẫn chạy; code Manim đã sinh trước đây không vỡ. Cần kiểm tác động lên timeline đồng bộ phụ đề/âm thanh của CR-002 và hợp đồng hai lượt render của CR-018.
- **FR123.5** Tạm thời, nếu ưu tiên cảm giác phim ngay: có thể chọn engine Remotion trong lúc chờ GĐ1 (không cần code).

### Giai đoạn 2 — bỏ ép dùng thẻ tiêu đề
**FR124**
- **FR124.1** Prompt Kỹ sư Manim: bỏ câu "DÙNG CHÚNG thay vì tự dựng lại bằng tay" cho `hook`/`recap`. Chỉ dùng khi kịch bản đạo diễn ghi rõ là thẻ tiêu đề; mặc định dựng hook bằng hình và gọi `self.beat("hook")` bằng tay.
- **FR124.2** `recap()` đổi thành cảnh quay lại nhân vật chính ở trạng thái cuối (thay bảng gạch đầu dòng). Giữ chữ ký cũ hoặc bản cũ đổi tên `recap_card()` để prompt/mã cũ không vỡ.
- **FR124.3** `hook()` giữ lại như `hook_card()` cho trường hợp đạo diễn ghi rõ thẻ tiêu đề.

### Giai đoạn 3 — bổ sung quy tắc 12–18 vào prompt Visual Director
**FR125** (chỉ đổi seed prompt, không đổi hợp đồng output)

| # | Quy tắc | Lý do |
|---|---|---|
| 12 | Nhịp thay đổi: khoảng mỗi 3–5 giây có một thay đổi hình có nghĩa; khoảnh khắc aha được lặng 1–2 nhịp | hiện chỉ có "mỗi shot ~6–15 từ", chưa nói nhịp thị giác |
| 13 | Cho người xem đoán trước: dựng xong tình huống, lời thoại đặt câu hỏi, giữ hình ~1 giây rồi mới lộ đáp án | người đã tự đoán mới muốn xem đáp án |
| 14 | Diễn xuất bằng chuyển động: do dự nhích tới rồi lùi, tự tin lao thẳng, thất vọng xẹp xuống và chậm lại | Biên kịch đã tạo nhân vật có tính cách, đạo diễn chưa được dặn cho "diễn" |
| 15 | Chuyển động nền có chủ đích là hợp lệ (đẩy máy rất chậm trong câu suy ngẫm) | quy tắc 5 hiện cấm mọi chuyển động trang trí; sửa để khớp FR123.3 |
| 16 | Khung kết vần với khung mở: cảnh cuối quay lại hình cảnh 1 nhưng mang nghĩa mới | callback bằng hình |
| 17 | Khuôn hình lặp cho chuỗi ví dụ: chung bố cục và chuyển động, chỉ đổi nội dung, có thể dựng như montage nhanh | hợp kiểu A (chuỗi ví dụ) của CR-041 |
| 18 | Hook không được là thẻ tiêu đề: frame đầu tiên đã có thứ chuyển động | chặn thói quen mở bằng TitleCard |

- **FR125.1** Thêm danh sách ngắn "vật liệu dựng tốt" (hình cơ bản, đàn chấm, lưới, đồ thị, mũi tên, số chạy, code, dòng thời gian), **không nhắc tên engine**, để đạo diễn không viết thứ không dựng nổi.
- **FR125.2** Cập nhật danh sách tự kiểm cho quy tắc 12–18; quy tắc 5 sửa theo #15.
- **FR125.3** Sau khi đổi seed: cập nhật hash `golden_prompts_test.go` có chủ đích, build lại và khởi động lại `authoring-service`, kiểm hàng prompt active không bị prompt Creator (`is_system = false`) che (xem memory "re-seed DB after prompt seed edit").

### Giai đoạn 4 — SFX và nhạc theo cảm xúc (**tách CR riêng khi tới lượt**)
- Bước render xuất các **mốc animation** (timestamp); `video-assembly` chèn SFX nhẹ (whoosh/pop) đúng các mốc.
- Nhạc nền nhỏ lại ở khoảnh khắc aha, lên lại khi chuyển phần.
- Ghi lại ở đây để không mất; không thiết kế chi tiết trong CR này.

## Ngoài phạm vi
- Viết lại engine Remotion; thay Manim.
- Playbook Visual Director theo kiểu video A–D (CR-041 đã đặt ngoài phạm vi).
- Thiết kế chi tiết SFX/nhạc (GĐ4).

## Kiểm thử dự kiến
- GĐ1: `narrate` có/không animation; `run_time` của animation bằng thời lượng câu; dry run và lượt thật cùng cấu trúc; code cũ một tham số chạy được; phụ đề/timeline CR-002 không lệch.
- GĐ2: prompt Kỹ sư không còn câu ép; `hook()`/`recap()` mới không sinh TitleCard/bảng; bản card vẫn dùng được khi gọi tường minh.
- GĐ3: test seed kiểm có quy tắc 12–18 và danh sách vật liệu; hash golden cập nhật có chủ đích.
- Chấm bằng chạy thật: 3–5 chủ đề, so sánh video trước/sau (10 giây đầu không còn thẻ tiêu đề; không đoạn đứng yên > 5 giây ngoài khoảnh khắc aha).

## Rủi ro
- Kéo dài animation theo thời lượng thoại có thể làm chuyển động quá chậm với câu dài; cần trần `run_time` hoặc chia nhịp.
- Ambient drift mặc định có thể làm lệch bố cục các cảnh đã tinh chỉnh; cần cờ tắt và kiểm trên vài cảnh mẫu.
- Bỏ ép dùng component có thể làm Kỹ sư sinh mã tự dựng nhiều hơn → tăng tỉ lệ lỗi Manim (liên quan retry/chunked pipeline CR-039).
- Prompt Creator tự viết che seed hệ thống sẽ không thấy quy tắc mới.

## Quyết định đã chốt (Creator, 2026-09-26)
1. Làm full GĐ1 → GĐ3; GĐ4 vào backlog.
2. Ambient drift **tắt mặc định** (`ambient_drift = False`; bật từng câu bằng `drift=True`; `recap()` tự bật).
3. `recap()` **đổi hành vi** (quay lại toàn cảnh, không bảng); bản bảng giữ ở `recap_card()`. `hook()` không còn thẻ; bản thẻ ở `hook_card()`.

## Đã thực hiện
- `rendering/conceptflow/narration.py`: `narrate(scene, text, *animations, drift=False)`; animation chạy với `run_time` = thời lượng câu (đã kiểm Manim 0.18 ép `run_time` lên cả `Animation` có sẵn); lượt dry vẫn chạy animation với thời lượng ước lượng theo số từ để trạng thái màn hình khớp lượt render.
- `scene.py`: `narrate(text, *animations, drift=None)`, `ambient_drift`, `hook`/`hook_card`, `recap`/`recap_card`; `api.SCENE_METHODS` thêm hai tên mới. Test mới trong `test_narration.py`.
- Seeds (`authoring-service`): Visual Director + bản AI v9/v2 (quy tắc 12–18 gộp thành 12–18 + vật liệu dựng tốt, sửa quy tắc 5, tự kiểm); Manim Engineer + bản AI v7/v2 (bỏ "DÙNG CHÚNG", dạy `narrate(text, animation)`/`drift`). Hash golden của `visual_director` và `manim_engineer` cập nhật có chủ đích; thêm `TestCinematicRulesAndMotionDuringNarration`.
- Đã build lại và khởi động lại `authoring-service` và `rendering`; 11 hàng prompt đang active đều là hàng hệ thống và có nội dung mới.
- Chưa làm: kiểm bằng video thật (tiêu chí chấm ở trên); `llm-service` prompt Shorts vẫn nhắc `self.recap([...])` (vẫn chạy, nhưng chỉ đọc lời, không hiện bảng).
