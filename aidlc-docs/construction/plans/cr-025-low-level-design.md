# CR-025 — Low-Level Design: pipeline soạn script bốn bước và cổng review

## Date
2026-09-11

## Stage
Low-Level Design (Functional/NFR/Infrastructure Design: SKIP — không hạ tầng
mới, không NFR mới; toàn bộ là prompt-builder + linter, cùng dạng
`scriptPrompts.ts`/`script_lint.py` đã có)

## Phạm vi
FR71–FR74. Không đụng saga render, không đụng `narrate()`/lượt dry của
CR-018/020, không đụng QC sau render của CR-021 — CR này chỉ ở phía soạn,
trước khi Creator dán code vào `ScriptEditor`.

---

## Quyết định kiến trúc

### D1 — Bốn màn hình tuần tự trong web-gui, không phải bốn field trên một màn
Theo đúng khuôn `ScriptAssistant.tsx` hiện có (điền form → sinh prompt → copy →
dán kết quả). Thêm 3 bước mới **trước** `ScriptAssistant`/`ScriptEditor` hiện
tại:

```
WizardNav bước "Soạn" tách thành:
  1a. StoryArchitect   — điền chủ đề + format → prompt bước 1 → dán story
  1b. VisualDirector   — nhận story tự động → prompt bước 2 → dán storyboard
  1c. ManimEngineer    — nhận storyboard tự động → prompt bước 3 → dán code
  1d. ScriptReviewer   — nhận story+storyboard+code+lint → prompt review →
                          PASS thì sang ScriptEditor, REVISE thì quay lại 1b
```
`ScriptAssistant` hiện tại (một prompt gộp) **không bị xoá** — theo FR71.5, nó
trở thành lối tắt "Bỏ qua, soạn trực tiếp" để không phá quy trình đã quen.

### D2 — Artefact lưu trên `ProjectDraftContext`, không lưu backend
`story` và `storyboard` là dữ liệu **trước khi có project** (project chỉ tồn
tại từ lúc `POST /v1/sagas/render`). Toàn bộ wizard hiện tại (chọn giọng, ngôn
ngữ, định dạng...) đã sống trong `ProjectDraftContext` theo đúng cách này —
CR này thêm hai field vào draft đó, không mở bảng DB mới:

```ts
interface ProjectDraft {
  ...
  authoringStory?: string;       // artefact bước 1
  authoringStoryboard?: string;  // artefact bước 2
}
```
Quay lại bước 1b từ 1c không mất `authoringStory` (FR71.4) vì nó không nằm
trong state cục bộ của màn 1c, nó nằm ở context cha — đúng pattern React đang
dùng cho các field khác của draft.

Khi Creator bấm "Tạo video" ở cuối wizard, `authoringStory`/`authoringStoryboard`
**không gửi lên server** — chúng chỉ là artefact soạn thảo, script cuối cùng
(kết quả bước 3) mới là thứ đi vào `RenderInput.script_content`. Không thêm cột
DB, không thêm field trên `Project`.

### D3 — Bốn prompt builder, cùng module `scriptPrompts.ts`
Thêm 4 hàm mới cạnh các hàm build prompt đã có (`buildBeatSheetSection` etc.),
tái dùng chúng thay vì viết lại:

```ts
buildStoryArchitectPrompt(topic, format, language, wpm): string
buildVisualDirectorPrompt(story, format, language): string
buildManimEngineerPrompt(storyboard, format, language): string
buildScriptReviewPrompt(story, storyboard, code, lintWarnings): string
```
`buildManimEngineerPrompt` giữ **nguyên văn** toàn bộ ràng buộc kỹ thuật đang
nằm trong prompt gộp hiện tại (whitelist conceptflow, `narrate()`/`beat()`,
một Scene class, không I/O) — FR72.5. Không viết lại từ đầu, cắt đoạn kỹ thuật
ra khỏi prompt cũ và ghép vào đây.

`buildScriptReviewPrompt` nhận `lintWarnings` — kết quả FR73 chạy **cục bộ
trong web-gui bằng TypeScript**, không gọi rendering service (script chưa có
project để gọi qua saga). Nghĩa là D4 cần một bản kiểm nhẹ phía client.

### D4 — Luật FR73 có hai bản: TypeScript (soạn) và Python (validate_script)
`script_lint.py` (rendering, Python, AST) là **nguồn sự thật** — nó chạy trong
lượt dry của `validate_script`, kiểm bằng cách parse code Python thật. Nhưng
lúc Creator còn ở bước 1d (ScriptReviewer), project chưa tồn tại, không có gì
để gọi `validate_script`.

Giải pháp: `web-gui/src/utils/scriptLint.ts` — bản kiểm **xấp xỉ** bằng regex
trên text code (không parse AST đầy đủ, JS không có `ast` module tương đương),
đủ để bắt bốn luật FR73:
- FR73.1: đếm số lần xuất hiện `Transform(`, `ReplacementTransform(`,
  `.animate` — nếu 0 và có ít nhất 2 `FadeIn(`/`FadeOut(` thì cảnh báo.
- FR73.2: đếm `Text(`/`MarkupText(`/`Tex(`/`MathTex(` trên tổng số lời gọi
  constructor mobject nhận diện được — tỉ lệ thô, không chính xác bằng AST
  nhưng đủ để cảnh báo.
- FR73.3: cần mốc thời gian thật của lời thoại — **không làm được ở bản
  TypeScript** vì chưa render. Bỏ ở bước 1d, chỉ chạy ở `script_lint.py` lúc
  `validate_script` thật (đã có sẵn hạ tầng report `validation_warnings` của
  CR-024).
- FR73.4: đếm chuỗi hex `#RRGGBB` khác nhau xuất hiện trong code.

Đây là đánh đổi có ý thức: bản TypeScript là "bắt sớm, không chính xác tuyệt
đối", bản Python là "bắt đúng lúc render, chính xác vì có AST + dữ liệu thật".
Không cố làm bản TypeScript hoàn hảo bằng cách nhúng một Python interpreter
vào trình duyệt.

`script_lint.py` (Python, rendering) thêm 3 luật còn lại được (FR73.1, 73.2,
73.4 — làm đúng bằng AST, thay bản xấp xỉ) + FR73.3 (cần `layout`/`mark` records
nên chỉ làm được ở đây). Cả bốn luật **đều ở đây**, chạy trong `validate_script`
thật, kết quả vào `validation_warnings` (FR73.6) — bản TypeScript ở D4 chỉ là
phản hồi tức thời cho bước 1d, không thay thế nó.

### D5 — Prompt review và vòng REVISE (FR74)
Màn `ScriptReviewer` (1d) không tự động làm gì với câu trả lời PASS/REVISE của
model — Creator dán kết quả review vào một ô, màn hình đọc dòng đầu tiên tìm
chuỗi `PASS` hoặc `REVISE` (case-insensitive), hiện huy hiệu tương ứng.

- **PASS**: nút "Sang bước soạn code" bật, dẫn tới `ScriptEditor` với code hiện
  tại đã điền sẵn.
- **REVISE**: nút "Sửa storyboard" bật, điều hướng về 1b (`VisualDirector`) với
  `authoringStoryboard` **giữ nguyên** (Creator sửa tay hoặc dán bản mới) và một
  field mới `revisionNotes` (toàn bộ nội dung Creator dán ở bước review) được
  nối vào cuối prompt bước 2 lần sau — đúng FR74.3.
- Không đếm số lần revise, không giới hạn cứng (FR74.4: Creator tự quyết định
  dừng) — nhưng hiện số lần đã revise ở góc màn để Creator tự biết đang lặp lại
  bao nhiêu lần, tránh vòng lặp vô thức.

## Rủi ro
- **Bản lint TypeScript và Python lệch nhau.** Chấp nhận có ý thức (D4) — bản
  TypeScript chỉ là phản hồi sớm, không phải cổng chặn. Test cả hai bản trên
  cùng bộ fixture để biết chúng lệch bao nhiêu, không để lệch âm thầm.
- **Bốn màn hình làm nặng wizard.** Giảm nhẹ bằng lối tắt "Bỏ qua" (D1) — luôn
  hiện, không chôn trong menu.
- **Regex đếm mobject/màu sai khi code có comment/string chứa các chuỗi đó.**
  Chấp nhận: đây là luật cảnh báo xấp xỉ, không phải luật đúng tuyệt đối; luật
  đúng tuyệt đối đã có ở bản Python.

## Không làm
- Không gọi API AI nào từ hệ thống (giữ đúng Quyết định copy tay).
- Không thêm bảng DB, không thêm field trên `Project`/`RenderInput`.
- Không giới hạn số lần REVISE.
- Không làm luật FR73.3 (đoạn thoại dài không animation) ở bản TypeScript —
  cần dữ liệu render thật, không giả được.

## Kiểm chứng
- Unit test (TypeScript): 4 hàm build prompt chứa đúng artefact truyền vào;
  `scriptLint.ts` bắt được 3/4 luật xấp xỉ trên fixture cố ý sai.
- Unit test (Python): `script_lint.py` bắt cả 4 luật FR73 trên fixture cố ý
  sai, và không bao giờ trả `severity=blocking` cho chúng.
- Unit test: quay lại `VisualDirector` từ `ScriptReviewer` giữ nguyên
  `authoringStory`; `revisionNotes` xuất hiện trong prompt bước 2 lần sau.
- Thủ công: soạn cùng một chủ đề qua đường bốn bước và đường một-prompt cũ, so
  chất lượng kể chuyện của video ra (Kiểm chứng gốc của CR-025).
