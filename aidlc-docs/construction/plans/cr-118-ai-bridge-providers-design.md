# CR-118 · Thiết kế — AI Bridge chỉ qua hàng đợi, DeepSeek / Gemini / Grok

**Trạng thái:** chờ Creator duyệt · **Yêu cầu:** [cr-118-ai-bridge-providers.md](../../inception/requirements/cr-118-ai-bridge-providers.md) · **Quyết định:** [ADR-0051](../../decisions/ADR-0051-ai-bridge-queue-only-multi-provider.md)

Không đổi service backend nào. Thay đổi nằm ở `tools/ai-bridge-extension` và `services/web-gui`.

## 1. Luồng sau CR

```
ConceptFlow (trang)                    Extension worker                         Trang nhà cung cấp
───────────────────                    ────────────────                         ──────────────────
describe request  ──queue_add──────►   lưu mục (provider=null|id)
                  ◄─queue_added{item_id}
                                       tới lượt: provider = item.provider ?? mặc định
                                       POST API  X-External-AI: prompt ─► prompt
                                       mở tab chat mới ──────────────────────►  gõ, gửi, chờ xong,
                                                                                bấm Copy → câu trả lời
                                       POST API  X-External-AI: paste{reply}
                  ◄─queue_item{item_id,status,stage,provider,error,reply}  (gửi mọi tab ConceptFlow)
màn hình tải lại / hộp thoại đóng
```

Mục bước 3 (loại `prompt`): bỏ bước "POST prompt", prompt lấy từ mục; bước paste gửi `{raw: reply}` tới `POST /v1/projects/{id}/research/paste`.

## 2. Extension

### 2.1 Bảng nhà cung cấp — `src/providers.js` (mới)

| id | Tên | Mở chat mới | Host | Chế độ |
|---|---|---|---|---|
| `deepseek` | DeepSeek | `https://chat.deepseek.com/` | `chat.deepseek.com` | DeepThink, Search theo mục |
| `gemini` | Gemini | `https://gemini.google.com/app` | `gemini.google.com` | không đổi |
| `grok` | Grok | `https://grok.com/` | `grok.com` | không đổi |

### 2.2 Adapter

- `src/adapter/core.js` (mới): phần chung lấy từ `src/deepseek/adapter.js` hiện có — chờ ô nhập, gõ (`execCommand insertText`, đã chạy được với contenteditable), bấm gửi, chờ xong (poll + hết giờ 15 phút), bấm Copy dưới câu trả lời cuối, nhận văn bản qua `clipboard-hook.js` (MAIN world), báo tiến độ / lỗi về worker.
- Mỗi nhà cung cấp một thư mục `src/providers/<id>/` gồm `selectors.js` (selectors mặc định, ghi đè được ở Cài đặt) và `adapter.js` (khai báo: bật chế độ nếu có, hàm `isDone()`).
- `src/deepseek/*` chuyển vào `src/providers/deepseek/`, giữ nguyên hành vi.

Selectors (lấy từ HTML Creator gửi 2026-11-20; chọn theo tên thẻ / `data-testid`, không theo chữ hiển thị):

| Vai trò | Gemini | Grok |
|---|---|---|
| Ô nhập | `rich-textarea .ql-editor[contenteditable="true"]` | `[data-testid="chat-input"] .ProseMirror[contenteditable="true"]` |
| Nút gửi | `[data-test-id="send-button-container"] .send-button.submit button` | `button[data-testid="chat-submit"]` |
| Lượt trả lời cuối | `model-response` (cuối) | `[data-testid="assistant-message"]` (cuối) |
| Đã xong | lượt cuối có `.response-footer.complete` và `message-content [aria-busy="false"]` | lượt cuối có `[data-testid="response-footer-actions"]` |
| Nút Copy | `message-actions copy-button button` trong lượt cuối | `[data-testid="response-footer-actions"] button[aria-label="Copy response"]` |

"Đã xong" chỉ được tính khi lượt cuối là lượt **mới** (số lượt trả lời tăng so với lúc gửi). Không tìm thấy ô nhập sau 30 giây → lỗi `not_logged_in` (chưa đăng nhập hoặc giao diện đổi), nêu tên nhà cung cấp.

Grok: nhãn `aria-label="Copy response"` là tiếng Anh vì giao diện Grok của Creator đang tiếng Anh; nếu đổi ngôn ngữ thì sửa ở Cài đặt. Đây là selector duy nhất phụ thuộc chữ.

### 2.3 Hàng đợi — `src/queue.js`

Mục thêm:
- `provider`: `null` | `"deepseek"` | `"gemini"` | `"grok"`.
- `kind`: `"request"` (như hiện nay) | `"prompt"` (bước 3: `prompt` cố định + `request` đọc kết quả, body `{raw}`).
- `ranOn`: nhà cung cấp thật của lần chạy gần nhất (để hiện sau khi xong/lỗi).

Hàm mới: `setProvider(state, id, provider)` (chỉ khi `waiting`/`failed`), `providerOf(item, defaultId)`.
Gộp trùng: giữ luật cũ (cùng request đang mở thì không thêm lại; trả `item_id` cũ).

### 2.4 Worker — `src/background.js`

- Bỏ nhánh `run` trực tiếp; `jobs.js` chỉ còn job của mục hàng đợi.
- Đọc mặc định từ `chrome.storage.local.defaultProvider` (thiếu → `deepseek`).
- Mỗi lần trạng thái mục đổi, gửi `queue_item` tới mọi tab ConceptFlow (thay `queue_item_done`), kèm `queue_state` như cũ.
- Mở tab theo URL của nhà cung cấp; tin nhắn từ adapter chỉ được nhận khi tab đúng là tab của job (giữ `belongsTo`).

### 2.5 Giao thức trang ↔ extension — `src/protocol.js`, `docs/contracts/ai-bridge-v2.md`

`v: 2`. Trang → extension: `ping`, `queue_add{item}`, `cancel{item_id}`. Extension → trang: `pong{version, providers, default_provider}`, `queue_added{request_id,item_id,added}`, `queue_state{…}`, `queue_item{item_id, path, project, status, stage, provider, error, reply}`, `error`. Bỏ `run`, `progress`, `result`.
Thông báo lỗi dùng tên nhà cung cấp của mục ("Chưa đăng nhập Gemini…").

### 2.6 Popup, Cài đặt, manifest

- Popup: mỗi mục có ô chọn "Mặc định (DeepSeek) / DeepSeek / Gemini / Grok" (khoá khi đang chạy); dòng mục ghi nhà cung cấp đang/đã chạy. Tiêu đề "Hàng đợi AI".
- Cài đặt: ô "Nhà cung cấp mặc định"; phần ghi đè selector và "Kiểm tra" tách theo từng nhà cung cấp.
- `manifest.json`: thêm host và content script cho `gemini.google.com`, `grok.com`; version `2.0.0`; tên mô tả không còn chỉ DeepSeek.

## 3. web-gui

- `hooks/useAiQueue.ts` (thay `useDeepSeekQueue`): `add(item) → {item_id}`, `counts`, `onItemDone(path)` như cũ, thêm theo dõi một mục: `watch(item_id) → {status, stage, provider, error, reply}`.
- `hooks/useQueuedExternalRun.ts` (thay `useDeepSeekRun`): nhận `run(external)` của bước như hiện nay, gọi nó ở chế độ `describe` để có request, thêm vào hàng đợi, rồi theo dõi mục: xong → `onDone()` (màn hình tải lại); lỗi → `onUnreadable(reply, error)`.
  - `onDone` không còn nhận kết quả của request (câu trả lời được đọc ở worker). Các màn hình đang dùng `result` sẽ chuyển sang tải lại dữ liệu; danh sách cụ thể kiểm khi code (`graphify affected`).
- `ExternalAiDialog`: nút "Chạy bằng DeepSeek" → **"Thêm vào hàng đợi AI"**; sau khi thêm, hộp thoại hiện trạng thái mục ("Đang chờ · 2 mục trước" / "Đang chạy trên Gemini · đang trả lời…" / lỗi + câu trả lời trong ô dán). Đóng hộp thoại không huỷ mục. Chép/dán tay vẫn giữ.
- Các chỗ dùng `useDeepSeekRun` (`ExternalAiButton`, `ExternalRedoDeepSeek`, `QuestionsCard`, `StoryCard`, `ScriptStepPage`) chuyển sang `useQueuedExternalRun`; các chỗ dùng `useDeepSeekQueue` (`BeatSheetCard`, `PlanHeadCard`, `SequenceRunCard`, `LockHandoffTab`, `VisualHeadCard`, `ScriptWriteStepPage`) đổi tên hook và chữ.
- `ResearchStepPage`: "Chạy bằng DeepSeek" → thêm mục loại `prompt` (prompt ở ô ②, paste tới `/research/paste`); xong thì tải lại bảng dữ kiện; lỗi thì đưa câu trả lời vào ô ③. Chế độ DeepThink + Search giữ cho DeepSeek.
- `AiBridgeRunStatus` hiện trạng thái mục thay cho stage của lượt trực tiếp.
- `useAiBridge.ts` chỉ còn phần dùng chung (`post`, `readExtensionMessage`, `newRequestId`, phát hiện extension + version). Extension `v: 1` → dòng "Cập nhật extension ConceptFlow AI Bridge lên bản mới".
- Mọi chữ "DeepSeek" trên giao diện → "AI ngoài" / tên nhà cung cấp thật của mục.

## 4. Test

- Extension (node): `queue.test.js` (provider, kind `prompt`, setProvider, gộp trùng), `protocol.test.js` (v2), `selectors.test.js` + test mới cho adapter core với DOM giả của từng nhà cung cấp (dựng từ HTML Creator gửi, rút gọn), `answer.test.js`.
- web-gui: `fakeAiBridge.ts` thành extension giả v2 có hàng đợi; test cho `useQueuedExternalRun`, `ExternalAiDialog`, `ResearchStepPage`; sửa các test đang dùng `run`.
- Không gọi DeepSeek/Gemini/Grok thật trong test. Chạy thật: Creator cài lại extension và thử mỗi nhà cung cấp một mục (mình chuẩn bị danh sách kiểm).

## 5. Rebuild & tài liệu

- Rebuild `web-gui` (`scripts/worktree.sh rebuild web-gui`). Extension không chạy trong Docker: Creator bấm ↻ ở `chrome://extensions`.
- README extension viết lại (3 nhà cung cấp, chỉ hàng đợi, Cài đặt mặc định, đổi theo mục). Contract v2 thay v1 trong `docs/contracts/README.md`.

## 6. Thứ tự làm

1. Extension: providers + adapter core + chuyển DeepSeek (không đổi hành vi) + test.
2. Extension: Gemini, Grok, hàng đợi (provider, kind `prompt`), giao thức v2, popup, Cài đặt, manifest.
3. web-gui: hook mới, hộp thoại, các màn hình, bước 3, test.
4. Contract, README, rebuild web-gui, báo Creator thử thật.
