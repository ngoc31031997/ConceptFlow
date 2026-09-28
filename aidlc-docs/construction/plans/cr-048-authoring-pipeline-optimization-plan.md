# CR-048 — Tối ưu pipeline authoring AI (1a → 1b → minh hoạ → 1c code → kiểm tra)

Trạng thái: **PLAN — chờ Creator duyệt**. Mỗi task bên dưới tự đủ ngữ cảnh để giao cho một agent riêng.

## 1. Bối cảnh

Lượt chạy lỗi ngày 2026-09-28 (bước code, chunk `3.3-3.7`, model `zai-org/glm-5.3-flash`):

```
request: max_tokens=128000 temperature=0.3 system_chars=52489 user_chars=9112
stream: reasoning_chars=380182 content_chars=0 finish_reason=length   elapsed: 780s
usage (cả lượt chạy): prompt=295687 completion=872830 reasoning=825778
```

- **95% token đầu ra là reasoning.** Chi phí và thời gian nằm ở việc model tự suy luận, không nằm ở code nó viết.
- Một chunk kẹt reasoning 13 phút làm hỏng cả lượt chạy; các chunk khác đang chạy song song bị huỷ, token đã trả tiền bị bỏ.
- Nguyên nhân gốc về prompt: system prompt `remotion_engineer_ai` bắt model **tự tính** hộp bao, bề rộng chữ, vị trí sau zoom cho từng vật (luật F L1/L4/L5/L8/L14, tự kiểm G6/G7/7b) vì Remotion **không có kiểm tra bố cục tự động** — chỉ có `tsc` + lint (`services/rendering/application/check_script.py`). Model có reasoning làm thật các phép tính đó và lặp "soát → sửa → soát lại".
- Mỗi lượt gọi chunk/repair mang theo **toàn bộ dàn ý bước 1a** trong system prompt (`render_prompt.go:279-282`), dù user turn của chunk đã có đủ `invariant/narration/visual`.

Đã làm (nhánh `claude/practical-dirac-y648rh`): mặc định `CODE_CHUNK_SHOTS` 5 → 3.

## 2. Nguyên tắc chung cho mọi agent

1. Đọc `CLAUDE.MD` ở gốc repo và tuân thủ: không code giả / không bỏ qua im lặng; dừng lại hỏi khi thiếu thông tin; báo trung thực test nào chạy, test nào không.
2. Nhánh: mỗi task một nhánh `feature/cr-048-<task-id>-<slug>` tách từ `main` (vd. `feature/cr-048-t1-reasoning-guard`). Không làm trên nhánh của task khác.
3. Chạy test của service bị sửa: `llm-service` → `pytest` trong `services/llm-service`; `authoring-service` → `go test ./...`; `rendering` → `pytest` trong `services/rendering`.
4. Sửa prompt: system prompt nằm ở `services/authoring-service/internal/domain/prompt_template_seeds*.go`. Khi đổi text phải **tăng `Version`** của role đó trong `DefaultPromptTemplates()` và cập nhật `prompt_golden_test.go` nếu nó khoá text. Bản system trong DB tự cập nhật khi service khởi động (`SeedPrompts` dùng `ON CONFLICT DO UPDATE`), nhưng **bản tuỳ chỉnh Creator đang bật (is_active, không phải system) sẽ không đổi** — ghi rõ điều này trong báo cáo.
5. **Chỉ sửa luồng AI** (`*_ai` roles, `llm-service`). Luồng thủ công (Copy prompt: `visual_director`, `remotion_engineer`, `manim_engineer`) giữ nguyên trừ khi task nói khác.
6. Sau khi xong: rebuild + restart service bị ảnh hưởng (`docker compose build <svc> && docker compose up -d <svc>`) nếu môi trường có Docker; nếu không có thì ghi "chưa rebuild".
7. Báo cáo cuối task: file đã sửa, test đã chạy (kết quả thật), tiêu chí nghiệm thu nào đạt / chưa đạt.

## 3. Thứ tự và phụ thuộc

```
Đợt 1 (song song, không đụng file của nhau):
  T1 reasoning guard ........ llm-service/provider.py, config.py, main.py
  T2 cô lập chunk + chia đôi  llm-service/pipeline/run.py (+ Go nhận event mới)
  T3 bớt ngữ cảnh bước code . authoring render_prompt.go + llm-service prompts.py
  T4 rút gọn tự kiểm tra .... authoring prompt_template_seeds_ai.go
  T5 spike tắt reasoning .... chỉ đo, chưa sửa code (trừ khi spike thành công)
  T6a spike đo bố cục ....... rendering/remotion_project (prototype)
Đợt 2:
  T6b kiểm tra bố cục thật ... (cần T6a)
  T8 kiểm tra độ dài lời thoại sau 1b
  T9 cảnh báo luật 19 sau 1b
Đợt 3:
  T7 rút gọn luật bố cục trong prompt AI (cần T6b đã chạy ổn)
  T10 đo lại trước/sau (cần T1–T4, lý tưởng cả T6b/T7)
```

Xung đột file dự kiến: T3 và T4 đều liên quan prompt code AI — T3 **không** sửa `prompt_template_seeds_ai.go` (phần đổi tiêu đề mục nằm trong T4). T2 và T1 cùng package `llm-service` nhưng khác file.

---

## T1 — Giới hạn reasoning khi chưa viết được chữ nào

**Mục tiêu:** một lượt gọi đã suy nghĩ quá N ký tự mà `content_chars == 0` thì dừng stream ngay, thay vì chờ tới `max_tokens` (13 phút).

**File:** `services/llm-service/app/provider.py`, `app/config.py`, `app/main.py`, `app/pipeline/run.py` (chỉ chỗ tạo `ChatRequest`), `tests/test_provider.py`, `docker-compose.yml`, `.env.example`.

**Việc cần làm:**
1. Thêm `ChatRequest.max_reasoning_chars: int = 0` (0 = tắt).
2. Trong `Provider._once`, trong vòng `async for chunk in stream`: nếu `max_reasoning_chars > 0` và `reasoning_chars > max_reasoning_chars` và `content_chars == 0` → đóng stream (`await stream.close()` hoặc tương đương của SDK), raise `LLMError` kind `errors.BUDGET`, `retryable=False`, message nói rõ: `"model suy nghĩ quá {N} ký tự mà chưa viết được chữ nào — dừng sớm"`. Diag ghi `stream: ... aborted=reasoning_limit`.
3. Usage: stream bị cắt nên provider không gửi usage. **Không bịa số.** Trả `Usage(model=...)` rỗng và ghi trong diag `usage: not reported (stream aborted), reasoning_chars=<n>`.
4. Config: `CODE_MAX_REASONING_CHARS` (mặc định `60000`) dùng cho mọi lượt gọi trong `CodePipeline` (layout/cast/chunk/repair); `CHAT_MAX_REASONING_CHARS` (mặc định `0` = tắt) cho `/v1/chat` (bước 1a/1b vốn suy nghĩ dài hợp lệ). Truyền qua `CodeRequest`/`CodePipeline` như `max_tokens`.
5. Thêm biến vào `docker-compose.yml` (service llm-service) và `.env.example` kèm chú thích tiếng Việt.

**Không làm:** không thêm retry trong provider (T2 lo việc chia chunk và thử lại). Không đổi kind mới — dùng lại `budget` để phía Go (`ErrKindBudget`) không phải sửa.

**Nghiệm thu:**
- Test: stream giả phát 70k ký tự `reasoning_content` không có `content` với giới hạn 60k → raise `budget`, số chunk đã đọc < tổng số chunk (đã dừng sớm).
- Test: giới hạn 0 → hành vi cũ không đổi.
- Test: đã có `content` rồi mới vượt ngưỡng reasoning → **không** dừng.
- Toàn bộ `pytest` của llm-service xanh.

---

## T2 — Một chunk hỏng không kéo cả lượt chạy chết; chia đôi chunk khi hết budget

**Mục tiêu:** tận dụng token đã trả, và tự vượt qua chunk "quá khó" bằng cách chia nhỏ.

**File:** `services/llm-service/app/pipeline/run.py`, `tests/test_pipeline.py`; phía Go nếu thêm event: `services/authoring-service/internal/application/code_pipeline.go`, `generate_authoring_code.go` (hàm cập nhật progress), test tương ứng.

**Hiện trạng:** `run.py:328-335` — `asyncio.gather(*tasks)` không `return_exceptions`; chunk đầu tiên lỗi → huỷ mọi task khác → kết quả đang chạy dở bị mất, không vào `ChunkCache`.

**Việc cần làm:**
1. Khi một chunk lỗi: **để các chunk đang chạy chạy xong** (kết quả thành công được `ChunkCache.put` như bình thường trong `_ask`), không khởi động chunk mới chưa bắt đầu, rồi raise `PipelineFailure` của chunk lỗi đầu tiên với `calls` đầy đủ. Ngoại lệ: lỗi `auth`/`balance`/`not_configured` → huỷ ngay như hiện tại (chạy tiếp chỉ tốn tiền vô ích). Huỷ do người dùng (CancelledError) → huỷ ngay.
2. Chia đôi khi hết budget: trong `do_chunk`, nếu `_ask` raise `PipelineFailure` có `kind in (budget, truncated)` và chunk có > 1 shot → chia `ids` làm hai nửa, gọi lần lượt hai nửa (mỗi nửa có `prev`/`nxt` đúng), gộp kết quả. Đệ quy tới 1 shot; 1 shot vẫn lỗi → fail như cũ. Mọi lượt gọi (kể cả lượt lỗi) vẫn nằm trong `calls`.
3. Emit event `{"type": "chunk_split", "index", "shots", "into": [[...],[...]]}`. Kiểm tra phía Go (`code_pipeline.go`, progress) xử lý event lạ thế nào: nếu bỏ qua an toàn thì chỉ cần hiển thị thêm (tuỳ chọn); nếu lỗi thì sửa cho nhận event này.
4. Cache key của nửa chunk dùng đúng danh sách shot của nửa đó (hàm `_key` hiện đã gồm `",".join(ids)`).

**Nghiệm thu (test với provider giả):**
- 3 chunk, chunk 2 lỗi `server` → chunk 1 và 3 chạy xong và nằm trong cache; lần chạy lại chỉ gọi model cho chunk 2.
- Chunk 5 shot lỗi `budget` → được chia 2+3 (hoặc 3+2), cả hai thành công → run thành công, `calls` có lượt lỗi + các lượt con.
- Chunk 1 shot lỗi `budget` → run fail với kind `budget`.
- Lỗi `auth` → các chunk khác bị huỷ ngay.
- `pytest` llm-service xanh; `go test ./...` authoring-service xanh nếu có sửa Go.

---

## T3 — Bớt ngữ cảnh thừa của bước code

**Mục tiêu:** system prompt bước code không mang cả dàn ý 1a; chỉ mang phần cốt lõi. Giảm token mọi lượt gọi chunk/repair và bớt lý do để model suy nghĩ lan sang shot khác.

**File:** `services/authoring-service/internal/application/render_prompt.go` (`previousOutputFor`), test `render_prompt_test.go`; `services/llm-service/app/pipeline/prompts.py`, `tests/test_prompts*.py` (hoặc test pipeline tương ứng).

**Việc cần làm:**
1. Với `RoleManimEngineerAI` / `RoleRemotionEngineerAI`: thay vì `add(story)` toàn văn, trích từ output 1a các dòng có nhãn cố định (định dạng output ở `storyArchitectVI`, `prompt_template_seeds.go:270-320`):
   `KIỂU VIDEO:`, `CÂU HỎI CỐT LÕI:`, `INSIGHT CỐT LÕI:`, `SAI LẦM TRỰC GIÁC:`, `AHA MOMENT:` (kèm 2 dòng con "Tôi từng nghĩ"/"Nhưng bây giờ tôi nhận ra"), và `Thế giới chính:` trong `KHUNG BÀI`.
2. Nếu **không trích được** nhãn nào (Creator sửa tay dàn ý, đổi format) → dùng lại toàn văn như cũ **và** `slog.Warn` kèm project_id. Không im lặng cắt nội dung.
3. Luồng thủ công (`RoleManimEngineer`, `RoleRemotionEngineer`) giữ nguyên.
4. `prompts.py`: thêm `_world(sb)` (dòng NHÂN VẬT CHÍNH / THẾ GIỚI) vào đầu user turn của `remotion_chunk` và `manim_chunk` (hiện chỉ lượt layout/cast có). Giữ nguyên thứ tự: phần thay đổi theo chunk chỉ nằm ở user turn — **system prompt phải giống hệt nhau giữa các lượt gọi** để prompt caching của Hive còn tác dụng.

**Nghiệm thu:**
- Test Go: dàn ý mẫu đầy đủ → `{{previous_output}}` của role AI chỉ còn các dòng trên (và ngắn hơn toàn văn); dàn ý không có nhãn → toàn văn + có log cảnh báo; role thủ công không đổi.
- Test Python: user turn chunk có dòng NHÂN VẬT CHÍNH/THẾ GIỚI.
- Ghi vào báo cáo: số ký tự system prompt bước code trước/sau với một dàn ý mẫu.

---

## T4 — Rút gọn phần tự kiểm tra trong prompt code AI

**Mục tiêu:** bỏ các mục bắt model tính toán bằng tay và vòng "sửa hết rồi mới trả lời", nguồn chính của reasoning dài.

**File:** `services/authoring-service/internal/domain/prompt_template_seeds_ai.go`, `DefaultPromptTemplates()` (tăng Version `RoleRemotionEngineerAI`, `RoleManimEngineerAI`), `prompt_golden_test.go` nếu cần.

**Việc cần làm:**
1. `remoGAIVI`: tiêu đề đổi thành "## G. SOÁT MỘT LƯỢT TRƯỚC KHI TRẢ LỜI". Bỏ mục 6 (liệt kê hộp bao), mục 7 (ước lượng bề rộng/chiều cao chữ), mục 7b (tính kích thước tối thiểu). Thay bằng một mục ngắn: "Bố cục: đặt vật theo LAYOUT và luật F; không tự tính lại từng hộp bao — hệ thống kiểm tra bố cục sau khi biên dịch và sẽ gửi lỗi cụ thể nếu có." (câu cuối chỉ giữ nếu T6b đã chạy; nếu chưa thì viết: "chọn phương án an toàn: chừa khoảng cách rộng, chữ ngắn").
2. Thêm cuối mục G (cả Remotion và Manim `manimCheckAIVI`): "Soát ĐÚNG MỘT lượt. Chỗ còn phân vân thì chọn cách đơn giản, an toàn nhất rồi viết code ngay — đừng cân nhắc lại nhiều lần."
3. Đổi tiêu đề mục trong `remoIntroAIVI`/`manimStoryAIVI` từ "CÂU CHUYỆN ĐÃ CHỐT" thành "CỐT LÕI CÂU CHUYỆN (từ Story Architect) — để hiểu ý nghĩa, không phải để dựng thêm cảnh" (khớp với T3).
4. **Không** sửa `remoFVI` (dùng chung với luồng thủ công) — việc rút gọn luật F cho luồng AI là T7.

**Nghiệm thu:** `go test ./...` xanh; diff prompt chỉ đụng các mục nêu trên; Version tăng; báo cáo ghi rõ bản tuỳ chỉnh active (nếu có) không được cập nhật tự động.

---

## T5 — Spike: tắt/giảm reasoning cho bước code

**Mục tiêu:** biết Hive có cho tắt hoặc giảm reasoning theo từng request với `zai-org/glm-5.3-flash` và `deepseek-ai/deepseek-v4.1-flash` không.

**Việc cần làm (cần `HIVE_API_KEY` thật — không có thì DỪNG và báo):**
1. Gọi thử cùng một prompt nhỏ với từng biến thể qua `extra_body`: `{"thinking": {"type": "disabled"}}`, `{"reasoning_effort": "low"}`, `{"enable_thinking": false}`, `{"chat_template_kwargs": {"enable_thinking": false}}`. Ghi lại cho mỗi model: HTTP status, `reasoning_tokens`, có `reasoning_content` không, thời gian.
2. Viết kết quả vào `aidlc-docs/construction/plans/cr-048-t5-reasoning-control-findings.md` (bảng model × tham số).
3. **Chỉ khi có tham số hoạt động:** thêm `ChatRequest.reasoning: str = ""` ("" = mặc định, "off"/"low") ánh xạ sang tham số đúng theo model; `CodePipeline` dùng giá trị từ config `CODE_REASONING` (mặc định "" — Creator bật sau khi so chất lượng). Không bật mặc định khi chưa đo chất lượng code.
4. Giao diện: trong `services/web-gui/src/components/AuthoringModelPicker.tsx` thêm dòng gợi ý khi chọn GLM cho bước Code (đọc `docs/ux-ui-design-rules.md` trước): "Model này hay suy nghĩ rất lâu ở bước dựng code; nên dùng DeepSeek hoặc mặc định."

**Nghiệm thu:** file findings có số đo thật; nếu thêm code thì có test ánh xạ tham số; gợi ý UI hiển thị đúng.

---

## T6a — Spike: đo bố cục shot Remotion trong trình duyệt

**Mục tiêu:** chứng minh cách lấy hộp bao (bounding box) các phần tử của từng shot tại các thời điểm cho trước, chạy được trong container `rendering`.

**Hướng đề xuất (agent kiểm chứng, được đổi nếu có cách tốt hơn):**
- Dùng `@remotion/player` (cùng version `4.0.525` với `remotion` trong `services/rendering/remotion_project/package.json`), component `<Thumbnail>` render đúng một frame của một composition vào DOM thường.
- Harness: bundle file script đã ghép + một trang harness (bằng `@remotion/bundler` hoặc esbuild) → mở bằng Chromium (Playwright/`@remotion/renderer` `openBrowser`) → với mỗi shot i và mỗi frame f ∈ {0, 50%, 85%, 100% của duration giả định, vd. 150 frame} → render `Thumbnail` của riêng `SHOTS[i]` → thu thập `getBoundingClientRect()` của: phần tử lá chứa chữ (kèm `fontSize`, `scrollWidth/clientWidth`), phần tử `svg`, và gốc component bộ minh hoạ.
- Đo thời gian cho một video ~30 shot; mục tiêu < 60 giây.

**Kết quả:** prototype script trong `services/rendering/remotion_project/` + file `aidlc-docs/construction/plans/cr-048-t6a-layout-probe-findings.md` (cách làm, giới hạn, thời gian đo được, ví dụ output JSON). Không nối vào pipeline ở bước này.

---

## T6b — Kiểm tra bố cục tự động trong vòng biên dịch/sửa lỗi

**Phụ thuộc:** T6a đã chốt cách đo.

**Mục tiêu:** lỗi bố cục (đè chữ, tràn khung, lấn vùng phụ đề, chữ nhỏ, vật trọng tâm quá nhỏ) được phát hiện bằng code và gửi vào vòng repair sẵn có, như lỗi `tsc`.

**File:** `services/rendering/application/check_script.py`, adapter mới cạnh `adapters/rendering/typescript_checker.py`, script node trong `remotion_project/`, `Dockerfile` nếu thêm dependency; `services/llm-service/app/pipeline/checker.py` / `run.py` nếu cần phân biệt loại lỗi; test cả hai service.

**Việc cần làm:**
1. Sau `tsc` thành công, chạy đo bố cục (T6a) cho các shot; áp luật lấy từ prompt F: vùng an toàn (96,96)–(1824,984); vùng phụ đề (lấy cấu hình subtitle zone của project — tìm nơi render `{{subtitle_zone}}` để dùng cùng nguồn số liệu); hai khối chữ giao nhau; chữ tràn (`scrollWidth > clientWidth + 1`); `fontSize < 32`; vật lớn nhất của shot < 30% chiều khung (chỉ cảnh báo).
2. Mỗi vi phạm thành một `CheckDiagnostic` với `line` = dòng khai báo hàm shot đó trong file đã ghép, để `_map_failures` (`run.py:446`) gán đúng về shot và vòng repair gửi đúng shot. Message cụ thể, tiếng Việt, có số: vd. `"Shot 3.4, frame 85%: nhãn 'Vi khuẩn axit' tràn khung chữ (rộng 412px > width 360px)"`.
3. Phân mức: vi phạm vùng an toàn/phụ đề, chữ đè chữ, chữ tràn → blocking (đưa vào repair); kích thước tối thiểu → warning (không chặn).
4. Nếu trình duyệt không khởi động được / quá thời gian → trả `CheckerUnavailable`-tương đương cho phần bố cục, **ghi warning rõ ràng** trong kết quả, không coi là "đạt" im lặng.
5. Có biến bật/tắt `LAYOUT_CHECK_ENABLED` (mặc định bật) và timeout `LAYOUT_CHECK_TIMEOUT_SECONDS`.

**Nghiệm thu:** test với script mẫu có (a) chữ tràn, (b) vật ra ngoài vùng an toàn, (c) hai nhãn đè nhau → 3 diagnostic đúng shot; script sạch → không diagnostic; thời gian đo một video ~30 shot được ghi trong báo cáo.

---

## T7 — Rút gọn luật bố cục trong prompt code AI

**Phụ thuộc:** T6b đã chạy ổn trên vài video thật.

**File:** `prompt_template_seeds_ai.go`, `DefaultPromptTemplates()` (tăng Version `RoleRemotionEngineerAI`), golden test.

**Việc cần làm:** tạo `remoFAIVI` riêng cho luồng AI (không sửa `remoFVI` của luồng thủ công): giữ các luật dạng quy tắc (vùng an toàn, một gốc AbsoluteFill, đặt vật theo LAYOUT, chữ có width cố định, cỡ chữ tối thiểu, clamp/inputRange, không CSS animation, SVG có viewBox, ký tự cấm trong JSX, TypeScript sạch); bỏ mọi câu yêu cầu **tính/ước lượng** (công thức `số ký tự × 0.58 × fontSize`, "kiểm tra ở cả vị trí đầu/cuối/lúc to nhất", công thức vị trí sau zoom, "chừa thêm 15%"...). Thêm một câu: "Hệ thống đo bố cục sau khi biên dịch và gửi lỗi cụ thể nếu có." Cập nhật câu bố cục trong mục G (T4) cho khớp.

**Nghiệm thu:** `go test ./...` xanh; so ký tự system prompt trước/sau; chạy lại 1 video thật và so số vòng repair (ghi vào báo cáo).

---

## T8 — Kiểm tra độ dài lời thoại sau bước 1b

**Mục tiêu:** Creator biết ngay cảnh nào dài/ngắn so với ngân sách thời lượng, trước khi tốn tiền cho bước code/TTS. Model đếm chữ tiếng Việt không tin được, nên để code đếm.

**File:** `services/authoring-service/internal/application/generate_authoring.go` (nhánh `step == "storyboard"` sau khi finalize thành công), domain (hàm mới tính thời lượng), test; web-gui chỉ nếu `Warnings` của bước storyboard chưa được hiển thị.

**Việc cần làm:**
1. Parse storyboard JSON đã chuẩn hoá: gom `narration` theo `scene.id` (= id beat).
2. Tính thời lượng ước lượng mỗi cảnh bằng tốc độ đọc đã hiệu chỉnh của giọng (dùng cùng nguồn WPM với `beatsFor` trong `render_prompt_input.go:132-148` và `VoiceCalibration.WordsPerMinute`), so với ngân sách từng beat của format (nguồn số liệu mà `BuildStoryBeatSheetSection` dùng, `prompt_vars.go:195`).
3. Lệch > 20% → thêm `Warnings` vào `GeneratedStep`: `"Cảnh concrete: ~48 giây, ngân sách 35 giây (+37%)"`. Chỉ cảnh báo, không chặn.

**Nghiệm thu:** test đơn vị cho hàm tính (cảnh đúng, dài, ngắn, beat không có trong format → cảnh báo riêng, không panic); cảnh báo hiện trên giao diện bước 1b.

---

## T9 — Cảnh báo "hình không cho thấy cái đang nói" (luật 19) sau bước 1b

**Mục tiêu:** bắt sớm, bằng heuristic rẻ, các shot mà `narration` nói về vật cụ thể có trong bộ minh hoạ nhưng `visual` không nhắc tới vật đó.

**File:** authoring-service (cùng chỗ với T8, file domain mới), dữ liệu từ `internal/domain/prompts/illustration_kit_vi.txt`.

**Việc cần làm:** xây bảng từ khoá tiếng Việt → component từ `illustration_kit_vi.txt` (vd. "răng" → `Tooth`, "vi khuẩn" → `Germ`); với mỗi shot, nếu `narration` chứa từ khoá mà `visual` không chứa từ khoá cùng nhóm → cảnh báo `"Shot 2.3: lời thoại nhắc 'vi khuẩn' nhưng HÌNH không có"`. Chỉ áp dụng Remotion. Chỉ cảnh báo. Bảng từ khoá sinh từ file kit (hoặc khai báo cạnh nó) để không lệch khi kit thay đổi; có test giữ đồng bộ như `lottie_catalog_test.go`.

**Nghiệm thu:** test với storyboard mẫu có 1 shot vi phạm và 1 shot đúng.

---

## T10 — Đo lại trước/sau

**Phụ thuộc:** T1–T4 đã merge (lý tưởng cả T6b/T7).

**Việc cần làm:** chọn 2–3 project mẫu (1 Remotion dài, 1 Manim), chạy bước code trên `main` trước CR-048 và sau; ghi bảng: tổng prompt/completion/reasoning tokens, thời gian, số lượt gọi, số vòng repair, `check_ok`, số lỗi bố cục phát hiện ở T6b. Dùng số từ bảng `llm_usage` và journal project, không ước lượng. Ghi vào `aidlc-docs/construction/plans/cr-048-results.md`.

---

## Backlog (chưa lên task)

- `ChunkCache` chỉ ở bộ nhớ — khởi động lại llm-service là mất cache của lượt chạy lỗi. Cân nhắc lưu Redis/Postgres nếu T2 cho thấy chạy lại thường xuyên.
- Duyệt ảnh keyframe bằng model thị giác cho luật 19 (đắt; chỉ xét sau khi T9 cho thấy heuristic chưa đủ).
