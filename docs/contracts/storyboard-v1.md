# storyboard v1 (rendering)

Hợp đồng nội bộ của bước 11 (Storyboard) ở rendering. Mô tả phần **Tài sản**, **Khung** và **Bản chốt**; các phần còn lại được bổ sung khi các route đó có. Mã: `services/rendering/` (`domain/storyboard.py`, `application/storyboard.py`, `adapters/http/storyboard.py`, `adapters/persistence/storyboard_repository.py`, `adapters/storage/storyboard_files.py`, `domain/project_asset.py`, `application/project_assets.py`, `adapters/http/project_assets.py`, `adapters/persistence/project_asset_repository.py`, `adapters/storage/project_asset_files.py`). Thiết kế: [cr-117-storyboard-design.md](../../aidlc-docs/construction/plans/cr-117-storyboard-design.md) §2.1, §3; [ADR-0051](../../aidlc-docs/decisions/ADR-0051-storyboard-split-visual-service-and-rendering.md). Bộ vẽ AI và thư viện hình: [`illustration-library-v1.md`](illustration-library-v1.md).

Chỉ visual-service gọi (mạng `backend`), không qua api-gateway. Thân JSON; lỗi `{"error":"<thông báo>"}`; thân hỏng → `400 {"error":"invalid JSON body"}`.

## Tài sản (`ProjectAsset`)

```json
{ "project_id": "…", "asset_id": "AST-SPACE-001", "concept_id": "C1", "name": "Space",
  "kind": "figure|backdrop", "tier": "hero|support|background", "portrayal": "…",
  "must_not_imply": [], "palette_tokens": [], "expressions": [], "base_illustration_id": "",
  "units": ["VP-001"], "has_detail": false, "code": "…", "version": 0, "preview_path": "", "status": "planned|drawn|approved",
  "to_library": true, "library_folder_id": "", "library_illustration_id": "",
  "warnings": [{"message","line"}], "created_at": "RFC3339", "updated_at": "RFC3339" }
```

- `asset_id` do máy cấp: `AST-<TÊN>-<nnn>` với `<TÊN>` là `name` viết hoa, `nnn` từ bộ đếm theo `(project_id, TÊN)`; xoá tài sản không làm số được cấp lại (cùng tên lần sau là `-002`, …).
- `name` là component được export: PascalCase, 2–41 ký tự chữ/số. `concept_id` bắt buộc. `portrayal` có thể trống khi tài sản mới chỉ có phần khung (xem "Chi tiết"). `base_illustration_id` (tuỳ chọn) là `illustrations.id` của thư viện, phải cùng `kind`; nếu hình đó có code thì làm Hình mẫu cho AI chỉnh.
- **Chi tiết.** Tài sản có hai phần: phần **khung** (`concept_id`, `name`, `kind`, `tier`, `base_illustration_id`, `units`) do lượt AI thứ nhất lập, và phần **chi tiết** (`portrayal`, `must_not_imply`, `palette_tokens`, `expressions`) do lượt AI thứ hai viết riêng từng tài sản. `has_detail` do máy tính: `true` khi `portrayal` (cắt khoảng trắng) không trống; không có cột riêng. `units` là danh sách mã unit kế hoạch hình (`VP-…`) mà tài sản xuất hiện, `TEXT[] NOT NULL DEFAULT '{}'` ở cột `project_assets.units` (bootstrap `ALTER TABLE … ADD COLUMN IF NOT EXISTS`); rendering cắt khoảng trắng, bỏ mục trống và mục trùng, giữ thứ tự, và không kiểm mã unit có trong kế hoạch (việc của visual-service).
- Trạng thái: `planned` → `drawn` (sau `draw`/`redraw`, `version` tăng 1, có `code`, `preview_path`) → `approved`. Sửa `name`, `kind`, `tier`, `portrayal`, `must_not_imply`, `palette_tokens`, `expressions`, `base_illustration_id` hay `units` của tài sản `drawn`/`approved` đưa nó về `planned` (code, version, ảnh giữ nguyên tới lần vẽ lại). Sửa `concept_id`, `to_library`, `library_folder_id` không đổi trạng thái.
- Ảnh xem trước: `/shared/{project_id}/storyboard/assets/{asset_id}/v{n}.png`, ghi file tạm rồi đổi tên; chỉ giữ ảnh phiên bản hiện tại.

## Route (`/internal/v1/projects/{project_id}/assets`)

| Route | Việc | Thành công |
|---|---|---|
| `GET …/assets` | liệt kê, cũ trước | `200 {"assets":[…]}` |
| `POST …/assets` | lập tài sản (`planned`); thân: `concept_id, name, kind, tier, portrayal, must_not_imply, palette_tokens, expressions, base_illustration_id, units, to_library, library_folder_id` | `201` tài sản |
| `GET …/assets/{aid}` | đọc một tài sản | `200` |
| `GET …/assets/{aid}/preview` | PNG xem trước hiện tại (`image/png`, `Cache-Control: private, no-cache`); `404` khi chưa vẽ; đường dẫn ảnh phải nằm trong `/shared`, ngoài đó bị từ chối | `200` |
| `PUT …/assets/{aid}` | sửa (cùng thân với tạo, thay toàn bộ các trường đó) | `200` |
| `DELETE …/assets/{aid}` | xoá hàng và ảnh; hình đã vào thư viện được giữ | `204` |
| `POST …/assets/{aid}/draw` | vẽ từ mô tả ghép; thân `{kit, ref_id, model?}`; `409 {"error","code":"asset_no_detail"}` khi chưa có chi tiết (`portrayal` trống), không gọi bộ vẽ | `200` tài sản `drawn` |
| `POST …/assets/{aid}/redraw` | vẽ lại từ code đang có; thân `{kit, ref_id, model?, note?}`; `409` khi chưa có hình | `200` |
| `POST …/assets/{aid}/approve` | duyệt tài sản đã vẽ; thân `{project_name?, ref_id?}` | `200` tài sản `approved` |
| `POST …/assets/{aid}/unapprove` | về `drawn`; không xoá hình thư viện | `200` |

- **Vẽ**: chỉ vẽ lần đầu khi `has_detail` (không thì `409` mã `asset_no_detail`); vẽ lại dùng code đang có nên không cần kiểm. Mô tả gửi bộ vẽ AI có sẵn ghép từ `portrayal` + `must_not_imply` + `palette_tokens` + `expressions` + hạng; component phải mang đúng `name` (AI đổi tên → `400`). `kit` (bộ thông số vẽ của dự án) do visual-service gửi mỗi lần, được kiểm bằng `domain.kit.validate_kit` (sai/thiếu → `400`); `ref_id` là phong cách tham chiếu của dự án, rendering đọc luật vẽ của nó ở visual-service. Prompt vẽ là prompt đã duyệt của thư viện, không đổi. Code bị bộ dựng từ chối sau 3 lượt → `422` kèm `diagnostics`; không có kết nối model → `503`.
- **Duyệt**: chưa vẽ → `409`. Nếu `to_library`, rendering tạo hình đã duyệt trong thư mục `library_folder_id` (mặc định thư mục hệ thống "Từ dự án", id `tu-du-an`, tạo khi chưa có; không được là thư mục Hình mẫu) và lưu `library_illustration_id`. Hàm export của hình thư viện là `{name}{8 ký tự chữ-số đầu của project_id}` (tên component phải hợp lệ và duy nhất cả thư viện), tiêu đề `{name}-{8 ký tự đầu}`, mô tả ghi `project_name`. Duyệt lại sau khi vẽ lại cập nhật chính hình đó (phiên bản tăng) nếu nó còn trong thư viện.
- **Xoá**: rendering không biết shot nào dùng tài sản, nên việc từ chối xoá tài sản đang được shot dùng thuộc về visual-service, trước khi gọi `DELETE`.
- Mã lỗi: `404` không có tài sản, thư mục hay hình gốc; `409` sai trạng thái hoặc tên hình thư viện bị trùng; `403` hình thư viện chỉ đọc; `422` code không qua bộ dựng hoặc thư mục Hình mẫu; `503` bộ vẽ/bộ dựng/visual-service không chạy được; lỗi model theo bảng của thư viện hình.

## Khung (`storyboard_shots`)

Bản ghi khung đã vẽ gần nhất của mỗi shot, khoá `(project_id, shot_id)`: `unit_id`, `beat_id`, `frame_path` (tương đối so với `/shared/{project_id}`, `null` khi lần vẽ gần nhất lỗi), `frame_hash` (SHA-256 của PNG, chỉ để tham khảo), `spec_hash` (do người gọi tính), `drawn_at`, `error`. Không giữ nội dung sửa của Creator.

### `POST /internal/v1/projects/{project_id}/storyboard/frames`

Thân (spec của thiết kế §5 kèm mã shot; **không có `scale`**: kích thước từng hình do rendering quyết):

```json
{ "shot_id": "SH-001", "unit_id": "VP-001", "beat_id": "B1", "spec_hash": "<do visual-service tính>",
  "kit": {…}, "size": {"w": 1280, "h": 720},
  "background": {"asset_id": "AST-SPACE-001", "color": "#0F1B3D"},
  "placements": [{"asset_id": "AST-HERO-001", "role": "subject", "position": "lower_left"}],
  "camera_view": "wide", "on_screen_text": [{"text": "Góc nhìn A", "min_px": 28}], "text_color": "#FFFFFF" }
```

- `spec_hash` do **người gọi** tính (composition, chữ, tài sản + version, kit, màu nền): chuỗi chữ-số 8–128 ký tự. Rendering không tự băm; dùng nó làm khoá cache, lưu vào `storyboard_shots.spec_hash` và trả lại.
- `background.asset_id` tuỳ chọn (tài sản `backdrop`, không thì `400`); `background.color` bắt buộc (`#RGB`/`#RRGGBB`) và là màu nền khi không có backdrop. `size`, nếu gửi, phải là 1280x720. Cần ≥ 1 placement, mỗi `position` dùng một lần; `role` ∈ `subject`, `main`; tối đa 6 dòng chữ, `min_px` nguyên 1–200.
- **Từ vựng đóng** (kiểm ở cả hai phía, host `host.tsx` dùng cùng bảng): `position` ∈ `upper_left, upper_center, upper_right, center_left, center, center_right, lower_left, lower_center, lower_right, split_left, split_right, full`; `camera_view` ∈ `wide, medium, close, comparison`. Giá trị lạ → `400`.
- `kit` được kiểm bằng `domain.kit.validate_kit` (sai/thiếu → `400`).
- Mỗi tài sản (nền và các hình) phải có và `approved`; code, tên component và **hạng** (`hero`/`support`; `background` tính như `support`) đọc từ `project_assets`, người gọi không gửi. Thiếu hoặc chưa duyệt → `409 {"error", "assets": [{"asset_id","reason"}]}`, không vẽ, không ghi gì.
- **Bố cục**: ô lưới 3x3 của vùng vẽ 1280x570 (dải chữ 150px ở dưới); `split_left`/`split_right` ở 1/4, 3/4 chiều ngang giữa; `full` căn giữa. Bộ dựng đo hộp thật của từng hình rồi cho vừa theo tier và `camera_view`, giữ hình trong vùng vẽ (lề 24px): không bị cắt, không chạm dải chữ. Hai hình cùng một cột có thể chồng nhau nếu cùng lớn; chọn vị trí là việc của người gọi.
- **Cache**: cùng `spec_hash` với bản ghi hiện có, `frame_path` còn và file còn → trả bản ghi cũ, `cache_hit: true`, không vẽ lại.
- Vẽ chạy qua cùng hàng đợi một yêu cầu một lúc với xem trước hình; ghi `/shared/{project_id}/storyboard/SH-xxx.png` (file tạm rồi đổi tên). Bộ dựng (Node/Chromium) không chạy được hoặc dựng hỏng → `503 {"error"}`, bản ghi shot ghi `error`, `frame_path = null`; không bao giờ có ảnh giả.
- Thành công `200`: `{project_id, shot_id, unit_id, beat_id, frame_path: "storyboard/SH-001.png", frame_hash, spec_hash, drawn_at, error: "", cache_hit}`.

### `GET /internal/v1/projects/{project_id}/storyboard/frames/{shot_id}`

`200` thân PNG (`image/png`) của khung đã vẽ gần nhất; chưa vẽ, lần vẽ gần nhất lỗi hoặc file mất → `404`.

## Bản chốt (`storyboard_versions`)

Chỉ thêm, không sửa, không xoá (trừ khi xoá cả dự án); khoá `(project_id, version)`: `document` JSONB, `sequence_version` (số bản chốt bước 10 mà bản này dựa vào), `note`, `locked_at`.

| Route | Việc | Thành công |
|---|---|---|
| `POST …/storyboard/versions` | thân `{document, sequence_version, note}` | `201 {project_id, version, sequence_version, note, locked_at, document}` |
| `GET …/storyboard/versions/latest` | bản chốt mới nhất | `200` cùng dạng; `404` khi chưa có |
| `GET …/storyboard/versions/{n}` | một bản chốt | `200`; `404` khi không có |

- `document` là tài liệu `storyboard.json` (thiết kế §7); bắt buộc có `shots` là danh sách ≥ 1 phần tử, mỗi phần tử có `shot_id` dạng `SH-nnn` không lặp (sai → `400`). Rendering không viết lại phần nào khác ngoài `frame_path`.
- Số phiên bản do rendering cấp: lớn nhất đang có cộng một (1 khi chưa có). Số đã bị chiếm (thư mục ảnh hay hàng đã có) → `409`, không ghi đè lên bản chốt cũ.
- Trước khi ghi, mọi shot của `document` phải có file khung đang làm `/shared/{project_id}/storyboard/SH-xxx.png`; thiếu → `409 {"error", "missing_frames": ["SH-002", …]}`, không ghi gì.
- Ghi: sao từng khung sang `/shared/{project_id}/storyboard/v{n}/SH-xxx.png`, đổi `frame_path` của từng shot trong tài liệu thành `storyboard/v{n}/SH-xxx.png`, lưu hàng, rồi ghi `/shared/{project_id}/storyboard/storyboard.json` (tài liệu của bản chốt mới nhất). Ghi hỏng giữa chừng thì dọn thư mục `v{n}` đã sao.
- `frame_path` là đường dẫn tương đối so với `/shared/{project_id}`; không nhúng base64.

## Xoá dự án

`DELETE /internal/v1/projects/{project_id}/storyboard` → `204`. Xoá trong một giao dịch mọi hàng `project_assets`, `project_asset_counters`, `storyboard_shots`, `storyboard_versions` của dự án, rồi xoá thư mục `/shared/{project_id}/storyboard` (khung, bản chốt, ảnh tài sản). Hình đã đưa vào thư viện được giữ. Chạy lại khi không còn gì vẫn `204`.

Mã lỗi chung của khung và bản chốt: `400` thân/giá trị sai; `404` chưa có khung/bản chốt; `409` tài sản chưa dùng được, thiếu khung, số phiên bản đã có; `503` bộ dựng không chạy được.

---

# visual-service: bước 11 Storyboard

Phần trên là hợp đồng của rendering. Phần này là của visual-service (`services/visual-service/`: `domain/storyboard*.go`, `application/storyboard_*.go`, `adapters/http/storyboard.go`, `adapters/rendering/storyboard.go`, `adapters/postgres/storyboard_repository.go`). visual-service đọc bản chốt bước 7–10, giữ shot và duyệt, nhờ rendering giữ tài sản, vẽ khung và giữ bản chốt. Thiết kế §3, §6, §13.

## Điều kiện và lỗi

Mọi route đọc và ghi cần: kịch bản, ngôn ngữ hình ảnh, kế hoạch hình và chuỗi hình đã chốt, không thì `409` với `code` `script_not_locked`, `concept_not_locked`, `plan_not_locked`, `sequence_not_locked`. Lỗi có dạng `{"error":"…","code":"…"}`:

| Mã | `code` | Khi |
|---|---|---|
| 409 | `sequence_outdated` | `lock`: chuỗi hình viết từ bản cũ của kịch bản, ngôn ngữ hình ảnh hay kế hoạch hình |
| 409 | `asset_no_detail` | vẽ tài sản chưa có chi tiết (`portrayal` trống) |
| 409 | `no_kit` | vẽ tài sản: dự án chưa chọn phong cách bước 8 hoặc phong cách chưa có kit |
| 409 | `assets_missing` | viết shot khi video chưa có tài sản nào hoặc còn tài sản chưa duyệt (thông báo kể từng mã và trạng thái) |
| 409 | `asset_in_use` | xoá tài sản mà shot đang dùng (thông báo kể các shot); rendering không được gọi |
| 409 | `last_shot` | xoá shot cuối của một unit |
| 409 | (không) | `lock` chưa đủ điều kiện (thông báo kể mọi lý do), không có thay đổi so với bản đã chốt, duyệt shot còn lỗi máy kiểm, AI đang viết |
| 404 | | không có tài sản, shot, unit, vấn đề, bản chốt |
| 400 | | thân hỏng; trường sửa lạ; `X-External-AI` lạ |
| 502 | | AI trả lời không dùng được (không ghi gì), llm-service hay rendering không trả lời |

Rendering từ chối thì visual-service giữ nguyên mã và lý do (`400`, `404`, `409`, `422`, `503`; `5xx` khác thành `502`), kèm các khoá rendering thêm: `409` với `assets` mang `code` `assets_not_ready`, với `missing_frames` mang `code` `missing_frames`. Lỗi **dựng khung không bao giờ làm hỏng lời gọi ghi shot**: nó nằm ở `frame.error` của shot (xem dưới), không có ảnh giả.

## Route công khai (api-gateway → visual-service), tiền tố `/v1/projects/{id}/storyboard`

Mọi route (trừ `GET /versions/{n}`, `GET /frames/{shot}` và `GET /assets/{aid}/preview`) trả toàn màn như `GET /`.

| | Đường dẫn | Thân | Việc |
|---|---|---|---|
| GET | `/` | | toàn màn |
| PUT | `/model` | `{model}` | model của bước (`""` = mặc định của bước `visual_storyboard`); lạ → `400` |
| POST | `/assets/generate` | `{note}` + `X-External-AI` | **Pha 1**: AI lập **khung** danh sách tài sản, một lượt, vai `storyboard_assets` (đầu vào gọn: kế hoạch hình, mục khái niệm, motif, thư viện hình đã duyệt chỉ id/tên/loại/tag, tài sản hiện có; không có bảng màu). Reply `{"assets":[{same_as, concept_id, name, kind, tier, base_illustration_id, units}]}` đọc chặt: khoá lạ (kể cả `portrayal`, `palette_tokens`…), `units` ngoài kế hoạch hay lặp, `concept_id` ngoài quy tắc thể hiện → `502`, rendering không bị gọi. Giữ tài sản qua `same_as` và các trường Creator đã sửa (kể cả `units`); tài sản mới chưa có chi tiết; tài sản AI bỏ đi bị xoá (không bỏ được tài sản shot đang dùng); tài sản không đổi thì không gửi rendering; chi tiết của tài sản được giữ không bị đụng tới |
| POST | `/assets/{aid}/detail` | `{note}` + `X-External-AI` | **Pha 2**: một lượt AI viết chi tiết của một tài sản, vai `storyboard_asset_detail` (đầu vào gọn: chỉ mục khái niệm của tài sản, bảng màu, các unit trong `units` của tài sản kèm lời thoại, tóm tắt các tài sản khác, chi tiết hiện có, trường Creator, ghi chú). Reply `{"detail":{portrayal, must_not_imply, palette_tokens, expressions}}` đọc chặt: khoá lạ ở mọi cấp, thiếu `detail`, `portrayal` trống, `palette_tokens` ngoài bảng màu → `502`, không ghi gì. Trường chi tiết Creator đã sửa được giữ; không đổi `concept_id`, `name`, `kind`, `tier`, `base_illustration_id`, `units`. Chi tiết đổi thì tài sản rời danh sách đã duyệt và, nếu đã vẽ, về `planned` ở rendering. Tài sản lạ → `404`; đang chạy → `409` |
| POST | `/assets/detail` | `{note}` | Viết chi tiết cho mọi tài sản chưa có (`has_detail = false`), lần lượt theo thứ tự danh sách, mỗi tài sản một lượt AI và đọc chi tiết của tài sản trước. Chỉ AI của hệ thống (không có `X-External-AI`); web-gui xếp hàng DeepSeek từng tài sản. Lỗi một tài sản dừng lượt (thông báo nêu mã), chi tiết đã viết giữ; không còn tài sản nào thiếu thì không chạy lượt nào |
| POST | `/assets` | `{fields}` | thêm tài sản (`planned`): `concept_id, name, kind, tier, portrayal, must_not_imply, palette_tokens, expressions, base_illustration_id, units, to_library, library_folder_id`; `portrayal` có thể trống (AI viết chi tiết sau) |
| PUT | `/assets/{aid}` | `{fields}` | sửa; các trường ghi là của Creator (AI giữ khi lập lại danh sách); tài sản rời danh sách đã duyệt |
| DELETE | `/assets/{aid}` | | xoá (xem `asset_in_use`) |
| POST | `/assets/{aid}/draw`, `/redraw` | `redraw`: `{note}` | rendering vẽ bằng bộ vẽ AI hệ thống, với kit và `ref_id` của phong cách bước 8. Không có chế độ AI ngoài. `draw` chưa có chi tiết → `409` mã `asset_no_detail`; `draw` hay `redraw` khi danh sách tài sản chưa duyệt → `409` mã `asset_list_not_approved` (visual-service chặn cả hai trước khi gọi rendering) |
| POST | `/assets/{aid}/approve` | tuỳ chọn `{project_name}` | duyệt tài sản đã vẽ ở rendering, gửi kèm `ref_id` và `project_name` (tên dự án, để mô tả hình trong thư viện ghi tên dự án). Không có thân vẫn chạy |
| POST | `/assets/{aid}/unapprove` | | bỏ duyệt |
| GET | `/assets/{aid}/preview` | | PNG bản vẽ hiện tại của tài sản, đọc qua rendering `GET /internal/v1/projects/{pid}/assets/{aid}/preview` (`image/png`, `Cache-Control: private, no-cache`); `404` khi chưa có ảnh |
| POST | `/assets/approve`, `/assets/unapprove` | | duyệt hay mở lại **danh sách** tài sản. Duyệt cần mọi tài sản có chi tiết (thiếu → `409` mã `asset_no_detail`, nêu các mã tài sản). Danh sách đã duyệt mới cho vẽ và vẽ lại tài sản; thêm, sửa, viết lại chi tiết hay lập lại danh sách đưa danh sách về chờ duyệt. Rendering vẫn tự giữ trạng thái `approved` của từng tài sản |
| POST | `/generate` | `{note}` | viết mọi unit chưa có shot, lần lượt theo thứ tự chuỗi, mỗi unit một lượt AI và đọc shot của unit trước. Lỗi một unit dừng lượt, các unit đã viết giữ |
| POST | `/units/{vp}/generate` | `{note}` + `X-External-AI` | một lượt AI viết các shot của unit, vai `storyboard_unit`; ghép bằng `MergeShots` (giữ mã shot và trường Creator; bỏ mất shot Creator đã sửa → `502`), rồi dựng khung từng shot |
| POST | `/units/{vp}/shots` | `{fields}` | thêm shot cuối unit, mã `SH-nnn` kế tiếp của dự án |
| PUT | `/units/{vp}/shots/{sid}` | `{fields}` | sửa: `script_refs, shot_purpose, composition, action_summary, on_screen_text, asset_refs, notes`; về `pending`; dựng lại khung nếu spec đổi |
| DELETE | `/units/{vp}/shots/{sid}` | | xoá, trừ shot cuối của unit; mã không cấp lại |
| POST | `/units/{vp}/shots/{sid}/frame` | | dựng lại khung từ shot hiện có |
| POST | `/units/{vp}/shots/{sid}/approve`, `/unapprove` | | duyệt cần qua mọi máy kiểm chặn (`m-shot-*`, gồm khung đã vẽ và không cũ) → `409` kèm lý do |
| POST | `/approve-all` | | duyệt mọi shot đang chờ qua được máy kiểm; shot lỗi hay có khung cũ ở lại |
| POST | `/review` | `{note}` + `X-External-AI` | một lượt AI rà soát, vai `storyboard_review`; vấn đề trỏ `SH-xxx`, `VP-xxx` hoặc `AST-xxx` |
| POST | `/findings/{fid}/apply` | | ghi `fix` vào trường của shot (hay tài sản) vấn đề trỏ tới; vấn đề trỏ unit không áp dụng được (`400`) |
| POST | `/findings/{fid}/accept` | `{why}` | chấp nhận vấn đề hay cảnh báo máy kiểm (`m-asset-unused`, `m-unit-cover`, `m-asset-count`); máy kiểm chặn `m-shot-*` không chấp nhận được |
| POST | `/lock` | `{note}` | dựng tài liệu bằng `BuildStoryboardDocument`, gửi rendering `POST …/storyboard/versions`; không đổi so với bản trước → `409` |
| GET | `/versions/{n}` | | `{project_id, version, sequence_version, note, locked_at, document}` đọc từ rendering; `404` khi không có |
| GET | `/frames/{shot}` | | PNG khung đã vẽ gần nhất, đọc qua rendering (`image/png`, `Cache-Control: private, no-cache`); `404` khi chưa vẽ |

Bốn lượt AI chữ (`/assets/generate`, `/assets/{aid}/detail`, `/units/{vp}/generate`, `/review`) nhận `X-External-AI: prompt` (trả `{"external_prompt":"…"}`, không gọi model, không ghi gì) hoặc `X-External-AI: paste` (thân thêm `external_reply`, đọc chặt như reply của AI trong): cùng cơ chế bước 8–10, ADR-0047. `/generate` (mọi unit) và `/assets/detail` (mọi tài sản chưa có chi tiết) không có chế độ ngoài; web-gui xếp hàng DeepSeek từng unit và từng tài sản (mỗi mục dựng yêu cầu bằng chế độ mô tả của web-gui, không gửi, extension chạy rồi dán kết quả về như `paste`).

## Toàn màn (`GET /`)

Các khoá: `input` (phiên bản bước 7–10 đang đọc, `outdated`), `kit` (`ref_id`, `missing`, `message`), `assets` (`approved` của danh sách, `approved_count`, `with_detail` và `without_detail` là số tài sản đã và chưa có chi tiết cho thẻ chạy, `items[]`: mỗi tài sản của rendering kèm `units`, `has_detail`, `list_status`, `creator_fields`, `used_by`), `beats[]` → `units[]` (kế hoạch của unit, `lines`, `written`, `approved`, `stale`, `stale_sources`, `shots[]`), `shots_approved`, `shots_total`, `checks[]` (`m-shot-unit`, `m-shot-script`, `m-shot-assets`, `m-shot-enum`, `m-shot-text`, `m-shot-frame`, `m-asset-unused`, `m-unit-cover`, `m-asset-count`; kèm `accepted`, `why`), `review`, `blockers` (mỗi loại vấn đề đúng một câu, gộp theo số lượng và tối đa 3 mã đầu, vd "67 unit chưa có shot (VP-001, VP-002, VP-003 và 64 unit khác)"; `checks[].items[]` giữ từng mục lỗi), `can_lock`, `locked_version`, `locked_note`, `locked_at`, `next_version`, `draft` (tài sản, shot đổi so với bản chốt; chỉ tính khi không còn lý do nào khác chặn), `pending_units`, `empty_units`, `model`, `default_model`, `models`.

Mỗi shot: các trường của shot (`shot_id`, `script_refs`, `shot_purpose`, `composition`, `action_summary`, `on_screen_text`, `asset_refs`, `notes`, `status`), `unit_id`, `beat_id`, `creator_fields`, `problems[]` (`{check, detail}`), `missing_assets[]` (tài sản shot nêu mà chưa duyệt hay không có) và `frame`: `{drawn, stale, path, error}` (`path` tương đối so với `/shared/{project_id}`).

## Đơn vị `review_units`

| `kind` | `unit_key` | `content` |
|---|---|---|
| `storyboard_assets` | `main` | `{"assets":[{asset_id, status}], "next_seq":1}`: duyệt **danh sách**; chi tiết tài sản ở rendering. `creator_fields`: `AST-…<trường>` |
| `storyboard_unit` | `VP-xxx` | `{"shots":[Shot…], "next_seq":n, "frames":{"SH-001":{frame_path, spec_hash, frame_hash, error}}}`; `creator_fields`: `SH-001.<trường>`; `input_hashes`: `seq`, `plan`, `script`, `assets` |
| `storyboard_review` | `main` | như bước 8–10 |

`frames` là bản ghi khung vẽ gần nhất rendering trả về (`spec_hash` do visual-service tính bằng `SpecHash` rồi gửi sang). Khung cũ (`stale`) khi `spec_hash` đã lưu khác `spec_hash` của shot hiện có (bố cục, chữ, tài sản và version của chúng, kit, màu nền). Dựng khung lỗi: `503` của rendering xoá đường dẫn ảnh (rendering cũng không còn ảnh); từ chối khác (`400`/`409`) giữ ảnh cũ và ghi `error`; không có kit hay không dựng được spec cũng ghi `error`.

## Route nội bộ

`GET /internal/v1/visual/{id}/storyboard`: tài liệu `storyboard.json` của bản chốt mới nhất, lấy từ rendering `GET …/storyboard/versions/latest`. `404` khi chưa chốt.

Cột `project_visual.storyboard_model` (`''` = mặc định của bước `visual_storyboard` ở llm-service). Khi xoá dự án (`DELETE /internal/v1/visual/{id}`) visual-service xoá hàng của mình rồi gọi rendering `DELETE /internal/v1/projects/{id}/storyboard`, cố gắng một lần, lỗi thì ghi log.
