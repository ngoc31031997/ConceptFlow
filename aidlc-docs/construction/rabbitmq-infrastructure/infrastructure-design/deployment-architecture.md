# Deployment Architecture — Unit 1: RabbitMQ Infrastructure

## docker-compose Service Definition (reference for Code Generation)

```yaml
services:
  rabbitmq:
    image: rabbitmq:3.13-management
    container_name: rabbitmq
    environment:
      RABBITMQ_DEFAULT_USER: ${RABBITMQ_USER}
      RABBITMQ_DEFAULT_PASS: ${RABBITMQ_PASS}
    ports:
      - "15672:15672"   # Management UI (dev/debug only)
    # 5672 (AMQP) intentionally NOT published to host — internal docker network only
    volumes:
      - rabbitmq_data:/var/lib/rabbitmq
    healthcheck:
      test: ["CMD", "rabbitmq-diagnostics", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - backend

volumes:
  rabbitmq_data:

networks:
  backend:
    driver: bridge
```

## Database Topology
N/A — RabbitMQ không phải database quan hệ; không có primary/replica hay sharding.

## Load Balancer Configuration
N/A — 1 instance duy nhất.

## API Gateway Configuration
N/A trong phạm vi unit này — xem `aidlc-docs/inception/high-level-design/integration-boundaries.md` (ADR-0004) và Unit 9 (API Gateway) cho quyết định gateway ở cấp hệ thống.

## Scaling Configuration
- **Type**: Fixed, 1 instance
- **Auto-scaling**: Không áp dụng
- **Trigger**: N/A

## Dependent Services (consume this infrastructure)
Content Plugin (Unit 2), Script Processing (Unit 4), Rendering (Unit 5), Video Assembly (Unit 6), Publisher (Unit 7), Orchestrator (Unit 8) — tất cả dùng `depends_on: rabbitmq: condition: service_healthy` trong `docker-compose.yml` gốc (theo `unit-of-work.md`, code organization strategy).
