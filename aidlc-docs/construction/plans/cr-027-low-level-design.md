# CR-027 — Low-Level Design: gọi LLM trong app qua Hive, prompt hai tầng

## Date
2026-09-21

## Stage
Low-Level Design (Functional/NFR/Infrastructure Design: SKIP — không service
mới, không container mới, không queue mới; hai bảng mới và một endpoint HTTP
nhỏ trên `rendering`, lý do ghi tại D5/D6)

## Phạm vi
Requirements: `aidlc-docs/inception/requirements/cr-027-hive-llm-provider-and-in-app-authoring.md`
(FR76–FR84, đã duyệt 2026-09-21). Nhánh `feature/cr-027-hive-llm-provider`,
đã merge `main` ở `12f613d` nên đã có đủ 6 vai trò prompt.

---

## Lỗ hổng phát hiện lúc thiết kế — `{{topic}}` không tồn tại ở server

Soi schema trước khi thiết kế FR77 và tìm ra một thứ chặn đường: **server không
có chủ đề của project ở bất cứ đâu.**

- `projects` có 52 cột — không cột nào là `topic`.
- `project_authoring` có `story_content`, `storyboard_content`, `code_content`,
  `review_content` — không có `topic`.
- Chủ đề sống trong `draft.authoringTopic`, tức React context + localStorage
  của **trình duyệt**.

`{{topic}}` xuất hiện trong prompt của `story_architect` và của cả hai
engineer. Nghĩa là FR77 (render prompt ở server) **không thể hoàn thành** nếu
không giải quyết chỗ này trước.

### D0 — Thêm cột `topic` vào `project_authoring`
`ALTER TABLE project_authoring ADD COLUMN IF NOT EXISTS topic TEXT NOT NULL DEFAULT ''`
— đúng khuôn 4 cột `*_content` đã có, cùng khoá, cùng vòng đời (dữ liệu
thời-soạn-thảo, một hàng một project, ghi bởi thao tác của Creator chứ không
phải do gấp sự kiện saga).

Không cho vào `projects` vì đúng lý do docstring của `project_authoring` đã
ghi khi tạo bảng này: `projects` được ghi bằng một câu UPDATE dài với tham số
theo vị trí, chen thêm một trường thời-soạn-thảo vào đó là rước rủi ro lệch
tham số cho mọi trường đang có.

- `POST /v1/projects/{id}/authoring/story` nhận thêm `topic` trong body và lưu
  cùng lúc với `story_content`. Tab 1a vốn đã là nơi Creator gõ chủ đề, nên
  không thêm thao tác nào.
- **Di trú project cũ**: cột mặc định `''`. Project trước CR-027 không có chủ
  đề ở server; prompt render ra sẽ mang `[DÁN CHỦ ĐỀ CỦA BẠN VÀO ĐÂY]` đúng như
  `TOPIC_PLACEHOLDER` hôm nay. Không sửa dữ liệu cũ, không đoán chủ đề từ
  `story_content`.

---

## Quyết định kiến trúc

### D1 — `LLMProviderPort` và hai adapter
```go
// internal/application/llm_provider.go
type ChatRequest struct {
    System      string
    User        string
    MaxTokens   int
    Temperature float64
}

type TokenUsage struct {
    Model            string
    PromptTokens     int
    CompletionTokens int
}

type LLMProviderPort interface {
    Name() string  // "hive" | "ollama"
    Chat(ctx context.Context, req ChatRequest) (content string, usage TokenUsage, err error)
}
```

`internal/adapters/llm/hive_client.go` — `net/http` thuần, cùng package với
`ollama_client.go`. Không SDK: Hive là OpenAI-compatible, request/response chỉ
là hai struct JSON, còn kéo `go-openai` vào là thêm một phụ thuộc để tiết kiệm
khoảng 40 dòng.

`OllamaClient` **không bị sửa**. Thêm `ollama_provider.go` bọc nó lại để thoả
`LLMProviderPort`; `MetadataSuggesterPort` và `ShortScriptSuggesterPort` giữ
nguyên chữ ký, `suggest-metadata` và `suggest-short-script` chạy y như cũ.
CR này không refactor hai đường đang hoạt động (FR76.3).

Ollama không trả `usage`, nên adapter bọc trả `TokenUsage{Model: ..., 0, 0}`.
Hàng `llm_usage` vẫn được ghi để biết **đã gọi**; số token bằng 0 là sự thật
về Ollama, không phải dữ liệu thiếu — màn FR82.3 ghi rõ "Ollama không báo
token" thay vì hiện 0 trống trơn.

### D2 — Fallback: chỉ cho tác vụ nhẹ, không cho bước soạn kịch bản
| Tác vụ | Provider | Hive lỗi thì sao |
|---|---|---|
| `suggest-metadata` (CR-014) | Hive nếu có key | Rơi về Ollama |
| `suggest-short-script` (CR-026) | Hive nếu có key | Rơi về Ollama |
| 4 bước soạn kịch bản (FR78) | Hive | **Trả lỗi.** Không fallback |

Lý do không fallback ở bước soạn (FR76.6 + bài học CR-014): prompt soạn kịch
bản dài 10–14k ký tự, `num_ctx=2048` của Ollama nuốt không nổi và sẽ trả về
một kết quả **trông có vẻ hợp lệ nhưng rỗng ruột**. Suy giảm âm thầm sang một
bản nháp vô dụng tệ hơn hẳn một thông báo lỗi — đây đúng là nợ kỹ thuật "key
Azure sai vẫn suy giảm âm thầm về Edge" đã ghi trong `aidlc-state.md`, và CR
này không được đẻ thêm một cái nữa.

### D3 — Hai tầng prompt: bảng riêng `prompt_overrides` (FR84.10)
```sql
CREATE TABLE IF NOT EXISTS prompt_overrides (
    role          TEXT NOT NULL,
    language      TEXT NOT NULL,
    template_text TEXT NOT NULL,
    is_active     BOOLEAN NOT NULL DEFAULT false,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (role, language)
);
```

`prompt_templates` **giữ nguyên schema**, trở thành thuần-seed.
`SeedPromptTemplates` đổi `ON CONFLICT DO NOTHING` → `DO UPDATE SET
template_text = EXCLUDED.template_text, version = EXCLUDED.version,
updated_at = now()`.

Docstring hiện tại của hàm này nói seed-theo-version *"đã thử rồi gỡ bỏ"* vì
nó âm thầm xoá bản sửa của admin. Lý do đó **không còn áp dụng**: sau D3 không
ai sửa được bảng này nữa, nên không còn gì để xoá. Docstring phải được viết
lại cho đúng, kèm con trỏ sang `prompt_overrides` — để lại nguyên văn cũ sẽ
khiến người đọc sau tưởng `DO UPDATE` là một lần lặp lại sai lầm đã biết.

Đọc template khi render:
```sql
SELECT COALESCE(o.template_text, t.template_text)
FROM prompt_templates t
LEFT JOIN prompt_overrides o
       ON o.role = t.role AND o.language = t.language AND o.is_active
WHERE t.role = $1 AND t.language = $2
```

**Di trú một lần (FR84.8)** — chạy lúc khởi động, sau seed, trong một transaction:
```
với mỗi hàng r trong prompt_templates_TRƯỚC_KHI_SEED_GHI_ĐÈ:
    nếu md5(r.template_text) == md5(seed[r.role][r.language]):
        bỏ qua
    ngược lại:
        INSERT prompt_overrides(role, language, template_text, is_active=true)
        log.Warn("CR-027 di trú: chuyển bản sửa tay sang override", role, language)
```
Thứ tự bắt buộc: **đọc bảng cũ trước, seed ghi đè sau.** Đảo lại là mất sạch
bản sửa tay — seed đã ghi đè thì không còn gì để so.

So bằng **md5 nội dung, không bao giờ bằng `version`**. Đo trên DB thật hôm
nay: `story_architect` ở version 4 (vi) và 3 (en) trong khi seed ship version
2, **nhưng nội dung khớp seed từng byte** — vì `ResetPromptTemplate` đi qua
`Update()`, mà `Update()` luôn `version + 1`. So bằng version sẽ đẻ ra 2
override ma đang bật, nội dung y hệt bản gốc, đóng băng vĩnh viễn trong khi
bản gốc tiến lên.

Cờ đánh dấu đã di trú: một hàng trong bảng `schema_migrations` nhẹ, hoặc đơn
giản hơn — di trú là no-op tự nhiên ở lần chạy thứ hai, vì sau lần đầu
`prompt_templates` đã bằng seed nên vòng lặp không tìm thấy hàng nào khác. Chọn
cách thứ hai: không thêm bảng, không thêm trạng thái.

### D4 — Ánh xạ `step` → vai trò là việc của server (FR78.5)
```go
func roleFor(step string, engine string) domain.PromptRole {
    switch step {
    case "story":      return RoleStoryArchitect          // chung cả 2 engine
    case "storyboard": if engine == "remotion" { return RoleRemotionVisualDirector }
                       return RoleVisualDirector
    case "code":       if engine == "remotion" { return RoleRemotionEngineer }
                       return RoleManimEngineer
    case "review":     return RoleScriptReviewer          // chung cả 2 engine
    }
}
```
`engine` đọc từ `projects.render_engine`. GUI **không** gửi `role` lên — hôm
nay `ScriptReviewerStepPage` và `ManimEngineerStepPage` tự tính `engineerRole`
ở client, và đó chính là lớp lỗi FR77 đang đi sửa.

### D5 — Bảng biến prompt và nguồn dữ liệu của từng biến
| Biến | Bước dùng | Nguồn ở server |
|---|---|---|
| `{{topic}}` | story, code | `project_authoring.topic` (D0) |
| `{{channel_identity}}` | story | Hằng số Go, chuyển từ `scriptPrompts.ts` |
| `{{format_beats}}` | story | `video_formats` theo `projects.video_format_id`, quy đổi giây→từ bằng WPM của `projects.voice_id` |
| `{{narration_language_rule}}` | story, code, review | Hằng số Go theo `projects.voice_language` |
| `{{previous_output}}` | storyboard, code, review | `project_authoring`: storyboard←story; code←story+storyboard; review←story+storyboard+code, nối bằng `\n\n---\n\n` |
| `{{lint_results}}` | review | `POST /lint` của `rendering` (D6) |

Ba hằng số (`CHANNEL_IDENTITY`, `NARRATION_LANGUAGE_RULE`,
`buildStoryBeatSheetSection`) chuyển sang `internal/domain/prompt_vars.go`.
`scriptPrompts.ts` bỏ phần điền biến, chỉ còn các prompt không thuộc pipeline
4 bước (`buildAdjustPromptFor`, `buildShortScriptPrompt`).

**Lưới an toàn bắt buộc trước khi nối vào FR78**: một test dựng dữ liệu cố
định, chạy qua renderer Go và qua `scriptPrompts.ts`, so **khớp từng ký tự**.
Đây là test then chốt của cả CR — nó là thứ duy nhất chứng minh đường copy tay
và đường AI dùng cùng một prompt.

### D6 — Lint thật lúc soạn: endpoint HTTP nhỏ trên `rendering`
`rendering` hiện thuần message-driven, không có bề mặt HTTP. Thêm
`POST /lint {script_content, engine}` → `{issues: [{line, message, severity}]}`,
gọi thẳng `lint_manim_script` đã có.

Chọn HTTP thay vì request/reply qua AMQP: `video-assembly` đã là tiền lệ
(`VIDEO_ASSEMBLY_URL` + `VIDEO_ASSEMBLY_TIMEOUT_SECONDS` trong config
orchestrator), và đây là một lời gọi **đọc-thuần, đồng bộ, thời-soạn-thảo** —
không phải một bước saga, không có trạng thái, không cần đảm bảo giao nhận.
Dựng request/reply qua queue cho nó là mang cả bộ máy saga vào một hàm thuần.

Chép luật lint sang orchestrator: **loại thẳng**. Hai bản lint trôi khỏi nhau
đúng là lớp lỗi cả CR này đang đi sửa.

Lint dùng ở hai chỗ: `{{lint_results}}` của bước review (thay bản regex yếu ở
client), và vòng tự-sửa của D8.

`utils/scriptValidation.ts` ở client **giữ nguyên** — nó còn việc riêng: ước
lượng thời lượng lúc gõ (CR-016 FR42), chạy tức thì, không cần mạng.

### D7 — Chạy một bước: `POST /v1/projects/{id}/authoring/{step}/generate`
```
1. đọc project (render_engine, voice_language, video_format_id, voice_id)
2. role  := roleFor(step, engine)                      [D4]
3. tmpl  := đọc override-đang-bật, nếu không có thì seed [D3]
4. prompt := render(tmpl, biến)                        [D5]
5. nếu vượt HIVE_MAX_INPUT_CHARS -> lỗi, không gọi
6. content, usage := provider.Chat(...)                [D1]
7. ghi llm_usage (không chặn)                          [D9]
8. hậu xử lý + validate                                [D8]
9. lưu project_authoring.<step>_content
10. trả {content, usage, lint_issues}
```

**Khoá chống bấm chồng (FR78.4)**: `map[string]struct{}` + `sync.Mutex` trong
process, khoá theo `project_id + step`, lượt thứ hai trả `409`. Orchestrator
chạy một replica (docker-compose, không scale) nên khoá in-process là đủ; nếu
sau này chạy nhiều replica thì đổi sang `pg_advisory_lock` — ghi lại làm con
trỏ, không làm bây giờ.

Timeout: `HIVE_TIMEOUT_SECONDS` mặc định 180. Bước code trả về hàng trăm dòng,
120s của Ollama là quá ngắn.

### D8 — Hậu xử lý đầu ra của model
Theo thứ tự:
1. **Bóc markdown fence** (FR81.4) — port logic `MARKDOWN_FENCE_RE` đã có
   trong `scriptValidation.ts` sang Go. Model OpenAI-compatible gần như luôn
   bọc code trong ```` ```python ````.
2. **Bắt rỗng** (FR81.3) — trống, hoặc chỉ còn fence sau khi bóc, hoặc bước
   code mà không có class kế thừa `ConceptFlowScene` ⇒ lỗi, không phải kết quả.
3. **Lint** (FR81.1) — chỉ bước `code`, qua D6.
4. **Một lượt tự sửa** (FR81.2) — có lỗi BLOCKING thì gọi lại **đúng một lần**,
   prompt lần hai = prompt gốc + code vừa sinh + danh sách lỗi lint. Vẫn lỗi ⇒
   trả về Creator **cả code lẫn lỗi**, lưu vào `code_content` bình thường.
   Trần cứng 2 lượt gọi cho một lần bấm.

Cảnh báo WARNING không kích hoạt lượt sửa — chỉ BLOCKING. Đúng tinh thần
CR-017: đường thoát hiểm là cảnh báo, không phải lỗi.

### D9 — `llm_usage` và màn theo dõi
```sql
CREATE TABLE IF NOT EXISTS llm_usage (
    id                BIGSERIAL PRIMARY KEY,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    provider          TEXT NOT NULL,
    model             TEXT NOT NULL,
    role              TEXT NOT NULL DEFAULT '',
    step              TEXT NOT NULL DEFAULT '',
    project_id        TEXT,
    prompt_tokens     INTEGER NOT NULL DEFAULT 0,
    completion_tokens INTEGER NOT NULL DEFAULT 0,
    duration_ms       INTEGER NOT NULL DEFAULT 0,
    ok                BOOLEAN NOT NULL,
    error_kind        TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS llm_usage_created_at_idx ON llm_usage (created_at DESC);
```
`project_id` nullable, **không** khoá ngoại sang `projects` — `suggest-short-script`
chạy được khi chưa có project nào (CR-026 FR71.1), và xoá project không được
làm mất lịch sử chi phí.

Không lưu prompt/kết quả (FR82.6) — đã có trong `project_authoring`.

Ghi bằng hàm nuốt lỗi (FR82.5): hỏng thì `log.Error` rồi đi tiếp. Đo đạc không
bao giờ được chặn tính năng đã thành công.

`GET /v1/admin/llm-usage?days=30` → `{daily: [...], by_role: [...], recent: [...]}`.
Màn `LLMUsagePage` cạnh `PromptSettingsPage`, dùng `components/ui` sẵn có.

### D10 — Biến môi trường
| Biến | Mặc định | Ghi chú |
|---|---|---|
| `LLM_PROVIDER` | `hive` nếu có key, ngược lại `ollama` | |
| `HIVE_API_KEY` | `""` | Rỗng ⇒ về `ollama`, **không crash lúc khởi động** (FR83.2) |
| `HIVE_BASE_URL` | `https://api-cdn.thehive.ai/api/v3` | |
| `HIVE_MODEL` | *chưa chốt* | Chờ phép đo — xem "Việc đang chờ" |
| `HIVE_TIMEOUT_SECONDS` | `180` | |
| `HIVE_MAX_INPUT_CHARS` | `120000` | ~30k token, thừa cho prompt dài nhất (14k ký tự) mà vẫn chặn được project hỏng |
| `HIVE_MAX_RETRIES` | `3` | |

### D11 — Phân loại lỗi (FR76.6)
| HTTP | `error_kind` | Retry | Câu hiện cho Creator |
|---|---|---|---|
| 401/403 | `auth` | không | "API key Hive sai hoặc thiếu — kiểm tra `HIVE_API_KEY` trong `.env`" |
| 402 | `balance` | không | "Tài khoản Hive hết số dư — nạp thêm trên dashboard" |
| 429 | `rate_limit` | **có** | "Hive đang quá tải, đã thử lại 3 lần" |
| 5xx | `server` | **có** | "Hive lỗi phía máy chủ" |
| timeout | `timeout` | không | "Hive không trả lời trong 180s" |
| rỗng/không parse được | `empty` | không | "Hive trả về kết quả rỗng hoặc sai định dạng" |

Mọi câu đều kèm "hoặc dùng nút Copy prompt như cũ" (FR79.3). Retry dùng
exponential backoff có jitter, đúng khuôn `AzureTTSAdapter` của CR-013.

---

## Thứ tự thực hiện
| # | Mốc | Test then chốt |
|---|---|---|
| 1 | D0 — cột `topic` + lưu từ tab 1a | project cũ (`topic=''`) vẫn render prompt được |
| 2 | D1, D10, D11 — provider + config + lỗi | thiếu key ⇒ khởi động bình thường, về `ollama` |
| 3 | D9 — `llm_usage` | ghi lỗi không làm hỏng lượt gọi |
| 4 | D3 — hai tầng prompt + di trú | hàng version cao nhưng nội dung khớp seed ⇒ KHÔNG sinh override |
| 5 | D5 — render prompt ở server | **khớp từng ký tự với `scriptPrompts.ts`** |
| 6 | D6 — `POST /lint` trên `rendering` | lint thật khác lint client ở case đã biết |
| 7 | D4, D7, D8 — generate + tự sửa | trần 2 lượt không bao giờ bị vượt |
| 8 | FR79/FR83 — GUI | nút Copy cũ còn nguyên và còn chạy |
| 9 | FR82.2/82.3 — màn token | đối chiếu với dashboard Hive |

Rebuild sau mốc chạm code: `docker compose build orchestrator web-gui` (mốc 6
thêm `rendering`) + `up -d`, xác nhận healthy.

---

## Việc đang chờ Creator
**`HIVE_API_KEY` chưa có trong `.env`.** Hai việc phụ thuộc vào nó, cả hai đều
không làm được bằng suy luận:

1. **Chốt `HIVE_MODEL`** — chạy cùng một chủ đề qua `deepseek-ai/deepseek-v4.1-flash`
   và `zai-org/glm-5.3-flash` ở bước sinh code, đếm lỗi lint BLOCKING của mỗi
   bên. Phải đo trên **cả hai engine** (Manim và Remotion) vì chúng có bộ từ
   vựng hình ảnh khác hẳn nhau — một model khá ở Manim chưa chắc khá ở Remotion.
2. **Xác nhận tên model trên dashboard Hive** — doc có thể đi sau bảng model
   thật, không hardcode theo doc.

Mốc 1–6 **không** cần key và làm được ngay.
