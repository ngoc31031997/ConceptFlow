# Deployment Architecture — Unit 7: Publisher Service

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
  publisher-db:
    image: postgres:16-alpine
    container_name: publisher-db
    environment:
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASS}
      POSTGRES_DB: publisher
    volumes:
      - publisher_db_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER} -d publisher"]
      interval: 5s
      timeout: 5s
      retries: 10
    networks:
      - backend

  publisher:
    build: ./services/publisher
    container_name: publisher
    environment:
      RABBITMQ_URL: amqp://${RABBITMQ_USER}:${RABBITMQ_PASS}@rabbitmq:5672/
      DATABASE_URL: postgresql://${POSTGRES_USER}:${POSTGRES_PASS}@publisher-db:5432/publisher
      GOOGLE_OAUTH_CLIENT_ID: ${GOOGLE_OAUTH_CLIENT_ID}
      GOOGLE_OAUTH_CLIENT_SECRET: ${GOOGLE_OAUTH_CLIENT_SECRET}
      GOOGLE_OAUTH_REDIRECT_URI: ${GOOGLE_OAUTH_REDIRECT_URI}
      UPLOAD_TIMEOUT_SECONDS: "600"
    healthcheck:
      test: ["CMD", "python", "-c", "import urllib.request; urllib.request.urlopen('http://localhost:8000/health')"]
      interval: 10s
      timeout: 5s
      retries: 5
    depends_on:
      rabbitmq:
        condition: service_healthy
      publisher-db:
        condition: service_healthy
    volumes:
      - shared_artifacts:/shared:ro
    networks:
      - backend

volumes:
  publisher_db_data:
```

**Note**: `shared_artifacts` volume đã khai báo ở root `docker-compose.yml` (từ Unit 3) — không khai báo lại, chỉ mount thêm (read-only) vào service `publisher`.

## Database Topology
1 PostgreSQL instance (`publisher-db`), single primary, no read replicas — Outbox/Inbox tables + `oauth_credentials`. Database-per-service (ADR-0013).

## Load Balancer / API Gateway
N/A load balancer trong phạm vi unit này. API Gateway (Unit 9) proxy `/v1/auth/youtube/*` tới `publisher:8000` — routing rule cụ thể thiết kế ở Unit 9.

## Scaling Configuration
- **Type**: Fixed, 1 instance
- **Auto-scaling**: Không áp dụng
