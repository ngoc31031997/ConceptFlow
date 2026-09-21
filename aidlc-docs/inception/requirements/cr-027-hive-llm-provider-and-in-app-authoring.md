# CR-027 — Gọi LLM ngay trong app qua Hive (OpenAI-compatible), bỏ vòng copy-paste ra AI ngoài (P1)

## Date
2026-09-21

## Stage
Requirements Analysis (Change Request)

## Intent Analysis
- **Request type**: New Feature + Refactor — thêm một provider LLM trả phí
  (Hive) và một đường chạy prompt **trong app**; đồng thời rút phần điền biến
  prompt từ `web-gui` về `orchestrator` để GUI và server dùng chung đúng một
  prompt.
- **Scope estimate**: 2 unit — `orchestrator` (provider port + Hive adapter +
  prompt rendering + use case chạy từng bước + bảng đo token) và `web-gui`
  (nút "Chạy bằng AI" ở 5 chỗ + màn theo dõi token).
- **Complexity estimate**: Moderate–High. Không đụng Saga, không đụng render,
  không đụng TTS — nhưng chạm vào đường soạn kịch bản mà Creator dùng mỗi lần
  tạo video, và lần đầu tiên đưa một dịch vụ **tính tiền theo lượt gọi** vào
  pipeline. Rủi ro nằm ở chất lượng đầu ra và ở ví tiền, không ở kiến trúc.

## Bối cảnh — vì sao CR-025 chưa đủ
CR-025 dựng xong pipeline soạn kịch bản 4 bước (Story Architect → Visual
Director → Manim Engineer → Script Reviewer), prompt của từng vai trò nằm
trong bảng `prompt_templates` và sửa được qua màn admin. Nhưng **cách chạy** là
copy tay: GUI render prompt → Creator bấm Copy → dán sang ChatGPT/Claude/Gemini
→ copy kết quả về → dán vào ô soạn thảo → `POST .../authoring/{step}`.

Đúng như mục "Rủi ro" của CR-025 đã lường trước: *"Bốn bước copy tay là bốn lần
Creator có thể bỏ giữa."* Thực tế dùng cho thấy chính xác điều đó — mỗi video
là 4–5 lần chuyển cửa sổ, và prompt bước sau phải mang theo output bước trước
nên càng về cuối càng dài, càng dễ dán thiếu.

Lý do CR-025 chọn copy tay được ghi rõ: *"Không tự gọi API trả phí"*, và Ollama
nội bộ không gánh nổi việc này — `OllamaClient` hiện phải cắt script xuống
`maxScriptChars = 4000` vì `num_ctx` mặc định 2048 (bài học CR-014: project
17.5k ký tự làm model trả về JSON rỗng). Storyboard + code Manim của một video
10 phút vượt xa ngưỡng đó.

**Cái đã thay đổi**: Hive cung cấp API OpenAI-compatible với context 1M token ở
mức giá flash. Ràng buộc kỹ thuật khiến CR-025 phải chọn copy tay không còn,
nên quyết định "không tự gọi API trả phí" được mở lại ở CR này.

## Hive API — dữ kiện đã xác minh
Nguồn: https://docs.thehive.ai/docs/chat-completions-openai-compatible-llms

| Mục | Giá trị |
|---|---|
| Endpoint | `POST https://api-cdn.thehive.ai/api/v3/chat/completions` (vùng Virginia: `https://api-va1.thehive.ai/api/v3/chat/completions`) |
| Base URL | `https://api-cdn.thehive.ai/api/v3` |
| Auth | `Authorization: Bearer <API key>` (tạo ở mục Service API Keys trên dashboard Hive) |
| Định dạng | OpenAI-compatible chat completions — request/response y hệt OpenAI |
| Model | `deepseek-ai/deepseek-v4.1-flash` (text + ảnh), `zai-org/glm-5.3-flash` (text + ảnh + video) |
| Context | 1M token cả hai |
| Prompt caching | Có |
| Rate limit | **5 request/giây** (mặc định) |
| Điều kiện | Số dư tài khoản phải dương |

Hệ quả cho thiết kế: **không cần SDK** — `net/http` + `encoding/json` là đủ,
đúng khuôn `OllamaClient` đã có. Tên model PHẢI xác nhận lại trên dashboard
trước khi chốt giá trị mặc định ở LLD (doc có thể đi sau bảng model thật).

## Quyết định đã chốt (Creator, 2026-09-21)

**D1 — Hive là provider chính, Ollama là fallback.**
Thêm một port `LLMProviderPort` trong `application`, hai adapter cùng
implement: `HiveClient` (mới) và `OllamaClient` (đã có, bọc lại). Provider chọn
qua env. Hive lỗi/hết số dư/mất mạng → tự rơi về Ollama cho **các tác vụ nhẹ**
(`suggest-metadata`, `suggest-short-script`). Các bước soạn kịch bản context
lớn **không fallback** — trả lỗi rõ ràng và giữ nguyên đường copy tay, vì một
bản nháp do model 2048-token sinh ra còn tệ hơn không có gì (CR-014).

**D2 — Cả 5 vai trò đều chạy được bằng AI.**
`story_architect`, `visual_director`, `manim_engineer`, `script_reviewer`,
`remotion_engineer`. Hai vai trò sinh code là chỗ rủi ro nhất, nên chúng — và
chỉ chúng — bắt buộc đi qua vòng lint-và-sửa ở FR81.

**D3 — Chạy từng bước, Creator duyệt giữa.**
Mỗi tab pipeline có nút "Chạy bằng AI" riêng. Không có nút chạy liền 4 bước ở
CR này. Lý do: một bước sai mà cứ chạy tiếp là tốn token cho ba bước vô ích, và
UX 4 tab hiện có vốn đã là chỗ để Creator xem/sửa giữa chừng — giữ nguyên nó
thì không phải dựng job chạy nền, không phải dựng theo dõi tiến độ, không phải
nghĩ cách huỷ giữa chừng. Nút chạy liền mạch để lại thành CR sau, khi đã có số
liệu về tỉ lệ một bước phải chạy lại.

**D4 — Key trong `.env`, số token hiện trên giao diện web.**
Key đi theo đúng khuôn `AZURE_SPEECH_KEY` đã có. Khác với đề xuất ban đầu (log
vào Loki): Creator muốn **xem ngay trên web**, nên mỗi lượt gọi được ghi vào
một bảng `llm_usage` và hiện thành một màn theo dõi trong mục Cài đặt. Không
đặt hạn mức chặn cứng ở CR này — đo trước, chặn sau (cùng logic `QC_ENFORCE`
của CR-021: bật cổng trước khi biết con số thật là cách làm mất niềm tin).

## Functional Requirements

### FR76 — Lớp provider LLM dùng chung
- **FR76.1**: Định nghĩa `LLMProviderPort` trong `internal/application` với
  một method chat-completion nhận `(system, user string, opts)` và trả
  `(content string, usage TokenUsage, err error)`. `TokenUsage` PHẢI mang
  prompt tokens, completion tokens và tên model — đây là đầu vào của FR84.
- **FR76.2**: `internal/adapters/llm/hive_client.go` implement port đó bằng
  `net/http` thuần, không SDK, đúng khuôn `ollama_client.go` (cùng package).
- **FR76.3**: `OllamaClient` hiện có PHẢI được bọc để cũng implement
  `LLMProviderPort`, **không thay đổi hành vi** của `suggest-metadata` và
  `suggest-short-script` đang chạy. Hai port cũ (`MetadataSuggesterPort`,
  `ShortScriptSuggesterPort`) giữ nguyên chữ ký — CR này không refactor chúng.
- **FR76.4**: Cấu hình qua env, đặt mặc định trong `config.go` đúng khuôn các
  biến `OLLAMA_*`:
  - `LLM_PROVIDER` (`hive` | `ollama`, mặc định `hive` khi có key, ngược lại `ollama`)
  - `HIVE_API_KEY` (rỗng ⇒ provider tự về `ollama`, **không** crash lúc khởi động)
  - `HIVE_BASE_URL` (mặc định `https://api-cdn.thehive.ai/api/v3`)
  - `HIVE_MODEL` (mặc định chốt ở LLD sau khi xác nhận trên dashboard)
  - `HIVE_TIMEOUT_SECONDS` (mặc định 180 — dài hơn 120 của Ollama vì bước sinh
    code Manim trả về hàng trăm dòng)
- **FR76.5**: Client PHẢI có retry với exponential backoff cho `429` và `5xx`,
  tôn trọng trần 5 req/s. Đúng bài học CR-013 (`AzureTTSAdapter`): burst nhiều
  lượt gọi liên tiếp sẽ gặp lỗi rải rác, retry là bảo hiểm bắt buộc chứ không
  phải tuỳ chọn.
- **FR76.6**: Lỗi PHẢI phân biệt được và hiện đúng nguyên nhân cho Creator:
  thiếu/sai key, hết số dư, vượt rate limit, timeout, model trả rỗng. Không gộp
  hết thành "AI lỗi". Đây là bài học trực tiếp từ nợ kỹ thuật đã ghi *"Key
  Azure sai vẫn suy giảm âm thầm về Edge"* — CR này không được lặp lại nó.

### FR77 — Render prompt ở server (rút từ web-gui về orchestrator)
Hôm nay GUI tự ghép prompt: lấy `template_text` từ `GET /v1/prompts/{role}`
rồi thay `{{topic}}`, `{{channel_identity}}`, `{{format_beats}}`,
`{{narration_language_rule}}`, `{{previous_output}}`, `{{lint_results}}` bằng
hằng số nằm trong `scriptPrompts.ts`. Nếu server cũng tự ghép một kiểu riêng,
sẽ có **hai prompt khác nhau cho cùng một vai trò** — đường copy tay và đường
gọi API cho ra kết quả lệch nhau mà không ai biết.

- **FR77.1**: Orchestrator PHẢI có một hàm render prompt duy nhất: nhận
  `role`, `language`, `project_id` → đọc `prompt_templates`, đọc
  `project_authoring` + `projects`, điền mọi biến `{{...}}`, trả prompt cuối.
- **FR77.2**: Thêm `GET /v1/projects/{id}/prompts/{role}` trả về **prompt đã
  điền xong**. GUI chuyển nút "Copy prompt" hiện có sang endpoint này.
- **FR77.3**: Sau FR77.2, `scriptPrompts.ts` KHÔNG còn tự điền biến. Các hằng
  số bản sắc kênh (`CHANNEL_IDENTITY`, `NARRATION_LANGUAGE_RULE`,
  `buildStoryBeatSheetSection`) chuyển về server. Đây là điều kiện để đường
  copy tay và đường gọi API **không bao giờ lệch nhau** — không phải việc dọn
  dẹp cho đẹp.
- **FR77.4**: Đường copy tay PHẢI còn nguyên vẹn sau CR này (xem FR83).

### FR78 — Chạy một bước bằng AI
- **FR78.1**: Thêm `POST /v1/projects/{id}/authoring/{step}/generate` với
  `step ∈ {story, storyboard, code, review}`. Server tự: render prompt
  (FR77.1) → gọi provider (FR76) → validate (FR81) → lưu vào
  `project_authoring` → trả kết quả về GUI.
- **FR78.2**: Kết quả PHẢI hiện trong ô soạn thảo của tab đó để Creator sửa,
  **không** tự động nhảy sang bước sau, **không** tự động nộp vào saga. Cùng
  nguyên tắc FR71.3 của CR-026: AI gợi ý, Creator quyết bản cuối.
- **FR78.3**: Chạy lại một bước PHẢI ghi đè bản trước ở bước đó và **không**
  xoá kết quả các bước khác (đúng hành vi FR71.4 của CR-025).
- **FR78.4**: Endpoint PHẢI idempotent theo nghĩa thực dụng: hai lần bấm liên
  tiếp không tạo hai lượt gọi tính tiền. Khoá theo `(project_id, step)` trong
  lúc một lượt đang chạy, lượt thứ hai trả `409`.
- **FR78.5**: `remotion_engineer` dùng đúng endpoint `step=code`, phân biệt
  bằng `render_engine` của project — không thêm step thứ năm.

### FR79 — Nút "Chạy bằng AI" trên web-gui
- **FR79.1**: Mỗi tab của `ScriptPipelineTabs` (1a dàn ý, 1b storyboard, 1c
  code, 1d duyệt) thêm nút "Chạy bằng AI" **bên cạnh** nút Copy hiện có, không
  thay thế nó.
- **FR79.2**: Trong lúc chạy PHẢI có trạng thái chờ thấy được (bước code có
  thể mất vài chục giây) và nút bị khoá lại để không bấm chồng (FR78.4).
- **FR79.3**: Lỗi từ FR76.6 PHẢI hiện đúng nguyên nhân bằng tiếng Việt, kèm câu
  chỉ đường: thiếu key → chỉ tới `.env`; hết số dư → chỉ tới dashboard Hive;
  và luôn kèm "hoặc dùng nút Copy prompt như cũ".
- **FR79.4**: Nút PHẢI ẩn/disable kèm lời giải thích khi `LLM_PROVIDER` không
  phải `hive` hoặc chưa có key — không hiện một nút bấm vào là lỗi.
- **FR79.5**: `ScriptAssistant` (luồng "draft" — chỉnh script có sẵn cho hợp
  chuẩn) cũng được một nút chạy trực tiếp, cùng cơ chế.

### FR80 — Đưa kết quả bước trước vào prompt bước sau
- **FR80.1**: `{{previous_output}}` PHẢI lấy từ `project_authoring` ở server,
  không do GUI gửi lên — Creator không còn phải tự dán qua lại.
- **FR80.2**: Với context 1M token của Hive, **không** cắt ngắn
  `{{previous_output}}` như `maxScriptChars = 4000` của Ollama. Nhưng PHẢI có
  một trần an toàn cấu hình được (`HIVE_MAX_INPUT_CHARS`) để một project hỏng
  không sinh ra một lượt gọi tốn bất thường.
- **FR80.3**: `{{lint_results}}` của bước 4 (`script_reviewer`) hôm nay ĐÃ được
  GUI tự điền — nhưng bằng `utils/scriptValidation.ts`, một lint regex ở client
  chỉ biết: có class Scene không, đếm `self.narrate(...)` viết thẳng, tổng số
  từ, ước lượng thời lượng. Chính docstring của nó nói rõ đây là **ước lượng
  tối thiểu**. Trong khi đó `lint_manim_script` ở `rendering` (CR-017 FR46) mới
  là lint thật: AST + whitelist API `conceptflow`, bắt màu hex viết thẳng, cỡ
  chữ ngoài thang `FontScale`, mọi API không có trong `conceptflow.__all__` —
  chia BLOCKING/WARNING.
  Hệ quả: **AI duyệt ở bước 4 đang đọc một bản lint yếu hơn hẳn cái lint sẽ
  thật sự chặn render ở saga.** Nó báo PASS trên script mà `validate_script`
  sau đó chặn. CR này PHẢI đưa `{{lint_results}}` về đúng kết quả của
  `lint_manim_script`.
- **FR80.4**: `rendering` hiện KHÔNG có bề mặt HTTP nào — nó thuần message-driven
  và `validate_script` chỉ chạy như một bước saga. Muốn lint đồng bộ lúc soạn
  thì phải thêm một đường gọi. Cách làm chốt ở LLD (xem câu hỏi 6).

### FR81 — Không tin đầu ra của model một cách mù quáng
- **FR81.1**: Kết quả bước `code` PHẢI đi qua `lint_manim_script` / `ast.parse`
  trước khi lưu — y như FR71.2 của CR-026.
- **FR81.2**: Lint fail ⇒ **tự chạy lại đúng một lần**, có kèm lỗi lint vào
  prompt để model tự sửa. Vẫn fail ⇒ trả về cho Creator **kèm cả code lẫn lỗi
  lint**, không nuốt. Tối đa 2 lượt gọi cho một lần bấm — trần cứng, để một
  script cứng đầu không thành một vòng lặp đốt tiền.
- **FR81.3**: Model trả rỗng / chỉ có markdown fence / không có class kế thừa
  `ConceptFlowScene` PHẢI bị coi là lỗi, không phải kết quả hợp lệ.
- **FR81.4**: Code trả về PHẢI được bóc khỏi khối ```` ```python ```` trước khi
  lưu — model OpenAI-compatible gần như luôn bọc code trong fence.

### FR82 — Đo và hiện mức dùng token trên web (thay cho log Loki)
- **FR82.1**: Mỗi lượt gọi LLM PHẢI ghi một hàng vào bảng mới `llm_usage`:
  thời điểm, provider, model, role/step, `project_id` (nullable — tác vụ không
  gắn project như `suggest-short-script`), prompt tokens, completion tokens,
  thời gian chạy, thành công/thất bại kèm loại lỗi.
- **FR82.2**: Thêm `GET /v1/admin/llm-usage` trả tổng theo ngày/theo model/theo
  vai trò, cộng danh sách các lượt gọi gần nhất.
- **FR82.3**: Thêm một màn theo dõi trong mục Cài đặt (cạnh
  `PromptSettingsPage`) hiện: tổng token hôm nay và 30 ngày, chia theo vai trò,
  và bảng các lượt gọi gần đây. Dùng `components/ui` sẵn có, đúng
  `DESIGN_SYSTEM.md`.
- **FR82.4**: Số token PHẢI lấy từ trường `usage` trong response của Hive, KHÔNG
  ước lượng ở client — ước lượng sai thì màn theo dõi vô nghĩa.
- **FR82.5**: Ghi `llm_usage` thất bại KHÔNG được làm hỏng lượt gọi đã thành
  công — đo đạc không bao giờ được chặn tính năng.
- **FR82.6**: Bảng `llm_usage` KHÔNG lưu nội dung prompt hay kết quả, chỉ lưu
  số đếm — tránh nhân đôi dữ liệu đã có trong `project_authoring`.

### FR83 — Đường copy tay là fallback vĩnh viễn, không phải tạm thời
- **FR83.1**: Mọi nút Copy prompt hiện có PHẢI còn nguyên sau CR này.
- **FR83.2**: Hệ thống chạy được **không cần `HIVE_API_KEY`** — không có key
  thì app trở về đúng hành vi trước CR-027, không lỗi, không cảnh báo đỏ.
- **FR83.3**: Mọi ô soạn thảo PHẢI nhận được cả nội dung dán tay lẫn nội dung
  do AI sinh — không có ô nào thành chỉ-đọc vì "AI lo rồi".

### FR84 — Tách prompt thành hai tầng: bản gốc (seed) và bản tuỳ chỉnh (Creator)
**Quyết định Creator, 2026-09-21.** Hôm nay `prompt_templates` trộn hai thứ
khác hẳn nhau vào cùng một hàng: prompt mặc định ship trong binary, và bản
Creator sửa tay qua màn admin. Vì trộn chung nên `SeedPromptTemplates` buộc
phải `ON CONFLICT DO NOTHING` — và docstring của chính nó ghi rõ rằng phương án
seed-theo-version **đã thử rồi gỡ bỏ**, vì nó âm thầm xoá bản sửa của admin.
Nghĩa là mâu thuẫn ở đây không phải lỗi cài đặt, mà là hệ quả tất yếu của việc
một hàng phải gánh hai vai. Tách hai tầng là gỡ đúng gốc.

- **FR84.1**: Prompt gốc (seed) là **bất biến với người dùng** — không sửa,
  không xoá được, không có nút nào trên UI làm được việc đó. Nó là nguồn sự
  thật duy nhất ánh xạ 1-1 với code trong binary.
- **FR84.2**: Vì không ai sửa được nó, seed chuyển từ `DO NOTHING` sang **luôn
  ghi đè từ binary mỗi lần khởi động**. Cái bẫy "sửa file `.go`, rebuild, DB
  vẫn text cũ" biến mất hoàn toàn — và biến mất *theo cấu trúc*, không phải nhờ
  ai đó nhớ bấm nút reset.
- **FR84.3**: Bản tuỳ chỉnh của Creator sống ở tầng riêng, khoá theo
  `(role, language)`, có cờ `is_active`. Tạo/sửa/xoá tự do trên UI.
- **FR84.4**: Lúc render prompt: `is_active = true` thì dùng bản tuỳ chỉnh,
  ngược lại dùng bản gốc. **Mặc định là bản gốc.** Không có bản tuỳ chỉnh nào
  cũng chạy bình thường — đó là trạng thái sạch, không phải trạng thái thiếu.
- **FR84.5**: Tắt `is_active` PHẢI quay về bản gốc ngay mà **không xoá** bản
  tuỳ chỉnh — bật lại được. Đây là thứ thay thế cho `POST .../reset` hiện tại,
  và tốt hơn nó: reset hôm nay là phá huỷ, một chiều, mất bản sửa vĩnh viễn.
- **FR84.6**: Màn admin PHẢI hiện cả hai cạnh nhau: bản gốc ở chế độ chỉ-đọc
  (kèm version của binary), bản tuỳ chỉnh sửa được, một công tắc active, và một
  nút "tạo bản tuỳ chỉnh từ bản gốc" để không phải chép tay làm điểm khởi đầu.
- **FR84.7**: Khi bản gốc đổi (rebuild với prompt mới) mà Creator đang bật một
  bản tuỳ chỉnh, UI PHẢI báo cho biết bản gốc đã đổi và cho xem khác biệt.
  Không tự động làm gì cả — chỉ để Creator không âm thầm chạy bằng một bản
  tuỳ chỉnh đã lỗi thời, vốn là đúng lớp lỗi FR84 đang đi sửa.
- **FR84.8 — Di trú dữ liệu đang chạy, KHÔNG được mất bản sửa của Creator**:
  lúc nâng cấp, với mỗi hàng `prompt_templates` hiện có:
  - Giống hệt bản gốc trong binary ⇒ không tạo bản tuỳ chỉnh nào.
  - **Khác bản gốc** ⇒ đó là bản Creator từng sửa tay: chép sang tầng tuỳ chỉnh
    và **bật `is_active` luôn**, để hành vi sau khi nâng cấp giống hệt trước
    khi nâng cấp. Ghi log rõ những hàng nào đã được chuyển.
  Cách khác — mặc định về bản gốc cho sạch — là âm thầm đổi prompt Creator đang
  chạy. Chính xác là lớp lỗi CR này tồn tại để diệt, nên không được làm.
- **FR84.9**: Một bản tuỳ chỉnh cho mỗi `(role, language)` là đủ cho CR này.
  Nhiều biến thể có tên để đổi qua lại là phần mở rộng dễ làm sau, không thuộc
  phạm vi ở đây.
- **FR84.10 — chốt Creator 2026-09-21: dùng BẢNG RIÊNG.** `prompt_templates`
  giữ nguyên schema và trở thành **thuần-seed** — xoá sạch và dựng lại từ
  binary bất cứ lúc nào mà không mất gì của Creator. Bản tuỳ chỉnh sống ở bảng
  mới `prompt_overrides (role, language, template_text, is_active, updated_at)`,
  PK `(role, language)`. Không đổi PK, không thêm cột `source` vào bảng cũ:
  ranh giới giữa "của hệ thống" và "của Creator" là ranh giới bảng, nhìn phát
  biết ngay, và `DELETE FROM prompt_templates` trở thành một thao tác an toàn.

#### Hiện trạng DB đã đo (2026-09-21, stack đang chạy)
So md5 của cả 10 hàng `prompt_templates` với `DefaultPromptTemplates()` trong
binary: **10/10 khớp từng byte.** Nghĩa là **hôm nay không có bản sửa tay nào
đang sống** — FR84.8 hiện không có gì phải cứu. Vẫn phải cài đặt và test đầy
đủ: trạng thái này có thể đổi bất cứ lúc nào trước khi CR được làm.

Một phát hiện quan trọng hơn cho FR84.8: **cột `version` KHÔNG dùng được để
biết một hàng đã bị sửa hay chưa.** `story_architect` đang ở version 4 (vi) và
3 (en) trong khi seed ship version 2 — nhưng nội dung vẫn khớp seed từng byte.
Lý do: `ResetPromptTemplate` đi qua `Update()`, mà `Update()` luôn `version + 1`
— nên mỗi lần reset lại đẩy version lên dù text quay về đúng bản gốc.

⇒ **FR84.8 PHẢI so bằng nội dung (md5/text), tuyệt đối không so bằng `version`.**
So bằng version sẽ tạo ra 2 bản tuỳ chỉnh ma cho `story_architect`, đang bật,
nội dung y hệt bản gốc — rác vĩnh viễn ngay từ lần di trú đầu tiên.

Dấu vết `updated_at` (cả 3 vai trò đều được ghi lại trong hôm nay) cũng xác
nhận cái bẫy `DO NOTHING` đang được xử lý bằng tay: rebuild xong phải nhớ bấm
reset từng vai trò thì prompt mới trong binary mới tới được DB.

## Non-goals (out of scope cho CR-027)
- **Không có nút chạy liền mạch 4 bước** (D3). Để CR sau, sau khi có số liệu
  từ FR82.
- **Không đặt hạn mức chi tiêu chặn cứng** (D4). Đo trước, chặn sau.
- **Không gỡ Ollama.** Container `ollama` và mọi hành vi hiện tại của
  `suggest-metadata` / `suggest-short-script` giữ nguyên.
- **Không dùng tính năng đọc ảnh/video của Hive.** Hai model đều nhìn được ảnh,
  nhưng review bằng VLM đã bị CR-025 loại tường minh; CR này không mở lại.
- **Không đụng Saga, `validate_script`, TTS, render, QC, publish.** Chỉ thêm ở
  phía **soạn**, đúng phạm vi CR-025.
- **Không streaming.** Trả nguyên khối. Streaming chỉ để làm đẹp thanh chờ,
  không đáng độ phức tạp SSE xuyên qua api-gateway.
- **Không prompt caching của Hive ở đợt này** — ghi nhận là hướng giảm chi phí
  sau khi FR82 cho biết chỗ nào tốn nhất.

## Rủi ro
- **Chất lượng bước sinh code là rủi ro số một.** Một model flash viết code
  Manim đúng chuẩn `ConceptFlowScene` + `self.narrate()` + whitelist API của
  CR-017 là việc khó. Giảm nhẹ: FR81 (lint + một lượt sửa), và FR83 giữ đường
  copy tay nguyên vẹn để Creator luôn lùi được về AI mạnh hơn ở ngoài.
- **Lần đầu có dịch vụ tính tiền trong pipeline.** Prompt của pipeline này rất
  dài (bản sắc kênh + beat sheet + toàn bộ output bước trước). Giảm nhẹ: FR82
  đo từ lượt gọi đầu tiên, FR80.2 có trần input, FR81.2 có trần 2 lượt/lần bấm.
- **Rate limit 5 req/s.** Một Creator không chạm tới, nhưng FR81.2 retry + hai
  tab mở song song thì có thể. Giảm nhẹ: FR76.5.
- **Rút prompt từ GUI về server (FR77) chạm vào đường Creator dùng hằng ngày.**
  Làm hỏng nó là hỏng cả đường copy tay lẫn đường mới. Giảm nhẹ: FR77 làm
  thành bước riêng, test so sánh prompt server sinh ra **khớp từng ký tự** với
  prompt GUI đang sinh, trước khi nối vào FR78.
- **Di trú `prompt_templates` sang hai tầng (FR84) chạm vào dữ liệu đang chạy.**
  Rủi ro cụ thể là làm mất bản prompt Creator đã sửa tay. Giảm nhẹ: FR84.8 quy
  định rõ hành vi, và PHẢI có test cho cả hai nhánh (hàng giống bản gốc / hàng
  đã sửa) trước khi chạm vào DB thật. Sao lưu bảng trước khi nâng cấp.
  *(FR84 đồng thời xoá sổ cái bẫy `ON CONFLICT DO NOTHING` — sửa prompt trong
  file `.go`, rebuild, mà DB vẫn giữ text cũ, không một thông báo nào.)*
- **Doc Hive có thể đi sau bảng model thật.** Tên model phải xác nhận trên
  dashboard ở bước LLD, không hardcode theo doc.

## Câu hỏi cần chốt ở Low-Level Design
1. Tên model mặc định — xác nhận trên dashboard Hive; `deepseek-v4.1-flash` hay
   `glm-5.3-flash` cho bước sinh code (cần đo thử trên cùng một chủ đề).
2. Template có cần tách `system` / `user` không, hay giữ một khối rồi đưa cả
   vào `system`. Sau FR84 câu này **rẻ đi hẳn**: đổi cấu trúc bản gốc chỉ là
   sửa code rồi rebuild, seed tự ghi đè. Chỉ còn phải quyết cách xử lý bản tuỳ
   chỉnh viết theo cấu trúc cũ (đề xuất: giữ nguyên, coi cả khối là `system`).
7. ~~Hai tầng nằm ở đâu~~ — **đã chốt 2026-09-21: bảng riêng
   `prompt_overrides`** (xem FR84.10).
3. Trần `HIVE_MAX_INPUT_CHARS` khởi đầu.
4. Khoá chống bấm chồng (FR78.4) đặt ở đâu — in-memory trong orchestrator hay
   một cột trạng thái trong `project_authoring`.
5. Có cần ADR mới không. Đề xuất: **có** — ADR-0029 (chọn LLM provider trả phí,
   quan hệ Hive/Ollama) vì nó lật lại quyết định tường minh "không tự gọi API
   trả phí" của CR-025.
6. **Đường gọi `lint_manim_script` đồng bộ lúc soạn (FR80.3/80.4)** — ba lựa chọn:
   (a) thêm một endpoint HTTP nhỏ `POST /lint` vào `rendering`, đúng khuôn
   `video-assembly` đã có (`VIDEO_ASSEMBLY_URL` trong config orchestrator là
   tiền lệ); (b) request/reply đồng bộ qua AMQP — hợp kiến trúc hiện tại hơn
   nhưng phức tạp hơn nhiều cho một lời gọi đọc-thuần; (c) chép luật lint sang
   orchestrator — **loại**, hai bản lint trôi khỏi nhau là đúng lớp lỗi mà
   FR77 đang đi sửa. Đề xuất (a).

## Kiểm chứng
- **Unit test (Go)**: `HiveClient` parse response đúng, bóc code fence (FR81.4),
  retry đúng trên 429/5xx và không retry trên 401 (FR76.5/76.6), phân loại lỗi
  đúng, `TokenUsage` đọc từ trường `usage` chứ không ước lượng (FR82.4).
- **Unit test (Go)**: prompt renderer điền đủ mọi biến, và prompt sinh ra
  **khớp từng ký tự** với prompt `scriptPrompts.ts` đang sinh cho cùng dữ liệu
  đầu vào (FR77 — test then chốt của cả CR).
- **Unit test (Go)**: lint fail → chạy lại đúng một lần → vẫn fail thì trả cả
  code lẫn lỗi (FR81.2); trần 2 lượt không bao giờ bị vượt.
- **Unit test (Go)**: thiếu `HIVE_API_KEY` ⇒ khởi động bình thường, provider về
  `ollama`, các endpoint cũ không đổi hành vi (FR83.2).
- **Unit test (Go)**: ghi `llm_usage` lỗi không làm hỏng lượt gọi (FR82.5).
- **Unit test (Go)**: seed ghi đè bản gốc mỗi lần khởi động (FR84.2), nhưng
  KHÔNG đụng tới bản tuỳ chỉnh đang bật (FR84.4).
- **Unit test (Go)**: di trú — hàng giống bản gốc không sinh bản tuỳ chỉnh;
  hàng đã sửa tay sinh bản tuỳ chỉnh **đang bật**, nội dung y nguyên (FR84.8).
- **Unit test (Go)**: di trú một hàng có `version` cao hơn seed nhưng nội dung
  khớp seed ⇒ KHÔNG sinh bản tuỳ chỉnh. Đây là trạng thái thật của
  `story_architect` trên DB hôm nay, không phải case giả định.
- **Unit test (Go)**: tắt active ⇒ prompt render ra khớp bản gốc, và bản tuỳ
  chỉnh vẫn còn nguyên trong DB (FR84.5).
- **Unit test (web-gui)**: nút "Chạy bằng AI" khoá trong lúc chạy; lỗi hiện
  đúng câu; nút Copy cũ vẫn còn và vẫn hoạt động (FR83.1).
- **Thủ công**: chạy trọn một chủ đề qua 4 bước **chỉ bằng nút AI**, không mở
  cửa sổ AI ngoài nào, tới lúc có video. So chất lượng với một video làm bằng
  đường copy tay trên cùng chủ đề — đây là phép thử "go pro" thật sự.
- **Thủ công**: đối chiếu số token trên màn FR82.3 với số trên dashboard Hive
  sau cùng một buổi làm việc.

## Kế hoạch thực hiện (thứ tự đề xuất, mỗi mốc tự đứng được)
| # | Mốc | Nội dung | Vì sao ở đây |
|---|---|---|---|
| 1 | Provider layer | FR76 — port, `HiveClient`, config, retry, phân loại lỗi, bọc `OllamaClient` | Không đụng gì đang chạy; test được độc lập |
| 2 | Đo token | FR82.1/82.4/82.5/82.6 — bảng `llm_usage` + ghi nhận | Vào trước lượt gọi thật đầu tiên, để không có lượt nào không được đo |
| 3 | Hai tầng prompt | FR84 — bảng tuỳ chỉnh + cờ active + di trú + màn admin | Làm TRƯỚC FR77: FR77 sẽ đổi chỗ đọc template, đụng vào một bảng đang mang hai vai thì rối gấp đôi |
| 4 | Prompt về server | FR77 — renderer + `GET .../prompts/{role}` + gỡ điền biến khỏi GUI | Bước rủi ro nhất, làm riêng, có test khớp-từng-ký-tự làm lưới an toàn |
| 5 | Lint thật lúc soạn | FR80.3/80.4 — đường gọi `lint_manim_script` đồng bộ | Cần cho cả bước 4 lẫn vòng tự-sửa FR81.2; độc lập với việc gọi LLM |
| 6 | Chạy một bước | FR78, FR80, FR81 — endpoint generate + lint + chống bấm chồng | Giờ mới có đủ cả prompt đúng, provider, lẫn lint thật |
| 7 | GUI | FR79, FR83 — nút "Chạy bằng AI" ở 5 chỗ, giữ nguyên nút Copy | Lúc này backend đã gọi được bằng curl |
| 8 | Màn token | FR82.2/82.3 — endpoint tổng hợp + màn theo dõi | Cần dữ liệu thật từ mốc 4–5 mới có gì để hiện |
| 9 | Verify | Chạy trọn một video chỉ bằng nút AI; đối chiếu token với dashboard | Nghiệm thu |

Nhánh: `feature/cr-027-hive-llm-provider` (đúng chính sách nhánh-mỗi-CR).
Rebuild sau mỗi mốc chạm code: `docker compose build orchestrator web-gui` +
`up -d`, xác nhận healthy.

## Liên quan
- CR-025 (`cr-025-authoring-pipeline-and-script-review.md`) — pipeline 4 bước,
  `prompt_templates`, `project_authoring`; CR này lật lại quyết định "copy tay"
  và mục Non-goal "không tự gọi API trả phí" của nó.
- CR-026 (`cr-026-ai-assisted-short-script.md`) — FR71.2 (bắt buộc lint đầu ra
  của model) là tiền lệ trực tiếp cho FR81.
- CR-014 (`cr-014-suggest-metadata-fixes.md`) — tràn `num_ctx` của Ollama, lý do
  kỹ thuật khiến CR-025 không thể tự động hoá; context 1M của Hive gỡ đúng nút
  này.
- CR-013 (`cr-013-azure-tts-retry.md`) — khuôn retry/backoff cho API ngoài.
- CR-011 / ADR-0025 — khuôn quản lý key dịch vụ trả phí trong `.env`, và bài
  học "key sai suy giảm âm thầm" mà FR76.6 phải tránh.
- CR-017 / CR-018 / CR-019 — whitelist API, `self.narrate()`, beat sheet: các
  ràng buộc mà đầu ra bước `code` phải thoả.
- CR-021 — tiền lệ "đo trước, chặn sau" (`QC_ENFORCE`) áp dụng cho hạn mức chi
  tiêu ở D4.
