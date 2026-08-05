# Sequence Flows — Unit 2: Content Plugin Service

## Flow 1: List Plugins (REST, ngoài Saga)

```mermaid
sequenceDiagram
    participant GW as API Gateway
    participant API as adapters/api/router.py
    participant UC as ListPluginsUseCase
    participant REG as ContentPluginRegistry

    GW->>API: GET /v1/plugins (X-Request-ID)
    API->>UC: execute()
    UC->>REG: list_all()
    REG-->>UC: [ProgrammingPlugin, ...]
    UC-->>API: PluginDTO[]
    API-->>GW: 200 { plugins: [...] } (X-Request-ID echoed)
```

## Flow 2: Classify Scenes (AMQP, trong Saga)

```mermaid
sequenceDiagram
    participant MQ as RabbitMQ
    participant CONSUMER as adapters/messaging/consumer.py
    participant UC as ClassifySceneUseCase
    participant REG as ContentPluginRegistry
    participant PLUGIN as ProgrammingPlugin
    participant PRODUCER as adapters/messaging/producer.py

    MQ->>CONSUMER: deliver classify_scenes (saga_id, scenes[])
    CONSUMER->>CONSUMER: check message_id in dedupe set (idempotency)
    alt message already processed
        CONSUMER->>MQ: ack (no-op, already handled)
    else new message
        loop mỗi scene
            CONSUMER->>UC: execute(plugin_id, scene)
            UC->>REG: get(plugin_id)
            alt plugin not found
                REG-->>UC: raise PluginNotFoundError
                UC-->>CONSUMER: error
                CONSUMER->>PRODUCER: publish classification_failed
                CONSUMER->>MQ: ack (không retry — lỗi cấu hình, không phải lỗi tạm thời)
            else plugin found
                REG-->>UC: ProgrammingPlugin instance
                UC->>PLUGIN: classify(scene)
                PLUGIN-->>UC: ClassificationResult(category, template_id)
                UC-->>CONSUMER: result
            end
        end
        CONSUMER->>PRODUCER: publish scenes_classified (all results)
        CONSUMER->>MQ: ack
    end
```

## Flow 3: Service Startup — Dynamic Plugin Discovery

```mermaid
sequenceDiagram
    participant MAIN as main.py
    participant REG as ContentPluginRegistry
    participant FS as adapters/plugins/*.py

    MAIN->>REG: discover("adapters/plugins/")
    loop mỗi file .py trong thư mục
        REG->>FS: importlib.import_module(file)
        alt import lỗi hoặc không implement ContentPluginPort
            REG->>REG: log warning, bỏ qua file này
        else hợp lệ
            REG->>REG: đăng ký instance vào registry dict
        end
    end
    REG-->>MAIN: registry sẵn sàng (>= 0 plugin đã đăng ký)
    MAIN->>MAIN: wire registry vào FastAPI app + AMQP consumer
```
