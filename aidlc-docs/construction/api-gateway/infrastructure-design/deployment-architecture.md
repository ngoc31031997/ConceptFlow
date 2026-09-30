# Deployment Architecture — Unit 9: API Gateway

> **Cập nhật (CR-055, 2026-09-30)**: Content Plugin Service (và route `GET /v1/plugins`) gỡ ở CR-020; từ CR-040 Gateway proxy thêm tới Authoring Service (`AUTHORING_SERVICE_URL`). Danh sách route thật (≈90 route) ở `services/api-gateway/src/routes/`; bảng dưới chỉ giữ các route gốc của Unit 9.

## Dockerfile (reference for Code Generation)
```dockerfile
FROM node:20-alpine
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --omit=dev
COPY . .
EXPOSE 8080
CMD ["node", "src/server.js"]
```

## docker-compose Service Entry (reference for Code Generation)
```yaml
services:
  api-gateway:
    build: ./services/api-gateway
    container_name: api-gateway
    environment:
      RABBITMQ_URL: amqp://${RABBITMQ_USER}:${RABBITMQ_PASS}@rabbitmq:5672/
      ORCHESTRATOR_URL: http://orchestrator:8000
      AUTHORING_SERVICE_URL: http://authoring-service:8000
      PUBLISHER_URL: http://publisher:8000
      PORT: "8080"
    ports:
      - "8080:8080"
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8080/health"]
      interval: 10s
      timeout: 5s
      retries: 5
    depends_on:
      rabbitmq:
        condition: service_healthy
      orchestrator:
        condition: service_healthy
      authoring-service:
        condition: service_healthy
      publisher:
        condition: service_healthy
    networks:
      - backend
```

**Note**: Gateway là service DUY NHẤT (ngoài RabbitMQ Management UI) publish port ra host — đúng vai trò entry point.

## Database Topology
Không áp dụng — Gateway không có database.

## Load Balancer / API Gateway
N/A load balancer (1 instance). Unit 9 chính là API Gateway của hệ thống (không có lớp gateway nào khác phía trước).

## Scaling Configuration
- **Type**: Fixed, 1 instance
- **Auto-scaling**: Không áp dụng
