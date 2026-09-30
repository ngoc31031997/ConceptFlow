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
| authoring-service → llm-service (`/v1/chat`, `/v1/storyboard/finalize`, `/v1/suggest-*`) | [`cr-039-chunked-code-pipeline-llm-service.md`](../../aidlc-docs/inception/requirements/cr-039-chunked-code-pipeline-llm-service.md); route thật ở `services/llm-service/app/main.py` |
| authoring-service → llm-service (`/v2/code/generate`, `/v2/code/plan`, `/v2/code/segment-prompt`, `/v2/code/segment-parse`; `/v1/code/generate` đã bỏ ở CR-056) | [`authoring-llm-code-v2.md`](authoring-llm-code-v2.md), [ADR-0030](../../aidlc-docs/decisions/ADR-0030-code-segments-and-v2-code-contract.md) |
| llm-service, authoring-service → rendering (`/v1/check/*`, `/v1/illustrations/preview`) | CR-039 (như trên); diagnostic có thêm `rule` từ CR-050 ([`authoring-llm-code-v2.md`](authoring-llm-code-v2.md)); route thật ở `services/rendering/adapters/http/check_server.py` |

### Đổi contract HTTP giữa hai service
- Mỗi service phải deploy được một mình. Đổi contract theo ba bước: **thêm bản mới, giữ bản cũ, bỏ bản cũ ở lần sau**.
- Không bao giờ xoá bản cũ trong cùng thay đổi thêm bản mới.
- Bên gọi dùng bản mới khi có, và quay về bản cũ khi bên nhận chưa có (404 cho chính route).
- Việc bỏ bản cũ được ghi vào backlog (`aidlc-docs/aidlc-state.md`), và chỉ làm khi mọi bên gọi đang chạy đã dùng bản mới.
- Chỉ thêm trường thì không cần version mới, nếu bên nhận cũ bỏ qua được trường lạ.
- Ví dụ: `/v1` và `/v2/code/*` (CR-050, ADR-0030); `/v1/code/generate` bị bỏ ở CR-056.

## Cơ sở dữ liệu

Mỗi service có Postgres riêng (ADR-0013); không service nào đọc DB của service khác. Lược đồ nằm trong code migration của từng service.
