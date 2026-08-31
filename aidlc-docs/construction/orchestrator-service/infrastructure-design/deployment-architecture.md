# Deployment Architecture — Unit 8: Orchestrator Service

## Dockerfile (reference for Code Generation)
```dockerfile
# Build stage
FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o orchestrator ./cmd/orchestrator

# Final stage
FROM alpine:3.19
WORKDIR /app
COPY --from=build /app/orchestrator .
EXPOSE 8000
CMD ["./orchestrator"]
```

## docker-compose Service Entry (reference for Code Generation)
```yaml
services:
  orchestrator-db:
    image: postgres:16-alpine
    container_name: orchestrator-db
    environment:
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASS}
      POSTGRES_DB: orchestrator
    volumes:
      - orchestrator_db_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER} -d orchestrator"]
      interval: 5s
      timeout: 5s
      retries: 10
    networks:
      - backend

  orchestrator:
    build: ./services/orchestrator
    container_name: orchestrator
    environment:
      RABBITMQ_URL: amqp://${RABBITMQ_USER}:${RABBITMQ_PASS}@rabbitmq:5672/
      DATABASE_URL: postgresql://${POSTGRES_USER}:${POSTGRES_PASS}@orchestrator-db:5432/orchestrator
      OUTBOX_POLL_INTERVAL_MS: "500"
      DATABASE_MAX_CONNS: "10"
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8000/health"]
      interval: 10s
      timeout: 5s
      retries: 5
    depends_on:
      rabbitmq:
        condition: service_healthy
      orchestrator-db:
        condition: service_healthy
    networks:
      - backend

volumes:
  orchestrator_db_data:
```

**Note**: health check dùng `wget` (có sẵn trong `alpine`) thay vì `curl`/`python` (khác các unit Python dùng `python -c "urllib.request..."`) vì final image không có Python runtime.

## Database Topology
1 PostgreSQL instance (`orchestrator-db`), single primary, no read replicas. Database-per-service (ADR-0013). Không sharding.

## Load Balancer / API Gateway
N/A load balancer (1 instance). API Gateway (Unit 9) proxy 4 endpoint REST tới `orchestrator:8000` — routing rule cụ thể thiết kế ở Unit 9's Infrastructure Design.

## Scaling Configuration
- **Type**: Fixed, 1 instance
- **Auto-scaling**: Không áp dụng
