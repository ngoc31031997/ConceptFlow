# CR-116 · Thiết kế — chuyển phần hình ảnh sang rendering

**Trạng thái:** chờ Creator duyệt · **Yêu cầu:** [cr-116-visuals-to-rendering.md](../../inception/requirements/cr-116-visuals-to-rendering.md) · **Quyết định:** [ADR-0050](../../decisions/ADR-0050-rendering-owns-illustrations-kit-is-reference-data.md)

## 1. Bức tranh sau CR

```
web-gui ──► api-gateway ──► rendering   /v1/illustration*, /v1/admin/illustrations*   (thư viện hình)
                       └──► visual      /v1/reference-styles* (thêm kit)
rendering ──► llm-service   /v1/chat (bộ vẽ AI), /internal/v1/prompts/* (owner "rendering")
rendering ──► visual        GET /internal/v1/reference-styles/{id}/kit (vẽ hình thư viện theo phong cách)
orchestrator ──► visual     GET /internal/v1/visual/{project_id}/kit  → lệnh render_scenes mang "kit"
authoring: chỉ còn kịch bản (không còn gọi rendering)
```

## 2. Kit là dữ liệu (FR-4)

**Hình dạng** — đúng giao diện `KitTokens` hiện có trong `kits.ts`:
`{outline, radius, rimWidth, rimStrength, shadowStrength, aoStrength, specular{glossy,metal,glass}, texture{fur,fabric,wood,stone,organic,leaf}, sss, glowBlur, bloom, haze{far,mid}, dof{sky,far,mid,near}, grain, vignette, ink, previewLook{light{x,y,color,intensity}, ambient, atmosphere{color,density}, focus, grade{warmth,contrast,saturation}}}`.

**visual-service**
- Lưu ở `reference_styles.detail.kit` (JSONB đã có, không thêm cột). Domain thêm kiểu `ReferenceKit` + kiểm tra: đủ trường, số trong khoảng 0..1 (trừ `dof` px ≥ 0), màu `#RRGGBB`, `focus` thuộc 5 giá trị.
- Phong cách chưa có kit thì được lưu, nhưng dự án chọn nó sẽ không render được (lỗi rõ).
- R12 nhận đúng số của `phatsang` (một lần, trong bước dọn dữ liệu).
- Màn hình Thư viện tham chiếu: thêm thẻ "Bộ thông số vẽ" (sửa từng số, có nút "Vẽ thử" gọi rendering vẽ một khung mẫu bằng kit đang sửa).
- Route nội bộ mới: `GET /internal/v1/visual/{project_id}/kit` → `{ref_id, kit}`; 404 nếu dự án chưa chọn phong cách, 409 nếu phong cách chưa có kit. `GET /internal/v1/reference-styles/{id}/kit` cho rendering.

**rendering**
- `kits.json`, bảng `KITS`, `KIT_VERSIONS`, `GET /v1/kits`, `domain/kits.py` (`load_kits`, `require_kit`) bị bỏ; `skeletons.json` giữ.
- Thêm `domain/kit.py`: kiểm tra object kit (cùng luật như visual-service) — sai thì 400 nói rõ trường nào.
- Remotion `kits.ts`: `currentKit()` đọc input prop `kit` (object), kiểm tra, ném lỗi nếu thiếu.
- `/v1/stills`, `/v1/illustrations/preview`, `/v1/check/remotion`, lệnh `render_scenes`: trường `kit_key` → `kit`. Hash cache khung tĩnh tính trên nội dung kit (thay `kit_version`).

**orchestrator** — `KitKeyPort` → `KitPort` (client visual-service); payload `render_scenes.kit` thay `kit_key`.

**llm-service** — `kit_key` ở `/v2/code/*` và `checker.py` → `kit` (object), chuyển nguyên cho `/v1/check/remotion`. Ghi chú: hiện không thấy service nào gọi `/v2/code/*` trong code (chỉ authoring trả `kit_key` cho web-gui); vẫn đổi để không còn `kit_key`, không bỏ route trong CR này.

## 3. Thư viện hình ở rendering (FR-1, FR-2, FR-5)

**Bảng (DB `rendering`, bootstrap `CREATE TABLE IF NOT EXISTS` như ADR-0013):**

| Bảng | Cột |
|---|---|
| `illustration_folders` | `id`, `name`, `description`, `position`, `is_system` |
| `illustrations` | `id`, `name` (unique), `title`, `folder_id`→folders, `tags[]`, `description`, `usage`, `code`, `builtin`, `status` (draft/approved…), `version`, `kind` (figure/backdrop), `detail_level`, `ref_id` (phong cách tham chiếu vẽ cho, thay `kit_key`), `exemplar`, `source_id`, `home_folder_id`, `warnings` JSONB, `preview_version`, `preview_png_path`, `preview_gif_path`, `created_at`, `updated_at` |

- Ảnh: `/shared/illustrations/{id}/v{version}.png|.gif`; ghi file tạm rồi đổi tên (không để file dở); xoá hình thì xoá thư mục.
- Thư mục hệ thống (Hình mẫu…) tạo lúc khởi động như authoring đang làm.

**Code (Python, hexagonal như phần còn lại của rendering):**
- `domain/illustration.py` — kiểu, trạng thái, luật đổi trạng thái, kiểm tra tên/tag/kind/detail_level (chuyển 1:1 từ `domain/illustration.go`).
- `application/illustration_library.py` — thư mục, CRUD, duyệt, hình mẫu (tối đa như hiện tại), vẽ thử, vẽ lại, xuất/nhập (JSON, cùng định dạng trừ `kit_key`→`ref_id`).
- `application/illustration_drawer.py` — tối đa 3 lượt: gọi llm-service, vẽ bằng `PreviewIllustrationUseCase` nội bộ, lỗi có số dòng gửi lại cho AI.
- `adapters/persistence/illustration_repository.py`, `adapters/storage/illustration_files.py`, `adapters/llm/client.py`, `adapters/visual/client.py`.
- `adapters/http/illustrations.py` — đúng các route, thân và mã lỗi như authoring hiện nay (đối chiếu từng handler trong `illustrations.go` và test HTTP của nó).
- Mở cổng HTTP cho gateway: hiện cổng 8000 chỉ ở mạng `backend` — api-gateway cũng ở `backend` nên không phải đổi mạng; thêm `RENDERING_URL` cho gateway (đã có cho `/v1/fonts`).

**api-gateway** — `routes/illustrations.js` đổi đích sang `RENDERING_URL`; xoá `routes/visualLanguages.js` và route hình theo dự án.

**web-gui**
- `IllustrationLibraryPage`: giữ nguyên; trường "kit" → chọn phong cách tham chiếu khi vẽ (mặc định R12).
- Xoá `VisualLanguagePage` và mục menu; xoá phần hình theo dự án; xoá `kit_key` khỏi `client.ts`.
- `ReferenceLibraryPage`: thêm thẻ "Bộ thông số vẽ" (§2).

## 4. Bộ vẽ AI và prompt (FR-3)
- 2 prompt hệ thống, `owner = "rendering"`: `illustration_drawer` (vẽ hình) và `illustration_backdrop_drawer` (vẽ phông nền). rendering đẩy lúc khởi động (`PUT /internal/v1/prompts/system`, thử lại tăng dần), đọc `GET /internal/v1/prompts/{role}` mỗi lần vẽ, model mặc định bước `illustrations` qua `GET /internal/v1/step-models`.
- Nội dung = prompt vẽ hiện tại + luật lấy từ phong cách tham chiếu (cách vẽ, màu, bố cục) thay cho các file `phatsang_*`. **Bản nguyên văn sẽ đưa Creator duyệt trước khi viết vào code** (ở bước code).
- `planner_prompt_vi.txt` (lập hình theo dự án) bị bỏ cùng `project_illustrations`.

## 5. Gỡ khỏi authoring (FR-6)
Xoá: `illustrations*.go`, `illustration_*`, `project_illustrations*`, `visual_languages*`, `domain/illustration.go`, `domain/visual_language.go`, `prompts/visual_languages/`, `drawer*_prompt_vi.txt`, `planner_prompt_vi.txt`, `adapters/rendering/`, các route tương ứng, khối `visual_language_*`/`kit_key` trong state trả web-gui, và trong `db.go`: lệnh tạo 6 bảng + cột. Bảng/cột thật bị `DROP` một lần (§6).

## 6. Dọn dữ liệu — không hoàn tác, không sao lưu
Chạy bằng một script SQL/shell một lần, **sau khi code mới đã deploy** (để service cũ không tạo lại bảng), Creator xem danh sách này trước. Giữ `K = 335da643-10dc-42d9-85b4-7f3925f4646e`.

| Service / DB | Xoá |
|---|---|
| orchestrator | `projects`, `project_events`, `qc_reports`, `project_authoring` với `project_id <> K` (7 dự án) |
| authoring | mọi bảng có `project_id` (`project_authoring`, `project_brief*`, `project_idea*`, `project_research*`, `research_pastes`, `project_story`, `project_beat*`, `project_script*`, `script_versions`, `project_glossary`, `project_handoff`, `review_units`, `voice_lines`, `authoring_segments`, `code_check_diagnostics`) với `project_id <> K` (579 dự án, phần lớn mồ côi); `DROP TABLE` `illustrations`, `illustration_folders`, `project_illustrations`, `visual_language_samples`, `visual_language_versions`, `visual_languages`; `DROP TRIGGER project_authoring_pin_language ON project_authoring`, `DROP FUNCTION pin_visual_language()` (trước khi DROP cột); `DROP COLUMN project_authoring.visual_language_id`, `visual_language_version`, `illustrations_planned_at`, `illustrations_storyboard_sha` |
| visual | `project_visual`, `project_reference_picks`, `review_units`, `visual_*_versions`, `visual_plan_seq` với `project_id <> K`; `reference_styles` R12: ghi `detail.kit` |
| llm | `llm_calls` với `project_id <> K` (giữ dòng `project_id` rỗng) |
| tts, rendering, video-assembly, publisher | `outbox_events` với `aggregate_id <> K` và đã gửi |
| `/shared` | `1332cd4d-…/` (thư mục dự án khác), `stills/` (khung tĩnh cũ). Giữ `voice_samples/`, `tts_usage.json` |

Script in số dòng sẽ xoá từng bảng, rồi mới xoá trong một transaction mỗi DB. Không đụng: thư viện prompt, thư viện tham chiếu (trừ ghi kit R12), cấu hình model, token YouTube.

## 7. Hợp đồng và tài liệu
- Mới: `docs/contracts/illustration-library-v1.md` (route ở rendering), cập nhật `visual-service-v1.md` (kit, route nội bộ), `prompt-library-v1.md` (owner rendering), `authoring-llm-code-v2.md` (`kit`), rendering `interface-contracts.md` (`render_scenes.kit`), `README.md` hàng 47–48.
- ADR-0036 → Superseded by ADR-0050.

## 8. Thứ tự làm (mỗi bước test xanh trước bước sau)
1. visual-service: kit trong phong cách tham chiếu + route nội bộ + màn hình.
2. rendering: nhận `kit` object (Remotion, stills, preview, check, render_scenes); bỏ `kits.json`.
3. orchestrator + llm-service: chuyển `kit`.
4. rendering: thư viện hình + bộ vẽ AI + prompt (duyệt prompt với Creator ở đây).
5. api-gateway + web-gui.
6. Gỡ khỏi authoring.
7. Rebuild, smoke test (dọn dữ liệu test), rồi chạy script dọn dữ liệu §6 sau khi Creator xác nhận lần cuối.

## 9. Test
- Unit: kiểm tra kit (cả hai service), domain hình, luật trạng thái, bộ vẽ (AI giả), xuất/nhập.
- HTTP: từng route thư viện hình — so thân/mã lỗi với test authoring cũ.
- Remotion: vẽ một khung bằng kit R12 khớp hash ảnh với `phatsang` hiện tại (chứng minh số chuyển đúng).
- Không gọi AI thật.

## 10. Rủi ro
- Viết lại Go→Python có thể lệch hành vi nhỏ → đối chiếu từng test HTTP cũ.
- Thư viện hình trống sau CR: Creator phải vẽ lại trước khi dùng ở bước 11.
- `/v2/code/*` của llm-service có vẻ không còn ai gọi — chỉ ghi nhận, không xử lý trong CR này.
