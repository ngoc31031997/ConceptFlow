# ConceptFlow — Review luồng dữ liệu từ ý tưởng đến phát hành

> Ngày review: 2026-09-22 · Phạm vi: toàn bộ pipeline (`services/*`, `docker-compose.yml`, `infra/rabbitmq/definitions.json`)
> Trạng thái: bản nháp để Creator review

---

## Phần 1 — Luồng dữ liệu theo user case

### Tổng quan 3 giai đoạn

Hệ thống thực chất có **ba giai đoạn tách biệt về mặt kiến trúc**, không phải một pipeline liền mạch như README mô tả:

```
GIAI ĐOẠN A — Sáng tác (đồng bộ, HTTP + LLM)
   Ý tưởng → story → storyboard → Manim code → review
   Chưa có project. Chưa có saga. Chưa tốn CPU render.
        │
        ▼  POST /v1/sagas/render
GIAI ĐOẠN B — Render Saga (bất đồng bộ, 7 bước qua RabbitMQ)
   parse → validate →[CỔNG DUYỆT]→ tts → render → assemble → qc → clips
        │
        ▼  POST /v1/sagas/publish
GIAI ĐOẠN C — Publish Saga (1 bước)
   publish_video → YouTube
```

---

### Giai đoạn A — Từ ý tưởng đến script Manim

Đây là wizard 4 bước trên Web GUI, mỗi bước là một "vai" LLM riêng với prompt template riêng
(`/v1/prompts/{role}`, chỉnh được trong `/settings/prompts`).

| # | Màn hình | Route GUI | Vai LLM | Dữ liệu sinh ra | Lưu ở đâu |
|---|---|---|---|---|---|
| 1 | Ý tưởng / chủ đề | `/` (`ScriptStepPage`) | — | chủ đề thô, ngôn ngữ, format | client state |
| 2 | Dàn ý kịch bản | `/create/script/outline` | Script Writer | outline + narration | `POST /authoring/story` |
| 3 | Storyboard | `/create/script/storyboard` | Visual Director | mô tả hình ảnh từng beat | `POST /authoring/storyboard` |
| 4 | Sinh code Manim | `/create/script/code` | Manim Engineer | `script_content` + `scene_class_name` | `POST /authoring/code` |
| 5 | Soát lỗi script | `/create/script/review` | Script Reviewer | nhận xét, sửa lỗi | `POST /authoring/review` |
| 6 | Cấu hình | `/create/settings` | — | engine, quality, giọng, sub, nhạc, intro/outro, output mode | client state |
| 7 | Xác nhận | `/create/review` | — | — | → `POST /v1/sagas/render` |

**Nhận xét giai đoạn A:** thiết kế tốt — LLM chỉ đụng tới giai đoạn này, hoàn toàn nằm ngoài
saga. Nghĩa là LLM hỏng/hết quota **không bao giờ** làm treo một saga đang chạy. Prompt tách
thành template + override có version là điểm cộng lớn cho việc tinh chỉnh chất lượng.

**Rủi ro:** `authoring/*` lưu theo `project_id`, nhưng project chỉ thực sự sinh ra ở bước
`POST /v1/sagas/render`. Cần kiểm tra lại project rỗng được tạo lúc nào — nếu Creator bỏ ngang
ở bước 3, hệ thống có để lại project mồ côi ở `draft` không, và có gì dọn chúng không.

---

### Giai đoạn B — Render Saga (5 bước hiệu lực, sau CR-029)

> **Cập nhật 2026-09-22 (CR-029):** `parse_script` + `validate_script` gộp
> thành một điểm dừng duy nhất; `qc_video` tắt khỏi luồng chính (đưa backlog —
> xem `cr-029-render-saga-consolidation.md`). Sơ đồ gốc 7 bước giữ lại bên dưới
> để tham chiếu lịch sử.

`Project` là aggregate root duy nhất, tích luỹ dần dữ liệu qua từng bước. Mọi bước đều đi qua
Outbox → `commands.direct` → queue của service → Inbox → xử lý → Outbox → `events.direct` →
`orchestrator.events`. Transactional Outbox/Inbox ở **cả hai đầu** — đây là phần làm rất chắc.

```
  ┌──────────────────────────────────────────────────────────────────┐
  │ POST /v1/sagas/render  →  Project{status: draft}                 │
  └───────────────────────────────┬──────────────────────────────────┘
                                  ▼
 1. parse_script          script_processing.commands    → script_parsed
      ghi: Scenes[], ManimSceneClassName, Chapters[]    status: parsing_script
                                  ▼
 2. validate_script       rendering.commands  (LƯỢT DRY MANIM!)   → script_validated
      ghi: Beats[], ValidationWarnings[]                status: validating_script
                                  ▼
      ┌─────────── CỔNG DUYỆT DÀN Ý (CR-024) ───────────┐
      │ status: awaiting_review — saga DỪNG, chờ người   │
      │ POST /approve → chạy tiếp   POST /reject → dừng  │
      │ POST /narration → sửa lời đọc trước khi chạy tiếp│
      │ (bỏ qua được bằng ReviewEnabled=false)           │
      └─────────────────────┬───────────────────────────┘
                            ▼
 3. synthesize_speech     tts.commands                  → speech_synthesized
      ghi: Scene.AudioPath, Scene.DurationSeconds       status: synthesizing_speech
      ⚠ BỎ QUA nếu TTSEnabled=false → duration ước lượng từ EstimateNarrationDuration
                                  ▼
 4. render_scenes         rendering.commands  (RENDER THẬT)  → rendering_completed
      ghi: RenderedVideoPath, WaitOffsets[], RenderedVideoSeconds,
           LayoutMarks[], ClipMarks[]                   status: rendering
      ↪ phát scene_rendered liên tục → progress.fanout → SSE → GUI
                                  ▼
 5. assemble_video        video_assembly.commands       → video_assembled
      ghép: video câm + audio + nhạc nền + sub + intro/outro (ffmpeg)
      ghi: VideoPath, CaptionPath, IntroDurationSeconds status: assembling_video
                                  ▼
 6. qc_video              video_assembly.commands       → qc_completed
      chấm chất lượng tự động (dùng LayoutMarks để soi bbox)
      ghi: QCReport                                     status: running_qc
      ⚠ KHÔNG CÓ NHÁNH FAIL — chấm không được thì status="not_scored", saga đi tiếp
                                  ▼
 7. generate_clips        video_assembly.commands       → clips_generated
      chỉ chạy nếu VideoOutputMode ∈ {short, both}      status: generating_clips
      ghi: Clips[]  ⚠ clip lỗi → status="error" trên 1 entry, saga vẫn đi tiếp
                                  ▼
                        status: ready_to_publish
```

### Sơ đồ hiện hành (CR-029)

```
  ┌──────────────────────────────────────────────────────────────────┐
  │ POST /v1/sagas/render  →  Project{status: draft}                 │
  └───────────────────────────────┬──────────────────────────────────┘
                                  ▼
 1. parse_and_validate_script   script_processing.commands → rendering.commands
      (gộp parse_script + validate_script — vẫn 1 dry-run Manim thật)
      ghi: Scenes[], ManimSceneClassName, Chapters[], Beats[], ValidationWarnings[]
      status: validating_script     ⚠ lỗi → cho sửa script/prompt ngay tại đây, chạy lại bước này
                                  ▼
      ┌─────────── CỔNG DUYỆT DÀN Ý (CR-024) ───────────┐
      │ status: awaiting_review — saga DỪNG, chờ người   │
      └─────────────────────┬───────────────────────────┘
                            ▼
 2. synthesize_speech     tts.commands                  → speech_synthesized
      status: synthesizing_speech   → phát % tiến trình (theo câu hoàn thành, không theo tick)
                                  ▼
 3. render_scenes         rendering.commands (RENDER THẬT) → rendering_completed
      status: rendering             → đã có scene_rendered/progress.fanout, giữ nguyên
                                  ▼
 4. assemble_video        video_assembly.commands       → video_assembled
      status: assembling_video      → phát % theo giai đoạn ffmpeg hoàn thành (mux audio/sub/intro)
                                  ▼
 5. generate_clips        video_assembly.commands       → clips_generated
      chỉ chạy nếu VideoOutputMode ∈ {short, both}      status: generating_clips
      status → phát % theo clip hoàn thành
                                  ▼
                        status: ready_to_publish
```

`qc_video` **không còn được dispatch** trong luồng chính — code vẫn giữ trong
`video-assembly` (tắt, không gọi) để thiết kế lại đúng vị trí sau (backlog: đưa
kiểm `LayoutMarks` lên trước bước 4, xem `cr-029-render-saga-consolidation.md`).

**Điểm thiết kế xuất sắc cần ghi nhận:**

1. **`WaitOffsets` thay vì cộng dồn duration.** Comment trong `project.go` nói rõ: animation
   giữa các câu đẩy mọi segment sau ra xa, cộng dồn `DurationSeconds` sẽ lệch tiếng/sub/hình
   đúng bằng tổng thời gian animation. Đây là loại bug rất khó tìm và đã được chặn từ thiết kế.
2. **QC và generate_clips không có nhánh fail.** "Một cổng hỏng không được biến thành cổng khoá"
   — triết lý đúng. Thước đo hỏng không được giữ con video làm con tin.
3. **`IntroDurationSeconds` truyền ngược từ video-assembly về để dịch clip selection.** Nếu
   thiếu, clip cắt từ project có intro sẽ lệch đúng bằng độ dài intro. Đã được lường trước (D5).
4. **`IntroAssetID` được chốt lại lúc dispatch** thay vì tra lại lúc retry — giữ đúng đảm bảo
   "retry dựng lại command thuần từ Project" (Rule 5).
5. **Cổng duyệt dàn ý đặt SAU validate, TRƯỚC tts.** Đúng chỗ: script sai bị chặn trước khi
   tiêu quota giọng đọc và trước khi tốn CPU render.

---

### Giai đoạn C — Phát hành

```
  ready_to_publish
        │
        │ (tuỳ chọn) POST /suggest-metadata → LLM gợi ý title/description/tags SEO
        │ (tuỳ chọn) upload thumbnail → YoutubeThumbnailPath
        │ OAuth: /v1/auth/youtube/start → callback → publisher-db.oauth_credentials
        ▼
  POST /v1/sagas/publish
        ▼
  publish_video   publisher.commands   → video_published
      YouTube Data API: upload video + thumbnail + caption (.srt)
      ghi: YoutubeVideoURL, CaptionStatus        status: publishing → published
```

**Nhận xét:** `CaptionStatus` được surface lên GUI (`uploaded`/`skipped_no_scope`/`failed`) —
tốt, một caption bị bỏ qua âm thầm là lỗi rất dễ không ai phát hiện. Quota YouTube được xử lý
bằng nhiều OAuth client trong `secrets/` (1 file = 1 rổ ~6 video/ngày, CR-012), đây là giải
pháp thực dụng và đúng.

---

## Phần 2 — Vấn đề rendering: có đang quá tải không?

### Trả lời ngắn

**Bản thân service rendering thì KHÔNG quá tải — nó đã được vá đúng. Nhưng nó là nút thắt cổ
chai toàn cục, và service video-assembly bên cạnh thì ĐANG hở đúng cái lỗi mà rendering đã vá.**

### 2.1 Rendering đã được bảo vệ đúng ✅

`services/rendering/main.py:92` có `await channel.set_qos(prefetch_count=1)`, kèm comment ghi lại
sự cố thật đã gặp:

> "Observed live with three concurrent dry runs: a single ReplacementTransform frame went from
> ~0.03s to 42s, and the run blew through DRY_RUN_TIMEOUT_SECONDS having rendered 148 of its
> animations."

Các lớp bảo vệ hiện có, đều đúng:
- `prefetch_count=1` — mỗi lúc đúng 1 lệnh
- `asyncio.to_thread` cho subprocess Manim — không khoá event loop, heartbeat AMQP và OutboxRelay vẫn sống
- `RLIMIT_AS` 4 GiB cho tiến trình render
- `deploy.limits`: 6 CPU / 5G
- timeout tách đôi: render thật 1800s, lượt dry 2000s
- cache `media_dir` theo project (đo được 21s → 4s khi render lại)

Đây là mức kỹ lưỡng hiếm thấy. Không cần sửa gì ở đây.

### 2.2 NHƯNG: rendering là nút thắt cổ chai toàn cục ⚠️

`container_name: rendering` cố định **đúng 1 instance**, `prefetch_count=1` cho **đúng 1 job
một lúc**, và queue `rendering.commands` gánh **ba loại lệnh khác nhau**:

| Lệnh | Bước saga | Thời lượng điển hình |
|---|---|---|
| `validate_script` | bước 2 | tới 2000s (chạy hết mọi animation!) |
| `render_scenes` | bước 4 | ~276s cho video 10' @1080p60 |
| `render_channel_asset` | ngoài saga | ngắn |

Hệ quả — **head-of-line blocking**:

> Project A đang chạy lượt dry 2000s ở bước *validate* (bước 2) thì Project B, C, D **không
> validate được, không render được, và intro/outro của kênh cũng không dựng được**. Chúng xếp
> hàng phía sau, im lặng. GUI của B, C, D chỉ hiện "validating_script" suốt 30 phút mà không
> có gì cho biết chúng đang *chờ*, chứ không phải đang *chạy*.

Đây là vấn đề trải nghiệm nghiêm trọng hơn là vấn đề tài nguyên. Hệ thống không sập, nó chỉ
trông như bị treo.

### 2.3 Video-assembly đang hở đúng lỗi rendering đã vá 🔴

`services/video-assembly/main.py` **không có `set_qos`** ở bất kỳ đâu. Kiểm chứng:

```
$ grep -rn "set_qos" services/video-assembly/main.py   → không có kết quả
```

Không set QoS nghĩa là `prefetch_count=0` = **không giới hạn**. Broker đẩy toàn bộ queue xuống,
và `queue.consume()` của aio-pika schedule mỗi delivery thành một task riêng. Trong khi đó
video-assembly multiplex **bốn** loại lệnh nặng ffmpeg trên cùng queue `video_assembly.commands`:

- `assemble_video` (ghép + có thể re-encode toàn bộ)
- `qc_video` (giải mã lại để chấm điểm)
- `generate_clips` (cắt N clip dọc, mỗi clip một lần encode)
- `normalize_channel_asset` (transcode intro/outro)

cộng thêm queue thứ hai `video_assembly.channel_asset_events` consume trên **cùng một channel**.

Container này chỉ có 6 CPU / **3G RAM**. Ba job ffmpeg song song trong 3G là kịch bản OOM-kill
rất thực tế — và OOM-kill giữa `assemble_video` nghĩa là message không được ack, bị redeliver,
rồi lại OOM. Vòng lặp chết.

**Đây là finding nghiêm trọng nhất của bản review này.**

### 2.4 Toàn bộ stack đang oversubscribe RAM của Docker VM 🔴

Đo được trên máy đang chạy:

```
Docker VM:  10 CPU / 8.0 GiB RAM
```

Trong khi compose khai báo 21 service, và **chỉ 2 service có resource limit**:

| Thành phần | Limit khai báo |
|---|---|
| rendering | 6 CPU / **5G** |
| video-assembly | 6 CPU / **3G** |
| **Cộng 2 service này** | **12 CPU / 8G** |
| 5× PostgreSQL | *không giới hạn* |
| ollama (chạy LLM local!) | *không giới hạn* |
| RabbitMQ | *không giới hạn* |
| Loki + Grafana + Promtail | *không giới hạn* |
| orchestrator, publisher, tts, script-processing, api-gateway, web-gui | *không giới hạn* |

Chỉ riêng rendering + video-assembly đã khai đúng **100% RAM của VM (8G/8G)** và **120% CPU
(12/10)**. Mọi thứ còn lại — kể cả `ollama` đang giữ một model LLM trong RAM — chạy ngoài
ngân sách đó.

Tình huống xấu cụ thể: một lượt dry Manim (4 GiB RLIMIT) chạy đồng thời với một lượt ffmpeg
assemble, trong khi ollama đang nạp model cho bước gợi ý metadata của project khác. Thứ bị
OOM-kill trước nhiều khả năng **không phải** service gây tải, mà là Postgres — và mất Postgres
giữa saga là mất dữ liệu bước đang chạy.

---

## Phần 3 — Các vấn đề khác phát hiện được

### 3.1 README đã lệch so với code

README mô tả "Render Saga (5 bước) + Publish Saga (1 bước)", nhưng code hiện có **7 bước**
(thêm `qc_video`, `generate_clips`) và một cổng duyệt `awaiting_review`. README cũng liệt kê
**Content Plugin Service** như một service đang chạy, nhưng service này **không còn trong
`docker-compose.yml`**:

```
$ grep -E "^  [a-z-]+:" docker-compose.yml
rabbitmq tts-db tts script-processing-db script-processing rendering-db rendering
video-assembly-db video-assembly publisher-db publisher orchestrator-db orchestrator
ollama ollama-pull api-gateway web-gui loki promtail grafana backend
```

Không có `content-plugin`, cũng không có `content-plugin-db`. Toàn bộ giai đoạn A (LLM authoring,
CR-027 Hive) và Remotion engine cũng không được nhắc tới. README đang mô tả một hệ thống của
vài CR trước.

### 3.2 `orchestrator.events` không có DLQ

Trong `infra/rabbitmq/definitions.json`, cả 5 queue `*.commands` đều có
`x-dead-letter-exchange` + DLQ tương ứng. Riêng `orchestrator.events` **không có** — và comment
trong `handle_step_event.go` xác nhận đây từng gây sự cố thật:

> "orchestrator.events is a classic queue with neither a delivery limit nor a dead-letter
> exchange — so a returned error on an event that can never succeed becomes an unbounded hot
> redelivery loop. Observed live: two validation_failed events for projects whose rows no longer
> existed produced thousands of identical ERROR lines per second, burning CPU and drowning Loki."

Đã được vá **ở tầng code** (warn + ack thay vì trả lỗi) — xử lý đúng cho đúng ca đó. Nhưng
nguyên nhân gốc ở tầng hạ tầng vẫn còn: bất kỳ lỗi *infrastructure* thật nào (Postgres rớt
chẳng hạn) vẫn trả error → nack requeue → loop nóng không giới hạn, không có gì chặn.

### 3.3 Trạng thái "đang chờ" không phân biệt được với "đang chạy"

`ProgressMessage.Status` chỉ có `in_progress | completed | failed`. Một project xếp hàng sau
job khác ở `rendering.commands` và một project đang render thật đều hiện `in_progress`. Kết hợp
với 2.2, Creator không có cách nào biết mình đang chờ 30 phút vì hàng đợi hay vì video của mình
nặng.

---

## Phần 4 — Đề xuất, theo thứ tự ưu tiên

### P0 — Làm ngay

**1. Thêm `set_qos` cho video-assembly.** Một dòng, chặn được kịch bản OOM loop ở 2.3:

```python
# services/video-assembly/main.py, ngay sau channel = await connection.channel()
await channel.set_qos(prefetch_count=1)
```

Lưu ý: QoS đặt trên channel, mà video-assembly consume **hai** queue trên cùng channel — nên
`prefetch_count=1` sẽ serialize cả `channel_asset_events` chung với command. Nếu không muốn
vậy, tách `channel_asset_events` sang channel thứ hai rồi set QoS riêng cho từng cái. Khuyến
nghị: cứ serialize trước đã (đúng và an toàn), tối ưu sau khi đo.

**2. Đặt resource limit cho những service còn lại, đặc biệt `ollama` và 5 Postgres.** Với VM
8 GiB, đề xuất khởi điểm:

| Service | mem_limit đề xuất | Lý do |
|---|---|---|
| ollama | 2G | giữ model LLM, hiện không giới hạn |
| mỗi postgres (×5) | 256M | workload nhỏ, chỉ inbox/outbox |
| rabbitmq | 512M | |
| loki + grafana + promtail | 256M mỗi cái | quan sát, không được cạnh tranh với sản xuất |
| rendering | **hạ 5G → 4.5G** | để chừa chỗ cho phần còn lại |
| video-assembly | giữ 3G nhưng chỉ khi đã có QoS=1 | |

Đồng thời cân nhắc nâng RAM của Docker Desktop VM từ 8G lên 12G — máy có 16 GiB vật lý, VM
đang chỉ lấy một nửa.

**3. Hạ `cpus` xuống tổng ≤ 10.** Hiện rendering 6 + video-assembly 6 = 12 trên VM 10 core.
Đề xuất rendering 5 / video-assembly 3, chừa 2 core cho hạ tầng.

### P1 — Nên làm sớm

**4. Tách `validate_script` ra khỏi queue `rendering.commands`.** Đây là cách sửa gốc cho
head-of-line blocking ở 2.2. Lượt dry 2000s và lượt render 276s có đặc tính hoàn toàn khác nhau
nhưng đang tranh nhau đúng một suất prefetch. Đề xuất: queue riêng `rendering.validate` với
consumer riêng (cùng container cũng được, miễn khác channel + khác QoS budget). Lợi ích trực
tiếp: một project đang dry không còn chặn project khác render.

**5. Thêm trạng thái `queued` vào `ProgressMessage`.** Khi orchestrator dispatch command, phát
`queued` trước; service phát `in_progress` khi thực sự nhận job. Creator nhìn GUI biết ngay mình
đang chờ hàng đợi hay đang được xử lý. Rẻ, và giải quyết phần lớn cảm giác "hệ thống treo".

**6. Thêm DLQ cho `orchestrator.events`.** Cho nó `x-dead-letter-exchange` + `x-delivery-limit`
như 5 queue kia. Tầng code đã vá đúng cho ca đã biết; đây là lưới an toàn cho ca chưa biết.

### P2 — Cải thiện

**7. Cập nhật README.** Ít nhất: 7 bước saga thay vì 5, cổng `awaiting_review`, xoá mục Content
Plugin Service, bổ sung giai đoạn A (LLM authoring + Hive) và Remotion engine.

**8. Dọn project mồ côi.** Xác định lại vòng đời project ở giai đoạn A và thêm cơ chế dọn
project kẹt ở `draft` quá N ngày.

**9. Cân nhắc render queue có ưu tiên.** Khi đã tách được validate (mục 4), bước tiếp theo là
cho `render_scenes` của project mà Creator **đang mở trên GUI** được ưu tiên hơn job nền. Chỉ
làm khi thực sự chạy nhiều project song song.

---

## Kết luận

Kiến trúc saga, Outbox/Inbox hai đầu, và việc xử lý các ca biên khó (`WaitOffsets`, QC không có
nhánh khoá, `IntroDurationSeconds`) cho thấy chất lượng thiết kế cao và các sự cố đã gặp đều
được ghi lại tử tế ngay trong code.

Rendering **không** quá tải — nó là service được bảo vệ kỹ nhất trong hệ thống. Vấn đề thật nằm
ở hai chỗ khác:

1. **video-assembly chưa được áp dụng chính bài học mà rendering đã học** (thiếu `set_qos`) —
   đây là P0.
2. **Ngân sách tài nguyên của cả stack vượt quá VM**: 2 service đã khai hết 100% RAM và 120% CPU,
   19 service còn lại chạy ngoài ngân sách.

Và về trải nghiệm: rendering serialize tuyệt đối 1 job/lần cho toàn hệ thống, nhưng GUI không
phân biệt "đang chờ" với "đang chạy" — nên giới hạn đó hiện ra với Creator dưới dạng một hệ
thống trông như bị treo.
