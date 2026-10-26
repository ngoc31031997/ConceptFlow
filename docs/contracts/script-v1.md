# Viết kịch bản: API của bước 6 (`/v1`) và đường đọc giọng từng câu

Bước Viết kịch bản (authoring-service) viết lời thoại cho từng beat đã duyệt, kiểm lời thoại và đưa từng câu đi đọc để nghe thử. Mọi đường dẫn công khai nằm dưới `/v1` (ADR-0008) qua api-gateway; chỉ thêm trường thì giữ `/v1`. Mã tương ứng: `services/authoring-service/internal/domain/script.go`, `internal/application/script_*.go`, `voice_lines.go`, `internal/adapters/http/script.go`; `services/orchestrator/internal/application/draft_voice.go`; `services/tts/application/synthesize_lines.py`.

## Đơn vị duyệt

| Kind | Key | Nội dung (`content`) |
|---|---|---|
| `script_beat` (bảng `project_script`: `lines` jsonb, `next_seq`, `status`, `creator_fields`, `redo_count`, `input_ref`, `input_sha`; khoá `project_id` + `beat_key`) | khoá của beat (`b1`, `b2`…) | `{lines:[{id,text,facts[]}], next_seq}`; `id` là `<khoá beat>.<n>`, không dùng lại trong beat; `facts` là mã dữ kiện của câu |

Dựng từ `beat:{key}` (`input_ref`): beat đổi thì lời thoại của beat đó `stale`, beat khác giữ nguyên. Bước xong khi mọi beat có lời thoại và mọi `script_beat` đã duyệt (`status` của màn). Sửa, làm lại, duyệt qua API đơn vị chung: `PUT /v1/projects/{id}/units/script_beat/{key}` với `{"fields":{…}}`: `line:<id>` là chữ của câu (rỗng thì bỏ câu), `facts:<id>` là `"F01, F02"`, `add:<id>` thêm câu mới sau câu `<id>` (`add:start` thêm lên đầu; beat chưa có lời thoại có thể tự viết bằng đường này); `POST …/redo` (AI viết lại cả lời thoại của beat, đánh số câu tiếp tục); `POST …/approve` (cần có ít nhất một câu). Sửa xong beat về chờ duyệt và thành của Creator (`mine: ["lines"]`).

## Đường dẫn công khai

| Phương thức | Đường dẫn | Việc |
|---|---|---|
| GET | `/v1/projects/{id}/script` | Toàn màn: `status`, `beats_approved`, `beats[]` (`key`, `number`, `fn`, `title`, `goal`, `target_seconds`, `written`, `status`, `mine[]`, `stale`, `seconds`, `join`, `lines[]`), `checks[]` (`{ok,text}`: câu mở, câu kết, câu dài hoặc khó nghe, chỗ nối), `stats`, `target_label`, `in_range`, `voice_id` |
| POST | `/v1/projects/{id}/script/generate` | AI viết lời thoại cho các beat chưa có, lần lượt từng beat, mỗi beat đọc lời thoại của beat liền trước và sau; body `{"beat":"","note":""}` tuỳ chọn, `beat` chỉ định một beat (phải chưa có lời thoại); trả state |
| POST | `/v1/projects/{id}/script/suggest` | AI gợi ý một câu thay: body `{"kind":"line"\|"join","line_id":"b2.1"}`; trả `{"text":"…"}`, không lưu gì (áp dụng bằng sửa câu) |
| POST | `/v1/projects/{id}/script/speak` | Đưa câu vào hàng chờ đọc: `{"line_ids":["b1.1"]}`; câu đã có file đúng chữ và giọng thì bỏ qua; trả state |
| POST | `/v1/projects/{id}/script/replace` | Tìm và thay một từ trong lời thoại: `{"find","replace","field":"spoken"\|"text","match_case","apply"}`. Khớp nguyên từ (ký tự liền kề không phải chữ hoặc số). `field = spoken` ghi vào lời đọc (câu chưa có lời đọc riêng thì lời đọc lấy chữ hiển thị đã thay); `text` ghi vào chữ hiển thị. `apply = false` chỉ trả danh sách; `true` ghi mọi beat chưa duyệt, mỗi beat về chờ duyệt như một lần sửa tay. Beat đã duyệt không bị đổi, chỉ đếm. Trả `{"matches":[{"beat","line","before","after"}],"skipped_approved":[{"beat","lines"}],"applied"}`; `400` khi từ rỗng, `field` sai, hoặc có câu trống / không hợp lệ sau khi thay (khi đó không ghi gì) |
| GET | `/v1/projects/{id}/audio/{file}` | Phát file giọng (`audio/wav`, có Range) từ `/shared/{id}/audio/`; chỉ tên dạng `b1-2_<12 hex>.wav` |

Mỗi câu trong `lines[]`: `id`, `text`, `facts[]`, `seconds`, `estimated` (true khi chưa có file giọng cho đúng chữ hiện tại, thời lượng tính theo tốc độ đọc), `flag` (`null` hoặc `{kind:"Câu dài"|"Khó nghe", msg}`) và `audio` (`{state:"none"|"pending"|"running"|"ready"|"failed", file?, error?}`). Câu sửa chữ hoặc đổi giọng về `none`. Cờ và kiểm chỉ cảnh báo, không chặn duyệt; **do máy tính** bằng bảng luật cố định (câu quá 28 từ; có ngoặc; chữ trộn số; số dài; chữ viết tắt in hoa; ký hiệu như `%`), AI chỉ viết gợi ý khi được hỏi.

Lượt AI nhận header `X-Operation-Id` và ghi nhận dưới kind `script_beat`. Model AI của bước là model Creator chọn cho bước ở khoá `story` của `PUT /v1/projects/{id}/authoring/models` (bước Kịch bản dùng lại khoá này); mặc định theo bước đặt ở Cài đặt · Model AI, khoá `story`.

## Mã lỗi

| Mã | Khi |
|---|---|
| `400` | Sửa trường lạ, câu trống hoặc quá 600 ký tự, mã dữ kiện không có, mã câu không có; duyệt beat không có lời thoại; viết một beat đã có lời thoại; xin gợi ý cho câu không có cờ |
| `404` | Beat hoặc câu không có |
| `409` | Beat sheet chưa duyệt, hoặc project không còn là bản nháp |
| `502` | Câu trả lời của AI không đọc được (JSON hỏng, câu rỗng, trích dữ kiện lạ); không lưu gì |
| `503` | Chưa cấu hình AI (viết, gợi ý) hoặc chưa nối đường đọc giọng (`speak`) |

## Đường đọc giọng (ADR-0040, ADR-0041)

```
authoring-service  ─ POST /internal/v1/projects/{id}/draft-voice ─▶ orchestrator
orchestrator       ─ POST /internal/v1/authoring/{id}/voice-lines/claim ─▶ authoring-service   (lấy câu chờ đọc, đánh dấu running)
orchestrator       ─ synthesize_lines (RabbitMQ, routing key tts, qua outbox) ─▶ tts
tts                ─ ghi /shared/{id}/audio/{line}_{cache_key}.wav ─ lines_synthesized ─▶ orchestrator (orchestrator.events)
orchestrator       ─ POST /internal/v1/authoring/{id}/voice-lines/results ─▶ authoring-service
```

- `claim` là `POST` (không phải `GET …?pending=1`) vì nó đổi trạng thái: trả `{"lines":[{line_id,text,language,voice_id,ref}]}`. `ref` là hash chữ của câu lúc gửi, được trả lại nguyên trong kết quả; kết quả có `ref` không còn khớp chữ hiện tại của câu (Creator vừa sửa) bị bỏ.
- `results`: `{"lines":[{line_id,ref,audio_path,duration_seconds,cache_key,cache_hit,error}]}` → `204`. Mỗi câu hoặc có file, hoặc có `error`; không có cả hai là `400`. `fail`: `{"reason":"…"}` → `204`, đánh `failed` các câu đang `running` (lệnh không tới được tts).
- Câu `running` quá 2 phút không có trả lời được gửi lại ở lần xin đọc kế tiếp.
- Lệnh `synthesize_lines`: envelope có `event_type` cấp trên cùng; `payload = {event_type:"synthesize_lines", lines:[{line_id,text,language,voice_id,ref}]}`. Trả lời `payload = {event_type:"lines_synthesized", lines:[{line_id,ref,audio_path,duration_seconds,cache_key,cache_hit,error}]}` (một mục mỗi câu; một câu hỏng mang `error` còn các câu khác vẫn được đọc), hoặc `{event_type:"lines_synthesis_failed", error_message}` khi cả lệnh hỏng. Lệnh bị dead-letter chỉ đánh hỏng các câu của nó, không chạm saga.
- Mã câu hợp lệ là `b<n>.<m>`; tts từ chối mã khác trước khi tạo tên file. Giọng rỗng dùng giọng mặc định của ngôn ngữ.
