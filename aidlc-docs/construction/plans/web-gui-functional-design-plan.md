# Functional Design Plan — Unit 10: Web GUI

## Unit Context
- **Scope**: Toàn bộ UI Creator (Epic A-F), state soạn project, validation form, luồng tương tác
- **KHÔNG chịu trách nhiệm**: business logic nghiệp vụ (đó là backend); GUI chỉ validate ở mức UX (chặn submit sai) — backend vẫn là nguồn xác thực cuối cùng (zero-trust, đã xác nhận ở các unit backend)

## Execution Checklist
- [x] Thu thập câu trả lời
- [x] Tạo `business-logic-model.md`
- [x] Tạo `business-rules.md`
- [x] Tạo `domain-entities.md`
- [x] Tạo `frontend-components.md`
- [ ] Trình bày để phê duyệt

---

## Clarifying Questions

### Question 1: Form Validation Rules — New Project (Story A1, B1, B4)
A) 💡 Suggested: `script_content` bắt buộc (không rỗng) trước khi enable nút "Bắt đầu render"; `plugin_id` bắt buộc chọn (Story B1's AC: "hệ thống yêu cầu tôi chọn plugin trước khi tiếp tục"); `voice_language` mặc định `"vi"` (không bắt buộc chọn lại, nhưng hiển thị rõ để đổi); `background_music_path` optional, không validate gì thêm ở GUI (backend validate định dạng file nếu cần — ngoài phạm vi GUI)
   - ✅ Strengths: khớp đúng acceptance criteria của story, UX rõ ràng (nút disable khi thiếu field bắt buộc thay vì submit rồi báo lỗi)
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 2: Form Validation Rules — Publish Metadata (Story E2)
A) 💡 Suggested: `youtube_title` bắt buộc (Story E2's AC: "hệ thống ngăn việc đăng tải và thông báo tiêu đề là bắt buộc"), tối đa 100 ký tự (giới hạn thực tế của YouTube API, hiển thị đếm ký tự). `description`/`tags` optional. `visibility` mặc định `"private"` (an toàn nhất, Creator có thể đổi) — bắt buộc phải chọn 1 trong 3 giá trị (radio/select, không có "chưa chọn")
   - ✅ Strengths: khớp acceptance criteria, mặc định an toàn (private) tránh đăng nhầm public
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 3: State Transition Logic trong GUI — Điều hướng theo Project.Status
A) 💡 Suggested: GUI tự điều hướng dựa trên `Project.Status` nhận được (qua SSE hoặc `GET`): các trạng thái "đang xử lý" (`parsing_script`...`assembling_video`) → ở lại `RenderPage`; `ready_to_publish` → tự động chuyển `ResultPage`; `failed_at_<step>` → ở lại `RenderPage`, hiển thị `ErrorBanner`; `publishing` → hiển thị trạng thái "Đang đăng..." trên `ResultPage`; `published` → hiển thị link video YouTube đã đăng
   - ✅ Strengths: điều hướng tự động theo đúng state machine đã thiết kế ở Orchestrator (Unit 8), không cần Creator tự bấm "Next"
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 4: Domain Entities phía GUI — ProjectDraft (state trước khi submit)
A) 💡 Suggested: `ProjectDraft` (local state, chưa có `project_id` cho tới khi submit thành công): `{ scriptContent: string, pluginId: string | null, voiceLanguage: "vi" | "en", backgroundMusicPath: string | null }`. Reducer actions: `SET_SCRIPT`, `SET_PLUGIN`, `SET_VOICE_LANGUAGE`, `SET_BACKGROUND_MUSIC`, `RESET`
   - ✅ Strengths: khớp `interface-contracts.md`'s `RenderInput`, đủ tối giản cho state cần thiết
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 5: Frontend Components — Props/State chi tiết (BẮT BUỘC cho unit có UI)
A) 💡 Suggested: Chi tiết hóa đầy đủ trong `frontend-components.md` theo cấu trúc đã có ở `module-structure.md` — mỗi component liệt kê props (typed), local state (nếu có), sự kiện người dùng xử lý, và API/context nào nó gọi. KHÔNG cần Storybook hay design system riêng — component styling tối giản (CSS module hoặc plain CSS, không cần thư viện UI ngoài như MUI/Chakra vì scope nhỏ, 1 luồng tuyến tính)
   - ✅ Strengths: đủ chi tiết cho Code Generation, không thêm dependency UI library không cần thiết ở quy mô này
   - ⚠️ Trade-offs: styling tối giản, không có polish thẩm mỹ cao — chấp nhận được cho MVP cá nhân

B) Other (please describe after [Answer]: tag below)

[Answer]: A

### Question 6: Data-testid Convention (Automation Friendly Code Rules, MANDATORY)
A) 💡 Suggested: `{component}-{element-role}` (vd. `new-project-script-textarea`, `new-project-plugin-select`, `new-project-submit-button`, `publish-form-title-input`, `publish-form-submit-button`, `error-banner-retry-button`) — áp dụng cho mọi input/button/link tương tác
   - ✅ Strengths: nhất quán convention chuẩn đã định nghĩa ở `code-generation.md`
   - ⚠️ Trade-offs: không có

B) Other (please describe after [Answer]: tag below)

[Answer]: A
