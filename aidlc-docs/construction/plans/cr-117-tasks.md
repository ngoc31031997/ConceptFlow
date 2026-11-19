# CR-117 · Danh sách task

Thiết kế: [cr-117-storyboard-design.md](cr-117-storyboard-design.md) · ADR: [ADR-0051](../../decisions/ADR-0051-storyboard-split-visual-service-and-rendering.md) · Prompt đã duyệt: [cr-117-prompts-draft.md](cr-117-prompts-draft.md) · Mockup: [cr-117-storyboard-mockup.html](cr-117-storyboard-mockup.html) (https://claude.ai/artifact/5D3n6WfATdcf76picCqoLt, bản 3)
Worktree: `.claude/worktrees/feature-cr-117-storyboard` (nhánh `feature/cr-117-storyboard`).

## Luật chung cho mọi task
- Đọc `CLAUDE.md`, `docs/code-standards-rules.md` (và `docs/ux-ui-design-rules.md` nếu sửa màn hình) trước khi code. Dùng graphify để định vị code. Đường dẫn luôn tuyệt đối trong worktree.
- Chỉ sửa file thuộc phạm vi task; các task chạy song song dùng chung worktree.
- **Không commit, không push, không merge.** Creator duyệt rồi mới commit.
- Test giả lời AI, không gọi model thật, không gọi llm-service thật. Dọn mọi dữ liệu test (hàng DB, file `/shared`, container tạm).
- **Không xoá dữ liệu Creator.** Dự án đang giữ trong DB là `335da643-10dc-42d9-85b4-7f3925f4646e`.
- Xong: chạy test và lint của service, `scripts/worktree.sh rebuild <service>` cho service đã sửa, xác nhận healthy.
- Không làm được hoặc có điểm mơ hồ đổi hành vi → dừng và báo, không đoán, không viết code giả, không giả thành công.
- Báo cáo bằng tiếng Việt: file đã sửa, test (số pass/fail, nguyên văn lỗi nếu fail), rebuild, dữ liệu test đã dọn, việc chưa làm.
- Prompt: chép **đúng nguyên văn** từ `cr-117-prompts-draft.md`, không sửa chữ.

## Bảng task
| # | Task | Service | Phụ thuộc | Đợt |
|---|---|---|---|---|
| S0 | Spike: dựng khung nhiều tài sản | rendering (Node: `illustration_preview.mjs`, `host.tsx`) | — | 1 |
| S1 | Tài sản dự án ở rendering + vẽ + đưa vào thư viện | rendering (Python) | — | 1 |
| S3 | Domain, máy kiểm, đọc chặt reply AI, hash/stale, bản chốt, seed prompt | visual-service | — | 1 |
| S2 | Dựng khung + `storyboard_shots` + `storyboard_versions` + route nội bộ | rendering | S0, S1 | 2 |
| S4 | Desk, viết tài sản/shot/rà soát, HTTP, kho Postgres, client rendering | visual-service | S1, S2, S3 | 2 |
| S6 | api-gateway + web-gui trang `/projects/:id/storyboard` theo mockup bản 3 | api-gateway, web-gui | S4 | 3 |
| S7 | `flow.ts` bước 11 (bỏ `comingSoon`, `screen: "storyboard"`, xong khi đã chốt) + bước đã chốt trong `ProjectFlowContext` | web-gui | S6 | 3 |
| S8 | Hợp đồng `storyboard-v1.md`, purge khi xoá dự án, smoke test | tất cả | S1–S7 | 3 |

## Chi tiết
- **S0:** theo thiết kế §5. Mở rộng host xem trước hiện có (cùng tiến trình Node) bằng lệnh `frame` nhận nhiều tài sản (`js` đã biên dịch bằng esbuild, đặt theo ô lưới `position` và `scale`), nền màu hoặc tài sản backdrop, chữ ở dải dưới. Chứng minh bằng một khung hai tài sản + chữ vẽ bằng kit R12 (lấy `services/rendering/tests/fixtures/phatsang_kit.json`), đọc ảnh ra xem có đúng bố cục. **Nếu không đạt, dừng và báo.** Không đổi hành vi xem trước một hình hiện có.
- **S1:** thiết kế §2.1, §3 bước 3–5. Bảng `project_assets` (bootstrap trong `db.py`), repository, use case, route nội bộ `/internal/v1/projects/{id}/assets…` (liệt kê, tạo, sửa, xoá, `draw`, `redraw`, `approve`, `unapprove`). Vẽ bằng bộ vẽ AI có sẵn, mô tả ghép từ `portrayal` + `must_not_imply` + `palette_tokens` + `expressions` + hạng; không đổi prompt vẽ đã duyệt. Đưa vào thư viện khi duyệt nếu `to_library`. Cấp mã `AST-<TÊN>-<số>` từ bộ đếm theo dự án, không dùng lại.
- **S2:** thiết kế §5, §7. `storyboard_shots`, `storyboard_versions`, `POST …/storyboard/frames` (vẽ, ghi PNG ở `/shared/{project}/storyboard/SH-xxx.png`, cache theo `spec_hash`), `POST …/storyboard/versions` (ghi bản chốt, `storyboard.json`, sao ảnh sang `v{n}/`), `GET …/versions/latest`, `DELETE …/storyboard`. Kiểm từ vựng đóng của vị trí và góc máy.
- **S3:** thiết kế §2.2, §3, §4, §7, §10. Domain Go: kiểu Shot, từ vựng đóng, máy kiểm `m-shot-*`, đọc chặt reply (tài sản, shot, rà soát), mã `SH-` không dùng lại, hash và stale, dựng tài liệu chốt, điều kiện chốt; `storyboard_prompt_seeds.go` với ba prompt đúng nguyên văn.
- **S4:** thiết kế §3, §6. Desk, người viết tài sản/shot, người rà soát, `review_units` (`storyboard_assets`, `storyboard_unit`, `storyboard_review`), cột `storyboard_model`, kho phiên bản không cần (bản chốt ở rendering), client rendering (`RENDERING_URL`), HTTP công khai và nội bộ, chế độ AI ngoài (`X-External-AI`) cho ba lượt AI chữ. Dùng lại khung desk của bước 10.
- **S6:** theo mockup bản 3 và `docs/ux-ui-design-rules.md`: thẻ "Chạy AI" (model, DeepSeek, hàng đợi), ba tab, gập mục đã duyệt, hộp thoại sửa, hộp thoại AI ngoài, chế độ lưới.
- **S8:** `docs/contracts/storyboard-v1.md`, cập nhật `visual-service-v1.md`, `illustration-library-v1.md`, README hợp đồng; xoá dữ liệu bước 11 khi xoá dự án (cả hai service); smoke test với dữ liệu giả rồi dọn.
