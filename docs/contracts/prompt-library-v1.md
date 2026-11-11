# Hợp đồng `prompt-library.v1` · thư viện prompt, danh mục model và model mặc định ở llm-service

Quyết định: [ADR-0048](../../aidlc-docs/decisions/ADR-0048-llm-service-owns-prompt-library.md). Yêu cầu: `aidlc-docs/inception/requirements/cr-113-prompt-library-llm-service.md`. Route thật ở `services/llm-service/app/main.py`, quy tắc ở `app/library.py`, danh mục model ở `app/modelcatalog.py`.

llm-service giữ bảng `prompts` và `step_default_models` trong `llm-db`. Prompt hệ thống nằm cạnh code dùng nó và được service đó đẩy lên lúc khởi động. Điền biến vào template vẫn do từng service làm, vì dữ liệu để điền là của service đó.

## Route công khai (web-gui → api-gateway → llm-service)

Hình dạng giữ nguyên như khi còn ở authoring-service. Lỗi trả `{"error": "<thông báo>"}`.

| Route | Ý nghĩa | Lỗi |
|---|---|---|
| `GET /v1/prompts/{role}` | bản role đang chạy: bản đang dùng, không có thì bản hệ thống | 400 role chưa ai gửi bản hệ thống; 404 |
| `GET /v1/admin/prompts[?role=]` | `{"prompts": [...]}`, nhóm theo role, bản hệ thống trước | 400 role lạ |
| `POST /v1/admin/prompts` | thân `{role, name, template_text}`; tạo bản của Creator, **chưa kích hoạt**; 201 | 400 thiếu tên/nội dung, role lạ |
| `POST /v1/admin/prompts/{id}/copy` | nhân bản (kể cả bản hệ thống) thành bản của Creator; 201 | 404 |
| `PUT /v1/admin/prompts/{id}` | thân `{name, template_text}` | 403 bản hệ thống; 404; 400 |
| `POST /v1/admin/prompts/{id}/activate` | thành bản đang dùng duy nhất của role | 404 |
| `DELETE /v1/admin/prompts/{id}` | 204; xoá bản đang dùng thì role quay về bản hệ thống | 403 bản hệ thống; 404 |
| `GET /v1/settings/ai-model` | `{"step_models": {bước: model}, "steps": [bước…]}`; bước chưa lưu lấy model đầu danh mục | |
| `PUT /v1/settings/ai-model/steps/{step}` | thân `{"model"}`; 204 | 400 bước lạ hoặc model ngoài danh mục |

Một prompt: `{id, role, name, template_text, is_system, is_active, created_at, updated_at}` (thời gian RFC 3339).

Một role "tồn tại" khi có ít nhất một bản hệ thống của nó đã được đẩy lên; llm-service không giữ danh sách role cứng.

## Route nội bộ (service → llm-service)

| Route | Ý nghĩa |
|---|---|
| `GET /internal/v1/prompts/{role}` | giống `GET /v1/prompts/{role}` |
| `PUT /internal/v1/prompts/system` | thân `{owner, prompts: [{role, name, template_text}]}`. Ghi (upsert theo `id = system-{role}`) mọi bản trong danh sách; xoá bản hệ thống của `owner` mà không còn trong danh sách. Bản của Creator và của `owner` khác giữ nguyên. Bản hệ thống chỉ được kích hoạt khi role chưa có bản nào đang dùng. Trả `{owner, written, removed}`; 409 nếu role đang thuộc `owner` khác; 400 nếu thiếu `owner`, trùng role hoặc bản rỗng |
| `GET /internal/v1/step-models` | như `GET /v1/settings/ai-model` |
| `PUT /internal/v1/step-models` | thân `{"models": {bước: model}}`; kiểm như route công khai, hỏng một mục thì không ghi mục nào; 204 |
| `GET /internal/v1/models` | `{"models": [{id, label, local}]}`; model đầu là mặc định của bước chưa lưu |

## Ai gọi gì

- authoring-service gọi `GET /internal/v1/prompts/{role}` mỗi lần một bước AI cần template, rồi điền biến và gọi `/v1/chat`. Khởi động xong nó đẩy bản hệ thống bằng `PUT /internal/v1/prompts/system` với `owner = "authoring"`, thử lại với quãng nghỉ tăng dần đến khi llm-service nhận.
- authoring-service gọi `GET /internal/v1/models` để kiểm model Creator chọn cho từng video và để trả danh mục cho `GET /v1/llm/status`; gọi `GET /internal/v1/step-models` để lấy model mặc định của bước.
- visual-service làm như authoring-service cho bước 8: đẩy 10 prompt với `owner = "visual"` (9 prompt `visual_concept_*` chạy trong hệ thống, và `reference_style_youtube`: prompt để Creator chép sang AI ngoài, đọc link YouTube thành một dòng thư viện tham chiếu, biến `{{url}}`; visual-service chỉ lưu vào thư viện, không bao giờ ghép hay gọi nó, thư viện tham chiếu lấy bằng `GET /v1/prompts/reference_style_youtube`), đọc `GET /internal/v1/prompts/{role}` mỗi lần gọi AI, model mặc định của bước `visual_concept` qua `GET /internal/v1/step-models`, danh mục qua `GET /internal/v1/models`. Model riêng từng video ở `project_visual.concept_model`.
- Các bước có model mặc định (`STEPS` ở `app/modelcatalog.py`): `brief`, `research`, `story_frame`, `beat_sheet`, `story`, `storyboard`, `code`, `publish`, `illustrations`, `visual_concept`.
- Khi llm-service không trả lời, bước AI báo lỗi có tên llm-service. Không có prompt hay danh mục dự phòng trong code.
- Model riêng từng video (`project_authoring.*_model`) vẫn ở authoring-service.

## Dữ liệu

`prompts(id, role, name, template_text, is_system, is_active, created_at, updated_at, owner)`; hai index duy nhất: một bản đang dùng mỗi role, một bản hệ thống mỗi role. `owner` rỗng với bản của Creator. `step_default_models(step, model, updated_at)`.

Chuyển từ authoring-db: `scripts/migrate_prompt_library.py` (`copy`, `verify`, `drop-source`).
