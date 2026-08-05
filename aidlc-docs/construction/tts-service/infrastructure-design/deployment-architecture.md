# Deployment Architecture — Unit 3: TTS Service

## Dockerfile (reference for Code Generation)
```dockerfile
FROM python:3.12-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    libsndfile1 \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Bundle Piper voice models at build time (Low-Level Design Question 4)
RUN mkdir -p /app/voices && \
    curl -L -o /app/voices/vi.onnx <piper-vi-model-url> && \
    curl -L -o /app/voices/vi.onnx.json <piper-vi-model-config-url> && \
    curl -L -o /app/voices/en.onnx <piper-en-model-url> && \
    curl -L -o /app/voices/en.onnx.json <piper-en-model-config-url>

COPY . .
EXPOSE 8000
CMD ["uvicorn", "main:create_app", "--factory", "--host", "0.0.0.0", "--port", "8000"]
```

**Note**: URL model cụ thể (Piper voice model repository, ví dụ Hugging Face `rhasspy/piper-voices`) sẽ được xác định và điền chính xác ở Code Generation, khi lựa chọn voice cụ thể (giọng nam/nữ, chất lượng) cho tiếng Việt và tiếng Anh.

## docker-compose Service Entry (reference for Code Generation)
```yaml
services:
  tts:
    build: ./services/tts
    container_name: tts
    healthcheck:
      test: ["CMD", "python", "-c", "import urllib.request; urllib.request.urlopen('http://localhost:8000/health')"]
      interval: 10s
      timeout: 5s
      retries: 5
    volumes:
      - shared_artifacts:/shared
    networks:
      - backend

volumes:
  shared_artifacts:
```

**Note**: `shared_artifacts` named volume được khai báo ở root `docker-compose.yml` (cùng cấp với `rabbitmq_data` hiện có), dùng chung giữa TTS Service và Rendering Service (Unit 5, khi phát triển).

## Database Topology
N/A — stateless.

## Load Balancer / API Gateway
N/A trong phạm vi unit này.

## Scaling Configuration
- **Type**: Fixed, 1 instance
- **Auto-scaling**: Không áp dụng
