# ADR-0029: Tách authoring-service khỏi orchestrator

## Status
Accepted

## Date
2026-09-26

## Stage
Requirements Analysis → Construction (CR-040 FR111)

## Context
Orchestrator (~21k dòng Go) ôm hai việc khác bản chất: điều phối saga và project, và toàn bộ phần soạn nội dung — thư viện prompt, chuỗi 1a/1b/1c gọi LLM, gợi ý metadata/short script, log chi phí LLM. Hai việc thay đổi vì lý do khác nhau, và phần soạn nội dung không cần biết gì về saga.

## Decision
Tách `authoring-service` (Go, cùng stack hexagonal, code chuyển nguyên trạng), **có database riêng** (ADR-0013).

**authoring-service sở hữu:** bảng `prompts`, `project_authoring` (topic, story/storyboard/code, mode, model mỗi bước, language), `llm_usage`; các route `/v1/prompts`, `/v1/admin/prompts`, `/v1/projects/{id}/authoring/**`, `/v1/projects/{id}/prompts/{role}`, `suggest-metadata`, `short-script-suggestions`, `/v1/llm/status`.

**orchestrator giữ:** project, saga, wizard settings, video formats, voice calibration, `project_errors`, `project_events`, fork, xoá project.

**Hai service gọi nhau qua HTTP nội bộ** (không qua gateway), mỗi phía tự suy giảm khi phía kia chết:
- authoring → orchestrator: `GET /internal/v1/projects/{id}` (cả Project), `/status`, `/internal/v1/formats/{id}?version=`, `/internal/v1/voices/{id}/calibration`; `POST .../errors`, `.../events` (nhật ký project).
- orchestrator → authoring: `GET|PUT|DELETE /internal/v1/authoring/{id}`, `GET /similar` (trùng topic), `GET /summaries?ids=` (danh sách project). Trạng thái của project trùng topic do orchestrator điền, vì authoring không biết.

Gateway định tuyến các route authoring sang authoring-service; đường dẫn và payload công khai không đổi.

## Consequences
- Orchestrator giảm quy mô và không còn phụ thuộc client LLM.
- `domain.Project` được nhân bản trong authoring-service (JSON trung chuyển qua endpoint nội bộ). Hai bản có thể lệch nhau; authoring chỉ đọc nhóm trường nhỏ (ngôn ngữ, giọng, format, script, engine, chapter...). Nếu sửa những trường đó ở orchestrator thì phải sửa cả bên authoring.
- Danh sách project gọi thêm một request sang authoring-service (best effort: lỗi thì danh sách vẫn trả, chỉ thiếu topic).
- Dữ liệu cũ phải chuyển sang DB mới: `scripts/migrate-authoring-data.sh`. Bảng cũ trong DB orchestrator (`project_authoring`, `llm_usage`, `prompts`) được giữ nguyên, không ai đọc/ghi nữa; xoá tay sau khi kiểm tra.
- Bổ sung một điểm hỏng. Nút Copy prompt vẫn cần authoring-service (render prompt phía server).
