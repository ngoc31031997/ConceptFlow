# CR-113 · Thư viện prompt và model mặc định chuyển về llm-service

## Bối cảnh
Bước 8 trở đi chuyển sang service mới `visual-service`. Các service đều gọi AI và đều cần prompt, nên Creator muốn **một nơi duy nhất quản lý prompt**. llm-service là cổng AI chung: mọi service đã gọi `/v1/chat` và `/v1/chat/external`, và llm-service đã giữ `llm_calls` kèm prompt đã gửi (ADR-0045, ADR-0046). Đặt prompt ở orchestrator đã bị loại, vì orchestrator chỉ điều phối flow và saga.

CR-112 (bước 8, `visual-service`) cần CR này xong trước.

## Hiện trạng (authoring-service)
- Bảng `prompts` (`internal/adapters/postgres/db.go`): `id`, `role`, `name`, `template_text`, `is_system`, `is_active`, `created_at`, `updated_at`. Hai index: mỗi role có một bản đang dùng và một bản hệ thống.
- Kho đọc ghi: `internal/adapters/postgres/prompt_repository.go`.
- Seed bản hệ thống: `internal/domain/prompt_seed_*.go` và `prompt_template_seeds.go`. Seeder upsert lúc khởi động và xoá bản hệ thống của các role đã bỏ (`PurgeLegacyPrompts`).
- Route (`internal/adapters/http/router.go`):
  - `GET /v1/prompts/{role}`
  - `GET`, `POST /v1/admin/prompts`
  - `POST /v1/admin/prompts/{id}/copy`
  - `PUT /v1/admin/prompts/{id}`
  - `POST /v1/admin/prompts/{id}/activate`
  - `DELETE /v1/admin/prompts/{id}`
  - `GET /v1/projects/{id}/prompts/{role}`: prompt đã điền biến.
- Model mặc định theo bước: bảng `app_settings` (key/value), đọc ghi qua `GetStepDefaultModels` / `SaveStepDefaultModel` (`app_settings_repository.go`, `application/default_model.go`).
- Model riêng từng video: các cột `*_model` trong `project_authoring`. **Không chuyển**, vì chúng đi theo dữ liệu của video.
- web-gui: `PromptSettingsPage.tsx`, `AiModelSettingsPage.tsx`.
- llm-service: Python, FastAPI, `asyncpg`, DB qua `LLM_DATABASE_URL`, đã có bảng `llm_calls` (`app/calllog.py`).

## Yêu cầu
1. llm-service sở hữu bảng `prompts`, cùng cột và cùng hai index như hiện nay, và bảng model mặc định theo bước.
2. llm-service có API cho thư viện prompt, giữ nguyên hình dạng request và response của các route `/v1/admin/prompts…` và `GET /v1/prompts/{role}` hiện tại. Khi đó trang Cài đặt › Prompt AI chỉ đổi nơi gọi.
3. API nội bộ cho các service:
   - `GET /internal/v1/prompts/{role}`: bản đang dùng.
   - `PUT /internal/v1/prompts/system`: upsert danh sách bản hệ thống của một service. Body gồm `owner` (tên service) và `prompts[{role, name, template_text}]`. Bản hệ thống thuộc `owner` mà không còn trong danh sách thì bị xoá; bản của Creator giữ nguyên. Đây chính là `PurgeLegacyPrompts` hiện tại.
   - `GET`, `PUT /internal/v1/step-models`: model mặc định theo bước.
4. **Seed vẫn nằm cạnh code dùng nó.** Lúc khởi động, authoring đẩy bản hệ thống của mình lên llm-service qua yêu cầu 3, thay cho việc ghi DB của mình. Thêm cột `owner` vào `prompts` để biết bản hệ thống nào thuộc service nào.
5. **Ghép prompt vẫn ở từng service.** Use case của authoring lấy template từ llm-service, điền biến, rồi gọi `Chat` như cũ (ADR-0047 không đổi). `GET /v1/projects/{id}/prompts/{role}` ở lại authoring vì nó điền biến bằng dữ liệu của authoring.
6. **Chuyển dữ liệu:** các bản Creator đã viết (`is_system = false`), đang dùng hay không, và model mặc định đang lưu phải sang llm-service đầy đủ, giữ nguyên `id`, `is_active`, thời gian. Không mất bản nào.
7. Sau khi chuyển, authoring không còn đọc ghi bảng `prompts` và các key model mặc định trong `app_settings`.
8. api-gateway: các route prompt và model mặc định trỏ sang llm-service.
9. Hợp đồng: thêm `docs/contracts/prompt-library-v1.md` và một dòng trong README hợp đồng. Thêm ADR-0049 (ADR-0048 đã dành cho CR-112) cho quyết định "llm-service sở hữu thư viện prompt".

## Việc agent phải hỏi Creator trước khi làm
1. **Cách chuyển dữ liệu:**
   - (a) Script một lần chép từ authoring-db sang llm-db, chạy tay rồi kiểm.
   - (b) llm-service tự chép lúc khởi động nếu bảng của nó còn trống.

   Đề xuất (a), vì dễ kiểm và không để đường kết nối chéo DB nằm lại trong code.
2. **Bảng cũ ở authoring:** xoá ngay trong CR này, hay giữ một thời gian rồi xoá ở CR sau?
3. **Khi llm-service không chạy:** authoring không lấy được prompt thì bước AI báo lỗi rõ. Không được rơi về một prompt cứng trong code, vì CLAUDE.md cấm đường giả thành công.
4. **Mốc thời gian:** ADR-0045 ("llm-service owns call log") có cần ghi bổ sung, hay chỉ cần ADR mới?

## Tasks
1. Đọc graphify: `graphify affected "PromptRepository"`, `graphify affected "GetStepDefaultModels"`, các nơi gọi `Active(`. Liệt kê mọi use case đọc prompt.
2. llm-service:
   - schema `prompts` (thêm `owner`) và `step_default_models`;
   - kho đọc ghi với asyncpg;
   - route công khai và nội bộ như yêu cầu 2 và 3;
   - test pytest, không gọi AI.
3. authoring-service:
   - adapter HTTP `PromptLibraryPort` gọi llm-service, thay cho `prompt_repository` và phần model mặc định của `app_settings_repository`;
   - seeder đổi sang `PUT /internal/v1/prompts/system`;
   - nối dây trong `cmd`;
   - test Go với fake port.
4. Script chuyển dữ liệu (nếu Creator chọn (a)), kèm bước kiểm: số bản trước và sau, so `id`, so bản đang dùng của từng role.
5. api-gateway: trỏ route sang llm-service; test.
6. web-gui: `PromptSettingsPage` và `AiModelSettingsPage` gọi đường mới (nếu đổi base URL); vitest.
7. docker compose: authoring cần `LLM_SERVICE_URL` (kiểm xem đã có chưa).
8. Hợp đồng, ADR, README hợp đồng.
9. Rebuild `llm-service`, `authoring-service`, `api-gateway` và `web-gui` (nếu đổi) bằng `scripts/worktree.sh rebuild …`, rồi kiểm healthy.
10. Kiểm tay trên stack:
    - Trang Prompt AI vẫn liệt kê đủ prompt; bản Creator viết còn nguyên.
    - Một bước AI của bước 1–7 chạy ở chế độ "Dùng AI ngoài · prompt" ra đúng prompt như trước. So với một prompt lưu trước khi chuyển.
    - Dọn mọi dữ liệu test đã tạo.

## Kiểm thử
pytest cho llm-service (kho, route, upsert bản hệ thống của một `owner`, không đụng bản của Creator và của `owner` khác). Go test cho authoring với fake port (seeder, use case lấy prompt, lỗi khi llm-service không trả lời). Vitest cho hai trang Cài đặt nếu đổi. Không gọi Hive hay Ollama thật.

## Ngoài phạm vi
- Model riêng từng video (`project_authoring.*_model`).
- Prompt của visual-service: CR-112 sẽ dùng API của CR này.
- Đổi nội dung prompt nào.
