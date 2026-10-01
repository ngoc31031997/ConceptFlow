# CR-060 — Thiết kế: đưa video đầu ra tiến gần hai video mẫu trong `example/`

## Yêu cầu gốc (nguyên văn)

> trong thư mục example là 2 video (1 dài 1 ngắn) tôi muốn huớng đến khi làm dự án này. bạn có thể xem có cách nào để dự án hiện tại cho ra kết quả như thế không

Trả lời câu hỏi vòng 1:

> 1. A
> 2. có chứ
> 3. chọn a
> 4. bỏ
> làm luôn ba đợt
> bỏ example vào gitignore

Chốt: hướng A (nâng cấp trong kiến trúc hiện tại, không thêm model sinh ảnh, không tài sản làm tay); nới luật [S1]; short dựng dọc riêng với kịch bản riêng; video dài không đốt phụ đề; làm cả ba đợt trong CR này; thêm `example/` vào `.gitignore`.

Trả lời Q6:

> chọn (a) Mình đề xuất: gỡ nó trong một CR riêng (CR-061) làm ngay sau. Việc gỡ đổi luồng 14 bước của server nên cần review riêng. Giữa hai CR, đoạn code này vẫn còn nhưng không chạy.
> (b) Gỡ luôn trong CR-060.

Chốt Q6: (a) — gỡ đường cắt clip ở CR-061, làm ngay sau CR-060 (câu trả lời mở đầu bằng "chọn (a)"; dòng (b) được hiểu là chép kèm).

## Hai video mẫu

| | `Bi kịch bên trong quả sung.mp4` (dài) | `Tại sao lại gọi là bệnh tiểu đường-.mp4` (ngắn) |
|---|---|---|
| Thời lượng, khung | 9 phút 05 giây, 16:9, có tiếng | 58 giây, **9:16 dựng sẵn dọc**, file không có tiếng |
| Số lần cắt cảnh (ngưỡng scene 0,25) | 50 → ~10,6 s/cảnh | 12 → ~4,5 s/cảnh |
| Phụ đề đốt vào hình | Không | Không; thay bằng **chữ động** vài từ khoá ("HUYẾT ĐƯỜNG", "TIỂU ĐƯỜNG") |

Đặc điểm hình ảnh:

1. **Cảnh kín khung**: mỗi shot là một môi trường hoàn chỉnh (vườn, lòng quả sung, bếp, chợ, phòng bệnh); vật chính chiếm 40–80% chiều cao khung; nhiều cận và đại cận.
2. **Chiều sâu nhiều lớp**: tiền cảnh mờ, trung cảnh sắc, hậu cảnh nhạt; quầng sáng, tia sáng, viền sáng, vignette.
3. **Màu bão hoà, một tông chủ đạo mỗi cảnh**; bóng và điểm sáng bằng gradient mềm.
4. **Nhân vật biểu cảm, có hành động** (đang tưới cây, đang nấu ăn).
5. **Khung hình gần như không lúc nào đứng yên**: máy đẩy/lia, zoom xuyên vào vật để chuyển cảnh.
6. **Chữ trên hình rất ít**, chỉ là nhãn/từ khoá, có hoạt ảnh.

## Hiện trạng

So với hai video mới nhất hệ thống render (`data/shared_artifacts/69948fcb…`, `f7103848…`):

| Tiêu chí | Mẫu | Hiện tại (code) |
|---|---|---|
| Bố cục | cảnh kín khung | vật nhỏ giữa nền một màu. `Backdrop` chỉ là màu phẳng + dải sàn (`services/rendering/remotion_project/src/conceptflow-mini/illustration.tsx:128-148`); nền mặc định `#080E1C` (`primitives.tsx:19, 68-76`). Prompt Đạo diễn: "Nền: mặc định là #080E1C… được phép có NỀN MÀU PHẲNG riêng" (`services/authoring-service/internal/domain/prompt_template_seeds.go:336`) |
| Ánh sáng, chiều sâu | gradient, glow, blur | **bị cấm**: luật [S1] (`authoring-service/internal/domain/prompts/illustration_style_vi.txt`, mục 1); bộ kiểm chặn lưu `linearGradient`, `radialGradient`, `filter`, `feGaussianBlur` (`services/rendering/domain/illustration_style.py:17-20`); bản tự soát của hoạ sĩ AI nhắc "không gradient/filter" (`authoring-service/internal/application/drawer_prompt_vi.txt`); Đạo diễn cấm "3D, hạt/khói" (`prompt_template_seeds.go:426`) |
| Máy quay | luôn chuyển động | storyboard có trường `camera` (`prompt_template_seeds_ai.go:42`), luật L8 cho một `div` transform viết tay (`prompt_template_seeds.go:807`); thực tế đa số shot "máy đứng yên". Không có component máy quay, không có parallax |
| Kích thước vật chính | 40–80% | L14 yêu cầu ≥ 30% (`prompt_template_seeds.go:824`), chỉ là cảnh báo (`rendering/domain/layout_rules.py:47`) |
| Phụ đề | không đốt | bản nháp mới mặc định `track` (`web-gui/src/context/ProjectDraftContext.tsx:168`), nhưng 7/17 project trong DB đã chọn `burn_in`; khi đốt, prompt chừa dải đáy khung (`authoring-service/internal/domain/narration.go:146-222`) |
| Short | dựng dọc riêng | cắt từ video 16:9 rồi đặt giữa nền mờ (`video-assembly/adapters/clips/vertical_clip.py:33-37`); chế độ `short` vẫn chạy cả pipeline 16:9 (`authoring-service/internal/domain/project.go:255-287`). Khung 1920×1080 viết cứng ở: prompt (`prompt_template_seeds.go:755-786`, `prompt_template_seeds_ai.go:61`), llm-service (`app/storyboard.py:21`, `app/pipeline/merger.py:100-101`, `app/main.py:85`), rendering (`domain/layout_rules.py:45`, `remotion_project/layout_probe_lib.mjs:185,192`, `primitives.tsx:20-21`). Prompt short hiện có (`prompts/short_script.txt`) chỉ dành cho Manim |
| Nhịp cắt | 10,6 s | 4,9–8,1 s — **không phải chỗ thiếu** |

Kiến trúc liên quan đã có sẵn và sẽ dùng lại: thư mục thư viện `boi-canh` (`authoring-service/internal/domain/illustration.go:190`); bước Hình minh hoạ lập danh sách hình từ storyboard (`application/project_illustrations.go:198`, `planner_prompt_vi.txt`) rồi vẽ (`illustration_drawer.go`, `drawer_prompt_vi.txt`); layout probe miễn kiểm vùng an toàn cho phần tử phủ kín khung (`full_frame`, `remotion_project/src/layout-probe/harness.tsx:291`, `layout_rules.py:244-258`); liên kết hai project cùng chủ đề `companion_project_id` (`orchestrator/internal/domain/project.go:453-461`, `application/start_render_saga.go:151-157`).

### Giới hạn thật

Mẫu là tranh do hoạ sĩ/animator làm tay. Hướng A đưa video về **cùng ngôn ngữ hình ảnh** (kín khung, nhiều lớp, có ánh sáng, máy chuyển động, short dọc thật); độ tinh xảo từng hình vẫn thấp hơn mẫu vì hình do model viết code SVG.

## Yêu cầu

### Đợt 1 — Luật style v2, cảnh nhiều lớp, máy quay, luật bố cục

- **FR1. Luật style v2.** [S1] cho phép `linearGradient`, `radialGradient` và `feGaussianBlur` (chỉ trong `<filter>`), tối đa 4 gradient mỗi hình; vẫn cấm `pattern`, `image`, `text`, `foreignObject`, mọi `fe*` khác. Thêm [S25]: `id` của gradient/filter phải lấy từ `useSvgId()` của bộ kit (hai lần dùng cùng một hình trên một khung không được trùng id). [S6]/[S7] cho phép bóng và điểm sáng bằng gradient mềm cùng hướng sáng trái-trên.
- **FR2. Cảnh nhiều lớp.** Kit có `Scene` (lớp `sky`, `far`, `mid`, `near`; tiền cảnh `near` mờ; `vignette`; `light` quầng sáng), `Backdrop` thêm gradient trên→dưới, và các mảnh ánh sáng `Glow`, `LightRays`, `Vignette`.
- **FR3. Máy quay.** Kit có `Camera` (từ `{x, y, zoom}` sang `{x, y, zoom}`, easing, mốc bắt đầu/kết thúc theo tỉ lệ `duration`); `Scene` đọc vị trí máy để làm parallax (lớp xa trôi chậm, lớp gần trôi nhanh).
- **FR4. Luật Đạo diễn mới.** Mỗi cảnh có trường `setting` (môi trường kín khung: nơi chốn, lớp xa/giữa/gần, ánh sáng, tông màu chủ đạo); mỗi shot có chuyển động máy hoặc hành động của nhân vật; không shot nào là "vật nhỏ giữa nền trống"; ưu tiên cận cảnh; cho phép quầng sáng/tia sáng; chuyển cảnh "đi xuyên qua" bằng zoom.
- **FR5. Luật Kỹ sư mới.** Mỗi shot dựng trong `<Scene>` (hoặc `Backdrop` gradient khi cảnh trừu tượng); máy quay dùng `<Camera>` thay `div` viết tay (L8); L14 nâng lên 40%; L15 mới: không đoạn nào quá 2 giây mà khung đứng yên hoàn toàn.
- **FR6. Bộ kiểm bố cục.** Lớp của `Scene` (đánh dấu `data-cf-layer="scene"`) không bị xét vùng an toàn/vùng phụ đề; ngưỡng cảnh báo vật chính 30% → 40%.

### Đợt 2 — Thư viện nền môi trường

- **FR7. Loại hình "nền".** Hình trong thư viện có `kind`: `figure` (như hiện nay) hoặc `backdrop`. Hình nền vẽ kín khung theo kích thước khung thật (ngang hoặc dọc), vẽ theo lớp (`layer`: `sky|far|mid|near`), không mặt, độ tương phản thấp hơn vật chính, chừa vùng giữa cho nhân vật.
- **FR8. 6 nền dựng sẵn trong kit**: đồng cỏ ngoài trời, căn phòng trong nhà, đường phố, bên trong cơ thể/tế bào, dưới nước, vũ trụ. Code nằm trong kit, có hàng `builtin` trong thư mục `boi-canh`.
- **FR9. Bước Hình minh hoạ lập và vẽ nền.** Trợ lý lập danh sách đọc `setting` của từng cảnh: dùng lại nền có sẵn, hoặc đề xuất vẽ nền mới (`kind: backdrop`). Hoạ sĩ AI có bản luật/linh kiện riêng cho nền. Creator duyệt nền như duyệt hình.
- **FR10. Bước Code dùng nền.** Kỹ sư nhận danh sách nền của video và cảnh dùng nền nào; mỗi shot đặt nền đó vào `<Scene>`.
- **FR11. Giao diện.** Thư viện hình có nhãn "Nền" và xem trước nền kín khung; bước Hình minh hoạ tách "Nền của cảnh" và "Hình".

### Đợt 3 — Short dọc dựng riêng, phụ đề

- **FR12. Hướng khung theo chế độ đầu ra.** `video_output_mode = "short"` nghĩa là **dựng dọc 1080×1920 từ đầu**: Story → Visual → Hình minh hoạ → Code → Render đều theo khung dọc; không chạy bước cắt clip. `long` giữ 16:9 như nay.
- **FR13. Format short dựng sẵn** `vertical_short_60s` (30–60 giây: hook 2–5 s, ý chính 20–40 s, chốt 5–12 s). Chế độ `short` chỉ chọn được format có `max_seconds ≤ 90`; chế độ `long` chỉ chọn được format có `max_seconds > 90`.
- **FR14. Luật riêng cho khung dọc** trong prompt Đạo diễn và Kỹ sư: vật xếp theo chiều dọc, vùng an toàn dọc chừa chỗ cho giao diện Shorts (trên 200 px, dưới 360 px, phải 140 px, trái 72 px); chữ động từ khoá (≤ 3 từ, kit `KeywordText`); hook chuyển động ngay frame đầu. Mọi chỗ viết cứng 1920×1080 lấy kích thước từ khung của project.
- **FR15. Không ghép intro/outro 16:9 vào short dọc.**
- **FR16. Làm short từ video dài.** Ở màn Kết quả của một video dài có nút "Làm bản short dọc": tạo project mới cùng chủ đề, chế độ `short`, format `vertical_short_60s`, nối với video dài qua `companion_project_id`, mở bước 2 của project mới.
- **FR17. Bộ chọn đầu ra** ở bước 2: "Video dài 16:9" và "Short dọc 9:16". Lựa chọn "Cả hai" bỏ khỏi bộ chọn (thay bằng nút ở FR16). Không project nào trong DB đang dùng `short`/`both` (đã kiểm: 17/17 là `long`).
- **FR18. Phụ đề.** Mặc định `track` cho video dài (giữ như hiện tại); short dọc mặc định `off` (chữ động nằm trong hình). Bộ chọn phụ đề ghi rõ "Đốt chữ vào hình — không khuyến nghị cho video dài".

### Tiêu chí chấp nhận

- Bộ kiểm style: hình có gradient/blur qua `useSvgId` lưu được; `<pattern>`, `<image>`, `<text>`, `feDropShadow` vẫn bị chặn; id gradient viết cứng bị cảnh báo [S25].
- Một video demo trong gallery (`scene-demo.tsx`, 16:9 và 9:16) render ra file: ≥ 80% khung trong ảnh ghép 16 khung kín nền môi trường, vật chính ≥ 40% chiều cao, có parallax khi máy lia.
- Dự án `short` mới đi hết luồng tới file `final.mp4` 1080×1920, không có bước cắt clip, không có intro/outro 16:9.
- Test của mọi service bị đổi đều pass; lint/format/tsc sạch theo `docs/code-standards-rules.md`.

### Ngoài phạm vi

- Độ chi tiết như tranh vẽ tay của mẫu; model sinh ảnh; tài sản do hoạ sĩ làm.
- Thẻ thương hiệu cuối short ("Vẽ chuyện Short"): cần tài sản kênh dọc, để CR sau.
- Engine Manim: chỉ Remotion nhận các nâng cấp này. Chế độ `short` bắt buộc engine Remotion (Manim không có khung dọc).
- Gỡ đường cắt clip cũ (bước 12 "Cắt short", `generate_clips`, `ClipsPanel`, `clip_marks`/`clip_requests`, `self.clip` của Manim, prompt `short_script` và `ShortScriptAssistant`): làm ở CR-061 (Q6).

## Giải pháp đề xuất

### Vì sao hướng khung đi theo chế độ đầu ra, không theo format

Chế độ đầu ra đã có ở cả authoring-service và orchestrator (`VideoOutputMode`), nên orchestrator biết ngay project dọc hay ngang để bỏ intro/outro, không cần thêm cột. Nếu gắn hướng khung vào format thì phải thêm cột `orientation` vào bảng `video_formats` của orchestrator và hai bản domain format. Format chỉ cần được lọc theo độ dài (FR13).

### Vì sao `Scene`/`Camera` là component của kit, không chỉ là luật trong prompt

Hiện L8 để model tự viết `transform` và gần như mọi shot ra "máy đứng yên". Một component có prop rõ ràng (`from`, `to`) dễ dùng đúng hơn, parallax không thể làm bằng tay ổn định qua nhiều shot chạy song song, và bộ kiểm bố cục nhận ra lớp nền qua `data-cf-layer` thay vì đoán.

### Vì sao nền là một `kind` của thư viện hình, không phải bảng mới

Nền cần đúng vòng đời của hình: lập danh sách từ storyboard, dùng lại, vẽ, kiểm style, xem trước, Creator duyệt, đưa cho Kỹ sư. Thêm một cột `kind` dùng lại toàn bộ vòng đó; bảng mới thì phải nhân đôi.

### Kích thước khung là một khái niệm chung

Một kiểu `Frame{Width, Height, Safe{Left, Top, Right, Bottom}}` có hai giá trị:

| | Ngang | Dọc |
|---|---|---|
| Kích thước | 1920×1080 | 1080×1920 |
| Vùng an toàn | (96, 96)–(1824, 984) | (72, 200)–(940, 1560) |

Mỗi service giữ một bản hằng số (Go `domain.FrameFor`, Python `llm-service/app/frame.py`, Python `rendering/domain/frame.py`, TS `conceptflow-mini/primitives.tsx`) và có test đối chiếu chéo như cách `TestSubtitleZoneFor_MatchesTheTypeScriptCharacterForCharacter` đang làm. authoring-service gửi `frame` cho llm-service; llm-service gửi `frame` cho rendering khi kiểm; lượt render đọc kích thước từ chính `<Composition>` mà merger viết.

## Phạm vi

| Service | Thay đổi |
|---|---|
| authoring-service | luật style v2, kit text, prompt Đạo diễn/Kỹ sư (thủ công + AI), prompt lập danh sách/vẽ hình (nền), `Frame`, `{{frame}}` `{{safe_area}}` `{{frame_rules}}`, `subtitle_zone` theo khung, cột `illustrations.kind` + `project_illustrations.kind`, 6 hàng builtin nền, format `vertical_short_60s`, kiểm chế độ↔format↔engine, gửi `frame` cho llm-service |
| llm-service | `frame` trong body v2, `storyboard.py` đọc `setting` + kiểm toạ độ theo khung, `merger.py` viết width/height + import `Scene`/`Camera`, `checker.py` gửi `frame`, `main.py` bỏ giới hạn `px < 1080` |
| rendering | kit `Scene`, `Camera`, `Glow`, `LightRays`, `Vignette`, `useSvgId`, `KeywordText`, 6 nền dựng sẵn, `Backdrop` gradient; bộ kiểm style; probe theo khung + bỏ qua `data-cf-layer="scene"`; `layout_rules` theo khung; previewer vẽ nền kín khung; gallery `scene-demo.tsx` |
| orchestrator | format `vertical_short_60s` seed; bỏ intro/outro khi `short`; `short` không chạy `generate_clips` |
| video-assembly | không đổi code (PlayRes phụ đề đã đọc theo kích thước video, `ffmpeg_assembler.py:560-580`) — chỉ kiểm |
| web-gui | bộ chọn đầu ra, lọc format theo chế độ, nút "Làm bản short dọc", nhãn phụ đề, thư viện "Nền", bước Hình minh hoạ tách nền |

Hợp đồng: `docs/contracts/authoring-llm-code-v2.md` thêm `frame` (tuỳ chọn, mặc định ngang → tương thích ngược) và `setting` trong storyboard; body `/v1/check/remotion` của rendering thêm `frame`. DB: `ALTER TABLE illustrations/project_illustrations ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'figure'` (authoring). ADR mới: **ADR-0031 — Khung hình theo chế độ đầu ra; short dựng dọc riêng**.

`graphify affected`: `Backdrop` chỉ được gallery `tooth-decay-demo.tsx` gọi trực tiếp; `check_style` được `application/preview_illustration.py` và `adapters/http/check_server.py` dùng; `SubtitleZoneFor` được `render_prompt_input.go`, `render_prompt.go`, `narration.go` dùng; `merge_remotion` chỉ test merger gọi ngoài pipeline. Kit được nhúng nguyên văn vào prompt qua `illustration_kit_vi.txt`, có test giữ prompt khớp các export (`illustration_kit_test.go`, test rendering).

## Kế hoạch thực hiện

Mỗi bước kết thúc bằng test của service đó. Comment/docstring theo `docs/code-standards-rules.md` (không ghi số CR).

### Đợt 1

1. **rendering — kit ánh sáng và id** (`remotion_project/src/conceptflow-mini/illustration.tsx`): thêm `useSvgId(prefix)` (React `useId`, bỏ ký tự `:`), `Glow({cx, cy, r, color, opacity})` (circle tô `radialGradient` id từ `useSvgId`), `LightRays({cx, cy, count, length, color, opacity, still})` (tia xoay chậm theo frame), `Vignette({strength})` (`AbsoluteFill` với `radial-gradient` CSS, màu tối 0–0.45). `Backdrop` thêm prop `to?: string`: có `to` thì nền là `linear-gradient(color → to)` dọc. JSDoc cho từng export.
2. **rendering — `scene.tsx` mới** (`conceptflow-mini/scene.tsx`): `CameraContext` (`{x, y, zoom}` hiện tại); `Camera({from, to, start = 0, end = 0.85, easing, children})` nội suy theo `useCurrentFrame()` và `duration` lấy từ `useVideoConfig`/`Sequence` (prop `duration` truyền vào), bọc một `div` `transformOrigin`/`transform`, cung cấp context; `Scene({sky, far, mid, near, vignette, children})`: mỗi lớp là `AbsoluteFill` có `data-cf-layer="scene"`, phủ dư 12% mỗi cạnh, dịch theo `(camera.x - W/2) × hệ số` (sky 0, far 0.25, mid 0.6, near 1.3), `near` có CSS `filter: blur(10px)`; `children` (vật của shot) nằm giữa `mid` và `near`. `KeywordText({children, x, y, size, color, start})` chữ đậm bật lên bằng `spring`, viền tối để đọc trên nền sáng. Kích thước khung đọc từ `useVideoConfig()`, không dùng hằng 1920/1080.
3. **rendering — `primitives.tsx`**: thêm `FRAMES = {landscape: {...}, portrait: {...}}` và `useFrame()` trả `{width, height, safe}` theo `useVideoConfig()`; giữ `WIDTH`/`HEIGHT`/`SAFE_MARGIN` cho khung ngang (đang được prompt và script cũ dùng).
4. **rendering — bộ kiểm style** (`domain/illustration_style.py`): bỏ `linearGradient`, `radialGradient`, `filter`, `feGaussianBlur` khỏi `ERROR_TAGS`; thêm lỗi cho mọi thẻ `fe*` khác `feGaussianBlur`; cảnh báo [S1] khi > 4 gradient; cảnh báo [S25] khi `id="..."` literal nằm trên thẻ gradient/filter. Docstring module cập nhật. Test `tests/domain/test_illustration_style.py`: gradient qua `useSvgId` sạch; `feDropShadow`, `pattern` vẫn lỗi; id literal cảnh báo; 5 gradient cảnh báo.
5. **rendering — layout probe** (`src/layout-probe/harness.tsx`): phần tử nằm trong tổ tiên có `data-cf-layer="scene"` được ghi `layer: "scene"` và bỏ qua khi đo; `domain/layout_rules.py`: bỏ qua `layer == "scene"`, `MIN_HERO_FRACTION = 0.40`. Test với JSON mẫu: lớp `near` tràn mép không bị chặn; vật 35% có cảnh báo.
6. **rendering — kit text cho prompt**: export mới vào bộ C3 (`authoring-service/internal/domain/prompts/illustration_kit_vi.txt` và `illustration_helpers_vi.txt`): mô tả `Scene`, `Camera`, `Glow`, `LightRays`, `Vignette`, `useSvgId`, `KeywordText`, `Backdrop to`; một ví dụ shot hoàn chỉnh dùng `Scene` + `Camera`. Cập nhật test khớp kit ↔ prompt (`illustration_kit_test.go`, test rendering tương ứng).
7. **authoring-service — luật style v2** (`prompts/illustration_style_vi.txt`): viết lại S1, S6, S7; thêm S25; `drawer_prompt_vi.txt` sửa dòng tự soát ("gradient/blur chỉ qua useSvgId; không pattern/ảnh/chữ").
8. **authoring-service — prompt Đạo diễn** (`prompt_template_seeds.go` `visualDirectorHeadVI`, phần "Nền", "Cỡ cảnh", luật 14 checklist; `prompt_template_seeds_ai.go` cấu trúc JSON): thêm `"setting"` cho mỗi cảnh; luật mới "cảnh kín khung, lớp xa/giữa/gần, một tông chủ đạo, quầng/tia sáng được phép"; "mỗi shot có máy chuyển động hoặc nhân vật hành động"; "không vật nhỏ giữa nền trống"; bỏ "hạt/khói" khỏi danh sách cấm (giữ 3D, ảnh chụp); checklist tự soát thêm 2 mục tương ứng. Bản prose (thủ công) thêm dòng `BỐI CẢNH:` cho mỗi cảnh.
9. **authoring-service — prompt Kỹ sư** (thủ công `remoCVI`, `remoFVI`, `remoGVI`; AI `remoAAIVI`, `remoDAIVI` và phần dùng chung): mục C "Nền" → mỗi shot đặt trong `<Scene>` dựng theo `setting` của cảnh; L1 ngoại lệ lớp `Scene`; L8 → `<Camera>`; L14 → 40%; L15 không đứng yên quá 2 s; khuôn code ví dụ (mục D) đổi sang `Scene` + `Camera`. Cập nhật `golden_prompts_test.go`/`prompt_golden_test.go` (cập nhật golden, đọc lại diff).
10. **llm-service** (`app/storyboard.py`): thêm `setting: str` vào cảnh (tuỳ chọn khi đọc để storyboard cũ vẫn chạy, có cảnh báo khi thiếu); `merger.py` import thêm `Scene, Camera, Glow, LightRays, Vignette, KeywordText, useSvgId` từ kit. Test `tests/test_storyboard.py`, `tests/test_merger.py`.
11. **gallery** (`remotion_project/src/gallery/scene-demo.tsx`): 4 shot dùng `Scene` + `Camera` + `Glow` + kit có sẵn, đăng ký trong gallery index, render bằng `render_gallery.mjs` để làm ảnh so sánh.

### Đợt 2

12. **authoring-service — domain** (`domain/illustration.go`): `IllustrationKind` (`figure`, `backdrop`), trường `Kind` trên `Illustration` và `ProjectIllustration`; mô tả thư mục `boi-canh` đổi thành "nền môi trường kín khung: ngoài trời, trong nhà, bên trong cơ thể/vật, dưới nước, vũ trụ"; `BuiltinIllustrations` thêm 6 hàng nền (`kind: backdrop`). DB (`adapters/postgres/db.go`): `ADD COLUMN IF NOT EXISTS kind TEXT NOT NULL DEFAULT 'figure'` cho hai bảng; repository đọc/ghi `kind`. Test repository + domain.
13. **rendering — 6 nền dựng sẵn** (`conceptflow-mini/backdrops.tsx`): `MeadowBackdrop`, `RoomBackdrop`, `StreetBackdrop`, `InsideBodyBackdrop`, `UnderwaterBackdrop`, `SpaceBackdrop`; mỗi cái nhận `{layer: 'sky'|'far'|'mid'|'near', palette?: {…}, still?}` và vẽ kín khung theo `useVideoConfig()` (bố cục khác nhau cho ngang/dọc). Chúng tuân luật style v2. Test rendering khớp tên builtin ↔ export.
14. **rendering — kiểm và xem trước nền**: `domain/illustration_style.py` nhận `kind`; với `backdrop`: không đòi `<Figure>` [S16], đòi đọc kích thước từ `useVideoConfig`/`useFrame`, cấm `<Face`, đòi prop `layer`. `adapters/rendering/illustration_previewer.py` + `illustration_preview.mjs`: nền xem trước ở dạng 4 lớp chồng nhau kín khung 16:9 và 9:16. Body preview nhận `kind`.
15. **authoring-service — lập danh sách** (`application/planner_prompt_vi.txt`, `project_illustrations.go` `Plan`/`parsePlan`): output thêm `"backdrops": [{"scene": "<id cảnh>", "reuse": "<Tên>"} | {"scene", "name", "description", "folder_id": "boi-canh"}]`; mục 4 không còn loại trừ "nền màu" mà chuyển sang `backdrops`; hàng `project_illustrations` có `kind = backdrop` và `shots` = mọi shot của cảnh. Test `parsePlan` + `Plan` với storyboard có `setting`.
16. **authoring-service — vẽ nền** (`illustration_drawer.go`, file mới `drawer_backdrop_prompt_vi.txt`, `prompts/backdrop_style_vi.txt`): khi `kind = backdrop` dùng prompt nền (luật nền B1–B6: kín khung, 4 lớp, vùng giữa trống cho nhân vật, tương phản thấp hơn vật chính, không mặt, đổi được bố cục theo khung ngang/dọc), tham chiếu là các nền builtin. Test chọn đúng prompt theo `kind`.
17. **authoring-service — bước Code** (`ForCode`, `generate_authoring_code.go`): danh sách thư viện gửi llm-service đánh dấu nền và cảnh dùng nó; llm-service (`merger.library_block`, prompt chunk trong `app/pipeline/prompts.py`) ghi "Cảnh N dùng nền <Tên>: đặt các lớp vào <Scene>". Test hai phía.
18. **web-gui**: kiểu `Illustration.kind`; thư viện hình hiện nhãn "Nền" và ảnh xem trước kín khung; bước Hình minh hoạ (`IllustrationsStepPage`) chia hai nhóm "Nền của cảnh" (trên) và "Hình" (dưới), cùng thao tác duyệt/vẽ lại. Theo `docs/ux-ui-design-rules.md` (Neubrutalism, `.reveal` cho phần mở/đóng). Test vitest cho hai nhóm.

### Đợt 3

19. **ADR-0031** (`aidlc-docs/decisions/ADR-0031-frame-orientation-by-output-mode.md`): khung theo chế độ đầu ra; short dọc dựng riêng; không ghép intro/outro 16:9; Manim không hỗ trợ dọc.
20. **authoring-service — khung**: `domain/frame.go` `Frame`, `FrameFor(mode)`; `SubtitleZoneFor`/`SubtitleBand` nhận `Frame` (dải đáy tính theo chiều cao khung); biến prompt `{{frame}}` ("1920×1080 (ngang)" / "1080×1920 (dọc)"), `{{safe_area}}`, `{{frame_rules}}` (rỗng cho ngang; luật khung dọc cho dọc). Thay mọi số 1920/1080/(96,96)–(1824,984) trong prompt Đạo diễn, Kỹ sư (thủ công + AI) bằng các biến này; khuôn code mục D dùng `width={{{frame_width}}}`. `render_prompt.go`/`render_prompt_input.go` cấp biến theo project. Test: biến theo chế độ, golden prompt ngang không đổi nghĩa, test đối chiếu chéo hằng số với TS/Python.
21. **authoring-service — format và kiểm hợp lệ** (`domain/builtin_formats.go`): `FormatVerticalShort60s`; `VideoFormat.IsShort()` (`MaxSeconds ≤ ShortFormatMaxSeconds = 90`). Khi lưu cài đặt project: `short` + format không short → lỗi "Short dọc cần format ngắn (≤ 90 giây)"; `long` + format short → lỗi ngược lại; `short` + engine `manim` → lỗi "Short dọc chỉ dựng được bằng Remotion". Mặc định phụ đề khi chọn `short`: `off`. Orchestrator (`domain/builtin_formats.go`) seed cùng format (hai service đang giữ bản format riêng). Test domain + handler.
22. **authoring-service → llm-service**: body `/v2/code/*` và storyboard gửi `frame: {width, height}`; vân tay đoạn `frame` (merger) gồm kích thước khung để đổi chế độ thì viết lại. Cập nhật `docs/contracts/authoring-llm-code-v2.md`.
23. **llm-service**: `app/frame.py` (hằng số khung + vùng an toàn); `storyboard.py` kiểm toạ độ theo `frame`; `merger.py` viết `width`/`height` theo `frame`, dòng "BỐ CỤC (… px trên khung WxH)"; `main.py` `px: int = Field(gt=0, lt=<chiều cao khung>)` kiểm trong validator theo `frame`; `checker.py` gửi `frame`. Test cả hai khung.
24. **rendering — kiểm theo khung**: `adapters/http/check_server.py` nhận `frame`; `layout_checker.py`/`layout_check.mjs`/`layout_probe_lib.mjs` mở trang và `#cf-stage` theo `frame`; `domain/frame.py` + `layout_rules.py` lấy vùng an toàn theo khung. Test `layout_rules` với probe JSON khung dọc.
25. **orchestrator**: payload `assemble_video` bỏ `intro_video_path`/`outro_video_path` khi `VideoOutputMode == short`; `WantsClips()` chỉ còn đúng với `both` (giá trị cũ, không còn chọn được — xem Q6); bước 12 hiện "Không dùng" cho `short`. Test `handle_step_event`.
26. **web-gui**: `VideoOutputModePicker` còn "Video dài 16:9" / "Short dọc 9:16" (bỏ "Cả hai"; JSDoc cập nhật); bước 2 lọc format theo chế độ và tự chọn `vertical_short_60s` khi chọn short, khoá engine Remotion; `SubtitleSettings` thêm nhãn "không khuyến nghị cho video dài" cho đốt chữ; màn Kết quả của video dài có nút "Làm bản short dọc" → gọi API tạo project (`video_output_mode: "short"`, `video_format_id: "vertical_short_60s"`, `render_engine: "remotion"`, `companion_project_id`), rồi mở bước 2 của project mới; màn Kết quả hiện liên kết tới video đi kèm. Test vitest.
27. **`.gitignore`**: thêm `example/`.

## Kiểm tra

- **Test**: rendering `pytest` + `node tscheck.mjs`; authoring-service `go test ./...`, `go vet`, `gofmt`; llm-service `pytest`, `ruff check`; orchestrator `go test ./...`; web-gui `vitest`, `tsc --noEmit`, `eslint`, `prettier`.
- **Rebuild**: `rendering`, `authoring-service`, `llm-service`, `orchestrator`, `web-gui` (`docker compose build <svc>` + `up -d`), xác nhận healthy. authoring-service upsert prompt seed khi khởi động: kiểm không có prompt Creator đang bật đè lên prompt hệ thống `visual_director*`, `remotion_engineer*`; nếu có thì báo Creator, không ghi đè.
- **Trực tiếp, không tốn token**: render gallery `scene-demo` ngang + dọc, ảnh ghép 16 khung so với mẫu theo tiêu chí chấp nhận; xem trước 6 nền dựng sẵn trong thư viện; tạo một project short mới tới bước 2, kiểm bộ lọc format/engine.
- **Trực tiếp, tốn token (cần Creator bấm hoặc cho phép)**: chạy AI một chủ đề video dài và một short dọc tới file cuối; so ảnh ghép với video `69948fcb…` và hai video mẫu. `/code` không tự chạy bước này.

## Rủi ro

- Model lạm dụng glow/blur làm hình rối → giới hạn 4 gradient/hình, blur chỉ ở lớp `near` của `Scene`.
- `filter: blur` và nhiều lớp làm render chậm hơn; đo thời gian render `scene-demo` trước/sau và báo số.
- Prompt dài hơn → token bước Code tăng; đo bằng `llm_usage` ở lần chạy thật.
- Storyboard cũ không có `setting`: vẫn chạy (trường tuỳ chọn), shot của chúng không có nền môi trường.
- Hình cũ trong thư viện vẫn hợp lệ nhưng phẳng hơn hình mới.
- Kết quả vẫn không bằng tranh vẽ tay của mẫu.

## Câu hỏi cần chốt

Q1–Q6 đã trả lời (xem đầu tài liệu); Q6 chốt (a).

**Q6. Đường cắt clip cũ.** Khi short là dựng dọc riêng và bỏ "Cả hai" khỏi bộ chọn, đường cắt clip (bước 12, `generate_clips`, `vertical_clip.py`, `ClipsPanel`, `clip_marks`/`clip_requests`, `self.clip` của Manim, prompt `short_script`/`ShortScriptAssistant`; khoảng 45 file ở 6 service) không còn lựa chọn nào dẫn tới.
- **(a) — Creator chọn: tách sang CR-061 làm ngay sau CR-060.** Gỡ nó đổi luồng 14 bước của server (số bước, `flow_step` đã lưu, nhãn trên web-gui), nên tách ra để review riêng và CR-060 không phình thêm. Giữa hai CR, đường này còn trong code nhưng không chạy.
- (b) Gỡ luôn trong CR-060 (đợt 3, bước cuối).
- (c) Giữ làm công cụ phụ: ở màn Kết quả video dài, Creator vẫn cắt clip dọc nhanh được.
