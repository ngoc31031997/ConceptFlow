# Business Rules — Unit 2: Content Plugin Service

## Rule 1: Category Source of Truth
`category_hint` do Creator chọn qua GUI (Story B2) là nguồn quyết định category duy nhất — hệ thống KHÔNG tự suy luận category từ nội dung text. `ClassifySceneUseCase` chỉ **validate** hint hợp lệ (nằm trong `supported_categories` của plugin), không "sửa" hay "gợi ý lại" category.

## Rule 2: Category → Animation Template Mapping (Programming Plugin, MVP)
| Category | animation_template_id |
|---|---|
| `algorithm` | `algorithm_visualization` |
| `concept` | `concept_illustration` |

Mapping tĩnh, định nghĩa trong `ProgrammingPlugin`. Rendering Service (Unit 5) chịu trách nhiệm chọn animation cụ thể hơn dựa trên nội dung scene thực tế — Content Plugin Service chỉ cung cấp "gợi ý cấp cao" (category + template id chung).

## Rule 3: Validation Failure Handling
- `category_hint` rỗng hoặc không nằm trong `supported_categories` của plugin → raise `InvalidCategoryError`
- `plugin_id` không tồn tại trong registry → raise `PluginNotFoundError`
- Cả 2 lỗi đều dẫn đến publish event `classification_failed` (không retry — lỗi cấu hình/logic, retry không giúp ích); message vẫn được ack để không bị đưa vào DLQ do "hết retry" (phân biệt lỗi logic vs lỗi tạm thời hạ tầng — chỉ lỗi hạ tầng mới nên đi qua cơ chế retry+DLQ của Unit 1)

## Rule 4: Scene Validation
- `narration_text` không được rỗng — scene không có lời thoại là dữ liệu không hợp lệ, raise `InvalidSceneError` trước khi classify
- `code_snippet` optional — chỉ có ở scene minh họa code walkthrough (Story B3); không bắt buộc cho scene thuật toán/khái niệm thuần túy

## Rule 5: Idempotency (liên kết Low-Level Design)
Mỗi `message_id` chỉ được xử lý (classify + publish event) đúng 1 lần — nếu nhận lại message trùng `message_id` (do RabbitMQ redeliver), consumer bỏ qua xử lý, chỉ ack lại (không classify lại, không publish lại event).
