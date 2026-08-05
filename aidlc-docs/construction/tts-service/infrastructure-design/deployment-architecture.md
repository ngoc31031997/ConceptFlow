# Deployment Architecture — Unit 3: TTS Service

## Dockerfile (reference for Code Generation)
```dockerfile
FROM python:3.12-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    tar \
    && rm -rf /var/lib/apt/lists/*

# Install the standalone Piper CLI binary (Code Generation revision: piper-tts's
# PyPI package depends on piper-phonemize, which has no prebuilt wheel for several
# platforms — the adapter shells out to this binary instead, per module-structure.md's
# "Piper CLI/binding" option)
RUN curl -L -o /tmp/piper.tar.gz <piper-release-tarball-url> && \
    tar -xzf /tmp/piper.tar.gz -C /usr/local/bin --strip-components=1 && \
    rm /tmp/piper.tar.gz

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

**Note**: URL binary/model cụ thể (Piper release trên GitHub, voice model trên Hugging Face `rhasspy/piper-voices`) sẽ được điền chính xác khi build image thật — cần network access lúc build, không lúc runtime (đúng NFR Requirements).

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
