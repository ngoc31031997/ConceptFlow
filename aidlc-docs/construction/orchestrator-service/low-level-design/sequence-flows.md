# Sequence Flows — Unit 8: Orchestrator Service

## Flow 1: Start Render Saga

```mermaid
sequenceDiagram
    participant GW as API Gateway
    participant API as adapters/http/router.go
    participant UC as StartRenderSagaUseCase
    participant REPO as ProjectRepository
    participant OUTBOX as OutboxRepository
    participant RELAY as OutboxRelay
    participant MQ as RabbitMQ

    GW->>API: POST /v1/sagas/render
    API->>UC: execute(input)
    UC->>UC: generate saga_id
    UC->>REPO: Save(Project{status: draft})
    UC->>REPO: UpdateStep(SagaStep{step: parse_script, status: in_progress})
    UC->>OUTBOX: enqueue command parse_script (same transaction as status update)
    UC->>REPO: UpdateStatus(parsing_script)
    UC-->>API: {saga_id, status: started}
    API-->>GW: 201 {saga_id, status: started}

    RELAY->>MQ: publish parse_script (poll định kỳ)
    MQ-->>SP: deliver parse_script
```

## Flow 2: Successful Step — Classify Scenes (mid-Saga)

```mermaid
sequenceDiagram
    participant MQ as RabbitMQ
    participant CONSUMER as adapters/amqp/consumer.go
    participant INBOX as InboxRepository
    participant UC as HandleStepEventUseCase
    participant REPO as ProjectRepository
    participant OUTBOX as OutboxRepository
    participant PROG as amqp.Publisher (ProgressPublisherPort)

    MQ->>CONSUMER: deliver scenes_classified
    CONSUMER->>INBOX: has_processed(message_id)?
    INBOX-->>CONSUMER: no
    CONSUMER->>UC: handle(event) [goroutine]

    UC->>REPO: GetStep(saga_id, "classify_scenes")
    REPO-->>UC: SagaStep{status: in_progress}
    UC->>REPO: UpdateStep(status: completed)
    UC->>REPO: Save(Project{scenes: merged with category})
    UC->>REPO: UpdateStep(SagaStep{step: synthesize_speech, status: in_progress})
    UC->>OUTBOX: enqueue command synthesize_speech
    UC->>REPO: UpdateStatus(synthesizing_speech)
    UC->>INBOX: mark_processed(message_id) — cùng transaction với Outbox/status update
    UC->>PROG: PublishProgress({step: classify_scenes, status: completed}) — sau khi transaction commit
    CONSUMER->>MQ: ack
```

## Flow 3: Per-Scene Progress (scene_rendered — không advance state machine)

```mermaid
sequenceDiagram
    participant MQ as RabbitMQ
    participant CONSUMER as adapters/amqp/consumer.go
    participant UC as HandleStepEventUseCase
    participant PROG as amqp.Publisher

    MQ->>CONSUMER: deliver scene_rendered (scene_index=2, scene_total=5)
    CONSUMER->>UC: handle(event) [goroutine]
    Note over UC: Không tra saga_steps/cập nhật status — chỉ forward tiến trình
    UC->>PROG: PublishProgress({step: render_scenes, status: in_progress, scene_index: 2, scene_total: 5})
    CONSUMER->>MQ: ack
```

## Flow 4: Step Failure — Compensating Action (Retry, không rollback)

```mermaid
sequenceDiagram
    participant MQ as RabbitMQ
    participant CONSUMER as adapters/amqp/consumer.go
    participant UC as HandleStepEventUseCase
    participant REPO as ProjectRepository
    participant PROG as amqp.Publisher

    MQ->>CONSUMER: deliver rendering_failed (error_message)
    CONSUMER->>UC: handle(event) [goroutine]
    UC->>REPO: GetStep(saga_id, "render_scenes")
    REPO-->>UC: SagaStep{status: in_progress}
    UC->>REPO: UpdateStep(status: failed, error_message)
    UC->>REPO: UpdateStatus(failed_at_render_scenes)
    Note over UC: KHÔNG rollback artifact — animation/audio đã render giữ nguyên (services.md)
    UC->>PROG: PublishProgress({step: render_scenes, status: failed, error_message})
    CONSUMER->>MQ: ack

    Note over MQ: Creator thấy lỗi qua SSE (Gateway/progress.fanout), bấm nút retry
```

## Flow 5: Retry Failed Step

```mermaid
sequenceDiagram
    participant GW as API Gateway
    participant API as adapters/http/router.go
    participant UC as RetryStepUseCase
    participant REPO as ProjectRepository
    participant OUTBOX as OutboxRepository
    participant RELAY as OutboxRelay
    participant MQ as RabbitMQ

    GW->>API: POST /v1/projects/{id}/retry
    API->>UC: execute(project_id)
    UC->>REPO: Get(project_id)
    REPO-->>UC: Project{status: failed_at_render_scenes, scenes: [...]}
    UC->>UC: rebuild render_scenes payload from stored Project data
    UC->>REPO: UpdateStep(SagaStep{step: render_scenes, status: in_progress})
    UC->>OUTBOX: enqueue command render_scenes (NEW message_id)
    UC->>REPO: UpdateStatus(rendering)
    UC-->>API: {saga_id, status: rendering}
    API-->>GW: 200 {saga_id, status: rendering}

    RELAY->>MQ: publish render_scenes (poll định kỳ)
    Note over MQ: Rendering Service's artifact-level idempotency (Unit 5)<br/>skip scene đã render thành công, chỉ render lại scene lỗi
```

## Flow 6: Unexpected/Out-of-Order Event (Question 5's safeguard)

```mermaid
sequenceDiagram
    participant MQ as RabbitMQ
    participant CONSUMER as adapters/amqp/consumer.go
    participant UC as HandleStepEventUseCase
    participant REPO as ProjectRepository

    MQ->>CONSUMER: deliver scenes_classified (saga_id=X)
    CONSUMER->>UC: handle(event)
    UC->>REPO: GetStep(saga_id=X, "classify_scenes")
    REPO-->>UC: SagaStep{status: completed} (đã xử lý trước đó — không phải in_progress)
    UC->>UC: log warning "unexpected event for non-in-progress step", skip processing
    CONSUMER->>MQ: ack (Inbox vẫn dedupe theo message_id nếu là redelivery thật)
```

## Flow 7: Successful Publish Saga

```mermaid
sequenceDiagram
    participant GW as API Gateway
    participant API as adapters/http/router.go
    participant UC as StartPublishSagaUseCase
    participant REPO as ProjectRepository
    participant OUTBOX as OutboxRepository
    participant MQ as RabbitMQ
    participant CONSUMER as adapters/amqp/consumer.go
    participant HUC as HandleStepEventUseCase

    GW->>API: POST /v1/sagas/publish {youtube_title, visibility, ...}
    API->>UC: execute(input)
    UC->>REPO: Get(project_id)
    REPO-->>UC: Project{status: ready_to_publish}
    UC->>REPO: UpdateStep(SagaStep{step: publish_video, status: in_progress})
    UC->>OUTBOX: enqueue command publish_video
    UC->>REPO: UpdateStatus(publishing)
    UC-->>API: {saga_id, status: started}

    MQ->>CONSUMER: deliver video_published (youtube_video_url)
    CONSUMER->>HUC: handle(event)
    HUC->>REPO: UpdateStep(status: completed)
    HUC->>REPO: Save(Project{youtube_video_url})
    HUC->>REPO: UpdateStatus(published)
    Note over HUC: Saga Publish kết thúc — không dispatch command tiếp theo (bước cuối)
```
