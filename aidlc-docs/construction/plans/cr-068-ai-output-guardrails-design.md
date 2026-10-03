# CR-068 — Thiết kế: chặn các lỗi lặp lại trong code do AI viết (bước Code, Remotion)

Trạng thái: **thiết kế đã duyệt** (mục 3: trần 100 000, bỏ chia đôi khi vượt trần).

## Yêu cầu gốc (nguyên văn)

> làm sao để ngăn chặn các lỗi từ kết quả trả về từ AI như của project 47745562 nhir

Trả lời khi duyệt:

> ok duyêt, ý 3 đưa trần lên 100k nhưng tạm thời comment backoff retry

> bỏ đoạn này b) Chia đôi khi vượt trần (run.py:705-736). Đây là cơ chế liên quan trực tiếp tới ý 3.
> Đoạn bị budget sẽ được viết lại thành hai nửa, rồi chia tiếp xuống tới từng shot. Repair cũng làm như vậy.
> Ở project 47745562, chính cơ chế này đẻ ra nhiều lượt bị cắt: đoạn 5.1-5.3 tốn 4 lượt, mỗi lượt khoảng 60k ký tự.

Chốt: thiết kế duyệt; trần 100 000; bỏ hẳn (không comment-out — quy chuẩn code cấm) cơ chế chia đôi khi `budget` ở cả chunk và repair; chia đôi khi `truncated` giữ nguyên.

## Hiện trạng

### Project 47745562 đã gặp gì (đọc từ DB, 2026-10-03)

Project `47745562-aad5-4134-91f5-140be148d69a`, chủ đề "Tại sao con người chưa khám phá được nền văn minh nào khác ngoài trái đất", chế độ AI, cả ba bước đều dùng `deepseek-ai/deepseek-v4.1-flash`, engine Remotion. Bước Code dừng ở `incomplete`, project vẫn là `draft`.

**Loại 1 — 5/15 đoạn shot hỏng vì trần suy nghĩ (`budget`).** `authoring_segments`:

| Đoạn | Shot không viết được | Lỗi |
|---|---|---|
| 2.4-3.1 | 3.1 | model suy nghĩ quá 60000 ký tự mà chưa viết được chữ nào (đã thử cả đoạn, rồi riêng shot 3.1) |
| 3.5-4.1 | 3.6 | như trên (cả đoạn → 3.5–3.6 → riêng 3.6) |
| 5.1-5.3 | 5.1, 5.3 | như trên |
| 5.4-6.2 | 6.2 | như trên |
| 6.3-7.1 | 6.4 | như trên |

`llm_usage` của project, phase `chunk`: **21 lượt thành công, 18 lượt bị cắt `budget` (46%)**. Lượt thành công dùng trung vị 46 185 ký tự suy nghĩ, p80 49 510, p90 56 436, max 58 688 — tức là sát trần 60 000. Mỗi lượt bị cắt chạy trung bình 218 giây rồi bỏ, không ra chữ nào. Chia nhỏ không cứu được: chính các lượt **một shot** cũng vượt trần, vì độ dài suy nghĩ chủ yếu do system prompt (khoảng 23 000 token) chứ không do số shot.

Số liệu 14 ngày toàn hệ thống (deepseek, phase `chunk`): 281 thành công, 76 `budget`, 45 `empty`, 17 `server`. CR-063 đã đo kỹ hơn (127 lượt): thành công phân bố đều tới sát 60 000 rồi dừng hẳn, không có lượt nào trên 60 000 — trần đang **cắt vào giữa mức bình thường**; CR-063 đề xuất 100 000 nhưng tách ra backlog ([cr-063 design](cr-063-code-gen-token-tuning-design.md) mục 1 và "Tách ra backlog").

**Loại 2 — lỗi biên dịch do model viết tên không tồn tại.** `code_check_diagnostics` của project:

- `TS2304: Cannot find name 'PALETTE_PLACEHOLDER'` — 34 dòng, ở đoạn 8.4-9.1 (vòng 0) và **xuất hiện mới ở vòng sửa thứ 1** của 1.1-1.3, 7.2-7.4, 9.5-10.2: lượt sửa lỗi khác lại sinh ra lỗi này.
- `TS2552: Cannot find name 'useCurrentFrameSafe'. Did you mean 'useCurrentFrame'?` — tsc đã chỉ đúng tên cần dùng.
- `TS2322: Type '"cup"' is not assignable to type 'HandPose'` — giá trị không có trong kiểu.

**Loại 3 — vi phạm vùng an toàn (`safe_area`)**: 7 dòng; vòng repair xử lý được phần lớn, vòng 2 của 1.1-1.3 còn một lỗi (hết lượt repair, để bước kiểm tra cuối quyết định).

### Các lỗi này lặp lại ở nhiều project (14 ngày)

| Lỗi | Số dòng | Số project |
|---|---|---|
| `TS2304` tên `PALETTE_...` không tồn tại | 69 | 5 |
| trong đó `PALETTE_PLACEHOLDER` | 56 | — |
| Thông báo tsc có "Did you mean '…'?" (TS2551 thuộc tính, TS2552 tên) | 232 | 5 |
| `TS2322` sai kiểu prop | 88 | 5 |
| `safe_area` | 192 | 8 |

Các biến thể `PALETTE_...` thấy được: `PALETTE_PLACEHOLDER`, `PALETTE_CHIM_PLACEHOLDER`, `PALETTE_NEN_TROI`, `PALETTE_GIAY`, `PALETTE_nenHangDong`, `PALETTE_chuNhan`...

### Vì sao lỗi `PALETTE_PLACEHOLDER` không hết

- System prompt `remotion_engineer_ai` đang **viết đúng tên sai đó ra như ví dụ cấm**: [prompt_template_seeds_ai.go:128](../../../services/authoring-service/internal/domain/prompt_template_seeds_ai.go#L128) "không bao giờ tự đặt tên khác như `PALETTE_NEN_TROI` hay `PALETTE_PLACEHOLDER`", và [:175](../../../services/authoring-service/internal/domain/prompt_template_seeds_ai.go#L175) "không có tên `PALETTE_...` nào?". Câu này vào từ CR-065 (2026-10-03 11:27) để chặn lỗi đã có từ 2026-09-30; sau đó lỗi vẫn xảy ra (project này, 2026-10-03 ~09:00 UTC). Model nhỏ hay chép lại chuỗi cụ thể có trong prompt, nhất là ở lượt repair vốn gửi lại nguyên system prompt.
- `golden_prompts_test.go:101-102` khoá hai câu đó vào prompt.
- Bản đang active trong DB là bản hệ thống (không có prompt Creator tự sửa đè lên vai trò này).

### Pipeline hiện tại (llm-service)

- [run.py:565-596](../../../services/llm-service/app/pipeline/run.py#L565-L596) `_ask`: một lượt gọi model, parse; parse lỗi thì hỏi lại (tối đa 2 lần).
- [run.py:705-736](../../../services/llm-service/app/pipeline/run.py#L705-L736) `write`: đoạn bị `budget`/`truncated` thì chia đôi tới một shot.
- [run.py:738-760](../../../services/llm-service/app/pipeline/run.py#L738-L760) `do_chunk` → [run.py:828-854](../../../services/llm-service/app/pipeline/run.py#L828-L854) `_settle_chunk`: biên dịch đoạn vừa viết (stub cho shot khác), lỗi thì gọi `_repair` (một lượt model cho mỗi đoạn), tối đa `repair_rounds` vòng.
- [run.py:779-819](../../../services/llm-service/app/pipeline/run.py#L779-L819): ghép cả file → kiểm → repair → kiểm lại.
- [run.py:856-927](../../../services/llm-service/app/pipeline/run.py#L856-L927) `_repair`: gửi lỗi + code cũ + bảng màu ([prompts.py:153-194](../../../services/llm-service/app/pipeline/prompts.py#L153-L194)).
- **Không có bước tự sửa nào không cần model**: mọi lỗi, kể cả lỗi mà tsc đã chỉ sẵn tên đúng, đều tốn một lượt repair (khoảng 22 000 token prompt, 33 giây) và có thể sinh lỗi mới.
- Trần suy nghĩ: [config.py:72](../../../services/llm-service/app/config.py#L72) `CODE_MAX_REASONING_CHARS` mặc định 60000, [docker-compose.yml:458](../../../docker-compose.yml#L458), [.env.example:82](../../../.env.example#L82); [provider.py:210](../../../services/llm-service/app/provider.py#L210) cắt stream khi vượt trần mà chưa có chữ trả lời.
- Sự kiện `check` (phase `chunk`|`final`) được authoring-service ghi nguyên vào `code_check_diagnostics` ([generate_authoring_code.go:350-366](../../../services/authoring-service/internal/application/generate_authoring_code.go#L350-L366)); hợp đồng ở [authoring-llm-code-v2.md:62](../../../docs/contracts/authoring-llm-code-v2.md).

## Yêu cầu

- **FR-1 — Prompt không gieo tên sai.** System prompt `remotion_engineer_ai` không còn chứa bất kỳ tên `PALETTE_...` nào làm ví dụ cấm; luật màu viết dạng khẳng định ("chỉ `PALETTE.<khoá trong danh sách>`; vai trò không có khoá → dùng khoá gần nghĩa nhất trong danh sách").
- **FR-2 — Tự sửa tên màu viết sai, không gọi model.** Mỗi hàm shot Remotion vừa nhận từ model (lượt chunk hoặc repair) được quét: định danh `PALETTE_<X>` mà `<X>` (bỏ phần `PLACEHOLDER`, bỏ `_`, không phân biệt hoa thường) trùng **đúng một** khoá trong bảng màu thì đổi thành `PALETTE.<khoá>`. Không trùng hoặc trùng nhiều khoá → giữ nguyên (để bước kiểm tra bắt và repair như hiện nay); không bao giờ đoán màu.
- **FR-3 — Áp gợi ý "Did you mean" của tsc, không gọi model.** Khi kiểm tra (chunk hoặc cuối) có lỗi `TS2552` ("Cannot find name 'A'. Did you mean 'B'?") hoặc `TS2551` ("Property 'A' does not exist on type … Did you mean 'B'?") nằm trong một hàm shot: đổi `A` thành `B` **chỉ trên đúng dòng đó**, chỉ khi:
  - `TS2552`: `B` là tên khung code đã import cho shot (`REMOTION_API`, `SEGMENT_API`, `PRIMITIVES`, `LottieClip`, các kit) hoặc `PALETTE`, `LAYOUT`, `clamp`;
  - `TS2551`: dòng đó có `.A` (truy cập thuộc tính) **và** `A`, `B` giống nhau khi bỏ hoa/thường và `_` (vd. `conNguoi` → `connguoi`). tsc gợi ý theo độ gần chữ nên có thể chỉ sang một khoá khác nghĩa; gợi ý như thế bị bỏ qua và để repair.
  Sau khi đổi, biên dịch lại ngay; lần tự sửa này **không tính** vào số vòng repair. Một cặp (shot, A→B) chỉ áp một lần (không lặp vô hạn).
- **FR-4 — Ghi lại mọi lần tự sửa.** Mỗi chỗ đổi của FR-2/FR-3 phát một sự kiện `check` với `phase: "autofix"`, mỗi chỗ một diagnostic `kind: "autofix"`, `rule: "palette_name"` hoặc `"tsc_suggestion"`, `message: "<A> → <B>"`, `shot`, `segment`, `line` → vào `code_check_diagnostics` như các lỗi khác. Không có thay đổi nào âm thầm.
- **FR-5 — Trần suy nghĩ của bước Code**: mặc định `CODE_MAX_REASONING_CHARS` = 100 000.
- **FR-6 — Không chia đôi khi vượt trần**: lượt chunk hoặc repair bị `budget` không được viết lại thành các phần nhỏ hơn; chỉ `truncated` còn chia đôi.

**Tiêu chí chấp nhận**

- AC-1: không còn chuỗi `PALETTE_` nào trong system prompt `remotion_engineer_ai` đã seed (test golden kiểm).
- AC-2: shot có `PALETTE_NEN_TROI`, `PALETTE_nenHangDong`, `PALETTE_CHIM_PLACEHOLDER` (bảng có khoá `nenTroi`, `nenHangDong`, `chim`) được đổi đúng trước khi biên dịch; `PALETTE_PLACEHOLDER` giữ nguyên và đi vào repair như cũ.
- AC-3: shot có `useCurrentFrameSafe` → `useCurrentFrame` sau một lần biên dịch, không có lượt repair nào được gọi cho shot đó; `PALETTE.conNguoi` với gợi ý `connguoi` → `PALETTE.connguoi`; `PALETTE.conNguoi` với gợi ý `nenpanel` → giữ nguyên, đi repair.
- AC-4: mỗi lần tự sửa có dòng `kind='autofix'` trong `code_check_diagnostics`.
- AC-6 (FR-6): test: lượt chunk bị `budget` → đúng một lượt gọi, đoạn hỏng với mọi shot chưa viết; lượt repair bị `budget` → đúng một lượt, code cũ giữ nguyên; lượt `truncated` vẫn chia đôi.
- AC-5 (FR-5, FR-6): chạy lại 5 đoạn hỏng của project 47745562 — kết quả (thành công / còn `budget`) được báo đúng như nó xảy ra.

**Ngoài phạm vi**

- `TS2322` sai kiểu prop (`'cup'` cho `HandPose`, số cho `DoorOpen`): cần hiểu ý shot, không tự sửa được an toàn; vẫn để repair (đã xử lý được ở vòng 1 của project này). Có thể làm CR sau: đưa danh sách giá trị hợp lệ vào lời nhắc repair.
- `safe_area` và các luật bố cục: giữ nguyên vòng repair hiện có (CR-062).
- Lỗi `empty`/`server` của nhà cung cấp; engine Manim; đổi model.
- Sửa các project cũ: Creator bấm chạy lại các đoạn hỏng sau khi triển khai.

## Giải pháp đề xuất

Ba lớp, xếp theo chi phí: (1) bỏ nguồn gây lỗi trong prompt, (2) tự sửa xác định những lỗi có đáp án duy nhất trước khi tốn một lượt model, (3) nới trần suy nghĩ đang cắt vào mức bình thường. Không thêm lượt gọi model nào mới; chỉ bớt.

### 1. Prompt (FR-1)

Sửa luật B.2 và câu G.3 của `remotion_engineer_ai` ([prompt_template_seeds_ai.go:128](../../../services/authoring-service/internal/domain/prompt_template_seeds_ai.go#L128), [:175](../../../services/authoring-service/internal/domain/prompt_template_seeds_ai.go#L175)):

- B.2: "MỌI màu trong code (…) là `PALETTE.<khoá>` với khoá viết y như danh sách trong tin nhắn (vd. `PALETTE.nenTroi`). Vật không có vai trò màu riêng → dùng khoá gần nghĩa nhất trong danh sách. Không viết mã hex/rgb/tên màu nào khác. …" — bỏ hẳn các ví dụ tên sai.
- G.3: "Màu: mọi màu đều viết `PALETTE.<khoá>` với khoá có trong danh sách? …"
- `golden_prompts_test.go:101-102`: đổi hai chuỗi bắt buộc theo câu mới, thêm kiểm "prompt không chứa `PALETTE_`".

Seed upsert khi authoring-service khởi động → rebuild authoring-service là đủ (bản active là bản hệ thống).

### 2. Tự sửa xác định (FR-2, FR-3, FR-4)

Module mới `services/llm-service/app/pipeline/autofix.py`, hàm thuần (không I/O), dễ test:

- `fix_palette_names(code: str, keys: Iterable[str]) -> tuple[str, list[Fix]]`
- `fix_tsc_suggestions(shots: dict[str, str], merged: Merged, diags: list[Diagnostic], frame_names: set[str], applied: set) -> tuple[dict[str, str], list[Fix]]` — trả code mới của các shot được sửa (dựa vào `merged.lines` để đổi số dòng trong file thành dòng trong shot).
- `Fix(shot, line, rule, before, after)`.

Gắn vào `run.py`:

- `do_chunk` (sau `write`) và `_repair` (sau khi nhận code sửa): chạy `fix_palette_names` trên từng shot Remotion mới; có sửa → phát sự kiện `check` phase `autofix`. Code đã sửa là code được lưu vào segment.
- `_settle_chunk` và vòng kiểm tra cuối: check không qua → `fix_tsc_suggestions`; có sửa → phát `autofix`, ghép lại, kiểm lại **không tăng `rounds`**; không có gì để sửa → repair như hiện nay. Tập `applied` chặn áp lại cùng một chỗ.
- `merger.frame_names()` (mới): tập tên khung code import cho shot, dựng từ cùng các tuple với `_REMOTION_HEAD`, để hai nơi không lệch nhau.

Vì sao chọn tự sửa có điều kiện chặt chứ không "đoán" rộng hơn: một chỗ đổi sai tên vẫn biên dịch được nhưng vẽ sai màu/sai hàm mà không ai thấy. Chỉ đổi khi có đáp án duy nhất (khoá trùng đúng một, hoặc tsc chỉ đích danh và tên đó thuộc khung code); mọi trường hợp khác giữ nguyên hành vi hiện nay.

Hợp đồng: thêm giá trị `"autofix"` cho `check.phase` và `diagnostics[].kind` — chỉ thêm giá trị; authoring-service ghi nguyên, không cần sửa code Go. Cập nhật `docs/contracts/authoring-llm-code-v2.md`.

### 3. Trần suy nghĩ và bỏ chia đôi khi vượt trần (FR-5, FR-6) — Creator đã chọn

Creator chọn phương án A (nâng trần) và bỏ cơ chế chia đôi khi vượt trần. Các phương án đã cân nhắc: A nâng trần 60 000 → 100 000; B giữ 60 000 và thử lại bằng model dự phòng; C giữ nguyên.

- **Nâng trần**: mặc định `CODE_MAX_REASONING_CHARS` 60 000 → **100 000** ([config.py:72](../../../services/llm-service/app/config.py#L72), [docker-compose.yml:458](../../../docker-compose.yml#L458), [.env.example:82](../../../.env.example#L82)). Số đo hai lần (CR-063 và project này) cho thấy lượt thành công dồn sát 60 000 rồi bị chặn cứng; 100 000 ký tự ≈ 32 000 token suy nghĩ (3,1 ký tự/token).
- **Bỏ chia đôi khi `budget`**: `SPLIT_KINDS` ([run.py:67](../../../services/llm-service/app/pipeline/run.py#L67)) chỉ còn `TRUNCATED`. Một đoạn (lượt chunk) hoặc một nhóm shot (lượt repair) bị `budget` thì không viết lại thành hai nửa nữa: lượt chunk báo đoạn hỏng ngay với mọi shot chưa viết (`failed_shots`), Creator chạy lại như hiện nay; lượt repair giữ code cũ của các shot đó (như một lượt repair hỏng hiện nay) và để lần kiểm tra sau báo lỗi. Lý do: ở project 47745562, chia đôi nhân số lượt bị cắt (đoạn 5.1-5.3 tốn 4 lượt × 60 000 ký tự) vì cả lượt một shot cũng vượt trần. `truncated` (bị cắt giữa câu trả lời do `max_tokens`) vẫn chia đôi: ở đó chia nhỏ thật sự làm câu trả lời ngắn lại.
- Thông báo lỗi của đoạn hỏng (`shot_failure_message`) vẫn đúng: chỉ nêu những gì đã thử (cả đoạn), không còn "rồi riêng shot …".
- Rủi ro: lượt vẫn hỏng thì tốn tới 100 000 ký tự thay vì 60 000; shot nào suy nghĩ > 100 000 vẫn hỏng. Đo lại tỉ lệ `budget` sau vài project; còn cao thì cân nhắc B ở CR sau.

## Phụ thuộc

Đã kiểm (2026-10-03): `git branch -a --no-merged origin/main` và `scripts/worktree.sh` worktrees.

- `feature/cr-064-screenwriter-prompt-trim`: chỉ sửa prompt `story_architect` (`prompt_template_seeds.go`) và dòng hash `RoleStoryArchitect` trong `golden_prompts_test.go:17`; CR-067 đã quyết định bỏ CR-064. CR-068 sửa `prompt_template_seeds_ai.go` và dòng 101-102 của cùng file test → khác hunk, **độc lập**.
- `cr-066`, `fix/run-segment-reruns-all`, `fix/video-list-paging`, `chore/worktree-per-cr`: không còn trong danh sách chưa merge → **độc lập**. Không xem được thay đổi chưa commit trong các worktree khác (phiên này bị khoá trong worktree CR-068); các worktree đó đều đã merge hoặc bị bỏ.

## Phạm vi

| Service | File / symbol | Thay đổi |
|---|---|---|
| llm-service | `app/pipeline/autofix.py` (mới) | `Fix`, `fix_palette_names`, `fix_tsc_suggestions` |
| llm-service | `app/pipeline/merger.py` | `frame_names()` mới |
| llm-service | `app/pipeline/run.py` `do_chunk`, `_settle_chunk`, `_repair`, vòng kiểm cuối trong `run`, `_check_event`-style helper cho autofix | gọi autofix, phát sự kiện |
| llm-service | `app/config.py:72` | mặc định 100000 |
| llm-service | `app/pipeline/run.py:67` `SPLIT_KINDS`, `write`, `_repair.fix_shots`, docstring đầu module | bỏ `BUDGET` khỏi chia đôi |
| llm-service | `tests/test_autofix.py` (mới), `tests/test_pipeline.py` | test |
| authoring-service | `internal/domain/prompt_template_seeds_ai.go:128,175`; `golden_prompts_test.go:101-102` | câu luật màu |
| hạ tầng | `docker-compose.yml:458`, `.env.example:82` | 100000 |
| tài liệu | `docs/contracts/authoring-llm-code-v2.md` | `phase`/`kind` `"autofix"` |

Không migration, không đổi bảng. `graphify affected`: `_repair` chỉ được `run` và `_settle_chunk` gọi; `_settle_chunk` chỉ được `do_chunk` gọi; `palette_keys` được `remotion_frame_text`, `_palette_table`, `make_plan` và test merger gọi (không đổi hàm này, chỉ dùng).

## Kế hoạch thực hiện

1. llm-service `autofix.py` + `merger.frame_names()` + `tests/test_autofix.py` (các ca AC-2, AC-3, ca không đổi: không trùng, trùng hai khoá, gợi ý ngoài khung, dòng ngoài shot).
2. llm-service `run.py`: gắn `fix_palette_names` sau chunk và repair; gắn `fix_tsc_suggestions` trước repair trong `_settle_chunk` và vòng cuối; sự kiện `check` phase `autofix`. Test pipeline với checker giả: (a) chunk có `PALETTE_NEN_TROI` → segment lưu `PALETTE.nenTroi`, có sự kiện autofix; (b) checker trả TS2552 có gợi ý → lần kiểm thứ hai qua, không có lượt `repair` nào trong `calls`, `repair_rounds == 0`; (c) gợi ý không thuộc khung → repair như cũ.
3. llm-service: trần 100 000 (config + docker-compose + .env.example); `SPLIT_KINDS = (errors.TRUNCATED,)`, sửa docstring module/`write`/`_repair` cho đúng; sửa các test pipeline đang kiểm chia đôi khi `budget` thành kiểm không chia, giữ test chia đôi cho `truncated`.
4. authoring-service prompt B.2/G.3 + golden test.
5. Hợp đồng `authoring-llm-code-v2.md`.

## Kiểm tra

- `cd services/llm-service && pytest` ; `ruff check app tests`.
- `cd services/authoring-service && go test ./... && go vet ./... && gofmt -l .`.
- Rebuild: `scripts/worktree.sh rebuild llm-service authoring-service`; xác nhận healthy; xác nhận trong DB prompt `remotion_engineer_ai` active không còn `PALETTE_`.
- Trực tiếp (Creator bấm, tốn token): chạy lại 5 đoạn hỏng của project 47745562; xem `llm_usage` (`budget` còn bao nhiêu) và `code_check_diagnostics` (`kind='autofix'`).

## Rủi ro

- Tự sửa đổi sai ý: chặn bằng điều kiện "đáp án duy nhất" và chỉ trên dòng tsc chỉ ra; mọi chỗ đổi được ghi lại (FR-4) để Creator xem.
- Dòng tsc lệch so với code shot (stub, thư viện hình chèn giữa): dùng đúng `merged.lines` như `_map_failures` đang dùng; dòng không thuộc shot nào thì bỏ qua.
- Trần 100 000 làm lượt hỏng đắt hơn; bỏ chia đôi làm đoạn có một shot "khó" hỏng cả đoạn (các shot còn lại của đoạn chạy lại cùng lượt sau). Cần đo sau khi chạy.
- Đổi prompt hệ thống: nếu sau này Creator bật prompt tự sửa cho `remotion_engineer_ai`, câu mới không có tác dụng với prompt đó (tự sửa ở llm-service vẫn chạy).
