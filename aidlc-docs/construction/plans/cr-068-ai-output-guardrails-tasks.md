# CR-068 — Danh sách task

Thiết kế: [cr-068-ai-output-guardrails-design.md](cr-068-ai-output-guardrails-design.md) (đã duyệt). Nhánh `feature/cr-068-ai-output-guardrails`.

Đường dẫn dưới đây tính từ gốc repo. Lệnh test llm-service chạy trong `services/llm-service` (`pytest`, `ruff check app tests`); authoring-service trong `services/authoring-service` (`go test ./...`, `go vet ./...`, `gofmt -l .`).

### T1 — `merger.frame_names()`: tập tên khung code đưa cho shot
- [x] Xong
- **Service**: llm-service
- **File**: `services/llm-service/app/pipeline/merger.py` — hàm mới `frame_names()` đặt ngay sau `available_names_text()` (khoảng dòng 118-132).
- **Thay đổi**: trả `frozenset[str]` gồm mọi tên `_REMOTION_HEAD` import cho shot và các hằng khung code khai báo, dựng từ chính các tuple để không lệch:
  ```python
  def frame_names() -> frozenset[str]:
      """Every name the Remotion frame puts in scope for the shot functions:
      the imports of the head and the constants the frame declares."""
      return frozenset((
          "React", *REMOTION_API, *SEGMENT_API, *PRIMITIVES, "LottieClip",
          *ILLUSTRATION_KIT, *ILLUSTRATION_HELPERS, "FigureProps", "Mood", "PersonPose",
          *SCENE_KIT, *SCENE_HELPERS, *BACKDROP_KIT, *RIG_KIT,
          "PALETTE", "LAYOUT", "clamp", "ShotProps",
      ))
  ```
- **Test**: `services/llm-service/tests/test_merger.py`, ca `test_frame_names_cover_every_import_of_the_head`: mọi tên đứng trong `{...}` của các dòng `import` trong `merger._REMOTION_HEAD` (và `React`, `FigureProps`, `Mood`, `PersonPose`) đều có trong `frame_names()`; `useCurrentFrame`, `PALETTE`, `LAYOUT`, `clamp` có; `useCurrentFrameSafe` không có. Chạy `pytest tests/test_merger.py`.
- **Xong khi**: test qua (nền cho FR-3).
- **Phụ thuộc**: —

### T2 — `autofix.fix_palette_names`: đổi `PALETTE_<X>` thành `PALETTE.<khoá>`
- [x] Xong
- **Service**: llm-service
- **File**: `services/llm-service/app/pipeline/autofix.py` (mới).
- **Thay đổi**:
  - Docstring module: các lỗi trong code AI viết có đúng một cách sửa được sửa ở đây, không gọi model; chỗ nào không chắc thì giữ nguyên để bước kiểm tra và repair xử lý.
  - Hằng `RULE_PALETTE_NAME = "palette_name"`, `RULE_TSC_SUGGESTION = "tsc_suggestion"`, `KIND_AUTOFIX = "autofix"`.
  - `@dataclass(frozen=True) class Fix: shot: str; line: int; rule: str; before: str; after: str` (line = dòng trong code của shot, đếm từ 1).
  - `_norm(s) = s.replace("_", "").lower()`.
  - `_PALETTE_IDENT = re.compile(r"\bPALETTE_([A-Za-z0-9_]+)\b")`.
  - `fix_palette_names(shot: str, code: str, keys: Iterable[str]) -> tuple[str, list[Fix]]`:
    - với mỗi match: `rest = match.group(1)`; bỏ các token `PLACEHOLDER` (không phân biệt hoa thường) khi tách theo `_`, ghép lại; rỗng → giữ nguyên.
    - ứng viên = các khoá `k` có `_norm(k) == _norm(rest)`; đúng một ứng viên → thay bằng `PALETTE.<k>`, ghi `Fix(shot, line, RULE_PALETTE_NAME, "PALETTE_<rest gốc>", "PALETTE.<k>")`; 0 hoặc ≥ 2 → giữ nguyên.
    - thay theo từng dòng (`code.split("\n")`) để biết số dòng; giữ nguyên mọi ký tự khác.
- **Test**: `services/llm-service/tests/test_autofix.py` (mới):
  - `test_palette_name_matching_one_key_becomes_a_palette_member`: khoá `["nenTroi", "nenHangDong", "chim"]`; code có `PALETTE_NEN_TROI`, `PALETTE_nenHangDong`, `PALETTE_CHIM_PLACEHOLDER` trên ba dòng → `PALETTE.nenTroi`, `PALETTE.nenHangDong`, `PALETTE.chim`; ba `Fix` với đúng dòng 1,2,3 và `before`/`after`.
  - `test_placeholder_without_a_role_is_left_for_repair`: `PALETTE_PLACEHOLDER` → code không đổi, không `Fix`.
  - `test_name_matching_two_keys_is_left_alone`: khoá `["nenTroi", "nen_troi"]` (cùng `_norm`) → không đổi.
  - `test_palette_member_access_is_not_touched`: `PALETTE.nenTroi` và chuỗi `'PALETTE.x'` không bị đụng.
- **Xong khi**: test qua (FR-2, AC-2).
- **Phụ thuộc**: —

### T3 — `autofix.fix_tsc_suggestions`: áp gợi ý "Did you mean" của tsc
- [x] Xong
- **Service**: llm-service
- **File**: `services/llm-service/app/pipeline/autofix.py`.
- **Thay đổi**:
  - `_SUGGESTION = re.compile(r"^(TS255[12]): (?:Cannot find name|Property) '([A-Za-z_$][\w$]*)'.*Did you mean '([A-Za-z_$][\w$]*)'\?", re.S)`.
  - `fix_tsc_suggestions(shots: dict[str, str], merged: Merged, diags: list[Diagnostic], names: Collection[str], applied: set[tuple[str, str, str]]) -> tuple[dict[str, str], list[Fix]]`:
    - với mỗi diag có `line` và khớp `_SUGGESTION` → `code, a, b`; `shot = merged.shot_at(line)`; bỏ qua nếu `shot` không có trong `shots` (frame, LAYOUT, stub, thư viện hình); bỏ qua nếu `(shot, a, b)` trong `applied`.
    - `TS2552`: chỉ khi `b in names`; trên dòng `local = line - merged.lines[shot][0]` (index 0-based) thay `\b{a}\b` (regex escape, không đứng sau `.`: `(?<![\w$.])a(?![\w$])`) bằng `b`.
    - `TS2551`: chỉ khi `_norm(a) == _norm(b)`; trên dòng đó thay `.a` (`\.a(?![\w$])`) bằng `.b`.
    - dòng thay được (có ít nhất một chỗ đổi) → cập nhật code shot, thêm `(shot, a, b)` vào `applied`, ghi `Fix(shot, local + 1, RULE_TSC_SUGGESTION, a, b)`; không đổi được gì → không ghi.
    - trả về `{shot: code mới}` chỉ cho các shot đã đổi, và danh sách `Fix`.
- **Test** (`tests/test_autofix.py`), dựng `Merged` bằng `merger.merge_remotion` từ một storyboard nhỏ (mẫu như `tests/test_pipeline.py` `storyboard()` + `tsx()`), tìm số dòng qua `merged.code.splitlines()`:
  - `test_cannot_find_name_takes_the_frame_name_tsc_suggests`: shot 1.1 dùng `useCurrentFrameSafe()`; diag `TS2552: Cannot find name 'useCurrentFrameSafe'. Did you mean 'useCurrentFrame'?` ở dòng đó, `names=merger.frame_names()` → code shot có `useCurrentFrame()`, một `Fix` rule `tsc_suggestion`, `applied` có cặp.
  - `test_suggestion_outside_the_frame_names_is_left_for_repair`: gợi ý `'Shot1_2'` → không đổi.
  - `test_property_suggestion_differing_only_in_case_is_applied`: `PALETTE.conNguoi`, diag `TS2551: Property 'conNguoi' does not exist on type '{...}'. Did you mean 'connguoi'?` → `PALETTE.connguoi`.
  - `test_property_suggestion_with_another_name_is_left_for_repair`: gợi ý `'nenpanel'` cho `conNguoi` → không đổi.
  - `test_a_suggestion_already_applied_is_not_applied_again`: `applied` chứa cặp → không đổi.
  - `test_a_line_outside_every_shot_is_ignored`: diag trỏ dòng trong phần khung → không đổi.
- **Xong khi**: test qua (FR-3, AC-3).
- **Phụ thuộc**: T1

### T4 — Gắn sửa tên màu sau mỗi lượt chunk và repair, phát sự kiện `autofix`
- [x] Xong
- **Service**: llm-service
- **File**: `services/llm-service/app/pipeline/run.py`.
- **Thay đổi**:
  - Import `autofix`.
  - Hàm module mới cạnh `_check_event` (khoảng dòng 930):
    ```python
    def _autofix_event(round_: int, segment: str, plan: Plan, fixes: list[autofix.Fix]) -> dict:
        """What the run changed on its own, in the shape of a check event so the
        caller logs it with the diagnostics."""
        return {"type": "check", "phase": autofix.KIND_AUTOFIX, "round": round_, "segment": segment,
                "diagnostics": [{"message": f"{f.before} → {f.after}", "line": f.line, "kind": autofix.KIND_AUTOFIX,
                                 "rule": f.rule, "shot": f.shot, "segment": plan.owner(f.shot) or segment}
                                for f in fixes]}
    ```
  - Phương thức `CodePipeline._fix_palette(self, plan, shots: dict[str, str], ids) -> list[Fix]`: chỉ khi engine Remotion; `keys = merger.palette_keys(plan.sb).values()`; với mỗi id trong `ids` chạy `autofix.fix_palette_names`, ghi đè `shots[id]`, gom `Fix`.
  - `do_chunk` (dòng 738-760): sau `written = {...}` và trước `if outcome.failures`, nếu `remotion`: `fixes = self._fix_palette(plan, written, list(outcome.shots))`; có `fixes` → `await emit(_autofix_event(0, seg.key, plan, fixes))`. Như vậy cả nội dung lưu dở (segment_failed) lẫn segment_done mang code đã sửa.
  - `_repair` (dòng 856-927): `_repair` không có `emit`; thêm tham số `emit: Emit` và `round_: int`, truyền từ hai nơi gọi (`run` dòng ~808 với `rounds`, `_settle_chunk` dòng ~854 với `rounds`). Sau vòng `for key, code in results.items()` nếu `remotion`: sửa tên màu cho các shot vừa nhận, phát `_autofix_event(round_, plan.owner(...) , plan, fixes)` theo từng segment (gom theo `plan.owner`).
- **Test**: `services/llm-service/tests/test_pipeline.py`:
  - Thêm vào `FakeProvider` tham số `palette_typo=set()`: shot trong tập này, lượt chunk đầu tiên trả thân `return <div style={{color: PALETTE_ACCENT}} />;` (storyboard mẫu có vai trò `accent`).
  - `test_a_palette_name_written_as_a_constant_is_fixed_before_the_check`: `FakeProvider(palette_typo={"1.2"})` → `segment_done` của đoạn chứa 1.2 có `PALETTE.accent`, không có `PALETTE_ACCENT`; có đúng một sự kiện `check` phase `autofix` với diagnostic `kind="autofix"`, `rule="palette_name"`, `shot="1.2"`, `message="PALETTE_ACCENT → PALETTE.accent"`; không có lượt `repair` nào.
  - `test_a_palette_name_in_a_repair_reply_is_fixed`: provider mà lượt repair trả về shot có `PALETTE_ACCENT` → code cuối có `PALETTE.accent`, có sự kiện autofix.
- **Xong khi**: test qua (FR-2, FR-4, AC-2, AC-4).
- **Phụ thuộc**: T2

### T5 — Áp gợi ý tsc trước khi gọi repair (chunk và kiểm tra cuối)
- [x] Xong
- **Service**: llm-service
- **File**: `services/llm-service/app/pipeline/run.py` — `_settle_chunk` (dòng 828-854) và vòng `while True` trong `run` (dòng 791-819).
- **Thay đổi**:
  - Phương thức `CodePipeline._names(self) -> frozenset[str]`: `merger.frame_names() | frozenset(self._library)`.
  - `_settle_chunk`: tạo `applied: set = set()` trước vòng lặp. Khi `check` không qua và đã phát `_check_event`: gọi `autofix.fix_tsc_suggestions(shots, merged, check.diagnostics, self._names(), applied)` (chỉ các shot của chunk này có trong `shots`); có thay đổi → `shots.update(changed)`, `await emit(_autofix_event(rounds, seg.key, plan, fixes))`, `continue` (biên dịch lại, **không** tăng `rounds`). Không có thay đổi → đi tiếp như hiện nay (`_map_failures` → repair).
  - `run`, vòng kiểm cuối: chỉ Remotion; tương tự, `applied` khai báo trước `while`; sau khi phát `_check_event("final", ...)` và trước điều kiện `if check.ok or rounds >= self._repair_rounds: break`: thử `fix_tsc_suggestions(shots, merged, ...)`; có thay đổi → `shots.update`, phát autofix (segment `""`), phát `segment_done` `repaired: False` cho mỗi segment sở hữu shot bị đổi (nội dung như nhánh repair), `merged = merge()`, `continue`. Lặp không vô hạn vì `applied` chặn cặp đã áp và mỗi lần phải có ít nhất một thay đổi.
- **Test** (`tests/test_pipeline.py`):
  - `FakeChecker` thêm chế độ `suggest=True`: dòng chứa `useCurrentFrameSafe` → `Diagnostic("TS2552: Cannot find name 'useCurrentFrameSafe'. Did you mean 'useCurrentFrame'?", line, rule="TS2552")`, và check không qua khi còn chuỗi đó.
  - `FakeProvider` thêm `typo_name={"1.2"}`: thân shot `const f = useCurrentFrameSafe(); return null;`.
  - `test_a_name_tsc_suggests_is_fixed_without_a_repair_turn` (chunk=2, 4 shot → có kiểm tra chunk): `res.check_ok`, `"useCurrentFrameSafe" not in res.code`, `res.repair_rounds == 0`, không có `Call` phase `repair`, có sự kiện autofix `rule="tsc_suggestion"`, `message="useCurrentFrameSafe → useCurrentFrame"`.
  - `test_a_single_chunk_run_fixes_the_suggestion_at_the_final_check` (chunk=10, chỉ một chunk → chỉ có kiểm tra cuối): như trên, và `segment_done` cuối của đoạn mang code đã sửa.
- **Xong khi**: test qua (FR-3, FR-4, AC-3, AC-4).
- **Phụ thuộc**: T3, T4

### T6 — Bỏ chia đôi khi vượt trần suy nghĩ
- [x] Xong
- **Service**: llm-service
- **File**: `services/llm-service/app/pipeline/run.py`.
- **Thay đổi**:
  - Dòng 66-67: `# A chunk or repair turn cut off mid-answer (max_tokens) is written again as two halves.` / `SPLIT_KINDS = (errors.TRUNCATED,)`.
  - Docstring module (dòng 17-21): "A chunk cut off mid-answer is written again as two halves, down to one shot; … A chunk that ran past the reasoning limit is not split: smaller turns think as long, so the segment fails with every unwritten shot and the Creator re-runs it." — giữ ý về đoạn còn thiếu shot.
  - Docstring `write` (dòng ~708-711) và `_repair` (dòng ~867-868): "ran out of budget" → "was cut off mid-answer".
  - `_failure_reason`/`shot_failure_message` giữ nguyên (`tried` chỉ còn một phần tử với `budget`, nên thông báo không còn "(đã thử …)").
- **Test** (`tests/test_pipeline.py`), sửa ca cũ cho đúng hành vi mới:
  - `test_a_grouped_repair_that_runs_out_of_budget_is_split_down_to_single_shots` → đổi tên `test_a_grouped_repair_over_the_reasoning_limit_keeps_the_old_code`: `budget_above=1`, rounds mặc định → nhóm `1.1,1.2,1.3` được gọi mỗi vòng một lần, không có nhãn một shot nào; `not res.check_ok`; mọi lượt repair lỗi là `budget`.
  - Thêm `FakeProvider` tham số `truncated_above=0` (như `budget_above` nhưng `errors.TRUNCATED`) và ca `test_a_grouped_repair_cut_off_mid_answer_is_split_down_to_single_shots` giữ nguyên các assert của ca cũ (với `TRUNCATED`).
  - `test_a_chunk_over_budget_is_split_in_two_and_the_run_succeeds` → đổi thành `test_a_chunk_cut_off_mid_answer_is_split_in_two_and_the_run_succeeds` dùng `errors.TRUNCATED` (assert giữ nguyên, nhãn lỗi `TRUNCATED`).
  - Ca mới `test_a_chunk_over_the_reasoning_limit_fails_its_segment_without_splitting`: `Scripted(fail=BUDGET khi len(ids)==5)`, chunk=5, 7 shot → `res.failed == ["1.1-1.5"]`, `chunk_turns` có đúng một lượt cho 1.1-1.5, không có `chunk_split`; `segment_failed.failed_shots == ["1.1",…,"1.5"]`, `content is None`.
  - `test_a_one_shot_chunk_over_budget_fails_its_segment_with_budget`: calls thành `[("1.1-1.2", False)]`, `failed_shots == ["1.1", "1.2"]`.
  - `test_a_split_chunk_keeps_the_shots_written_before_one_shot_fails`, `test_both_halves_run_when_the_first_half_fails`, `test_a_partly_written_segment_writes_only_its_missing_shots`, `test_a_stop_now_error_inside_a_split_still_stops_the_run`: đổi lỗi chia đôi sang `errors.TRUNCATED`; thông báo mong đợi đổi lý do thành `"câu trả lời bị cắt giữa chừng vì hết token"`.
  - Ca mới `test_a_budget_failure_message_names_the_chunk_only`: SceneThree, BUDGET mọi lượt, `max_reasoning_chars=100000` → `message == "Shot 3.3–3.5: model suy nghĩ quá 100000 ký tự mà chưa viết được chữ nào."`.
  - Chạy `pytest tests/test_pipeline.py`.
- **Xong khi**: test qua (FR-6, AC-6).
- **Phụ thuộc**: — (làm sau T4/T5 để tránh xung đột sửa cùng file)

### T7 — Trần suy nghĩ 100 000
- [x] Xong
- **Service**: llm-service, hạ tầng
- **File**: `services/llm-service/app/config.py:72`; `docker-compose.yml:458`; `.env.example:82`; `services/llm-service/tests/test_api.py:21,290`.
- **Thay đổi**: `_non_negative("CODE_MAX_REASONING_CHARS", 100000)`; compose `${CODE_MAX_REASONING_CHARS:-100000}`; `.env.example` `# CODE_MAX_REASONING_CHARS=100000`; test: fixture dòng 21 `code_max_reasoning_chars=100000`, assert dòng 290 `(100000, 0)`.
- **Test**: `pytest tests/test_api.py -k reasoning`.
- **Xong khi**: mặc định 100 000 ở cả code và compose (FR-5).
- **Phụ thuộc**: —

### T8 — Prompt `remotion_engineer_ai`: bỏ tên sai làm ví dụ
- [x] Xong
- **Service**: authoring-service
- **File**: `services/authoring-service/internal/domain/prompt_template_seeds_ai.go:128,175`; `services/authoring-service/internal/domain/golden_prompts_test.go:101-102`.
- **Thay đổi**:
  - Dòng 128 thành: `2. MỌI màu trong code (fill, stroke, color, background, border, boxShadow) phải là ¤PALETTE.xxx¤: hằng ¤PALETTE¤, dấu chấm, rồi khoá viết y như danh sách trong tin nhắn (vd. ¤PALETTE.nenTroi¤). Không viết mã hex/rgb/tên màu nào khác. Cần độ trong suốt → dùng ¤opacity¤ của phần tử, không tự pha màu. (Màu mặc định BÊN TRONG các hình của bộ minh hoạ ở C3 không tính: chúng thuộc bộ hình.)` (mục 4 "chọn khoá gần nghĩa nhất" đã có sẵn, giữ nguyên).
  - Dòng 175, thay đoạn `viết dạng ¤PALETTE.khoá¤ — không có tên ¤PALETTE_...¤ nào?` bằng `viết dạng ¤PALETTE.khoá¤ với khoá có trong danh sách?`.
  - Golden test dòng 102: thay hai chuỗi bằng `"khoá viết y như danh sách trong tin nhắn (vd. `+"`PALETTE.nenTroi`"+`)"` và `"viết dạng `+"`PALETTE.khoá`"+` với khoá có trong danh sách?"` (viết đúng cú pháp chuỗi Go của file); thêm sau vòng `want`: `if strings.Contains(remo, "PALETTE_") { t.Error("remotion_engineer_ai must not spell a PALETTE_ name, the model copies it") }`.
- **Test**: `go test ./internal/domain/ -run TestEngineerAIPrompts`; rồi `go test ./... && go vet ./... && gofmt -l .`.
- **Xong khi**: prompt không còn `PALETTE_` (FR-1, AC-1).
- **Phụ thuộc**: —

### T9 — Hợp đồng sự kiện `check` phase `autofix`
- [x] Xong
- **Service**: tài liệu
- **File**: `docs/contracts/authoring-llm-code-v2.md:62`.
- **Thay đổi**: dòng `check`: `phase ("chunk"|"final"|"autofix")`; thêm câu: "`autofix`: những gì llm-service tự sửa không gọi model; mỗi diagnostic `kind: "autofix"`, `rule: "palette_name"|"tsc_suggestion"`, `message: "<trước> → <sau>"`, `line` là dòng trong code của shot." Không đổi code authoring-service (ghi nguyên `phase`/`kind`).
- **Test**: —
- **Xong khi**: hợp đồng khớp với T4/T5 (FR-4).
- **Phụ thuộc**: T4, T5

## Sau cùng

- `cd services/llm-service && pytest && ruff check app tests` (lỗi ruff có sẵn trên main thì báo riêng, không tính).
- `cd services/authoring-service && go test ./... && go vet ./... && gofmt -l .`.
- Rebuild: `scripts/worktree.sh rebuild llm-service authoring-service`, xác nhận healthy.
- Kiểm trực tiếp:
  - DB: `select position('PALETTE_' in template_text) from prompts where role='remotion_engineer_ai' and is_active` → 0.
  - `docker exec llm-service env | grep CODE_MAX_REASONING_CHARS` hoặc log khởi động → 100000 (nếu `.env` của máy đặt giá trị khác thì báo).
  - Creator bấm chạy lại 5 đoạn hỏng của project 47745562 (tốn token, không tự chạy): xem `llm_usage` (`error_kind='budget'`) và `code_check_diagnostics` (`kind='autofix'`).
