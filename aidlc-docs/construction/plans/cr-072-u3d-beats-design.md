# CR-072 U3d · Bước 5 Beat sheet — thiết kế

Nhánh `feature/cr-072-u3d-beats` (worktree riêng, từ main `7d2a5b1`). Nguồn: FR-30, `unit-of-work.md` mục U3d, mockup "Giao diện đã chốt" (màn `beats`). Khuôn: bước Câu chuyện (U3c) đã có trong code. Trạng thái: Creator duyệt thiết kế và chốt hai điểm (không đụng orchestrator; khoá bền b1, b2…); đã code, chờ Creator kiểm.

## Phụ thuộc

- Cần U3a và U3c: cả hai đã vào main. Chặn U3e (Viết kịch bản đọc `beat`).
- Không trùng với CR nào đang mở (U3d là bước kế tiếp sau Câu chuyện; nhánh U1, U2a còn mở nhưng không đụng bảng/tệp của bước này).

## Phạm vi (theo cách U3c đã làm)

Làm đúng màn Beat sheet theo mockup, đủ backend. Không đụng orchestrator (`flow.go`) và bước Kịch bản đang chạy; nút "Sang viết kịch bản" mở trang "chưa phát triển" cho tới U3e. (Đơn vị công việc gốc có ghi orchestrator, nhưng U3c đã chọn không đụng và Creator chấp nhận; xem "Cần Creator quyết".)

## Thiết kế chức năng

### Thực thể (đơn vị duyệt, ADR-0035; bảng `review_units` đã có)

| Kind | Key | Nội dung |
|---|---|---|
| `beat_sheet` | `main` | cổng giữ thứ tự: `order[]` (key beat), `next_seq`; bước xong khi mọi beat đã duyệt |
| `beat` | `b1`, `b2`… (key tăng dần, không dùng lại) | `fn`, `title`, `goal`, `content`, `facts[]`, `dur` (giây), `next` |

- `beat` dựng từ `story:main` (`input_ref`/`input_sha`): đổi câu chuyện thì `stale`.
- Trường Creator sửa tay vào `creator_fields` của beat; AI không ghi đè (FR-17).
- Chức năng `fn` thuộc: Hook, Đặt vấn đề, Giải thích cơ chế, Cao trào, Mở rộng hệ quả, Payoff.

### Luồng nghiệp vụ

1. Điều kiện vào: câu chuyện đã duyệt.
2. **Viết beat sheet** (AI, vai `beat_sheet`): đọc brief, khung câu chuyện, dữ kiện dùng, thời lượng mục tiêu; trả cả danh sách beat một lượt. Mã dữ kiện lạ, thiếu trường, JSON hỏng là lỗi, không lưu gì.
3. **Từng beat**: Sửa tay (hộp thoại), Làm lại kèm ghi chú (AI viết riêng beat đó, đọc các beat còn lại), Duyệt. Đổi chức năng, đổi thời lượng ± 5 s, gắn/bỏ dữ kiện làm beat về chờ duyệt.
4. **Cấu trúc**: thêm beat sau một beat (AI viết beat mới nối giữa hai beat), xoá beat (xác nhận, nêu dữ kiện bắt buộc sẽ không còn trong beat nào), đổi thứ tự lên/xuống.
5. **Kiểm cấu trúc** (máy, tính lại mỗi lần tải): tổng thời lượng trong khung mục tiêu; hook ≤ 20 s; dữ kiện bắt buộc có trong ít nhất một beat; beat nào cũng có mục tiêu; beat cuối là Payoff. Là cảnh báo, không chặn duyệt.
6. **Duyệt**: từng beat; "Duyệt tất cả beat còn lại". Bước xong khi mọi beat đã duyệt.

### Quy tắc

- Mục tiêu trống thì beat không duyệt được. Tối thiểu 1 beat; thời lượng beat ≥ 5 s.
- Mã dữ kiện của beat phải là dữ kiện đang giữ trong bảng dữ kiện (như mockup chọn trong mọi dữ kiện giữ lại).
- Đổi thứ tự không đổi trạng thái duyệt của beat; chỉ cổng `beat_sheet` ghi lại thứ tự mới. Hộp xác nhận "lời thoại đã duyệt" của FR-30 chưa áp dụng vì bước Viết kịch bản chưa có dữ liệu (làm ở U3e).
- Tác động: sửa câu chuyện thêm "Beat sheet cần xem lại" vào `Impact` của câu chuyện (nay còn rỗng); sửa một beat liệt kê lời thoại/ghi chú bàn giao của beat đó chỉ khi U3e/U3f có dữ liệu (nay rỗng); đổi thời lượng chỉ đưa chính beat về chờ duyệt.
- Cần AI mà chưa cấu hình: báo rõ, Creator tự viết từng beat.

### Giao diện (frontend-components)

Trang `/projects/:id/beats`, bước 5 thành bước có màn riêng (`screen: "beats"`). Dải trạng thái; thẻ **Cấu trúc** (dải thời gian chia theo thời lượng, năm dòng kiểm); nút "Duyệt tất cả beat còn lại"; mỗi beat một thẻ: tiêu đề `B1 · tên`, chức năng (select), thời lượng (− 5 s / + 5 s), mục tiêu, nội dung chính, dữ kiện dùng (thẻ nhãn có nút ×, select "+ Gắn dữ kiện…"), chuyển sang beat sau (trừ beat cuối), thanh duyệt (Duyệt / Sửa tay / Làm lại), hàng nút ↑ Lên, ↓ Xuống, "+ Thêm beat sau" (tím), "Xoá beat" (đỏ, có xác nhận). Nút "Sang viết kịch bản →" bật khi mọi beat đã duyệt. Dùng lại `UnitEditModal`, `UnitRedoModal`, `UnitReviewBar`, `ImpactConfirm`, `AiOperationCard`, `AiModelSelect` (khoá `beat_sheet`), `FactTags`. Màu nút theo `docs/ux-ui-design-rules.md`.

## Thiết kế mức thấp

### Module (authoring-service, hexagonal như U3c)

| Lớp | Tệp mới |
|---|---|
| domain | `beat.go` (beat, cổng, đọc câu trả lời AI của cả bảng và của một beat, kiểm cấu trúc), `prompt_seed_beat.go` (vai `beat_sheet`, dùng cho cả viết bảng, làm lại và thêm beat) |
| application | `beat_desk.go` (state, viết bảng, thêm/xoá/đổi thứ tự, gắn dữ kiện, đổi thời lượng/chức năng), `beat_units.go` (UnitKind của `beat` và `beat_sheet`) |
| adapters/http | `beat.go` (route) |
| cmd | nối dây trong `main.go` |

Phụ thuộc đi vào trong; tiêm bằng hàm khởi tạo; dùng lại `ReviewUnitPort`, `LLMProviderPort`, `PromptActivePort`, kho dữ kiện.

### Hợp đồng HTTP (qua api-gateway, tiền tố `/v1/projects/{id}`)

| Phương thức | Đường dẫn | Việc |
|---|---|---|
| GET | `/beats` | toàn màn: cổng, beat theo thứ tự, dữ kiện dùng, kiểm cấu trúc, điều kiện vào |
| POST | `/beats/generate` | AI viết cả bảng (kind `beat_sheet`, theo dõi bằng `X-Operation-Id`) |
| POST | `/beats` | thêm beat sau một beat (`after`; AI viết, kind `beat`) |
| PUT | `/beats/order` | đổi thứ tự |
| DELETE | `/beats/{key}` | xoá beat |
| GET/PUT/POST | `/units/beat/{key}` (+ `/redo`, `/impact`, `/approve`, `/unapprove`); sửa cả chức năng, thời lượng, dữ kiện | API đơn vị chung |

Phiên bản `/v1`, chỉ thêm trường. Tài liệu hợp đồng `docs/contracts/beats-v1.md`.

### Trình tự chính

1. Mở màn: `GET /beats`; thiếu điều kiện vào thì báo "Duyệt câu chuyện trước".
2. Viết bảng: đọc brief + khung + dữ kiện dùng + thời lượng → AI → kiểm câu trả lời → lưu các `beat` (giữ trường Creator của beat cùng key) + cập nhật `order` → các beat về chờ duyệt.
3. Thêm beat: đọc hai beat kề → AI → lưu beat mới → chèn vào `order`.
4. Xoá beat: UI hỏi xác nhận từ `Impact`; backend xoá và cập nhật `order`.

### Kiểm thử

Go: đọc câu trả lời AI (đúng, thiếu trường, mã lạ, JSON hỏng), kiểm cấu trúc, use case với AI giả, kho lưu với Postgres tạm, HTTP. Gateway: route. Web-gui: vitest cho màn, hook, hộp thoại. Dọn dữ liệu test sau khi chạy.

## Model AI của bước

Ô chọn model riêng khoá `beat_sheet` (cột `beat_sheet_model`), như `story_frame`.

## Cần Creator quyết

1. **Orchestrator**: giữ như U3c (không đụng `flow.go`; bước Kịch bản cũ vẫn tự dựng dàn ý đến khi U3e thay) — đề xuất — hay sửa `flow.go` ngay để Kịch bản cũ đọc beat đã duyệt?
2. **Key beat**: dùng key bền `b1…` (đề xuất, đổi thứ tự không đổi key; U3e gắn lời thoại theo key) hay đánh số theo vị trí?

## Chưa làm (nói rõ)

- Bước Viết kịch bản và sau; tác động của beat lên lời thoại/ghi chú bàn giao; xác nhận đổi thứ tự khi lời thoại đã duyệt.
- Chưa chạy AI thật nếu local thiếu `HIVE_API_KEY`; phần AI chỉ kiểm bằng test giả lập câu trả lời.
