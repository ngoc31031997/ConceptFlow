# Mục lục hợp đồng giữa các service

Những gì graphify không thấy (message RabbitMQ, HTTP giữa service, volume dùng chung) được ghi ở các file dưới đây. Đọc file tương ứng thay vì dò code của từng service.

## Volume `shared_artifacts`

| Hợp đồng | File |
|---|---|
| Cây thư mục `/shared`, service nào ghi thư mục nào | [`shared-artifacts.md`](shared-artifacts.md) — giữ khớp bởi `tests/contracts/test_shared_artifacts_contract.py` |

## RabbitMQ

| Hợp đồng | File |
|---|---|
| Topology thật (exchange, queue, DLQ, binding) | [`infra/rabbitmq/definitions.json`](../../infra/rabbitmq/definitions.json) |
| Envelope, routing key, quy ước command/event, DLQ | [`rabbitmq-infrastructure/nfr-design/messaging-design.md`](../../aidlc-docs/construction/rabbitmq-infrastructure/nfr-design/messaging-design.md) |
| Command orchestrator gửi và event nhận về (Saga) | [`orchestrator-service/nfr-design/messaging-design.md`](../../aidlc-docs/construction/orchestrator-service/nfr-design/messaging-design.md), ADR-0007, ADR-0019 |
| Tiến độ qua `progress.fanout` tới SSE | [`api-gateway/nfr-design/messaging-design.md`](../../aidlc-docs/construction/api-gateway/nfr-design/messaging-design.md), ADR-0017 |
| Command/event của từng service xử lý | `interface-contracts.md` của [tts](../../aidlc-docs/construction/tts-service/low-level-design/interface-contracts.md), [rendering](../../aidlc-docs/construction/rendering-service/low-level-design/interface-contracts.md), [video-assembly](../../aidlc-docs/construction/video-assembly-service/low-level-design/interface-contracts.md), [publisher](../../aidlc-docs/construction/publisher-service/low-level-design/interface-contracts.md) |

## HTTP

| Bên gọi → bên nhận | Hợp đồng |
|---|---|
| web-gui → api-gateway (`/v1/*`, SSE) | [`api-gateway/low-level-design/interface-contracts.md`](../../aidlc-docs/construction/api-gateway/low-level-design/interface-contracts.md), [`web-gui/low-level-design/interface-contracts.md`](../../aidlc-docs/construction/web-gui/low-level-design/interface-contracts.md); route thật ở `services/api-gateway/src/routes/` |
| api-gateway → orchestrator | [`orchestrator-service/low-level-design/interface-contracts.md`](../../aidlc-docs/construction/orchestrator-service/low-level-design/interface-contracts.md) |
| api-gateway → publisher (OAuth YouTube) | [`publisher-service/low-level-design/interface-contracts.md`](../../aidlc-docs/construction/publisher-service/low-level-design/interface-contracts.md), ADR-0026 |
| api-gateway → authoring-service; orchestrator ↔ authoring-service (`/internal/v1/authoring/*`) | [ADR-0029](../../aidlc-docs/decisions/ADR-0029-authoring-service-boundary.md), [`cr-040-service-boundary-cleanup.md`](../../aidlc-docs/inception/requirements/cr-040-service-boundary-cleanup.md) |
| authoring-service → llm-service (`/v1/chat`, `/v1/storyboard/finalize`, `/v1/code/generate`, `/v1/suggest-*`) | [`cr-039-chunked-code-pipeline-llm-service.md`](../../aidlc-docs/inception/requirements/cr-039-chunked-code-pipeline-llm-service.md); route thật ở `services/llm-service/app/main.py` |
| llm-service, authoring-service → rendering (`/v1/check/*`, `/v1/illustrations/preview`) | CR-039 (như trên); route thật ở `services/rendering/adapters/http/check_server.py` |

## Cơ sở dữ liệu

Mỗi service có Postgres riêng (ADR-0013); không service nào đọc DB của service khác. Lược đồ nằm trong code migration của từng service.
