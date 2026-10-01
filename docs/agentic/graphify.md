# graphify: đồ thị tri thức của code cho agent

> **2026-09-29:** lớp agentic (skills, agents, `/cr-review`, mục "Graph impact") đã bị gỡ; graphify vẫn giữ và là cách **đầu tiên** để đọc codebase (luật trong `CLAUDE.MD`). Các đoạn nhắc tới skill và brief review bên dưới chỉ còn giá trị lịch sử.

> Cập nhật: 2026-09-29. Người đọc: Creator và agent làm CR / sửa bug trong repo này.
> Công cụ: [graphify](https://github.com/Graphify-Labs/graphify) (gói PyPI `graphifyy`). Quyết định: D12 trong `implementation-audit.md` §11 (đã xoá 2026-09-30, xem git history).

## 1. Để làm gì

graphify phân tích repo bằng tree-sitter (AST) và dựng một đồ thị: node là file, hàm, class, heading markdown; cạnh là import, gọi hàm, tham chiếu, kế thừa, link. Agent hỏi đồ thị để **định hướng** thay vì grep và đọc hàng chục file:

- Hành vi X nằm ở đâu, ai gọi hàm Y, đổi file Z thì ảnh hưởng tới đâu.
- Trong `/cr-review`, brief gửi cho 3 agent review có thêm mục **Graph impact**: các file ngoài diff phụ thuộc trực tiếp vào file bị đổi, để `reviewer` biết nên mở caller nào khi xét rủi ro regression.

Đồ thị là **bản đồ, không phải bằng chứng**: agent vẫn phải đọc code trước khi sửa hay kết luận nguyên nhân.

## 2. Cách tích hợp (quyết định của Creator, 2026-09-29)

| Chủ đề | Chọn | Lý do |
|---|---|---|
| Hook của graphify cho Claude Code | **Không cài** (`graphify claude install` không dùng) | Hook đó chèn lời nhắc "MANDATORY" vào *mọi* lệnh Read/Grep/Glob/Bash, kể cả của agent review chỉ có Read/Grep và không chạy được graphify. Thay vào đó, hướng dẫn nằm trong `CLAUDE.MD` và các skill |
| Phạm vi | **Chỉ phân tích cục bộ** (code + cấu trúc markdown bằng AST) | Không cần API key, không gửi gì ra ngoài, không tốn token. Không dùng trích xuất ngữ nghĩa bằng LLM |
| Lưu trữ | `graphify-out/` **nằm trong `.gitignore`** | `graph.json` khoảng 15 MB, dễ xung đột khi merge, và repo có origin public |
| Giữ đồ thị mới | `make graph-hooks`: hook git `post-commit` / `post-checkout` của graphify tự cập nhật đồ thị (chạy nền) | Ngoài ra có `make graph` để cập nhật tay; `review-prep.sh` luôn tự cập nhật cho tree đang review |

## 3. Cài đặt (mỗi máy, một lần)

```bash
brew install pipx
pipx install --python python3.12 graphifyy   # cần Python >= 3.10; python3 hệ thống của macOS là 3.9
pipx ensurepath                              # thêm ~/.local/bin vào PATH (mở terminal mới sau đó)
make graph                                   # dựng graphify-out/ lần đầu (khoảng 10 giây)
make graph-hooks                             # hook git cho clone này
```

`make graph-hooks` ghi vào `.git/hooks/post-commit` và `.git/hooks/post-checkout` (nối thêm, không đè hook có sẵn) và đăng ký `merge.graphify` trong `git config`. graphify còn ghi dòng merge driver cho `graph.json` vào `.gitattributes`; vì đồ thị không được commit, `scripts/graph.sh` bỏ dòng đó (giữ nguyên `.gitattributes` đã commit, hoặc xoá file nếu chính lệnh này vừa tạo ra nó). Tắt hook cho một lệnh: `GRAPHIFY_SKIP_HOOK=1 git commit …`. Gỡ: `graphify hook uninstall`.

Mỗi worktree có `graphify-out/` riêng: `scripts/worktree.sh add` dựng sẵn khi tạo worktree; sau đó chạy `make graph` trong chính worktree đó.

## 4. Dùng hằng ngày

| Lệnh | Trả lời câu hỏi |
|---|---|
| `graphify query "<câu hỏi>"` | Phần đồ thị liên quan tới câu hỏi (BFS, giới hạn `--budget` token) |
| `graphify explain "<symbol>"` | Symbol này là gì, nối với những gì |
| `graphify path "<A>" "<B>"` | Đường đi ngắn nhất giữa hai symbol (chuỗi gọi) |
| `graphify affected "<symbol>"` | Ai phụ thuộc vào symbol này (duyệt ngược, `--depth 2`) |
| `graphify god-nodes` | Các node nhiều cạnh nhất (trung tâm kiến trúc) |
| `graphify-out/GRAPH_REPORT.md` | Tổng quan: cộng đồng (cụm), node trung tâm, commit dựng đồ thị |
| `graphify-out/graph.html` | Xem đồ thị trên trình duyệt |

Tên trùng ở nhiều file (ví dụ `progressHandler()`) thì graphify báo *Ambiguous* và gợi ý dạng `<đường dẫn>::<symbol>` hoặc id của node; dùng lại dạng đó.

Skill đã gắn sẵn:

- `/cr-start`: bước phân tích phạm vi dùng `graphify query` / `affected` để liệt kê service và file *ứng viên*, rồi xác nhận bằng đọc code.
- `/fix-bug`: tìm đường đi của lỗi bằng `query` / `explain` / `path`; sau khi đổi hành vi một hàm, `affected` để rà các caller.
- `/cr-review`: `scripts/review-prep.sh` chạy `make graph` cho tree đang review rồi `scripts/hooks/graph_impact.py` ghi mục **Graph impact** vào brief. Agent `reviewer` dùng danh sách này để chọn caller cần mở.

## 5. Mục "Graph impact" trong brief

`scripts/hooks/graph_impact.py` đọc `graphify-out/graph.json` và, với mỗi file bị đổi, liệt kê các file **ngoài diff** có cạnh phụ thuộc (import, gọi, tham chiếu, kế thừa…) trỏ vào nó; tối đa 15 file mỗi dòng, kèm tổng số. Chỉ tính cạnh `EXTRACTED` (xác định từ AST); cạnh `INFERRED` là graphify đoán theo tên (ví dụ heading tài liệu có chữ "node" bị nối với hàm `node()`) nên bị bỏ.

- **Go**: một `import ".../domain"` là import cả package, nhưng graphify gắn cạnh đó vào một file bất kỳ của package (có khi là file `_test.go`). Script tính các import này theo **package**: dòng `Go package <thư mục>/ (imported as a whole)` liệt kê file import package có file (không phải test) bị đổi.
- **An toàn cho brief**: brief được hook `record_review.py` tin nguyên văn, nên script chỉ in đường dẫn file được git theo dõi và gồm ký tự an toàn; không in nhãn node hay chữ lấy từ code.
- **Không đoán**: nếu đồ thị không có, không đọc được, hoặc `built_at_commit` khác HEAD, mục này ghi `Graph impact: not available (<lý do>)` thay vì liệt kê caller của một tree khác. Review vẫn chạy bình thường.

## 6. Giới hạn

- Chỉ có cạnh **tĩnh**. Tin nhắn RabbitMQ, lời gọi HTTP giữa các service, truy cập DB, cấu hình Docker không nằm trong đồ thị. Với các phụ thuộc đó, xem `docs/contracts/` và ADR.
- Một số cạnh là `INFERRED` (khoảng 7%, graphify đoán theo tên). `query` là tìm theo từ khoá + BFS, nên kết quả có thể lẫn node không liên quan; tăng `--budget` hoặc hỏi cụ thể hơn.
- `affected` trên CLI không nhận đường dẫn file; muốn biết ai phụ thuộc vào cả một file thì xem mục Graph impact hoặc hỏi theo symbol.
- Hook git chạy nền sau commit/checkout; ngay sau đó đồ thị có thể chưa kịp cập nhật. So `built_at_commit` trong `graph.json` (hoặc "Graph Freshness" trong `GRAPH_REPORT.md`) với `git rev-parse HEAD`; lệch thì chạy `make graph`.
- Trích xuất ngữ nghĩa bằng LLM (tài liệu, PDF, ảnh) **không bật**. Muốn bật cần API key, và nội dung tài liệu sẽ được gửi cho nhà cung cấp: cần Creator duyệt riêng.

## 7. File

```
.graphifyignore                 đường dẫn graphify không bao giờ index (secrets/, data/, media/), cộng với .gitignore
graphify-out/                   đồ thị cục bộ (git-ignored): graph.json, GRAPH_REPORT.md, graph.html, cache/
scripts/graph.sh                make graph (graphify update . --force) | make graph-hooks
scripts/hooks/graph_impact.py   mục "Graph impact" của brief review
scripts/hooks/test_graph_impact.py  test (chạy trong make check khi scripts/hooks/ đổi, và make check-all)
```
