# Unit of Work Dependency

## Dependency Matrix

| Unit | Depends On | Reason |
|---|---|---|
| 1. RabbitMQ Infrastructure | — | Hạ tầng nền, không phụ thuộc unit khác |
| 2. Content Plugin Service | 1 | Consumer command `classify_scenes` qua RabbitMQ |
| 3. TTS Service | — | Chỉ expose REST nội bộ, không qua RabbitMQ, không phụ thuộc unit khác |
| 4. Script Processing Service | 1, 2 | Consumer command `parse_script` qua RabbitMQ; gọi Content Plugin Service |
| 5. Rendering Service | 1, 3 | Consumer command `render_scenes` qua RabbitMQ; gọi TTS Service (REST) |
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
    U1 --> U4
    U1 --> U5
    U1 --> U6
    U1 --> U7
    U3 --> U5
    U2 --> U4
    U2 --> U8
    U4 --> U8
    U5 --> U8
    U6 --> U8
    U7 --> U8
    U8 --> U9
    U2 --> U9
    U7 --> U9
    U9 --> U10
```

**Thứ tự phát triển đề xuất**: Unit 1 → (Unit 2, Unit 3 song song — không phụ thuộc nhau) → Unit 4, Unit 6, Unit 7 (song song, đều chỉ phụ thuộc Unit 1 + tương ứng) → Unit 5 (sau Unit 3) → Unit 8 (sau khi 2,3,4,5,6,7 xong) → Unit 9 → Unit 10.

## Circular Dependency Check
Không có dependency vòng — đồ thị trên là DAG (Directed Acyclic Graph) hợp lệ, khớp với `component-dependency.md` ở Application Design.
