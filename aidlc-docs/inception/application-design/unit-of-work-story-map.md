# Unit of Work — Story Map

Ánh xạ toàn bộ 17 story trong `stories.md` tới unit tương ứng, đảm bảo mọi story đều có unit sở hữu.

| Story | Epic | Unit(s) chịu trách nhiệm chính |
|---|---|---|
| A1 — Soạn script nội dung video trong GUI | Content Authoring | Unit 10 (Web GUI), Unit 9 (API Gateway) |
| A2 — Phân tích script thành cấu trúc scene | Content Authoring | Unit 4 (Script Processing), Unit 8 (Orchestrator) |
| B1 — Chọn content type plugin cho video | Plugin & Configuration | Unit 10, Unit 9, Unit 2 (Content Plugin) |
| B2 — Chọn loại nội dung lập trình cụ thể | Plugin & Configuration | Unit 2 (Content Plugin) |
| B3 — Sử dụng scene minh họa code có syntax highlight | Plugin & Configuration | Unit 5 (Rendering) |
| B4 — Chọn ngôn ngữ giọng đọc cho video | Plugin & Configuration | Unit 10, Unit 8 (state), Unit 3 (TTS) |
| C1 — Khởi chạy render video từ GUI | Rendering | Unit 10, Unit 9, Unit 8 (Orchestrator) |
| C2 — Sinh giọng đọc tự động từ script | Rendering | Unit 3 (TTS), Unit 5 (Rendering — caller) |
| C3 — Đồng bộ animation với thời lượng giọng đọc | Rendering | Unit 5 (Rendering) |
| C4 — Ghép animation và audio thành video hoàn chỉnh | Rendering | Unit 6 (Video Assembly) |
| C5 — Thêm nhạc nền tùy chọn cho video | Rendering | Unit 6 (Video Assembly), Unit 10 (cấu hình) |
| C6 — Theo dõi tiến trình render trong GUI | Rendering | Unit 10, Unit 9 (SSE), Unit 8 (Orchestrator — nguồn sự kiện) |
| D1 — Xem trước video sau khi render xong | Preview & Review | Unit 10 (Web GUI) |
| E1 — Xác thực tài khoản YouTube qua OAuth | Publishing | Unit 7 (Publisher), Unit 9 (Gateway — proxy) |
| E2 — Cấu hình metadata video trước khi đăng | Publishing | Unit 10, Unit 8 (Orchestrator — lưu vào Saga publish) |
| E3 — Tự động đăng video lên YouTube | Publishing | Unit 7 (Publisher), Unit 8 (Orchestrator) |
| F1 — Chạy toàn bộ công cụ qua Docker trên máy cá nhân | Platform & Runtime | Toàn bộ 10 unit + `docker-compose.yml` root (không unit riêng — thuộc code organization strategy, `unit-of-work.md`) |

## Coverage Check
- **17/17 story** được gán ít nhất 1 unit chịu trách nhiệm chính.
- **10/10 unit** đều xuất hiện trong bảng trên — không có unit "mồ côi" (không gắn với story nào), kể cả Unit 1 (RabbitMQ Infrastructure) và Unit 8 (Orchestrator) tuy không có story riêng nhưng là hạ tầng/điều phối nền cho các story C1, C6, E3.
