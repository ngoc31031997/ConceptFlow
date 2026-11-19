# CR-117 · Bước 11 Storyboard — thiết kế

**Trạng thái:** đã duyệt 2026-11-20 · **Yêu cầu:** [cr-117-storyboard.md](../../inception/requirements/cr-117-storyboard.md) · **Quyết định:** [ADR-0051](../../decisions/ADR-0051-storyboard-split-visual-service-and-rendering.md) · **Khuôn:** [cr-115-visual-sequencing-design.md](cr-115-visual-sequencing-design.md)

Bước 11 làm lại các mảnh của bước 9–10 (review_units, máy kiểm khi đọc, vòng AI rà soát, bản chốt bất biến), thêm hai thứ mới: **tài sản dự án** và **khung hình**. Mockup giao diện được duyệt trước khi code giao diện (§9).

## 1. Input
| Input | Nguồn | Phần dùng |
|---|---|---|
| visual-sequence | bản chốt bước 10 (`visual_sequence_versions`) | `sequence[]` (thứ tự, `purpose_in_flow`, nhịp, chuyển cảnh, `motif_refs`), `motifs[]` |
| visual-plan | bản chốt bước 9 | `units[]`: `unit_id`, `beat_id`, `script_refs`, `communication_goal`, `essential_visual_info`, `on_screen_text`, `visual_approach`, `duration` |
| visual-concept | bản chốt bước 8 | `representation_rules` (nhân vật, vật thể, khái niệm: `concept_id`, `draw`, `must_not_imply`), `color_system` (bảng màu, journey), `typography` (số từ tối đa, cỡ chữ), `visual_language.composition_rules`, `do_not_use` |
| script-final | bản chốt kịch bản (authoring) | lời thoại của các câu trong `script_refs`, ghi chú bàn giao |
| kit | phong cách tham chiếu dự án chọn ở bước 8 (`GET /internal/v1/visual/{id}/kit`) | bộ thông số vẽ |

**Phiên bản lệch** như bước 10: bước 11 làm trên bản chốt mới nhất của bước 10; nếu bước 8/9 hoặc kịch bản có số mới hơn bản mà bước 10 ghi, hiện cảnh báo và `lock` trả `409 sequence_outdated`. Chưa chốt bước 10 → `409 sequence_not_locked` cho mọi route ghi. Dự án chưa chọn phong cách bước 8, hoặc phong cách chưa có kit → `409 no_kit`, không vẽ.

## 2. Mô hình dữ liệu

### 2.1 rendering (`rendering-db`)
| Bảng | Khoá | Cột chính |
|---|---|---|
| `project_assets` | `(project_id, asset_id)` | `asset_id` (`AST-<TÊN>-<số>`, do máy cấp, không dùng lại), `concept_id`, `name` (component PascalCase), `kind` (figure/backdrop), `tier` (hero/support/background), `portrayal`, `must_not_imply[]`, `palette_tokens[]`, `expressions[]`, `base_illustration_id`, `code`, `version`, `preview_path`, `status` (`planned` → `drawn` → `approved`), `to_library BOOL DEFAULT true`, `library_folder_id`, `library_illustration_id`, `warnings JSONB`, thời gian |
| `storyboard_shots` | `(project_id, shot_id)` | `unit_id`, `beat_id`, `frame_path`, `frame_hash`, `spec_hash`, `drawn_at`, `error` (lần vẽ gần nhất) — bản ghi **khung đã vẽ**, không giữ nội dung sửa của Creator |
| `storyboard_versions` | `(project_id, version)` | `document JSONB` (tài liệu chốt), `sequence_version`, `note`, `locked_at`; chỉ thêm |

Bootstrap `CREATE TABLE IF NOT EXISTS` trong `adapters/persistence/db.py`. Mã `AST-` cấp từ bộ đếm theo `(project_id)` (giống `next_seq`); tên dùng lại cho cùng khái niệm được `-002`.

`concept_id` nối tới visual concept; `base_illustration_id` nối `illustrations.id` của thư viện (có thể trống).

### 2.2 visual-service (`visual-db`)
| `kind` (`review_units`) | `unit_key` | `content` |
|---|---|---|
| `storyboard_assets` | `main` | `{"assets":[{asset_id, status}], "next_seq": n}` — danh sách và trạng thái duyệt; **chi tiết tài sản nằm ở rendering** |
| `storyboard_unit` | `VP-xxx` | `{"shots":[Shot…], "next_seq": n}` |
| `storyboard_review` | `main` | cùng khuôn bước 8–10 |

```jsonc
// Shot (trong review_units)
{ "shot_id": "SH-001", "script_refs": ["SC-001"], "shot_purpose": "…",
  "composition": { "subject": "astronaut", "subject_position": "lower_left",
                   "main_object": "black_hole", "main_object_position": "upper_right",
                   "camera_view": "wide" },
  "action_summary": "…", "on_screen_text": [], "asset_refs": ["AST-ASTRONAUT-001"],
  "notes": [], "status": "pending" }   // pending | approved
```
- `unit_id` và `beat_id` suy ra từ unit chứa shot, không do AI viết.
- `shot_id` do máy cấp `SH-001`… từ `next_seq` theo dự án, không dùng lại.
- `composition.subject` và `main_object` là `asset_id` hoặc tên tự do (nhân vật chưa thành tài sản); `subject_position`, `main_object_position`, `camera_view` thuộc **từ vựng đóng** (§5).
- Bảng mới `storyboard_status` không cần: trạng thái unit và bước tính từ `status` các shot khi đọc.
- `project_visual` thêm `storyboard_model TEXT NOT NULL DEFAULT ''`.

## 3. Luồng nghiệp vụ
**Pha tài sản**
1. **AI lập danh sách tài sản** (`POST /assets/generate`, `X-External-AI`): một lượt, prompt `storyboard_assets`. Input: `representation_rules`, các unit của bước 9 (`essential_visual_info`, `visual_approach`), motif, bảng màu, danh sách hình thư viện (id, tên, tag, mô tả, `kind`, tier mặc định, chỉ hình đã duyệt). Reply `{"assets":[{same_as, concept_id, name, kind, tier, portrayal, must_not_imply, palette_tokens, expressions, base_illustration_id}]}`. `concept_id` phải có trong `representation_rules`; `palette_tokens` phải có trong bảng màu; `base_illustration_id` phải là hình đã duyệt, cùng `kind`; sai → 502, không ghi. `same_as` giữ mã cũ; trường Creator đã sửa được giữ.
2. **Creator sửa, thêm, xoá, duyệt danh sách** (`PUT/POST/DELETE /assets/{aid}`, `POST /assets/approve`).
3. **Vẽ tài sản** (`POST /assets/{aid}/draw`): visual-service gọi rendering, rendering ghép mô tả từ `portrayal` + `must_not_imply` + `palette_tokens` + `expressions` + tier thành `Description` rồi chạy bộ vẽ AI đã có (`illustration_drawer`, tối đa 3 lượt, kiểm tsc + luật style theo tier), với kit của dự án. Có `base_illustration_id` thì code hình gốc làm Hình mẫu để AI chỉnh. Kết quả: `code`, PNG xem trước ở `/shared/{project_id}/storyboard/assets/{asset_id}/v{n}.png`, `status = drawn`. Không đổi prompt hệ thống đã duyệt ở CR-116.
4. **Duyệt tài sản** (`POST /assets/{aid}/approve`): nếu `to_library` tick, rendering tạo hình trong thư mục thư viện (`library_folder_id`, mặc định thư mục "Từ dự án"), tên `{asset name}-{project short id}`, mô tả ghi tên dự án; lưu `library_illustration_id`. Bỏ duyệt không xoá hình đã đưa vào thư viện.
5. **Vẽ lại** (`POST /assets/{aid}/redraw`, `{note}`): tăng `version`; các shot dùng tài sản này có khung cũ → `frame_stale`.

**Pha shot** (chỉ chạy khi mọi tài sản mà unit cần đã `approved`; không thì `409 assets_missing` kèm danh sách)
6. **AI viết shot một unit** (`POST /units/{vp}/generate`, `{note}`, `X-External-AI`): prompt `storyboard_unit`. Reply `{"shots":[{script_refs, shot_purpose, composition, action_summary, on_screen_text, asset_refs, notes}]}`; phải ≥ 1 shot, `script_refs` ⊆ `script_refs` của unit và phủ hết chúng, `asset_refs` là tài sản đã duyệt, enum vị trí/góc máy hợp lệ, số từ chữ không vượt typography; sai → 502. Truyền các shot của unit trước. Giữ `creator_fields`.
7. **Viết cho mọi unit còn trống** (`POST /generate`): lần lượt theo thứ tự chuỗi; DeepSeek xếp hàng đợi như bước 9–10.
8. **Dựng khung** (`POST /units/{vp}/shots/{sid}/frame`, và tự động sau khi viết/sửa bố cục): visual-service gửi `spec` cho rendering (§5); rendering vẽ, ghi PNG, cập nhật `storyboard_shots`. Vẽ lỗi không chặn lưu shot; hiện lỗi trên thẻ.
9. **Sửa tay shot** (`PUT /units/{vp}/shots/{sid}`), thêm/xoá shot trong unit (shot cuối của unit không xoá được; mỗi unit ≥ 1 shot). Sửa → `pending`, khung cũ.
10. **Duyệt shot**; unit tự duyệt khi mọi shot duyệt; "Duyệt tất cả còn lại" bỏ qua shot có lỗi máy kiểm hoặc khung stale.
11. **Rà soát** (`POST /review`, prompt `storyboard_review`): nhóm `accuracy` (khớp plan, lời thoại), `consistency` (khớp `must_not_imply`, tài sản nhất quán, nhịp so với bước 10), `clarity` (chữ đọc được, bố cục), `feasibility`. Vấn đề trỏ tới `VP-xxx`/`SH-xxx`/`AST-xxx`. Áp dụng gợi ý, chấp nhận kèm lý do.
12. **Chốt** (`POST /lock`). Điều kiện: bước 10 đã chốt và không lệch phiên bản; mọi tài sản `approved`; mọi unit của bản chốt bước 10 có ≥ 1 shot; mọi shot `approved`, khung đã vẽ và không stale; không shot stale; đã rà soát ít nhất một lần, không còn vấn đề mở; máy kiểm đạt hoặc đã chấp nhận; có thay đổi so với bản trước. Rendering ghi `storyboard_versions`, `storyboard.json`, sao ảnh khung của phiên bản vào `/shared/{project_id}/storyboard/v{n}/SH-xxx.png`.

## 4. Stale và máy kiểm
| Đối tượng | Hash |
|---|---|
| unit `VP-xxx` | `seq` (mục của unit trong bước 10), `plan` (unit trong bước 9), `script` (các câu), `assets` (version các tài sản mà shot dùng) |
| khung `SH-xxx` | `spec_hash` (composition, chữ, tài sản + version, kit, màu nền) so với `frame_hash` đã vẽ |

Máy kiểm (không lưu, tính mỗi lần đọc), lỗi chặn duyệt shot: `m-shot-unit` (shot thuộc unit có thật), `m-shot-script` (`script_refs` ⊆ unit), `m-shot-assets` (`asset_refs` tồn tại và `approved`), `m-shot-enum` (từ vựng §5), `m-shot-text` (số từ/số mục chữ trong luật typography), `m-shot-frame` (ảnh khung tồn tại, không stale). Cảnh báo (chấp nhận kèm lý do): `m-asset-unused` (tài sản đã duyệt không shot nào dùng), `m-unit-cover` (unit chưa có shot), `m-asset-count` (tài sản dùng ở một shot duy nhất).

## 5. Dựng khung (rendering)
**`spec`** do visual-service dựng từ shot và gửi:
```jsonc
{ "project_id", "shot_id", "kit": {…}, "size": {"w":1280,"h":720},
  "background": {"asset_id": "AST-SPACE-001" | null, "color": "#0F1B3D"},   // màu journey của beat
  "placements": [{"asset_id","role":"subject"|"main","position":"lower_left","scale":0.45}],
  "camera_view": "wide", "on_screen_text": [{"text":"Góc nhìn A","min_px":28}],
  "text_color": "#FFFFFF" }
```
**Từ vựng đóng** (kiểm bằng code, ở cả hai service; AI trả giá trị lạ → 502):
- `position`: `upper_left, upper_center, upper_right, center_left, center, center_right, lower_left, lower_center, lower_right, split_left, split_right, full`. `split_screen` của mẫu JSON là cách gọi chung khi bố cục chia đôi: hai đối tượng nhận `split_left`/`split_right`; AI viết `split_screen` thì máy đổi thành cặp này khi có đúng hai đối tượng, ngược lại 502.
- `camera_view`: `wide, medium, close, comparison`. Hệ số `scale` theo tier và góc máy (bảng cố định trong code, ví dụ hero/close lớn hơn support/wide).

**Cách dựng:** mở rộng host xem trước hiện có (`illustration_preview.mjs`, `host.tsx`): thêm lệnh `frame`, nhận `placements` kèm `js` đã biên dịch của từng tài sản (esbuild như hình xem trước) và đặt mỗi tài sản tại ô lưới theo `position`; nền là backdrop tài sản hoặc màu; chữ vẽ ở dải dưới theo `min_px`. Cùng tiến trình Node, cùng hàng đợi một yêu cầu một lúc với xem trước hình. Cache theo `spec_hash` (cùng spec trả lại ảnh cũ).

**Rủi ro kỹ thuật:** dựng nhiều tài sản trong một khung là việc mới so với "một hình một ảnh". **Bước đầu của kế hoạch là một thử nghiệm nhỏ (spike)**: dựng một khung hai tài sản + chữ bằng kit R12 và đọc ảnh ra; nếu không đạt thì dừng và báo.

## 6. Hợp đồng HTTP
**Công khai (api-gateway → visual-service)**, tiền tố `/v1/projects/{id}/storyboard`:
| | Đường dẫn | Việc |
|---|---|---|
| GET | `` | toàn màn (tài sản, shot theo beat → unit, trạng thái, máy kiểm, khung) |
| PUT | `/model` | `{model}` |
| POST | `/assets/generate` | `{note}`, `X-External-AI` |
| POST/PUT/DELETE | `/assets`, `/assets/{aid}` | thêm, sửa, xoá |
| POST | `/assets/{aid}/draw` · `/redraw` · `/approve` · `/unapprove` | |
| POST | `/assets/approve` | duyệt danh sách |
| POST | `/generate` · `/units/{vp}/generate` | `{note}`, `X-External-AI` |
| PUT/POST/DELETE | `/units/{vp}/shots`, `/units/{vp}/shots/{sid}` | sửa, thêm, xoá |
| POST | `/units/{vp}/shots/{sid}/frame` · `/approve` · `/unapprove` | |
| POST | `/approve-all` · `/review` · `/findings/{fid}/apply` · `/accept` · `/lock` | |
| GET | `/versions/{n}` · `/frames/{shot}` | tài liệu chốt; ảnh khung (qua rendering) |

**Nội bộ:** `GET /internal/v1/visual/{id}/storyboard` (visual-service, proxy bản chốt mới nhất từ rendering, 404 khi chưa chốt). Rendering (`/internal/v1/…`): `GET/PUT /projects/{id}/assets…`, `POST /projects/{id}/assets/{aid}/draw`, `POST /projects/{id}/storyboard/frames`, `POST /projects/{id}/storyboard/versions`, `GET /projects/{id}/storyboard/versions/latest`, `DELETE /projects/{id}/storyboard` (xoá dự án). Hợp đồng mới `docs/contracts/storyboard-v1.md`; cập nhật `visual-service-v1.md`, `illustration-library-v1.md`, README hợp đồng.

## 7. Tài liệu bản chốt (`storyboard.json`)
Theo mẫu của Creator, thêm vài trường kiểm soát:
```jsonc
{ "schema_version": "1.0", "project_id": "…", "document_id": "storyboard-v1", "status": "in_review",
  "source_versions": {"script": 2, "visual_concept": 1, "visual_plan": 1, "visual_sequence": 1},
  "assets": [{"asset_id","concept_id","name","kind","tier","portrayal","preview_path"}],
  "shots": [{ "shot_id","unit_id","beat_id","script_refs","frame_path":"storyboard/v1/SH-001.png",
              "shot_purpose","composition":{…},"action_summary","on_screen_text":[],"asset_refs":[],"notes":[] }] }
```
`status` của tài liệu chốt do máy ghi `locked`; `in_review` là trạng thái bước khi đang làm (trả ở route toàn màn). `frame_path` tương đối so với `/shared/{project_id}`; không nhúng base64. Thứ tự `shots` theo `sequence_index` của bước 10 rồi theo `shot_id`.

## 8. Phân rã công việc (cho bước code)
| # | Việc | Service | Phụ thuộc |
|---|---|---|---|
| S0 | **Spike** dựng khung nhiều tài sản (§5), báo kết quả | rendering | — |
| S1 | Bảng và API tài sản + vẽ tài sản + đưa vào thư viện | rendering | — |
| S2 | Dựng khung `frame` + `storyboard_shots` + `storyboard_versions` + route nội bộ | rendering | S0 |
| S3 | Domain, máy kiểm, đọc chặt reply AI, hash/stale, bản chốt | visual-service | — |
| S4 | Desk, viết tài sản/shot/rà soát, HTTP, kho Postgres, client rendering | visual-service | S1, S2, S3 |
| S5 | Prompt (`storyboard_assets`, `storyboard_unit`, `storyboard_review`) | visual-service, llm-service | **Creator duyệt nguyên văn** |
| S6 | Gateway + web-gui theo mockup đã duyệt | api-gateway, web-gui | mockup, S4 |
| S7 | `flow.ts` bước 11: bỏ `comingSoon`, `screen: "storyboard"`; đánh dấu xong khi đã chốt | web-gui | S6 |
| S8 | Hợp đồng, ADR, purge (xoá dự án), smoke test | tất cả | S1–S7 |

## 9. Mockup và giao diện
Trang `/projects/:id/storyboard`, hai tab (câu 8A): **Tài sản** (danh sách thẻ tài sản: ảnh, `portrayal`, tier, hình gốc đề xuất, ô tick "Đưa vào thư viện" + chọn thư mục, nút vẽ/vẽ lại/duyệt) và **Shot** (nhóm beat → unit; thẻ shot: ảnh khung, mục đích, bố cục, hành động, chữ, tài sản, ghi chú, duyệt/sửa/vẽ lại; nút "Xem dạng lưới" chỉ xem). Có thêm tab hoặc phần **Rà soát và chốt** như bước 10. **Mockup (trang HTML) được Creator duyệt trước khi giao việc S6.**

## 10. Prompt
Ba vai `storyboard_assets`, `storyboard_unit`, `storyboard_review`, owner `visual`, step `storyboard`. **Nguyên văn đưa Creator duyệt trước khi ghi seed.** Bộ vẽ tài sản dùng lại prompt `illustration_drawer` đã duyệt ở CR-116 (mô tả tài sản đi qua trường `Description`).

## 11. Kiểm thử
- Go (visual-service): đọc chặt reply (tài sản, shot), từ vựng đóng, máy kiểm, mã `AST-`/`SH-` không dùng lại, stale theo unit và khung, điều kiện chốt, v1 rồi v2, HTTP với fake port rendering, kho Postgres.
- Python (rendering): bảng, vẽ tài sản (AI giả), đưa vào thư viện, dựng khung (có trình duyệt thì ảnh khớp mốc), `storyboard_versions`, xoá dự án.
- Vitest: trang bước 11.
- Không gọi AI thật. Dọn dữ liệu test sau khi chạy.

## 12. Rủi ro
- Dựng nhiều tài sản trong một khung chưa từng làm → S0 chạy đầu tiên.
- Bộ vẽ AI vẽ sai hoặc lệch kit nhiều lần → tài sản ở `planned`/lỗi, Creator sửa mô tả rồi vẽ lại; không giả thành công.
- Bước 12 "Thiết kế tài nguyên" trên bản đồ luồng trùng ý với pha tài sản: để CR sau.
