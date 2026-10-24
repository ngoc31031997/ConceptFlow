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
| web-gui → api-gateway → authoring-service (bước Ý tưởng: `/v1/projects/{id}/idea…`, bảng `project_idea` và `project_idea_review`); authoring-service → orchestrator (`GET /internal/v1/projects/{id}/idea-input`) | [`idea-v1.md`](idea-v1.md) |
| web-gui → api-gateway → authoring-service (bước Creative brief: `/v1/projects/{id}/creative-brief…`, bảng `project_brief` và `project_brief_review`) | [`creative-brief-v1.md`](creative-brief-v1.md) |
| web-gui → api-gateway → authoring-service (bước Nghiên cứu: `/v1/projects/{id}/research…`, mẫu JSON dán vào `conceptflow.research.v1`); authoring-service → Internet (tải trang nguồn để kiểm đoạn trích, chặn địa chỉ nội bộ) | [`research-v1.md`](research-v1.md), [ADR-0043](../../aidlc-docs/decisions/ADR-0043-pasted-research-with-source-verification.md) |
| web-gui → api-gateway → authoring-service (bước Câu chuyện: `/v1/projects/{id}/story…`, các đơn vị `story`, `story_field`, `story_approach`) | [`story-v1.md`](story-v1.md) |
| web-gui → api-gateway → authoring-service (bước Beat sheet: `/v1/projects/{id}/beats…`, các đơn vị `beat`, `beat_sheet`) | [`beats-v1.md`](beats-v1.md) |
| web-gui → api-gateway → authoring-service (bước Viết kịch bản: `/v1/projects/{id}/script…`, đơn vị `script_beat`); phát file giọng `/v1/projects/{id}/audio/{file}`; authoring-service → orchestrator (`POST /internal/v1/projects/{id}/draft-voice`) → tts (`synthesize_lines`) → orchestrator (`lines_synthesized`) → authoring-service (`/internal/v1/authoring/{id}/voice-lines/*`) | [`script-v1.md`](script-v1.md), [ADR-0040](../../aidlc-docs/decisions/ADR-0040-draft-voice-over-rabbitmq-and-dictionary.md), [ADR-0041](../../aidlc-docs/decisions/ADR-0041-shared-volume-as-file-store.md) |
| web-gui → api-gateway → authoring-service (bước Rà soát và chốt: `/v1/projects/{id}/script-lock…`, đơn vị `handoff_beat`); phiên bản kịch bản đã chốt đọc qua `/internal/v1/authoring/{id}/handoff?role=…` | [`script-lock-v1.md`](script-lock-v1.md), [ADR-0044](../../aidlc-docs/decisions/ADR-0044-locked-script-versions-and-handoff.md) |
| authoring-service → llm-service (`/v1/chat`, `/v1/storyboard/finalize`, `/v1/suggest-*`) | [`cr-039-chunked-code-pipeline-llm-service.md`](../../aidlc-docs/inception/requirements/cr-039-chunked-code-pipeline-llm-service.md); route thật ở `services/llm-service/app/main.py` |
| authoring-service → llm-service (`/v2/code/generate`, `/v2/code/plan`, `/v2/code/segment-prompt`, `/v2/code/segment-parse`; `/v1/code/generate` đã bỏ ở CR-056) | [`authoring-llm-code-v2.md`](authoring-llm-code-v2.md), [ADR-0030](../../aidlc-docs/decisions/ADR-0030-code-segments-and-v2-code-contract.md), [ADR-0031](../../aidlc-docs/decisions/ADR-0031-frame-orientation-by-output-mode.md) |
| llm-service, authoring-service → rendering (`/v1/check/*`, `/v1/illustrations/preview`) | CR-039 (như trên); diagnostic có thêm `rule` từ CR-050 ([`authoring-llm-code-v2.md`](authoring-llm-code-v2.md)); route thật ở `services/rendering/adapters/http/check_server.py` |
| authoring-service → rendering (`/v1/kits`, `/v1/stills`, `/v1/stills/prune`); web-gui → api-gateway → authoring-service (`/v1/visual-languages`, `/v1/projects/{id}/visual-language`, `/v1/projects/{id}/illustrations/{rid}/tier`); orchestrator → authoring (`kit_key`) → payload `render_scenes.kit_key` | ADR-0036; route thật ở `services/rendering/adapters/http/check_server.py`, `services/authoring-service/internal/adapters/http/visual_languages.go`; `kit_key` bắt buộc ở mọi lượt kiểm/xem trước/render |

### Đổi contract HTTP giữa hai service
- Mỗi service phải deploy được một mình. Đổi contract theo ba bước: **thêm bản mới, giữ bản cũ, bỏ bản cũ ở lần sau**.
- Không bao giờ xoá bản cũ trong cùng thay đổi thêm bản mới.
- Bên gọi dùng bản mới khi có, và quay về bản cũ khi bên nhận chưa có (404 cho chính route).
- Việc bỏ bản cũ được ghi vào backlog (`aidlc-docs/aidlc-state.md`), và chỉ làm khi mọi bên gọi đang chạy đã dùng bản mới.
- Chỉ thêm trường thì không cần version mới, nếu bên nhận cũ bỏ qua được trường lạ.
- Ví dụ: `/v1` và `/v2/code/*` (CR-050, ADR-0030); `/v1/code/generate` bị bỏ ở CR-056.

## Cơ sở dữ liệu

Mỗi service có Postgres riêng (ADR-0013); không service nào đọc DB của service khác. Lược đồ nằm trong code migration của từng service.
