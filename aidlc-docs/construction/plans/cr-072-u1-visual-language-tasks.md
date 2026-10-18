# CR-072-U1 — Ngôn ngữ hình: danh sách task

Trạng thái: **chờ Creator duyệt task**. Thiết kế: `cr-072-u1-visual-language-design.md` (đã duyệt, mục "Kế hoạch thực hiện"). Nhánh `feature/cr-072-u1-visual-language`, đã merge `main`.

Mỗi task: đánh dấu `[x]` khi xong, chạy test của service đó, không commit trước khi Creator duyệt giai đoạn. Điểm dừng bắt buộc sau nhóm B: Creator xem ảnh gallery rồi mới nối vào luồng.

## A. rendering — nền móng (design bước 1)
- [ ] A1. Thêm `@remotion/noise`, `@remotion/motion-blur` (4.0.525) vào `remotion_project/package.json`.
- [ ] A2. Đặt `chromiumOptions.gl = 'swangle'` ở `render.mjs`, `layout_check.mjs`/`layout_probe_lib.mjs`, `illustration_preview.mjs`, `render_gallery.mjs`.
- [ ] A3. `conceptflow-mini/kits.ts`: `KITS = {phatsang}` với bảng token (design mục 1), `useKit()` đọc `getInputProps().kitKey`, sai hoặc thiếu thì ném lỗi rõ.
- [ ] A4. Test khói WebGL trong container rendering (báo rõ nếu `swangle` không chạy).

## B. rendering — năm hệ thống kit (design bước 2–6)
- [ ] B1. `film.tsx`: `<Shot look motionBlur?>` (chỉnh màu, hạt phim, vignette từ `ambient`, motion blur); `scene.tsx`: mờ theo độ sâu, sương theo `haze`, quầng nguồn sáng; thiếu `Shot` thì ném "thiếu look của cảnh".
- [ ] B2. `material.tsx`: `<Shape material>` với 13 vật liệu, context vị trí của `Figure`, `GroundShadow` theo hướng sáng; test viền sáng đúng phía nguồn sáng.
- [ ] B3. `rig.tsx`: sáu bộ xương, `Bone`, thư viện dáng, hành động theo mốc, chu kỳ, IK hai đoạn, biểu cảm; `Mechanism`/`Part`; dựng lại `Person`, `ReachingHand` giữ API; test góc khớp và IK.
- [ ] B4. `procedural.tsx` (bộ sinh noise tất định) và `shader.tsx` (`ShaderLayer`, 8 shader có sẵn); dựng lại sáu nền trong `backdrops.tsx`, `color`, `ground`, `light` bắt buộc; test cùng seed cùng hình.
- [ ] B5. Vẽ lại 30 hình có sẵn bằng `Shape`/vật liệu, mỗi hình khai báo mức chi tiết; `Airplane` hạng chính trên `Mechanism` có cắt lớp.
- [ ] B6. Composition `LanguageSample` (lòng quả sung sáng hồng, vũ trụ tối), gallery nhận `--kit`; dựng ảnh 16:9 và 9:16.
- [ ] **B7. ĐIỂM DỪNG: Creator xem ảnh gallery, chỉnh chất lượng trước khi nối luồng.**

## C. rendering — API và bộ kiểm (design bước 7)
- [ ] C1. `kit_key` ở `render_scenes` (`props.json`), kiểm code (`CheckBody.kit_key`, thiếu thì 422), xem trước (`kit_key`, `tier`).
- [ ] C2. `POST /v1/stills`, `GET /v1/kits`, `POST /v1/stills/prune`; cache theo hash có `kit_version`.
- [ ] C3. `illustration_style.py` theo hạng (S4, gradient, R1, M1, T1); viết lại `illustration_kit_vi.txt`; `test_illustration_kit.py` khớp export và tài liệu.

## D. llm-service (design bước 8)
- [ ] D1. `Scene.light` bắt buộc trong schema storyboard và kiểm (vai trò thật trong `palette`, giá trị 0..1).
- [ ] D2. `merger.py` sinh hằng `LOOKS`; prompt Kỹ sư dùng `<Shot look={LOOKS.<scene_id>}>`.
- [ ] D3. `kit_key` trong `/v2/code/*` và `LayoutContext`.

## E. authoring-service (design bước 9–11)
- [ ] E1. Bảng `visual_languages`, `visual_language_versions`, `visual_language_samples`; cột `project_authoring.visual_language_id/version`, `illustrations.kit_key`, `illustrations.detail_level`, `project_illustrations.tier/tier_reason`; gieo Khoa học phát sáng từ ba file luật.
- [ ] E2. Nghiệp vụ: kho ngôn ngữ (bật, sao chép, sửa tạo phiên bản), ghim khi tạo `project_authoring`, `state` nội bộ trả `kit_key`, ảnh mẫu, prune hằng ngày, HTTP admin và đổi hạng.
- [ ] E3. Luồng hiện có: người vẽ theo bản ghim và hạng, bộ lập danh sách có hạng và lý do, lọc thư viện theo kit và mức chi tiết, prompt Đạo diễn (`:332`, `:375`, `{{scene_rules}}`, `{{visual_language}}`, định dạng `light`), prompt Kỹ sư, gỡ vai trò `illustration_style`; cập nhật golden test và `illustration_kit_test.go`.

## F. orchestrator và api-gateway (design bước 12–13)
- [ ] F1. Client authoring đọc `kit_key`; `startRenderScenes` gửi `kit_key`, thiếu thì bước render thất bại "project chưa ghim ngôn ngữ hình".
- [ ] F2. api-gateway: `sendSharedFile`, `GET /v1/stills/:file`, chuyển tiếp route ngôn ngữ hình.

## G. web-gui (design bước 14)
- [ ] G1. Trang Ngôn ngữ hình (ảnh mẫu, ba luật; Bật, Sao chép, Sửa, Xoá bản sao, Dựng lại ảnh mẫu).
- [ ] G2. Chọn hạng kèm lý do ở danh sách hình của project; huy hiệu mức chi tiết ở Thư viện hình.
- [ ] G3. Mục 5 "Ngôn ngữ hình" ở bước Cấu hình đọc bản ghim thật (`GET /v1/projects/{id}/visual-language`) và nút "Xem ở Cài đặt".

## H. Script, hợp đồng, tài liệu (design bước 15)
- [ ] H1. Bước `--legacy-kit` trong `purge-legacy-data.sh` (nếu script đã có trên main, nếu chưa thì báo Creator).
- [ ] H2. `docs/contracts/`, ADR-0036, `unit-of-work.md`.

## I. Kiểm tra cuối
- [ ] I1. `tsc`, `pytest`, `go test ./...`, `npm test` toàn bộ service đổi; `tests/contracts/`.
- [ ] I2. Rebuild rendering, llm-service, authoring-service, orchestrator, api-gateway, web-gui; kiểm healthy.
- [ ] I3. Kiểm trên app theo mục "Kiểm tra" của design (hai project vũ trụ và lòng quả sung, đo thời gian render). Nếu không có `HIVE_API_KEY` thì báo rõ phần chạy AI thật chưa kiểm.
