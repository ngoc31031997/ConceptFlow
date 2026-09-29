# Implementation Audit — Agentic Software Engineering (Phase 1)

> Ngày audit: 2026-09-28 · Nhánh: `main` @ `204f9d8` (đồng bộ `origin/main`)
> Spec: `docs/AGENTIC_SOFTWARE_ENGINEERING_IMPLEMENTATION.md` (hiện chưa được commit)
> Phạm vi: chỉ đọc repo và chạy test/lint sẵn có. **Không sửa file nào ngoài file này.**

## 0. Quyết định của Creator (đã chốt trước audit)

| # | Câu hỏi | Quyết định | Hệ quả so với spec |
|---|---|---|---|
| D1 | Chính sách merge | **Giữ tự merge vào `main`**, nhưng bắt buộc `make check` + review phải pass trước khi merge | Lệch spec Phase C (spec yêu cầu PR). Phải enforce cổng merge **ở máy local** (hook), không dựa vào PR. |
| D2 | GitHub remote | Đã có: `origin = https://github.com/ngoc31031997/ConceptFlow.git` (repo public) | Phase B (CI) làm được. |
| D3 | Phạm vi Stop hook | Chỉ lint + unit test | `make check` = lint + unit; integration/contract/build chuyển sang `make check-all` và CI. |

---

## 1. Năng lực đã có (Existing capabilities)

### 1.1 AI-DLC
| Thành phần | Vị trí | Ghi chú |
|---|---|---|
| Rule AI-DLC | `.ai-dlc/steering/`, `.ai-dlc/aws-aidlc-rule-details/` (inception, construction, operations, extensions/security, extensions/testing, common) | Tổng ~300 KB markdown |
| State | `aidlc-docs/aidlc-state.md` (132 dòng, bảng CR tới CR-046) | Nguồn sự thật cho trạng thái CR |
| Audit log | `aidlc-docs/audit.md` (1 993 dòng) | |
| ADR | `aidlc-docs/decisions/` — 29 ADR + README | Reviewer agent phải đối chiếu |
| Tài liệu construction | `aidlc-docs/construction/<service>/`, `plans/`, `build-and-test/` | Có `ci-cd-integration-instructions.md` (bản nháp CI, xem §3.4) |
| CR requirements | `aidlc-docs/inception/requirements/cr-*.md` | QC agent dùng làm acceptance criteria |
| Quy tắc branch | `.ai-dlc/aws-aidlc-rule-details/construction/git-branching.md` | `feature/cr-<NNN>-<slug>` |

Spec giả định `aidlc-docs/`, `audit.md`, `aidlc-state.md` — **khớp với repo**, không cần map lại.

### 1.2 Service và bộ công cụ test/lint

| Service | Stack | Lint | Unit test | Ghi chú |
|---|---|---|---|---|
| `authoring-service` | Go | `go vet`, `gofmt` | `go test ./...` | |
| `orchestrator` | Go | `go vet`, `gofmt` | `go test ./...` | |
| `llm-service` | Python 3.12 | ruff (có trong venv) | pytest | |
| `publisher` | Python 3.12 | ruff (có trong venv) | pytest | |
| `rendering` | Python 3.12 | ruff khai báo trong `requirements-dev.txt` nhưng **chưa cài trong `.venv`** | pytest | Manim được mock trong test |
| `tts` | Python 3.12 | ruff **chưa cài trong `.venv`** | pytest | |
| `video-assembly` | Python 3.12 | ruff (có trong venv) | pytest | |
| `api-gateway` | Node 20 | `eslint .` | jest | |
| `web-gui` | Node 20 + TS | `eslint .`, `tsc -b` (qua `npm run build`) | vitest | |
| (repo) | Python | — | `tests/contracts/test_shared_artifacts_contract.py` | Contract test duy nhất ở root |

Mỗi service Python có `.venv` riêng (Python 3.12.14). Python hệ thống là 3.9.6, **không dùng được** cho test.

### 1.3 Hạ tầng khác
- `docker-compose.yml`: 22 service (app + Postgres riêng mỗi service + RabbitMQ + Grafana/Promtail), có `mem_limit`/`deploy`.
- `infra/observability/`: Grafana + Promtail cho **ứng dụng** (không phải cho agent run).
- `.claude/settings.json` (được commit): allow list cho git.
- `.claude/settings.local.json` (máy local): allow `go vet/test/build`, `docker compose *`, `docker exec *`, `docker restart *`, vài lệnh `curl` một lần, `git commit -m`.
- `scripts/migrate-authoring-data.sh`: script migrate dữ liệu một lần (CR-040), không liên quan verification.
- Git: 86 nhánh local, trong đó 82 đã merge vào `main`. Worktree duy nhất là repo chính; `.claude/worktrees/` rỗng.

### 1.4 Baseline verification (chạy thật ngày 2026-09-28)

| Service | Lint | Unit test | Thời gian |
|---|---|---|---|
| authoring-service | vet ✅ · gofmt ❌ 3 file | ✅ | ~2 s |
| orchestrator | vet ✅ · gofmt ❌ 3 file | ✅ (cached) | ~1 s |
| llm-service | ruff ✅ | ✅ 91 passed | ~6 s |
| publisher | ruff ✅ | ✅ 88 passed | ~1 s |
| rendering | ruff **chưa chạy** (chưa cài) | ✅ 313 passed, 5 skipped | ~8 s |
| tts | ruff **chưa chạy** (chưa cài) | ✅ 104 passed | ~1 s |
| video-assembly | ruff ❌ 16 lỗi (7 tự sửa được) | ✅ 186 passed | ~1 s |
| api-gateway | eslint ✅ | ✅ | ~3 s |
| web-gui | eslint ❌ 9 errors, 18 warnings | ❌ **3 failed** / 380 passed (`tests/pages/VideoListPage.test.tsx` — hàng video không biến mất sau khi xoá) | ~8 s |

Chưa chạy: contract test ở root, `tsc -b`/build của web-gui, integration test qua Docker.

File gofmt lỗi:
- authoring-service: `internal/adapters/http/router_test.go`, `internal/application/project_illustrations_test.go`, `internal/domain/qc_report.go`
- orchestrator: `internal/adapters/postgres/qc_reports.go`, `internal/application/retry_step.go`, `internal/domain/qc_report.go`

**Kết luận:** lint + unit test của toàn repo chạy dưới ~40 giây. Stop hook chỉ gồm lint + unit (D3) vì vậy khả thi về thời gian. Nhưng **`main` hiện đang đỏ**. Đây là bằng chứng trực tiếp rằng luồng tự merge hiện tại không có cổng kiểm tra.

---

## 2. Năng lực còn thiếu (Missing capabilities)

| Spec phase | Hạng mục | Trạng thái |
|---|---|---|
| A | `Makefile`, `make check`, `make check-all` | **Chưa có** (không có Makefile nào trong repo) |
| A | Phát hiện service bị thay đổi | Chưa có |
| B | `.github/workflows/ci.yml` | **Chưa có** (chỉ có bản nháp trong tài liệu, §3.4) |
| C | Branch protection / ruleset trên GitHub | Chưa rõ, chưa có tài liệu. `gh` CLI **chưa cài** nên chưa kiểm tra được từ máy |
| D | Deny rules trong `.claude/settings.json` | **Không có** rule deny nào |
| E | Hooks (PostToolUse lint, Stop verification) | **Không có** hook nào (cả project lẫn user settings) |
| E | Cổng merge (enforce D1) | Chưa có |
| F | Skills `/cr-start`, `/rebuild`, `/cr-check`, `/cr-review`, `/cr-finish` | Chưa có (`.claude/skills/`, `.claude/commands/` không tồn tại) |
| G | `.claude/agents/` (reviewer, security-reviewer, qc) | Chưa có |
| H | Script worktree, port/`COMPOSE_PROJECT_NAME` theo worktree | Chưa có |
| I | Background agent workflow | Chưa có |
| J | `evals/` | Chưa có |
| K | Log agent run có cấu trúc | Chưa có |
| L | `docs/agentic/autonomy-policy.md` | Chưa có |
| — | ruff trong `.venv` của rendering, tts | Thiếu (môi trường local, không phải file repo) |

---

## 3. Cấu hình xung đột (Conflicting configurations)

### 3.1 Chính sách merge: CLAUDE.MD ↔ spec Phase C
`CLAUDE.MD` (mục *CR completion policy*) cho agent tự merge vào `main` và tự push. Spec cấm luồng này. Theo **D1**, giữ tự merge nhưng thêm cổng. Hệ quả:
- **Không bật** rule "Require a pull request" trên `main`, vì nó sẽ chặn chính luồng tự merge.
- Cổng phải nằm ở máy local: hook chặn `git merge` vào `main` và `git push origin main` nếu `make check` hoặc review chưa pass trên đúng commit đó.
- CI trên GitHub chạy **sau khi push** (trên nhánh feature và `main`), nên chỉ phát hiện lỗi chứ không chặn trước merge. Cần ghi rõ giới hạn này trong tài liệu.

### 3.2 CLAUDE.MD bắt đọc toàn bộ rule AI-DLC ở mọi task ↔ spec §18
`CLAUDE.MD` yêu cầu đọc *mọi* markdown trong `.ai-dlc/aws-aidlc-rule-details/`, khoảng 300 KB (≈ 75 000 token) mỗi task. Spec §18 yêu cầu `CLAUDE.md` ngắn và chỉ nạp ngữ cảnh khi cần. Đề xuất: chuyển sang nạp theo giai đoạn qua skill. Việc này **thay đổi quy trình AI-DLC**, nên cần Creator duyệt.

### 3.3 Tên file `CLAUDE.MD` (chữ hoa)
File được commit với tên `CLAUDE.MD` và `core.ignorecase=true`. Trên macOS (file system không phân biệt hoa thường) file vẫn được nạp. Trên Linux (CI runner, background agent ở Phase I) Claude Code tìm `CLAUDE.md` và **sẽ không thấy** file này. Cần `git mv CLAUDE.MD CLAUDE.md` trước Phase 9.

### 3.4 Bản nháp CI đã lỗi thời
`aidlc-docs/construction/build-and-test/ci-cd-integration-instructions.md` đề xuất một `ci.yml`, nhưng:
- liệt kê `content-plugin` và `script-processing`: không còn là service đang chạy;
- **thiếu** `authoring-service` và `llm-service`;
- chỉ có job Go cho `orchestrator`, pin `go-version: '1.22'` (máy local đang dùng Go 1.27);
- có SonarQube, OWASP và bước deploy, vượt phạm vi Phase B.

→ Phase 3 nên **cập nhật tài liệu này** và dẫn chiếu tới `ci.yml` thật, không tạo tài liệu CI thứ hai.

### 3.5 `services/content-plugin/` là thư mục chết
Git không theo dõi file nào trong thư mục này (chỉ còn `__pycache__`/`.pytest_cache`), và nó không có trong `docker-compose.yml`. Tuy vậy `README.md` vẫn hướng dẫn chạy `pytest` cho nó. Cơ chế phát hiện service trong `make check` phải bỏ qua thư mục này. README cần sửa riêng.

### 3.6 Docker rebuild policy ↔ worktree song song
`CLAUDE.MD` yêu cầu rebuild và restart container sau mỗi thay đổi code. `docker-compose.yml` hard-code `container_name` (rabbitmq, tts-db, tts, rendering-db, …) và cổng host (`15672`, `8080`, `3000`, `3001`). `container_name` **ghi đè** `COMPOSE_PROJECT_NAME`, nên chỉ đặt `COMPOSE_PROJECT_NAME` riêng cho từng worktree **không đủ**. Hai worktree cùng rebuild sẽ tranh nhau một container. Đây là trở ngại chính của Phase 8.

---

## 4. Cấu hình rủi ro (Risky configurations)

| # | Mức | Cấu hình | Rủi ro | Đề xuất (phase) |
|---|---|---|---|---|
| R1 | **Cao** | `.claude/settings.json` (được commit): `Bash(git push:*)` | Cho phép luôn `git push --force`, `git push -f origin main`, `git push origin --delete …` mà không hỏi | Thay bằng allow cụ thể + deny `--force`/`-f`/`--delete`/`--mirror` (Phase 4) |
| R2 | **Cao** | Không có deny rule nào; `.env` chứa secret thật (`HIVE_API_KEY`, `AZURE_SPEECH_KEY`, `GOOGLE_OAUTH_CLIENT_SECRET`, mật khẩu Postgres/RabbitMQ/Grafana); `secrets/` chứa file `client_secret_*.json` | Agent đọc được secret, và nội dung có thể lọt vào log, commit hoặc transcript. Origin là repo **public** | Deny `Read`/`Edit` cho `.env`, `.env.window`, `secrets/**`. Thêm hook PreToolUse chặn Bash đọc các file này, vì deny của `Read` không chặn `cat .env` (Phase 4–5) |
| R3 | **Trung bình–Cao** | `settings.local.json`: `Bash(docker compose *)`, `Bash(docker exec *)` | `docker compose down -v` xoá volume Postgres (mất dữ liệu). `docker exec … env` in secret của container | Deny `docker compose down -v*`, `docker volume rm*`. Chuyển `docker exec` sang dạng hỏi (Phase 4) |
| R4 | Trung bình | `Bash(git checkout:*)`, `Bash(git merge:*)` được allow tự do | `git checkout -- .` hoặc `git checkout .` huỷ thay đổi chưa commit. Merge vào `main` không có cổng | Deny dạng huỷ thay đổi. Cổng merge bằng hook (Phase 4–5) |
| R5 | Trung bình | Tự merge + push `main` không có kiểm tra (baseline đỏ, §1.4) | Lỗi đi thẳng vào `main` | `make check` + cổng merge (Phase 2, 5) |
| R6 | Thấp–Trung bình | Chưa xác nhận branch protection trên GitHub | Có thể force push hoặc xoá `main` từ bất kỳ đâu | Bật ruleset: chặn force push và chặn xoá `main` (Phase 3, làm tay) |
| R7 | Thông tin | ADR-0016: OAuth credential lưu plaintext | Quyết định đã có, ngoài phạm vi. Security reviewer cần biết để không báo sai | Ghi vào context của security-reviewer (Phase 7) |
| R8 | Thấp | `settings.local.json` có các lệnh `curl` một lần với payload cố định | Chỉ gây nhiễu, không nguy hiểm | Dọn khi làm Phase 4 |
| R9 | Thấp | 82 nhánh local đã merge nhưng chưa xoá | Gây nhiễu cho `/cr-start`, dễ nhầm nhánh | Không tự xoá. Có thể thêm lệnh dọn có xác nhận |

Đã kiểm tra: `.env`, `.env.window`, `secrets/*.json` và `client_secret_*.json` đều nằm trong `.gitignore`, và **không có secret nào đang được git theo dõi**. Các file được theo dõi có tên chứa "secret" hoặc "credential" đều là code hoặc tài liệu (`.env.example`, `secrets/README.md`, `credential_store.py`, ADR-0016, `web-gui/.env.test` chỉ chứa `VITE_API_BASE_URL`).

---

## 5. Thứ tự triển khai đề xuất

Bám theo §20 của spec, điều chỉnh theo D1–D3 và các phát hiện ở trên.

| Bước | Nội dung | Phụ thuộc / điều kiện |
|---|---|---|
| **2a** | Đưa baseline về xanh: sửa 3 test `VideoListPage`, 9 lỗi eslint web-gui, 16 lỗi ruff video-assembly, 6 file gofmt; cài ruff vào venv rendering/tts rồi chạy lint | **Cần Creator duyệt** (đây là sửa code ứng dụng, làm trên nhánh `chore/…` riêng). Nếu không làm, `make check-all` sẽ đỏ ngay từ đầu |
| **2b** | `Makefile`: `make check` (lint + unit, chỉ service thay đổi so với `main` hoặc so với working tree), `make check-all` (toàn bộ lint + unit + contract test + `tsc -b` web-gui). Tái dùng lệnh sẵn có của từng service, không viết lại logic test | Bỏ qua `content-plugin`. Python chạy bằng `.venv` local, trên CI dùng `python` của runner |
| **3** | `.github/workflows/ci.yml` chạy `make check-all` trên push (mọi nhánh) và PR. Cập nhật `ci-cd-integration-instructions.md`. Viết hướng dẫn ruleset GitHub: **chặn force push + chặn xoá `main`**, không bắt buộc PR (theo D1) | Creator bật ruleset bằng tay (hoặc cài `gh` CLI) |
| **4** | Permissions: sửa R1–R4, R8; deny secret; tách phần dùng chung (`settings.json`) và phần local | Kiểm tra cú pháp theo phiên bản Claude Code đang cài (bản extension VS Code; CLI `claude` không có trong PATH) |
| **5** | Hooks: PostToolUse lint file vừa sửa; Stop → `make check` (có chống lặp qua `stop_hook_active`); PreToolUse chặn Bash đọc secret; **cổng merge**: chặn `git merge` vào `main` hoặc `git push origin main` nếu thiếu marker "check + review pass" gắn với commit hiện tại | Đây là cơ chế enforce D1 |
| **6** | Skills: `/cr-start`, `/cr-check`, `/rebuild`, `/cr-review`, `/cr-finish`. `/cr-finish` = check → review → cập nhật `aidlc-state.md` → commit → merge theo `CLAUDE.MD`, và phải đi qua cổng merge | Tái dùng policy branch và rebuild sẵn có |
| **7** | `.claude/agents/`: reviewer, security-reviewer, qc, chỉ cho tool đọc. Đọc ADR và `inception/requirements/cr-*.md` | Tận dụng `/code-review`, `/security-review` có sẵn nếu phù hợp |
| **—** | Sửa `CLAUDE.MD`: đổi tên thành `CLAUDE.md`, rút gọn, nạp rule AI-DLC theo giai đoạn, ghi lại chính sách merge có cổng | **Cần Creator duyệt** (§3.2, §3.3). Nên làm cùng hoặc ngay sau bước 6 |
| **8** | Worktree isolation: bỏ `container_name` hoặc tham số hoá nó, tham số hoá cổng host theo worktree | Sửa `docker-compose.yml` ảnh hưởng tới cách chạy dev hằng ngày → **cần duyệt**. Nên hoãn tới khi thực sự chạy nhiều agent song song |
| **9** | Background agent | **Chưa khuyến nghị.** Cần đổi tên `CLAUDE.md` (§3.3) trước, cần API key trong GitHub secret, và background agent **không được** dùng luồng tự merge của D1 (phải mở PR) |
| **10** | Evals + observability: bản nhẹ, log JSONL mỗi CR qua Stop hook, metric tính tay theo tháng | Ít mẫu nên chưa đủ ý nghĩa thống kê |
| **11** | `docs/agentic/autonomy-policy.md` | Viết sớm được (cùng Phase 4) vì nó là tài liệu hoá deny/allow |

## 6. Quyết định bổ sung (Creator, 2026-09-28)

| # | Câu hỏi | Quyết định |
|---|---|---|
| D4 | Lỗi baseline | **Sửa trước** trên nhánh `chore/` riêng (bước 2a), rồi mới dựng `make check` |
| D5 | `make check` so với cái gì | **So với `main`** (mọi thay đổi của nhánh hiện tại + working tree) |
| D6 | Sửa `CLAUDE.MD` | **Đồng ý**: đổi tên thành `CLAUDE.md`, rút gọn, nạp rule AI-DLC theo giai đoạn |

## 7. Kết quả bước 2a — baseline xanh (nhánh `chore/baseline-green`)

| Hạng mục | Cách xử lý |
|---|---|
| gofmt (6 file Go) | `gofmt -w`, chỉ đổi khoảng trắng |
| ruff video-assembly (16), tts (3) | `ruff --fix` phần an toàn; `zip(x, x[1:])` thêm `strict=False` (cố ý lệch 1 phần tử, giữ nguyên hành vi); ngắt các dòng dài; dòng format ASS là chuỗi cố định nên dùng `noqa: E501` |
| ruff rendering (96, trước đây chưa từng chạy) | Tự sửa phần an toàn; UP038 (`isinstance(x, A \| B)`, tương đương trên py3.12); ngắt dòng dài trong code và test. Thêm `per-file-ignores` có ghi chú: `tests/fixtures/conceptflow_template.py` (F403/F405/E501, vì cố ý dùng `import *` giống template cho Creator) và `tools/*` (E501/E702, là script sinh asset offline) |
| web-gui 3 test `VideoListPage` | Test đã cũ sau commit `0f3364b` (đổi `window.confirm` sang `ConfirmModal`): test bấm nút trong modal thay vì mock `window.confirm`. Test "cancel" trước đây pass nhưng không kiểm tra gì, nay thực sự bấm Hủy |
| web-gui 9 lỗi eslint | `KeyboardShortcutsHelp`: overlay `role="presentation"` + chỉ đóng khi bấm đúng nền (theo mẫu `ConfirmModal`), card thêm `role="dialog"`; test: bỏ `onClick` thừa, thay `any` bằng kiểu cụ thể |
| ruff trong venv rendering/tts | Đã cài `ruff==0.7.*` theo `requirements-dev.txt` (chỉ thay đổi môi trường local) |

Kết quả sau khi sửa (chạy lại ngày 2026-09-28): mọi lint (vet, gofmt, ruff, eslint, `tsc -b`) đều pass. Unit test: Go ✅, llm-service 91, publisher 88, rendering 313 (+5 skip), tts 104, video-assembly 186, api-gateway 107, web-gui 383, contract test 2, tất cả pass.

## 8. Phase 3 (CI) — quyết định và backlog (Creator, 2026-09-28)

CI (`.github/workflows/ci.yml`) chạy `make setup` → `make build` → `make check-all` trên mọi push.

| # | Quyết định | Chi tiết |
|---|---|---|
| D7 | CI gồm build + lint + unit test | `make build` (compile từng service) + `make check-all` |
| D8 | SonarQube, OWASP Dependency-Check → **backlog** | Chưa cần token/NVD key; chưa tích hợp |
| D9 | Test manim và mockup răng **bỏ qua trên CI** | Marker `manim` (`tests/conceptflow/`, 98 test) và `tooth_mockup` (`test_layout_checker`, `test_illustration_kit`, `test_illustration_previewer`, 20 test) trong `services/rendering/tests/conftest.py`; CI lọc bằng `PYTEST_ADDOPTS`. Pass ở local, có thể bỏ trong tương lai |
| D10 | Test CR-048 T8/T9 "hiện cảnh báo của bước Visual… và đóng được" **bỏ qua trên CI** | `it.skipIf(process.env.CI)` trong `web-gui/tests/pages/ScriptOutlineStepPage.test.tsx`. Trên runner, khung cảnh báo không đóng sau khi bấm "Đóng thông báo"; không tái hiện được ở local (chạy song song, 2 worker + tải CPU, container Linux Node 20) |

Backlog:
- Tích hợp SonarQube/SonarCloud (`SONAR_TOKEN`) và OWASP Dependency-Check (`NVD_API_KEY`); quyết định chính sách fail.
- Điều tra D10 (cần xem trạng thái `dismissedAt`/`usePresence` trên runner), rồi bỏ `skipIf`.
- Khi quyết định giữ hay bỏ manim / mockup răng: bật lại hoặc xoá các test ở D9.

Phát hiện nhờ CI: `services/rendering/requirements-dev.txt` thiếu `httpx` (test import nó, venv local có sẵn do cài tay). Đã bổ sung `httpx==0.28.*`.

## 9. Phase 7: cổng review, giới hạn chấp nhận và backlog (Creator, 2026-09-29)

**Quyết định D11:** dừng gia cố hook ở máy sau 3 vòng review thật. Cổng ở máy được giữ với vai trò **chống sơ suất và bỏ bước**, không chống một agent cố tình gian lận: hook chạy chung máy, chung quyền với agent và nhận diện lệnh qua chữ. Các mục dưới đây vào backlog.

| # | Mức | Nguồn | Nội dung | Hướng sửa |
|---|---|---|---|---|
| H1 | Major | reviewer, vòng 3 | `record_review.py` chỉ so **tin nhắn đầu tiên** của agent review với brief. Session có thể nhắn thêm (SendMessage) giữa chừng để "gợi ý" verdict | Từ chối PASS nếu transcript có tin nhắn `user` dạng text sau brief, trừ các thông báo do Claude Code chèn (cần xác định chắc định dạng) |
| H2 | Major | security-reviewer, vòng 3 | Diff review tính so với **local `main`**. Commit nằm trên local `main` nhưng chưa push sẽ nằm dưới merge-base: không được review nhưng vẫn lên `origin/main` | Tính diff so với `origin/main` (fetch trước) trong `review-prep.sh` và hook; chặn `git commit` trực tiếp trên `main`; thêm test |
| H3 | Chiến lược | Creator + agent | **Cổng phía server (triệt để):** agent mở PR; GitHub chạy CI + 3 agent review (GitHub Action gọi Claude API); ruleset "Require PR + required checks" + auto-merge. Agent không sửa được quy tắc của GitHub | Đổi D1 (từ merge local sang PR + auto-merge); cần `ANTHROPIC_API_KEY` trong GitHub Secrets và chi phí mỗi PR. Cần Creator duyệt |
| H4 | Minor | vòng 3 | Tên nhánh và tên file thay đổi đi vào brief "được tin" mà không kiểm tra ký tự | Kiểm tra tên nhánh như label; liệt kê tên file qua bộ lọc ký tự an toàn |
| H5 | Minor | vòng 3 | Diff chịu ảnh hưởng của `.git/info/attributes`, `diff.external`, `textconv` | `git -c core.attributesFile=/dev/null diff --no-ext-diff --no-textconv --text` ở cả hai chỗ |
| H6 | Minor | vòng 3 | Guard không bắt lệnh nhắm vào thư mục cha (`rm -r .git/conceptflow`, `cd` rồi xoá) → xoá được `.fail` | Khớp cả `conceptflow` trần và `cd` vào đó; về lâu dài để marker ngoài vùng agent ghi được (hoặc làm H3) |
| H7 | Minor | vòng 3 | Hook không đối chiếu `agent_type` với loại agent Claude Code ghi cho `agent_id` | Đọc loại agent từ transcript/meta nếu Claude Code có lưu |
| H8 | Minor | vòng 3 | FAIL chỉ "dính" theo tree; một commit vặt tạo tree mới là chạy lại được | Ghi rõ trong tài liệu, hoặc mang FAIL sang tree sau cho tới khi các file bị nêu thực sự đổi |
| H9 | Minor | vòng 3 | Brief không có trỏ tới mục audit mà `tester` cần khi không có requirement doc | Thêm dòng `Audit: aidlc-docs/audit.md (## <label>)` cố định vào brief |
| H10 | Minor | vòng 3 | `implementation-audit.md` bước 7 vẫn ghi "qc"; agent thực tế tên `tester` | Đã ghi nhận ở đây: vai trò QC = agent `tester` |

Trạng thái lúc chốt: tree `aa9d777…` có `make check` pass; `tester` PASS; `reviewer` FAIL (H1); `security-reviewer` FAIL (H2). Merge Phase 7 do Creator quyết định (xem tin nhắn cuối phiên).

## 10. graphify: đồ thị tri thức của code (Creator, 2026-09-29)

**Quyết định D12:** dùng [graphify](https://github.com/Graphify-Labs/graphify) (`graphifyy` 0.9.71) làm bản đồ code cho agent. Chi tiết và cách dùng: [`graphify.md`](graphify.md).

| Chủ đề | Chọn | Không chọn |
|---|---|---|
| Tích hợp với Claude Code | Hướng dẫn trong `CLAUDE.MD` + bước trong `/cr-start`, `/fix-bug`, `/cr-review` | Hook PreToolUse của graphify (`graphify claude install`): nhắc "MANDATORY" ở mọi Read/Grep/Glob/Bash, kể cả agent review chỉ đọc |
| Phạm vi | Phân tích AST cục bộ (code + cấu trúc markdown), không LLM, không API key | Trích xuất ngữ nghĩa tài liệu bằng LLM (gửi nội dung ra ngoài, tốn token) |
| Lưu trữ / độ mới | `graphify-out/` git-ignored; hook git `post-commit`/`post-checkout` của graphify (`make graph-hooks`); `review-prep.sh` luôn cập nhật cho tree đang review | Commit `graph.json` (~15 MB, xung đột khi merge, origin public) |

Brief review có mục "Graph impact" (`scripts/hooks/graph_impact.py`): file ngoài diff phụ thuộc trực tiếp vào file bị đổi; import Go tính theo package; chỉ in đường dẫn được git theo dõi; ghi rõ "not available" khi đồ thị thiếu hoặc không dựng tại HEAD.
