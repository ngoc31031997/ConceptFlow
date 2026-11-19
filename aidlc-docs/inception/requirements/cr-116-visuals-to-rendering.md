# CR-116 · Chuyển phần hình ảnh từ authoring-service sang rendering-service — yêu cầu

**Trạng thái:** chờ Creator duyệt · **Nhánh:** `feature/cr-116-visuals-to-rendering` · **Câu hỏi:** [cr-116-requirement-questions.md](cr-116-requirement-questions.md)

## Mục tiêu
authoring-service chỉ còn lo viết kịch bản. Mọi thứ về hình (thư viện hình minh hoạ, bộ vẽ AI, vẽ khung tĩnh) nằm ở rendering-service. Phong cách hình của dự án là phong cách tham chiếu chọn ở bước 8 (visual-service); bộ thông số vẽ (kit) là dữ liệu của phong cách tham chiếu đó. Đây là nền cho CR-117 (bước 11 Storyboard).

## Phụ thuộc
- Không có CR/nhánh nào đang mở → không chồng lấn.
- CR-117 phụ thuộc CR-116.

## Yêu cầu chức năng

**FR-1 · Thư viện hình minh hoạ ở rendering (viết lại bằng Python, câu 1A)**
Rendering có thư viện hình: thư mục, hình (figure/backdrop, tier hero/support/background), trạng thái duyệt, hình mẫu, vẽ thử, vẽ bằng AI, vẽ lại, xuất/nhập — cùng hành vi như ở authoring hiện nay. Dữ liệu ở DB `rendering`.

**FR-2 · Ảnh xem trước lưu thành file (câu 7B)**
PNG/GIF xem trước lưu ở `/shared/illustrations/{id}.png|.gif`; DB chỉ giữ đường dẫn và phiên bản.

**FR-3 · Bộ vẽ AI gọi model qua llm-service (câu 6A)**
Prompt vẽ hình và vẽ phông nền chuyển vào thư viện prompt của llm-service, có nhật ký cuộc gọi. Nội dung prompt đưa Creator duyệt nguyên văn trước khi sửa seed. Luật vẽ lấy từ phong cách tham chiếu bước 8 (câu 10); các file luật `phatsang_*` của authoring bị bỏ.

**FR-4 · Kit là dữ liệu của phong cách tham chiếu (câu 9, 1a)**
- Phong cách tham chiếu trong thư viện tham chiếu (visual-service) có thêm bộ thông số kit: viền, bo góc, viền sáng, bóng, AO, độ bóng, vân bề mặt, SSS, toả sáng, bloom, sương, độ nhoè theo lớp, hạt, tối góc, màu nét, ánh sáng xem trước.
- R12 "Glowing Flat Vector" nhận đúng bộ số của kit `phatsang` hiện nay.
- Rendering nhận bộ thông số kèm mỗi lệnh vẽ (khung tĩnh, xem trước hình, render video) thay vì đọc `kits.json`; `kits.json` và giá trị viết cứng trong `kits.ts` bị bỏ, `GET /v1/kits` bị bỏ.
- Orchestrator lấy bộ thông số của dự án từ visual-service (phong cách đã chọn ở bước 8) thay vì hỏi authoring `kit_key`. Dự án chưa chọn phong cách ở bước 8 thì không render được, báo lỗi rõ ràng.
- Thư viện hình dùng chung (không gắn dự án) cần một phong cách tham chiếu để vẽ: chọn khi vẽ (mặc định R12).

**FR-5 · Giữ URL cho web-gui (câu 5A)**
web-gui gọi đúng các URL `/v1/illustration-folders`, `/v1/illustrations*`, `/v1/illustration-tries`, `/v1/illustration-style`, `/v1/admin/illustrations*`; api-gateway đổi đích sang rendering. Các route phong cách hình (`/v1/visual-languages*`, `/v1/projects/{id}/visual-language`) và hình theo dự án (`/v1/projects/{id}/illustrations*`) bị bỏ cùng màn hình của chúng.

**FR-6 · Gỡ phần hình khỏi authoring (câu 3A, 4)**
Bỏ code, route, bảng: `illustration_folders`, `illustrations`, `project_illustrations`, `visual_languages`, `visual_language_versions`, `visual_language_samples`, cột `project_authoring.visual_language_id`; bỏ client gọi rendering của authoring.

**FR-7 · Khung tĩnh và dọn khung cũ ở rendering**
`/v1/stills` và `/v1/stills/prune` ở lại rendering, nhận bộ thông số kit thay cho `kit_key`; bên gọi duy nhất về sau là visual-service (CR-117).

## Dọn dữ liệu (câu 2, 8 và quyết định 2026-11-19) — KHÔNG HOÀN TÁC, KHÔNG SAO LƯU
Chạy một lần khi triển khai CR-116, sau khi Creator duyệt thiết kế:
1. Xoá toàn bộ 6 bảng hình ở authoring (293 hình, 14 thư mục, 1 phong cách, 1 phiên bản, 4 khung mẫu, 0 hình dự án) và ảnh khung tĩnh cũ ở `/shared/stills`.
2. **Xoá mọi dự án trên mọi service, chỉ giữ dự án `335da643-10dc-42d9-85b4-7f3925f4646e`** (đã chọn R12 ở bước 8). Phạm vi: dòng của các dự án khác ở orchestrator, authoring, visual, llm, tts, rendering, video-assembly, publisher và thư mục `/shared/{project_id}/` của chúng. Danh sách bảng/thư mục cụ thể từng service lập ở bước thiết kế và đưa Creator xem trước khi chạy.
3. Không đụng dữ liệu không gắn dự án khác (thư viện prompt, thư viện tham chiếu, cấu hình model).

## Yêu cầu phi chức năng
- Viết lại theo `docs/code-standards-rules.md`; test rendering mới không gọi AI thật (giả lời AI).
- Màn hình thư viện hình giữ nguyên trải nghiệm; mọi màn hình sửa theo `docs/ux-ui-design-rules.md`.
- Cập nhật hợp đồng (`docs/contracts/`) và một ADR cho việc chuyển kit thành dữ liệu của phong cách tham chiếu.
- Rebuild: rendering, authoring-service, visual-service, orchestrator, llm-service, api-gateway, web-gui.

## Ngoài phạm vi
- Tài sản dự án `AST-…` và bước 11 (CR-117).
- Đồng bộ Google Drive.
