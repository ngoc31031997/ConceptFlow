# Component Methods (API Contracts & Message Schemas)

Mô tả ở mức API contract (REST endpoint) cho giao tiếp đồng bộ, và **message schema** (RabbitMQ command/event) cho giao tiếp Saga bất đồng bộ — theo quyết định tại `application-design-plan.md` Question 4 (mức API contract) và ADR-0007 (Saga qua Message Queue). Business rule chi tiết sẽ được định nghĩa ở Functional Design (per-unit, Construction Phase).

## API Gateway (REST + SSE, cho GUI)

### `POST /projects`
- **Purpose**: Tạo video project mới (trạng thái khởi tạo `draft`). Gateway proxy trực tiếp tới Orchestrator Service.
- **Input**: `{ title: string }`
- **Output**: `{ project_id: string, status: "draft" }`

### `POST /projects/{project_id}/script`
- **Purpose**: Gửi script; Gateway gọi Orchestrator Service để khởi tạo Saga bước `ParseScript`.
- **Input**: `{ script_content: string }`
- **Output**: `{ project_id: string, status: "script_parsed" (khi Saga hoàn tất bước), scenes: Scene[] }` (theo dõi tiếp qua SSE nếu xử lý không đồng bộ ngay)

### `POST /projects/{project_id}/configure`
- **Purpose**: Chọn plugin content-type + ngôn ngữ giọng đọc.
- **Input**: `{ plugin_id: string, voice_language: "vi" | "en" }`
- **Output**: `{ project_id: string, status: "plugin_configured" }`

### `POST /projects/{project_id}/render`
- **Purpose**: Yêu cầu Orchestrator Service khởi chạy Saga "Render Pipeline" (`RenderScenes → AssembleVideo`).
- **Input**: `{}`
- **Output**: `{ project_id: string, status: string }` (trạng thái tại thời điểm gọi; theo dõi tiếp qua SSE)

### `GET /projects/{project_id}/events` (SSE)
- **Purpose**: Stream sự kiện tiến trình Saga real-time cho GUI (Gateway forward event nhận được từ Orchestrator Service).
- **Event schema**: `{ event: "progress", data: { project_id, step, scene_index, scene_total, status } }`, `{ event: "error", data: { project_id, step, message } }`, `{ event: "done", data: { project_id, status } }`

### `GET /projects/{project_id}`
- **Purpose**: Lấy trạng thái/toàn bộ thông tin hiện tại của project (proxy truy vấn Orchestrator Service).
- **Output**: `{ project_id, status, video_path?, scenes, plugin_id, voice_language }`

### `POST /projects/{project_id}/publish`
- **Purpose**: Gửi metadata publish, yêu cầu Orchestrator Service khởi chạy Saga "Publish".
- **Input**: `{ youtube_title: string, description?: string, tags?: string[], visibility: "public" | "unlisted" | "private" }`
- **Output**: `{ project_id: string, status: string }` (kết quả cuối `youtube_video_url` theo dõi qua SSE/`GET /projects/{project_id}`)

## Orchestrator Service

### `POST /sagas/render` (gọi bởi Gateway)
- **Purpose**: Khởi tạo Saga Render Pipeline cho 1 project.
- **Input**: `{ project_id, script_content, plugin_id, voice_language }`
- **Output**: `{ saga_id: string, status: "started" }`

### `POST /sagas/publish` (gọi bởi Gateway)
- **Purpose**: Khởi tạo Saga Publish cho 1 project.
- **Input**: `{ project_id, youtube_title, description?, tags?, visibility }`
- **Output**: `{ saga_id: string, status: "started" }`

### Message: Command `parse_script` → queue `script_processing.commands`
- **Payload**: `{ saga_id, project_id, script_content }`

### Message: Event `script_parsed` / `parse_failed` ← queue `orchestrator.events`
- **Payload (success)**: `{ saga_id, project_id, scenes: Scene[] }`
- **Payload (failure)**: `{ saga_id, project_id, step: "parse_script", error_message }`

### Message: Command `classify_scenes` → queue `content_plugin.commands`
- **Payload**: `{ saga_id, project_id, plugin_id, scenes: Scene[] }`

### Message: Event `scenes_classified` / `classification_failed` ← queue `orchestrator.events`
- **Payload (success)**: `{ saga_id, project_id, scenes: Scene[] }` (mỗi scene có thêm `category`, `animation_template_id`)

### Message: Command `render_scenes` → queue `rendering.commands`
- **Payload**: `{ saga_id, project_id, scenes: Scene[], voice_language }`

### Message: Event `scene_rendered` (tiến trình, per-scene) ← queue `orchestrator.events`
- **Payload**: `{ saga_id, project_id, scene_index, scene_total }`

### Message: Event `rendering_completed` / `rendering_failed` ← queue `orchestrator.events`
- **Payload (success)**: `{ saga_id, project_id, scene_clip_paths: string[], scene_audio_paths: string[] }`
- **Payload (failure)**: `{ saga_id, project_id, step: "render_scenes", failed_scene_index, error_message }`

### Message: Command `assemble_video` → queue `video_assembly.commands`
- **Payload**: `{ saga_id, project_id, scene_clip_paths: string[], scene_audio_paths: string[], background_music_path? }`

### Message: Event `video_assembled` / `assembly_failed` ← queue `orchestrator.events`
- **Payload (success)**: `{ saga_id, project_id, video_path }`

### Message: Command `publish_video` → queue `publisher.commands`
- **Payload**: `{ saga_id, project_id, video_path, title, description?, tags?, visibility }`

### Message: Event `video_published` / `publish_failed` ← queue `orchestrator.events`
- **Payload (success)**: `{ saga_id, project_id, youtube_video_url }`

## Content Plugin Service

### `GET /plugins` (REST, gọi trực tiếp bởi Gateway — ngoài Saga)
- **Purpose**: Liệt kê plugin content-type hiện có (nạp động từ thư mục `plugins/`).
- **Output**: `Plugin[]` với `{ plugin_id, name, supported_categories: string[] }`

### Consumer: command `classify_scenes` (từ `content_plugin.commands`)
- **Purpose**: Gắn loại minh họa (category) cho từng scene dựa trên plugin đã chọn; publish `scenes_classified`/`classification_failed`.

## Script Processing Service

### Consumer: command `parse_script` (từ `script_processing.commands`)
- **Purpose**: Phân tích script thô thành danh sách scene chuẩn hóa; gọi Content Plugin Service (REST nội bộ hoặc qua Orchestrator, xác định ở Low-Level Design) để gắn category; publish `script_parsed`/`parse_failed`.
- **Scene schema**: `{ scene_index, narration_text, illustration_hint, code_snippet? }`

## Rendering Service

### Consumer: command `render_scenes` (từ `rendering.commands`)
- **Purpose**: Render toàn bộ scene của 1 project (gọi TTS Service qua REST nội bộ cho từng scene, đồng bộ timing); publish `scene_rendered` (per scene) và `rendering_completed`/`rendering_failed` khi xong.

## TTS Service

### `POST /tts/synthesize` (REST, gọi trực tiếp bởi Rendering Service)
- **Purpose**: Sinh audio giọng đọc cho một đoạn lời thoại.
- **Input**: `{ text: string, language: "vi" | "en" }`
- **Output**: `{ audio_path: string, duration_seconds: number }`

## Video Assembly Service

### Consumer: command `assemble_video` (từ `video_assembly.commands`)
- **Purpose**: Ghép animation clip + audio clip (+ nhạc nền tùy chọn) của tất cả scene thành video .mp4; publish `video_assembled`/`assembly_failed`.

## Publisher Service

### `GET /auth/youtube/start` / `GET /auth/youtube/callback` (REST, luồng OAuth ngoài Saga)
- **Purpose**: Xác thực OAuth 2.0 với Google/YouTube, lưu credential.

### Consumer: command `publish_video` (từ `publisher.commands`)
- **Purpose**: Upload video lên YouTube kèm metadata; publish `video_published`/`publish_failed` (cho phép Orchestrator yêu cầu retry theo Story E3).
