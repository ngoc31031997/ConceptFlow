# CR-063 — Giảm token bước code: trần suy nghĩ, gộp repair, thu gọn system prompt

**Trạng thái**: Creator chốt tách phạm vi ("chắc tách các phần khác ra làm cái nào clear trước đi"): CR này chỉ làm **FR-2 (gộp repair)** và **FR-3 (thu gọn prompt)**. **FR-1 (hiệu chỉnh trần suy nghĩ)** tách ra backlog, chờ Q5.
**Nhánh**: `feature/cr-063-code-gen-token-tuning`

## Phạm vi sau khi tách (quyết định của Creator)

- **Làm trong CR-063**: FR-2 và FR-3 (đã rõ: Q3-a, Q4-a).
- **Tách ra backlog**: FR-1 (đổi mặc định `CODE_MAX_REASONING_CHARS`), vì số đo cho thấy công thức Q2(a) không dùng được và con số mới (Q5) chưa chốt. Phần số đo và đề xuất 100 000 giữ lại bên dưới để làm CR sau.
- **Hệ quả cần biết**: trần vẫn là 60 000 khi repair gộp chạy. Một lượt repair gộp nhiều shot suy nghĩ nhiều hơn một shot, nên dễ chạm trần hơn. Để không tệ hơn hiện nay, FR-2 có thêm quy tắc lùi: lượt repair gộp bị cắt (`budget`/`truncated`) thì nhóm được chia đôi, xuống tới một shot, đúng như cách chunk đang xử lý; mỗi nửa là một lượt riêng.

## Yêu cầu gốc (nguyên văn)

> Hiệu chỉnh CODE_MAX_REASONING_CHARS (CR-056): đang 60 000, cần dựa trên tỉ lệ reasoning_chars / reasoning_tokens đo được trong llm_usage.
> Gộp shot lỗi cùng đoạn vào một lượt repair (CR-056): mỗi lượt repair hiện gửi lại khoảng 21k token system prompt.
> Thu gọn system prompt remotion_engineer_ai (CR-056): hiện 28 500 ký tự.

Trả lời câu hỏi (nguyên văn): "q1 tôi đã bật docker / q2 a / q3 a / q4 a".

- Q1: đo trên `llm_usage` thật (chỉ khởi động `authoring-service-db`, không bật cả stack).
- Q2 (a): giữ biến tính bằng ký tự, đặt từ số đo, ghi cách tính và SQL vào `.env.example`.
- Q3 (a): một lượt repair cho mọi shot lỗi cùng đoạn.
- Q4 (a): chỉ viết gọn câu chữ, giữ nguyên mọi luật.

## Hiện trạng

### 1. Trần suy nghĩ `CODE_MAX_REASONING_CHARS`

- Mặc định 60 000 ([config.py:72](../../../services/llm-service/app/config.py#L72)), áp cho mọi lượt gọi của bước code: layout, cast, chunk, repair ([main.py:329-360](../../../services/llm-service/app/main.py#L329-L360)).
- `provider` đếm độ dài `reasoning_content` trên stream; vượt trần mà chưa viết chữ nào thì đóng stream và báo lỗi `budget` ([provider.py:210](../../../services/llm-service/app/provider.py#L210), [provider.py:240-251](../../../services/llm-service/app/provider.py#L240-L251)).
- Từ CR-056 mỗi lượt gọi ghi `reasoning_chars` và `usage_reported` vào `llm_usage` ([db.go:61-62](../../../services/authoring-service/internal/adapters/postgres/db.go#L61-L62)). Lượt bị cắt có `usage_reported=false` nên không có `reasoning_tokens`.

### 2. Repair từng shot một

- Repair cuối, sau khi ghép cả file ([run.py:767-793](../../../services/llm-service/app/pipeline/run.py#L767-L793)): `_map_failures` nhóm lỗi theo shot, `_repair` gọi **một lượt cho mỗi shot lỗi**, song song trong giới hạn `sem` ([run.py:832-871](../../../services/llm-service/app/pipeline/run.py#L832-L871)).
- Repair sớm của Remotion ([run.py:804-830](../../../services/llm-service/app/pipeline/run.py#L804-L830)): ngay khi một đoạn viết xong, `_settle_chunk` biên dịch đoạn đó rồi gọi `_repair`, cũng mỗi shot một lượt.
- Mỗi lượt gửi nguyên system prompt (`_ask` dùng `req.system`, [run.py:555-558](../../../services/llm-service/app/pipeline/run.py#L555-L558)). User turn gồm shot, lỗi, code cũ, palette, LAYOUT ([prompts.py:130-183](../../../services/llm-service/app/pipeline/prompts.py#L130-L183)). Reply phải đúng một hàm ([run.py:852-855](../../../services/llm-service/app/pipeline/run.py#L852-L855)).

### 3. System prompt `remotion_engineer_ai`

Ghép từ 11 khối ([prompt_template_seeds_ai.go:193](../../../services/authoring-service/internal/domain/prompt_template_seeds_ai.go#L193)). Ký tự, chưa render biến:

| Khối | Ký tự | Ghi chú |
|---|---|---|
| Intro | 1 049 | vai trò + `{{previous_output}}` (cốt lõi câu chuyện, dài tuỳ project) |
| A | 1 998 | bám kịch bản |
| B | 965 | màu |
| C | 1 159 | nền, font, phụ đề + `{{subtitle_zone}}` |
| C2 | 1 383 | cách dùng Lottie + catalog (`lottie_catalog_vi.txt`, **3 554**) |
| C3 | 426 | + bộ minh hoạ (`illustration_kit_vi.txt`, **11 400**) |
| D | 2 704 | khung code + ví dụ một hàm shot |
| E | 1 006 | thư viện import |
| F | 5 791 | luật bố cục L1–L15 + `{{frame_rules}}` |
| G | 2 015 | tự soát |
| Output | 421 | định dạng trả lời |

- Phần cố định khoảng **34 000 ký tự**, chưa tính cốt lõi câu chuyện và `library_section` do llm-service nối thêm ([run.py:446-449](../../../services/llm-service/app/pipeline/run.py#L446-L449)). Con số 28 500 trong backlog đã cũ.
- Hai khối lớn nhất là bộ minh hoạ và luật F. G lặp lại phần lớn A/B/F dưới dạng câu hỏi; D lặp danh sách import mà tin nhắn chunk cũng đã liệt kê ([prompts.py:102-110](../../../services/llm-service/app/pipeline/prompts.py#L102-L110)).
- Seed ở authoring-service, upsert khi khởi động; nếu Creator đang bật một prompt tự sửa cho vai trò này thì seed mới không có tác dụng. `golden_prompts_test.go:87-92` khoá một số câu bắt buộc.

## Số đo `llm_usage` (2026-09-27 → 2026-10-02: 1 064 dòng, 312 lượt thành công có `reasoning_chars`)

**Tỉ lệ ký tự/token** (lượt thành công, `deepseek-v4.1-flash`): chunk trung vị **3,06** (khoảng 2,70–3,31); repair **3,08** (2,01–5,17); `attempt_1` 3,15. Tức khoảng **3,1 ký tự/token**, không phải 4 như CR-056 giả định: trần 60 000 ký tự chỉ bằng khoảng **19 400 token** suy nghĩ.

**Chunk theo `reasoning_chars`** (127 lượt: 91 thành công, 36 bị cắt `budget`):

| Cận trên (ký tự) | 10k | 15k | 20k | 25k | 30k | 35k | 40k | 45k | 50k | 55k | 60k | >60k |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| Thành công | 1 | 3 | 9 | 6 | 10 | 12 | 13 | 11 | 13 | 8 | 5 | 0 |
| Bị cắt | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 36 |

Trung vị 36 492, p90 51 206, p95 55 243, max 59 570. **28% lượt chunk bị cắt**, mỗi lượt bị cắt chạy trung bình 209–283 giây rồi bỏ.

**Repair** (112 thành công, 5 bị cắt): trung vị 3 002 ký tự, p90 26 346, p95 44 987, trung bình khoảng 9 000–10 000 mỗi shot. Mỗi lượt chạy khoảng 36–43 giây; prompt khoảng 22k token (khoảng 16k là cache-hit), completion khoảng 4k.

### Kết luận từ số đo

1. **Phân bố bị kiểm duyệt phía phải.** Mọi lượt thành công đều dưới 60 000 vì chính trần đó cắt phần còn lại. Mật độ ở 50–60k vẫn còn (13, 8, 5 lượt), chưa tắt về 0. Cho nên **công thức "p95 × 1,3" ở Q2(a) không dùng được**: p95 đo được (55 243) chỉ là p95 của phần dưới trần. Muốn biết đuôi phải nâng trần rồi đo lại.
2. **Gộp repair làm tăng suy nghĩ mỗi lượt.** Một shot repair khoảng 10k ký tự; gộp 3 shot khoảng 30k trung bình, đuôi có thể vượt 60k. Giữ trần 60 000 thì repair gộp sẽ bị cắt nhiều hơn hiện nay (5/117 lượt repair một shot đã bị cắt).
3. Lượt bị cắt không ghi token, nên chi phí thật của 36 + 5 lượt bị cắt cao hơn những gì bảng cho thấy.

## Yêu cầu

- **FR-1 (trần suy nghĩ) — TÁCH RA BACKLOG, không làm trong CR này**: nâng mặc định `CODE_MAX_REASONING_CHARS` từ 60 000 lên **100 000** (khoảng 32 000 token ở 3,1 ký tự/token), chờ Creator chốt ở Q5.
  - Lý do: 28% chunk đang chết đúng ở 60 000 trong khi mật độ lượt thành công ở 50–60k chưa tắt; repair gộp cần chỗ rộng hơn. Đây là một **bước nới có đo**, không phải kết quả của công thức: số đo chỉ nói 60 000 quá thấp, chưa nói đuôi dài tới đâu.
  - `.env.example` ghi tỉ lệ đo được (3,1 ký tự/token) và hai câu SQL hiệu chỉnh lại (tỉ lệ; phân bố `reasoning_chars` và số lượt `budget` của chunk/repair). Sau một video thật với trần mới, phân bố không còn bị cắt ở 60k nên lần sau mới đặt được trần theo p99 đo thật.
  - `config.py`, `docker-compose.yml`, `.env.example` đổi mặc định cùng lúc.
- **FR-2 (gộp repair theo đoạn)**: mọi shot lỗi của **cùng một đoạn** được sửa trong **một** lượt gọi.
  - Áp cho repair sớm (`_settle_chunk`) và repair cuối; ở repair cuối, nhóm theo `plan.owner(shot)`.
  - LAYOUT/cast vẫn sửa riêng, mỗi cái một lượt như nay.
  - Một shot lỗi duy nhất dùng đúng prompt và định dạng hiện tại.
  - Reply gộp thiếu hàm nào thì hàm đó giữ code cũ và lỗi được báo ở lần check sau (đúng như repair hiện tại); một hàm thiếu không làm hỏng các hàm còn lại. Reply không có hàm nào dùng được thì lượt đó ghi là `malformed`, thử lại một lần (`EXTRACT_ATTEMPTS`), rồi giữ code cũ.
  - Nhãn lượt (`label`) là danh sách shot nối bằng dấu phẩy, ví dụ `1.3,1.4`, để `llm_usage` vẫn phân biệt được.
  - Lượt gộp bị cắt (`errors.BUDGET`/`errors.TRUNCATED`, tức `SPLIT_KINDS`) thì nhóm chia đôi và thử lại từng nửa, xuống một shot; lỗi khác hoặc nhóm một shot đã bị cắt thì giữ code cũ như hiện nay. Các lỗi dừng cả run (`STOP_NOW_KINDS`) vẫn ném ra ngay.
- **FR-3 (thu gọn prompt, Q4-a)**: viết gọn câu chữ seed `remotion_engineer_ai`, **giữ nguyên mọi luật**. Mục tiêu giảm 25–35% (còn khoảng 22 000–25 000 ký tự).
  - G chỉ còn danh sách kiểm ngắn trỏ về A/B/F thay vì chép lại luật; D bỏ danh sách import trùng với tin nhắn chunk và bỏ ví dụ thừa nhưng giữ cấu trúc bắt buộc; F viết gọn văn xuôi, giữ số luật L1–L15 và mọi con số (32px, 40%, 85%, 5%, 16–24px…); bộ minh hoạ và catalog Lottie rút gọn mô tả, giữ nguyên tên component, props và quy ước toạ độ.
  - Giữ nguyên các biến `{{...}}` và chữ mà `golden_prompts_test.go` đòi.
- **Ngoài phạm vi**: lọc bộ minh hoạ theo kịch bản (Q4-b); system prompt riêng cho repair (Q4-c); hiện `reasoning_chars` trên giao diện chi phí (backlog riêng); đổi biến sang token (Q2-b); chia lô khi có nhiều shot lỗi (Q3-b).

**Tiêu chí nghiệm thu**
1. (FR-1, tách ra: không thuộc CR này.)
2. Đoạn có k > 1 shot lỗi → đúng **một** lượt `repair` cho đoạn đó; LAYOUT lỗi vẫn là lượt riêng.
3. Shot lỗi duy nhất → prompt repair giống hệt hiện tại.
4. Reply gộp thiếu một hàm → hàm đó giữ code cũ, các hàm khác được cập nhật, không ném lỗi.
4b. Lượt repair gộp 3 shot bị cắt `budget` → chia thành 2 + 1 shot (rồi tới từng shot nếu vẫn bị cắt); không shot nào bị bỏ sót.
5. System prompt render cho một project mẫu ngắn hơn ít nhất 25% (số ký tự trước/sau ghi trong báo cáo `/code`); test golden và test kit vẫn pass.

## Giải pháp đề xuất

**Trần suy nghĩ.** Giữ một biến duy nhất, đổi mặc định, không thêm cơ chế. Không dùng công thức p95 × hệ số vì dữ liệu bị cắt đúng ở giá trị cần ước lượng; cách duy nhất biết đuôi là nâng trần một lần rồi đo lại bằng SQL ghi trong `.env.example`. 100 000 ký tự vẫn chặn được lượt treo kiểu 380k ký tự / 13 phút từng gặp ở CR-048.

**Gộp repair.** `remotion_repair` và `manim_repair` hiện nhận một `key`. Thêm hàm dựng prompt nhiều shot: phần lỗi và code hiện tại lặp theo từng shot, bảng màu và LAYOUT in một lần. Reply là một khối code chứa các hàm; parse bằng parser mới `parse_repaired_shots` chấp nhận thiếu (khác `parse_shots` của chunk vốn ném lỗi khi thiếu). `_repair` đổi từ "một `fix` mỗi shot" sang "một `fix` mỗi nhóm" (nhóm = các shot cùng `plan.owner`; LAYOUT/cast từng cái riêng). Manim dùng chung đường này để hai engine không lệch nhau.

**Thu gọn prompt.** Sửa seed trong `prompt_template_seeds.go`, `prompt_template_seeds_ai.go` và hai file `prompts/*.txt`. Seed upsert khi khởi động; rebuild và restart authoring-service trong cùng lượt, kiểm tra không có prompt Creator đang bật đè lên bản seed.

## Phụ thuộc

Đã kiểm (2026-10-02): không còn nhánh `feature/cr-*`, `fix/*`, `chore/*` nào chưa merge vào `origin/main`; primary checkout ở `main` sạch. → **Độc lập.**

## Phạm vi

- **llm-service**: `app/pipeline/prompts.py`, `app/pipeline/run.py` (`_repair`, `_settle_chunk`, vòng repair cuối), `tests/test_pipeline.py`.
- **authoring-service**: `internal/domain/prompt_template_seeds.go`, `prompt_template_seeds_ai.go`, `prompts/illustration_kit_vi.txt`, `prompts/lottie_catalog_vi.txt`; `testdata/prompt_golden.json` nếu đổi; `golden_prompts_test.go`.
- **Cấu hình**: không đổi.
- **Tài liệu**: `docs/contracts/authoring-llm-code-v2.md` (nhãn lượt `repair` có thể là danh sách shot), ADR-0030 (một dòng), `aidlc-state.md` (gạch ba dòng backlog).
- **Không đổi**: contract HTTP, schema DB, giao diện.
- `graphify affected` cho `_repair`, `remotion_repair`, `manim_repair`, `_settle_chunk` chạy ở bước 1 của `/code`, trước khi sửa.

## Kế hoạch thực hiện

1. `graphify affected` cho `_repair`, `remotion_repair`, `manim_repair`, `_settle_chunk`; đọc `docs/code-standards-rules.md`.
2. (Trần: tách ra backlog, bỏ qua. Không đổi `config.py`, `docker-compose.yml`, `.env.example`.)
3. **Prompt repair nhiều shot**: `prompts.py` thêm `remotion_repair_many(sb, layout, items, canvas)` và `manim_repair_many(...)`, với `items = [(key, code, diags)]`; một item thì gọi lại hàm một shot hiện có để prompt không đổi.
4. **Parser**: `run.py` thêm `parse_repaired_shots(engine, ids, text) -> dict[str, str]`: trả các hàm có mặt trong `ids`, bỏ hàm thừa, chỉ ném `ExtractError` khi không có hàm nào.
5. **`_repair`**: nhóm `targets` theo `plan.owner(key)` (LAYOUT/cast để riêng); mỗi nhóm một `_ask(req, "repair", ",".join(ids), owner, build, parse, record)`; áp từng hàm có mặt, hàm vắng giữ code cũ. Nhóm bị cắt (`SPLIT_KINDS`) thì chia đôi và thử từng nửa, đệ quy xuống một shot (cùng kiểu `write` của chunk, [run.py:681-712](../../../services/llm-service/app/pipeline/run.py#L681-L712)); mỗi nửa giữ một slot `sem` riêng, tính ngoài semaphore của lượt trước. `_settle_chunk` gọi qua cùng `_repair`.
6. **Test llm-service** (`tests/test_pipeline.py`): (a) đoạn có 2 shot lỗi → đúng một lượt `repair` nhãn `1.2,1.3`; (b) shot lỗi duy nhất → prompt như cũ; (c) reply gộp thiếu một hàm → hàm đó giữ code cũ, hàm kia đổi; (d) LAYOUT lỗi cùng shot lỗi → hai lượt riêng; (e) hai đoạn khác nhau → hai lượt; (f) repair sớm và repair cuối đều gộp; (g) lượt gộp 3 shot bị cắt `budget` → chia 2 + 1, đủ 3 shot được sửa. Cập nhật các test đang khẳng định "mỗi shot một lượt" (khoảng dòng 166–172, 271–288).
7. **Thu gọn prompt**: sửa `remoDAIVI`, `remoGAIVI`, `remoFVI`, `remoC2VI`, `remoIntroAIVI`, `remoAAIVI` (nếu cần) và hai file `prompts/*.txt` theo FR-3. Đo ký tự trước/sau bằng script ngắn (render seed với project mẫu). Chạy `go test ./internal/domain/...` và test kit của rendering.
8. Cập nhật `docs/contracts/authoring-llm-code-v2.md`, ADR-0030, `aidlc-state.md`.
9. Chạy toàn bộ test llm-service và authoring-service.

## Kiểm tra

- Test: `pytest` trong llm-service, `go test ./...` trong authoring-service, test kit trong rendering.
- Rebuild và restart `llm-service`, `authoring-service` bằng `scripts/worktree.sh rebuild llm-service authoring-service`; xác nhận healthy; kiểm authoring-service đã upsert seed mới.
- Kiểm trực tiếp (Creator): chạy bước code một project thật rồi chạy lại SQL hiệu chỉnh; so số lượt `budget`, số lượt repair, tổng token prompt với bảng trên.

## Rủi ro

- **Đuôi suy nghĩ chưa biết.** Nếu đuôi của chunk kéo xa hơn 100 000 ký tự thì lượt vẫn bị cắt, và mỗi lượt cắt tốn tối đa khoảng 5–6 phút thay vì 3,5 phút. Nâng trần chưa chắc giảm tổng chi phí cho tới khi đo lại; đó là lý do cần SQL hiệu chỉnh.
- **Repair gộp dài hơn**: một lượt nhiều shot sinh nhiều token hơn và mất phần song song trong đoạn (các đoạn khác nhau vẫn song song).
- **Thu gọn prompt có thể đổi hành vi model** dù luật giữ nguyên; test đơn vị không đo được điều này, chỉ thấy qua video thật. Creator nên so chất lượng một video trước/sau.
- **Seed prompt**: nếu Creator đang bật prompt `remotion_engineer_ai` tự sửa thì bản gọn không có hiệu lực.
- Lượt bị cắt tiếp tục không có số token (hạn chế đã biết của CR-056).

## Câu hỏi còn lại (thuộc phần đã tách ra backlog, không chặn CR này)

**Q5 — Con số trần.** Công thức p95 × 1,3 không dùng được vì dữ liệu bị cắt ở 60 000. Đề xuất:
- (a) **100 000 ký tự cho mọi lượt**, một biến, đo lại bằng SQL sau video thật kế tiếp. **(khuyến nghị)**
- (b) 100 000 cho chunk/layout/cast; repair gộp cộng thêm 50 000 cho mỗi shot lỗi thêm trong cùng lượt (2 shot → 150 000, 3 shot → 200 000). Phản ánh việc suy nghĩ của repair gộp tỉ lệ với số shot, nhưng thêm một hằng số cấu hình và nới trần xa hơn mức số đo hỗ trợ.
- (c) Giữ 60 000: không khuyến nghị, 28% chunk đang chết ở đó.
