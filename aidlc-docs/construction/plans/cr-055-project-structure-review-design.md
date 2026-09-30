# CR-055 — Rà soát cấu trúc toàn dự án

## 1. Yêu cầu gốc (nguyên văn)

> review và kiểm tra sruture toàn bộ dư án xem dự án đã được tổ chức đúng chưa cần sửa chỗ nào không

Trả lời của Creator cho mục 4 (nguyên văn):

> 1 => chọn a
> 2 => update hoặc xoá lại tài liệu thiết kế cho đúng
> 3 => tôi chưa nhớ nữa, bạn có thể compare nhánh nào không làm thì xoá đi
> 4 => ok đồng ý

→ 1a (test chống lệch). 2: xoá tài liệu thiết kế riêng của hai service đã gỡ, cập nhật tài liệu kiến trúc còn dùng cho khớp 9 service hiện tại (mục 2.4). 3: đã so sánh (mục 2.5) — xoá 4 nhánh bỏ dở, giữ CR-050. 4a.

## 2. Hiện trạng

### 2.1 Cách rà

- Graph graphify tại `dff096c` (khớp HEAD): đọc `GRAPH_REPORT.md` (không có import cycle; god node là `Audit Log`, `ConceptFlowScene`, `FfmpegVideoAssembler`, `Project`), `graphify explain` cho các symbol domain dùng chung giữa hai service Go.
- Cây file lấy từ `git ls-files` (gom theo thư mục), không đọc từng file. File bị `.gitignore` lấy từ `git status --ignored`.
- So sánh các file cùng tên giữa các service Python bằng `diff`.
- Đối chiếu tài liệu (README, ADR, `aidlc-state.md`, Grafana dashboard, CLAUDE.md) với code thật.
- CR-053 đã xoá file code chết; CR này **không** lặp lại việc đó, chỉ xét cách tổ chức.

### 2.2 Phần đã đúng — giữ nguyên

| Hạng mục | Nhận xét |
|---|---|
| Ranh giới service | 9 service trong `services/`, mỗi service một Dockerfile, build context riêng (`docker-compose.yml:60,117,208,281,337,400,438,515,551`), DB riêng (ADR-0013). Không có import chéo giữa service; graphify báo không có import cycle. |
| Hexagonal (ADR-0002) | tts, rendering, video-assembly, publisher: `domain/ → application/ → adapters/<công nghệ>/`, test soi gương `tests/{domain,application,adapters}`. orchestrator, authoring-service: layout Go chuẩn `cmd/<svc>/` + `internal/{domain,application,adapters,config}`, test đặt cạnh code (`*_test.go`). |
| api-gateway | Phân lớp `src/{routes,handlers,clients,middleware,config}` đúng ADR-0020, test soi gương. |
| web-gui | `src/{pages,components,hooks,api,context,utils,types,styles}` + `tests/` soi gương, đúng ADR-0021. |
| Hợp đồng chéo service | `tests/contracts/` ở gốc kiểm hợp đồng volume `shared_artifacts` (`docs/contracts/shared-artifacts.md`) và luồng bước (`test_flow_steps_contract.py`); `tests/fixtures/narration-duration-vectors.json` giữ ba bản ước lượng thời lượng (orchestrator, authoring-service, web-gui) khớp nhau. |
| Secret | `.env`, `.env.window`, `secrets/*.json`, `client_secret_*.json` đều bị ignore; trong git chỉ có `secrets/README.md` và `.env.example`. |
| Hạ tầng | `infra/rabbitmq`, `infra/observability` tách khỏi code service. |

### 2.3 Vấn đề tìm thấy

#### Nhóm 1 — Tài liệu cấu trúc sai so với code (ưu tiên cao: người/agent mới đọc sẽ hiểu sai)

| # | Chỗ | Sai thế nào |
|---|---|---|
| 1.1 | `README.md:47,60,64,101,111,128,132` | Còn mô tả `content-plugin` (đã gỡ ở CR-020, commit `59cb813`) và "Script Processing Service" (đã gỡ ở CR-040); nhắc thư mục `shared/` không tồn tại; **thiếu** authoring-service, llm-service, ollama; cây "Project Structure" không có `docs/`, `tests/`, `scripts/`, `Makefile`; mục CI/CD và mục test tổng hợp nói "sẽ bổ sung" dù lớp đó đã bị gỡ (29–30/09). |
| 1.2 | `aidlc-docs/decisions/ADR-0006-*.md`, `ADR-0012-*.md` | Trạng thái vẫn `Accepted` dù Content Plugin Service đã bị gỡ. Phải là `Superseded`/`Deprecated` kèm CR đã gỡ. |
| 1.3 | `aidlc-docs/decisions/README.md` | Bảng chỉ liệt kê ADR-0001…0015, rồi danh sách rời 0023/0024/0025/0029. **Thiếu** ADR-0016…0022, 0026…0028; ADR-0006/0012 sai trạng thái. |
| 1.4 | `aidlc-docs/construction/content-plugin-service/`, `script-processing-service/` + 12 file `plans/content-plugin-service-*`, `plans/script-processing-service-*` | Thiết kế của hai service đã gỡ nằm lẫn với service đang chạy, không có dấu hiệu "đã gỡ". Ngược lại **không có** thư mục thiết kế cho `authoring-service` và `llm-service` (thiết kế của chúng nằm rải trong CR-039, CR-040, ADR-0029). |
| 1.5 | `aidlc-docs/aidlc-state.md:54-…` | Bảng CR dừng ở CR-051; thiếu CR-052, 053, 054. |
| 1.6 | CLAUDE.md, mục "Read the codebase through graphify first" | Nói hợp đồng RabbitMQ/HTTP nằm ở `docs/contracts/`, nhưng thư mục này chỉ có `shared-artifacts.md`. Hợp đồng thật nằm ở `aidlc-docs/construction/<service>/low-level-design/interface-contracts.md` (và các file messaging-design). |
| 1.7 | `infra/observability/grafana/provisioning/dashboards/json/pipeline-overview.json:36-37,69,168,181,465` | Dashboard còn biến, panel và truy vấn Loki cho container `script-processing` (không còn tồn tại) → panel luôn trống. |
| 1.8 | `services/rendering/adapters/rendering/remotion_renderer.py:8-11`, `services/rendering/remotion_project/render.mjs:12` | Chú thích trỏ tới `script-processing/adapters/parsing/manim_script_parser.py` và "script-processing's parsed scene_class_name" — file/service không còn; logic đó giờ là `services/rendering/domain/script_locator.py`. |
| 1.9 | File `CLAUDE.MD` (git theo dõi tên viết hoa) | Claude Code tìm `CLAUDE.md`. Chạy được trên macOS/Windows vì hệ file không phân biệt hoa thường, nhưng trên Linux (container, cloud session) file sẽ không được nạp. |
| 1.10 | README từng service | Chỉ api-gateway, orchestrator, publisher, web-gui có README; tts, rendering, video-assembly, authoring-service, llm-service không có. |

#### Nhóm 2 — Mã hạ tầng Python chép tay giữa 4 service (ưu tiên trung bình: rủi ro lệch dần)

Đo bằng `diff` (số dòng khác):

| File | tts | rendering | video-assembly | publisher |
|---|---|---|---|---|
| `adapters/persistence/outbox.py` (23 dòng) | gốc | 0 | 0 | 0 |
| `adapters/persistence/relay.py` (80 dòng) | gốc | 0 | 0 | 0 |
| `adapters/persistence/inbox.py` (40 dòng) | gốc | 0 | 6 (chỉ docstring) | 6 (chỉ docstring) |
| `adapters/persistence/db.py` | gốc | 0 | 29 (có thêm) | 86 (có thêm) |
| `adapters/messaging/cancellation.py` | gốc | 0 | 47 (có thêm) | — |
| `adapters/messaging/purge.py` (71 dòng) | gốc | 4 (tên service) | 4 (tên service) | — |
| `tests/adapters/fake_postgres.py` (82 dòng) | gốc | 0 | 41 (có thêm) | 0 |

`consumer.py`, `producer.py`, `progress.py`, `artifact_paths.py` khác nhau thật theo nghiệp vụ từng service — không tính là trùng. Hiện không có gì bảo đảm các bản giống hệt nhau vẫn giống nhau: sửa lỗi relay/outbox ở một service rất dễ quên ba service còn lại.

Cũng có trùng ở Go: `orchestrator/internal/domain/narration.go` (232 dòng) và `authoring-service/internal/domain/narration.go` (276 dòng) — phần ước lượng thời lượng đã có vector chung giữ khớp; phần phụ đề (`SubtitleZone`…) đã bắt đầu lệch (authoring có thêm `SubtitleZoneFor`, `SubtitleBandFor`, `ProjectSubtitleBand`). Tách service là quyết định có chủ ý (ADR-0029), nên CR này chỉ ghi nhận, không gộp.

#### Nhóm 3 — Rác cục bộ và nhánh cũ (không nằm trong git, nhưng làm rối máy và graph)

| # | Chỗ | Ghi chú |
|---|---|---|
| 3.1 | `services/content-plugin/` | Chỉ còn `__pycache__` và `.pytest_cache` của service đã gỡ. Thư mục rỗng về mã, nhưng vẫn hiện trong `services/`. |
| 3.2 | `services/web-gui/services/tts/.ruff_cache` | Cache ruff do chạy lệnh sai thư mục. |
| 3.3 | `.claude/worktrees/cr-052` | Worktree của `feature/cr-052-…` — nhánh đã merge vào main. |
| 3.4 | Nhánh git | 106 nhánh local (99 đã merge vào main), 91 nhánh remote (84 đã merge). |
| 3.5 | 5 nhánh **chưa** merge | `feature/cr-029-progress-percent`, `cr-034-qc-severity-threshold`, `cr-035-validate-script-cache`, `cr-036-auto-approve-outline`, `cr-050-llm-call-chunking-resume` — cần Creator cho biết còn làm tiếp hay bỏ. |
| 3.6 | `services/*/.venv` | `rendering/.venv/bin/pytest` trỏ tới đường dẫn cũ `ConcertFlow/...` (ghi nhận trong audit CR-053); venv cục bộ hỏng, phải gọi `.venv/bin/python -m pytest`. |

#### Nhóm 4 — Ghi nhận, không đề xuất sửa

- `llm-service` dùng layout phẳng `app/{main,provider,registry,pipeline/…}.py` thay vì hexagonal. Service nhỏ (≈15 file mã), chỉ bọc lời gọi LLM; ép vào `domain/application/adapters` không đem lại lợi ích tương xứng.
- `services/rendering/conceptflow/` là thư viện chạy **bên trong** script Manim của Creator, không phải lớp của service — đặt ở gốc service là đúng (được copy vào image).
- Tên tài liệu CR không đồng nhất (`cr-0xx-execution-plan.md`, `-low-level-design.md`, `-functional-design.md`, `-design.md`) và CR-053/054 không có file requirements riêng — là lịch sử theo từng giai đoạn quy trình; đổi tên sẽ làm gãy liên kết trong `audit.md`.
- `docs/brand/` chứa ảnh PNG/JPEG lớn (banner 2560×1440) trong git — chấp nhận được vì là nguồn của `make-banner.py`.

### 2.4 Tài liệu thiết kế nhắc service đã gỡ (cho câu trả lời 2)

- **Tài liệu riêng của service đã gỡ** — 41 file: `aidlc-docs/construction/content-plugin-service/**`, `aidlc-docs/construction/script-processing-service/**`, `aidlc-docs/construction/plans/content-plugin-service-*.md` (6), `plans/script-processing-service-*.md` (6). Chỉ 5 ADR link vào chúng: ADR-0008, 0009, 0011, 0012, 0013 (mỗi file 1 link); `audit.md` không link. → Xoá được; bản cũ vẫn xem được bằng `git show dff096c:<đường dẫn>`.
- **Tài liệu kiến trúc còn dùng nhưng mô tả bộ service cũ** (số lần nhắc content-plugin/script-processing): `inception/high-level-design/{architecture-overview (6), integration-boundaries (6), high-level-design (3), architectural-style (2), system-context (1)}.md`; `inception/application-design/{unit-of-work (9), component-methods (8), unit-of-work-dependency (6), component-dependency (6), services (4), components (4), application-design (4), unit-of-work-story-map (3)}.md`; `construction/build-and-test/{integration-test-instructions (7), build-instructions (4), unit-test-instructions (3), ci-cd-integration-instructions (3), build-and-test-summary (3)}.md`; `construction/rabbitmq-infrastructure/{nfr-design/logical-components (8), nfr-requirements/tech-stack-decisions (1), infrastructure-design/deployment-architecture (1)}.md`; `construction/api-gateway/{nfr-design/logical-components (2), low-level-design/module-structure (2), infrastructure-design/deployment-architecture (2), low-level-design/{sequence-flows,interface-contracts,dependency-injection} (1), infrastructure-design/infrastructure-design (1)}.md`; `construction/orchestrator-service/low-level-design/interface-contracts.md (2)`. Các tài liệu này cũng **không có** authoring-service, llm-service, ollama.
- Không đụng: `audit.md`, `aidlc-state.md` (phần lịch sử), `inception/requirements/**`, tài liệu CR trong `plans/cr-*` — là nhật ký, ghi đúng tại thời điểm viết.
- Thực tế hiện tại để đối chiếu: topology RabbitMQ (`infra/rabbitmq/definitions.json`) chỉ còn `tts|rendering|video_assembly|publisher.commands` (+dlq), `orchestrator.events` (+dlq), `progress.fanout`, `control.fanout`, `commands.direct`, `events.direct`, `dlx.direct`; api-gateway không còn route plugin.

### 2.5 So sánh 5 nhánh chưa merge (cho câu trả lời 3)

| Nhánh | Commit cuối | Nội dung | Có trên main chưa | Merge vào main hiện tại |
|---|---|---|---|---|
| `feature/cr-029-progress-percent` | 2026-09-22 | 1 commit, web-gui: thêm "(x%)" sau "cảnh i/n" trong `ProgressTracker` | Không đúng như vậy, nhưng main đã có thanh % (`ProgressTracker.tsx:45-58,103`) | Xung đột |
| `feature/cr-034-qc-severity-threshold` | 2026-09-23 | 1 commit, orchestrator: `QC_ENFORCE` → `QC_ENFORCE_THRESHOLD` theo mức | Chưa (main vẫn `QC_ENFORCE` bool, `config.go:94`) | Xung đột |
| `feature/cr-035-validate-script-cache` | 2026-09-23 | 1 commit, rendering: cache lượt dry theo hash | Chưa (không có `DryRunCache` trên main) | Xung đột |
| `feature/cr-036-auto-approve-outline` | 2026-09-23 | 1 commit, orchestrator + web-gui: tự duyệt dàn ý khi không có cảnh báo | Chưa. Code nằm ở orchestrator nhưng dàn ý/soạn đã chuyển sang authoring-service ở CR-040, và nhánh còn sửa `useRequireScript.test.tsx` đã xoá ở CR-053 | Xung đột |
| `feature/cr-050-llm-call-chunking-resume` | 2026-09-29 | 2 commit: requirements + Unit 1 (5 unit), 37 file | Chưa. `cr-050-workflow-plan.md` ghi U1 "Code xong, chờ duyệt" | Xung đột |

Kết luận: 029, 034, 035, 036 **bỏ dở** — một commit duy nhất từ 22–23/09, không có mục audit trên main, dựng trên kiến trúc trước CR-040 nên không merge thẳng được. Xoá. Để không mất hẳn phần việc, trước khi xoá gắn tag `archive/cr-0xx-<slug>` vào commit cuối và đẩy tag lên origin (khôi phục bằng `git checkout -b <nhánh> archive/...`). CR-050 **đang làm dở** (hôm qua, còn chờ duyệt Unit 1) — giữ nguyên.

## 3. Yêu cầu

- **FR1** — Sửa toàn bộ Nhóm 1 để tài liệu khớp code:
  - FR1.1 Viết lại các mục "Running the Project", "Running Tests", "Project Structure", "CI/CD" của `README.md` theo 9 service + ollama + observability hiện có.
  - FR1.2 Đổi trạng thái ADR-0006, ADR-0012 sang `Superseded` (nêu CR-020 đã gỡ content-plugin), giữ nguyên nội dung quyết định.
  - FR1.3 Làm lại `aidlc-docs/decisions/README.md` thành một bảng đủ 29 ADR với trạng thái đúng.
  - FR1.4 Tài liệu thiết kế (câu trả lời 2, mục 2.4):
    - FR1.4.1 Xoá 41 file thiết kế riêng của content-plugin-service và script-processing-service; sửa 5 link trong ADR-0008, 0009, 0011, 0012, 0013 thành chữ thường kèm ghi chú "đã gỡ ở CR-0xx, bản cũ: `git show dff096c:<đường dẫn>`".
    - FR1.4.2 Cập nhật các tài liệu kiến trúc còn dùng ở mục 2.4 cho khớp 9 service hiện tại: bỏ hoặc sửa mọi đoạn mô tả content-plugin/script-processing như thành phần đang chạy; thêm authoring-service, llm-service, ollama vào sơ đồ/bảng thành phần, phụ thuộc, ranh giới tích hợp, hướng dẫn build/test.
  - FR1.5 Bổ sung CR-052…055 vào bảng CR của `aidlc-state.md`.
  - FR1.6 Thêm `docs/contracts/README.md` làm mục lục trỏ tới mọi hợp đồng (HTTP, RabbitMQ, volume), và sửa câu trong CLAUDE.md cho khớp.
  - FR1.7 Gỡ biến/panel/truy vấn `script-processing` khỏi dashboard Grafana.
  - FR1.8 Sửa 2 chú thích lỗi thời trong rendering trỏ về `domain/script_locator.py`.
  - FR1.9 Đổi tên `CLAUDE.MD` → `CLAUDE.md` trong git.
- **FR2** — Test chống lệch cho mã hạ tầng Python chép tay (1a).
- **FR3** — Dọn rác cục bộ 3.1–3.3; xoá nhánh local đã merge, giữ nhánh remote đã merge (4a).
- **FR4** — Xoá nhánh bỏ dở `feature/cr-029-progress-percent`, `cr-034-qc-severity-threshold`, `cr-035-validate-script-cache`, `cr-036-auto-approve-outline` (local + remote) sau khi gắn tag `archive/…` và đẩy tag lên origin. Giữ `feature/cr-050-llm-call-chunking-resume`.
- **FR5** — Không đổi hành vi runtime: không đổi contract, DB, message, API.

**Tiêu chí chấp nhận**
- `git grep -il "content.plugin\|script.processing"` trong `aidlc-docs/inception/{high-level-design,application-design}`, `aidlc-docs/construction/{build-and-test,rabbitmq-infrastructure,api-gateway,orchestrator-service}` chỉ còn các câu ghi rõ là lịch sử ("đã gỡ ở CR-0xx").
- Ngoài `aidlc-docs/`, `git grep -i "content-plugin\|script-processing"` chỉ còn chú thích lịch sử có ghi CR (vd. `script_locator.py:3`).
- Không còn file nào dưới `aidlc-docs/construction/{content-plugin-service,script-processing-service}/` và `plans/{content-plugin-service,script-processing-service}-*`; không link markdown nào trỏ tới chúng.
- Mọi ADR trong `aidlc-docs/decisions/` có đúng một dòng trong bảng README.
- `pytest tests/contracts` pass, kể cả test mới; sửa tạm một file bị giám sát thì test fail.
- Test của video-assembly, publisher, rendering pass.
- Grafana dashboard nạp được, không còn panel `script-processing`.
- `git tag -l 'archive/*'` có 4 tag, có trên origin; 4 nhánh bỏ dở không còn ở local lẫn remote; `feature/cr-050-…` còn nguyên.

**Ngoài phạm vi**: gộp `narration.go` giữa hai service Go; tái cấu trúc llm-service; đổi tên tài liệu CR cũ; README riêng cho từng service; viết bộ thiết kế chi tiết (functional/NFR/infrastructure) mới cho authoring-service và llm-service — chỉ đưa chúng vào tài liệu kiến trúc chung; sửa `audit.md`, `inception/requirements/**`, tài liệu CR cũ (là nhật ký); làm tiếp nội dung các nhánh bỏ dở.

## 4. Câu hỏi cho Creator (đã trả lời: 1a; 2 cập nhật/xoá; 3 so sánh rồi xoá nhánh bỏ dở; 4a)

1. Mã hạ tầng Python chép tay: a) giữ bản sao + test chống lệch (**chọn**); b) thư viện chung; c) không làm.
2. Thiết kế của service đã gỡ: ban đầu đề xuất thêm banner; Creator chọn "update hoặc xoá lại cho đúng" → xoá tài liệu riêng của service đã gỡ, cập nhật tài liệu kiến trúc chung (mục 2.4, FR1.4).
3. 5 nhánh chưa merge: Creator không nhớ, nhờ so sánh → mục 2.5: xoá 029/034/035/036 (có tag archive), giữ 050.
4. Nhánh đã merge: a) xoá local + worktree cr-052, giữ remote (**chọn**).

## 5. Giải pháp đề xuất

Cấu trúc code của dự án về cơ bản **đúng**: ranh giới service rõ, mỗi service theo đúng phong cách đã chọn trong ADR, test soi gương code, hợp đồng chéo service có test giữ. Chỗ sai chủ yếu nằm ở **tài liệu không theo kịp hai lần gỡ service** (CR-020, CR-040) và lần gỡ lớp agentic (29–30/09), cộng một điểm rủi ro kỹ thuật là mã hạ tầng Python chép tay.

Cách làm: sửa tài liệu/chú thích/dashboard (không đụng logic); xoá tài liệu của service không còn thay vì gắn banner, vì git history vẫn giữ bản cũ và chỉ 5 link cần sửa; tài liệu kiến trúc chung được sửa tại chỗ để người đọc mới thấy đúng 9 service. Mã hạ tầng giữ bản sao nhưng có test chống lệch. Nhánh bỏ dở được gắn tag trước khi xoá để có thể lấy lại.

## 6. Phạm vi

- **Tài liệu**: `README.md`, `CLAUDE.MD`→`CLAUDE.md`, `docs/contracts/README.md` (mới), `aidlc-docs/decisions/README.md`, ADR-0006/0008/0009/0011/0012/0013, `aidlc-docs/aidlc-state.md`, xoá 41 file thiết kế service đã gỡ, sửa các tài liệu kiến trúc liệt kê ở mục 2.4.
- **Hạ tầng**: `infra/observability/grafana/provisioning/dashboards/json/pipeline-overview.json` → restart `grafana`.
- **rendering**: chú thích trong `adapters/rendering/remotion_renderer.py`, `remotion_project/render.mjs` (không đổi logic).
- **video-assembly / publisher**: chỉ docstring đầu `adapters/persistence/inbox.py`.
- **Test mới**: `tests/contracts/test_python_infra_copies.py`.
- **Git**: tag `archive/cr-029-progress-percent`, `archive/cr-034-qc-severity-threshold`, `archive/cr-035-validate-script-cache`, `archive/cr-036-auto-approve-outline`; xoá 4 nhánh đó (local + origin); xoá nhánh local đã merge.
- Không đổi contract, DB, migration, message. `graphify affected` không cần cho symbol nào vì không symbol nào đổi hành vi.

## 7. Kế hoạch thực hiện (cho `/code`)

1. **CLAUDE.md**: `git mv CLAUDE.MD CLAUDE.tmp && git mv CLAUDE.tmp CLAUDE.md` (hai bước vì macOS không phân biệt hoa thường). Sửa câu "For what the graph cannot see … go straight to `docs/contracts/`" thành trỏ tới `docs/contracts/README.md` (mục lục).
2. **`docs/contracts/README.md`** (mới): bảng liệt kê từng hợp đồng và file nguồn — `shared-artifacts.md`; với mỗi service còn chạy trong `aidlc-docs/construction/*/low-level-design/` liệt kê `interface-contracts.md` và file messaging-design nếu có (tìm bằng `ls aidlc-docs/construction/*/low-level-design/`, sau bước 5); ADR-0029 cho HTTP nội bộ orchestrator ↔ authoring-service; `infra/rabbitmq/definitions.json` cho topology queue/exchange.
3. **`README.md`**:
   - "Running the Project": xoá dòng Content Plugin; thêm authoring-service (Go, DB `authoring-service-db`, gọi nội bộ từ orchestrator — ADR-0029), llm-service, ollama (lấy vai trò/port từ `docker-compose.yml:399-513`); sửa dòng API Gateway (bỏ `/v1/plugins`, Content Plugin; liệt kê route theo `services/api-gateway/src/routes/`); rendering có HTTP nội bộ (`services/rendering/adapters/http/`).
   - "Running Tests": bỏ ví dụ content-plugin và "Script Processing"; thêm authoring-service (`go test ./...`), llm-service (`pytest`), và `pytest tests/contracts` ở gốc; câu cuối về Build and Test trỏ tới `aidlc-docs/construction/build-and-test/`.
   - "Project Structure": vẽ lại cây theo `git ls-files` thật (9 service + `infra/`, `docs/`, `tests/`, `scripts/`, `data/`, `secrets/`, `aidlc-docs/`, `.ai-dlc/`, `.claude/skills/`); bỏ `shared/`.
   - "CI/CD": ghi rõ không có CI (lớp verification đã gỡ 2026-09-30), kiểm tra bằng test từng service.
4. **ADR**: trong `ADR-0006-*.md` và `ADR-0012-*.md`, đổi `## Status` thành `Superseded — Content Plugin Service đã gỡ ở CR-020 (commit 59cb813)`. Làm lại `aidlc-docs/decisions/README.md` thành một bảng duy nhất 29 dòng (ADR, tiêu đề lấy từ dòng `#` đầu file, trạng thái lấy từ mục `## Status` của từng file, ngày nếu file có).
5. **Xoá thiết kế service đã gỡ**: `git rm -r aidlc-docs/construction/content-plugin-service aidlc-docs/construction/script-processing-service aidlc-docs/construction/plans/content-plugin-service-*.md aidlc-docs/construction/plans/script-processing-service-*.md`. Trong ADR-0008, 0009, 0011, 0012, 0013 thay link tới các file đó bằng chữ thường + "(đã gỡ ở CR-020/CR-040; bản cũ: `git show dff096c:<đường dẫn>`)". Kiểm `git grep -n "content-plugin-service/\|script-processing-service/\|plans/content-plugin-service-\|plans/script-processing-service-"` chỉ còn trong `audit.md`/tài liệu CR (nếu có).
6. **Cập nhật tài liệu kiến trúc chung** (danh sách ở mục 2.4). Nguồn sự thật: `docker-compose.yml`, `infra/rabbitmq/definitions.json`, `services/api-gateway/src/routes/`, ADR-0029, `docs/contracts/shared-artifacts.md`, `aidlc-docs/inception/requirements/cr-040-*.md` (ranh giới sau CR-040), `cr-039-*.md` (llm-service). Với từng file:
   - Đoạn mô tả content-plugin/script-processing là thành phần đang chạy → xoá (bảng, sơ đồ ASCII, danh sách queue, route gateway, bước build/test) hoặc viết lại theo thành phần đang đảm nhận việc đó (vd. phân tích script → `rendering/domain/script_locator.py` + `validate_script`; soạn kịch bản → authoring-service).
   - Thêm authoring-service, llm-service, ollama vào: `architecture-overview.md`, `system-context.md`, `integration-boundaries.md`, `high-level-design.md`, `application-design/{services,components,component-dependency,application-design}.md`, `build-and-test/{build-instructions,unit-test-instructions,integration-test-instructions,build-and-test-summary}.md`.
   - `unit-of-work*.md`, `component-methods.md` (kế hoạch unit ban đầu): thêm một đoạn đầu file "Hiện trạng" nêu Unit 2 (content-plugin) gỡ ở CR-020, script-processing gỡ ở CR-040, authoring-service/llm-service thêm ở CR-039/CR-040; sửa các bảng phụ thuộc cho đúng, không viết lại lịch sử lập kế hoạch.
   - `ci-cd-integration-instructions.md`: đầu file ghi rõ không có CI (gỡ 2026-09-30); bỏ các bước nhắc service đã gỡ.
   - Tài liệu rabbitmq-infrastructure/api-gateway/orchestrator-service: sửa đúng các dòng nhắc service đã gỡ theo topology/route hiện có.
   - Cuối bước: chạy lại lệnh đếm ở mục 2.4, mọi lần nhắc còn lại phải là câu lịch sử có ghi CR.
7. **`aidlc-state.md`**: thêm dòng CR-052, CR-053, CR-054, CR-055 vào bảng CR, nội dung lấy từ các mục tương ứng trong `audit.md`.
8. **Grafana**: trong `pipeline-overview.json` bỏ lựa chọn `script-processing` khỏi biến (dòng 36-37 và chuỗi `query` dòng 69), xoá panel `"title": "script-processing"` (quanh dòng 166-204), bỏ `script-processing|` khỏi regex dòng 465. Kiểm JSON hợp lệ bằng `python3 -m json.tool`. Nếu bỏ panel để lại lỗ trên lưới, dịch `gridPos` các panel sau cho khít.
9. **Chú thích rendering**: `remotion_renderer.py:8-11` sửa thành "`domain/script_locator.py` đã yêu cầu cặp Composition/registerRoot…"; `render.mjs:12` sửa "from script-processing's parsed scene_class_name" thành "from domain/script_locator.py". Không đổi code.
10. **Test chống lệch (1a)**:
    - Thống nhất docstring `adapters/persistence/inbox.py` của video-assembly và publisher về đúng bản của tts (để 4 file giống hệt).
    - Thêm `tests/contracts/test_python_infra_copies.py`: với mỗi nhóm `(đường dẫn tương đối, [service])` — `adapters/persistence/outbox.py`, `adapters/persistence/relay.py`, `adapters/persistence/inbox.py` × {tts, rendering, video-assembly, publisher}; `adapters/persistence/db.py`, `adapters/messaging/cancellation.py` × {tts, rendering}; `tests/adapters/fake_postgres.py` × {tts, rendering, publisher}; `adapters/messaging/purge.py` × {tts, rendering, video-assembly} sau khi bỏ dòng docstring đầu và dòng `SERVICE_NAME = ...` — assert nội dung bằng nhau, thông báo lỗi nêu file lệch và gợi ý chép bản sửa sang các service còn lại. Docstring đầu file test ghi lý do (mỗi service build độc lập, ADR-0001) và cách bỏ một file khỏi nhóm khi service cố ý cần khác.
    - Thử: sửa tạm một dòng trong `services/rendering/adapters/persistence/relay.py` → test phải fail → khôi phục.
11. **Nhánh bỏ dở** (mục 2.5): với mỗi `<slug>` trong `cr-029-progress-percent`, `cr-034-qc-severity-threshold`, `cr-035-validate-script-cache`, `cr-036-auto-approve-outline`: `git tag archive/<slug> feature/<slug>` → `git push origin archive/<slug>` → kiểm `git ls-remote --tags origin archive/<slug>` có kết quả → `git branch -D feature/<slug>` → `git push origin --delete feature/<slug>`. Không đụng `feature/cr-050-llm-call-chunking-resume`.
12. **Dọn cục bộ (không commit)**: `rm -rf services/content-plugin services/web-gui/services`; `git worktree remove .claude/worktrees/cr-052` (nếu báo có thay đổi chưa commit thì dừng và báo); `git branch --merged main | grep -vE '^\*|^\+|^  main$' | xargs git branch -d` (dùng `-d` để git từ chối nhánh chưa merge). Không xoá nhánh remote đã merge.
13. `make graph` để graph phản ánh cấu trúc mới.

## 8. Kiểm tra

- `pytest tests/contracts -q` ở gốc (dùng venv của một service Python có pytest).
- `pytest -q` trong rendering, video-assembly, publisher (chỉ đổi docstring/chú thích, vẫn chạy để chắc).
- `python3 -m json.tool pipeline-overview.json`; `docker compose restart grafana` rồi mở dashboard "Pipeline overview" kiểm không còn panel `script-processing`.
- Theo chính sách Docker: rebuild `rendering`, `video-assembly`, `publisher` (có file đổi dù chỉ chú thích/docstring) và xác nhận healthy.
- Các lệnh `git grep` ở tiêu chí chấp nhận; `git ls-files | grep -c '^CLAUDE.md$'` = 1.
- `git tag -l 'archive/*'`, `git ls-remote --tags origin 'archive/*'`, `git branch -a | grep -E 'cr-0(29|34|35|36)'` rỗng, `git branch -a | grep cr-050` còn.

## 9. Rủi ro

- Đổi tên `CLAUDE.MD` trên macOS: nếu làm một bước, git có thể không ghi nhận. Đã tách hai bước ở kế hoạch.
- Xoá nhánh bỏ dở: có tag `archive/*` trên origin nên lấy lại được; nếu đẩy tag thất bại thì không xoá nhánh.
- Xoá nhánh local đã merge: `git branch -d` chỉ xoá nhánh đã merge; nhánh remote vẫn còn.
- Xoá tài liệu thiết kế service đã gỡ: bản cũ vẫn trong git history (`git show dff096c:<đường dẫn>`).
- Cập nhật tài liệu kiến trúc là việc viết tay trên ~30 file; có thể sót ý. Tiêu chí `git grep` chỉ bắt được tên service, không bắt được mô tả sai theo cách khác — báo cáo của `/code` phải liệt kê file đã sửa và phần nào chỉ thêm ghi chú.
- Sửa dashboard Grafana: sai JSON sẽ làm dashboard không nạp — đã có bước `json.tool` và kiểm trên Grafana.
- Test chống lệch không ngăn được việc hai service **cố ý** cần khác nhau; khi đó bỏ file đó khỏi nhóm — ghi rõ trong docstring test.
- Không có rủi ro mất dữ liệu runtime: không đụng DB, volume, message.
