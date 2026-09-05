# Deployment Architecture — Unit 10: Web GUI

## Dockerfile (multi-stage)

```dockerfile
# Stage 1: build
FROM node:20-alpine AS build
WORKDIR /app
ARG VITE_API_BASE_URL=http://localhost:8080
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

# Stage 2: serve
FROM nginx:alpine
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
```

## nginx.conf
SPA cần fallback về `index.html` cho client-side routing (`react-router-dom`):

```nginx
server {
    listen 80;
    root /usr/share/nginx/html;
    index index.html;

    location / {
        try_files $uri /index.html;
    }
}
```

## docker-compose.yml — service entry

```yaml
  web-gui:
    build:
      context: ./web-gui
      args:
        VITE_API_BASE_URL: http://localhost:8080
    container_name: web-gui
    ports:
      - "3000:80"
    networks:
      - backend
```

Không có `depends_on`/`healthcheck` bắt buộc — GUI không có dependency khởi động nội bộ (mọi gọi API là từ browser, không phải từ container lúc start).
