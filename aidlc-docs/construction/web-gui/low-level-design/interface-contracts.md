# Interface Contracts — Unit 10: Web GUI

## API Client Methods (`src/api/client.ts`) — khớp API Gateway's routing table (Unit 9)

| Function | Gọi tới (qua Gateway) | Dùng ở |
|---|---|---|
| `getPlugins(): Promise<Plugin[]>` | `GET /v1/plugins` | `PluginSelector` |
| `startRenderSaga(input: RenderInput): Promise<{saga_id, status}>` | `POST /v1/sagas/render` | `NewProjectPage` |
| `getProject(id: string): Promise<Project>` | `GET /v1/projects/{id}` | `useProject`, `ResultPage` |
| `retryProject(id: string): Promise<{saga_id, status}>` | `POST /v1/projects/{id}/retry` | `ErrorBanner` |
| `startPublishSaga(id: string, metadata: PublishMetadata): Promise<{saga_id, status}>` | `POST /v1/sagas/publish` | `PublishForm` |
| `getYoutubeAuthStartUrl(): string` | (không fetch — trả string URL) `${GATEWAY_URL}/v1/auth/youtube/start` | `YoutubeConnectButton` (dùng làm `href`) |
| `subscribeProgress(projectId: string, onMessage: (msg: ProgressMessage) => void): () => void` | `EventSource(${GATEWAY_URL}/v1/progress/${projectId})` — trả cleanup function | `useSSE` |

## Request/Response Types (`src/types/index.ts`)

```typescript
interface RenderInput {
  project_id: string;
  script_content: string;
  plugin_id: string;
  voice_language: "vi" | "en";
  background_music_path?: string;
}

interface Project {
  project_id: string;
  status: string; // ProjectStatus enum từ Orchestrator (Unit 8)
  video_path?: string;
  scenes: Scene[];
  plugin_id: string;
  voice_language: string;
  youtube_video_url?: string;
  error_message?: string;
}

interface ProgressMessage {
  project_id: string;
  step: string;
  status: "in_progress" | "completed" | "failed";
  scene_index?: number;
  scene_total?: number;
  error_message?: string;
}

interface Plugin {
  plugin_id: string;
  name: string;
}

interface PublishMetadata {
  youtube_title: string;
  description?: string;
  tags?: string[];
  visibility: "public" | "unlisted" | "private";
}
```

## Error Contract (Question 7)
- HTTP 4xx/5xx từ Gateway (forward nguyên trạng từ downstream): parse `error.message`/`error_message` field nếu có, hiển thị trong `ErrorBanner`.
- HTTP 502 (Gateway's `upstream_unavailable`): hiển thị thông báo chung "Không thể kết nối máy chủ, thử lại sau" — không hiển thị chi tiết `service` field (ẩn kiến trúc nội bộ khỏi Creator).
- Network error (fetch throw, không có response — vd. Gateway down hoàn toàn): cùng thông báo chung như trên.
- `Project.Status = failed_at_<step>`: hiển thị `error_message` cụ thể + nút "Thử lại".

## API Versioning
Không áp dụng phía GUI — chỉ gọi đúng path `/v1/...` mà Gateway expose, không tự quản lý version.

## Correlation ID
GUI KHÔNG tự sinh `X-Request-ID` — Gateway là nơi sinh ID này (Unit 9's Question 6). GUI có thể tùy chọn log ID này (đọc từ response header nếu cần debug) nhưng không bắt buộc.
