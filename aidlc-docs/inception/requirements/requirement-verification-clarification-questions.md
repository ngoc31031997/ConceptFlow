# Requirements Clarification — Follow-up Questions

Cảm ơn bạn đã trả lời. Tôi phát hiện một số điểm cần làm rõ thêm trước khi viết tài liệu requirements chính thức.

## Ambiguity 1: Kiến trúc mở rộng cho các chủ đề giáo dục khác (không chỉ lập trình)
Ở Câu 2, bạn nói rõ: nội dung lập trình chỉ là **một implement cụ thể**; mục tiêu dài hạn là công cụ giáo dục đa lĩnh vực (ví dụ sau này chuyển sang dạy Tiếng Anh). Điều này ảnh hưởng lớn đến kiến trúc — công cụ cần được thiết kế theo dạng **plugin/pluggable content-type** ngay từ đầu, thay vì hard-code riêng cho lập trình.

### Clarification Question 1
Với ràng buộc kiến trúc mở rộng đa lĩnh vực này, bạn muốn xử lý ở mức độ nào trong giai đoạn đầu?

A) Thiết kế kiến trúc plugin ngay từ đầu (interface/abstract base cho "content type", "scene generator", v.v.), nhưng chỉ **implement** plugin cho lập trình (thuật toán + khái niệm lập trình) trong MVP
   - ✅ Strengths: tránh phải refactor lớn sau này khi thêm domain mới (Tiếng Anh, toán, v.v.)
   - ⚠️ Trade-offs: MVP tốn thêm thời gian thiết kế abstraction thay vì code thẳng

B) Tập trung 100% vào lập trình trước, không thiết kế abstraction đa lĩnh vực ngay — chấp nhận refactor kiến trúc khi thực sự cần mở rộng sang domain khác
   - ✅ Strengths: MVP ra nhanh nhất có thể
   - ⚠️ Trade-offs: rủi ro phải viết lại phần lớn kiến trúc khi mở rộng sang Tiếng Anh/domain khác

X) Other (please describe after [Answer]: tag below)

[Answer]: A

## Ambiguity 2: Giao diện GUI có nằm trong MVP không?
Câu 4 bạn chọn giao diện Web/Desktop GUI (C), nhưng Câu 9 bạn chọn MVP là "pipeline end-to-end tối thiểu: script/markdown → video có giọng đọc" (B) — không đề cập GUI. Cần làm rõ thứ tự ưu tiên.

### Clarification Question 2
GUI sẽ được xây dựng ở giai đoạn nào?

A) MVP chỉ cần chạy qua CLI/script (input là file markdown/script, output là video) — GUI là giai đoạn sau, sau khi pipeline lõi hoạt động ổn định

B) GUI cần có ngay trong MVP — người dùng phải có thể tạo video mà không cần chạy lệnh dòng lệnh ngay từ bản đầu tiên

C) Other (please describe after [Answer]: tag below)

[Answer]: B

## Ambiguity 3: Phạm vi "xuất bản lên YouTube" trong MVP
Câu 1 bạn chọn (C) — pipeline sản xuất đầy đủ, mô tả có nhắc "xuất bản lên YouTube". Câu 9 (MVP) không nhắc đến việc publish.

### Clarification Question 3
Việc tự động đăng video lên YouTube có nằm trong phạm vi MVP không?

A) Không — MVP chỉ cần xuất ra file video hoàn chỉnh (.mp4) trên máy local; việc đăng tải do người dùng tự làm thủ công

B) Có — MVP cần tích hợp API YouTube để tự động upload sau khi render xong

C) Other (please describe after [Answer]: tag below)

[Answer]: B

## Ambiguity 4: Text-to-Speech — nhà cung cấp và ngôn ngữ
Câu 5 bạn chọn TTS tự động (B), Câu 6 bạn chọn hỗ trợ song ngữ Việt/Anh (C). TTS tiếng Việt chất lượng tốt thường cần dịch vụ cloud (trả phí), trong khi TTS local/offline (vd. Coqui, Piper) hỗ trợ tiếng Việt hạn chế hơn.

### Clarification Question 4
Bạn ưu tiên loại TTS nào?

A) 💡 Suggested: Dịch vụ Cloud TTS trả phí (Google Cloud TTS, Azure TTS, ElevenLabs, Amazon Polly) — chất lượng giọng đọc tự nhiên cao, hỗ trợ tốt cả tiếng Việt và tiếng Anh
   - ✅ Strengths: chất lượng giọng tốt, ít công sức tích hợp, hỗ trợ đa ngôn ngữ tốt
   - ⚠️ Trade-offs: phát sinh chi phí theo lượng ký tự, cần API key, phụ thuộc internet

B) TTS mã nguồn mở/offline (Coqui TTS, Piper, v.v.) — chạy hoàn toàn local, miễn phí
   - ✅ Strengths: miễn phí, không cần internet, không giới hạn số lượng
   - ⚠️ Trade-offs: chất lượng giọng tiếng Việt hạn chế hơn, cần tài nguyên máy tính để chạy model

C) Cả hai — cho phép cấu hình chọn provider TTS (pluggable), mặc định dùng open-source, có thể chuyển sang cloud khi cần chất lượng cao hơn
   - ✅ Strengths: linh hoạt nhất, phù hợp với định hướng kiến trúc pluggable đã nêu ở Ambiguity 1
   - ⚠️ Trade-offs: tốn thêm công sức xây dựng abstraction layer cho TTS

D) Other (please describe after [Answer]: tag below)

[Answer]: B 
