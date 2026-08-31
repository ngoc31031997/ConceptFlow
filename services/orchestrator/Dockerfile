# Build stage
FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o orchestrator ./cmd/orchestrator

# Final stage
FROM alpine:3.19
WORKDIR /app
COPY --from=build /app/orchestrator .
EXPOSE 8000
CMD ["./orchestrator"]
