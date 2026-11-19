# ADR-0050: rendering sở hữu thư viện hình; kit là dữ liệu của phong cách tham chiếu

## Status
Proposed

## Date
2026-11-19

## Stage
Inception / Application Design (CR-116)

## Context
authoring-service đang giữ thư viện hình minh hoạ, bộ vẽ AI, "ngôn ngữ hình" (`visual_languages`, gắn `kit_key` cho từng dự án) và gọi rendering để vẽ. Creator muốn authoring chỉ viết kịch bản. Phong cách hình của dự án đã có ở bước 8: phong cách tham chiếu (`reference_styles`, visual-service) mà dự án chọn. Bộ thông số vẽ thật (kit `phatsang`: viền sáng, bóng, bloom…) lại nằm cứng trong code Remotion (`kits.ts`, `kits.json`). Bước 11 Storyboard (CR-117) cần vẽ khung và dùng thư viện hình theo phong cách của dự án.

## Options Considered
### Thư viện hình
- **A. Viết lại bằng Python trong rendering** — một service lo mọi việc vẽ; gọi vẽ là gọi hàm nội bộ. Trade-off: viết lại logic Go và test.
- B. Chạy thêm chương trình Go trong container rendering — hai ngôn ngữ trong một container.
- C. Service mới `asset-service` — thêm service và DB.

### Kit
- **A. Bộ thông số là dữ liệu của phong cách tham chiếu**; rendering nhận bộ thông số kèm mỗi lệnh vẽ. Thêm phong cách mới không phải sửa code.
- B. Phong cách tham chiếu chỉ giữ `kit_key`, bộ số vẫn trong code. Thêm phong cách mới phải sửa code.

## Decision
Thư viện hình: **A** (Creator chọn). Kit: **A** (Creator chọn).
- rendering có thêm thư viện hình (DB `rendering`, ảnh ở `/shared/illustrations/`), bộ vẽ AI gọi llm-service, prompt đẩy lên thư viện prompt với `owner = "rendering"`.
- `reference_styles.detail.kit` giữ bộ thông số; R12 "Glowing Flat Vector" nhận đúng số của `phatsang`. visual-service có route nội bộ trả kit của dự án (phong cách đã chọn ở bước 8).
- Mọi lệnh vẽ của rendering (`/v1/stills`, `/v1/illustrations/preview`, `/v1/check/remotion`, lệnh `render_scenes`) nhận `kit` (object) thay `kit_key`; Remotion đọc kit từ input props. `kits.json`, bảng `KITS` trong `kits.ts`, `GET /v1/kits` bị bỏ.
- authoring bỏ thư viện hình, ngôn ngữ hình, hình theo dự án, client rendering.
- Dữ liệu cũ bị xoá, không chuyển (Creator chọn vẽ lại từ đầu).

## Consequences
- Dự án chưa chọn phong cách ở bước 8 không render được; lỗi nói rõ phải chọn ở bước 8.
- Thay ADR-0036 (ngôn ngữ hình ghim theo dự án ở authoring).
- orchestrator, llm-service, api-gateway, web-gui, visual-service đều phải sửa.
- Bước 11 (CR-117) đọc thư viện hình và vẽ khung ngay trong rendering.
