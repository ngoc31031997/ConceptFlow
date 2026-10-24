# Ví dụ thực tế Input / Output xuyên suốt pipeline video animation

## Bối cảnh dự án mẫu

-   **Chủ đề:** Chuyện gì xảy ra nếu ta rơi vào hố đen?
-   **Khán giả:** Người xem phổ thông, từ cấp 2 trở lên.
-   **Thời lượng mục tiêu:** 4--5 phút.
-   **Mục tiêu:** Giải thích chân trời sự kiện, sự khác nhau giữa người
    rơi và người quan sát ở xa, cùng vai trò của lực thủy triều.
-   **Lưu ý:** Đây là ví dụ minh họa quy trình. Các khẳng định khoa học
    và nguồn phải được xác minh trước khi phát hành thật.

Mỗi bước bên dưới gồm input, công việc, output mẫu và điều kiện chuyển
bước. Những ví dụ file là bản mẫu có thể dùng làm starting point, không
phải tài liệu nghiên cứu đã được xác minh hoàn chỉnh.

------------------------------------------------------------------------

# A. Phát triển nội dung

## 1. IDEA --- Ý tưởng

**Input** \> "Chuyện gì xảy ra nếu ta rơi vào hố đen? Ta có bị kéo dài
như sợi mì không? Người bên ngoài có thấy ta biến mất không?"

**Công việc** - Chọn câu hỏi trung tâm. - Ghi các điểm gây tò mò, chưa
cần viết lời dẫn hay nghĩ hình ảnh chi tiết.

**Output --- `idea.md`**

``` markdown
# Idea
Chủ đề: Điều gì xảy ra khi một người rơi vào hố đen?

Câu hỏi trung tâm:
Người rơi trải nghiệm điều gì, và người quan sát ở xa sẽ thấy gì?

Điểm gây tò mò:
- Chân trời sự kiện là gì?
- Vì sao hai người quan sát có thể mô tả diễn biến khác nhau?
- Cơ thể có bị kéo dài không?

Phạm vi:
Giải thích trực quan cho khán giả phổ thông, không đi sâu vào toán học.
```

**Hoàn thành khi:** Có câu hỏi đủ rõ để bắt đầu nghiên cứu.

## 2. CREATIVE BRIEF --- Định hướng sáng tạo

**Input:** `idea.md`

**Công việc:** Chốt đối tượng, mục tiêu học tập, thời lượng, giọng điệu
và nội dung cần tránh.

**Output --- `creative-brief.md`**

``` markdown
# Creative Brief
Audience: Khán giả phổ thông từ cấp 2 trở lên.
Duration: 4–5 phút.
Tone: Tò mò, dễ hiểu, chính xác; không hù dọa quá mức.

Learning objectives:
1. Giải thích chân trời sự kiện ở mức khái niệm.
2. Phân biệt trải nghiệm của người rơi và thông tin người ở xa nhận được.
3. Hiểu lực thủy triều phụ thuộc vào điều kiện vật lý.

Avoid:
- Mô tả chân trời sự kiện như một lớp vỏ cứng.
- Nói mọi hố đen đều làm cơ thể bị kéo dài theo cùng một cách.
- Trình bày các chi tiết chưa chắc chắn như sự thật đã xác lập.
```

**Hoàn thành khi:** Nhóm hiểu video dành cho ai, muốn người xem hiểu gì
và video dài bao lâu.

## 3. RESEARCH & FACT-CHECK --- Nghiên cứu

**Input:** `idea.md`, `creative-brief.md`

**Công việc** - Thu thập nguồn khoa học đáng tin cậy. - Gắn nguồn với
từng khẳng định quan trọng. - Ghi rõ điểm nào đã xác minh, còn tranh
luận hoặc cần diễn đạt thận trọng.

**Output A --- `research/research-dossier.md`**

``` markdown
# Research Dossier

## R1 — Event horizon
Khái niệm: Ranh giới nhân quả mà từ bên trong, tín hiệu không thể thoát
ra đến người quan sát ở xa theo mô tả cổ điển của thuyết tương đối rộng.
Cần kiểm tra: Cách diễn đạt chính xác và phạm vi áp dụng.

## R2 — Distant observer
Tín hiệu từ người rơi bị dịch đỏ và ngày càng yếu theo cách mô tả của mô hình.
Cần kiểm tra: Tránh câu đơn giản hóa gây hiểu nhầm rằng người ngoài xem
được mọi thứ theo thời gian thực.

## R3 — Falling observer
Người rơi tự do có thể vượt chân trời trong thời gian riêng hữu hạn trong
mô hình phù hợp; không nhất thiết gặp một bức tường vật chất tại đó.
Cần kiểm tra: Các giả định và cách giải thích cho người phổ thông.

## R4 — Tidal forces
Tác động thủy triều phụ thuộc khối lượng hố đen và vị trí.
Cần kiểm tra: Tránh khẳng định một kết cục giống nhau cho mọi hố đen.

## Open questions
- Nên giải thích khác biệt giữa hố đen sao và hố đen siêu nặng đến mức nào?
- Cách nói về người quan sát ở xa cần được chuyên gia/nguồn xác nhận.
```

**Output B --- `research/source-register.csv`**

``` csv
claim_id,claim,source_title,source_url,status,notes
R1,Định nghĩa chân trời sự kiện,Chưa điền nguồn,Chưa điền,needs_verification,Phải xác minh trước khi khóa script
R2,Tín hiệu đến người quan sát ở xa,Chưa điền nguồn,Chưa điền,needs_verification,Cần nguồn chuyên môn
R3,Người rơi vượt chân trời,Chưa điền nguồn,Chưa điền,needs_verification,Kiểm tra giả định mô hình
R4,Lực thủy triều,Chưa điền nguồn,Chưa điền,needs_verification,Kiểm tra phạm vi áp dụng
```

**Hoàn thành khi:** Có đủ bằng chứng cho các claim chính. Không được
đánh dấu nghiên cứu hoàn tất nếu nguồn vẫn chưa xác minh.

## 4. STORY DEVELOPMENT --- Phát triển câu chuyện

**Input:** `creative-brief.md`, `research-dossier.md`

**Công việc:** Chọn góc kể và xây dựng mạch từ câu hỏi đến lời giải.

**Output --- `story-structure.md`**

``` markdown
# Story Structure
Central question:
Điều gì xảy ra khi ta rơi vào hố đen, và tại sao người ở xa mô tả
diễn biến khác với người đang rơi?

Core answer:
Người rơi và người ở xa có những trải nghiệm/nguồn thông tin khác nhau.
Chân trời sự kiện là ranh giới quan trọng đối với khả năng gửi tín hiệu ra ngoài.

Narrative arc:
1. Hook: Người ngoài có thấy khoảnh khắc ta vượt ranh giới không?
2. Đặt bối cảnh: Hố đen và chân trời sự kiện là gì?
3. Hai góc nhìn: Người rơi và người ở xa.
4. Giải thích tín hiệu và chân trời sự kiện.
5. Lực thủy triều và điều kiện ảnh hưởng cơ thể.
6. Payoff: Tổng hợp điều mỗi góc nhìn có thể trải nghiệm/biết được.
```

**Hoàn thành khi:** Có đường dây kể chuyện rõ ràng, không chỉ là danh
sách thông tin.

## 5. OUTLINE / BEAT SHEET --- Dàn ý theo nhịp

**Input:** `story-structure.md`, `research-dossier.md`

**Output --- `beat-sheet.md`**

``` markdown
| Beat ID | Chức năng | Ý chính | Thời lượng |
|---|---|---|---:|
| B001 | Hook | Người rơi và người ngoài thấy gì? | 20s |
| B002 | Bối cảnh | Chân trời sự kiện là gì? | 35s |
| B003 | Hai góc nhìn | Người rơi và người ở xa nhận thông tin khác nhau | 40s |
| B004 | Giải thích | Tín hiệu ánh sáng đến người ở xa | 45s |
| B005 | Mấu chốt | Chân trời không phải bề mặt rắn | 45s |
| B006 | Hệ quả | Lực thủy triều phụ thuộc điều kiện | 40s |
| B007 | Payoff | Tổng hợp câu trả lời | 30s |
```

**Hoàn thành khi:** Có thể kể lại video từ đầu đến cuối chỉ bằng các
beat.

## 6. SCRIPTWRITING --- Viết kịch bản

**Input:** `beat-sheet.md`, `story-structure.md`, `research-dossier.md`,
`creative-brief.md`

**Output --- `script-v1.md` (trích đoạn)**

``` markdown
## B001 — Hook
Hãy tưởng tượng bạn đang rơi về phía một hố đen.
Bạn nhìn thấy gì? Và một người bạn đứng thật xa sẽ thấy gì?

Có vẻ như cả hai đang quan sát cùng một sự kiện.
Nhưng câu trả lời không đơn giản như vậy.

## B002 — Chân trời sự kiện
Hố đen không có một lớp vỏ cứng để bạn đâm vào.
Điều quan trọng là một ranh giới gọi là chân trời sự kiện.

Một khi đã ở bên trong ranh giới này, ánh sáng cũng không thể truyền
ra ngoài đến người quan sát ở xa theo mô tả cổ điển của thuyết tương đối rộng.
```

**Hoàn thành khi:** Lời dẫn đọc được thành tiếng, logic liền mạch và các
claim có thể truy về nghiên cứu.

## 7. SCRIPT REVIEW & LOCK --- Rà soát và khóa

**Input:** `script-v1.md`, `creative-brief.md`, `research-dossier.md`,
`beat-sheet.md`, `review-rules.md`

**Công việc** 1. AI review độ chính xác, logic, mức độ dễ hiểu, thời
lượng và khả năng sản xuất. 2. Tổng hợp cảnh báo, phân loại
Critical/Major/Minor. 3. Người phụ trách quyết định sửa gì. 4. Sửa
script và chạy review lại. 5. Người có trách nhiệm nội dung phê duyệt
bản cuối.

**Output A --- `script-review-report.md`**

``` markdown
# Review Report
- [Critical] Claim về người quan sát ở xa cần nguồn đã xác minh.
  Status: OPEN
- [Major] Phần định nghĩa chân trời và hệ quả đang gộp quá nhiều ý.
  Suggested action: Tách thành hai đoạn.
  Status: OPEN
- [Minor] Cụm "cùng xem một sự kiện" có thể gây hiểu nhầm.
  Suggested action: Diễn đạt lại.
  Status: ACCEPTED

Decision: Không khóa script khi còn lỗi Critical chưa xử lý.
```

**Output B:** `script-final.md`, `script-handoff-notes.md`,
`script-version.json`

**Hoàn thành khi:** Không còn lỗi nghiêm trọng chưa xử lý, nguồn quan
trọng được xác minh, thời lượng phù hợp và người phụ trách đã duyệt.

------------------------------------------------------------------------

# B. Tiền kỳ sản xuất (Pre-production)

## 8. SCRIPT BREAKDOWN --- Phân rã kịch bản

**Input:** `script-final.md`, `script-handoff-notes.md`

**Output --- `script-breakdown.csv`**

``` csv
beat_id,script_segment,purpose,key_fact,estimated_seconds
B001,"Hãy tưởng tượng bạn đang rơi về phía một hố đen...",Tạo tò mò,Câu hỏi trung tâm,20
B002,"Hố đen không có một lớp vỏ cứng...",Định nghĩa,Chân trời không phải bề mặt rắn,35
B003,"Hãy tách câu chuyện thành hai góc nhìn...",Thiết lập góc nhìn,Người rơi và người ở xa nhận thông tin khác nhau,40
```

**Hoàn thành khi:** Mỗi đoạn có ID ổn định, mục tiêu và thời lượng dự
kiến.

## 9. VISUAL CONCEPT --- Ngôn ngữ hình ảnh

**Input:** `creative-brief.md`, `script-breakdown.csv`,
`script-final.md`

**Output --- `visual-concept.md`**

``` markdown
# Visual Concept
Approach: 2D tối giản, dùng hình học và sơ đồ để giải thích.
- Dùng một đường ranh giới nhất quán cho chân trời sự kiện.
- Không vẽ chân trời như một lớp vỏ vật chất.
- Dùng bố cục nhất quán để phân biệt "người rơi" và "người quan sát ở xa".
- Chỉ dùng chữ trên màn hình khi cần; tránh lặp nguyên lời dẫn.
- Sơ đồ là mô hình giải thích, không phải ảnh chụp thực tế.
```

**Hoàn thành khi:** Có quy tắc hình ảnh áp dụng được nhất quán trong
toàn video.

## 10. VISUAL PLANNING --- Lên phương án hình ảnh

**Input:** `script-breakdown.csv`, `visual-concept.md`,
`research-dossier.md`

**Output --- `visual-plan.csv`**

``` csv
beat_id,viewer_should_understand,visual_method,essential_information,risk
B001,Hai góc nhìn khác nhau,Đặt hai góc nhìn trong cùng tình huống,Người rơi và người ở xa,Không ngụ ý họ nhận cùng tín hiệu
B002,Chân trời không phải vỏ cứng,Mô hình hố đen với ranh giới,Ý nghĩa của ranh giới,Không làm nó giống bề mặt vật chất
B003,Tín hiệu cần truyền đến người quan sát,So sánh hai góc nhìn theo thời gian,Tín hiệu ánh sáng,Không thể hiện mọi thứ tức thời
B004,Lực thủy triều phụ thuộc điều kiện,So sánh hai trường hợp khái niệm,Khối lượng và vị trí,Không khái quát mọi hố đen
```

**Hoàn thành khi:** Mỗi beat có mục tiêu hình ảnh, phương án thể hiện và
rủi ro hiểu sai.

## 11. VISUAL SEQUENCING --- Sắp xếp diễn tiến hình ảnh

**Input:** `visual-plan.csv`, `script-breakdown.csv`

**Output --- `visual-sequence.md`**

``` markdown
# Sequence S02 — Event Horizon (B002)
1. Hiện mô hình hố đen.
2. Đưa đường ranh giới vào.
3. Hiện nhãn "chân trời sự kiện".
4. Làm rõ đường này biểu thị ranh giới khái niệm, không phải bề mặt rắn.

# Sequence S03 — Two Viewpoints (B003–B004)
1. Đặt hai góc nhìn.
2. Cho thấy tín hiệu truyền từ người rơi đến người quan sát.
3. Minh họa tín hiệu thay đổi khi đến người ở xa.
4. Chuyển sang ý nghĩa vật lý của chân trời sự kiện.
```

**Hoàn thành khi:** Có trình tự tiết lộ thông tin và quan hệ giữa các
diễn biến.

## 12. STORYBOARD --- Phác thảo shot

**Input:** `visual-sequence.md`, `visual-concept.md`,
`script-breakdown.csv`

**Output --- `storyboard-index.csv`**

``` csv
shot_id,beat_id,frame_purpose,estimated_seconds,status
SH001,B001,Đặt người rơi vào tình huống,8,thumbnail
SH002,B001,Đặt câu hỏi về người quan sát ở xa,12,thumbnail
SH003,B002,Giới thiệu mô hình hố đen,12,thumbnail
SH004,B002,Làm rõ chân trời không phải vỏ cứng,23,thumbnail
SH005,B003,Thiết lập hai góc nhìn,18,thumbnail
```

Mỗi khung thực tế còn có hình phác thảo, lời dẫn liên quan, bố cục,
trạng thái/hành động chính và ghi chú khoa học.

**Hoàn thành khi:** Có thể đọc storyboard theo thứ tự và đánh giá được
bố cục, logic và nhịp.

## 13. ASSET DESIGN --- Thiết kế tài nguyên

**Input:** `visual-concept.md`, storyboard, danh sách asset cần thiết.

**Output --- `asset-manifest.csv`**

``` csv
asset_id,name,type,used_in,description,status
A001,black_hole_diagram,diagram,SH003;SH004,Mô hình hố đen tối giản,planned
A002,event_horizon_ring,shape,SH004,Đường biểu thị chân trời sự kiện,planned
A003,falling_observer,character,SH001;SH002;SH005,Nhân vật biểu tượng người rơi,planned
A004,distant_observer,character,SH002;SH005,Người quan sát ở xa,planned
A005,light_signal,graphic,SH005,Đồ họa biểu thị tín hiệu,planned
```

**Output bổ sung --- `asset-guidelines.md`**

``` markdown
- Dùng cùng hình dạng/màu sắc cho cùng khái niệm.
- Mỗi asset có ID và tên file ổn định.
- Tách layer cần animate.
- Tránh thiết kế khiến đường chân trời giống một bề mặt rắn.
```

**Hoàn thành khi:** Biết asset nào cần có, xuất hiện ở shot nào và cần
đáp ứng quy tắc gì.

## 14. MOTION PLANNING --- Thiết kế chuyển động

**Input:** Storyboard, `asset-manifest.csv`, `script-final.md`,
`visual-sequence.md`

**Output --- `motion-spec.md`**

``` markdown
## SH001 — Hook
Start: Nhân vật ở gần mép khung; hố đen chưa hiện đầy đủ.
Action: Nhân vật dịch chuyển về phía vùng hố đen.
End: Người xem nhận ra hướng chuyển động.
Sync: Chuyển động đồng bộ với câu mở đầu.

## SH004 — Event horizon
Start: Mô hình hố đen đã hiện.
Action: Đường ranh giới xuất hiện, sau đó là nhãn.
End: Người xem hiểu đây là ranh giới khái niệm.
Constraint: Không mô tả người rơi va vào một bề mặt rắn.
```

**Hoàn thành khi:** Mỗi shot có trạng thái đầu/cuối, hành động chính và
ràng buộc cần tuân thủ.

## 15. ANIMATIC & PRE-PRODUCTION REVIEW

**Input:** Storyboard, script/voice-over tạm, motion spec, thời lượng
shot.

**Công việc:** Ghép khung storyboard theo timeline, thêm lời dẫn và
placeholder chuyển động để kiểm tra nhịp.

**Output:** `animatic-v1.mp4`, `animatic-review.md`

``` markdown
# Animatic Review
- AN-01 / SH003–SH004: Định nghĩa và hệ quả xuất hiện quá sát nhau.
  Action: Tăng thời gian giữ khung hoặc tách phần giải thích.
- AN-02 / SH005: Hai góc nhìn chưa đủ khác biệt khi xem liên tục.
  Action: Làm rõ nhãn và giữ bố cục nhất quán.

Decision: Chưa duyệt cho đến khi các vấn đề ảnh hưởng khả năng hiểu được xử lý.
```

**Hoàn thành khi:** Nhịp kể, logic hình ảnh và thời lượng được duyệt
trước khi sản xuất đầy đủ.

------------------------------------------------------------------------

# C. Sản xuất (Production)

## 16. VISUAL / ASSET PRODUCTION --- Hoàn thiện asset

**Input:** Gói tiền kỳ đã duyệt, storyboard, asset manifest, visual
guidelines.

**Output --- manifest cập nhật**

``` csv
asset_id,file_path,format,layers,used_in,status
A001,assets/diagrams/black_hole_diagram.svg,SVG,background;ring;labels,SH003;SH004,approved
A002,assets/shapes/event_horizon_ring.svg,SVG,ring,SH004,approved
A003,assets/characters/falling_observer.svg,SVG,body;arms;legs,SH001;SH002;SH005,approved
A004,assets/characters/distant_observer.svg,SVG,body;arms;legs,SH002;SH005,approved
A005,assets/graphics/light_signal.svg,SVG,path;point,SH005,approved
```

**Hoàn thành khi:** Các asset có file thực tế, đúng định dạng, sẵn sàng
đưa vào animation.

## 17. ANIMATION PRODUCTION --- Sản xuất animation

**Input:** Asset production-ready, storyboard, `motion-spec.md`, shot
timing.

**Output --- `shot-status.csv`**

``` csv
shot_id,asset_ids,render_path,status,review_note
SH001,A003,shots/SH001_v03.mp4,ready_for_review,
SH002,A003;A004,shots/SH002_v02.mp4,ready_for_review,
SH003,A001,shots/SH003_v01.mp4,needs_fix,Nhãn xuất hiện quá nhanh
SH004,A001;A002,shots/SH004_v02.mp4,approved,
SH005,A003;A004;A005,shots/SH005_v01.mp4,ready_for_review,
```

**Hoàn thành khi:** Các shot được render, có trạng thái và sẵn sàng kiểm
tra.

## 18. SHOT REVIEW & FIXES --- Duyệt shot

**Input:** Shot render, storyboard, motion spec, checklist chất lượng.

**Output --- `shot-review-log.csv`**

``` csv
issue_id,shot_id,severity,issue,action,status
SR001,SH003,major,Nhãn xuất hiện quá nhanh,Tăng thời gian hiển thị,open
SR002,SH004,critical,Đường chân trời trông như bề mặt cứng,Sửa cách thể hiện,open
SR003,SH002,minor,Căn lề nhân vật chưa nhất quán,Điều chỉnh vị trí,closed
```

**Hoàn thành khi:** Không còn lỗi nghiêm trọng chưa xử lý và shot được
duyệt để ghép dựng.

------------------------------------------------------------------------

# D. Hậu kỳ (Post-production)

## 19. EDITING & COMPOSITING --- Dựng và tổng hợp

**Input:** Shot đã duyệt, voice-over chính thức, thứ tự shot.

**Công việc:** Ghép SH001 → SH002 → SH003 → SH004 → SH005; đồng bộ lời
dẫn; điều chỉnh nhịp, chuyển cảnh và lớp đồ họa.

**Output:** `edit-project/`, `picture-lock-v1.mp4`

**Hoàn thành khi:** Hình ảnh chạy liên tục và cấu trúc dựng được duyệt.

## 20. SOUND DESIGN & MIXING --- Hoàn thiện âm thanh

**Input:** Bản dựng hình, voice-over, nhạc nền, hiệu ứng âm thanh.

**Công việc:** Làm sạch lời dẫn, cân bằng nhạc và voice-over, đồng bộ
hiệu ứng, kiểm tra clipping và âm lượng.

**Output:** `final-mix.wav`, `video-with-final-audio.mp4`

**Hoàn thành khi:** Lời dẫn rõ, âm thanh đồng bộ và đáp ứng yêu cầu phát
hành.

## 21. FINAL QUALITY CONTROL --- QC cuối

**Input:** Video hình + tiếng hoàn chỉnh, creative brief, QC checklist.

**Output --- `final-qc-report.md`**

``` markdown
# Final QC Report
Content:
- [x] Câu hỏi trung tâm được trả lời
- [ ] Các claim khoa học đã được đối chiếu với nguồn
Picture:
- [x] Không có khung đen ngoài chủ ý
- [ ] Kiểm tra chuyển cảnh ở SH005
Audio:
- [x] Voice-over rõ
- [x] Không clipping nghe thấy
Technical:
- [ ] Xác nhận độ phân giải/frame rate theo nền tảng

Decision: HOLD cho đến khi hoàn tất các mục chưa đánh dấu.
```

**Hoàn thành khi:** Tất cả mục bắt buộc đều đạt và lỗi đã được sửa, xác
nhận.

## 22. EXPORT & DELIVERY --- Xuất và bàn giao

**Input:** Video đã qua QC, yêu cầu kỹ thuật, danh sách file bàn giao.

**Output --- `delivery-manifest.md`**

``` markdown
# Delivery Manifest
Final master:
- final-video.mp4
Captions:
- subtitles/vi-VN.srt
Publishing:
- thumbnail.png
- video-description.md
- credits.md
Archive:
- edit-project/
- audio/
- assets/
- script-final.md
- final-qc-report.md
Release status: Approved after final QC.
```

**Hoàn thành khi:** File xuất mở/phát được, thông số đúng và gói bàn
giao đầy đủ.

------------------------------------------------------------------------

# 25. Truy vết một ý từ kịch bản đến video cuối

Ví dụ ý: **"Chân trời sự kiện không phải một lớp vỏ cứng."**

``` text
script-final.md — B002
        ↓
script-breakdown.csv — B002
        ↓
visual-plan.csv — người xem phải hiểu ranh giới không phải vật chất
        ↓
visual-sequence.md — mô hình → đường ranh giới → giải thích
        ↓
storyboard — SH004
        ↓
asset-manifest.csv — A001 + A002
        ↓
motion-spec.md — đường ranh giới và nhãn xuất hiện theo thứ tự
        ↓
animation — SH004_v02.mp4
        ↓
shot-review-log.csv — kiểm tra hình ảnh không gây hiểu lầm
        ↓
final-video.mp4
```

Nếu shot khiến người xem hiểu sai, có thể lần ngược từ shot đến
storyboard và visual plan để sửa đúng nơi phát sinh vấn đề.

------------------------------------------------------------------------

# 26. Nguyên tắc dùng trong dự án thật

1.  **Các file là ví dụ cấu trúc:** Nội dung, đường dẫn và trạng thái
    trên là mẫu minh họa, không phải bằng chứng rằng dự án đã sản xuất.
2.  **Không bịa nguồn:** Các trường nguồn đang để trống phải được xác
    minh trước khi khóa kịch bản.
3.  **ID ổn định:** Dùng `B001` cho beat, `SH001` cho shot, `A001` cho
    asset để liên kết các bước.
4.  **Có vòng lặp:** Nếu animatic phát hiện vấn đề, quay lại visual
    plan/storyboard; nếu shot sai, sửa asset/motion/shot tương ứng.
5.  **AI hỗ trợ kiểm tra:** AI có thể tìm vấn đề, tạo bản nháp và đối
    chiếu; con người chịu trách nhiệm phê duyệt các claim khoa học và
    bản phát hành.
6.  **Không cần một file cho mỗi bước:** Có thể gộp tài liệu khi dự án
    nhỏ, miễn input, output, trạng thái và tiêu chí hoàn thành vẫn rõ.
