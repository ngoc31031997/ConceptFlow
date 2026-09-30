# Logical Components — Unit 1: RabbitMQ Infrastructure

> **Cập nhật (CR-055, 2026-09-30)**: bảng và sơ đồ dưới đây theo topology thật trong `infra/rabbitmq/definitions.json`. Queue `script_processing.commands` (gỡ ở CR-040) và `content_plugin.commands` (gỡ ở CR-020) không còn; `tts.commands` (ADR-0014) trước đây thiếu trong bảng; `control.fanout` (lệnh huỷ) và `orchestrator.events.dlq` được thêm.

## Exchange Topology (Direct Exchange — theo NFR Design Question 2)

### Command Exchange: `commands.direct`
| Routing Key | Bound Queue | Consumer |
|---|---|---|
| `tts` | `tts.commands` | TTS Service (Unit 3) |
| `rendering` | `rendering.commands` | Rendering Service (Unit 5) |
| `video_assembly` | `video_assembly.commands` | Video Assembly Service (Unit 6) |
| `publisher` | `publisher.commands` | Publisher Service (Unit 7) |

### Event Exchange: `events.direct`
| Routing Key | Bound Queue | Consumer |
|---|---|---|
| `orchestrator` | `orchestrator.events` | Orchestrator Service (Unit 8) |

Tất cả 4 service nghiệp vụ (TTS, Rendering, Video Assembly, Publisher) đều publish event vào `events.direct` với routing key `orchestrator` — event type nằm trong message payload (không phải routing key), vì chỉ có 1 consumer (Orchestrator).

### Progress Exchange: `progress.fanout` (MỚI — Revision, Unit 8 Low-Level Design Question 4, ADR-0017)
Type `fanout`, không pre-declare queue nào (Gateway, Unit 9, sẽ tự khai báo + bind queue riêng khi khởi động). Orchestrator Service (Unit 8) publish progress message (project_id, step, status, scene_index?, scene_total?, error_message?) vào exchange này sau mỗi lần xử lý event Saga — Gateway subscribe để forward qua SSE cho GUI (Story C6). Dùng fanout (không phải direct/topic) vì chỉ có 1 consumer thực tế (Gateway) và không cần routing key phân biệt — Gateway tự lọc theo `project_id` trong payload để định tuyến tới đúng SSE client.

### Control Exchange: `control.fanout`
Type `fanout`. Orchestrator phát lệnh huỷ; mỗi worker (tts, rendering, video-assembly) tự khai báo queue riêng và dừng bước đang chạy của đúng `project_id` (`adapters/messaging/cancellation.py`).

## Dead-Letter Topology
Mỗi queue command có DLQ riêng, cấu hình qua `x-dead-letter-exchange` + `x-dead-letter-routing-key`:

| Queue | DLQ |
|---|---|
| `tts.commands` | `tts.commands.dlq` |
| `rendering.commands` | `rendering.commands.dlq` |
| `video_assembly.commands` | `video_assembly.commands.dlq` |
| `publisher.commands` | `publisher.commands.dlq` |
| `orchestrator.events` | `orchestrator.events.dlq` |

Orchestrator Service consume tất cả DLQ (qua 1 consumer chung lắng nghe pattern `*.dlq`, hoặc nhiều consumer riêng — quyết định cụ thể ở Low-Level Design của Unit 8).

## Queue Configuration (áp dụng cho mọi queue command)
```
durable: true
arguments:
  x-message-ttl: 86400000        # 24h in ms
  x-dead-letter-exchange: "dlx.direct"
  x-dead-letter-routing-key: "<queue-name>.dlq"
```

## Diagram

```mermaid
flowchart LR
    ORCH["Orchestrator<br/>Service"]

    subgraph MQ["RabbitMQ"]
        CEX{{"commands.direct"}}
        EEX{{"events.direct"}}
        Q1[["tts<br/>.commands"]]
        Q3[["rendering<br/>.commands"]]
        Q4[["video_assembly<br/>.commands"]]
        Q5[["publisher<br/>.commands"]]
        QE[["orchestrator<br/>.events"]]
        DLQ[["*.dlq queues"]]
    end

    TTS["TTS"]
    RD["Rendering"]
    VA["Video Assembly"]
    PB["Publisher"]

    ORCH -->|publish command| CEX
    CEX --> Q1 --> TTS
    CEX --> Q3 --> RD
    CEX --> Q4 --> VA
    CEX --> Q5 --> PB

    TTS -->|publish event| EEX
    RD -->|publish event| EEX
    VA -->|publish event| EEX
    PB -->|publish event| EEX
    EEX --> QE --> ORCH

    Q1 -.->|retry exceeded| DLQ
    Q3 -.->|retry exceeded| DLQ
    Q4 -.->|retry exceeded| DLQ
    Q5 -.->|retry exceeded| DLQ
    DLQ -.-> ORCH

    style MQ fill:#FFCCBC,stroke:#BF360C,stroke-width:2px,color:#000
```
