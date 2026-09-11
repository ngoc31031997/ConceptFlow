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
| CR-012 | Nhiều tài khoản YouTube trên nhiều OAuth client | P1 | **Code Generation ✅** | **Chờ Creator verify với Google thật** — 72/72 test publisher, 42/42 gateway, 68/68 web-gui, Go xanh. ADR-0026. Sửa luôn bug ghi đè credential khi nối kênh thứ hai |
| CR-013 | Retry có backoff cho AzureTTSAdapter | P1 | **Code Generation ✅** | **HOÀN THÀNH** — 87/87 test tts (19 cho azure_adapter). Đo thực tế trước khi sửa: 16/20 request 401 rải rác; sau khi sửa burst 8 scene đạt 8/8. Azure đã tự ổn định nên retry chưa bị kích hoạt thật — vẫn giữ làm bảo hiểm |
| CR-014 | Sửa lỗi Gợi ý AI (tags null, title rỗng, prompt tràn context) | P0 bug | **Code Generation ✅** | **HOÀN THÀNH** — 3 lỗi chồng nhau; gốc rễ là script 17.5k ký tự vượt num_ctx=2048 của Ollama. Verify: project luôn lỗi nay trả 200 |
| CR-015 | Caption track YouTube + burn-in theo format | P1 | **Code Generation** | Thiết kế đã duyệt (2026-09-09) — ADR-0027, ADR-0028. Ở lại Testing, chấp nhận nối lại kênh hàng tuần; verification hoãn sang CR riêng |
| CR-016 | Ước lượng thời lượng lúc soạn, hiệu chỉnh WPM theo giọng | P1 | **Code Generation ✅** | **HOÀN THÀNH** (2026-09-10) |
| CR-017 | Design system `conceptflow` — khoá bản sắc kênh vào code | P1 | **Code Generation ✅** | **HOÀN THÀNH** (2026-09-10) |
| CR-018 | `self.narrate()` và render hai lượt thay marker comment | P1 | **Code Generation ✅** | **HOÀN THÀNH** (2026-09-10) |
| CR-019 | Beat sheet là hợp đồng; hook/recap/CTA thành method | P1 | **Code Generation ✅** | **HOÀN THÀNH** (2026-09-10) |
| CR-020 | Cổng `validate_script` trước TTS, gỡ hẳn content-plugin | P1 | **Code Generation ✅** | **HOÀN THÀNH** (2026-09-10) — điều chỉnh: `validate_script` sống trong `rendering`, không dựng quality-service |
| CR-021 | Chấm chất lượng video tự động và cổng publish | P2 | **Code Generation ✅** | **HOÀN THÀNH** (2026-09-11) — chạy chế độ chỉ-báo (`QC_ENFORCE=false`). Điều chỉnh: `qc_video` sống trong `video-assembly`, không dựng quality-service. **Việc tồn đọng: hiệu chỉnh ngưỡng trên video thật trước khi bật cổng chặn** |
| CR-022 | Vòng phản hồi retention/analytics | P2 | Requirements Analysis | **HOÃN** — ngoài phạm vi đợt CR-016..024 |
| CR-023 | Intro/outro cố định làm bản sắc kênh | P1 | **Code Generation ✅** | **HOÀN THÀNH** (2026-09-11) — **việc tồn đọng: Creator cung cấp file sting intro dựng ngoài; pipeline đang chạy bằng bản Manim mặc định** |
| CR-024 | Cổng duyệt dàn ý trước khi tốn TTS | P1 | **Code Generation ✅** | **HOÀN THÀNH** (2026-09-10) |

Plan thực hiện: `aidlc-docs/construction/plans/cr-002-007-execution-plan.md`,
`cr-016-024-execution-plan.md`, `cr-023-low-level-design.md`,
`cr-021-low-level-design.md`

## Current Status
*Cập nhật 2026-09-11 — đợt CR-016..024 đã đóng.*

- **Lifecycle Phase**: POST-CONSTRUCTION — 10/10 unit đã build và chạy; công việc đi theo từng Change Request.
- **Saga hiện tại** (sau CR-020/021/023/024):
  `parse_script -> validate_script -> [CHỜ DUYỆT] -> synthesize_speech -> render_scenes -> assemble_video -> qc_video -> publish_video`
- **Đã giao và verify E2E trên stack thật**: CR-001 → CR-006, CR-008. Lệch tiếng/hình 0.003s (trước 61.64s); 1080p60 + faststart; chapters/thumbnail/metadata SEO; ngôn ngữ nội dung thông suốt cả pipeline.
- **Đã giao, chưa verify E2E**: CR-010, CR-011 (chưa có key Azure thật), và **toàn bộ đợt CR-016..024** — tất cả test unit xanh (orchestrator 6/6 gói, rendering 106, video-assembly 134, api-gateway 53, web-gui 128) nhưng chưa render một project đầy đủ trên stack thật sau đợt này.
- **Backlog**: CR-007 (clip dọc 9:16, hoãn 2026-09-08), CR-022 (vòng phản hồi retention, hoãn ngoài phạm vi đợt).
- **Next Stage**: chờ Creator chọn — (a) verify E2E đợt CR-016..024 trên stack thật, (b) hiệu chỉnh ngưỡng QC rồi bật `QC_ENFORCE` (CR-021), (c) CR-007, (d) verify Azure khi có key, hoặc (e) Change Request mới.

## Việc tồn đọng cần Creator làm (không phải việc code)
| Việc | Nguồn | Vì sao không làm bằng code được |
|---|---|---|
| Cung cấp file sting intro dựng ngoài (Runway/Kling/Blender) | CR-023 §Quyết định #2 | Logo là ảnh 3D photorealistic; Manim chỉ fade/scale được nó. Pipeline đang chạy bằng bản Manim mặc định |
| Chọn và upload nhạc hiệu intro/outro | CR-023 Quyết định #4 | Chưa chọn nguồn; khe nhận file và chuẩn hoá -14 LUFS đã sẵn |
| Hiệu chỉnh ngưỡng QC trên video thật rồi bật `QC_ENFORCE=true` | CR-021 Quyết định #3 | Ngưỡng hiện là ước lượng nới rộng. Bật cổng trước khi biết tỉ lệ báo động giả là cách làm Creator mất niềm tin vào báo cáo |

## Nợ kỹ thuật đã biết
| Mục | Ghi nhận | Trạng thái |
|---|---|---|
| Không còn engine TTS chạy offline sau khi xoá Piper | ADR-0024 | Chấp nhận có ý thức; Azure (CR-011) vá phần fallback nhưng vẫn cần mạng |
| Dùng edge-tts cho mục đích thương mại vi phạm ToS Microsoft, không SLA | ADR-0024 | Chưa xử lý — Azure là đường thoát khi Creator sẵn sàng trả phí |
| Key Azure sai (có nhưng không hợp lệ) vẫn suy giảm âm thầm về Edge | Review 2026-09-09 | Đã note trong `cr-011-azure-tts-engine.md`, làm khi có key thật |
| Nhánh Google TTS chưa từng chạy thật | CR-005 / ADR-0023 | Ngủ đông — Google đã bỏ free tier (CR-010) |
| OAuth app External + Testing ⇒ refresh token hết hạn sau 7 ngày, phải nối lại mọi kênh hàng tuần | CR-015 / ADR-0028 | **Chấp nhận tạm thời (Creator, 2026-09-09)** — lên production đòi Google verification vì `force-ssl` là sensitive scope; hoãn tới một CR riêng để chỉ nộp hồ sơ đúng một lần với bộ scope cuối |
| Scope `force-ssl` rộng hơn mức cần (cấp cả quyền xoá video), nhưng YouTube không có scope hẹp hơn cho `captions.insert` | ADR-0028 / ADR-0016 | Chấp nhận có ý thức; giảm nhẹ bởi mô hình đe doạ local-only |
| Ghép intro/outro làm mất đường stream-copy, phải re-encode toàn bộ | CR-023 §Rủi ro | Chấp nhận có ý thức; chưa đo con số thật sau khi ghép |
| Ngưỡng QC chưa hiệu chỉnh bằng số đo — đang chạy chỉ-báo | CR-021 Quyết định #3 | Có chủ ý, không phải nợ ngầm: siết bằng biến `QC_*` sau khi đo trên video thật |
| Upload video intro mới sẽ xoá nhạc hiệu đang có (phải upload nhạc lại) | CR-023 FR65.4 | Chấp nhận tạm: luồng thực tế là upload clip trước, nhạc sau |
| `quality-service` không tồn tại — `validate_script` ở `rendering`, `qc_video` ở `video-assembly` | CR-020, CR-021 D1 | Có chủ ý: dựng image thứ hai chỉ để chạy một lệnh là cái giá không đáng. Tách ra khi QC mọc thêm chấm frame bằng VLM |
