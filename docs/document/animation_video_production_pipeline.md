# Pipeline sản xuất video animation giáo dục

## 1. Mục tiêu và phạm vi

Tài liệu này mô tả pipeline tổng quát để sản xuất video animation giáo
dục/khoa học, từ một ý tưởng ban đầu đến video hoàn chỉnh. Mỗi bước có
mục tiêu, đầu vào, công việc chính, đầu ra và tiêu chí hoàn thành.

Pipeline được chia thành bốn giai đoạn lớn:

1.  **Phát triển nội dung** --- xác định câu chuyện và hoàn thiện kịch
    bản.
2.  **Tiền kỳ sản xuất (Pre-production)** --- lập kế hoạch hình ảnh,
    storyboard, tài nguyên và chuyển động; kiểm tra nhịp kể bằng
    animatic.
3.  **Sản xuất (Production)** --- tạo hình, animation và các cảnh hoàn
    chỉnh.
4.  **Hậu kỳ (Post-production)** --- dựng, âm thanh, hoàn thiện, kiểm
    tra chất lượng và xuất bản.

> **Lưu ý về phân loại:** `Visual Concept`, `Visual Planning`,
> `Visual Sequencing`, `Storyboard`, `Asset Design` và `Motion Planning`
> thuộc tiền kỳ. `Visual Production` là giai đoạn sản xuất thực tế.
> `Animatic` thường được làm trong tiền kỳ để duyệt nhịp kể, nhưng có
> thể được cập nhật khi sản xuất thay đổi. Một số studio có thể gộp hoặc
> đổi tên các bước tùy quy mô dự án.

------------------------------------------------------------------------

## 2. Sơ đồ tổng quan

``` text
DEVELOPMENT
  1. IDEA
      ↓
  2. CREATIVE BRIEF
      ↓
  3. RESEARCH & FACT-CHECK
      ↓
  4. STORY DEVELOPMENT
      ↓
  5. OUTLINE / BEAT SHEET
      ↓
  6. SCRIPTWRITING
      ↓
  7. SCRIPT REVIEW & LOCK

PRE-PRODUCTION
      ↓
  8. SCRIPT BREAKDOWN
      ↓
  9. VISUAL CONCEPT
      ↓
 10. VISUAL PLANNING
      ↓
 11. VISUAL SEQUENCING
      ↓
 12. STORYBOARD
      ↓
 13. ASSET DESIGN
      ↓
 14. MOTION PLANNING
      ↓
 15. ANIMATIC & PRE-PRODUCTION REVIEW

PRODUCTION
      ↓
 16. VISUAL / ASSET PRODUCTION
      ↓
 17. ANIMATION PRODUCTION
      ↓
 18. SHOT REVIEW & FIXES

POST-PRODUCTION
      ↓
 19. EDITING & COMPOSITING
      ↓
 20. SOUND DESIGN & MIXING
      ↓
 21. FINAL QUALITY CONTROL
      ↓
 22. EXPORT & DELIVERY
```

Các bước có thể lặp lại. Ví dụ, nếu animatic cho thấy câu chuyện khó
hiểu hoặc thời lượng không đạt mục tiêu, nhóm có thể quay lại
storyboard, visual planning hoặc kịch bản. Những thay đổi lớn sau khi
khóa kịch bản cần được ghi nhận và phê duyệt.

------------------------------------------------------------------------

# PHẦN A --- PHÁT TRIỂN NỘI DUNG

## 1. IDEA --- Ý tưởng ban đầu

**Mục tiêu:** Xác định câu hỏi, hiện tượng hoặc câu chuyện mà video muốn
khám phá.

**Đầu vào** - Một ý tưởng, câu hỏi hoặc chủ đề sơ bộ. - Có thể có một
quan sát, giả thuyết hoặc điều gây tò mò.

**Công việc chính** - Viết ý tưởng thành một câu hỏi rõ ràng. - Xác định
điều gì khiến chủ đề đáng xem. - Kiểm tra sơ bộ xem chủ đề có thể phát
triển thành một câu chuyện hay không.

**Đầu ra** - `idea.md`

**Nội dung tối thiểu của đầu ra** - Tên/chủ đề tạm thời. - Câu hỏi trung
tâm. - Vì sao câu hỏi thú vị. - Những điều đã biết và chưa biết.

**Tiêu chí hoàn thành** - Có thể mô tả ý tưởng bằng một hoặc hai câu. -
Không cần có sẵn lời giải hoặc toàn bộ cấu trúc video.

------------------------------------------------------------------------

## 2. CREATIVE BRIEF --- Định hướng sáng tạo

**Mục tiêu:** Thống nhất video được làm cho ai, nhằm đạt mục tiêu gì và
có giới hạn nào.

**Đầu vào** - `idea.md`

**Công việc chính** - Xác định khán giả mục tiêu và kiến thức nền. - Xác
định mục tiêu học tập hoặc cảm xúc. - Đặt thời lượng dự kiến, nền tảng
phát hành và phong cách nội dung ở mức tổng quát. - Ghi nhận các giới
hạn về độ phức tạp, nguồn lực hoặc nội dung cần tránh.

**Đầu ra** - `creative-brief.md`

**Tiêu chí hoàn thành** - Nhóm hiểu được video dành cho ai, muốn người
xem hiểu/nhớ điều gì và thời lượng dự kiến là bao nhiêu.

------------------------------------------------------------------------

## 3. RESEARCH & FACT-CHECK --- Nghiên cứu và xác minh thông tin

**Mục tiêu:** Xây dựng nền tảng kiến thức đáng tin cậy trước khi quyết
định cách kể chuyện.

**Đầu vào** - `idea.md` - `creative-brief.md`

**Công việc chính** - Tìm nguồn đáng tin cậy và phù hợp với chủ đề. -
Thu thập các sự kiện, cơ chế, ví dụ và quan điểm liên quan. - Phân biệt
dữ kiện đã xác lập, giả thuyết, mô hình giải thích và điểm còn tranh
luận. - Ghi lại nguồn cho từng khẳng định quan trọng. - Tìm hiểu các
hiểu lầm phổ biến cần xử lý.

**Đầu ra** - `research/research-dossier.md` -
`research/source-register.csv` - `research/open-questions.md`

**Tiêu chí hoàn thành** - Các khẳng định chính có nguồn truy xuất
được. - Các điểm chưa chắc chắn được ghi rõ, không được trình bày như sự
thật đã xác lập.

------------------------------------------------------------------------

## 4. STORY DEVELOPMENT --- Phát triển câu chuyện

**Mục tiêu:** Chọn góc tiếp cận và xác định câu trả lời cốt lõi mà video
sẽ dẫn dắt người xem khám phá.

**Đầu vào** - `creative-brief.md` - `research-dossier.md` -
`open-questions.md`

**Công việc chính** - Chọn góc tiếp cận phù hợp nhất với khán giả. - Xác
định câu hỏi trung tâm và lời hứa của video. - Xác định câu trả lời cốt
lõi. - Sắp xếp các bước suy luận hoặc diễn biến theo quan hệ nguyên
nhân--kết quả. - Xác định mở đầu, điểm chuyển biến, cao trào hoặc phát
hiện chính, và kết luận/payoff. - Quyết định phạm vi: nội dung nào cần
đưa vào và nội dung nào nên bỏ.

**Đầu ra** - `story-structure.md`

**Tiêu chí hoàn thành** - Có một đường dây kể chuyện rõ ràng. - Các phần
hỗ trợ cùng một câu hỏi trung tâm, không chỉ là danh sách thông tin rời
rạc.

------------------------------------------------------------------------

## 5. OUTLINE / BEAT SHEET --- Dàn ý theo nhịp

**Mục tiêu:** Chia câu chuyện thành các nhịp nội dung trước khi viết lời
dẫn hoàn chỉnh.

**Đầu vào** - `story-structure.md` - `research-dossier.md`

**Công việc chính** - Chia câu chuyện thành các beat có chức năng
riêng. - Xác định mục tiêu của từng beat: đặt câu hỏi, giải thích, đưa
bằng chứng, chuyển ý, tạo bất ngờ hoặc kết luận. - Ước tính thời lượng
cho từng beat. - Kiểm tra sự tăng tiến của thông tin và tránh lặp ý.

**Đầu ra** - `beat-sheet.md`

**Tiêu chí hoàn thành** - Có thể kể lại toàn bộ video bằng dàn ý mà
không cần lời thoại chi tiết. - Mỗi beat có vai trò rõ ràng trong câu
chuyện.

------------------------------------------------------------------------

## 6. SCRIPTWRITING --- Viết kịch bản

**Mục tiêu:** Chuyển dàn ý thành lời dẫn có thể thu âm hoặc tạo giọng
đọc.

**Đầu vào** - `creative-brief.md` - `story-structure.md` -
`beat-sheet.md` - `research-dossier.md`

**Công việc chính** - Viết lời dẫn đầy đủ. - Làm rõ thuật ngữ và khái
niệm theo trình độ khán giả. - Viết câu nối giữa các đoạn. - Điều chỉnh
nhịp, độ dài câu và cách diễn đạt để phù hợp với lời nói. - Đánh dấu các
thuật ngữ, con số hoặc phát âm cần chú ý.

**Đầu ra** - `script-v1.md`

**Tiêu chí hoàn thành** - Kịch bản có thể được đọc liên tục và truyền
tải được câu chuyện từ đầu đến cuối. - Những khẳng định quan trọng có
thể truy về tài liệu nghiên cứu.

------------------------------------------------------------------------

## 7. SCRIPT REVIEW & LOCK --- Rà soát và chốt kịch bản

**Mục tiêu:** Đảm bảo kịch bản chính xác, mạch lạc, đúng thời lượng mục
tiêu và đủ rõ để chuyển sang thiết kế.

**Đầu vào** - `script-v1.md` - `creative-brief.md` -
`research-dossier.md` - `story-structure.md` - `beat-sheet.md` -
`review-rules.md`

**Công việc chính** 1. **AI Review:** kiểm tra độ chính xác, logic, mức
độ dễ hiểu, nhịp kể, thời lượng và rủi ro khi sản xuất. 2. **Review
Consolidation:** tổng hợp phát hiện, loại bỏ cảnh báo trùng lặp và phân
loại mức độ nghiêm trọng. 3. **Revision & Re-review:** chỉnh sửa và chạy
lại các kiểm tra liên quan. 4. **Human Approval:** người chịu trách
nhiệm nội dung xử lý các vấn đề còn tranh luận và phê duyệt. 5.
**Lock:** lưu phiên bản đã duyệt, đánh dấu phiên bản và ghi nhận các
thay đổi.

**Đầu ra** - `script-final.md` - `script-review-report.md` -
`script-handoff-notes.md` - `script-version.json` hoặc nhật ký phiên bản
tương đương

**Tiêu chí hoàn thành** - Không còn lỗi nghiêm trọng chưa giải quyết. -
Các khẳng định quan trọng đã được kiểm tra nguồn. - Thời lượng nằm trong
giới hạn đã thống nhất. - Người phụ trách nội dung đã duyệt. - Bản cuối
có phiên bản rõ ràng và không bị chỉnh sửa âm thầm sau khi khóa.

**Vai trò của AI** - AI có thể phát hiện vấn đề, đưa ra đề xuất và kiểm
tra lại. - AI không tự thay thế trách nhiệm phê duyệt khoa học hoặc tự
khóa kịch bản mà không có quy tắc và sự chấp thuận phù hợp.

------------------------------------------------------------------------

# PHẦN B --- TIỀN KỲ SẢN XUẤT (PRE-PRODUCTION)

## 8. SCRIPT BREAKDOWN --- Phân rã kịch bản

**Mục tiêu:** Chia kịch bản thành những đơn vị nội dung ổn định để có
thể lập kế hoạch hình ảnh, storyboard và sản xuất.

**Đầu vào** - `script-final.md` - `script-handoff-notes.md`

**Công việc chính** - Chia lời dẫn thành các đoạn/beat có chức năng rõ
ràng. - Gán ID ổn định cho từng beat, ví dụ `B001`, `B002`. - Ghi thời
lượng dự kiến, nội dung chính và mục tiêu người xem cần hiểu. - Đánh dấu
dữ kiện, thuật ngữ hoặc quan hệ nhân quả không được làm sai lệch.

**Đầu ra** - `script-breakdown.md` hoặc `script-breakdown.csv`

**Tiêu chí hoàn thành** - Mỗi đơn vị có ID duy nhất và có thể liên kết
ngược về đoạn kịch bản. - Không làm thay đổi nội dung đã khóa; nếu cần
sửa lời dẫn đáng kể, phải đưa về quy trình quản lý thay đổi kịch bản.

------------------------------------------------------------------------

## 9. VISUAL CONCEPT --- Xác định ngôn ngữ hình ảnh

**Mục tiêu:** Định nghĩa hệ thống quy tắc hình ảnh chung cho toàn video.

**Đầu vào** - `creative-brief.md` - `script-final.md` -
`script-breakdown.md` - Tài liệu tham khảo phong cách nếu có

**Công việc chính** - Chọn phong cách tổng thể và mức độ trừu tượng. -
Xác định cách sử dụng nhân vật, hình khối, biểu tượng, chữ, nền và màu
sắc. - Định nghĩa nguyên tắc về bố cục, độ tương phản, phân cấp thông
tin và tính nhất quán. - Tạo một số mẫu thử nhỏ để xác nhận phong cách
trước khi mở rộng.

**Đầu ra** - `visual-concept.md` - `style-frames/` hoặc
`style-reference/`

**Tiêu chí hoàn thành** - Nhóm có thể dùng cùng một bộ quy tắc để thiết
kế các cảnh khác nhau mà vẫn nhất quán. - Phong cách hỗ trợ mục tiêu
giải thích, không chỉ đẹp về thẩm mỹ.

------------------------------------------------------------------------

## 10. VISUAL PLANNING --- Lên phương án hình ảnh

**Mục tiêu:** Quyết định mỗi beat cần giúp người xem hiểu điều gì và nên
dùng cách thể hiện hình ảnh nào.

**Đầu vào** - `script-breakdown.md` - `visual-concept.md` -
`research-dossier.md`

**Công việc chính** - Xác định mục tiêu truyền đạt của từng beat. - Ghi
thông tin người xem cần nhìn thấy hoặc hiểu được. - Chọn phương án biểu
đạt ở mức khái niệm: mô hình, so sánh, sơ đồ, diễn tiến, ví dụ hoặc phép
ẩn dụ. - Ghi nhận nguy cơ gây hiểu lầm và giới hạn của phép ẩn dụ. -
Liên kết phương án hình ảnh với câu thoại hoặc đoạn lời dẫn tương ứng.

**Đầu ra** - `visual-plan.md`

**Tiêu chí hoàn thành** - Mỗi beat có một mục tiêu hình ảnh rõ ràng. -
Không có đoạn quan trọng chỉ được minh họa trang trí mà không giúp người
xem hiểu nội dung. - Các phép ẩn dụ không làm sai bản chất kiến thức.

------------------------------------------------------------------------

## 11. VISUAL SEQUENCING --- Sắp xếp diễn tiến hình ảnh

**Mục tiêu:** Xây dựng thứ tự xuất hiện và biến đổi của hình ảnh để
người xem tiếp nhận thông tin đúng nhịp.

**Đầu vào** - `visual-plan.md` - `script-breakdown.md` -
`visual-concept.md`

**Công việc chính** - Chia phương án hình ảnh thành chuỗi diễn biến. -
Xác định hình ảnh nào xuất hiện trước, được giữ lại, biến đổi, được so
sánh hoặc được tiết lộ sau. - Xác định cách chuyển từ một trạng thái
thông tin sang trạng thái tiếp theo. - Ghi các mối liên hệ giữa những
cảnh để tránh nhảy ý. - Ước tính thời gian dành cho từng diễn biến.

**Đầu ra** - `visual-sequence.md` hoặc `visual-sequence.csv`

**Tiêu chí hoàn thành** - Có thể hiểu trình tự kể chuyện bằng hình ảnh
mà chưa cần bản vẽ chi tiết. - Mỗi thay đổi hình ảnh có lý do truyền đạt
rõ ràng.

**Phân biệt với Visual Planning:** Visual Planning chọn cách biểu đạt
cho từng ý; Visual Sequencing xác định cách các ý và hình ảnh nối tiếp,
thay đổi và tiết lộ thông tin theo thời gian.

------------------------------------------------------------------------

## 12. STORYBOARD --- Phác thảo từng shot/cảnh

**Mục tiêu:** Chuyển chuỗi diễn biến thành các khung hình có bố cục cụ
thể.

**Đầu vào** - `visual-sequence.md` - `visual-plan.md` -
`visual-concept.md` - `script-breakdown.md`

**Công việc chính** - Chia thành scene/shot theo nhu cầu sản xuất. -
Phác thảo bố cục, chủ thể, vị trí thành phần và góc nhìn. - Ghi hành
động hoặc trạng thái chính của mỗi khung. - Gắn lời dẫn, beat ID và thời
lượng dự kiến. - Đánh dấu các cảnh cần giải thích kỹ thuật hoặc xác minh
khoa học.

**Đầu ra** - `storyboard/` - `storyboard-index.csv`

**Tiêu chí hoàn thành** - Nhóm có thể theo dõi câu chuyện bằng cách xem
storyboard từ đầu đến cuối. - Các khung hình đủ rõ để đánh giá tính dễ
hiểu, bố cục và nhịp kể. - Mỗi shot có ID và liên kết đến đoạn kịch bản.

------------------------------------------------------------------------

## 13. ASSET DESIGN --- Thiết kế tài nguyên hình ảnh

**Mục tiêu:** Xây dựng bộ tài nguyên hình ảnh nhất quán, sẵn sàng đưa
vào sản xuất.

**Đầu vào** - `visual-concept.md` - `storyboard/` - `visual-plan.md` -
`asset-requirements.md` hoặc danh sách tài nguyên rút ra từ storyboard

**Công việc chính** - Lập danh sách tài nguyên cần có. - Thiết kế nhân
vật, vật thể, biểu tượng, background, typography và các thành phần đồ
họa. - Tạo biến thể cần thiết, quy định kích thước, màu sắc, lớp và quy
tắc tái sử dụng. - Đặt tên, phiên bản và tổ chức tài nguyên. - Xác nhận
tài nguyên đáp ứng yêu cầu kỹ thuật của công cụ sản xuất.

**Đầu ra** - `assets/` - `asset-library/` - `asset-manifest.csv` -
`asset-guidelines.md`

**Tiêu chí hoàn thành** - Mỗi tài nguyên cần thiết có trạng thái rõ
ràng. - Tài nguyên nhất quán về phong cách, có thể tìm kiếm và tái sử
dụng. - Các tài nguyên quan trọng đã được duyệt trước khi dùng hàng
loạt.

------------------------------------------------------------------------

## 14. MOTION PLANNING --- Thiết kế chuyển động

**Mục tiêu:** Xác định cách hình ảnh thay đổi theo thời gian trước khi
thực hiện animation hoàn chỉnh.

**Đầu vào** - `storyboard/` - `visual-sequence.md` - `script-final.md` -
`asset-manifest.csv`

**Công việc chính** - Xác định thứ tự xuất hiện, biến mất, di chuyển và
biến đổi. - Ghi quan hệ giữa lời dẫn và hành động hình ảnh. - Xác định
chuyển cảnh, nhịp chuyển động, điểm nhấn và thời gian giữ khung. - Định
nghĩa các quy tắc chuyển động lặp lại. - Xác định các cảnh có rủi ro kỹ
thuật hoặc cần thử nghiệm.

**Đầu ra** - `motion-spec.md` - `motion-board/` nếu cần -
`shot-timing.csv`

**Tiêu chí hoàn thành** - Đội animation hiểu trạng thái đầu, hành động
chính và trạng thái cuối của từng shot. - Các hành động quan trọng gắn
được với lời dẫn hoặc mục tiêu truyền đạt. - Các rủi ro kỹ thuật đã được
đánh dấu trước khi sản xuất.

------------------------------------------------------------------------

## 15. ANIMATIC & PRE-PRODUCTION REVIEW --- Dựng animatic và duyệt tiền kỳ

**Mục tiêu:** Kiểm tra toàn bộ cách kể chuyện bằng hình ảnh tĩnh/đơn
giản, lời dẫn và thời lượng trước khi đầu tư vào animation hoàn chỉnh.

**Đầu vào** - `storyboard/` - `script-final.md` hoặc bản thu lời dẫn
thử - `shot-timing.csv` - `motion-spec.md`

**Công việc chính** - Ghép storyboard theo thứ tự thời gian. - Đồng bộ
với lời dẫn tạm thời và thời lượng dự kiến. - Dùng chuyển động đơn giản
hoặc placeholder ở những chỗ cần kiểm tra nhịp. - Kiểm tra tính dễ hiểu,
nhịp kể, chuyển cảnh, thời lượng và những đoạn hình ảnh chưa đủ thông
tin. - Ghi các yêu cầu sửa đổi và phê duyệt gói tiền kỳ.

**Đầu ra** - `animatic.mp4` - `animatic-review.md` -
`preproduction-approval.md`

**Tiêu chí hoàn thành** - Video có thể được xem từ đầu đến cuối để đánh
giá nhịp kể. - Những lỗi lớn về cấu trúc cảnh, thời lượng và mức độ dễ
hiểu đã được giải quyết. - Các tài liệu và tài nguyên đủ rõ để bắt đầu
sản xuất.

**Lưu ý:** Animatic không nhất thiết phải có hình đẹp. Nó là công cụ
kiểm tra cách kể chuyện và thời gian, không phải video cuối.

------------------------------------------------------------------------

# PHẦN C --- SẢN XUẤT (PRODUCTION)

## 16. VISUAL / ASSET PRODUCTION --- Tạo hình và hoàn thiện tài nguyên

**Mục tiêu:** Tạo các tài nguyên hình ảnh cuối cùng theo gói tiền kỳ đã
duyệt.

**Đầu vào** - `preproduction-approval.md` - `assets/` và
`asset-manifest.csv` - `storyboard/` - `visual-concept.md`

**Công việc chính** - Hoàn thiện hình ảnh, vector, background, nhân vật,
biểu đồ và các thành phần cần thiết. - Chuẩn hóa layer, naming, kích
thước và định dạng file. - Tạo hoặc điều chỉnh tài nguyên theo yêu cầu
kỹ thuật của công cụ animation. - Cập nhật trạng thái tài nguyên và ghi
nhận các thay đổi.

**Đầu ra** - Bộ asset production-ready - `asset-manifest.csv` được cập
nhật

**Tiêu chí hoàn thành** - Tài nguyên đạt chất lượng và định dạng cần
thiết để animate. - Không còn thiếu tài nguyên bắt buộc cho các shot đã
duyệt.

**Phân biệt:** Asset Design trong tiền kỳ định nghĩa và thiết kế hệ
thống tài nguyên; Asset Production hoàn thiện hoặc tạo ra các file thực
tế dùng trong video.

------------------------------------------------------------------------

## 17. ANIMATION PRODUCTION --- Sản xuất animation

**Mục tiêu:** Biến storyboard, tài nguyên và đặc tả chuyển động thành
các shot có chuyển động hoàn chỉnh.

**Đầu vào** - `storyboard/` - `motion-spec.md` - Asset
production-ready - `script-final.md` - `shot-timing.csv`

**Công việc chính** - Dựng từng shot theo thứ tự sản xuất. - Thực hiện
chuyển động, biến đổi, hiệu ứng và đồng bộ lời dẫn. - Duy trì quy tắc
hình ảnh và chuyển động. - Xuất bản xem trước theo từng shot hoặc từng
phân đoạn. - Theo dõi trạng thái, lỗi và phụ thuộc giữa các shot.

**Đầu ra** - Các shot animation đã render - `shot-status.csv` - Bản xem
trước theo phân đoạn

**Tiêu chí hoàn thành** - Các shot đáp ứng storyboard và motion spec đã
duyệt. - Chuyển động, timing và thông tin khoa học chính xác. - Các shot
có thể ghép nối trong quy trình hậu kỳ.

------------------------------------------------------------------------

## 18. SHOT REVIEW & FIXES --- Duyệt shot và sửa lỗi

**Mục tiêu:** Kiểm tra chất lượng từng shot trước khi đưa vào bản dựng
cuối.

**Đầu vào** - Các shot animation đã render - Storyboard, motion spec và
kịch bản - Tiêu chí chất lượng của dự án

**Công việc chính** - Kiểm tra lỗi hình ảnh, chuyển động, chữ, timing,
tính nhất quán và độ chính xác. - Ghi nhận lỗi theo shot ID, mức độ và
người phụ trách. - Sửa và render lại. - Duyệt shot trước khi chuyển sang
hậu kỳ.

**Đầu ra** - Các shot đã duyệt - `shot-review-log.csv`

**Tiêu chí hoàn thành** - Không còn lỗi nghiêm trọng trong các shot được
bàn giao. - Các vấn đề còn lại đã được ghi nhận và chấp thuận rõ ràng.

------------------------------------------------------------------------

# PHẦN D --- HẬU KỲ (POST-PRODUCTION)

## 19. EDITING & COMPOSITING --- Dựng và tổng hợp hình ảnh

**Mục tiêu:** Ghép các shot thành một video liên tục và hoàn thiện lớp
hình ảnh cuối.

**Đầu vào** - Các shot đã duyệt - Lời dẫn chính thức - Kế hoạch thời
lượng và thứ tự shot

**Công việc chính** - Ghép shot theo timeline. - Điều chỉnh nhịp dựng,
khoảng nghỉ, chuyển cảnh và thời lượng. - Tổng hợp các lớp hình ảnh,
hiệu ứng, typography và đồ họa nếu cần. - Đồng bộ hình ảnh với lời
dẫn. - Kiểm tra tính liên tục và sự nhất quán giữa các shot.

**Đầu ra** - `picture-lock` hoặc bản dựng hình đã chốt - File project
dựng và các thành phần compositing liên quan

**Tiêu chí hoàn thành** - Nội dung hình ảnh chạy liên tục theo đúng thứ
tự. - Không có khoảng trống, cắt nối lỗi hoặc sai lệch timeline chưa xử
lý.

------------------------------------------------------------------------

## 20. SOUND DESIGN & MIXING --- Thiết kế và hòa âm âm thanh

**Mục tiêu:** Hoàn thiện trải nghiệm âm thanh và bảo đảm lời dẫn rõ
ràng.

**Đầu vào** - Bản dựng hình - Lời dẫn/voice-over chính thức - Nhạc nền
và hiệu ứng âm thanh đã được cấp phép hoặc tự tạo

**Công việc chính** - Làm sạch và chỉnh sửa lời dẫn. - Đồng bộ lời dẫn,
nhạc nền và hiệu ứng âm thanh. - Điều chỉnh âm lượng, khoảng nghỉ và cân
bằng âm thanh. - Kiểm tra độ rõ của lời nói và sự phù hợp với hình
ảnh. - Áp dụng yêu cầu âm lượng/định dạng của nền tảng phát hành.

**Đầu ra** - `final-mix.wav` hoặc định dạng âm thanh theo yêu cầu - Bản
dựng video có âm thanh hoàn chỉnh

**Tiêu chí hoàn thành** - Lời dẫn nghe rõ, không bị nhạc hoặc hiệu ứng
lấn át. - Âm thanh đồng bộ với hình ảnh và không có lỗi đột ngột.

**Lưu ý:** Một số dự án tạo voice-over sớm để phục vụ animatic và
animation. Bước này là hoàn thiện âm thanh cuối cùng, không nhất thiết
là lần đầu tạo lời dẫn.

------------------------------------------------------------------------

## 21. FINAL QUALITY CONTROL --- Kiểm tra chất lượng cuối

**Mục tiêu:** Đảm bảo video hoàn chỉnh đáp ứng yêu cầu nội dung, hình
ảnh, âm thanh và kỹ thuật.

**Đầu vào** - Bản dựng hoàn chỉnh - Creative brief - Checklist chất
lượng - Yêu cầu kỹ thuật của nền tảng phát hành

**Công việc chính** - Xem toàn bộ video từ đầu đến cuối. - Kiểm tra nội
dung, phát âm, phụ đề, chính tả, tên riêng, con số và nguồn trích dẫn
khi cần. - Kiểm tra hình ảnh, khung hình, lỗi render, nhấp nháy, mất
asset và sai lệch đồng bộ. - Kiểm tra âm thanh, mức âm lượng, clipping
và khoảng lặng bất thường. - Kiểm tra định dạng, độ phân giải, frame
rate, tỷ lệ khung hình và các yêu cầu phân phối. - Ghi lỗi, sửa và chạy
QC lại.

**Đầu ra** - `final-qc-report.md` - Bản video được duyệt phát hành

**Tiêu chí hoàn thành** - Không còn lỗi nghiêm trọng về nội dung hoặc kỹ
thuật. - Tất cả lỗi bắt buộc đã được sửa và xác nhận.

------------------------------------------------------------------------

## 22. EXPORT & DELIVERY --- Xuất bản và bàn giao

**Mục tiêu:** Tạo các file cuối đúng chuẩn và tổ chức tài liệu để phát
hành hoặc lưu trữ.

**Đầu vào** - Video đã qua QC - Cấu hình xuất bản - Danh sách tài liệu
cần bàn giao

**Công việc chính** - Xuất video theo thông số kỹ thuật của nền tảng. -
Kiểm tra file xuất thực tế sau khi render. - Chuẩn bị phụ đề, thumbnail,
mô tả, credit và tài liệu nguồn nếu thuộc phạm vi dự án. - Lưu project,
asset, file âm thanh, báo cáo QC và phiên bản kịch bản. - Ghi phiên bản
phát hành và ngày bàn giao.

**Đầu ra** - `final-video.mp4` hoặc định dạng được yêu cầu - File phụ đề
và tài liệu phát hành - Gói project/archive - `delivery-manifest.md`

**Tiêu chí hoàn thành** - File mở và phát được, thông số đúng yêu cầu. -
Tài liệu và tài nguyên cần thiết đã được bàn giao, có thể truy xuất.

------------------------------------------------------------------------

# 3. Bảng tóm tắt Input / Output

  ------------------------------------------------------------------------------------------
                 \# Bước             Giai đoạn         Input chính   Output chính
  ----------------- ---------------- ----------------- ------------- -----------------------
                  1 IDEA             Development       Ý tưởng sơ bộ `idea.md`

                  2 CREATIVE BRIEF   Development       Ý tưởng       `creative-brief.md`

                  3 RESEARCH &       Development       Brief, ý      Research dossier,
                    FACT-CHECK                         tưởng         source register

                  4 STORY            Development       Brief,        `story-structure.md`
                    DEVELOPMENT                        research      

                  5 OUTLINE / BEAT   Development       Story         `beat-sheet.md`
                    SHEET                              structure,    
                                                       research      

                  6 SCRIPTWRITING    Development       Beat sheet,   `script-v1.md`
                                                       research      

                  7 SCRIPT REVIEW &  Development       Script,       `script-final.md`,
                    LOCK                               brief,        review report
                                                       research,     
                                                       rules         

                  8 SCRIPT BREAKDOWN Pre-production    Script final, `script-breakdown.md`
                                                       handoff notes 

                  9 VISUAL CONCEPT   Pre-production    Brief, script `visual-concept.md`,
                                                       breakdown     style frames

                 10 VISUAL PLANNING  Pre-production    Breakdown,    `visual-plan.md`
                                                       concept,      
                                                       research      

                 11 VISUAL           Pre-production    Visual plan,  `visual-sequence.md`
                    SEQUENCING                         breakdown     

                 12 STORYBOARD       Pre-production    Visual        Storyboard, shot index
                                                       sequence,     
                                                       concept       

                 13 ASSET DESIGN     Pre-production    Storyboard,   Asset specs, asset
                                                       visual        library
                                                       concept       

                 14 MOTION PLANNING  Pre-production    Storyboard,   `motion-spec.md`, shot
                                                       assets,       timing
                                                       script        

                 15 ANIMATIC &       Pre-production    Storyboard,   `animatic.mp4`,
                    PRE-PRODUCTION                     script,       approval
                    REVIEW                             timing        

                 16 VISUAL / ASSET   Production        Approved      Production-ready assets
                    PRODUCTION                         design        
                                                       package       

                 17 ANIMATION        Production        Assets,       Rendered shots
                    PRODUCTION                         storyboard,   
                                                       motion spec   

                 18 SHOT REVIEW &    Production        Rendered      Approved shots, review
                    FIXES                              shots, specs  log

                 19 EDITING &        Post-production   Approved      Picture lock /
                    COMPOSITING                        shots,        assembled video
                                                       voice-over    

                 20 SOUND DESIGN &   Post-production   Picture edit, Final mix
                    MIXING                             voice, music, 
                                                       SFX           

                 21 FINAL QUALITY    Post-production   Complete      QC report, approved
                    CONTROL                            video, QC     master
                                                       checklist     

                 22 EXPORT &         Delivery          Approved      Final video, delivery
                    DELIVERY                           master,       package
                                                       export spec   
  ------------------------------------------------------------------------------------------

------------------------------------------------------------------------

# 4. Các cổng duyệt (approval gates)

Không nhất thiết phải duyệt thủ công mọi file nhỏ. Nên có các cổng duyệt
tại những điểm mà thay đổi sẽ gây tốn kém nếu phát hiện muộn.

## Gate A --- Script Lock

-   Kịch bản đã được kiểm tra và người phụ trách nội dung phê duyệt.
-   Thay đổi sau đó cần được ghi nhận.

## Gate B --- Pre-production Approval

-   Visual concept, visual plan, visual sequence, storyboard, asset plan
    và motion spec đủ rõ.
-   Animatic đã được xem và các vấn đề lớn đã giải quyết.

## Gate C --- Production Approval

-   Các shot đã đạt tiêu chí chất lượng và sẵn sàng ghép dựng.
-   Các lỗi nghiêm trọng đã được sửa.

## Gate D --- Final Delivery Approval

-   Bản dựng hoàn chỉnh đã qua QC.
-   File xuất đúng thông số và gói bàn giao đầy đủ.

------------------------------------------------------------------------

# 5. Những nguyên tắc thiết kế pipeline

1.  **Mỗi bước có đầu ra rõ ràng.** Không chuyển bước chỉ vì đã hoàn
    thành một tác vụ; cần kiểm tra đầu ra có đủ điều kiện sử dụng hay
    chưa.
2.  **Dùng ID ổn định.** Beat ID, scene ID, shot ID và asset ID giúp
    truy vết từ kịch bản đến video cuối.
3.  **Tách nội dung khỏi cách thể hiện.** Kịch bản mô tả điều cần nói;
    visual plan mô tả cách diễn đạt bằng hình ảnh; storyboard cụ thể hóa
    bố cục; motion spec mô tả sự thay đổi theo thời gian.
4.  **Cho phép vòng lặp có kiểm soát.** Nếu animatic phát hiện vấn đề,
    quay lại đúng bước liên quan và ghi nhận thay đổi.
5.  **Dùng AI để hỗ trợ, không mặc định thay người duyệt.** AI phù hợp
    để phân tích, tạo bản nháp, kiểm tra tính nhất quán và báo lỗi;
    quyết định về độ chính xác khoa học và phê duyệt cuối cần có trách
    nhiệm rõ ràng.
6.  **Không cần tài liệu hóa quá mức.** Với video ngắn hoặc ít phức tạp,
    có thể gộp một số đầu ra vào cùng một file, miễn các mục đích và
    tiêu chí duyệt vẫn rõ ràng.

------------------------------------------------------------------------

# 6. Cấu trúc thư mục tham khảo

``` text
video-project/
├── 01-development/
│   ├── idea.md
│   ├── creative-brief.md
│   ├── research/
│   │   ├── research-dossier.md
│   │   ├── source-register.csv
│   │   └── open-questions.md
│   ├── story-structure.md
│   ├── beat-sheet.md
│   ├── script-v1.md
│   ├── script-final.md
│   ├── script-review-report.md
│   └── script-handoff-notes.md
├── 02-preproduction/
│   ├── script-breakdown.md
│   ├── visual-concept.md
│   ├── visual-plan.md
│   ├── visual-sequence.md
│   ├── storyboard/
│   ├── asset-design/
│   ├── asset-manifest.csv
│   ├── motion-spec.md
│   ├── shot-timing.csv
│   ├── animatic.mp4
│   └── preproduction-approval.md
├── 03-production/
│   ├── assets/
│   ├── shots/
│   ├── previews/
│   └── shot-status.csv
├── 04-postproduction/
│   ├── edit-project/
│   ├── audio/
│   ├── compositing/
│   ├── final-qc-report.md
│   └── exports/
└── 05-delivery/
    ├── final-video.mp4
    ├── subtitles/
    ├── thumbnail/
    └── delivery-manifest.md
```

## 7. Phiên bản tinh gọn để bắt đầu

Nếu bắt đầu với một quy trình sản xuất nhỏ, có thể giữ nguyên 22 bước về
mặt logic nhưng chưa cần tạo 22 file riêng. Các tài liệu có thể gộp như
sau:

-   `development-pack.md`: idea, brief, story structure, beat sheet.
-   `research-dossier.md`: nghiên cứu và nguồn.
-   `script-final.md`: kịch bản đã khóa.
-   `visual-design-pack.md`: breakdown, visual concept, visual plan và
    visual sequence.
-   `storyboard/`: storyboard theo shot.
-   `production-pack.md`: asset manifest, motion spec và thông tin sản
    xuất.
-   `animatic.mp4`: kiểm tra nhịp kể.
-   `shot-status.csv`: trạng thái sản xuất và duyệt shot.
-   `final-qc-report.md`: kiểm tra chất lượng cuối.
-   `delivery-manifest.md`: danh sách file bàn giao.

**Nguyên tắc cuối cùng:** Quy trình nên đủ chi tiết để biết mỗi bước cần
nhận gì, tạo ra gì và khi nào được xem là hoàn thành; nhưng không nên
biến việc tạo tài liệu thành mục tiêu thay cho việc làm ra một video
tốt.
