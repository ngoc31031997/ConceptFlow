# Deployment Architecture — Unit 3: TTS Service

**Revision (2026-08-07, ADR-0014, ADR-0013)**: Dockerfile no longer runs `uvicorn`/FastAPI; docker-compose entry adds `tts-db` (Postgres) and drops the HTTP healthcheck in favor of a sentinel file.

## Dockerfile (reference for Code Generation)
```dockerfile
FROM python:3.12-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    tar \
    && rm -rf /var/lib/apt/lists/*

# Install the standalone Piper CLI binary (unchanged from original Code Generation)
RUN curl -L -o /tmp/piper.tar.gz <piper-release-tarball-url> && \
    tar -xzf /tmp/piper.tar.gz -C /usr/local/bin --strip-components=1 && \
    rm /tmp/piper.tar.gz

WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Bundle Piper voice models at build time (unchanged)
RUN mkdir -p /app/voices && \
    curl -L -o /app/voices/vi.onnx <piper-vi-model-url> && \
    curl -L -o /app/voices/vi.onnx.json <piper-vi-model-config-url> && \
    curl -L -o /app/voices/en.onnx <piper-en-model-url> && \
    curl -L -o /app/voices/en.onnx.json <piper-en-model-config-url>

COPY . .
CMD ["python", "main.py"]
```

**Note**: `EXPOSE 8000` removed — no longer serving HTTP. `CMD` changed from `uvicorn` to a plain Python entrypoint that starts the AMQP consumer + `OutboxRelay` (`main.py` no longer defines a FastAPI `create_app()` factory).

## docker-compose Service Entry (reference for Code Generation)
```yaml
services:
  tts-db:
    image: postgres:16-alpine
    container_name: tts-db
    environment:
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASS}
      POSTGRES_DB: tts
    volumes:
      - tts_db_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER} -d tts"]
      interval: 5s
      timeout: 5s
      retries: 10
    networks:
      - backend

  tts:
    build: ./services/tts
    container_name: tts
    environment:
      RABBITMQ_URL: amqp://${RABBITMQ_USER}:${RABBITMQ_PASS}@rabbitmq:5672/
      DATABASE_URL: postgresql://${POSTGRES_USER}:${POSTGRES_PASS}@tts-db:5432/tts
    healthcheck:
      test: ["CMD", "test", "-f", "/tmp/ready"]
      interval: 10s
      timeout: 5s
      retries: 5
    depends_on:
      rabbitmq:
        condition: service_healthy
      tts-db:
        condition: service_healthy
    volumes:
      - shared_artifacts:/shared
    networks:
      - backend

volumes:
  shared_artifacts:
  tts_db_data:
```

**Note**: `shared_artifacts` volume unchanged (still shared with Rendering Service, Unit 5). `tts_db_data` is new, private to `tts-db` (database-per-service, ADR-0013).

## Database Topology
**Revised**: 1 PostgreSQL instance (`tts-db`), single primary, no read replicas — same rationale as Unit 2 (Outbox/Inbox tables only, negligible volume).

## Load Balancer / API Gateway
N/A — không đổi (không còn REST endpoint nào để cân nhắc Gateway).

## Scaling Configuration
- **Type**: Fixed, 1 instance
- **Auto-scaling**: Không áp dụng
