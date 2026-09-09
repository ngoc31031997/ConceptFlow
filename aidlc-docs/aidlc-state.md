# AI-DLC State Tracking

## Project Information
- **Project Type**: Greenfield
- **Start Date**: 2026-08-04T00:00:00Z
- **Current Stage**: POST-CONSTRUCTION — cả 10/10 unit đã qua Code Generation và Build and Test. Công việc hiện tại là các Change Request trên hệ đã chạy, không còn theo vòng lặp per-unit.

## Extension Configuration
| Extension | Enabled | Decided At |
|---|---|---|
| Security Baseline | No | Requirements Analysis |
| Property-Based Testing | No | Requirements Analysis |

## Workspace State
- **Existing Code**: No
- **Reverse Engineering Needed**: No
- **Workspace Root**: /Users/hoangbaminhngoc/Documents/Project/ConcertFlow/ConceptFlow

## Code Location Rules
- **Application Code**: Workspace root (NEVER in aidlc-docs/)
- **Documentation**: aidlc-docs/ only
- **Structure patterns**: See code-generation.md Critical Rules

## Execution Plan Summary
- **Total Stages**: 11 EXECUTE, 1 SKIP (Operations)
- **Stages to Execute**: High-Level Design, Application Design, Units Generation, (per-unit) Low-Level Design, Functional Design, NFR Requirements, NFR Design, Infrastructure Design, Code Generation, Build and Test
- **Stages to Skip**: Operations — no cloud/production deployment target for this phase (Docker-local-first, per user decision)

## Stage Progress

### 🔵 INCEPTION PHASE
- [x] Workspace Detection
- [x] Requirements Analysis
- [x] User Stories
- [x] Workflow Planning
- [x] High-Level Design - EXECUTE
- [x] Application Design - EXECUTE
- [x] Units Generation - EXECUTE (10 units: RabbitMQ Infra, Content Plugin, TTS, Script Processing, Rendering, Video Assembly, Publisher, Orchestrator, API Gateway, Web GUI)

### 🟢 CONSTRUCTION PHASE
Đã chạy đủ cho cả 10 unit; artefact nằm dưới `aidlc-docs/construction/{unit}/`.
- [x] Low-Level Design (per-unit) - EXECUTE — 9/10 unit. SKIP có lý do: `rabbitmq-infrastructure` (không có module/class nội bộ, chỉ là definitions.json khai báo).
- [x] Functional Design (per-unit) - EXECUTE — 8/10 unit. SKIP có lý do: `rabbitmq-infrastructure` và `api-gateway` (proxy/định tuyến, không sinh business rule nào của riêng nó).
- [x] NFR Requirements (per-unit) - EXECUTE — 10/10
- [x] NFR Design (per-unit) - EXECUTE — 10/10
- [x] Infrastructure Design (per-unit) - EXECUTE — 10/10
- [x] Code Generation (per-unit) - EXECUTE — 10/10
- [x] Build and Test - EXECUTE — `aidlc-docs/construction/build-and-test/`

### 🟡 OPERATIONS PHASE
- [ ] Operations - SKIP

## Change Requests
| CR | Tên | Ưu tiên | Stage | Trạng thái |
|---|---|---|---|---|
| CR-001 | Tuỳ chọn TTS/phụ đề, chọn giọng | — | Delivered | Đã hoàn thành |
| CR-002 | Đồng bộ narration/phụ đề theo timeline thật | P0 blocker | **Code Generation ✅** | **HOÀN THÀNH** — verify E2E 0.003s |
| CR-003 | Năng lực render video dài 5–10 phút | P0 blocker | **Code Generation ✅** | **HOÀN THÀNH** (FR11.7 hoãn có lý do) |
| CR-004 | 1080p60 + profile encode chuẩn YouTube | P1 | **Code Generation ✅** | **HOÀN THÀNH + verify E2E ✅** |
| CR-005 | Giọng đọc chất lượng cao, ducking, loudnorm | P1 | **Code Generation ✅** | **HOÀN THÀNH + verify E2E ✅** (fallback Piper xác nhận đúng; nhánh Google chưa test vì chưa có credential) |
| CR-006 | Chapters, thumbnail, hook, metadata SEO | P2 | **Code Generation ✅** | **HOÀN THÀNH + verify E2E ✅** |
| CR-007 | Clip dọc 9:16 cho Shorts/TikTok | P1 | Requirements Analysis | **HOÃN theo yêu cầu Creator (2026-09-08)** — làm sau, sau khi các CR còn lại đã ổn định |
| CR-008 | Ngôn ngữ nội dung áp dụng toàn pipeline | P1 | **Code Generation ✅** | **HOÀN THÀNH + verify E2E ✅** |
| CR-009 | Azure Neural TTS song song Google (Google mất free tier) | P1 | Requirements Analysis | **SUPERSEDED (2026-09-09)** — không triển khai; cả Google lẫn Azure tắc ở tầng tài khoản |
| CR-010 | Edge TTS thay Piper làm engine giọng đọc nền | P1 | **Code Generation ✅** | **HOÀN THÀNH** — 49/49 unit test pass; burst 8 scene 8/8 sau khi thêm retry (trước: 1/8). ADR-0024 |
| CR-011 | Azure AI Speech làm engine thứ ba, song song Edge | P1 | **Code Generation ✅** | **HOÀN THÀNH** — 78/78 unit test pass. ADR-0025. Chưa verify với key Azure thật — việc tồn đọng ghi ở mục "Việc tồn đọng" trong `cr-011-azure-tts-engine.md` |

Plan thực hiện: `aidlc-docs/construction/plans/cr-002-007-execution-plan.md`

## Current Status
*Cập nhật 2026-09-09. Trước đó mục này còn dừng ở thời điểm kết thúc Pha 1 và ghi "chờ quyết định làm Pha 2" trong khi Pha 2 đã xong từ 08/09.*

- **Lifecycle Phase**: POST-CONSTRUCTION — 10/10 unit đã build và chạy; công việc đi theo từng Change Request.
- **Đã giao và verify E2E trên stack thật**: CR-001 → CR-006, CR-008. Lệch tiếng/hình 0.003s (trước 61.64s); 1080p60 + faststart; chapters/thumbnail/metadata SEO; ngôn ngữ nội dung thông suốt cả pipeline.
- **Đã giao, chưa verify E2E**: CR-010 (Edge TTS thay Piper — đã đo burst 8/8 scene nhưng chưa render project đầy đủ), CR-011 (Azure engine thứ ba — chưa có key thật; việc tồn đọng liệt kê trong `cr-011-azure-tts-engine.md`).
- **Backlog**: CR-007 (clip dọc 9:16) — hoãn theo quyết định của Creator ngày 2026-09-08.
- **Next Stage**: chờ Creator chọn — (a) CR-007, (b) verify Azure khi có key, hoặc (c) Change Request mới.

## Nợ kỹ thuật đã biết
| Mục | Ghi nhận | Trạng thái |
|---|---|---|
| Không còn engine TTS chạy offline sau khi xoá Piper | ADR-0024 | Chấp nhận có ý thức; Azure (CR-011) vá phần fallback nhưng vẫn cần mạng |
| Dùng edge-tts cho mục đích thương mại vi phạm ToS Microsoft, không SLA | ADR-0024 | Chưa xử lý — Azure là đường thoát khi Creator sẵn sàng trả phí |
| Key Azure sai (có nhưng không hợp lệ) vẫn suy giảm âm thầm về Edge | Review 2026-09-09 | Đã note trong `cr-011-azure-tts-engine.md`, làm khi có key thật |
| Nhánh Google TTS chưa từng chạy thật | CR-005 / ADR-0023 | Ngủ đông — Google đã bỏ free tier (CR-010) |
