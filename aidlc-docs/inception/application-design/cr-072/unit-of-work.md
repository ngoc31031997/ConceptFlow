# CR-072 — Các CR con (Units of Work)

## Date
2026-10-13 (cập nhật 2026-10-17: thứ tự theo luồng, phạm vi U1, U2a, U7b; sau đó cùng ngày: Tiền kỳ · Nội dung thay U3 bằng U3a..U3f, U2b gộp vào U3a và U5; Creator duyệt cùng ngày)

Đầu vào: yêu cầu `aidlc-docs/inception/requirements/cr-072-2d-film-pipeline.md`; kế hoạch luồng `aidlc-docs/inception/plans/cr-072-workflow-plan.md`; thiết kế trong thư mục này (`components.md`, `component-methods.md`, `services.md`, `component-dependency.md`); câu trả lời U-1..U-5, U-3.1, U-3.2, U-4.1 trong `aidlc-docs/inception/plans/cr-072-unit-of-work-plan.md`.

Phụ thuộc giữa các CR con: [`unit-of-work-dependency.md`](unit-of-work-dependency.md). FR, NFR và tiêu chí chấp nhận gán vào CR con: [`unit-of-work-story-map.md`](unit-of-work-story-map.md).

## Quyết định chung

| Quyết định | Nguồn |
|---|---|
| 19 CR con. Bốn CR con cỡ XL của kế hoạch luồng được tách: U2 → U2a, U2b; U4 → U4a, U4b, U4c; U7 → U7a, U7b, U7c; U8 → U8a, U8b. Số cũ giữ, thêm chữ cái, nên cột "CR con" trong `components.md` và `application-design.md` vẫn đúng (U4 nghĩa là U4a..U4c) | U-1 A |
| **Tiền kỳ · Nội dung thay U3 bằng sáu CR con**, mỗi CR con một bước của mockup bản 8: U3a Ý tưởng, U3b Nghiên cứu, U3c Câu chuyện, U3d Beat sheet, U3e Viết kịch bản, U3f Rà soát và chốt. Bước 2 Cấu hình không có CR con riêng | Creator, 2026-10-17 (câu 3 A) |
| **U2b không còn là CR con riêng.** Khung sửa theo đơn vị dựng trong U3a, CR con đầu tiên dùng nó. Phần áp khung lên Storyboard hiện có gộp vào U5, CR con chuyển Storyboard sang dạng mới; áp lên Kịch bản hiện có bỏ vì U3e thay bước đó | Creator, 2026-10-17 (câu 2 B; phần Storyboard duyệt cùng tài liệu này) |
| Mỗi CR con là `CR-072-U<N>`, nhánh `feature/cr-072-u<N>-<slug>`, worktree riêng, vào main riêng | W-1 A |
| Làm tuần tự, mỗi lúc một CR con, **theo thứ tự các bước của luồng** (Tiền kỳ → Sản xuất → Hậu kỳ): U2a → U3a → U3b → U3c → U3d → U3e → U3f → U1 → U4a → U4b → U5 → U6 → U4c → U7a → U7b → U7c → U8a → U8b → U9 | U-5 B; thứ tự theo luồng (Creator, 2026-10-17); Tiền kỳ ngay sau U2a (Creator, 2026-10-17, câu 2 B) |
| **Đọc giọng từng câu** (`SynthesizeLines`, `DraftVoiceDispatcher`, lệnh `synthesize_lines`) và **cách phát file chung từ `/shared`** làm ở U3e, vì nghe thử ở Viết kịch bản là người dùng đầu tiên. U6 thêm cách đọc ("đọc là", tốc độ, khoảng nghỉ), từ điển phát âm, Giọng sơ bộ và Animatic trên cùng đường đó; U1 dùng cách phát file đã có cho ảnh tĩnh | Creator, 2026-10-17 (câu 4 A) |
| Trong lúc Tiền kỳ · Nội dung làm dần, bước Kịch bản hiện có vẫn chạy và đọc thêm đầu vào mới của từng CR con (brief, dữ kiện, câu chuyện, beat), nên luồng luôn đi được tới video đăng được. U3e thay bước đó bằng Viết kịch bản | Kế hoạch luồng ("bước chưa làm thì giữ như hiện nay") |
| Không viết code chuyển đổi cho project làm dở. Trước khi chạy `/deliver` một CR con, Creator làm xong hoặc xoá project đang dở. Project đã có video vẫn mở được Kết quả và Đăng video; mỗi CR con có test cho điều này | U-2 A |
| **Khoa học phát sáng** là ngôn ngữ hình được bật khi U1 vào main. Không có bước chờ Creator so ảnh mẫu | U-3.1 A |
| **Chỉ một ngôn ngữ hình**: Khoa học phát sáng, làm đầy đủ chi tiết. Ba ngôn ngữ còn lại (Sách tranh tươi, Giấy cắt lớp, Đường nét tối giản) không có trong CR-072, kể cả ảnh mẫu (backlog). Kho ngôn ngữ, sao chép, bật và ghim vẫn làm đủ; bản sao của Khoa học phát sáng bật được | Creator, 2026-10-14 (thay U-3.2 A) |
| **Vẽ bằng code Remotion**, không dùng AI tạo ảnh; AI chỉ viết code. Chất lượng đi trước, chấp nhận render chậm hơn. U1 làm năm cách nâng chất lượng trong kit: lớp chất phim, vẽ theo hạng chính/phụ/phông, vật liệu và ánh sáng do kit tính, rig cho nhân vật và máy móc, nền thủ tục. Không có vòng AI tự xem ảnh rồi sửa | Creator, 2026-10-14 |
| **Tách cách vẽ khỏi màu**: ngôn ngữ hình quy định cách vẽ; màu đến từ bảng màu của video và ánh sáng của từng cảnh, kit không tự chọn màu. AI gợi ý bảng màu thế giới ở U4a, color key ở U5, chỉnh màu theo color key khi render ở U8a | Creator, 2026-10-17 |
| **Hạng chính / phụ / phông là vai trong từng video**, không phải của bản thân vật: con ong là Chính trong video về ong thụ phấn cho sung, Phụ trong video về cánh đồng hoa. AI đề xuất hạng kèm một câu lý do từ kịch bản và storyboard của video đó; Creator đổi được. Thư viện hình chỉ lưu **mức chi tiết** của bản vẽ (Kỹ / Vừa / Bóng); một hình dùng lại được khi mức chi tiết đủ cho hạng cần (Kỹ cho mọi hạng, Vừa cho Phụ và Phông, Bóng chỉ cho Phông) | Creator, 2026-10-17 |
| **Ba bước tạo hình làm ba việc khác nhau**: Mỹ thuật lượt 1 (trước Storyboard) thiết kế thứ dùng chung cả video — thế giới và bảng màu thế giới, bối cảnh, nhân vật, đạo cụ, và hạng của nhân vật, đạo cụ; Mỹ thuật lượt 2 (sau Storyboard) quyết định màu và ánh sáng từng phần của chuyện (color key) và chốt nhìn bằng khung mẫu; Hình minh hoạ (sản xuất, sau Animatic) vẽ nốt hình riêng của từng shot mà bộ thiết kế chưa có, và đặt hạng cho hình mới | Creator, 2026-10-17 |
| ~~Khung sửa theo đơn vị (U2b) áp lên cả Kịch bản và Storyboard hiện có~~. Thay bằng: khung ở U3a, áp lên Storyboard ở U5 (xem dòng U2b ở trên) | U-4.1 A; đổi 2026-10-17 |
| Xoá dữ liệu cũ khi **U2a** vào main (CR con vào main đầu tiên), sau khi sao lưu. U1 gỡ thêm luật style cũ và hình vẽ theo kit cũ khi nó vào main | W-2 A; dời từ U1 sang U2a (Creator, 2026-10-17) |
| Luồng thêm bước dần theo CR con, không có bước giả: 13 bước hiện có → 21 bước khi U8b xong (ADR-0042) | Kế hoạch luồng ("bước chưa làm thì giữ như hiện nay") |

**Thay đổi so với yêu cầu đã duyệt:** FR-18 ghi "Trước CR con đầu tiên, AI dựng bộ ảnh mẫu cho cả bốn, Creator so rồi bật một (P-4 A)". Theo U-3.1 A, Creator chọn Khoa học phát sáng ngay, nên không còn cổng chọn ngôn ngữ. Theo quyết định 2026-10-14, chỉ Khoa học phát sáng có trong CR-072; ba ngôn ngữ còn lại không có ảnh mẫu. Trang Ngôn ngữ hình, việc chỉ một ngôn ngữ bật, sao chép và việc project ghim phiên bản vẫn giữ như FR-18. FR-16 (xoá dữ liệu cũ) chuyển từ U1 sang U2a. Ngày 2026-10-17 yêu cầu được cập nhật cho Tiền kỳ · Nội dung (FR-2 thay bằng FR-27..FR-32; FR-1, FR-19 đổi; tiêu chí chấp nhận 11–14), ghi ở đầu `cr-072-2d-film-pipeline.md`.

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

- **Nhánh**: `feature/cr-072-u1-visual-language` · **Cỡ**: XL · **Phụ thuộc**: U3e (cách phát file chung từ `/shared`) (làm sau U3f theo thứ tự; phải có trước U4a)
- **FR**: FR-18, FR-15 (luật hình, luật nền, luật cảnh theo ngôn ngữ), FR-11 (rig và dáng)
- **Service**: rendering, llm-service, authoring-service, orchestrator, api-gateway, web-gui
- **Phạm vi**:
  - `VisualLanguageStore` (authoring-service): ngôn ngữ hình có phiên bản; đúng một ngôn ngữ bật; ngôn ngữ hệ thống chỉ đọc, sao chép để sửa; project ghim id và phiên bản lúc tạo (bước Ý tưởng, nút "Tạo project và viết brief"); ô Ngôn ngữ hình chỉ đọc ở màn Ý tưởng và Cấu hình.
  - Một ngôn ngữ hệ thống: Khoa học phát sáng, với luật vẽ hình, luật vẽ nền và luật cảnh; luật cảnh chỉ nói cách dùng ánh sáng, không quy định màu.
  - Kit `phatsang` (rendering), token chỉ giữ cách tô, chọn qua input prop `kitKey` (đi theo đường của `videoFont`); năm cách nâng chất lượng: lớp chất phim (`Shot`, `Scene`), vẽ theo hạng chính/phụ/phông, vật liệu và ánh sáng do kit tính (`Shape`), rig (bộ xương, dáng, hành động, chu kỳ, IK, biểu cảm; `Mechanism` cho máy móc), nền thủ tục (noise, shader WebGL có sẵn). Hình có sẵn và sáu nền vẽ lại. `GET /v1/kits`.
  - Màu từ ngoài vào: `light` của từng cảnh trong storyboard (vai trò trong bảng màu), bộ ghép code sinh `LOOKS`; prompt Đạo diễn bỏ chỗ ép "nền sáng bão hoà".
  - `StillRenderService`: `POST /v1/stills` (cache theo hash nội dung), `/shared/stills/{hash}.png`, dọn ảnh không còn ai trỏ tới. Composition `LanguageSample` với hai cảnh thử (lòng quả sung sáng, vũ trụ tối).
  - api-gateway: ảnh tĩnh đi qua cách phát file chung từ `/shared` đã có từ U3e (ADR-0041).
  - Hạng theo vai trong video ở danh sách hình của project (AI đề xuất kèm lý do, Creator đổi); thư viện hình lưu mức chi tiết Kỹ / Vừa / Bóng và dùng lại hình theo mức chi tiết. Khi chưa có Mỹ thuật lượt 1, mọi hạng đặt ở bước Hình minh hoạ hiện có.
  - web-gui: trang cài đặt Ngôn ngữ hình; chọn hạng (kèm lý do của AI) ở danh sách hình của project; huy hiệu mức chi tiết ở Thư viện hình.
  - Luồng hiện có (Hình minh hoạ, Code, Render) đọc luật của ngôn ngữ đã ghim và dùng kit Khoa học phát sáng; gỡ vai trò `illustration_style`; xoá hình vẽ theo kit cũ.
  - Thiết kế: `aidlc-docs/construction/plans/cr-072-u1-visual-language-design.md`.
- **Giao cho CR con sau**: `VisualLanguageStore.Active/PinForProject`, `/v1/stills`, kit `phatsang` (lớp chất phim, vật liệu, rig, bộ sinh nền, shader), `Scene.light`/`LOOKS`.
- **Tiêu chí xong riêng**:
  - Trang Ngôn ngữ hình hiện ảnh mẫu lòng quả sung (sáng) và vũ trụ (tối) ở hai khung, cùng một lối vẽ.
  - Đổi ngôn ngữ đang bật (sang một bản sao của Khoa học phát sáng) không đổi project đã tạo; project mới ghim ngôn ngữ mới (tiêu chí chấp nhận 8).
  - Gọi `/v1/stills` hai lần với cùng nội dung thì lần hai dùng lại file (`cache_hit`).
  - Hai project chủ đề vũ trụ và lòng quả sung cho hai bảng màu khác nhau, video một tối một sáng; một hình hạng chính có bộ phận chuyển động và kiểu cắt lớp; một nhân vật trên rig đi bộ và đổi biểu cảm.
  - Luồng sau khi vào main: 17 bước.

## U2a — Luồng ba giai đoạn

- **Nhánh**: `feature/cr-072-u2a-three-phase-flow` · **Cỡ**: M · **Phụ thuộc**: — (làm đầu tiên)
- **FR**: FR-1 (phần thanh bước và tên bước), FR-16
- **Service**: orchestrator, authoring-service, web-gui; script vận hành
- **Phạm vi**: `FlowDefinition` ở ba bản chia 13 bước hiện có vào bốn nhóm của luồng mục tiêu (Tiền kỳ · Nội dung, Tiền kỳ · Hình ảnh, Sản xuất, Hậu kỳ; ADR-0042); `StepRail` theo nhóm; tiêu đề màn; đổi tên các bước đã có theo tên trong mockup bản 8 (bước 1 "Ý tưởng"; tên cụ thể chốt ở Functional Design). Test hợp đồng giữ ba bản khớp nhau.
- Script sao lưu và xoá dữ liệu cũ (FR-16, W-2 A): có bước xác nhận; dump DB của orchestrator và authoring-service, nén `/shared`; xoá mọi project qua saga xoá, `llm_usage`, `code_check_diagnostics`, prompt Creator đã sửa; giữ kiểu video, cấu hình kênh, kết nối YouTube, thư viện hình; chạy từ primary checkout ngay sau khi U2a vào main. Thư viện hình và luật style cũ do U1 xoá khi U1 vào main.
- **Giao cho CR con sau**: cách thêm một bước vào một giai đoạn ở cả ba bản.
- **Tiêu chí xong riêng**: thanh bước hiện bốn nhóm của ba giai đoạn; số bước, tên bước, nhãn sự kiện tiến độ khớp nhau giữa server và web-gui; sao lưu tạo được file dump DB và file nén `/shared` trước khi xoá, báo cáo ghi rõ đã xoá gì, kiểu video, cấu hình kênh, kết nối YouTube còn nguyên. Luồng: 13 bước.

## U2b — gộp vào U3a và U5

Không còn là CR con riêng (Creator, 2026-10-17, câu 2 B). Khung sửa theo đơn vị (`ReviewUnitStore`, `DependencyTracker`, `RedoService`, `UnitGenerator`, HTTP `/v1/projects/{id}/units/...`, bộ thành phần đơn vị ở web-gui) làm ở **U3a**. Phần áp khung lên Storyboard hiện có, cùng việc đánh "cần làm lại" đúng hình minh hoạ và đoạn code bị ảnh hưởng khi sửa một cảnh, làm ở **U5**. Áp khung lên Kịch bản hiện có bỏ, vì U3e thay bước đó. Tên nhánh `feature/cr-072-u2b-per-unit-review` không dùng.

## Tiền kỳ · Nội dung: U3a..U3f

Thay U3 cũ (Creator, 2026-10-17). Giao diện theo mockup bản 8, bước 1–7, đã chốt. Yêu cầu: FR-27..FR-32. Mỗi CR con thêm hoặc đổi đúng một bước, đăng ký loại đơn vị của nó vào khung của U3a, và nối đầu ra của nó vào bước Kịch bản hiện có cho tới khi U3e thay bước đó, để luồng luôn chạy tới video đăng được.

Tiêu chí xong chung của sáu CR con, ngoài tiêu chí xong chung ở trên: mỗi chỗ sửa tay, làm lại, xoá hay làm phần phía sau phải làm lại có hộp xác nhận nêu ảnh hưởng (`docs/ux-ui-design-rules.md` luật 4–6); ô Creator viết không bị AI ghi đè trừ khi Creator bấm làm lại; sửa một đơn vị chỉ làm phần phụ thuộc vào nó thành "cần làm lại", và thanh bước hiện "↻" ở bước có phần đó.

## U3a — Ý tưởng · creative brief, khung sửa theo đơn vị

- **Nhánh**: `feature/cr-072-u3a-idea-brief` · **Cỡ**: L · **Phụ thuộc**: U2a
- **FR**: FR-27, FR-17, NFR-2 (đếm lần làm lại)
- **Service**: authoring-service, orchestrator, api-gateway, web-gui
- **Phạm vi**:
  - Khung sửa theo đơn vị theo ADR-0035 (từ U2b cũ): `ReviewUnitStore`, `DependencyTracker`, `RedoService`, giao diện `UnitGenerator`; HTTP `/v1/projects/{id}/units/...`; bộ thành phần đơn vị ở web-gui (thẻ đơn vị, sửa tay trong hộp thoại, làm lại kèm ghi chú, dấu "Bạn viết", hộp "sẽ làm lại", hộp xác nhận kèm ảnh hưởng, duyệt tất cả mục còn lại); "↻" trên thanh bước.
  - `CreativeBrief`: tạo project với thời lượng mục tiêu (trường mới ở orchestrator, như dạng video và ngôn ngữ nội dung); AI viết brief; đơn vị `brief`; prompt vai mới viết brief.
  - Màn Ý tưởng thay màn Khởi tạo: ① Đề tài (chủ đề, dạng video, ngôn ngữ nội dung, thời lượng mục tiêu) và nút "Tạo project và viết brief"; ② Creative brief; duyệt brief mới sang Cấu hình. Bước 1 thành chỗ duyệt (◆). Chọn kiểu video giữ ở màn này như hiện nay tới U3c. Ô Ngôn ngữ hình do U1 thêm.
  - Bước Kịch bản hiện có đọc brief đã duyệt (câu hỏi trung tâm, khán giả, biết / hiểu / cảm thấy, giọng điệu, ngoài phạm vi, thời lượng mục tiêu).
- **Giao cho CR con sau**: khung đơn vị; đơn vị `brief`; thời lượng mục tiêu của project.
- **Tiêu chí xong riêng**: brief AI viết đủ bảy ô; sửa một ô rồi làm lại kèm ghi chú thì ô đã sửa giữ nguyên; đổi brief đã duyệt thì màn báo trước các bước sau sẽ làm lại; số lần làm lại được ghi theo loại đơn vị. Luồng: 13 bước.

## U3b — Nghiên cứu và kiểm chứng

- **Nhánh**: `feature/cr-072-u3b-research` · **Cỡ**: L · **Phụ thuộc**: U3a
- **FR**: FR-28, tiêu chí chấp nhận 11–13
- **Service**: authoring-service, orchestrator (`flow.go`), api-gateway, web-gui
- **Phạm vi**:
  - `ResearchDesk`: câu hỏi (`research_question`, AI sinh từ brief, không cần duyệt); prompt nghiên cứu và prompt bổ sung; đọc câu trả lời dán vào theo mẫu `conceptflow.research.v1` (tìm khối JSON, gộp theo mã, nhật ký từng lần dán, dòng lỗi bị bỏ kèm lý do); bảng dữ kiện (`fact`: loại, bắt buộc nói, giữ / bỏ, thêm tay, sửa); độ phủ câu hỏi; duyệt bảng; bỏ qua bước và "Làm nghiên cứu" để quay lại.
  - `SourceVerifier` + `SourceFetcher` theo ADR-0043: tải trang chạy nền, chặn địa chỉ nội bộ, chuẩn hoá chữ, kết quả khớp / không khớp (câu gần nhất) / không tải được (lý do); "Kiểm lại". Cổng `ResearchSource` với một bản "dán kết quả".
  - Bước mới "Nghiên cứu" (bước 3, tuỳ chọn) sau Cấu hình ở cả ba bản định nghĩa luồng; nhãn "tuỳ chọn" / "bỏ qua" trên thanh bước; màn Nghiên cứu theo mockup.
  - Bước Kịch bản hiện có đọc dữ kiện đang giữ (mã, loại, nhãn nguồn, bắt buộc nói); khi bỏ qua thì bước Kịch bản chạy như trước, chưa có dữ kiện có mã (dữ kiện "chưa có nguồn" làm ở U3c).
  - Functional Design chốt mẫu JSON, cách chuẩn hoá khi so đoạn trích, số liệu giới hạn của `SourceFetcher`, và cập nhật `docs/contracts/` cho mẫu `conceptflow.research.v1`.
- **Giao cho CR con sau**: đơn vị `fact` có mã và nhãn nguồn; trạng thái "Nghiên cứu bỏ qua".
- **Tiêu chí xong riêng**: dán một câu trả lời có 12 dữ kiện và một dòng lỗi thì bảng có 12 dữ kiện và nhật ký ghi dòng bị bỏ kèm lý do; đoạn trích không có trong trang hiện "không khớp"; link tới `localhost`, `10.x`, `169.254.169.254` hay tên service trong mạng Docker bị từ chối, có test; dán lần hai cập nhật đúng mã đã có; bỏ qua bước thì luồng vẫn đi tiếp. Luồng: 14 bước.

## U3c — Xây dựng câu chuyện

- **Nhánh**: `feature/cr-072-u3c-story` · **Cỡ**: M · **Phụ thuộc**: U3a, U3b
- **FR**: FR-29, FR-19, FR-28 mục 8
- **Service**: authoring-service, orchestrator (`flow.go`), api-gateway, web-gui
- **Phạm vi**:
  - `ArchetypeAdvisor` ở bước Câu chuyện: 3–5 gợi ý từ brief và bảng dữ kiện, mỗi gợi ý có góc kể, lý do có mã dữ kiện, câu mở mẫu; kiểu mới kèm playbook nháp, lưu vào bảng Kiểu video; "Để AI tự chọn", "Xem tất cả kiểu", "Gợi ý lại". Chọn kiểu video rời khỏi màn Ý tưởng.
  - `StoryFrame`: khung câu chuyện sáu ô (câu hỏi mở đầu, lời hứa, chuỗi nhân quả có mã dữ kiện, payoff có mã dữ kiện, logline, tóm tắt), đơn vị `story`, mỗi ô có nguồn riêng; dữ kiện dùng trong chuyện; khi bỏ qua Nghiên cứu thì AI tự liệt kê dữ kiện (`fact` mang nhãn "chưa có nguồn").
  - Bước mới "Câu chuyện" (bước 4) sau Nghiên cứu; màn Câu chuyện theo mockup; đổi góc khi câu chuyện đã duyệt có xác nhận.
  - Bước Kịch bản hiện có đọc câu chuyện đã duyệt và dữ kiện được dùng.
- **Giao cho CR con sau**: đơn vị `story`; danh sách dữ kiện dùng.
- **Tiêu chí xong riêng**: gợi ý kiểu video có lý do dẫn mã dữ kiện; sửa logline chỉ làm phần phụ thuộc vào logline thành "cần làm lại"; bỏ qua Nghiên cứu thì dữ kiện hiện nhãn "chưa có nguồn" (tiêu chí chấp nhận 12, phần Câu chuyện). Luồng: 15 bước.

## U3d — Beat sheet

- **Nhánh**: `feature/cr-072-u3d-beat-sheet` · **Cỡ**: M · **Phụ thuộc**: U3a, U3c
- **FR**: FR-30
- **Service**: authoring-service, orchestrator (`flow.go`), api-gateway, web-gui
- **Phạm vi**:
  - `BeatSheet`: beat (`beat`) có chức năng, tên, mục tiêu, nội dung, dữ kiện, thời lượng mục tiêu, chuyển tiếp; thêm beat sau một beat (AI viết), xoá, đổi thứ tự, đổi thời lượng; kiểm cấu trúc (tổng trong khung, hook ≤ 20 giây, dữ kiện bắt buộc có trong beat, beat nào cũng có mục tiêu, beat cuối là payoff); dải cấu trúc.
  - Bước mới "Beat sheet" (bước 5) sau Câu chuyện; màn Beat sheet theo mockup thay màn dàn ý hiện có.
  - Bước Kịch bản hiện có viết lời thoại theo beat đã duyệt, không tự dựng dàn ý nữa.
- **Giao cho CR con sau**: đơn vị `beat` (thứ tự, thời lượng, dữ kiện).
- **Tiêu chí xong riêng**: sửa một beat chỉ làm lời thoại của beat đó thành "cần làm lại"; đổi thời lượng chỉ đưa beat đó về chờ duyệt; xoá beat có dữ kiện bắt buộc thì hộp xác nhận nêu dữ kiện sẽ không còn trong beat nào. Luồng: 16 bước.

## U3e — Viết kịch bản, đọc giọng từng câu

- **Nhánh**: `feature/cr-072-u3e-script-writing` · **Cỡ**: L · **Phụ thuộc**: U3a, U3d
- **FR**: FR-31, FR-9 (phần đọc từng câu)
- **Service**: authoring-service, orchestrator, tts, api-gateway, web-gui
- **Phạm vi**:
  - `ScriptWriter`: lời thoại theo beat (`script_beat`, mỗi câu có mã và mã dữ kiện); cờ câu dài, câu khó nghe kèm gợi ý; kiểm câu mở và câu kết; kiểm chỗ nối giữa hai beat kèm gợi ý; prompt vai viết lời viết lại cho dạng theo beat.
  - Đọc từng câu theo ADR-0040, phần không có cách đọc: `DraftVoiceDispatcher` (orchestrator), lệnh `synthesize_lines` và sự kiện `lines_synthesized`, `SynthesizeLines` (tts, cache theo chữ và giọng), file `/shared/{id}/audio/{line}_{cache_key}.wav`; `VoiceLines` ghi đường dẫn và thời lượng; ước tính thời lượng kịch bản từ giọng thật. Cập nhật `docs/contracts/` và `shared-artifacts.md`.
  - api-gateway: cách phát file chung từ `/shared` (ADR-0041), dùng trước cho file giọng nghe thử.
  - Màn Viết kịch bản theo mockup thay màn Kịch bản hiện có (bước 6): toàn bộ kịch bản và các kiểm, lời thoại theo beat, nghe từng câu, cả beat, cả kịch bản.
  - Storyboard hiện có đọc lời thoại theo beat (thay nội dung bước Kịch bản cũ).
- **Giao cho CR con sau**: đơn vị `script_beat`; đường đọc từng câu và file giọng; cách phát file chung.
- **Tiêu chí xong riêng**: nghe được từng câu bằng giọng chọn ở Cấu hình; sửa một câu chỉ đọc lại câu đó, câu khác dùng lại file (`cache_hit`); thời lượng ước tính đổi theo câu vừa đọc lại; áp gợi ý chỗ nối chỉ đổi câu đầu beat sau. Luồng: 16 bước.

## U3f — Rà soát và chốt kịch bản, bộ bàn giao

- **Nhánh**: `feature/cr-072-u3f-review-lock` · **Cỡ**: L · **Phụ thuộc**: U3a, U3b, U3e
- **FR**: FR-32, tiêu chí chấp nhận 12, 14
- **Service**: authoring-service, orchestrator (`flow.go`), api-gateway, web-gui
- **Phạm vi**:
  - `ScriptReview`: máy kiểm và AI rà soát theo sáu nhóm (`review_check`); mở câu đó, áp gợi ý, mở ở Nghiên cứu, thêm vào ghi chú bàn giao, chấp nhận kèm lý do; ghi chú rà soát của AI và Creator.
  - `HandoffNotes`: năm mục cho từng beat (`handoff_beat`), mỗi mục ghi vai AI đọc nó; bảng thuật ngữ (`glossary_term`); nhãn "chưa có nguồn" khi bỏ qua Nghiên cứu.
  - `ScriptVersions` theo ADR-0044: điều kiện chốt, chốt phiên bản (ảnh chụp bất biến, hash từng phần), bản nháp và danh sách thay đổi, xác nhận trước khi chốt bản mới kèm phần sẽ làm lại, lịch sử phiên bản; `HandoffFor(role)`.
  - Bước mới "Rà soát và chốt" (bước 7) sau Viết kịch bản; màn ba tab theo mockup.
  - Storyboard, Hình minh hoạ và Code hiện có đọc lời thoại và đúng phần bộ bàn giao của vai mình từ bản đã chốt. Chốt bản mới đánh cả bước Storyboard hiện có (chưa theo đơn vị tới U5) là "cần làm lại"; từ U5 trở đi chỉ phần phụ thuộc chỗ đổi.
- **Giao cho CR con sau**: `script_version`, `HandoffFor(role)` cho Mỹ thuật lượt 1 (U4a, U4b), Storyboard (U5), Mỹ thuật lượt 2 (U5), Giọng (U6), Hình minh hoạ (U4c), Code (U7a), Validate (U7c).
- **Tiêu chí xong riêng**: chưa đủ điều kiện thì nút chốt tắt và màn nói thiếu gì; sửa lời thoại một beat sau khi chốt v1 thì Storyboard vẫn đọc v1, chốt v2 thì lịch sử có v1, v2 (tiêu chí chấp nhận 14); bỏ qua Nghiên cứu thì có đúng một cảnh báo "chưa có nguồn" và chấp nhận là chốt được (tiêu chí chấp nhận 12). Luồng: 17 bước.

## U4a — Mỹ thuật lượt 1: thế giới, bối cảnh, đạo cụ

- **Nhánh**: `feature/cr-072-u4a-art-world` · **Cỡ**: L · **Phụ thuộc**: U1, U3a, U3f
- **FR**: FR-3 (mục 1, 2, 4), FR-5, FR-15 (phần luật thế giới), NFR-4
- **Service**: authoring-service, rendering, api-gateway, web-gui
- **Phạm vi**:
  - `ArtBible` lượt 1 phần thế giới: mô tả thế giới (nơi chốn, thời đại, chất liệu, logic ánh sáng, bảng màu thế giới) và bảng tham khảo; bối cảnh (nền 4 lớp ở 16:9 và 9:16, biến thể ánh sáng khi cần); đạo cụ, mỗi đạo cụ có hạng trong video này (AI đề xuất kèm lý do, Creator đổi; đổi hạng thì đạo cụ thành "cần làm lại"). Đơn vị `art_world`, `art_location`, `art_prop`.
  - Phong cách theo thế giới (ADR-0037): luật thế giới trong prompt, lớp chất liệu theo chủ đề ở nền và đạo cụ.
  - Ảnh Creator tải lên cho bảng tham khảo (`/shared/{id}/uploads/moodboard/`). Functional Design kiểm model vẽ có đọc được ảnh không (P-5 A); không đọc được thì ảnh chỉ để Creator đối chiếu.
  - Bốn cách sửa một hình: làm lại kèm ghi chú, sửa mô tả rồi vẽ lại, tải SVG lên (`/shared/{id}/uploads/svg/`), thay bằng hình trong thư viện.
  - Composition `ArtSheet` (bối cảnh, đạo cụ) trong kit `phatsang`.
  - Bước mới "Mỹ thuật lượt 1" (bước 8, mở nhóm Tiền kỳ · Hình ảnh) sau Rà soát và chốt; màn có các tab Thế giới, Bối cảnh, Đạo cụ. Prompt đọc bản kịch bản đã chốt và phần bộ bàn giao của Mỹ thuật (giọng điệu và ngoài phạm vi của brief, thuật ngữ, phải giữ chính xác) qua `HandoffFor`.
- **Tiêu chí xong riêng**: hai chủ đề khác nhau (ví dụ vũ trụ, lòng quả sung) cho hai thế giới khác hẳn về bối cảnh, màu, ánh sáng, chất liệu, cùng ngôn ngữ hình (tiêu chí chấp nhận 2, phần thế giới); nền đúng ở cả hai khung hình; Storyboard hiện có đọc bộ mỹ thuật đã duyệt. Luồng: 18 bước.

## U4b — Nhân vật và thư viện nhân vật

- **Nhánh**: `feature/cr-072-u4b-characters` · **Cỡ**: L · **Phụ thuộc**: U1, U3a, U4a
- **FR**: FR-3 (mục 3), FR-4
- **Service**: authoring-service, llm-service, rendering, web-gui
- **Phạm vi**:
  - Nhân vật cấp video, đơn vị `art_character`: model sheet trước/nghiêng/sau, biểu cảm, tư thế tay, nhân vật đặt trên từng nền chính (`ArtSheet` nhân vật trong kit `phatsang`); hạng của nhân vật trong video này, như đạo cụ ở U4a. Nhân vật lưu vào thư viện nhân vật không mang hạng; video sau đặt lại hạng theo chuyện của nó. Miệng giữ hình cố định theo biểu cảm, bỏ nhép miệng.
  - Schema storyboard (llm-service): khai báo nhân vật một lần; shot chỉ tham chiếu nhân vật kèm biểu cảm và tư thế. Storyboard hiện có dùng schema này.
  - `CharacterLibrary` và trang cài đặt Thư viện nhân vật; lưu nhân vật đã duyệt, dùng lại ở video sau.
  - Tab Nhân vật trong màn Mỹ thuật lượt 1, cùng bốn cách sửa hình của U4a.
- **Tiêu chí xong riêng**: shot không đặt lại dáng, tuổi, áo của nhân vật; một nhân vật lưu vào thư viện được chọn lại ở project sau. Luồng: 18 bước.

## U4c — Hình minh hoạ theo bộ mỹ thuật

- **Nhánh**: `feature/cr-072-u4c-illustrations-from-art` · **Cỡ**: M · **Phụ thuộc**: U3a, U4b (làm sau U6 theo thứ tự: bước 13 Hình minh hoạ)
- **FR**: FR-13
- **Service**: authoring-service, web-gui
- **Phạm vi**: `IllustrationPlanner` chỉ đưa vào danh sách phần còn thiếu sau bộ mỹ thuật; hình của bộ mỹ thuật hiện là đã duyệt và chỉ sửa ở mục gốc, kèm hạng đã đặt ở lượt 1; hình mới có hạng do AI đề xuất theo shot; đơn vị `illustration` có "làm lại kèm ghi chú"; giữ vẽ song song, duyệt, sửa, bỏ, sửa prompt như hiện nay.
- **Tiêu chí xong riêng**: một project có bộ mỹ thuật đã duyệt không vẽ lại bối cảnh, nhân vật, đạo cụ đã có; sửa một mục gốc thì các hình dùng nó thành "cần làm lại". Luồng: 20 bước.

## U5 — Storyboard mới và Mỹ thuật lượt 2

- **Nhánh**: `feature/cr-072-u5-storyboard-color-key` · **Cỡ**: L (lớn hơn trước vì nhận phần Storyboard của U2b) · **Phụ thuộc**: U1, U3a, U3f, U4a, U4b
- **FR**: FR-3 (mục 5, 6), FR-6, FR-7, FR-8 (phần storyboard), FR-17 (phần Storyboard), NFR-4
- **Service**: authoring-service, llm-service, rendering, web-gui
- **Phạm vi**:
  - Storyboard theo cảnh dạng mới trên khung đơn vị của U3a, đơn vị `board_scene`: duyệt, sửa tay, làm lại kèm ghi chú, gom ghi chú theo cảnh và shot; cách nối sang cảnh sau và vật nối dùng chung (hành trình máy quay); cảnh giải thích ý đặt trong thế giới, thay FR7 "nền sáng bão hoà mặc định" của CR-067 (ADR-0037); prompt Đạo diễn viết lại, đọc lời thoại và phần bộ bàn giao của Storyboard từ bản kịch bản đã chốt.
  - Từ phần của U2b: sửa một cảnh thì đúng hình minh hoạ và đoạn code đang có của cảnh đó thành "cần làm lại", màn báo trước danh sách đó; chốt bản kịch bản mới chỉ làm `board_scene` phụ thuộc phần đổi thành "cần làm lại".
  - Mỹ thuật lượt 2: đơn vị `color_key` (mỗi phần dàn ý một khung màu; sửa tay bằng bộ chọn màu; kiểm tương phản chữ, hai cảnh liền nhau không quá giống, không lệch bảng màu thế giới; một khoảnh khắc phá bảng màu ở cao trào) và `style_frame` (3–5 khung mẫu). Composition `ColorKeyFrame`, `StyleFrame`. Dải màu của cả video trên một hàng.
  - Bước mới "Mỹ thuật lượt 2" sau Storyboard.
- **Tiêu chí xong riêng**: dải màu cho thấy color key đổi theo từng phần dàn ý (tiêu chí chấp nhận 6); không cảnh giải thích nào chuyển sang nền màu đặc; mỗi chỗ nối liền mạch ghi vật nối; sửa tay hoặc làm lại một cảnh storyboard chỉ đánh "cần làm lại" phần phụ thuộc của cảnh đó, và màn báo trước danh sách đó. Luồng: 19 bước.

## U6 — Giọng sơ bộ, sửa giọng, animatic

- **Nhánh**: `feature/cr-072-u6-draft-voice-animatic` · **Cỡ**: L · **Phụ thuộc**: U1, U3a, U3e, U4a, U4b, U5
- **FR**: FR-9, FR-10, FR-20, NFR-4
- **Service**: orchestrator, authoring-service, tts, rendering, api-gateway, web-gui
- **Phạm vi**:
  - Trên đường đọc từng câu đã có từ U3e (ADR-0040): `DraftVoiceDispatcher` thêm lúc gọi sau khi duyệt Mỹ thuật lượt 2 và khi một câu đổi; `synthesize_lines` thêm trường `say_as`, `speed`, `pause_before`, `pause_after` (chỉ thêm trường); `SynthesizeLines` (tts): "đọc là", tốc độ, khoảng nghỉ bằng SSML Azure, khoá cache thêm cách đọc (đổi khoá theo ba bước).
  - `PronunciationDictionary` và trang cài đặt Từ điển phát âm; `VoiceLines` (đơn vị `voice_line`), sửa chữ hoặc sửa cách đọc từng câu, đọc lại đúng câu đó.
  - `AnimaticPlan` (đơn vị `animatic_shot`), composition `AnimaticStill`, `AnimaticPlayer` phát trong trình duyệt (ADR-0039); sửa thời lượng, sửa câu, đổi khung tĩnh kèm ghi chú; ghi chú nhịp cả cảnh, AI đề xuất, Creator xem rồi áp dụng.
  - Bước mới "Giọng sơ bộ" và "Animatic" sau Mỹ thuật lượt 2. Bước "Giọng đọc" của saga rời thanh bước; saga vẫn gửi `synthesize_speech` và dùng lại file đã có theo cùng `cache_key`.
- **Tiêu chí xong riêng**: Creator xem animatic có giọng trước khi bước Code chạy (tiêu chí chấp nhận 3); một từ sửa trong từ điển được đọc đúng ở video sau (tiêu chí chấp nhận 10); sửa một câu chỉ đọc lại câu đó và chỉ dựng lại khung animatic của shot đó; câu đã đọc ở Viết kịch bản mà chữ và cách đọc không đổi thì dùng lại file. Luồng: 20 bước.

## U7a — Code theo cảnh, lỗi theo shot, xem nhanh shot

- **Nhánh**: `feature/cr-072-u7a-code-per-scene` · **Cỡ**: L, có NFR · **Phụ thuộc**: U1, U3a, U4c, U5
- **FR**: FR-22, FR-8 (phần code: hai cảnh dùng chung vật nối)
- **Service**: authoring-service, llm-service, rendering, api-gateway, web-gui
- **Phạm vi**:
  - `CodeSegments`: đoạn code là một cảnh, mỗi shot một trạng thái (đơn vị `code_shot`); bỏ lượt AI tự sửa; giữ autofix máy móc của CR-068 và ghi rõ đã sửa gì; shot lỗi hiện đúng shot và thông báo cụ thể, Creator chọn chạy lại, chạy lại kèm ghi chú, sửa tay hoặc dán kết quả AI ngoài.
  - llm-service `/v2/code/generate` nhận `segment_kind = "scene"`, trả chẩn đoán theo shot. Đoạn theo số shot của CR-050 bỏ khi U7a xong (đổi hợp đồng theo ba bước).
  - `ShotPreview`: `POST /v1/previews/shot` (480p), file `/shared/{id}/previews/`; xem nhanh ở màn Code.
  - **NFR**: thời gian mục tiêu của xem nhanh một shot (kế hoạch luồng: khoảng 10 giây).
- **Tiêu chí xong riêng**: shot lỗi hiện đúng shot và lỗi, không bị AI tự sửa (tiêu chí chấp nhận 9, phần Code); một nhân vật giữ đúng model trên mọi shot (tiêu chí chấp nhận 4); xem nhanh một shot đạt mục tiêu NFR. Luồng: 20 bước.

## U7b — Bộ VFX và gọi dáng theo key pose

- **Nhánh**: `feature/cr-072-u7b-key-pose-vfx` · **Cỡ**: M · **Phụ thuộc**: U1 (theo thứ tự làm sau U7a)
- **FR**: FR-12, FR-11 (phần gọi dáng; rig ở U1)
- **Service**: rendering, authoring-service (prompt của Kỹ sư Remotion)
- **Phạm vi**: bộ hiệu ứng có tên (khói, lửa, mưa, lấp lánh, tia điện, nổ, bụi…) lấy màu từ bảng màu của video; prompt Kỹ sư Remotion gọi hiệu ứng theo tên và gọi dáng của rig theo key pose của animatic. Rig, thư viện dáng và nguyên lý diễn hoạt đã có từ U1.
- **Tiêu chí xong riêng**: một cảnh có nhân vật dùng key pose và ít nhất một hiệu ứng có tên, render đúng ở cả hai khung hình; tsc và test của kit xanh. Luồng: 20 bước.

## U7c — Validate đúng chỗ, Review đủ thông tin

- **Nhánh**: `feature/cr-072-u7c-validate-review` · **Cỡ**: M · **Phụ thuộc**: U3a, U3f, U7a
- **FR**: FR-23, FR-24, FR-32 mục 4 (phần Validate), NFR-2 (hiện tỉ lệ duyệt ngay lần đầu)
- **Service**: authoring-service, web-gui
- **Phạm vi**: `ValidationFindings` (mỗi phát hiện chỉ tới một đơn vị; "Mở chỗ đó"; "AI sửa chỗ này" chỉ chạy khi Creator bấm; "Bỏ qua cảnh báo" có ghi lại; sửa xong chỉ kiểm lại đơn vị đó; kiểm hình đã dựng theo mục "phải giữ chính xác" của bộ bàn giao); `ReviewSummary` (chỗ đã duyệt, kể cả bảy bước Tiền kỳ · Nội dung, cảnh báo đã bỏ qua, mục rà soát đã chấp nhận, thời lượng từng phần, ước tính render, chi phí token, tỉ lệ duyệt ngay lần đầu, thứ sẽ vào thư viện).
- **Tiêu chí xong riêng**: mỗi phát hiện mở đúng shot hoặc mục; cảnh báo đã bỏ qua hiện ở Review. Luồng: 20 bước.

## U8a — Render từng cảnh, Merge và QC

- **Nhánh**: `feature/cr-072-u8a-render-per-scene` · **Cỡ**: L, có NFR · **Phụ thuộc**: U3a, U5, U7a
- **FR**: FR-14, FR-26, FR-8 (phần render và ghép), NFR-4
- **Service**: orchestrator, rendering, video-assembly, api-gateway, web-gui
- **Phạm vi**:
  - `RenderSaga` (ADR-0038): lệnh `render_scene` mỗi cảnh, sự kiện `scene_rendered` / `scene_render_failed`; tự thử lại một lần khi lỗi tạm thời; huỷ và chạy lại từng cảnh; đơn vị `render_scene` ở orchestrator. `render_scenes` cả video giữ tới khi saga mới chạy ổn, bỏ ở một CR sau (backlog).
  - `SceneRenderer`: đoạn chồng ở chỗ nối liền mạch, chỉnh màu theo color key, tiến độ theo khung, phân loại lỗi tạm thời và lỗi do code (shot, khung); file `/shared/{id}/video/scenes/{scene}.mp4` và `{scene}.json`.
  - `SceneAssembler`: `assemble_video` nhận `scenes[]`, cắt đoạn chồng, áp chuyển cảnh ở chỗ cắt, tiến độ theo chặng (ghép cảnh → giọng → nhạc → âm lượng → QC). `qc_rules` thêm chỗ nối khớp và khung đen hoặc trống; mọi kết quả là cảnh báo, `QC_ENFORCE` giữ tắt.
  - Màn Render theo cảnh ("Sửa shot lỗi" mở đúng shot ở bước Code), bước đổi tên thành "Render từng cảnh"; màn Merge có tiến độ theo chặng và bảng QC. Cập nhật `docs/contracts/shared-artifacts.md`.
  - **NFR**: số cảnh render song song theo CPU và bộ nhớ, kích thước đoạn chồng, phân loại lỗi tạm thời.
- **Tiêu chí xong riêng**: một cảnh lỗi không dừng các cảnh khác (tiêu chí chấp nhận 9, phần render); chỗ nối liền mạch không giật (tiêu chí chấp nhận 5, phần ghép); sửa một cảnh storyboard chỉ làm lại giọng, animatic, hình, code và render của cảnh đó (tiêu chí chấp nhận 7). Luồng: 20 bước.

## U8b — Dựng theo ghi chú mốc thời gian

- **Nhánh**: `feature/cr-072-u8b-edit-notes` · **Cỡ**: M · **Phụ thuộc**: U3a, U8a
- **FR**: FR-25
- **Service**: orchestrator, authoring-service, api-gateway, web-gui
- **Phạm vi**: `EditSession` (orchestrator: ghi chú theo mốc gắn vào shot đang chạy, xin kế hoạch, chạy kế hoạch, "Chỉ render lại"); `EditPlanner` (authoring-service: tầng sửa cho từng ghi chú và danh sách sẽ làm lại); đổi chuyển cảnh ở chỗ cắt không render lại; bước mới "Dựng" sau Render.
- **Tiêu chí xong riêng**: làm lại một cảnh ở màn Dựng chỉ render lại cảnh đó và đoạn nối liền kề (tiêu chí chấp nhận 5). Luồng: 21 bước.

## U9 — Thumbnail bằng Remotion, rà prompt

- **Nhánh**: `feature/cr-072-u9-thumbnail-prompts` · **Cỡ**: M · **Phụ thuộc**: U1, U3a, U5 (theo thứ tự làm cuối)
- **FR**: FR-21, FR-15 (phần còn lại)
- **Service**: authoring-service, rendering, web-gui
- **Phạm vi**: `ThumbnailComposer` (3 phương án 1280×720 từ khung mẫu, nhân vật, bối cảnh; Creator sửa chữ và chọn; ảnh chọn ghi vào `/shared/{id}/thumbnail/`); prompt cho công cụ tạo ảnh ngoài sinh từ ngôn ngữ hình, thế giới, bảng màu; rà toàn bộ prompt hệ thống cho khớp luồng 21 bước.
- **Tiêu chí xong riêng**: publisher đăng được với thumbnail Remotion đã chọn; một project mới đi hết luồng mục tiêu tới video đăng được, Creator duyệt đúng các chỗ trong bảng luồng (tiêu chí chấp nhận 1). Luồng: 21 bước.

---

## Đối chiếu với kế hoạch luồng

| Kế hoạch luồng | CR con |
|---|---|
| U1 | U1 |
| Cổng chọn ngôn ngữ | Bỏ: Khoa học phát sáng bật khi U1 vào main (U-3.1 A) |
| U2 | U2a; phần khung đơn vị → U3a, phần áp lên Storyboard → U5 (2026-10-17) |
| U3 | U3a, U3b, U3c, U3d, U3e, U3f (Tiền kỳ · Nội dung, 2026-10-17) |
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
| Tra nguồn tự động bằng API tìm kiếm (Tavily, Wikipedia, Semantic Scholar) qua cổng `ResearchSource` | Creator, 2026-10-17; ADR-0043 |
| Kiểm chứng lại sau khi chốt kịch bản | Creator, 2026-10-17 ("chưa cần") |
| Gom bớt chỗ duyệt của Tiền kỳ · Nội dung (hiện mỗi bước một chỗ) | Creator, 2026-10-17 ("tối ưu sau") |
