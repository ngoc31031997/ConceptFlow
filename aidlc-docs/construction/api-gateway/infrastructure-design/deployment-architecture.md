# Deployment Architecture — Unit 9: API Gateway

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
      CONTENT_PLUGIN_URL: http://content-plugin:8000
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
      content-plugin:
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
