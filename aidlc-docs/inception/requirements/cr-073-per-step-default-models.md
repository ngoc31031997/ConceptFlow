# CR-073 — Model AI mặc định theo từng bước

## Date
2026-10-20

## Stage
Requirements + thiết kế — **chờ Creator duyệt**. Chưa viết code.

## Yêu cầu gốc (nguyên văn)
> "ở menu này Model AI mặc định chúng ta được phép cấu hình model default cho từng bước. ví dụ sinh brief default ollama local ai sinh câu hỏi dùng deepseek v.v"

Trả lời của Creator:
> "model cấu hình này ko liên quan theo dự án. nó được cấu hình chung để suggest default khi làm dự án"
> Phạm vi bước: "Mọi bước có gọi AI"

## Hiện trạng (đã đọc code)
- Menu Cài đặt · Model AI mặc định ([AiModelSettingsPage.tsx](/Users/hoangbaminhngoc/Documents/Project/ConceptFlow/services/web-gui/src/pages/AiModelSettingsPage.tsx)) lưu **một** model chung: `app_settings` key `default_model`, qua `DefaultModelUseCase` và `GET/PUT /v1/settings/ai-model`.
- Model của từng bước trong dự án là `domain.AuthoringStepModels` (brief, research, story_frame, story, storyboard, code, publish). Rỗng nghĩa là chưa chọn.
- Server chạy lời gọi AI theo thứ tự: model của bước trong dự án → mặc định chung (`Client.modelOrDefault`) → mặc định của llm-service. `GetAuthoringModels` được 5 use case gọi riêng (brief, research, story_desk, generate_authoring, prompt_templates).
- Bước code và minh hoạ từ chối Ollama (`ModelAllowedForStep`). Minh hoạ trong thư viện không thuộc dự án, chọn model bằng `AiModelField`.
- Dropdown trong dự án (`AiModelSelect`) hiện điền sẵn mặc định chung và gắn badge "Mặc định".

Yêu cầu bổ sung của Creator (2026-10-21):
> "bỏ cái Model chung này, mỗi bước dc cấu hình ở đây mặc định là cái đầu tiên (đưa local AI ollama lên đầu)"

## Yêu cầu
1. Trang Cài đặt có một dropdown cho từng bước có gọi AI: brief, câu hỏi nghiên cứu (research), khung câu chuyện (story_frame), kịch bản (story), storyboard, code, đề xuất publish, vẽ minh hoạ (thư viện hình). Không còn "Model chung".
2. Danh mục model đặt Ollama (local AI) đầu tiên. Bước chưa được cấu hình mặc định là model đầu danh mục, tức Ollama.
3. Cấu hình này là **chung toàn hệ thống, không gắn dự án**. Dự án được chọn sẵn model theo bảng; người dùng vẫn đổi được tại từng bước.
4. Thứ tự chọn model khi chạy: lựa chọn ở dự án/nút → mặc định của bước (đã lưu, hoặc model đầu danh mục). Không còn mặc định chung.
5. Mọi model trong danh mục chọn được cho mọi bước, kể cả Ollama ở code và minh hoạ. Ràng buộc cứng bị bỏ; thay bằng cảnh báo trên giao diện rằng model chạy trên máy có thể hết giờ ở bước code (Creator chọn 2026-10-21).
6. Bước hoặc model không có trong danh sách bị từ chối khi lưu; model rỗng cũng bị từ chối (không có "xoá về mặc định").

## Thiết kế
- **Lưu trữ**: tái dùng bảng `app_settings`, mỗi bước một khóa `default_model.<step>`. Không cần migration. Khóa `default_model` cũ (model chung) không còn được đọc; hàng còn lại trong DB vô hại.
- **domain**: `ConfigurableModelSteps` (7 bước của `AuthoringStepModels` và `illustrations`), `WithStepModelFallback` điền model đầu danh mục cho bước chưa lưu. Xoá `ModelAllowedForStep`, cờ `CodeOK`; thêm cờ `Local` cho Ollama.
- **application**: `DefaultModelUseCase` còn `StepModels` và `SetStep`; bỏ `Get`/`Set` model chung và `checkCodeModel`. Vẽ hình trong thư viện không đặt model thì chạy theo mặc định của bước `illustrations`.
- **postgres**: `GetStepDefaultModels` luôn trả đủ mọi bước. `GetAuthoringModels` điền mặc định của bước vào chỗ dự án chưa chọn (cho mọi nơi chạy AI); `GetChosenAuthoringModels` chỉ trả phần dự án đã chọn (cho giao diện, nên mặc định không bị lưu cứng vào dự án).
- **llm client**: bỏ `WithDefaultModel`/`modelOrDefault`; lời gọi không nêu model thì llm-service tự dùng `HIVE_MODEL`.
- **API**: `GET /v1/settings/ai-model` trả `step_models` và `steps`; `PUT /v1/settings/ai-model/steps/{step}`; `GET /v1/llm/status` trả `step_defaults`, bỏ `default_model`.
- **web-gui**: trang Cài đặt là lưới dropdown theo bước (`wide`); `AiModelSelect`, `AiModelField`, `AuthoringModelPicker` chọn sẵn mặc định của bước và gắn badge "Mặc định"; cảnh báo khi chọn Ollama ở code/minh hoạ.
- **Hành vi cần biết**: mặc định được áp **lúc chạy**. Đổi bảng sẽ đổi model chạy của các dự án chưa tự chọn ở bước đó; vì mặc định ban đầu là Ollama, mọi bước chưa cấu hình giờ chạy bằng Ollama.


## Phụ thuộc
- Độc lập với `fix/research-checking-loading` và `fix/step-rail-gating` (hai nhánh này sửa màn hình dự án, không đụng trang Cài đặt hay `default_model.go`).
- Các nhánh `cr-072-*` còn dở: chưa so file từng nhánh. Cần kiểm tra `AuthoringStepModels` và các pickers trước khi code.

## Kiểm thử dự kiến
Test đơn vị giả AI, không gọi Hive hay Ollama: mặc định mọi bước là Ollama, lưu và đọc lại từng bước, từ chối bước/model lạ hoặc rỗng, dự án chọn riêng thắng mặc định; test Postgres thật cho phần điền mặc định; test vitest cho trang Cài đặt và các ô chọn model. Test DB dùng một Postgres tạm và xoá sau khi chạy.

## Triển khai
Dựng lại `authoring-service` và `web-gui` bằng `scripts/worktree.sh rebuild`.
