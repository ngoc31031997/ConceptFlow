# Agentic Software Engineering trong ConceptFlow: tổng quan và cách dùng

> Cập nhật: 2026-09-28 (hết Phase 7). Người đọc: Creator và bất kỳ ai (người hoặc agent) làm Change Request (CR) trong repo này.
> Spec gốc: [`docs/AGENTIC_SOFTWARE_ENGINEERING_IMPLEMENTATION.md`](../AGENTIC_SOFTWARE_ENGINEERING_IMPLEMENTATION.md). Quyết định D1–D10: [`implementation-audit.md`](implementation-audit.md).

## 1. Ý tưởng chính

AI-DLC vẫn là quy trình (Inception → Design → Code → Build & Test). Lớp agentic **không thay** AI-DLC. Nó thêm các cơ chế **tự động và bắt buộc**, để agent làm được nhiều việc hơn mà repo vẫn tự giữ chất lượng:

```
Creator ──yêu cầu──► Agent (session chính = "software engineer")
                         │
      ┌──────────────────┼────────────────────────────────────────────┐
      │ Guardrails       │ permissions (allow/ask/deny) + hook chặn secret, chặn tự sửa cổng  │
      │ Verification     │ make check (hook Stop) · lint từng file (hook PostToolUse) · CI    │
      │ Independent      │ agent reviewer · security-reviewer · tester (chỉ đọc)             │
      │ review           │ solution-architect cho bước Design                                 │
      │ Merge gate (D1)  │ chỉ merge/push main khi tree đã qua make check + 3 review PASS     │
      └──────────────────┴────────────────────────────────────────────┘
                         │
                     main ──push──► GitHub CI (make build + make check-all)
```

Nguyên tắc: **luật quan trọng được enforce bằng script/hook**, không dựa vào việc agent "nhớ". Agent viết code **không tự chấm bài của chính mình**.

## 2. Những gì đã tích hợp

| Phase | Thành phần | File chính | Tài liệu chi tiết |
|---|---|---|---|
| 1 | Audit hiện trạng, quyết định D1–D10 | `docs/agentic/implementation-audit.md` | same |
| 2 | Lệnh kiểm tra chung cho người, hook và CI | `Makefile`, `scripts/{setup,build,check}.sh` | [`verification.md`](verification.md) |
| 3 | CI trên GitHub: mọi push chạy `make build` + `make check-all` | `.github/workflows/ci.yml` | [`branch-protection.md`](branch-protection.md) |
| 4 | Quyền của agent: allow / ask / deny | `.claude/settings.json` | [`autonomy-policy.md`](autonomy-policy.md) |
| 5 | Hooks: chặn secret, cổng merge, lint file vừa sửa, `make check` khi kết thúc lượt | `scripts/hooks/*` | [`hooks.md`](hooks.md) |
| 6 | Skills theo bước quy trình: `/cr-start`, `/cr-check`, `/rebuild`, `/cr-review`, `/cr-finish` | `.claude/skills/*/SKILL.md`, `scripts/rebuild.sh` | [`skills.md`](skills.md) |
| 7 | Agent theo vai trò (chỉ đọc) + review bắt buộc trong cổng merge | `.claude/agents/*.md`, `scripts/review-prep.sh`, `scripts/review-status.sh`, `scripts/hooks/record_review.py` | [`hooks.md`](hooks.md), [`skills.md`](skills.md) |
| D12 | Đồ thị tri thức của code (graphify, cục bộ, không LLM): agent định hướng bằng `graphify query/affected`; brief review có mục "Graph impact" | `scripts/graph.sh`, `scripts/hooks/graph_impact.py`, `make graph`, `make graph-hooks` | [`graphify.md`](graphify.md) |

### 2.1 Skills (lệnh gõ `/…`), đặt tên theo **bước**

| Lệnh | Khi nào dùng | Làm gì |
|---|---|---|
| `/cr-start <slug>` | Có yêu cầu mới | Kiểm tra working tree sạch → cập nhật `main` → lấy số CR kế tiếp → tạo `feature/cr-NNN-slug` → mở Requirements Analysis của AI-DLC → **dừng chờ duyệt** |
| `/cr-check [all]` | Trước khi báo "xong" | `make check` (hoặc `check-all`), trả bảng lỗi kèm nguyên nhân và bước tiếp |
| `/rebuild [svc…]` | Sau khi sửa code service | Build + restart đúng service đã đổi, chờ `healthy`, in log nếu fail |
| `/cr-review [NNN]` | Trước khi merge | Chạy song song 3 agent review trên tree đã commit, PASS chỉ khi cả 3 PASS |
| `/fix-bug <mô tả lỗi>` | Bạn thấy một chỗ **đang chạy sai** (lỗi, kết quả sai, giao diện vỡ) | Nhánh `fix/<slug>` → ghi báo lỗi vào `audit.md` → **viết test tái hiện lỗi trước** → tìm nguyên nhân gốc (`file:dòng`) → sửa tối thiểu → `/cr-check` → `/rebuild` → kiểm tra trực tiếp → báo cáo, **chờ duyệt**. Nếu "lỗi" thực ra là đổi yêu cầu, đổi contract/schema hay nhiều service thì chuyển sang `/cr-start` |
| `/cr-finish` | Sau khi Creator **duyệt** bước cuối | Kiểm tra diff + hồ sơ AI-DLC → merge `origin/main` → check → review → merge vào `main` qua cổng → push → theo dõi CI |

### 2.2 Agents, đặt tên theo **vai trò**

Chỉ có Read/Grep/Glob: không chạy lệnh, không sửa file.

| Agent | Vai trò | Trong cổng merge? |
|---|---|---|
| `solution-architect` | Review **thiết kế** trước khi code: ADR, ranh giới service, contract, failure mode; chỉ ra ADR cần viết | Không (dùng ở bước Design) |
| `reviewer` | Bug logic, edge case, regression, vi phạm ADR, idempotency/transaction/concurrency, thiếu test, code giả | **Có** |
| `security-reviewer` | Secret, injection, auth, lộ dữ liệu nhạy cảm, dependency, leo quyền | **Có** |
| `tester` | QC: đối chiếu từng tiêu chí chấp nhận của CR ↔ code ↔ test | **Có** |

Không có agent "software engineer" riêng: session chính là người viết code.

### 2.3 Hooks: chạy tự động, không cần gọi

| Khi | Hook | Tác dụng |
|---|---|---|
| Trước mỗi lệnh Bash | `guard_bash.py` | Chặn lệnh nhắc tới `.env`, `secrets/`, `client_secret_*.json`, `docker compose config`; chặn đụng thư mục marker; **cổng merge** |
| Sau mỗi lần sửa file | `lint-edited.sh` | Lint riêng file đó (`gofmt` / `ruff` / `eslint`) |
| Khi agent muốn kết thúc lượt | `stop-check.sh` | Chạy `make check`; fail thì bắt agent sửa (chặn 1 lần; có cache nên lượt không đổi gì gần như tức thì) |
| Khi một agent review kết thúc | `record_review.py` | Kiểm tra rồi ghi verdict làm marker cho cổng merge |

### 2.4 Cổng merge vào `main` (D1: vẫn tự merge, nhưng có cổng)

Merge hoặc push vào `main` chỉ được khi **tree** của nhánh có đủ 4 dấu:

| Dấu | Ai ghi | Điều kiện |
|---|---|---|
| `make check` | `scripts/check.sh` | Pass trên working tree sạch, nhánh khác `main` |
| `review: reviewer` | hook `record_review.py` | Agent trả `VERDICT: PASS` cho đúng tree HEAD; lời dặn (brief) nhận được **trùng từng chữ** với brief do `review-prep.sh` tạo; file diff trùng diff thật |
| `review: security-reviewer` | như trên | như trên |
| `review: tester` | như trên | như trên |

Thêm hai điều kiện:
- Nhánh phải **chứa `main` mới nhất**, để kết quả merge đúng là tree đã được kiểm tra.
- Push `main` phải là **lệnh riêng**, sau lệnh merge.

Session viết code **không tự ghi được** marker. Cụ thể:
- thư mục marker và transcript của agent bị chặn cả ở shell lẫn Edit/Write;
- không chạy tay được hook `record_review`;
- hook chỉ tin transcript nằm đúng chỗ Claude Code ghi cho agent con của session.

Lần review thật đầu tiên đã phát hiện một lỗ hổng giả mạo và nó đã được vá (xem [`hooks.md`](hooks.md)).

### 2.5 Quyền của agent (tóm tắt)

- **Tự làm:** đọc/sửa code, `make …`, tạo nhánh, commit, merge/push (qua cổng).
- **Phải hỏi Creator:**
  - thao tác huỷ thay đổi chưa commit (`git restore`, `reset --hard`, `clean`, `branch -D`);
  - `docker exec`, script migrate dữ liệu;
  - sửa `.github/`, `.claude/settings*.json`, `.claude/agents/`, `.claude/skills/`, `scripts/hooks/`, `scripts/check.sh`, `scripts/review-prep.sh`, `scripts/review-status.sh`.
- **Cấm:**
  - force push, xoá nhánh remote;
  - `rm -rf`, xoá volume Docker;
  - đọc/sửa `.env`, `secrets/`;
  - sửa thư mục marker.

### 2.6 Model và effort (tiết kiệm token)

Mỗi agent và skill có thể chọn model (`sonnet`, `opus`, `haiku`, `fable`, `inherit`) và mức suy nghĩ `effort` (`low`, `medium`, `high`, `xhigh`, `max`) trong phần frontmatter đầu file.

| Thành phần | model | effort | Lý do |
|---|---|---|---|
| agent `solution-architect` | `opus` | `high` | Ít dùng, nhưng quyết định thiết kế đắt nếu sai |
| agent `reviewer` | `sonnet` | `high` | Chạy mỗi lần merge; cần suy luận kỹ nhưng không cần model lớn nhất |
| agent `security-reviewer` | `sonnet` | `high` | Như trên |
| agent `tester` | `sonnet` | `medium` | Đối chiếu tiêu chí ↔ code ↔ test, ít suy luận sâu hơn |
| skill `/cr-start` | theo phiên | `medium` | Phân tích yêu cầu |
| skill `/fix-bug` | theo phiên | `medium` | Tìm nguyên nhân gốc cần model của phiên; effort vừa phải |
| skill `/cr-finish` | theo phiên | `low` | Các bước cơ học (git, check); review do agent làm |
| skill `/cr-check`, `/rebuild`, `/cr-review` | theo phiên | theo phiên | **Cố ý không đặt**: model/effort của skill áp dụng **cho hết lượt**, nên nếu `/rebuild` đặt `haiku` thì phần sửa code sau đó trong cùng lượt cũng chạy bằng `haiku` |

Cách đổi:
- Sửa dòng `model:` / `effort:` trong `.claude/agents/<tên>.md` hoặc `.claude/skills/<tên>/SKILL.md`. Claude Code sẽ hỏi bạn trước khi sửa, vì đây là file cấu hình cổng.
- Mở session mới để chắc chắn bản mới được nạp.
- Test `SettingsTest.test_model_and_effort_values_are_valid` bắt lỗi gõ sai, vì Claude Code lặng lẽ bỏ qua giá trị sai.
- Muốn tiết kiệm hơn nữa: đổi `reviewer`/`security-reviewer` sang `effort: medium`, hoặc `tester` sang `haiku`. Đánh đổi là review kém kỹ hơn.

Ghi chú: trong chế độ quyền *auto*, model nào auto mode không hỗ trợ sẽ bị bỏ qua và phiên giữ model hiện tại.

## 3. Cách dùng hằng ngày: một CR từ đầu đến cuối

Creator chỉ cần nói bằng lời (ví dụ "làm CR: …"). Agent sẽ tự dùng skill. Tên lệnh ghi ở dưới để bạn biết đang ở bước nào, hoặc gõ trực tiếp.

| # | Creator | Agent |
|---|---|---|
| 1 | Nêu yêu cầu | `/cr-start <slug>`: nhánh `feature/cr-NNN-slug`, ghi yêu cầu vào `audit.md`, làm Requirements Analysis, **chờ duyệt** |
| 2 | Duyệt requirement | Design theo AI-DLC. CR có thay đổi kiến trúc thì gọi agent `solution-architect` review thiết kế, **chờ duyệt** |
| 3 | Duyệt design | Viết code + test. Mỗi lần sửa file có lint tự động. Xong thì `/rebuild` service bị ảnh hưởng. Hook Stop tự chạy `make check` |
| 4 | Kiểm tra trên app (stack Docker đã rebuild) | Báo kết quả, **chờ duyệt** |
| 5 | "ok" | `/cr-finish`: commit → merge `origin/main` → `/cr-check` → `/cr-review` (3 agent) → merge vào `main` qua cổng → push → báo kết quả CI |

Nếu `/cr-review` ra **FAIL**:
- Agent sửa các lỗi trong phạm vi CR, commit, rồi chạy lại check và review. Mỗi commit mới là một tree mới, nên dấu cũ không còn giá trị.
- FAIL **dính cứng với tree đó**: chạy lại agent trên cùng tree sẽ không ghi PASS nữa (hook chặn).
- Nếu bạn cho rằng review sai, bạn có thể tự gỡ bằng cách xoá file `.fail` trong terminal. Agent không làm được việc này.

### Xem nhanh trạng thái cổng

```bash
scripts/review-status.sh     # 4 dấu của tree HEAD + log hook review; exit 0 = cổng sẽ cho qua
make check                   # kiểm tra nhanh các service đã đổi
make check-all               # toàn bộ, giống CI
```

### Khi bị chặn: thông báo nghĩa là gì

| Thông báo (tóm tắt) | Nghĩa | Làm gì |
|---|---|---|
| `Blocked: the command references .env…` | Lệnh Bash nhắc tới file secret. Có thể là chặn nhầm, ví dụ heredoc viết tài liệu có chữ `.env` | Dùng `.env.example`; viết nội dung bằng Edit/Write hoặc file tạm |
| `Blocked by merge gate (D1): … lacks a pass of: …` | Tree chưa đủ dấu | Về nhánh, commit hết, `/cr-check`, `/cr-review`, rồi merge lại |
| `… does not contain the current main` | Nhánh cũ hơn `main` | Trên nhánh: `git merge origin/main`, check + review lại |
| `push main as a separate command…` | Gộp merge và push trong một dòng lệnh | Chạy `git push origin main` riêng |
| `Stop blocked: make check failed` | Còn lỗi lint/test | Agent sửa. Nếu lỗi không liên quan CR, agent báo kèm bằng chứng |
| `review-status.sh` thiếu dấu review | Hook không ghi dấu | Xem dòng `NOT RECORDED …` trong log: sai tree, brief bị sửa, diff lệch, hoặc thiếu dòng VERDICT |

### Sửa một bug bạn tìm thấy

1. Gõ `/fix-bug` kèm mô tả: lỗi gì, ở màn hình/API nào, làm sao để gặp lại; ảnh chụp hoặc log càng tốt. Nói thường ("sửa lỗi …") agent cũng tự dùng skill này.
2. Agent tạo nhánh `fix/<slug>`, **viết test tái hiện lỗi**. Không tái hiện được thì agent dừng và hỏi thêm thông tin, không sửa theo phỏng đoán.
3. Agent nêu nguyên nhân gốc rồi sửa, chạy `make check`, rebuild service, kiểm tra trực tiếp, báo cáo kèm cách bạn tự kiểm tra.
4. Bạn trả lời "ok" → `/cr-finish` merge qua cổng như một CR.

## 4. Việc Creator cần tự làm

1. **Bật ruleset `protect-main` trên GitHub** (chặn force push và xoá `main`): làm theo [`branch-protection.md`](branch-protection.md). **Chưa làm.**
2. Duyệt từng bước AI-DLC ("ok"/"approve"). Agent không commit khi chưa được duyệt.
3. Mọi thay đổi cấu hình cổng (settings, agents, hooks) sẽ hiện hộp thoại hỏi bạn. Hãy đọc kỹ trước khi đồng ý.

## 5. Giới hạn đã biết

- **Cổng ở máy là "khoá chống sơ suất", không phải két sắt** (quyết định D11). Hook chạy chung máy, chung quyền với agent, nên một agent *cố tình* gian lận vẫn có đường vòng. Hai lỗ đã biết (H1: tin nhắn thêm cho agent review; H2: diff so với local `main`) và phương án triệt để (H3: cổng trên GitHub) nằm trong backlog, xem `implementation-audit.md` §9.

- **Hook khớp theo chữ trong lệnh.** `sh -c "$VAR"`, `grep -r` quét qua `.env`, hoặc một lệnh Bash tự sửa script hook sẽ không bị bắt. Lưới cuối cùng là review (thay đổi ở `scripts/hooks/`, `.claude/` nằm trong diff được review) và `git log`.
- **Hook chỉ áp dụng cho Claude Code.** Người gõ git trong terminal không đi qua cổng; đó là quyền của Creator.
- **CI chạy sau khi push**, nên nó chỉ phát hiện lỗi chứ không chặn trước. Chặn trước là việc của cổng ở máy local.
- **Nhiều agent dùng chung một thư mục checkout là rủi ro thật.** Ngày 2026-09-28 một agent khác đổi nhánh giữa chừng làm commit rơi nhầm vào `main` (không bị push). Cách sửa gốc là Phase 8. Trong lúc chờ, agent kiểm tra `git branch --show-current` trước khi commit.
- **Agent/skill có thể bị cache**: agent tự tạo được nạp khi session bắt đầu (có lúc giữa session), và skill có thể chạy bản cũ đã cache. Sửa `.claude/agents/` hoặc `.claude/skills/` xong nên mở session mới.
- **`ask` trong chế độ auto**: trong session chạy chế độ quyền *auto*, các lần sửa file cấu hình cổng (thuộc diện `ask`) đã chạy mà không thấy hộp thoại hỏi. Ở chế độ mặc định thì sẽ hỏi Creator. Muốn chắc chắn thì không dùng auto mode khi agent sửa `.claude/` hoặc `scripts/hooks/`.
- **CI bỏ qua một số test** (manim, mockup răng, 1 test CR-048), xem D9/D10.

## 6. Còn lại trong lộ trình

| Việc | Nội dung | Ghi chú |
|---|---|---|
| D6 | Đổi `CLAUDE.MD` → `CLAUDE.md`, rút gọn, nạp rule AI-DLC theo giai đoạn, ghi luồng merge có cổng | Đã duyệt, chưa làm |
| Phase 8 | Worktree riêng cho mỗi agent; tham số hoá `container_name`/cổng trong `docker-compose.yml` | Nên làm sớm vì sự cố ở §5 |
| Phase 9 | Background agent | Cần Phase 8 + D6; phải mở PR, không dùng tự merge |
| Phase 10 | Evals + observability (log mỗi CR, số liệu hằng tháng) | |
| Phase 11 | Rà lại chính sách tự chủ theo rủi ro | `autonomy-policy.md` đã có, cập nhật theo thực tế |
| Backlog hook (H1–H10) | Gia cố cổng ở máy (tin nhắn thêm cho agent review, diff so với `origin/main`, …) và **cổng phía GitHub** (PR + CI + review bằng GitHub Action) | Quyết định D11: tạm dừng, xem `implementation-audit.md` §9 |

## 7. Bản đồ file

```
Makefile                         make setup | build | check | check-all | graph | graph-hooks
scripts/
  setup.sh build.sh check.sh     logic kiểm tra (check.sh ghi dấu make check)
  rebuild.sh                     /rebuild
  graph.sh                       make graph / graph-hooks (graphify, xem graphify.md)
  review-prep.sh                 tạo diff + brief cho agent review
  review-status.sh               xem 4 dấu của tree HEAD
  hooks/
    guard_bash.py                PreToolUse Bash: secret, thư mục marker, cổng merge
    lint-edited.sh               PostToolUse: lint file vừa sửa
    stop-check.sh                Stop: make check
    record_review.py             SubagentStop: ghi verdict review
    test_hooks.py                42 test (chạy trong make check / CI)
    graph_impact.py              mục "Graph impact" của brief review (+ test_graph_impact.py)
.claude/
  settings.json                  quyền + đăng ký hook (commit, dùng chung)
  settings.local.json            quyền cá nhân (không commit)
  skills/<tên>/SKILL.md          cr-start, fix-bug, cr-check, rebuild, cr-review, cr-finish
  agents/<vai trò>.md            solution-architect, reviewer, security-reviewer, tester
.github/workflows/ci.yml         CI
docs/agentic/                    tài liệu này + chi tiết từng phần
```
