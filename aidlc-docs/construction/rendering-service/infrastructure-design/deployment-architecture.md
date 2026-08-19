# Deployment Architecture — Unit 5: Rendering Service

## Dockerfile (reference for Code Generation)
```dockerfile
FROM python:3.12-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    ffmpeg \
    libcairo2-dev \
    libpango1.0-dev \
    pkg-config \
    build-essential \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY . .
CMD ["python", "main.py"]
```

**Note**: Không cài `texlive` (LaTeX) — Manim's `Tex`/`MathTex` mobject sẽ lỗi nếu dùng, nhưng 2 template MVP (`algorithm_visualization`, `concept_illustration`) không cần công thức toán.

## docker-compose Service Entry (reference for Code Generation)
```yaml
services:
  rendering-db:
    image: postgres:16-alpine
    container_name: rendering-db
    environment:
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASS}
      POSTGRES_DB: rendering
    volumes:
      - rendering_db_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER} -d rendering"]
      interval: 5s
      timeout: 5s
      retries: 10
    networks:
      - backend

  rendering:
    build: ./services/rendering
    container_name: rendering
    environment:
      RABBITMQ_URL: amqp://${RABBITMQ_USER}:${RABBITMQ_PASS}@rabbitmq:5672/
      DATABASE_URL: postgresql://${POSTGRES_USER}:${POSTGRES_PASS}@rendering-db:5432/rendering
      RENDER_TIMEOUT_SECONDS: "300"
    healthcheck:
      test: ["CMD", "test", "-f", "/tmp/ready"]
      interval: 10s
      timeout: 5s
      retries: 5
    depends_on:
      rabbitmq:
        condition: service_healthy
      rendering-db:
        condition: service_healthy
    volumes:
      - shared_artifacts:/shared
    networks:
      - backend

volumes:
  rendering_db_data:
```

**Note**: `shared_artifacts` volume đã khai báo ở root `docker-compose.yml` (từ Unit 3) — không khai báo lại, chỉ mount thêm vào service `rendering`.

## Database Topology
1 PostgreSQL instance (`rendering-db`), single primary, no read replicas — Outbox/Inbox tables only. Database-per-service (ADR-0013).

## Load Balancer / API Gateway
N/A trong phạm vi unit này.

## Scaling Configuration
- **Type**: Fixed, 1 instance
- **Auto-scaling**: Không áp dụng
