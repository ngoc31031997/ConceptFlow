# Deployment Architecture — Unit 2: Content Plugin Service

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
  content-plugin:
    build: ./services/content-plugin
    container_name: content-plugin
    environment:
      RABBITMQ_URL: amqp://${RABBITMQ_USER}:${RABBITMQ_PASS}@rabbitmq:5672/
    healthcheck:
      test: ["CMD", "python", "-c", "import urllib.request; urllib.request.urlopen('http://localhost:8000/health')"]
      interval: 10s
      timeout: 5s
      retries: 5
    depends_on:
      rabbitmq:
        condition: service_healthy
    networks:
      - backend
```

## Database Topology
N/A — stateless.

## Load Balancer / API Gateway
N/A trong phạm vi unit này.

## Scaling Configuration
- **Type**: Fixed, 1 instance
- **Auto-scaling**: Không áp dụng
