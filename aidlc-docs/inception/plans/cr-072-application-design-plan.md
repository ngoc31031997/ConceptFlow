# CR-072 — Kế hoạch Application Design

Trạng thái: **đã duyệt** (Creator, 2026-10-13: "tôi duyệt hết rồi"). Câu trả lời AD-1..AD-9 đủ, không có chỗ mơ hồ; tài liệu ở `aidlc-docs/inception/application-design/cr-072/`; ADR-0034..0041 Accepted.

## Date
2026-10-13

Đầu vào: yêu cầu `aidlc-docs/inception/requirements/cr-072-2d-film-pipeline.md` (FR-1..FR-26), kế hoạch `aidlc-docs/inception/plans/cr-072-workflow-plan.md` (U1..U9). Application Design chỉ chốt **thành phần, trách nhiệm, giao diện và cách các service nói chuyện với nhau**. Luật chi tiết, prompt, schema cụ thể để Functional Design của từng CR con.

Tài liệu dự án đã có ở `aidlc-docs/inception/application-design/` (mô tả các service hiện tại). Tài liệu của CR-072 đặt riêng ở `aidlc-docs/inception/application-design/cr-072/` để không đè lên.

## Hiện trạng liên quan (đã kiểm trong code)

- **authoring-service** giữ mọi thứ AI soạn: prompt, kiểu video, thư viện hình (`illustrations`, `illustration_folders`), hình của project (`project_illustrations`), đoạn code (`authoring_segments`), nội dung soạn (`project_authoring`), usage (`llm_usage`). Ranh giới theo ADR-0029.
- **orchestrator** giữ project, luồng 13 bước và saga render. Saga gửi lệnh tới tts, rendering, video-assembly qua RabbitMQ; tiến độ đi qua `progress.fanout` tới SSE.
- **rendering** có Remotion, kit (`rig.tsx`, `illustration.tsx`, `backdrops.tsx`, `scene.tsx`), đã dùng `renderStill` cho ảnh xem trước hình (`/v1/illustrations/preview`).
- **tts, rendering, video-assembly** mỗi service có DB riêng (inbox/outbox), nhận lệnh qua RabbitMQ.
- Định nghĩa luồng nằm ở ba chỗ (`orchestrator/internal/domain/flow.go`, `authoring-service/internal/domain/flow.go`, `web-gui/src/utils/flow.ts`), có test hợp đồng giữ khớp nhau (CR-051).

## Kế hoạch

- [x] Trả lời các câu hỏi AD-1..AD-9 bên dưới (Creator)
- [x] Rà câu trả lời, hỏi thêm nếu còn mơ hồ
- [x] `cr-072/components.md`: thành phần mới và thành phần đổi, trách nhiệm, thuộc service nào
- [x] `cr-072/component-methods.md`: chữ ký phương thức và API ở mức cao (đầu vào, đầu ra)
- [x] `cr-072/services.md`: điều phối giữa các service cho từng chặng của luồng
- [x] `cr-072/component-dependency.md`: ma trận phụ thuộc, kiểu giao tiếp (HTTP, RabbitMQ, `/shared`), luồng dữ liệu
- [x] `cr-072/application-design.md`: gom các tài liệu trên, chỉ ra ADR
- [x] ADR (bản nháp, duyệt cùng thiết kế):
  - ADR-0034: luồng ba giai đoạn và bộ thiết kế mỹ thuật
  - ADR-0035: khung sửa theo đơn vị và đồ thị phụ thuộc
  - ADR-0036: ngôn ngữ hình của kênh
  - ADR-0037: phong cách theo thế giới, thay FR7 nền sáng của CR-067
  - ADR-0038: render theo cảnh có đoạn chồng
  - ADR-0039: một API ảnh tĩnh chung; animatic phát trong trình duyệt (AD-4, AD-5)
  - ADR-0040: giọng sơ bộ qua RabbitMQ; từ điển phát âm ở authoring-service (AD-6, AD-7)
  - ADR-0041: `/shared` là kho file, dữ liệu chỉ giữ đường dẫn (đề nghị của Creator)
- [x] Kiểm tính đầy đủ: mọi FR-1..FR-26 có thành phần đảm nhận; mỗi CR con U1..U9 biết mình đụng thành phần nào

---

## Câu hỏi

## Question AD-1 — Ai giữ "đơn vị duyệt" (FR-17)

Đơn vị là thứ Creator duyệt, ghi chú hoặc cho làm lại: logline, beat, cảnh lời thoại, mục mỹ thuật, cảnh storyboard, color key, khung mẫu, câu giọng, shot animatic, shot code, cảnh render.

A) authoring-service giữ mọi đơn vị của tiền kỳ và sản xuất; orchestrator giữ đơn vị của hậu kỳ (cảnh render, ghi chú dựng). Cả hai dùng cùng một mô hình: trạng thái, ghi chú, dấu đầu vào. web-gui đọc qua api-gateway như hiện nay (AI đề xuất).
   - ✅ Ưu: đúng ranh giới ADR-0029 (authoring-service giữ những gì AI soạn, orchestrator giữ saga); không thêm service.
   - ⚠️ Nhược: mô hình đơn vị có ở hai service, phải giữ khớp bằng test.

B) Thêm một service mới chuyên giữ trạng thái duyệt và ghi chú cho mọi bước.
   - ✅ Ưu: một chỗ duy nhất.
   - ⚠️ Nhược: thêm một service để chạy và deploy; mọi service phải gọi sang nó; nội dung nằm một nơi, trạng thái nằm nơi khác.

C) orchestrator giữ hết.
   - ⚠️ Nhược: orchestrator phải biết chi tiết nội dung soạn, trái ADR-0029.

X) Other (please describe after [Answer]: tag below)

[Answer]: A

## Question AD-2 — Cách biết phần nào phải làm lại (FR-17)

A) Mỗi đơn vị ghi dấu (hash) của các đầu vào nó được dựng từ. Đầu vào đổi thì đơn vị tự thành "cũ", màn hình hiện "cần làm lại" và liệt kê trước những gì sẽ làm lại. CR-050 đã làm đúng cách này cho danh sách hình (`illustrations_storyboard_sha`); đây là mở rộng cho mọi đơn vị (AI đề xuất).
   - ✅ Ưu: đúng cả khi Creator sửa tay; không phải viết luật xoá riêng cho từng thao tác; không xoá nhầm phần không liên quan.
   - ⚠️ Nhược: mỗi loại đơn vị phải khai báo đầu vào của nó.

B) Luật xoá viết tay theo từng thao tác (sửa cảnh 3 thì xoá X, Y, Z), như các hộp xác nhận hiện nay.
   - ⚠️ Nhược: dễ sót; thêm thao tác mới là phải thêm luật; hiện nay chính cách này làm đổi dàn ý thì xoá cả video.

X) Other (please describe after [Answer]: tag below)

[Answer]: A

## Question AD-3 — Ngôn ngữ hình có cả luật (chữ) lẫn rig (code) thì lưu thế nào (FR-18)

A) Luật hình, luật nền, mô tả lưu DB ở authoring-service, có phiên bản, như prompt. Phần code (rig cơ sở, hình nền tảng của ngôn ngữ) là một bộ kit trong project Remotion của rendering, mỗi ngôn ngữ hệ thống một kit. Ngôn ngữ do Creator sao chép thì dùng lại kit của ngôn ngữ gốc, chỉ sửa luật (AI đề xuất).
   - ✅ Ưu: code rig được kiểm kiểu và test lúc build như hiện nay; luật sửa được trên giao diện.
   - ⚠️ Nhược: muốn một ngôn ngữ có cách vẽ rig khác hẳn thì phải viết kit mới, tức là một CR.

B) Lưu cả code rig dạng chữ trong DB, rendering nạp lúc chạy.
   - ✅ Ưu: thêm ngôn ngữ không cần build lại.
   - ⚠️ Nhược: code không qua kiểm tra lúc build; lỗi chỉ lộ khi render.

X) Other (please describe after [Answer]: tag below)

[Answer]: A

## Question AD-4 — Ảnh tĩnh dùng ở nhiều nơi

Ảnh mẫu ngôn ngữ hình, bộ mỹ thuật, khung mẫu, khung animatic, thumbnail đều là ảnh tĩnh render bằng Remotion.

A) Một API ảnh tĩnh chung ở rendering: nhận composition và props, trả PNG, có hàng đợi và cache theo nội dung. Mở rộng từ `/v1/illustrations/preview` đang có (AI đề xuất).
   - ✅ Ưu: một chỗ để tối ưu tốc độ và cache; thêm loại ảnh mới chỉ là thêm composition.

B) Mỗi tính năng một endpoint riêng.
   - ⚠️ Nhược: lặp code hàng đợi và cache.

X) Other (please describe after [Answer]: tag below)

[Answer]: A

## Question AD-5 — Animatic là gì về mặt kỹ thuật (FR-10)

A) rendering dựng thành video 480p bằng Remotion: khung tĩnh, giọng sơ bộ, phụ đề.
   - ✅ Ưu: có file video để xem lại hay chia sẻ; phụ đề giống bản cuối.
   - ⚠️ Nhược: mỗi lần sửa một shot phải dựng lại (dù chỉ đoạn đó); tốn CPU của rendering.

B) video-assembly ghép bằng ffmpeg: ảnh tĩnh, âm thanh, phụ đề.
   - ✅ Ưu: nhẹ, nhanh.
   - ⚠️ Nhược: phụ đề khác kiểu với bản cuối; vẫn phải ghép lại khi sửa.

C) 💡 Gợi ý: không dựng video. web-gui phát animatic ngay trong trình duyệt: ảnh tĩnh từng shot, file giọng từng câu, phụ đề theo mốc thời gian (AI đề xuất).
   - ✅ Ưu: không tốn lượt render; sửa thời lượng, câu hay khung tĩnh là thấy ngay; chỉ ảnh tĩnh của shot bị sửa phải render lại.
   - ⚠️ Nhược: không có file video animatic để tải về.

X) Other (please describe after [Answer]: tag below)

[Answer]: C

## Question AD-6 — Giọng sơ bộ chạy sớm (FR-9)

A) Khi storyboard được duyệt, orchestrator gửi lệnh đọc cho các câu qua RabbitMQ như saga hiện nay, nhưng là lệnh riêng, không thuộc saga render. File lưu ở `/shared`; bước TTS sau dùng lại theo nội dung câu (AI đề xuất).
   - ✅ Ưu: tts giữ một kiểu giao tiếp (RabbitMQ); có sẵn hàng đợi, thử lại, tiến độ.

B) authoring-service gọi thẳng tts qua HTTP (endpoint mới ở tts).
   - ⚠️ Nhược: tts hiện chỉ nhận lệnh qua RabbitMQ; thêm HTTP là thêm một kiểu giao tiếp, một đường lỗi mới.

X) Other (please describe after [Answer]: tag below)

[Answer]: A

## Question AD-7 — Từ điển phát âm nằm ở đâu (FR-20)

A) authoring-service lưu từ điển cùng các cài đặt kênh khác. Khi gửi lệnh đọc, mỗi câu đã kèm sẵn cách đọc (đọc là, tốc độ, nghỉ) sau khi tra từ điển; tts chỉ chuyển thành SSML (AI đề xuất).
   - ✅ Ưu: tts không cần biết từ điển; giọng dùng lại theo nội dung câu và cách đọc, nên đổi từ điển thì đúng các câu bị ảnh hưởng tự đọc lại.

B) tts lưu và tự tra từ điển lúc đọc.
   - ⚠️ Nhược: cache giọng phải tính cả phiên bản từ điển; giao diện phải gọi thêm một service để sửa từ điển.

X) Other (please describe after [Answer]: tag below)

[Answer]: A

## Question AD-8 — Render từng cảnh (FR-14)

A) orchestrator gửi một lệnh render cho mỗi cảnh. rendering xử lý song song theo số worker cấu hình. orchestrator giữ trạng thái từng cảnh (chờ, chạy, lỗi, xong) để chạy lại hay huỷ riêng từng cảnh (AI đề xuất).
   - ✅ Ưu: chạy lại, huỷ, tự thử lại theo từng cảnh là việc của saga, đúng chỗ đang làm; dễ báo tiến độ từng cảnh.
   - ⚠️ Nhược: saga render đổi nhiều (một bước thành nhiều lệnh con).

B) Một lệnh render cả video như hiện nay; rendering tự chia cảnh bên trong và báo tiến độ từng cảnh.
   - ⚠️ Nhược: chạy lại một cảnh phải gửi lại lệnh cả video kèm danh sách cảnh; orchestrator không thấy trạng thái từng cảnh.

X) Other (please describe after [Answer]: tag below)

[Answer]: A

## Question AD-9 — Định nghĩa luồng 17 bước

A) Giữ ba bản (orchestrator, authoring-service, web-gui) cùng test hợp đồng giữ chúng khớp nhau, như CR-051 đang làm (AI đề xuất).
   - ✅ Ưu: cách đang chạy ổn; mỗi service tự đủ, không phụ thuộc thêm.
   - ⚠️ Nhược: đổi luồng phải sửa ba chỗ (test bắt lệch).

B) Một nguồn duy nhất: orchestrator trả định nghĩa luồng qua API; authoring-service và web-gui đọc từ đó.
   - ✅ Ưu: không thể lệch.
   - ⚠️ Nhược: web-gui phải nạp định nghĩa trước khi vẽ thanh bước; authoring-service phụ thuộc thêm orchestrator.

X) Other (please describe after [Answer]: tag below)

[Answer]: A
