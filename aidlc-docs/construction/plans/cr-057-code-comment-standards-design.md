# CR-057 — Thiết kế: comment dạng tài liệu và bộ quy tắc chuẩn code

- **Branch**: `feature/cr-057-code-comment-standards` (tạo từ `main` @ `a5f44de`)
- **Trạng thái**: Creator đã duyệt (ADR: A, buộc tuân thủ: A)
- **Ngày**: 2026-09-30

## Yêu cầu gốc (nguyên văn)

> bạn kiểm tra giúp tôi cac comment và summary, xoá các phần ko liên quan kiểu như backlog ghi lại lí do vì sao sửa v,v ngoài phần code tôi chỉnh muốn note lại kiểu documentary code thôi thêm rule này vào 1 file tương tự file Ui Rule luôn, các rule này buộc code phải tuân thủ các code stander basic và cos comment code documentary code

Trả lời các câu hỏi làm rõ:

> 1. a
> 2. a
> 3. đông ý
> cr kia xong rồi bạn có thể bắt đầu

Duyệt thiết kế:

> buộc tuân thủ A, tham chiếu adr a luôn code đi

Nghĩa là:
1. Giao CR-056 trước (đã xong, `main` @ `a5f44de`), rồi mới mở CR-057 từ `main`.
2. Phạm vi là comment và docstring trong code của mọi service: bỏ tham chiếu `CR-xxx`, `FR-xx`, `review Cx`, `Unit N`, lý do sửa và lịch sử; chỉ giữ comment mô tả code.
3. Tạo `docs/code-standards-rules.md` theo khuôn `docs/ux-ui-design-rules.md` và thêm mục vào `CLAUDE.md` để mọi thay đổi code phải tuân theo file này.

## Hiện trạng

Graphify không lưu comment, nên phạm vi được đo bằng grep trong `services/` (bỏ qua `node_modules`, `dist`, `.venv`, `__pycache__`, `build`). Mẫu tìm: `CR-\d+`, `review [A-Z]\d+`, `ADR-\d+`, `Unit \d`, `previously`, `used to`, `no longer`, `bug fix`, `regression`, `was changed/removed/added`.

| Service | Dòng khớp | File | trong đó file test |
|---|---|---|---|
| authoring-service | 494 | 93 | 36 |
| orchestrator | 388 | 54 | 16 |
| web-gui | 341 | 114 | 31 |
| rendering | 251 | 78 | 24 |
| video-assembly | 136 | 35 | 12 |
| publisher | 92 | 30 | 12 |
| tts | 87 | 33 | 11 |
| llm-service | 77 | 16 | 5 |
| api-gateway | 0 | 0 | 0 |

Tổng khoảng 1.870 dòng. Theo loại: `CR-` 1.689, `ADR-` 150, `used to` 58, `no longer` 40, `Unit N` 22, `regression` 5, `review C` 2, `bug fix` 1. Một số chỗ `used to` và `no longer` là câu mô tả bình thường, nên phải đọc từng chỗ, không thay hàng loạt. Ngoài `services/` còn 10 dòng trong `tests/`, `scripts/`, `infra/` và 27 dòng trong `docker-compose.yml` và `.env.example`.

Các dạng đang có:

- **Ticket ghi thay cho mô tả**: [flow.go:15](../../../services/authoring-service/internal/domain/flow.go#L15) `FlowIllustrations = 5 // Hình minh hoạ (CR-046: promoted from a source="illustrations" sub-state of Code; ...)`.
- **Docstring kể lại lý do sửa**: [naming.py:17-19](../../../services/llm-service/app/pipeline/naming.py#L17-L19) `(CR-056: lowering it made the model write a key PALETTE did not have)`.
- **Tóm tắt module ghi số ticket**: [run.py:1](../../../services/llm-service/app/pipeline/run.py#L1) `"""The chunked code pipeline (CR-039 FR102-FR105, CR-050 Unit 2 / ADR-0030).`; `#: CR-048 T1 — ...` ở [run.py:108-113](../../../services/llm-service/app/pipeline/run.py#L108-L113).
- **Nhật ký phiên bản prompt trong comment**: [prompt_template_seeds.go:111](../../../services/authoring-service/internal/domain/prompt_template_seeds.go#L111) `// v7 (CR-041 phase 1) stops forcing ...`, cùng các dòng 637 và 719. Chỉ là comment Go; nội dung prompt gửi cho model không chứa `CR-`.
- **Comment SQL trong schema nhúng**: [db.go:16-60](../../../services/authoring-service/internal/adapters/postgres/db.go#L16-L60) `-- CR-025/027/028: ...`, `-- CR-040 FR111: ... used to ...`. Đây là comment SQL, không đổi DDL.
- **Chuỗi runtime có số ticket**: log [publisher/adapters/persistence/db.py:118](../../../services/publisher/adapters/persistence/db.py#L118) `"Migrated the pre-CR-012 OAuth credential ..."`; thông báo assert trong test, ví dụ [project_illustration_cr045_test.go:41](../../../services/authoring-service/internal/adapters/postgres/project_illustration_cr045_test.go#L41) `(CR-050 FR-17)`.
- **Tên file test mang số CR**: `summaries_cr051_test.go`, `project_illustration_cr045_test.go`, `code_segment_cr050_test.go` (đều ở `authoring-service/internal/adapters/postgres/`). Không có hàm test nào mang số CR.
- **Tham chiếu ADR**: ví dụ `(see ADR-0013)` ở [publisher/adapters/persistence/relay.py:11](../../../services/publisher/adapters/persistence/relay.py#L11). Đây là liên kết tới tài liệu thiết kế, không phải lịch sử.

Công cụ chuẩn code hiện có: `ruff` trong `pyproject.toml` của 5 service Python; `eslint`, `prettier`, `tsconfig` cho web-gui; Go dùng `gofmt`/`go vet`. Chưa có tài liệu nào quy định cách viết comment. `CLAUDE.md:56` và `.claude/skills/code/SKILL.md:24` chỉ trỏ tới quy tắc UI.

## Yêu cầu

**FR-1. File quy tắc** `docs/code-standards-rules.md`, viết tiếng Anh, cùng khuôn với `docs/ux-ui-design-rules.md` (các mục đánh số, câu ngắn, có ví dụ). Nội dung:

1. *Comments document the code, not its history.*
   - Được viết: code làm gì; hợp đồng của hàm (tham số, giá trị trả về, lỗi ném ra, tác dụng phụ); bất biến; lý do kỹ thuật của thiết kế hiện tại khi không hiển nhiên ("vì sao code như thế này").
   - Không được viết: số CR/FR/ticket/review/Unit; code trước đây ra sao, vì sao đã đổi, ai yêu cầu; nhật ký phiên bản; "fixed", "no longer", "used to". Những thông tin này thuộc về git log, `aidlc-docs/` và ADR.
   - Được phép: một dòng `See ADR-00NN.` khi quyết định kiến trúc nằm ở đó.
   - Không để lại code bị comment out. Không có `TODO` thiếu mô tả việc cần làm (xem CLAUDE.md "No fake code").
2. *Documentation comments* theo ngôn ngữ:
   - Python: docstring cho module, class và hàm/method public (PEP 257): câu đầu tóm tắt, sau đó tham số, trả về, ngoại lệ khi không hiển nhiên.
   - Go: doc comment cho mọi identifier export, bắt đầu bằng tên của nó (`// RenderScript renders ...`), và comment package.
   - TypeScript/React: JSDoc `/** ... */` cho function, component, hook và type export.
   - SQL trong schema: comment mô tả cột hoặc bảng dùng để làm gì.
3. *Basic code standards*:
   - Format và lint: Python theo `ruff` của service; Go theo `gofmt` và `go vet`; TS theo `eslint`, `prettier`, `tsc`. Code phải sạch lỗi trước khi báo xong.
   - Đặt tên có nghĩa theo quy ước ngôn ngữ (snake_case, MixedCaps, camelCase). Không dùng tên viết tắt khó hiểu.
   - Hàm làm một việc. Tránh lồng sâu; ưu tiên trả về sớm.
   - Không nuốt lỗi. Lỗi phải được trả về, ném ra hoặc log kèm ngữ cảnh.
   - Không để code chết, import thừa, hay số/chuỗi "ma thuật" lặp lại; đặt chúng thành hằng có tên.
   - Tuân theo kiến trúc hexagonal hiện có (domain / application / adapters), không gọi adapter từ domain.
4. *Tests*: tên file và tên test mô tả hành vi được kiểm, không mang số CR. Thông báo assert mô tả điều sai, không trích FR.
5. *Chuỗi runtime* (log, error, UI text) không chứa số CR/FR.

**FR-2. Áp dụng quy tắc**: thêm mục vào `CLAUDE.md`, cạnh mục "Web UI / UX design rules": mọi thay đổi code phải theo `docs/code-standards-rules.md`, đọc trước khi viết code. Skill `/code` (`.claude/skills/code/SKILL.md:24`) trỏ tới file này. Skill `/cr` thêm bước đọc file này khi thiết kế thay đổi code.

**FR-3. Dọn comment toàn repo** cho cả 8 service có dữ liệu khớp, cùng `tests/`, `scripts/`, `infra/`, `docker-compose.yml`, `.env.example`:
- Viết lại mọi comment, docstring và comment SQL chứa tham chiếu ticket hoặc lịch sử thành câu mô tả code hiện tại. Giữ lại lý do kỹ thuật còn đúng, bỏ phần lịch sử và số ticket.
- Nhật ký phiên bản prompt trong `prompt_template_seeds*.go` rút gọn thành mô tả prompt hiện tại làm gì.
- Tham chiếu ADR giữ dạng `See ADR-00NN.` (xem phương án ở dưới).
- Chuỗi runtime và thông báo assert có số CR thì bỏ số CR, giữ nghĩa.
- Đổi tên 3 file test `*_crNNN_test.go` theo nội dung của chúng.

**Tiêu chí chấp nhận**
- Grep `CR-[0-9]+|FR-?[0-9]+|review [A-Z][0-9]+|Unit [0-9]` trên các file code (`*.py *.go *.ts *.tsx *.css *.sql *.yml *.txt`, `.env.example`) trong `services/ tests/ scripts/ infra/` và `docker-compose.yml` trả về 0 dòng. Ngoại lệ duy nhất là giá trị dữ liệu thật mà code phụ thuộc, nếu gặp thì liệt kê trong báo cáo.
- Mọi dòng còn khớp `used to|no longer|previously|regression` đã được đọc và là câu mô tả, không phải lịch sử.
- Không thay đổi hành vi: toàn bộ test của các service vẫn pass; lint sạch như trước.
- `docs/code-standards-rules.md` tồn tại; `CLAUDE.md` và 2 skill đã trỏ tới nó.

**Ngoài phạm vi**
- Tài liệu ngoài code: `aidlc-docs/`, `docs/` (trừ file quy tắc mới), README của từng service, ADR. Lịch sử và lý do vốn thuộc về đó.
- Viết docstring mới cho mọi hàm đang thiếu trong toàn repo. Quy tắc áp dụng cho code mới và code bị sửa; chỉ bổ sung docstring ở những chỗ đang viết lại comment.
- Đổi cấu trúc code, refactor, đổi tên symbol.
- Nội dung prompt gửi cho model và dữ liệu seed.

## Giải pháp đề xuất

### Tham chiếu ADR: giữ hay bỏ?
- **A (khuyên chọn)**: giữ, nhưng chỉ ở dạng một câu `See ADR-00NN.` đặt cạnh code thực thi quyết định đó. ADR giải thích vì sao kiến trúc như hiện nay, tức là "tài liệu", không phải "lý do đã sửa". Bỏ đi thì người đọc mất đường tới quyết định kiến trúc.
- **B**: bỏ hết. Code sạch hơn, nhưng người đọc phải tự tìm ADR tương ứng.

### Cách buộc tuân thủ
- **A (khuyên chọn)**: quy tắc nằm trong `docs/code-standards-rules.md`, được `CLAUDE.md`, `/cr` và `/code` bắt đọc. Cách này giống cách đang áp dụng quy tắc UI, và không dựng lại lớp kiểm tra tự động đã bị gỡ ngày 2026-09-29/30.
- **B**: thêm một target `make comment-check` (grep mẫu cấm, trả lỗi nếu còn). Cách này bắt được lỗi máy móc, nhưng là thêm một bước kiểm tra kiểu `make check` mà CLAUDE.md vừa yêu cầu không dựng lại. Chỉ làm nếu Creator muốn.

### Cách dọn
Viết lại bằng tay theo từng file, không dùng sed hàng loạt: nhiều dòng là docstring nhiều dòng, và câu chứa số CR thường cũng chứa lý do kỹ thuật cần giữ. Ví dụ:

- `naming.camel`, trước: *"camelCase that keeps the capitals already inside a word, so a role the storyboard wrote as `conNguoi` stays `conNguoi` (CR-056: lowering it made the model write a key PALETTE did not have). A word in all capitals is lowered."*
  Sau: *"camelCase that keeps the capitals already inside a word, so a role written as `conNguoi` stays `conNguoi` and matches its PALETTE key. A word in all capitals is lowered."*
- `flow.go:15`, trước: `// Hình minh hoạ (CR-046: promoted from a source="illustrations" sub-state of Code; runs BEFORE Code, whose output Code reads)`
  Sau: `// Hình minh hoạ: runs before Code, which reads its output.`

Làm theo từng service để diff dễ xem và test ngay sau mỗi service.

## Phạm vi

- **Tài liệu và cấu hình agent**: `docs/code-standards-rules.md` (mới), `CLAUDE.md`, `.claude/skills/code/SKILL.md`, `.claude/skills/cr/SKILL.md`.
- **Code, chỉ sửa comment**: authoring-service, orchestrator, web-gui, rendering, video-assembly, publisher, tts, llm-service; `tests/`, `scripts/`, `infra/`, `docker-compose.yml`, `.env.example`. api-gateway không có gì phải sửa.
- **Chuỗi runtime**: 1 câu log ở publisher, cùng các câu khác có số CR mà `/code` gặp khi quét (liệt kê trong báo cáo).
- **Đổi tên file**: 3 file test Go.
- **Contract, DB, migration**: không đổi. Comment SQL trong `db.go` là một phần của chuỗi schema chạy lúc khởi động; bỏ comment không đổi DDL. Nếu schema được so bằng checksum thì phải kiểm tra lại (xem Rủi ro).
- **`graphify affected`**: không áp dụng, vì không symbol nào đổi hành vi. Đổi tên file test không ảnh hưởng import trong Go (cùng package).

## Kế hoạch thực hiện (cho `/code`)

1. Tạo `docs/code-standards-rules.md` theo FR-1, theo phương án ADR và phương án buộc tuân thủ mà Creator chọn.
2. Sửa `CLAUDE.md`: thêm mục `## Code standards and documentation comments` ngay sau mục "Web UI / UX design rules". Sửa `.claude/skills/code/SKILL.md:24` để trỏ tới file quy tắc mới cho mọi thay đổi code. Sửa `.claude/skills/cr/SKILL.md` bước 2 để đọc file này khi thiết kế thay đổi code.
3. Với mỗi service, theo thứ tự llm-service → tts → publisher → video-assembly → rendering → orchestrator → authoring-service → web-gui:
   1. Liệt kê chỗ khớp bằng grep ở mục Tiêu chí chấp nhận, cộng `ADR-|used to|no longer|previously|regression|bug ?fix|was (changed|removed|added)`.
   2. Viết lại từng comment, docstring và comment SQL theo FR-1/FR-3. Giữ lý do kỹ thuật, bỏ lịch sử. Không đổi dòng code nào ngoài comment, trừ chuỗi runtime hoặc thông báo assert chứa số CR.
   3. Chạy test và lint của service đó:
      - Python: `pytest` và `ruff check`.
      - Go: `gofmt -l`, `go vet ./...`, `go test ./...`.
      - web-gui: `npm run lint`, `npx tsc --noEmit`, `npm test`.
   4. Kiểm `git diff` của service: chỉ có dòng comment hoặc chuỗi đã nêu.
4. authoring-service: đổi tên bằng `git mv`, chọn tên theo nội dung test khi đọc file:
   - `summaries_cr051_test.go` → ví dụ `summaries_test.go`, hoặc ghép vào file đã có nếu trùng tên;
   - `project_illustration_cr045_test.go` → ví dụ `project_illustration_planning_test.go`;
   - `code_segment_cr050_test.go` → ví dụ `code_segment_repository_test.go`.
5. publisher `adapters/persistence/db.py:118`: câu log thành `"Migrated the legacy OAuth credential into youtube_accounts"`.
6. `tests/`, `scripts/`, `infra/`, `docker-compose.yml`, `.env.example`: dọn comment như bước 3; chạy `pytest tests/contracts`.
7. Grep toàn phạm vi theo tiêu chí chấp nhận; báo cáo số dòng còn lại (mục tiêu 0) và mọi ngoại lệ.
8. `make graph` để làm mới graph sau khi đổi tên file.

## Kiểm tra

- Test và lint của 8 service như bước 3.3, và `pytest tests/contracts`. Tất cả phải pass như trên `main`; nếu `main` đã có test lỗi sẵn thì ghi rõ.
- Grep ở tiêu chí chấp nhận trả về 0 dòng.
- Rebuild theo chính sách Docker: `docker compose build` và `docker compose up -d` cho 8 service đã đổi (mọi service trừ api-gateway), rồi xác nhận tất cả healthy. Hành vi không đổi, nhưng image chứa mã nguồn đã đổi.
- Kiểm trực tiếp: authoring-service và publisher khởi động và chạy schema/migration không lỗi (xem log); web-gui mở được danh sách video.

## Rủi ro

- **Diff lớn** (khoảng 450 file): dễ xung đột với các branch còn mở (`feature/cr-tts-auto-retry-transient-failures`, `feature/cr-wizard-flow-idea-first`, và các CR chạy song song). Nên giao sớm, và các branch khác rebase sau.
- **Mất thông tin khi viết lại**: có thể lỡ bỏ một lý do kỹ thuật cùng với số CR. Cách giảm: viết lại bằng tay, giữ câu "vì sao code như thế này", review diff theo từng service. Lịch sử đầy đủ vẫn còn trong `aidlc-docs/` và git log.
- **Schema nhúng**: nếu một service so chuỗi schema bằng checksum hoặc version, sửa comment SQL có thể kích hoạt migration lại. Hiện đọc thấy là `CREATE ... IF NOT EXISTS` chạy lúc khởi động; `/code` phải xác nhận trước khi sửa comment SQL. Nếu có checksum thì dừng và hỏi.
- **Đổi tên file test** làm `git log` của file khó theo dõi hơn; `git log --follow` vẫn dùng được.
- **Không có kiểm tra tự động** nếu chọn phương án A: tuân thủ phụ thuộc vào việc agent đọc quy tắc. Phương án B khắc phục điều này nếu Creator muốn.
