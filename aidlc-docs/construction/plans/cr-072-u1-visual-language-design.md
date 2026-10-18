# CR-072-U1 — Ngôn ngữ hình của kênh

Trạng thái: **chờ Creator duyệt thiết kế (vòng 3)**. Nhánh `feature/cr-072-u1-visual-language`.

## Yêu cầu gốc (nguyên văn)

> "tôi duyệt hết rồi" (Creator, 2026-10-13: duyệt Application Design và Units Generation của CR-072; bước tiếp theo đã nêu là mở U1 bằng `/cr`)

> "làm CR U1 của https://claude.ai/artifact/R7tXPyTdJx3E7tUosuE2RV" (2026-10-14)

> "chỉ làm 1 ngôn ngữ tại thời điểm này.
> 1 A
> 2.B
> 3 A
> luật vẽ hình vẽ nền và vẽ cảnh sao còn đơn giản vậy, tôi muốn chi tiết và detail hơn nữa" (2026-10-14)

> "tôi thấy hướng đi này của U1 không ổn. bạn có solution nào khác để render hình nhân vật, cảnh nền v.v nói chung là ảnh minh hoạ thì còn cách nào khác không nhỉ" (2026-10-14)

> "1 không ổn ở chỗ hình quá đơn giản không giống phim. tôi không quan tâm cách làm phức tạp hay đơn giản nhưng tôi muốn đầu ra phải có chất lượng tốt
> 2. tôi không biết ngân sách có thể tối ưu sau
> 3. cái này tuỳ vào nội dung và chủ đề phải linh hoạt nếu chủ đề nói về chiếc máy bay thì ta phải mô tả nó thật rõ ràng còn nếu máy bay chỉ là nhân vật phụ thì không cần rõ ràng quá về nó
> 4 tôi đang có api của hive nhưng thời điểm này tôi muốn nói về solution trước" (2026-10-14)

> "vẫn phải vẽ bằng code remotion chứ không có AI nào cả AI hive chỉ chịu trách nhiệm sinh code remotion" (2026-10-14)

> "mục 6 thì có quá tốn công và nguy cơ AI mắc kẹt trong vòng lặp không
> mục 7 thì mẫu có thể chọn sau khi ảnh render được từ các video đầu tiên và có thể thay đổi hoặc xoá hình mẫu." (2026-10-14)

> "làm 5 cách này [1 Lớp chất phim tự động · 2 Vẽ theo hạng chính / phụ / phông · 3 Vật liệu và ánh sáng do kit tính · 4 Rig cho nhân vật và máy móc · 5 Nền thủ tục] và tính năng 7 có sẵn rồi" (2026-10-14)

> "đồng ý render chậm hơn" (2026-10-14)

> "làm vậy có khiến cho video 1 màu quá không, ví dụ nhé tôi muốn video làm về vũ trụ thì có màu tối (thực tế là như vậy) nhưng video về con sâu trong quả sung thì lại có màu sáng hồng (thực tế ruột quả sung cũng màu đó) hay làm video về rừng nhiệt đới thì hình minh hoạ phải có các gam màu như xanh lá xanh lá đậm v.v còn về đồng cỏ châu phi sẽ có màu vàng nhưng nếu ta dùng kit như hiện tại nó có đáp ứng được việc đó không hay ta phải có hướng đi khác ví dụ sau khi có kịch bản và bối cảnh câu chuyện AI sẽ tự suggest các bộ màu phù hợp với từng cảnh trong câu chuyện đó" (2026-10-17)

> "chính phụ phông phụ thuộc vào chủ đề và bối cảnh video chứ nếu như con ong mà trong bối cảnh nói về việc con ong giúp quả sung thụ tinh thì nó là nhân vật chính nhưng nếu video nói về cánh đồng hoa thì ong chỉ là phụ. ..." · "ok sửa theo bạn đề xuất vào mockup và tài liệu đi" (Creator, 2026-10-17)

> "vậy sửa U1 theo hướng này đi [Bỏ mọi chỗ ép tông tối trong luật và trong màu mặc định của sáu nền. Luật cảnh chỉ nói cách dùng ánh sáng, không quy định màu: mỗi cảnh có một nguồn sáng, viền sáng hướng về nguồn, lớp xa nhạt dần. Màu nền theo thực tế của nơi chốn, lấy từ bảng màu. Kit nhận màu từ ngoài vào: lớp chất phim và vật liệu đọc màu ánh sáng, màu không khí và tông của cảnh từ thông số mà AI đặt cho từng cảnh. Kit không tự chọn màu. Trước khi có U4a và U5, U1 dùng cơ chế đang có: Đạo diễn lập BẢNG MÀU cho cả phim và mỗi cảnh một vai trò màu nền (prompt_template_seeds.go:332). Chỉ sửa đoạn đang ép "nền sáng bão hoà" thành "màu theo thực tế của nơi chốn và chủ đề".]" (2026-10-17)

Phạm vi U1 đã duyệt trong `aidlc-docs/inception/application-design/cr-072/unit-of-work.md` mục U1: FR-18, FR-15 (luật hình và luật nền theo ngôn ngữ), FR-16; ADR-0036, ADR-0037, ADR-0039, ADR-0041.

### Quyết định đã chốt
- **Chỉ một ngôn ngữ hình**: Khoa học phát sáng. Ba ngôn ngữ kia vào backlog (thay U-3.2 A và phần "bốn ngôn ngữ" của ADR-0036). Kho ngôn ngữ, sao chép, bật, ghim vẫn làm đủ.
- **Vẽ bằng code Remotion, không AI tạo ảnh.** AI (Hive) chỉ viết code Remotion.
- **Chất lượng là ưu tiên số một**, chấp nhận render chậm hơn; ngân sách tối ưu sau.
- **Năm cách nâng chất lượng làm trong U1**: (1) lớp chất phim tự động, (2) vẽ theo hạng chính/phụ/phông, (3) vật liệu và ánh sáng do kit tính, (4) rig cho nhân vật và máy móc, (5) nền thủ tục.
- **Không làm vòng AI tự xem ảnh rồi sửa** (backlog). Sửa chất lượng qua "Vẽ lại có ghi chú" do Creator bấm (đã có).
- **Hình mẫu giữ nguyên** tính năng đang có.
- **Hạng là vai trong từng video** (Creator, 2026-10-17): AI đề xuất kèm lý do từ kịch bản và storyboard của video, Creator đổi. Thư viện hình lưu mức chi tiết của bản vẽ (Kỹ / Vừa / Bóng), không lưu hạng.
- **Sao lưu và xoá dữ liệu cũ chuyển sang U2a** (Creator, 2026-10-17). U1 chỉ xoá luật style cũ và hình vẽ theo kit cũ khi vào main.
- **Mockup đã chốt** (bản 6, 2026-10-17).
- **Tách cách vẽ khỏi màu**: ngôn ngữ hình quy định cách vẽ; màu đến từ bảng màu của video và ánh sáng của từng cảnh. Kit không tự chọn màu. AI gợi ý bảng màu thế giới và color key vẫn ở U4a và U5 như kế hoạch.
- **1 A**: một bộ component, kit là bộ token (token giữ cách tô, không giữ màu của cảnh).
- **2 B**: `kitKey` đi theo đường của `videoFont`.
- **3 A**: dọn ảnh tĩnh theo danh sách giữ lại.
- **U-3.1 A, W-2 A, U-2 A** giữ nguyên: bật khi vào main, xoá dữ liệu cũ sau khi sao lưu, không chuyển đổi project dở.

## Hiện trạng

### Vì sao hình bây giờ trông đơn giản
- Luật ép đơn giản: tối đa 4 gradient (`services/rendering/domain/illustration_style.py:29`), cấm `feTurbulence` (S1), 5–25 hình con (S4), mỗi khối một mảng bóng (S6) (`services/authoring-service/internal/domain/prompts/illustration_style_vi.txt`).
- AI tự tính bóng và sáng cho từng khối bằng `shadeOf` (`remotion_project/src/conceptflow-mini/illustration.tsx:115`).
- Không có hậu kỳ: `Scene` chỉ làm mờ lớp gần (`blurNear`), vignette màu đen cố định (`scene.tsx:186-244`); không chỉnh màu, không hạt phim, không bloom, không motion blur. `remotion_project` chỉ có `@remotion/lottie` (`package.json`).
- `Person` đổi dáng theo pose cố định, không có bộ xương (`illustration.tsx:425`); `ReachingHand` là rig duy nhất (`rig.tsx:77`).
- Sáu nền vẽ tay bằng vài khối SVG (`backdrops.tsx:92-390`).
- AI vẽ một lượt; chỉ thử lại khi mã lỗi (`illustration_drawer.go:19`, `:225`).

### Luật, prompt và màu
- Luật vẽ hình là vai trò prompt `illustration_style` (`domain/prompt_template.go:32`, `prompts/illustration_style_vi.txt`), đọc qua `IllustrationsUseCase.StyleRules` (`application/illustrations.go:176`). Luật vẽ nền là file nhúng `prompts/backdrop_style_vi.txt` (`domain/illustration.go:23-30`); prompt người vẽ ghép ở `illustration_drawer.go:153-170`.
- Prompt Đạo diễn ép tông: `prompt_template_seeds.go:332` ("MỘT MÀU NỀN SÁNG, BÃO HOÀ phủ kín khung… còn lại nền sáng, tươi") và `:375` ("TRANH PHẲNG KỂ CHUYỆN… như một cuốn truyện tranh thiếu nhi"); `illustration_kit_test.go:13` kiểm hai câu này. Cũng ở `:332`: "MỖI CẢNH BẮT BUỘC có một vai trò màu nền trong BẢNG MÀU", tối đa 12 màu.
- Storyboard là JSON (`services/llm-service/app/storyboard.py:125-145`): `palette: list[PaletteEntry]` (vai trò, hex, ý nghĩa), mỗi `Scene` có `setting` chữ tự do. Bộ ghép code sinh hằng `PALETTE` từ storyboard (`app/pipeline/merger.py:171`); prompt Kỹ sư chỉ cho dùng khoá của `PALETTE` (`app/pipeline/prompts.py:51`, `:109`).
- Kit viết cứng màu: `BACKGROUND` (`primitives.tsx:21`), `INK`, `SHADE`, `BLUSH` (`illustration.tsx:37-40`), mặc định của sáu nền (`backdrops.tsx:92-344`).
- API kit cho AI: `prompts/illustration_kit_vi.txt`, mở rộng lúc gieo (`domain/prompt_vars.go:35-41`); `rendering/tests/domain/test_illustration_kit.py` bắt lỗi khi export và tài liệu lệch nhau.

### Danh sách hình của project
- Bộ lập danh sách đọc storyboard và thư viện, AI trả danh sách (tên, mô tả, thư mục, loại, shot) (`application/project_illustrations.go:262`). Bảng `project_illustrations` (`adapters/postgres/db.go:141-155`) không có trường độ chi tiết.
- Hình mẫu: tối đa 5, chỉ hình, Creator đặt/bỏ (`application/illustrations.go:464-560`); người vẽ đọc code của chúng (`illustration_drawer.go:102-137`).

### Đường của `videoFont` (khuôn cho `kitKey`)
- orchestrator: `projects.video_font` (`orchestrator/internal/adapters/postgres/db.go:118`); `startRenderScenes` gửi `video_font` trong `render_scenes` (`application/handle_step_event.go:758-768`).
- rendering ghi `videoFont` vào `props.json` (`adapters/rendering/remotion_renderer.py:227-231`); kit đọc `getInputProps()` (`primitives.tsx:61-66`).
- Kiểm code: `llm_service_client.go:445` → llm-service `LayoutContext.video_font` (`app/pipeline/checker.py:42`) → rendering `CheckBody.video_font` (`adapters/http/check_server.py:51`) → `layout_probe_lib.mjs:236`.
- orchestrator đọc dữ liệu authoring qua `GET /internal/v1/authoring/{id}` (`orchestrator/internal/adapters/authoring/client.go:65-80`).

### Trình duyệt của Remotion
- Bốn chỗ chạy Remotion: `render.mjs`, `layout_check.mjs`/`layout_probe_lib.mjs`, `illustration_preview.mjs`, `render_gallery.mjs`. Không chỗ nào đặt chế độ đồ hoạ (`render.mjs:82`, `illustration_preview.mjs:29` chỉ có `enableMultiProcessOnLinux`). Trong container không GPU, WebGL cần `gl: 'swangle'` (đồ hoạ bằng phần mềm).

### Ảnh tĩnh, phát file, trang cài đặt, xoá project
- `POST /v1/illustrations/preview` trả PNG base64, không ghi file, một yêu cầu một lúc (`check_server.py:127`).
- api-gateway chỉ phát video cuối (`handlers/videoHandler.js`).
- Trang Kiểu video là khuôn cho trang cài đặt (`web-gui/src/pages/VideoArchetypeSettingsPage.tsx`).
- Saga xoá gọi `DeleteAuthoring` (xoá `project_illustrations`, `authoring_segments`, `project_authoring`; `prompt_template_repository.go:311-322`); `llm_usage`, `code_check_diagnostics` giữ có chủ ý (`db.go:190`).

## Yêu cầu

**FR-U1-1. Kho ngôn ngữ hình.** Tên, ghi chú tham khảo, mô tả, `kit_key`, luật vẽ hình, luật vẽ nền, luật cảnh, phiên bản, cờ hệ thống, cờ đang bật. Đúng một ngôn ngữ bật. Ngôn ngữ hệ thống chỉ đọc, sao chép để sửa. Sửa luật tạo phiên bản mới.

**FR-U1-2. Một ngôn ngữ hệ thống**: Khoa học phát sáng (`phatsang`), bật mặc định; luật là văn bản Phụ lục A, B, C. Không nhắc tên kênh nào ngoài ghi chú tham khảo.

**FR-U1-3. Bật ngôn ngữ**: ngôn ngữ đang bật tắt trong cùng giao dịch.

**FR-U1-4. Project ghim ngôn ngữ** (id, phiên bản) khi dòng `project_authoring` được tạo; mọi prompt vẽ, Đạo diễn, Kỹ sư, kiểm code, xem trước, render của project dùng bản ghim. Không có bản ghim thì báo lỗi rõ.

**FR-U1-5. Kit theo token, chọn qua `kitKey`** (input prop). Token chỉ giữ cách tô (mục 1). Thiếu hoặc sai `kitKey` thì báo lỗi rõ.

**FR-U1-6. Màu từ ngoài vào.** Kit không tự chọn màu của cảnh:
- Mỗi cảnh trong storyboard có thông số ánh sáng `light` (nguồn sáng chính, vị trí, màu, cường độ; ánh sáng phụ; không khí; lớp lấy nét; chỉnh màu), mọi màu là vai trò trong `PALETTE`.
- Bộ ghép code sinh hằng `LOOKS` từ đó; mỗi shot bọc trong `<Shot look={LOOKS.<cảnh>}>`.
- Nền nhận màu bắt buộc từ props (không mặc định). Hình có màu tự nhiên của vật làm mặc định (quả táo đỏ, lá xanh), Kỹ sư đổi theo `PALETTE`.
- Prompt Đạo diễn bỏ chỗ ép tông: màu theo thực tế của nơi chốn và chủ đề.

**FR-U1-7. Lớp chất phim** (cách 1): mọi shot có mờ theo độ sâu, sương theo khoảng cách, quầng sáng của nguồn sáng, chỉnh màu, hạt phim, vignette theo màu cảnh; motion blur cho shot máy chạy nhanh.

**FR-U1-8. Vẽ theo hạng** (cách 2): hạng chính / phụ / phông là **vai của hình trong video này**. Mỗi hình trong danh sách của project có hạng và một câu lý do do AI đề xuất từ kịch bản và storyboard; Creator đổi được trước khi vẽ. Luật và bộ kiểm có giới hạn theo hạng. Thư viện hình lưu **mức chi tiết** của bản vẽ (Kỹ / Vừa / Bóng); một hình thư viện chỉ được đưa vào project khi mức chi tiết đủ cho hạng cần.

**FR-U1-9. Vật liệu và ánh sáng do kit tính** (cách 3): khối chính vẽ bằng `<Shape material>`; kit tính màu nền, bóng lõi, bóng tiếp xúc, viền sáng, điểm bóng, vân bề mặt, ánh sáng xuyên mép theo nguồn sáng của cảnh.

**FR-U1-10. Rig** (cách 4): bộ xương cho người, thú bốn chân, chim, cá, côn trùng, giun; thư viện dáng, hành động theo mốc, chu kỳ đi/chạy/bay/bơi, IK hai đoạn, biểu cảm, chớp mắt, nguyên lý diễn hoạt. Máy móc có bộ phận quay, bản lề, trượt, và kiểu nhìn ngoài / cắt lớp / tách rời. `Person` dựng lại trên rig, giữ API.

**FR-U1-11. Nền thủ tục** (cách 5): bộ sinh nền bằng noise tất định và lớp shader WebGL có sẵn; sáu nền có sẵn dựng lại trên chúng.

**FR-U1-12. Hình có sẵn của kit** (30 hình, `Person`, `ReachingHand`, sáu nền) vẽ lại bằng API mới.

**FR-U1-13. Ảnh mẫu** của kit: hai cảnh thử, **lòng quả sung (sáng, hồng)** và **vũ trụ (tối)**, ở 16:9 và 9:16, để thấy cùng một ngôn ngữ cho hai bảng màu khác hẳn. Tự dựng khi chưa có hoặc `kit_version` đổi; Creator bấm dựng lại được.

**FR-U1-14. API ảnh tĩnh chung** `/v1/stills` (cache theo hash, `/shared/stills/`), dọn theo danh sách giữ lại (3 A).

**FR-U1-15. Phát file chung** ở api-gateway (thư mục được phép, Range, 404).

**FR-U1-16. Trang Ngôn ngữ hình**: ảnh mẫu, mô tả, ghi chú, ba luật; Bật, Sao chép, Sửa, Xoá bản sao, Dựng lại ảnh mẫu. **Hạng** (kèm lý do của AI) hiện và đổi được ở danh sách hình của project; **huy hiệu mức chi tiết** ở Thư viện hình.

**FR-U1-17. Luồng hiện có đọc ngôn ngữ đã ghim**: người vẽ đọc luật theo bản ghim và hạng; Đạo diễn đọc luật cảnh và mô tả; kiểm code, xem trước, render chạy với `kitKey`; thư viện chỉ đưa vào project hình cùng kit.

**FR-U1-18. Gỡ phần của kit cũ khi vào main**: xoá vai trò `illustration_style` và hình trong thư viện vẽ theo kit cũ (mục 11). Sao lưu và xoá project thuộc U2a.

### Tiêu chí chấp nhận
1. Trang Ngôn ngữ hình hiện bốn ảnh mẫu: lòng quả sung sáng hồng và vũ trụ tối, mỗi cảnh ở 16:9 và 9:16, cùng một lối vẽ (viền sáng, vật liệu, chiều sâu).
2. Bản sao và ghim: project tạo trước giữ bản cũ, project tạo sau ghim bản sao (tiêu chí 8 của CR-072).
3. `/v1/stills` gọi hai lần cùng nội dung: lần hai `cache_hit: true`.
4. Hai project mới, chủ đề vũ trụ và lòng quả sung, chạy trọn 13 bước: storyboard có hai bảng màu khác nhau theo chủ đề, video một tối một sáng hồng; mọi shot có lớp chất phim; hình có vật liệu và viền sáng theo nguồn sáng.
5. Một hình hạng chính (máy bay) có cánh quạt quay, cánh tà nâng hạ, xem được kiểu cắt lớp; một nhân vật trên rig đi bộ, chỉ tay, đổi biểu cảm.
6. `render_scenes` có `kit_key = "phatsang"`; xoá bản ghim thì render báo lỗi "project chưa ghim ngôn ngữ hình".
7. Sau khi U1 vào main và chạy bước gỡ kit cũ: thư viện chỉ còn hình có sẵn của kit mới và hình vẽ bằng API mới; vai trò `illustration_style` không còn.
8. Một hình được duyệt ở video A với hạng Chính có mức chi tiết Kỹ trong thư viện; video B cần hình đó làm Phụ thì dùng lại; một hình mức Bóng không được đưa vào project cần nó làm Chính.
8. Ghi thời gian render một video thật trước và sau U1 vào báo cáo (chấp nhận chậm hơn; số đo để tối ưu sau).
9. Tiêu chí xong chung của CR con (`unit-of-work.md`).

### Không làm trong U1
- Ba ngôn ngữ còn lại; vòng AI tự xem ảnh; AI viết GLSL (AI chỉ chọn shader có sẵn).
- AI gợi ý bảng màu thế giới (U4a), color key, dải màu (U5), chỉnh màu theo color key khi render (U8a), bộ VFX có tên (U7b).
- Đổi luồng, thêm bước.

## Giải pháp đề xuất

### 1. Token: cách tô, không phải màu (1 A)

`src/conceptflow-mini/kits.ts`: `KITS = {phatsang: KitTokens}`; `useKit()` đọc `getInputProps().kitKey`, sai thì ném lỗi. Token của `phatsang`:

| Nhóm | Token | Giá trị khởi đầu |
|---|---|---|
| Khối | `outline` 0 · `radius` 0.18 (theo cạnh ngắn) | không viền, bo tròn |
| Viền sáng | `rimWidth` 0.035 · `rimStrength` 0.7 | dải lưỡi liềm ở mép hướng nguồn sáng |
| Bóng | `shadowStrength` 0.35 · `aoStrength` 0.25 | bóng lõi, bóng tiếp xúc |
| Bóng loáng | `specular` {glossy 0.5, metal 0.8, glass 0.9} | điểm bóng |
| Vân | `texture` {fur 0.25, fabric 0.12, wood 0.2, stone 0.22, organic 0.15, leaf 0.12} | độ đậm của vân |
| Xuyên sáng | `sss` 0.4 | ánh sáng xuyên mép (da, lá, mô) |
| Phát sáng | `glowBlur` 0.08 · `bloom` 0.5 | quầng, loang sáng |
| Không khí | `haze` {far 0.35, mid 0.15} | phủ màu không khí theo lớp |
| Độ sâu | `dof` {sky 6, far 3, mid 1, near 10} px | mờ theo lớp |
| Phim | `grain` 0.05 · `vignette` 0.3 | hạt phim, góc tối |
| Nét chi tiết | `ink` `#141A3A` | mắt, miệng, nét mảnh |
| Xem trước | `previewLook` | look trung tính, **chỉ** cho xem trước hình lẻ và gallery, không bao giờ vào video |

Không còn bảng màu mặc định của cảnh hay của nền trong token.

### 2. Màu từ ngoài vào: `light` trong storyboard → `LOOKS` (FR-U1-6)

Đạo diễn ghi cho mỗi cảnh (storyboard JSON, `Scene.light`, bắt buộc):
```json
"light": {
  "source": "nắng chiều xuyên qua lỗ trên vỏ quả",
  "x": 0.72, "y": 0.18,
  "color": "anh_sang_chinh", "intensity": 0.8,
  "ambient": "anh_sang_phu",
  "atmosphere": "khong_khi", "haze": 0.3,
  "focus": "figures",
  "grade": {"warmth": 0.1, "contrast": 0.05, "saturation": 0}
}
```
- `color`, `ambient`, `atmosphere` là khoá vai trò trong `palette`; llm-service kiểm vai trò có thật, `x`, `y` trong 0..1, `haze`, `intensity` trong 0..1; sai thì `StoryboardError`.
- Bộ ghép code (`merger.py`, cạnh `PALETTE`) sinh `const LOOKS = {<scene_id>: {light: {x, y, color: PALETTE.…, intensity}, ambient, atmosphere: {color, density}, focus, grade}}`.
- Prompt Kỹ sư: mọi shot bọc trong `<Shot look={LOOKS.<scene_id>}>`; không tự viết look.
- Prompt Đạo diễn (`:332`): thay "Ý tự nó trừu tượng… MỘT MÀU NỀN SÁNG, BÃO HOÀ… còn lại nền sáng, tươi" bằng "màu theo thực tế của nơi chốn và chủ đề, lấy từ BẢNG MÀU; sáng hay tối là do nơi chốn quyết định"; giữ "mỗi cảnh một vai trò màu nền", tối đa 12 màu, tương phản chữ. `:375` thay đoạn phong cách bằng `{{visual_language}}`. Thêm `{{scene_rules}}` (Phụ lục C).
- U5 mở rộng `light` thành color key theo phần dàn ý; U8a đưa color key vào `grade`. U1 dựng sẵn cơ chế.

### 3. Lớp chất phim (cách 1) — `film.tsx` (mới), `scene.tsx`

- `<Shot look motionBlur?>`: đưa look vào context; áp **chỉnh màu** (CSS `contrast`, `saturate`, lớp ấm/lạnh `soft-light`), **hạt phim** (ô `feTurbulence` 256 px, dịch theo khung, `data-cf-layer="light"`), **vignette** màu tối dẫn xuất từ `ambient`, **motion blur** (`@remotion/motion-blur` `CameraMotionBlur`, 6 mẫu, chỉ khi shot đặt `motionBlur`).
- `Scene` đọc look: **mờ theo độ sâu** cho mọi lớp theo khoảng cách tới lớp lấy nét; **sương** phủ màu `atmosphere` lên lớp xa và giữa theo `haze`; **quầng nguồn sáng** (bloom kiểu `screen`) khi nguồn nằm trong khung.
- Shot không có `Shot` thì component của kit cần ánh sáng ném lỗi "thiếu look của cảnh".
- Bloom chỉ là quầng quanh nguồn sáng và vật phát sáng: DOM không có phép lọc "chỗ sáng tự loang" như phần mềm dựng phim.

### 4. Vẽ theo hạng (cách 2)

| Hạng | Dùng khi | Hình con | Bộ phận | Riêng |
|---|---|---|---|---|
| `hero` (chính) | vật/nhân vật chủ đề của phim hoặc của cảnh | 80–400 | `<Part>` đặt tên; nhân vật trên `Rig`, máy trên `Mechanism` | máy có `view` ngoài/cắt lớp/tách rời; vân qua vật liệu |
| `support` (phụ) | xuất hiện, có vai nhỏ | 20–80 | vài bộ phận chuyển động | không cắt lớp |
| `background` (phông) | đám đông, vật xa, trang trí | 3–15 | không | bóng một khối + viền sáng, không mặt |

Hạng là vai trong video, không phải tính chất của vật: con ong là Chính trong video về ong thụ phấn cho sung, Phụ trong video về cánh đồng hoa, Phông trong video về thành phố mùa xuân.

- Dữ liệu:
  - `project_illustrations.tier`, `project_illustrations.tier_reason`: hạng của hình trong video này và câu lý do của AI;
  - `illustrations.detail_level` (`rich` / `medium` / `silhouette`, hiện là Kỹ / Vừa / Bóng): mức chi tiết của bản vẽ, ghi bằng hạng mà nó được vẽ ra lần đầu.
- Bộ lập danh sách: prompt thêm hạng và lý do cho từng hình, suy ra từ logline, kịch bản và storyboard (vật mà phim hoặc cảnh nói về nó là Chính). Dùng lại hình thư viện chỉ khi mức chi tiết đủ: `rich` cho mọi hạng, `medium` cho Phụ và Phông, `silhouette` chỉ cho Phông; không đủ thì AI vẽ bản mới.
- Creator đổi hạng ở danh sách hình của project trước khi vẽ; đổi hạng hình đã vẽ thì hình thành "cần vẽ lại" khi hạng mới cần mức chi tiết cao hơn bản đã có.
- Người vẽ: prompt có mục luật của đúng hạng; xem trước và bộ kiểm nhận `tier`.
- Khi U4a, U4b vào main, hạng của nhân vật và đạo cụ chuyển lên Mỹ thuật lượt 1; danh sách hình giữ hạng đã đặt ở đó.

### 5. Vật liệu và ánh sáng do kit tính (cách 3) — `material.tsx` (mới)

- `<Shape d|ellipse|rect|capsule material color depth?>`: từ hình khối, màu gốc và nguồn sáng của cảnh, kit vẽ:
  - màu nền gradient từ phía sáng sang phía tối;
  - bóng lõi lưỡi liềm (bản sao hình dịch ngược nguồn sáng, cắt trong hình, làm mờ);
  - bóng tiếp xúc ở chỗ khối chồng nhau và mép dưới;
  - viền sáng (hình trừ bản dịch về phía nguồn sáng, màu ánh sáng chính);
  - điểm bóng (glossy, metal, glass);
  - vân (`feTurbulence` của kit theo vật liệu, cắt trong hình);
  - xuyên sáng ở mép khi ngược sáng (skin, leaf, organic);
  - phát sáng (emissive: lõi, quầng, chiếu nhẹ lên khối gần, không có bóng).
- Vật liệu: `matte`, `glossy`, `metal`, `glass`, `skin`, `fur`, `fabric`, `wood`, `stone`, `organic`, `leaf`, `liquid`, `emissive`.
- Hướng sáng tính từ vị trí của hình trong khung (`Figure` đưa vị trí, cỡ, lật, xoay vào context) tới `look.light`; bóng nhuốm màu `ambient`. Cùng một hình ở cảnh khác tự đổi sáng.
- `GroundShadow` kéo dài theo hướng sáng; vật phát sáng tạo vệt sáng thay bóng.
- Bộ kiểm: khối lớn phải là `Shape` có vật liệu; AI không tự viết filter (kit lo).

### 6. Rig (cách 4) — `rig.tsx` (mở rộng)

- `<Rig skeleton pose? actions? cycle? reach? lookAt? mood? still?>` với `skeleton` ∈ `humanoid`, `quadruped`, `bird`, `fish`, `insect`, `worm`. Da do AI vẽ: `<Bone name pivot={[x, y]}>…Shape…</Bone>`; xương lồng nhau theo cây của bộ xương; thiếu xương bắt buộc thì lỗi.
- **Dáng**: thư viện góc khớp có tên mỗi bộ xương (người: `stand`, `sit`, `wave`, `point`, `think`, `cheer`, `ouch`, `shrug`, `hold`, `reach`, `crouch`, `jump`, `lookUp`…).
- **Hành động theo mốc**: `actions=[{at: 0.1, pose: 'point'}, {at: 0.6, pose: 'cheer', hold: 0.2}]`. Kit nội suy góc khớp với lấy đà (lùi nhẹ trước khi đi), vượt đà (spring), quán tính (xương con trễ vài khung), co giãn khi tiếp đất, thở khi đứng yên.
- **Chu kỳ**: `walk`, `run`, `fly`, `swim`, `crawl`, với bước chân chạm đất.
- **IK hai đoạn** cho tay với tới `reach` và chân đặt trên mặt đất; `lookAt` xoay đầu và mắt.
- **Biểu cảm**: `<Eyes>`, `<Brows>`, `<Mouth>` của kit theo `mood`; chớp mắt tự động, lệch pha theo vị trí.
- **Máy móc**: `<Mechanism view>` với `<Part name pivot motion="rotate|hinge|slide|none" speed|angle|offset>`; cắt lớp và tách rời là hai kiểu hiện bộ phận do hình khai báo.
- `Person` dựng lại trên `humanoid`, giữ props hiện có (`pose`, `toPose`, `poseT`, `mood`, `age`, `outfit`, `talking`…), thêm `actions`, `cycle`, `reach`. `ReachingHand` dựng lại trên IK.

### 7. Nền thủ tục (cách 5) — `procedural.tsx`, `shader.tsx` (mới)

- Bộ sinh SVG bằng `@remotion/noise` (seed cố định, co theo khung, màu từ props):
  - địa hình: `Ridges`, `Hills`, `Dunes`;
  - cây cỏ: `Canopy`, `Grass`;
  - bầu trời: `Clouds`, `Stars`;
  - đá và nước: `Rocks`, `Coral`, `Waves`;
  - sinh học: `Cells` (mô kiểu Voronoi);
  - không khí: `Particles` (bụi sáng, bọt, phấn hoa).
- `<ShaderLayer shader uniforms>` WebGL2 một mặt phủ khung, thời gian = khung/fps.
  - Shader có sẵn: `sky`, `nebula`, `water`, `caustics`, `fog`, `godrays`, `tissue`, `aurora`.
  - Màu truyền từ props.
  - AI chỉ chọn shader và màu, không viết GLSL.
  - Không có WebGL thì ném lỗi rõ.
- Bốn chỗ chạy Remotion đặt `chromiumOptions.gl = 'swangle'`.
- Sáu nền có sẵn dựng lại trên bộ sinh, shader và vật liệu; `color`, `ground`, `light` thành bắt buộc.

### 8. Hình có sẵn và tài liệu kit

- 30 hình có sẵn vẽ lại bằng `Shape`/vật liệu; mỗi hình khai báo mức chi tiết; `Airplane` là hình hạng chính trên `Mechanism`, có cắt lớp.
- `illustration_kit_vi.txt` viết lại cho API mới; `test_illustration_kit.py` vẫn giữ export và tài liệu khớp nhau.
- Gallery nhận `--kit`; composition `LanguageSample` có hai cảnh thử với look riêng (quả sung sáng hồng, vũ trụ tối).

### 9. Kho ngôn ngữ, ghim, `kitKey`, ảnh tĩnh, phát file (giữ từ vòng 2)

- Dữ liệu authoring-service:
  - `visual_languages`, `visual_language_versions` (ba luật, chỉ thêm), `visual_language_samples` (theo `kit_key`, cảnh, khung);
  - `project_authoring.visual_language_id/version` ghi ở nhánh INSERT của `SaveAuthoringTopic`;
  - `illustrations.kit_key`;
  - thêm `project_illustrations.tier`, `tier_reason`, `illustrations.detail_level` (mục 4).
  - Gieo Khoa học phát sáng từ `prompts/visual_languages/phatsang_{illustration,backdrop,scene}_vi.txt`.
  - Gỡ vai trò `illustration_style` và hai file luật cũ.
- **2 B**:
  - orchestrator hỏi bản ghim qua `GET /internal/v1/authoring/{id}` (thêm `kit_key`, `visual_language_id`, `visual_language_version`) lúc gửi `render_scenes`, đặt `kit_key`; thiếu thì bước render thất bại rõ. Không lưu bản sao ở `projects`.
  - rendering ghi `kitKey` vào `props.json`.
  - Kiểm code: `/v2/code/*` → `LayoutContext.kit_key` → `CheckBody.kit_key` → `inputProps.kitKey` (thiếu thì 422).
  - Xem trước: `POST /v1/illustrations/preview` thêm `kit_key`, `tier`.
  - Ảnh tĩnh: `POST /v1/stills` có `kit_key`.
- **3 A**: `POST /v1/stills/prune {keep, min_age_hours}`, authoring-service gửi mỗi ngày.
- rendering:
  - `POST /v1/stills {composition, kit_key, props, width, height, frame}` → `{path, cache_hit}`, hash gồm `kit_version`;
  - `GET /v1/kits`;
  - ảnh tĩnh dùng chung tiến trình Node của xem trước.
- api-gateway: `sendSharedFile`, `GET /v1/stills/:file` (`^[0-9a-f]{64}\.png$`), chuyển tiếp route ngôn ngữ hình.
- HTTP authoring-service: `GET/PUT/DELETE /v1/admin/visual-languages[/{id}]`, `/activate`, `/copy`, `POST /v1/admin/visual-languages/samples`, `GET /v1/projects/{id}/visual-language`; route đổi hạng hình của project.

### 10. Bộ kiểm hình (`illustration_style.py`)

- Nhận `tier`; giới hạn theo hạng:
  - số hình con S4: hero 80–400, support 20–80, background 3–15; ngoài khoảng thì cảnh báo;
  - gradient tự viết: hero 8, support 4, background 0; quá thì lỗi.
- Giữ lỗi: thẻ cấm, filter tự viết (kit lo filter), `Math.random`/`Date.now`, thiếu `Figure`, nền thiếu `layer`, nền có mặt.
- Thêm:
  - lỗi R1: rig thiếu xương bắt buộc;
  - cảnh báo M1: khối lớn không phải `Shape` có vật liệu;
  - cảnh báo T1: hình hạng chính không chia `Part`/`Bone`.
- Bỏ: kiểm "có viền sáng" (kit tự làm).

### 11. Gỡ phần của kit cũ khi vào main

Sao lưu và xoá project nằm ở U2a (`scripts/purge-legacy-data.sh`). U1 thêm vào script đó một bước riêng `--legacy-kit`, chạy sau khi U1 vào main, có cùng cách sao lưu và xác nhận `XOA`:
1. xoá mọi dòng vai trò `illustration_style` trong `prompts`;
2. xoá `illustrations` không phải hình có sẵn của kit và có `kit_key` rỗng (vẽ theo kit cũ), cùng thư mục hình không phải hệ thống nếu đã rỗng;
3. in báo cáo.

Giữ: kiểu video, cấu hình kênh, YouTube, `/shared/stills/`, ngôn ngữ hình, hình vẽ bằng API mới.

### 12. Ảnh hưởng tới các CR con sau (sửa `unit-of-work.md` khi duyệt)

- **U7b — Rig key pose và bộ VFX**: rig và diễn theo dáng chuyển sang U1. U7b còn bộ VFX có tên (khói, mưa, lửa, lấp lánh…) lấy màu từ bảng màu, và prompt Kỹ sư gọi dáng theo key pose của animatic.
- **U4a**: dùng vật liệu, bộ sinh nền, shader của U1 cho bối cảnh và đạo cụ; thêm bảng màu thế giới vào `palette`.
- **U4b**: model sheet nhân vật dựng trên `Rig`.
- **U5**: `Scene.light` thành một phần của color key theo phần dàn ý.
- **U8a**: color key đổi `grade` của `Shot` khi render.
- **U1 trong `unit-of-work.md`**: thay phạm vi bốn kit bằng năm cách nâng chất lượng; cỡ L → XL.

## Phụ thuộc

Đã kiểm lại 2026-10-17: `origin/main` vẫn ở `20419a6`; không có nhánh `feature/cr-*`, `fix/*`, `chore/*` nào chưa vào main. **Độc lập** với việc đang mở. Trùng phạm vi trong kế hoạch CR-072 (chưa có nhánh): U7b (rig), U4a, U4b, U5, U8a, xử lý ở mục 12.

## Phạm vi

| Service | Thay đổi |
|---|---|
| rendering | `kits.ts`, `film.tsx`, `material.tsx`, `procedural.tsx`, `shader.tsx` (mới); `scene.tsx`, `rig.tsx`, `illustration.tsx`, `backdrops.tsx`, `primitives.tsx` viết lại phần lớn; thêm `@remotion/noise`, `@remotion/motion-blur` (4.0.525); `gl: 'swangle'` ở bốn entry Node; `kit_key` ở `render_scenes`, kiểm code, xem trước; `tier` ở xem trước; `/v1/stills`, `/v1/kits`, `/v1/stills/prune`; `LanguageSample`; bộ kiểm hình |
| llm-service | `Scene.light` trong schema storyboard và kiểm; bộ ghép sinh `LOOKS`; prompt code nói về `Shot`/`LOOKS`; `kit_key` trong `/v2/code/*` và `LayoutContext` |
| authoring-service | kho ngôn ngữ (domain, application, postgres, HTTP); ghim; `state` nội bộ; hạng (`tier`) ở danh sách hình, bộ lập danh sách, người vẽ, đổi hạng; prompt Đạo diễn (`:332`, `:375`, `{{scene_rules}}`, `{{visual_language}}`, định dạng `light`) và Kỹ sư; `illustration_kit_vi.txt`; gỡ `illustration_style`; `kit_key` khi kiểm code và xem trước; lọc thư viện theo kit; ảnh mẫu; dọn ảnh tĩnh |
| orchestrator | client authoring đọc `kit_key`; `startRenderScenes` gửi `kit_key` |
| api-gateway | `sendSharedFile`, `/v1/stills/:file`, chuyển tiếp route mới |
| web-gui | trang Ngôn ngữ hình; chọn hạng (kèm lý do) ở danh sách hình của project; huy hiệu mức chi tiết ở Thư viện hình (theo mockup đã chốt) |
| scripts, docs | bước `--legacy-kit` của `purge-legacy-data.sh` (script do U2a tạo); `docs/contracts/` (`render_scenes`, `/v2/code/*`, `/v1/check/remotion`, `/v1/illustrations/preview`, nội bộ authoring, storyboard `light`, `/shared/stills/`); ADR-0036 ghi chú; `unit-of-work.md` (mục 12) |

Chạy `graphify affected` ở bước chia task cho: `Scene`, `Figure`, `Person`, `ReachingHand`, `shadeOf`, các nền, `merge_remotion`/`merger`, `Storyboard`/`Scene` (llm-service), `StyleRules`, `drawerSystem`, `BackdropStyleGuide`, `RoleIllustrationStyle`, `ProjectIllustrationsUseCase.Plan`, `SaveAuthoringTopic`, `startRenderScenes`, `ScriptRenderRequest`, `LayoutContext`, `CheckBody`, `IllustrationBody`, `videoHandler`.

## Kế hoạch thực hiện

Thứ tự đặt để Creator **xem ảnh sớm**: bước 1–6 cho ra gallery và ảnh mẫu. Đề xuất một điểm dừng sau bước 6: Creator duyệt ảnh gallery trước khi nối vào luồng.

1. **rendering, nền móng**: `gl: 'swangle'`, thêm gói; `kits.ts`/`useKit`; test khói WebGL trong container.
2. **Lớp chất phim**: `Shot`, look, chỉnh màu, hạt phim, vignette, motion blur; `Scene` mờ theo độ sâu, sương, quầng.
3. **Vật liệu**: `Shape`, 13 vật liệu, context vị trí của `Figure`, `GroundShadow` theo hướng sáng.
4. **Rig**: sáu bộ xương, `Bone`, dáng, hành động, chu kỳ, IK, biểu cảm; `Person`, `ReachingHand` trên rig; `Mechanism`/`Part`.
5. **Nền thủ tục**: bộ sinh noise, `ShaderLayer` và 8 shader; sáu nền dựng lại.
6. **Hình có sẵn và gallery**: 30 hình vẽ lại, `Airplane` hạng chính; `LanguageSample`; gallery `--kit`. **Điểm dừng: Creator xem ảnh.**
7. **rendering, API**: `kit_key`/`tier` ở các route; `/v1/stills`, `/v1/kits`, prune; bộ kiểm hình theo hạng; tài liệu kit và test.
8. **llm-service**: `Scene.light`, `LOOKS`, prompt code, `kit_key`.
9. **authoring-service, dữ liệu**: bảng, cột, gieo ba luật.
10. **authoring-service, nghiệp vụ**: kho ngôn ngữ, ghim, `state`, ảnh mẫu, prune, HTTP, đổi hạng.
11. **authoring-service, luồng hiện có**: người vẽ theo bản ghim và hạng; bộ lập danh sách có hạng; prompt Đạo diễn và Kỹ sư; `kit_key` khi kiểm code và xem trước; lọc thư viện; gỡ `illustration_style`; golden test, `illustration_kit_test.go`.
12. **orchestrator**: `kit_key` ở `render_scenes`.
13. **api-gateway**.
14. **web-gui** (theo mockup đã chốt): trang Ngôn ngữ hình, chọn hạng kèm lý do, huy hiệu mức chi tiết.
15. **Script, hợp đồng, tài liệu**: bước `--legacy-kit`; `docs/contracts/`; ADR-0036; `unit-of-work.md`.

## Kiểm tra

- Test: `tsc --noEmit` (remotion_project, web-gui); test kit (rig: góc khớp và IK; vật liệu: đúng phía viền sáng theo vị trí nguồn sáng; procedural: cùng seed cùng hình); `pytest` (rendering, llm-service); `go test ./...` (authoring-service, orchestrator); `npm test` (api-gateway, web-gui); `tests/contracts/`.
- Build lại: rendering, llm-service, authoring-service, orchestrator, api-gateway, web-gui.
- Kiểm trên app:
  - gallery và ảnh mẫu (sáng hồng / tối) ở hai khung;
  - hai project chủ đề vũ trụ và lòng quả sung chạy trọn luồng;
  - máy bay hạng chính cắt lớp;
  - nhân vật đi bộ, chỉ tay;
  - đổi hạng một hình rồi vẽ lại; dùng lại hình thư viện theo mức chi tiết;
  - ghim với bản sao;
  - thời gian render trước và sau.

## Rủi ro

- **U1 rất lớn (XL)**: năm hệ thống mới trong kit. Giảm rủi ro bằng điểm dừng sau bước 6 để Creator xem ảnh trước khi nối luồng.
- **Render chậm hơn nhiều**: filter SVG trên từng khối, mờ theo lớp, WebGL bằng phần mềm (`swangle`), motion blur 6 mẫu. Đã chấp nhận; đo và báo số.
- **AI phải học API lớn hơn**: dễ gọi sai. Có tài liệu kit, tsc, bộ kiểm và vòng thử lại lỗi mã (tối đa 3) chặn; hình mẫu sẽ giúp sau vài video.
- **Da trên rig**: AI phải đặt khớp đúng chỗ; bộ kiểm bắt thiếu xương, còn khớp lệch chỉ thấy bằng mắt (Creator "Vẽ lại có ghi chú").
- **Hình hạng chính dài**: tới 400 hình con, tốn token đầu ra; ngân sách tối ưu sau.
- **Chất lượng phụ thuộc phần kit viết tay**: vật liệu, rig, shader là code người viết; điểm dừng sau bước 6 để chỉnh trước khi chạy video thật.
- **Xoá dữ liệu không quay lui được** ngoài file sao lưu.
- **Màn hình** (chọn hạng, huy hiệu mức chi tiết, trang Ngôn ngữ hình một ngôn ngữ) theo mockup đã chốt (bản 6).

---

## Phụ lục A — Luật vẽ hình của Khoa học phát sáng

`phatsang_illustration_vi.txt`, gieo làm luật vẽ hình phiên bản 1. Mã [..] là mã bộ kiểm dùng; (LỖI) chặn lưu, (cảnh báo) để Creator quyết định.

```text
# LUẬT VẼ HÌNH — KHOA HỌC PHÁT SÁNG

Phim khoa học vẽ bằng tranh vector 2D: khối hình học bo tròn, không viền, và ÁNH SÁNG KỂ CHUYỆN — mọi khối được nguồn sáng của cảnh chiếu vào: mép hướng về nguồn có viền sáng, phía kia chìm vào bóng, vật phát sáng toả quầng. Màu theo thực tế của vật và theo bảng màu của video: cảnh có thể sáng rực hay tối sâu, lối vẽ vẫn là một. Bạn DỰNG HÌNH KHỐI và CHỌN VẬT LIỆU; ánh sáng, bóng, viền sáng, vân bề mặt do kit tính theo nguồn sáng của cảnh. Hình phải đọc được trong nửa giây khi nhỏ, và giàu chi tiết khi phóng to nếu nó là vật chính.

## 0. Tinh thần
- HÌNH HỌC TRƯỚC, CHI TIẾT SAU: dựng vật từ khối cơ bản (tròn, bầu dục, viên thuốc, chữ nhật bo góc, tam giác bo đỉnh); khối nào không giúp nhận ra vật thì bỏ.
- ĐỌC BẰNG SILHOUETTE: tô kín bằng một màu vẫn đoán ra vật gì.
- MỘT ĐIỂM NHẤN: mỗi hình một chỗ thu hút mắt nhất (mắt, lõi phát sáng, bộ phận đang được giải thích).
- DÁNG CÓ HƯỚNG: vật sống nghiêng, vươn, cúi theo ý của nó; vật vô tri đặt chéo 3–8° hoặc có một chi tiết đang xảy ra.
- PHÓNG ĐẠI CÓ CHỦ Ý: thứ quan trọng cho câu chuyện to và rõ hơn thật, thứ phụ nhỏ và đơn giản hơn.

## 1. Hạng của hình (ghi kèm mô tả: vai của hình trong video này)
- [T1] CHÍNH (hero) — vật hay nhân vật mà phim hoặc cảnh nói về nó. 80–400 hình con. Chia bộ phận bằng <Part name> (máy móc) hoặc <Bone name> (nhân vật, con vật). Đủ mọi chi tiết nhận dạng và chi tiết kỹ thuật mà lời thoại có thể nhắc (máy bay: thân, cánh, cánh tà, đuôi đứng, đuôi ngang, động cơ, cánh quạt hoặc tua-bin, càng, cửa sổ buồng lái). Máy móc mà phim giải thích cấu tạo có prop view: 'outside' | 'cutaway' (bổ đôi, thấy bên trong) | 'exploded' (bộ phận tách ra theo trục). Chất liệu khác nhau cho từng phần.
- [T2] PHỤ (support) — có mặt, có vai nhỏ. 20–80 hình con. Đủ để nhận ra ngay, vài bộ phận chuyển động; không cắt lớp.
- [T3] PHÔNG (background) — đám đông, vật xa, vật trang trí. 3–15 hình con. Một khối silhouette rõ dáng, một vật liệu, không mặt, không chi tiết bên trong.
→ cảnh báo khi số hình con ngoài khoảng của hạng.

## 2. Hình khối và vật liệu
- [S1] KHỐI CHÍNH LÀ <Shape> CỦA KIT: <Shape d="…" | ellipse | rect | capsule material="…" color={…} depth={0..1} />. Tự vẽ <path>/<circle> tô phẳng chỉ cho chi tiết nhỏ (mắt, vạch, chấm, khe). KHÔNG pattern, <image>, <text>, foreignObject, KHÔNG tự viết <filter> (kit lo mờ, vân, quầng). Gradient tự viết chỉ khi vật thật sự có dải màu (cầu vồng, quang phổ): tối đa 8 (chính), 4 (phụ), 0 (phông). → LỖI.
- [S2] KHÔNG VIỀN: khối có fill không có stroke. Stroke chỉ cho chi tiết mảnh (miệng, râu, dây, tia), màu ink của kit hoặc một màu sáng. → cảnh báo.
- [S3] BO TRÒN: góc rx ≥ 6, khối lớn bo theo token radius; strokeLinecap/strokeLinejoin="round". Vật nhọn thật (gai, răng, tinh thể) bo nhẹ đỉnh 2–4 px.
- [S5] GÓC NHÌN PHẲNG: thẳng hoặc ngang, không phối cảnh 3D. Muốn có khối thì thêm MỘT mặt bên hoặc mặt trên mỏng (Shape riêng, depth thấp).
- [S5b] KHỐI LỒNG LỚP: vật phức tạp (tế bào, cỗ máy, cơ quan) là các lớp từ ngoài vào trong, lớp trong nhỏ và sáng hơn.
- [M1] MỖI KHỐI MỘT VẬT LIỆU, chọn theo vật thật:
  matte — nhựa mờ, gốm, giấy, sơn · glossy — nhựa bóng, vỏ cứng, quả chín, vỏ bọ · metal — kim loại (dải phản chiếu) · glass — kính, giọt nước, màng trong (thấy lớp sau, viền sáng mạnh) · skin — da người · fur — lông thú · fabric — vải · wood — gỗ · stone — đá, đất · organic — thịt quả, mô, nội tạng (trong mờ, sáng xuyên mép) · leaf — lá, cánh hoa (gân, xuyên sáng) · liquid — chất lỏng · emissive — vật phát sáng.
  Hai khối cạnh nhau khác vật liệu hoặc khác độ sáng rõ. → cảnh báo khi khối lớn không phải Shape có vật liệu.
- [M2] depth: 0 là mảnh phẳng (lá, tờ giấy), 1 là khối tròn đầy (quả, đầu); kit đặt độ cong của sáng tối theo nó.

## 3. Ánh sáng (kit tính — bạn KHÔNG tự vẽ)
- [L1] Không tự vẽ bóng, viền sáng, điểm bóng cho khối đã có vật liệu: kit tính theo nguồn sáng của cảnh và vị trí của hình. Bạn chỉ quyết thứ tự vẽ (khối trước đè khối sau) để kit đặt bóng tiếp xúc đúng chỗ.
- [L2] Vật phát sáng: material="emissive" và prop glow (0–1); kit tạo lõi, quầng, loang sáng và chiếu nhẹ lên khối gần. Tối đa hai chỗ phát sáng một hình.
- [L3] Vật đứng trên sàn có <GroundShadow cx cy rx />; kit kéo bóng theo hướng sáng, đổi thành vệt sáng dưới vật phát sáng.

## 4. Màu
- [S9] MÀU THEO THỰC TẾ CỦA VẬT: mặc định của prop color là màu tự nhiên dễ nhận ra (lá xanh lục, máu đỏ, ruột quả sung hồng, vỏ sung chín tím, cát vàng); Kỹ sư đổi theo PALETTE của video.
- [S10] TỐI ĐA 5 MÀU GỐC một hình (không tính màu kit dẫn xuất). Một màu chủ đạo (~60% diện tích), một–hai màu phụ cùng nhóm, một màu nhấn tương phản (~10%).
- [S10b] HÌNH PHẢI NỔI TRÊN NỀN CỦA CẢNH, dù nền sáng hay tối: giữ màu chính đổi được để Kỹ sư đẩy độ sáng khác hẳn nền.
- [S11] KHÔNG ĐEN THUẦN: nét và mắt dùng ink của kit, không #000000. → cảnh báo.
- [S12] Màu chính đổi qua prop color (và prop màu phụ nếu cần); không viết cứng màu bên trong ngoài chi tiết nhỏ. → cảnh báo.

## 5. Nhân vật, con vật, máy móc
- [S13] MẶC ĐỊNH KHÔNG CÓ MẶT. Đồ vật, công trình, nơi chốn, thiên nhiên không gắn mặt. Chỉ nhân vật được nhân hoá (mô tả ghi rõ, hoặc phải biểu lộ cảm xúc) mới có mặt; không chắc thì không vẽ mặt.
- [R1] NHÂN VẬT VÀ CON VẬT hạng chính và phụ dựng trên <Rig skeleton="humanoid|quadruped|bird|fish|insect|worm">: vẽ da cho từng <Bone name pivot={[x, y]}> theo bảng xương của kit (pivot là chỗ khớp quay, đặt ở mép chồng giữa hai bộ phận để khi xoay không hở). Thiếu xương bắt buộc → LỖI. Dáng, hành động, chu kỳ, biểu cảm KHÔNG tự viết: nhận props pose, actions, cycle, reach, lookAt, mood và chuyển nguyên cho Rig.
- [R2] MẮT VÀ MIỆNG dùng <Eyes>, <Brows>, <Mouth> của kit (mắt chấm ink có chấm sáng, lòng trắng tròn cho nhân vật nhân hoá); không tự vẽ kiểu mắt khác.
- [R3] MÁY MÓC hạng chính dựng trên <Mechanism view={view}> với <Part name pivot motion="rotate|hinge|slide|none">: cánh quạt quay theo speed, cánh tà và cửa theo angle, piston theo offset. Bộ phận nào xuất hiện ở kiểu cutaway/exploded thì khai báo ở Part đó.
- [S14] NGƯỜI: dùng <Person> (đã dựng trên rig). Chỉ vẽ người mới khi cần trang phục, dáng người mà Person không có — khi đó dựng Rig humanoid với da mới.
- [S15] CON VẬT ĐÚNG LOÀI: đặc điểm nhận dạng (tai, mõm, mỏ, sừng, vây, đuôi) bằng khối hình học; không lông từng sợi (vật liệu fur lo vân lông).
- [S15b] VI MÔ (tế bào, vi khuẩn, virus, phân tử): khối tròn hoặc viên thuốc vật liệu organic hoặc glass, màng ngoài là một Shape riêng, 2–5 bào quan bên trong; gai, roi, thụ thể là Part nhỏ phân bố đều; nhân tế bào là điểm nhấn.

## 6. Khung và toạ độ
- [S16] MỘT component export function TênPascalCase(props) bọc trong <Figure {...fig} size={fig.size ?? <cỡ>} vw={..} vh={..}>; vật lấp gần kín hộp, chừa lề 5% và chỗ cho quầng sáng, biên độ chuyển động. → LỖI khi thiếu Figure.
- [S17] Nhận FigureProps (x, y, size, rotate, flip, scale, opacity, still) và truyền nguyên cho Figure.
- [S18] Vật nhìn ngang quay đầu sang PHẢI (flip để quay trái).
- [S25] Id của gradient, clipPath, mask sinh bằng useSvgId('ten'). → cảnh báo.

## 7. Chuyển động
- [S19] Hình có Rig hoặc Mechanism: chuyển động do kit (thở, chớp mắt, quán tính, quay). Hình không có rig mà là vật sống hoặc có năng lượng: chuyển động tự thân nhỏ theo frame (nhún sin(frame/14), bập bềnh sin(frame/22)), phần mềm đung đưa trễ pha, cộng phaseOf(x, y). → cảnh báo khi hình có mặt mà không chuyển động.
- [S20] Chuyển động nhỏ, đều, lặp liền mạch, nhịp mềm; tắt được bằng still. → cảnh báo.
- [S21] Trạng thái mang nghĩa là PROP (mood, open, on, level, glow, view…), khác nhau rõ khi nhìn từ xa.
- [S22] Chỉ useCurrentFrame, interpolate, spring, Easing; KHÔNG Math.random, Date.now, CSS animation, setTimeout; vị trí "ngẫu nhiên" tính từ chỉ số. → LỖI.

## 8. Tên và mô tả
- [S23] Tên tiếng Anh PascalCase, danh từ của vật (Airplane, RedBloodCell), không mang màu hay trạng thái.
- [S24] Tên hiển thị tiếng Việt ngắn; thẻ là từ người ta sẽ tìm; dòng "Cách gọi" liệt kê mọi prop, hạng và tỉ lệ hộp: <Airplane color view /> — chính — 480×200.
```

## Phụ lục B — Luật vẽ nền của Khoa học phát sáng

`phatsang_backdrop_vi.txt`.

```text
# LUẬT VẼ NỀN — KHOA HỌC PHÁT SÁNG

Một NỀN là một NƠI CHỐN kín khung (lòng quả sung, rừng nhiệt đới, đồng cỏ, đáy biển, vũ trụ, mạch máu, căn bếp) mà nhân vật đứng bên trong. Nền vẽ cùng lối với hình (Shape có vật liệu, không viền, không chữ, không ảnh, chuyển động theo frame) và MÀU THEO THỰC TẾ CỦA NƠI CHỐN: đồng cỏ châu Phi buổi trưa vàng sáng, rừng nhiệt đới nhiều tầng xanh lá đậm nhạt, lòng quả sung hồng sáng, vũ trụ tối sâu. Ánh sáng, sương và độ mờ theo lớp do kit áp theo look của cảnh.

## 1. Cấu trúc
- [B1] BỐN LỚP: prop layer ('sky' | 'far' | 'mid' | 'near'), chỉ vẽ lớp đó, lớp trống trả null. → LỖI khi thiếu layer.
  sky: lớp xa nhất phủ kín (trời, vòm, nước sâu, khoảng không); far: dáng xa ít chi tiết (núi, rặng cây, tầng tế bào); mid: khối phía sau nhân vật, dồn về hai mép và mặt đất; near: 2–4 khối lớn sát ống kính ở góc khung (kit làm mờ).
- [B2] KÍN KHUNG, CO THEO KHUNG: useVideoConfig() và useLayerBox(), vẽ theo tỉ lệ, đẹp ở 1920×1080 và 1080×1920 (khung dọc: dồn khối mid xuống đáy, mở khoảng trời). Không Figure, không toạ độ cố định. → LỖI.
- [B3] CHỪA GIỮA KHUNG: 25%–75% bề ngang phía trên mặt đất để trống cho nhân vật; ở đó nền trơn nhất.

## 2. Màu và ánh sáng
- [B4] MÀU BẮT BUỘC TỪ PROPS: color (sắc chủ đạo của nơi chốn), ground (đất, sàn, đáy), light (màu nguồn sáng chính), không có mặc định — Kỹ sư truyền từ PALETTE. Mọi màu khác dẫn xuất bằng shadeOf từ ba màu này. Một sắc chủ đạo theo nơi chốn; sáng hay tối theo thực tế của nơi chốn.
- [B4b] NGUỒN SÁNG ĐÚNG CHỖ: nếu nguồn sáng của cảnh nằm trong khung (mặt trời, đèn, lõi phát quang, khe sáng), vẽ nó ở vị trí useSceneLight() trả về, bằng Shape material="emissive"; kit lo quầng.
- [B4c] LỚP XA ÍT CHI TIẾT, LỚP GẦN ĐẬM: lớp far ít chi tiết và độ tương phản thấp, lớp mid rõ hơn, lớp near tối và tương phản; kit phủ màu không khí lên lớp xa theo look — đừng tự phủ.
- [B5] KHÔNG MẶT: nơi chốn không có mặt, không <Face>. → LỖI.
- [B6] Prop still tắt chuyển động tự thân.

## 3. Dựng bằng bộ sinh của kit
- [G1] Địa hình và thiên nhiên dùng bộ sinh có seed cố định: <Ridges>, <Hills>, <Dunes>, <Canopy>, <Grass>, <Clouds>, <Rocks>, <Coral>, <Waves>, <Cells>, <Stars>; mỗi lớp có thể chồng 2–3 bộ sinh khác seed và khác độ sáng để có chiều sâu. Vật riêng của nơi chốn (thớ quả sung, cây baobab, rễ cây đa) tự vẽ bằng Shape có vật liệu.
- [G2] Lớp sky hoặc far có thể là <ShaderLayer shader="sky|nebula|water|caustics|fog|godrays|tissue|aurora" …màu từ props />: bầu trời chuyển màu, tinh vân, mặt nước, vân sáng đáy nước, sương, tia sáng xuyên sương, mô sinh học, cực quang. Chọn shader hợp nơi chốn; không tự viết shader.
- [G3] KHÔNG KHÍ: <Particles kind="dust|pollen|bubbles|spores|motes|stars"> ở lớp sky hoặc far theo nơi chốn (bụi đồng cỏ, phấn hoa rừng, bọt nước, bào tử, bụi sáng, sao).

## 4. Sự sống của nền
- [B8] CHUYỂN ĐỘNG CHẬM: cỏ, lá, rong, mây, sóng, tế bào trôi — biên độ nhỏ, chu kỳ ≥ 3 giây, lệch pha theo vị trí; không gì nhanh hơn nhân vật.
- [B9] Nguồn sáng và đốm phát quang thở chậm, lệch pha nhau.

## 5. Tên
- [B7] Tên tiếng Anh PascalCase kết thúc bằng Backdrop (FigInteriorBackdrop, RainforestBackdrop, SavannaBackdrop).
```

## Phụ lục C — Luật cảnh của Khoa học phát sáng

`phatsang_scene_vi.txt`, chèn vào prompt Đạo diễn ở `{{scene_rules}}`. Chỉ nói cách dùng ánh sáng; màu do nơi chốn, chủ đề và BẢNG MÀU quyết định.

```text
LUẬT CẢNH — KHOA HỌC PHÁT SÁNG (ÁNH SÁNG KỂ CHUYỆN)

- [C1] MÀU THEO THỰC TẾ: màu nền và tông của cảnh theo thực tế của nơi chốn và chủ đề (vũ trụ tối, lòng quả sung hồng sáng, rừng nhiệt đới xanh lá nhiều tầng, đồng cỏ châu Phi vàng). Sáng hay tối là do nơi chốn quyết định, không do phong cách. Mọi màu là một vai trò trong BẢNG MÀU.
- [C2] MỖI CẢNH MỘT NGUỒN SÁNG CHÍNH: ghi trong "light" của cảnh — nguồn là gì (nắng chiều qua tán lá, đèn bàn, lõi phát quang của tế bào, ngôi sao), vị trí trong khung (x, y từ 0 đến 1), vai trò màu của nó, cường độ. Đặt nguồn lệch về một phía để có hướng sáng; đặt sau lưng nhân vật chỉ khi muốn bóng ngược có chủ ý.
- [C3] ÁNH SÁNG PHỤ VÀ KHÔNG KHÍ: ghi "ambient" (vai trò màu của ánh sáng trời hoặc ánh sáng hắt — màu của bóng) và "atmosphere" + "haze" (màu và độ dày của không khí: sương rừng dày, bụi đồng cỏ vừa, nước biển dày, vũ trụ gần 0).
- [C4] VIỀN SÁNG HƯỚNG VỀ NGUỒN: mọi nhân vật và vật có viền sáng ở phía nguồn sáng (kit tự làm theo vị trí). Đặt nhân vật để phía viền sáng quay về phía người xem nhìn vào.
- [C5] LỚP XA NHẠT DẦN: bốn lớp chiều sâu; lớp càng xa càng ngả về màu không khí và càng mờ; chọn "focus" — lớp được lấy nét (thường là figures).
- [C6] ÁNH SÁNG CHỈ CHỖ CẦN NHÌN: thứ đang được nói tới nằm ở vùng sáng nhất hoặc tự phát sáng; thứ đã xong vai lùi vào bóng hay vào không khí. Ghi khoảnh khắc ánh sáng đổi trong cảnh (đèn bật, mây che, lõi sáng lên).
- [C7] ĐỔI NƠI CHỐN THÌ ĐỔI ÁNH SÁNG: mỗi nơi chốn một nguồn sáng và một không khí riêng; đi xuyên vào bên trong (quả → lòng quả → tế bào) thì nguồn sáng của lớp ngoài thành ánh sáng lọt vào lớp trong.
- [C8] Ý TRỪU TƯỢNG (số, đồ thị, so sánh) đặt trong không gian của thế giới hiện tại, cùng nguồn sáng của cảnh; màu nền lấy từ bảng màu của thế giới.
- [C9] CHỈNH MÀU NHẸ: "grade" của cảnh (warmth, contrast, saturation từ -0.3 đến 0.3) chỉ để nhấn cảm xúc; mặc định 0.
- [C10] MÁY CHẠY NHANH (lia mạnh, lao vào): ghi "motion_blur": true ở shot đó.
- [C11] CHỮ VÀ NHÃN tương phản tối thiểu 4.5:1 với vùng nó nằm trên; không đặt chữ đè quầng sáng.
```
