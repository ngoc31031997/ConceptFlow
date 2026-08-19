# Deployment Architecture — Unit 2: Content Plugin Service

**Revision (2026-08-07, ADR-0013)**: adds a dedicated `content-plugin-db` PostgreSQL container for the Inbox/Outbox tables.

## Dockerfile (reference for Code Generation)
```dockerfile
FROM python:3.12-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
EXPOSE 8000
CMD ["uvicorn", "main:create_app", "--factory", "--host", "0.0.0.0", "--port", "8000"]
```

## docker-compose Service Entry (reference for Code Generation)
```yaml
services:
  content-plugin-db:
    image: postgres:16-alpine
    container_name: content-plugin-db
    environment:
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASS}
      POSTGRES_DB: content_plugin
    volumes:
      - content_plugin_db_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER} -d content_plugin"]
      interval: 5s
      timeout: 5s
      retries: 10
    networks:
      - backend

  content-plugin:
    build: ./services/content-plugin
    container_name: content-plugin
    environment:
      RABBITMQ_URL: amqp://${RABBITMQ_USER}:${RABBITMQ_PASS}@rabbitmq:5672/
      DATABASE_URL: postgresql://${POSTGRES_USER}:${POSTGRES_PASS}@content-plugin-db:5432/content_plugin
    healthcheck:
      test: ["CMD", "python", "-c", "import urllib.request; urllib.request.urlopen('http://localhost:8000/health')"]
      interval: 10s
      timeout: 5s
      retries: 5
    depends_on:
      rabbitmq:
        condition: service_healthy
      content-plugin-db:
        condition: service_healthy
    networks:
      - backend

volumes:
  content_plugin_db_data:
```

## Database Topology
**Revised**: 1 PostgreSQL instance (`content-plugin-db`), single primary, no read replicas — Inbox/Outbox tables only, negligible read/write volume at this scale. Database-per-service (ADR-0013): not shared with any other unit.

## Load Balancer / API Gateway
N/A trong phạm vi unit này.

## Scaling Configuration
- **Type**: Fixed, 1 instance
- **Auto-scaling**: Không áp dụng
