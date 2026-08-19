# Deployment Architecture — Unit 4: Script Processing Service

## Dockerfile (reference for Code Generation)
```dockerfile
FROM python:3.12-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
CMD ["python", "main.py"]
```

## docker-compose Service Entry (reference for Code Generation)
```yaml
services:
  script-processing-db:
    image: postgres:16-alpine
    container_name: script-processing-db
    environment:
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASS}
      POSTGRES_DB: script_processing
    volumes:
      - script_processing_db_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER} -d script_processing"]
      interval: 5s
      timeout: 5s
      retries: 10
    networks:
      - backend

  script-processing:
    build: ./services/script-processing
    container_name: script-processing
    environment:
      RABBITMQ_URL: amqp://${RABBITMQ_USER}:${RABBITMQ_PASS}@rabbitmq:5672/
      DATABASE_URL: postgresql://${POSTGRES_USER}:${POSTGRES_PASS}@script-processing-db:5432/script_processing
    healthcheck:
      test: ["CMD", "test", "-f", "/tmp/ready"]
      interval: 10s
      timeout: 5s
      retries: 5
    depends_on:
      rabbitmq:
        condition: service_healthy
      script-processing-db:
        condition: service_healthy
    networks:
      - backend

volumes:
  script_processing_db_data:
```

## Database Topology
1 PostgreSQL instance (`script-processing-db`), single primary, no read replicas — Outbox/Inbox tables only, negligible volume. Database-per-service (ADR-0013).

## Load Balancer / API Gateway
N/A trong phạm vi unit này.

## Scaling Configuration
- **Type**: Fixed, 1 instance
- **Auto-scaling**: Không áp dụng
