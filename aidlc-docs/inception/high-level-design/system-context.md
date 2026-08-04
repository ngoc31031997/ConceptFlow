# System Context

## External Actors
- **Creator** (persona duy nhất, xem `personas.md`) — người dùng cá nhân tương tác trực tiếp với hệ thống qua GUI web để soạn nội dung, cấu hình, khởi chạy render, xem kết quả, và đăng video.

## External Systems
- **YouTube Data API (Google)** — hệ thống bên ngoài duy nhất mà hệ thống tích hợp, dùng để xác thực OAuth 2.0 và tự động tải video đã render lên kênh YouTube của Creator (FR7).

Không có hệ thống bên ngoài nào khác (đã xác nhận: TTS chạy offline nội bộ, không dùng cloud storage ngoài, không dùng dịch vụ TTS cloud).

## System Boundary
Toàn bộ hệ thống — GUI, các microservice backend (Plugin/Content, Script Processing, Rendering, TTS, Video Assembly, Publishing), và API Gateway — chạy trong **docker-compose trên một máy cá nhân** duy nhất, do Creator vận hành trực tiếp. Ranh giới hệ thống là toàn bộ tập hợp container này; điểm giao tiếp duy nhất vượt ra ngoài ranh giới là kết nối HTTPS đến YouTube Data API.

## Context Diagram

```mermaid
flowchart LR
    Creator(["👤 Creator<br/>(người dùng cá nhân)"])

    subgraph System["🐳 Manim Educational Video System<br/>(docker-compose, 1 máy cá nhân)"]
        GUI["Web GUI"]
        GW["API Gateway"]
        Services["Backend Microservices<br/>(Plugin, Script, Render, TTS,<br/>Video Assembly, Publisher)"]
    end

    YouTube[["☁️ YouTube Data API"]]

    Creator -- "Soạn nội dung, cấu hình,<br/>xem tiến trình, xem kết quả" --> GUI
    GUI -- "REST / SSE" --> GW
    GW -- "route request" --> Services
    Services -- "OAuth 2.0 + Upload video" --> YouTube

    style Creator fill:#CE93D8,stroke:#6A1B9A,stroke-width:3px,color:#000
    style YouTube fill:#FFF59D,stroke:#F9A825,stroke-width:2px,color:#000
    style System fill:#BBDEFB,stroke:#1565C0,stroke-width:2px,color:#000
```
