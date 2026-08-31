# Sequence Flows — Unit 10: Web GUI

## Flow 1: Soạn Project + Khởi chạy Render (Epic A/B, Story C1)

```mermaid
sequenceDiagram
    participant Creator
    participant NPP as NewProjectPage
    participant CTX as ProjectDraftContext
    participant API as api/client.ts
    participant GW as API Gateway

    Creator->>NPP: Nhập script (ScriptEditor)
    NPP->>CTX: dispatch({type: 'SET_SCRIPT', ...})
    Creator->>NPP: Chọn plugin (PluginSelector, gọi getPlugins() lúc mount)
    NPP->>CTX: dispatch({type: 'SET_PLUGIN', ...})
    Creator->>NPP: Chọn ngôn ngữ + nhạc nền (tùy chọn)
    NPP->>CTX: dispatch({type: 'SET_VOICE_LANG'/'SET_MUSIC', ...})
    Creator->>NPP: Bấm "Bắt đầu render"
    NPP->>API: startRenderSaga(draftFromContext)
    API->>GW: POST /v1/sagas/render
    GW-->>API: 201 {saga_id, status: started}
    API-->>NPP: response
    NPP->>NPP: navigate(`/projects/${project_id}/render`)
```

## Flow 2: Theo dõi Tiến trình qua SSE (Story C6)

```mermaid
sequenceDiagram
    participant RP as RenderPage
    participant SSE as useSSE hook
    participant GW as API Gateway
    participant PT as ProgressTracker

    RP->>SSE: useSSE(projectId)
    SSE->>GW: new EventSource(/v1/progress/{id})
    GW-->>SSE: data: {step: "parse_script", status: "in_progress"}
    SSE->>RP: onMessage(msg) → setState
    RP->>PT: render(progressState)

    loop Mỗi progress message
        GW-->>SSE: data: {...}
        SSE->>RP: onMessage(msg)
        RP->>PT: re-render
    end

    alt status = failed_at_<step>
        RP->>RP: render ErrorBanner (error_message + nút Retry)
    else status = ready_to_publish (ngầm định sau video_assembled)
        RP->>RP: navigate(`/projects/${id}/result`)
    end
```

## Flow 3: Xem Kết quả + Phát Video (Story D1)

```mermaid
sequenceDiagram
    participant ResP as ResultPage
    participant UP as useProject hook
    participant API as api/client.ts
    participant GW as API Gateway
    participant VP as VideoPlayer

    ResP->>UP: useProject(projectId)
    UP->>API: getProject(id)
    API->>GW: GET /v1/projects/{id}
    GW-->>API: 200 {video_path, status: ready_to_publish, ...}
    API-->>UP: project
    UP-->>ResP: project
    ResP->>VP: render(<video src={video_path} controls />)
```

## Flow 4: Kết nối YouTube + Đăng Video (Story E1/E2/E3)

```mermaid
sequenceDiagram
    participant Creator
    participant ResP as ResultPage
    participant YCB as YoutubeConnectButton
    participant Browser
    participant GW as API Gateway
    participant PF as PublishForm
    participant API as api/client.ts

    Creator->>YCB: Bấm "Kết nối YouTube"
    YCB->>Browser: window.location.href = getYoutubeAuthStartUrl()
    Browser->>GW: GET /v1/auth/youtube/start (browser tự theo redirect chain tới Google rồi quay lại callback)
    GW-->>Browser: 200 {connected: true} (sau callback)
    Browser->>ResP: Creator quay lại GUI (redirect_uri trỏ về ResultPage hoặc app base URL)

    Creator->>PF: Điền title/description/tags/visibility
    Creator->>PF: Bấm "Đăng lên YouTube"
    PF->>API: startPublishSaga(id, metadata)
    API->>GW: POST /v1/sagas/publish
    GW-->>API: 201 {saga_id, status: started}
    API-->>PF: response
    PF->>PF: hiển thị trạng thái "Đang đăng..." (poll getProject hoặc theo dõi SSE tiếp)
```

## Flow 5: Xử lý Lỗi + Retry (Question 7)

```mermaid
sequenceDiagram
    participant EB as ErrorBanner
    participant Creator
    participant API as api/client.ts
    participant GW as API Gateway

    Note over EB: Project.Status = failed_at_render_scenes, error_message hiển thị
    Creator->>EB: Bấm "Thử lại"
    EB->>API: retryProject(id)
    API->>GW: POST /v1/projects/{id}/retry
    GW-->>API: 200 {saga_id, status: rendering}
    API-->>EB: response
    EB->>EB: quay lại trạng thái theo dõi tiến trình bình thường (SSE tiếp tục)
```
