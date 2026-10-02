# CR-062 — Thiết kế: lỗi bố cục `safe_area` báo nhầm và thông báo sai tên

## Yêu cầu gốc (nguyên văn)

> kiểm tra các lỗi bố cục  Kiểm tra vẫn báo 29 lỗi bố cục: một số khối nằm ngoài vùng an toàn ở các shot 2.x và 3.x. Lần trước các lỗi này không chặn dự án đi tới render. Chúng chỉ làm vài hình bị lấn ra mép khung, không làm render hỏng.

Trả lời các câu hỏi:

> làm theo mục 1 đi

Chốt: Phương án 1. Q2 và Q3 không được trả lời riêng nên theo khuyến nghị: Q2 = (a) cảnh báo, Q3 = (a) 1% cạnh ngắn của khung.

Đoạn sau dấu cách đôi là báo cáo của lượt sửa script project `257acc58` (mục "Fix — Sửa script đã lưu của project 257acc58" trong `aidlc-docs/audit.md`). Yêu cầu: xem 29 lỗi đó là gì, lỗi nào là thật, lỗi nào là luật báo nhầm, và nên sửa ở đâu.

## Hiện trạng

### Kiểm tra bố cục chạy ở đâu và chặn cái gì

- `services/rendering/application/check_script.py:115-150` (`_check_layout`): sau khi `tsc` qua, harness vẽ từng shot trong Chromium, `domain/layout_rules.py:evaluate` chấm số đo. Luật trong `BLOCKING_RULES` (`layout_rules.py:60`: `safe_area`, `subtitle_zone`, `text_overlap`, `text_overflow`, `min_font`, `runtime_error`) thành diagnostic, các luật khác thành cảnh báo.
- Diagnostic bố cục làm `ok = false`. llm-service đưa chúng vào vòng sửa như lỗi tsc (`services/llm-service/app/pipeline/prompts.py:150-163`). Hết vòng sửa vẫn còn lỗi thì kết quả là `check_failed`.
- authoring-service: `generate_authoring_code.go:93-103` đặt `CheckFailed` và ghép diagnostic thành chuỗi. Chuỗi AI dừng với ghi chú "Đã sinh và lưu code nhưng **vẫn lỗi biên dịch** sau N vòng sửa" (`authoring_chain.go:199-209`). Nhật ký ghi lượt chạy là `failed` với "code đã lưu nhưng vẫn lỗi biên dịch" (`generate_authoring.go:237-252`). Câu này sai khi lỗi còn lại chỉ là lỗi bố cục.
- **Render không bị chặn.** Nút "Kiểm tra kịch bản" ở bước Code (`web-gui/src/pages/ManimEngineerStepPage.tsx:119-160, 287-296`) chỉ xét `validateRemotionScript` phía client rồi gọi `startRenderSaga`. Saga không chạy kiểm tra bố cục (orchestrator không gọi `/v1/check`). Vì vậy câu "không chặn dự án đi tới render" là đúng. Cái giá thật nằm ở chỗ khác: (1) vòng sửa tốn token cho các lỗi không sửa được hoặc không cần sửa, (2) chuỗi AI báo thất bại với câu sai tên lỗi.

### Harness bỏ qua gì

- `remotion_project/src/layout-probe/harness.tsx:72-74, 246`: harness bỏ qua mọi thứ nằm dưới `[data-cf-layer]`. Đó là các lớp nền và ánh sáng của `Scene` (`scene.tsx:134`, chỉ lớp có overscan), cùng `Glow`, `LightRays`, `Vignette`.
- `harness.tsx:301`: phần tử phủ **đúng trọn** khung được đánh dấu `full_frame` và luật bỏ qua nó (`layout_rules.py:248, 266`).
- `Camera` (`scene.tsx:83-97`) và lớp `children` của `Scene` (`scene.tsx:205`, `overscan={false}`) **không** có `data-cf-layer`. Vật trong máy quay được đo theo vị trí đã zoom và lia.
- `Panel` và `Backdrop` (`conceptflow-mini/illustration.tsx:129-195`) tự mô tả là "một vùng của khung, không phải một hình" và "nền toàn khung". Dù vậy chúng vẫn được đo như một hình.
- Vật trượt (`layout_rules.py:246-250, 267`): một vật có lúc ra ngoài khung thì chỉ bị xét ở các mẫu ≥ 85% shot. Luật này giả định vật trượt **vào**. Với vật trượt **ra** lúc cuối shot, mẫu 85% và 100% lại chính là lúc nó đang lấn ra.

### 29 lỗi của project 257acc58

Chạy lại `/v1/check/remotion` trên script hiện tại (2026-10-02) cho ra 29 diagnostic, tất cả là `safe_area`, cùng 12 cảnh báo cỡ vật trọng tâm. Chia theo nguyên nhân:

| Nhóm | Shot (dòng) | Ví dụ | Đánh giá |
|---|---|---|---|
| **A. Vùng nền và trang trí lớn bị đo như hình** | 2.2 (2307), 2.5 (2437), 3.1 (2526, 2541, 2577), 3.2 (2610, 2624, 2658), 3.3 (2742, 2744, 2755) — 11 lỗi | `Panel` nửa khung 960×1080 của màn chia đôi; vạch chia 2×1080; dải Ngân Hà `div` rộng 2200, mờ 16%; `<svg>` cỡ khung; `Backdrop` đặt trong `Camera` thay vì `Scene` | **Báo nhầm.** Các phần tử này cố ý phủ tới mép khung. Bắt LLM "sửa" chỉ làm hình xấu đi |
| **B. Máy quay zoom/lia đẩy vật ra mép** | 2.4 (2402, 2404), 9.3 (4550), 10.2 (4898), 10.3 (4951), có thể cả 4.5 (3403, 3404) | `camera` lia x 800→1200 ở zoom 1.15; zoom 1→1.55 vào cửa | **Một phần thật.** Prompt L8 chỉ yêu cầu vật trọng tâm và nhãn của nó ở trong vùng an toàn sau khi zoom. Vật phụ bị khung cắt là ý đồ của cú máy |
| **C. Vật trượt ra lúc cuối shot** | 7.2 (4028, x=-1050), 7.3 (4068, x=-525), có thể cả 9.5 (4726, 4735) | khối lệch sang trái hơn 1000px ở mẫu 85%/100% | **Báo nhầm theo luật vật trượt** (xem trên); cần đọc từng shot để chắc |
| **D. Lấn ít (≤ 40px)** | 2.1 (2283), 2.6 (2480), 6.5 (3894, 3895: 5px), 9.5 (4748, 4758: 2–5px), 10.1 (4861: 36px) | `Star` lấn 5px; `AntennaDish` lấn 18px | **Thật nhưng nhẹ.** Mắt người khó thấy, khung vẫn đúng |

Các nhóm trên là phân loại khi đọc code shot, chưa đo lại sau khi sửa. Bước `/code` sẽ đo lại từng nhóm trên chính script này và báo con số thật.

Đây không phải lỗi của riêng một project. Project `6deec24f` cũng có 15 `safe_area` và 12 `subtitle_zone` ở vòng đầu, project `82dbfbb7` có 7 `safe_area` (bảng `code_check_diagnostics`). Ở `257acc58`, số lỗi bố cục qua các vòng sửa của chunk là 29 → 34 → 16 → 1, rồi file gộp cuối cùng vẫn còn 29. Vòng sửa không hội tụ với nhóm A–C.

## Yêu cầu

1. **FR-1** Phần tử là vùng khung (`Panel`, `Backdrop`) và phần tử trang trí cắt ngang trọn một chiều của vùng an toàn không bị luật `safe_area`/`subtitle_zone` coi là hình.
2. **FR-2** Vật trượt ra khỏi khung lúc cuối shot không bị báo lấn ở các mẫu nó đang trượt ra.
3. **FR-3** Vật bị máy quay zoom/lia đẩy ra ngoài vùng an toàn được xử lý theo lựa chọn ở câu hỏi Q2.
4. **FR-4** Chỗ lấn nhẹ (≤ ngưỡng ở Q3) là cảnh báo, không vào vòng sửa.
5. **FR-5** Khi code đã lưu và chỉ còn lỗi bố cục, chuỗi AI và nhật ký nói "còn N lỗi bố cục (không chặn render)", không nói "lỗi biên dịch". Khi còn cả hai loại thì nói rõ số lượng của từng loại.
6. **FR-6** Lấn thật vẫn chặn như hôm nay: vật đứng yên lấn rõ ra ngoài vùng an toàn, chữ lấn vùng phụ đề, chữ đè chữ, chữ tràn, chữ nhỏ, shot ném lỗi.

**Tiêu chí nhận:** chạy lại `/v1/check/remotion` trên script hiện tại của `257acc58`, nhóm A, C (và B, D theo lựa chọn) không còn là diagnostic. Test `test_layout_rules.py` cũ vẫn qua, có thêm test cho từng nhóm. Ba lỗi cài sẵn trong `layout_probe_samples` vẫn bị chặn.

**Ngoài phạm vi:** sửa tay script của `257acc58`; đổi prompt Remotion Engineer; thêm cổng kiểm tra bố cục vào saga render.

## Giải pháp đề xuất

### Phương án 1 (khuyến nghị): luật đúng ý đồ, vẫn chặn lấn thật

1. **Vùng khung là phông cảnh.** Gắn `data-cf-layer="scenery"` cho `Panel` và `Backdrop` (`illustration.tsx`). Harness đã bỏ qua mọi thứ bên dưới thuộc tính này. Chữ và hình con đặt trong `Panel` cũng nằm dưới nó, nên `Panel` phải giữ lại các con. Cách làm: thuộc tính đi trên một lớp nền riêng (`<div data-cf-layer ...>` vẽ màu), còn `children` nằm ở lớp anh em không mang thuộc tính. Như vậy chữ trong panel vẫn được đo.
2. **Phần tử cắt ngang trọn một chiều.** Trong `layout_rules.py`, phần tử không phải chữ mà phủ trọn chiều ngang **hoặc** trọn chiều dọc của vùng an toàn (lấn qua cả hai mép đối diện, ví dụ dải rộng 2200, vạch 2×1080) được coi là trang trí của khung. Nó bỏ qua `safe_area`/`subtitle_zone`, nhưng vẫn được tính trong chữ đè chữ (vốn chỉ xét chữ). Cách này bắt được dải sao, vạch chia, `<svg>` cỡ khung mà không cần đánh dấu trong code.
3. **Vật trượt ra.** Hiện nay vật trượt chỉ bị xét ở mẫu ≥ 85%. Đổi thành: vật trượt bị xét ở các mẫu mà ở cả mẫu đó lẫn **mọi mẫu sau** nó đều nằm trọn trong khung (đã yên ở chỗ đứng cuối). Vật trượt vào rồi đứng yên vẫn bị xét như hôm nay. Vật đang trượt ra thì không.
4. **Máy quay** (theo Q2, khuyến nghị (a)): `Camera` và lớp `children` của `Scene` ghi `data-cf-zoom={zoom}` và `data-cf-pan="1"` khi máy đang zoom hoặc lia so với khung gốc. Harness ghi `camera_moved: true` cho phần tử nằm dưới chúng. Luật: `safe_area` của phần tử có `camera_moved` thành cảnh báo "máy quay đẩy X ra ngoài vùng an toàn", **trừ** vật trọng tâm của shot (vật lớn nhất, đã tính sẵn ở `layout_rules.py:268-271`). Vật trọng tâm vẫn chặn, đúng như prompt L8.
5. **Lấn nhẹ** (theo Q3, khuyến nghị 1% cạnh ngắn của khung: 11px ngang, 11px dọc short): chỗ lấn ≤ ngưỡng thì thành cảnh báo cùng tên luật. Mỗi phát hiện đã có `score = worst`, nên chỉ cần so `worst` với ngưỡng.
6. **Thông báo đúng tên.** llm-service đã trả `kind` (`compile`/`layout`) cho từng diagnostic. authoring-service đếm theo `kind` khi dựng ghi chú chuỗi và chi tiết nhật ký:
   - chỉ còn lỗi bố cục: "Đã sinh và lưu code; còn N lỗi bố cục sau K vòng sửa (không chặn render — vài hình có thể lấn mép khung): …";
   - còn lỗi biên dịch: giữ câu "vẫn lỗi biên dịch", thêm "(+M lỗi bố cục)" nếu có.

   `ErrKindCheckFailed` vẫn giữ: không giả là đạt.

Ưu: vòng sửa chỉ nhận lỗi đáng sửa, nên đỡ token và hội tụ được; lấn thật vẫn bị chặn; không đổi contract. Nhược: sửa ở bốn chỗ (kit, harness, luật, authoring-service). Cần test kỹ để luật không bỏ sót vật thật.

### Phương án 2: `safe_area` chỉ còn là cảnh báo

Bỏ `SAFE_AREA` khỏi `BLOCKING_RULES`, giữ các luật khác chặn, cộng thêm bước 6 của PA1. Ưu: một dòng, hết ngay mọi báo nhầm. Nhược: mất luôn việc tự sửa vật trọng tâm hay chữ thật sự bị cắt ở mép. Đây là lỗi mà vòng sửa từng bắt được, ví dụ spring vọt lố trong `test_spring_overshoot_between_the_old_four_samples_is_caught`.

### Phương án 3: chỉ sửa thông báo

Giữ luật nguyên, chỉ làm bước 6. Ưu: nhỏ nhất. Nhược: vòng sửa vẫn đốt token cho nhóm A–C, và lần sau vẫn còn chừng ấy lỗi báo nhầm.

**Khuyến nghị: Phương án 1. Creator đã chọn Phương án 1.**

### Câu hỏi cho Creator

- **Q1.** Chọn phương án nào? (1 — khuyến nghị / 2 / 3)
- **Q2.** (PA1) Vật **không phải** trọng tâm bị máy quay đẩy ra ngoài vùng an toàn thì xử lý thế nào?
  - (a) cảnh báo — khuyến nghị;
  - (b) vẫn chặn như hôm nay;
  - (c) bỏ qua hẳn.
- **Q3.** (PA1) Ngưỡng lấn nhẹ thành cảnh báo:
  - (a) 1% cạnh ngắn của khung, khoảng 11px — khuyến nghị;
  - (b) 2%, khoảng 22px;
  - (c) không có ngưỡng, giữ 0.5px.

## Phụ thuộc

Đã kiểm tra: `git branch -a --no-merged origin/main` (feature/cr-*, fix/*, chore/*) không có nhánh nào. `scripts/worktree.sh list` chỉ có checkout chính (main sạch) và worktree `chore/worktree-per-cr`, nhánh này đã merge vào main và không còn thay đổi chưa commit. **Độc lập.**

## Phạm vi (PA1)

- **rendering**
  - `remotion_project/src/conceptflow-mini/illustration.tsx`: `Panel`, `Backdrop`.
  - `remotion_project/src/conceptflow-mini/scene.tsx`: `Camera`, `Layer` (thuộc tính zoom/lia).
  - `remotion_project/src/layout-probe/harness.tsx`: `collect` (ghi `camera_moved`).
  - `domain/layout_rules.py`: `_evaluate_shot`, `_check_placement`, `LayoutFinding.blocking` (đổi sang cờ do luật đặt, không chỉ theo tên luật).
  - `tests/domain/test_layout_rules.py`; mẫu JSON nếu cần.
  - `graphify affected evaluate()`: chỉ `CheckScriptUseCase._check_layout` và test của nó. `Panel` được gọi ở `gallery/tooth-decay-demo.tsx` (hình không đổi, chỉ thêm thuộc tính).
- **authoring-service**
  - `internal/application/generate_authoring_code.go`: giữ `kind` khi dựng `out.Diagnostics`, thêm đếm số lỗi theo loại vào `GeneratedStep`.
  - `internal/application/authoring_chain.go`: ghi chú.
  - `internal/application/generate_authoring.go`: chi tiết nhật ký và log.
  - Test tương ứng.
- **web-gui**: không đổi. Ghi chú hiện nguyên văn từ server, `ScriptOutlineStepPage.test.tsx:334-352` vẫn đúng với trường hợp lỗi biên dịch.
- Contract, DB, migration: không đổi. Trường `GeneratedStep` mới chỉ là thêm trường JSON (`layout_issues`/`compile_issues`, kiểu số); web-gui cũ bỏ qua được.

## Kế hoạch thực hiện (PA1, Q2 = a, Q3 = a — đã chốt)

1. `illustration.tsx`
   - `Backdrop`: thêm `data-cf-layer="scenery"` lên `AbsoluteFill` gốc.
   - `Panel`: tách thành `div` ngoài (vị trí, cỡ, `overflow: hidden`, `opacity`, không màu), bên trong là một `div` nền `inset: 0` mang màu, `borderRadius` và `data-cf-layer="scenery"`, rồi đến `children`. Hình hiển thị phải giữ nguyên.
   - Cập nhật JSDoc: vùng khung được kiểm tra bố cục coi là phông cảnh.
2. `scene.tsx`
   - `Camera` và `Layer` khi `overscan === false`: thêm `data-cf-camera="moved"` khi `cam` khác khung gốc (zoom ≠ 1 hoặc tâm ≠ (width/2, height/2) quá 0.5px).
   - Cập nhật chú thích đầu file.
3. `harness.tsx` `collect`: `entry.camera_moved = !!el.closest('[data-cf-camera="moved"]')`; thêm trường vào kiểu `Box`.
4. `layout_rules.py`
   - Thêm hằng `SOFT_EXCESS_FRACTION = 0.01`.
   - `LayoutFinding` có thêm trường `soft: bool`; `blocking` = `rule in BLOCKING_RULES and not soft`. Thông báo cảnh báo nói rõ lý do ("máy quay đẩy … ra ngoài", "lấn nhẹ …").
   - `_evaluate_shot`:
     - (a) tính `settled_from[ident]` = chỉ số mẫu đầu tiên mà từ đó tới cuối vật luôn nằm trọn trong khung. Vật trượt chỉ bị xét ở các mẫu từ chỉ số đó; vật trượt ra thì không có chỉ số nào. Thay cho `settled or ident not in sliding` ở phần placement. Cỡ chữ vẫn dùng `settled` như cũ.
     - (b) bỏ `safe_area`/`subtitle_zone` cho phần tử không phải chữ mà phủ trọn một chiều của vùng an toàn.
     - (c) truyền `hero` (tính sau một lượt đầu qua các mẫu, nên phải tách hai lượt: lượt 1 tìm `hero` và `settled_from`, lượt 2 chấm).
   - `_check_placement`: `soft = (camera_moved and ident != hero_ident) or worst <= SOFT_EXCESS_FRACTION * min(w, h)`.
   - Cập nhật docstring đầu module (bảng chặn/cảnh báo).
5. `tests/domain/test_layout_rules.py`, mỗi nhóm một test:
   - panel và backdrop không có trong số đo (test harness nếu có, nếu không thì test luật với phần tử đánh dấu);
   - dải trang trí cắt ngang khung không bị báo;
   - vật trượt ra lúc cuối không bị báo, vật trượt vào rồi lấn vẫn bị báo;
   - vật phụ bị máy quay đẩy ra mép chỉ là cảnh báo, vật trọng tâm bị đẩy ra vẫn chặn;
   - lấn 5px là cảnh báo, lấn 40px là chặn.

   Các test cũ phải qua nguyên vẹn.
6. authoring-service
   - `generate_authoring_code.go`: đếm `d.Kind == "layout"` và phần còn lại vào `out.LayoutIssues` và `out.CompileIssues`.
   - `authoring_chain.go`: ghi chú theo FR-5.
   - `generate_authoring.go`: chi tiết `recordEvent` và lỗi trong `logError` nói đúng loại.
   - Test: `authoring_chain_test.go` và `generate_authoring_code_test.go` cho ba trường hợp: chỉ bố cục, chỉ biên dịch, cả hai.
7. Lint/format: ruff (rendering), `gofmt` + `go vet` (authoring-service), `tsc --noEmit` của `remotion_project`.

## Kiểm tra

- `pytest services/rendering/tests/domain/test_layout_rules.py` và toàn bộ test rendering; `go test ./...` trong authoring-service.
- Rebuild: `scripts/worktree.sh rebuild rendering authoring-service`, xác nhận healthy.
- Kiểm trực tiếp:
  - gọi `/v1/check/remotion` với script hiện tại của `257acc58`, báo số lỗi trước (29) và sau, chia theo nhóm A–D;
  - chạy `layout_probe_samples`: ba lỗi cài sẵn vẫn chặn;
  - xem một khung render của shot 3.1 (màn chia đôi) để chắc `Panel` hiển thị như cũ.

## Rủi ro

- Luật nới ra có thể bỏ sót vật thật, ví dụ một hình lớn cố ý rộng hơn vùng an toàn. Bước 4(b) chỉ áp cho phần tử **cắt qua cả hai mép đối diện**, và hình như vậy đằng nào cũng bị khung cắt. Test giữ các lỗi cài sẵn.
- Đổi cấu trúc DOM của `Panel` có thể làm lệch hình nếu code shot dựa vào con trực tiếp của nó. Kiểm bằng ảnh render shot có panel.
- Không mất dữ liệu, không đổi DB/contract. Script đã lưu không bị đụng tới.
