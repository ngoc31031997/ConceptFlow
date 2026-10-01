# CR-058 — Thiết kế: sắp xếp lại luồng tạo video cho dễ hiểu, dễ thao tác

## Yêu cầu gốc (nguyên văn)

> review và sắp xếp lại workflow tạo video, tôi muốn tách biệt thành các bước và màn hình riêng biệt hoặc bằng cách nào đó làm cho quy trình tạo video trở nên dễ dàng thao tác và dễ hiểu hơn cho người dùng. nói chung là tối ưu trải nghiệm UX UI cho người dùng đó

Trả lời đề xuất:

> làm B đi

Chốt: hướng B; câu 2 và 3 theo mặc định (bỏ hẳn nhánh "đã có sẵn"; một tracker chung cho Sản xuất).

Sau khi xem bản code đầu, Creator xem bản mẫu HTML (https://claude.ai/artifact/SDR1QRjERhPvm67wGWxG82) và đổi phần điều hướng:

> tôi đang muốn đưa các step về menu sibar bên trái hết mà, tốt nhất bạn nên render thử 1 file html ý tưởng của bạn hoặc artiface dơn giản để tôi review trướ

> ko xem màn đi detail của mỗi menu con được hả

> tôi cũng thích ý tương 2 menu như cũ hơn vì như hiện tại mỗi lần muốn xem danh sách tôi phải scroll khá mệt

> ok theo ý tưởng này đi

Chốt (bản mẫu phiên bản 3): giữ hai lớp menu bên trái; các bước nằm ở menu bước; bỏ thanh giai đoạn ở đầu màn (FR3 thay bằng FR3a–FR3d).

## Hiện trạng

Luồng đã có 14 bước, mỗi bước một URL (`services/web-gui/src/App.tsx:53-85`). Server tự suy ra bước hiện tại (`flow_step`, `run_state`) từ trạng thái saga (`services/orchestrator/internal/domain/flow.go:81`), nên **thứ tự và số bước là của server, giao diện chỉ hiển thị**. Các vấn đề tìm thấy khi đọc code:

### V1. 14 bước ngang hàng, tên kỹ thuật, không phân biệt bước "bạn làm" với bước "máy tự chạy"
- `FLOW_LABELS` (`src/utils/flow.ts:14-29`): "Validate", "Review", "TTS", "Render", "Merge", "Publish", "Visual", "Code". Người mới không hiểu TTS/Merge là gì.
- 5/14 bước (7, 9, 10, 11, 12) chạy tự động, Creator không có gì để làm, nhưng menu bước (`src/components/StepRail.tsx:96-129`) vẽ chúng y như các bước cần thao tác.
- Có 4 giai đoạn (`FLOW_PHASES`, `flow.ts:84-89`) nhưng chỉ thấy được trong menu dọc, và menu dọc **chỉ hiện khi đã có project** (`AppShell.tsx:62`). Ở bước 1 (chưa có project) Creator không thấy phía trước còn gì.
- Thanh bên trái ghi "▶ Tiếp tục · bước 9/14" (`AppShell.tsx:107-110`): con số không nói lên đang làm gì.

### V2. Cài đặt nằm rải rác ở 4 chỗ, có chỗ lặp
| Cài đặt | Chọn ở đâu |
|---|---|
| Ngôn ngữ, engine, cách làm (AI/tự làm), model AI, giọng đọc, format, kiểu đầu ra | Bước 2 (`ScriptAuthoringSettingsStepPage.tsx:103-172`), một cột dài 7 bộ chọn, không chia nhóm |
| Kiểu video (archetype) + gợi ý đổi format | Bước 3 (`ScriptOutlineStepPage.tsx:214-225`), cách xa bộ chọn format ở bước 2 |
| Engine, cách làm, model (lần 2) | Thanh "Đã chọn … Đổi" ở bước 3, 4, 5, 6 (`PipelineSettingsBar.tsx:70-106`) |
| Chất lượng, font, phụ đề, nhạc nền | Bước 8, cột phải, chen giữa danh sách lời thoại và nút Duyệt (`ValidatePage.tsx:162-172`) |

### V3. Chủ đề nhập ở hai nơi
Bước 1 chỉ có một ô chủ đề (`ScriptStepPage.tsx:62-73`); bước 3 lại cho sửa chủ đề lần nữa và tự lưu theo debounce (`ScriptOutlineStepPage.tsx:56-76, 206-213`), kèm cảnh báo trùng chủ đề (`:226-239`).

### V4. Bố cục bước 3–6 không thống nhất
Bước 3 đặt thanh chạy AI **dưới** hai thẻ (`ScriptOutlineStepPage.tsx:287-325`); bước 4, 5, 6 đặt nó **trên** (`VisualDirectorStepPage.tsx:142-161`, `IllustrationsStepPage.tsx:75-88`, `ManimEngineerStepPage.tsx:212-238`). Lời nhắc dưới đáy vẫn nói "Dán dàn ý từ AI để tiếp tục" cả khi đang ở chế độ AI (`ScriptOutlineStepPage.tsx:175-181`, `VisualDirectorStepPage.tsx:122-128`).

### V5. Nhánh giao diện không bao giờ chạy tới
`scriptSource` chỉ được khởi tạo là `"idea"` (`ProjectDraftContext.tsx:179`) và không có chỗ nào dispatch `SET_SCRIPT_SOURCE`. Vì vậy các nhánh "đã có dàn ý / storyboard / code" (`hasOwnOutline`, `hasOwnStoryboard`, `hasOwnCode`) và khối "Code chưa đúng chuẩn? Nhờ AI chỉnh lại" cùng `ScriptAssistant` (`ManimEngineerStepPage.tsx:245-266`) không bao giờ hiện. Comment ở các trang còn nhắc `ScriptPipelineTabs`, component không còn tồn tại.

### V6. Màn Duyệt (bước 8) quá dày, nút "Quay lại sửa script" đưa về sai chỗ
- Một màn chứa: danh sách lời thoại (sửa được), bảng cài đặt xuất video, nút Duyệt/Từ chối, thanh tiến trình (`ValidatePage.tsx:153-182`).
- "Quay lại sửa script" đưa về `/` (bước 1, ô chủ đề) (`ValidatePage.tsx:68`), trong khi thứ cần sửa là code (bước 6). Creator phải tự bấm qua 5 màn.

### V7. Sản xuất (9–12) bị chẻ thành 4 màn, mỗi màn chỉ thấy một việc
`RenderPage.tsx:100-104` lọc tracker chỉ còn bước đang chạy; Creator không thấy tổng thể "đã xong giọng đọc, đang dựng hình, còn ghép và cắt short".

## Yêu cầu

**FR1. Năm giai đoạn dễ hiểu.** Giao diện nhóm 14 bước của server thành 5 giai đoạn: Chuẩn bị (1–2), Soạn nội dung (3–6), Duyệt (7–8), Sản xuất (9–12), Hoàn tất (13–14). Số bước và `flow_step` của server không đổi.

**FR2. Tên bước bằng lời thường.** 1 Ý tưởng · 2 Cấu hình · 3 Kịch bản · 4 Hình ảnh · 5 Hình minh hoạ · 6 Code · 7 Kiểm tra tự động · 8 Duyệt nội dung · 9 Giọng đọc · 10 Dựng hình · 11 Ghép video · 12 Cắt short · 13 Kết quả · 14 Đăng video. Mọi nơi dùng `FLOW_LABELS` (tiêu đề màn, menu bước, danh sách video, nhật ký, hộp "tạo bản mới") tự đổi theo.

**FR3a. Hai lớp menu bên trái.** Menu chính (Tạo video mới, Tiếp tục, Danh sách video, Nhật ký, nhóm Cài đặt) giữ nguyên chỗ và luôn ngắn. Các bước nằm ở menu bước (cột thứ hai). Không có thanh giai đoạn ở đầu màn; đầu màn chỉ có dòng nhỏ "<GIAI ĐOẠN> · BƯỚC n/14" trên tiêu đề.

**FR3b. Menu bước hiện trên mọi màn của luồng, kể cả bước 1** khi chưa có project. Đầu menu: tên video (hoặc "Video mới"), dòng "<giai đoạn> · <bước> · x/y bước", thanh tiến độ.

**FR3c. Menu bước chia 5 giai đoạn thu gọn được.** Mỗi giai đoạn có số (✓ khi xong), tên, "đã xong/tổng" (không tính bước "Không dùng"). Giai đoạn chứa bước đang xem tự mở; bấm tên giai đoạn để mở/đóng. Ở chế độ thu gọn menu (64px), mọi bước vẫn hiện dạng dấu tròn.

**FR3d. Bấm được mọi bước.** Bước đã tới mở màn thật như hiện nay. Bước chưa tới, hoặc "Không dùng", mở màn xem trước (`/projects/:id/preview/:step`, hoặc `/create/preview/:step` khi chưa có project): tiêu đề bước, bước đó làm gì, dòng báo "Chưa tới bước này" hoặc lý do "Không dùng", và nút quay lại. Màn xem trước không có ô nhập hay nút hành động.

**FR4. Bước tự động được đánh dấu.** Trong menu bước, bước 7, 9, 10, 11, 12 có nhãn "Tự động" và kiểu chữ nhạt hơn, để Creator biết mình không phải làm gì ở đó.

**FR5. Bước 2 chia ba nhóm có đánh số**, theo đúng thứ tự quyết định: "1. Nội dung video" (ngôn ngữ, kiểu video, format, kiểu đầu ra), "2. Giọng đọc", "3. Cách soạn" (engine, AI/tự làm, model). Bộ chọn kiểu video chuyển từ bước 3 về đây, nằm ngay trên bộ chọn format mà nó gợi ý. Chủ đề hiện ở đầu bước 2 dạng chỉ đọc, kèm nút "Sửa" về bước 1. Cảnh báo trùng chủ đề hiện ở đầu bước 2.

**FR6. Bước 3 không còn ô chủ đề.** Chủ đề chỉ sửa ở bước 1. Bước 3 hiện chủ đề dạng chỉ đọc.

**FR7. Bước 3–6 cùng một bố cục**, trên xuống: (a) thanh "Đã chọn … Đổi" + nút chạy AI, (b) ở chế độ tự làm: thẻ "1. Sao chép prompt" và "2. Dán kết quả"; ở chế độ AI: một thẻ kết quả, (c) thanh Tiếp tục. Lời nhắc dưới đáy nói đúng chế độ ("Bấm Chạy bằng AI, hoặc tự viết…" ở chế độ AI; "Dán … từ AI" ở chế độ tự làm).

**FR8. Bỏ nhánh không dùng tới**: `scriptSource`, `SET_SCRIPT_SOURCE`, các nhánh `hasOwn*`, khối `ScriptAssistant` ở bước Code và component `ScriptAssistant` (không còn ai dùng; `ScriptAssistant.module.css` giữ lại vì `ShortScriptAssistant` dùng chung).

**FR9. Màn Duyệt tách hai phần**, cùng URL, phân biệt bằng `?part=settings`:
- Phần 1 "Duyệt lời thoại": danh sách lời thoại (rộng) + cột phải có tiến trình kiểm tra; thanh dưới đáy: "Quay lại sửa script" (phụ) và "Tiếp: cài đặt xuất video" (chính).
- Phần 2 "Cài đặt xuất video": bảng chất lượng / font / phụ đề / nhạc nền chiếm cả màn; thanh dưới đáy: "Quay lại duyệt lời thoại" và "Duyệt và bắt đầu tạo video".
- Nút Back của trình duyệt và tải lại trang giữ đúng phần đang xem.

**FR10. "Quay lại sửa script" mở thẳng bước Code** (`/projects/:id/resume?step=6`), nạp lại bản nháp từ server.

**FR11. Màn Sản xuất cho thấy cả 4 việc** (giọng đọc, dựng hình, ghép video, cắt short) trong một tracker; việc đang chạy nổi bật. "Cắt short" không hiện khi dự án chỉ làm video dài. Bấm bước 9/10/11/12 trên menu đều mở màn này.

**FR12. Nút "Tiếp tục" ở menu chính** ghi "▶ Tiếp tục · <giai đoạn> · <tên bước>" thay vì "bước n/14".

### Tiêu chí chấp nhận
- Một Creator mới, không đọc tài liệu, nhìn đầu màn bất kỳ biết mình ở giai đoạn nào và còn mấy giai đoạn.
- Mỗi cài đặt chỉ có **một** chỗ chọn chính (bước 2 hoặc phần 2 của màn Duyệt). Thanh "Đổi" ở bước 3–6 vẫn còn, là lối tắt sửa nhanh, không phải chỗ chọn lần đầu.
- Không có thay đổi API, DB, contract, migration. `flow_step` server trả về giữ nguyên ý nghĩa.
- Toàn bộ test web-gui pass; `tsc --noEmit` sạch; eslint không thêm lỗi.

### Ngoài phạm vi
- Đổi số bước, thêm/bớt bước ở server, đổi trạng thái saga.
- Đổi nội dung prompt, logic chuỗi AI (`AuthoringModeBar`), cách chuỗi AI tự chuyển màn khi chạy.
- Màn cài đặt (prompt, kiểu video, thư viện hình), danh sách video, nhật ký, trừ phần tên bước tự đổi theo FR2.

## Giải pháp đề xuất

### Ba hướng đã cân nhắc

| | Hướng | Được | Mất |
|---|---|---|---|
| A | Sửa tại chỗ: đổi tên bước, chia nhóm bước 2, sửa lời nhắc, bỏ code chết | Ít rủi ro, nhanh | Không giải quyết cảm giác "14 bước dài", màn Duyệt vẫn dày |
| **B (đề xuất)** | **A + 5 giai đoạn có thanh định hướng, đánh dấu bước tự động, tách màn Duyệt thành 2 phần, gộp tracker Sản xuất** | Creator luôn biết mình đang ở đâu; mỗi màn chỉ một việc; chỉ đụng web-gui | Nhiều test phải sửa; duyệt cần thêm một lần bấm |
| C | Đổi hẳn sang một "không gian dự án" dạng tab, bỏ wizard | Linh hoạt cho người dùng quen | Viết lại gần hết web-gui, đổi cách điều hướng đã ổn định, rủi ro cao, không khớp quy tắc luồng trên→dưới hiện tại |

**Đề xuất B**, vì: vấn đề lớn nhất không phải thiếu màn hình (luồng đã tách 14 màn) mà là Creator không thấy bức tranh tổng, không biết bước nào phải làm, và cài đặt nằm rải rác. B sửa đúng ba chỗ đó mà không đụng server.

### Vì sao giữ 14 bước của server
`flow_step` dùng chung cho wizard, danh sách video, nhật ký, fork, và có test ở hai service. Gom nhóm ở giao diện cho cùng kết quả với người dùng mà không phải đổi contract.

### Vì sao tách màn Duyệt thay vì đưa cài đặt xuất video về bước 2
Chất lượng, phụ đề, nhạc nền chỉ bước Dựng hình/Ghép video đọc, và vẫn sửa được khi các bước đó lỗi (`RenderPage.tsx:18-22`). Đặt chúng ngay trước nút "bắt đầu tạo video" là đúng thời điểm quyết định (bước tốn tiền bắt đầu sau đó), chỉ cần cho chúng một màn riêng thay vì chen cạnh danh sách lời thoại.

### Câu hỏi đã chốt
1. **Hướng làm**: **B** (Creator chọn).
2. **Nhánh "đã có sẵn dàn ý / storyboard / code"** (V5): **bỏ hẳn (đề xuất)**, hay khôi phục lại lựa chọn này ở bước 1? Khôi phục là một tính năng mới, nên tách CR riêng nếu muốn.
3. **Màn Sản xuất** (FR11): **một tracker chung cho cả 4 việc (đề xuất)**, hay giữ mỗi bước 9–12 một màn như hiện tại?

## Phạm vi

Chỉ service **web-gui**. Không đổi contract, DB, migration, API.

| File | Thay đổi |
|---|---|
| `src/utils/flow.ts` | `FLOW_LABELS` mới; `FLOW_PHASES` 5 giai đoạn; `AUTO_STEPS`; `phaseOf(step)` |
| `src/components/PhaseStepper.tsx` + `.module.css` (mới) | Thanh 5 giai đoạn |
| `src/components/AppShell.tsx` | Gắn `PhaseStepper`; nhãn nút "Tiếp tục" |
| `src/components/StepRail.tsx` + `.module.css` | Nhãn "Tự động" |
| `src/pages/ScriptStepPage.tsx` | Đưa `similarProjects` sang bước 2 qua router state |
| `src/pages/ScriptAuthoringSettingsStepPage.tsx` | 3 nhóm, chủ đề chỉ đọc, bộ chọn kiểu video, cảnh báo trùng |
| `src/pages/ScriptOutlineStepPage.tsx` | Bỏ ô chủ đề, kiểu video, effect debounce; bố cục FR7; lời nhắc |
| `src/pages/VisualDirectorStepPage.tsx`, `ManimEngineerStepPage.tsx` | Bỏ nhánh `hasOwn*`, `ScriptAssistant`; lời nhắc |
| `src/context/ProjectDraftContext.tsx` | Bỏ `ScriptSource`, `scriptSource`, `SET_SCRIPT_SOURCE` |
| `src/components/ScriptAssistant.tsx` | Xoá (cùng `tests/components/ScriptAssistant.test.tsx`) |
| `src/pages/ValidatePage.tsx`, `src/components/OutlineActions.tsx` | Hai phần `?part=settings`; điều hướng "Quay lại sửa script" |
| `src/pages/RenderPage.tsx` | Tracker chung |
| `src/pages/WizardSteps.module.css` | Kiểu cho nhóm có số ở bước 2, dòng chủ đề chỉ đọc |

`graphify affected` cần chạy ở bước code cho: `FLOW_LABELS`, `FLOW_PHASES`, `ScriptSource`, `ScriptAssistant`, `OutlineActions`, `useOutlineReview`. Từ đọc code: `FLOW_LABELS` được dùng ở `StepRail`, `StatusStrip`, `ForkDialog`, `VideoListPage`, `JournalPage`, `AuthoringModeBar`, `pipelineLabels.stepLabel` và mọi trang bước (qua `flowTitle`); `FLOW_PHASES` chỉ ở `StepRail`; `ScriptAssistant` chỉ ở `ManimEngineerStepPage`.

## Kế hoạch thực hiện

Theo `docs/ux-ui-design-rules.md` (trên→dưới, Neubrutalism, token trong `theme.css`, ẩn/hiện dùng `.reveal`/`usePresence`) và `docs/code-standards-rules.md` (JSDoc cho mọi export, comment không nhắc CR/lịch sử, xoá comment nhắc `ScriptPipelineTabs`).

1. **`src/utils/flow.ts`**
   - `FLOW_LABELS` = ["Ý tưởng", "Cấu hình", "Kịch bản", "Hình ảnh", "Hình minh hoạ", "Code", "Kiểm tra tự động", "Duyệt nội dung", "Giọng đọc", "Dựng hình", "Ghép video", "Cắt short", "Kết quả", "Đăng video"].
   - `FLOW_PHASES` = [{name:"Chuẩn bị", steps:[1,2]}, {name:"Soạn nội dung", steps:[3,4,5,6]}, {name:"Duyệt", steps:[7,8]}, {name:"Sản xuất", steps:[9,10,11,12]}, {name:"Hoàn tất", steps:[13,14]}].
   - `export const AUTO_STEPS: ReadonlySet<number> = new Set([7, 9, 10, 11, 12])`.
   - `export function phaseOf(step: number): { index: number; name: string; steps: readonly number[] } | null`.
   - Sửa các chuỗi trong `readOnlyReason` còn nhắc "bước Kiểm tra" cho khớp tên mới ("bước Duyệt nội dung").
2. **`src/components/PhaseStepper.tsx`** (mới): prop `currentStep`. Dùng `useStepNav(currentStep).status` để tính mỗi giai đoạn: `done` khi mọi bước (trừ `skipped`) là `done`; `current` khi chứa `currentStep`; còn lại `pending`. Hiển thị 5 ô ngang (`<ol aria-label="Các giai đoạn">`, ô hiện tại `aria-current="step"`), dưới ô hiện tại: "Bước {vị trí trong giai đoạn}/{số bước không skipped} · {FLOW_LABELS}". Chỉ hiển thị, không bấm được (điều hướng vẫn ở menu bước, một luật duy nhất). Màn hẹp: chỉ hiện số + tên giai đoạn hiện tại. CSS dùng token (`--accent` cho ô hiện tại, `--success` cho ✓, `--border-w`, `--shadow-sm`).
3. **`src/components/AppShell.tsx`**: render `<PhaseStepper currentStep={currentStep} />` ngay trên `.heading` khi `currentStep` có giá trị (kể cả khi chưa có project). Nút "Tiếp tục" ở sidebar: `▶ Tiếp tục · {phaseOf(flow_step).name} · {FLOW_LABELS[flow_step-1]}` (giữ " · đang chạy").
4. **`src/components/StepRail.tsx`**: với bước thuộc `AUTO_STEPS`, thêm `<span className={styles.autoTag}>Tự động</span>` cạnh tên (ẩn khi thu gọn, đưa vào `title`), class `styles.auto` làm chữ nhạt. Cập nhật comment "13 bước" → "14 bước".
5. **`src/pages/ScriptStepPage.tsx`**: `handleContinue` lấy `{ similarProjects }` từ `createProjectDraft` và `navigate("/create/script/settings", { state: { similarProjects } })`.
6. **`src/pages/ScriptAuthoringSettingsStepPage.tsx`**:
   - Đầu trang: dòng chủ đề chỉ đọc + nút "Sửa" (`navigate("/")`); dưới đó banner trùng chủ đề đọc từ `location.state?.similarProjects` (chuyển markup + class `topicCollisionBanner` từ bước 3).
   - Ba `Card` có tiêu đề "1. Nội dung video", "2. Giọng đọc", "3. Cách soạn", nội dung như bảng FR5. Bộ chọn kiểu video: `VideoArchetypePicker` với `onTopicChange` → dispatch `SET_AUTHORING_TOPIC` + `createProjectDraft(projectId, topic, voiceLanguage, renderEngine)` (lỗi → `saveError` như các field khác, không nuốt lỗi); `onFormatChange` → cùng đường với `VideoFormatPicker` (`dispatch` + `send({ videoFormatId })`).
   - Cập nhật subtitle trang cho khớp ("Chất lượng, phụ đề và nhạc nền chọn ở bước Duyệt nội dung").
7. **`src/pages/ScriptOutlineStepPage.tsx`**:
   - Xoá ô chủ đề, `VideoArchetypePicker`, effect debounce `createProjectDraft`, `similarProjects`, `useDebounce`, `patchWizardSettings`.
   - Đầu trang: dòng chủ đề chỉ đọc (cùng component nhỏ với bước 2 — tách `TopicSummary` vào `src/components/TopicSummary.tsx`, prop `topic`, `onEdit`).
   - Thứ tự: `TopicSummary` → `PipelineSettingsBar` (giữ `beforeRun`) → hai thẻ (tự làm) hoặc một thẻ kết quả (AI, dùng `scriptLayoutSingle` như bước 4/6) → `WizardNav`.
   - `runDisabled` vẫn khi chủ đề rỗng, lý do "Chưa có chủ đề — quay lại bước 1".
8. **Lời nhắc theo chế độ** (bước 3, 4, 6): AI → "Bấm Chạy bằng AI, hoặc tự viết {thứ} vào ô bên dưới."; tự làm → "Dán {thứ} từ AI để tiếp tục"; có nội dung → câu "sẵn sàng" như hiện tại.
9. **Bỏ `scriptSource`**: xoá type `ScriptSource`, field, action, case reducer, giá trị khởi tạo trong `ProjectDraftContext.tsx`; xoá mọi nhánh `hasOwnOutline/hasOwnStoryboard/hasOwnCode` ở bước 3, 4, 6 (giữ nhánh mặc định); ở bước 6 xoá khối `Disclosure` + `ScriptAssistant` + `useScriptTemplates` nếu không còn dùng; xoá `src/components/ScriptAssistant.tsx` và test của nó. Kiểm `graphify affected "ScriptAssistant"` và `"useScriptTemplates"` trước khi xoá. Kiểm bản nháp đọc từ localStorage không vỡ khi còn key `scriptSource` cũ (reducer bỏ qua field thừa).
10. **`src/pages/ValidatePage.tsx` + `OutlineActions.tsx`**:
    - `const part = search.get("part") === "settings" ? "settings" : "review"`.
    - Phần `review` (khi `isAwaitingReview`): `OutlineReview` + cột phải chỉ có `ProgressTracker`; `WizardNav` với `onBack` = `outline.reject` (nhãn "Quay lại sửa script"), `onNext` = `setSearchParams({ part: "settings" })` (push, để Back trình duyệt về phần 1), nhãn "Tiếp: cài đặt xuất video", `nextDisabled={outline.busy}`.
    - Phần `settings`: `ProductionSettingsPanel` (stages render+merge) rộng; `WizardNav` `onBack` = về phần 1, `onNext` = `outline.approve`, nhãn "Duyệt và bắt đầu tạo video", `nextDisabled={outline.busy}`.
    - `OutlineActions` không còn dùng ở màn này → xoá component nếu `graphify affected` cho thấy không ai khác dùng.
    - Callback reject: `dispatchDraft({ type: "RESUME_EDITING" }); navigate(\`/projects/${projectId}/resume?step=${FLOW_CODE}\`)`.
    - Chế độ xem lại (`?view=1`) giữ như cũ, không có thanh hành động.
    - `WizardNav` khoá nút khi `flow.project && !flow.editable` — kiểm `awaiting_review` có bị coi là không sửa được không (`isAuthoringEditable` trả false cho `awaiting_review`). Nếu có, thêm prop `allowWhenLocked` cho `WizardNav` (mặc định false) và bật ở màn Duyệt; ghi JSDoc rõ lý do: hành động ở màn Duyệt là của saga, không phải sửa bước soạn.
11. **`src/pages/RenderPage.tsx`**: `shownSteps = PROCESS_STEPS.filter(s => s !== "generate_clips" || (project?.video_output_mode || "long") !== "long")`; tracker luôn nhận cả danh sách; `currentStep` của `AppShell` = `viewOnly ? viewStep : activeFlowStep`; tiêu đề `flowTitle(...)` giữ; subtitle nói việc đang chạy. Phần sửa cài đặt sau lỗi + `ErrorBanner` giữ nguyên điều kiện.
12. **Dọn comment** ở các file đã sửa: bỏ nhắc `ScriptPipelineTabs`, "1a/1b/1c", "situation-chooser", "Trước đây…" (quy tắc §1).
13. **Test** (`services/web-gui/tests`):
    - `utils/flowLabels.test.ts`: tên mới ("Bước 4 — Hình ảnh", "Bước 14 — Đăng video"); `phaseOf`, `AUTO_STEPS`.
    - `components/PhaseStepper.test.tsx` (mới): ô hiện tại theo `currentStep`, ✓ cho giai đoạn đã xong, hiện ở bước 1 khi chưa có project, bỏ bước skipped khỏi mẫu số.
    - `components/StepRail.test.tsx`: nhãn "Tự động" ở 7, 9–12, không có ở bước khác.
    - `components/AppShell.test.tsx`: có `PhaseStepper`; nhãn nút Tiếp tục mới.
    - `pages/ScriptStepPage.test.tsx`: truyền `similarProjects` qua state.
    - `pages/ScriptAuthoringSettingsStepPage.test.tsx` (mới): 3 nhóm theo thứ tự; chủ đề chỉ đọc + Sửa; banner trùng; đổi kiểu video gọi `createProjectDraft` với chủ đề có "kiểu: X"; lỗi lưu hiện ở thanh dưới.
    - `pages/ScriptOutlineStepPage.test.tsx`: không còn ô chủ đề/kiểu video; thanh AI ở trên thẻ; lời nhắc đúng theo chế độ.
    - `pages/VisualDirectorStepPage.test.tsx`, `pages/ManimEngineerStepPage.test.tsx`: bỏ case `scriptSource`; lời nhắc theo chế độ.
    - `pages/ValidatePage.test.tsx`: phần 1 không có bảng cài đặt; "Tiếp" đổi URL sang `?part=settings`; phần 2 có bảng + nút duyệt gọi approve; reject điều hướng tới `/projects/:id/resume?step=6`.
    - `pages/RenderPage.test.tsx`: tracker có cả 4 việc với video có short, 3 việc với video dài.
    - `context/ProjectDraftContext.test.tsx`: bỏ case `SET_SCRIPT_SOURCE` nếu có.
    - Xoá `components/ScriptAssistant.test.tsx` (và `OutlineActions` test nếu có).
14. Chạy `npx tsc --noEmit`, `npx eslint src tests`, `npx vitest run` trong `services/web-gui`.
15. `docker compose build web-gui && docker compose up -d web-gui`.

### Bổ sung theo bản mẫu đã chốt (FR3a–FR3d)

16. Xoá `PhaseStepper` (+ CSS, test); chuyển test nhãn nút "Tiếp tục" sang `AppShell`/`StepRail` test.
17. `AppShell`: dòng "<GIAI ĐOẠN> · BƯỚC n/14" trên `h1` khi có `currentStep`; menu bước hiện khi có `currentStep` (bỏ điều kiện `hasProject`); prop `preview` truyền xuống `useStepNav`.
18. `utils/flow.ts`: `FLOW_STEP_PURPOSE` (một câu cho mỗi bước), `skippedReason(step)`, `previewRoute(step, projectId)`.
19. `useStepNav(currentStep, { preview })`: `isClickable` = mọi bước khác bước đang xem; `go` mở màn thật nếu bước đã tới (luật cũ), ngược lại mở `previewRoute`. Ở màn xem trước, "đã tới" tính theo server (`flow.flowStep`), không theo bước đang xem.
20. `StepRail`: đầu menu có tiến độ; giai đoạn thu gọn được (state cục bộ, mở theo bước đang xem).
21. `StepPreviewPage` (mới) + route `/projects/:id/preview/:step` và `/create/preview/:step`.
22. Test: `StepRail` (hiện ở bước 1, đếm và thu gọn giai đoạn, bấm bước chưa tới mở xem trước), `StepPreviewPage`, `useStepNav`, `AppShell` (dòng giai đoạn, nhãn Tiếp tục).

## Kiểm tra

- Tự động: vitest toàn bộ web-gui, tsc, eslint (so số cảnh báo với `main`: 20).
- Rebuild: chỉ `web-gui`.
- Kiểm trực tiếp trên trình duyệt:
  1. "Tạo video mới": bước 1 có thanh 5 giai đoạn, ô "Chuẩn bị" nổi bật.
  2. Bước 2: ba nhóm theo thứ tự; chọn kiểu video → gợi ý format → "Dùng format này" đổi format; chủ đề ở đầu, "Sửa" về bước 1.
  3. Bước 3–6: thanh AI luôn ở trên, không còn ô chủ đề ở bước 3; lời nhắc đúng chế độ.
  4. Chạy tới màn Duyệt: phần 1 chỉ lời thoại; "Tiếp" sang phần 2; Back trình duyệt về phần 1; "Duyệt và bắt đầu" chạy sản xuất. Thử "Quay lại sửa script" → mở bước Code với code cũ.
  5. Màn Sản xuất: thấy cả 4 việc (hoặc 3 với video dài); menu bước có nhãn "Tự động".
  6. Danh sách video, Nhật ký: tên bước mới.

## Rủi ro

- **Không mất dữ liệu**: không đổi DB/API. Bản nháp trong localStorage còn field `scriptSource` cũ: reducer bỏ qua, cần test ở bước 9.
- **Thêm một lần bấm khi duyệt** (FR9). Đổi lại, cài đặt xuất video được nhìn rõ trước khi tốn tiền.
- **`WizardNav` khoá nút ở trạng thái `awaiting_review`** (bước 10 của kế hoạch): nếu cần prop mới, phải bảo đảm các màn soạn 1–6 vẫn bị khoá như cũ khi dự án đã khoá.
- **Tên bước mới** hiện ở cả danh sách video và nhật ký; ai quen tên cũ ("TTS", "Merge") sẽ thấy khác.
- **Chủ đề không còn sửa ở bước 3**: sửa chủ đề phải về bước 1 (một lần bấm "Sửa"). Chuỗi AI vẫn lưu chủ đề trước khi chạy (`beforeRun`).
- Nhiều test UI phải viết lại; rủi ro sót hành vi cũ được che bởi test cũ. Bước code phải chạy đủ vitest, không bỏ qua test nào.
