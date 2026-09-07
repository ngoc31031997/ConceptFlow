# CR-008 — Ngôn ngữ nội dung: chọn một lần, áp dụng toàn pipeline (P1)

## Date
2026-09-07

## Stage
Requirements Analysis (Change Request)

## Intent Analysis
Creator muốn làm kênh **tiếng Anh hướng tới khán giả nước ngoài**. Kỳ vọng: chọn ngôn ngữ một lần lúc tạo project, rồi **mọi thứ liên quan tới nội dung** đều ra đúng ngôn ngữ đó — giọng đọc, phụ đề, tiêu đề, mô tả, tags, prompt thumbnail, script mẫu.

Hiện tại điều này **không xảy ra**. `voice_language` chỉ điều khiển đúng 2 thứ; mọi phần sinh nội dung khác đều hardcode tiếng Việt.

## Vấn đề (đã xác minh trong code)

### 1. `voice_language` có phạm vi rất hẹp
`domain.VoiceLanguage` (`vi` | `en`) hiện chỉ ảnh hưởng:
- `voice_registry.py` — chọn model giọng Piper.
- `domain.EstimateNarrationDuration` — tốc độ đọc ước lượng (150 wpm en / 140 wpm vi).

Ngoài 2 chỗ đó, ngôn ngữ **không được truyền đi đâu cả**.

### 2. System prompt SEO hardcode tiếng Việt — và không hề nhận ngôn ngữ
`services/orchestrator/internal/adapters/llm/ollama_client.go::Suggest` yêu cầu model:
```
- "title": ... tiếng Việt.
- "description": ... tiếng Việt.
- "tags": ... (tiếng Việt, không dấu #)
```
Chữ "tiếng Việt" xuất hiện **3 lần**, và chữ ký hàm là `Suggest(ctx, scriptContent, categoryHint)` — **không có tham số ngôn ngữ**. `SuggestPublishMetadataUseCase.Execute` có sẵn `project` trong tay (đã đọc từ repo) nhưng chỉ truyền `ScriptContent` và `CategoryHint`.

**Hệ quả trực tiếp**: chọn giọng tiếng Anh, đọc tiếng Anh, phụ đề tiếng Anh — nhưng tiêu đề, mô tả và tags YouTube vẫn ra **tiếng Việt**. Đây là lỗi chặn hoàn toàn use case kênh tiếng Anh.

### 3. Bug cắt tiêu đề theo byte (phát hiện kèm theo)
`ollama_client.go`:
```go
if len(title) > maxTitleLength {
    title = title[:maxTitleLength]
}
```
Trong Go, `len()` trên `string` đếm **byte**, và `title[:100]` cắt theo **byte**. Ký tự tiếng Việt có dấu chiếm 2–3 byte UTF-8, nên một tiêu đề dài bị cắt **giữa một ký tự**, tạo ra chuỗi UTF-8 hỏng gửi thẳng lên YouTube API.

Lỗi này chưa lộ ra vì (a) tiêu đề hiếm khi vượt 100 ký tự, (b) với tiếng Anh thuần ASCII thì byte = ký tự nên vô hại. Nó sẽ càng dễ gặp khi hỗ trợ thêm ngôn ngữ. YouTube giới hạn tiêu đề theo **ký tự**, không phải byte, nên `len()` cũng đang đo sai đại lượng.

### 4. Prompt sinh thumbnail viết bằng tiếng Việt
`services/web-gui/src/components/ThumbnailUpload.tsx` — toàn bộ phần hướng dẫn là tiếng Việt, ví dụ minh hoạ cũng là chủ đề tiếng Việt.

Lưu ý: việc prompt **xuất ra tiếng Anh** là **đúng và phải giữ nguyên** — model sinh ảnh cho kết quả tốt hơn hẳn với prompt tiếng Anh, và nhận xét này đã được ghi ngay trong file. Vấn đề chỉ nằm ở phần hướng dẫn/ví dụ dẫn dắt theo hướng nội dung tiếng Việt.

### 5. Script mẫu chỉ có tiếng Việt
`ScriptEditor.tsx::SCRIPT_TEMPLATE` — narration trong template là tiếng Việt. Creator làm kênh tiếng Anh bấm "Dùng script mẫu" sẽ nhận một script phải dịch lại toàn bộ.

### 6. Giao diện hardcode tiếng Việt
17 file `.tsx`/`.ts` chứa chuỗi tiếng Việt cứng, không có lớp i18n nào.

## Quyết định thiết kế then chốt: TÁCH ngôn ngữ nội dung khỏi ngôn ngữ giao diện

Đây là điểm dễ làm sai nhất của CR này.

| | Ngôn ngữ **nội dung** (`content_language`) | Ngôn ngữ **giao diện** (`ui_language`) |
|---|---|---|
| Phạm vi | Mỗi project | Mỗi Creator (toàn app) |
| Điều khiển | Giọng đọc, phụ đề, tiêu đề/mô tả/tags, prompt thumbnail, chapters, script mẫu | Nhãn nút, thông báo lỗi, hướng dẫn |
| Ví dụ | `en` — kênh hướng tới khán giả nước ngoài | `vi` — Creator là người Việt |

**Creator người Việt làm kênh tiếng Anh cần `ui_language=vi` + `content_language=en`.** Gộp hai thứ này làm một sẽ ép Creator dùng giao diện tiếng Anh chỉ để xuất nội dung tiếng Anh — sai yêu cầu.

**Phạm vi CR này: chỉ `content_language`.** i18n giao diện là việc riêng, khối lượng lớn hơn nhiều (17 file + hạ tầng i18n), và **không chặn** mục tiêu kênh tiếng Anh.

## Functional Requirements

### FR21 — Ngôn ngữ nội dung (mới)
- **FR21.1**: Project PHẢI có trường `content_language`, Creator chọn ở bước tạo video. `voice_language` hiện tại được **đổi tên/nâng cấp** thành trường này (nó đã mang đúng ngữ nghĩa, chỉ là phạm vi áp dụng quá hẹp).
- **FR21.2**: `content_language` PHẢI điều khiển toàn bộ: chọn giọng TTS, tốc độ đọc ước lượng, ngôn ngữ tiêu đề/mô tả/tags, prompt thumbnail, và (khi CR-006 xong) tiêu đề chapter.
- **FR21.3**: `LLMSuggesterPort.Suggest` PHẢI nhận thêm `language`; prompt SEO dựng động theo ngôn ngữ đó thay vì hardcode "tiếng Việt".
- **FR21.4**: PHẢI có script mẫu riêng cho từng ngôn ngữ; nút "Dùng script mẫu" chọn theo `content_language` đang chọn.
- **FR21.5**: Prompt sinh thumbnail PHẢI dựng động theo `content_language` (phần hướng dẫn), nhưng **vẫn luôn xuất prompt ảnh bằng tiếng Anh** — đây là chủ ý, không phải thiếu sót.
- **FR21.6**: Thêm ngôn ngữ mới PHẢI chỉ cần thêm dữ liệu (giọng + wpm + mẫu prompt), không phải sửa logic rải rác. Hiện tại `wordsPerMinuteVietnamese`/`wordsPerMinuteEnglish` là hằng số riêng lẻ với một nhánh `if` — không mở rộng được.

### FR18 — Sửa lỗi kèm theo
- **FR18.3 (mới)**: Cắt tiêu đề PHẢI theo **ký tự (rune)**, không theo byte — `[]rune(title)[:100]`, để không sinh UTF-8 hỏng và để đo đúng đại lượng YouTube giới hạn.

## Ràng buộc
- **C1 — Giọng tiếng Anh sẵn có tốt hơn tiếng Việt**: Piper có `en_US-ryan-high` (chất lượng `high`), trong khi giọng Việt cao nhất chỉ `medium` (CR-001 §C1). Nghĩa là **kênh tiếng Anh hiện đã có lợi thế chất lượng giọng**, và rủi ro monetization vì giọng máy (CR-005) thấp hơn — nhưng vẫn nên làm CR-005.
- **C2 — Tương thích ngược**: project cũ chỉ có `voice_language`; mặc định `content_language` = giá trị đó.
- **C3 — Phụ đề**: font `DejaVu Sans` trong `subtitle_file.py` phủ được cả tiếng Việt lẫn tiếng Anh, không cần đổi. Ngôn ngữ khác (CJK, Ả Rập) sẽ cần font khác — ngoài phạm vi.
- **C4 — Chất lượng model SEO**: prompt tiếng Anh do model Ollama local nhỏ sinh ra có thể yếu. Liên quan CR-006 FR18.1 (cho phép cấu hình model mạnh hơn).
- **C5 — KHÔNG dịch narration**: hệ thống không tự dịch script. Creator viết narration bằng ngôn ngữ nào thì `content_language` phải khớp ngôn ngữ đó. Nên **cảnh báo** nếu phát hiện lệch, không tự động dịch (dịch máy narration sẽ hạ chất lượng và tăng rủi ro monetization).

## Phạm vi tác động
- **Orchestrator** (Unit 8): `domain.Project.ContentLanguage`, `LLMSuggesterPort.Suggest` + prompt động, sửa lỗi cắt rune, bảng wpm thay cho hằng số rời.
- **Web GUI** (Unit 10): chọn ngôn ngữ nội dung, script mẫu theo ngôn ngữ, prompt thumbnail động.
- **TTS** (Unit 3): không đổi logic — đã lọc giọng theo `language`.
- **API Gateway** (Unit 9): forward trường mới.

## Tiêu chí nghiệm thu
1. Tạo project `content_language=en`, script narration tiếng Anh → tiêu đề, mô tả, tags sinh ra **bằng tiếng Anh**.
2. Cùng project → giọng đọc tiếng Anh, phụ đề tiếng Anh.
3. Nút "Dùng script mẫu" cho ra template tiếng Anh.
4. Tiêu đề dài > 100 ký tự có dấu bị cắt vẫn là UTF-8 hợp lệ, và dài đúng 100 **ký tự**.
5. Giao diện vẫn **tiếng Việt** (chứng minh 2 trục ngôn ngữ độc lập).

## Câu hỏi còn mở
1. Có làm i18n giao diện (`ui_language`) trong CR riêng không, hay giữ giao diện tiếng Việt vô thời hạn?
2. Có cảnh báo khi ngôn ngữ narration trong script lệch với `content_language` không (§C5)? Phát hiện bằng heuristic đơn giản (dấu tiếng Việt) hay bỏ qua?
3. Ngoài `vi`/`en` có định mở thêm ngôn ngữ nào không? Ảnh hưởng mức độ đầu tư cho FR21.6.
