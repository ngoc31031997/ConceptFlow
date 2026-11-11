# CR-112b · Bước 8 Ngôn ngữ hình ảnh (Visual Concept) — giao việc

Nhánh `feature/cr-112b-visual-concept-step`, worktree `.claude/worktrees/feature-cr-112b-visual-concept-step`, tạo từ main sau CR-112a và CR-113. Chạy song song với CR-112c (`feature/cr-112c-reference-library`, thư viện tham chiếu).

## Đọc trước
- **Thiết kế đã duyệt:** `aidlc-docs/construction/plans/cr-112-visual-concept-design.md`. Đây là nguồn chính: input, khuôn tài liệu bản chốt, 8 phần, máy kiểm, luồng, API, kiểm thử.
- **ADR:**
  - `ADR-0049-visual-service-owns-visual-preproduction.md`
  - `ADR-0048-llm-service-owns-prompt-library.md`
  - `ADR-0047-external-ai-through-the-chat-gate.md`
  - `ADR-0035` (đơn vị duyệt)
- **Hợp đồng:** `docs/contracts/visual-service-v1.md`, `prompt-library-v1.md`, `external-ai-v1.md`, `llm-chat-v1.md`, `script-lock-v1.md` (route `script-versions/latest`), `ai-bridge-v1.md`.
- **Quy tắc:** `docs/ux-ui-design-rules.md`, `docs/code-standards-rules.md`, `CLAUDE.md`.
- **Mockup đã chốt (v3):** https://claude.ai/artifact/JSuFKcJQBkQ2qAVZRKRsck. Đọc bằng Artifact tool, action `read`.
- **Khuôn code để làm theo:**
  - Bước 7 (authoring `script_lock_*`, web-gui `LockStepPage`) cho rà soát, vấn đề và chốt phiên bản.
  - Bước 5 (`BeatSheetCard`, `BeatAiDialog`, `useDeepSeekQueue`, CR-109, CR-111) cho thẻ đầu, phần gấp lại, hộp thoại AI và "Chạy tất cả bằng DeepSeek".
  - Middleware `X-External-AI` của authoring (`internal/adapters/http/external_ai.go`, `internal/application/external_ai.go`).
  - Client thư viện prompt của authoring (`internal/adapters/llm/library.go`).

## Phạm vi

### visual-service
1. **Đọc input:**
   - Gọi `GET /internal/v1/authoring/{id}/script-versions/latest`.
   - `404` nghĩa là chưa chốt kịch bản → trả `409 script_not_locked`.
   - Lỗi khác báo rõ tên authoring-service.
   - Hash từng phần của bản chốt (`sha`) chính là `input_hashes`. Thêm hash của các mô tả tham chiếu đã chọn vào khoá `refs`.
2. **domain:**
   - 8 phần và schema của từng phần (bảng "8 phần" trong thiết kế).
   - Kiểm từng phần khi lưu.
   - Máy kiểm cả bộ: tương phản WCAG, phủ thuật ngữ, phủ giả thuyết và ngộ nhận.
   - Ghép tài liệu bản chốt theo khuôn JSON trong thiết kế, gom `accuracy_notes`.
   - Đọc reply của AI; gộp các vấn đề rà soát qua nhiều lượt.
   - Stale theo từng nguồn input; `creator_fields` theo đường dẫn trường.
3. **AI:**
   - `/v1/chat` và `/v1/chat/external` của llm-service. Template lấy bằng `GET /internal/v1/prompts/{role}`.
   - Lúc khởi động, đẩy bản hệ thống bằng `PUT /internal/v1/prompts/system` với `owner = "visual"`, thử lại với quãng nghỉ tăng dần như authoring.
   - 9 role: `visual_concept_creative_direction`, `…_color_system`, `…_visual_language`, `…_representation_rules`, `…_typography`, `…_motion_language`, `…_do_not_use`, `…_feasibility`, `visual_concept_review`.
   - Middleware `X-External-AI`: mỗi request đúng một prompt, chỉ ghi DB sau khi có kết quả.
   - Model: `project_visual.concept_model`; để trống thì dùng model mặc định của bước `visual_concept` ở llm-service.
4. **API công khai:** đúng bảng "Hợp đồng HTTP công khai" trong thiết kế, gồm cả `PUT /v1/projects/{id}/visual-concept/refs`. Route này lưu `project_reference_picks`; tên và mô tả đọc thẳng từ bảng `reference_styles` qua repository của CR-112b.
5. **API nội bộ:** `GET /internal/v1/visual/{id}/concept?step=…` (bản chốt mới nhất; `404` khi chưa chốt).

### llm-service
- Thêm `"visual_concept"` vào `STEPS` ở `app/modelcatalog.py`.
- Sửa test liên quan.
- Cập nhật `docs/contracts/prompt-library-v1.md` nếu có liệt kê bước.

### api-gateway
- Proxy `/v1/projects/:id/visual-concept…` sang visual-service. Route AI dùng timeout dài như các route AI của authoring.
- Cho phép header `X-External-AI` như hiện có.
- Thêm `VISUAL_SERVICE_URL` trong `docker-compose.yml`. CR-112c cũng thêm biến này: nếu đã có thì giữ một dòng.

### docker compose
visual-service thêm `AUTHORING_SERVICE_URL` và `LLM_SERVICE_URL`.

### web-gui
1. **Flow 23 bước:**
   - Bỏ bước Phân rã kịch bản. Bước 8 là "Ngôn ngữ hình ảnh"; các bước sau lùi một số.
   - Sửa ở thanh bước, bản đồ flow, menu, `useStepNav`, `ComingStepPage`.
   - Các bước 9 trở đi vẫn mở trang "chưa phát triển" như hiện nay.
   - Không xoá trang Cài đặt › Ngôn ngữ hình (`VisualLanguagePage`) và không đổi luồng render cũ. Ghi vào báo cáo là chưa làm.
2. **Trang** `/projects/:id/visual-concept`, bám mockup v3:
   - thẻ đầu (đầu vào, "Xem đầu vào của AI", dải màu theo beat, máy kiểm, chọn model, viết lại cả bộ, "Chạy tất cả bằng DeepSeek (N)", "Duyệt tất cả");
   - tab "Quy tắc hình ảnh": thẻ tham chiếu có tick chọn, nút "Mở thư viện tham chiếu" tới `/settings/reference-styles` của CR-112c, 8 phần gấp lại khi đã duyệt;
   - tab "Rà soát và chốt": vấn đề, 5 tiêu chí, chốt v1/v2.
   - "Sửa tay" dùng form theo schema của từng phần (đây là điểm khác mockup đã ghi trong thiết kế).
3. **Bước 7:** nút "Sang …" sau khi chốt mở bước 8 mới.
4. **`ProjectFlowContext`:** bước 8 xong khi đã chốt.

### Tài liệu
- `docs/contracts/visual-concept-v1.md`.
- `docs/contracts/visual-service-v1.md` và `docs/contracts/README.md`.

## Prompt: phải để Creator duyệt
- Viết nguyên văn 9 prompt hệ thống vào `aidlc-docs/construction/plans/cr-112b-prompts-draft.md`, mỗi role một mục: nội dung, các biến dùng, JSON trả về.
- Đọc các prompt có sẵn của bước 5–7 trong authoring (`internal/domain/prompt_seed_*.go`) để viết cùng giọng và cùng quy ước. Không con số cứng, không ví dụ theo một chủ đề cụ thể.
- Seed trong code dùng đúng nội dung của file nháp.
- **Không rebuild visual-service và không đẩy seed lên llm-service của stack trước khi Creator duyệt prompt.** Đẩy seed lên là prompt có hiệu lực ngay.
- Phần không cần prompt (đọc input, sửa tay, duyệt, máy kiểm, chốt, web-gui) làm và test được bằng AI giả.

## Trùng với CR-112c
Cả hai CR đều sửa `cmd/visual-service/main.go`, `internal/adapters/http/router.go`, `docker-compose.yml`, `web-gui/src/api/client.ts`, `App.tsx` và menu. Thay đổi ở các file này **chỉ là thêm vào**, không sắp xếp lại code có sẵn.

Trước khi báo xong: `git fetch` rồi merge `origin/main`. Nếu có xung đột không phải dạng "hai bên cùng thêm dòng" thì dừng và báo.

## Kiểm thử
Đúng mục "Kiểm thử" của thiết kế, thêm test cho `STEPS` của llm-service. Không gọi Hive hay Ollama thật. Dọn mọi dữ liệu test (tiền tố `test-cr112b`).

## Xong khi
- Test Go, pytest và vitest đều qua.
- Đã rebuild `llm-service`, `api-gateway`, `web-gui` và cả ba healthy. **visual-service chỉ rebuild sau khi Creator duyệt prompt.**
- **Không commit.** Báo lại cho Creator:
  - đã làm gì, test nào chạy và kết quả;
  - đã dọn gì;
  - chỗ nào khác mockup, chỗ nào chưa làm;
  - đường dẫn file prompt nháp để duyệt.
