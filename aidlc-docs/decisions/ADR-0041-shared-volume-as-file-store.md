# ADR-0041: `/shared` là kho file; dữ liệu và lệnh chỉ giữ đường dẫn

## Status
Accepted

## Date
2026-10-13

## Stage
Application Design (CR-072)

## Context
CR-072 thêm nhiều file: ảnh tĩnh (ảnh mẫu ngôn ngữ hình, bộ mỹ thuật, khung mẫu, animatic, thumbnail), file giọng từng câu, bản xem nhanh từng shot, file video theo cảnh, ảnh tham khảo và SVG Creator tải lên. Creator đề nghị coi `/shared` như một server lưu file: dữ liệu chỉ lưu đường dẫn, ai cần thì cầm đường dẫn đó để đọc file. Hệ thống đã làm vậy cho video (orchestrator lưu `video_path`, `rendered_video_path`, `caption_path`…; lệnh RabbitMQ chở đường dẫn; api-gateway kiểm đường dẫn nằm trong `/shared` trước khi phát), nhưng chưa thành luật chung cho mọi file.

## Options Considered
### Option A: `/shared` là kho file, DB và lệnh chỉ giữ đường dẫn, kèm bốn luật
- What it is: file nhị phân và file lớn nằm trên `/shared`; DB, lệnh RabbitMQ và HTTP giữa service chỉ chở đường dẫn. Kèm luật: một chủ ghi cho mỗi thư mục; tên file theo hash nội dung nên file không bị ghi đè; trình duyệt lấy file qua api-gateway, có kiểm đường dẫn; dọn file không còn ai trỏ tới.
- Strengths: đúng cách đang chạy; message và DB nhỏ; nhiều service đọc cùng một file mà không chép; đường dẫn đổi khi nội dung đổi, nên khớp với dấu đầu vào của ADR-0035.
- Trade-offs: cần lịch dọn file mồ côi; file và DB có thể lệch nếu một bên bị xoá tay.

### Option B: lưu file trong DB (blob)
- What it is: ảnh, giọng, video nằm trong Postgres.
- Strengths: giao dịch gồm cả file; không có file mồ côi.
- Trade-offs: DB phình rất nhanh (video hàng trăm MB); mọi service phải đọc file qua DB; ngược với cách đang chạy.

### Option C: chuyển sang kho đối tượng (MinIO hoặc S3) ngay
- What it is: thay volume `/shared` bằng một dịch vụ lưu đối tượng, đường dẫn thành khoá đối tượng.
- Strengths: tách được máy lưu và máy chạy; có sẵn URL có hạn dùng.
- Trade-offs: thêm một dịch vụ để chạy; mọi service phải đổi cách đọc ghi file; chưa có nhu cầu vì cả hệ thống chạy trên một máy.

## Decision
A. Chi tiết:
1. **File nằm ở `/shared`, dữ liệu chỉ giữ đường dẫn.** Áp cho mọi thứ nhị phân hoặc lớn: ảnh, giọng, video, file tải lên. Lệnh và sự kiện RabbitMQ chở đường dẫn, không chở nội dung file.
2. **Thứ vẫn nằm trong DB**: chữ nhỏ cần sửa, tra cứu, tính hash, ghi trong giao dịch. Ví dụ storyboard, prompt, luật, trạng thái và ghi chú của đơn vị, code của shot, SVG/TSX của hình thư viện.
3. **Một chủ ghi cho mỗi thư mục**, ghi trong `docs/contracts/shared-artifacts.md` và giữ khớp bằng test hợp đồng như hiện nay. Service khác chỉ đọc theo đường dẫn.
4. **Tên file theo hash nội dung** cho file sinh ra (ví dụ `/shared/stills/{hash}.png`, `/shared/{id}/audio/{line}_{cache_key}.wav`): nội dung đổi thì đường dẫn đổi, file cũ không bị ghi đè khi đang có người đọc.
5. **Trình duyệt không đọc `/shared` trực tiếp.** api-gateway phát file theo đường dẫn lấy từ DB của service chủ, kiểm đường dẫn nằm trong `/shared` và thuộc đúng project (mở rộng cách `videoHandler.js` đang làm), hỗ trợ tua.
6. **Dọn file mồ côi**: mỗi chủ ghi dọn file trong thư mục của mình mà DB không còn trỏ tới, theo lịch; xoá project vẫn đi qua saga xoá hiện có.
7. Giữ dạng đường dẫn tuyệt đối dưới `/shared` như đang lưu. Nếu sau này chuyển sang MinIO hoặc S3 thì đổi tiền tố `/shared/` thành khoá đối tượng; dữ liệu cũ đổi được bằng một lần thay chuỗi.

## Rationale
Đây là cách hệ thống đang làm cho video và đã chạy ổn; CR-072 chỉ biến nó thành luật chung cho mọi file mới. Tên file theo hash nối thẳng với cơ chế "cần làm lại" (ADR-0035): đơn vị đổi thì file mới có đường dẫn mới, đơn vị phụ thuộc thấy dấu đầu vào đổi. Chưa cần kho đối tượng vì mọi service chạy trên cùng một máy và cùng mount `/shared`.

## Consequences
- **Positive**: message và DB nhỏ; file dùng chung giữa service mà không chép; cache ảnh tĩnh và giọng tự nhiên theo hash; trình duyệt lấy file qua một cổng có kiểm.
- **Negative / Accepted Trade-offs**: phải viết việc dọn file mồ côi cho từng chủ ghi; file và DB có thể lệch nếu ai đó xoá tay trên `/shared` (phát file trả 404 rõ ràng, không giả thành công).
