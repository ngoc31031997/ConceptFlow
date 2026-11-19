# CR-116 · Danh sách task

Thiết kế: [cr-116-visuals-to-rendering-design.md](cr-116-visuals-to-rendering-design.md) · ADR: [ADR-0050](../../decisions/ADR-0050-rendering-owns-illustrations-kit-is-reference-data.md)
Worktree: `.claude/worktrees/feature-cr-116-visuals-to-rendering` (nhánh `feature/cr-116-visuals-to-rendering`).

## Luật chung cho mọi task
- Đọc `CLAUDE.md`, `docs/code-standards-rules.md` (và `docs/ux-ui-design-rules.md` nếu sửa màn hình) trước khi code. Dùng graphify để định vị code.
- Chỉ sửa file thuộc phạm vi task; không sửa file của task khác đang chạy song song.
- **Không commit, không push, không merge** — Creator duyệt rồi mới commit.
- Test giả lời AI, không gọi model thật. Dọn mọi dữ liệu test tạo ra.
- Xong: chạy test của service, `scripts/worktree.sh rebuild <service>` cho service đã sửa, xác nhận healthy.
- Không làm được / có điểm mơ hồ đổi hành vi → dừng và báo, không đoán, không viết code giả.
- **Không xoá dữ liệu Creator** (bảng, dự án, `/shared`) — đó là T7.
- Báo cáo: file đã sửa, test (số pass/fail, nguyên văn lỗi nếu fail), rebuild, dữ liệu test đã dọn, việc chưa làm.

## Bảng task

| # | Task | Service | Phụ thuộc | Song song với |
|---|---|---|---|---|
| T1 | Kit trong phong cách tham chiếu | visual-service, web-gui (ReferenceLibraryPage) | — | T2, T3 |
| T2 | rendering nhận `kit` object | rendering (Python + Remotion) | — | T1, T3 |
| T3 | orchestrator + llm-service chuyển `kit` | orchestrator, llm-service | hợp đồng §2 | T1, T2 |
| T4 | Thư viện hình + bộ vẽ AI ở rendering | rendering | T2 | — |
| T5 | api-gateway + web-gui thư viện hình | api-gateway, web-gui | T1, T4 | T6 |
| T6 | Gỡ phần hình khỏi authoring | authoring-service | T4 | T5 |
| T7 | Dọn dữ liệu (không hoàn tác) | mọi DB, `/shared` | T1–T6 deploy | — (session chính, Creator xác nhận) |

## T1 · Kit trong phong cách tham chiếu (thiết kế §2 "visual-service")
- Domain `ReferenceKit` trong `reference_style.go`, lưu ở `detail.kit`; kiểm tra: đủ trường như `KitTokens` (`services/rendering/remotion_project/src/conceptflow-mini/kits.ts`), số 0..1 trừ `dof` (px ≥ 0), màu `#RRGGBB`, `focus` ∈ {sky, far, mid, near, figures}. Kit trống được phép.
- Route nội bộ `GET /internal/v1/visual/{project_id}/kit` → `{ref_id, kit}` (404 chưa chọn phong cách, 409 phong cách chưa có kit) và `GET /internal/v1/reference-styles/{id}/kit`.
- Route admin sửa phong cách nhận/trả `kit`.
- web-gui `ReferenceLibraryPage`: thẻ "Bộ thông số vẽ" sửa từng số. Nút "Vẽ thử" để lại cho T5 (cần rendering).
- KHÔNG ghi kit cho R12 trong DB (T7 làm).
- Cập nhật `docs/contracts/visual-service-v1.md`.

## T2 · rendering nhận `kit` object (thiết kế §2 "rendering")
- Thêm `domain/kit.py` (cùng luật T1); bỏ `domain/kits.py` phần `load_kits`/`require_kit` (giữ `load_skeletons`), `kits.json`, bảng `KITS`/`KIT_VERSIONS` trong `kits.ts`, `GET /v1/kits`.
- Remotion `currentKit()` đọc input prop `kit` (object), ném lỗi rõ nếu thiếu/sai.
- `/v1/stills`, `/v1/illustrations/preview`, `/v1/check/remotion`, lệnh `render_scenes`: `kit_key` → `kit`; hash cache khung tĩnh tính trên nội dung kit.
- Test chứng minh: kit có đúng số của `phatsang` hiện nay vẽ ra ảnh giống bản cũ (ghi số cũ vào fixture test trước khi xoá).
- Cập nhật `docs/contracts/` (rendering interface-contracts, README hàng 47–48).

## T3 · orchestrator + llm-service chuyển `kit`
- orchestrator: `KitKeyPort` → `KitPort` gọi visual `GET /internal/v1/visual/{project_id}/kit`; payload `render_scenes.kit` thay `kit_key`; lỗi 404/409 thành lỗi bước rõ ràng.
- llm-service: `kit_key` ở `/v2/code/*`, `checker.py`, `run.py` → `kit` (dict), chuyển nguyên cho `/v1/check/remotion`. Không bỏ route.
- Cập nhật `authoring-llm-code-v2.md`.

## T4 · Thư viện hình + bộ vẽ AI ở rendering (thiết kế §3, §4)
- Viết lại bằng Python, hành vi 1:1 với authoring (`internal/application/illustrations*.go`, `illustration_drawer.go`, `illustration_backup.go`, `internal/adapters/http/illustrations.go` và test của chúng), trừ: `kit_key` → `ref_id`; ảnh ở `/shared/illustrations/{id}/v{version}.png|.gif`.
- Bảng `illustration_folders`, `illustrations` trong bootstrap `db.py`.
- Bộ vẽ AI gọi llm-service; prompt owner `rendering`, role `illustration_drawer`, `illustration_backdrop_drawer`, đẩy lúc khởi động.
- **Prompt: KHÔNG tự viết nội dung.** Soạn bản nháp nguyên văn (prompt vẽ hiện tại + luật lấy từ phong cách tham chiếu thay các file `phatsang_*`) vào `aidlc-docs/construction/plans/cr-116-prompts-draft.md`, rồi dừng phần seed và báo để Creator duyệt. Code còn lại làm xong được.
- Cập nhật compose (env `LLM_SERVICE_URL`, `VISUAL_SERVICE_URL`), `docs/contracts/illustration-library-v1.md`, `prompt-library-v1.md`.

## T5 · api-gateway + web-gui
- Gateway: `routes/illustrations.js` → `RENDERING_URL`; xoá `routes/visualLanguages.js`, route hình theo dự án.
- web-gui: `IllustrationLibraryPage` chọn phong cách tham chiếu (mặc định R12) thay kit; xoá `VisualLanguagePage`, menu, phần hình theo dự án, `kit_key` trong `client.ts`; nút "Vẽ thử" ở thẻ kit (T1) gọi rendering.

## T6 · Gỡ khỏi authoring (thiết kế §5)
- Xoá code/route/prompt liệt kê ở §5; bỏ lệnh tạo 6 bảng và cột trong `db.go`. KHÔNG chạy DROP (T7).

## T7 · Dọn dữ liệu (thiết kế §6)
- Session chính làm, sau khi T1–T6 đã deploy: in số dòng sẽ xoá, Creator xác nhận, rồi xoá; ghi kit `phatsang` vào R12.
