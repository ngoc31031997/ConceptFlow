# CR-100 · Nhật ký cuộc gọi AI ở llm-service — yêu cầu và thiết kế

Nhánh `feature/cr-100-llm-call-log`. Quyết định kiến trúc: `aidlc-docs/decisions/ADR-0045-llm-service-owns-call-log.md`. Trạng thái: đã code và kiểm thử trên nhánh; chờ Creator duyệt. Hợp đồng: `docs/contracts/llm-chat-v1.md`.

## Chốt của Creator (2026-10-25)

1. llm-service có DB riêng, mở cổng ra máy (`127.0.0.1`) để vào bằng DBeaver.
2. Log mọi cuộc gọi AI. **Không lưu prompt, chỉ lưu response** (để tra lỗi và, về sau, đọc lại mà không gọi AI).
3. Toàn bộ bước gọi AI đi qua llm-service và nhận về JSON sạch; authoring-service bỏ `jsonObjectIn`, giữ phần kiểm tra nghiệp vụ.
4. Bảng `llm_usage` của authoring-service bị xoá; số liệu chuyển sang bảng của llm-service. Authoring-service không lưu gì liên quan tới LLM nữa. Không chuyển dữ liệu cũ.
5. Thống kê chunk của pipeline code: api-gateway chuyển thẳng cho llm-service, không đi qua authoring-service.
6. Màn hình ở menu **Nhật ký**: tải bảng thô lên giao diện; chọn tất cả, xoá nhiều dòng, hoặc xoá theo cơ chế 30 ngày.
7. Chưa làm: đọc lại response đã lưu để thử lại không tốn token.
8. Bảng `llm_usage` ở orchestrator cũng xoá. Mọi chỗ còn gọi model đều chuyển qua llm-service.
9. Xoá theo 30 ngày là nút bấm tay trong Nhật ký, không có tác vụ nền. Xoá dòng làm mất số liệu thống kê tương ứng: chấp nhận.
10. Bước 8 trở đi (storyboard, minh hoạ, code) sẽ làm lại sau và gọi thẳng llm-service: xoá cụm `GenerateAuthoringUseCase` khỏi authoring-service; các trang đó thành "Sắp có". Giữ `llm-status` (bộ chọn model).
11. Số CR là CR-100.

## Hiện trạng (đối chiếu code)

- Mọi cuộc gọi model đi qua llm-service, hai đường vào: `/v1/chat` (14 chỗ gọi từ authoring-service) và ba endpoint tự gọi model (`/v1/suggest-metadata`, `/v1/storyboard/finalize`, `/v2/code/generate`).
- Điểm duy nhất gọi model trong llm-service là `Provider.chat` (`app/provider.py`); mọi đường trên đều qua nó.
- authoring-service ghi số liệu mỗi cuộc gọi vào `llm_usage` qua `LLMUsageRecorder`. Bảng này được đọc ở một chỗ: `GET .../llm-status` trả `code_stats` (trung bình token và thời gian của chunk, theo model, trong 30 ngày) cho `PipelineSettingsBar`/`AuthoringModelPicker`.
- Reply không được lưu ở đâu; lỗi parse chỉ trả lý do ngắn. `jsonObjectIn` (`domain/story.go`) cắt từ `{` đầu tới `}` cuối, nên một khối JSON kèm ký tự `}` thừa phía sau gây `invalid character '}' after top-level value`.
- Có chỗ đọc JSON theo cùng kiểu cắt `{`…`}`: `jsonObjectIn` ở `beat.go`, `handoff.go`, `project_story.go`, `script.go` (2), `script_review.go`, `story.go`; cắt tay trong `research.go` (dòng 178) và `creative_brief.go` (dòng 193). Dòng 310 của `research.go` đọc phần dán tay của Creator, giữ nguyên.
- Bảng `llm_usage` cũng còn trong `orchestrator/internal/adapters/postgres/db.go` (xem Câu hỏi mở).

## Phạm vi

### A. llm-service

**DB `llm-db`**: `postgres:16-alpine`, database `llm`, cổng `127.0.0.1:5435:5432`, volume `llm_db_data`, cùng `POSTGRES_USER`/`POSTGRES_PASS` với các DB khác, healthcheck như `orchestrator-db`. llm-service thêm phụ thuộc `asyncpg`.

**Bảng `llm_calls`**, mỗi lần `Provider.chat` kết thúc (kể cả lần thử lại và lần lỗi) ghi một dòng:

| Cột | Kiểu | Ý nghĩa |
|---|---|---|
| `call_id` | uuid, khoá | trả về cho người gọi |
| `created_at` | timestamptz | lúc cuộc gọi kết thúc |
| `provider`, `model` | text | |
| `role` | text | tên bước/vai gọi AI (`story_frame`, `beat_sheet`…); `''` nếu người gọi không gửi |
| `step`, `phase` | text | như `llm_usage` cũ (`code`/`chunk`, `attempt_2`…) |
| `project_id` | text, null, không FK | xoá project không xoá log |
| `response` | text, nén lz4 | nội dung thô model trả về; rỗng khi lỗi và chưa có gì |
| `ok`, `error_kind`, `error_message` | | `error_message` là thông báo lỗi, không phải prompt |
| `prompt_tokens`, `completion_tokens`, `reasoning_tokens`, `cached_tokens`, `reasoning_chars`, `usage_reported` | | như `llm_usage` cũ |
| `duration_ms` | int | |

Chỉ mục `created_at DESC`, `(step, phase, created_at)`, `project_id`.

**Quy tắc ghi:** ghi lỗi (DB sập…) không làm hỏng cuộc gọi AI: ghi vào log của llm-service rồi trả kết quả bình thường. Đây là nhật ký, không phải đường chính. Không lưu prompt (`system`/`user`) ở bất cứ đâu.

**Nhận `project_id`, `role`, `step`, `phase`:** thêm vào `ChatBody` của `/v1/chat` và vào body các endpoint khác; truyền xuống `Provider.chat` qua một `contextvar` để hàm này không đổi chữ ký. Các cuộc gọi trong pipeline code (`/v2/code/generate`) tự đặt `step = "code"` và `phase` theo giai đoạn (`layout`, `chunk`, `repair`…).

**`expect_json` cho `/v1/chat`:**
1. đọc khối JSON **đầu tiên** của reply (`json.JSONDecoder().raw_decode` từ `{` đầu tiên, có hoặc không có hàng rào ```json), bỏ phần thừa phía sau;
2. nếu không đọc được: một lượt gọi model để sửa (như `/v1/storyboard/finalize`), mỗi lượt cũng ghi một dòng `llm_calls`;
3. vẫn hỏng: trả lỗi `kind = "malformed"` kèm `call_id`, và `response` thô đã nằm trong bảng.

Response thêm `call_id` và, khi `expect_json`, trường `json` (object đã đọc) bên cạnh `content`. Không đoán nghĩa của object: llm-service không biết schema từng bước.

**Endpoint thống kê và nhật ký** (llm-service, nội bộ):
- `GET /v1/llm/stats?step=code&phase=chunk&days=30` → `{ "<model>": {model, calls, ok, failures:{kind:n}, avg_completion_tokens, avg_duration_ms} }`, cùng hình dạng `code_stats` hiện nay.
- `GET /v1/llm/calls?limit&before&project_id&role&ok` → các dòng thô (phân trang theo `created_at`), kể cả `response`.
- `POST /v1/llm/calls/delete` thân `{ "ids": [...] }` hoặc `{ "older_than_days": N }` hoặc `{ "all": true }` (đúng một) → số dòng đã xoá. POST thay cho DELETE có thân để đi qua gateway và proxy không lệch. Xoá theo ngày là nút bấm tay, không có tác vụ nền.

### B. api-gateway

- Route mới chuyển thẳng sang llm-service (client HTTP `llmClient`, cấu hình `LLM_SERVICE_URL`): `GET /v1/llm/stats`, `GET /v1/llm/calls`, `POST /v1/llm/calls/delete`.
- Bỏ các route authoring đã xoá: `/authoring/{step}/generate`, `/progress`, `/chain`, `/code/segments…`, `/code/chunk-shots`.

### C. authoring-service

- Bỏ `jsonObjectIn` và các chỗ cắt tay `{`…`}`; các hàm `Parse…Reply` nhận object JSON sạch từ llm-service rồi giữ nguyên kiểm tra nghiệp vụ (`CheckStoryFacts`, `KeepStoryCreatorFields`, `Validate`…). Phần dán tay của Creator ở bước Nghiên cứu (`ParseResearchPaste`) giữ nguyên.
- `ChatRequest` thêm `ProjectID`, `Role`, `Step`, `Phase`, `ExpectJSON`; `ChatResult` thêm `CallID`, `JSON`; `LLMError` thêm `CallID`, và thông báo lỗi nêu "mã cuộc gọi".
- Mười ba chỗ gọi model gửi `Role`, `ProjectID`, `Step` và `ExpectJSON`, riêng người vẽ hình (nhận văn bản có khối ```tsx, không phải JSON) không dùng `ExpectJSON`.
- Xoá hẳn: `LLMUsageRecorder`, `RecordFor`, cổng và kho `llm_usage`, bảng `llm_usage` (`DROP TABLE IF EXISTS`, cả ở orchestrator), trường `code_stats` của `llm-status`.
- Xoá cụm `GenerateAuthoringUseCase`: `generate_authoring*.go`, `authoring_chain.go`, `storyboard_warnings.go`, `code_pipeline.go` (cổng và hợp đồng gọi `/v2/code/*`, `/v1/storyboard/finalize`), các route `/authoring/{step}/generate`, `/progress`, `/chain`, `/code/segments…`, `/code/chunk-shots` và các test của chúng. `GET /v1/llm/status` (danh mục model, model mặc định theo bước) giữ, đọc `llmClient.Ready()`.
- Còn lại, không xoá: bảng và kho `project_code_segment`, `code_check_diagnostics`, `domain.CodeSegment` (dữ liệu của bước code, bước 8 sẽ dùng lại); các route lưu tay `POST /v1/projects/{id}/authoring/{story,storyboard,code}`.

### D. web-gui

- Màn hình **Nhật ký** (`/journal`, `JournalPage`) thêm tab **Cuộc gọi AI**, luôn hiện (kể cả khi chưa có dự án nào chạy). Tab tải bảng thô `llm_calls` qua gateway: thời điểm, dự án, vai AI, bước, model, kết quả, token, thời gian; nút **Xem** mở phản hồi trong hộp thoại (kèm mã cuộc gọi, lỗi và nút Sao chép).
- Bộ lọc dự án, vai AI, kết quả ở trên bảng; chọn từng dòng và "Chọn tất cả" (các dòng đang hiện); "Tải thêm" theo `next_before`. Hai hành động xoá ở dưới bảng, nút đỏ, đi qua hộp xác nhận nói rõ số dòng, rằng không khôi phục được và số liệu thống kê tính trên các dòng đó cũng mất: **Xoá N dòng đã chọn**, **Xoá các dòng cũ hơn 30 ngày** (kể cả dòng chưa hiện trên trang).
- Các trang của bước 8 trở đi bị gỡ: `VisualDirectorStepPage` (Storyboard), `IllustrationsStepPage`, `CodeStepPage`, cùng `AuthoringModeBar`, `PipelineSettingsBar`, `AuthoringModelPicker`, `CodeSegmentsPanel`, `ProjectIllustrationsPanel`, `IllustrationsGateNote`, `AuthoringRunContext` và `ScriptOutlineStepPage` (màn cũ không còn đường vào). Các bước 9, 13, 14 của luồng 21 bước hiện trang "Bước này chưa phát triển" (cờ `comingSoon` trong `utils/flow.ts`; bước server phía sau vẫn tồn tại). Các địa chỉ cũ `/create/script/storyboard|illustrations|code` chuyển sang trang đó, `/create/script/outline` về bước Creative brief.
- `GET /v1/llm/stats` đã có ở gateway nhưng chưa có màn hình nào đọc: chỗ hiển thị số liệu chunk (trước nằm ở bộ chọn model của bước Code) bị gỡ cùng bước Code.

### E. Hợp đồng và tài liệu

- `docs/contracts/README.md` và một tài liệu hợp đồng mới `llm-chat-v1.md` cho `/v1/chat` (các trường mới, `call_id`, `expect_json`), và cho `/v1/llm/*`.
- Cập nhật mục `/v1/chat` trong `cr-039-chunked-code-pipeline-llm-service.md` trỏ tới hợp đồng mới.
- `docker-compose.yml`: dịch vụ `llm-db`, `depends_on` của llm-service; `.env.example` nếu có biến mới.

## Ngoài phạm vi

- Lưu prompt.
- Đọc lại response đã lưu / nút thử lại không gọi AI (phần sau).
- Chuyển dữ liệu cũ của `llm_usage` (bỏ hẳn).
- Tác vụ nền tự xoá theo ngày; chỉ có nút xoá tay.
- Làm lại bước 8 (storyboard, minh hoạ, code) gọi thẳng llm-service.
- Một màn hình đọc `/v1/llm/stats`.

## Câu hỏi mở (đã trả lời 2026-10-25)

1. `llm_usage` ở orchestrator: xoá luôn.
2. Xoá theo 30 ngày: nút bấm tay trong Nhật ký.
3. Xoá dòng làm mất số liệu thống kê: chấp nhận.
4. `generate_authoring`: xoá cả cụm, bước 8 làm lại sau và gọi thẳng llm-service (xem C).
5. Số CR-100.

## Kiểm thử (dự kiến)

- llm-service: ghi một dòng cho mỗi cuộc gọi (thành công, lỗi, thử lại); ghi lỗi không làm hỏng cuộc gọi; `expect_json` với JSON có `}` thừa, có ghi chú phía sau, có hàng rào, hỏng rồi được sửa, hỏng sau khi sửa; endpoint stats trả đúng hình dạng; xoá theo `ids`, `older_than_days`, `all`.
- authoring-service: các `Parse…Reply` nhận JSON sạch; lỗi kèm `call_id`; không còn tham chiếu `llm_usage`.
- api-gateway: ba route chuyển đúng sang llm-service.
- web-gui: tab Cuộc gọi AI tải và hiển thị bảng, chọn tất cả, hai hành động xoá có modal; `PipelineSettingsBar` đọc stats từ route mới.
- AI giả lập ở mọi test; không gọi Hive hay Ollama. Smoke test trên stack đang chạy xong xoá các dòng `llm_calls` do test tạo (nhận ra bằng `project_id` của test).

## Triển khai

Rebuild và restart: `llm-service`, `authoring-service`, `api-gateway`, `web-gui`, và dịch vụ mới `llm-db`, rồi xác nhận healthy.
