# Personas

## Persona 1: The Creator (người tạo nội dung)

- **Vai trò**: Chủ sở hữu duy nhất và người vận hành công cụ. Vừa là "product owner" vừa là "end user".
- **Mục tiêu**: Sản xuất video giáo dục chất lượng cao (phong cách 3Blue1Brown) một cách hiệu quả, ban đầu tập trung vào chủ đề lập trình (thuật toán, cấu trúc dữ liệu, khái niệm lập trình), với định hướng mở rộng sang các chủ đề giáo dục khác trong tương lai.
- **Bối cảnh kỹ thuật**: Có kiến thức lập trình (đủ để hiểu nội dung kỹ thuật cần minh họa) nhưng ưu tiên **không phải viết code Manim thủ công** cho từng video — muốn thao tác qua GUI.
- **Môi trường làm việc**: Máy cá nhân, chạy công cụ qua Docker, không có hạ tầng cloud/server.
- **Động lực**: Xây dựng kênh giáo dục lập trình, muốn quy trình từ ý tưởng đến video đăng tải nhanh và lặp lại được nhiều lần.
- **Điểm đau (pain points)**:
  - Tạo animation minh họa thuật toán/khái niệm lập trình thủ công bằng Manim tốn nhiều thời gian
  - Lồng tiếng thủ công cho từng video tốn công sức
  - Đăng tải video lên YouTube thủ công (điền metadata, upload) là công việc lặp đi lặp lại
- **Kỳ vọng chính**: Một luồng làm việc mạch lạc trong GUI: soạn nội dung → cấu hình → render → xem trước → đăng tải, không cần rời khỏi công cụ hoặc viết code.

**Ghi chú**: Đây là persona duy nhất được xác định cho giai đoạn này (theo quyết định tại story-generation-plan.md, Question 4, Answer A). Không có persona "người xem/học viên" vì họ không tương tác trực tiếp với công cụ.
