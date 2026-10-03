# CR-067 — Thiết kế: làm video giống hai video mẫu (nội dung và chuyển động)

Trạng thái: **thiết kế đầy đủ, chờ Creator duyệt** (đã có trả lời Q1–Q6). Gộp toàn bộ CR-066; CR-064 bị bỏ.

## Yêu cầu gốc (nguyên văn)

> xem lại 2 file video và 2 file txt dược dịch ngược từ 2 video đó về định dạng base64. xem giúp tôi nếu muốn làm video y như vậy về các nội dung và hình ảnh chuyển động. ta cần phải chỉnh sửa workflow hiện tại như thế nào?

> 2 file trong thư mục example nhé

Trả lời câu hỏi vòng 1:

> Q1 => chọn a
> Q2 => a
> Q3 => bỏ cr 66 làm trong cr này , gộp vào CR này, tôi muốn tham khảo thôi chứ không phải tham chiếu
> Q4 => bỏ cr 64 làm theo quyết định của cr này
> q5 => a
> q6 => làm hết tron cr 067

Chốt:

- **Q1 (a)**: hướng tới cùng nhịp và ngôn ngữ chuyển động của mẫu; hình vẫn do AI viết SVG, không thêm model sinh ảnh hay tài sản người vẽ.
- **Q2 (a)**: mốc thời gian lấy bằng cách TTS đọc **từng câu thoại thành một file**; mốc của câu = tổng độ dài các câu trước.
- **Q3 (c)**: CR-066 (luật style hình vào DB và viết lại) gộp vào CR-067, CR-066 bỏ. Mẫu chỉ để **tham khảo**: prompt không nhắc tên kênh nào, chỉ mô tả lối vẽ (giữ đúng tinh thần "bỏ Vẽ Chuyện" của CR-066).
- **Q4**: bỏ CR-064 (rút gọn prompt Biên kịch); prompt Biên kịch đổi theo CR-067 (chỉ thêm luật câu ngắn, xem FR5).
- **Q5 (a)**: short dọc bật phụ đề cụm ngắn, mỗi câu thoại một cụm.
- **Q6 (b)**: làm cả ba đợt trong CR-067.

## Tư liệu đã xem

| File trong `example/` | Thực chất |
|---|---|
| `Tá có tác dụng gì- #fact #science #vechuyen.mp4` | Short dọc 1080×1920, 58,8 s, có tiếng, AV1 60 fps |
| `Tá có tác dụng gì- … .b64.txt` | Base64 của đúng file mp4 trên (giải mã ra trùng từng byte) |
| `Bi kịch bên trong quả sung.mp4` | Video dài 1280×720, 9 phút 05 s, có tiếng |
| `Tại sao lại gọi là bệnh tiểu đường-.txt` | Base64 của **`Bi kịch bên trong quả sung.mp4`** (trùng từng byte) — tên file bị đặt nhầm |

So sánh với hai video mới nhất hệ thống render: `9400f01e…` (2026-10-03, đã có CR-060/CR-065) và `257acc58…` (2026-10-02).

## Hiện trạng: khoảng cách đo được

### Số đo chuyển động (khung 96×54 xám, lấy mẫu 4 khung/giây, "đứng yên" = chênh lệch trung bình < 0,5)

| | Tá (mẫu short) | Quả sung (mẫu dài) | 9400f01e (mình) | 257acc58 (mình) |
|---|---|---|---|---|
| Độ thay đổi khung trung vị | 1,11 | **3,39** | **0,76** | 1,03 |
| Tổng thời gian đứng yên ≥ 2 s | 7 s / 59 s | 34 s / 545 s (6%) | 66 s / 361 s (18%) | 71 s / 368 s (19%) |
| Đoạn đứng yên dài nhất | 2,5 s | 5,0 s | **13,5 s** | 11,2 s |
| Số lần cắt cảnh (ngưỡng 0,3) | 10 / 59 s | 47 / 545 s | 22 / 361 s | 49 / 368 s |

Video của mình chuyển động ít hơn mẫu dài khoảng 4 lần, đứng yên lâu gấp 3 lần. Luật L15 ("không quá 2 s đứng yên") có trong prompt nhưng không ai đo trên video thật.

### So sánh theo từng mặt

| Mặt | Mẫu | Hiện tại (từ code và dữ liệu) |
|---|---|---|
| **Đơn vị thay đổi hình** | Mỗi câu/vế thoại (2–4 s) là một "nhịp" có thay đổi hình rõ: hộp bút mở → rút bút → tách đôi → nhân thành lưới; 10 → 12 → 2×3×2; đồng hồ → chia múi màu. Hình **bám lời** | Shot là đơn vị nhỏ nhất. Storyboard `Shot` chỉ có `camera`, `visual`, `narration` (`services/llm-service/app/storyboard.py:48-53`). Project 9400f01e: 35 shot, trung bình 29,5 từ và 2,8 câu thoại mỗi shot (~10 s), ~3,4 lệnh `interpolate` mỗi shot, 0 `Sequence`. Hoạt ảnh tính theo tỉ lệ `duration` của cả shot (`<Shot duration={segment.durationInFrames}/>`, `merger.py:127-135`) nên không thể khớp từng câu |
| **Mốc thời gian lời đọc** | Hình đổi đúng lúc từ khoá được đọc | TTS chỉ trả **thời lượng cả đoạn** (`services/tts/domain/ports.py:17` trả `float`); rendering chỉ biết `duration_seconds` từng đoạn (`services/rendering/domain/models.py:16-29`). Không có mốc câu/từ |
| **Màu** | Mỗi cảnh một nền bão hoà sáng (vàng, xanh cyan, hồng, tím) cho cảnh giải thích; môi trường đầy màu cho cảnh kể chuyện | Prompt Đạo diễn vẫn giữ nền mặc định tối `#080E1C` (`authoring-service/internal/domain/prompt_template_seeds_ai.go:57`). 9400f01e tự chọn nền hang `#140E0C`, `#1E2230` → cả video tối |
| **Nhân vật, bàn tay** | Nhân vật to, biểu cảm, đang làm việc gì đó; rất nhiều cận cảnh **bàn tay người** tương tác với vật (rút bút, bấm đốt ngón tay, cầm kính lúp) | Người dùng `Person` nhỏ, đứng giữa khung, ít tư thế; tay là một hình `Hand` với vài `pose` |
| **Chuyển cảnh** | Biến hình/khớp hình liên tục (bút → lưới hộp bút, quả sung → lòng quả, zoom xuyên) | Đã có trong prompt (`transition_in`, `prompt_template_seeds_ai.go:36,114`) nhưng chỉ ở ranh giới **cảnh**; trong cảnh là cắt shot |
| **Chữ trên hình** | Dài: không phụ đề. Short Tá: **phụ đề cụm ngắn** (1–2 dòng, ~6–10 từ, phía trên) đổi theo nhịp; tiêu đề chữ động lúc mở | 9400f01e đốt phụ đề **cả đoạn 2–3 câu** ở đáy khung. CR-060 chốt short mặc định tắt phụ đề — mẫu short mới (Tá) lại có phụ đề cụm ngắn |
| **Độ tinh xảo hình** | Tranh vector do hoạ sĩ/animator làm tay | Hình do model viết SVG (thư viện 265 hình + 18 nền). Đây là giới hạn đã ghi ở CR-060 |
| **Khung, bố cục cảnh, nền môi trường** | | Đã có từ CR-060: `Scene` 4 lớp, `Camera`, `Glow`, 18 nền, short dọc riêng. 9400f01e dùng `<Scene` 33/35 shot — phần này **không còn là chỗ thiếu chính** |

### Kết luận phân tích

CR-060 đã giải quyết **khung cảnh** (nhiều lớp, nền, máy quay, short dọc). Thứ còn thiếu để "chuyển động y như vậy" nằm ở **cấu trúc thời gian** của workflow: mẫu thay đổi hình theo từng câu thoại, còn workflow hiện tại chỉ biết shot ~10 s và không có mốc câu từ TTS. Đây là thay đổi workflow, không chỉnh prompt là đủ. Ba chỗ thiếu tiếp theo: nền tối mặc định, nhân vật/bàn tay nhỏ và ít hành động, phụ đề cả đoạn.

### Đường đi của lời thoại hôm nay (đã đọc code)

1. Bước Code: merger ghi `export const narrations: string[]` — **một dòng cho mỗi shot** (`services/llm-service/app/pipeline/merger.py:299-300`), kiểu `ShotProps = {duration: number}` (`merger.py:301`), `CreatorComposition` gọi `<Shot duration={segment.durationInFrames}/>` (`merger.py:127-135`).
2. Rendering chạy thử script, đọc `narrations` (`services/rendering/adapters/rendering/remotion_renderer.py:347`, `_extract_narrations`) → orchestrator lưu thành `project.Scenes`.
3. Orchestrator gửi `synthesize_speech` với mỗi dòng là một `scene_index`; TTS trả `audio_path`, `duration_seconds` cho từng dòng (`aidlc-docs/construction/tts-service/low-level-design/interface-contracts.md`).
4. Rendering dựng `segments` mỗi dòng một đoạn, cách nhau `INTER_SHOT_GAP_SECONDS = 0.3` (`remotion_renderer.py:67, 354-368`), trả `wait_offsets` mỗi dòng một mốc.
5. Orchestrator đòi `len(wait_offsets) == len(project.Scenes)` (`services/orchestrator/internal/application/handle_step_event.go:815-818`), rồi dựng `narration_segments` và `subtitle_cues` từ chính các mốc đó (`handle_step_event.go:1000-1006, 1056`).
6. Video-assembly đặt từng file tiếng vào mốc của nó và tạo phụ đề mỗi dòng một cue (`services/video-assembly/adapters/assembly/subtitle_file.py`).

Nhận xét quyết định thiết kế: **chỉ cần mỗi câu thoại là một dòng trong `narrations`**, các bước 2–6 tự đọc từng câu thành một file (Q2-a), có mốc từng câu, có phụ đề từng câu (Q5-a) — **không đổi hợp đồng TTS, orchestrator, video-assembly**. Chỉ còn phải cho rendering biết câu nào thuộc shot nào.

Về tên gọi: trong dự án "beat" đã là id cảnh của Biên kịch (`scenes[].id`, `BeatOccurrence` ở orchestrator, `DryRunResult.beats` ở rendering). Đơn vị mới vì vậy gọi là **câu thoại — `lines`** trong code.

## Yêu cầu

### Đợt 1 — Hình đổi theo từng câu thoại

- **FR1. Câu thoại trong shot.** Storyboard (JSON của Đạo diễn AI) cho mỗi shot một mảng `lines`: `[{"say": "<một câu hoặc một vế>", "show": "<thay đổi nhìn thấy được trong lúc câu này được đọc>"}]`, 1–4 câu mỗi shot, mỗi câu ≤ 20 từ. `narration` của shot do llm-service tự ghép từ các `say` (cách nhau một dấu cách) khi chuẩn hoá storyboard; shot không có `lines` (storyboard cũ, hoặc model bỏ trống) được coi là một câu duy nhất = `narration`.
- **FR2. Script khai báo số câu của mỗi shot.** Merger ghi `narrations` = mọi câu của mọi shot theo thứ tự, thêm `export const shotLineCounts: number[]` (số câu của từng shot), `ShotProps = {duration: number; lines: number[]}` (`lines[i]` = frame bắt đầu câu i, tính từ đầu shot; `lines[0] = 0`).
- **FR3. Rendering gom câu thành shot.** Khi script có `shotLineCounts`: tổng phải bằng số dòng thoại (sai → `AnimationEngineError` nêu rõ hai con số); đoạn của shot = các câu của nó nối liền, cách nhau `LINE_GAP_SECONDS = 0.1`, giữa hai shot vẫn `INTER_SHOT_GAP_SECONDS`; mỗi đoạn mang thêm `lines`; `wait_offsets` vẫn **một mốc cho mỗi câu** (khớp `project.Scenes` của orchestrator). Script không có `shotLineCounts` (project cũ, Manim, prompt thủ công) → mỗi dòng một shot như hôm nay.
- **FR4. Kit.** `Segment` có thêm `lines?: number[]`; `evenLines(duration, count)` chia đều khi không có mốc thật (layout probe, Remotion Studio); `lineSpan(lines, i, duration)` trả `[bắt đầu, kết thúc]` của câu i. Composition truyền `lines={segment.lines ?? evenLines(segment.durationInFrames, shotLineCounts[index])}`.
- **FR5. Prompt.**
  - Đạo diễn AI: viết `lines`; mỗi câu một thay đổi hình **cụ thể** (vật vào/ra, biến hình, đếm, tách, nhân lên, tay cầm/rút/chỉ, máy đẩy vào chi tiết); một câu không quá ~4 s đọc; ưu tiên ẩn dụ bằng đồ vật đời thường và bàn tay người như mẫu; tự kiểm thêm "câu nào không có `show`".
  - Kỹ sư AI: mốc thời gian của mỗi thay đổi lấy theo `lines[i]` (không theo tỉ lệ `duration` khi thay đổi gắn với một câu), thay đổi của câu i bắt đầu trong 12 frame đầu của câu; L15 giữ (không đứng yên quá 2 s).
  - Biên kịch: thêm luật **"viết để vẽ"** — câu ngắn (≤ 20 từ), mỗi câu nói một ý có thể vẽ thành một thay đổi hình, ưu tiên con số và đồ vật cụ thể. Không làm phần rút gọn của CR-064.
  - Tin nhắn chunk/layout của llm-service gửi kèm `lines` của shot.
- **FR6. Tương thích.** Project, storyboard, script cũ render ra đúng như hôm nay; Manim và luồng prompt thủ công không đổi hành vi (xem Ngoài phạm vi).

### Đợt 2 — Hình: nền sáng theo cảnh, nhân vật và bàn tay, luật style

- **FR7. Nền sáng theo cảnh.** Bỏ "nền mặc định `#080E1C`" khỏi prompt Đạo diễn (cả thủ công và AI, phần dùng chung) và Kỹ sư: mỗi cảnh **bắt buộc** khai báo một vai trò màu nền; cảnh giải thích ý trừu tượng dùng **một màu nền bão hoà sáng** (gợi ý: vàng `#FFD84D`, xanh cyan `#2EC4F0`, hồng `#FF9EC4`, tím `#B48CFF`, xanh lá `#8BD86B`, cam `#FFB25B`), đổi màu khi đổi ý; cảnh kể chuyện dùng nền môi trường (CR-060). Phim tối chỉ khi chủ đề đòi (đêm, hang, vũ trụ) và phải ghi lý do trong `setting`. Mỗi shot đặt nền (`Scene`/`Backdrop`) — không dựa vào nền mặc định của `Stage`.
- **FR8. Nhân vật diễn được.** `Person` có thêm: `toPose` + `poseT` (0→1, nội suy vị trí tay/chân giữa hai tư thế, để nhân vật *chuyển* dáng trong một câu); `framing: 'full' | 'bust'` (cận nửa người: đầu + vai chiếm khung, dùng cho cận cảnh biểu cảm); tư thế mới `hold` (hai tay đưa ra trước, cầm vật — vật đặt qua `children` ở điểm giữa hai bàn tay).
- **FR9. Bàn tay cận cảnh.** Component kit mới `ReachingHand`: một cánh tay đi vào từ mép khung (`edge: 'bottom' | 'left' | 'right' | 'top'`), tới điểm `target {x, y}` theo `reach` (0 = ngoài khung, 1 = chạm điểm); `pose`: `open | point | grip | pinch | count`; `count` 1–5 (số ngón giơ lên); `sleeve` (màu tay áo), `skin`; `children` được vẽ trong lòng bàn tay/giữa hai ngón (vật đang cầm). Dùng được cho cận cảnh rút bút, cầm kính lúp, bấm đốt ngón tay như mẫu.
- **FR10. Luật style hình minh hoạ vào DB và viết lại (gộp CR-066, FR-1…FR-5 của CR-066):**
  - vai trò prompt mới `illustration_style`, dòng hệ thống seed từ `internal/domain/prompts/illustration_style_vi.txt`, Creator xem/nhân bản/sửa/bật trên màn Prompt;
  - AI vẽ (hình và nền) và `GET /v1/illustration-style` đọc dòng **active** trong DB; đọc DB lỗi → lượt vẽ báo lỗi;
  - `illustration_style` không render cho AI ngoài (`/v1/prompt-renders`, render theo project trả 400);
  - văn bản mới = **Phụ lục A** (văn bản CR-066, không nhắc tên kênh nào) cộng hai chỉnh sửa của CR-067 ghi ở đầu Phụ lục A.
- **FR11. Kit trong prompt.** Phần mô tả kit trong prompt Kỹ sư (`withIllustrationKit`) có `Person` mới (FR8), `ReachingHand` (FR9) và một ví dụ ngắn dùng `lines[i]`.

### Đợt 3 — Phụ đề short, đo chuyển động

- **FR12. Phụ đề short theo câu.** Chọn đầu ra short → phụ đề mặc định `burn_in`, vị trí `top` (Creator vẫn đổi được). Trên khung dọc, cụm phụ đề ở trên nằm **dưới vùng giao diện Shorts phía trên (200 px)**: video-assembly dùng `MarginV` 260 px cho vị trí `top` khi video dọc; dải phụ đề báo cho Kỹ sư và bộ kiểm bố cục tương ứng là từ mép trên tới `200 + dải thường`. Mỗi câu thoại là một cue (tự có nhờ FR2–FR3). Video dài giữ mặc định `track`.
- **FR13. Đo chuyển động sau render.** Video-assembly đo trên `final.mp4` (khung xám 96×54, 4 khung/giây, "đứng yên" = chênh lệch trung bình < 0,5): mỗi đoạn đứng yên ≥ `QC_MAX_STILL_PICTURE_SECONDS` (mặc định 2,5) thành một phát hiện `still_picture` mức **cảnh báo** kèm mốc giây; báo cáo QC thêm tổng giây đứng yên và độ thay đổi trung vị. Không chặn xuất bản.
- **FR14. Giao diện.** Màn Prompt có mục "Thư viện hình — luật style hình minh hoạ (AI vẽ)"; bộ chọn đầu ra short hiện phụ đề mặc định mới; báo cáo QC hiện tên tiếng Việt của `still_picture` ("Hình đứng yên quá lâu").

### Tiêu chí chấp nhận

- Script mới có `shotLineCounts`; rendering trả số `wait_offsets` = số câu; orchestrator đi hết saga tới `final.mp4`; tiếng của từng câu bắt đầu đúng frame `lines[i]` của shot (kiểm bằng test đơn vị trên `_segments_from` và một render gallery).
- Script cũ (không `shotLineCounts`) và Manim: test hiện có vẫn pass, kết quả `segments`/`wait_offsets` y như cũ.
- Prompt Đạo diễn/Kỹ sư không còn "nền mặc định #080E1C"; golden test cập nhật.
- `Person` (`toPose`/`poseT`, `framing="bust"`, `hold`) và `ReachingHand` render trong gallery ở 1920×1080 và 1080×1920.
- Bảng `prompts` có `system-illustration_style` (active nếu Creator chưa có dòng active); AI vẽ đọc DB; văn bản không có "Vẽ Chuyện".
- Short mới: phụ đề `burn_in`/`top`, cue mỗi câu, nằm dưới 200 px trên cùng.
- QC của một video có đoạn đứng yên 13 s báo `still_picture` tại đúng mốc.
- **Mục tiêu đo (không phải test, vì model không tất định)**: một video AI mới làm theo luồng mới có tổng đứng yên ≥ 2 s ≤ 8% thời lượng, đoạn đứng yên dài nhất ≤ 4 s, trung bình mỗi câu thoại ≤ 4 s.

### Ngoài phạm vi

- Độ tinh xảo tranh vẽ tay của mẫu; model sinh ảnh; tài sản do người vẽ (Q1-a).
- Luồng prompt **thủ công** của Kỹ sư (Creator dán sang AI ngoài và dán code về): không viết `shotLineCounts` → mỗi dòng một shot như hôm nay. Chỉ phần dùng chung về nền (FR7) đổi cho cả hai luồng.
- Engine Manim.
- Phần rút gọn prompt Biên kịch của CR-064 (Creator bỏ CR-064).
- Vẽ lại hình đã có trong thư viện theo luật style mới.

## Giải pháp đề xuất

### 1. Câu thoại là một dòng `narrations` — vì sao chọn cách này

| Cách | Ưu | Nhược |
|---|---|---|
| **A (chọn).** Mỗi câu là một dòng `narrations`; script thêm `shotLineCounts`; rendering gom câu thành shot | TTS, orchestrator, video-assembly, hợp đồng RabbitMQ **không đổi**; Q2-a (mỗi câu một file) và Q5-a (mỗi câu một cue) có sẵn; mốc tuyệt đối chính xác | Nhiều lượt TTS hơn (~3× số dòng); số "scene" trong orchestrator giờ là số câu (chỉ là tên) |
| B. TTS trả mốc từng câu trong một file | Một file tiếng mỗi shot | Đổi hợp đồng TTS + orchestrator + rendering; TTS phải tự tách câu và đo — là Q2-b |

Chèn `LINE_GAP_SECONDS = 0.1` giữa các câu trong một shot để giọng không dính câu (file TTS đã có chút lặng đầu/cuối); giữa shot giữ 0,3 s.

### 2. Tương thích ngược ở mọi tầng

- Storyboard: `lines` tuỳ chọn; thiếu thì một câu = `narration`. authoring-service (Go) chỉ đọc `narration` (`internal/domain/storyboard_scenes.go:13-17` — `CheckNarrationBudgets`, `CheckIllustratedNarration`) nên **không phải đổi**: llm-service luôn ghi `narration` = ghép các `say`.
- Script: không có `shotLineCounts` → hành vi cũ. Layout probe (`src/layout-probe/harness.tsx:354-360`) truyền `segments` không có `lines` → composition chia đều bằng `evenLines`.
- Màn duyệt dàn ý (đọc `narrations` từ chạy thử) sẽ hiện mỗi câu một dòng — đúng với cái sẽ được đọc.

### 3. Nền sáng

Màu nền là lựa chọn của Đạo diễn; hôm nay prompt nói "không ghi gì thì nền #080E1C" và model hay chọn tối (9400f01e: `#140E0C`, `#1E2230`). Đổi thành "mỗi cảnh bắt buộc có vai trò nền, mặc định là một màu sáng bão hoà; tối phải có lý do". `BACKGROUND` của kit vẫn `#080E1C` (Manim theme và QC tương phản Manim dùng chung) nhưng Kỹ sư phải đặt nền cho mọi shot.

### 4. Nhân vật và bàn tay — dựng trong kit, không phải hình thư viện

Giống `Scene`/`Camera` ở CR-060: là phần cấp khung (khớp, tư thế nội suy được), cần code ổn định và được kiểm kỹ, không để AI vẽ lại mỗi lần. `Person` hiện chỉ có tư thế rời (`illustration.tsx:372-430`), tay là các điểm `Pt` nên nội suy giữa hai tư thế là lerp từng điểm. `ReachingHand` là module mới `conceptflow-mini/rig.tsx`.

### 5. Luật style (từ CR-066)

Dùng nguyên cách 1 của CR-066: vai trò mới trong thư viện prompt hiện có, không migration. Văn bản = Phụ lục A, thêm hai chỉnh sửa để khớp hướng của CR-067 (ghi ở đầu Phụ lục A).

### 6. Phụ đề short

Hệ thống dải phụ đề đã hỗ trợ vị trí `top` (`authoring-service/internal/domain/narration.go:177-196`, `SubtitleBandFor`), video-assembly đã lấy độ phân giải thật của video (`ffmpeg_assembler.py:429`). Chỉ thêm: mặc định cho short, và độ lệch 200 px ở mép trên khi khung dọc (vùng giao diện Shorts, cùng số với vùng an toàn dọc của CR-060).

### 7. Đo chuyển động

QC hiện có `static_frame` nhưng chỉ đo khoảng trống giữa hai mốc thoại (`video-assembly/domain/qc_rules.py:564-588`), không nhìn hình. Thêm phép đo thật trên pixel (cùng phép đo dùng cho bảng số liệu ở trên) bằng ffmpeg đã có trong container, như `measure_loudness` (`adapters/qc/ffmpeg_probe.py:40`).

## Phụ thuộc

| Việc đang mở | Kết quả | Quyết định của Creator |
|---|---|---|
| `feature/cr-066-illustration-style-prompt-db` | Trùng phạm vi (luật style hình) | **Gộp vào CR-067, bỏ CR-066** (Q3). Lấy thiết kế + Phụ lục A của CR-066; nhánh/worktree CR-066 không dùng nữa — chờ Creator cho phép xoá |
| `feature/cr-064-screenwriter-prompt-trim` | Trùng phạm vi (prompt Biên kịch, `prompt_template_seeds.go`) | **Bỏ CR-064** (Q4). CR-067 làm từ `main` hiện tại; nhánh/worktree CR-064 không dùng nữa — chờ Creator cho phép xoá |
| `fix/run-segment-reruns-all`, `fix/video-list-paging` | Chưa có commit; theo tên thuộc web-gui (chạy lại đoạn, phân trang danh sách video) | Độc lập |
| `chore/worktree-per-cr` | Không có thay đổi | Độc lập |

## Phạm vi

| Service | File / symbol | Thay đổi |
|---|---|---|
| llm-service | `app/storyboard.py` `Shot` (+ `Line`), chuẩn hoá storyboard, `storyboard.py:214` (bảng tóm tắt shot) | `lines`, ghép `narration`, hiện câu trong tóm tắt |
| | `app/pipeline/merger.py` (`narrations`, `ShotProps`, `CreatorComposition`, stub shot `merger.py:182`) | `shotLineCounts`, `lines` |
| | `app/pipeline/prompts.py` `_shot_json` | gửi `lines` |
| rendering | `adapters/rendering/remotion_renderer.py` `_segments_from`, `render`, hằng mới, đọc `shotLineCounts` | gom câu thành shot |
| | `remotion_project/src/conceptflow-mini/segments.tsx` | `lines`, `evenLines`, `lineSpan` |
| | `remotion_project/src/conceptflow-mini/illustration.tsx` `Person` | `toPose`/`poseT`, `framing`, `hold` |
| | `remotion_project/src/conceptflow-mini/rig.tsx` (mới), gallery `scene-demo.tsx` | `ReachingHand`, demo |
| | `domain/illustration_style.py` | chỉ docstring (nguồn luật nằm trong DB) |
| authoring-service | `internal/domain/prompt_template_seeds.go`, `prompt_template_seeds_ai.go` | Biên kịch v10→11, Đạo diễn v11→12, Kỹ sư v7→8, Đạo diễn AI v5→6, Kỹ sư AI v5→6; thêm seed `illustration_style` |
| | `internal/domain/prompt_template.go` | `RoleIllustrationStyle`, `IsRenderablePromptRole` |
| | `internal/domain/prompts/illustration_style_vi.txt` | Phụ lục A |
| | `internal/application/illustrations.go`, `illustration_drawer.go`, `adapters/http/illustrations.go`, `router.go`, `render_prompt_input.go`, `cmd/authoring/main.go` | đọc luật từ DB, chặn render vai trò |
| | `internal/domain/narration.go` `SubtitleBandFor`/`SubtitleZoneIn` | độ lệch 200 px trên khung dọc |
| | phần kit trong prompt (`withIllustrationKit`) | `Person` mới, `ReachingHand`, ví dụ `lines` |
| orchestrator | `internal/domain/wizard.go` `ApplyShortDefaults` | short → `burn_in`, style `top` |
| video-assembly | `adapters/assembly/subtitle_file.py` | `MarginV` 260 px cho `top` khi video dọc |
| | `adapters/qc/ffmpeg_probe.py`, `domain/qc_rules.py`, `adapters/messaging/consumer.py` | `measure_still_runs`, `check_still_picture`, nối vào `evaluate_all` |
| web-gui | `src/api/client.ts`, `src/pages/PromptSettingsPage.tsx`, bộ chọn đầu ra short, nhãn QC | FR14 |
| Tài liệu | ADR-0032 (câu thoại là đơn vị thời gian của Remotion), `docs/contracts/authoring-llm-code-v2.md` (khung script có `shotLineCounts`, `ShotProps.lines`) | |

- **Hợp đồng**: RabbitMQ, TTS, orchestrator ↔ rendering không đổi dạng (`wait_offsets` vẫn một mốc mỗi dòng thoại). Thêm trường tuỳ chọn trong storyboard JSON và khung script.
- **DB**: không migration; dòng prompt mới do `SeedPrompts` tạo khi khởi động.
- **`graphify affected`**: `StoryboardShot` ← `CheckNarrationBudgets`, `CheckIllustratedNarration`, `CheckSceneSettings` (không đổi vì `narration` vẫn có); `Segments()` ← `CreatorComposition` do merger sinh, layout probe harness; `IllustrationStyleGuide` ← `drawerSystem`, `handleIllustrationStyle`; `ValidPromptRole` ← router (3 route), `PromptsUseCase`, `RenderPromptUseCase.Render`.

## Kế hoạch thực hiện

### Đợt 1 — câu thoại

1. **llm-service `app/storyboard.py`**: thêm `class Line(BaseModel)` (`say: str`, `show: str`, cả hai không rỗng); `Shot.lines: list[Line] = []`; `Shot.narration` thành tuỳ chọn khi có `lines`; `model_validator(mode="after")`: có `lines` → `narration = " ".join(l.say.strip() for l in lines)`; không có → `lines = []` giữ nguyên, `narration` bắt buộc như cũ. Hàm `Shot.spoken_lines() -> list[str]` trả `[l.say ...]` hoặc `[narration]`. Kiểm: tối đa 4 câu/shot và mỗi `say` ≤ 30 từ (vượt → lỗi xác thực để model viết lại, cùng cơ chế lỗi hiện có). `storyboard.py:214`: tóm tắt shot in các câu dạng `THOẠI: "câu 1" / "câu 2"`. JSON chuẩn hoá ghi cả `lines` và `narration`.
2. **`app/pipeline/merger.py`**: `narrations` = mọi `spoken_lines()` nối theo thứ tự shot; thêm `export const shotLineCounts: number[] = [...]`; `type ShotProps = {duration: number; lines: number[]}`; stub shot (`merger.py:182`) nhận `{duration, lines}`; `CreatorComposition` kiểu segment thêm `lines?: number[]` và gọi `<Shot duration={segment.durationInFrames} lines={segment.lines ?? evenLines(segment.durationInFrames, shotLineCounts[index] ?? 1)} />`; import `evenLines` từ `./conceptflow-mini/segments`; thêm `evenLines`, `lineSpan` vào danh sách tên có sẵn (`available_names_text`).
3. **`app/pipeline/prompts.py` `_shot_json`**: thêm `"lines": [{"say", "show"}]` khi shot có `lines`.
4. **rendering `segments.tsx`**: `Segment.lines?: number[]`; `normalizeSegments` giữ `lines` (làm tròn, kẹp trong `[0, durationInFrames-1]`, tăng dần); export `evenLines(duration, count)` (`i * duration / count`, làm tròn, `count < 1` → `[0]`) và `lineSpan(lines, i, duration)` (`[lines[i], lines[i+1] ?? duration]`).
5. **rendering `remotion_renderer.py`**: `LINE_GAP_SECONDS = 0.1`; `_extract_shot_line_counts(script) -> list[int] | None` (regex như `_extract_narrations`, số nguyên ≥ 1); `_segments_from(narration_segments, shot_line_counts=None)` trả `(segments, wait_offsets)`: không có counts → như cũ; có → kiểm `sum == len(narration_segments)` (sai → `AnimationEngineError` "shotLineCounts cộng lại X nhưng có Y dòng thoại"), gom câu, `lines` tương đối, `wait_offsets` từng câu. `render()` dùng `wait_offsets` từ hàm này thay vì `startFrame` của đoạn.
6. **authoring-service prompt (đợt 1)**:
   - `visualDirectorOutputJSONVI` (`prompt_template_seeds_ai.go:19-63`): trong shot thay `"narration"` bằng `"lines": [{"say": "<một câu hoặc một vế, ≤ 20 từ>", "show": "<thay đổi nhìn thấy được trong lúc câu này được đọc>"}]`; quy tắc: 1–4 câu mỗi shot, mỗi câu một thay đổi hình cụ thể, không câu nào `show` kiểu "vẫn hiển thị"; câu không chứa xuống dòng.
   - `visualDirectorTailAIVI`: mục 2, 3, 12 đổi theo câu (mỗi câu ≤ ~4 s và có `show`; gộp câu vào chung shot khi cùng một khung, tách shot khi đổi khung); thêm mục: ẩn dụ bằng đồ vật đời thường và bàn tay cận cảnh.
   - Kỹ sư AI (`prompt_template_seeds_ai.go:130-175`): khuôn hàm `function ShotN_M({duration, lines}: ShotProps)`; luật 2: thay đổi gắn với câu i lấy mốc `lines[i]` (ví dụ `interpolate(frame, [lines[1], lines[1] + 15], [0, 1], clamp)`), chuyển động nền/máy vẫn theo `duration`; thay đổi của câu i bắt đầu trong 12 frame đầu câu; ví dụ `Shot1_2` cập nhật; mục tự kiểm 4 thêm "mỗi câu có thay đổi riêng".
   - Biên kịch (`storyArchitectVI`, `prompt_template_seeds.go`): thêm mục "VIẾT ĐỂ VẼ" (câu ≤ 20 từ, mỗi câu một ý vẽ được, ưu tiên số và đồ vật cụ thể, tránh câu ghép nhiều mệnh đề). Không đổi phần khác, OUTPUT giữ nguyên.
   - Version: Biên kịch 10→11, Đạo diễn AI 5→6, Kỹ sư AI 5→6.
7. **Test đợt 1**: llm-service — `Line`/`Shot` (có/không `lines`, ghép `narration`, quá 4 câu, câu rỗng), merger (`shotLineCounts`, `narrations` phẳng, stub, composition truyền `lines`), `_shot_json`. rendering — `_extract_shot_line_counts`, `_segments_from` (không counts = như cũ; có counts: offsets, `lines`, khoảng 0,1 s và 0,3 s; tổng sai → lỗi), `segments.tsx` (`evenLines`, `lineSpan`, normalize) qua test TS hiện có hoặc `tsc`. authoring-service — golden test: câu bắt buộc mới (`lines`, `shotLineCounts`/`lines[i]`, "VIẾT ĐỂ VẼ"), hash prompt thủ công đổi theo bước 9.

### Đợt 2 — hình

8. **Kit `illustration.tsx` `Person`**: `toPose?: PersonPose`, `poseT?: number` (kẹp 0–1, lerp từng điểm tay/chân giữa `arms[pose]` và `arms[toPose]`, và độ cao khi `sit`); `framing?: 'full' | 'bust'` (`bust`: hộp vẽ cắt ở ngang ngực, `vh` nhỏ hơn để đầu+vai lấp hộp; cập nhật mô tả hộp trong docstring); `pose: 'hold'` + `children` vẽ tại trung điểm hai bàn tay. Giữ nguyên mặc định để code cũ ra y hệt.
9. **Kit `rig.tsx` (mới)**: `ReachingHand` như FR9 — cánh tay là đường thuôn (hai cạnh song song bo tròn) từ điểm ngoài mép khung tới cổ tay, cổ tay = lerp(điểm ngoài khung, `target`, Easing.out(`reach`)); bàn tay vẽ trong hộp 200×220 xoay theo hướng cánh tay; ngón tay theo `pose`/`count`; `children` đặt ở `palmAnchor`; tuân luật style (không viền khối, bóng phải-dưới, `useSvgId` nếu có gradient); export từ module được merger import (thêm vào `REMOTION_API`/`PRIMITIVES` hoặc dòng import kit tương ứng trong `merger.py:101-118`) và vào danh sách tên có sẵn. Gallery `scene-demo.tsx`: thêm 2 shot (Person `toPose` + `bust`; `ReachingHand` rút một cây bút), `render_gallery.mjs` không đổi.
10. **Nền sáng (FR7)**: `prompt_template_seeds.go:335` (phần dùng chung "Bối cảnh và nền") viết lại: mỗi cảnh BẮT BUỘC có vai trò nền; cảnh ý trừu tượng một màu sáng bão hoà (bảng gợi ý FR7), đổi màu khi đổi ý; phim tối chỉ khi chủ đề đòi và ghi lý do trong bối cảnh; bỏ câu "Không ghi gì thì nền mặc định là #080E1C". `prompt_template_seeds.go:397` và `prompt_template_seeds_ai.go:57`: bỏ "nền mặc định #080E1C không khai báo lại", thay "mỗi cảnh có đúng một vai trò nền". `prompt_template_seeds.go:638`: "đọc rõ trên nền của cảnh" thay cho `#080E1C`. `prompt_template_seeds.go:644`: bỏ "Stage đã tô nền mặc định", thêm "mọi shot đặt nền bằng `Scene`/`Backdrop` từ PALETTE". Version Đạo diễn 11→12, Kỹ sư 7→8 (AI đã tăng ở bước 6).
11. **Kit trong prompt (FR11)**: phần `withIllustrationKit` thêm `Person` (`toPose`, `poseT`, `framing`, `hold`), `ReachingHand` (props, cỡ hộp, ví dụ một dòng).
12. **Luật style vào DB (gộp CR-066, theo kế hoạch 1–11 của CR-066)**:
    - `internal/domain/prompts/illustration_style_vi.txt` = văn bản Phụ lục A (đúng từng chữ sau khi Creator duyệt).
    - `prompt_template.go`: `RoleIllustrationStyle PromptRole = "illustration_style"`, thêm vào `ValidPromptRole`, `IsRenderablePromptRole(role)` (mọi vai trò hợp lệ trừ `illustration_style`).
    - `prompt_template_seeds.go`: dòng seed `{Role: RoleIllustrationStyle, Language: "vi", Version: 1, TemplateText: IllustrationStyleGuide()}` sau `RoleThumbnailDesign`.
    - `application/illustrations.go`: cổng `StylePromptPort { GetActive(ctx, role) (domain.Prompt, error) }`, `WithStylePrompts`, `StyleRules(ctx)` (lỗi nếu chưa gắn cổng hoặc DB lỗi).
    - `illustration_drawer.go` `drawerSystem`: dùng `StyleRules` cho `{{style}}` và `{{figure_style}}`; lỗi → trả lỗi `load illustration style: %w`.
    - `adapters/http/illustrations.go` `handleIllustrationStyle`: `rules` từ `StyleRules`.
    - `router.go` (hai route render) và `render_prompt_input.go`: `IsRenderablePromptRole`, sai → 400 `role is not renderable`.
    - `cmd/authoring/main.go`: `.WithStylePrompts(authoringRepo)`.
    - Docstring `illustration.go`, `rendering/domain/illustration_style.py`.
13. **web-gui (phần luật style)**: `client.ts` thêm `"illustration_style"` vào union vai trò; `PromptSettingsPage.tsx` thêm mục "Thư viện hình — luật style hình minh hoạ (AI vẽ)".
14. **Test đợt 2**: rendering — test TS/tsc cho `Person` (mặc định không đổi DOM so với trước cho các tư thế cũ: so snapshot SVG của `pose="wave"` trước/sau), `ReachingHand` (`reach=0` nằm ngoài khung, `reach=1` cổ tay tại `target`), gallery render cả hai khung. authoring-service — như CR-066 mục 11 (không "Vẽ Chuyện", đủ mã `- [Sn] `, `SystemPrompts()` có `system-illustration_style`, `IsRenderablePromptRole`, `drawerSystem` qua cổng giả cả hai nhánh và nhánh lỗi, `GET /v1/illustration-style`, `POST /v1/prompt-renders` → 400); golden: prompt không còn "nền mặc định", hash thủ công cập nhật. web-gui — test `PromptSettingsPage` nếu có kiểm danh sách.

### Đợt 3 — phụ đề short, đo chuyển động

15. **orchestrator `wizard.go` `ApplyShortDefaults`**: `SubtitleMode` mặc định `burn_in`; nếu patch không có `SubtitleStyle` thì đặt style mặc định với `Position = "top"`. Cập nhật docstring (short mang phụ đề cụm theo câu). Test `wizard_test.go`.
16. **authoring-service `narration.go`**: `SubtitleBandFor(mode, style, frame Frame)`: vị trí `top` trên khung dọc → `Px = band + 200` (vùng giao diện Shorts); sửa mọi nơi gọi (`SubtitleZone`, `SubtitleZoneFor`, `SubtitleZoneIn`, `ProjectSubtitleBand` và nơi gửi dải sang bộ kiểm bố cục). Test cho khung dọc/ngang.
17. **video-assembly `subtitle_file.py`**: `SHORTS_TOP_UI_PX = 200`; khi `position == "top"` và `play_res` dọc (`height > width`) → `MarginV = SHORTS_TOP_UI_PX + MARGIN_VERTICAL` (theo pixel thật của video, không nhân `scale`). Test sinh ASS cho dọc/ngang.
18. **video-assembly đo chuyển động**: `adapters/qc/ffmpeg_probe.py` `measure_still_runs(video_path) -> StillStats(runs: list[tuple[start, seconds]], still_seconds, median_change)` (ffmpeg `fps=4,scale=96:54`, `gray`, rawvideo qua pipe, thuần Python); `domain/qc_rules.py`: `QCThresholds.max_still_picture_seconds` (env `QC_MAX_STILL_PICTURE_SECONDS`, mặc định 2.5), `check_still_picture(stats, thresholds)` → `QCFinding(rule="still_picture", severity=cảnh báo)`; `evaluate_all(..., still_stats=None)`; `consumer.py` gọi đo cạnh `measure_loudness` (đo lỗi → ghi log và bỏ qua luật này, báo rõ trong báo cáo là "chưa đo được", không im lặng). Test: video tổng hợp bằng ffmpeg (`color` 3 s đứng yên + `testsrc` chuyển động) cho `measure_still_runs`, và test thuần cho `check_still_picture`.
19. **web-gui**: bộ chọn đầu ra short (`VideoOutputModePicker` / nơi gửi patch) hiện phụ đề mặc định mới; nhãn tiếng Việt cho `still_picture` ở nơi hiện báo cáo QC.
20. **Tài liệu**: ADR-0032; `docs/contracts/authoring-llm-code-v2.md` (khung script `shotLineCounts`, `ShotProps.lines`, `lines` trong storyboard); cập nhật mục lục `docs/contracts/README.md` nếu cần.
21. Chạy toàn bộ test/lint theo `docs/code-standards-rules.md`; rebuild `llm-service rendering authoring-service orchestrator video-assembly web-gui` bằng `scripts/worktree.sh rebuild ...`, xác nhận healthy.

## Kiểm tra

- **Test**: llm-service `pytest` + ruff; rendering `pytest` (máy và container có node) + `tsc` của Remotion; authoring-service `go test ./...`, `go vet`, `gofmt -l`; orchestrator `go test ./...`; video-assembly `pytest` + ruff; web-gui vitest, `tsc`, eslint.
- **Rebuild**: 6 service trên, kiểm healthy.
- **Kiểm trực tiếp (không tốn token)**:
  - gallery `scene-demo.tsx` có câu thoại: render 16:9 và 9:16, xem `Person`/`ReachingHand`;
  - render lại một script cũ đã lưu (ví dụ project `9400f01e`) qua đường render: `segments`/`wait_offsets` như cũ;
  - dựng tay một script nhỏ có `shotLineCounts` đi qua đường render với TTS thật (Edge, miễn phí): tiếng từng câu khớp `lines[i]`, phụ đề mỗi câu một cue;
  - QC của `9400f01e` (đọc lại `final.mp4`) báo `still_picture` ở đoạn 13,5 s;
  - bảng `prompts` có `system-illustration_style`; `curl /v1/illustration-style` trả văn bản mới;
  - short nháp: phụ đề `burn_in`/`top`, nằm dưới 200 px trên cùng.
- **Cần Creator bấm (tốn token)**: chạy AI trọn luồng một video dài và một short (đề xuất chủ đề "Tá có tác dụng gì" để so trực tiếp với mẫu), đo lại bảng chuyển động so với mục tiêu ở Tiêu chí chấp nhận; vẽ thử một hình và một nền bằng AI với luật style mới.

## Rủi ro

- **Số lượt TTS tăng ~3 lần** (35 shot → ~100 câu): thời gian bước TTS tăng; Azure/Google tính theo ký tự nên chi phí gần như không đổi; Edge miễn phí. Giọng có thể ngắt nhẹ giữa câu (đã chọn ở Q2).
- **Token bước Code tăng** vì mỗi shot có nhiều thay đổi hơn; ngược hướng CR-056/CR-063. Đo trên lần chạy thật đầu tiên; nếu tăng quá 30% thì báo Creator.
- **Model không tuân luật câu** (bỏ `lines`, câu quá dài): có xác thực ở llm-service (tối đa 4 câu, ≤ 30 từ), thiếu `lines` thì quay về một câu — không hỏng render.
- **Đổi prompt dùng chung** (nền) ảnh hưởng cả luồng thủ công; hash golden đổi.
- **Luật style mới** chưa đo trên hình thật; seed ghi đè dòng hệ thống cũ (muốn quay lại lấy từ git). Dòng Creator đang active (nếu có) không bị đổi.
- **Phụ lục A cần Creator duyệt chữ** trước khi seed (yêu cầu gốc của CR-066 "cho tôi review trước khi seed").
- **Không đạt độ tinh xảo tranh vẽ tay** của mẫu (Q1-a — đã chấp nhận).
- Nhánh/worktree CR-064 và CR-066 còn trên máy cho tới khi Creator cho phép xoá.

## Phụ lục A — Văn bản luật style hình minh hoạ (để Creator duyệt trước khi seed)

Văn bản Phụ lục A của CR-066, với hai chỉnh sửa của CR-067:
1. [S10]: hình phải đọc rõ trên **nền sáng bão hoà** của cảnh (FR7), không chỉ trên `#F3F6FB` và `#140B3A`.
2. [S14]: cận cảnh người dùng `<Person framing="bust">`, cận cảnh bàn tay dùng `<ReachingHand>` (FR8, FR9); thêm dáng "cầm, đếm".

Không câu nào nhắc tên kênh (Q3: mẫu chỉ để tham khảo).

```text
# LUẬT STYLE HÌNH MINH HOẠ CỦA KÊNH

Mọi hình trong thư viện phải trông như do CÙNG MỘT HOẠ SĨ vẽ: tranh vector 2D phẳng kiểu sách tranh và phim hoạt hình hiện đại — khối đầy đặn bo tròn, màu tươi và trong, có chiều sâu nhờ ba tầng sáng (màu thân, một mảng bóng mềm, một điểm sáng), và luôn có chút sự sống: dáng có hướng, chuyển động nhẹ nhàng như đang thở. Hình phải đọc được trong nửa giây khi nhỏ trên màn hình, và vẫn dễ thương khi phóng to. Luật dưới đây áp dụng cho hình AI vẽ, hình bạn tự viết code và hình SVG tải lên. Mã [S..] là mã luật bộ kiểm tra tự động dùng khi báo lỗi (chặn lưu) hay cảnh báo (Creator quyết định).

## 0. Sức sống của hình (đọc trước khi vẽ)
- DÁNG CÓ HƯỚNG: tưởng tượng một đường cong chạy dọc thân (đường hành động) — vật hơi nghiêng, vươn, cúi hay nảy theo ý của nó, không đứng thẳng đơ như bản vẽ kỹ thuật. Vật vô tri cũng có thể đặt hơi chéo 3–8° hoặc có một chi tiết "đang xảy ra" (hơi nước bốc, lá rung, tia sáng loé).
- TƯƠNG PHẢN KHỐI: kết hợp khối to với khối nhỏ, đường cong với đường thẳng, chỗ dày với chỗ mảnh. Một khối chính chiếm phần lớn hình, các khối phụ nhỏ hơn hẳn — tránh nhiều khối cùng cỡ xếp cạnh nhau.
- MỘT ĐIỂM NHẤN: mỗi hình có đúng một chỗ mắt người xem rơi vào trước (mặt, điểm sáng, màu nhấn). Đừng làm mọi chỗ cùng nổi.
- ĐƠN GIẢN HOÁ CÓ CHỦ Ý: giữ các đặc điểm nhận dạng, phóng đại chúng một chút (tai thỏ dài hơn, vỏ ốc xoắn rõ hơn), bỏ mọi thứ còn lại.

## 1. Hình khối
- [S1] KHỐI PHẲNG, ÁNH SÁNG MỀM: mảng chính là màu đặc; gradient (linearGradient, radialGradient) chỉ dùng cho bóng mềm, điểm sáng và quầng sáng, TỐI ĐA 4 gradient một hình (quá thì cảnh báo) — dành chúng cho khối chính và nguồn sáng, khối phụ tô phẳng. Làm mờ chỉ bằng <filter> chứa <feGaussianBlur> (quầng sáng, vệt sáng). KHÔNG pattern, KHÔNG bộ lọc nào khác (feDropShadow, feTurbulence...), KHÔNG ảnh nhúng (<image>), KHÔNG chữ vẽ trong hình (<text>, foreignObject). → LỖI, chặn lưu.
- [S2] KHÔNG VIỀN quanh khối: khối có màu (fill) không có stroke; hình tách khỏi nền nhờ tương phản màu và mảng bóng, không nhờ viền. Nét (stroke) chỉ dùng cho chi tiết mảnh: mắt nhắm, miệng, lông mày, que, dây, tia sáng, lông bàn chải, nét chuyển động. → cảnh báo.
- [S3] BO TRÒN: góc chữ nhật có rx ≥ 6 (khối lớn rx 20–40); đầu nét strokeLinecap="round" (thiếu thì cảnh báo), góc nét strokeLinejoin="round". Tránh góc nhọn trừ khi vật thật sự nhọn (gai, mũi tên, mái nhà) — khi đó bo nhẹ đỉnh.
- [S4] ÍT CHI TIẾT: một vật 5–25 hình con là đủ. Không vẽ hoạ tiết nhỏ hơn 1/40 cạnh hộp, vì chúng biến mất khi hình nhỏ trên màn hình. Chi tiết dồn vào chỗ điểm nhấn, chỗ khác để trống. → cảnh báo khi quá 60 hình con.
- [S5] HÌNH KHỐI TO, RÕ SILHOUETTE: tô đen cả hình vẫn đoán được là vật gì — chọn góc nhìn lộ đặc điểm nhận dạng nhất (con cá nhìn ngang, bông hoa nhìn chính diện, cái cốc nhìn ngang thấy quai). Ưu tiên góc nhìn thẳng hoặc nhìn ngang, không phối cảnh 3D, không nghiêng xa gần; muốn có khối thì thêm một mặt bên hoặc mặt trên mỏng, tô tối/sáng hơn.

## 2. Tô bóng và ánh sáng
- [S6] MỘT MẢNG BÓNG cho mỗi khối lớn, ánh sáng của cả kênh đến từ TRÁI-TRÊN nên bóng nằm phía PHẢI (hoặc phải-dưới): màu thân tối đi bằng shadeOf(màu, -0.12), một hình màu đen opacity 0.06–0.12 đè lên, hoặc một gradient tuyến tính từ màu thân sang shadeOf(màu, -0.18) theo hướng trái-trên → phải-dưới. Mảng bóng có mép cong theo khối (hình lưỡi liềm, nửa vầng trăng), không phải một dải thẳng. Chỗ hai khối chồng lên nhau (đầu trên thân, tay trước bụng) thêm một vệt bóng tiếp xúc mảnh, tối hơn, để khối phía trước nổi lên. Không tô bóng hai phía.
- [S7] ĐIỂM SÁNG VÀ VIỀN SÁNG: vật bóng hay tròn (táo, kẹo, kính, giọt nước, vỏ côn trùng, mắt to) có một vệt trắng opacity 0.3–0.5 ở phía TRÁI-TRÊN, dạng bầu dục nhỏ hoặc vệt cong theo khối, có thể thêm một chấm sáng nhỏ hơn cạnh nó. Vật mờ (vải, lông, gỗ) dùng shadeOf(màu, +0.12) cho mép sáng thay vì vệt trắng. Vật phát sáng hoặc đứng trước nguồn sáng có thể có một quầng radialGradient màu sáng, mờ dần ra ngoài, hoặc một viền sáng mảnh ở mép trái-trên.
- [S8] BÓNG ĐỔ dưới vật đứng trên sàn: <GroundShadow cx cy rx /> (elip đen opacity 0.12) ở đáy hộp, rộng hơn chân vật một chút. Vật bay/lơ lửng thì không có, hoặc có bóng nhỏ và nhạt hơn đặt thấp hơn hẳn để thấy rõ khoảng cách.

## 3. Màu
- [S9] ƯU TIÊN BẢNG MÀU KÊNH cho mặc định của hình (xem danh sách dưới); màu tối/sáng dẫn xuất bằng shadeOf. Vật có màu riêng dễ nhận ra (lá cây, quả chuối, cảnh sát...) được dùng đúng màu của nó.
- [S10] TỐI ĐA 6 MÀU gốc trong một hình (không tính màu dẫn xuất bằng shadeOf và đen/trắng dùng làm bóng/sáng). Phối theo vai: MỘT màu chủ đạo cho khối chính (khoảng 60% diện tích màu), một–hai màu phụ hài hoà với nó, MỘT màu nhấn tương phản cho điểm nhấn (khoảng 10%). Hai khối cạnh nhau phải khác nhau rõ về độ sáng, không chỉ khác sắc độ. Hình phải đọc rõ trên nền sáng bão hoà của cảnh (vàng #FFD84D, xanh cyan #2EC4F0, hồng #FF9EC4), trên nền sáng #F3F6FB và trên nền đêm #140B3A — tránh để khối chính gần trắng, gần tím đen, hoặc trùng sắc với các nền sáng đó.
- [S11] KHÔNG ĐEN THUẦN làm mảng lớn: nét và mắt dùng INK (#3A1F4B), không #000000; đen chỉ dùng làm bóng với opacity thấp. → cảnh báo.
- [S12] Màu chính của vật phải đổi được qua prop color (và các prop màu phụ nếu cần), mọi màu bóng/sáng của khối đó dẫn xuất từ prop bằng shadeOf để đổi màu vẫn đẹp. → cảnh báo.

Bảng màu kênh (mặc định cho hình):
  vàng #FFC857, #FFD23F, #FFC72C · cam #FF9F43, #FF7A45, #E07A5F · đỏ #E8453C, #FF4D6D · hồng #FF5C8A, #FF7AB6, #FF78D6 · tím #7B3FC4, #6C3FC5, #5B2C6F · xanh dương #2D5BFF, #3D7BFF, #8FD3FF · xanh ngọc #2BB6A8, #2EC4B6 · xanh lá #2BB673, #4CAF50, #8BC34A, #B6E34A · nâu #8A5A3C, #D9975B, #9A6A45 · da #F9C4B4 · trung tính #FFFFFF, #F3F6FB, #DDE3EC, #8F9BB3, #2B2140 · mực #3A1F4B · nền đêm #140B3A, #1B1650.

## 4. Nhân vật và khuôn mặt
- [S13] MẶC ĐỊNH KHÔNG CÓ MẶT. Đồ vật, công trình, nơi chốn, thiên nhiên (kim tự tháp, ngôi nhà, núi, cái bàn, đồng xu...) vẽ đúng như vật thật, KHÔNG gắn mặt — sức sống của chúng đến từ dáng, ánh sáng và chuyển động. Chỉ gắn mặt người <Face> khi vật đó là NHÂN VẬT ĐƯỢC NHÂN HOÁ trong phim — mô tả hình ghi rõ nó có mặt, hoặc nó phải biểu lộ cảm xúc (răng đau, vi khuẩn gian ác, giọt nước sợ hãi). Không chắc thì KHÔNG vẽ mặt. Khi được dùng <Face>: đặt nó lên phần thân chính (mặt trước chiếc răng, đầu xe), hơi lệch về phía vật đang hướng tới, giữ nguyên hình dáng thật của vật; cảm xúc thể hiện bằng mood VÀ bằng dáng của cả thân (vui thì nảy lên, sợ thì co lại, buồn thì cúi xuống); KHÔNG tự vẽ mắt kiểu khác (mắt to long lanh, mắt có tròng màu, lông mi).
- [S14] Người: dùng <Person> có sẵn. Chỉ vẽ người mới khi cần trang phục/dáng mà Person không có, và giữ tỉ lệ đầu to (đầu ≈ 1/3 chiều cao), đầu vuông bo tròn, tóc là một mảng, tay chân là khối thuôn tròn; dáng có hành động rõ (vẫy, chỉ, chạy, cầm, đếm) thay vì đứng nghiêm. Cận cảnh người dùng <Person framing="bust">; cận cảnh bàn tay cầm/rút/chỉ/đếm dùng <ReachingHand> của bộ kit, không vẽ bàn tay mới.
- [S15] CON VẬT GIỐNG ĐÚNG LOÀI CỦA NÓ, KHÔNG DÙNG <Face> (mặt người). Giữ đặc điểm nhận dạng của loài: tai, mõm, mũi, mỏ, râu, sừng, vằn, vây, đuôi, dáng đứng — nhìn silhouette phải đoán được là con gì. Mặt con vật tự vẽ bằng vài khối: mắt chấm INK có một chấm sáng trắng nhỏ (không lông mày, không má hồng, không miệng người), mũi/mõm/mỏ đúng loài (mèo: mũi tam giác hồng + miệng chữ "ω" + ria; chó: mõm bầu dục + mũi đen; chim: mỏ tam giác; cá: miệng chấm ở đầu nhọn; giun: đầu tròn trơn, chỉ có mắt chấm). Thân khối tròn/bầu dục, chân ngắn, đầu hơi to là được; không vẽ lông từng sợi, móng, răng nanh chi tiết — một mảng bụng/ngực sáng màu hơn là đủ cho cảm giác lông. Cảm xúc của con vật thể hiện bằng dáng (tai cụp, đuôi vẫy, cúi đầu) và mắt (chấm → nhắm cong khi vui), không bằng mặt người.

## 5. Khung và toạ độ
- [S16] Hình là MỘT component export function TênPascalCase(props) bọc trong <Figure {...fig} size={fig.size ?? <cỡ mặc định>} vw={..} vh={..}>. vw × vh là hộp vẽ, tỉ lệ đúng với vật, vật lấp gần kín hộp (chừa lề khoảng 5%); mọi nét nằm TRONG hộp (trừ bóng đổ sát đáy và tia/gai nhỏ). Chừa chỗ trong hộp cho biên độ chuyển động. → LỖI khi thiếu Figure.
- [S17] Nhận FigureProps (x, y, size, rotate, flip, scale, opacity, still) và truyền nguyên cho Figure. Không tự đặt vị trí tuyệt đối trên khung 1920×1080.
- [S18] Vật nhìn ngang thì đầu/mũi quay sang PHẢI (flip để quay trái), giống Airplane, Toothbrush.
- [S25] Id của gradient, filter, clipPath, mask PHẢI sinh bằng const id = useSvgId('ten') rồi dùng id={id} và fill={`url(#${id})`} — không viết id="..." cố định: cùng một hình xuất hiện hai lần trên một khung sẽ trùng id và hình thứ hai tô sai màu. → cảnh báo.

## 6. Chuyển động
- [S19] Vật sống hoặc có năng lượng CÓ chuyển động tự thân theo frame: thở/nhún (sin(frame/14) biên độ 1–3), chớp mắt (useBlink), lắc/rung (sin(frame/4)), quay (xoay tia, bánh xe), nhấp nháy (đèn), bập bềnh (vật bay). Làm cho nó sống: CHUYỂN ĐỘNG PHÂN LỚP — một chuyển động chính cho thân, và các phần mềm/rời (đuôi, tai, lá, tóc, ăng-ten, ngọn lửa) đung đưa theo cùng nhịp nhưng TRỄ PHA (sin(frame/14 - 0.8)) để có cảm giác quán tính; CO GIÃN GIỮ THỂ TÍCH — khi nhún, scaleY tăng thì scaleX giảm tương ứng (scaleX ≈ 1/scaleY), gốc biến đổi ở đáy vật; LỆCH PHA THEO VỊ TRÍ — cộng phaseOf(fig.x ?? 960, fig.y ?? 540) vào mọi sin để hai hình cùng loại không nhún đồng loạt. Vật chết (bàn, sách, đá) thì đứng yên, hoặc chỉ có một chi tiết nhỏ chuyển động (hơi nước, tia sáng). → cảnh báo khi hình có mặt mà không có chuyển động nào.
- [S20] Chuyển động NHỎ và ĐỀU: không quá 5% cạnh hộp, lặp liền mạch, không giật; ưu tiên nhịp mềm (sin, hoặc interpolate với Easing.inOut) thay cho tuyến tính. Mọi chuyển động tắt được bằng prop still. → cảnh báo.
- [S21] Trạng thái mang nghĩa là PROP, không phải chuyển động tự động: mood, open, decay, on, level... để Kỹ sư điều khiển theo câu thoại. Mỗi trạng thái phải khác nhau rõ khi nhìn từ xa (dáng, màu, chi tiết), không chỉ khác vài pixel.
- [S22] Chỉ dùng useCurrentFrame, interpolate, spring, Easing của remotion. KHÔNG Math.random, Date.now, CSS animation, setTimeout. → LỖI.

## 7. Đặt tên và mô tả
- [S23] Tên component tiếng Anh PascalCase, là danh từ của vật (SchoolBus, Microscope, Cat) — không mang màu hay trạng thái (không RedCar, SadCat).
- [S24] Tên hiển thị tiếng Việt ngắn; thẻ là các từ người ta sẽ tìm; dòng "Cách gọi" liệt kê mọi prop và tỉ lệ hộp: <SchoolBus color mood /> — 320×200.
```
