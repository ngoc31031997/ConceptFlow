# Frontend Components — Unit 10: Web GUI

## Component Hierarchy
```
App
├── ProjectDraftContext.Provider
│   └── Router
│       ├── NewProjectPage (/)
│       │   ├── ScriptEditor
│       │   ├── PluginSelector
│       │   ├── VoiceLanguageSelector
│       │   └── BackgroundMusicPicker
│       ├── RenderPage (/projects/:id/render)
│       │   ├── ProgressTracker
│       │   └── ErrorBanner (conditional)
│       └── ResultPage (/projects/:id/result)
│           ├── VideoPlayer
│           ├── YoutubeConnectButton
│           └── PublishForm
```

## Pages

### `NewProjectPage`
- **Props**: none (route component)
- **State**: đọc/ghi qua `useContext(ProjectDraftContext)`; local `isSubmitting: boolean`
- **User interactions**: nhập script, chọn plugin/ngôn ngữ/nhạc nền, bấm submit
- **API integration**: `api.startRenderSaga(draft)` khi submit
- **data-testid**: `new-project-page`

### `RenderPage`
- **Props**: none (route param `:id` qua `useParams`)
- **State**: `progressState: ProgressState` (từ `useSSE`), `project: Project | null` (fallback từ `useProject`)
- **User interactions**: bấm "Thử lại" (`ErrorBanner`) khi lỗi
- **API integration**: `useSSE(projectId)`, `useProject(projectId)` fallback poll khi SSE gián đoạn
- **data-testid**: `render-page`

### `ResultPage`
- **Props**: none (route param `:id`)
- **State**: `project: Project | null` (từ `useProject`), `publishMetadata: PublishMetadata` (local form state), `isPublishing: boolean`
- **User interactions**: phát video, bấm kết nối YouTube, điền + submit form metadata
- **API integration**: `useProject(projectId)`, `api.startPublishSaga(id, metadata)`
- **data-testid**: `result-page`

## Components

### `ScriptEditor`
- **Props**: `value: string`, `onChange: (value: string) => void`
- **State**: none (controlled component)
- **User interactions**: gõ text, chọn file import (`<input type="file">`, đọc qua `FileReader`, set vào `value`)
- **data-testid**: `new-project-script-textarea`

### `PluginSelector`
- **Props**: `value: string | null`, `onChange: (pluginId: string) => void`
- **State**: `plugins: Plugin[]` (load lúc mount qua `useEffect` + `api.getPlugins()`), `isLoading: boolean`
- **User interactions**: chọn 1 plugin từ dropdown
- **data-testid**: `new-project-plugin-select`

### `VoiceLanguageSelector`
- **Props**: `value: "vi" | "en"`, `onChange: (lang: "vi" | "en") => void`
- **State**: none
- **data-testid**: `new-project-voice-language-select`

### `BackgroundMusicPicker`
- **Props**: `value: string | null`, `onChange: (path: string | null) => void`
- **State**: none
- **User interactions**: bật/tắt checkbox nhạc nền, nhập đường dẫn file (MVP: input text đường dẫn file local, không upload — nhất quán scope FR5.2 không yêu cầu file upload service riêng)
- **data-testid**: `new-project-music-input`

### `ProgressTracker`
- **Props**: `progressState: ProgressState`
- **State**: none (controlled bởi parent)
- **data-testid**: `progress-tracker-step-label` (label step hiện tại), `progress-tracker-bar` (progress bar nếu có scene_index/scene_total)

### `ErrorBanner`
- **Props**: `errorMessage: string`, `onRetry: () => void`, `isRetrying: boolean`
- **State**: none
- **User interactions**: bấm nút "Thử lại"
- **data-testid**: `error-banner-message`, `error-banner-retry-button`

### `VideoPlayer`
- **Props**: `videoSrc: string`
- **State**: none
- **data-testid**: `video-player-element`

### `YoutubeConnectButton`
- **Props**: none
- **State**: none
- **User interactions**: bấm → `window.location.href = api.getYoutubeAuthStartUrl()`
- **data-testid**: `youtube-connect-button`

### `PublishForm`
- **Props**: `onSubmit: (metadata: PublishMetadata) => void`, `isSubmitting: boolean`
- **State**: local form state (`youtubeTitle`, `description`, `tags`, `visibility`) — không cần đẩy vào Context (chỉ dùng 1 lần lúc submit)
- **User interactions**: điền form, chọn visibility, bấm submit (disabled nếu `youtubeTitle` rỗng — Rule 2)
- **data-testid**: `publish-form-title-input`, `publish-form-description-textarea`, `publish-form-tags-input`, `publish-form-visibility-select`, `publish-form-submit-button`

## Form Validation Rules
Xem `business-rules.md` Rule 1 (New Project), Rule 2 (Publish Metadata).

## API Integration Points Summary
| Component/Hook | API Method |
|---|---|
| `PluginSelector` | `api.getPlugins()` |
| `NewProjectPage` (submit) | `api.startRenderSaga(input)` |
| `useSSE` | `api.subscribeProgress(projectId, onMessage)` |
| `useProject` | `api.getProject(id)` |
| `ErrorBanner` (via `RenderPage`) | `api.retryProject(id)` |
| `YoutubeConnectButton` | `api.getYoutubeAuthStartUrl()` |
| `PublishForm` (via `ResultPage`) | `api.startPublishSaga(id, metadata)` |

## Styling
CSS module đơn giản (`*.module.css`) per component — KHÔNG dùng thư viện UI ngoài (MUI/Chakra/Tailwind), phù hợp quy mô 1 luồng người dùng tuyến tính, MVP cá nhân (Question 5).
