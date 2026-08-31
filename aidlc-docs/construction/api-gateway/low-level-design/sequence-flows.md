# Sequence Flows — Unit 9: API Gateway

## Flow 1: Generic REST Proxy (vd. GET /v1/plugins)

```mermaid
sequenceDiagram
    participant GUI as Web GUI
    participant MW as middleware/correlation.js
    participant RT as routes/plugins.js
    participant PH as handlers/proxyHandler.js
    participant CL as clients/httpClient.js
    participant CP as Content Plugin Service

    GUI->>MW: GET /v1/plugins
    MW->>MW: X-Request-ID có sẵn? Nếu không, sinh mới
    MW->>RT: forward (req có X-Request-ID)
    RT->>PH: proxyHandler(contentPluginClient).handle(req)
    PH->>CL: request({method, headers, body})
    CL->>CP: GET /v1/plugins (forward header)
    CP-->>CL: 200 [...plugins]
    CL-->>PH: response
    PH-->>GUI: 200 [...plugins] (nguyên trạng)
```

## Flow 2: OAuth Redirect (GET /v1/auth/youtube/start)

```mermaid
sequenceDiagram
    participant GUI as Web GUI (Creator click "Connect YouTube")
    participant GW as API Gateway
    participant PUB as Publisher Service
    participant Google as Google OAuth

    GUI->>GW: GET /v1/auth/youtube/start
    GW->>PUB: GET /v1/auth/youtube/start (proxy)
    PUB-->>GW: 302 Found (Location: accounts.google.com/...)
    GW-->>GUI: 302 Found (forward Location nguyên trạng)
    GUI->>Google: Redirect theo Location
    Google->>GW: GET /v1/auth/youtube/callback?code=...
    GW->>PUB: GET /v1/auth/youtube/callback?code=... (proxy)
    PUB->>PUB: đổi code lấy token, lưu oauth_credentials
    PUB-->>GW: 200 {connected: true}
    GW-->>GUI: 200 {connected: true}
```

## Flow 3: SSE Subscribe + AMQP Fan-out

```mermaid
sequenceDiagram
    participant GUI as Web GUI
    participant RT as routes/progress.js
    participant PGH as handlers/progressHandler.js
    participant AMQP as clients/amqpClient.js
    participant MQ as RabbitMQ (progress.fanout)
    participant ORCH as Orchestrator Service

    GUI->>RT: GET /v1/progress/{project_id} (SSE)
    RT->>PGH: subscribe(project_id, res)
    PGH->>PGH: connections.set(project_id, [...existing, res])
    PGH-->>GUI: 200 (Content-Type: text/event-stream, kết nối giữ mở)

    ORCH->>MQ: publish ProgressMessage {project_id, step, status}
    MQ->>AMQP: deliver (Gateway's exclusive queue)
    AMQP->>PGH: onMessage(msg)
    PGH->>PGH: lookup connections.get(msg.project_id)
    PGH-->>GUI: data: {"project_id":..., "step":..., "status":...}\n\n

    GUI->>PGH: connection closed (tab đóng/GUI dừng theo dõi)
    PGH->>PGH: connections.delete(res khỏi mảng project_id)
```

## Flow 4: Downstream Error/Timeout

```mermaid
sequenceDiagram
    participant GUI as Web GUI
    participant PH as handlers/proxyHandler.js
    participant CL as clients/httpClient.js
    participant SVC as Service downstream (bất kỳ)

    GUI->>PH: request bất kỳ (vd. POST /v1/sagas/render)
    PH->>CL: request(...)
    alt Service trả lỗi (4xx/5xx)
        CL->>SVC: forward request
        SVC-->>CL: 409 {error: "..."}
        CL-->>PH: response (409, body)
        PH-->>GUI: 409 {error: "..."} (forward nguyên trạng)
    else Service không kết nối được (timeout/connection refused)
        CL->>SVC: forward request
        SVC--xCL: connection error/timeout
        CL-->>PH: throw error
        PH-->>GUI: 502 {error: "upstream_unavailable", service: "orchestrator"}
    end
```
