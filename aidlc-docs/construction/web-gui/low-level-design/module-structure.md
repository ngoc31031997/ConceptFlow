# Module Structure — Unit 10: Web GUI

## Layering (Feature-based, Question 1)

```
services/web-gui/
├── src/
│   ├── pages/
│   │   ├── NewProjectPage.tsx     # Epic A/B/C1: soạn script, chọn plugin/ngôn ngữ/nhạc nền, submit render
│   │   ├── RenderPage.tsx          # Story C6: theo dõi tiến trình qua SSE
│   │   └── ResultPage.tsx          # Story D1/E1/E2/E3: xem video, connect YouTube, cấu hình metadata, đăng
│   ├── components/
│   │   ├── ScriptEditor.tsx        # A1: vùng soạn thảo script + import file
│   │   ├── PluginSelector.tsx      # B1: chọn content type plugin
│   │   ├── VoiceLanguageSelector.tsx # B4: chọn ngôn ngữ giọng đọc
│   │   ├── BackgroundMusicPicker.tsx # C5: chọn file nhạc nền tùy chọn
│   │   ├── ProgressTracker.tsx     # C6: hiển thị step/scene_index/scene_total
│   │   ├── ErrorBanner.tsx         # Question 7: hiển thị error_message + nút Retry
│   │   ├── VideoPlayer.tsx         # D1: phát video preview
│   │   ├── YoutubeConnectButton.tsx # E1: link tới Gateway's OAuth start
│   │   └── PublishForm.tsx         # E2: form title/description/tags/visibility
│   ├── hooks/
│   │   ├── useSSE.ts                # Custom hook wrap EventSource, dùng cho ProgressTracker
│   │   └── useProject.ts            # Fetch + poll GET /v1/projects/{id} (fallback khi SSE gián đoạn)
│   ├── api/
│   │   └── client.ts                # Toàn bộ hàm gọi API Gateway (Question 5)
│   ├── context/
│   │   └── ProjectDraftContext.tsx  # React Context + useReducer cho state soạn project (Question 2)
│   ├── types/
│   │   └── index.ts                 # Type dùng chung: Project, ProgressMessage, Plugin, PublishMetadata...
│   ├── App.tsx                      # react-router-dom setup (Question 3)
│   └── main.tsx                     # Entry point, render App vào DOM
├── tests/
│   ├── components/
│   ├── hooks/
│   └── api/
├── index.html
├── package.json
├── vite.config.ts
├── tsconfig.json
└── Dockerfile
```

## Dependency Direction
`pages/` → `components/`, `hooks/` → `api/`. `components/` không phụ thuộc `pages/` (tái sử dụng được). `hooks/` phụ thuộc `api/` (gọi API) nhưng không phụ thuộc `components/`. `context/` độc lập, được `pages/`/`components/` consume qua `useContext`. `api/` và `types/` không phụ thuộc bất kỳ layer nào khác (leaf module).

## Module Responsibilities

| Module | Responsibility |
|---|---|
| `pages/NewProjectPage.tsx` | Compose `ScriptEditor` + `PluginSelector` + `VoiceLanguageSelector` + `BackgroundMusicPicker`, đọc state từ `ProjectDraftContext`, submit `startRenderSaga` khi Creator bấm "Bắt đầu render", điều hướng sang `/projects/:id/render` |
| `pages/RenderPage.tsx` | Compose `ProgressTracker` + `ErrorBanner`, dùng `useSSE` theo dõi tiến trình, điều hướng sang `/projects/:id/result` khi status = `ready_to_publish` |
| `pages/ResultPage.tsx` | Compose `VideoPlayer` + `YoutubeConnectButton` + `PublishForm`, đọc `video_path` từ `useProject`, submit `startPublishSaga` |
| `components/ScriptEditor.tsx` | Textarea soạn script + input file import (đọc file client-side, không upload riêng — script_content gửi cùng `POST /v1/sagas/render`) |
| `components/PluginSelector.tsx` | Dropdown/list plugin, gọi `api.getPlugins()` lúc mount |
| `components/ProgressTracker.tsx` | Hiển thị step hiện tại + progress bar (`scene_index`/`scene_total` nếu có) từ `ProgressMessage` |
| `components/ErrorBanner.tsx` | Hiển thị `error_message`, nút "Thử lại" gọi `api.retryProject(id)` |
| `hooks/useSSE.ts` | Wrap `EventSource(gatewayUrl + '/v1/progress/' + projectId)`, parse JSON event, cleanup khi unmount |
| `hooks/useProject.ts` | `api.getProject(id)` + polling fallback (interval khi SSE không hoạt động) |
| `api/client.ts` | Các hàm: `getPlugins()`, `startRenderSaga(input)`, `getProject(id)`, `retryProject(id)`, `startPublishSaga(id, metadata)`, `getYoutubeAuthStartUrl()` |
| `context/ProjectDraftContext.tsx` | State soạn project trước khi submit: `script_content`, `plugin_id`, `voice_language`, `background_music_path?` |
| `App.tsx` | `react-router-dom` routes: `/`, `/projects/:id/render`, `/projects/:id/result` |
