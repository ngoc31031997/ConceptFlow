# Hợp đồng `ai-runs.v1` · lượt chạy AI có xem trước

Quyết định: [ADR-0052](../../aidlc-docs/decisions/ADR-0052-ai-runs-preview-before-apply.md).

Mỗi lượt AI một-prompt của bước 1–11 chạy qua một **lượt chạy**. Lượt chạy giữ chính request AI của bước (method, path, body như web-gui gửi) và phát lại request đó bên trong service sở hữu bước:

| Bước | Service | Gốc route |
|---|---|---|
| 1–7 | authoring-service | `/v1/projects/{id}/ai-runs` |
| 8–11 | visual-service | `/v1/projects/{id}/visual/ai-runs` |

Request được phát lại ở bốn chế độ `ExternalAIRun` (`application/external_ai.go` của mỗi service):

| Chế độ | Dùng khi | Luồng của bước |
|---|---|---|
| `prompt` | bắt đầu lượt | dựng prompt, không gọi model, không ghi |
| `capture` | kênh `api`, chạy nền | gọi model một lần, giữ câu trả lời, dừng ở `stopForPreview` |
| `preview` | câu trả lời từ extension hoặc dán tay | đọc câu trả lời, dừng ở `stopForPreview` |
| `paste` | áp dụng | đọc câu trả lời và lưu như thường |

`stopForPreview(ctx, v)` là điểm mỗi luồng gọi **ngay sau khi đọc và kiểm tra câu trả lời, trước mọi lần ghi** (DB, rendering, service khác). `v` là thứ sẽ được lưu, dạng JSON; nó thành `preview` của lượt. Một luồng chưa gọi `stopForPreview` sẽ chạy hết và lưu luôn ở chế độ `preview`/`capture`; lượt khi đó báo `failed` "bước này chưa dừng được để xem trước kết quả", nên mọi luồng chạy qua lượt chạy phải có điểm dừng này.

## Route

| Method | Path | Body | Trả |
|---|---|---|---|
| POST | `` | `{label, method, path, body, channel}` | `201` lượt mới. `path` phải bắt đầu bằng `/v1/projects/{id}/`, `method` là `POST`/`PUT`, `body` là JSON (chuỗi). Request không gọi AI → `400`. Bước từ chối (chưa sẵn sàng, đang chạy…) → status và thân lỗi của chính bước. Lượt còn mở cùng `path` chuyển `cancelled`. `channel = api`: `status: running`, model gọi nền; còn lại `waiting`. |
| GET | `` | | `{runs: [...]}`: lượt mới nhất của mỗi `path` |
| GET | `/{rid}` | | một lượt; `404` khi không có |
| POST | `/{rid}/reply` | `{raw}` | câu trả lời (extension mục `kind: "prompt"`, hoặc ô dán, hoặc bản sửa). Chỉ khi `waiting`, `ready`, `unreadable`, `failed`; khác → `409 code ai_run_state`. Trả `200` với lượt `ready` (có `preview`) hoặc `unreadable` (có `error`). |
| POST | `/{rid}/apply` | | chỉ khi `ready`. Phát lại ở `paste`; trả `{run, state}`, `state` là thân trả về của request bước. Bước từ chối → status, thân của bước; lượt về `unreadable`. |
| POST | `/{rid}/cancel` | | bỏ lượt chưa áp dụng |

Một lượt: `{id, project_id, label, channel, status, method, path, body, prompt, reply, preview, error, error_detail, created_at, updated_at}`. `error_detail` là `null` trừ khi bước trả lỗi có cấu trúc (xem "Vẽ tài sản bước 11"). `status`: `waiting`, `running`, `ready`, `unreadable`, `failed`, `applied`, `cancelled`. Lượt `running` khi service khởi động lại chuyển `failed`. Bảng `ai_runs` ở DB của mỗi service, xoá theo dự án.

## Vẽ tài sản bước 11

Vẽ, vẽ lại kèm ghi chú và gửi lại kèm lỗi của một tài sản (`POST .../storyboard/assets/{aid}/draw`, `/redraw`, `/fix`, xem [`storyboard-v1.md`](storyboard-v1.md)) là lượt AI một-prompt chạy qua lượt chạy ở visual-service, trên kênh mặc định `visual_storyboard`.

- Kênh `api`: model được gọi **đúng một lần**, không thử lại. Kênh extension và dán tay: cùng luồng `prompt` → `reply` → `apply`.
- `preview` của các lượt này là `{code, png_base64, warnings}` (`png_base64` là PNG dựng thử, `warnings` là `[{message, line}]`). Chưa lưu gì (không file, không đổi tài sản) cho tới khi `apply`; `apply` kiểm và dựng lại rồi mới lưu.
- Code AI trả không qua kiểm (không có code, sai tên component, lỗi cú pháp, bộ dựng từ chối, vi phạm kit) → lượt `unreadable` ở **mọi** kênh, kể cả `api`; không có lần gọi lại tự động. Lượt mang lỗi có cấu trúc trong `error_detail`: `{kind: "no_code"|"wrong_name"|"syntax"|"render"|"kit", message, line, snippet}` (`line`, `snippet` có thể `null`), còn `error` là `message`. Creator chọn tiếp: gửi lại kèm lỗi (một lượt mới cùng kênh, prompt chứa code hỏng và lỗi có số dòng), vẽ lại kèm ghi chú, hoặc bỏ (`cancel`).
- Lỗi của chính lần gọi API (hết số dư, bị giới hạn tốc độ, reply rỗng) giữ thông báo cũ và lượt `failed`; không có `error_detail`.
- "Vẽ tất cả" mở mỗi tài sản một lượt; lượt kế chỉ mở khi lượt trước `applied` hoặc `cancelled`.
- Luồng tài sản không còn vòng tự sửa 3 lượt; Thư viện hình vẫn giữ vòng đó và không đi qua lượt chạy.

## Kênh mặc định

llm-service `GET /v1/settings/ai-channel` (xem [`prompt-library-v1.md`](prompt-library-v1.md)): khoá bước `idea`, `brief`, `research`, `story`, `beats`, `script`, `lock`, `visual_concept`, `visual_plan`, `visual_sequence`, `visual_storyboard`.

## Web-gui

- `AiRunsProvider({projectId, step, service})` bọc màn hình của bước (`step` là khoá kênh ở trên, `service` là `authoring` hoặc `visual`).
- `AiRunDialog({open, onClose, title, runPath, describe, noNote?, impact?, renderPreview?, onApplied, testId})`: `describe(note)` dựng request bằng `describeAiRequest((external) => <hàm client của bước>(…, note, { external }))`; `runPath` là `path` mà `describe` trả về; `onApplied(state)` nạp lại màn hình.
- `AiRunPill({runPath, onOpen, testId})` đặt cạnh nút mở hộp thoại.
- `ChannelButton` cho nút chạy hàng loạt (tự lưu từng mục, không xem trước).

Chạy hàng loạt mà mỗi mục đọc kết quả **đã lưu** của mục trước (mọi unit, mọi tài sản chưa có chi tiết…) vẫn lưu ngay từng mục; nó không đi qua lượt chạy.
