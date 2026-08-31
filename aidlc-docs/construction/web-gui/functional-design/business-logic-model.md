# Business Logic Model — Unit 10: Web GUI

## Overview
Web GUI không chứa business logic nghiệp vụ (đó là trách nhiệm backend) — logic ở đây là **UI/UX orchestration**: quản lý state soạn thảo trước khi submit, điều hướng theo `Project.Status`, validate form để chặn submit sai trước khi gọi API (UX convenience, KHÔNG phải nguồn xác thực cuối cùng — backend vẫn zero-trust validate lại).

## Luồng 1: Soạn Project (Epic A/B)
1. Creator nhập script vào `ScriptEditor` → cập nhật `ProjectDraft.scriptContent` (Context reducer).
2. Creator chọn plugin từ danh sách (`PluginSelector`, load từ `GET /v1/plugins` lúc mount) → cập nhật `ProjectDraft.pluginId`.
3. Creator chọn ngôn ngữ giọng đọc (mặc định `"vi"`) → cập nhật `ProjectDraft.voiceLanguage`.
4. Creator tùy chọn thêm nhạc nền → cập nhật `ProjectDraft.backgroundMusicPath`.
5. Nút "Bắt đầu render" chỉ enable khi `scriptContent` không rỗng VÀ `pluginId` đã chọn (Rule 1).
6. Submit → `POST /v1/sagas/render` với `ProjectDraft` map sang `RenderInput` — nhận `project_id`, điều hướng sang `RenderPage`.

## Luồng 2: Theo dõi Tiến trình (Story C6)
1. `RenderPage` mount → `useSSE(projectId)` mở kết nối `EventSource`.
2. Mỗi `ProgressMessage` nhận được → cập nhật UI (`ProgressTracker` hiển thị step + scene_index/scene_total nếu có).
3. Khi `status = "failed"` trong message hoặc `Project.Status` (qua fallback `GET`) là `failed_at_<step>` → hiển thị `ErrorBanner` với `error_message` + nút Retry (Rule 3).
4. Khi `Project.Status` chuyển `ready_to_publish` → tự động điều hướng `ResultPage` (Rule 3).

## Luồng 3: Xem Video + Đăng (Epic D/E)
1. `ResultPage` mount → `useProject(projectId)` gọi `GET /v1/projects/{id}`, lấy `video_path` → `VideoPlayer` phát trực tiếp.
2. Nếu chưa kết nối YouTube (GUI không biết trạng thái OAuth trực tiếp — chỉ hiển thị nút "Kết nối YouTube" luôn sẵn có, Creator tự biết đã kết nối hay chưa qua kết quả OAuth redirect trước đó), Creator bấm → redirect `getYoutubeAuthStartUrl()`.
3. Creator điền `PublishForm` (title bắt buộc, mô tả/tag optional, visibility mặc định `private`) → validate (Rule 2) → enable nút "Đăng lên YouTube".
4. Submit → `POST /v1/sagas/publish` → hiển thị trạng thái "Đang đăng..." → theo dõi tiếp qua SSE/poll cho tới `published`, hiển thị `youtube_video_url`.

## Xử lý lỗi (chung, Rule 4)
- Lỗi nghiệp vụ (`failed_at_<step>`, `error_message` từ backend) → hiển thị nguyên trạng thông điệp backend.
- Lỗi hạ tầng (network error, Gateway `502`) → thông điệp chung "Không thể kết nối máy chủ, thử lại sau" (không lộ chi tiết kiến trúc nội bộ).
