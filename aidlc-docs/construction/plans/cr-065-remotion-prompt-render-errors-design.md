# CR-065 — Code Remotion hay lỗi biên dịch: prompt hứa những thứ khung code không có

**Trạng thái**: Creator duyệt FR-1, FR-2, FR-4, FR-5, FR-7; bỏ FR-3 và FR-6 ("ko làm 2 cái đó").
**Nhánh**: `feature/cr-065-remotion-prompt-render-errors`

## Yêu cầu gốc (nguyên văn)

> review lại chỗ system promt code remotion sao hay bị render lỗi thế nhỉ

Góp ý của Creator sau bản đề xuất đầu (nguyên văn):

> nếu chỗ promt đưa ra các bảng màu hoặc các thẻ chưa tồn tại ta có thể dùng lại logic của hình minh hoạ ko nhỉ, nếu có rồi reuse còn không có thì tạo mới. cái thứ 2 ở promt code ta cũng nên áp dụng tương tự không ví dụ như một dự án video đều có một dự án vẽ svg nhỏ bên trong ta cũng apply oop hay solid và các coding stander khác để reuse tránh lặp code. code sau dùng chung 1 đoạn minh hoạ hoặc hình ảnh minh hoạ hay bất kì thứ gì có thể gọi ra để dùng lại thôi. xem đề xuất này của tôi nhé
> Lỗi random: (a) import thêm vào khung, thay vì (b) bỏ random khỏi prompt. => tôi nhớ đã làm cái này rồi
> Kiểu prop: (a) lấy từ code lúc dựng prompt, có tác dụng ngay với cả hình cũ; (b) sửa prompt tạo hình thì chỉ có tác dụng với hình mới. => ý tưởng này rất tốt

## Đánh giá góp ý của Creator (đo trên dữ liệu thật)

**`random` "đã làm rồi" — đúng một nửa.** Commit `7b5b241` (2026-10-02) đưa import của **hình thư viện** lên khung, nên `random` mà một hình thư viện dùng không còn lỗi. Trong 146 lỗi `random`: **103** nằm trong code hình thư viện (đã sửa), **43** nằm trong code shot do model viết (3 project: `82dbfbb7`, `257acc58`, `6deec24f`). Phần sau chưa sửa: shot chỉ có `random` khi tình cờ có một hình thư viện import nó. FR-1 vẫn cần, nhưng chỉ là phần còn lại này.

**Ý 1 — màu / thẻ chưa có thì "có rồi dùng lại, chưa có thì tạo mới" như hình minh hoạ.**
- *Thẻ (component)*: logic này **đã có**. Bước "Hình minh hoạ" (giữa Visual và Code) đọc storyboard, đối chiếu thư viện, đưa vật đã có vào `reuse` và vật chưa có vào `draw`; hoạ sĩ AI vẽ, Creator duyệt, rồi bước code nhận hình qua mục C4/C5 ([project_illustrations.go:262-392](../../../services/authoring-service/internal/application/project_illustrations.go#L262-L392), [planner_prompt_vi.txt](../../../services/authoring-service/internal/application/planner_prompt_vi.txt)). Từ CR-056 đến nay không có lỗi nào do model gọi một component không tồn tại (lỗi `TS2304` còn lại chỉ là `random`, `PALETTE_xxx` và một tên bịa `useVideoConfigSafe`).
- *Màu*: dữ liệu cho thấy vấn đề **không phải thiếu màu**. Ở project `257acc58`, model viết `PALETTE_NEN_TROI`, `PALETTE_NHANMANH_PLACEHOLDER`, `PALETTE_CHIM_PLACEHOLDER` trong khi bảng màu **đã có** `nenTroi`, `nhanManh`, `chim`. Model biết màu nào, chỉ viết sai cách gọi. Cho bước code tự tạo màu mới thì phải tự chọn mã hex, tức lấy mất quyền quyết định màu của Đạo diễn, mà không sửa được lỗi thật. Đề xuất giữ FR-5: bảng màu ghi đúng dạng `PALETTE.nenTroi` và cấm tự đặt tên `PALETTE_...`.

**Ý 2 — trong code cũng tách phần dùng chung (OOP/SOLID) để không lặp.**
- Với vật cụ thể (người, đồ vật, nền), việc này **đã có** đúng như ý: mỗi vật là một component trong thư viện, vẽ một lần và gọi lại ở mọi shot, mọi video. Đây chính là "dự án vẽ SVG nhỏ bên trong".
- Phần model còn vẽ tay trong từng shot là hình trừu tượng (mũi tên, đồ thị, lưới), vì planner được dặn không đưa chúng vào thư viện ([planner_prompt_vi.txt:23](../../../services/authoring-service/internal/application/planner_prompt_vi.txt#L23)). Đo trên 3 script AI gần nhất (`257acc58`, `6deec24f`, `f7103848`; 46–55 shot mỗi script):
  - khối code ≥ 6 dòng lặp giữa các shot: **0–4%** code shot, chủ yếu là props `<Scene backdrop camera>` (mỗi shot một bộ số khác nhau);
  - dòng lặp ở ≥ 3 shot: **7–9%**, chủ yếu là cấu hình máy quay và một đoạn zoom tự viết bằng `div transform` (lặp ở 25/46 shot của `6deec24f`, 13/52 shot của `f7103848`). Đoạn zoom này còn **vi phạm luật L8** (phải dùng `Camera`/prop `camera`).
  - không shot nào khai báo component phụ được shot khác dùng lại.
- Kết luận: tách thêm lớp component dùng chung trong code chỉ bớt được rất ít. Ngược lại, nó tốn một bước phải chạy trước mọi chunk (như LAYOUT), và đó lại là nơi dễ lỗi và trùng tên khi nhiều chunk viết song song (chính lý do của luật D.3). Cái nên làm là bắt model dùng `Camera` có sẵn thay vì tự viết zoom; đây là FR-7 bên dưới.
- Nếu Creator vẫn muốn các hình trừu tượng lặp lại (một đồ thị, một sơ đồ xuất hiện ở nhiều shot) được vẽ một lần và dùng lại, có hai đường. Cả hai đều đề xuất làm thành **CR riêng**, không gộp vào đây:
  - (a) planner đưa hình trừu tượng xuất hiện ở ≥ 2 shot vào `draw` → thành hình thư viện, Creator duyệt, video sau cũng dùng được;
  - (b) lượt LAYOUT viết thêm một bảng component dùng chung cho video đó.

## Hiện trạng

### 1. "Render lỗi" thực chất là lỗi ở bước kiểm code, không phải ở lúc render

- 10 ngày qua chỉ có **1** lần render Remotion thật sự hỏng (`saga_steps`, 2026-10-01: `The symbol "LAND" has already been declared`). Lỗi này do hai hình thư viện trùng tên hằng phụ và đã được sửa ở commit `48b3225`.
- Phần còn lại là lỗi ở bước **kiểm code** (`code_check_diagnostics`): mỗi đoạn code viết xong được biên dịch, đoạn nào lỗi thì gọi model sửa (repair). Sửa không được sau số vòng tối đa thì cả script trượt check cuối, Creator thấy "lỗi".
- Repair tốn nhiều: 2026-09-29 → 10-01 có **244 lượt repair** so với **141 lượt chunk** thành công (`llm_usage`, step `code`).

### 2. Phân loại lỗi biên dịch từ khi CR-056 sửa lỗi hoa/thường của khoá PALETTE (sau 2026-09-30 16:30 UTC)

| # | Nguyên nhân | Số lỗi | Project | Shot |
|---|---|---|---|---|
| 1 | `Cannot find name 'random'` | 146 | 3 | 11 |
| 2 | Prop của hình thư viện sai kiểu (`TS2322`, `TS2339`) | 90 | 3 | 34 |
| 3 | Tự đặt tên `PALETTE_PLACEHOLDER`, `PALETTE_NEN_TROI`... thay cho `PALETTE.khoa` | 26 | 3 | 12 |
| 4 | `interpolateColors(..., clamp)` (`TS2559`) | 3 | 2 | 3 |
| 5 | Khác (`LAND` trùng tên — đã sửa; `useVideoConfigSafe` bịa) | 8 | — | 3 |

Lỗi bố cục (`safe_area`...) đã giảm mạnh sau CR-062 (98 lỗi ngày 10-01 → 19 ngày 10-02 → 1 ngày 10-03); không thuộc CR này.

Check cuối của project `257acc58` (2026-10-01) trượt sau **cả 2 vòng repair** chỉ vì 11 lỗi `random` và 1 lỗi prop `DoorOpen`.

### 3. Nguyên nhân gốc trong code

**(1) `random` — prompt bảo dùng, khung code không import.**
- Luật L9 bắt buộc: "ngẫu nhiên dùng `random('seed-cố-định')` của remotion" ([prompt_template_seeds.go:816](../../../services/authoring-service/internal/domain/prompt_template_seeds.go#L816)).
- Mục E (dùng chung với prompt thủ công) nói remotion "mọi API... `random`", primitives có `BACKGROUND`, `useFrameBox()` ([prompt_template_seeds.go:780-782](../../../services/authoring-service/internal/domain/prompt_template_seeds.go#L780-L782)).
- Nhưng ở chế độ AI, model không viết import; khung do `merger._REMOTION_HEAD` dựng chỉ import `registerRoot, Composition, AbsoluteFill, interpolate, interpolateColors, spring, Easing, useCurrentFrame, useVideoConfig` và `Stage, SAFE_MARGIN, WIDTH, HEIGHT` ([merger.py:85-93](../../../services/llm-service/app/pipeline/merger.py#L85-L93)). `random`, `BACKGROUND`, `useFrameBox` không có. `random` chỉ xuất hiện khi tình cờ một hình thư viện cũng import nó ([test_backdrops.py:54-61](../../../services/llm-service/tests/test_backdrops.py#L54-L61)).
- Prompt repair chỉ đưa lỗi "Cannot find name 'random'", không nói tên nào có sẵn, nên model sửa xong vẫn dùng `random` (vòng lặp đến hết số vòng).
- Thêm một chỗ lệch: danh sách "đã có sẵn" trong tin nhắn chunk ([prompts.py:102-103](../../../services/llm-service/app/pipeline/prompts.py#L102-L103)) liệt kê tay, thiếu `scene` (`Scene`, `Camera`, `Glow`...) và `backdrops` dù khung có import. Hai danh sách viết tay ở hai nơi nên lệch nhau.

**(2) Prop hình thư viện — model chỉ thấy tên prop, không thấy kiểu.**
- `library_section` đưa vào system prompt mỗi hình một dòng `usage — description` ([run.py:141-170](../../../services/llm-service/app/pipeline/run.py#L141-L170)). `usage` do người/AI viết khi tạo hình, chỉ có tên prop: `<NarrowDoor color panel open />`, `<Galaxy color starColor coreColor core />`.
- Kiểu thật nằm trong code của hình: `open?: DoorOpen` với `type DoorOpen = 'closed' | 'ajar' | 'open'`; `core?: 'normal' | 'flare' | 'faded'`. Model đoán `open={0.5}`, `open={true}`, `core={true}` → `TS2322`.
- Prompt repair không có gì thêm về hình thư viện, nên repair cũng đoán lại.
- Bộ minh hoạ dựng sẵn (C3, `illustration_kit_vi.txt`) đã ghi rõ giá trị (vd. `hairStyle: 'short' | 'long' | 'bald' | 'bun'`), nên ít lỗi hơn hẳn.

**(3) `PALETTE_xxx`.**
- Bảng màu đưa cho model ở dạng `- nenTroi = #… — vai trò: ý nghĩa` ([prompts.py:47-49](../../../services/llm-service/app/pipeline/prompts.py#L47-L49)), không có chữ `PALETTE.` ở đầu khoá. Khi cần một vai trò không có khoá (hoặc phân vân), model tự đặt hằng `PALETTE_PLACEHOLDER` / `PALETTE_NEN_TROI` thay vì chọn khoá gần nghĩa như mục B.4 yêu cầu ([prompt_template_seeds_ai.go:121-128](../../../services/authoring-service/internal/domain/prompt_template_seeds_ai.go#L121-L128)).

**(4) `interpolateColors` + `clamp`.**
- Mục D-AI cho sẵn hằng `clamp` và L9 bảo "mọi `interpolate` có clamp"; model áp cả cho `interpolateColors`, hàm này không nhận tham số thứ tư.

### 4. Prompt và seed

- `remotion_engineer_ai` = `remoIntroAIVI + remoAAIVI + remoBAIVI + remoCVI + remoC2VI + remoC3VI + remoDAIVI + remoEVI + remoFVI + remoGAIVI + remoOutputAIVI` ([prompt_template_seeds_ai.go:185](../../../services/authoring-service/internal/domain/prompt_template_seeds_ai.go#L185)). `remoCVI`, `remoEVI`, `remoFVI` dùng chung với prompt thủ công `remotion_engineer`, có hash golden khoá ([golden_prompts_test.go:16-21](../../../services/authoring-service/internal/domain/golden_prompts_test.go#L16-L21)).
- Seed version `remotion_engineer_ai` = 4 ([prompt_template_seeds.go:26](../../../services/authoring-service/internal/domain/prompt_template_seeds.go#L26)). DB hiện chỉ có bản hệ thống đang bật cho vai trò này (28 973 ký tự), không có prompt Creator đè lên.

## Yêu cầu

**FR-1 — Khung code có đủ mọi tên prompt bảo dùng.** Khung Remotion của chế độ AI import thêm `random` (remotion), `BACKGROUND`, `useFrameBox` (primitives). `7b5b241` chỉ đưa import của hình thư viện lên khung; FR-1 lo phần còn lại, là `random` trong code shot do model viết (43/146 lỗi).
- AC: một shot gọi `random('x')`, `useFrameBox()`, dùng `BACKGROUND` biên dịch qua mà không cần hình thư viện nào.

**FR-2 — Một nguồn duy nhất cho danh sách tên có sẵn.** Danh sách tên trong tin nhắn chunk sinh từ cùng hằng mà `_REMOTION_HEAD` dùng (gồm cả `scene`, `backdrops`), không viết tay.
- AC: test khẳng định mọi tên trong `_REMOTION_HEAD` xuất hiện trong tin nhắn chunk và ngược lại.

~~**FR-3 — Prompt AI nói đúng những gì có.**~~ **Creator bỏ — không làm.** Mục E dùng chung và mục D-AI giữ nguyên.

**FR-4 — Hình thư viện kèm kiểu prop.** Mỗi hình trong mục C4 có thêm dòng kiểu prop lấy thẳng từ code của hình: phần `{ ... }` sau `FigureProps &` của component được export, các `type X = 'a' | 'b'` trong cùng file được thay bằng giá trị. Không trích được (code viết khác khuôn) thì giữ dòng như hiện nay, không bịa.
- AC: `NarrowDoor` ra `open?: 'closed' | 'ajar' | 'open'`; `Galaxy` ra `core?: 'normal' | 'flare' | 'faded'`; một hình không có khối props vẫn ra đúng dòng cũ.
- Prompt nhắc: giá trị prop phải đúng kiểu ghi ở đó.

**FR-5 — Bảng màu ghi dạng truy cập thật.** Bảng màu trong tin nhắn chunk/repair ghi `PALETTE.nenTroi = #…`. Mục B và G của prompt AI thêm: chỉ viết `PALETTE.<khoá>` (dấu chấm), không bao giờ tự đặt hằng `PALETTE_...`/`PLACEHOLDER`.

~~**FR-6 — `interpolateColors` không nhận `clamp`.**~~ **Creator bỏ — không làm.**

**FR-7 — Máy quay dùng `Camera`, không tự viết zoom.** Mục G-AI thêm câu soát: không có `div` nào mang `transform: scale(...)` để làm máy quay; zoom/lia là prop `camera` của `<Scene>` hoặc `<Camera>` (luật L8). Ví dụ ở mục D-AI đã dùng `<Camera>`.
- AC: golden test của `remotion_engineer_ai` có câu soát này.

**Ngoài phạm vi**: lớp component dùng chung cho hình trừu tượng (xem "Đánh giá góp ý", đề xuất CR riêng); luật bố cục / kiểm bố cục (CR-062 đã xử lý, số lỗi đã thấp); độ dài prompt (CR-063); trần suy nghĩ (backlog); prompt thủ công `remotion_engineer`; Manim.

## Giải pháp đề xuất

Nguyên tắc: **sửa chỗ prompt và khung code nói khác nhau, thay vì thêm luật cấm**. Ba nguyên nhân lớn nhất (≈ 92% lỗi biên dịch) đều là model làm đúng theo những gì nó được cho thấy, nhưng thứ nó được cho thấy sai hoặc thiếu.

**FR-1 `random` — có hai cách:**
- **(a) Import thêm vào khung (đề xuất).** Một dòng ở `_REMOTION_HEAD`. `random` của remotion là ngẫu nhiên có seed, an toàn cho render nhiều lần. Import thừa không gây lỗi (khung hiện đã import `spring`, `Easing` mà nhiều script không dùng).
- (b) Bỏ `random` khỏi prompt AI. Model mất cách duy nhất để rải sao/hạt theo luật L9, và vẫn có thể tự nghĩ ra `random`.

**FR-4 kiểu prop — có hai cách:**
- **(a) Trích kiểu từ code lúc dựng prompt (đề xuất).** Code của hình đã có trong `req.illustrations`; 221/256 hình hiện có viết theo khuôn `export function X({...}: FigureProps & {...})`. Áp ngay cho mọi hình cũ, luôn khớp code thật.
- (b) Sửa prompt tạo hình để `usage` ghi kiểu. Chỉ có tác dụng với hình mới; `usage` vẫn có thể lệch code sau khi Creator sửa hình.

## Phụ thuộc

Đã kiểm `git branch -a --no-merged origin/main` và `scripts/worktree.sh list`:
- `feature/cr-064-screenwriter-prompt-trim` (thu gọn prompt Story Architect): **độc lập**. Cùng sửa `prompt_template_seeds.go` nhưng ở dòng version của `story_architect` (dòng 20) và các khối của Story Architect; CR này chỉ đổi dòng version `remotion_engineer_ai` (dòng 26). Hai thay đổi cách nhau 5 dòng không đổi, git ghép không xung đột. Không chung vai trò, contract hay bảng.
- `chore/worktree-per-cr`: không có thay đổi so với `origin/main` — độc lập.

## Phạm vi

- **llm-service**
  - `app/pipeline/merger.py`: hằng mới cho các tên import của khung (`REMOTION_API`, `PRIMITIVES`); `_REMOTION_HEAD` dựng từ chúng.
  - `app/pipeline/prompts.py`: `remotion_chunk` (danh sách tên sinh từ hằng của merger), `_palette_table` (FR-5) — dùng ở chunk, layout, repair.
  - `app/pipeline/run.py`: `library_section` thêm dòng kiểu prop.
  - Module mới `app/pipeline/library_props.py`: trích kiểu prop từ code hình.
  - Contract HTTP, DB: không đổi.
- **authoring-service**
  - `internal/domain/prompt_template_seeds_ai.go`: `remoBAIVI`, `remoGAIVI` (chỉ prompt AI).
  - `internal/domain/prompt_template_seeds.go`: chỉ dòng 26, version `remotion_engineer_ai` 4 → 5.
  - `internal/domain/golden_prompts_test.go`: thêm kiểm cho FR-5/7 trong `TestEngineerAIPromptsWriteShotsOnlyAndShareTheRulebook`.
- **Ai phụ thuộc**: `_REMOTION_HEAD` dùng ở `merge_remotion`, `remotion_frame_text`, `head_with_library_imports`; `_palette_table` dùng ở mọi prompt Remotion của llm-service; `library_section` dùng ở `_system` và `run`. Prompt thủ công `remotion_engineer` không đổi.

## Kế hoạch thực hiện

1. **merger.py** — tách tên import thành hằng:
   - `REMOTION_API = ("registerRoot", "Composition", "AbsoluteFill", "interpolate", "interpolateColors", "spring", "Easing", "useCurrentFrame", "useVideoConfig", "random")`.
   - `PRIMITIVES = ("Stage", "SAFE_MARGIN", "WIDTH", "HEIGHT", "BACKGROUND", "useFrameBox")`.
   - `_REMOTION_HEAD` dựng hai dòng đó từ hằng. Hàm mới `available_names_text()` trả chuỗi danh sách theo module (react; remotion: …; segments; primitives: …; lottie: LottieClip; illustration: kit; scene: SCENE_KIT; backdrops: BACKDROP_KIT) để prompts dùng.
   - Kiểm `head_with_library_imports`: hình thư viện import `random` thì không import lần hai (đã có logic bỏ trùng; cập nhật kỳ vọng ở `test_backdrops.py:61`).
2. **prompts.py**
   - `remotion_chunk`: dòng "các dòng import (...)" dùng `merger.available_names_text()`.
   - `_palette_table`: `- PALETTE.{key} = {hex} — {role}: {meaning}`. Sửa các test đang khớp chuỗi cũ.
3. **library_props.py** (mới) — `prop_signature(code: str, name: str) -> str | None`:
   - Tìm `export function {name}(`; đi qua khối destructure `{...}` bằng đếm ngoặc; sau đó phải là `:`; lấy kiểu tới `)` đóng hàm bằng đếm ngoặc `{}`/`()`/`<>`.
   - Kiểu dạng `FigureProps & { ... }` → lấy phần trong `{ ... }`; dạng khác → `None`.
   - Thay mỗi tên kiểu khai báo trong cùng file bằng `type X = <hợp các literal>;` bằng giá trị của nó (chỉ alias là hợp literal chuỗi/số).
   - Gộp khoảng trắng thành một dòng `color?: string; open?: 'closed' | 'ajar' | 'open'`. Quá 400 ký tự → `None` (giữ prompt gọn, rơi về dòng cũ).
4. **run.py `library_section`** — mỗi hình không phải nền: dòng cũ, thêm `\n  props: {signature}` khi trích được. Đoạn mở đầu C4 thêm câu: "Giá trị của mỗi prop phải đúng kiểu ghi ở dòng props (chuỗi trong dấu nháy là danh sách giá trị cho phép)."
5. **prompt_template_seeds_ai.go**
   - `remoBAIVI` mục 2: "viết đúng ¤PALETTE.<khoá>¤ (có dấu chấm); không tự đặt hằng ¤PALETTE_...¤ hay tên 'PLACEHOLDER'". (FR-3, FR-6 bỏ: không đổi `remoDAIVI`, không thêm mục E riêng, không thêm câu về `interpolateColors`.)
   - `remoGAIVI` mục 3: thêm "không có tên ¤PALETTE_...¤"; mục 5: "máy quay chỉ bằng prop ¤camera¤ / ¤<Camera>¤ — không ¤div¤ nào mang ¤transform: scale(...)¤ để zoom" (FR-7); mục 6: "chỉ dùng tên ở mục E; giá trị prop của hình thư viện đúng kiểu ở dòng props".
6. **prompt_template_seeds.go:26** — version 4 → 5.
7. **Test**
   - llm-service: `test_merger.py` (head có `random`, `BACKGROUND`, `useFrameBox`; biên dịch không cần), `test_prompts.py` (danh sách tên chunk = tên trong head; bảng màu có `PALETTE.`), `test_backdrops.py` (cập nhật dòng 61; C4 có dòng props), test mới `test_library_props.py` (NarrowDoor với alias, Galaxy literal inline, hình không có khối props → `None`, khối quá dài → `None`).
   - authoring-service: `golden_prompts_test.go` — `remotion_engineer_ai` có câu cấm `PALETTE_` (FR-5) và câu soát máy quay (FR-7); hash `remotion_engineer` không đổi.
   - rendering: chạy `tests/domain/test_illustration_kit.py` (danh sách kit giữ nguyên).

## Kiểm tra

- `pytest` llm-service, `go test ./...` + `go vet` + `gofmt` authoring-service, test kit của rendering.
- Biên dịch thật: gửi một script có shot dùng `random('x')`, `useFrameBox()` tới `/v1/check/remotion` của rendering đang chạy → `ok`.
- Rebuild `llm-service`, `authoring-service`; xác nhận DB có `remotion_engineer_ai` version mới đang bật, không có prompt Creator đè lên.
- Live: chạy lại bước code của project `257acc58` (có `NarrowDoor`, `Galaxy`) và so `code_check_diagnostics` / số lượt repair trong `llm_usage` với lần trước.

## Rủi ro

- Trích kiểu bằng đếm ngoặc, không phải trình phân tích TypeScript: hình viết khác khuôn sẽ rơi về dòng cũ (không tệ hơn hiện nay). Không bao giờ ghi kiểu đoán.
- Tin nhắn chunk và system prompt dài thêm một ít (vài trăm ký tự cho danh sách tên, ~80 ký tự mỗi hình thư viện).
- Không đảm bảo hết lỗi `TS2322`: model vẫn có thể dùng sai kiểu dù đã thấy. Đo lại sau khi chạy thật.
- Không mất dữ liệu, không đổi contract/DB/giao diện.
