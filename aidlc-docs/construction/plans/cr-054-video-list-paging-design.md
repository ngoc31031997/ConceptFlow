# CR-054 — Thiết kế: phân trang danh sách video (phía server)

Trạng thái: **DESIGN — Creator đã chọn phương án B (phân trang ở server)**.

Service bị ảnh hưởng: `orchestrator`, `web-gui`. `api-gateway` không đổi code (proxy đã chuyển nguyên query string, `proxyHandler.js:30-36`).

## 1. Yêu cầu gốc (nguyên văn)

> "phân danh sách video thêm paging"

Trả lời sau đề xuất đầu (A: client, B: server):

> "B chia trang trên servier"

## 2. Hiện trạng

- Màn "Danh sách video" là `VideoListPage` (`services/web-gui/src/pages/VideoListPage.tsx`). Nó gọi `listProjects()` một lần khi mở (`:80-92`), hàm này gọi `GET /v1/projects` (`services/web-gui/src/api/client.ts:270-273`).
- Gateway: `router.get('/v1/projects', proxyHandler(orchestratorClient, ...))` (`services/api-gateway/src/routes/projects.js:52`); proxy chuyển `req.query` sang orchestrator.
- Orchestrator: `Router.handleListProjects` (`internal/adapters/http/router.go:531`) gọi `projects.List` rồi `toProjectListResponse` (`internal/adapters/http/dto.go:372`), trả `{"projects":[…]}` — **toàn bộ** dự án.
- `projects.List` là `projectStoreWithAuthoring.List` (`cmd/orchestrator/main.go:187-210`): đọc `ProjectRepository.List` (`internal/adapters/postgres/project_repository.go:171`, mọi dự án khác `deleting`, `ORDER BY updated_at DESC`, không `LIMIT`), rồi hỏi authoring-service `Summaries(ids)` để điền `topic` và tính lại `flow_step` của bản nháp theo nội dung đã có (bước 3–6).
- `flow_step` và `run_state` không có cột trong DB: tính bằng `domain.FlowStateFor` / `domain.RunStateOf` (`internal/domain/flow.go`), với bản nháp còn cần dữ liệu từ authoring-service. Vì vậy **không lọc được bằng SQL**.
- Lọc hoàn toàn ở client: chip trạng thái (`matches`, `VideoListPage.tsx:35-49`), lọc theo bước (`:94-99`), số đếm trên chip tính trên toàn bộ danh sách (`:206`). Dòng "Bản mới từ “…”" tra tên dự án nguồn trong danh sách đã tải (`nameOf`, `:100-103`).
- Xoá: sau khi saga xoá xong, hàng bị bỏ khỏi state cục bộ (`handleDeleteDone`, `:145-157`; `handleBulkDelete`, `:159-173`).
- "Chọn tất cả" (`:119-121`) thay tập đã chọn bằng toàn bộ `visible`.
- `listProjects()` còn được `JournalPage` (`JournalPage.tsx:100`) và `useRecentProject` (`useRecentProject.ts:37`) dùng; cả hai cần toàn bộ danh sách.
- Hiện DB có 124 dự án. Web-gui chưa có component phân trang.
- Contract `GET /v1/projects` chưa được ghi trong `aidlc-docs/construction/orchestrator-service/low-level-design/interface-contracts.md`.

## 3. Yêu cầu

**FR1.** `GET /v1/projects` nhận thêm query `page`, `page_size`, `filter`, `steps` và trả về đúng một trang, kèm tổng số và số đếm từng nhóm lọc.
**FR2.** Gọi **không có** `page` thì trả như cũ (toàn bộ danh sách, `{"projects":[…]}`), để `JournalPage` và `useRecentProject` không đổi.
**FR3.** Lọc (`filter`, `steps`) chạy ở server, trước khi cắt trang. Luật lọc giống hệt `matches` hiện nay ở client.
**FR4.** Số đếm từng chip (`counts`) tính trên toàn bộ dự án (không theo `steps`, không theo trang), như hiện nay.
**FR5.** `page` vượt quá trang cuối thì server trả trang cuối (và báo `page` thật); danh sách rỗng thì `page=1`.
**FR6.** Mỗi dự án trong trang có thêm `forked_from_topic`: tên dự án nguồn, vì dự án nguồn có thể nằm ở trang khác.
**FR7.** Web: 20 video/trang mặc định, chọn được 10 / 20 / 50. Thanh phân trang dưới danh sách: "Hiển thị 21–40 / 124 video", ô "Mỗi trang", "‹ Trước", số trang (rút gọn "…"), "Sau ›". Chỉ 1 trang thì ẩn.
**FR8.** Web: đổi bộ lọc hoặc số video/trang → về trang 1 và tải lại. Đổi trang → tải trang đó và cuộn lên đầu danh sách.
**FR9.** Web: xoá xong (một hoặc nhiều) → tải lại trang đang xem (hàng từ trang sau dồn lên; hết trang thì server trả trang cuối).
**FR10.** Web: "Chọn tất cả" chỉ chọn / bỏ chọn video của trang đang xem; video đã chọn ở trang khác giữ nguyên, "Xóa đã chọn (N)" xoá mọi video đã chọn.
**FR11.** Web: phản hồi về muộn của một lần tải cũ (bấm trang liên tiếp) không được ghi đè trang mới hơn.
**FR12.** Tham số sai (`page<1`, `page_size` ngoài 1–100, `filter` lạ, `steps` ngoài 1–14 hoặc không phải số) → 400 `{"error":"invalid_query", "message": …}`.

**Tiêu chí chấp nhận**
- `GET /v1/projects?page=7&page_size=20` với 124 dự án → 4 hàng (hoặc tuỳ số dự án thật), `total=124`, `page=7`.
- `GET /v1/projects?page=1&page_size=20&filter=problem` → chỉ dự án `failed`/`cancelled`, `total` = `counts.problem`.
- `GET /v1/projects` (không `page`) → như cũ, không có `total`/`counts`.
- Web với 124 dự án: 7 trang, trang 7 ghi "Hiển thị 121–124 / 124 video"; chọn chip "Lỗi / đã hủy" ở trang 3 → về trang 1.

**Ngoài phạm vi**
- `LIMIT/OFFSET` trong SQL (xem mục 4 — không làm được đúng khi lọc theo bước của bản nháp).
- Nhớ trang qua lần mở sau / đưa trang vào URL. Tìm kiếm theo tên.
- Đổi `JournalPage`, `useRecentProject`.

## 4. Giải pháp

### Lọc và cắt trang ở đâu
`flow_step` của bản nháp phụ thuộc nội dung nằm ở authoring-service (DB khác), `run_state` là hàm Go của `status` + thông báo lỗi. Muốn `LIMIT/OFFSET` bằng SQL phải chép luật `FlowStateFor` sang SQL và kéo dữ liệu authoring sang DB orchestrator — hai bản luật dễ lệch nhau, đúng điều `flow.go` được viết để tránh. Nên:

- Orchestrator vẫn đọc toàn bộ bản tóm tắt nhẹ (đang làm), điền topic/flow_step như cũ, rồi **lọc, đếm và cắt trang trong Go** trước khi trả về.
- Lợi ích thật: trình duyệt chỉ nhận và vẽ một trang; bộ lọc, số đếm và luật lọc nằm một chỗ ở server, cạnh `FlowStateFor`.
- Chi phí DB/authoring không đổi so với hôm nay (một truy vấn nhẹ + một lần gọi `Summaries`). Khi số dự án lên hàng chục nghìn thì cần CR riêng để lưu `flow_step`/`run_state` thành cột.

### Orchestrator
- **Domain** — file mới `internal/domain/project_list.go` (hàm thuần):
  ```go
  type ListFilter string // "all" | "running" | "waiting" | "problem" | "done"
  type ProjectListQuery struct { Page, PageSize int; Filter ListFilter; Steps []int }
  type ProjectListCounts struct { All, Running, Waiting, Problem, Done int }
  type ProjectPage struct {
      Projects []ProjectSummary; Total, Page, PageSize int; Counts ProjectListCounts
  }
  func ParseListFilter(s string) (ListFilter, error)   // "" → all
  func (f ListFilter) Matches(s ProjectSummary) bool   // luật y hệt matches() ở web
  func PageProjects(all []ProjectSummary, q ProjectListQuery) ProjectPage
  ```
  Luật `Matches` (chép từ `VideoListPage.tsx:35-49`): running = `RunState==running`; problem = `failed|cancelled`; done = `FlowStep>=FlowResult && RunState!=failed`; waiting = `RunState==idle && 0<FlowStep<FlowResult`; all = mọi dự án.
  `PageProjects`: đếm `Counts` trên `all`; lọc theo `Filter` và `Steps` (rỗng = mọi bước); `Total` = số sau lọc; `pages = max(1, ceil(Total/PageSize))`; kẹp `Page` vào `[1, pages]`; cắt; giữ thứ tự đầu vào (đã `updated_at DESC`).
- **ProjectSummary** thêm `ForkedFromTopic string`. `PageProjects` điền nó cho các dự án trong trang từ map `project_id → topic` của `all` (nguồn đã xoá → rỗng). Chế độ không phân trang cũng điền (dùng chung hàm `FillForkedFromTopics(all)`).
- **HTTP** — `handleListProjects`:
  - Không có `page` → như cũ (thêm `forked_from_topic`).
  - Có `page` → parse `page`, `page_size` (mặc định 20), `filter`, `steps` (chuỗi phân tách bằng dấu phẩy); sai → 400 `invalid_query`. Gọi `projects.List`, rồi `domain.PageProjects`, trả:
  ```json
  { "projects": [...], "total": 124, "page": 1, "page_size": 20,
    "counts": { "all": 124, "running": 2, "waiting": 30, "problem": 10, "done": 60 } }
  ```
  `projectListResponse` thêm các trường `total`, `page`, `page_size`, `counts` với `omitempty`/con trỏ để chế độ cũ không đổi JSON.
- Không đổi `projectStore` interface, repository, DB, migration, saga.

### Web-gui
- `types/index.ts`: `ProjectSummary.forked_from_topic?: string`; kiểu mới `ProjectListFilter`, `ProjectPage { projects, total, page, page_size, counts }`.
- `api/client.ts`: hàm mới `listProjectsPage({ page, pageSize, filter, steps }): Promise<ProjectPage>` gọi `/v1/projects?page=…&page_size=…&filter=…&steps=…` (bỏ `steps` khi rỗng). `listProjects()` giữ nguyên.
- `utils/pagination.ts`: `PAGE_SIZES = [10, 20, 50]`, `pageCount(total, size)`, `pageItems(current, count)` (trang đầu, cuối, hiện tại ±1; khoảng trống ≥ 2 trang thành `"…"`).
- `components/ui/Pagination.tsx` + `.module.css` (xuất qua `components/ui/index.ts`): component không giữ state. Props `page, pageCount, total, pageSize, pageSizes, label, onPageChange, onPageSizeChange`. Neubrutalism (`docs/ux-ui-design-rules.md` §2): token `--border-w`, `--shadow-sm`, `--radius`; trang hiện tại nền `--accent`, `aria-current="page"`; Trước/Sau `disabled` ở hai đầu; `<nav aria-label="Phân trang">`. Thứ tự trái → phải: "Hiển thị …", "Mỗi trang", các nút trang (§1).
- `VideoListPage`:
  - State: `data: ProjectPage | null`, `page`, `pageSize` (20), `filter`, `stepFilter`, `loading`.
  - `refetch` gọi `listProjectsPage` với state hiện tại; dùng `requestSeq` (ref tăng dần) để bỏ phản hồi cũ (FR11); đặt `page` bằng `data.page` server trả (FR5). Effect chạy lại khi `page`, `pageSize`, `filter`, `stepFilter` đổi.
  - Bỏ `matches` và lọc client. Số trên chip lấy từ `data.counts`. Hàng vẽ từ `data.projects`. Đang tải lại thì giữ trang cũ trên màn (không nhấp nháy về "Đang tải…").
  - "Chưa có video nào." khi `data.counts.all === 0`; "Không có video nào ở nhóm này." khi `data.projects` rỗng.
  - Lineage: `project.forked_from_topic || project.forked_from.slice(0, 8)`; bỏ `nameOf`.
  - Đổi chip / lọc bước / "Bỏ chọn" / "Mỗi trang" → `setPage(1)` trong handler.
  - `handleDeleteDone` và `handleBulkDelete`: bỏ id khỏi `selected` như cũ, rồi `refetch()` thay vì lọc state cục bộ (FR9).
  - `allSelected` / `toggleSelectAll` theo `data.projects`: thêm hoặc bớt id của trang (FR10).
  - `Pagination` dưới `<Card>` khi `pageCount(total, pageSize) > 1`. Đổi trang: `setPage(n)` rồi `listTopRef.current?.scrollIntoView?.({ block: "start", behavior: "smooth" })`.

## 5. Phạm vi

| Service | File | Thay đổi |
|---|---|---|
| orchestrator | `internal/domain/project_list.go` (mới) + `project_list_test.go` | lọc, đếm, cắt trang, `forked_from_topic` |
| orchestrator | `internal/domain/project.go` | `ProjectSummary.ForkedFromTopic` |
| orchestrator | `internal/adapters/http/router.go` (`handleListProjects`) | parse query, gọi `PageProjects`, 400 khi sai |
| orchestrator | `internal/adapters/http/dto.go` (`projectListResponse`, `projectSummaryResponse`, `toProjectListResponse`) | trường mới |
| orchestrator | `internal/adapters/http/router_test.go` | test chế độ trang + 400 + chế độ cũ |
| web-gui | `src/types/index.ts`, `src/api/client.ts` | kiểu + `listProjectsPage` |
| web-gui | `src/utils/pagination.ts`, `src/components/ui/Pagination.tsx` + css, `src/components/ui/index.ts` | mới |
| web-gui | `src/pages/VideoListPage.tsx` | dùng trang từ server |
| web-gui | `tests/utils/pagination.test.ts`, `tests/components/ui/Pagination.test.tsx` (mới), `tests/api/client.test.ts`, `tests/pages/VideoListPage.test.tsx` | test |
| docs | `aidlc-docs/construction/orchestrator-service/low-level-design/interface-contracts.md` | ghi contract `GET /v1/projects` |

- Không đổi DB, migration, RabbitMQ, api-gateway, authoring-service.
- `graphify affected "listProjects()"`: `VideoListPage`, `JournalPage`, `useRecentProject`. Hai nơi sau vẫn dùng `listProjects()` (không `page`) → không đổi hành vi (FR2).
- `projects.List` (router.go:94) chỉ được `handleListProjects` gọi; `projectStoreWithAuthoring.List` không đổi.

## 6. Kế hoạch thực hiện

1. **Orchestrator domain**: thêm `ForkedFromTopic string` vào `ProjectSummary` (`internal/domain/project.go:500`). Tạo `internal/domain/project_list.go` với `ListFilter` (+ hằng `ListAll/ListRunning/ListWaiting/ListProblem/ListDone`), `ParseListFilter`, `(ListFilter).Matches`, `ProjectListQuery`, `ProjectListCounts`, `ProjectPage`, `FillForkedFromTopics([]ProjectSummary)`, `PageProjects` như mục 4. Hằng `MaxProjectPageSize = 100`, `DefaultProjectPageSize = 20`.
2. **Test domain** `internal/domain/project_list_test.go` (table-driven): `Matches` cho từng filter với các cặp `FlowStep/RunState` (gồm done + failed ở bước 13 → không done; idle bước 13 → không waiting); `PageProjects`: 45 dự án, trang 3 cỡ 20 → 5 hàng, `Total=45`; `Page=9` → kẹp về 3; danh sách rỗng → `Page=1, Total=0`; lọc `Steps=[10]` + `Filter=problem`; `Counts` không phụ thuộc `Steps`; `ForkedFromTopic` điền từ dự án ở trang khác, rỗng khi nguồn không còn; `ParseListFilter("x")` lỗi.
3. **HTTP**: trong `dto.go` thêm `ForkedFromTopic string \`json:"forked_from_topic,omitempty"\`` vào `projectSummaryResponse`; `projectListResponse` thêm `Total *int \`json:"total,omitempty"\``, `Page *int`, `PageSize *int \`json:"page_size,omitempty"\``, `Counts *projectListCountsResponse \`json:"counts,omitempty"\``; thêm `toProjectPageResponse(domain.ProjectPage)`. Trong `router.go` `handleListProjects`: nếu `r.URL.Query().Get("page") == ""` → `FillForkedFromTopics` + response cũ; ngược lại parse (`strconv.Atoi`; `steps` tách dấu phẩy, bỏ phần rỗng), lỗi → `writeJSON(w, 400, {"error":"invalid_query","message":…})`, rồi `PageProjects` → `toProjectPageResponse`.
4. **Test HTTP** `router_test.go`: giữ `TestHandleListProjects_OK`; thêm `_Paged` (25 dự án giả, `?page=2&page_size=10` → 10 hàng, `total=25`, `page=2`, `counts.all=25`), `_PagedFilter` (`filter=problem`), `_InvalidQuery` (`page=0`, `page_size=500`, `filter=x`, `steps=abc`, `steps=15` → 400), `_LegacyHasNoPaging` (không `page` → JSON không có `total`).
5. **Contract**: thêm mục `### GET /v1/projects` vào `aidlc-docs/construction/orchestrator-service/low-level-design/interface-contracts.md` (2 chế độ, query, response, 400).
6. **Web types/client**: `types/index.ts` thêm `forked_from_topic?: string` vào `ProjectSummary`, `ProjectListFilter`, `ProjectListCounts`, `ProjectPage`. `api/client.ts` thêm `listProjectsPage` (dùng `URLSearchParams`). Test trong `tests/api/client.test.ts`: URL đúng (có/không `steps`), trả nguyên `ProjectPage`.
7. **Web pagination**: tạo `src/utils/pagination.ts` + `tests/utils/pagination.test.ts` (`pageCount(0,20)=1`, `pageCount(124,20)=7`; `pageItems(1,1)=[1]`, `(1,3)=[1,2,3]`, `(4,7)=[1,"…",3,4,5,"…",7]`, `(1,7)=[1,2,"…",7]`, `(7,7)=[1,"…",6,7]`, `(3,7)=[1,2,3,4,"…",7]`). Tạo `Pagination.tsx` + css, `data-testid`: `pagination`, `pagination-summary`, `pagination-prev`, `pagination-next`, `pagination-page-<n>`, `pagination-size`; export trong `components/ui/index.ts`. Test `tests/components/ui/Pagination.test.tsx`: dòng tóm tắt, `aria-current`, Trước/Sau tắt ở hai đầu, click gọi callback đúng số, đổi select gọi `onPageSizeChange(50)`.
8. **VideoListPage** theo mục 4 (Web-gui). Xoá `matches`, `nameOf`.
9. **Test VideoListPage** (`tests/pages/VideoListPage.test.tsx`): cập nhật mock `fetch` để trả dạng `ProjectPage` cho URL có `page=` (các test cũ dùng helper chung trả `{projects, total, page:1, page_size:20, counts}`); test "filters by what needs attention" đổi thành kiểm URL gọi có `filter=problem`/`running`/`done` và vẽ đúng hàng server trả; thêm: số trên chip lấy từ `counts`; thanh phân trang với `total=45` (bấm "Sau" → gọi `page=2`); chip lọc ở trang 3 → gọi `page=1`; "Mỗi trang" 50 → gọi `page_size=50&page=1`; "Chọn tất cả" chỉ chọn hàng trang hiện tại; phản hồi cũ về muộn bị bỏ; lineage dùng `forked_from_topic`; xoá xong → gọi lại danh sách.

## 7. Kiểm tra

- `cd services/orchestrator && go vet ./... && go test ./...`
- `cd services/web-gui && npx tsc -b && npx vitest run`
- Rebuild: `docker compose build orchestrator web-gui && docker compose up -d orchestrator web-gui`; orchestrator phải healthy.
- Kiểm trực tiếp: `curl 'localhost:<gateway>/v1/projects?page=1&page_size=20'` (có `total`, `counts`), `curl .../v1/projects` (như cũ), `?page=0` → 400; màn "Danh sách video" với 124 dự án thật: đổi trang, đổi "Mỗi trang", lọc rồi phân trang, xoá một video; màn Nhật ký vẫn hiện tên dự án.

## 8. Rủi ro

- Không mất dữ liệu, không đổi DB. Contract mở rộng tương thích ngược (chế độ cũ giữ nguyên JSON, chỉ thêm `forked_from_topic`).
- Orchestrator vẫn đọc toàn bộ bản tóm tắt và gọi authoring `Summaries` cho mọi dự án ở mỗi lần đổi trang (lý do ở mục 4). Với vài trăm dự án không đáng kể; hàng chục nghìn thì cần CR lưu `flow_step`/`run_state` thành cột để `LIMIT/OFFSET` bằng SQL.
- authoring-service lỗi: như hôm nay, danh sách vẫn trả (không topic, bản nháp ở bước mặc định) — lúc đó lọc theo bước của bản nháp có thể lệch, giống hành vi hiện tại.
- Thay đổi hành vi: "Chọn tất cả" chỉ chọn trang đang xem (trước là mọi video đang lọc).
- Dữ liệu thay đổi giữa hai lần đổi trang (dự án mới cập nhật nhảy lên đầu) có thể làm một dự án xuất hiện ở hai trang hoặc bị lướt qua — chấp nhận với phân trang theo offset.
