# Logical Components — Unit 3: TTS Service

## Components
| Component | Type | Purpose |
|---|---|---|
| FastAPI app | HTTP server | Expose `POST /v1/tts/synthesize` |
| `PiperTTSAdapter` | In-process TTS engine wrapper | Implement `TTSEnginePort`, chạy synthesis trong threadpool, 60s timeout nội bộ |
| Voice model in-process cache | In-memory dict (`language -> model instance`) | Tránh load lại Piper voice model mỗi request; load eager lúc startup |
| Shared Docker Volume | File storage (bên ngoài process) | Lưu file audio `.wav`, đồng thời là cơ chế idempotency (kiểm tra tồn tại file) |

## No External Infrastructure Components
Không cần cache ngoài (Redis), không cần database, không cần message broker (không tham gia RabbitMQ), không cần circuit breaker riêng (không gọi service ngoài nào — leaf node duy nhất phụ thuộc là engine TTS local + file system).

## Diagram

```mermaid
flowchart TB
    subgraph Unit3["TTS Service (Python/FastAPI)"]
        API["FastAPI: POST /v1/tts/synthesize"]
        UC["SynthesizeSpeechUseCase"]
        ADAPTER["PiperTTSAdapter<br/>(threadpool, 60s timeout)"]
        CACHE["Voice Model Cache<br/>(in-memory, loaded at startup)"]
    end

    RD["Rendering Service"]
    FS[("Shared Docker Volume<br/>/shared/{project_id}/audio/")]

    RD -->|"REST (X-Saga-ID header)"| API
    API --> UC
    UC -->|"check file exists (idempotency)"| FS
    UC --> ADAPTER
    ADAPTER --> CACHE
    ADAPTER -->|"write .wav"| FS

    style Unit3 fill:#C8E6C9,stroke:#2E7D32,stroke-width:2px,color:#000
    style FS fill:#FFCCBC,stroke:#BF360C,stroke-width:2px,color:#000
```
