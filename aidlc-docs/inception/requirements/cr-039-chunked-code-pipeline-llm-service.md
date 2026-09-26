# CR-039 — Bước 1c chia lô + tự kiểm tra biên dịch, và service LLM Python dùng OpenAI SDK

## Date
2026-09-26

## Stage
Requirements Analysis (Change Request) — chờ Creator duyệt thiết kế trước khi code

## Intent Analysis
- **Request type**: Tái cấu trúc độ tin cậy (1c hay hỏng) + đổi cách gọi Hive
- **Scope estimate**: 4 unit — `llm-service` (MỚI, Python), `orchestrator`, `rendering`, `web-gui` (chỉ hiển thị tiến độ theo lô)
- **Complexity estimate**: Cao. Đổi định dạng output của 1b, thêm một service, thêm một hợp đồng HTTP mới với `rendering`.

## Bối cảnh — vì sao 1c hay hỏng
Hiện 1c là **một lần gọi LLM** (`generate_authoring.go`) và model phải viết cả file trong một lượt. Những lỗi đã thấy:
1. Bị cắt ở `max_tokens`, hoặc reasoning dùng hết budget trước khi viết được chữ nào (D13, `ErrKindBudget`/`ErrKindTruncated`).
2. `narrations.length ≠ SHOTS.length`, hoặc sai thứ tự so với storyboard.
3. TSX/Python sai cú pháp, sai kiểu dữ liệu, import thừa. Những lỗi này chỉ lộ ra ở saga (`parse_and_validate_script`), sau khi Creator đã bấm chạy.
4. Storyboard 1b là văn xuôi, nên 1c phải tự đọc hiểu cấu trúc (shot nào, bao nhiêu shot) và đọc lệch là chuyện thường.

## Luồng mới

```
STORY ─▶ Story Architect (1a) ─▶ Visual Director (1b) ─▶ Storyboard JSON (validate schema)
                                                              │
                                             chia lô 10 shot (theo thứ tự)
                                         ┌────────────┬───────┴──────┐
                                      Lô 1..10     Lô 11..20   ...  (song song, có trần)
                                         │ DeepSeek    │ DeepSeek
                                         └──────┬──────┘
                                           Code Merger (deterministic, không LLM)
                                                │
                                  Kiểm tra: Remotion → tsc --noEmit │ Manim → lượt dry
                                        ┌───────┴───────┐
                                      PASS            FAIL ─▶ Repair Agent (chỉ sửa shot lỗi, ≤3 vòng) ─▶ kiểm tra lại
                                        │
                                  lưu vào bước 1c
```

## Quyết định (Creator chốt 2026-09-26)

| # | Quyết định | Lựa chọn |
|---|---|---|
| D1 | Gọi Hive bằng SDK | **Service Python mới `llm-service`**, dùng `openai` SDK (`base_url=https://api-cdn.thehive.ai/api/v3/`, `stream=True`, `extra_headers={"Accept": "text/event-stream"}`) |
| D2 | Chạy `tsc` ở đâu | **Trong `rendering`**, vì image này đã có Node, `typescript` và `remotion_project` |
| D3 | Engine | **Cả Remotion và Manim** |
| D4 | Tham số | **10 shot/lô, tối đa 3 vòng Repair**, chỉnh được qua biến môi trường |

## Yêu cầu chức năng

### FR100 — `llm-service` (MỚI, Python + FastAPI)
- **FR100.1** Là nơi duy nhất cầm `HIVE_API_KEY`. Gọi Hive bằng `openai.OpenAI(base_url=HIVE_BASE_URL, api_key=HIVE_API_KEY)`, luôn `stream=True`.
- **FR100.2** `POST /v1/chat`: nhận `{model, system, user, max_tokens, temperature}` và trả về stream NDJSON gồm các sự kiện `progress {reasoning_chars, content_chars}`, sau đó `result {content, usage}` hoặc `error {kind, message, diag, usage, partial}`. Endpoint này thay `hive_client.go` cho các bước 1a/1b và các lượt gọi LLM khác.
- **FR100.3** Giữ nguyên toàn bộ hành vi mà `hive_client.go` đã được đo và sửa:
  - bảng phân loại lỗi `ErrKind*`;
  - đọc cả hai dạng `reasoning_tokens` (top-level ở GLM, nested ở DeepSeek) và `cached_tokens`;
  - phát hiện error chunk nằm giữa stream;
  - D13: nội dung rỗng mà đã dùng hết budget thì là lỗi budget, không phải lỗi empty;
  - chỉ retry khi gặp 429/5xx, backoff có jitter.
  Phần diag vẫn gồm model, max_tokens, elapsed, http status, request id, finish_reason và số chunk.
- **FR100.4** **Sửa lỗi phân loại**: tài liệu Hive ghi `405 = Out of Balance`. Cần map 405 → `balance`. `hive_client.go` hiện chỉ map 402, nên 405 đang bị coi là `malformed`.
- **FR100.5** Tôn trọng trần 5 req/s của Hive bằng một token-bucket dùng chung cho mọi lượt gọi, kể cả các lô chạy song song.
- **FR100.7** **Gom mọi kết nối LLM về `llm-service`** (Creator yêu cầu 2026-09-26). Sau CR này orchestrator **không còn** client HTTP nào tới Hive hay Ollama; chỉ `llm-service` biết `HIVE_*` và `OLLAMA_*`.
  - Provider trong `llm-service`: `hive` và `ollama`, cả hai đều gọi qua `openai` SDK (Ollama có endpoint tương thích OpenAI tại `{OLLAMA_URL}/v1`, JSON mode qua `response_format`).
  - `POST /v1/suggest-metadata` và `POST /v1/suggest-short-script` thay `OllamaClient.Suggest` / `SuggestShortScript`. Prompt của hai việc này chuyển từ `ollama_client.go` sang Python, giữ nguyên nội dung, retry 2 lần và các giới hạn (`maxScriptChars=4000`, tiêu đề ≤100 ký tự tính theo rune, `normalizeTags`).
  - Xoá `hive_client.go`, `ollama_client.go`, `ollama_provider.go` cùng test cũ; test tương đương được viết lại ở `llm-service`.
  - Các use case `SuggestPublishMetadata`, `SuggestShortScript` giữ nguyên port, chỉ đổi adapter sang `llm_service_client`.
- **FR100.6** `hive_client.go` bị thay bằng `llm_service_client.go` (adapter HTTP, vẫn implement `LLMProviderPort`). Ghi usage (`LLMUsageRecorder`), `project_errors`, khoá double-click và lưu bước vẫn nằm ở orchestrator, không đổi.

### FR101 — Storyboard JSON (bước 1b)
- **FR101.1** Prompt Visual Director đổi phần OUTPUT sang **một khối JSON** theo schema:
  ```json
  {
    "hero": "…", "world": null,
    "palette": [{"role": "accent", "hex": "#F5B841", "meaning": "…"}],
    "scenes": [{
      "id": "beat-1", "title": "…", "invariant": "…", "transition_in": null, "mood": "…", "end_frame": "…",
      "shots": [{"id": "1.1", "camera": "…", "visual": "…", "narration": "…"}]
    }]
  }
  ```
  Phần hướng dẫn sáng tác (luật đạo diễn, tự kiểm tra) giữ nguyên.
- **FR101.2** `llm-service` parse và validate schema. Các điều kiện: id shot không trùng nhau, `narration` không rỗng, hex đúng dạng `#RRGGBB`, mọi màu tham chiếu tới vai trò có trong `palette`. JSON hỏng thì chạy **một lượt sửa JSON** (gửi kèm lỗi parse). Vẫn hỏng thì báo lỗi `malformed`.
- **FR101.3** Bản lưu của 1b là JSON. web-gui hiển thị bản văn xuôi dựng lại từ JSON (định dạng `CẢNH n / n.m | MÁY | HÌNH | THOẠI` như cũ) để Creator đọc và sửa được. Creator dán văn xuôi kiểu cũ (luồng Copy) thì 1c báo "storyboard chưa ở dạng JSON" và đề nghị chạy lại 1b bằng AI.
- **FR101.4** Theo memory "re-seed DB after prompt seed edits": sửa seed xong phải rebuild và restart orchestrator trong cùng lượt, đồng thời kiểm tra `prompt_overrides` có đang che bản mới không.

### FR102 — Chia lô và sinh code song song (bước 1c)
- **FR102.1** Chia các shot theo thứ tự thành lô `CODE_CHUNK_SHOTS=10`. Lô cuối có thể ít hơn.
- **FR102.2** Mỗi lô gọi model của bước `code` với một prompt con gồm:
  - bộ luật Engineer như hiện nay (mục A–G / luật Manim);
  - `PALETTE` và `LAYOUT` chung;
  - JSON của các shot trong lô;
  - **shot cuối của lô trước và `end_frame` của cảnh đó**, để chuyển cảnh biến hình ở ranh giới lô vẫn khớp.
  
  Model **chỉ** trả về các hàm shot:
  - Remotion: `function ShotN_M({duration}: ShotProps) {…}`, mỗi hàm kèm comment `// Shot n.m`.
  - Manim: `def shot_N_M(self):`, gồm các animation cùng `self.narrate("…")` với nguyên văn narration (tách câu theo luật Manim thì được, nhưng không đổi chữ).
- **FR102.3** `LAYOUT` dùng chung giữa các lô được sinh ở **một lượt LLM nhỏ chạy trước** (lượt "Layout", nhận toàn bộ storyboard JSON và chỉ trả về object `LAYOUT`). Nhờ vậy các lô chạy song song không tự đặt toạ độ riêng cho cùng một nhân vật chính.
- **FR102.4** Chạy song song với trần `CODE_CHUNK_CONCURRENCY=3` (nằm dưới token-bucket của FR100.5). Một lô lỗi retryable thì retry riêng lô đó. Lô lỗi không retryable làm cả lượt hỏng, nhưng các lô đã xong được giữ lại để lần chạy sau dùng lại (cache theo hash của nội dung lô).
- **FR102.5** Tiến độ: web-gui thấy được `layout → lô i/N → merge → kiểm tra → sửa vòng k/3`, mở rộng từ `AuthoringProgress` hiện có.

### FR103 — Code Merger (deterministic)
- **FR103.1** Do code tự sinh, không qua LLM:
  - Remotion: imports cố định; `PALETTE` từ `palette` của JSON; `LAYOUT`; `export const narrations` **lấy thẳng từ JSON**; `SHOTS = [Shot1_1, …]` theo đúng thứ tự JSON; `CreatorComposition` và `registerRoot` như khuôn hiện tại.
  - Manim: `class <Topic>Scene(ConceptFlowScene)`, trong `construct` gọi lần lượt `self.shot_1_1()` … và `self.beat("<scene id>")` ở đầu mỗi cảnh; các hàm shot là method của class.
- **FR103.2** Nhờ vậy, số narration khớp số shot, đúng thứ tự và đúng `id="creator"` **theo cấu trúc**, không còn phụ thuộc vào việc model tự đếm.
- **FR103.3** Kiểm tra tĩnh trước khi biên dịch: mỗi shot trong JSON có đúng một hàm, không có hàm lạ, không còn code fence hay chữ trần. Hàm thiếu thì gửi lô đó chạy lại một lần.

### FR104 — Kiểm tra biên dịch trong `rendering`
- **FR104.1** `rendering` hiện là worker RabbitMQ, chưa có HTTP. Cần thêm một HTTP server nhỏ (FastAPI/uvicorn trong cùng process, chỉ mở trong mạng compose) với:
  - `POST /v1/check/remotion {code}`: ghi vào thư mục tạm cạnh `remotion_project/src` rồi chạy `tsc --noEmit -p` với tsconfig của project, timeout 60s. Kết quả `{ok, diagnostics: [{line, col, code, message}]}`.
  - `POST /v1/check/manim {code, scene_class_name}`: gọi lại **`ValidateScriptUseCase`** đang có (lint + lượt dry). Trả về cùng dạng, kèm `narrations` của lượt dry.
- **FR104.2** Remotion cũng chạy lint Lottie id (CR-038) trong cùng endpoint, để lỗi này được phát hiện trước saga.
- **FR104.3** Giới hạn: tối đa 2 lượt kiểm tra chạy cùng lúc. Lượt dry của Manim nặng nên chịu chung trần này.

### FR105 — Repair Agent
- **FR105.1** Map dòng lỗi của diagnostics về hàm shot chứa dòng đó (Merger giữ bảng `shot id → khoảng dòng`). Lỗi nằm ở phần khung do Merger sinh là bug của hệ thống, nên báo lỗi ngay, không gửi cho LLM.
- **FR105.2** Mỗi vòng chỉ gửi **các hàm shot lỗi** + diagnostics + `PALETTE`/`LAYOUT` + JSON của shot đó. Model trả về đúng các hàm đã sửa. Merge lại rồi kiểm tra lại.
- **FR105.3** Tối đa `CODE_REPAIR_MAX_ROUNDS=3`. Hết số vòng mà vẫn FAIL thì **vẫn lưu code** (Creator đã trả tiền token và có thể tự sửa), đánh dấu `check_failed` kèm diagnostics trong `project_errors` và trên GUI.
- **FR105.4** Mọi lượt gọi (layout, từng lô, từng lần repair) đều ghi một dòng usage riêng, gắn nhãn `step=code, phase=chunk|layout|repair`.

## Ngoài phạm vi
- Không đổi saga render. `parse_and_validate_script` vẫn chạy như cũ, nên lỗi ngoài tsc (runtime) vẫn được bắt ở đó.
- Không lint bố cục (đè chữ, tràn khung) cho Remotion ở CR này.
- Không đổi bước 1a.

## Rủi ro và cách giảm
| Rủi ro | Giảm |
|---|---|
| Chuyển cảnh ở ranh giới lô bị lệch | FR102.2 gửi shot cuối của lô trước; FR102.3 dùng chung `LAYOUT` |
| Thêm một service thì thêm một điểm hỏng | Healthcheck compose. Orchestrator phân loại lỗi `llm-service` không kết nối được thành `server`; luồng Copy vẫn là đường dự phòng (FR83.2) |
| Chi phí tăng vì có nhiều lượt gọi | Mỗi lô ngắn hơn và ít bị cắt, nên ít phải chạy lại cả bước. Lượt repair chỉ gửi shot lỗi. Theo dõi qua màn usage theo `phase` |
| Storyboard cũ dạng văn xuôi | FR101.3 báo rõ và đề nghị chạy lại 1b |
| `tsc` bắt thiếu lỗi runtime của Remotion | Chấp nhận. Saga vẫn là chốt cuối |

## Kế hoạch triển khai (sau khi duyệt)
1. `llm-service`: `/v1/chat` + test chuyển từ `hive_client_*_test.go` sang (phân loại lỗi, hai dạng usage, error chunk, D13, 405).
2. Orchestrator: `llm_service_client.go`, thay wiring ở `main.go`/`config.go`, xoá `hive_client.go`.
3. Prompt 1b → JSON, validator, bản văn xuôi dựng lại để hiển thị; re-seed.
4. `rendering`: HTTP `/v1/check/*`.
5. `llm-service`: pipeline 1c (layout → lô → merger → check → repair) cho Remotion, sau đó Manim.
6. web-gui: tiến độ theo phase, trạng thái `check_failed`.
7. Rebuild và restart các service bị ảnh hưởng, rồi chạy E2E một project cho mỗi engine.

## Cập nhật thiết kế (Creator chốt 2026-09-26) — tách hai luồng

Đổi 1b sang JSON làm hỏng nút Copy prompt và việc Creator tự sửa/dán storyboard, nên FR101 và FR102 được điều chỉnh:

- **Luồng thủ công (Copy prompt) giữ nguyên hoàn toàn**: các role `visual_director`, `manim_engineer`, `remotion_engineer` không đổi một chữ; storyboard vẫn là văn xuôi; code vẫn sinh cả file một lượt ở ngoài hệ thống.
- **Luồng AI ("Chạy bằng AI") đi luồng mới** với ba role riêng: `visual_director_ai` (xuất JSON), `remotion_engineer_ai`, `manim_engineer_ai` (chỉ xuất các hàm shot, không xuất cả file). Ba role này là prompt trong thư viện như các role khác, nên vẫn sửa được trên màn quản lý prompt.
- Phần luật dùng chung (bảng màu, bố cục, API Manim, catalog Lottie...) được tách thành hằng số dùng chung để hai luồng không lệch nhau. Có test khẳng định văn bản render của ba role cũ **không đổi** sau khi tách.
- Storyboard của luồng AI lưu ở cột storyboard hiện có dưới dạng JSON. Bước 1c của luồng AI thấy storyboard không phải JSON thì dừng và báo rõ ("storyboard này viết theo luồng thủ công; hãy chạy lại 1b bằng AI"), không đoán.
- **Gợi ý để hợp nhất hai luồng sau này** (chưa làm ở CR này): Creator có thể Copy prompt của `visual_director_ai` để nhận JSON từ một AI bên ngoài rồi dán vào; luồng AI sẽ chạy tiếp bình thường. Khi đó chỉ còn khác nhau ở chỗ ai gọi model.
- Hiển thị: editor bước 1b của luồng AI hiện JSON (đã format). Bản văn xuôi dựng lại từ JSON làm sau nếu Creator thấy khó đọc.

## Trạng thái triển khai (2026-09-26)

Đã làm và kiểm chứng:
- `llm-service` (Python, `openai` SDK): `/v1/chat`, `/v1/suggest-metadata`, `/v1/suggest-short-script`, `/v1/storyboard/finalize`, `/v1/code/generate`. 72 test.
- Orchestrator không còn client Hive/Ollama: chỉ `llm.Client` gọi `llm-service`. Đã sửa mã 405 = hết số dư.
- Ba role AI mới (`visual_director_ai`, `manim_engineer_ai`, `remotion_engineer_ai`); ba role thủ công không đổi một byte (test golden SHA-256).
- `rendering`: `POST /v1/check/{remotion,manim}`; `tsc --noEmit` thật. Phát hiện và sửa hai lỗi nền: thiếu `@types/react`, `SegmentsProps` là interface.
- Cột `llm_usage.phase`; tiến độ theo lô và vòng sửa; web-gui hiển thị `check_failed`.
- E2E thật (Remotion, 3 shot, Hive): layout + 1 lô + check PASS, usage ghi đúng theo phase.

Chưa kiểm chứng: engine Manim với Hive thật (chỉ có test đơn vị và kiểm tra lượt dry qua fake); lô > 10 shot chạy song song với Hive thật; vòng Repair với lỗi thật do model sinh ra.
Chưa làm: hiển thị bản văn xuôi dựng lại từ JSON storyboard (editor 1b hiện JSON).

## Tối ưu thời gian bước Code (2026-09-26)

Creator báo bước 5 (Code) chạy quá lâu. Đường găng của một lượt là `Layout → ⌈số lô / trần song song⌉ × thời gian một lô → kiểm tra → các vòng repair`. Bốn thay đổi:

| # | Thay đổi | Ở đâu |
|---|---|---|
| A | Mặc định `CODE_CHUNK_SHOTS=5` (trước 10), `CODE_CHUNK_CONCURRENCY=10` (trước 3). Lô nhỏ hơn thì lô chậm nhất xong sớm hơn; video thường gặp chạy mọi lô cùng lúc. Trần 5 req/s của FR100.5 giữ nguyên, 429 vẫn được retry. | `docker-compose.yml`, `llm-service/app/config.py`, `.env.example` |
| B | `visual_director_ai` v3 xuất thêm `"layout"` (toạ độ tâm px của các vật xuyên suốt, trong vùng an toàn và ngoài `{{subtitle_zone}}`). Storyboard có `layout` hợp lệ thì Remotion **bỏ qua lượt gọi LAYOUT**, và Merger ghi `const LAYOUT` thẳng từ JSON. Storyboard cũ không có `layout` vẫn đi đường cũ. Manim vẫn cần lượt `setup_cast`, vì đó là code Python. | `storyboard.py` (validate `layout`), `merger.layout_from_storyboard`, seed `visual_director_ai` v3 |
| C | Remotion: mỗi lô viết xong được kiểm tra ngay (các shot của lô khác thay bằng stub `return null`) và sửa ngay các shot lỗi **trong lúc các lô khác còn đang sinh**. Số vòng đã dùng ở lô được tính vào cùng trần `CODE_REPAIR_MAX_ROUNDS`. Lượt kiểm tra cả file cuối cùng vẫn là cổng. Checker không liên lạc được ở bước lô thì bỏ qua, để lượt cuối quyết định. Sự kiện tiến độ mới: `chunk_repair`. Manim giữ kiểm tra cả file, vì lượt dry của một shot phụ thuộc trạng thái khung hình do các shot trước để lại. | `pipeline/run.py` (`_settle_chunk`), `merger.merge_remotion(stub_missing=True)` |
| D | `tsc` không còn khởi động lại mỗi lần kiểm tra. Một process Node chạy nền (`remotion_project/tscheck.mjs`) giữ sẵn khai báo kiểu của React/Remotion/lib, và chỉ kiểm tra script (không ghi file ra đĩa). Đo trên file 30 shot: khoảng 1,1s/lần xuống 0,13–0,19s/lần (lần đầu ~1,3s, được làm nóng lúc service khởi động). Kết quả lỗi trùng với `tsc --noEmit`. Process chết hoặc treo thì báo lỗi (không bao giờ PASS) và được khởi động lại ở lần sau. | `rendering/adapters/rendering/typescript_checker.py`, `main.py` |

Sau khi triển khai: rebuild `llm-service`, `rendering`, `authoring-service` (vì seed prompt đổi), restart. Kiểm tra `prompt_overrides` có đang che `visual_director_ai` không; nếu có, bản mới sẽ không có `layout` và bước Code quay về lượt gọi LAYOUT (vẫn đúng, chỉ chậm hơn).
