# Unit of Work Dependency

**Revision (2026-08-07, ADR-0012, ADR-0014)**: TTS Service (Unit 3) là message-driven, Script Processing Service (Unit 4) không còn gọi Content Plugin Service trực tiếp, Rendering Service (Unit 5) không còn gọi TTS Service trực tiếp — Orchestrator điều phối các bước này. Bảng dưới đã cập nhật; đồ thị dependency đơn giản hơn (mọi business service chỉ phụ thuộc Unit 1).

## Dependency Matrix

| Unit | Depends On | Reason |
|---|---|---|
| 1. RabbitMQ Infrastructure | — | Hạ tầng nền, không phụ thuộc unit khác |
| 2. Content Plugin Service | 1 | Consumer command `classify_scenes` qua RabbitMQ |
| 3. TTS Service | 1 | Consumer command `synthesize_speech` qua RabbitMQ (ADR-0014) |
| 4. Script Processing Service | 1 | Consumer command `parse_script` qua RabbitMQ; không còn gọi Content Plugin Service trực tiếp (ADR-0012) |
| 5. Rendering Service | 1 | Consumer command `render_scenes` qua RabbitMQ; không còn gọi TTS Service trực tiếp (ADR-0014) |
| 6. Video Assembly Service | 1 | Consumer command `assemble_video` qua RabbitMQ |
| 7. Publisher Service | 1 | Consumer command `publish_video` qua RabbitMQ |
| 8. Orchestrator Service | 1, 2, 3, 4, 5, 6, 7 | Cần publish/consume message với tất cả service nghiệp vụ để test đầy đủ 2 Saga |
| 9. API Gateway | 2, 7, 8 | Proxy REST tới Orchestrator (khởi tạo Saga), Content Plugin (`GET /plugins`), Publisher (OAuth) |
| 10. Web GUI | 9 | Gọi REST/SSE tới API Gateway |

## Development Sequence (theo Question 2 — dependency-first)

```mermaid
flowchart LR
    U1["1. RabbitMQ<br/>Infrastructure"]
    U3["3. TTS Service"]
    U2["2. Content Plugin<br/>Service"]
    U4["4. Script Processing<br/>Service"]
    U5["5. Rendering<br/>Service"]
    U6["6. Video Assembly<br/>Service"]
    U7["7. Publisher<br/>Service"]
    U8["8. Orchestrator<br/>Service"]
    U9["9. API Gateway"]
    U10["10. Web GUI"]

    U1 --> U2
    U1 --> U3
    U1 --> U4
    U1 --> U5
    U1 --> U6
    U1 --> U7
    U2 --> U8
    U3 --> U8
    U4 --> U8
    U5 --> U8
    U6 --> U8
    U7 --> U8
    U8 --> U9
    U2 --> U9
    U7 --> U9
    U9 --> U10
```

**Thứ tự phát triển đề xuất (đã cập nhật)**: Unit 1 → (Unit 2, Unit 3, Unit 4, Unit 5, Unit 6, Unit 7 song song — tất cả chỉ phụ thuộc Unit 1) → Unit 8 (sau khi 2-7 xong) → Unit 9 → Unit 10.

## Circular Dependency Check
Không có dependency vòng — đồ thị trên là DAG (Directed Acyclic Graph) hợp lệ, khớp với `component-dependency.md` ở Application Design.
