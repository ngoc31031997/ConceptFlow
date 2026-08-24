# Deployment Architecture — Unit 6: Video Assembly Service

## Dockerfile (reference for Code Generation)
```dockerfile
FROM python:3.12-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
CMD ["python", "main.py"]
```

**Note**: `ffmpeg` package bundles `ffprobe` — không cần cài riêng.

## docker-compose Service Entry (reference for Code Generation)
```yaml
services:
  video-assembly-db:
    image: postgres:16-alpine
    container_name: video-assembly-db
    environment:
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASS}
      POSTGRES_DB: video_assembly
    volumes:
      - video_assembly_db_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER} -d video_assembly"]
      interval: 5s
      timeout: 5s
      retries: 10
    networks:
      - backend

  video-assembly:
    build: ./services/video-assembly
    container_name: video-assembly
    environment:
      RABBITMQ_URL: amqp://${RABBITMQ_USER}:${RABBITMQ_PASS}@rabbitmq:5672/
      DATABASE_URL: postgresql://${POSTGRES_USER}:${POSTGRES_PASS}@video-assembly-db:5432/video_assembly
      ASSEMBLY_TIMEOUT_SECONDS: "180"
    healthcheck:
      test: ["CMD", "test", "-f", "/tmp/ready"]
      interval: 10s
      timeout: 5s
      retries: 5
    depends_on:
      rabbitmq:
        condition: service_healthy
      video-assembly-db:
        condition: service_healthy
    volumes:
      - shared_artifacts:/shared
    networks:
      - backend

volumes:
  video_assembly_db_data:
```

**Note**: `shared_artifacts` volume đã khai báo ở root `docker-compose.yml` (từ Unit 3) — không khai báo lại, chỉ mount thêm vào service `video-assembly`.

## Database Topology
1 PostgreSQL instance (`video-assembly-db`), single primary, no read replicas — Outbox/Inbox tables only. Database-per-service (ADR-0013).

## Load Balancer / API Gateway
N/A trong phạm vi unit này.

## Scaling Configuration
- **Type**: Fixed, 1 instance
- **Auto-scaling**: Không áp dụng
