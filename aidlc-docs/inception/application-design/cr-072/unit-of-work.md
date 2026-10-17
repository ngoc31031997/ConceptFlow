# CR-072 — Các CR con (Units of Work)

## Date
2026-10-13 (cập nhật 2026-10-17: thứ tự theo luồng, phạm vi U1, U2a, U7b)

Đầu vào: yêu cầu `aidlc-docs/inception/requirements/cr-072-2d-film-pipeline.md`; kế hoạch luồng `aidlc-docs/inception/plans/cr-072-workflow-plan.md`; thiết kế trong thư mục này (`components.md`, `component-methods.md`, `services.md`, `component-dependency.md`); câu trả lời U-1..U-5, U-3.1, U-3.2, U-4.1 trong `aidlc-docs/inception/plans/cr-072-unit-of-work-plan.md`.

Phụ thuộc giữa các CR con: [`unit-of-work-dependency.md`](unit-of-work-dependency.md). FR, NFR và tiêu chí chấp nhận gán vào CR con: [`unit-of-work-story-map.md`](unit-of-work-story-map.md).

## Quyết định chung

| Quyết định | Nguồn |
|---|---|
| 15 CR con. Bốn CR con cỡ XL của kế hoạch luồng được tách: U2 → U2a, U2b; U4 → U4a, U4b, U4c; U7 → U7a, U7b, U7c; U8 → U8a, U8b. Số cũ giữ, thêm chữ cái, nên cột "CR con" trong `components.md` và `application-design.md` vẫn đúng (U4 nghĩa là U4a..U4c) | U-1 A |
| Mỗi CR con là `CR-072-U<N>`, nhánh `feature/cr-072-u<N>-<slug>`, worktree riêng, vào main riêng | W-1 A |
| Làm tuần tự, mỗi lúc một CR con, **theo thứ tự các bước của luồng** (Tiền kỳ → Sản xuất → Hậu kỳ), khung chung làm trước vì mọi CR con sau dựng trên nó: U2a → U2b → U3 → U1 → U4a → U4b → U5 → U6 → U4c → U7a → U7b → U7c → U8a → U8b → U9 | U-5 B; thứ tự theo luồng (Creator, 2026-10-17) |
| Không viết code chuyển đổi cho project làm dở. Trước khi chạy `/deliver` một CR con, Creator làm xong hoặc xoá project đang dở. Project đã có video vẫn mở được Kết quả và Đăng video; mỗi CR con có test cho điều này | U-2 A |
| **Khoa học phát sáng** là ngôn ngữ hình được bật khi U1 vào main. Không có bước chờ Creator so ảnh mẫu | U-3.1 A |
| **Chỉ một ngôn ngữ hình**: Khoa học phát sáng, làm đầy đủ chi tiết. Ba ngôn ngữ còn lại (Sách tranh tươi, Giấy cắt lớp, Đường nét tối giản) không có trong CR-072, kể cả ảnh mẫu (backlog). Kho ngôn ngữ, sao chép, bật và ghim vẫn làm đủ; bản sao của Khoa học phát sáng bật được | Creator, 2026-10-14 (thay U-3.2 A) |
| **Vẽ bằng code Remotion**, không dùng AI tạo ảnh; AI chỉ viết code. Chất lượng đi trước, chấp nhận render chậm hơn. U1 làm năm cách nâng chất lượng trong kit: lớp chất phim, vẽ theo hạng chính/phụ/phông, vật liệu và ánh sáng do kit tính, rig cho nhân vật và máy móc, nền thủ tục. Không có vòng AI tự xem ảnh rồi sửa | Creator, 2026-10-14 |
| **Tách cách vẽ khỏi màu**: ngôn ngữ hình quy định cách vẽ; màu đến từ bảng màu của video và ánh sáng của từng cảnh, kit không tự chọn màu. AI gợi ý bảng màu thế giới ở U4a, color key ở U5, chỉnh màu theo color key khi render ở U8a | Creator, 2026-10-17 |
| Khung sửa theo đơn vị (U2b) áp lên cả Kịch bản và Storyboard hiện có, đúng như kế hoạch luồng. U3 và U5 chuyển các bước đó sang dạng mới, vẫn trên khung | U-4.1 A |
| Xoá dữ liệu cũ khi **U2a** vào main (CR con vào main đầu tiên), sau khi sao lưu. U1 gỡ thêm luật style cũ và hình vẽ theo kit cũ khi nó vào main | W-2 A; dời từ U1 sang U2a (Creator, 2026-10-17) |
| Luồng thêm bước dần theo CR con, không có bước giả: 13 bước hiện có → 17 bước khi U8b xong | Kế hoạch luồng ("bước chưa làm thì giữ như hiện nay") |

**Thay đổi so với yêu cầu đã duyệt:** FR-18 ghi "Trước CR con đầu tiên, AI dựng bộ ảnh mẫu cho cả bốn, Creator so rồi bật một (P-4 A)". Theo U-3.1 A, Creator chọn Khoa học phát sáng ngay, nên không còn cổng chọn ngôn ngữ. Theo quyết định 2026-10-14, chỉ Khoa học phát sáng có trong CR-072; ba ngôn ngữ còn lại không có ảnh mẫu. Trang Ngôn ngữ hình, việc chỉ một ngôn ngữ bật, sao chép và việc project ghim phiên bản vẫn giữ như FR-18. FR-16 (xoá dữ liệu cũ) chuyển từ U1 sang U2a.

## Cách làm mỗi CR con

1. `/cr`: mở nhánh và worktree, kiểm việc trùng với CR khác, viết bản thiết kế (`aidlc-docs/construction/plans/cr-072-u<N>-<slug>-design.md`). Bản thiết kế là **Functional Design** của CR con: dữ liệu, schema, luật, prompt, mã lỗi, màn hình. Với U7a và U8a, bản thiết kế có thêm phần **NFR Requirements và NFR Design**. Infrastructure Design bỏ (kế hoạch luồng).
2. Creator duyệt thiết kế, rồi duyệt danh sách task.
3. `/code`: làm từng task, chạy test, build lại và khởi động lại các service đã đổi.
4. Creator kiểm trên app, rồi `/deliver`: commit, push, merge vào main, làm mới graphify, build lại các service đã đổi.
5. Creator chạy một video thật qua luồng để đo tỉ lệ duyệt ngay lần đầu (NFR-2).

### Tiêu chí xong chung

Áp cho mọi CR con, ngoài tiêu chí riêng ở từng mục dưới:
- Test của mọi service đụng tới xanh. Test hợp đồng của luồng (ba bản định nghĩa) và của `/shared` (`tests/contracts/test_shared_artifacts_contract.py`) xanh khi CR con đổi chúng.
- Các service đã đổi được build lại, khởi động lại và báo healthy.
- Một project mới chạy trọn luồng tới video đăng được.
- Project đã có video vẫn mở được Kết quả và Đăng video (U-2 A).
- Mọi bước mới có trạng thái chạy, lỗi, huỷ, chạy lại; không có đường giả thành công (NFR-3).
- Bước mới gọi AI ghi token và thời gian vào `llm_usage` (NFR-1).
- Prompt hệ thống của bước mới hoặc bước đổi được viết lại theo luồng mới và đọc luật của ngôn ngữ hình đã ghim (FR-15).
- Giao diện theo `docs/ux-ui-design-rules.md`, code theo `docs/code-standards-rules.md` (NFR-5, NFR-6). Nội dung chạy được ở tiếng Việt và tiếng Anh (NFR-7).

---

## U1 — Ngôn ngữ hình của kênh

- **Nhánh**: `feature/cr-072-u1-visual-language` · **Cỡ**: XL · **Phụ thuộc**: — (làm sau U3 theo thứ tự; phải có trước U4a)
- **FR**: FR-18, FR-15 (luật hình, luật nền, luật cảnh theo ngôn ngữ), FR-11 (rig và dáng)
- **Service**: rendering, llm-service, authoring-service, orchestrator, api-gateway, web-gui
- **Phạm vi**:
  - `VisualLanguageStore` (authoring-service): ngôn ngữ hình có phiên bản; đúng một ngôn ngữ bật; ngôn ngữ hệ thống chỉ đọc, sao chép để sửa; project ghim id và phiên bản lúc tạo (bước Khởi tạo).
  - Một ngôn ngữ hệ thống: Khoa học phát sáng, với luật vẽ hình, luật vẽ nền và luật cảnh; luật cảnh chỉ nói cách dùng ánh sáng, không quy định màu.
  - Kit `phatsang` (rendering), token chỉ giữ cách tô, chọn qua input prop `kitKey` (đi theo đường của `videoFont`); năm cách nâng chất lượng: lớp chất phim (`Shot`, `Scene`), vẽ theo hạng chính/phụ/phông, vật liệu và ánh sáng do kit tính (`Shape`), rig (bộ xương, dáng, hành động, chu kỳ, IK, biểu cảm; `Mechanism` cho máy móc), nền thủ tục (noise, shader WebGL có sẵn). Hình có sẵn và sáu nền vẽ lại. `GET /v1/kits`.
  - Màu từ ngoài vào: `light` của từng cảnh trong storyboard (vai trò trong bảng màu), bộ ghép code sinh `LOOKS`; prompt Đạo diễn bỏ chỗ ép "nền sáng bão hoà".
  - `StillRenderService`: `POST /v1/stills` (cache theo hash nội dung), `/shared/stills/{hash}.png`, dọn ảnh không còn ai trỏ tới. Composition `LanguageSample` với hai cảnh thử (lòng quả sung sáng, vũ trụ tối).
  - api-gateway: một cách phát file chung từ `/shared` (ADR-0041).
  - web-gui: trang cài đặt Ngôn ngữ hình; chọn hạng ở danh sách hình của project; huy hiệu hạng ở Thư viện hình.
  - Luồng hiện có (Hình minh hoạ, Code, Render) đọc luật của ngôn ngữ đã ghim và dùng kit Khoa học phát sáng; gỡ vai trò `illustration_style`; xoá hình vẽ theo kit cũ.
  - Thiết kế: `aidlc-docs/construction/plans/cr-072-u1-visual-language-design.md`.
- **Giao cho CR con sau**: `VisualLanguageStore.Active/PinForProject`, `/v1/stills`, phát file chung, kit `phatsang` (lớp chất phim, vật liệu, rig, bộ sinh nền, shader), `Scene.light`/`LOOKS`.
- **Tiêu chí xong riêng**:
  - Trang Ngôn ngữ hình hiện ảnh mẫu lòng quả sung (sáng) và vũ trụ (tối) ở hai khung, cùng một lối vẽ.
  - Đổi ngôn ngữ đang bật (sang một bản sao của Khoa học phát sáng) không đổi project đã tạo; project mới ghim ngôn ngữ mới (tiêu chí chấp nhận 8).
  - Gọi `/v1/stills` hai lần với cùng nội dung thì lần hai dùng lại file (`cache_hit`).
  - Hai project chủ đề vũ trụ và lòng quả sung cho hai bảng màu khác nhau, video một tối một sáng; một hình hạng chính có bộ phận chuyển động và kiểu cắt lớp; một nhân vật trên rig đi bộ và đổi biểu cảm.
  - Luồng sau khi vào main: 13 bước.

## U2a — Luồng ba giai đoạn

- **Nhánh**: `feature/cr-072-u2a-three-phase-flow` · **Cỡ**: M · **Phụ thuộc**: — (làm đầu tiên)
- **FR**: FR-1 (phần thanh bước và tên bước), FR-16
- **Service**: orchestrator, authoring-service, web-gui; script vận hành
- **Phạm vi**: `FlowDefinition` ở ba bản chia 13 bước hiện có vào Tiền kỳ, Sản xuất, Hậu kỳ (`PhaseOf`); `StepRail` theo giai đoạn; tiêu đề màn; đổi tên các bước đã có theo tên trong luồng mục tiêu (tên cụ thể chốt ở Functional Design). Test hợp đồng giữ ba bản khớp nhau.
- Script sao lưu và xoá dữ liệu cũ (FR-16, W-2 A): có bước xác nhận; dump DB của orchestrator và authoring-service, nén `/shared`; xoá mọi project qua saga xoá, `llm_usage`, `code_check_diagnostics`, prompt Creator đã sửa; giữ kiểu video, cấu hình kênh, kết nối YouTube, thư viện hình; chạy từ primary checkout ngay sau khi U2a vào main. Thư viện hình và luật style cũ do U1 xoá khi U1 vào main.
- **Giao cho CR con sau**: cách thêm một bước vào một giai đoạn ở cả ba bản.
- **Tiêu chí xong riêng**: thanh bước hiện ba giai đoạn; số bước, tên bước, nhãn sự kiện tiến độ khớp nhau giữa server và web-gui; sao lưu tạo được file dump DB và file nén `/shared` trước khi xoá, báo cáo ghi rõ đã xoá gì, kiểu video, cấu hình kênh, kết nối YouTube còn nguyên. Luồng: 13 bước.

## U2b — Khung sửa theo đơn vị

- **Nhánh**: `feature/cr-072-u2b-per-unit-review` · **Cỡ**: L · **Phụ thuộc**: U2a
- **FR**: FR-17, NFR-2 (đếm lần làm lại)
- **Service**: authoring-service, api-gateway, web-gui
- **Phạm vi**:
  - `ReviewUnitStore`, `DependencyTracker`, `RedoService`, giao diện `UnitGenerator` (authoring-service), theo ADR-0035; HTTP `/v1/projects/{id}/units/...`.
  - Bộ thành phần đơn vị ở web-gui: thẻ đơn vị (trạng thái, duyệt, làm lại kèm ghi chú, sửa tay), thanh gom ghi chú, hộp "sẽ làm lại", nút duyệt tất cả mục còn lại.
  - Áp lên Kịch bản hiện có và Storyboard hiện có (U-4.1 A). Phần phụ thuộc đang có (hình minh hoạ, đoạn code) được đánh "cần làm lại" theo đúng phần bị ảnh hưởng. Loại đơn vị cụ thể của hai bước này chốt ở Functional Design.
- **Giao cho CR con sau**: khung để mỗi CR con sau đăng ký loại đơn vị của mình bằng một `UnitGenerator`.
- **Tiêu chí xong riêng**: sửa tay hoặc làm lại một cảnh storyboard chỉ đánh "cần làm lại" phần phụ thuộc của cảnh đó, và màn hình báo trước danh sách đó; phần Creator tự viết không bị AI ghi đè trừ khi Creator bấm làm lại; bước đi tiếp được khi mọi đơn vị đã duyệt; số lần làm lại được ghi theo loại đơn vị. Luồng: 13 bước.

## U3 — Kịch bản mới

- **Nhánh**: `feature/cr-072-u3-script-logline` · **Cỡ**: M · **Phụ thuộc**: U2b
- **FR**: FR-2, FR-19
- **Service**: authoring-service, api-gateway, web-gui (llm-service chỉ được gọi qua `/v1/chat` như hiện nay)
- **Phạm vi**: `ScriptAuthoring` (logline, tóm tắt, dàn ý theo beat có trường "Kiểm chứng", lời thoại theo cảnh; ô Creator đã viết thì AI dùng chữ của Creator); đơn vị `logline`, `beat`, `script_scene` thay đơn vị Kịch bản cũ của U2b; `ArchetypeAdvisor` (3–5 gợi ý, kiểu mới kèm playbook nháp, lưu vào bảng Kiểu video); màn Kịch bản và gợi ý kiểu video ở bước Khởi tạo; prompt các vai của bước Kịch bản.
- **Tiêu chí xong riêng**: đổi logline thì dàn ý thành "cần làm lại"; làm lại một cảnh lời thoại kèm ghi chú không đụng cảnh khác; nhập chủ đề thì có gợi ý kiểu video kèm lý do và câu mở mẫu, và "Để AI tự chọn", "Xem tất cả kiểu" vẫn còn. Luồng: 13 bước.

## U4a — Mỹ thuật lượt 1: thế giới, bối cảnh, đạo cụ

- **Nhánh**: `feature/cr-072-u4a-art-world` · **Cỡ**: L · **Phụ thuộc**: U1, U2b, U3
- **FR**: FR-3 (mục 1, 2, 4), FR-5, FR-15 (phần luật thế giới), NFR-4
- **Service**: authoring-service, rendering, api-gateway, web-gui
- **Phạm vi**:
  - `ArtBible` lượt 1 phần thế giới: mô tả thế giới (nơi chốn, thời đại, chất liệu, logic ánh sáng, bảng màu thế giới) và bảng tham khảo; bối cảnh (nền 4 lớp ở 16:9 và 9:16, biến thể ánh sáng khi cần); đạo cụ. Đơn vị `art_world`, `art_location`, `art_prop`.
  - Phong cách theo thế giới (ADR-0037): luật thế giới trong prompt, lớp chất liệu theo chủ đề ở nền và đạo cụ.
  - Ảnh Creator tải lên cho bảng tham khảo (`/shared/{id}/uploads/moodboard/`). Functional Design kiểm model vẽ có đọc được ảnh không (P-5 A); không đọc được thì ảnh chỉ để Creator đối chiếu.
  - Bốn cách sửa một hình: làm lại kèm ghi chú, sửa mô tả rồi vẽ lại, tải SVG lên (`/shared/{id}/uploads/svg/`), thay bằng hình trong thư viện.
  - Composition `ArtSheet` (bối cảnh, đạo cụ) trong kit `phatsang`.
  - Bước mới "Mỹ thuật lượt 1" sau Kịch bản; màn có các tab Thế giới, Bối cảnh, Đạo cụ.
- **Tiêu chí xong riêng**: hai chủ đề khác nhau (ví dụ vũ trụ, lòng quả sung) cho hai thế giới khác hẳn về bối cảnh, màu, ánh sáng, chất liệu, cùng ngôn ngữ hình (tiêu chí chấp nhận 2, phần thế giới); nền đúng ở cả hai khung hình; Storyboard hiện có đọc bộ mỹ thuật đã duyệt. Luồng: 14 bước.

## U4b — Nhân vật và thư viện nhân vật

- **Nhánh**: `feature/cr-072-u4b-characters` · **Cỡ**: L · **Phụ thuộc**: U1, U2b, U4a
- **FR**: FR-3 (mục 3), FR-4
- **Service**: authoring-service, llm-service, rendering, web-gui
- **Phạm vi**:
  - Nhân vật cấp video, đơn vị `art_character`: model sheet trước/nghiêng/sau, biểu cảm, tư thế tay, nhân vật đặt trên từng nền chính (`ArtSheet` nhân vật trong kit `phatsang`). Miệng giữ hình cố định theo biểu cảm, bỏ nhép miệng.
  - Schema storyboard (llm-service): khai báo nhân vật một lần; shot chỉ tham chiếu nhân vật kèm biểu cảm và tư thế. Storyboard hiện có dùng schema này.
  - `CharacterLibrary` và trang cài đặt Thư viện nhân vật; lưu nhân vật đã duyệt, dùng lại ở video sau.
  - Tab Nhân vật trong màn Mỹ thuật lượt 1, cùng bốn cách sửa hình của U4a.
- **Tiêu chí xong riêng**: shot không đặt lại dáng, tuổi, áo của nhân vật; một nhân vật lưu vào thư viện được chọn lại ở project sau. Luồng: 14 bước.

## U4c — Hình minh hoạ theo bộ mỹ thuật

- **Nhánh**: `feature/cr-072-u4c-illustrations-from-art` · **Cỡ**: M · **Phụ thuộc**: U2b, U4b (làm sau U6 theo thứ tự: bước 9 Hình minh hoạ)
- **FR**: FR-13
- **Service**: authoring-service, web-gui
- **Phạm vi**: `IllustrationPlanner` chỉ đưa vào danh sách phần còn thiếu sau bộ mỹ thuật; hình của bộ mỹ thuật hiện là đã duyệt và chỉ sửa ở mục gốc; đơn vị `illustration` có "làm lại kèm ghi chú"; giữ vẽ song song, duyệt, sửa, bỏ, sửa prompt như hiện nay.
- **Tiêu chí xong riêng**: một project có bộ mỹ thuật đã duyệt không vẽ lại bối cảnh, nhân vật, đạo cụ đã có; sửa một mục gốc thì các hình dùng nó thành "cần làm lại". Luồng: 14 bước.

## U5 — Storyboard mới và Mỹ thuật lượt 2

- **Nhánh**: `feature/cr-072-u5-storyboard-color-key` · **Cỡ**: L · **Phụ thuộc**: U1, U2b, U3, U4a, U4b
- **FR**: FR-3 (mục 5, 6), FR-6, FR-7, FR-8 (phần storyboard), NFR-4
- **Service**: authoring-service, llm-service, rendering, web-gui
- **Phạm vi**:
  - Storyboard theo cảnh dạng mới, đơn vị `board_scene`: cách nối sang cảnh sau và vật nối dùng chung (hành trình máy quay); cảnh giải thích ý đặt trong thế giới, thay FR7 "nền sáng bão hoà mặc định" của CR-067 (ADR-0037); prompt Đạo diễn viết lại.
  - Mỹ thuật lượt 2: đơn vị `color_key` (mỗi phần dàn ý một khung màu; sửa tay bằng bộ chọn màu; kiểm tương phản chữ, hai cảnh liền nhau không quá giống, không lệch bảng màu thế giới; một khoảnh khắc phá bảng màu ở cao trào) và `style_frame` (3–5 khung mẫu). Composition `ColorKeyFrame`, `StyleFrame`. Dải màu của cả video trên một hàng.
  - Bước mới "Mỹ thuật lượt 2" sau Storyboard.
- **Tiêu chí xong riêng**: dải màu cho thấy color key đổi theo từng phần dàn ý (tiêu chí chấp nhận 6); không cảnh giải thích nào chuyển sang nền màu đặc; mỗi chỗ nối liền mạch ghi vật nối. Luồng: 15 bước.

## U6 — Giọng sơ bộ, sửa giọng, animatic

- **Nhánh**: `feature/cr-072-u6-draft-voice-animatic` · **Cỡ**: L · **Phụ thuộc**: U1, U2b, U4a, U4b, U5
- **FR**: FR-9, FR-10, FR-20, NFR-4
- **Service**: orchestrator, authoring-service, tts, rendering, api-gateway, web-gui
- **Phạm vi**:
  - `DraftVoiceDispatcher` (orchestrator), lệnh `synthesize_lines` và sự kiện `lines_synthesized` (ADR-0040); `SynthesizeLines` (tts): "đọc là", tốc độ, khoảng nghỉ bằng SSML Azure, cache theo nội dung và cách đọc; file `/shared/{id}/audio/{line}_{cache_key}.wav`.
  - `PronunciationDictionary` và trang cài đặt Từ điển phát âm; `VoiceLines` (đơn vị `voice_line`), sửa chữ hoặc sửa cách đọc từng câu, đọc lại đúng câu đó.
  - `AnimaticPlan` (đơn vị `animatic_shot`), composition `AnimaticStill`, `AnimaticPlayer` phát trong trình duyệt (ADR-0039); sửa thời lượng, sửa câu, đổi khung tĩnh kèm ghi chú; ghi chú nhịp cả cảnh, AI đề xuất, Creator xem rồi áp dụng.
  - Bước mới "Giọng sơ bộ" và "Animatic" sau Mỹ thuật lượt 2. Bước "Giọng đọc" của saga rời thanh bước; saga vẫn gửi `synthesize_speech` và dùng lại file đã có theo cùng `cache_key`.
- **Tiêu chí xong riêng**: Creator xem animatic có giọng trước khi bước Code chạy (tiêu chí chấp nhận 3); một từ sửa trong từ điển được đọc đúng ở video sau (tiêu chí chấp nhận 10); sửa một câu chỉ đọc lại câu đó và chỉ dựng lại khung animatic của shot đó. Luồng: 16 bước.

## U7a — Code theo cảnh, lỗi theo shot, xem nhanh shot

- **Nhánh**: `feature/cr-072-u7a-code-per-scene` · **Cỡ**: L, có NFR · **Phụ thuộc**: U1, U2b, U4c, U5
- **FR**: FR-22, FR-8 (phần code: hai cảnh dùng chung vật nối)
- **Service**: authoring-service, llm-service, rendering, api-gateway, web-gui
- **Phạm vi**:
  - `CodeSegments`: đoạn code là một cảnh, mỗi shot một trạng thái (đơn vị `code_shot`); bỏ lượt AI tự sửa; giữ autofix máy móc của CR-068 và ghi rõ đã sửa gì; shot lỗi hiện đúng shot và thông báo cụ thể, Creator chọn chạy lại, chạy lại kèm ghi chú, sửa tay hoặc dán kết quả AI ngoài.
  - llm-service `/v2/code/generate` nhận `segment_kind = "scene"`, trả chẩn đoán theo shot. Đoạn theo số shot của CR-050 bỏ khi U7a xong (đổi hợp đồng theo ba bước).
  - `ShotPreview`: `POST /v1/previews/shot` (480p), file `/shared/{id}/previews/`; xem nhanh ở màn Code.
  - **NFR**: thời gian mục tiêu của xem nhanh một shot (kế hoạch luồng: khoảng 10 giây).
- **Tiêu chí xong riêng**: shot lỗi hiện đúng shot và lỗi, không bị AI tự sửa (tiêu chí chấp nhận 9, phần Code); một nhân vật giữ đúng model trên mọi shot (tiêu chí chấp nhận 4); xem nhanh một shot đạt mục tiêu NFR. Luồng: 16 bước.

## U7b — Bộ VFX và gọi dáng theo key pose

- **Nhánh**: `feature/cr-072-u7b-key-pose-vfx` · **Cỡ**: M · **Phụ thuộc**: U1 (theo thứ tự làm sau U7a)
- **FR**: FR-12, FR-11 (phần gọi dáng; rig ở U1)
- **Service**: rendering, authoring-service (prompt của Kỹ sư Remotion)
- **Phạm vi**: bộ hiệu ứng có tên (khói, lửa, mưa, lấp lánh, tia điện, nổ, bụi…) lấy màu từ bảng màu của video; prompt Kỹ sư Remotion gọi hiệu ứng theo tên và gọi dáng của rig theo key pose của animatic. Rig, thư viện dáng và nguyên lý diễn hoạt đã có từ U1.
- **Tiêu chí xong riêng**: một cảnh có nhân vật dùng key pose và ít nhất một hiệu ứng có tên, render đúng ở cả hai khung hình; tsc và test của kit xanh. Luồng: 16 bước.

## U7c — Validate đúng chỗ, Review đủ thông tin

- **Nhánh**: `feature/cr-072-u7c-validate-review` · **Cỡ**: M · **Phụ thuộc**: U2b, U7a
- **FR**: FR-23, FR-24, NFR-2 (hiện tỉ lệ duyệt ngay lần đầu)
- **Service**: authoring-service, web-gui
- **Phạm vi**: `ValidationFindings` (mỗi phát hiện chỉ tới một đơn vị; "Mở chỗ đó"; "AI sửa chỗ này" chỉ chạy khi Creator bấm; "Bỏ qua cảnh báo" có ghi lại; sửa xong chỉ kiểm lại đơn vị đó); `ReviewSummary` (chỗ đã duyệt, cảnh báo đã bỏ qua, thời lượng từng phần, ước tính render, chi phí token, tỉ lệ duyệt ngay lần đầu, thứ sẽ vào thư viện).
- **Tiêu chí xong riêng**: mỗi phát hiện mở đúng shot hoặc mục; cảnh báo đã bỏ qua hiện ở Review. Luồng: 16 bước.

## U8a — Render từng cảnh, Merge và QC

- **Nhánh**: `feature/cr-072-u8a-render-per-scene` · **Cỡ**: L, có NFR · **Phụ thuộc**: U2b, U5, U7a
- **FR**: FR-14, FR-26, FR-8 (phần render và ghép), NFR-4
- **Service**: orchestrator, rendering, video-assembly, api-gateway, web-gui
- **Phạm vi**:
  - `RenderSaga` (ADR-0038): lệnh `render_scene` mỗi cảnh, sự kiện `scene_rendered` / `scene_render_failed`; tự thử lại một lần khi lỗi tạm thời; huỷ và chạy lại từng cảnh; đơn vị `render_scene` ở orchestrator. `render_scenes` cả video giữ tới khi saga mới chạy ổn, bỏ ở một CR sau (backlog).
  - `SceneRenderer`: đoạn chồng ở chỗ nối liền mạch, chỉnh màu theo color key, tiến độ theo khung, phân loại lỗi tạm thời và lỗi do code (shot, khung); file `/shared/{id}/video/scenes/{scene}.mp4` và `{scene}.json`.
  - `SceneAssembler`: `assemble_video` nhận `scenes[]`, cắt đoạn chồng, áp chuyển cảnh ở chỗ cắt, tiến độ theo chặng (ghép cảnh → giọng → nhạc → âm lượng → QC). `qc_rules` thêm chỗ nối khớp và khung đen hoặc trống; mọi kết quả là cảnh báo, `QC_ENFORCE` giữ tắt.
  - Màn Render theo cảnh ("Sửa shot lỗi" mở đúng shot ở bước Code); màn Merge có tiến độ theo chặng và bảng QC. Cập nhật `docs/contracts/shared-artifacts.md`.
  - **NFR**: số cảnh render song song theo CPU và bộ nhớ, kích thước đoạn chồng, phân loại lỗi tạm thời.
- **Tiêu chí xong riêng**: một cảnh lỗi không dừng các cảnh khác (tiêu chí chấp nhận 9, phần render); chỗ nối liền mạch không giật (tiêu chí chấp nhận 5, phần ghép); sửa một cảnh storyboard chỉ làm lại giọng, animatic, hình, code và render của cảnh đó (tiêu chí chấp nhận 7). Luồng: 16 bước.

## U8b — Dựng theo ghi chú mốc thời gian

- **Nhánh**: `feature/cr-072-u8b-edit-notes` · **Cỡ**: M · **Phụ thuộc**: U2b, U8a
- **FR**: FR-25
- **Service**: orchestrator, authoring-service, api-gateway, web-gui
- **Phạm vi**: `EditSession` (orchestrator: ghi chú theo mốc gắn vào shot đang chạy, xin kế hoạch, chạy kế hoạch, "Chỉ render lại"); `EditPlanner` (authoring-service: tầng sửa cho từng ghi chú và danh sách sẽ làm lại); đổi chuyển cảnh ở chỗ cắt không render lại; bước mới "Dựng" sau Render.
- **Tiêu chí xong riêng**: làm lại một cảnh ở màn Dựng chỉ render lại cảnh đó và đoạn nối liền kề (tiêu chí chấp nhận 5). Luồng: 17 bước.

## U9 — Thumbnail bằng Remotion, rà prompt

- **Nhánh**: `feature/cr-072-u9-thumbnail-prompts` · **Cỡ**: M · **Phụ thuộc**: U1, U2b, U5 (theo thứ tự làm cuối)
- **FR**: FR-21, FR-15 (phần còn lại)
- **Service**: authoring-service, rendering, web-gui
- **Phạm vi**: `ThumbnailComposer` (3 phương án 1280×720 từ khung mẫu, nhân vật, bối cảnh; Creator sửa chữ và chọn; ảnh chọn ghi vào `/shared/{id}/thumbnail/`); prompt cho công cụ tạo ảnh ngoài sinh từ ngôn ngữ hình, thế giới, bảng màu; rà toàn bộ prompt hệ thống cho khớp luồng 17 bước.
- **Tiêu chí xong riêng**: publisher đăng được với thumbnail Remotion đã chọn; một project mới đi hết luồng mục tiêu tới video đăng được, Creator duyệt đúng các chỗ trong bảng luồng (tiêu chí chấp nhận 1). Luồng: 17 bước.

---

## Đối chiếu với kế hoạch luồng

| Kế hoạch luồng | CR con |
|---|---|
| U1 | U1 |
| Cổng chọn ngôn ngữ | Bỏ: Khoa học phát sáng bật khi U1 vào main (U-3.1 A) |
| U2 | U2a, U2b |
| U3 | U3 |
| U4 | U4a, U4b, U4c |
| U5 | U5 |
| U6 | U6 |
| U7 | U7a, U7b, U7c |
| U8 | U8a, U8b |
| U9 | U9 |

## Việc để lại sau CR-072

| Việc | Nguồn |
|---|---|
| Ba ngôn ngữ hình còn lại (Sách tranh tươi, Giấy cắt lớp, Đường nét tối giản): token, luật, ảnh mẫu và phần nặng | Creator, 2026-10-14 (thay U-3.2 A) |
| Vòng AI tự xem ảnh đã render rồi sửa hình (chỉ cho hình hạng chính, một vòng) | Creator, 2026-10-14 |
| Bỏ lệnh `render_scenes` cả video sau khi saga theo cảnh chạy ổn trên stack thật | `component-dependency.md`, đổi hợp đồng theo ba bước |
