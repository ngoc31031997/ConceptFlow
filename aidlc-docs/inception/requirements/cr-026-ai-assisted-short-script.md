# CR-026 — Kịch bản riêng cho bản Shorts/TikTok, hỗ trợ bằng AI (P1, tiếp nối CR-007)

## Date
2026-09-12

## Stage
Requirements Analysis (Change Request)

## Intent Analysis
- **Request type**: New Feature — thêm một cách để có script cho bản ngắn, không đổi cách hệ thống render/cắt clip; cộng thêm liên kết hiển thị hai project cùng chủ đề chung một màn hình (FR73, bổ sung 2026-09-12)
- **Scope estimate**: 2 unit — `web-gui` (2 lối vào script ngắn + ResultPage hiện cặp project liên kết), `orchestrator` (endpoint gọi Ollama, trường liên kết trên Project, tạo cặp project khi chọn "cả hai loại")
- **Complexity estimate**: Moderate — tái dùng gần như toàn bộ hạ tầng đã có (`self.clip()`, `generate_clips`, `video_output_mode`, `OllamaClient`), không đổi Saga/kiến trúc render; phần liên kết project là state mới nhưng không đụng saga hiện có

## Bối cảnh — vì sao CR-007 (bản follow-up) chưa đủ
Sau khi thêm `video_output_mode` (long/short/both), một project chọn "Short"
hoặc "Cả hai" vẫn dùng **đúng một script** cho cả hai bản. Video dọc chỉ là
`generate_clips` cắt một đoạn từ video 16:9 đã render — đúng quyết định D1 gốc
của CR-007 ("không sản xuất riêng video dọc").

Vấn đề Creator nêu ra khi dùng thử: cắt một đoạn từ video dài ra không tạo
được nội dung **hay** cho Shorts/TikTok — một đoạn giữa video dài thường thiếu
ngữ cảnh, không có hook riêng, không tự đứng được một mình. Cái Creator muốn là
bản ngắn có **kịch bản của riêng nó**: tự có hook, tự cô đọng, tự đứng được —
không phải một lát cắt của cái khác.

## Quyết định
**Không đổi kiến trúc render.** Bản ngắn vẫn đi qua đúng pipeline hiện có
(`validate_script → TTS → render_scenes → assemble_video → qc_video →
generate_clips`) và vẫn được cắt ra bởi `generate_clips` — chỉ khác là script
đưa vào được viết **để tự nó là một video ngắn hoàn chỉnh**, và luôn tự bọc
toàn bộ nội dung trong đúng một khối `with self.clip("short"):` (quy ước bắt
buộc của prompt/AI, không phải thay đổi ở `rendering`). Nhờ vậy toàn bộ hạ
tầng CR-007 (`ClipMarks`, `buildClipRequests`, `ClipsPanel`, preset
short/long) dùng lại nguyên vẹn — cái mới duy nhất là **cách có được nội dung
script đó**.

Hai lối, dùng song song (không loại trừ nhau):

1. **AI nội bộ (Ollama)** — bấm một nút, hệ thống tự soạn bản nháp script
   ngắn từ chủ đề/script dài đã có, hiện ra để Creator sửa trước khi nộp.
   Dùng lại đúng `OllamaClient` đã có cho `suggest-metadata` (CR-014), thêm
   một method mới cùng client.
2. **Copy-prompt cho AI ngoài** — một prompt mới trong `scriptPrompts.ts`
   (cùng họ với prompt sinh script dài đã có), Creator tự dán qua
   ChatGPT/Claude/Gemini rồi dán kết quả vào editor — đúng luồng đã quen từ
   script dài, không cần học thêm.

## Vấn đề với hướng "AI tự tóm tắt từ script dài" cần Creator xác nhận
Script dài là Manim code (animation, layout), không phải văn bản thuần —
"tóm tắt" nó không có nghĩa như tóm tắt một bài viết. Đề xuất: cả 2 lối trên
đều **soạn script ngắn mới từ chủ đề**, không cố gắng cắt/dịch code Manim của
script dài. Script dài chỉ dùng làm ngữ cảnh (đưa narration/chủ đề của nó vào
prompt) để bản ngắn nói cùng chủ đề, đúng tinh thần "cô đọng lại", không phải
"biến đổi code".

## Functional Requirements

### FR70 — Prompt sinh script ngắn (copy ra AI ngoài)
- **FR70.1**: Thêm `buildShortScriptPrompt(language, topic)` trong
  `scriptPrompts.ts`, cùng khuôn với `buildGenerationSystemPrompt` (API được
  phép dùng, quy tắc `self.narrate`, tự-kiểm-tra trước khi trả lời) nhưng:
  - Độ dài mục tiêu 30–60 giây lời thoại (khớp preset `short` của CR-007),
    không dùng ngân sách 6-8 phút của bản dài.
  - **Bắt buộc** bọc toàn bộ `construct()` trong đúng một
    `with self.clip("short"):` — nêu rõ đây là điều kiện để hệ thống nhận ra
    đây là clip, không phải gợi ý.
  - Cấu trúc bắt buộc: hook trong 2 giây đầu, một ý duy nhất, không hạ nhiệt
    ở giữa (khác bản dài vốn có nhịp nghỉ) — nhấn mạnh rule "one idea, no
    filler" trong prompt.
- **FR70.2**: `ScriptAssistant` (nơi có nút "Copy prompt" hiện tại) thêm một
  lựa chọn "Script dài" / "Script ngắn (Shorts/TikTok)" — chọn cái nào build
  đúng prompt đó. Không đổi luồng dán-kết-quả-vào-editor đã có.

### FR71 — Soạn bằng AI nội bộ (Ollama)
- **FR71.1**: Thêm `POST /v1/projects/{id}/suggest-short-script` (hoặc
  endpoint không cần project_id có sẵn — quyết định ở LLD) nhận `topic`
  (hoặc `script_content` dài đã có để rút chủ đề) và `language`, trả về một
  bản nháp script ngắn — cùng nội dung/ràng buộc như FR70.1, do
  `OllamaClient` sinh.
- **FR71.2**: Kết quả PHẢI được lint qua `lint_manim_script`/`ast.parse`
  trước khi trả về Creator — model nội bộ (đã từng lỗi tràn `num_ctx`, CR-014)
  không được tin tưởng mù; lỗi cú pháp phải báo rõ, không trả script hỏng.
- **FR71.3**: Bản nháp PHẢI hiện trong ô soạn thảo để Creator sửa tiếp trước
  khi nộp — không tự động submit thẳng vào saga. Cùng nguyên tắc với
  "Copy prompt": AI chỉ gợi ý, Creator quyết định bản cuối.

### FR72 — Nộp bản ngắn như một project riêng, có liên kết
- **FR72.1**: Script ngắn (từ FR70 hoặc FR71) nộp như một **project mới**,
  `video_output_mode = "short"` — không phải sửa project video dài đã có (hai
  bản chạy hai lượt saga độc lập, TTS/render riêng).
- **FR72.2**: Không cần validate gì thêm ngoài `validate_script` sẵn có —
  script tự bọc `self.clip("short")` đúng cú pháp sẽ tự nhiên có
  `ClipMarks` khi qua dry-run, không cần code mới ở `rendering`/`orchestrator`.

### FR73 — Hiện project cùng chủ đề chung một màn hình (bổ sung 2026-09-12)
Bug report: FR72.1 ban đầu để hai project là hai hàng độc lập trong danh sách
video — đúng object model nhưng sai trải nghiệm. Khi cùng một chủ đề sinh ra cả
bản dài lẫn bản ngắn riêng (dù chọn "cả hai loại" ngay từ đầu, hay sau này mới
vào một video dài có sẵn để tạo thêm bản ngắn), Creator muốn thấy **cả hai ở
Bước 5**, không phải lục lại danh sách video tìm project kia.

- **FR73.1**: `Project` PHẢI có một trường liên kết (`companion_project_id` —
  tên chính xác chốt ở LLD) trỏ sang project kia của cùng chủ đề, gán **hai
  chiều** lúc project ngắn được tạo:
  - Nếu tạo từ màn hình Result của một video dài có sẵn (dùng FR70/FR71 ngay
    tại đó) → liên kết với project dài đó.
  - Nếu chọn "cả hai loại" ngay từ Bước 1/2 (trước khi render) → orchestrator
    tạo hai project cùng lúc (một `long`, một `short` với script riêng từ
    FR70/FR71) và tự gán liên kết cho nhau khi khởi tạo.
- **FR73.2**: `GET /v1/projects/{id}` PHẢI trả kèm thông tin tối thiểu của
  project liên kết (id, status, video_path nếu có) — đủ để `ResultPage` gọi
  thêm một lượt fetch và hiện nó, không cần Creator tự đi tìm project_id.
- **FR73.3**: `ResultPage` (Bước 5) khi có `companion_project_id` PHẢI hiện cả
  hai bản trên cùng một trang — video dài + khu đăng YouTube, và video/clip
  ngắn + khu tải clip, xếp thành hai khối rõ ràng trên cùng màn hình. Trạng
  thái của bản kia (đang render/lỗi/sẵn sàng) cũng phải thấy được ngay, không
  phải đoán qua việc tự vào xem.
- **FR73.4**: Một project không có liên kết (mọi project trước CR-026, hoặc
  Creator chỉ tạo một loại mà không tạo thêm loại kia) hiện y như hiện tại —
  liên kết là optional, không đổi hành vi mặc định.

## Non-goals (out of scope cho CR-026)
- Không đổi `generate_clips`, `ClipsPanel`, `video_output_mode` — dùng nguyên
  trạng đã có từ CR-007 follow-up.
- Không tự động cắt/tóm tắt CODE của script dài — cả 2 lối đều sinh script
  ngắn MỚI, chỉ mượn chủ đề.
- Không thêm preset TikTok "long" (60-180s) vào phạm vi này — vẫn theo preset
  `short` (30-60s) đã có; nếu Creator gắn `self.clip("short")` thì cả hai
  preset đều tự cắt như CR-007 đã làm.
- Không tự publish lên Shorts/TikTok — vẫn tải tay như CR-007.

## Liên quan
- CR-007 (`cr-007-vertical-shorts-tiktok.md`) — hạ tầng cắt clip được dùng lại
  nguyên vẹn.
- CR-007 follow-up (`video_output_mode`, `ClipsPanel`) — project ngắn tạo ra ở
  đây dùng đúng field/UI đó.
- CR-014 (`OllamaClient`, num_ctx overflow) — bài học tràn context áp dụng lại
  cho FR71.
- `scriptPrompts.ts` (CR-nào tạo `buildGenerationSystemPrompt`) — khuôn mẫu
  cho `buildShortScriptPrompt`.
