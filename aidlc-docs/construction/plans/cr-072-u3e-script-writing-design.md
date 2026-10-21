# CR-072 U3e · Bước 6 Viết kịch bản — thiết kế

Nhánh `feature/cr-072-u3e-script-writing` (worktree riêng, từ main `83f2b8a`). Nguồn: FR-31, FR-9 (phần đọc từng câu), ADR-0035, ADR-0040, ADR-0041, `unit-of-work.md` mục U3e, mockup "Giao diện đã chốt" (màn `write`). Khuôn: Câu chuyện (U3c) và Beat sheet (U3d). Trạng thái: Creator duyệt thiết kế và chốt bốn điểm (1 giữ nguyên `flow.go`; 2 AI viết từng beat; 3 làm đủ đường RabbitMQ; 4 máy cờ bằng bảng luật cố định); đã code, chờ Creator kiểm.

## Phụ thuộc và trùng lặp

- Cần U3a (khung đơn vị), U3d (beat có khoá bền b1, b2…): cả hai đã vào main. Chặn U3f (Rà soát và chốt đọc `script_beat`).
- Cỡ L, đụng năm service: authoring-service, orchestrator, tts, api-gateway, web-gui. Nhánh `fix/drop-dead-prompt-vars` còn mở nhưng chỉ đụng biến prompt; sẽ kiểm lại khi code nếu nó chạm `prompt_vars.go`. Nhánh U1, U2a không đụng tệp của bước này.

## Phạm vi

Làm đúng màn Viết kịch bản theo mockup, đủ backend, gồm cả đọc từng câu để nghe thử:

1. Lời thoại theo beat do AI viết, Creator sửa/làm lại/duyệt từng beat.
2. Cờ câu dài và câu khó nghe (máy tính, kèm gợi ý AI), kiểm câu mở và câu kết, kiểm chỗ nối giữa hai beat kèm gợi ý.
3. Nghe từng câu, cả beat, cả kịch bản bằng giọng chọn ở Cấu hình; ước tính thời lượng kịch bản từ giọng thật khi đã đọc.
4. Đường đọc từng câu (ADR-0040, phần chưa có cách đọc) và cách phát file chung từ `/shared` (ADR-0041).

Không làm ở CR này (nói rõ): từ điển phát âm, "đọc là", tốc độ, khoảng nghỉ, Giọng sơ bộ (bước 11) và Animatic: U6. Bộ bàn giao và chốt: U3f. Storyboard đọc lời thoại theo beat: ghi ở "Cần Creator quyết" câu 1.

## Thiết kế chức năng

### Thực thể (đơn vị duyệt, ADR-0035; bảng `review_units` đã có)

| Kind | Key | Nội dung |
|---|---|---|
| `script_beat` | theo khoá beat (`b1`, `b2`…) | `lines[]`: mỗi câu `{id, text, facts[]}`; `id` ổn định `{beat}.{n}` không dùng lại; `facts[]` là mã dữ kiện của câu |

- Dựng từ `beat:{key}` (`input_ref`/`input_sha`): beat đổi thì `script_beat` tương ứng `stale`; chỉ beat đó cần làm lại, beat khác giữ nguyên.
- Chữ Creator sửa tay vào `creator_fields` của câu; AI không ghi đè (FR-17).
- Cờ câu dài, câu khó nghe, kiểm câu mở/kết, kiểm chỗ nối là **kết quả tính lại mỗi lần tải**, không lưu; gợi ý AI cho cờ/chỗ nối sinh khi Creator bấm "Xem gợi ý" và chỉ lưu khi bấm "Áp dụng".
- Ước tính thời lượng: ưu tiên thời lượng thật của file giọng; câu chưa đọc dùng `EstimateNarrationDuration`. Hiện rõ bao nhiêu câu là ước tính.

### Luồng nghiệp vụ

1. **Điều kiện vào**: beat sheet đã duyệt (mọi beat duyệt). Chưa thì báo "Duyệt beat sheet trước".
2. **Viết lời thoại cho một beat** (AI, vai `script_writer`): đọc brief, khung câu chuyện, beat đó (mục tiêu, nội dung, dữ kiện, thời lượng), beat liền trước và liền sau, ngôn ngữ nội dung. Trả danh sách câu kèm mã dữ kiện. Mã dữ kiện lạ, câu rỗng, JSON hỏng là lỗi, không lưu gì. Mockup viết theo beat; nút "Viết lời thoại" ở mỗi beat chưa có lời, và "Viết các beat còn lại" chạy tuần tự từng beat (kind AI `script_beat`, theo dõi bằng `X-Operation-Id`).
3. **Từng beat**: Sửa tay (hộp thoại, sửa/thêm/xoá câu), Làm lại kèm ghi chú (AI viết lại beat đó), Duyệt. Sửa chữ một câu chỉ đưa beat đó về chờ duyệt.
4. **Gợi ý**: câu có cờ → "Xem gợi ý" (AI viết lại đúng câu đó) → "Áp dụng" thay chữ câu. Chỗ nối B(n-1) → B(n) có cờ → gợi ý thay **câu đầu của beat sau** (chỉ đổi câu đó, như tiêu chí xong).
5. **Nghe thử**: ▶ Nghe (một câu), ▶ Nghe cả beat, ▶ Nghe cả kịch bản. Câu chưa có file thì gửi đi đọc rồi phát khi xong; sửa một câu chỉ đưa đúng câu đó vào danh sách chờ đọc, câu khác dùng lại file (`cache_hit`).
6. **Duyệt**: từng beat; "Duyệt tất cả beat còn lại". Bước xong khi mọi beat đã duyệt; nút "Sang rà soát và chốt →" bật khi đó.

### Quy tắc

- Beat chưa có lời thoại thì không duyệt được. Câu phải có chữ. Mỗi beat ≥ 1 câu.
- Dữ kiện của câu phải là dữ kiện đang giữ trong bảng dữ kiện; dữ kiện có nhãn "chưa có nguồn" vẫn dùng được (bước Nghiên cứu tuỳ chọn), hiện nhãn như mockup.
- **Câu dài**: vượt ngưỡng ký tự/giây đọc theo ngôn ngữ (hằng số trong domain, kiểm bằng test). **Câu khó nghe**: có cụm chữ-số/viết tắt/dấu khó đọc theo bảng luật cố định trong domain (mã máy, không gọi AI); AI chỉ viết gợi ý.
- Kiểm câu mở: câu đầu của beat Hook có dấu hỏi hoặc ≤ 7 s; kiểm câu kết: câu cuối của beat Payoff đối chiếu `story.payoff`/beat cuối (máy kiểm cờ hiệu, AI không đánh giá).
- Tác động: sửa beat thêm "Lời thoại của beat này cần viết lại" vào `Impact` của beat (nay rỗng ở mục này); sửa lời thoại liệt kê ghi chú bàn giao/giọng chỉ khi U3f/U6 có dữ liệu (nay rỗng). Đổi thứ tự beat: không đổi trạng thái duyệt của lời thoại, nhưng chỗ nối mới được kiểm lại, và hộp xác nhận "lời thoại đã duyệt" của FR-30 nay áp dụng (đã chuẩn bị ở U3d).
- Cần AI mà chưa cấu hình: báo rõ, Creator tự viết lời.

### Đọc từng câu (ADR-0040 phần chưa có cách đọc; ADR-0041)

```
web-gui ─ POST /v1/projects/{id}/script/speak {line_ids} ─▶ api-gateway ─▶ authoring-service
authoring-service ─ POST /internal/v1/projects/{id}/draft-voice ─▶ orchestrator (DraftVoiceDispatcher)
orchestrator ─ GET /internal/v1/authoring/{id}/voice-lines?pending=1 ─▶ authoring-service
orchestrator ─ synthesize_lines (RabbitMQ) ─▶ tts   (SynthesizeLines, cache theo chữ + giọng)
tts ─ ghi /shared/{id}/audio/{line}_{cache_key}.wav ─ lines_synthesized ─▶ orchestrator
orchestrator ─ POST /internal/v1/authoring/{id}/voice-lines/results ─▶ authoring-service (VoiceLines: đường dẫn + thời lượng)
web-gui ─ GET /v1/projects/{id}/audio/{file} (api-gateway phát từ /shared, có Range) ─▶ <audio>
```

- `cache_key` = hash(chữ đã chuẩn hoá, giọng, engine). U6 thêm cách đọc vào khoá; file dựng theo khoá U3e vẫn đúng với câu chưa có cách đọc.
- tts chủ ghi thư mục `/shared/{id}/audio/`; authoring-service chỉ lưu đường dẫn và thời lượng (bảng `voice_lines`: `project_id`, `line_id`, `cache_key`, `path`, `duration_s`).
- Giọng lấy từ Cấu hình của project. Chưa chọn giọng hoặc tts lỗi: báo rõ trên câu, không giả thành công.
- Kết quả đọc báo về màn bằng thăm dò (như thao tác AI hiện có), không cần WebSocket.

### Giao diện (frontend-components)

Trang `/projects/:id/script`, bước 6 thành bước có màn riêng (`screen: "write"`). Dải trạng thái; thẻ **Toàn bộ kịch bản** (số câu, số beat, ước tính thời lượng so với mục tiêu, kiểm câu mở/kết, số câu cờ, số chỗ nối cần xem, nút "Nghe cả kịch bản"); nút "Duyệt tất cả beat còn lại"; mỗi beat một thẻ: `B1 · chức năng · tên`, thời lượng ước tính, mục tiêu, chỗ nối từ beat trước (chip ✓/!, "Xem gợi ý"), từng câu (`id`, giây, chữ, nhãn dữ kiện, chip cờ, nút ▶ Nghe, "Xem gợi ý"/"Áp dụng"), "▶ Nghe cả beat", thanh duyệt (Duyệt / Sửa tay / Làm lại). Dùng lại `UnitEditModal`, `UnitRedoModal`, `UnitReviewBar`, `ImpactConfirm`, `AiOperationCard`, `AiModelSelect` (khoá `script_writer`), `FactTags`. Màu nút theo `docs/ux-ui-design-rules.md`. Trình phát là `<audio>` ẩn, một câu một lần.

## Thiết kế mức thấp

### authoring-service (Go, hexagonal như U3d)

| Lớp | Tệp mới |
|---|---|
| domain | `script.go` (câu, lời thoại một beat, đọc câu trả lời AI, cờ câu dài/khó nghe, kiểm câu mở/kết, kiểm chỗ nối), `prompt_seed_script.go` (vai `script_writer` cho viết, làm lại, gợi ý câu, gợi ý chỗ nối) |
| application | `script_desk.go` (state, viết một beat, viết các beat còn lại, áp gợi ý, sửa câu), `script_units.go` (UnitKind `script_beat`: Edit, Redo, Impact, CheckApprovable), `voice_lines.go` (danh sách chờ đọc, nhận kết quả, ước tính thời lượng) |
| adapters/http | `script.go` (route công khai), `voice_internal.go` (route nội bộ cho orchestrator) |
| adapters/postgres | `voice_line_repo.go` + migration bảng `voice_lines`, cột `script_writer_model` |
| cmd | nối dây trong `main.go` |

### orchestrator (Go)

`DraftVoiceDispatcher` (application): nhận lệnh nội bộ, lấy danh sách câu chờ đọc từ authoring-service, gửi `synthesize_lines`, nhận `lines_synthesized`, chuyển kết quả về authoring-service. Ngoài saga render (`flow.go` không đụng). Thêm exchange/queue và consumer, bám khuôn consumer đang có.

### tts (Python)

`SynthesizeLines` (application) bám `SynthesizeSpeechUseCase`, một lượt nhiều câu, bỏ qua câu đã có file cùng `cache_key` (báo `cache_hit`), kiểm `should_stop` như batch hiện có; consumer cho `synthesize_lines`; producer cho `lines_synthesized`; ghi qua `artifact_paths`.

### api-gateway (Node)

Route `/v1/projects/:id/script...` proxy sang authoring-service; route `GET /v1/projects/:id/audio/:file` dùng `sharedFileHandler` trên `/shared/{id}/audio` với `namePattern` chặt (`{line}_{cache_key}.wav`), kiểm đường dẫn nằm trong thư mục, có Range.

### Hợp đồng HTTP công khai (tiền tố `/v1/projects/{id}`, chỉ thêm trường)

| Phương thức | Đường dẫn | Việc |
|---|---|---|
| GET | `/script` | toàn màn: lời thoại theo beat, cờ, chỗ nối, kiểm, ước tính thời lượng, trạng thái đọc, điều kiện vào |
| POST | `/script/generate` | AI viết lời cho các beat chưa có lời (`beat` tuỳ chọn để chỉ một beat); theo dõi bằng `X-Operation-Id` |
| POST | `/script/lines/{line}/suggest` | AI gợi ý thay một câu hoặc câu đầu beat sau (`kind: line|join`) |
| POST | `/script/speak` | `{line_ids}` đưa câu vào hàng chờ đọc |
| GET | `/audio/{file}` | phát file giọng (api-gateway) |
| GET/PUT/POST | `/units/script_beat/{key}` (+ `/redo`, `/impact`, `/approve`, `/unapprove`) | API đơn vị chung |

Hợp đồng nội bộ (orchestrator ↔ authoring-service): `GET /internal/v1/authoring/{id}/voice-lines?pending=1`, `POST /internal/v1/authoring/{id}/voice-lines/results`. Lệnh/sự kiện RabbitMQ: `synthesize_lines`, `lines_synthesized`. Tài liệu `docs/contracts/script-v1.md`, cập nhật `docs/contracts/shared-artifacts.md` (thư mục `/shared/{id}/audio/` có chủ ghi là tts) và README hợp đồng.

### Kiểm thử

Go: đọc câu trả lời AI (đúng, thiếu trường, mã lạ, câu rỗng, JSON hỏng), cờ và kiểm câu mở/kết/chỗ nối, use case với AI giả và tts giả, kho `voice_lines` (Postgres tạm), HTTP, dispatcher orchestrator với fake cổng. Python: `SynthesizeLines` (có/không cache, lỗi engine, dừng giữa chừng), consumer. Gateway: route và `sharedFileHandler` từ chối tên sai/đi ra ngoài thư mục. Web-gui: vitest cho màn, hook, trình phát. Test đọc giọng dùng engine giả, không gọi Hive, Ollama hay Azure. Dọn dữ liệu test sau khi chạy.

## Model AI của bước

Ô chọn model riêng khoá `script_writer` (cột `script_writer_model`), như `beat_sheet`.

## Khác với thiết kế lúc duyệt (khi code)

- **Model AI**: dùng lại khoá `story` (cột `story_model`, nhãn "Kịch bản") của bước Kịch bản cũ thay vì thêm cột `script_writer_model`; bước này thay bước đó. Vai prompt vẫn là `script_writer` mới.
- **Lấy câu chờ đọc**: `POST /internal/v1/authoring/{id}/voice-lines/claim` thay cho `GET …?pending=1`, vì lời gọi đổi trạng thái (đánh `running`). Thêm `…/voice-lines/fail` cho lệnh không tới được tts.
- **Nút viết lời thoại** chạy thẳng, không có hộp thoại ghi chú: hộp "Làm lại kèm ghi chú" bắt buộc có ghi chú, còn viết beat chưa có lời không xoá gì. Làm lại một beat vẫn qua hộp thoại có ghi chú.
- **Kiểm câu kết**: máy kiểm câu cuối của kịch bản khép câu (có dấu kết câu) và không bị cờ "Câu dài"; không đối chiếu với `story.payoff` vì đối chiếu chữ với chữ sẽ chỉ là đoán. **Kiểm chỗ nối**: máy kiểm hai câu liền nhau cùng mở bằng một từ, hoặc câu cuối beat trước thiếu dấu kết câu; AI chỉ gợi ý thay câu đầu beat sau.
- **Khung đổi thứ tự beat**: hộp xác nhận ở bước Beat sheet hiện khi có beat đã duyệt lời thoại (`script_approved` của `GET /beats`).
- **Sửa dữ kiện từng câu** có ở API (`facts:<id>`) nhưng chưa có ô trên màn; dữ kiện của câu hiện chỉ do AI gắn.

## Đã chốt (câu hỏi lúc duyệt)

1. **Storyboard và bước Kịch bản cũ**: unit-of-work ghi "Storyboard hiện có đọc lời thoại theo beat". Đề xuất như U3c/U3d: **không đụng `flow.go` và bước Kịch bản đang chạy**; "Sang rà soát và chốt" mở trang "chưa phát triển" tới U3f; Storyboard đọc lời thoại mới do U5 làm (U5 đã được giao chuyển Storyboard sang dạng mới). Hay sửa orchestrator ngay để Storyboard cũ đọc `script_beat` đã duyệt?
2. **Viết một beat hay cả kịch bản một lượt**: đề xuất AI viết **từng beat** (một lời gọi AI mỗi beat, thấy beat liền trước/sau) vì làm lại một beat không đụng beat khác; "Viết các beat còn lại" chạy tuần tự. Phương án khác: một lời gọi cả kịch bản (rẻ hơn, nhưng làm lại phải giữ cả kịch bản).
3. **Phạm vi đọc giọng**: làm đủ đường RabbitMQ qua orchestrator + tts đúng ADR-0040 (đề xuất, nhiều phần di chuyển nhất), hay tạm nghe thử qua một route HTTP trực tiếp từ authoring-service tới tts? ADR-0040 đã chọn RabbitMQ nên chỉ đổi khi Creator muốn.
4. **Cờ câu khó nghe do máy** (bảng luật cố định) thay vì AI: đề xuất máy; AI chỉ viết gợi ý khi Creator mở. Đồng ý?

## Chưa làm (nói rõ)

- Cách đọc ("đọc là", tốc độ, khoảng nghỉ), từ điển phát âm, Giọng sơ bộ, Animatic (U6).
- Ghi chú bàn giao, bộ bàn giao, chốt phiên bản (U3f).
- Chưa chạy AI thật nếu local thiếu `HIVE_API_KEY`; chưa nghe giọng thật nếu thiếu khoá giọng; phần đó chỉ kiểm bằng test giả lập.
