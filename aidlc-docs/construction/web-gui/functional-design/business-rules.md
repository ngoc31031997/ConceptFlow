# Business Rules — Unit 10: Web GUI

## Rule 1: New Project Form Validation (Question 1)
- `script_content`: bắt buộc, không rỗng (sau khi trim khoảng trắng).
- `plugin_id`: bắt buộc chọn — Story B1's AC yêu cầu hệ thống "yêu cầu chọn plugin trước khi tiếp tục".
- `voice_language`: mặc định `"vi"`, không bắt buộc Creator chọn lại nhưng hiển thị rõ để đổi sang `"en"`.
- `background_music_path`: optional, không validate định dạng ở GUI.
- Nút "Bắt đầu render" DISABLED cho tới khi `script_content` và `plugin_id` đều hợp lệ.

## Rule 2: Publish Metadata Form Validation (Question 2)
- `youtube_title`: bắt buộc, tối đa 100 ký tự (giới hạn thực tế YouTube API), hiển thị bộ đếm ký tự. Story E2's AC: thiếu title → chặn submit + thông báo rõ.
- `description`, `tags`: optional.
- `visibility`: bắt buộc chọn 1 trong 3 (`public`/`unlisted`/`private`), mặc định `"private"` (an toàn nhất — tránh đăng công khai ngoài ý muốn).
- Nút "Đăng lên YouTube" DISABLED cho tới khi `youtube_title` hợp lệ.

## Rule 3: State-Driven Navigation (Question 3)
GUI tự động điều hướng theo `Project.Status` nhận được (SSE hoặc GET fallback), không cần Creator tự bấm "Next":
| Status | GUI Behavior |
|---|---|
| `draft`, `parsing_script` ... `assembling_video` | Ở lại `RenderPage`, hiển thị `ProgressTracker` |
| `failed_at_<step>` | Ở lại `RenderPage`, hiển thị `ErrorBanner` (error_message + nút Retry) |
| `ready_to_publish` | Tự động điều hướng `ResultPage` |
| `publishing` | Hiển thị trạng thái "Đang đăng..." trên `ResultPage` |
| `published` | Hiển thị `youtube_video_url` (link tới video đã đăng) trên `ResultPage` |

## Rule 4: Error Message Mapping (Question 7 của LLD, chi tiết hóa)
- Lỗi nghiệp vụ có `error_message` từ backend (event `*_failed`, `Project.error_message`) → hiển thị NGUYÊN VĂN thông điệp đó (backend đã viết message rõ ràng, không cần GUI diễn giải lại).
- Lỗi hạ tầng (network exception, HTTP 502 `upstream_unavailable`) → thông điệp CHUNG cố định "Không thể kết nối máy chủ, thử lại sau" — không hiển thị field `service` hay chi tiết kỹ thuật (ẩn kiến trúc nội bộ khỏi Creator, đúng nguyên tắc UX).
- Lỗi validation form (Rule 1/2) → thông điệp inline cạnh field lỗi, tiếng Việt, mô tả rõ field nào và tại sao (vd. "Vui lòng nhập tiêu đề video").

## Rule 5: GUI Validation Is UX-Only, Not Source of Truth
Mọi validation ở GUI (Rule 1, Rule 2) chỉ nhằm cải thiện trải nghiệm (chặn submit sai sớm) — KHÔNG thay thế validate của backend. Nếu backend trả lỗi 4xx dù GUI đã validate qua, GUI vẫn hiển thị lỗi đó bình thường (Rule 4) thay vì coi là trường hợp không thể xảy ra.

## Rule 6: Data-testid Convention (Question 6, Automation Friendly Code Rules)
Format `{component}-{element-role}`, ổn định qua các lần thay đổi code (chỉ đổi khi ý nghĩa element đổi):
- `new-project-script-textarea`, `new-project-plugin-select`, `new-project-voice-language-select`, `new-project-music-input`, `new-project-submit-button`
- `progress-tracker-step-label`, `error-banner-message`, `error-banner-retry-button`
- `video-player-element`, `youtube-connect-button`
- `publish-form-title-input`, `publish-form-description-textarea`, `publish-form-tags-input`, `publish-form-visibility-select`, `publish-form-submit-button`
